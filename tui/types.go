// Package tui — TUI 与业务逻辑之间的数据传递类型。
//
// 定义设备、固件、刷写结果等数据结构，
// 将 TUI 层与 cmd 包解耦，避免循环依赖。
package tui

// DeviceInfo 表示扫描发现的蓝牙设备。
type DeviceInfo struct {
	Index      int    // 序号
	Name       string // 显示名称
	Port       string // COM 端口
	MAC        string // MAC 地址 (XX:XX:XX:XX:XX:XX)
	DeviceName string // 设备名称 (如 "NVIDIA Controller")
	FwVersion  string // 当前固件版本 (如 "1.33")
}

// DeviceDetail 表示通过 HID 查询到的设备详细信息。
type DeviceDetail struct {
	DeviceName   string
	MAC          string
	FwVersion    string // 如 "1.33"
	FwVersionHex string // 如 "0x010E"
	CsrVersion   string // 蓝牙芯片版本
	HotwordVer   string // 热词引擎版本
	Serial       string
	MacHID       string // HID 查询到的 MAC

	// 电池信息（来自 CMD_BATTERY_STATE 响应）
	BatteryPct   int  // 电量百分比 0-100
	ReservePower bool // 是否进入储备电量
}

// FirmwareInfo 表示一个可选固件包的信息。
type FirmwareInfo struct {
	Index     int    // 序号
	Path      string // 文件完整路径
	Name      string // 文件名
	Version   string // 版本号 (如 "1.14")
	VersionHex string // 十六进制版本 (如 "010E")
	Checksum  string // MD5 校验状态: "match", "mismatch", "unknown", "error"
	CsActual  string // 实际 MD5
	CsExpected string // 期望 MD5
	IsLocale  bool   // 是否多语言固件
	Languages []string // 语言列表
	LanguageIndex int // 选中的语言索引（多语言固件使用，默认 0）
	OtaSize   int    // 固件大小(bytes)
}

// FlashStep 表示刷写流程的一个步骤。
type FlashStep struct {
	Index  int    // 步骤序号 (1-based)
	Total  int    // 总步骤数
	Name   string // 步骤名称
	Done   bool   // 是否完成
}

// FlashProgressInfo 表示刷写进度信息。
type FlashProgressInfo struct {
	BytesWritten int
	TotalBytes   int
	Percent      int
}

// FlashResultInfo 表示刷写结果。
type FlashResultInfo struct {
	Success   bool
	Error     string
	Elapsed   string // 耗时
	BytesWrit int
	FromVer   string
	ToVer     string
	LogPath   string
}
