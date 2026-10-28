// Package gui — 原生 GUI 界面层（MyGo 自绘 UI）。
//
// 单页布局：设备信息、固件列表、待刷版本与刷写操作同屏呈现，
// 不做页面跳转。视觉只用白底 + NVIDIA 绿色家族 + 黑灰文字。
package gui

import "github.com/egoist/mygo/ui"

// 版式常量。
const (
	headerHeight = 44
	footerHeight = 34
	actionHeight = 76
	pagePadding  = 18
)

// 色板：白底 + NVIDIA 绿色家族。
// 官方主绿 #76B900（Pantone 376 C），其余深浅绿都是它的衍生。
var (
	colorBrand      = ui.Hex("#76B900") // 主绿：刷写按钮、进度、选中
	colorBrandHover = ui.Hex("#8BD400")
	colorBrandDark  = ui.Hex("#5C9100") // 浅底上的绿色文字
	colorBrandTrack = ui.Hex("#DFF0BC") // 轨道、描边
	colorBrandTint  = ui.Hex("#EDF7DA") // 淡绿底：选中行、徽章

	colorBg    = ui.Hex("#FFFFFF")
	colorPanel = ui.Hex("#F7F9F3") // 分区底
	colorLine  = ui.Hex("#E5E8E0")

	colorText      = ui.Hex("#141414")
	colorTextMuted = ui.Hex("#5F6459")
	colorTextFaint = ui.Hex("#9AA093")

	colorInk = ui.Hex("#0F0F0F") // 绿底黑字
)

// buildTheme 构造浅色主题。
func buildTheme() *ui.Theme {
	t := ui.LightTheme()

	t.Background = colorBg
	t.Surface = colorPanel
	t.Border = colorLine
	t.Text = colorText
	t.TextMuted = colorTextMuted

	t.Accent = colorBrand
	t.AccentHover = colorBrandHover
	t.AccentPressed = colorBrandDark
	t.AccentText = colorInk

	t.Success = colorBrandDark
	t.Focus = colorBrand
	t.Selection = colorBrandTint
	t.Scrollbar = colorBrandTrack

	t.Radius = 4
	t.Spacing = 4
	t.FontSize = 13
	return t
}

// hairline 渲染一条极细分隔线。
func hairline(c *ui.Context) {
	ui.Box(c).FillWidth().Height(1).Background(colorLine)
}

// sectionLabel 渲染小节标签（灰色小字，可带右侧计数）。
func sectionLabel(c *ui.Context, text string, trailing func()) {
	ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Text(c, text).FontSize(10).TextColor(colorTextFaint).LetterSpacing(0.8)
		ui.Spacer(c)
		if trailing != nil {
			trailing()
		}
	})
}

// bar 渲染一条进度条，ratio 限制在 0–1。
func bar(c *ui.Context, ratio float32, fg ui.Color) {
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}

	ui.Box(c).FillWidth().Height(4).Radius(2).Background(colorBrandTrack).
		Children(func() {
			if ratio <= 0 {
				return
			}
			ui.Box(c).WidthPercent(ratio * 100).FillHeight().Radius(2).Background(fg)
		})
}

// chip 渲染状态小标签：彩色文字 + 淡绿底。
func chip(c *ui.Context, text string, fg ui.Color) {
	ui.Text(c, text).
		FontSize(10).
		TextColor(fg).
		Background(colorBrandTint).
		Padding(2, 6).
		Radius(3)
}

// versionText 把版本号格式化为展示文本，空值用占位符。
func versionText(v string) string {
	if v == "" {
		return "—"
	}
	return "V" + v
}

// orDash 在字符串为空时返回占位符。
func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

// batteryColor 按电量返回绿色深浅；低于 20% 用黑色（安全信号）。
func batteryColor(pct int) ui.Color {
	switch {
	case pct >= 60:
		return colorBrand
	case pct >= 20:
		return colorBrandDark
	default:
		return colorText
	}
}
