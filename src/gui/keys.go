package gui

import "github.com/egoist/mygo/ui"

// handleKeys 处理单页快捷键。
//
// 界面只有一页，所以没有前进/返回：上下选固件、回车开刷、F5 重新扫描。
func (m *Model) handleKeys(c *ui.Context) {
	const none = ui.Modifiers(0)

	up := c.Shortcut(none, ui.KeyUp)
	down := c.Shortcut(none, ui.KeyDown)
	enter := c.Shortcut(none, ui.KeyEnter)
	refresh := c.Shortcut(none, ui.KeyF5) || c.Shortcut(ui.Ctrl, ui.KeyR)
	quit := c.Shortcut(ui.Ctrl, ui.KeyQ)

	if quit {
		if m.win != nil {
			m.win.Close()
		}
		return
	}

	if refresh {
		m.refreshAll()
		return
	}

	// 刷写中锁定所有操作，避免误触
	if m.flashing {
		return
	}

	switch {
	case up:
		m.moveFirmware(-1)
	case down:
		m.moveFirmware(1)
	case enter && m.canFlash():
		m.startFlash()
	}
}

// moveFirmware 在可选固件之间上下移动，跳过校验不通过的条目。
func (m *Model) moveFirmware(delta int) {
	n := len(m.firmwares)
	if n <= 0 {
		return
	}

	for step := 1; step < n; step++ {
		next := (m.fwIdx + delta*step%n + n) % n
		fw := m.firmwares[next]
		if fw.Checksum != "mismatch" && fw.Checksum != "error" {
			m.fwIdx = next
			m.result = nil
			return
		}
	}
}

// hintText 返回底部快捷键提示。
func (m *Model) hintText() string {
	switch {
	case m.flashing:
		return "刷写中，操作已锁定"
	case len(m.firmwares) == 0:
		return "F5 重新扫描"
	}
	return "↑↓ 选择固件 · Enter 开始刷写 · F5 重新扫描"
}
