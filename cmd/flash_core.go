// Package cmd — 可复用的固件刷写核心逻辑。
//
// 从 flash.go 和交互模式共用。封装了：
//   - 固件解析与验证
//   - SPP 连接建立
//   - 完整刷写流程（NOP → ERASE → WRITE → VALIDATE → APPLY）
//   - 进度回调
package cmd

import (
	"fmt"
	"io"
	"time"

	"go.bug.st/serial"
	"thunderstrike-controller-tool/firmware"
	"thunderstrike-controller-tool/spp"
)

// FlashOptions 控制刷写行为的参数集。
type FlashOptions struct {
	Port       string // SPP 串口路径 (如 COM3)
	OtaData    []byte // 要写入的 .ota 固件二进制数据
	OtaSize    int    // .ota 数据大小（字节）
	VersionStr string // 固件版本号（用于显示）
	NoErase    bool   // 跳过擦除步骤（危险）
	NoApply    bool   // 跳过应用步骤（测试用）
}

// FlashResult holds the outcome of a flash operation.
type FlashResult struct {
	Elapsed    time.Duration
	BytesWrite int
}

// flashFirmware 执行完整的固件刷写流程。
//
// 该函数封装了 flashCmd 和交互模式共用的核心逻辑：
//  1. 打开串口
//  2. 创建 SPP 客户端
//  3. 执行刷写序列（NOP → ERASE → WRITE → VALIDATE → APPLY）
//  4. 等待设备重启
//
// opts 参数携带串口路径、固件数据和选项。
// progressCB 可为 nil，否则在每个 1024 字节块写入后被调用。
func flashFirmware(opts *FlashOptions, progressCB func(written, total int)) (*FlashResult, error) {
	// 打开串口
	mode := &serial.Mode{BaudRate: 115200}
	port, err := serial.Open(opts.Port, mode)
	if err != nil {
		return nil, fmt.Errorf("open serial port %s: %w", opts.Port, err)
	}
	defer port.Close()

	// 创建 SPP 客户端和 flasher
	client := spp.NewClient(port)
	timings := spp.DefaultTimings()
	flasher := spp.NewFlasher(client, timings)

	if progressCB != nil {
		flasher.SetProgressCallback(progressCB)
	}

	// 执行刷写
	flashStart := time.Now()

	err = flasher.Flash(newBytesReader(opts.OtaData), opts.OtaSize)
	if err != nil {
		return nil, fmt.Errorf("flash: %w", err)
	}

	elapsed := time.Since(flashStart)

	return &FlashResult{
		Elapsed:    elapsed,
		BytesWrite: opts.OtaSize,
	}, nil
}

// verifyFirmwareBeforeFlash 在刷写前验证固件包的完整性和安全性。
// 返回 OtaEntry（多语言固件时可能由调用方选择）和错误。
type firmwareChoice struct {
	blkz     *firmware.BlkzFile
	entryIdx int // 选中的 OtaEntry 索引（0 = 标准，多语言时由调用方选择）
}

// getOtaData 获取选定固件条目的 .ota 二进制数据。
func getOtaData(blkz *firmware.BlkzFile, entryIdx int) ([]byte, int, error) {
	if entryIdx < 0 || entryIdx >= len(blkz.OtaEntries) {
		return nil, 0, fmt.Errorf("ota entry index %d out of range (have %d)", entryIdx, len(blkz.OtaEntries))
	}
	data := blkz.OtaEntries[entryIdx].Data
	return data, len(data), nil
}

// printFlashSummary 打印刷写前的摘要信息。
func printFlashSummary(blkz *firmware.BlkzFile, entryIdx int, port string) {
	entry := blkz.OtaEntries[entryIdx]
	fmt.Printf("  串口       : %s\n", port)
	fmt.Printf("  固件版本   : V%s (0x%s)\n",
		blkz.Manifest.VersionString(),
		blkz.Manifest.VersionHex())
	fmt.Printf("  设备类型   : %s\n", blkz.Manifest.AccessoryType())
	if blkz.Manifest.IsLocale() && entry.Language != "" {
		fmt.Printf("  语言       : %s\n", entry.Language)
	}
	fmt.Printf("  固件大小   : %d bytes (%.1f KB)\n", len(entry.Data), float64(len(entry.Data))/1024.0)
}

// bytesReader wraps a byte slice as an io.Reader.
type bytesReader struct {
	data []byte
	pos  int
}

func newBytesReader(data []byte) *bytesReader {
	return &bytesReader{data: data}
}

func (r *bytesReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
