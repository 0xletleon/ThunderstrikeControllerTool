package gui

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"thunderstrike-controller-tool/src/core"
)

// appVersion 显示在页头的版本号。
const appVersion = "V0.4"

// Model 是界面的全部状态。单页布局，没有页面栈：
// 设备、固件、刷写进度同时可见。
type Model struct {
	svc core.Services
	win *mygo.Window

	// 设备
	devices  []core.DeviceInfo
	devIdx   int
	scanning bool
	scanErr  string

	// 选中设备的 HID 详情
	detail        *core.DeviceDetail
	detailLoading bool

	// 固件
	firmwares []core.FirmwareInfo
	fwIdx     int
	langIdx   int // 多语言固件选中的子包索引
	fwLoading bool
	fwErr     string

	// 刷写
	steps    []core.FlashStep
	curStep  int
	prog     core.FlashProgressInfo
	flashing bool
	result   *core.FlashResultInfo

	// 手柄输入指示
	padReady  bool
	padActive bool
	padShown  atomic.Bool
	padLastNs atomic.Int64
}

// Run 创建窗口并运行 GUI，直到用户关闭窗口。
func Run(svc core.Services) error {
	m := &Model{svc: svc}

	mygo.App.WhenReady(func() {
		m.win = mygo.NewWindow(mygo.WindowOptions{
			Title:           "Thunderstrike Controller Tool",
			Width:           760,
			Height:          700,
			MinWidth:        680,
			MinHeight:       560,
			BackgroundColor: "#FFFFFF",
			// 自绘标题栏：系统标题栏是浅灰的，与白底界面割裂
			TitleBarStyle:  mygo.TitleBarHidden,
			TitleBarHeight: headerHeight,
			Content:        ui.View(m.View),
		})
		m.start()
	})

	if err := mygo.App.Run(); err != nil {
		return err
	}

	if svc.Monitor != nil {
		svc.Monitor.StopMonitoring()
	}
	return nil
}

// post 把 fn 投递到主线程执行并重绘。窗口未建立（单元测试）时直接执行。
func (m *Model) post(fn func()) {
	if m.win == nil {
		fn()
		return
	}
	defer func() { _ = recover() }()
	m.win.Update(fn)
}

// start 在窗口建立后启动手柄监听与首次扫描。
func (m *Model) start() {
	if m.svc.Monitor != nil {
		m.svc.Monitor.StartMonitoring(m.onGamepadInput, m.onGamepadReady)
	}
	m.refreshAll()
}

// View 构建整窗界面：页头 + 设备区 + 固件区 + 底部操作区。
func (m *Model) View(c *ui.Context) {
	c.SetTheme(buildTheme())
	m.handleKeys(c)

	// 完全自然流：页头 / 设备 / 固件 / 操作 / 状态依次排列。
	// 不设 Width：FillWidth 会按窗口宽度算，不扣内边距，导致右侧被裁。
	ui.Column(c).Fill().Background(colorBg).Children(func() {
		m.headerBar(c)
		m.deviceBlock(c)
		m.firmwareBlock(c)
		m.actionBlock(c)
		m.statusBar(c)
	})
}

// headerBar 渲染页头：品牌、手柄状态与版本。
func (m *Model) headerBar(c *ui.Context) {
	tb := c.TitleBar()

	row := ui.Row(c).FillWidth().Height(headerHeight).Padding(0, pagePadding).Gap(8).
		AlignItems(ui.Center)
	if tb.Right > 0 {
		row.PaddingEnd(tb.Right + 8)
	}
	row.DragWindow()

	row.Children(func() {
		ui.Box(c).Size(8, 8).Background(colorBrand)
		ui.Text(c, "THUNDERSTRIKE").FontSize(11).Bold().
			LetterSpacing(1.6).TextColor(colorText)
		ui.Text(c, "SHIELD TV 2017").FontSize(10).TextColor(colorTextFaint)

		ui.Spacer(c)

		m.padDot(c)
		ui.Text(c, appVersion).FontSize(10).TextColor(colorTextFaint)
	})

	hairline(c)
}

// padDot 渲染手柄连接状态：等待 / 就绪 / 有输入。
func (m *Model) padDot(c *ui.Context) {
	dot := colorTextFaint
	text := "等待手柄"
	switch {
	case m.padActive:
		dot = colorBrand
		text = "手柄活动"
	case m.padReady:
		dot = colorBrandDark
		text = "手柄就绪"
	}

	ui.Row(c).Gap(5).AlignItems(ui.Center).Children(func() {
		ui.Box(c).Size(6, 6).Radius(3).Background(dot)
		ui.Text(c, text).FontSize(10).TextColor(colorTextMuted)
	})
}

// --- 数据加载 ---

// refreshAll 同时刷新设备与固件列表。
func (m *Model) refreshAll() {
	m.refreshDevices()
	m.loadFirmwares()
}

// refreshDevices 后台扫描蓝牙设备，选中第一台并读取详情。
func (m *Model) refreshDevices() {
	m.scanning = true
	m.scanErr = ""

	go func() {
		devices, err := m.svc.Scanner.ScanDevices()
		m.post(func() {
			m.scanning = false
			if err != nil {
				m.scanErr = err.Error()
				m.devices = nil
				return
			}
			m.devices = devices
			m.devIdx = 0
			m.detail = nil
			m.refreshDetail()
		})
	}()
}

