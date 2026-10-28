package gui

import (
	"fmt"

	"github.com/egoist/mygo/ui"

	"thunderstrike-controller-tool/src/core"
)

// viewDetail 渲染设备信息页：HID 查询到的版本、序列号与电量。
func (m *Model) viewDetail(c *ui.Context) {
	dev := m.selected
	sectionTitle(c, "设备信息", fmt.Sprintf("%s · %s · %s", dev.DeviceName, dev.MAC, dev.Port))

	if m.detailLoading {
		card(c).FillWidth().MarginY(12).Children(func() {
			ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
				ui.Spinner(c)
				ui.Text(c, "正在通过 HID 读取设备信息…").FontSize(13).TextColor(colorMuted)
			})
		})
	}

	d := m.detail
	if d == nil {
		d = &core.DeviceDetail{DeviceName: dev.DeviceName, MAC: dev.MAC}
	}

	fwVer := "读取失败"
	if d.FwVersion != "" {
		fwVer = fmt.Sprintf("V%s (%s)", d.FwVersion, d.FwVersionHex)
	}

	ui.Row(c).FillWidth().Gap(14).MarginY(12).AlignItems(ui.Start).Children(func() {
		// 左栏：基本信息
		card(c).Grow(1).Children(func() {
			ui.Text(c, "基本信息").FontSize(12).Bold().TextColor(colorMuted).
				Padding(0, 0, 6, 0)
			infoRow(c, "设备名称", d.DeviceName)
			infoRow(c, "MAC 地址", d.MAC)
			infoRow(c, "MAC (HID)", d.MacHID)
			infoRow(c, "软件版本", fwVer)
			infoRow(c, "蓝牙芯片", d.CsrVersion)
			infoRow(c, "热词引擎", d.HotwordVer)
			infoRow(c, "序列号", d.Serial)
			infoRow(c, "传输方式", "蓝牙 HID + SPP")
		})

		// 右栏：电量
		card(c).Width(300).Children(func() {
			ui.Text(c, "电池").FontSize(12).Bold().TextColor(colorMuted).
				Padding(0, 0, 6, 0)

			if d.BatteryPct <= 0 {
				ui.Text(c, "未读取到电量").FontSize(13).TextColor(colorMuted)
				return
			}

			ui.Row(c).Gap(8).AlignItems(ui.End).Children(func() {
				ui.Text(c, fmt.Sprintf("%d", d.BatteryPct)).
					FontSize(34).Bold().TextColor(batteryColor(d.BatteryPct))
				ui.Text(c, "%").FontSize(15).TextColor(batteryColor(d.BatteryPct)).
					Padding(0, 0, 6, 0)
			})

			ui.Box(c).PaddingY(8).Children(func() {
				bar(c, float32(d.BatteryPct)/100, batteryColor(d.BatteryPct))
			})

			if d.ReservePower {
				ui.Box(c).Padding(6, 0, 0, 0).Children(func() {
					chipWarn(c, "储备电量")
				})
			}
		})
	})

	if d.BatteryPct > 0 && d.BatteryPct < 20 {
		ui.Box(c).FillWidth().Children(func() {
			noticeWarn(c, "电量低于 20%：刷写过程中断电可能导致手柄变砖，建议先充满电再刷写。")
		})
	}
}

// batteryColor 按电量返回颜色：80 以上绿、20 以下红、其余琥珀。
func batteryColor(pct int) ui.Color {
	switch {
	case pct >= 80:
		return colorBrand
	case pct < 20:
		return colorDanger
	default:
		return colorWarning
	}
}
