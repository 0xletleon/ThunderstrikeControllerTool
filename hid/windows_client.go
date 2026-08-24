// Package hid — Windows HID API client for Bluetooth-connected controllers.
//
// On Windows, Bluetooth HID devices (like the Thunderstrike controller)
// expose a HID device interface at \\?\hid#... path. This is separate from
// the SPP COM port used for firmware flashing.
//
// Device info queries (VERSION, BOARD_INFO, MAC_ADDRESS, SET_TRANSPORT)
// must be sent through this HID device interface, not through the SPP COM.
//
// Protocol (same as USB HID / Android /dev/hidraw):
//   Request:  [0x04] [cmd_ordinal] [transaction_id] [data...] [zero-padded to 33]
//   Response: [0x03] [cmd_ordinal] [transaction_id] [response_data...]
package hid

import (
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
	procHidD_GetAttributes         = hidDll.NewProc("HidD_GetAttributes")
	procSetupDiGetClassDevsW      = setupAPI.NewProc("SetupDiGetClassDevsW")
	procSetupDiEnumDeviceInterfaces = setupAPI.NewProc("SetupDiEnumDeviceInterfaces")
	procSetupDiGetDeviceInterfaceDetailW = setupAPI.NewProc("SetupDiGetDeviceInterfaceDetailW")
	procSetupDiDestroyDeviceInfoList = setupAPI.NewProc("SetupDiDestroyDeviceInfoList")
	procCreateFile  = kernel32.NewProc("CreateFileW")
	procCloseHandle = kernel32.NewProc("CloseHandle")
	procWriteFile   = kernel32.NewProc("WriteFile")
	procReadFile    = kernel32.NewProc("ReadFile")
)

// hidGuid is the Windows GUID for HID devices.
var hidGuid = syscall.GUID{
	Data1: 0x4D1E55B2,
	Data2: 0xF16F,
	Data3: 0x11CF,
	Data4: [8]byte{0x88, 0xCB, 0x00, 0x11, 0x11, 0x00, 0x00, 0x30},
}

const (
	digcfDeviceInterface = 0x00000010
	digcfPresent         = 0x00000002
	genericRead          = 0x80000000
	genericWrite         = 0x40000000
	fileShareAll         = 0x00000007
	openExisting         = 3
)

// spDeviceInterfaceData matches SP_DEVICE_INTERFACE_DATA.
type spDeviceInterfaceData struct {
	cbSize             uint32
	interfaceClassGuid syscall.GUID
	flags              uint32
	reserved           [2]uint32
}

// hidAttributes matches HIDD_ATTRIBUTES.
type hidAttributes struct {
	Size          uint32
	VendorID      uint16
	ProductID     uint16
	VersionNumber uint16
}

// vidNvidia is the NVIDIA USB Vendor ID.
const vidNvidia uint16 = 0x0955

// pidThunderstrike is the Thunderstrike Product ID.
const pidThunderstrike uint16 = 0x7214

// WindowsHidClient communicates with the controller via the Windows HID
// device interface (\\?\hid#... path). It implements DeviceInterfacer.
type WindowsHidClient struct {
	handle         uintptr
	transactionID  byte
}

// FindThunderstrikeHidDevice searches for an NVIDIA Thunderstrike HID
// device and returns its device path. Returns empty string if not found.
func FindThunderstrikeHidDevice() (string, error) {
	hInfo, _, _ := procSetupDiGetClassDevsW.Call(
		uintptr(unsafe.Pointer(&hidGuid)),
		0, 0,
		digcfDeviceInterface|digcfPresent,
	)
	if hInfo == 0 || hInfo == uintptr(^uintptr(0)) {
		return "", fmt.Errorf("SetupDiGetClassDevs failed")
	}
	defer procSetupDiDestroyDeviceInfoList.Call(hInfo)

	index := uint32(0)
	for {
		var ifaceData spDeviceInterfaceData
		ifaceData.cbSize = uint32(unsafe.Sizeof(ifaceData))

		ret, _, _ := procSetupDiEnumDeviceInterfaces.Call(
			hInfo, 0,
			uintptr(unsafe.Pointer(&hidGuid)),
			uintptr(index),
			uintptr(unsafe.Pointer(&ifaceData)),
		)
		if ret == 0 {
			break
		}
		index++

		// Get detail size
		var detailSize uint32
		procSetupDiGetDeviceInterfaceDetailW.Call(
			hInfo,
			uintptr(unsafe.Pointer(&ifaceData)),
			0, 0,
			uintptr(unsafe.Pointer(&detailSize)),
			0,
		)
		if detailSize == 0 {
			continue
		}

		// Get detail data
		detailBuf := make([]byte, detailSize)
		*(*uint32)(unsafe.Pointer(&detailBuf[0])) = 8

		ret, _, _ = procSetupDiGetDeviceInterfaceDetailW.Call(
			hInfo,
			uintptr(unsafe.Pointer(&ifaceData)),
			uintptr(unsafe.Pointer(&detailBuf[0])),
			uintptr(detailSize),
			0, 0,
		)
		if ret == 0 {
			continue
		}

		devicePath := utf16BytesToString(detailBuf[4:])

		// Open and check attributes
		pathUTF16, _ := syscall.UTF16PtrFromString(devicePath)
		handle, _, _ := procCreateFile.Call(
			uintptr(unsafe.Pointer(pathUTF16)),
			genericRead|genericWrite,
			fileShareAll,
			0, openExisting, 0, 0,
		)
		invalidHandle := uintptr(^uintptr(0))
		if handle == 0 || handle == invalidHandle {
			continue
		}

		var attrs hidAttributes
		attrs.Size = uint32(unsafe.Sizeof(attrs))
		procHidD_GetAttributes.Call(handle, uintptr(unsafe.Pointer(&attrs)))
		procCloseHandle.Call(handle)

		if attrs.VendorID == vidNvidia && attrs.ProductID == pidThunderstrike {
			return devicePath, nil
		}
	}

	return "", nil
}

