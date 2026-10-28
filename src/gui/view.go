package gui

import (
	"fmt"

	"github.com/egoist/mygo/ui"

	"thunderstrike-controller-tool/src/core"
)

// deviceBlockH 设备信息区固定高度。
const deviceBlockH = deviceHeight

// deviceBlock 渲染设备区：名称、固件版本、电量、端口。
func (m *Model) deviceBlock(c *ui.Context) {
	ui.Box(c).FillWidth().Height(deviceBlockH).Padding(10, pagePadding).
		Children(func() {
			if m.scanning {
				m.blockHint(c, "正在扫描蓝牙设备…", true)
				return
			}
			dev := m.selectedDevice()
			if dev == nil {
				m.deviceEmpty(c)
				return
			}

			d := m.detail
			if d == nil {
				d = &core.DeviceDetail{DeviceName: dev.DeviceName, MAC: dev.MAC}
			}

			// 纵向排列：标题行带刷新，下面依次是读数与端口。
			ui.Column(c).FillWidth().Gap(8).Children(func() {
				// 第 1 行：名称、状态与刷新。
				// 刷新按钮紧跟在状态后面而不是推到右端：真机上右对齐按钮不渲染。
				ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
					ui.Text(c, dev.DeviceName).FontSize(15).Bold().
						TextColor(colorText)
					if dev.Online() {
						chip(c, "在线", colorBrandDark)
					} else {
						chip(c, "休眠", colorTextMuted)
					}

					// 唯一的刷新入口：设备与固件一起刷
					if m.scanning || m.detailLoading || m.fwLoading {
						ui.Button(c, "刷新中…").Disabled(true)
						return
					}
					if ui.Button(c, "刷新").Clicked() {
						m.refreshAll()
					}
				})

				// 第 2 行：两个读数
				ui.Row(c).Gap(36).AlignItems(ui.End).Children(func() {
					ui.Column(c).Gap(2).Children(func() {
						ui.Text(c, "固件版本").FontSize(10).TextColor(colorTextFaint)
						ui.Text(c, versionText(d.FwVersion)).FontSize(20).Bold().
							TextColor(colorText)
					})

					ui.Column(c).Gap(2).Children(func() {
						ui.Text(c, "电量").FontSize(10).TextColor(colorTextFaint)
						if d.BatteryPct > 0 {
							ui.Row(c).Gap(10).AlignItems(ui.End).Children(func() {
								ui.Text(c, fmt.Sprintf("%d", d.BatteryPct)).
									FontSize(20).Bold().
									TextColor(batteryColor(d.BatteryPct))
								ui.Text(c, "%").FontSize(10).
									TextColor(colorTextMuted).Padding(0, 0, 3, 0)
								ui.Box(c).Width(150).Padding(0, 0, 0, 6).
									Children(func() {
										bar(c, float32(d.BatteryPct)/100,
											batteryColor(d.BatteryPct))
									})
							})
						} else {
							// 与版本号同字号，两列底部才对得齐
							ui.Text(c, m.batteryFallback()).FontSize(20).
								TextColor(colorTextFaint)
						}
					})
				})

				// 第 3 行：端口 · MAC
				ui.Text(c, orDash(dev.Port)+"  ·  "+orDash(d.MacHID)).
					FontSize(10).TextColor(colorTextFaint)
			})
		})

	hairline(c)
}

// batteryFallback 在电量读不到时给出原因提示。
func (m *Model) batteryFallback() string {
	if m.detailLoading {
		return "读取中…"
	}
	return "未能读取"
}

