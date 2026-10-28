package gui

import (
	"fmt"

	"github.com/egoist/mygo/ui"

	"thunderstrike-controller-tool/src/core"
)

// viewDevices 渲染设备列表页。只有多设备时才需要用户在这里做选择，
// 单设备场景由 refreshDevices 直接跳到设备信息页。
func (m *Model) viewDevices(c *ui.Context) {
	hint := "手柄需在 Windows 设置中配对并连接"
	if m.scanning {
		hint = "正在扫描蓝牙 SPP 设备…"
	}

	sectionHead(c, "设备", hint, func() {
		if ui.Button(c, "刷新").Clicked() {
			m.refreshDevices()
		}
	})

	ui.Box(c).Height(12)

	if m.scanErr != "" {
		ui.Box(c).FillWidth().Padding(8, 10).Radius(4).
			Background(colorPanelAlt).
			Children(func() {
				ui.Text(c, "扫描出错："+m.scanErr).FontSize(11).
					TextColor(colorDanger).Wrap()
			})
		ui.Box(c).Height(10)
	}

	if m.scanning || len(m.devices) == 0 {
		m.devicesEmpty(c)
		return
	}

	ui.Column(c).FillWidth().Gap(4).Children(func() {
		for i := range m.devices {
			m.deviceRow(c, i)
		}
	})
}

// devicesEmpty 渲染扫描中或未发现设备的空状态。
func (m *Model) devicesEmpty(c *ui.Context) {
	ui.Column(c).FillWidth().Padding(56, 0).Center().Gap(8).Children(func() {
		if m.scanning {
			ui.Spinner(c)
			ui.Text(c, "正在扫描…").FontSize(12).TextColor(colorTextMuted)
			return
		}
		ui.Text(c, "未发现手柄").FontSize(13).TextColor(colorTextMuted)
		ui.Text(c, "确认手柄已开机，并与本机完成配对").
			FontSize(11).TextColor(colorTextFaint)
	})
}

// deviceRow 渲染单个设备行，整行可点击。
func (m *Model) deviceRow(c *ui.Context, i int) {
	d := m.devices[i]
	selected := i == m.devIdx

	row := ui.Row(c).FillWidth().Height(52).Padding(0, 12).Gap(12).
		Background(colorPanel).Radius(4).AlignItems(ui.Center)

	switch {
	case selected:
		row.Background(colorPanelAlt).Border(1, colorBrandTrack)
	case row.Hovered():
		row.Background(colorPanelAlt)
	}

	row.OnClick(func() {
		m.devIdx = i
		m.openDevice(d)
	})

	row.Children(func() {
		// 左侧状态条：在线亮绿，休眠暗
		accent := colorTrack
		if d.Online() {
			accent = colorBrand
		}
		ui.Box(c).Width(2).Height(22).Radius(1).Background(accent)

		ui.Column(c).Gap(3).Width(280).Children(func() {
			ui.Text(c, d.DeviceName).FontSize(13).TextColor(colorText)
			ui.Text(c, d.MAC).FontSize(10).TextColor(colorTextFaint)
		})

		ui.Text(c, d.Port).FontSize(12).TextColor(colorTextMuted).Width(58)

		if d.Online() {
			ui.Text(c, "V"+d.FwVersion).FontSize(12).
				TextColor(colorTextMuted).Width(54)
			chip(c, "在线", colorBrand)
		} else {
			ui.Text(c, "—").FontSize(12).TextColor(colorTextFaint).Width(54)
			chip(c, "休眠", colorWarning)
		}

		ui.Spacer(c)
		ui.Text(c, "›").FontSize(15).TextColor(colorTextFaint)
	})
}

// deviceSubtitle 返回设备信息页的副标题。
func deviceSubtitle(d core.DeviceInfo) string {
	name := d.DeviceName
	if name == "" {
		name = "未知设备"
	}
	if d.Port == "" {
		return name
	}
	return fmt.Sprintf("%s  ·  %s  ·  %s", name, d.Port, d.MAC)
}
