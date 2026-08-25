// Package cmd — Windows Bluetooth SPP COM port discovery via WMI.
//
// Instead of opening each COM port and sending NOP probes (which is slow
// and can block for seconds per port), we query WMI for Win32_SerialPort.
// Bluetooth SPP ports have a PNPDeviceID starting with BTHENUM\.
//
// PNPDeviceID structure for BTHENUM ports:
//
//   BTHENUM\{GUID}_VID&XXXX_PID&YYYY\7&NNNNNNN&N&MMMMMMMMMMM_C00000000
//   └─ SPP UUID ┘ └─ remote ─┘   └─ device MAC (12 hex) ┘
//
//   BTHENUM\{GUID}_LOCALMFG&NNNN\7&NNNNNNN&N&000000000000_00000004
//   └─ SPP UUID ┘ └─ local  ─┘   └─ all zeros = no remote device ─┘
//
// Outgoing ports (with VID/PID) connect to a remote Bluetooth device.
// Incoming ports (with LOCALMFG) are local service listeners — useless
// for communicating with a controller. We only keep outgoing ports.
//
// The remote device MAC is embedded in the last path segment of the
// PNPDeviceID, formatted as a 12-hex-digit run before "_C00000000".
package cmd

import (
	"encoding/hex"
	"fmt"
	"os/exec"
	"strings"
)

// Known Bluetooth device Vendor/Product IDs.
const (
	vidNvidia    = "00020955"
	pidThunderstrike = "7214"
)

// btComPort represents a Bluetooth SPP outgoing COM port discovered via WMI.
type btComPort struct {
	comPort    string // e.g. "COM3"
	mac        string // 12-digit hex MAC, e.g. "00044B9204F4"
	macColon   string // formatted MAC, e.g. "00:04:4B:92:04:F4"
	vid        string // vendor ID, e.g. "00020955" (NVIDIA)
	pid        string // product ID, e.g. "7214" (Thunderstrike)
	btName     string // Bluetooth friendly name, e.g. "NVIDIA Controller v01.04"
}

// deviceName returns the Bluetooth friendly name from Windows, or
// falls back to VID/PID-based name if not available.
func (b btComPort) deviceName() string {
	if b.btName != "" {
		return b.btName
	}
	if b.vid == vidNvidia && b.pid == pidThunderstrike {
		return "NVIDIA Controller"
	}
	return "蓝牙设备"
}

// queryBtDeviceName queries Windows for the Bluetooth device friendly name
// by MAC address (colon-separated format, e.g. "00:04:4B:92:04:F4").
// Returns the friendly name or empty string if not found.
func queryBtDeviceName(macColon string) string {
	// Query the Bluetooth device registry. Paired Bluetooth device names are
	// stored under BTHPORT\Parameters\Devices\<MAC_without_colons>.
	// The Name value is REG_BINARY (ASCII bytes as hex).
	macNoColon := strings.ReplaceAll(macColon, ":", "")
	regPath := fmt.Sprintf(
		`HKLM\SYSTEM\CurrentControlSet\Services\BTHPORT\Parameters\Devices\%s`,
		macNoColon,
	)

	cmd := exec.Command("reg", "query", regPath, "/v", "Name")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	// "reg query" output format:
	//       Name    REG_BINARY    4E5649444941...0000
	// Parse the hex bytes after "REG_BINARY" and decode as UTF-16LE.
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "REG_BINARY") {
			parts := strings.SplitN(line, "REG_BINARY", 2)
			if len(parts) == 2 {
				hexStr := strings.ReplaceAll(strings.TrimSpace(parts[1]), " ", "")
				return decodeRegistryNameHex(hexStr)
			}
		}
		// Fallback: some systems may use REG_SZ
		if strings.Contains(line, "REG_SZ") {
			parts := strings.SplitN(line, "REG_SZ", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}

// decodeRegistryNameHex decodes a hex string from REG_BINARY Name value
// to a Go string. The hex bytes are typically ASCII-encoded with a null
// terminator (0x00). Reads 2 hex chars per byte, stops at null byte.
func decodeRegistryNameHex(hexStr string) string {
	if len(hexStr)%2 != 0 {
		return ""
	}
	var runes []rune
	for i := 0; i+1 < len(hexStr); i += 2 {
		var b byte
		fmt.Sscanf(hexStr[i:i+2], "%02X", &b)
		if b == 0 {
			break // null terminator
		}
		runes = append(runes, rune(b))
	}
	return string(runes)
}

// discoverBtComPorts queries WMI for Bluetooth SPP serial ports.
// Only returns outgoing ports (with VID/PID and a non-zero device MAC).
// Incoming/local listener ports (LOCALMFG) are filtered out.
func discoverBtComPorts() ([]btComPort, error) {
	// PowerShell command: query Win32_SerialPort for DeviceID and PNPDeviceID.
	const psScript = `Get-WmiObject Win32_SerialPort | ForEach-Object { "$($_.DeviceID)|$($_.PNPDeviceID)" }`

	cmd := exec.Command("powershell", "-NoProfile", "-Command", psScript)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("WMI query: %w", err)
	}

	var ports []btComPort
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Split "DeviceID|PNPDeviceID"
		parts := strings.SplitN(line, "|", 2)
		if len(parts) < 2 {
			continue
		}
		deviceID := strings.TrimSpace(parts[0])
		pnpDeviceID := strings.TrimSpace(parts[1])

		// Only Bluetooth SPP ports have BTHENUM prefix.
		if !strings.HasPrefix(strings.ToUpper(pnpDeviceID), "BTHENUM\\") {
			continue
		}

		// Only outgoing ports have VID& — incoming ports have LOCALMFG&.
		if !strings.Contains(strings.ToUpper(pnpDeviceID), "VID&") {
			continue
		}

		// Extract VID/PID and MAC from the PNPDeviceID.
		vid, pid := extractVidPid(pnpDeviceID)
		mac := extractBtMac(pnpDeviceID)
		if mac == "" || isAllZeros(mac) {
			continue
		}

		macFmt := formatMacColon(mac)

		// Query Bluetooth friendly name from Windows
		btName := queryBtDeviceName(macFmt)

		ports = append(ports, btComPort{
			comPort:    deviceID,
			mac:        mac,
			macColon:   macFmt,
			vid:        vid,
			pid:        pid,
			btName:     btName,
		})
	}

	return ports, nil
}

