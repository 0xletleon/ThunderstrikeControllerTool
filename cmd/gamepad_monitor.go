// Package cmd — 手柄输入监听适配器。
//
// 实现 core.GamepadMonitor 接口，使用 hid.InputMonitor
// 在后台持续读取 HID 输入报告，检测手柄按键/摇杆活动。
package cmd

import (
	"sync"

	"thunderstrike-controller-tool/core"
	"thunderstrike-controller-tool/hid"
)

// gamepadMonitor 实现 core.GamepadMonitor 接口。
type gamepadMonitor struct {
	mu      sync.Mutex
	monitor *hid.InputMonitor
}

// NewGamepadMonitor 创建手柄监听器。
func NewGamepadMonitor() core.GamepadMonitor {
	return &gamepadMonitor{}
}

// StartMonitoring 启动后台 HID 输入监听。
// 检测到手柄输入时调用 onInput；设备上线（句柄打开成功）时调用 onReady。
// 手柄当前不在线也可以启动：监听器会在后台等待设备上线。
func (g *gamepadMonitor) StartMonitoring(onInput func(), onReady func()) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// 如果已有监听器在运行，先停止
	if g.monitor != nil {
		g.monitor.Close()
		g.monitor = nil
	}

	monitor := hid.NewInputMonitor()
	g.monitor = monitor
	monitor.Start(onInput, onReady)
}

// StopMonitoring 停止后台 HID 输入监听。
func (g *gamepadMonitor) StopMonitoring() {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.monitor != nil {
		g.monitor.Close()
		g.monitor = nil
	}
}

// 编译时接口检查
var _ core.GamepadMonitor = (*gamepadMonitor)(nil)