// deviceEmpty 渲染未检测到设备时的引导：给出可执行的步骤，而不只是一句提示。
func (m *Model) deviceEmpty(c *ui.Context) {
	ui.Row(c).FillWidth().Gap(20).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Grow(1).Gap(3).Children(func() {
			ui.Row(c).Gap(7).AlignItems(ui.Center).Children(func() {
				ui.Box(c).Size(5, 5).Radius(2.5).Background(colorText)
				ui.Text(c, "未检测到 Thunderstrike 手柄").
					FontSize(14).Bold().TextColor(colorText)
			})

			if m.scanErr != "" {
				ui.Text(c, "扫描出错："+m.scanErr).FontSize(11).
					TextColor(colorTextMuted).Wrap()
				return
			}

			ui.Text(c, "① 长按手柄上的电源键开机（指示灯闪烁）").
				FontSize(11).TextColor(colorTextMuted)
			ui.Text(c, "② 在 Windows 设置 → 蓝牙和其他设备 中完成配对").
				FontSize(11).TextColor(colorTextMuted)
			ui.Text(c, "③ 回到这里点击右侧「重新扫描设备」").
				FontSize(11).TextColor(colorTextMuted)
		})

		if ui.Button(c, "刷新").Clicked() {
			m.refreshAll()
		}
	})
}

// blockHint 渲染区块内的提示文案。
func (m *Model) blockHint(c *ui.Context, text string, busy bool) {
	ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
		if busy {
			ui.Spinner(c)
		}
		ui.Text(c, text).FontSize(12).TextColor(colorTextMuted)
	})
}

// firmwareBlock 渲染固件列表区：固定高度，超出部分在区内滚动。
func (m *Model) firmwareBlock(c *ui.Context) {
	ui.Row(c).FillWidth().Height(listLabelH).Padding(0, pagePadding).Gap(8).
		AlignItems(ui.Center).
		Children(func() {
			ui.Text(c, "固件").FontSize(10).TextColor(colorTextFaint).
				LetterSpacing(0.8)
			if n := len(m.firmwares); n > 0 {
				ui.Text(c, fmt.Sprintf("%d 个", n)).FontSize(10).
					TextColor(colorTextFaint)
			}
		})

	ui.Box(c).FillWidth().Height(m.listHeight(c)).Children(func() {
		if m.fwLoading || len(m.firmwares) == 0 {
			ui.Box(c).FillWidth().Padding(24, pagePadding).Children(func() {
				m.blockHint(c, m.firmwaresEmptyText(), m.fwLoading)
			})
			return
		}

		ui.Scroll(c).Fill().Children(func() {
			for i := range m.firmwares {
				m.firmwareRow(c, i)
			}
		})
	})
}

// listHeight 按当前窗口高度动态计算固件列表高度。
// 窗口拉高时列表跟着变高；其余版块都是固定高度，窗口拉高只会让列表变大。
func (m *Model) listHeight(c *ui.Context) float32 {
	_, sh := c.Size()
	h := sh - headerHeight - deviceHeight - listLabelH - actionHeight - footerHeight - 2
	if h < listRowH {
		h = listRowH
	}
	return h
}

// firmwaresEmptyText 返回固件列表为空时的提示。
func (m *Model) firmwaresEmptyText() string {
	if m.fwErr != "" {
		return "读取固件目录失败：" + m.fwErr
	}
	if m.fwLoading {
		return "正在扫描 blkz 目录…"
	}
	return "blkz/ 目录里没有固件包"
}

