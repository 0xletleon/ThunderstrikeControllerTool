// Package tui — 主 TUI 状态机。
//
// 使用 Bubble Tea 的 Elm 架构 (Model-View-Update) 统一管理
// 设备列表 → 设备信息 → 固件列表 → 刷写进度 的页面切换。
//
// 通过 DeviceScanner 和 FirmwareScanner 接口与业务逻辑交互，
// 避免与 cmd 包产生循环依赖。
package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// TUIState 表示当前所在的视图状态。
type TUIState int

const (
	StateScanning      TUIState = iota // 正在扫描设备
	StateDeviceList                    // 设备列表
	StateDeviceInfo                    // 设备信息
	StateFirmwareList                  // 固件列表
	StateLanguageSelect                // 多语言固件选择语言
	StateFlashing                      // 刷写中
	StateFlashDone                     // 刷写完成
)

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

// Model 是 Bubble Tea 主模型，管理所有子视图和状态切换。
type Model struct {
	state        TUIState
	scanner      DeviceScanner
	fwScanner    FirmwareScanner
	flasher      Flasher

	// 子视图
	deviceList  DeviceListModel
	deviceInfo  DeviceInfoModel
	fwList      FirmwareListModel
	langSelect  LanguageSelectModel
	flashProg   FlashProgressModel

	// 当前选中
	selectedDevice   *DeviceInfo
	selectedFirmware *FirmwareInfo

	// 错误
	errMsg string

	width  int
	height int
}

// NewModel 创建 TUI 主模型。
func NewModel(scanner DeviceScanner, fwScanner FirmwareScanner, flasher Flasher) Model {
	return Model{
		state:     StateScanning,
		scanner:   scanner,
		fwScanner: fwScanner,
		flasher:   flasher,
		flashProg: NewFlashProgressModel(),
	}
}

// Init 初始化 TUI，启动设备扫描。
func (m Model) Init() tea.Cmd {
	return scanDevicesCmd(m.scanner)
}

// Update 处理所有 TUI 消息，协调子视图状态切换。
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// 只更新当前活跃视图
		return m.handleCurrentViewResize(msg)

	case tea.KeyMsg:
		// 全局快捷键
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}

	// 根据当前状态分发消息到对应子视图
	switch m.state {
	case StateScanning, StateDeviceList:
		return m.handleDeviceListState(msg)

	case StateDeviceInfo:
		return m.handleDeviceInfoState(msg)

	case StateFirmwareList:
		return m.handleFirmwareListState(msg)

	case StateLanguageSelect:
		return m.handleLanguageSelectState(msg)

	case StateFlashing, StateFlashDone:
		return m.handleFlashingState(msg)
	}

	return m, nil
}

// handleDeviceListState 处理设备列表状态的消息。
func (m Model) handleDeviceListState(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case DevicesScannedMsg:
		if msg.Error != "" {
			m.errMsg = msg.Error
			m.deviceList.errMsg = msg.Error
		}
		if len(msg.Devices) > 0 {
			m.deviceList = NewDeviceListModel(msg.Devices)
		} else if msg.Error == "" {
			m.deviceList = NewDeviceListModel(nil)
		}
		m.state = StateDeviceList
		m.deviceList.scanning = false
		return m, nil

	case RefreshDevicesMsg:
		m.state = StateScanning
		m.deviceList.scanning = true
		return m, scanDevicesCmd(m.scanner)

	case DeviceSelectedMsg:
		// 用户选择了设备，进入设备信息页
		dev := msg.Device
		m.selectedDevice = &dev
		m.deviceInfo = NewDeviceInfoModel(dev)
		m.state = StateDeviceInfo
		// 启动 HID 查询
		return m, queryDeviceDetailCmd(m.scanner, dev)

	case tea.KeyMsg:
		newList, cmd := m.deviceList.Update(msg)
		m.deviceList = newList

		// 检查是否退出
		if newList.IsQuitting() {
			return m, tea.Quit
		}

		return m, cmd
	}

	// 其他消息传递给子视图
	newList, cmd := m.deviceList.Update(msg)
	m.deviceList = newList
	return m, cmd
}

