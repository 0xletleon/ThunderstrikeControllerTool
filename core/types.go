// Package core — 业务层与 UI 层共用的数据类型与接口定义。
//
// 设备扫描/查询、固件扫描、刷写执行由 cmd 包实现这里的接口，
// 终端界面（tui 包）与原生 GUI（gui 包）消费同一套接口与类型，
// 使两套界面互不依赖，也不会与业务层形成循环依赖。
package core

// DeviceInfo 表示扫描发现的蓝牙设备。
type DeviceInfo struct {
	Index      int    // 序号
	Name       string // 显示名称
	Port       string // COM 端口
	MAC        string // MAC 地址 (XX:XX:XX:XX:XX:XX)
	DeviceName string // 设备名称 (如 "NVIDIA Controller")
	FwVersion  string // 当前固件版本 (如 "1.33")，空表示设备休眠
}

// Online 返回设备是否处于在线状态（能查到固件版本即为在线）。
func (d DeviceInfo) Online() bool { return d.FwVersion != "" }

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
	Index         int      // 序号
	Path          string   // 文件完整路径
	Name          string   // 文件名
	Version       string   // 版本号 (如 "1.14")
	VersionHex    string   // 十六进制版本 (如 "010E")
	Checksum      string   // MD5 校验状态: "match", "mismatch", "unknown", "error"
	CsActual      string   // 实际 MD5
	CsExpected    string   // 期望 MD5
	IsLocale      bool     // 是否多语言固件
	Languages     []string // 语言列表
	LanguageIndex int      // 选中的语言索引（多语言固件使用，默认 0）
	OtaSize       int      // 固件大小(bytes)
}

// FlashStep 表示刷写流程的一个步骤。
type FlashStep struct {
	Index int    // 步骤序号 (1-based)
	Total int    // 总步骤数
	Name  string // 步骤名称
	Done  bool   // 是否完成
}

// FlashProgressInfo 表示刷写进度信息。
type FlashProgressInfo struct {
	BytesWritten int
	TotalBytes   int
	Percent      int
}

// Ratio 返回 0–1 的进度比例，供进度条使用。
func (p FlashProgressInfo) Ratio() float64 {
	if p.TotalBytes <= 0 {
		return 0
	}
	return float64(p.BytesWritten) / float64(p.TotalBytes)
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

// flashStepNames 是刷写流程的标准步骤名称，顺序即执行顺序。
var flashStepNames = []string{"连接握手", "擦除闪存", "写入数据", "校验数据", "应用固件"}

// NewFlashSteps 返回全部未完成的刷写步骤，供界面在刷写开始前初始化。
func NewFlashSteps() []FlashStep {
	steps := make([]FlashStep, len(flashStepNames))
	for i, name := range flashStepNames {
		steps[i] = FlashStep{Index: i + 1, Total: len(flashStepNames), Name: name}
	}
	return steps
}

// languageNames 是 .blkz 清单中出现过的语言代码到显示名称的映射。
var languageNames = map[string]string{
	"de_de": "Deutsch (德语)",
	"fr_fr": "Français (法语)",
	"es_es": "Español (西班牙语)",
	"it_it": "Italiano (意大利语)",
	"en_us": "English (英语)",
	"en_gb": "English UK (英式英语)",
	"pt_br": "Português (葡萄牙语)",
	"nl_nl": "Nederlands (荷兰语)",
	"pl_pl": "Polski (波兰语)",
	"ru_ru": "Русский (俄语)",
	"ja_jp": "日本語 (日语)",
	"ko_kr": "한국어 (韩语)",
	"zh_cn": "简体中文 (中文)",
	"zh_tw": "繁體中文 (中文)",
	"tr_tr": "Türkçe (土耳其语)",
	"sv_se": "Svenska (瑞典语)",
	"da_dk": "Dansk (丹麦语)",
	"fi_fi": "Suomi (芬兰语)",
	"nb_no": "Norsk (挪威语)",
}

// LanguageDisplayName 将固件语言代码转为可读的显示名称。
func LanguageDisplayName(code string) string {
	if name, ok := languageNames[code]; ok {
		return name
	}
	if code == "" {
		return "默认 (Standard)"
	}
	return code
}
