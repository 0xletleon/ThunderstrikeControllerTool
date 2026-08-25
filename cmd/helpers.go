// Package cmd — 旧 CLI 模式遗留的辅助函数。
//
// 这些函数被 tui_adapter.go 和 flash_core.go 调用，
// 从旧的 interactive.go、bt_info.go、interactive_flash.go、blkz_list.go 中提取。
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"thunderstrike-controller-tool/hid"
)

// queryHidFirmwareVersion 通过 HID 快速查询固件版本号，失败返回空字符串。
func queryHidFirmwareVersion() string {
	client, err := hid.OpenWindowsHidClient()
	if err != nil {
		return ""
	}
	defer client.Close()

	verData, err := client.SendCommand(hid.CmdVersion, nil)
	if err != nil || len(verData) < 2 {
		return ""
	}
	major := int(verData[1])
	minor := int(verData[0])
	return fmt.Sprintf("%d.%02d", major, minor)
}

// extractSerial 从 board info 数据中提取 ASCII 序列号。
func extractSerial(data []byte) string {
	var sb strings.Builder
	for _, b := range data {
		if b == 0xFF || b == 0x00 {
			break
		}
		if b >= 0x20 && b < 0x7F {
			sb.WriteByte(b)
		}
	}
	return sb.String()
}

// adcToPercent 将原始 ADC 值（0-255）转换为电池电量百分比。
//
// 电池参数：2×NiMH LSD 串联，标称 2.4V DC，1900mAh。
// 电压范围 2.0V（空）~ 2.8V（满），CSR ADC 参考电压 3.3V。
//
//	ADC 空电 ≈ 2.0/3.3×255 = 155
//	ADC 满电 ≈ 2.8/3.3×255 = 216
func adcToPercent(raw int) int {
	const (
		adcEmpty = 155 // 2.0V — 0%
		adcFull  = 216 // 2.8V — 100%
	)
	if raw <= adcEmpty {
		return 0
	}
	if raw >= adcFull {
		return 100
	}
	return (raw - adcEmpty) * 100 / (adcFull - adcEmpty)
}

// batteryLevelText 将电量百分比转换为可读描述。
func batteryLevelText(pct int) string {
	if pct < 20 {
		return "偏低"
	}
	if pct < 50 {
		return "中等"
	}
	return "正常"
}

// checkBatteryRaw 通过 HID 查询手柄电量原始 ADC 值。
// 返回 0-255 的原始值，供调用方自行判断电量水平。
func checkBatteryRaw() (int, error) {
	client, err := hid.OpenWindowsHidClient()
	if err != nil {
		return 0, fmt.Errorf("打开 HID 设备: %w", err)
	}
	defer client.Close()

	data, err := client.SendCommand(hid.CmdBatteryState, nil)
	if err != nil {
		return 0, fmt.Errorf("电池查询: %w", err)
	}
	if len(data) < 1 {
		return 0, fmt.Errorf("电池: 空响应")
	}
	return int(data[0]), nil
}

// sendHidReset 通过 HID 发送重启指令。
// APPLY_OTA 已触发重启，HID reset 失败不影响刷写结果，故吞掉 error。
func sendHidReset() {
	client, err := hid.OpenWindowsHidClient()
	if err != nil {
		return
	}
	defer client.Close()

	_, _ = client.SendCommand(hid.CmdReset, []byte{0x01})
}

// versionDirection 判断固件版本是升级、降级还是平刷。
func versionDirection(current, target string) string {
	if current == "" {
		return "未知"
	}
	if target == current {
		return "平刷"
	}
	if target > current {
		return "升级"
	}
	return "降级"
}

// findLogDir 返回日志目录路径（exe 同级 logs/）。
func findLogDir() string {
	if exePath, err := os.Executable(); err == nil {
		return filepath.Join(filepath.Dir(exePath), "logs")
	}
	return "logs"
}

// findBlkzDir 查找 blkz/ 固件目录。
// 查找顺序：1. exe 同目录  2. 当前工作目录
// 未找到返回空字符串。
func findBlkzDir() string {
	if exePath, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exePath), "blkz")
		if dirExists(candidate) {
			return candidate
		}
	}
	candidate := filepath.Join(".", "blkz")
	if dirExists(candidate) {
		if abs, err := filepath.Abs(candidate); err == nil {
			return abs
		}
		return candidate
	}
	return ""
}

// dirExists 判断路径是否存在且为目录。
func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
