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

	// 与窗口默认尺寸保持一致，快照才代表真实比例。
	tester := ui.NewTester(m.View, 780, 440)
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

// demoDevices 返回两个演示设备：在线与休眠各一个。
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

// demoFirmwares 返回三个固件包：校验通过、未收录、MD5 不符。
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
			Checksum:   "unknown",
			OtaSize:    322_560,
		},
		{
			Index:      3,
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

// TestClickDeviceOpensDetail 验证点击设备行会切换到设备信息页。
func TestClickDeviceOpensDetail(t *testing.T) {
	m := &Model{
		page:    screenDevices,
		devices: demoDevices(),
		svc:     core.Services{Scanner: fakeScanner{}},
	}

	tester := ui.NewTester(m.View, 780, 440)
	tester.Frame()

	// 第一行的中部
	tester.ClickAt(300, 140)
	tester.Frame()

	if m.page != screenDetail {
		t.Fatalf("点击设备行后应在设备信息页，实际为 %v", m.page)
	}
	if m.selected.Port != "COM3" {
		t.Fatalf("选中的设备应为 COM3，实际为 %q", m.selected.Port)
	}
}

// TestClickFirmwareOpensLanguages 验证点击多语言固件会先进入语言选择页。
func TestClickFirmwareOpensLanguages(t *testing.T) {
	firmwares := demoFirmwares()
	firmwares[0].IsLocale = true
	firmwares[0].Languages = []string{"en_us", "zh_cn"}

	m := &Model{
		page:      screenFirmwares,
		firmwares: firmwares,
		selected:  demoDevices()[0],
		svc:       core.Services{Firmwares: fakeFirmwareScanner{}},
	}

	tester := ui.NewTester(m.View, 780, 440)
	tester.Frame()

	// 第一行的中部
	tester.ClickAt(300, 140)
	tester.Frame()

	if m.page != screenLanguages {
		t.Fatalf("点击多语言固件后应在语言选择页，实际为 %v", m.page)
	}
}

// newBaseModel 返回一个带演示设备的初始模型。Model 内含 atomic 字段，
// 不能按值复制，因此每个用例都新建一份。
func newBaseModel() *Model {
	return &Model{
		page:     screenDevices,
		devices:  demoDevices(),
		selected: demoDevices()[0],
	}
}

// TestSnapshots 渲染全部页面状态，产出 PNG 供人工核对。
func TestSnapshots(t *testing.T) {
	cases := []struct {
		name  string
		setup func(m *Model)
	}{
		{"01_devices", func(m *Model) {
			m.page = screenDevices
		}},
		{"02_scanning", func(m *Model) {
			m.page = screenScanning
			m.scanning = true
		}},
		{"03_devices_empty", func(m *Model) {
			m.page = screenDevices
			m.devices = nil
		}},
		{"04_detail", func(m *Model) {
			m.page = screenDetail
			m.detail = demoDetail()
		}},
		{"05_firmwares", func(m *Model) {
			m.page = screenFirmwares
			m.firmwares = demoFirmwares()
		}},
		{"06_languages", func(m *Model) {
			m.page = screenLanguages
			m.pendingFw = core.FirmwareInfo{
				Name:      "Thunderstrike_0x0112_locale.blkz",
				Version:   "1.18",
				IsLocale:  true,
				Languages: []string{"en_us", "zh_cn", "ja_jp"},
			}
		}},
		{"07_flashing", func(m *Model) {
			m.page = screenFlashing
			m.steps = core.NewFlashSteps()
			m.steps[0].Done = true
			m.steps[1].Done = true
			m.curStep = 3
			m.flashing = true
			m.pendingFw = demoFirmwares()[1]
			m.prog = core.FlashProgressInfo{
				BytesWritten: 122_880,
				TotalBytes:   322_560,
				Percent:      38,
			}
		}},
		{"08_flash_done", func(m *Model) {
			m.page = screenFlashing
			m.steps = core.NewFlashSteps()
			for i := range m.steps {
				m.steps[i].Done = true
			}
			m.curStep = len(m.steps) + 1
			m.pendingFw = demoFirmwares()[1]
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
				LogPath:   "Release/logs/flash_20261028_1204.log",
			}
		}},
		{"09_flash_failed", func(m *Model) {
			m.page = screenFlashing
			m.steps = core.NewFlashSteps()
			m.steps[0].Done = true
			m.steps[1].Done = true
			m.curStep = 3
			m.pendingFw = demoFirmwares()[1]
			m.result = &core.FlashResultInfo{
				Success: false,
				Error:   "写入手柄超时：WRITE_SQIF 未收到 ACK（40s）",
			}
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := newBaseModel()
			tc.setup(m)
			snapshot(t, tc.name, m)
		})
	}
}
