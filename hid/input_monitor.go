// Package hid — 后台输入监听器。
//
// 手柄会注册多个 HID 集合（蓝牙 HID 服务路径与物理设备路径各一个），
// 按键上报可能落在其中任意一个，因此这里监听全部匹配的集合。
//
// 每个集合用只读共享句柄阻塞读取，不干扰其他进程或刷写流程对设备的访问。
// 设备未连接（休眠、未配对、刷写后重启）时每 2 秒重试，
// 设备出现后自动开始读取，无需重新启动监听。
package hid

import (
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

// InputMonitor 后台监听 HID 输入报告。
type InputMonitor struct {
	mu      sync.Mutex
	closed  bool
	cancel  chan struct{}
	handles map[uintptr]struct{} // 当前打开的句柄，Close 时用于中断阻塞读
}

// NewInputMonitor 创建 HID 输入监听器。
//
// 这里不检查设备当前是否存在：手柄休眠或尚未连接时，监听循环会持续重试，
// 设备上线后自动接入。调用方只需 Start 一次。
func NewInputMonitor() *InputMonitor {
	return &InputMonitor{
		cancel:  make(chan struct{}),
		handles: make(map[uintptr]struct{}),
	}
}

// openPath 打开指定 HID 接口路径的只读共享句柄。失败返回 0。
func openPath(path string) uintptr {
	pathUTF16, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0
	}

	handle, _, _ := procCreateFile.Call(
		uintptr(unsafe.Pointer(pathUTF16)),
		genericRead,
		fileShareAll,
		0, openExisting, 0, 0,
	)
	invalidHandle := uintptr(^uintptr(0))
	if handle == 0 || handle == invalidHandle {
		return 0
	}
	return handle
}

// Start 在后台持续读取所有匹配集合的 HID 输入报告。
// 检测到非零输入时调用 onInput；集合句柄成功打开（设备在线）时调用 onReady（可为 nil）。
func (m *InputMonitor) Start(onInput func(), onReady func()) {
	go m.supervise(onInput, onReady)
}

// supervise 枚举全部 HID 集合，为每个集合启动一个读循环。
// 全部断开后重新枚举，等待设备回来（例如刷写后重启）。
func (m *InputMonitor) supervise(onInput func(), onReady func()) {
	// 多个集合共享同一个通知节流，避免一次按键被重复报告
	var lastNotify atomic.Int64

	for {
		if m.isClosed() {
			return
		}

		paths, err := findThunderstrikeHidPaths()
		if err != nil || len(paths) == 0 {
			select {
			case <-m.cancel:
				return
			case <-time.After(2 * time.Second):
				continue
			}
		}

		var wg sync.WaitGroup
		for _, path := range paths {
			wg.Add(1)
			go func(p string) {
				defer wg.Done()
				m.readHandle(p, onInput, onReady, &lastNotify)
			}(path)
		}
		wg.Wait()

		if m.isClosed() {
			return
		}

		// 设备可能正在重启，稍等后再重新枚举
		select {
		case <-m.cancel:
			return
		case <-time.After(2 * time.Second):
		}
	}
}

// readHandle 在一个集合句柄上阻塞读取，直到设备断开或监听被关闭。
//
// 手柄静止时没有输入报告，ReadFile 会一直阻塞——这是正常状态，
// 不做超时取消（取消会让句柄失效，导致按键被漏掉）。
// 只有在 Close 时才用 CancelIoEx 打断阻塞读。
func (m *InputMonitor) readHandle(path string, onInput func(), onReady func(), lastNotify *atomic.Int64) {
	handle := openPath(path)
	if handle == 0 {
		return
	}
	if !m.addHandle(handle) {
		procCloseHandle.Call(handle)
		return
	}
	defer func() {
		m.removeHandle(handle)
		procCloseHandle.Call(handle)
	}()

	if onReady != nil {
		onReady()
	}

	buf := make([]byte, 512)
	for {
		var bytesRead uint32
		ret, _, _ := procReadFile.Call(
			handle,
			uintptr(unsafe.Pointer(&buf[0])),
			uintptr(len(buf)),
			uintptr(unsafe.Pointer(&bytesRead)),
			0,
		)
		// ReadFile 失败或读到 0 字节：设备断开（或 Close 打断了读取）
		if ret == 0 || bytesRead == 0 {
			return
		}

		if !hasNonZero(buf[:bytesRead]) {
			continue
		}

		// 摇杆/扳机连续上报时限制通知频率，避免刷屏
		now := time.Now().UnixNano()
		if now-lastNotify.Load() < int64(100*time.Millisecond) {
			continue
		}
		lastNotify.Store(now)
		onInput()
	}
}

// addHandle 登记句柄。若监听已被关闭，返回 false。
func (m *InputMonitor) addHandle(h uintptr) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return false
	}
	m.handles[h] = struct{}{}
	return true
}

// removeHandle 注销句柄。
func (m *InputMonitor) removeHandle(h uintptr) {
	m.mu.Lock()
	delete(m.handles, h)
	m.mu.Unlock()
}

// isClosed 报告监听是否已关闭。
func (m *InputMonitor) isClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closed
}

// hasNonZero 判断报告里是否有非零字节。
// 手柄空闲时会周期性上报全零报告，这些不算输入。
func hasNonZero(b []byte) bool {
	for _, v := range b {
		if v != 0 {
			return true
		}
	}
	return false
}

// Close 停止监听，并打断所有阻塞中的 ReadFile。
func (m *InputMonitor) Close() error {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil
	}
	m.closed = true
	open := make([]uintptr, 0, len(m.handles))
	for h := range m.handles {
		open = append(open, h)
	}
	m.mu.Unlock()

	close(m.cancel)

	// 打断 readHandle 里阻塞的 ReadFile，让它尽快退出
	for _, h := range open {
		procCancelIoEx.Call(h, 0)
	}
	return nil
}