// firmwareRow 渲染单行固件。校验不通过的不可选。
func (m *Model) firmwareRow(c *ui.Context, i int) {
	fw := m.firmwares[i]
	blocked := fw.Checksum == "mismatch" || fw.Checksum == "error"
	selected := i == m.fwIdx && !blocked

	row := ui.Row(c).FillWidth().Height(34).Padding(0, pagePadding).Gap(12).
		AlignItems(ui.Center).Background(colorBg)

	switch {
	case selected:
		row.Background(colorBrandTint)
	case blocked:
		// 校验失败的固件不给 hover 反馈
	case row.Hovered():
		row.Background(colorPanel)
	}

	if !blocked {
		row.OnClick(func() {
			m.fwIdx = i
			m.langIdx = 0
			m.result = nil
		})
	}

	row.Children(func() {
		// 选中提示条
		accent := colorBg
		if selected {
			accent = colorBrand
		}
		ui.Box(c).Width(2).Height(16).Background(accent)

		ui.Text(c, "V"+fw.Version).FontSize(13).Bold().
			TextColor(colorText).Width(52)

		ui.Text(c, fw.Name).FontSize(11).TextColor(colorTextMuted).
			Width(300).NoWrap().Ellipsis("…")

		ui.Spacer(c)
		switch fw.Checksum {
		case "match":
			chip(c, "校验通过", colorBrandDark)
		case "mismatch":
			chip(c, "MD5 不符", colorText)
		case "unknown":
			chip(c, "未收录", colorTextMuted)
		case "error":
			chip(c, "校验出错", colorText)
		default:
			chip(c, "未知", colorTextFaint)
		}

		if fw.IsLocale && len(fw.Languages) > 0 {
			chip(c, fmt.Sprintf("%d 语言", len(fw.Languages)), colorTextMuted)
		}
	})
}

// actionBlock 渲染底部操作区：第一行是状态与按钮，第二行是进度条。
func (m *Model) actionBlock(c *ui.Context) {
	ui.Box(c).FillWidth().Height(actionHeight).Background(colorPanel).
		Children(func() {
			hairline(c)

			ui.Column(c).FillWidth().Padding(10, pagePadding).Gap(8).
				Children(func() {
					ui.Row(c).FillWidth().Gap(12).AlignItems(ui.Center).
						Children(func() {
							m.actionHeadline(c)
							ui.Spacer(c)
							m.actionButton(c)
						})

					m.actionProgress(c)
				})
		})
}

// actionHeadline 渲染操作区主文案。
func (m *Model) actionHeadline(c *ui.Context) {
	fw := m.selectedFirmware()
	// 优先用 HID 读到的版本；读不到时退化到扫描时的版本
	from := "—"
	if d := m.detail; d != nil && d.FwVersion != "" {
		from = "V" + d.FwVersion
	} else if dev := m.selectedDevice(); dev != nil && dev.FwVersion != "" {
		from = "V" + dev.FwVersion
	}

	switch {
	case m.flashing:
		if fw != nil {
			m.headlineTwoLines(c, "正在刷写", "V"+fw.Version, colorBrandDark,
				fmt.Sprintf("%s → V%s", from, fw.Version), nil)
		}

	case m.result != nil && m.result.Success:
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "✓").FontSize(12).Bold().TextColor(colorBrandDark)
			ui.Text(c, fmt.Sprintf("刷写完成  %s → V%s", from, m.result.ToVer)).
				FontSize(13).Bold().TextColor(colorBrandDark)
			ui.Text(c, "耗时 "+m.result.Elapsed).FontSize(10).
				TextColor(colorTextFaint)
		})

	case m.result != nil && !m.result.Success:
		ui.Text(c, "✗ 刷写失败："+m.result.Error).FontSize(12).
			TextColor(colorText).Wrap()

	case fw != nil:
		var extra func()
		if names := m.languageNames(fw); len(names) > 0 {
			extra = func() { m.languageSelect(c, names) }
		}
		m.headlineTwoLines(c, "待刷写", "V"+fw.Version, colorText,
			fmt.Sprintf("%s → V%s", from, fw.Version), extra)

	default:
		ui.Text(c, "选择一个固件包开始刷写").FontSize(12).
			TextColor(colorTextMuted)
	}
}

// headlineTwoLines 渲染两行式主文案：第一行是小标签，第二行是版本号。
func (m *Model) headlineTwoLines(c *ui.Context, tag, version string, verColor ui.Color,
	sub string, extra func()) {
	ui.Column(c).Gap(2).Children(func() {
		ui.Text(c, tag).FontSize(10).TextColor(colorTextFaint)

		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Text(c, version).FontSize(16).Bold().TextColor(verColor)
			if sub != "" {
				ui.Text(c, sub).FontSize(10).TextColor(colorTextMuted)
			}
			if extra != nil {
				extra()
			}
		})
	})
}

