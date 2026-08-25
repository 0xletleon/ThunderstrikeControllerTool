// Package hid — 后台输入监听器。
//
// 打开 Thunderstrike HID 设备的只读共享句柄，
// 持续读取 HID 输入报告。当检测到手柄按键/摇杆活动时，
// 通过 channel 通知调用方。
//
// 使用共享模式打开，不干扰其他进程或刷写流程对设备的访问。
// 当设备断开（如刷写后重启）时，自动尝试重新连接。
package hid

import (
	"sync"
	"syscall"
	"time"
	"unsafe"
)

// InputMonitor 后台监听 HID 输入报告。
type InputMonitor struct {
	mu     sync.Mutex
	closed bool
	cancel chan struct{}
}

// NewInputMonitor 打开 Thunderstrike HID 设备的只读共享句柄。
// 如果设备未连接，返回 nil, nil（不报错，只是没有设备可监听）。
func NewInputMonitor() (*InputMonitor, error) {
	// 只检查设备是否存在，不持有句柄
	devicePath, err := FindThunderstrikeHidDevice()
	if err != nil {
		return nil, err
	}
	if devicePath == "" {
		return nil, nil
	}

	return &InputMonitor{
		cancel: make(chan struct{}),
	}, nil
}

// openDevice 尝试打开 HID 设备句柄。
// 成功返回句柄，失败返回 0。
func openDevice() uintptr {
	devicePath, err := FindThunderstrikeHidDevice()
	if err != nil || devicePath == "" {
		return 0
	}

	pathUTF16, _ := syscall.UTF16PtrFromString(devicePath)
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

// Start 在后台 goroutine 中持续读取 HID 输入报告。
// 检测到非零输入时，通过 onInput 回调通知。
func (m *InputMonitor) Start(onInput func()) {
	go m.readLoop(onInput)
}

// readLoop 持续读取 HID 输入报告的循环。
// 当设备断开时，自动重试连接。
func (m *InputMonitor) readLoop(onInput func()) {
	lastNonZeroTime := time.Time{}

	for {
		m.mu.Lock()
		if m.closed {
			m.mu.Unlock()
			return
		}
		m.mu.Unlock()

		// 打开设备句柄
		handle := openDevice()
		if handle == 0 {
			// 设备未连接，等待后重试
			select {
			case <-m.cancel:
				return
			case <-time.After(2 * time.Second):
				continue
			}
		}

		// 在此句柄上循环读取
		readBuf := make([]byte, 256)
		disconnected := false

		for !disconnected {
			m.mu.Lock()
			if m.closed {
				m.mu.Unlock()
				procCloseHandle.Call(handle)
				return
			}
			m.mu.Unlock()

			result := make(chan struct {
				data []byte
				n    uint32
			}, 1)

			go func() {
				var bytesRead uint32
				procReadFile.Call(
					handle,
					uintptr(unsafe.Pointer(&readBuf[0])),
					uintptr(len(readBuf)),
					uintptr(unsafe.Pointer(&bytesRead)),
					0,
				)
				result <- struct {
					data []byte
					n    uint32
				}{readBuf, bytesRead}
			}()

			select {
			case r := <-result:
				if r.n > 0 {
					// 检查是否有非零数据
					hasData := false
					for i := 0; i < int(r.n); i++ {
						if r.data[i] != 0 {
							hasData = true
							break
						}
					}
					if hasData {
						now := time.Now()
						if now.Sub(lastNonZeroTime) >= 100*time.Millisecond {
							lastNonZeroTime = now
							onInput()
						}
					}
				} else {
					// ReadFile 返回 0 字节，设备可能已断开
					disconnected = true
				}

			case <-m.cancel:
				procCancelIoEx.Call(handle, 0)
				procCloseHandle.Call(handle)
				return

			case <-time.After(500 * time.Millisecond):
				// 超时取消当前读取，继续下一轮
				procCancelIoEx.Call(handle, 0)
				// 等待 goroutine 退出
				<-result
			}
		}

		// 设备断开，关闭旧句柄，外层循环会重新打开
		procCloseHandle.Call(handle)

		// 等待一小段时间再重试（设备重启可能需要几秒）
		select {
		case <-m.cancel:
			return
		case <-time.After(2 * time.Second):
		}
	}
}

// Close 停止监听。
func (m *InputMonitor) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil
	}
	m.closed = true
	close(m.cancel)
	return nil
}
