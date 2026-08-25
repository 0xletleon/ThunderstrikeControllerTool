// Package main — Thunderstrike Controller Tool
//
// 基于 Bubble Tea TUI 框架的 NVIDIA SHIELD TV 2017 Thunderstrike
// 手柄固件管理工具。
//
// 支持设备信息读取、固件刷写/降级/升级/平刷。
// 仅支持 Windows，通过蓝牙连接。
package main

import (
	"fmt"
	"os"

	"thunderstrike-controller-tool/cmd"
	"thunderstrike-controller-tool/tui"
)

func main() {
	scanner := cmd.NewTuiDeviceScanner()
	fwScanner := cmd.NewTuiFirmwareScanner()
	flasher := cmd.NewTuiFlasher()
	monitor := cmd.NewTuiGamepadMonitor()

	if err := tui.Run(scanner, fwScanner, flasher, monitor); err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
}
