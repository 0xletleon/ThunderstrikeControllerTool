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
	"thunderstrike-controller-tool/hid"
	"thunderstrike-controller-tool/logger"
)

func findLogDir() string {
	if exePath, err := os.Executable(); err == nil {
		return filepath.Join(filepath.Dir(exePath), "logs")
	}
	return "logs"
}

func runInteractiveFlash(d *discoveredDevice, reader *bufio.Reader) error {
	blkzDir := findBlkzDir()
	if blkzDir == "" {
		fmt.Println()
		fmt.Println("  未找到 blkz 固件目录。")
		fmt.Println("  请将 .blkz 固件包放入程序所在目录的 blkz/ 子目录中。")
		return nil
	}

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

	otaData, otaSize, err := getOtaData(blkz, entryIdx)
	if err != nil {
		fmt.Printf("  获取固件数据失败: %v\n", err)
		return nil
	}

	fmt.Println()
	fmt.Print("  正在检查手柄电量... ")
	rawBatt, battErr := checkBatteryRaw()
	if battErr != nil {
		fmt.Printf("无法获取 (%v)\n", battErr)
	} else {
		pct := adcToPercent(rawBatt)
		level := batteryLevelText(pct)
		fmt.Printf("%d%%  ADC: %d (%s)\n", pct, rawBatt, level)
		if pct < 20 {
			fmt.Println("  ⚠ 电量低于 20%，刷写过程中断电可能导致变砖！")
			fmt.Print("  是否继续? [y/n]: ")
			confirm, _ := reader.ReadString('\n')
			if strings.TrimSpace(strings.ToLower(confirm)) != "y" {
				fmt.Println("  已取消。")
				return nil
			}
		}
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

	fmt.Println()
	fmt.Println("  ── 刷写中 ───────────────────────────────────────────────────────────────────")
	fmt.Println()

	flashStart := time.Now()

	stepCB := func(step, total int, name string) {
		fmt.Printf("    [%d/%d] %s  ...\n", step, total, name)
	}

	var lastPct int = -1
	progressCB := func(written, total int) {
		pct := written * 100 / total
		if pct != lastPct || written == total {
			barLen := 30
			filled := barLen * written / total
			bar := strings.Repeat("█", filled) + strings.Repeat("░", barLen-filled)
			fmt.Printf("\r      %s %3d%%  %6d/%d", bar, pct, written, total)
			if written == total {
				fmt.Printf("\r      %s 完成  %d bytes\n", strings.Repeat("█", 30), total)
			}
			lastPct = pct
		}
		if fwLog != nil {
			fwLog.LogProgress(written, total)
		}
	}

	_, err = flashFirmware(&FlashOptions{
		Port:       d.port,
		OtaData:    otaData,
		OtaSize:    otaSize,
		VersionStr: blkz.Manifest.VersionString(),
	}, stepCB, progressCB)

	elapsed := time.Since(flashStart)

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

	fmt.Println("    [1/5] 连接握手  ✓")
	fmt.Println("    [2/5] 擦除闪存  ✓")
	fmt.Println("    [3/5] 写入数据  ✓")
	fmt.Println("    [4/5] 校验数据  ✓")
	fmt.Println("    [5/5] 应用固件  ✓")

	fmt.Println()
	fmt.Println("  ────────────────────────────────────────────────────────────────────────────")

	fmt.Println()
	fmt.Println("  ── 刷写完成 ─────────────────────────────────────────────────────────────────")
	fmt.Println()
	fmt.Println("    ✓ 刷写成功")
	fmt.Println()
	printInfoLine("耗时", fmt.Sprintf("%.1fs", elapsed.Seconds()))
	printInfoLine("写入", fmt.Sprintf("%d bytes (%.1f KB)", otaSize, float64(otaSize)/1024.0))
	printInfoLine("固件", fmt.Sprintf("V%s → V%s", d.fwVersion, blkz.Manifest.VersionString()))
	fmt.Println()

	fmt.Println("    正在重启手柄...")
	if resetErr := sendHidReset(); resetErr != nil {
		fmt.Printf("    ⚠ HID 重启失败: %v\n", resetErr)
		fmt.Println("      APPLY_OTA 已触发重启，请等待重连。")
	} else {
		fmt.Println("    ✓ 重启指令已发送")
	}

	for i := 5; i > 0; i-- {
		fmt.Printf("\r    等待重连... %ds  ", i)
		time.Sleep(1 * time.Second)
	}
	if path, _ := hid.FindThunderstrikeHidDevice(); path != "" {
		fmt.Println("\r    ✓ 设备已重新连接      ")
	} else {
		fmt.Println("\r    设备尚未重连，请稍候  ")
	}
	fmt.Println()

	if fwLog != nil {
		fmt.Printf("    日志已保存: %s\n", fwLog.FilePath())
	}

	fmt.Println()
	fmt.Println("  ────────────────────────────────────────────────────────────────────────────")

	return nil
}

// checkBatteryRaw 通过 HID 查询手柄电量原始 ADC 值。
// 返回 0-255 的原始值，供调用方自行判断电量水平。
func checkBatteryRaw() (int, error) {
	client, err := hid.OpenWindowsHidClient()
	if err != nil {
		return 0, fmt.Errorf("打开 HID 设备: %w", err)
	}
	defer client.Close()

	data, err := client.SendCommand(hid.CmdBatteryState, nil)
	if err != nil {
		return 0, fmt.Errorf("电池查询: %w", err)
	}
	if len(data) < 1 {
		return 0, fmt.Errorf("电池: 空响应")
	}
	return int(data[0]), nil
}

func sendHidReset() error {
	client, err := hid.OpenWindowsHidClient()
	if err != nil {
		return fmt.Errorf("打开 HID 设备: %w", err)
	}
	defer client.Close()

	_, err = client.SendCommand(hid.CmdReset, []byte{0x01})
	if err != nil {
		return nil
	}
	return nil
}

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