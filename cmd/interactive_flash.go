// Package cmd — 交互模式中的蓝牙刷写流程。
//
// 当用户在交互模式中选择蓝牙设备后，可以进入刷写流程：
//   1. 列出 blkz/ 目录下的可用固件包
//   2. 用户选择固件包
//   3. MD5 校验（安全检查）
//   4. 版本对比与用户确认
//   5. 多语言固件时选择语言
//   6. 执行刷写
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
)

// runInteractiveFlash 是交互模式中的刷写入口。
// port 是用户选择的蓝牙 SPP 串口路径（如 COM3）。
// reader 用于读取用户输入。
func runInteractiveFlash(port string, reader *bufio.Reader) error {
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
	fmt.Println("═══════════════════════════════════════════")
	fmt.Println("  可用固件包")
	fmt.Println("═══════════════════════════════════════════")
	fmt.Println()

	type fwInfo struct {
		path    string
		name    string
		blkz    *firmware.BlkzFile
		csRes   *firmware.ChecksumResult
		csErr   error
	}
	var fwList []fwInfo

	for i, path := range blkzFiles {
		filename := filepath.Base(path)

		// 解压
		_, err := firmware.ExtractBlkz(path)
		if err != nil {
			fmt.Printf("  [%d] %s  (解压失败: %v)\n", i+1, filename, err)
			continue
		}

		blkz, err := firmware.OpenBlkz(path)
		if err != nil {
			fmt.Printf("  [%d] %s  (解析失败: %v)\n", i+1, filename, err)
			continue
		}

		csResult, csErr := firmware.VerifyBlkzMd5(path)

		fwList = append(fwList, fwInfo{
			path: path,
			name: filename,
			blkz: blkz,
			csRes: csResult,
			csErr: csErr,
		})

		// 显示固件信息
		mf := blkz.Manifest
		fmt.Printf("  [%d] %s\n", i+1, filename)
		fmt.Printf("      版本 : V%s (0x%s)\n", mf.VersionString(), mf.VersionHex())
		if mf.IsLocale() {
			langs := mf.Languages()
			fmt.Printf("      类型 : 多语言版 (%d: %s)\n", len(langs), strings.Join(langs, ", "))
		} else {
			fmt.Printf("      类型 : 标准版\n")
		}

		if csErr != nil {
			fmt.Printf("      MD5  : 读取失败\n")
		} else {
			switch csResult.Status {
			case firmware.ChecksumMatch:
				fmt.Printf("      MD5  : %s ✓\n", csResult.Actual)
			case firmware.ChecksumMismatch:
				fmt.Printf("      MD5  : %s ✗ 不匹配！\n", csResult.Actual)
			default:
				fmt.Printf("      MD5  : %s (未收录)\n", csResult.Actual)
			}
		}
	}

	if len(fwList) == 0 {
		fmt.Println()
		fmt.Println("  没有可用的固件包。")
		return nil
	}

	fmt.Println()
	fmt.Print("  请选择固件编号 (回车返回): ")
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
		fmt.Print("  是否继续? (y/n): ")
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
		fmt.Print("  是否继续? (y/n): ")
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
	fmt.Println("═══════════════════════════════════════════")
	fmt.Println("  刷写确认")
	fmt.Println("═══════════════════════════════════════════")
	fmt.Printf("  串口     : %s\n", port)
	fmt.Printf("  固件     : %s\n", selected.name)
	fmt.Printf("  版本     : V%s (0x%s)\n",
		blkz.Manifest.VersionString(), blkz.Manifest.VersionHex())
	if blkz.Manifest.IsLocale() && entryIdx < len(blkz.OtaEntries) {
		lang := blkz.OtaEntries[entryIdx].Language
		if lang == "" {
			lang = "(默认)"
		}
		fmt.Printf("  语言     : %s\n", lang)
	}
	fmt.Printf("  大小     : %d bytes (%.1f KB)\n", otaSize, float64(otaSize)/1024.0)
	fmt.Println()
	fmt.Print("  确认开始刷写? (y/n): ")

	confirm, _ := reader.ReadString('\n')
	if strings.TrimSpace(strings.ToLower(confirm)) != "y" {
		fmt.Println("  已取消。")
		return nil
	}

	// Step 7: 执行刷写
	fmt.Println()
	fmt.Println("  开始刷写固件...")
	fmt.Println("  [1] Connect (NOP)")
	fmt.Println("  [2] Erase SQIF")
	fmt.Println("  [3] Write firmware data")
	fmt.Println("  [4] Validate SQIF")
	fmt.Println("  [5] Apply OTA")
	fmt.Println("  [6] Wait for restart")
	fmt.Println()

	result, err := flashFirmware(&FlashOptions{
		Port:       port,
		OtaData:    otaData,
		OtaSize:    otaSize,
		VersionStr: blkz.Manifest.VersionString(),
	}, func(written, total int) {
		pct := float64(written) / float64(total) * 100.0
		fmt.Printf("\r  写入进度: %d/%d bytes (%.1f%%)", written, total, pct)
		if written == total {
			fmt.Println()
		}
	})
	if err != nil {
		fmt.Printf("\n  刷写失败: %v\n", err)
		return nil
	}

	// Step 8: 结果
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════")
	fmt.Println("  固件刷写成功！")
	fmt.Println("═══════════════════════════════════════════")
	fmt.Printf("  总耗时 : %s\n", result.Elapsed.Truncate(time.Millisecond))
	fmt.Printf("  写入量 : %d bytes\n", result.BytesWrite)
	fmt.Printf("  新版本 : V%s\n", blkz.Manifest.VersionString())
	fmt.Println()
	fmt.Println("  手柄应该会自动重启并加载新固件。")
	fmt.Println("  如果在 30 秒内没有重启，请手动开机。")

	return nil
}
