// Package tui — 刷写进度视图。
//
// 展示刷写流程的步骤进度和写入进度条，
// 使用 Bubbles progress 组件渲染百分比进度条。
package tui

import (
	"fmt"
	"os"
	"path/filepath"
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

// 步骤索引常量（1-based）。
const (
	stepConnect  = 1 // 连接握手
	stepErase    = 2 // 擦除闪存
	stepWrite    = 3 // 写入数据
	stepValidate = 4 // 校验数据
	stepApply    = 5 // 应用固件
)

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
		// 防止窗口过窄时 progress.Width 为负数
		w := msg.Width - 10
		if w < 10 {
			w = 10
		}
		m.progress.Width = w
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}

	case StepUpdateMsg:
		// StepCallback 在步骤**开始**时调用，标记前一个步骤为完成
		m.currentStep = msg.Step - 1
		if m.currentStep > 0 && m.currentStep-1 < len(m.steps) {
			m.steps[m.currentStep-1].Done = true
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
	b.WriteString(RenderSubtitlePlaceholder())
	b.WriteString("\n\n")

	b.WriteString(SectionHeaderStyle.Render(" ■ 刷写固件"))
	b.WriteString("\n")
	b.WriteString(WarningStyle.Render("  请勿操作手柄,耐心等待！"))
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
	if m.currentStep == stepWrite-1 || m.progressInfo.TotalBytes > 0 {
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
			logPath := relativePath(m.result.LogPath)
			b.WriteString(SuccessBoxStyle.Render(
				"✓ 刷写成功\n\n" +
					renderInfoLine("耗时", m.result.Elapsed) + "\n" +
					renderInfoLine("写入", fmt.Sprintf("%d bytes", m.result.BytesWrit)) + "\n" +
					renderInfoLine("固件", fmt.Sprintf("V%s → V%s", m.result.FromVer, m.result.ToVer)) + "\n" +
					renderInfoLine("日志", logPath)))
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

// relativePath 将绝对路径转换为相对于当前工作目录的相对路径。
// 如果转换失败或路径为空，则原样返回。
func relativePath(path string) string {
	if path == "" {
		return path
	}
	wd, err := os.Getwd()
	if err != nil {
		return path
	}
	rel, err := filepath.Rel(wd, path)
	if err != nil {
		return path
	}
	return rel
}
