// Package tui — 设备列表选择视图。
//
// 展示蓝牙 SPP 设备列表，用户通过方向键浏览、Enter 选择、r 刷新、q 退出。
// 采用手写渲染方式（分隔式列表、高亮选中项），风格与原 CLI 一致。
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// DeviceListModel 是设备列表选择视图的状态。
type DeviceListModel struct {
	devices     []DeviceInfo
	selectedIdx int
	selected    *DeviceInfo
	quitting    bool
	errMsg      string
	scanning    bool
	width       int
	height      int
}

// NewDeviceListModel 创建设备列表模型。
func NewDeviceListModel(devices []DeviceInfo) DeviceListModel {
	return DeviceListModel{
		devices:     devices,
		selectedIdx: 0,
	}
}

// Update 处理设备列表视图的消息。
func (m DeviceListModel) Update(msg tea.Msg) (DeviceListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if len(m.devices) == 0 {
			switch msg.String() {
			case "ctrl+c", "q":
				m.quitting = true
				return m, tea.Quit
			case "r":
				return m, func() tea.Msg { return RefreshDevicesMsg{} }
			}
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit

		case "r":
			return m, func() tea.Msg { return RefreshDevicesMsg{} }

		case "up", "k":
			if m.selectedIdx > 0 {
				m.selectedIdx--
			}

		case "down", "j":
			if m.selectedIdx < len(m.devices)-1 {
				m.selectedIdx++
			}

		case "enter":
			if m.selectedIdx >= 0 && m.selectedIdx < len(m.devices) {
				dev := m.devices[m.selectedIdx]
				m.selected = &dev
				return m, func() tea.Msg { return DeviceSelectedMsg{Device: dev} }
			}
		}
	}

	return m, nil
}

// View 渲染设备列表视图。
func (m DeviceListModel) View() string {
	var b strings.Builder

	b.WriteString(RenderAsciiTitle("Thunderstrike", "smslant", colorPrimary))
	b.WriteString(RenderSubtitlePlaceholder())
	b.WriteString("\n\n")

	if m.scanning {
		b.WriteString("  正在扫描蓝牙 SPP 设备...")
		b.WriteString("\n\n")
	}

	if m.errMsg != "" {
		b.WriteString(ErrorStyle.Render("  扫描出错: " + m.errMsg))
		b.WriteString("\n\n")
	}

	if len(m.devices) == 0 && !m.scanning {
		b.WriteString(WarningStyle.Render("  未发现蓝牙设备"))
		b.WriteString("\n")
		b.WriteString("  请确保手柄已与电脑配对并连接")
		b.WriteString("\n\n")
		b.WriteString("  按 r 刷新 | 按 q 退出")
		return b.String()
	}

	b.WriteString(SectionHeaderStyle.Render(" ■ 设备列表"))
	b.WriteString("\n\n")
	
	// 表头（使用显示宽度 padding，确保中文对齐）
	b.WriteString(fmt.Sprintf("  %s  %s  %s  %s  %-10s  %s\n",
		padRight("No.", 4),
		padRight("设备名称", 24),
		padRight("端口", 8),
		padRight("MAC 地址", 18),
		"固件版本", "状态"))
	b.WriteString("  " + renderSeparator(100) + "\n")

	// 设备列表
	for i, d := range m.devices {
		name := truncateDisplay(d.DeviceName, 24)

		// 设备状态：FwVersion 非空 = 在线，为空 = 休眠
		var fwVer, status string
		if d.FwVersion != "" {
			fwVer = "V" + d.FwVersion
			status = SuccessStyle.Render("在线")
		} else {
			fwVer = "V0.00"
			status = WarningStyle.Render("休眠")
		}

		line := fmt.Sprintf(" %s  %s  %s  %s  %s  %s",
			padRight(fmt.Sprintf("%d", d.Index), 4),
			padRight(name, 24),
			padRight(d.Port, 8),
			padRight(d.MAC, 18),
			padRight(fwVer,14),
			status,
		)

		if i == m.selectedIdx {
			b.WriteString(SelectedItemStyle.Render(line))
		} else {
			b.WriteString(NormalItemStyle.Render(line))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString("  ↑↓ 浏览 | Enter 选择 | r 刷新 | q 退出")

	return b.String()
}

// SelectedDevice 返回用户选中的设备，未选择返回 nil。
func (m DeviceListModel) SelectedDevice() *DeviceInfo {
	return m.selected
}

// IsQuitting 返回是否正在退出。
func (m DeviceListModel) IsQuitting() bool {
	return m.quitting
}

// --- Messages ---

// RefreshDevicesMsg 请求刷新设备列表。
type RefreshDevicesMsg struct{}

// DeviceSelectedMsg 用户选择了某个设备。
type DeviceSelectedMsg struct {
	Device DeviceInfo
}

// DevicesScannedMsg 设备扫描完成。
type DevicesScannedMsg struct {
	Devices []DeviceInfo
	Error   string
}


