package gui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"

	"thunderstrike-controller-tool/core"
)

// viewFirmwares 渲染固件列表页：blkz 包、版本、大小与校验状态。
func (m *Model) viewFirmwares(c *ui.Context) {
	sectionTitle(c, "选择固件", "固件包位于程序目录下的 blkz/ 中，校验状态来自内置 MD5 记录")

	if m.fwLoading {
		card(c).FillWidth().MarginY(12).Children(func() {
			ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
				ui.Spinner(c)
				ui.Text(c, "正在扫描 blkz 固件目录…").FontSize(13).TextColor(colorMuted)
			})
		})
		return
	}

	if m.fwErr != "" {
		ui.Box(c).FillWidth().MarginY(12).Children(func() {
			noticeErr(c, "读取固件目录失败："+m.fwErr)
		})
	}

	if len(m.firmwares) == 0 {
		card(c).FillWidth().MarginY(12).Children(func() {
			ui.Column(c).Gap(6).Center().PaddingY(18).Children(func() {
				ui.Text(c, "没有可用的固件包").FontSize(15).Bold()
				ui.Text(c, "请把 .blkz 固件包放到程序目录的 blkz/ 文件夹后重新扫描").
					FontSize(13).TextColor(colorMuted)
			})
		})
		return
	}

	ui.Column(c).FillWidth().Gap(6).MarginY(12).Children(func() {
		for i := range m.firmwares {
			m.firmwareRow(c, i)
		}
	})
}

// firmwareRow 渲染单个固件行。校验不通过的固件不可点击。
func (m *Model) firmwareRow(c *ui.Context, i int) {
	fw := m.firmwares[i]
	blocked := fw.Checksum == "mismatch" || fw.Checksum == "error"

	row := ui.Row(c).FillWidth().Padding(10, 12).Gap(14).
		Background(c.Theme().Surface).Radius(10).Border(1, colorLine).
		AlignItems(ui.Center)

	switch {
	case i == m.fwIdx && !blocked:
		row.Background(colorRowSel).BorderColor(colorBrand).ScrollIntoView()
	case blocked:
		row.BorderColor(chipErrBg)
	case row.Hovered():
		row.Background(colorRowHover)
	}

	if !blocked {
		row.OnClick(func() {
			m.fwIdx = i
			m.chooseFirmware(fw)
		})
	}

	row.Children(func() {
		ui.Column(c).Gap(2).Width(360).Children(func() {
			ui.Text(c, fw.Name).FontSize(14).Bold().TextColor(c.Theme().Text).
				Ellipsis("…")
			ui.Text(c, fmt.Sprintf("%s · %s", humanSize(fw.OtaSize), versionHex(fw.VersionHex))).
				FontSize(12).TextColor(colorMuted)
		})

		ui.Text(c, "V"+fw.Version).FontSize(14).Bold().
			TextColor(c.Theme().Text).Width(80)

		// 校验状态
		switch fw.Checksum {
		case "match":
			chipOK(c, "校验通过")
		case "mismatch":
			chipErr(c, "MD5 不符")
		case "unknown":
			chipWarn(c, "未收录")
		case "error":
			chipErr(c, "校验出错")
		default:
			chipNeutral(c, "未知")
		}

		// 多语言固件标记
		if fw.IsLocale && len(fw.Languages) > 0 {
			chipNeutral(c, fmt.Sprintf("%d 种语言", len(fw.Languages)))
		}

		// 与当前设备版本比较
		if m.selected.FwVersion != "" && versionLower(fw.Version, m.selected.FwVersion) {
			chipWarn(c, "降级")
		}

		ui.Spacer(c)

		if blocked {
			ui.Text(c, "已禁用").FontSize(13).TextColor(colorDanger)
		} else {
			ui.Text(c, "刷写 ›").FontSize(13).TextColor(colorBrand)
		}
	})
}

// viewLanguages 渲染多语言固件的语言选择页。
func (m *Model) viewLanguages(c *ui.Context) {
	fw := m.pendingFw
	sectionTitle(c, "选择语言版本",
		fmt.Sprintf("%s  ·  V%s  ·  多语言固件", fw.Name, fw.Version))

	if len(fw.Languages) == 0 {
		card(c).FillWidth().MarginY(12).Children(func() {
			ui.Text(c, "该固件包没有可用的语言条目").FontSize(13).TextColor(colorMuted)
		})
		return
	}

	ui.Column(c).FillWidth().Gap(6).MarginY(12).Children(func() {
		for i, lang := range fw.Languages {
			idx := i
			code := lang

			row := ui.Row(c).FillWidth().Padding(10, 12).Gap(14).
				Background(c.Theme().Surface).Radius(10).Border(1, colorLine).
				AlignItems(ui.Center)

			switch {
			case idx == m.langIdx:
				row.Background(colorRowSel).BorderColor(colorBrand).ScrollIntoView()
			case row.Hovered():
				row.Background(colorRowHover)
			}

			row.OnClick(func() {
				m.langIdx = idx
				m.chooseLanguage(idx)
			})
			row.Children(func() {
				ui.Text(c, core.LanguageDisplayName(code)).
					FontSize(14).TextColor(c.Theme().Text)
				ui.Spacer(c)
				ui.Text(c, "刷写此版本 ›").FontSize(13).TextColor(colorBrand)
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

// versionHex 返回带 0x 前缀的十六进制版本号。
func versionHex(hex string) string {
	if hex == "" {
		return "0x----"
	}
	return "0x" + hex
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