// handleDeviceInfoState 处理设备信息状态的消息。
func (m Model) handleDeviceInfoState(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case DeviceDetailReadyMsg:
		if msg.Detail != nil {
			m.deviceInfo.SetDetail(msg.Detail)
		}
		if msg.Error != "" {
			m.errMsg = msg.Error
		}
		return m, nil

	case EnterFlashMsg:
		// 用户选择刷写固件，进入固件列表
		m.fwList = FirmwareListModel{}
		m.state = StateFirmwareList
		return m, scanFirmwaresCmd(m.fwScanner)

	case tea.KeyMsg:
		newInfo, cmd := m.deviceInfo.Update(msg)
		m.deviceInfo = newInfo

		if newInfo.IsQuitting() {
			// 返回设备列表
			m.state = StateDeviceList
			return m, nil
		}

		return m, cmd
	}

	newInfo, cmd := m.deviceInfo.Update(msg)
	m.deviceInfo = newInfo
	return m, cmd
}

// handleFirmwareListState 处理固件列表状态的消息。
func (m Model) handleFirmwareListState(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case FirmwareListReadyMsg:
		if msg.Error != "" {
			m.fwList.errMsg = msg.Error
		}
		m.fwList = NewFirmwareListModel(msg.Firmwares)
		return m, nil

	case FirmwareSelectedMsg:
		// 用户选择了固件
		fw := msg.Firmware
		m.selectedFirmware = &fw
		if fw.IsLocale && len(fw.Languages) > 1 {
			// 多语言固件：先进入语言选择
			m.langSelect = NewLanguageSelectModel(fw)
			m.state = StateLanguageSelect
			return m, nil
		}
		// 普通固件：直接刷写
		m.flashProg = NewFlashProgressModel()
		m.state = StateFlashing
		return m, startFlashCmd(m.flasher, *m.selectedDevice, fw)

	case tea.KeyMsg:
		newFw, cmd := m.fwList.Update(msg)
		m.fwList = newFw

		if newFw.IsQuitting() {
			// 返回设备信息
			m.state = StateDeviceInfo
			return m, nil
		}

		return m, cmd
	}

	newFw, cmd := m.fwList.Update(msg)
	m.fwList = newFw
	return m, cmd
}

// handleLanguageSelectState 处理语言选择状态的消息。
func (m Model) handleLanguageSelectState(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case LanguageSelectedMsg:
		// 用户选择了语言，进入刷写
		fw := msg.Firmware
		m.selectedFirmware = &fw
		m.flashProg = NewFlashProgressModel()
		m.state = StateFlashing
		return m, startFlashCmd(m.flasher, *m.selectedDevice, fw)

	case tea.KeyMsg:
		newLang, cmd := m.langSelect.Update(msg)
		m.langSelect = newLang

		if newLang.IsQuitting() {
			// 返回固件列表
			m.state = StateFirmwareList
			return m, nil
		}

		return m, cmd
	}

	newLang, cmd := m.langSelect.Update(msg)
	m.langSelect = newLang
	return m, cmd
}

// handleFlashingState 处理刷写状态的消息。
func (m Model) handleFlashingState(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case StepUpdateMsg:
		newFlash, cmd := m.flashProg.Update(msg)
		m.flashProg = newFlash
		return m, cmd

	case ProgressUpdateMsg:
		newFlash, cmd := m.flashProg.Update(msg)
		m.flashProg = newFlash
		return m, cmd

	case FlashCompleteMsg:
		newFlash, _ := m.flashProg.Update(msg)
		m.flashProg = newFlash
		m.state = StateFlashDone
		return m, nil

	case FlashFailedMsg:
		newFlash, _ := m.flashProg.Update(msg)
		m.flashProg = newFlash
		m.state = StateFlashDone
		return m, nil

	case tea.KeyMsg:
		if m.state == StateFlashDone {
			switch msg.String() {
			case "esc", "q":
				// 返回设备列表
				m.state = StateDeviceList
				return m, scanDevicesCmd(m.scanner)
			}
		}
		newFlash, cmd := m.flashProg.Update(msg)
		m.flashProg = newFlash
		if newFlash.IsQuitting() {
			return m, tea.Quit
		}
		return m, cmd
	}

	newFlash, cmd := m.flashProg.Update(msg)
	m.flashProg = newFlash
	return m, cmd
}

