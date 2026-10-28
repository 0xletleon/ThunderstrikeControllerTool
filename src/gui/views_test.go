package gui

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"

	"thunderstrike-controller-tool/src/core"
)

// snapshotDir 返回渲染快照的输出目录。默认写到 .opencode/temp/renders，
// 可用 GUI_SNAPSHOT_DIR 覆盖。
func snapshotDir() string {
	dir := os.Getenv("GUI_SNAPSHOT_DIR")
	if dir == "" {
		dir = filepath.Join("..", "..", ".opencode", "temp", "renders")
	}
	_ = os.MkdirAll(dir, 0o755)
	return dir
}

// snapshot 在内存中渲染一帧并保存为 PNG，用于人工核对页面布局。
func snapshot(t *testing.T, name string, m *Model) {
	t.Helper()

	tester := ui.NewTester(m.View, 708, 562)
	tester.Frame()

	f, err := os.Create(filepath.Join(snapshotDir(), name+".png"))
	if err != nil {
		t.Fatalf("创建快照文件失败: %v", err)
	}
	defer f.Close()

	if err := png.Encode(f, tester.Image()); err != nil {
		t.Fatalf("编码快照失败: %v", err)
	}
}

// demoDevices 返回两台演示设备：在线与休眠各一台。
func demoDevices() []core.DeviceInfo {
	return []core.DeviceInfo{
		{
			Index:      1,
			Name:       "NVIDIA Controller  COM3  (00:04:4B:92:04:F4)",
			Port:       "COM3",
			MAC:        "00:04:4B:92:04:F4",
			DeviceName: "NVIDIA Controller",
			FwVersion:  "1.33",
		},
		{
			Index:      2,
			Name:       "NVIDIA Controller  COM5  (00:04:4B:92:04:F5)",
			Port:       "COM5",
			MAC:        "00:04:4B:92:04:F5",
			DeviceName: "NVIDIA Controller",
		},
	}
}

// demoDetail 返回一份完整的 HID 查询结果。
func demoDetail() *core.DeviceDetail {
	return &core.DeviceDetail{
		DeviceName:   "NVIDIA Controller",
		MAC:          "00:04:4B:92:04:F4",
		FwVersion:    "1.33",
		FwVersionHex: "0x0121",
		CsrVersion:   "CSR v0.18",
		HotwordVer:   "v1.05",
		Serial:       "NCC-1701-A",
		MacHID:       "00:04:4B:92:04:F4",
		BatteryPct:   64,
	}
}

// demoFirmwares 返回四个固件包：校验通过、未收录、多语言、MD5 不符。
func demoFirmwares() []core.FirmwareInfo {
	return []core.FirmwareInfo{
		{
			Index:      1,
			Name:       "Thunderstrike_0x010E.blkz",
			Version:    "1.14",
			VersionHex: "010E",
			Checksum:   "match",
			OtaSize:    318_464,
		},
		{
			Index:      2,
			Name:       "Thunderstrike_0x0112.blkz",
			Version:    "1.18",
			VersionHex: "0112",
			Checksum:   "match",
			OtaSize:    322_560,
		},
		{
			Index:      3,
			Name:       "Thunderstrike_locale_0x0121.blkz",
			Version:    "1.33",
			VersionHex: "0121",
			Checksum:   "match",
			OtaSize:    328_704,
			IsLocale:   true,
			Languages:  []string{"en_us", "zh_cn", "ja_jp"},
		},
		{
			Index:      4,
			Name:       "Thunderstrike_0x0124.blkz",
			Version:    "1.36",
			VersionHex: "0124",
			Checksum:   "mismatch",
			OtaSize:    330_752,
		},
	}
}

// fakeScanner 用固定数据替代真实的蓝牙扫描与 HID 查询。
type fakeScanner struct{}

func (fakeScanner) ScanDevices() ([]core.DeviceInfo, error) { return demoDevices(), nil }

func (fakeScanner) QueryDeviceDetail(d core.DeviceInfo) (*core.DeviceDetail, error) {
	return demoDetail(), nil
}

