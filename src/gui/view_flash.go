package gui

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/egoist/mygo/ui"
)

// viewFlash 渲染刷写页：左侧步骤列表，右侧写入进度；结束后显示结果。
func (m *Model) viewFlash(c *ui.Context) {
	fw := m.pendingFw
	hint := fmt.Sprintf("从 V%s 刷写到 V%s", m.selected.FwVersion, fw.Version)
	if m.selected.FwVersion != "" && versionLower(fw.Version, m.selected.FwVersion) {
		hint += "（降级）"
	}
	sectionTitle(c, "刷写固件", hint)

	if m.flashing {
		ui.Box(c).FillWidth().MarginY(8).Children(func() {
			noticeWarn(c, "刷写过程中请保持手柄连接、不要关闭程序或断电。")
		})
	}

	ui.Row(c).FillWidth().Gap(14).MarginY(12).AlignItems(ui.Start).Children(func() {
		card(c).Width(340).Children(func() {
			ui.Text(c, "刷写步骤").FontSize(12).Bold().TextColor(colorMuted).
				Padding(0, 0, 8, 0)
			for i := range m.steps {
				m.stepRow(c, i)
			}
		})

		card(c).Grow(1).Children(func() {
			ui.Text(c, "写入进度").FontSize(12).Bold().TextColor(colorMuted).
				Padding(0, 0, 8, 0)

			ui.Row(c).Gap(10).AlignItems(ui.End).Children(func() {
				ui.Text(c, fmt.Sprintf("%d", m.prog.Percent)).
					FontSize(38).Bold().TextColor(colorBrand)
				ui.Text(c, "%").FontSize(16).TextColor(colorBrand).Padding(0, 0, 8, 0)
				ui.Spacer(c)
				if m.prog.TotalBytes > 0 {
					ui.Text(c, fmt.Sprintf("%s / %s",
						humanSize(m.prog.BytesWritten), humanSize(m.prog.TotalBytes))).
						FontSize(13).TextColor(colorMuted).Padding(0, 0, 8, 0)
				}
			})

			ui.Box(c).PaddingY(8).Children(func() {
				bar(c, float32(m.prog.Ratio()), colorBrand)
			})

			if m.flashing && m.prog.Percent == 0 {
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Spinner(c)
					ui.Text(c, "等待设备响应…").FontSize(13).TextColor(colorMuted)
				})
			}
		})
	})

	if m.result != nil {
		m.resultCard(c)
	}
}

// stepRow 渲染单个刷写步骤。
func (m *Model) stepRow(c *ui.Context, i int) {
	step := m.steps[i]

	var icon, color string
	active := step.Index == m.curStep
	switch {
	case step.Done:
		icon, color = "✓", "done"
	case active:
		icon, color = "→", "active"
	default:
		icon, color = "○", "pending"
	}

	var fg ui.Color
	var nameColor ui.Color
	switch color {
	case "done":
		fg = colorSuccess
		nameColor = colorMuted
	case "active":
		fg = colorBrand
		nameColor = c.Theme().Text
	default:
		fg = ui.Hex("#B0B4BA")
		nameColor = colorMuted
	}

	ui.Row(c).FillWidth().Gap(10).PaddingY(5).AlignItems(ui.Center).Children(func() {
		ui.Text(c, icon).FontSize(14).Bold().TextColor(fg).Width(16)
		ui.Text(c, fmt.Sprintf("%d/%d", step.Index, step.Total)).
			FontSize(12).TextColor(colorMuted).Width(34)
		ui.Text(c, step.Name).FontSize(14).TextColor(nameColor)
		if active && m.flashing {
			ui.Spacer(c)
			ui.Text(c, "进行中").FontSize(12).TextColor(colorBrand)
		}
	})
}

// resultCard 渲染刷写结果卡片（成功或失败）。
func (m *Model) resultCard(c *ui.Context) {
	res := m.result

	if res.Success {
		card(c).FillWidth().Border(1, colorSuccess).Children(func() {
			ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "✓").FontSize(20).Bold().TextColor(colorSuccess)
				ui.Text(c, "刷写成功").FontSize(16).Bold().TextColor(colorSuccess)
				ui.Spacer(c)
				chipOK(c, "已完成")
			})

			ui.Box(c).Padding(8, 0, 0, 0).Children(func() {
				infoRow(c, "耗时", res.Elapsed)
				infoRow(c, "写入", humanSize(res.BytesWrit))
				infoRow(c, "版本", fmt.Sprintf("V%s → V%s", res.FromVer, res.ToVer))
				infoRow(c, "日志", shortPath(res.LogPath))
			})

			ui.Text(c, "手柄正在重启，稍后可在设备列表中重新连接查看版本。").
				FontSize(12).TextColor(colorMuted).Padding(6, 0, 0, 0)
		})
		return
	}

	card(c).FillWidth().Border(1, colorDanger).Children(func() {
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "✗").FontSize(20).Bold().TextColor(colorDanger)
			ui.Text(c, "刷写失败").FontSize(16).Bold().TextColor(colorDanger)
		})
		ui.Box(c).Padding(8, 0, 0, 0).Children(func() {
			ui.Text(c, res.Error).FontSize(13).TextColor(c.Theme().Text).Wrap().
				Selectable()
		})
		ui.Text(c, "手柄固件未改变。可检查电量、重新连接手柄后重试。").
			FontSize(12).TextColor(colorMuted).Padding(6, 0, 0, 0)
	})
}

// shortPath 把绝对路径转成相对当前目录的短路径，便于阅读。
func shortPath(path string) string {
	if path == "" {
		return ""
	}
	wd, err := os.Getwd()
	if err != nil {
		return path
	}
	if rel, err := filepath.Rel(wd, path); err == nil {
		return rel
	}
	return path
}
