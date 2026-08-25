// Package tui — 设备信息展示视图。
//
// 展示选中设备的 HID 查询结果（固件版本、硬件版本、序列号、MAC、电量等），
// 然后提供"刷写固件"或"返回"操作。
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// DeviceInfoModel 是设备信息展示视图的状态。
type DeviceInfoModel struct {
	device   DeviceInfo
	detail   *DeviceDetail
	quitting bool
	width    int
	height   int
}

// NewDeviceInfoModel 创建设备信息展示模型。
func NewDeviceInfoModel(device DeviceInfo) DeviceInfoModel {
	return DeviceInfoModel{device: device}
}

// SetDetail 设置通过 HID 查询到的设备详细信息。
func (m *DeviceInfoModel) SetDetail(d *DeviceDetail) {
	m.detail = d
}

// Update 处理设备信息视图的消息。
func (m DeviceInfoModel) Update(msg tea.Msg) (DeviceInfoModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.quitting = true
			return m, nil

		case "f":
			// 刷写固件
			return m, func() tea.Msg { return EnterFlashMsg{Device: m.device} }
		}
	}

	return m, nil
}

// View 渲染设备信息视图。
func (m DeviceInfoModel) View() string {
	var b strings.Builder
	b.WriteString(RenderAsciiTitle("Thunderstrike", "smslant", colorPrimary))
	b.WriteString(RenderSubtitlePlaceholder())
	b.WriteString("\n\n")

	b.WriteString(SectionHeaderStyle.Render(" ■ 设备信息"))
	b.WriteString("\n\n")

	b.WriteString("  " + renderInfoLine("设备名称", m.device.DeviceName) + "\n")
	b.WriteString("  " + renderInfoLine("MAC 地址", m.device.MAC) + "\n")

	if m.detail != nil {
		// 固件版本
		fwVer := m.detail.FwVersion
		if fwVer != "" {
			fwVer = fmt.Sprintf("V%s (%s)", fwVer, m.detail.FwVersionHex)
		} else {
			fwVer = "读取失败"
		}
		b.WriteString("  " + renderInfoLine("软件版本", fwVer) + "\n")

		// 蓝牙芯片版本
		if m.detail.CsrVersion != "" {
			b.WriteString("  " + renderInfoLine("蓝牙芯片", m.detail.CsrVersion) + "\n")
		}

		// 热词引擎版本
		if m.detail.HotwordVer != "" {
			b.WriteString("  " + renderInfoLine("热词引擎", m.detail.HotwordVer) + "\n")
		}

		// 硬件版本
		boardInfo := ""
		if m.detail.BoardType > 0 {
			boardInfo = fmt.Sprintf("Board %d Rev %d", m.detail.BoardType, m.detail.BoardRev)
			b.WriteString("  " + renderInfoLine("硬件版本", boardInfo) + "\n")
		}

		// 序列号
		if m.detail.Serial != "" {
			b.WriteString("  " + renderInfoLine("序列号", m.detail.Serial) + "\n")
		}

		// HID MAC
		if m.detail.MacHID != "" {
			b.WriteString("  " + renderInfoLine("MAC(HID)", m.detail.MacHID) + "\n")
		}

		// 传输方式
		b.WriteString("  " + renderInfoLine("传输方式", "蓝牙 HID") + "\n")

		// 电量
		if m.detail.BatteryRaw > 0 {
			battStr := fmt.Sprintf("%d%%  ADC: %d (%s)",
				m.detail.BatteryPct, m.detail.BatteryRaw, m.detail.BatteryLevel)
			b.WriteString("  " + renderInfoLine("电量", battStr) + "\n")

			if m.detail.BatteryPct < 20 {
				b.WriteString("\n")
				b.WriteString("  " + WarningStyle.Render("⚠ 电量低于 20%，刷写过程中断电可能导致变砖！"))
				b.WriteString("\n")
			}
		}
	}

	b.WriteString("\n")
	b.WriteString("  " + renderSeparator(70) + "\n\n")
	b.WriteString("  f 刷写固件 | ESC/q 返回设备列表")

	return b.String()
}

// IsQuitting 返回是否正在退出（返回设备列表）。
func (m DeviceInfoModel) IsQuitting() bool {
	return m.quitting
}

// --- Messages ---

// EnterFlashMsg 用户请求进入刷写流程。
type EnterFlashMsg struct {
	Device DeviceInfo
}

// DeviceDetailReadyMsg 设备详细信息已就绪。
type DeviceDetailReadyMsg struct {
	Detail  *DeviceDetail
	Error   string
}
