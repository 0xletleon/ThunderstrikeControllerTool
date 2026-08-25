// Package cmd — 设备信息读取（通过 Windows HID API）。
//
// 手柄通过蓝牙注册两个独立接口：
//   - SPP (COM 口) → 固件刷写
//   - HID (\\?\hid#... 路径) → 设备信息查询
//
// 通过 Windows HID API 发送 HID 报告查询 VERSION/BOARD_INFO/MAC_ADDRESS。
package cmd

import (
	"fmt"
	"strings"

	"thunderstrike-controller-tool/hid"
)

// printBluetoothDeviceInfo 通过 Windows HID API 查询设备信息并显示。
// 设备名称和 MAC 来自 WMI，固件版本/硬件版本/序列号来自 HID 查询。
// 输出格式：缩进对齐，无框线。
// 查询成功后将固件版本写入 d.fwVersion 供后续刷写流程使用。
func printBluetoothDeviceInfo(d *discoveredDevice) {
	printInfoLine("设备名称", d.deviceName)
	printInfoLine("MAC 地址", d.mac)

	// 打开 Windows HID 设备
	client, err := hid.OpenWindowsHidClient()
	if err != nil {
		printInfoLine("软件版本", "读取失败 (HID 设备未找到)")
		printInfoLine("提示", "请确保手柄已连接并唤醒")
		return
	}
	defer client.Close()

	// 查询固件版本
	// 响应: [fw_minor][fw_major][csr_minor][csr_major][hw_minor][hw_major]...
	verData, err := client.SendCommand(hid.CmdVersion, nil)
	if err != nil {
		printInfoLine("软件版本", fmt.Sprintf("读取失败 (%v)", err))
	} else if len(verData) >= 2 {
		major := int(verData[1])
		minor := int(verData[0])
		d.fwVersion = fmt.Sprintf("%d.%02d", major, minor)
		printInfoLine("软件版本", fmt.Sprintf("V%d.%02d (0x%02X%02X)", major, minor, major, minor))
		if len(verData) >= 4 {
			printInfoLine("蓝牙芯片",
				fmt.Sprintf("CSR v%d.%02d", int(verData[3]), int(verData[2])))
		}
		if len(verData) >= 6 {
			printInfoLine("热词引擎",
				fmt.Sprintf("v%d.%02d", int(verData[5]), int(verData[4])))
		}
	} else {
		printInfoLine("软件版本", "响应数据不足")
	}

	// 查询硬件版本 + 序列号
	boardData, err := client.SendCommand(hid.CmdBoardInfo, nil)
	if err != nil {
		if err == hid.ErrCmdError {
			printInfoLine("硬件版本", "不支持")
		} else {
			printInfoLine("硬件版本", fmt.Sprintf("读取失败 (%v)", err))
		}
	} else if len(boardData) >= 2 {
		boardType := int(boardData[0])
		boardRev := int(boardData[1])
		printInfoLine("硬件版本", fmt.Sprintf("Board %d Rev %d", boardType, boardRev))
		serial := extractSerial(boardData[2:])
		if serial != "" {
			printInfoLine("序列号", serial)
		}
	} else {
		printInfoLine("硬件版本", "数据不足")
	}

	// 查询 MAC 地址（与 WMI 对比验证）
	macData, err := client.SendCommand(hid.CmdMacAddress, nil)
	if err != nil {
		if err == hid.ErrCmdError {
			printInfoLine("MAC(HID)", "不支持")
		} else {
			printInfoLine("MAC(HID)", fmt.Sprintf("读取失败 (%v)", err))
		}
	} else if len(macData) >= 6 {
		printInfoLine("MAC(HID)", hid.FormatMacAddress(macData[:6]))
	} else {
		printInfoLine("MAC(HID)", fmt.Sprintf("数据不足 (%d bytes)", len(macData)))
	}

	printInfoLine("传输方式", "蓝牙 HID")

	// 查询电量
	battData, err := client.SendCommand(hid.CmdBatteryState, nil)
	if err != nil {
		printInfoLine("电量", fmt.Sprintf("读取失败 (%v)", err))
	} else if len(battData) >= 1 {
		raw := int(battData[0])
		pct := adcToPercent(raw)
		level := batteryLevelText(pct)
		printInfoLine("电量", fmt.Sprintf("%d%%  ADC: %d (%s)", pct, raw, level))
	} else {
		printInfoLine("电量", "数据不足")
	}
}

// printInfoLine 打印一行 "    标签      值"。
// 标签固定占 8 个显示宽度，值跟在后面，无框线不需要截断。
func printInfoLine(label, value string) {
	// 中文标签宽度处理：4 个中文字 = 8 显示宽度
	labelWidth := displayWidth(label)
	pad := 8 - labelWidth
	if pad < 1 {
		pad = 1
	}
	fmt.Printf("    %s%s%s\n", label, strings.Repeat(" ", pad), value)
}

// displayWidth 计算字符串的显示宽度（中文=2，ASCII=1）。
func displayWidth(s string) int {
	w := 0
	for _, r := range s {
		w += runeWidth(r)
	}
	return w
}

// runeWidth 返回字符的显示宽度（中文=2，ASCII=1）。
func runeWidth(r rune) int {
	if r >= 0x1100 && (
		r <= 0x115F || // 韩文
		r >= 0x2E80 && r <= 0xA4CF || // 中日韩
		r >= 0xAC00 && r <= 0xD7A3 || // 韩文音节
		r >= 0xF900 && r <= 0xFAFF || // 兼容汉字
		r >= 0xFE30 && r <= 0xFE4F || // 兼容形式
		r >= 0xFF00 && r <= 0xFF60 || // 全角
		r >= 0xFFE0 && r <= 0xFFE6) { // 全角符号
		return 2
	}
	return 1
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
