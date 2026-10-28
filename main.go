// Package main — Thunderstrike Controller Tool
//
// NVIDIA SHIELD TV 2017 Thunderstrike 手柄固件管理工具。
//
// 原生 GUI 窗口（MyGo 自绘界面），业务实现来自 cmd 包。
//
// 支持设备信息读取、固件刷写/降级/升级/平刷。仅支持 Windows，通过蓝牙连接。
package main

import (
	"syscall"
	"unsafe"

	"thunderstrike-controller-tool/src/cmd"
	"thunderstrike-controller-tool/src/core"
	"thunderstrike-controller-tool/src/gui"
)

func main() {
	// 程序以 console 子系统构建（-H=windowsgui 下 mygo 无法启动），
	// 启动后立即脱离控制台，避免窗口旁边挂一个黑框。
	detachConsole()

	svc := core.Services{
		Scanner:   cmd.NewDeviceScanner(),
		Firmwares: cmd.NewFirmwareScanner(),
		Flasher:   cmd.NewFlasher(),
	}

	if err := gui.Run(svc); err != nil {
		showError("启动失败：" + err.Error())
	}
}

// detachConsole 释放当前进程的控制台。
// 子进程（powershell / reg）另由 exec.SysProcAttr.HideWindow 保证不闪窗。
func detachConsole() {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	kernel32.NewProc("FreeConsole").Call()
}

// showError 用系统弹窗报告启动错误。
//
// 程序启动后已脱离控制台，无法靠 stderr 让用户看到错误。
func showError(msg string) {
	const mbIconError = 0x00000010

	title, err := syscall.UTF16PtrFromString("Thunderstrike Controller Tool")
	if err != nil {
		return
	}
	text, err := syscall.UTF16PtrFromString(msg)
	if err != nil {
		return
	}

	user32 := syscall.NewLazyDLL("user32.dll")
	user32.NewProc("MessageBoxW").Call(
		0,
		uintptr(unsafe.Pointer(text)),
		uintptr(unsafe.Pointer(title)),
		mbIconError,
	)
}
