// debug_hid — enumerate Windows HID devices using simpler approach.
// Usage: go run ./cmd/debug_hid/
package main

import (
	"encoding/binary"
	"fmt"
	"syscall"
	"unsafe"
)

var (
	hidDll   = syscall.NewLazyDLL("hid.dll")
	setupAPI = syscall.NewLazyDLL("setupapi.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
)

var (
	procHidD_GetAttributes = hidDll.NewProc("HidD_GetAttributes")

	procSetupDiGetClassDevsW             = setupAPI.NewProc("SetupDiGetClassDevsW")
	procSetupDiEnumDeviceInterfaces      = setupAPI.NewProc("SetupDiEnumDeviceInterfaces")
	procSetupDiGetDeviceInterfaceDetailW = setupAPI.NewProc("SetupDiGetDeviceInterfaceDetailW")
	procSetupDiDestroyDeviceInfoList     = setupAPI.NewProc("SetupDiDestroyDeviceInfoList")

	procCreateFile  = kernel32.NewProc("CreateFileW")
	procCloseHandle = kernel32.NewProc("CloseHandle")
	procWriteFile   = kernel32.NewProc("WriteFile")
	procReadFile    = kernel32.NewProc("ReadFile")
)

// HID GUID: {4D1E55B2-F16F-11CF-88CB-001111000030}
var hidGuid = syscall.GUID{
	Data1: 0x4D1E55B2,
	Data2: 0xF16F,
	Data3: 0x11CF,
	Data4: [8]byte{0x88, 0xCB, 0x00, 0x11, 0x11, 0x00, 0x00, 0x30},
}

type spDeviceInterfaceData struct {
	cbSize             uint32
	interfaceClassGuid syscall.GUID
	flags              uint32
	reserved           [2]uint32 // padding to match Windows struct size
}

type hidAttributes struct {
	Size          uint32
	VendorID      uint16
	ProductID     uint16
	VersionNumber uint16
}

func main() {
	fmt.Println("=== Windows HID Device Enumeration ===")

	// Get all HID device interfaces
	hInfo, _, callErr := procSetupDiGetClassDevsW.Call(
		uintptr(unsafe.Pointer(&hidGuid)),
		0, 0,
		0x00000010|0x00000002, // DIGCF_DEVICEINTERFACE | DIGCF_PRESENT
	)
	fmt.Printf("SetupDiGetClassDevsW: hInfo=%v, err=%v\n", hInfo, callErr)
	if hInfo == 0 || hInfo == uintptr(^uintptr(0)) {
		fmt.Println("Failed to get device info set")
		return
	}
	defer procSetupDiDestroyDeviceInfoList.Call(hInfo)

	index := uint32(0)
	for {
		var ifaceData spDeviceInterfaceData
		ifaceData.cbSize = uint32(unsafe.Sizeof(ifaceData))
		fmt.Printf("ifaceData.cbSize = %d\n", ifaceData.cbSize)

		ret, _, err := procSetupDiEnumDeviceInterfaces.Call(
			hInfo,
			0,
			uintptr(unsafe.Pointer(&hidGuid)),
			uintptr(index),
			uintptr(unsafe.Pointer(&ifaceData)),
		)
		fmt.Printf("Enum[%d]: ret=%d err=%v\n", index, ret, err)
		if ret == 0 {
			break
		}
		index++

		// Get required size
		var detailSize uint32
		ret, _, err = procSetupDiGetDeviceInterfaceDetailW.Call(
			hInfo,
			uintptr(unsafe.Pointer(&ifaceData)),
			0, 0,
			uintptr(unsafe.Pointer(&detailSize)),
			0,
		)
		fmt.Printf("  GetDetailSize: ret=%d size=%d err=%v\n", ret, detailSize, err)
		if detailSize == 0 {
			continue
		}

		// Allocate and fill detail data
		detailBuf := make([]byte, detailSize)
		*(*uint32)(unsafe.Pointer(&detailBuf[0])) = 8 // cbSize for SP_DEVICE_INTERFACE_DETAIL_DATA_W

		ret, _, err = procSetupDiGetDeviceInterfaceDetailW.Call(
			hInfo,
			uintptr(unsafe.Pointer(&ifaceData)),
			uintptr(unsafe.Pointer(&detailBuf[0])),
			uintptr(detailSize),
			0, 0,
		)
		fmt.Printf("  GetDetail: ret=%d err=%v\n", ret, err)
		if ret == 0 {
			continue
		}

		devicePath := utf16BytesToString(detailBuf[4:])
		fmt.Printf("  Path: %s\n", devicePath)

		// Open device
		pathUTF16, _ := syscall.UTF16PtrFromString(devicePath)
		handle, _, _ := procCreateFile.Call(
			uintptr(unsafe.Pointer(pathUTF16)),
			0x80000000|0x40000000, // GENERIC_READ | GENERIC_WRITE
			0x00000007,           // FILE_SHARE_READ | WRITE | DELETE
			0,
			3, // OPEN_EXISTING
			0, 0,
		)
		invalidHandle := uintptr(^uintptr(0))
		if handle == 0 || handle == invalidHandle {
			fmt.Println("  -> Open failed, trying read-only")
			handle, _, _ = procCreateFile.Call(
				uintptr(unsafe.Pointer(pathUTF16)),
				0x80000000, // GENERIC_READ only
				0x00000007,
				0,
				3, 0, 0,
			)
			if handle == 0 || handle == invalidHandle {
				fmt.Println("  -> Open failed")
				continue
			}
		}

		// Get attributes
		var attrs hidAttributes
		attrs.Size = uint32(unsafe.Sizeof(attrs))
		procHidD_GetAttributes.Call(handle, uintptr(unsafe.Pointer(&attrs)))

		fmt.Printf("  VID: 0x%04X  PID: 0x%04X  Version: 0x%04X\n",
			attrs.VendorID, attrs.ProductID, attrs.VersionNumber)

		// If this is an NVIDIA device (VID 0x0955)
		if attrs.VendorID == 0x0955 {
			fmt.Println("  >>> NVIDIA Controller! <<<")

			// Send HID VERSION command: [0x04][0x01][txn][pad to 33]
			report := make([]byte, 33)
			report[0] = 0x04 // RequestPrefix
			report[1] = 0x01 // CmdVersion
			report[2] = 0x01 // txn ID

			var written uint32
			ret, _, _ := procWriteFile.Call(
				handle,
				uintptr(unsafe.Pointer(&report[0])),
				uintptr(len(report)),
				uintptr(unsafe.Pointer(&written)),
				0,
			)
			fmt.Printf("  Write: ret=%d written=%d\n", ret, written)

			if ret != 0 {
				readBuf := make([]byte, 256)
				var bytesRead uint32
				ret, _, _ = procReadFile.Call(
					handle,
					uintptr(unsafe.Pointer(&readBuf[0])),
					uintptr(len(readBuf)),
					uintptr(unsafe.Pointer(&bytesRead)),
					0,
				)
				fmt.Printf("  Read: ret=%d bytes=%d\n", ret, bytesRead)
				if bytesRead > 0 {
					fmt.Printf("  Data: % X\n", readBuf[:bytesRead])
				}
			}
		}

		procCloseHandle.Call(handle)
		fmt.Println()
	}

	fmt.Printf("\nTotal devices found: %d\n", index)
	_ = binary.LittleEndian
}

func utf16BytesToString(b []byte) string {
	if len(b) < 2 {
		return ""
	}
	var runes []rune
	for i := 0; i+1 < len(b); i += 2 {
		c := uint16(b[i]) | uint16(b[i+1])<<8
		if c == 0 {
			break
		}
		runes = append(runes, rune(c))
	}
	return string(runes)
}
