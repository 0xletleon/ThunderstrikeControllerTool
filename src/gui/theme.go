// Package gui — 原生 GUI 界面层（MyGo 自绘 UI）。
//
// 视觉基调：NVIDIA 官方品牌只有绿（#76B900）、黑、白三色，
// 因此界面以黑灰阶打底，绿色只出现在需要用户做决定的地方
// （主按钮、选中项、进度、在线状态、完成）。
package gui

import "github.com/egoist/mygo/ui"

// 版式常量：页头、底栏与内容区内边距。视图里需要计算可用高度时统一引用。
const (
	headerHeight   float32 = 48
	footerHeight   float32 = 40
	contentPadding float32 = 14
)

// 色板。
var (
	colorBrand      = ui.Hex("#76B900") // NVIDIA 官方绿（Pantone 376 C）
	colorBrandHover = ui.Hex("#8BD400")
	colorBrandDark  = ui.Hex("#5C9100")
	colorBrandTrack = ui.Hex("#2E4200") // 绿的低调轨道色

	colorBg       = ui.Hex("#0F0F0F")
	colorPanel    = ui.Hex("#171717")
	colorPanelAlt = ui.Hex("#1E1E1E")
	colorLine     = ui.Hex("#2A2A2A")
	colorTrack    = ui.Hex("#242424")

	colorText      = ui.Hex("#EDEDED")
	colorTextMuted = ui.Hex("#8A8A8A")
	colorTextFaint = ui.Hex("#5C5C5C")

	colorWarning = ui.Hex("#E8A33D")
	colorDanger  = ui.Hex("#D93A2B")
	colorInk     = ui.Hex("#0F0F0F") // 绿底上的文字色
)

// buildTheme 构造深色主题。品牌绿既作强调色也作成功色——
// 只用一种绿，视觉语义才干净。
func buildTheme() *ui.Theme {
	t := ui.DarkTheme()

	t.Background = colorBg
	t.Surface = colorPanel
	t.Border = colorLine
	t.Text = colorText
	t.TextMuted = colorTextMuted

	t.Accent = colorBrand
	t.AccentHover = colorBrandHover
	t.AccentPressed = colorBrandDark
	t.AccentText = colorInk

	t.Danger = colorDanger
	t.Warning = colorWarning
	t.Success = colorBrand
	t.Focus = colorBrand
	t.Selection = colorBrandTrack
	// 滚动条默认是灰色长条，在深色底上很跳；直接与背景同色彻底隐形
	t.Scrollbar = colorBg

	// 更小的圆角 + 更紧的间距，接近 NVIDIA 软件的锐利观感
	t.Radius = 4
	t.Spacing = 4
	t.FontSize = 13
	return t
}

// brandMark 渲染斜切的品牌标识块（NVIDIA 标识体系的倾斜语言）。
func brandMark(c *ui.Context, size float32) {
	ui.Box(c).Size(size, size).Radius(0).Background(colorBrand).Skew(-8, 0)
}

// sectionHead 渲染页面标题行：左侧标题与说明，右侧放当前页的操作。
func sectionHead(c *ui.Context, title, hint string, action func()) {
	ui.Row(c).FillWidth().Gap(12).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Gap(2).Children(func() {
			ui.Text(c, title).FontSize(15).Bold().TextColor(colorText)
			if hint != "" {
				ui.Text(c, hint).FontSize(11).TextColor(colorTextFaint)
			}
		})
		ui.Spacer(c)
		if action != nil {
			action()
		}
	})
}

// hairline 渲染一条极细分隔线。
func hairline(c *ui.Context) {
	ui.Box(c).FillWidth().Height(1).Background(colorLine)
}

// hairlineBrand 渲染页头收边线：从品牌绿向右渐隐。
func hairlineBrand(c *ui.Context) {
	ui.Box(c).FillWidth().Height(1).Gradient(colorBrand, colorBg, 0)
}

