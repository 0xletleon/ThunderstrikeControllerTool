// Package cmd — blkz firmware list display.
//
// Scans the blkz/ subdirectory next to the executable for .blkz firmware
// packages, parses each one's manifest, and prints the available firmware
// files with version information.
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"thunderstrike-controller-tool/firmware"
)

// printBlkzList scans the blkz/ directory next to the executable and prints
// all available .blkz firmware packages with their version info.
func printBlkzList() {
	// Find the blkz directory next to the executable or in the working directory.
	blkzDir := findBlkzDir()
	if blkzDir == "" {
		fmt.Println()
		fmt.Println("═══════════════════════════════════════════")
		fmt.Println("  可用固件")
		fmt.Println("═══════════════════════════════════════════")
		fmt.Println()
		fmt.Println("  未找到 blkz 固件目录。")
		fmt.Println("  请将 .blkz 固件包放入程序所在目录的 blkz/ 子目录中。")
		return
	}

	// Scan for .blkz files.
	entries, err := os.ReadDir(blkzDir)
	if err != nil {
		fmt.Printf("  读取固件目录失败: %v\n", err)
		return
	}

	var blkzFiles []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) == ".blkz" {
			blkzFiles = append(blkzFiles, filepath.Join(blkzDir, entry.Name()))
		}
	}

	sort.Strings(blkzFiles)

	fmt.Println()
	fmt.Println("═══════════════════════════════════════════")
	fmt.Println("  可用固件")
	fmt.Println("═══════════════════════════════════════════")
	fmt.Println()

	if len(blkzFiles) == 0 {
		fmt.Println("  blkz 目录中没有 .blkz 固件包。")
		fmt.Printf("  目录: %s\n", blkzDir)
		return
	}

	for i, path := range blkzFiles {
		printBlkzEntry(i+1, path)
	}

	fmt.Printf("\n  目录: %s\n", blkzDir)
}

// printBlkzEntry opens a single .blkz file and prints its info.
func printBlkzEntry(index int, path string) {
	filename := filepath.Base(path)

	// Extract .blkz to same-named subdirectory (idempotent)
	extractedDir, err := firmware.ExtractBlkz(path)
	if err != nil {
		fmt.Printf("  [%d] %s  (解压失败: %v)\n", index, filename, err)
		return
	}

	blkz, err := firmware.OpenBlkz(path)
	if err != nil {
		fmt.Printf("  [%d] %s  (解析失败: %v)\n", index, filename, err)
		return
	}

	mf := blkz.Manifest
	if mf == nil {
		fmt.Printf("  [%d] %s  (无 manifest)\n", index, filename)
		return
	}

	// Version info from manifest: version hex "010E" → "1.14"
	verStr := mf.VersionString()
	verHex := mf.VersionHex()

	// OTA file size
	otaSize := blkz.OtaFileSize()

	// Accessory type
	accessory := mf.AccessoryType()

	// Fingerprint (shortened for display)
	// fp := mf.Update.Fingerprint

	// Downgrade support
	downgrade := ""
	if mf.SupportsDowngrade() {
		downgrade = " [支持降级]"
	}

	// MD5 checksum verification
	csResult, err := firmware.VerifyBlkzMd5(path)
	checksumLine := ""
	if err != nil {
		checksumLine = fmt.Sprintf("MD5 校验   : 读取失败 (%v)", err)
	} else {
		switch csResult.Status {
		case firmware.ChecksumMatch:
			checksumLine = fmt.Sprintf("MD5 校验   : %s ✓", csResult.Actual)
		case firmware.ChecksumMismatch:
			checksumLine = fmt.Sprintf("MD5 校验   : %s ✗ (预期: %s) — 危险！", csResult.Actual, csResult.Expected)
		default:
			checksumLine = fmt.Sprintf("MD5 校验   : %s (未收录，无法验证)", csResult.Actual)
		}
	}

	fmt.Printf("  [%d] %s\n", index, filename)
	fmt.Printf("      固件版本 : V%s (0x%s)%s\n", verStr, verHex, downgrade)
	fmt.Printf("      设备类型 : %s\n", accessory)
	fmt.Printf("      固件大小 : %d KB (%d bytes)\n", otaSize/1024, otaSize)

	// Multi-language firmware info
	if mf.IsLocale() {
		langs := mf.Languages()
		totalSize := blkz.TotalOtaSize()
		fmt.Printf("      固件类型 : 多语言版 (%d 种语言)\n", len(langs))
		fmt.Printf("      总大小   : %d KB (%d bytes)\n", totalSize/1024, totalSize)
		// List language codes
		langStr := strings.Join(langs, ", ")
		fmt.Printf("      语言     : %s\n", langStr)
	} else {
		fmt.Printf("      固件类型 : 标准版\n")
	}

	fmt.Printf("      %s\n", checksumLine)
	_ = extractedDir // directory path available for later use (e.g., reading .ota directly)
}

// findBlkzDir locates the blkz/ directory.
// Search order:
//  1. Next to the executable (exeDir/blkz)
//  2. Current working directory (./blkz)
// Returns "" if not found.
func findBlkzDir() string {
	// 1. Next to the executable
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		candidate := filepath.Join(exeDir, "blkz")
		if dirExists(candidate) {
			return candidate
		}
	}

	// 2. Current working directory
	candidate := filepath.Join(".", "blkz")
	if dirExists(candidate) {
		abs, err := filepath.Abs(candidate)
		if err == nil {
			return abs
		}
		return candidate
	}

	return ""
}

// dirExists returns true if path exists and is a directory.
func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