// fakeFirmwareScanner 用固定数据替代 blkz 目录扫描。
type fakeFirmwareScanner struct{}

func (fakeFirmwareScanner) ScanFirmwares() ([]core.FirmwareInfo, error) {
	return demoFirmwares(), nil
}

// newTestModel 返回一个填好演示数据的模型。
func newTestModel() *Model {
	return &Model{
		devices:   demoDevices(),
		detail:    demoDetail(),
		firmwares: demoFirmwares(),
		svc: core.Services{
			Scanner:   fakeScanner{},
			Firmwares: fakeFirmwareScanner{},
		},
	}
}

// TestSnapshots 渲染各种状态，产出 PNG 供人工核对。
func TestSnapshots(t *testing.T) {
	cases := []struct {
		name  string
		setup func(m *Model)
	}{
		{"01_ready", func(m *Model) {
			// 默认：设备在线、固件列表、待刷写
		}},
		{"02_scanning", func(m *Model) {
			m.scanning = true
			m.devices = nil
			m.detail = nil
		}},
		{"03_no_device", func(m *Model) {
			m.devices = nil
			m.detail = nil
			m.scanErr = "蓝牙适配器未响应"
		}},
		{"04_flashing", func(m *Model) {
			m.flashing = true
			m.steps = core.NewFlashSteps()
			m.steps[0].Done = true
			m.steps[1].Done = true
			m.curStep = 3
			m.prog = core.FlashProgressInfo{
				BytesWritten: 122_880,
				TotalBytes:   322_560,
				Percent:      38,
			}
		}},
		{"05_done", func(m *Model) {
			m.fwIdx = 1
			m.steps = core.NewFlashSteps()
			for i := range m.steps {
				m.steps[i].Done = true
			}
			m.curStep = len(m.steps) + 1
			m.prog = core.FlashProgressInfo{
				BytesWritten: 322_560,
				TotalBytes:   322_560,
				Percent:      100,
			}
			m.result = &core.FlashResultInfo{
				Success:   true,
				Elapsed:   "12.4s",
				BytesWrit: 322_560,
				FromVer:   "1.33",
				ToVer:     "1.18",
			}
		}},
		{"06_failed", func(m *Model) {
			m.flashing = false
			m.steps = core.NewFlashSteps()
			m.steps[0].Done = true
			m.steps[1].Done = true
			m.curStep = 3
			m.result = &core.FlashResultInfo{
				Success: false,
				Error:   "写入手柄超时：WRITE_SQIF 未收到 ACK（40s）",
			}
		}},
		{"07_locale", func(m *Model) {
			m.fwIdx = 2 // 多语言固件，操作区应出现语言下拉
			m.langIdx = 1
		}},
		{"08_no_device_clean", func(m *Model) {
			m.devices = nil
			m.detail = nil
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := newTestModel()
			tc.setup(m)
			snapshot(t, tc.name, m)
		})
	}
}

// TestClickSelectsFirmware 验证点击固件行会切换选中项。
func TestClickSelectsFirmware(t *testing.T) {
	m := newTestModel()
	tester := ui.NewTester(m.View, 708, 562)
	tester.Frame()

	// 固件列表第三行（行高 44，首行从 y≈173 开始）
	tester.ClickAt(300, 283)
	tester.Frame()

	if m.fwIdx != 2 {
		t.Fatalf("应选中第 3 个固件，实际 fwIdx=%d", m.fwIdx)
	}
}

// TestMoveFirmwareSkipsBroken 验证键盘移动会跳过校验失败的固件。
func TestMoveFirmwareSkipsBroken(t *testing.T) {
	m := newTestModel()
	m.fwIdx = 2 // 多语言固件，下一个是 MD5 不符

	m.moveFirmware(1)
	if m.fwIdx != 0 {
		t.Fatalf("应跳过校验失败项回到第 1 个，实际 fwIdx=%d", m.fwIdx)
	}
}
