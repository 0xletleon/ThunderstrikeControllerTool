// Package hid defines the HID report protocol constants and command codes
// used by the NVIDIA Thunderstrike controller over Bluetooth SPP.
//
// HID command format (from Bt_Ops.requestData / onReport smali analysis):
//   Request:  [0x04] [cmd_ordinal] [transaction_id] [data...] [zero-padded]
//   Response: [0x03] [cmd_ordinal] [transaction_id] [response_data...]
package hid

import "fmt"

// BtHidrawCommand represents a HID command for the controller.
// The byte value is the ordinal of the Bt_Ops$BtHidrawCommand enum
// (extracted from the decompiled NvAccessories APK smali).
type BtHidrawCommand byte

// HID command codes — exact ordinal values from Bt_Ops$BtHidrawCommand.<clinit>.
const (
	CmdNop             BtHidrawCommand = 0x00 // No operation
	CmdVersion         BtHidrawCommand = 0x01 // Firmware version
	CmdPioState        BtHidrawCommand = 0x02 // PIO pin state
	CmdButtonState     BtHidrawCommand = 0x03 // Button state
	CmdAudioLoopback   BtHidrawCommand = 0x04 // Audio loopback test
	CmdShipMode        BtHidrawCommand = 0x05 // Ship mode
	CmdLedControl      BtHidrawCommand = 0x06 // LED control
	CmdBatteryState    BtHidrawCommand = 0x07 // Battery state/level
	CmdI2c             BtHidrawCommand = 0x08 // I2C read/write
	CmdSqif            BtHidrawCommand = 0x09 // SQIF flash access
	CmdPskey           BtHidrawCommand = 0x0A // PS key access
	CmdMacAddress      BtHidrawCommand = 0x0B // MAC address
	CmdVmmMsg          BtHidrawCommand = 0x0C // VM message
	CmdResetSource     BtHidrawCommand = 0x0D // Reset source
	CmdMic             BtHidrawCommand = 0x0E // Microphone
	CmdCypressOta      BtHidrawCommand = 0x0F // Cypress OTA
	CmdBoardInfo       BtHidrawCommand = 0x10 // Board information / serial
	CmdAnalogStatus    BtHidrawCommand = 0x11 // Analog stick status
	CmdHp              BtHidrawCommand = 0x12 // Headphone
	CmdBrightness      BtHidrawCommand = 0x13 // Brightness / debug level
	CmdLog             BtHidrawCommand = 0x14 // Log
	CmdButtonInject    BtHidrawCommand = 0x15 // Button inject
	CmdPairedMac       BtHidrawCommand = 0x16 // Paired MAC address
	CmdPairWhite       BtHidrawCommand = 0x17 // Pair whitelist
	CmdTouchDataEnable BtHidrawCommand = 0x18 // Touch data enable
	CmdTouchData       BtHidrawCommand = 0x19 // Touch data
	CmdSpiIds          BtHidrawCommand = 0x1A // SPI Flash IDs
	CmdAccelIntCountEn BtHidrawCommand = 0x1B // Accel interrupt count enable
	CmdAccelGetIntCnt  BtHidrawCommand = 0x1C // Accel get interrupt count
	CmdAccelGetAxis    BtHidrawCommand = 0x1D // Accel get axis data
	CmdAccelSelfTest   BtHidrawCommand = 0x1E // Accel self test
	CmdTestmode        BtHidrawCommand = 0x1F // Test mode
	CmdGetPors         BtHidrawCommand = 0x20 // Get power-on reset source
	CmdOtaInit         BtHidrawCommand = 0x21 // OTA init
	CmdOtaWrite        BtHidrawCommand = 0x22 // OTA write
	CmdOtaVerify       BtHidrawCommand = 0x23 // OTA verify
	CmdOtaCommit       BtHidrawCommand = 0x24 // OTA commit
	CmdBtTest          BtHidrawCommand = 0x25 // Bluetooth test
	CmdIrTest          BtHidrawCommand = 0x26 // IR test
	CmdGetIrParam      BtHidrawCommand = 0x27 // Get IR parameters
	CmdSetIrCmd        BtHidrawCommand = 0x28 // Set IR command
	CmdGetIrCmd        BtHidrawCommand = 0x29 // Get IR command
	CmdSetIrCmdFreq    BtHidrawCommand = 0x2A // Set IR command frequency
	CmdSetIrMacro      BtHidrawCommand = 0x2B // Set IR macro
	CmdGetIrMacro      BtHidrawCommand = 0x2C // Get IR macro
	CmdSetIrEvent      BtHidrawCommand = 0x2D // Set IR event
	CmdReadStat        BtHidrawCommand = 0x2E // Read statistics
	CmdReadyAccelReg   BtHidrawCommand = 0x2F // Read accelerometer register
	CmdSetFactoryPair  BtHidrawCommand = 0x30 // Set factory pair
	CmdTestmodeResult  BtHidrawCommand = 0x31 // Test mode result
	CmdJsRaw           BtHidrawCommand = 0x32 // Joystick raw data
	CmdJsCalibration   BtHidrawCommand = 0x33 // Joystick calibration
	CmdDeadzone        BtHidrawCommand = 0x34 // Deadzone
	CmdDumpLog         BtHidrawCommand = 0x35 // Dump log
	CmdReset           BtHidrawCommand = 0x36 // Reset
	CmdAudio           BtHidrawCommand = 0x37 // Audio
	CmdPairing         BtHidrawCommand = 0x38 // Pairing
	CmdHaptics         BtHidrawCommand = 0x39 // Haptics
	CmdChargeState     BtHidrawCommand = 0x3A // Charge state
	CmdDebugLevel      BtHidrawCommand = 0x3B // Debug level
	CmdIo              BtHidrawCommand = 0x3C // I/O
	CmdSysmsg          BtHidrawCommand = 0x3D // System message
	CmdOta             BtHidrawCommand = 0x3E // OTA
	CmdTsIrTest        BtHidrawCommand = 0x3F // Thunderstrike IR test
	CmdEcho            BtHidrawCommand = 0x40 // Echo
	CmdFactoryCharge   BtHidrawCommand = 0x41 // Factory charge
	CmdBundleMac       BtHidrawCommand = 0x42 // Bundle MAC
	CmdTickle          BtHidrawCommand = 0x43 // Tickle
	CmdAccelerometer   BtHidrawCommand = 0x44 // Accelerometer
	CmdIdentify        BtHidrawCommand = 0x45 // Identify
	CmdNickname        BtHidrawCommand = 0x46 // Nickname
	CmdGetConnMac      BtHidrawCommand = 0x47 // Get connected MAC
	CmdLinkAction      BtHidrawCommand = 0x48 // Link action
	CmdSetTransport    BtHidrawCommand = 0x49 // Set transport
	CmdFactoryReset    BtHidrawCommand = 0x4A // Factory reset
	// CMD_75 through CMD_97 are unnamed reserved commands (0x4B–0x61)
	CmdPhoneHome       BtHidrawCommand = 0x62 // Phone home
	CmdWakeupSource    BtHidrawCommand = 0x63 // Wakeup source
	CmdQueryDevice     BtHidrawCommand = 0x64 // Query device
	CmdSetLinkPolicy   BtHidrawCommand = 0x65 // Set link policy
	CmdGetLinkPolicy   BtHidrawCommand = 0x66 // Get link policy
	// CMD_103 through CMD_116 are unnamed reserved commands (0x67–0x74)
	CmdFeature         BtHidrawCommand = 0x52 // Feature bits
	CmdHotword         BtHidrawCommand = 0x54 // Hotword
	CmdBacklight       BtHidrawCommand = 0x75 // Backlight
	// CMD_118 through CMD_155 are unnamed reserved commands (0x76–0x9B)
	CmdHotwordLanguage BtHidrawCommand = 0x98 // Hotword language
	// CMD_155 through CMD_157 are unnamed reserved commands (0x9B–0x9D)
	CmdError           BtHidrawCommand = 0x9E // Error
	CmdMax             BtHidrawCommand = 0x9F // Max sentinel
)

