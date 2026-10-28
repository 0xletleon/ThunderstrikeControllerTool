package gui

import (
	"fmt"

	"github.com/egoist/mygo/ui"

	"thunderstrike-controller-tool/src/core"
)

// viewDetail 渲染设备信息页。核心只有两个数字：固件版本与电量，
// 其余信息（芯片、热词、序列号）降为右侧小字。
func (m *Model) viewDetail(c *ui.Context) {
	d := m.detail
	if d == nil {
		d = &core.DeviceDetail{
			DeviceName: m.selected.DeviceName,
			MAC:        m.selected.MAC,
		}
	}

	sectionHead(c, "设备", deviceSubtitle(m.selected), func() {
		if ui.PrimaryButton(c, "刷写固件").Clicked() {
			m.openFirmwares()
		}
	})

	ui.Box(c).Height(6)

	if m.detailLoading {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Spinner(c)
			ui.Text(c, "正在读取设备信息…").FontSize(11).TextColor(colorTextFaint)
		})
	}

	ui.Box(c).Height(12)

	ui.Row(c).FillWidth().Gap(28).AlignItems(ui.Start).Children(func() {
		// 左栏：两个关键读数
		ui.Column(c).Grow(1).Children(func() {
			ui.Text(c, "软件版本").FontSize(11).TextColor(colorTextFaint)
			stat(c, "V"+orDash(d.FwVersion), "", colorText)
			label(c, "0x"+orDash(d.FwVersionHex))

			ui.Box(c).Height(20)

			ui.Text(c, "电量").FontSize(11).TextColor(colorTextFaint)
			if d.BatteryPct > 0 {
				stat(c, fmt.Sprintf("%d", d.BatteryPct), "%",
					batteryColor(d.BatteryPct))
				ui.Box(c).Width(220).Padding(4, 0, 0, 0).Children(func() {
					bar(c, float32(d.BatteryPct)/100, batteryColor(d.BatteryPct))
				})
			} else {
				ui.Text(c, "未能读取").FontSize(18).TextColor(colorTextFaint)
			}
		})

		// 右栏：次要明细，两列小字
		ui.Column(c).Width(300).Gap(14).Children(func() {
			kvPair(c, "蓝牙芯片", orDash(d.CsrVersion), "热词引擎", orDash(d.HotwordVer))
			kvPair(c, "序列号", orDash(d.Serial), "MAC (HID)", orDash(d.MacHID))
			kvPair(c, "端口", orDash(m.selected.Port), "传输", "蓝牙 HID + SPP")
		})
	})

	if d.BatteryPct > 0 && d.BatteryPct < 20 {
		ui.Box(c).Height(18)
		ui.Row(c).FillWidth().Gap(8).Padding(8, 10).Radius(4).
			Background(colorPanelAlt).
			AlignItems(ui.Center).
			Children(func() {
				ui.Box(c).Width(2).Height(16).Radius(1).Background(colorWarning)
				ui.Text(c, "电量低于 20%，刷写过程中断电可能导致变砖，建议先充满电").
					FontSize(11).TextColor(colorWarning).Wrap()
			})
	}
}

// kvPair 渲染并排的两组「标签 / 值」。
func kvPair(c *ui.Context, k1, v1, k2, v2 string) {
	ui.Row(c).FillWidth().Gap(16).Children(func() {
		ui.Column(c).Grow(1).Gap(2).Children(func() {
			ui.Text(c, k1).FontSize(10).TextColor(colorTextFaint)
			ui.Text(c, v1).FontSize(12).TextColor(colorTextMuted).Selectable()
		})
		ui.Column(c).Grow(1).Gap(2).Children(func() {
			ui.Text(c, k2).FontSize(10).TextColor(colorTextFaint)
			ui.Text(c, v2).FontSize(12).TextColor(colorTextMuted).Selectable()
		})
	})
}

// orDash 在字符串为空时返回占位符。
func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
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
