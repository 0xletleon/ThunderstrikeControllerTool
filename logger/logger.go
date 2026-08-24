// Package logger 提供刷写过程的日志记录功能。
//
// 日志文件保存到程序所在目录的 logs/ 子目录下，
// 文件名格式为 flash_YYYYMMDD_HHMMSS.log。
//
// 日志记录完整的刷写流程：
//   - 刷写前后设备信息（版本、MAC、序列号）
//   - 固件包信息（版本、大小、MD5）
//   - 各步骤耗时（连接、擦除、写入、校验、应用）
//   - 写入进度（每块 1024 字节的 ACK 时间）
//   - 最终结果（成功/失败、总耗时、异常信息）
package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FlashLogger 记录刷写过程日志到文件。
type FlashLogger struct {
	file    *os.File
	startAt time.Time
}

// DeviceInfo 记录刷写前后的设备信息。
type DeviceInfo struct {
	Name        string
	MAC         string
	FwVersion   string
	CsrVersion  string
	HwVersion   string
	Serial     string
}

// FirmwareInfo 记录固件包信息。
type FirmwareInfo struct {
	Filename string
	Version  string
	Size     int
	MD5      string
	Language string
}

// StepResult 记录单一步骤的执行结果。
type StepResult struct {
	Name     string
	Elapsed  time.Duration
	Error    error
}

// NewFlashLogger 创建一个新的日志记录器，立即创建日志文件。
// logDir 为日志目录路径，如果不存在会自动创建。
func NewFlashLogger(logDir string) (*FlashLogger, error) {
	// 确保日志目录存在
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}

	// 生成文件名：flash_20260824_153012.log
	filename := fmt.Sprintf("flash_%s.log", time.Now().Format("20060102_150405"))
	path := filepath.Join(logDir, filename)

	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("create log file: %w", err)
	}

	l := &FlashLogger{
		file:    f,
		startAt: time.Now(),
	}

	// 写入文件头
	l.writeln("========================================")
	l.writeln("Thunderstrike Controller Tool - 刷写日志")
	l.writeln("时间: %s", time.Now().Format("2006-01-02 15:04:05"))
	l.writeln("========================================")
	l.writeln("")

	return l, nil
}

// LogDeviceInfo 记录设备信息（刷写前/后）。
func (l *FlashLogger) LogDeviceInfo(phase string, info DeviceInfo) {
	l.writeln("--- 设备信息 [%s] ---", phase)
	l.writeln("  设备名称: %s", info.Name)
	l.writeln("  MAC 地址: %s", info.MAC)
	if info.FwVersion != "" {
		l.writeln("  软件版本: %s", info.FwVersion)
	}
	if info.CsrVersion != "" {
		l.writeln("  蓝牙芯片: %s", info.CsrVersion)
	}
	if info.HwVersion != "" {
		l.writeln("  硬件版本: %s", info.HwVersion)
	}
	if info.Serial != "" {
		l.writeln("  序列号:  %s", info.Serial)
	}
	l.writeln("")
}

// LogFirmwareInfo 记录固件包信息。
func (l *FlashLogger) LogFirmwareInfo(info FirmwareInfo) {
	l.writeln("--- 固件包信息 ---")
	l.writeln("  文件名: %s", info.Filename)
	l.writeln("  版本:   %s", info.Version)
	l.writeln("  大小:   %d bytes (%.1f KB)", info.Size, float64(info.Size)/1024.0)
	if info.MD5 != "" {
		l.writeln("  MD5:    %s", info.MD5)
	}
	if info.Language != "" {
		l.writeln("  语言:   %s", info.Language)
	}
	l.writeln("")
}

// LogFlashStart 记录刷写开始。
func (l *FlashLogger) LogFlashStart(port string) {
	l.writeln("--- 刷写开始 ---")
	l.writeln("  串口:   %s", port)
	l.writeln("  开始时间: %s", time.Now().Format("15:04:05.000"))
	l.writeln("")
}

// LogStep 记录一个刷写步骤的执行结果。
func (l *FlashLogger) LogStep(step int, total int, result StepResult) {
	status := "成功"
	if result.Error != nil {
		status = fmt.Sprintf("失败: %v", result.Error)
	}
	l.writeln("  [%d/%d] %s - %s (%.3fs)", step, total, result.Name, status, result.Elapsed.Seconds())
}

// LogProgress 记录写入进度（仅记录关键节点，避免日志过大）。
// 当 written == total 或每 10% 进度时记录一次。
func (l *FlashLogger) LogProgress(written, total int) {
	if total == 0 {
		return
	}
	pct := written * 100 / total
	// 仅在每 10% 和 100% 时记录
	if pct%10 == 0 || written == total {
		l.writeln("  进度: %d/%d bytes (%d%%)", written, total, pct)
	}
}

// LogFlashSuccess 记录刷写成功结果。
func (l *FlashLogger) LogFlashSuccess(oldVer, newVer string, elapsed time.Duration, bytesWrite int) {
	l.writeln("")
	l.writeln("--- 刷写结果 ---")
	l.writeln("  状态:   成功")
	l.writeln("  耗时:   %.3fs", elapsed.Seconds())
	l.writeln("  写入:   %d bytes (%.1f KB)", bytesWrite, float64(bytesWrite)/1024.0)
	l.writeln("  版本:   %s -> %s", oldVer, newVer)
	l.writeln("  结束时间: %s", time.Now().Format("15:04:05.000"))
	l.writeln("")
}

// LogFlashError 记录刷写失败。
func (l *FlashLogger) LogFlashError(err error, elapsed time.Duration) {
	l.writeln("")
	l.writeln("--- 刷写结果 ---")
	l.writeln("  状态:   失败")
	l.writeln("  耗时:   %.3fs", elapsed.Seconds())
	l.writeln("  错误:   %v", err)
	l.writeln("  结束时间: %s", time.Now().Format("15:04:05.000"))
	l.writeln("")
}

// LogRaw 写入原始文本行（不加时间戳前缀）。
func (l *FlashLogger) LogRaw(format string, args ...interface{}) {
	l.writeln(format, args...)
}

// LogSeparator 写入分隔线。
func (l *FlashLogger) LogSeparator() {
	l.writeln("")
	l.writeln("----------------------------------------")
	l.writeln("")
}

// FilePath 返回日志文件路径。
func (l *FlashLogger) FilePath() string {
	if l.file == nil {
		return ""
	}
	return l.file.Name()
}

// Close 关闭日志文件并写入结束标记。
func (l *FlashLogger) Close() error {
	if l.file == nil {
		return nil
	}
	l.writeln("")
	l.writeln("========================================")
	l.writeln("日志结束")
	l.writeln("总耗时: %.3fs", time.Since(l.startAt).Seconds())
	l.writeln("========================================")

	err := l.file.Close()
	l.file = nil
	return err
}

// writeln 写入一行文本并换行。
func (l *FlashLogger) writeln(format string, args ...interface{}) {
	if l.file == nil {
		return
	}
	line := fmt.Sprintf(format, args...)
	// 去掉末尾多余换行
	line = strings.TrimRight(line, "\n")
	fmt.Fprintln(l.file, line)
}
