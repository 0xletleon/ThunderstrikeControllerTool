// Package main — Thunderstrike Controller Tool
//
// 交互式命令行工具，用于读取 NVIDIA SHIELD TV 2017 Thunderstrike
// 手柄的设备信息、刷写/降级固件。
//
// 仅支持蓝牙连接。启动后自动扫描蓝牙 SPP 串口，列出可用端口，
// 选择后通过 Windows HID API 查询设备信息。
package main

import (
	"fmt"
	"os"

	"thunderstrike-controller-tool/cmd"
	"thunderstrike-controller-tool/term"
)

func main() {
	// 启用 Windows 终端 ANSI 支持（清屏、光标控制）
	term.EnableANSI()

	// 无参数时启动交互模式
	if len(os.Args) <= 1 {
		if err := cmd.RunInteractive(); err != nil {
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// 有参数时走 cobra 子命令
	cmd.Execute()
}
