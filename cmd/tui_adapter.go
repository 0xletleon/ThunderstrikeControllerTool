// Package cmd — TUI 适配器实现。
//
// 实现 tui 包定义的 DeviceScanner、FirmwareScanner、Flasher 接口，
// 将现有业务逻辑（WMI 设备扫描、HID 查询、固件解析、SPP 刷写）
// 适配为 TUI 可调用的接口。
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"thunderstrike-controller-tool/firmware"
	"thunderstrike-controller-tool/hid"
	"thunderstrike-controller-tool/logger"
	"thunderstrike-controller-tool/spp"
	"thunderstrike-controller-tool/tui"
)

// --- DeviceScanner 实现 ---

// tuiDeviceScanner 实现 tui.DeviceScanner 接口。
type tuiDeviceScanner struct{}

// NewTuiDeviceScanner 创建 TUI 设备扫描器。
func NewTuiDeviceScanner() tui.DeviceScanner {
	return &tuiDeviceScanner{}
}

// ScanDevices 扫描蓝牙 SPP 设备，转换为 tui.DeviceInfo。
func (s *tuiDeviceScanner) ScanDevices() ([]tui.DeviceInfo, error) {
	btPorts, err := discoverBtComPorts()
	if err != nil {
		return nil, err
	}

	fwVer := queryHidFirmwareVersion()

	var devices []tui.DeviceInfo
	idx := 1
	for _, bp := range btPorts {
		devName := bp.deviceName()
		name := fmt.Sprintf("%s  %s  (%s)", devName, bp.comPort, bp.macColon)
		devices = append(devices, tui.DeviceInfo{
			Index:      idx,
			Name:       name,
			Port:       bp.comPort,
			MAC:        bp.macColon,
			DeviceName: devName,
			FwVersion:  fwVer,
		})
		idx++
	}
	return devices, nil
}

// QueryDeviceDetail 通过 HID 查询设备详细信息。
func (s *tuiDeviceScanner) QueryDeviceDetail(device tui.DeviceInfo) (*tui.DeviceDetail, error) {
	client, err := hid.OpenWindowsHidClient()
	if err != nil {
		return &tui.DeviceDetail{
			DeviceName: device.DeviceName,
			MAC:        device.MAC,
		}, nil
	}
	defer client.Close()

	detail := &tui.DeviceDetail{
		DeviceName: device.DeviceName,
		MAC:        device.MAC,
	}

	// 查询固件版本
	verData, err := client.SendCommand(hid.CmdVersion, nil)
	if err == nil && len(verData) >= 2 {
		major := int(verData[1])
		minor := int(verData[0])
		detail.FwVersion = fmt.Sprintf("%d.%02d", major, minor)
		detail.FwVersionHex = fmt.Sprintf("0x%02X%02X", major, minor)
		if len(verData) >= 4 {
			detail.CsrVersion = fmt.Sprintf("CSR v%d.%02d", int(verData[3]), int(verData[2]))
		}
		if len(verData) >= 6 {
			detail.HotwordVer = fmt.Sprintf("v%d.%02d", int(verData[5]), int(verData[4]))
		}
	}

	// 查询序列号（BOARD_INFO 响应 data[2:] 为 ASCII 序列号）
	boardData, err := client.SendCommand(hid.CmdBoardInfo, nil)
	if err == nil && len(boardData) >= 2 {
		detail.Serial = extractSerial(boardData[2:])
	}

	// 查询 MAC 地址
	macData, err := client.SendCommand(hid.CmdMacAddress, nil)
	if err == nil && len(macData) >= 6 {
		detail.MacHID = hid.FormatMacAddress(macData[:6])
	}

	// 查询电量
	battInfo, err := client.GetBatteryInfo()
	if err == nil && battInfo != nil {
		detail.BatteryPct = battInfo.Percent
		detail.ReservePower = battInfo.Reserve
	}

	return detail, nil
}

// --- FirmwareScanner 实现 ---

// tuiFirmwareScanner 实现 tui.FirmwareScanner 接口。
type tuiFirmwareScanner struct{}

// NewTuiFirmwareScanner 创建 TUI 固件扫描器。
func NewTuiFirmwareScanner() tui.FirmwareScanner {
	return &tuiFirmwareScanner{}
}

