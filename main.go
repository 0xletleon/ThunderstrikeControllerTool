// Package main — Thunderstrike Controller Tool
//
// 交互式工具，用于读取 NVIDIA SHIELD TV 2017 Thunderstrike
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
	term.EnableANSI()

	if err := cmd.RunInteractive(); err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
}