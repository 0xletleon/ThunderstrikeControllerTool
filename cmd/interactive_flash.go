// Package cmd — 交互模式刷写流程。
//
// 用户选择设备后进入刷写流程：
//   1. 列出 blkz/ 目录下的可用固件包
//   2. 用户选择固件包
//   3. MD5 校验（安全检查）
//   4. 版本对比与用户确认
//   5. 多语言固件时选择语言
//   6. 执行刷写（逐步显示进度）
//   7. 保存日志到 logs/ 目录
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"thunderstrike-controller-tool/firmware"
	"thunderstrike-controller-tool/logger"
)

// findLogDir 查找程序所在目录的 logs/ 子目录。
func findLogDir() string {
	if exePath, err := os.Executable(); err == nil {
		return filepath.Join(filepath.Dir(exePath), "logs")
	}
	return "logs"
}

// runInteractiveFlash 是交互模式刷写入口。
// d 是用户选中的设备，reader 用于读取用户输入。
func runInteractiveFlash(d *discoveredDevice, reader *bufio.Reader) error {
	// Step 1: 查找 blkz 目录
	blkzDir := findBlkzDir()
	if blkzDir == "" {
		fmt.Println()
		fmt.Println("  未找到 blkz 固件目录。")
		fmt.Println("  请将 .blkz 固件包放入程序所在目录的 blkz/ 子目录中。")
		return nil
	}

	// Step 2: 扫描可用固件
	entries, err := os.ReadDir(blkzDir)
	if err != nil {
		fmt.Printf("  读取固件目录失败: %v\n", err)
		return nil
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

	if len(blkzFiles) == 0 {
		fmt.Println()
		fmt.Println("  blkz 目录中没有 .blkz 固件包。")
		fmt.Printf("  目录: %s\n", blkzDir)
		return nil
	}

	// Step 3: 列出并选择固件
	fmt.Println()
	fmt.Println("  ── 可用固件 ─────────────────────────────────────────────────────────────────")
	fmt.Println()
	fmt.Println("    No.  固件包                              版本     校验")
	fmt.Println("    ─────────────────────────────────────────────────────")

	type fwInfo struct {
		path  string
		name  string
		blkz  *firmware.BlkzFile
		csRes *firmware.ChecksumResult
		csErr error
	}
	var fwList []fwInfo

	for i, path := range blkzFiles {
		filename := filepath.Base(path)

		_, err := firmware.ExtractBlkz(path)
		if err != nil {
			fmt.Printf("     %d   %-37s %-8s %s\n", i+1, truncate(filename, 37), "?", "解压失败")
			continue
		}

		blkz, err := firmware.OpenBlkz(path)
		if err != nil {
			fmt.Printf("     %d   %-37s %-8s %s\n", i+1, truncate(filename, 37), "?", "解析失败")
			continue
		}

		csResult, csErr := firmware.VerifyBlkzMd5(path)

		fwList = append(fwList, fwInfo{
			path:  path,
			name:  filename,
			blkz:  blkz,
			csRes: csResult,
			csErr: csErr,
		})

		// 显示固件信息
		mf := blkz.Manifest
		verStr := fmt.Sprintf("V%s", mf.VersionString())
		csStr := "?"
		if csErr != nil {
			csStr = "错误"
		} else {
			switch csResult.Status {
			case firmware.ChecksumMatch:
				csStr = "✓"
			case firmware.ChecksumMismatch:
				csStr = "✗"
			default:
				csStr = "未知"
			}
		}
		fmt.Printf("     %d   %-37s %-8s %s\n", i+1, truncate(filename, 37), verStr, csStr)
	}

	if len(fwList) == 0 {
		fmt.Println()
		fmt.Println("  没有可用的固件包。")
		return nil
	}

	fmt.Println()
	fmt.Printf("  选择固件 [1-%d] | 返回 [Enter]: ", len(fwList))
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return nil
	}

	choice, err := strconv.Atoi(input)
	if err != nil || choice < 1 || choice > len(fwList) {
		fmt.Printf("  无效选择: %s\n", input)
		return nil
	}

	selected := fwList[choice-1]

	// Step 4: MD5 安全检查
	if selected.csErr != nil {
		fmt.Printf("\n  ⚠ MD5 读取失败: %v\n", selected.csErr)
		fmt.Print("  是否继续? [y/n]: ")
		confirm, _ := reader.ReadString('\n')
		if strings.TrimSpace(strings.ToLower(confirm)) != "y" {
			fmt.Println("  已取消。")
			return nil
		}
	} else if selected.csRes.Status == firmware.ChecksumMismatch {
		fmt.Println()
		fmt.Println("  ⚠⚠⚠ MD5 校验失败！固件可能损坏或被篡改！")
		fmt.Printf("  实际: %s\n", selected.csRes.Actual)
		fmt.Printf("  期望: %s\n", selected.csRes.Expected)
		fmt.Println("  强烈建议不要继续刷写！")
		fmt.Print("  确认继续刷写? (输入 YES 继续): ")
		confirm, _ := reader.ReadString('\n')
		if strings.TrimSpace(confirm) != "YES" {
			fmt.Println("  已取消（安全选择）。")
			return nil
		}
	} else if selected.csRes.Status == firmware.ChecksumUnknown {
		fmt.Println()
		fmt.Println("  ⚠ 此固件未在已知 MD5 列表中，无法验证完整性。")
		fmt.Print("  是否继续? [y/n]: ")
		confirm, _ := reader.ReadString('\n')
		if strings.TrimSpace(strings.ToLower(confirm)) != "y" {
			fmt.Println("  已取消。")
			return nil
		}
	}

	// Step 5: 多语言固件选择
	entryIdx := 0
	blkz := selected.blkz
	if blkz.Manifest.IsLocale() && len(blkz.OtaEntries) > 1 {
		fmt.Println()
		fmt.Println("  多语言固件，请选择语言：")
		for i, entry := range blkz.OtaEntries {
			lang := entry.Language
			if lang == "" {
				lang = "(默认)"
			}
			fmt.Printf("    [%d] %s  (%d KB)\n", i+1, lang, len(entry.Data)/1024)
		}
		fmt.Print("  请选择: ")
		langInput, _ := reader.ReadString('\n')
		langInput = strings.TrimSpace(langInput)
		langChoice, err := strconv.Atoi(langInput)
		if err != nil || langChoice < 1 || langChoice > len(blkz.OtaEntries) {
			fmt.Printf("  无效选择，使用默认语言。\n")
		} else {
			entryIdx = langChoice - 1
		}
	}

	// Step 6: 确认刷写
	otaData, otaSize, err := getOtaData(blkz, entryIdx)
	if err != nil {
		fmt.Printf("  获取固件数据失败: %v\n", err)
		return nil
	}

	fmt.Println()
	fmt.Println("  ── 刷写确认 ─────────────────────────────────────────────────────────────────")
	fmt.Println()
	printInfoLine("端口", d.port)
	printInfoLine("固件", selected.name)
	printInfoLine("版本", fmt.Sprintf("V%s (0x%s)  ← 当前 V%s (%s)",
		blkz.Manifest.VersionString(), blkz.Manifest.VersionHex(),
		d.fwVersion, versionDirection(d.fwVersion, blkz.Manifest.VersionString())))
	printInfoLine("大小", fmt.Sprintf("%.1f KB (%d bytes)", float64(otaSize)/1024.0, otaSize))
	fmt.Println()
	fmt.Print("  确认开始刷写? [y/n]: ")

	confirm, _ := reader.ReadString('\n')
	if strings.TrimSpace(strings.ToLower(confirm)) != "y" {
		fmt.Println("  已取消。")
		return nil
	}

	// Step 7: 创建日志记录器
	logDir := findLogDir()
	fwLog, logErr := logger.NewFlashLogger(logDir)
	if logErr != nil {
		fmt.Printf("  ⚠ 无法创建日志文件: %v（继续刷写）\n", logErr)
	}
	defer func() {
		if fwLog != nil {
			fwLog.Close()
		}
	}()

	// 记录固件信息到日志
	if fwLog != nil {
		lang := ""
		if blkz.Manifest.IsLocale() && entryIdx < len(blkz.OtaEntries) {
			lang = blkz.OtaEntries[entryIdx].Language
		}
		md5Str := ""
		if selected.csRes != nil {
			md5Str = selected.csRes.Actual
		}
		fwLog.LogFirmwareInfo(logger.FirmwareInfo{
			Filename: selected.name,
			Version:  fmt.Sprintf("V%s (0x%s)", blkz.Manifest.VersionString(), blkz.Manifest.VersionHex()),
			Size:     otaSize,
			MD5:      md5Str,
			Language: lang,
		})
		fwLog.LogFlashStart(d.port)
	}

	// Step 8: 执行刷写（逐步显示进度）
	fmt.Println()
	fmt.Println("  ── 刷写中 ───────────────────────────────────────────────────────────────────")
	fmt.Println()

	flashStart := time.Now()

	// 进度回调
	var lastPct int = -1
	progressCB := func(written, total int) {
		pct := written * 100 / total
		if pct != lastPct || written == total {
			barLen := 30
			filled := barLen * written / total
			bar := strings.Repeat("█", filled) + strings.Repeat("░", barLen-filled)
			fmt.Printf("\r    [3/5] 写入数据  %s %3d%%  %6d/%d",
				bar, pct, written, total)
			if written == total {
				fmt.Printf("\r    [3/5] 写入数据  %s 完成  %d bytes\n",
					strings.Repeat("█", 30), total)
			}
			lastPct = pct
		}
		if fwLog != nil {
			fwLog.LogProgress(written, total)
		}
	}

	fmt.Println("    [1/5] 连接握手  ...")
	_, err = flashFirmware(&FlashOptions{
		Port:       d.port,
		OtaData:    otaData,
		OtaSize:    otaSize,
		VersionStr: blkz.Manifest.VersionString(),
	}, progressCB)

	elapsed := time.Since(flashStart)

	// 记录结果到日志
	if fwLog != nil {
		if err != nil {
			fwLog.LogFlashError(err, elapsed)
		} else {
			fwLog.LogFlashSuccess("V"+d.fwVersion, "V"+blkz.Manifest.VersionString(),
				elapsed, otaSize)
		}
	}

	if err != nil {
		fmt.Println()
		fmt.Printf("  ✗ 刷写失败: %v\n", err)
		return nil
	}

	// 回填步骤 1-2 和 4-5 的状态（flashFirmware 内部已完成）
	fmt.Printf("\r    [1/5] 连接握手  ✓\n")
	fmt.Println("    [2/5] 擦除闪存  ✓")
	fmt.Println("    [4/5] 校验数据  ✓")
	fmt.Println("    [5/5] 应用固件  ✓")

	fmt.Println()
	fmt.Println("  ────────────────────────────────────────────────────────────────────────────")

	// Step 9: 成功结果
	fmt.Println()
	fmt.Println("  ── 刷写完成 ─────────────────────────────────────────────────────────────────")
	fmt.Println()
	fmt.Println("    ✓ 刷写成功")
	fmt.Println()
	printInfoLine("耗时", fmt.Sprintf("%.1fs", elapsed.Seconds()))
	printInfoLine("写入", fmt.Sprintf("%d bytes (%.1f KB)", otaSize, float64(otaSize)/1024.0))
	printInfoLine("固件", fmt.Sprintf("V%s → V%s", d.fwVersion, blkz.Manifest.VersionString()))
	fmt.Println()
	fmt.Println("    手柄将自动重启，请等待重新连接。")
	fmt.Println("    30 秒内未重启请手动开机。")
	fmt.Println()

	if fwLog != nil {
		fmt.Printf("    日志已保存: %s\n", fwLog.FilePath())
	}

	fmt.Println()
	fmt.Println("  ────────────────────────────────────────────────────────────────────────────")

	return nil
}

// versionDirection 比较版本号，返回 "降级"/"升级"/"平刷"。
func versionDirection(current, target string) string {
	if current == "" {
		return "未知"
	}
	if target == current {
		return "平刷"
	}
	if target > current {
		return "升级"
	}
	return "降级"
}

// truncate 截断字符串到指定显示宽度（中文字符占 2 宽）。
func truncate(s string, maxDisplay int) string {
	displayWidth := 0
	result := []rune(s)
	for i, r := range result {
		w := runeWidth(r)
		if displayWidth+w > maxDisplay {
			return string(result[:i])
		}
		displayWidth += w
	}
	return s
}
