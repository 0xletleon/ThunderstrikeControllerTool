// Package tui — 固件列表选择视图。
//
// 扫描 blkz/ 目录下的固件包，以列表形式展示，
// 用户选择后进入刷写确认流程。
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// FirmwareListModel 是固件列表选择视图的状态。
type FirmwareListModel struct {
	firmwares  []FirmwareInfo
	selectedIdx int
	quitting   bool
	errMsg     string
	width      int
	height     int
}

// NewFirmwareListModel 创建固件列表模型。
func NewFirmwareListModel(firmwares []FirmwareInfo) FirmwareListModel {
	return FirmwareListModel{
		firmwares: firmwares,
	}
}

// Update 处理固件列表视图的消息。
func (m FirmwareListModel) Update(msg tea.Msg) (FirmwareListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if len(m.firmwares) == 0 {
			switch msg.String() {
			case "ctrl+c", "q", "esc":
				m.quitting = true
				return m, nil
			}
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.quitting = true
			return m, nil

		case "up", "k":
			if m.selectedIdx > 0 {
				m.selectedIdx--
			}

		case "down", "j":
			if m.selectedIdx < len(m.firmwares)-1 {
				m.selectedIdx++
			}

		case "enter":
			if m.selectedIdx >= 0 && m.selectedIdx < len(m.firmwares) {
				fw := m.firmwares[m.selectedIdx]
				return m, func() tea.Msg { return FirmwareSelectedMsg{Firmware: fw} }
			}
		}
	}

	return m, nil
}

// View 渲染固件列表视图。
func (m FirmwareListModel) View() string {
	var b strings.Builder

	b.WriteString(RenderAsciiTitle("Thunderstrike", "smslant", colorPrimary))
	b.WriteString(SubtitleStyle.Render("  NVIDIA SHIELD TV 2017 Thunderstrike Controller Tool"))
	b.WriteString("\n\n")

	b.WriteString(SectionHeaderStyle.Render(" ■ 可用固件"))

	b.WriteString("\n\n")

	if m.errMsg != "" {
		b.WriteString(ErrorStyle.Render("  "+m.errMsg))
		b.WriteString("\n\n")
	}

	if len(m.firmwares) == 0 {
		b.WriteString(WarningStyle.Render("  blkz 目录中没有可用的固件包"))
		b.WriteString("\n\n")
		b.WriteString(DimStyle.Render("  按 ESC/q 返回"))
		return b.String()
	}

	// 表头（使用显示宽度 padding，确保中文对齐）
	b.WriteString(fmt.Sprintf("  %s  %s  %s  %s\n",
		padRight("No.", 4),
		padRight("固件包", 37),
		padRight("版本", 10),
		"校验"))
	b.WriteString("  " + renderSeparator(70) + "\n")

	for i, fw := range m.firmwares {
		// 校验状态显示
		csStr := ""
		switch fw.Checksum {
		case "match":
			csStr = SuccessStyle.Render("✓")
		case "mismatch":
			csStr = ErrorStyle.Render("✗ 危险")
		case "unknown":
			csStr = WarningStyle.Render("? 未知")
		case "error":
			csStr = ErrorStyle.Render("错误")
		default:
			csStr = DimStyle.Render("-")
		}

		verStr := fmt.Sprintf("V%s", fw.Version)
		name := truncateDisplay(fw.Name, 37)

		line := fmt.Sprintf("  %s  %s  %s  %s",
			padRight(fmt.Sprintf("%d", fw.Index), 4),
			padRight(name, 37),
			padRight(verStr, 10),
			csStr)

		if i == m.selectedIdx {
			b.WriteString(SelectedItemStyle.Render(line))
		} else {
			b.WriteString(NormalItemStyle.Render(line))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString("  ↑↓ 浏览 | Enter 选择固件 | ESC/q 返回设备列表")

	return b.String()
}

// IsQuitting 返回是否正在退出（返回设备信息页）。
func (m FirmwareListModel) IsQuitting() bool {
	return m.quitting
}

// --- Messages ---

// FirmwareSelectedMsg 用户选择了某个固件。
type FirmwareSelectedMsg struct {
	Firmware FirmwareInfo
}

// FirmwareListReadyMsg 固件列表已就绪。
type FirmwareListReadyMsg struct {
	Firmwares []FirmwareInfo
	Error     string
}
