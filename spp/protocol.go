// Package spp implements the Bluetooth SPP (Serial Port Profile) protocol
// used by NVIDIA Thunderstrike controller for firmware flashing.
//
// Protocol format:
//   Command (PC → controller): [1 byte CommandCode] [2 bytes LE length] [N bytes data]
//   Response (controller → PC): [1 byte CommandCode] [1 byte StatusCode] [1 byte payload_len] [N bytes payload]
package spp

// CommandCode represents SPP firmware flashing command codes.
// These are sent as the first byte of each SPP command packet.
type CommandCode byte

// SPP command codes (from BtUpdater$CommandCode enum ordinals).
const (
	CmdNop          CommandCode = 0 // No operation (handshake)
	CmdEraseSqif    CommandCode = 1 // Erase SQIF flash memory
	CmdWriteSqif    CommandCode = 2 // Write data to SQIF flash
	CmdValidateSqif CommandCode = 3 // Validate written SQIF data
	CmdNotSupported CommandCode = 4 // Not supported command
	CmdApplyOta     CommandCode = 5 // Apply OTA firmware update
)

// String returns a human-readable name for the command code.
func (c CommandCode) String() string {
	switch c {
	case CmdNop:
		return "NOP"
	case CmdEraseSqif:
		return "ERASE_SQIF"
	case CmdWriteSqif:
		return "WRITE_SQIF"
	case CmdValidateSqif:
		return "VALIDATE_SQIF"
	case CmdNotSupported:
		return "NOT_SUPPORTED"
	case CmdApplyOta:
		return "APPLY_OTA"
	default:
		return "UNKNOWN"
	}
}

// StatusCode represents the response status from the controller.
type StatusCode byte

// SPP status codes (from BtUpdater$StatusCode enum ordinals).
const (
	StatusCompletedOk     StatusCode = 0 // Command completed successfully
	StatusInProgress      StatusCode = 1 // Command in progress
	StatusFailed          StatusCode = 2 // Command failed
	StatusTimeout         StatusCode = 3 // Command timed out
	StatusOtherError      StatusCode = 4 // Other error
	StatusDiagInProgress  StatusCode = 5 // Diagnostics in progress
	StatusFailedLowBatt   StatusCode = 6 // Failed: low battery
	StatusFailedSignature StatusCode = 7 // Failed: signature verification
	StatusFailedFlashErr  StatusCode = 8 // Failed: flash error
)

// String returns a human-readable description of the status code.
func (s StatusCode) String() string {
	switch s {
	case StatusCompletedOk:
		return "COMPLETED_OK"
	case StatusInProgress:
		return "IN_PROGRESS"
	case StatusFailed:
		return "FAILED"
	case StatusTimeout:
		return "TIMEOUT"
	case StatusOtherError:
		return "OTHER_ERROR"
	case StatusDiagInProgress:
		return "DIAG_IN_PROGRESS"
	case StatusFailedLowBatt:
		return "FAILED_LOW_BATTERY"
	case StatusFailedSignature:
		return "FAILED_SIGNATURE"
	case StatusFailedFlashErr:
		return "FAILED_FLASH_ERROR"
	default:
		return "UNKNOWN"
	}
}

// IsOk returns true if the status indicates success.
func (s StatusCode) IsOk() bool {
	return s == StatusCompletedOk
}

// IsRetryable returns true if the status indicates a temporary state
// that can be retried (e.g. in-progress).
// Note: smali ReadResponse only skips STATUS_CMD_IN_PROGRESS.
func (s StatusCode) IsRetryable() bool {
	return s == StatusInProgress
}

// SppUuid is the standard Bluetooth SPP UUID used by the Thunderstrike controller.
const SppUuid = "00001101-0000-1000-8000-00805f9b34fb"

// MaxChunkSize is the maximum data chunk size for WRITE_SQIF commands (bytes).
const MaxChunkSize = 1024

// DefaultTimeouts for various operations (milliseconds).
const (
	TimeoutNopMs       = 5000  // NOP handshake timeout
	TimeoutEraseMs     = 20000 // Erase SQIF timeout
	TimeoutWriteMs     = 2000  // Per-chunk write ACK timeout
	TimeoutValidateMs  = 20000 // Validate SQIF timeout
	TimeoutApplyMs     = 30000 // Apply OTA timeout
	TimeoutDefaultMs   = 10000 // Default timeout
)

// UpgradeTimings holds the estimated timing parameters for firmware upgrade.
// Extracted from ThunderstrikeController.mUpgradeTimings in smali.
type UpgradeTimings struct {
	EraseMs       int // 0x960  = 2400ms  — SQIF erase time
	BytesPerSec   int // 0x6784 = 26484   — Estimated write speed
	ApplyDelayMs  int // 0x157C = 5500ms  — Apply OTA delay
	ReadyTimeMs   int // 0x251C = 9500ms  — Device ready wait time
	ReconnectMs   int // 0      — Reconnect delay (0 for non-locale)
}

// DefaultTimings returns the default upgrade timings for Thunderstrike controller.
func DefaultTimings() UpgradeTimings {
	return UpgradeTimings{
		EraseMs:      0x960,
		BytesPerSec:  0x6784,
		ApplyDelayMs: 0x157C,
		ReadyTimeMs:  0x251C,
		ReconnectMs:  0,
	}
}
