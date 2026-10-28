package gui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"

	"thunderstrike-controller-tool/src/core"
)

// viewFirmwares 渲染固件列表页。行高压缩到 52，版本号是每行唯一的粗体，
// 文件名退为次要信息。
func (m *Model) viewFirmwares(c *ui.Context) {
	hint := "固件包位于程序目录的 blkz/ 中"
	if m.fwLoading {
		hint = "正在扫描 blkz 目录…"
	}

	sectionHead(c, "选择固件", hint, func() {
		if ui.Button(c, "重新扫描").Clicked() {
			m.openFirmwares()
		}
	})

	ui.Box(c).Height(12)

	if m.fwErr != "" {
		ui.Box(c).FillWidth().Padding(8, 10).Radius(4).
			Background(colorPanelAlt).
			Children(func() {
				ui.Text(c, "读取固件目录失败："+m.fwErr).FontSize(11).
					TextColor(colorDanger).Wrap()
			})
		ui.Box(c).Height(10)
	}

	if m.fwLoading || len(m.firmwares) == 0 {
		ui.Column(c).FillWidth().Padding(52, 0).Center().Gap(8).Children(func() {
			if m.fwLoading {
				ui.Spinner(c)
				ui.Text(c, "正在扫描…").FontSize(12).TextColor(colorTextMuted)
				return
			}
			ui.Text(c, "没有可用的固件包").FontSize(13).TextColor(colorTextMuted)
			ui.Text(c, "把 .blkz 文件放进 blkz/ 后点击「重新扫描」").
				FontSize(11).TextColor(colorTextFaint)
		})
		return
	}

	ui.Column(c).FillWidth().Gap(4).Children(func() {
		for i := range m.firmwares {
			m.firmwareRow(c, i)
		}
	})
}

// firmwareRow 渲染单个固件行。校验不通过的固件不可点击。
func (m *Model) firmwareRow(c *ui.Context, i int) {
	fw := m.firmwares[i]
	blocked := fw.Checksum == "mismatch" || fw.Checksum == "error"
	selected := i == m.fwIdx && !blocked

	row := ui.Row(c).FillWidth().Height(52).Padding(0, 12).Gap(12).
		Background(colorPanel).Radius(4).AlignItems(ui.Center)

	switch {
	case selected:
		row.Background(colorPanelAlt).Border(1, colorBrandTrack)
	case blocked:
		// 校验失败的固件保持暗色，不给 hover 反馈
	case row.Hovered():
		row.Background(colorPanelAlt)
	}

	if !blocked {
		row.OnClick(func() {
			m.fwIdx = i
			m.chooseFirmware(fw)
		})
	}

	row.Children(func() {
		ui.Text(c, "V"+fw.Version).FontSize(14).Bold().
			TextColor(colorText).Width(58)

		ui.Column(c).Gap(3).Grow(1).Children(func() {
			ui.Text(c, fw.Name).FontSize(12).TextColor(colorTextMuted).
				Ellipsis("…").NoWrap()
			ui.Text(c, fmt.Sprintf("%s · 0x%s", humanSize(fw.OtaSize), orDash(fw.VersionHex))).
				FontSize(10).TextColor(colorTextFaint)
		})

		switch fw.Checksum {
		case "match":
			chip(c, "校验通过", colorBrand)
		case "mismatch":
			chip(c, "MD5 不符", colorDanger)
		case "unknown":
			chip(c, "未收录", colorWarning)
		case "error":
			chip(c, "校验出错", colorDanger)
		default:
			chip(c, "未知", colorTextFaint)
		}

		if fw.IsLocale && len(fw.Languages) > 0 {
			chip(c, fmt.Sprintf("%d 语言", len(fw.Languages)), colorTextMuted)
		}

		// 低于设备当前版本时标一个小黄点，不占用徽章位置
		if m.selected.FwVersion != "" && versionLower(fw.Version, m.selected.FwVersion) {
			ui.Box(c).Size(5, 5).Radius(2.5).Background(colorWarning).
				Tooltip("降级：低于当前固件版本")
		}

		if blocked {
			ui.Text(c, "×").FontSize(14).TextColor(colorTextFaint)
		} else {
			ui.Text(c, "›").FontSize(15).TextColor(colorTextFaint)
		}
	})
}

// viewLanguages 渲染多语言固件的语言选择页。
func (m *Model) viewLanguages(c *ui.Context) {
	fw := m.pendingFw

	sectionHead(c, "选择语言",
		fmt.Sprintf("%s  ·  V%s", fw.Name, fw.Version), nil)

	ui.Box(c).Height(12)

	if len(fw.Languages) == 0 {
		ui.Text(c, "该固件包没有可用的语言条目").
			FontSize(12).TextColor(colorTextMuted)
		return
	}

	ui.Column(c).FillWidth().Gap(4).Children(func() {
		for i, lang := range fw.Languages {
			idx := i
			code := lang

			row := ui.Row(c).FillWidth().Height(44).Padding(0, 12).Gap(12).
				Background(colorPanel).Radius(4).AlignItems(ui.Center)

			switch {
			case idx == m.langIdx:
				row.Background(colorPanelAlt).Border(1, colorBrandTrack)
			case row.Hovered():
				row.Background(colorPanelAlt)
			}

			row.OnClick(func() {
				m.langIdx = idx
				m.chooseLanguage(idx)
			})

			row.Children(func() {
				ui.Text(c, core.LanguageDisplayName(code)).
					FontSize(12).TextColor(colorText)
				ui.Text(c, code).FontSize(10).TextColor(colorTextFaint)
				ui.Spacer(c)
				ui.Text(c, "›").FontSize(15).TextColor(colorTextFaint)
			})
		}
	})
}

// humanSize 把字节数格式化为可读大小。
func humanSize(n int) string {
	switch {
	case n <= 0:
		return "大小未知"
	case n >= 1024*1024:
		return fmt.Sprintf("%.1f MB", float64(n)/(1024*1024))
	default:
		return fmt.Sprintf("%d KB", n/1024)
	}
}

// versionLower 判断版本 a 是否低于 b（按点分十进制逐段比较）。
func versionLower(a, b string) bool {
	as := strings.Split(a, ".")
	bs := strings.Split(b, ".")
	for i := 0; i < len(as) && i < len(bs); i++ {
		ai, errA := strconv.Atoi(as[i])
		bi, errB := strconv.Atoi(bs[i])
		if errA != nil || errB != nil {
			return false
		}
		if ai != bi {
			return ai < bi
		}
	}
	return len(as) < len(bs)
}