// commandNames maps command ordinals to their smali enum names.
var commandNames = map[BtHidrawCommand]string{
	CmdNop: "NOP", CmdVersion: "VERSION", CmdPioState: "PIO_STATE",
	CmdButtonState: "BUTTON_STATE", CmdAudioLoopback: "AUDIO_LOOPBACK",
	CmdShipMode: "SHIP_MODE", CmdLedControl: "LED_CONTROL",
	CmdBatteryState: "BATTERY_STATE", CmdI2c: "I2C", CmdSqif: "SQIF",
	CmdPskey: "PSKEY", CmdMacAddress: "MAC_ADDRESS", CmdVmmMsg: "VMM_MSG",
	CmdResetSource: "RESET_SOURCE", CmdMic: "MIC", CmdCypressOta: "CYPRESS_OTA",
	CmdBoardInfo: "BOARD_INFO", CmdAnalogStatus: "ANALOG_STATUS",
	CmdHp: "HP", CmdBrightness: "BRIGHTNESS", CmdLog: "LOG",
	CmdButtonInject: "BUTTON_INJECT", CmdPairedMac: "PAIRED_MAC",
	CmdPairWhite: "PAIR_WHITE", CmdTouchDataEnable: "TOUCH_DATA_ENABLE",
	CmdTouchData: "TOUCH_DATA", CmdSpiIds: "SPI_IDS",
	CmdAccelIntCountEn: "ACCEL_INT_COUNT_ENABLE",
	CmdAccelGetIntCnt:  "ACCEL_GET_INT_COUNT",
	CmdAccelGetAxis:    "ACCEL_GET_AXIS_DATA",
	CmdAccelSelfTest:   "ACCEL_SELF_TEST", CmdTestmode: "TESTMODE",
	CmdGetPors: "GET_PORS", CmdOtaInit: "OTA_INIT", CmdOtaWrite: "OTA_WRITE",
	CmdOtaVerify: "OTA_VERIFY", CmdOtaCommit: "OTA_COMMIT",
	CmdBtTest: "BT_TEST", CmdIrTest: "IR_TEST", CmdGetIrParam: "GET_IR_PARAM",
	CmdSetIrCmd: "SET_IR_CMD", CmdGetIrCmd: "GET_IR_CMD",
	CmdSetIrCmdFreq: "SET_IR_CMD_FREQ", CmdSetIrMacro: "SET_IR_MACRO",
	CmdGetIrMacro: "GET_IR_MACRO", CmdSetIrEvent: "SET_IR_EVENT",
	CmdReadStat: "READ_STAT", CmdReadyAccelReg: "READY_ACCEL_REG",
	CmdSetFactoryPair: "SET_FACTORY_PAIR",
	CmdTestmodeResult:  "TESTMODE_RESULT",
	CmdJsRaw: "JS_RAW", CmdJsCalibration: "JS_CALIBRATION",
	CmdDeadzone: "DEADZONE", CmdDumpLog: "DUMP_LOG", CmdReset: "RESET",
	CmdAudio: "AUDIO", CmdPairing: "PAIRING", CmdHaptics: "HAPTICS",
	CmdChargeState: "CHARGE_STATE", CmdDebugLevel: "DEBUG_LEVEL",
	CmdIo: "IO", CmdSysmsg: "SYSMSG", CmdOta: "OTA",
	CmdTsIrTest: "TS_IR_TEST", CmdEcho: "ECHO",
	CmdFactoryCharge: "FACTORY_CHARGE", CmdBundleMac: "BUNDLE_MAC",
	CmdTickle: "TICKLE", CmdAccelerometer: "ACCELEROMETER",
	CmdIdentify: "IDENTIFY", CmdNickname: "NICKNAME",
	CmdGetConnMac: "GET_CONN_MAC", CmdLinkAction: "LINK_ACTION",
	CmdSetTransport: "SET_TRANSPORT", CmdFactoryReset: "FACTORY_RESET",
	CmdPhoneHome: "PHONE_HOME", CmdWakeupSource: "WAKEUP_SOURCE",
	CmdQueryDevice: "QUERY_DEVICE", CmdSetLinkPolicy: "SET_LINKPOLICY",
	CmdGetLinkPolicy: "GET_LINKPOLICY", CmdFeature: "FEATURE",
	CmdHotword: "HOTWORD", CmdBacklight: "BACKLIGHT",
	CmdHotwordLanguage: "HOTWORD_LANGUAGE", CmdError: "ERROR",
	CmdMax: "MAX",
}

