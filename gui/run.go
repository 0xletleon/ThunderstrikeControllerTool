package gui

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"thunderstrike-controller-tool/core"
)

// appVersion 显示在页头的版本号。
const appVersion = "V0.4"

// screen 表示当前显示的页面。
type screen int

const (
	screenScanning  screen = iota // 首次扫描中
	screenDevices                 // 设备列表
	screenDetail                  // 设备信息
	screenFirmwares               // 固件列表
	screenLanguages               // 多语言固件语言选择
	screenFlashing                // 刷写进度与结果
)

// Model 是 GUI 的全部界面状态。
//
// 所有字段只在主线程读写：View 函数、控件回调，以及后台任务通过
// Window.Update 投递的回调。后台 goroutine 不直接改这些字段。
type Model struct {
	svc core.Services
	win *mygo.Window

	dark     bool // 系统是否为暗色外观（首次建帧时锁定）
	themeSet bool
	page     screen

	// 设备扫描
	devices  []core.DeviceInfo
	devIdx   int
	scanning bool
	scanErr  string

	// 选中设备与 HID 详情
	selected      core.DeviceInfo
	detail        *core.DeviceDetail
	detailLoading bool

	// 固件列表
	firmwares []core.FirmwareInfo
	fwIdx     int
	fwLoading bool
	fwErr     string

	// 语言选择
	langIdx   int
	pendingFw core.FirmwareInfo

	// 刷写
	steps    []core.FlashStep
	curStep  int
	prog     core.FlashProgressInfo
	flashing bool
	result   *core.FlashResultInfo

	// 手柄输入指示器
	padReady  bool // HID 句柄已打开（设备在线）
	padActive bool // 最近 400ms 内有输入
	padHeard  bool // 本次运行是否收到过任何手柄输入
	padShown  atomic.Bool
	padLastNs atomic.Int64
}

// Run 创建主窗口并运行 GUI，直到用户关闭窗口。
func Run(svc core.Services) error {
	m := &Model{svc: svc, page: screenScanning}

	mygo.App.WhenReady(func() {
		m.win = mygo.NewWindow(mygo.WindowOptions{
			Title:           "Thunderstrike Controller Tool",
			Width:           920,
			Height:          640,
			MinWidth:        820,
			MinHeight:       540,
			BackgroundColor: "#F2F3F5",
			StateKey:        "main",
			Content:         ui.View(m.View),
		})
		m.start()
	})

	if err := mygo.App.Run(); err != nil {
		return err
	}

	// 窗口已关闭，停止后台监听
	if svc.Monitor != nil {
		svc.Monitor.StopMonitoring()
	}
	return nil
}

// post 把 fn 投递到主线程执行并重绘。窗口尚未建立（如单元测试）时直接执行。
func (m *Model) post(fn func()) {
	if m.win == nil {
		fn()
		return
	}
	// 窗口可能在监听 goroutine 回调前已关闭，此时 Update 会失败，忽略即可。
	defer func() { _ = recover() }()
	m.win.Update(fn)
}

// start 在窗口建立后启动手柄监听与首次设备扫描。
func (m *Model) start() {
	if m.svc.Monitor != nil {
		m.svc.Monitor.StartMonitoring(m.onGamepadInput, m.onGamepadReady)
	}
	m.refreshDevices()
}

// View 构建整窗界面：页头 + 页面内容 + 底部操作栏。
func (m *Model) View(c *ui.Context) {
	if !m.themeSet {
		m.dark = c.Theme().Dark
		m.themeSet = true
	}
	c.SetTheme(buildTheme(m.dark))
	theme := c.Theme()

	// 先处理键盘，再按新状态构建本帧。
	m.handleKeys(c)

	// 显式给出根尺寸：内部大量 FillWidth / Grow 依赖确定的父级宽度。
	w, h := c.Size()
	ui.Column(c).Width(w).Height(h).Background(theme.Background).Children(func() {
		m.header(c)

		// 内容超过可视高度时可滚动，窗口大小不影响内容可达性。
		ui.Box(c).Grow(1).FillWidth().Children(func() {
			ui.Scroll(c).Fill().Padding(16, 20).Children(func() {
				switch m.page {
				case screenScanning, screenDevices:
					m.viewDevices(c)
				case screenDetail:
					m.viewDetail(c)
				case screenFirmwares:
					m.viewFirmwares(c)
				case screenLanguages:
					m.viewLanguages(c)
				case screenFlashing:
					m.viewFlash(c)
				}
			})
		})

		m.footer(c)
	})
}

