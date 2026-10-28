package core

// DeviceScanner 设备扫描接口（由 cmd 包实现）。
type DeviceScanner interface {
	// ScanDevices 扫描蓝牙 SPP 设备，返回设备列表。
	ScanDevices() ([]DeviceInfo, error)
	// QueryDeviceDetail 通过 HID 查询设备详细信息。
	QueryDeviceDetail(device DeviceInfo) (*DeviceDetail, error)
}

// FirmwareScanner 固件扫描接口（由 cmd 包实现）。
type FirmwareScanner interface {
	// ScanFirmwares 扫描 blkz/ 目录，返回固件列表。
	ScanFirmwares() ([]FirmwareInfo, error)
}

// Flasher 刷写执行接口（由 cmd 包实现）。
type Flasher interface {
	// ExecuteFlash 执行固件刷写，通过回调报告进度。
	ExecuteFlash(
		device DeviceInfo,
		firmware FirmwareInfo,
		stepCB func(step, total int, name string),
		progressCB func(written, total int),
	) (*FlashResultInfo, error)
}

// GamepadMonitor 手柄输入监听接口（由 cmd 包实现）。
type GamepadMonitor interface {
	// StartMonitoring 启动后台监听。
	// onInput 在手柄有按键/摇杆输入时调用；
	// onReady 在 HID 句柄成功打开（设备在线）时调用，可为 nil。
	StartMonitoring(onInput func(), onReady func())
	// StopMonitoring 停止后台监听。
	StopMonitoring()
}

// Services 汇总一套界面所需的全部业务能力。
type Services struct {
	Scanner   DeviceScanner
	Firmwares FirmwareScanner
	Flasher   Flasher
	Monitor   GamepadMonitor
}