// OpenWindowsHidClient opens the Thunderstrike HID device for read/write.
func OpenWindowsHidClient() (*WindowsHidClient, error) {
	devicePath, err := FindThunderstrikeHidDevice()
	if err != nil {
		return nil, err
	}
	if devicePath == "" {
		return nil, fmt.Errorf("Thunderstrike HID device not found")
	}

	pathUTF16, _ := syscall.UTF16PtrFromString(devicePath)
	handle, _, _ := procCreateFile.Call(
		uintptr(unsafe.Pointer(pathUTF16)),
		genericRead|genericWrite,
		fileShareAll,
		0, openExisting, 0, 0,
	)
	invalidHandle := uintptr(^uintptr(0))
	if handle == 0 || handle == invalidHandle {
		return nil, fmt.Errorf("failed to open HID device: %s", devicePath)
	}

	return &WindowsHidClient{handle: handle}, nil
}

// SendCommand sends a HID report command and reads the matching response.
func (c *WindowsHidClient) SendCommand(cmd BtHidrawCommand, data []byte) ([]byte, error) {
	// Build 33-byte report
	report := c.buildRequestReport(cmd, data)
	txnID := report[2]

	// Write the report
	var written uint32
	ret, _, _ := procWriteFile.Call(
		c.handle,
		uintptr(unsafe.Pointer(&report[0])),
		uintptr(len(report)),
		uintptr(unsafe.Pointer(&written)),
		0,
	)
	if ret == 0 {
		return nil, fmt.Errorf("%s: WriteFile failed", cmd)
	}

	// Read response
	readBuf := make([]byte, 256)
	var bytesRead uint32
	ret, _, _ = procReadFile.Call(
		c.handle,
		uintptr(unsafe.Pointer(&readBuf[0])),
		uintptr(len(readBuf)),
		uintptr(unsafe.Pointer(&bytesRead)),
		0,
	)
	if ret == 0 || bytesRead == 0 {
		return nil, fmt.Errorf("%s: ReadFile timeout", cmd)
	}

	resp := readBuf[:bytesRead]
	// Verify response: [0x03][cmd][txn][data...]
	if len(resp) < 4 {
		return nil, fmt.Errorf("%s: response too short (%d bytes)", cmd, bytesRead)
	}
	if resp[0] != ResponsePrefix {
		return nil, fmt.Errorf("%s: bad response prefix 0x%02X", cmd, resp[0])
	}

	respCmd := BtHidrawCommand(resp[1])
	respTxn := resp[2]

	// Check for CMD_ERROR
	if byte(respCmd) == 0xFF && respTxn == txnID {
		return nil, ErrCmdError
	}

	// Check for matching response
	if respCmd != cmd || respTxn != txnID {
		return nil, fmt.Errorf("%s: mismatched response cmd=0x%02X txn=%d", cmd, respCmd, respTxn)
	}

	// Return response data (after 3-byte header)
	result := make([]byte, len(resp)-3)
	copy(result, resp[3:])
	return result, nil
}

// Close closes the HID device handle.
func (c *WindowsHidClient) Close() error {
	if c.handle != 0 {
		procCloseHandle.Call(c.handle)
		c.handle = 0
	}
	return nil
}

// buildRequestReport constructs the 33-byte request report buffer.
func (c *WindowsHidClient) buildRequestReport(cmd BtHidrawCommand, data []byte) []byte {
	buf := make([]byte, ReportLen)
	buf[0] = RequestPrefix
	buf[1] = byte(cmd)
	c.transactionID++
	buf[2] = c.transactionID

	maxDataLen := ReportLen - 3
	dataLen := len(data)
	if dataLen > maxDataLen {
		dataLen = maxDataLen
	}
	if dataLen > 0 {
		copy(buf[3:3+dataLen], data[:dataLen])
	}
	return buf
}

// utf16BytesToString converts a UTF-16LE byte slice to a Go string.
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