// header 渲染页头：品牌标识、标题与手柄活动指示器。
func (m *Model) header(c *ui.Context) {
	theme := c.Theme()

	ui.Row(c).FillWidth().Padding(12, 20).Gap(14).
		Background(theme.Surface).
		BorderWidth(0, 0, 1, 0).BorderColor(colorLine).
		AlignItems(ui.Center).
		Children(func() {
			ui.Box(c).Size(42, 42).Radius(10).Background(colorBrand).Center().
				Children(func() {
					ui.Text(c, "TS").Bold().FontSize(16).TextColor(ui.Hex("#FFFFFF"))
				})

			ui.Column(c).Gap(3).Children(func() {
				ui.Text(c, "Thunderstrike Controller Tool").
					FontSize(18).Bold().TextColor(theme.Text)
				ui.Text(c, "NVIDIA SHIELD TV 2017 手柄固件管理").
					FontSize(12).TextColor(colorMuted)
			})

			ui.Spacer(c)

			m.padIndicator(c)
			ui.Text(c, "@0xletleon "+appVersion).FontSize(12).TextColor(colorMuted)
		})
}

// padIndicator 渲染手柄输入指示器：设备在线变灰绿，收到输入时点亮。
func (m *Model) padIndicator(c *ui.Context) {
	dot := ui.Hex("#C9CDD3")
	label := "等待手柄"
	bg := chipNeBg
	switch {
	case m.padActive:
		dot = colorBrand
		label = "手柄活动"
		bg = chipOkBg
	case m.padReady:
		dot = colorSuccess
		label = "手柄就绪"
	}

	ui.Row(c).Gap(6).Padding(5, 10).Radius(10).Background(bg).
		AlignItems(ui.Center).
		Children(func() {
			ui.Box(c).Size(8, 8).Radius(4).Background(dot)
			ui.Text(c, label).FontSize(12).TextColor(colorMuted)
		})
}

// footer 渲染底部操作栏，按钮随页面变化。
func (m *Model) footer(c *ui.Context) {
	footerBar(c, m.statusText(), m.hintText(), func() {
		switch m.page {
		case screenScanning, screenDevices:
			if ui.Button(c, "刷新设备").Clicked() {
				m.refreshDevices()
			}

		case screenDetail:
			if ui.Button(c, "返回设备列表").Clicked() {
				m.page = screenDevices
			}
			if ui.PrimaryButton(c, "刷写固件").Clicked() {
				m.openFirmwares()
			}

		case screenFirmwares:
			if ui.Button(c, "返回设备信息").Clicked() {
				m.page = screenDetail
			}
			if ui.Button(c, "重新扫描").Clicked() {
				m.openFirmwares()
			}

		case screenLanguages:
			if ui.Button(c, "返回固件列表").Clicked() {
				m.page = screenFirmwares
			}

		case screenFlashing:
			if m.result != nil {
				if ui.PrimaryButton(c, "返回设备列表").Clicked() {
					m.resetToDevices()
				}
			}
		}
	})
}

// statusText 返回底部状态栏文案。
func (m *Model) statusText() string {
	text := m.pageStatus()
	if m.padHeard {
		text += "  ·  已收到手柄输入"
	}
	return text
}

// pageStatus 返回当前页面的基础状态文案。
func (m *Model) pageStatus() string {
	switch m.page {
	case screenScanning:
		return "正在扫描蓝牙 SPP 设备…"
	case screenDevices:
		if m.scanning {
			return "正在扫描蓝牙 SPP 设备…"
		}
		return fmt.Sprintf("共 %d 个蓝牙设备", len(m.devices))
	case screenDetail:
		if m.detailLoading {
			return "正在通过 HID 读取设备信息…"
		}
		return "设备： " + m.selected.DeviceName
	case screenFirmwares:
		if m.fwLoading {
			return "正在扫描 blkz 固件目录…"
		}
		return fmt.Sprintf("共 %d 个固件包", len(m.firmwares))
	case screenLanguages:
		return "请选择要刷写的语言版本"
	case screenFlashing:
		if m.result != nil {
			return "刷写流程已结束"
		}
		return "刷写中，请勿操作手柄"
	}
	return ""
}

// --- 业务流程 ---

// refreshDevices 在后台扫描蓝牙设备，完成后回到主线程更新界面。
func (m *Model) refreshDevices() {
	m.scanning = true
	m.scanErr = ""
	m.page = screenScanning

	go func() {
		devices, err := m.svc.Scanner.ScanDevices()
		m.post(func() {
			m.scanning = false
			m.page = screenDevices
			if err != nil {
				m.scanErr = err.Error()
				m.devices = nil
				return
			}
			m.devices = devices
			if m.devIdx >= len(devices) {
				m.devIdx = 0
			}
		})
	}()
}

