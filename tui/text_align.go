// Package tui — 文本对齐工具。
//
// 提供基于显示宽度的字符串 padding，替代 fmt 的 %-Ns。
// Go 的 fmt %-Ns 按字节数计算，UTF-8 中文每字 3 字节但显示宽度 2，
// 导致表头（中文）和内容（混合中英文）无法对齐。
package tui

import "strings"

// runeDisplayWidth 返回字符的显示宽度（中文=2，ASCII=1）。
func runeDisplayWidth(r rune) int {
	if r >= 0x1100 && (
		r <= 0x115F || // 韩文
		r >= 0x2E80 && r <= 0xA4CF || // 中日韩
		r >= 0xAC00 && r <= 0xD7A3 || // 韩文音节
		r >= 0xF900 && r <= 0xFAFF || // 兼容汉字
		r >= 0xFE30 && r <= 0xFE4F || // 兼容形式
		r >= 0xFF00 && r <= 0xFF60 || // 全角
		r >= 0xFFE0 && r <= 0xFFE6) { // 全角符号
		return 2
	}
	return 1
}

// strDisplayWidth 计算字符串的显示宽度。
func strDisplayWidth(s string) int {
	w := 0
	for _, r := range s {
		w += runeDisplayWidth(r)
	}
	return w
}

// padRight 将字符串右侧用空格填充到指定显示宽度。
// 如果字符串显示宽度超过 max，原样返回（不截断）。
func padRight(s string, displayWidth int) string {
	w := strDisplayWidth(s)
	if w >= displayWidth {
		return s
	}
	return s + strings.Repeat(" ", displayWidth-w)
}

// truncateDisplay 截断字符串到指定显示宽度，超长则加省略号。
func truncateDisplay(s string, maxDisplayWidth int) string {
	w := 0
	runes := []rune(s)
	for i, r := range runes {
		rw := runeDisplayWidth(r)
		if w+rw > maxDisplayWidth-1 && i < len(runes)-1 {
			return string(runes[:i]) + "…"
		}
		w += rw
	}
	return s
}
