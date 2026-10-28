package gui

import (
	"fmt"

	"github.com/egoist/mygo/ui"
)

// viewDevices 渲染设备列表页（含首次扫描状态与空状态）。
func (m *Model) viewDevices(c *ui.Context) {
	sectionTitle(c, "选择设备", "请确保手柄已在 Windows 蓝牙设置中配对并连接（使用 SPP 通道刷写）")

	if m.scanning {
		card(c).FillWidth().MarginY(12).Children(func() {
			ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
				ui.Spinner(c)
				ui.Text(c, "正在扫描蓝牙 SPP 设备…").FontSize(13).TextColor(colorMuted)
			})
		})
		return
	}

	if m.scanErr != "" {
		ui.Box(c).MarginY(12).Children(func() {
			noticeErr(c, "扫描出错："+m.scanErr)
		})
	}

	if len(m.devices) == 0 {
		card(c).FillWidth().MarginY(12).Children(func() {
			ui.Column(c).Gap(6).Center().PaddingY(18).Children(func() {
				ui.Text(c, "未发现 Thunderstrike 手柄").FontSize(15).Bold()
				ui.Text(c, "请确认手柄已开机并与电脑配对，然后点击右下角「刷新设备」").
					FontSize(13).TextColor(colorMuted)
			})
		})
		return
	}

	ui.Column(c).FillWidth().Gap(6).MarginY(12).Children(func() {
		for i := range m.devices {
			m.deviceRow(c, i)
		}
	})
}

// deviceRow 渲染单个设备行，整行可点击进入设备信息页。
func (m *Model) deviceRow(c *ui.Context, i int) {
	d := m.devices[i]

	row := ui.Row(c).FillWidth().Padding(10, 12).Gap(14).
		Background(c.Theme().Surface).Radius(10).Border(1, colorLine).
		AlignItems(ui.Center)

	if i == m.devIdx {
		row.Background(colorRowSel).BorderColor(colorBrand).ScrollIntoView()
	} else if row.Hovered() {
		row.Background(colorRowHover)
	}

	row.OnClick(func() {
		m.devIdx = i
		m.openDevice(d)
	})
	row.Children(func() {
		ui.Box(c).Size(26, 26).Radius(13).Background(chipNeBg).Center().
			Children(func() {
				ui.Text(c, fmt.Sprint(d.Index)).FontSize(12).Bold().TextColor(colorMuted)
			})

		ui.Column(c).Gap(2).Width(320).Children(func() {
			ui.Text(c, d.DeviceName).FontSize(14).Bold().TextColor(c.Theme().Text)
			ui.Text(c, d.MAC).FontSize(12).TextColor(colorMuted)
		})

		ui.Text(c, d.Port).FontSize(13).TextColor(c.Theme().Text).Width(90)

		if d.Online() {
			ui.Text(c, "V"+d.FwVersion).FontSize(13).TextColor(c.Theme().Text).Width(80)
			chipOK(c, "在线")
		} else {
			ui.Text(c, "—").FontSize(13).TextColor(colorMuted).Width(80)
			chipWarn(c, "休眠")
		}

		ui.Spacer(c)
		ui.Text(c, "查看详情 ›").FontSize(13).TextColor(colorBrand)
	})
}
