// Package cmd — 手柄输入监听适配器。
//
// 实现 tui.GamepadMonitor 接口，使用 hid.InputMonitor
// 在后台持续读取 HID 输入报告，检测手柄按键/摇杆活动。
package cmd

import (
	"sync"

	"thunderstrike-controller-tool/hid"
	"thunderstrike-controller-tool/tui"
)

// tuiGamepadMonitor 实现 tui.GamepadMonitor 接口。
type tuiGamepadMonitor struct {
	mu      sync.Mutex
	monitor *hid.InputMonitor
}

// NewTuiGamepadMonitor 创建 TUI 手柄监听器。
func NewTuiGamepadMonitor() tui.GamepadMonitor {
	return &tuiGamepadMonitor{}
}

// StartMonitoring 启动后台 HID 输入监听。
// 检测到手柄输入时调用 onInput 回调。
func (g *tuiGamepadMonitor) StartMonitoring(onInput func()) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// 如果已有监听器在运行，先停止
	if g.monitor != nil {
		g.monitor.Close()
		g.monitor = nil
	}

	// 尝试打开 HID 设备
	monitor, err := hid.NewInputMonitor()
	if err != nil || monitor == nil {
		// 设备未连接或出错，静默失败
		return
	}

	g.monitor = monitor
	monitor.Start(onInput)
}

// StopMonitoring 停止后台 HID 输入监听。
func (g *tuiGamepadMonitor) StopMonitoring() {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.monitor != nil {
		g.monitor.Close()
		g.monitor = nil
	}
}

// 编译时接口检查
var _ tui.GamepadMonitor = (*tuiGamepadMonitor)(nil)
