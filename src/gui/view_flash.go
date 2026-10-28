package gui

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/egoist/mygo/ui"

	"thunderstrike-controller-tool/src/core"
)

// viewFlash 渲染刷写页。整个界面唯一"戏剧化"的地方：
// 大号百分比 + 分段步骤条，其余全部压到最小。
func (m *Model) viewFlash(c *ui.Context) {
	fw := m.pendingFw

	hint := fmt.Sprintf("V%s → V%s", orDash(m.selected.FwVersion), fw.Version)
	if m.selected.FwVersion != "" && versionLower(fw.Version, m.selected.FwVersion) {
		hint += "（降级）"
	}
	sectionHead(c, "刷写固件", hint, nil)

	ui.Box(c).Height(20)

	done := m.result != nil && m.result.Success

	// 主体在可用高度内垂直居中
	ui.Column(c).FillWidth().MinHeight(bodyHeight(c) - 46).
		Justify(ui.Center).
		Children(func() {
			// 成功后不再展示百分比与步骤：信息已完成使命，只留结果
			if !done {
				// 大号百分比，整页视觉锚点
				ui.Row(c).FillWidth().Justify(ui.Center).AlignItems(ui.End).Gap(2).
					Children(func() {
						ui.Text(c, fmt.Sprintf("%d", m.prog.Percent)).
							FontSize(56).Bold().TextColor(colorBrand)
						ui.Text(c, "%").FontSize(16).TextColor(colorBrand).
							Padding(0, 0, 12, 0)
					})

				ui.Box(c).Height(14)

				// 细进度条居中，两侧留白让它成为一个「读数」而不是页面元素
				ui.Row(c).FillWidth().Justify(ui.Center).Children(func() {
					ui.Box(c).Width(440).Children(func() {
						bar(c, float32(m.prog.Ratio()), colorBrand)
					})
				})

				if m.prog.TotalBytes > 0 {
					ui.Box(c).Height(8)
					ui.Row(c).FillWidth().Justify(ui.Center).Children(func() {
						ui.Text(c, fmt.Sprintf("%s / %s",
							humanSize(m.prog.BytesWritten), humanSize(m.prog.TotalBytes))).
							FontSize(10).TextColor(colorTextFaint)
					})
				}

				ui.Box(c).Height(30)

				// 步骤：横向分段条 + 名称
				ui.Row(c).FillWidth().Justify(ui.Center).Children(func() {
					ui.Column(c).Width(440).Gap(7).Children(func() {
						segBar(c, len(m.steps), m.curStep, done)
						stepLegend(c, stepNames(m.steps), m.curStep, done)
					})
				})

				if m.flashing {
					ui.Box(c).Height(22)
					ui.Row(c).FillWidth().Justify(ui.Center).Children(func() {
						ui.Text(c, "请保持手柄连接，不要关闭程序").
							FontSize(11).TextColor(colorTextFaint)
					})
				}
			}

			if m.result != nil {
				m.resultBlock(c)
			}
		})
}

// resultBlock 渲染刷写结果。成功与失败用同一种结构，只换颜色和内容。
func (m *Model) resultBlock(c *ui.Context) {
	res := m.result
	ok := res.Success

	fg := colorDanger
	mark := "✗"
	title := "刷写失败"
	if ok {
		fg = colorBrand
		mark = "✓"
		title = "刷写完成"
	}

	ui.Box(c).Height(24)

	ui.Column(c).FillWidth().Padding(14, 16).Radius(4).Gap(10).
		Background(colorPanel).Border(1, fg).
		Children(func() {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, mark).FontSize(13).Bold().TextColor(fg)
				ui.Text(c, title).FontSize(13).Bold().TextColor(fg)
			})

			if ok {
				ui.Row(c).FillWidth().Gap(28).Children(func() {
					inlineKV(c, "耗时", res.Elapsed)
					inlineKV(c, "版本", fmt.Sprintf("V%s → V%s", res.FromVer, res.ToVer))
					inlineKV(c, "写入", humanSize(res.BytesWrit))
				})
				if path := shortPath(res.LogPath); path != "" {
					ui.Text(c, path).FontSize(10).TextColor(colorTextMuted).Selectable()
				}
				ui.Text(c, "手柄正在重启，稍后可重新连接查看版本").
					FontSize(10).TextColor(colorTextMuted).Padding(4, 0, 0, 0)
				return
			}

			ui.Text(c, res.Error).FontSize(11).TextColor(colorTextMuted).
				Wrap().Selectable()
			ui.Text(c, "固件未改变，可检查电量与连接后重试").
				FontSize(10).TextColor(colorTextFaint)
		})
}

// inlineKV 渲染横向排列的「标签 值」，用于结果摘要。
func inlineKV(c *ui.Context, key, value string) {
	ui.Column(c).Grow(1).Gap(2).Children(func() {
		ui.Text(c, key).FontSize(10).TextColor(colorTextFaint)
		ui.Text(c, value).FontSize(12).TextColor(colorTextMuted)
	})
}

// stepNames 提取刷写步骤名称。
func stepNames(steps []core.FlashStep) []string {
	names := make([]string, len(steps))
	for i, s := range steps {
		names[i] = s.Name
	}
	return names
}

// shortPath 把绝对路径转成相对当前目录的短路径。
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
