// Package cmd — device info display via Windows HID API.
//
// The Thunderstrike controller exposes two Bluetooth interfaces:
//  1. SPP (COM port) — for firmware flashing only (NOP/ERASE/WRITE/...)
//  2. HID device (\\?\hid#... path) — for device info queries (VERSION, etc.)
//
// This file uses the Windows HID API to query firmware version, board info,
// and MAC address. Device name comes from WMI discovery data.
package cmd

import (
	"fmt"

	"thunderstrike-controller-tool/hid"
)

// printBluetoothDeviceInfo queries device info via the Windows HID interface.
// It opens the HID device, sends VERSION/BOARD_INFO/MAC_ADDRESS commands,
// and displays the results alongside WMI-discovered name and MAC.
func printBluetoothDeviceInfo(d *discoveredDevice) {
	fmt.Printf("  设备名称   : %s\n", d.deviceName)
	fmt.Printf("  MAC 地址   : %s (WMI)\n", d.mac)

	// Open Windows HID device
	client, err := hid.OpenWindowsHidClient()
	if err != nil {
		fmt.Printf("  软件版本   : 读取失败 (HID 设备未找到)\n")
		fmt.Printf("  硬件版本   : 读取失败\n")
		fmt.Println("  提示       : 请确保手柄已连接并唤醒")
		return
	}
	defer client.Close()

	// Query firmware version
	// Response: [fw_minor][fw_major][csr_minor][csr_major][hw_minor][hw_major]...
	fmt.Printf("  软件版本   : ")
	verData, err := client.SendCommand(hid.CmdVersion, nil)
	if err != nil {
		fmt.Printf("读取失败 (%v)\n", err)
	} else if len(verData) >= 2 {
		major := int(verData[1])
		minor := int(verData[0])
		fmt.Printf("V%d.%02d (0x%02X%02X)\n", major, minor, major, minor)
		if len(verData) >= 4 {
			cMajor := int(verData[3])
			cMinor := int(verData[2])
			fmt.Printf("  蓝牙芯片   : CSR v%d.%02d\n", cMajor, cMinor)
		}
		if len(verData) >= 6 {
			hMajor := int(verData[5])
			hMinor := int(verData[4])
			fmt.Printf("  热词引擎   : v%d.%02d\n", hMajor, hMinor)
		}
	} else {
		fmt.Println("响应数据不足")
	}

	// Query board info (hardware version + serial number)
	fmt.Printf("  硬件版本   : ")
	boardData, err := client.SendCommand(hid.CmdBoardInfo, nil)
	if err != nil {
		if err == hid.ErrCmdError {
			fmt.Println("不支持 (CMD_ERROR)")
		} else {
			fmt.Printf("读取失败 (%v)\n", err)
		}
	} else if len(boardData) >= 2 {
		// Response: [board_type][board_rev][serial_string...]
		// e.g. 00 02 30 33 32 31 38 31 37 30 38 36 39 34 38 = type=0 rev=2 serial="0321817086948"
		boardType := int(boardData[0])
		boardRev := int(boardData[1])
		serial := extractSerial(boardData[2:])
		if serial != "" {
			fmt.Printf("Board %d Rev %d, S/N: %s\n", boardType, boardRev, serial)
		} else {
			fmt.Printf("Board %d Rev %d\n", boardType, boardRev)
		}
	} else {
		fmt.Println("数据不足")
	}

	// Query MAC address via HID (verify against WMI)
	fmt.Printf("  MAC(HID)   : ")
	macData, err := client.SendCommand(hid.CmdMacAddress, nil)
	if err != nil {
		if err == hid.ErrCmdError {
			fmt.Println("不支持 (CMD_ERROR)")
		} else {
			fmt.Printf("读取失败 (%v)\n", err)
		}
	} else if len(macData) >= 6 {
		fmt.Println(hid.FormatMacAddress(macData[:6]))
	} else {
		fmt.Printf("数据不足 (%d bytes)\n", len(macData))
	}

	fmt.Printf("  传输方式   : 蓝牙 HID\n")
}
