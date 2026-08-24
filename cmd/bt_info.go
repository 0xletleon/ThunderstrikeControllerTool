// Package cmd — Bluetooth SPP device info display.
//
// This file handles reading and displaying device information for
// Bluetooth-connected controllers using the HID-over-SPP protocol.
package cmd

import (
	"errors"
	"fmt"

	"thunderstrike-controller-tool/hid"
	"thunderstrike-controller-tool/spp"
)

// printBluetoothDeviceInfo opens an SPP connection to the given COM port
// and queries the controller for basic device information using the same
// HID report protocol as USB connections.
func printBluetoothDeviceInfo(port string) {
	fmt.Printf("  串口       : %s\n", port)

	client, err := spp.NewHidrawClient(port)
	if err != nil {
		fmt.Printf("  连接失败   : %v\n", err)
		return
	}
	defer client.Close()

	// Query firmware version
	if resp, err := client.SendCommand(hid.CmdVersion, nil); err == nil && len(resp) >= 2 {
		fwStr := fmt.Sprintf("%d.%02d", int(resp[1]), int(resp[0]))
		fmt.Printf("  软件版本   : V%s (0x%02X%02X)\n", fwStr, resp[1], resp[0])
		if len(resp) >= 4 {
			cypStr := fmt.Sprintf("%d.%02d", int(resp[3]), int(resp[2]))
			fmt.Printf("  蓝牙芯片   : CSR v%s\n", cypStr)
		}
		if len(resp) >= 6 {
			hotStr := fmt.Sprintf("%d.%02d", int(resp[5]), int(resp[4]))
			fmt.Printf("  热词引擎   : v%s\n", hotStr)
		}
	} else if errors.Is(err, hid.ErrCmdError) {
		fmt.Printf("  软件版本   : 不支持\n")
	} else {
		fmt.Printf("  软件版本   : 读取失败 (%v)\n", err)
	}

	// Query board info (hardware version / serial)
	if resp, err := client.SendCommand(hid.CmdBoardInfo, nil); err == nil && len(resp) >= 2 {
		fmt.Printf("  硬件版本   : 0x%02X\n", resp[1])
		if len(resp) > 2 {
			serial := extractSerial(resp[2:])
			if serial != "" {
				fmt.Printf("  序列号     : %s\n", serial)
			}
		}
	} else if errors.Is(err, hid.ErrCmdError) {
		fmt.Printf("  硬件版本   : 不支持\n")
	} else {
		fmt.Printf("  硬件版本   : 读取失败\n")
	}

	// Query MAC address
	if resp, err := client.SendCommand(hid.CmdMacAddress, nil); err == nil && len(resp) >= 6 {
		fmt.Printf("  MAC 地址   : %s\n", hid.FormatMacAddress(resp[:6]))
	} else if errors.Is(err, hid.ErrCmdError) {
		fmt.Printf("  MAC 地址   : 不支持\n")
	} else {
		fmt.Printf("  MAC 地址   : 读取失败\n")
	}

	// Query transport type
	if resp, err := client.SendCommand(hid.CmdSetTransport, nil); err == nil && len(resp) >= 1 {
		transport := "wired"
		if resp[0] == 0x02 {
			transport = "wireless"
		}
		fmt.Printf("  传输方式   : %s\n", transport)
	} else if errors.Is(err, hid.ErrCmdError) {
		fmt.Printf("  传输方式   : 不支持\n")
	} else {
		fmt.Printf("  传输方式   : 读取失败\n")
	}
}
