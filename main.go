// Package main — Thunderstrike Controller Tool
//
// 交互式命令行工具，用于读取 NVIDIA SHIELD TV 2017 Thunderstrike
// 手柄的设备信息、刷写/降级固件。
//
// 仅支持蓝牙 SPP 连接。启动后自动扫描蓝牙串口，列出可用端口，
// 选择后显示基本信息（硬件版本、软件版本、MAC 地址等）。
package main

import (
	"fmt"
	"os"

	"thunderstrike-controller-tool/cmd"
)

func main() {
	// If no args, run interactive mode
	if len(os.Args) <= 1 {
		runInteractive()
		return
	}

	// If args provided, fall through to cobra subcommands
	cmd.Execute()
}

func runInteractive() {
	printBanner()

	if err := cmd.RunInteractive(); err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
}

func printBanner() {
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════")
	fmt.Println("  Thunderstrike Controller Tool")
	fmt.Println("  NVIDIA SHIELD TV 2017 手柄管理工具")
	fmt.Println("  支持设备信息读取 · 固件刷写/降级/升级/平刷")
	fmt.Println("═══════════════════════════════════════════════")
	fmt.Println()
}