// ScanFirmwares 扫描 blkz/ 目录，转换为 tui.FirmwareInfo。
func (s *tuiFirmwareScanner) ScanFirmwares() ([]tui.FirmwareInfo, error) {
	blkzDir := findBlkzDir()
	if blkzDir == "" {
		return nil, fmt.Errorf("未找到 blkz 固件目录")
	}

	entries, err := os.ReadDir(blkzDir)
	if err != nil {
		return nil, fmt.Errorf("读取固件目录失败: %w", err)
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

	var fwList []tui.FirmwareInfo
	for i, path := range blkzFiles {
		filename := filepath.Base(path)

		blkz, err := firmware.OpenBlkz(path)
		if err != nil {
			continue
		}

		csResult, csErr := firmware.VerifyBlkzMd5(path)

		info := tui.FirmwareInfo{
			Index:      i + 1,
			Path:       path,
			Name:       filename,
			Version:    blkz.Manifest.VersionString(),
			VersionHex: blkz.Manifest.VersionHex(),
			OtaSize:    blkz.OtaFileSize(),
		}

		if blkz.Manifest.IsLocale() {
			info.IsLocale = true
			info.Languages = blkz.Manifest.Languages()
		}

		if csErr != nil {
			info.Checksum = "error"
		} else {
			switch csResult.Status {
			case firmware.ChecksumMatch:
				info.Checksum = "match"
			case firmware.ChecksumMismatch:
				info.Checksum = "mismatch"
				info.CsExpected = csResult.Expected
			default:
				info.Checksum = "unknown"
			}
			info.CsActual = csResult.Actual
		}

		fwList = append(fwList, info)
	}

	return fwList, nil
}

// --- Flasher 实现 ---

// tuiFlasher 实现 tui.Flasher 接口。
type tuiFlasher struct{}

// NewTuiFlasher 创建 TUI 刷写执行器。
func NewTuiFlasher() tui.Flasher {
	return &tuiFlasher{}
}

// ExecuteFlash 执行固件刷写，通过回调报告进度。
// 同时记录完整日志（设备信息、电量、固件信息、刷写步骤、进度、结果）。
func (f *tuiFlasher) ExecuteFlash(
	device tui.DeviceInfo,
	fw tui.FirmwareInfo,
	stepCB func(step, total int, name string),
	progressCB func(written, total int),
) (*tui.FlashResultInfo, error) {
	// 解析固件包获取 OTA 数据
	blkz, err := firmware.OpenBlkz(fw.Path)
	if err != nil {
		return &tui.FlashResultInfo{Success: false, Error: fmt.Sprintf("打开固件: %v", err)}, nil
	}

	entryIdx := fw.LanguageIndex
	otaData, otaSize, err := getOtaData(blkz, entryIdx)
	if err != nil {
		return &tui.FlashResultInfo{Success: false, Error: fmt.Sprintf("获取固件数据: %v", err)}, nil
	}

	// --- 创建日志 ---
	logDir := findLogDir()
	fwLog, logErr := logger.NewFlashLogger(logDir)
	if logErr != nil {
		// 日志创建失败不阻塞刷写
		fwLog = nil
	}

	// 记录刷写前设备信息（含电量）
	if fwLog != nil {
		battInfo, battErr := checkBatteryRaw()
		battPct := 0
		if battErr == nil && battInfo != nil {
			battPct = battInfo.Percent
		}
		fwLog.LogDeviceInfo("刷写前", logger.DeviceInfo{
			Name:       device.DeviceName,
			MAC:        device.MAC,
			FwVersion:  device.FwVersion,
			BatteryPct: battPct,
		})

		// 记录固件包信息
		lang := ""
		if blkz.Manifest.IsLocale() && entryIdx < len(blkz.OtaEntries) {
			lang = blkz.OtaEntries[entryIdx].Language
		}
		md5Str := ""
		if fw.CsActual != "" {
			md5Str = fw.CsActual
		}
		fwLog.LogFirmwareInfo(logger.FirmwareInfo{
			Filename: fw.Name,
			Version:  fmt.Sprintf("V%s (0x%s)", fw.Version, fw.VersionHex),
			Size:     otaSize,
			MD5:      md5Str,
			Language: lang,
		})
		fwLog.LogFlashStart(device.Port)
	}

	// --- 执行刷写 ---
	flashStart := time.Now()

	result, err := flashFirmware(&FlashOptions{
		Port:       device.Port,
		OtaData:    otaData,
		OtaSize:    otaSize,
		VersionStr: fw.Version,
	}, spp.StepCallback(stepCB), spp.ProgressCallback(progressCB))

	elapsed := time.Since(flashStart)

	// --- 记录刷写结果 ---
	if fwLog != nil {
		if err != nil {
			fwLog.LogFlashError(err, elapsed)
		} else {
			fwLog.LogFlashSuccess("V"+device.FwVersion, "V"+fw.Version, elapsed, otaSize)

			// 记录刷写后设备信息（含电量）
			// 注意：手柄此时可能正在重启，HID 查询可能失败
			battInfo, battErr := checkBatteryRaw()
			if battErr == nil && battInfo != nil {
				fwLog.LogDeviceInfo("刷写后", logger.DeviceInfo{
					Name:       device.DeviceName,
					MAC:        device.MAC,
					FwVersion:  fw.Version,
					BatteryPct: battInfo.Percent,
				})
			} else {
				// HID 已断开（手柄重启中），仅记录基本信息
				fwLog.LogDeviceInfo("刷写后", logger.DeviceInfo{
					Name:      device.DeviceName,
					MAC:       device.MAC,
					FwVersion: fw.Version,
				})
				fwLog.LogRaw("  (电量: 设备重启中，无法查询)")
			}
		}
	}

	logPath := ""
	if fwLog != nil {
		logPath = fwLog.FilePath()
		fwLog.Close()
	}

	if err != nil {
		return &tui.FlashResultInfo{
			Success: false,
			Error:   err.Error(),
			Elapsed: fmt.Sprintf("%.1fs", elapsed.Seconds()),
		}, nil
	}

	// 发送 HID 重启指令（APPLY_OTA 已触发重启，此处是补充）
	sendHidReset()

	return &tui.FlashResultInfo{
		Success:   true,
		Elapsed:   fmt.Sprintf("%.1fs", elapsed.Seconds()),
		BytesWrit: result.BytesWrite,
		FromVer:   device.FwVersion,
		ToVer:     fw.Version,
		LogPath:   logPath,
	}, nil
}

// Compile-time interface compliance checks.
var (
	_ tui.DeviceScanner  = (*tuiDeviceScanner)(nil)
	_ tui.FirmwareScanner = (*tuiFirmwareScanner)(nil)
	_ tui.Flasher         = (*tuiFlasher)(nil)
)
