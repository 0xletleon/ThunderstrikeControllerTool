// Package tui — 多语言固件的语言选择视图。
//
// 当用户选择了一个多语言固件包（IsLocale=true）时，
// 先进入此视图让用户选择要刷写的语言版本，
// 选择完成后再进入刷写流程。
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// LanguageSelectModel 是语言选择视图的状态。
type LanguageSelectModel struct {
	firmware    FirmwareInfo
	languages   []string
	selectedIdx int
	quitting    bool
	width       int
	height      int
}

// NewLanguageSelectModel 创建语言选择模型。
func NewLanguageSelectModel(fw FirmwareInfo) LanguageSelectModel {
	return LanguageSelectModel{
		firmware:  fw,
		languages: fw.Languages,
	}
}

// Update 处理语言选择视图的消息。
func (m LanguageSelectModel) Update(msg tea.Msg) (LanguageSelectModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if len(m.languages) == 0 {
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
			if m.selectedIdx < len(m.languages)-1 {
				m.selectedIdx++
			}

		case "enter":
			if m.selectedIdx >= 0 && m.selectedIdx < len(m.languages) {
				fw := m.firmware
				fw.LanguageIndex = m.selectedIdx
				return m, func() tea.Msg { return LanguageSelectedMsg{Firmware: fw} }
			}
		}
	}

	return m, nil
}

// View 渲染语言选择视图。
func (m LanguageSelectModel) View() string {
	var b strings.Builder

	b.WriteString(RenderAsciiTitle("Thunderstrike", "smslant", colorPrimary))
	b.WriteString(RenderSubtitlePlaceholder())
	b.WriteString("\n\n")

	b.WriteString(SectionHeaderStyle.Render(" ■ 选择语言版本"))
	b.WriteString("\n\n")

	b.WriteString("  " + DimStyle.Render(fmt.Sprintf("固件: %s  V%s (多语言)", m.firmware.Name, m.firmware.Version)))
	b.WriteString("\n\n")

	if len(m.languages) == 0 {
		b.WriteString(WarningStyle.Render("  没有可用的语言选项"))
		b.WriteString("\n\n")
		b.WriteString(DimStyle.Render("  按 ESC/q 返回固件列表"))
		return b.String()
	}

	// 表头
	b.WriteString(fmt.Sprintf("  %s  %s\n",
		padRight("No.", 4),
		padRight("语言", 20)))
	b.WriteString("  " + renderSeparator(30) + "\n")

	for i, lang := range m.languages {
		langDisplay := languageDisplayName(lang)
		line := fmt.Sprintf("  %s  %s",
			padRight(fmt.Sprintf("%d", i+1), 4),
			padRight(langDisplay, 20))

		if i == m.selectedIdx {
			b.WriteString(SelectedItemStyle.Render(line))
		} else {
			b.WriteString(NormalItemStyle.Render(line))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString("  ↑↓ 浏览 | Enter 选择语言 | ESC/q 返回固件列表")

	return b.String()
}

// IsQuitting 返回是否正在退出（返回固件列表）。
func (m LanguageSelectModel) IsQuitting() bool {
	return m.quitting
}

// languageDisplayName 将语言代码转为可读的显示名称。
func languageDisplayName(code string) string {
	names := map[string]string{
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
	if name, ok := names[code]; ok {
		return name
	}
	if code == "" {
		return "默认 (Standard)"
	}
	return code
}

// --- Messages ---

// LanguageSelectedMsg 用户选择了某个语言版本。
type LanguageSelectedMsg struct {
	Firmware FirmwareInfo
}
