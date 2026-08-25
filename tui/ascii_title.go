// Package tui — ASCII Art 标题渲染。
//
// 使用 go-figure 库将文本转为 ASCII Art 大字体，
// 作为 TUI 顶部标题展示。
package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/common-nighthawk/go-figure"
)

// RenderAsciiTitle 将文本渲染为 ASCII Art 字符串。
// font 指定字体（如 "standard"、"slant"、"small"、"speed" 等）。
// color 是 ANSI 颜色码（如 "#7B68EE"）。
func RenderAsciiTitle(text, font, color string) string {
	f := figure.NewFigure(text, font, true)
	art := f.String()

	// 逐行着色，统一缩进 2 空格
	lines := strings.Split(art, "\n")
	var b strings.Builder
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		b.WriteString("  ")
		b.WriteString(colorize(line, color))
		b.WriteString("\n")
	}
	return b.String()
}

// colorize 用 ANSI 前景色给字符串着色。
func colorize(s, color string) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(color)).
		Render(s)
}
