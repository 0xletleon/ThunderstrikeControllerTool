// Package gui — 原生 GUI 界面层（MyGo 自绘 UI）。
//
// 与终端界面（tui 包）平级：两者都只依赖 core 包定义的业务接口，
// 业务逻辑（设备扫描、HID 查询、固件解析、SPP 刷写）完全复用 cmd 包。
package gui

import "github.com/egoist/mygo/ui"

// 品牌色系：沿用终端版的 NVIDIA 配色，供强调、状态与徽章使用。
var (
	colorBrand   = ui.Hex("#76B900") // NVIDIA 绿
	colorAccent  = ui.Hex("#9FE400") // 亮绿
	colorDanger  = ui.Hex("#E03C31") // NVIDIA 红
	colorWarning = ui.Hex("#FFB300") // 琥珀金
	colorSuccess = ui.Hex("#4E9A06") // 深一点的成功绿（浅底上更清晰）
	colorMuted   = ui.Hex("#6B6B6B") // 次要文字
	colorLine    = ui.Hex("#E3E6EA") // 分隔线/描边
	colorRowHover = ui.Hex("#F6F8F3")
	colorRowSel   = ui.Hex("#EDF6DA")
)

// 徽章底色（浅色底 + 深色字，保证亮暗主题下都可读）。
var (
	chipOkBg   = ui.Hex("#E8F5E0")
	chipWarnBg = ui.Hex("#FFF3D9")
	chipErrBg  = ui.Hex("#FCEAE8")
	chipNeBg   = ui.Hex("#EDF0F3")
)

// buildTheme 以系统亮/暗主题为基底，替换为 NVIDIA 品牌强调色。
func buildTheme(dark bool) *ui.Theme {
	t := ui.LightTheme()
	if dark {
		t = ui.DarkTheme()
		t.Background = ui.Hex("#141414")
		t.Surface = ui.Hex("#1E1F21")
	} else {
		t.Background = ui.Hex("#F2F3F5")
		t.Surface = ui.Hex("#FFFFFF")
	}
	t.Accent = colorBrand
	t.AccentHover = ui.Hex("#86CC00")
	t.AccentPressed = ui.Hex("#5C9100")
	t.AccentText = ui.Hex("#FFFFFF")
	t.Danger = colorDanger
	t.Warning = colorWarning
	t.Success = colorSuccess
	t.Radius = 8
	t.Spacing = 5
	return t
}

// chip 渲染一个状态徽章：彩色文字 + 浅色底 + 圆角。
func chip(c *ui.Context, text string, fg, bg ui.Color) {
	ui.Text(c, text).
		FontSize(12).
		Bold().
		TextColor(fg).
		Background(bg).
		Padding(3, 9).
		Radius(10)
}

// chipOK 渲染绿色成功徽章。
func chipOK(c *ui.Context, text string) {
	chip(c, text, colorSuccess, chipOkBg)
}

// chipWarn 渲染黄色警告徽章。
func chipWarn(c *ui.Context, text string) {
	chip(c, text, colorWarning, chipWarnBg)
}

// chipErr 渲染红色错误徽章。
func chipErr(c *ui.Context, text string) {
	chip(c, text, colorDanger, chipErrBg)
}

// chipNeutral 渲染灰色中性徽章。
func chipNeutral(c *ui.Context, text string) {
	chip(c, text, colorMuted, chipNeBg)
}

// card 返回一张内容卡片（白底、圆角、细描边）。宽度由调用方决定。
func card(c *ui.Context) ui.Element {
	return ui.Column(c).
		Padding(18, 20).
		Gap(2).
		Background(c.Theme().Surface).
		Radius(12).
		Border(1, colorLine)
}

// infoRow 渲染一行「标签  值」信息，值可选中复制（序列号、MAC 常用）。
func infoRow(c *ui.Context, label, value string) {
	if value == "" {
		return
	}
	ui.Row(c).FillWidth().Gap(14).PaddingY(6).Children(func() {
		ui.Text(c, label).Width(96).FontSize(13).TextColor(colorMuted)
		ui.Text(c, value).FontSize(13).TextColor(c.Theme().Text).Selectable()
	})
}

// sectionTitle 渲染页面标题与一行说明。
func sectionTitle(c *ui.Context, title, hint string) {
	ui.Column(c).Gap(4).Padding(0, 0, 4, 0).Children(func() {
		ui.Text(c, title).FontSize(19).Bold().TextColor(c.Theme().Text)
		if hint != "" {
			ui.Text(c, hint).FontSize(13).TextColor(colorMuted)
		}
	})
}

// notice 渲染一条提示条：左边色条 + 文字。
func notice(c *ui.Context, text string, fg, bg ui.Color) {
	ui.Row(c).FillWidth().Gap(10).Padding(10, 12).Radius(8).Background(bg).
		AlignItems(ui.Start).
		Children(func() {
			ui.Box(c).Width(3).FillHeight().Background(fg).Radius(2)
			ui.Text(c, text).FontSize(13).TextColor(fg).Wrap()
		})
}

// noticeWarn 渲染黄色警告条。
func noticeWarn(c *ui.Context, text string) {
	notice(c, text, ui.Hex("#8A6100"), chipWarnBg)
}

// noticeErr 渲染红色错误条。
func noticeErr(c *ui.Context, text string) {
	notice(c, text, ui.Hex("#A32319"), chipErrBg)
}

// bar 渲染一条自绘进度条，颜色可自定义（电量、写入进度共用）。
// ratio 会被限制在 0–1 之间。
func bar(c *ui.Context, ratio float32, fg ui.Color) {
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}

	ui.Box(c).FillWidth().Height(10).Radius(5).Background(chipNeBg).
		Children(func() {
			if ratio <= 0 {
				return
			}
			ui.Box(c).WidthPercent(ratio * 100).FillHeight().Radius(5).Background(fg)
		})
}

// footerBar 渲染底部操作栏：左侧状态与快捷键提示，右侧按钮组。
func footerBar(c *ui.Context, status, hint string, buttons func()) {
	ui.Row(c).FillWidth().Padding(12, 20).Gap(10).
		Background(c.Theme().Surface).
		BorderWidth(1, 0, 0, 0).
		BorderColor(colorLine).
		AlignItems(ui.Center).
		Children(func() {
			ui.Text(c, status).FontSize(12).TextColor(colorMuted)
			if hint != "" {
				ui.Text(c, "·  "+hint).FontSize(12).TextColor(ui.Hex("#9AA0A6"))
			}
			ui.Spacer(c)
			buttons()
		})
}
