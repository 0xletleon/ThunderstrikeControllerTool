// Package cmd — 可复用的固件刷写核心逻辑。
//
// 封装了：
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
}

// FlashResult holds the outcome of a flash operation.
type FlashResult struct {
	Elapsed    time.Duration
	BytesWrite int
}

// flashFirmware 执行完整的固件刷写流程。
//
// 该函数封装了交互模式使用的核心逻辑：
//  1. 打开串口
//  2. 创建 SPP 客户端
//  3. 执行刷写序列（NOP → ERASE → WRITE → VALIDATE → APPLY）
//  4. 等待设备重启
//
// stepCB 在每个步骤开始时调用，progressCB 在写入数据时调用。
func flashFirmware(opts *FlashOptions, stepCB spp.StepCallback, progressCB spp.ProgressCallback) (*FlashResult, error) {
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

	if stepCB != nil {
		flasher.SetStepCallback(stepCB)
	}
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