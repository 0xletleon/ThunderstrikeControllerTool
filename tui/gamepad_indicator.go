// Package tui — 手柄输入指示器。
//
// 将副标题本身作为手柄输入指示器：
//   - 无输入：副标题用暗灰色 SubtitleStyle 渲染（默认外观）
//   - 有输入：副标题用亮绿色 SectionHeaderStyle 渲染（高亮闪烁）
//
// HID 监听器检测到手柄按键/摇杆活动时发送 GamepadInputMsg，
// 指示器立即切换为高亮状态，短暂延迟后恢复待机。
package tui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// subtitlePlaceholder 是各子视图中副标题的占位文本。
// 子视图渲染副标题时使用 RenderSubtitlePlaceholder()，
// model.go 的 View() 会将其替换为带手柄状态的副标题。
const subtitlePlaceholder = "\x00SUBTITLE\x00"

// GamepadIndicator 是手柄输入指示器的状态。
type GamepadIndicator struct {
	active    bool      // 是否正在高亮
	lastInput time.Time // 最后一次收到输入的时间
}

// NewGamepadIndicator 创建手柄指示器。
func NewGamepadIndicator() GamepadIndicator {
	return GamepadIndicator{}
}

// GamepadInputMsg 手柄输入检测消息。
type GamepadInputMsg struct{}

// gamepadTickMsg 动画 tick 消息。
type gamepadTickMsg struct{}

// Update 处理指示器消息。
func (g GamepadIndicator) Update(msg tea.Msg) (GamepadIndicator, tea.Cmd) {
	switch msg.(type) {
	case GamepadInputMsg:
		g.active = true
		g.lastInput = time.Now()
		// 150ms 后发 tick 检查是否该恢复
		return g, tea.Tick(150*time.Millisecond, func(time.Time) tea.Msg {
			return gamepadTickMsg{}
		})

	case gamepadTickMsg:
		// 超过 150ms 没有新输入，恢复待机
		if time.Since(g.lastInput) > 150*time.Millisecond {
			g.active = false
			return g, nil
		}
		// 仍在活跃期，继续 tick
		return g, tea.Tick(50*time.Millisecond, func(time.Time) tea.Msg {
			return gamepadTickMsg{}
		})
	}
	return g, nil
}

// RenderSubtitle 根据手柄输入状态渲染副标题。
// 无输入：暗灰色（SubtitleStyle）
// 有输入：亮绿色粗体（SectionHeaderStyle）
func (g GamepadIndicator) RenderSubtitle() string {
	text := "  NVIDIA SHIELD TV 2017 Thunderstrike Controller Tool. @0xletleon V0.2"
	if g.active {
		return SectionHeaderStyle.Render(text)
	}
	return SubtitleStyle.Render(text)
}

// RenderSubtitlePlaceholder 返回副标题占位符，供子视图使用。
func RenderSubtitlePlaceholder() string {
	return subtitlePlaceholder
}

// ApplySubtitle 将内容中的副标题占位符替换为带手柄状态的副标题。
func ApplySubtitle(content string, styledSubtitle string) string {
	return strings.ReplaceAll(content, subtitlePlaceholder, styledSubtitle)
}