// languageNames 返回多语言固件的子包显示名；非多语言返回 nil。
func (m *Model) languageNames(fw *core.FirmwareInfo) []string {
	if fw == nil || !fw.IsLocale || len(fw.Languages) < 2 {
		return nil
	}
	names := make([]string, len(fw.Languages))
	for i, code := range fw.Languages {
		names[i] = core.LanguageDisplayName(code)
	}
	return names
}

// languageSelect 渲染子包（语言）选择下拉。
func (m *Model) languageSelect(c *ui.Context, names []string) {
	if m.langIdx <= 0 || m.langIdx >= len(names) {
		m.langIdx = 0
	}

	current := names[m.langIdx]
	ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
		ui.Text(c, "语言").FontSize(10).TextColor(colorTextFaint)
		if ui.Select(c, &current, names).Width(150).Changed() {
			for i, n := range names {
				if n == current {
					m.langIdx = i
					m.result = nil
					break
				}
			}
		}
	})
}

// actionProgress 渲染进度条行。未开始刷写时也显示，保持版式稳定。
func (m *Model) actionProgress(c *ui.Context) {
	ui.Row(c).FillWidth().Gap(12).AlignItems(ui.Center).Children(func() {
		ui.Box(c).Width(300).Children(func() {
			bar(c, float32(m.prog.Ratio()), colorBrand)
		})

		ui.Text(c, fmt.Sprintf("%d%%", m.prog.Percent)).FontSize(10).
			TextColor(colorTextMuted)

		if name := m.currentStepName(); name != "" && m.flashing {
			ui.Text(c, name).FontSize(10).TextColor(colorTextFaint)
		}

		if m.prog.TotalBytes > 0 {
			ui.Text(c, fmt.Sprintf("%s / %s",
				humanSize(m.prog.BytesWritten), humanSize(m.prog.TotalBytes))).
				FontSize(10).TextColor(colorTextFaint)
		}
	})
}

// actionButton 渲染刷写按钮，按状态切换文案与可用性。
func (m *Model) actionButton(c *ui.Context) {
	switch {
	case m.flashing:
		ui.Button(c, "刷写中…").Disabled(true)

	case m.result != nil && m.result.Success:
		if ui.Button(c, "继续刷写").Clicked() {
			m.result = nil
		}

	case m.result != nil && !m.result.Success:
		if ui.PrimaryButton(c, "重试").Clicked() {
			m.startFlash()
		}

	case !m.canFlash():
		ui.PrimaryButton(c, "开始刷写").Disabled(true)

	default:
		if ui.PrimaryButton(c, "开始刷写").Clicked() {
			m.startFlash()
		}
	}
}

// statusBar 渲染底部状态条：左侧运行状态，右侧快捷键提示。
func (m *Model) statusBar(c *ui.Context) {
	ui.Box(c).FillWidth().Height(footerHeight).Children(func() {
		hairline(c)
		ui.Row(c).FillWidth().Height(footerHeight-1).
			Padding(0, pagePadding).Gap(10).AlignItems(ui.Center).
			Children(func() {
				ui.Text(c, m.statusText()).FontSize(10).TextColor(colorTextFaint)
				ui.Spacer(c)
				ui.Text(c, m.hintText()).FontSize(10).TextColor(colorTextFaint)
			})
	})
}

// humanSize 把字节数格式化为可读大小。
func humanSize(n int) string {
	switch {
	case n <= 0:
		return "—"
	case n >= 1024*1024:
		return fmt.Sprintf("%.1f MB", float64(n)/(1024*1024))
	default:
		return fmt.Sprintf("%d KB", n/1024)
	}
}
