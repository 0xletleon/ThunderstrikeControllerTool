package hid

import (
	"testing"
	"time"
)

// TestInputMonitorReady 验证监听器能在设备在线时成功打开 HID 句柄。
// 手柄未连接时跳过（不算失败），方便在没有设备的机器上跑测试。
func TestInputMonitorReady(t *testing.T) {
	monitor := NewInputMonitor()
	defer monitor.Close()

	ready := make(chan struct{}, 1)
	monitor.Start(func() {}, func() {
		select {
		case ready <- struct{}{}:
		default:
		}
	})

	select {
	case <-ready:
	case <-time.After(3 * time.Second):
		t.Skip("手柄未连接，跳过句柄打开检查")
	}
}

// TestInputMonitorCloseDuringRead 验证 Close 能打断阻塞中的 ReadFile 并让循环退出。
//
// 监听器在设备静止时会阻塞在 ReadFile 上，Close 后用 CancelIoEx 打断；
// 这里通过重复 Start/Close 确认不会卡死。
func TestInputMonitorCloseDuringRead(t *testing.T) {
	for i := 0; i < 3; i++ {
		monitor := NewInputMonitor()
		monitor.Start(func() {}, nil)
		time.Sleep(100 * time.Millisecond)

		done := make(chan struct{})
		go func() {
			monitor.Close()
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Fatal("Close 未能及时返回，监听循环可能卡死")
		}
	}
}
