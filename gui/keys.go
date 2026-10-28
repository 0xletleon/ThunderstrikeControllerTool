package gui

import "github.com/egoist/mygo/ui"

// handleKeys 处理全局与页面级快捷键。
//
// 在每帧构建界面之前调用：这里改完状态，当帧渲染出来的就是新状态。
// 每个按键只读取一次再分发，避免同一次按键事件被重复消费。
func (m *Model) handleKeys(c *ui.Context) {
	const none = ui.Modifiers(0)

	esc := c.Shortcut(none, ui.KeyEscape)
	enter := c.Shortcut(none, ui.KeyEnter)
	up := c.Shortcut(none, ui.KeyUp)
	left := c.Shortcut(none, ui.KeyLeft)
	down := c.Shortcut(none, ui.KeyDown)
	right := c.Shortcut(none, ui.KeyRight)
	refresh := c.Shortcut(none, ui.KeyF5) || c.Shortcut(ui.Ctrl, ui.KeyR)
	quit := c.Shortcut(ui.Ctrl, ui.KeyQ)
	flashKey := c.Shortcut(none, ui.KeyF)

	if quit {
		if m.win != nil {
			m.win.Close()
		}
		return
	}

	if refresh {
		m.refreshCurrent()
		return
	}

	// 刷写进行中屏蔽一切导航，避免误触中断刷写。
	if m.page == screenFlashing && m.flashing {
		return
	}

	if esc {
		m.back()
		return
	}

	switch m.page {
	case screenScanning, screenDevices:
		m.moveSel(len(m.devices), &m.devIdx, up || left, down || right)
		if enter && m.devIdx < len(m.devices) {
			m.openDevice(m.devices[m.devIdx])
		}

	case screenDetail:
		if enter || flashKey {
			m.openFirmwares()
		}

	case screenFirmwares:
		m.moveSel(len(m.firmwares), &m.fwIdx, up || left, down || right)
		if enter && m.fwIdx < len(m.firmwares) {
			m.chooseFirmware(m.firmwares[m.fwIdx])
		}

	case screenLanguages:
		m.moveSel(len(m.pendingFw.Languages), &m.langIdx, up || left, down || right)
		if enter && m.langIdx < len(m.pendingFw.Languages) {
			m.chooseLanguage(m.langIdx)
		}

	case screenFlashing:
		if m.result != nil && enter {
			m.resetToDevices()
		}
	}
}

// moveSel 按方向键移动选中项，索引夹在 [0, n-1] 内。
func (m *Model) moveSel(n int, idx *int, up, down bool) {
	if n <= 0 || idx == nil {
		return
	}
	if up && *idx > 0 {
		*idx--
	}
	if down && *idx < n-1 {
		*idx++
	}
}

// back 返回上一页。刷写进行中由 handleKeys 拦下，不会走到这里。
func (m *Model) back() {
	switch m.page {
	case screenDetail:
		m.page = screenDevices
	case screenFirmwares:
		m.page = screenDetail
	case screenLanguages:
		m.page = screenFirmwares
	case screenFlashing:
		if m.result != nil {
			m.resetToDevices()
		}
	}
}

// refreshCurrent 按当前页面刷新数据。
func (m *Model) refreshCurrent() {
	switch m.page {
	case screenScanning, screenDevices:
		m.refreshDevices()
	case screenDetail:
		// 重新读取一遍 HID 信息（电量、版本可能已变化）
		if m.selected.Port != "" {
			m.openDevice(m.selected)
		}
	case screenFirmwares, screenLanguages:
		m.openFirmwares()
	}
}

// hintText 返回当前页的快捷键提示，显示在底部操作栏。
func (m *Model) hintText() string {
	switch m.page {
	case screenScanning:
		return "F5 重新扫描"
	case screenDevices:
		return "↑↓ 选择 · Enter 打开 · F5 刷新 · Ctrl+Q 退出"
	case screenDetail:
		return "F 刷写固件 · Esc 返回 · F5 重新读取 · Ctrl+Q 退出"
	case screenFirmwares:
		return "↑↓ 选择 · Enter 刷写 · Esc 返回 · F5 重新扫描"
	case screenLanguages:
		return "↑↓ 选择 · Enter 确认 · Esc 返回"
	case screenFlashing:
		if m.flashing {
			return "刷写中，已锁定键盘操作"
		}
		return "Enter 或 Esc 返回设备列表"
	}
	return ""
}
