// Package tui — 刷写进度视图。
//
// 展示刷写流程的步骤进度和写入进度条，
// 使用 Bubbles progress 组件渲染百分比进度条。
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

// FlashProgressModel 是刷写进度视图的状态。
type FlashProgressModel struct {
	progress   progress.Model
	steps      []FlashStep
	currentStep int
	progressInfo FlashProgressInfo
	done       bool
	result     *FlashResultInfo
	quitting   bool
	width      int
	height     int
}

// defaultSteps 返回刷写流程的标准步骤。
func defaultSteps() []FlashStep {
	names := []string{"连接握手", "擦除闪存", "写入数据", "校验数据", "应用固件"}
	steps := make([]FlashStep, 5)
	for i, name := range names {
		steps[i] = FlashStep{Index: i + 1, Total: 5, Name: name, Done: false}
	}
	return steps
}

// NewFlashProgressModel 创建刷写进度模型。
func NewFlashProgressModel() FlashProgressModel {
	p := progress.New(
		progress.WithSolidFill(colorPrimary),
		progress.WithoutPercentage(),
	)
	return FlashProgressModel{
		progress:    p,
		steps:       defaultSteps(),
		currentStep: 0,
	}
}

// Update 处理刷写进度视图的消息。
func (m FlashProgressModel) Update(msg tea.Msg) (FlashProgressModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.progress.Width = msg.Width - 10
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}

	case StepUpdateMsg:
		// 更新当前步骤
		m.currentStep = msg.Step - 1
		if m.currentStep >= 0 && m.currentStep < len(m.steps) {
			m.steps[m.currentStep].Done = true
		}
		return m, nil

	case ProgressUpdateMsg:
		m.progressInfo = FlashProgressInfo{
			BytesWritten: msg.Written,
			TotalBytes:   msg.Total,
			Percent:      msg.Percent,
		}
		// 更新进度条
		cmd := m.progress.SetPercent(float64(msg.Percent) / 100.0)
		return m, cmd

	case FlashCompleteMsg:
		m.done = true
		if msg.Result != nil {
			m.result = msg.Result
			// 标记所有步骤完成
			for i := range m.steps {
				m.steps[i].Done = true
			}
		}
		return m, nil

	case FlashFailedMsg:
		m.done = true
		m.result = &FlashResultInfo{
			Success: false,
			Error:   msg.Error,
		}
		return m, nil

	case progress.FrameMsg:
		model, cmd := m.progress.Update(msg)
		if p, ok := model.(progress.Model); ok {
			m.progress = p
		}
		return m, cmd
	}

	return m, nil
}

// View 渲染刷写进度视图。
func (m FlashProgressModel) View() string {
	var b strings.Builder
	b.WriteString(RenderAsciiTitle("Thunderstrike", "smslant", colorPrimary))
	b.WriteString(SubtitleStyle.Render("  NVIDIA SHIELD TV 2017 Thunderstrike Controller Tool"))
	b.WriteString("\n\n")

	b.WriteString(SectionHeaderStyle.Render(" ■ 刷写固件"))
	b.WriteString("\n\n")

	// 渲染步骤列表
	for _, step := range m.steps {
		var marker string
		var name string
		if step.Done {
			marker = StepDoneStyle.Render("✓")
			name = DimStyle.Render(step.Name)
		} else if step.Index == m.currentStep+1 {
			marker = StepLabelStyle.Render("→")
			name = HighlightStyle.Render(step.Name)
		} else {
			marker = StepPendingStyle.Render("○")
			name = DimStyle.Render(step.Name)
		}
		b.WriteString(fmt.Sprintf("  %s [%d/%d] %s\n", marker, step.Index, step.Total, name))
	}

	b.WriteString("\n")

	// 渲染写入进度条（仅在写入阶段）
	if m.currentStep == 2 || m.progressInfo.TotalBytes > 0 { // Step 3 = 写入数据
		b.WriteString("  " + m.progress.View() + "\n")
		if m.progressInfo.TotalBytes > 0 {
			b.WriteString(fmt.Sprintf("  %d/%d bytes (%d%%)\n",
				m.progressInfo.BytesWritten,
				m.progressInfo.TotalBytes,
				m.progressInfo.Percent))
		}
	}

	b.WriteString("\n")

	// 刷写结果
	if m.done {
		if m.result != nil && m.result.Success {
			b.WriteString(SuccessBoxStyle.Render(
				"✓ 刷写成功\n\n" +
					renderInfoLine("耗时", m.result.Elapsed) + "\n" +
					renderInfoLine("写入", fmt.Sprintf("%d bytes", m.result.BytesWrit)) + "\n" +
					renderInfoLine("固件", fmt.Sprintf("V%s → V%s", m.result.FromVer, m.result.ToVer)) + "\n" +
					renderInfoLine("日志", m.result.LogPath)))
			b.WriteString("\n\n")
			b.WriteString(DimStyle.Render("  正在重启手柄..."))
			b.WriteString("\n")
			b.WriteString(DimStyle.Render("  按 ESC 返回设备列表"))
		} else if m.result != nil && !m.result.Success {
			b.WriteString(ErrorBoxStyle.Render(
				"✗ 刷写失败\n\n" + m.result.Error))
			b.WriteString("\n\n")
			b.WriteString(DimStyle.Render("  按 ESC 返回设备列表"))
		}
	}

	return b.String()
}

// IsQuitting 返回是否正在退出。
func (m FlashProgressModel) IsQuitting() bool {
	return m.quitting
}

// IsDone 返回刷写是否已完成。
func (m FlashProgressModel) IsDone() bool {
	return m.done
}

// --- Messages ---

// StepUpdateMsg 刷写步骤更新消息。
type StepUpdateMsg struct {
	Step int // 当前步骤 (1-based)
	Total int
	Name  string
}

// ProgressUpdateMsg 写入进度更新消息。
type ProgressUpdateMsg struct {
	Written int
	Total   int
	Percent int
}

// FlashCompleteMsg 刷写完成消息。
type FlashCompleteMsg struct {
	Result *FlashResultInfo
}

// FlashFailedMsg 刷写失败消息。
type FlashFailedMsg struct {
	Error string
}
