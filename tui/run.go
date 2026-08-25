// Package tui — TUI 启动入口。
//
// 提供 Run 函数，由 main.go 调用，启动 Bubble Tea 程序。
// 需要传入 DeviceScanner、FirmwareScanner 和 Flasher 接口实现。
package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// Run 启动 TUI 程序。
//
// scanner 提供 Bluetooth SPP 设备扫描和 HID 信息查询能力。
// fwScanner 提供 blkz 固件目录扫描能力。
// flasher 提供固件刷写执行能力。
// monitor 提供手柄输入监听能力（可为 nil）。
//
// 返回程序退出时的错误。
func Run(scanner DeviceScanner, fwScanner FirmwareScanner, flasher Flasher, monitor GamepadMonitor) error {
	m := NewModel(scanner, fwScanner, flasher, monitor)

	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	// 设置全局 Program 引用，供刷写回调发送消息
	globalProgram = p

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI 运行出错: %w", err)
	}

	// 程序退出，停止手柄监听
	if monitor != nil {
		monitor.StopMonitoring()
	}

	return nil
}
