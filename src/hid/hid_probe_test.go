package hid

import (
	"testing"
	"unsafe"
)

// 以下三个 API 仅用于诊断/测试，生产代码路径不依赖它们。
var (
	procHidD_GetPreparsedData  = hidDll.NewProc("HidD_GetPreparsedData")
	procHidD_FreePreparsedData = hidDll.NewProc("HidD_FreePreparsedData")
	procHidP_GetCaps           = hidDll.NewProc("HidP_GetCaps")
)

// hidpCaps 对应 Windows 的 HIDP_CAPS 结构。
type hidpCaps struct {
	Usage                     uint16
	UsagePage                 uint16
	InputReportByteLength     uint16
	OutputReportByteLength    uint16
	FeatureReportByteLength   uint16
	Reserved                  [17]uint16
	NumberLinkCollectionNodes uint16
	NumberInputButtonCaps     uint16
	NumberInputValueCaps      uint16
	NumberInputDataIndices    uint16
	NumberOutputButtonCaps    uint16
	NumberOutputValueCaps     uint16
	NumberOutputDataIndices   uint16
	NumberFeatureButtonCaps   uint16
	NumberFeatureValueCaps    uint16
	NumberFeatureDataIndices  uint16
}

// TestListHidCollections 列出手柄暴露的所有 HID 集合及其 UsagePage/Usage 与报告长度。
//
// 用途：确认按键上报与命令收发分别落在哪个集合上（输入监听会监听全部集合）。
// 运行：go test ./hid -run TestListHidCollections -v
func TestListHidCollections(t *testing.T) {
	paths, err := findThunderstrikeHidPaths()
	if err != nil {
		t.Fatalf("枚举 HID 接口失败: %v", err)
	}
	if len(paths) == 0 {
		t.Skip("手柄未连接，跳过")
	}

	for i, path := range paths {
		handle := openPath(path)
		if handle == 0 {
			t.Logf("集合 #%d 打开失败（可能被独占）:\n  %s", i+1, path)
			continue
		}

		var caps hidpCaps
		var preparsed uintptr
		if got, _, _ := procHidD_GetPreparsedData.Call(handle, uintptr(unsafe.Pointer(&preparsed))); got != 0 {
			procHidP_GetCaps.Call(preparsed, uintptr(unsafe.Pointer(&caps)))
			procHidD_FreePreparsedData.Call(preparsed)
		}

		t.Logf("集合 #%d  UsagePage=0x%04X  Usage=0x%04X  输入报告=%d 字节  输出报告=%d 字节\n  %s",
			i+1, caps.UsagePage, caps.Usage,
			caps.InputReportByteLength, caps.OutputReportByteLength, path)

		procCloseHandle.Call(handle)
	}

	t.Logf("共发现 %d 个 Thunderstrike HID 集合", len(paths))
}
