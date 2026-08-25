// Package tui — Lipgloss 样式定义。
//
// 集中管理所有 TUI 组件的视觉样式：颜色、对齐、边框、间距等。
// 通过修改此文件即可全局调整 TUI 外观。
package tui

import "github.com/charmbracelet/lipgloss"

// 色彩主题（NVIDIA 品牌色系）
//
// 主色：NVIDIA Green #76B900 — 标题、边框、选中项、进度条
// 辅色：亮绿 #9FE400 — 强调、当前步骤
// 背景：深黑 #1A1A1A — 选中项背景替代
// 文字：浅灰 #CCCCCC — 默认文本
// 语义：成功 #76B900（绿）、警告 #FFB300（琥珀）、错误 #E03C31（红）
const (
	colorPrimary    = "#76B900" // 主色（NVIDIA 绿）
	colorAccent      = "#9FE400" // 辅色（亮绿，高亮当前步骤）
	colorSuccess     = "#76B900" // 成功（NVIDIA 绿）
	colorWarning     = "#FFB300" // 警告（琥珀金）
	colorError       = "#E03C31" // 错误（NVIDIA 红）
	colorDim         = "#6B6B6B" // 暗淡（灰）
	colorHighlight   = "#9FE400" // 高亮（亮绿）
	colorBgDark      = "#1A1A1A" // 深黑背景
	colorTextDefault = "#CCCCCC" // 默认文本（浅灰）
)

// 标题样式：主色、粗体、下边框
var TitleStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color(colorPrimary)).
	BorderStyle(lipgloss.NormalBorder()).
	BorderBottom(true).
	Padding(0, 1)

// 副标题样式：暗淡色
var SubtitleStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color(colorDim))

// 节标题样式：强调色、粗体
var SectionHeaderStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color(colorAccent)).
	PaddingLeft(1)

// 信息行标签样式：主色、固定宽度
var LabelStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color(colorPrimary)).
	Width(12)

// 信息行值样式：浅灰文字
var ValueStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color(colorTextDefault))

// 成功文本样式
var SuccessStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color(colorSuccess)).
	Bold(true)

// 警告文本样式
var WarningStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color(colorWarning)).
	Bold(true)

// 错误文本样式
var ErrorStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color(colorError)).
	Bold(true)

// 暗淡文本样式
var DimStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color(colorDim))

// 高亮文本样式
var HighlightStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color(colorHighlight)).
	Bold(true)

// 列表选中项样式：NVIDIA 深黑背景 + 亮绿文字
var SelectedItemStyle = lipgloss.NewStyle().
	Background(lipgloss.Color(colorBgDark)).
	Foreground(lipgloss.Color(colorAccent)).
	Bold(true).
	Padding(0, 1)

// 列表普通项样式
var NormalItemStyle = lipgloss.NewStyle().
	Padding(0, 1)

// 分隔线样式
var SeparatorStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color(colorDim))

// 步骤标签样式
var StepLabelStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color(colorAccent)).
	Bold(true)

// 步骤完成样式
var StepDoneStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color(colorSuccess))

// 步骤等待样式
var StepPendingStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color(colorDim))

// ErrorBoxStyle 错误容器样式
var ErrorBoxStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color(colorError)).
	Padding(1, 2)

// SuccessBoxStyle 成功容器样式
var SuccessBoxStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color(colorSuccess)).
	Padding(1, 2)

// renderInfoLine 渲染一行 "标签: 值" 信息。
func renderInfoLine(label, value string) string {
	return LabelStyle.Render(label) + " " + ValueStyle.Render(value)
}

// renderSeparator 渲染分隔线。
func renderSeparator(width int) string {
	line := ""
	for i := 0; i < width; i++ {
		line += "─"
	}
	return SeparatorStyle.Render(line)
}
