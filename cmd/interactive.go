// Package cmd — interactive Bluetooth SPP device selection and info display.
//
// This file implements the interactive CLI flow:
//  1. Scan for Bluetooth SPP serial ports
//  2. Probe each port to identify Thunderstrike controllers
//  3. List found devices
//  4. User selects a device by number
//  5. Print device info (hardware version, firmware version, MAC address)
//  6. Offer firmware flashing option
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"go.bug.st/serial"
	"thunderstrike-controller-tool/spp"
)

// discoveredDevice represents a Bluetooth SPP port found during scanning.
type discoveredDevice struct {
	index    int
	name     string // display name
	port     string // COM port path
	isCtrl   bool   // true if identified as Thunderstrike controller
}

// runInteractive is the main interactive entry point.
func runInteractive() error {
	reader := bufio.NewReader(os.Stdin)

	for {
		// Scan for Bluetooth SPP ports
		devices, err := scanBluetoothSpp()
		if err != nil {
			fmt.Printf("  扫描出错: %v\n", err)
		}

		if len(devices) == 0 {
			fmt.Println()
			fmt.Println("  未发现蓝牙串口设备。")
			fmt.Println("  请确保手柄已与电脑配对并连接。")
		} else {
			fmt.Println()
			fmt.Printf("  发现 %d 个蓝牙串口：\n", len(devices))
			fmt.Println()
			for _, d := range devices {
				fmt.Printf("  [%d] %s\n", d.index, d.name)
			}
			fmt.Println()

			// Prompt for selection
			fmt.Print("  请选择设备编号 (回车键刷新): ")
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)

			choice, err := strconv.Atoi(input)
			if err != nil {
				fmt.Printf("  无效输入: %s\n\n", input)
				continue
			}

			if choice == 0 {
				continue // refresh
			}

			// Find selected device
			var selected *discoveredDevice
			for i := range devices {
				if devices[i].index == choice {
					selected = &devices[i]
					break
				}
			}
			if selected == nil {
				fmt.Printf("  编号 %d 不在列表中。\n\n", choice)
				continue
			}

			// Print device info and offer flashing
			printDeviceInfo(selected, reader)
		}

		// Prompt to continue
		fmt.Println()
		fmt.Print("  按 Enter 刷新设备列表，输入 q 退出: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input == "q" || input == "quit" || input == "exit" {
			return nil
		}
	}
}

// scanBluetoothSpp scans for Bluetooth SPP serial ports.
// On Windows, Bluetooth SPP ports are named COMxx.
// Each port is probed to check if it's a Thunderstrike controller.
func scanBluetoothSpp() ([]discoveredDevice, error) {
	ports, err := serial.GetPortsList()
	if err != nil {
		return nil, err
	}

	var devices []discoveredDevice
	idx := 1
	for _, port := range ports {
		if !strings.HasPrefix(strings.ToUpper(port), "COM") {
			continue
		}

		// Probe to check if this is a Thunderstrike controller.
		fmt.Printf("  探测 %s ...\r", port)
		isCtrl := spp.ProbeController(port)

		var name string
		if isCtrl {
			name = fmt.Sprintf("Thunderstrike (蓝牙)  %s", port)
		} else {
			name = fmt.Sprintf("蓝牙串口 (SPP)  %s", port)
		}
		devices = append(devices, discoveredDevice{
			index:  idx,
			name:   name,
			port:   port,
			isCtrl: isCtrl,
		})
		idx++
	}
	fmt.Print("                          \r") // clear probe message
	return devices, nil
}

// printDeviceInfo connects to the selected Bluetooth device and prints info,
// then offers firmware flashing.
func printDeviceInfo(d *discoveredDevice, reader *bufio.Reader) {
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════")
	fmt.Println("  设备基本信息")
	fmt.Println("═══════════════════════════════════════════")
	fmt.Println()

	printBluetoothDeviceInfo(d.port)

	// Offer firmware flashing
	fmt.Println()
	fmt.Println("───────────────────────────────────────────")
	fmt.Print("  是否刷写固件? (y/n): ")
	input, _ := reader.ReadString('\n')
	if strings.TrimSpace(strings.ToLower(input)) == "y" {
		_ = runInteractiveFlash(d.port, reader)
	}
}

// extractSerial extracts the ASCII serial number from board info data.
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