// stat 渲染一个大号关键数字，可带单位。这是界面的视觉锚点：
// 版本号、电量、进度百分比都走这里。
func stat(c *ui.Context, value, unit string, color ui.Color) {
	ui.Row(c).Gap(2).AlignItems(ui.End).Children(func() {
		ui.Text(c, value).FontSize(32).Bold().TextColor(color)
		if unit != "" {
			ui.Text(c, unit).FontSize(12).TextColor(colorTextMuted).Padding(0, 0, 5, 0)
		}
	})
}

// label 渲染一行小号灰字，用于数字下方的补充说明。
func label(c *ui.Context, text string) {
	ui.Text(c, text).FontSize(11).TextColor(colorTextFaint)
}

// bar 渲染一条细进度条（4px），颜色可自定义。
func bar(c *ui.Context, ratio float32, fg ui.Color) {
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}

	ui.Box(c).FillWidth().Height(4).Radius(2).Background(colorTrack).
		Children(func() {
			if ratio <= 0 {
				return
			}
			ui.Box(c).WidthPercent(ratio * 100).FillHeight().Radius(2).Background(fg)
		})
}

// segBar 渲染分段进度条：每段对应一个刷写步骤，
// 已完成的段填满绿色，当前段半亮，未开始的段为暗轨。
//
// 比竖排步骤文字更紧凑，也更像仪器的读数面板。
func segBar(c *ui.Context, total, current int, done bool) {
	ui.Row(c).FillWidth().Gap(4).Children(func() {
		for i := 1; i <= total; i++ {
			fill := colorTrack
			switch {
			case done || i < current:
				fill = colorBrand
			case i == current:
				fill = colorBrandDark
			}
			ui.Box(c).Grow(1).Height(3).Radius(1.5).Background(fill)
		}
	})
}

// stepLegend 渲染分段条下方的步骤名称，当前步骤高亮。
func stepLegend(c *ui.Context, names []string, current int, done bool) {
	ui.Row(c).FillWidth().Gap(4).Children(func() {
		for i, name := range names {
			fg := colorTextFaint
			switch {
			case done || i+1 < current:
				fg = colorTextMuted
			case i+1 == current:
				fg = colorBrand
			}
			ui.Box(c).Grow(1).Children(func() {
				ui.Text(c, name).FontSize(10).TextColor(fg)
			})
		}
	})
}

// chip 渲染一个状态小标签：强调色文字 + 极暗底色，不抢视觉。
func chip(c *ui.Context, text string, fg ui.Color) {
	ui.Text(c, text).
		FontSize(11).
		TextColor(fg).
		Background(colorPanelAlt).
		Padding(2, 7).
		Radius(3)
}

// kv 渲染一组「标签  值」小字，用于页脚区域的次要信息。
func kv(c *ui.Context, key, value string) {
	ui.Column(c).Gap(2).Children(func() {
		ui.Text(c, key).FontSize(10).TextColor(colorTextFaint)
		ui.Text(c, value).FontSize(12).TextColor(colorTextMuted)
	})
}

// bodyHeight 返回内容区可用高度（扣除页头、底栏与内边距）。
// 内容较少的页面用它把主体垂直居中，避免底部大片空白。
func bodyHeight(c *ui.Context) float32 {
	_, h := c.Size()
	return h - headerHeight - footerHeight - contentPadding*2
}

// footerBar 渲染底部状态条：左侧状态与快捷键提示，右侧可选操作。
func footerBar(c *ui.Context, status, hint string, trailing func()) {
	ui.Box(c).FillWidth().Children(func() {
		hairline(c)
		ui.Row(c).FillWidth().Height(footerHeight).Padding(0, 16).Gap(10).
			AlignItems(ui.Center).
			Children(func() {
				ui.Text(c, status).FontSize(11).TextColor(colorTextMuted)
				if hint != "" {
					ui.Text(c, "·  "+hint).FontSize(11).TextColor(colorTextFaint)
				}
				ui.Spacer(c)
				if trailing != nil {
					trailing()
				}
			})
	})
}