// handleCurrentViewResize 将窗口大小消息传递给当前活跃视图。
func (m Model) handleCurrentViewResize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	switch m.state {
	case StateScanning, StateDeviceList:
		newList, cmd := m.deviceList.Update(msg)
		m.deviceList = newList
		return m, cmd
	case StateDeviceInfo:
		newInfo, cmd := m.deviceInfo.Update(msg)
		m.deviceInfo = newInfo
		return m, cmd
	case StateFirmwareList:
		newFw, cmd := m.fwList.Update(msg)
		m.fwList = newFw
		return m, cmd
	case StateLanguageSelect:
		newLang, cmd := m.langSelect.Update(msg)
		m.langSelect = newLang
		return m, cmd
	case StateFlashing, StateFlashDone:
		newFlash, cmd := m.flashProg.Update(msg)
		m.flashProg = newFlash
		return m, cmd
	}
	return m, nil
}

// View 渲染当前视图。
func (m Model) View() string {
	switch m.state {
	case StateScanning:
		var b strings.Builder
		b.WriteString(RenderAsciiTitle("Thunderstrike", "smslant", colorPrimary))
		b.WriteString(SubtitleStyle.Render("  NVIDIA SHIELD TV 2017 Thunderstrike Controller Tool"))
		b.WriteString("\n\n")
		b.WriteString("\n\n")
		b.WriteString(DimStyle.Render("  正在扫描蓝牙 SPP 设备..."))
		return b.String()

	case StateDeviceList:
		return m.deviceList.View()

	case StateDeviceInfo:
		return m.deviceInfo.View()

	case StateFirmwareList:
		return m.fwList.View()

	case StateLanguageSelect:
		return m.langSelect.View()

	case StateFlashing, StateFlashDone:
		return m.flashProg.View()
	}

	return ""
}

// --- Commands ---

// scanDevicesCmd 异步扫描设备的 tea.Cmd。
func scanDevicesCmd(scanner DeviceScanner) tea.Cmd {
	return func() tea.Msg {
		devices, err := scanner.ScanDevices()
		if err != nil {
			return DevicesScannedMsg{Error: err.Error()}
		}
		return DevicesScannedMsg{Devices: devices}
	}
}

// queryDeviceDetailCmd 异步查询设备详细信息的 tea.Cmd。
func queryDeviceDetailCmd(scanner DeviceScanner, device DeviceInfo) tea.Cmd {
	return func() tea.Msg {
		detail, err := scanner.QueryDeviceDetail(device)
		if err != nil {
			return DeviceDetailReadyMsg{Error: err.Error()}
		}
		return DeviceDetailReadyMsg{Detail: detail}
	}
}

// scanFirmwaresCmd 异步扫描固件的 tea.Cmd。
func scanFirmwaresCmd(fwScanner FirmwareScanner) tea.Cmd {
	return func() tea.Msg {
		firmwares, err := fwScanner.ScanFirmwares()
		if err != nil {
			return FirmwareListReadyMsg{Error: err.Error()}
		}
		return FirmwareListReadyMsg{Firmwares: firmwares}
	}
}

// startFlashCmd 启动异步刷写的 tea.Cmd。
// 通过 stepCB 和 progressCB 将刷写进度转为 TUI 消息。
func startFlashCmd(flasher Flasher, device DeviceInfo, fw FirmwareInfo) tea.Cmd {
	return func() tea.Msg {
		result, flashErr := flasher.ExecuteFlash(
			device, fw,
			func(step, total int, name string) {
				if globalProgram != nil {
					globalProgram.Send(StepUpdateMsg{Step: step, Total: total, Name: name})
				}
			},
			func(written, total int) {
				pct := 0
				if total > 0 {
					pct = written * 100 / total
				}
				if globalProgram != nil {
					globalProgram.Send(ProgressUpdateMsg{Written: written, Total: total, Percent: pct})
				}
			},
		)

		if flashErr != nil {
			return FlashFailedMsg{Error: flashErr.Error()}
		}
		if result != nil && result.Success {
			return FlashCompleteMsg{Result: result}
		}
		errStr := ""
		if result != nil {
			errStr = result.Error
		}
		return FlashFailedMsg{Error: errStr}
	}
}

// globalProgram 全局 Program 引用，用于回调中发送消息。
// 在 Run() 中设置。
var globalProgram *tea.Program
