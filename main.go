// Package main — Thunderstrike Controller Tool
//
// NVIDIA SHIELD TV 2017 Thunderstrike 手柄固件管理工具。
//
// 原生 GUI 窗口（MyGo 自绘界面），业务实现来自 cmd 包。
//
// 支持设备信息读取、固件刷写/降级/升级/平刷。仅支持 Windows，通过蓝牙连接。
package main

import (
	"fmt"
	"os"

	"thunderstrike-controller-tool/cmd"
	"thunderstrike-controller-tool/core"
	"thunderstrike-controller-tool/gui"
)

func main() {
	svc := core.Services{
		Scanner:   cmd.NewDeviceScanner(),
		Firmwares: cmd.NewFirmwareScanner(),
		Flasher:   cmd.NewFlasher(),
		Monitor:   cmd.NewGamepadMonitor(),
	}

	if err := gui.Run(svc); err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
}
