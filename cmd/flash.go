// Package cmd — flash 子命令（命令行直接调用模式）。
//
// 用法：
//   tsct flash --port COM3 -f firmware.blkz
//   tsct flash -f firmware.blkz --dry-run
//
// 交互模式中刷写走 cmd/interactive_flash.go。
package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"thunderstrike-controller-tool/firmware"
)

var (
	flashPort    string
	flashFile    string
	flashDryRun  bool
	flashNoErase bool
	flashNoApply bool
)

// flashCmd flashes firmware to the controller via Bluetooth SPP.
var flashCmd = &cobra.Command{
	Use:   "flash",
	Short: "Flash firmware via Bluetooth SPP",
	Long: `Flash firmware to the Thunderstrike controller via Bluetooth SPP.

This performs the complete firmware flashing sequence:
  1. Connect (NOP handshake)
  2. Erase SQIF flash memory
  3. Write firmware data (in 1024-byte chunks)
  4. Validate written data
  5. Apply OTA update
  6. Wait for device restart

The firmware file should be a .blkz package (ZIP archive containing
manifest.xml and thunderstrike.ota).

Requirements:
  - Controller must be paired to the computer
  - SPP COM port must be available
  - Battery level should be above 20%

Example:
  tsct flash --port COM3 -f firmware.blkz
  tsct flash --port COM3 -f firmware.blkz --dry-run`,
	RunE: runFlash,
}

func init() {
	flashCmd.Flags().StringVarP(&flashPort, "port", "p", "", "Serial port path (e.g. COM3)")
	flashCmd.Flags().StringVarP(&flashFile, "file", "f", "", "Firmware .blkz file path")
	flashCmd.Flags().BoolVar(&flashDryRun, "dry-run", false, "Parse and verify firmware without flashing")
	flashCmd.Flags().BoolVar(&flashNoErase, "no-erase", false, "Skip SQIF erase step (dangerous)")
	flashCmd.Flags().BoolVar(&flashNoApply, "no-apply", false, "Skip apply OTA step (for testing)")
	_ = flashCmd.MarkFlagRequired("file")
}

func runFlash(cmd *cobra.Command, args []string) error {
	// Step 1: 打开并解析固件文件
	fmt.Printf("打开固件文件: %s\n", flashFile)

	blkz, err := firmware.OpenBlkz(flashFile)
	if err != nil {
		return fmt.Errorf("open firmware: %w", err)
	}

	fmt.Printf("  固件版本 : V%s (0x%s)\n",
		blkz.Manifest.VersionString(),
		blkz.Manifest.VersionHex())
	fmt.Printf("  固件大小 : %d bytes (%.1f KB)\n",
		blkz.OtaFileSize(),
		float64(blkz.OtaFileSize())/1024.0)
	fmt.Println()

	if flashDryRun {
		fmt.Println("Dry run: 固件解析成功，未执行刷写。")
		return nil
	}

	// Step 2: 验证串口
	if flashPort == "" {
		return fmt.Errorf("serial port is required (use --port)")
	}

	// Step 3: 打印刷写摘要
	printFlashSummary(blkz, 0, flashPort)
	fmt.Println()

	// Step 4: 执行刷写
	fmt.Println("开始刷写固件...")
	fmt.Println("  [1] Connect (NOP)")
	fmt.Println("  [2] Erase SQIF")
	fmt.Println("  [3] Write firmware data")
	fmt.Println("  [4] Validate SQIF")
	fmt.Println("  [5] Apply OTA")
	fmt.Println("  [6] Wait for restart")
	fmt.Println()

	result, err := flashFirmware(&FlashOptions{
		Port:       flashPort,
		OtaData:    blkz.OtaData(),
		OtaSize:    blkz.OtaFileSize(),
		VersionStr: blkz.Manifest.VersionString(),
		NoErase:    flashNoErase,
		NoApply:    flashNoApply,
	}, func(written, total int) {
		pct := float64(written) / float64(total) * 100.0
		fmt.Printf("\r  Write progress: %d/%d bytes (%.1f%%)", written, total, pct)
		if written == total {
			fmt.Println()
		}
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n刷写失败: %v\n", err)
		return err
	}

	// Step 5: 打印结果
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════")
	fmt.Println("  固件刷写成功！")
	fmt.Println("═══════════════════════════════════════════")
	fmt.Printf("  总耗时   : %s\n", result.Elapsed.Truncate(time.Millisecond))
	fmt.Printf("  写入量   : %d bytes\n", result.BytesWrite)
	fmt.Printf("  新版本   : V%s\n", blkz.Manifest.VersionString())
	fmt.Println()
	fmt.Println("手柄应该会自动重启并加载新固件。")
	fmt.Println("如果在 30 秒内没有重启，请手动开机。")
	return nil
}