// String returns a human-readable name for the command.
func (c BtHidrawCommand) String() string {
	if name, ok := commandNames[c]; ok {
		return name
	}
	return fmt.Sprintf("CMD_%d", byte(c))
}

// Protocol constants (from DevHidraw.smali and Bt_Ops.smali analysis).

// Request/Response prefixes — the first byte of every HID report.
//   Request:  [0x04] [cmd_ordinal] [transaction_id] [data...]
//   Response: [0x03] [cmd_ordinal] [transaction_id] [response_data...]
const (
	RequestPrefix  = 0x04 // Outgoing report prefix (host → device)
	ResponsePrefix = 0x03 // Incoming report prefix (device → host)
)

// DefaultReportLength is the HID data report length for Thunderstrike.
// From ThunderstrikeController.getHidrawReportLength() = 0x1E = 30 bytes.
const DefaultReportLength = 30

// MaxResponseSize is the maximum HID response size.
// From DevHidraw.MAX_RESPONSE_SIZE = 0x41 = 65 bytes.
const MaxResponseSize = 65

// ErrCmdError is returned when the device responds with CMD_ERROR (0xFF),
// indicating the command is not supported by the firmware.
var ErrCmdError = fmt.Errorf("device returned CMD_ERROR (0xFF): command not supported")

// FormatMacAddress formats a 6-byte MAC address as XX:XX:XX:XX:XX:XX.
func FormatMacAddress(b []byte) string {
	if len(b) < 6 {
		return ""
	}
	return fmt.Sprintf("%02X:%02X:%02X:%02X:%02X:%02X",
		b[0], b[1], b[2], b[3], b[4], b[5])
}