// extractVidPid extracts the VID and PID values from a BTHENUM PNPDeviceID.
//
// Example: BTHENUM\{...}_VID&00020955_PID&7214\...
//   → VID = "00020955", PID = "7214"
func extractVidPid(pnpDeviceID string) (vid, pid string) {
	upper := strings.ToUpper(pnpDeviceID)

	if idx := strings.Index(upper, "VID&"); idx >= 0 {
		start := idx + len("VID&")
		vid = extractHexRun(pnpDeviceID[start:])
	}
	if idx := strings.Index(upper, "PID&"); idx >= 0 {
		start := idx + len("PID&")
		pid = extractHexRun(pnpDeviceID[start:])
	}
	return vid, pid
}

// extractBtMac extracts the 12-digit hex MAC from a BTHENUM PNPDeviceID.
//
// The MAC is in the last path segment (after the final '\'), formatted as
// a 12-hex-digit run followed by "_C00000000". We skip the first segment
// (the GUID) entirely.
//
// Example: BTHENUM\{GUID}_VID&..._PID&...\7&...&00044B9204F4_C00000000
//                                              └── MAC ──┘
func extractBtMac(pnpDeviceID string) string {
	// Get the last segment after the final '\'.
	idx := strings.LastIndex(pnpDeviceID, "\\")
	if idx < 0 {
		return ""
	}
	lastSegment := pnpDeviceID[idx+1:]

	// In the last segment, find a 12-hex-digit run.
	// Format: "7&A895487&0&00044B9204F4_C00000000"
	// The MAC is the last 12-hex run before "_C...".
	// Split by '&' or '_' and find the 12-char hex token.
	tokens := strings.FieldsFunc(lastSegment, func(c rune) bool {
		return c == '&' || c == '_'
	})
	for _, t := range tokens {
		if len(t) == 12 && isAllHex(t) {
			return strings.ToUpper(t)
		}
	}

	// Fallback: scan for any 12-char hex run in the last segment.
	return extractHexRun12(lastSegment)
}

// extractHexRun extracts the first run of hex digits from s.
func extractHexRun(s string) string {
	start := -1
	for i, c := range s {
		if isHexChar(byte(c)) {
			if start < 0 {
				start = i
			}
		} else {
			if start >= 0 {
				return s[start:i]
			}
		}
	}
	if start >= 0 {
		return s[start:]
	}
	return ""
}

// extractHexRun12 finds the first 12-char hex run in s.
func extractHexRun12(s string) string {
	start := -1
	count := 0
	for i, c := range s {
		if isHexChar(byte(c)) {
			if start < 0 {
				start = i
			}
			count++
			if count == 12 {
				return strings.ToUpper(s[start : start+12])
			}
		} else {
			start = -1
			count = 0
		}
	}
	return ""
}

// isAllHex returns true if s consists entirely of hex digits.
func isAllHex(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, c := range s {
		if !isHexChar(byte(c)) {
			return false
		}
	}
	return true
}

// isHexChar returns true if c is a hex digit (0-9, A-F, a-f).
func isHexChar(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'A' && c <= 'F') || (c >= 'a' && c <= 'f')
}

// isAllZeros returns true if the hex string is all zeros.
func isAllZeros(mac string) bool {
	for _, c := range mac {
		if c != '0' {
			return false
		}
	}
	return true
}

// formatMacColon converts "00044B9204F4" to "00:04:4B:92:04:F4".
func formatMacColon(mac12 string) string {
	if len(mac12) != 12 {
		return mac12
	}
	b, err := hex.DecodeString(mac12)
	if err != nil {
		return mac12
	}
	return fmt.Sprintf("%02X:%02X:%02X:%02X:%02X:%02X",
		b[0], b[1], b[2], b[3], b[4], b[5])
}