// openDevice 读取选中设备的 HID 详情并进入设备信息页。
func (m *Model) openDevice(d core.DeviceInfo) {
	m.selected = d
	m.detail = nil
	m.detailLoading = true
	m.page = screenDetail

	go func() {
		detail, err := m.svc.Scanner.QueryDeviceDetail(d)
		m.post(func() {
			m.detailLoading = false
			if err != nil {
				m.detail = &core.DeviceDetail{DeviceName: d.DeviceName, MAC: d.MAC}
				return
			}
			m.detail = detail
		})
	}()
}

// openFirmwares 扫描固件目录并进入固件列表页。
func (m *Model) openFirmwares() {
	m.page = screenFirmwares
	m.fwLoading = true
	m.fwErr = ""
	m.firmwares = nil

	go func() {
		list, err := m.svc.Firmwares.ScanFirmwares()
		m.post(func() {
			m.fwLoading = false
			if err != nil {
				m.fwErr = err.Error()
				return
			}
			m.firmwares = list
			if m.fwIdx >= len(list) {
				m.fwIdx = 0
			}
		})
	}()
}

// chooseFirmware 处理固件选择：多语言固件先选语言，其余直接开刷。
func (m *Model) chooseFirmware(fw core.FirmwareInfo) {
	if fw.IsLocale && len(fw.Languages) > 1 {
		m.pendingFw = fw
		m.langIdx = 0
		m.page = screenLanguages
		return
	}
	m.startFlash(fw)
}

// chooseLanguage 选定语言版本后开始刷写。
func (m *Model) chooseLanguage(i int) {
	fw := m.pendingFw
	fw.LanguageIndex = i
	m.startFlash(fw)
}

// startFlash 在后台执行刷写，通过回调把进度投递回主线程。
func (m *Model) startFlash(fw core.FirmwareInfo) {
	m.pendingFw = fw
	m.steps = core.NewFlashSteps()
	m.curStep = 0
	m.prog = core.FlashProgressInfo{}
	m.result = nil
	m.flashing = true
	m.page = screenFlashing

	device := m.selected
	var lastPush atomic.Int64

	// 进度回调可能非常密集（每 1KB 一次），节流到约 25 次/秒。
	pushProgress := func(written, total int) {
		now := time.Now().UnixNano()
		if written < total && now-lastPush.Load() < int64(40*time.Millisecond) {
			return
		}
		lastPush.Store(now)
		pct := 0
		if total > 0 {
			pct = written * 100 / total
		}
		m.post(func() {
			m.prog = core.FlashProgressInfo{
				BytesWritten: written,
				TotalBytes:   total,
				Percent:      pct,
			}
		})
	}

	go func() {
		result, err := m.svc.Flasher.ExecuteFlash(device, fw,
			func(step, total int, name string) {
				m.post(func() {
					m.curStep = step
					if step >= 2 && step-2 < len(m.steps) {
						m.steps[step-2].Done = true
					}
				})
			},
			pushProgress,
		)

		m.post(func() {
			m.flashing = false
			switch {
			case err != nil:
				m.result = &core.FlashResultInfo{Success: false, Error: err.Error()}
			case result == nil:
				m.result = &core.FlashResultInfo{Success: false, Error: "刷写器未返回结果"}
			default:
				m.result = result
				if result.Success {
					for i := range m.steps {
						m.steps[i].Done = true
					}
					m.curStep = len(m.steps) + 1
					m.prog = core.FlashProgressInfo{
						BytesWritten: m.prog.TotalBytes,
						TotalBytes:   m.prog.TotalBytes,
						Percent:      100,
					}
				}
			}
		})
	}()
}

// resetToDevices 回到设备列表并重新扫描。
func (m *Model) resetToDevices() {
	m.result = nil
	m.detail = nil
	m.devIdx = 0
	m.refreshDevices()
}

// --- 手柄输入指示 ---

// onGamepadReady 由 HID 监听在句柄打开（设备在线）时调用。
func (m *Model) onGamepadReady() {
	m.post(func() { m.padReady = true })
}

// onGamepadInput 由 HID 监听在后台 goroutine 中调用。
//
// 只有「待机 → 活动」或「活动 → 待机」的状态切换才触发重绘，
// 避免摇杆连续输入导致的高频重绘。
func (m *Model) onGamepadInput() {
	m.padLastNs.Store(time.Now().UnixNano())

	if m.padShown.CompareAndSwap(false, true) {
		m.post(func() {
			m.padActive = true
			m.padHeard = true
		})
		time.AfterFunc(400*time.Millisecond, m.padCooldown)
	}
}

// padCooldown 在输入停止后把指示器切回待机。
func (m *Model) padCooldown() {
	last := time.Unix(0, m.padLastNs.Load())
	if time.Since(last) < 400*time.Millisecond {
		// 仍有输入，稍后再检查
		time.AfterFunc(200*time.Millisecond, m.padCooldown)
		return
	}
	m.padShown.Store(false)
	m.post(func() { m.padActive = false })
}
