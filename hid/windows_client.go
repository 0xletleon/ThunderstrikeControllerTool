package hid

import (
	"fmt"
	"syscall"
	"time"
	"unsafe"
)

var (
	hidDll    = syscall.NewLazyDLL("hid.dll")
	setupAPI  = syscall.NewLazyDLL("setupapi.dll")
	kernel32  = syscall.NewLazyDLL("kernel32.dll")
)

var (
	procHidD_GetAttributes              = hidDll.NewProc("HidD_GetAttributes")
	procSetupDiGetClassDevsW            = setupAPI.NewProc("SetupDiGetClassDevsW")
	procSetupDiEnumDeviceInterfaces     = setupAPI.NewProc("SetupDiEnumDeviceInterfaces")
	procSetupDiGetDeviceInterfaceDetailW = setupAPI.NewProc("SetupDiGetDeviceInterfaceDetailW")
	procSetupDiDestroyDeviceInfoList    = setupAPI.NewProc("SetupDiDestroyDeviceInfoList")
	procCreateFile                      = kernel32.NewProc("CreateFileW")
	procCloseHandle                     = kernel32.NewProc("CloseHandle")
	procWriteFile                       = kernel32.NewProc("WriteFile")
	procReadFile                        = kernel32.NewProc("ReadFile")
	procCancelIoEx                      = kernel32.NewProc("CancelIoEx")
)

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
	hidReadTimeout       = 5 * time.Second
)

type spDeviceInterfaceData struct {
	cbSize             uint32
	interfaceClassGuid syscall.GUID
	flags              uint32
	reserved           [2]uint32
}

type hidAttributes struct {
	Size          uint32
	VendorID      uint16
	ProductID     uint16
	VersionNumber uint16
}

const vidNvidia uint16 = 0x0955
const pidThunderstrike uint16 = 0x7214

// WindowsHidClient communicates with the controller via Windows HID API.
type WindowsHidClient struct {
	handle        uintptr
	transactionID byte
}

// FindThunderstrikeHidDevice searches for a Thunderstrike HID device.
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

		detailBuf := make([]byte, detailSize)
		var detailData struct {
			cbSize     uint32
			devicePath [1]uint16
		}
		cbSize := uint32(unsafe.Sizeof(detailData))
		*(*uint32)(unsafe.Pointer(&detailBuf[0])) = cbSize

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
// Uses CancelIoEx to enforce a timeout on the blocking ReadFile call.
func (c *WindowsHidClient) SendCommand(cmd BtHidrawCommand, data []byte) ([]byte, error) {
	report := c.buildRequestReport(cmd, data)
	txnID := report[2]

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

	type readResult struct {
		data []byte
		n    uint32
	}
	ch := make(chan readResult, 1)

	go func() {
		readBuf := make([]byte, 256)
		var bytesRead uint32
		procReadFile.Call(
			c.handle,
			uintptr(unsafe.Pointer(&readBuf[0])),
			uintptr(len(readBuf)),
			uintptr(unsafe.Pointer(&bytesRead)),
			0,
		)
		ch <- readResult{readBuf, bytesRead}
	}()

	var respData []byte
	var bytesRead uint32

	select {
	case r := <-ch:
		respData = r.data
		bytesRead = r.n
	case <-time.After(hidReadTimeout):
		procCancelIoEx.Call(c.handle, 0)
		return nil, fmt.Errorf("%s: ReadFile timeout (%v)", cmd, hidReadTimeout)
	}

	if bytesRead == 0 {
		return nil, fmt.Errorf("%s: ReadFile timeout", cmd)
	}

	resp := respData[:bytesRead]
	if len(resp) < 4 {
		return nil, fmt.Errorf("%s: response too short (%d bytes)", cmd, bytesRead)
	}
	if resp[0] != ResponsePrefix {
		return nil, fmt.Errorf("%s: bad response prefix 0x%02X", cmd, resp[0])
	}

	respCmd := BtHidrawCommand(resp[1])
	respTxn := resp[2]

	if byte(respCmd) == 0xFF && respTxn == txnID {
		return nil, ErrCmdError
	}

	if respCmd != cmd || respTxn != txnID {
		return nil, fmt.Errorf("%s: mismatched response cmd=0x%02X txn=%d", cmd, respCmd, respTxn)
	}

	result := make([]byte, len(resp)-3)
	copy(result, resp[3:])
	return result, nil
}

// GetBatteryLevel queries the controller's battery level via HID.
// The response format from the controller firmware varies:
// some versions return a simple 0-100 percentage byte, others return
// a raw ADC value. Returns the percentage if the value looks valid
// (0-100), otherwise returns the raw value with a note.
func (c *WindowsHidClient) GetBatteryLevel() (int, error) {
	data, err := c.SendCommand(CmdBatteryState, nil)
	if err != nil {
		return 0, fmt.Errorf("battery query: %w", err)
	}
	if len(data) < 1 {
		return 0, fmt.Errorf("battery: empty response")
	}

	raw := int(data[0])

	// If the value is in a reasonable 0-100 range, treat as percentage.
	// Values above 100 are likely raw ADC readings (observed 216 / 0xD8
	// on some firmware versions) and should not be shown as percentage.
	if raw >= 0 && raw <= 100 {
		return raw, nil
	}

	// Raw value outside 0-100 — likely an ADC reading. Return the raw
	// value so the caller can display it appropriately.
	return raw, fmt.Errorf("raw ADC value: %d (not a percentage)", raw)
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