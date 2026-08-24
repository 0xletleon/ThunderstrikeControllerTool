// Package cmd — 交互模式：蓝牙设备发现、选择、信息展示。
//
// 交互流程：
//  1. WMI 查询蓝牙 SPP 串口
//  2. 列出设备（分隔式列表，无框线）
//  3. 用户选择设备
//  4. 通过 Windows HID API 查询设备信息
//  5. 提示刷写固件
//
// 每次刷新设备列表时清屏，保持界面干净不滚动。
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"thunderstrike-controller-tool/term"
)

// discoveredDevice 表示扫描发现的蓝牙 SPP 设备。
type discoveredDevice struct {
	index      int
	name       string // 显示名称
	port       string // COM 端口
	mac        string // MAC 地址 (XX:XX:XX:XX:XX:XX)
	deviceName string // 设备名称 (如 "NVIDIA Controller")
	fwVersion  string // 当前固件版本 (如 "1.33")，查询后填充
}

// RunInteractive 是交互模式主入口。
func RunInteractive() error {
	reader := bufio.NewReader(os.Stdin)

	for {
		// 清屏 + 标题
		term.ClearScreen()
		printHeader()

		// 扫描蓝牙 SPP 设备
		fmt.Println()
		devices, err := scanBluetoothSpp()
		if err != nil {
			fmt.Printf("  扫描出错: %v\n", err)
		}

		if len(devices) == 0 {
			fmt.Println("  未发现蓝牙设备。")
			fmt.Println("  请确保手柄已与电脑配对并连接。")
		} else {
			fmt.Printf("  找到 %d 个设备\n", len(devices))
			fmt.Println()

			// 打印设备列表
			printDeviceTable(devices)

			fmt.Println()
			fmt.Print("  选择设备 [1] | 刷新 [Enter] | 退出 [q]: ")
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)

			if input == "q" || input == "quit" || input == "exit" {
				return nil
			}

			if input == "" {
				continue // 刷新
			}

			choice, err := strconv.Atoi(input)
			if err != nil {
				continue // 无效输入，刷新
			}

			// 查找选中设备
			var selected *discoveredDevice
			for i := range devices {
				if devices[i].index == choice {
					selected = &devices[i]
					break
				}
			}
			if selected == nil {
				continue
			}

			// 显示设备信息 + 刷写选项
			printDeviceInfo(selected, reader)
		}

		// 刷新提示
		fmt.Println()
		fmt.Print("  按 Enter 刷新，输入 q 退出: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input == "q" || input == "quit" || input == "exit" {
			return nil
		}
	}
}

// printHeader 打印程序标题。
func printHeader() {
	fmt.Println("  Thunderstrike Controller Tool")
	fmt.Println("  NVIDIA SHIELD TV 2017 手柄管理工具")
	fmt.Println("  设备信息读取 · 固件刷写/降级/升级/平刷")
	fmt.Println("  ──────────────────────────────────────────────────────────────────────────")
}

// printDeviceTable 打印设备列表（分隔式，无框线）。
func printDeviceTable(devices []discoveredDevice) {
	fmt.Println("    No.  设备名称             端口    MAC 地址")
	fmt.Println("    ─────────────────────────────────────────────")
	for _, d := range devices {
		fmt.Printf("     %d    %-18s %-6s   %s\n",
			d.index, d.deviceName, d.port, d.mac)
	}
}

// scanBluetoothSpp 通过 WMI 查询蓝牙 SPP 串口。
// 使用 Get-WmiObject Win32_SerialPort 查找 BTHENUM 端口，
// 提取 MAC 地址，瞬时完成，无需串口探测。
func scanBluetoothSpp() ([]discoveredDevice, error) {
	btPorts, err := discoverBtComPorts()
	if err != nil {
		return nil, err
	}

	var devices []discoveredDevice
	idx := 1
	for _, bp := range btPorts {
		name := fmt.Sprintf("%s  %s  (%s)", bp.deviceName(), bp.comPort, bp.macColon)
		devices = append(devices, discoveredDevice{
			index:      idx,
			name:       name,
			port:       bp.comPort,
			mac:        bp.macColon,
			deviceName: bp.deviceName(),
		})
		idx++
	}
	return devices, nil
}

// printDeviceInfo 显示选中设备的信息，然后提示刷写。
func printDeviceInfo(d *discoveredDevice, reader *bufio.Reader) {
	term.ClearScreen()
	printHeader()

	fmt.Println()
	fmt.Println("  ── 设备信息 ─────────────────────────────────────────────────────────────────")
	fmt.Println()

	printBluetoothDeviceInfo(d)

	fmt.Println()
	fmt.Println("  ────────────────────────────────────────────────────────────────────────────")
	fmt.Println()
	fmt.Print("  刷写固件? [y/n]: ")
	input, _ := reader.ReadString('\n')
	if strings.TrimSpace(strings.ToLower(input)) == "y" {
		_ = runInteractiveFlash(d, reader)
	}
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