// refreshDetail 读取当前选中设备的 HID 详情（版本、电量、序列号）。
func (m *Model) refreshDetail() {
	if len(m.devices) == 0 {
		return
	}
	dev := m.devices[m.devIdx]
	m.detailLoading = true

	go func() {
		detail, err := m.svc.Scanner.QueryDeviceDetail(dev)
		m.post(func() {
			m.detailLoading = false
			if err != nil {
				m.detail = &core.DeviceDetail{DeviceName: dev.DeviceName, MAC: dev.MAC}
				return
			}
			m.detail = detail
		})
	}()
}

// loadFirmwares 后台扫描固件目录。
func (m *Model) loadFirmwares() {
	m.fwLoading = true
	m.fwErr = ""

	go func() {
		list, err := m.svc.Firmwares.ScanFirmwares()
		m.post(func() {
			m.fwLoading = false
			if err != nil {
				m.fwErr = err.Error()
				m.firmwares = nil
				return
			}
			m.firmwares = list
			if m.fwIdx >= len(list) {
				m.fwIdx = 0
				m.langIdx = 0
			}
		})
	}()
}

// --- 刷写 ---

// selectedFirmware 返回当前选中的固件，没有则返回 nil。
func (m *Model) selectedFirmware() *core.FirmwareInfo {
	if m.fwIdx < 0 || m.fwIdx >= len(m.firmwares) {
		return nil
	}
	return &m.firmwares[m.fwIdx]
}

// selectedDevice 返回当前选中设备，没有则返回 nil。
func (m *Model) selectedDevice() *core.DeviceInfo {
	if m.devIdx < 0 || m.devIdx >= len(m.devices) {
		return nil
	}
	return &m.devices[m.devIdx]
}

// canFlash 报告当前是否允许开始刷写。
func (m *Model) canFlash() bool {
	fw := m.selectedFirmware()
	dev := m.selectedDevice()
	if fw == nil || dev == nil || m.flashing {
		return false
	}
	// 休眠设备查不到版本与电量，刷写风险高，先要求唤醒
	if !dev.Online() {
		return false
	}
	// 校验不通过的固件禁止刷写
	return fw.Checksum != "mismatch" && fw.Checksum != "error"
}

// startFlash 在后台执行刷写，进度通过回调投递回主线程。
func (m *Model) startFlash() {
	dev := m.selectedDevice()
	fw := m.selectedFirmware()
	if dev == nil || fw == nil || m.flashing {
		return
	}

	target := *fw
	target.LanguageIndex = m.langIdx
	m.steps = core.NewFlashSteps()
	m.curStep = 0
	m.prog = core.FlashProgressInfo{}
	m.result = nil
	m.flashing = true

	var lastPush atomic.Int64

	go func() {
		result, err := m.svc.Flasher.ExecuteFlash(*dev, target,
			func(step, total int, name string) {
				m.post(func() {
					m.curStep = step
					if step >= 2 && step-2 < len(m.steps) {
						m.steps[step-2].Done = true
					}
				})
			},
			func(written, total int) {
				// 每 1KB 回调一次，节流到约 25 次/秒
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
			},
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

// currentStepName 返回当前正在执行的步骤名。
func (m *Model) currentStepName() string {
	if m.curStep >= 1 && m.curStep <= len(m.steps) {
		return m.steps[m.curStep-1].Name
	}
	return ""
}

// statusText 返回底部状态文案。
func (m *Model) statusText() string {
	switch {
	case m.scanning:
		return "正在扫描设备…"
	case m.scanErr != "":
		return "扫描失败：" + m.scanErr
	}

	dev := m.selectedDevice()
	if dev == nil {
		return "未发现手柄，请确认已开机并配对"
	}
	if m.flashing {
		return "刷写中，请勿操作手柄"
	}
	if !dev.Online() {
		return "手柄休眠中 · 按键唤醒后点右上角「刷新」"
	}
	return fmt.Sprintf("%s  ·  %s", orDash(dev.DeviceName), orDash(dev.Port))
}

// --- 手柄输入指示 ---

// onGamepadReady 由 HID 监听在句柄打开（设备在线）时调用。
func (m *Model) onGamepadReady() {
	m.post(func() { m.padReady = true })
}

// onGamepadInput 由 HID 监听在后台 goroutine 中调用。
// 只在状态切换时重绘，避免摇杆连续输入导致高频刷新。
func (m *Model) onGamepadInput() {
	m.padLastNs.Store(time.Now().UnixNano())

	if m.padShown.CompareAndSwap(false, true) {
		m.post(func() {
			m.padActive = true
			m.padReady = true
		})
		time.AfterFunc(400*time.Millisecond, m.padCooldown)
	}
}

// padCooldown 在输入停止后把指示器切回就绪。
func (m *Model) padCooldown() {
	last := time.Unix(0, m.padLastNs.Load())
	if time.Since(last) < 400*time.Millisecond {
		time.AfterFunc(200*time.Millisecond, m.padCooldown)
		return
	}
	m.padShown.Store(false)
	m.post(func() { m.padActive = false })
}
