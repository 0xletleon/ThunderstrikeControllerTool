// Package cmd implements the Thunderstrike Controller Tool CLI.
//
// 默认运行交互式模式（无参数时）：
//   1. 扫描蓝牙 SPP 串口
//   2. 列出发现的端口
//   3. 选择后显示设备基本信息
//
// 也可以通过子命令直接运行（保留供高级用户/脚本使用）：
//   scan  — 扫描蓝牙 SPP 串口
//   flash — 刷写/降级固件（蓝牙 SPP）
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// rootCmd is the root command. When called without subcommands,
// it launches the interactive mode via RunInteractive().
var rootCmd = &cobra.Command{
	Use:   "thunderstrike-controller-tool",
	Short: "Thunderstrike Controller Tool",
	Long: `Thunderstrike Controller Tool — NVIDIA SHIELD TV 2017 手柄管理工具

支持设备信息读取、固件刷写/降级/升级/平刷。

无参数启动进入交互式模式：
  自动扫描蓝牙 SPP 串口 → 列出端口 → 选择后显示基本信息

也可使用子命令：
  scan    扫描蓝牙 SPP 串口
  flash   刷写/降级固件 (蓝牙 SPP)`,
}

// Execute runs the root command (used when subcommands are passed).
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	// Register legacy subcommands (kept for advanced/scripted use).
	rootCmd.AddCommand(scanCmd)
	rootCmd.AddCommand(flashCmd)
}
