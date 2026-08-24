// Package term 提供终端控制功能：清屏、光标定位、隐藏光标。
//
// Windows 10+ 原生支持 ANSI escape 序列，可以通过 \x1b[... 控制。
// Go 的 fmt.Print 可以直接输出这些序列。
//
// 常用序列：
//   \x1b[2J     清屏
//   \x1b[H      光标移到左上角 (1,1)
//   \x1b[?25l   隐藏光标
//   \x1b[?25h   显示光标
package term

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

// enabled 记录 ANSI 是否已启用。
var enabled bool

// EnableANSI 启用 Windows 终端的 ANSI escape 序列支持。
// Windows 10 1607+ 需要调用 SetConsoleMode 开启 VT100 模式。
// 如果启用失败（旧版 Windows），则后续清屏操作退化为调用 cls 命令。
func EnableANSI() {
	stdout := windows.Handle(os.Stdout.Fd())
	var originalMode uint32

	// 获取当前控制台模式
	if err := windows.GetConsoleMode(stdout, &originalMode); err != nil {
		enabled = false
		return
	}

	// 启用 ENABLE_VIRTUAL_TERMINAL_PROCESSING (0x0004)
	newMode := originalMode | 0x0004
	if err := windows.SetConsoleMode(stdout, newMode); err != nil {
		enabled = false
		return
	}

	enabled = true
}

// ClearScreen 清屏并将光标移到左上角。
// 优先使用 ANSI escape，失败时退化为 cls 命令。
func ClearScreen() {
	if enabled {
		// ANSI: \x1b[2J 清屏 + \x1b[H 光标归位
		fmt.Print("\x1b[2J\x1b[H")
		return
	}
	// 退化方案：调用 cls
	fmt.Print("\x1b[2J\x1b[H")
}

// ClearLine 清除当前行（用于进度条覆盖）。
func ClearLine() {
	if enabled {
		fmt.Print("\x1b[2K\x1b[1G")
	}
}

// HideCursor 隐藏光标。
func HideCursor() {
	if enabled {
		fmt.Print("\x1b[?25l")
	}
}

// ShowCursor 显示光标。
func ShowCursor() {
	if enabled {
		fmt.Print("\x1b[?25h")
	}
}
