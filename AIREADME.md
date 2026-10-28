![Thunderstrike Controller](README/shieldtv-2017-controller-Thunderstrike.png)

# AIREADME — AI 技术参考

> 本文档供 AI 助手阅读，包含项目的技术实现细节、协议逆向分析、架构决策。
> 用户文档见 `README.md`。

## 项目概述

NVIDIA SHIELD TV 2017（中国版）配套的 **Thunderstrike** 蓝牙手柄固件管理工具。
Go 语言编写，仅支持 Windows，通过蓝牙连接手柄。界面为原生窗口（MyGo 自绘）。

核心能力：设备信息读取（HID）、固件刷写/降级/升级/平刷（SPP）、刷写后自动重启（HID CMD_RESET）。

## 通信架构

手柄通过蓝牙注册**两个独立接口**，各自使用不同协议：

| 接口 | UUID | Windows 路径 | 协议格式 | 用途 |
|------|------|-------------|----------|------|
| SPP | `{00001101-0000-1000-8000-00805f9b34fb}` | COM 口 (BTHENUM) | `[cmd][2B LE len][data]` | 固件刷写 |
| HID | `{00001124-0000-1000-8000-00805f9b34fb}` | `\\?\hid#vid_0955&pid_7214#...` | `[0x04][cmd][txn][data pad 33B]` | 设备信息查询 + 重启 |

> **关键发现**：SPP 串口上只支持刷写协议，不支持 HID 报告格式。
> 设备信息查询必须通过 Windows HID API（`SetupDiGetClassDevsW` + `CreateFileW` + `WriteFile`/`ReadFile`）。

### HID 报告格式

```
请求: [0x04] [cmd_ordinal] [transaction_id] [data...] [zero-padded to 33 bytes]
响应: [0x03] [cmd_ordinal] [transaction_id] [response_data...]
```

- ReportLen = 33（30 字节数据区 + 3 字节头部，来自 `Bt_Ops.smali`: `mReportLen = getHidrawReportLength() + 3`）
- MaxResponseSize = 65（来自 `DevHidraw.MAX_RESPONSE_SIZE`）
- 设备返回 `cmd=0xFF` 表示命令不支持（`ErrCmdError`）

### SPP 刷写协议

**命令格式**（PC → 手柄）：`[1B CommandCode] [2B LE length] [N bytes data]`

**响应格式**（手柄 → PC）：`[1B CommandCode] [1B StatusCode] [1B payload_len] [N bytes payload]`

#### SPP 命令码

| 命令 | Code | 超时 | 说明 |
|------|------|------|------|
| NOP | 0x00 | 5s | 握手（发送两次：先 WriteCommand 唤醒，再 ExecuteCommand 确认） |
| ERASE_SQIF | 0x01 | 20s | 擦除 SQIF 闪存，参数 `[0x00]` |
| WRITE_SQIF | 0x02 | 2s/块 | 分块写入，1024 字节/块，每块等 ACK |
| VALIDATE_SQIF | 0x03 | 20s | 验证写入数据 |
| APPLY_OTA | 0x05 | — | 应用固件，参数 `[0x00]`，发送后设备自动重启 |

#### SPP 状态码

| 状态 | Code | 说明 |
|------|------|------|
| COMPLETED_OK | 0x00 | 成功 |
| IN_PROGRESS | 0x01 | 执行中（ReadResponse 自动跳过重试） |
| FAILED | 0x02 | 失败 |
| TIMEOUT | 0x03 | 超时 |
| FAILED_LOW_BATTERY | 0x06 | 电量不足 |
| FAILED_SIGNATURE | 0x07 | 签名验证失败 |
| FAILED_FLASH_ERROR | 0x08 | 闪存错误 |

> `ReadResponse` 只跳过 `IN_PROGRESS`，不跳过 `DIAG_IN_PROGRESS`（与 smali 一致）。

### HID 命令码（设备信息查询 + 重启）

完整 159 个命令枚举见 `src/hid/commands.go`。实际使用的：

| 命令 | Code | 说明 |
|------|------|------|
| VERSION | 0x01 | 返回 6 字节 `[fw_minor][fw_major][csr_minor][csr_major][hot_minor][hot_major]` |
| BATTERY_STATE | 0x07 | 返回电池完整状态（见下方电池响应格式） |
| MAC_ADDRESS | 0x0B | 返回 6 字节 MAC |
| BOARD_INFO | 0x10 | 返回序列号（13 字节 ASCII），原版不提取 boardType/boardRev |
| RESET | 0x36 | 重启设备，参数 `[0x01]`（对应 `Bt_Ops.requestReboot()`） |

> **注意**：VERSION 响应中的 `version[4]`/`version[5]` 是热词引擎版本（`hot=x.y`），
> 不是硬件版本。原版 APK 无“硬件版本”概念，BOARD_INFO 仅用于提取序列号。

### HID 电池响应格式（CMD_BATTERY_STATE）

来自 `Bt_Ops.smali` `onReport()` 的 `CMD_BATTERY_STATE` 分支逆向。
`SendCommand` 返回的 `data[]` 已去掉 3 字节头部，偏移如下：

| data 偏移 | 字段 | 说明 |
|-----------|------|------|
| `data[0:2]` | voltage | 小端 `(data[1]<<8) | data[0]` |
| `data[2]` | batteryLevel | 枚举序号 0-7（见下方分级） |
| `data[3:5]` | thermistor | 小端 `(data[4]<<8) | data[3]` |
| `data[5:7]` | esr | 小端 `(data[6]<<8) | data[5]` |
| `data[11]` | **percent** | **真正的电量百分比 0-100** |
| `data[12]` | reserve | 非 0 = 储备电量 |

> `batteryLevel` 和 `percent` 是固件独立返回的两个字段，不保证一致。
> 例如固件可能返回 `percent=99` 但 `batteryLevel=5`(GOOD_60)。
> 本工具仅显示 `percent`，不使用 `batteryLevel` 分级。

原版 `BatteryLevel$Classic` 枚举（来自 `BatteryLevel$Classic.smali`）：

| 枚举 | ordinal | max percent |
|------|---------|-------------|
| UNKNOWN | 0 | -1 (sentinel) |
| RESERVE | 1 | 0 |
| EMPTY | 2 | 5 |
| LOW | 3 | 20 |
| GOOD_40 | 4 | 40 |
| GOOD_60 | 5 | 60 |
| GOOD_80 | 6 | 80 |
| FULL | 7 | 100 |

## 设备发现（WMI）

通过 PowerShell 调 `Get-WmiObject Win32_SerialPort` 获取蓝牙 SPP COM 口：

1. `PNPDeviceID` 格式：`BTHENUM\{SPP_UUID}_VID&XXXX_PID&YYYY\7&...&MAC_C00000000`（outgoing）或 `BTHENUM\{SPP_UUID}_LOCALMFG&NNNN\...`（incoming）
2. 同一蓝牙设备分配两个 COM 口：outgoing（有 VID&/PID& 和真实 MAC）和 incoming（LOCALMFG& 和全零 MAC）
3. 只保留 outgoing 端口（含 `VID&`），过滤 incoming 和 MAC 全零端口
4. MAC 提取：从 PNPDeviceID 最后一个 `\` 分隔段提取 12 位 hex（跳过 GUID 段，避免误提 SPP UUID 中的 `00805F9B34FB`）
5. 设备识别：`VID&00020955` + `PID&7214` = NVIDIA Thunderstrike 手柄

## 刷写流程

### 完整序列

```
NOP(握手) → ERASE_SQIF(擦除) → WRITE_SQIF(循环写入.ota, 1024B/块) → VALIDATE_SQIF(校验) → APPLY_OTA(应用) → HID CMD_RESET(确保重启)
```

### APPLY_OTA 后的处理

原版 APK（`BtUpdater.smali` `sendUpdateCore`）在 APPLY_OTA 后：
1. `WriteCommand(APPLY_OTA)` → `sleep(1s)` → `close()`
2. `waitForDisconnect(40s)` — 等待设备断开
3. `waitForConnect(120s)` — 等待重连
4. `waitForDeviceReady(30s)` — 等待就绪

PC 端无法自动重连蓝牙，本工具的策略：
1. `applyOta()` 发送 APPLY_OTA + `sleep(1s)` + 关闭 SPP 串口
2. 刷写完成后通过 HID 接口发送 `CMD_RESET(0x36)` 确保重启
3. 等待 5 秒后检测 HID 设备是否重新连接

### HID CMD_RESET 重启

原版 APK `Bt_Ops.requestReboot()` 调用 `requestData(CMD_RESET, 1)`。
实测确认：发送后设备断开约 1 秒，然后自动重连。

实现见 `src/cmd/helpers.go` 的 `sendHidReset()`：
- 打开 HID 设备 → `SendCommand(CmdReset, [0x01])`
- 设备收到后立即断开，`ReadFile` 超时属正常，只要 `WriteFile` 成功即可

## 固件降级原理

1. Android `BtUpdater` 在升级前检查 `manifest.xml` 版本号，新版本 <= 当前则拒绝
2. 但 SPP 协议的 `ERASE_SQIF` / `WRITE_SQIF` / `APPLY_OTA` **不检查版本号**
3. 只要 `VALIDATE_SQIF` 阶段签名验证通过，即可刷写成功
4. 本工具直接通过 SPP 刷写旧固件，绕过 Android 端降级限制

### 为什么不能通过 USB 刷固件

从 smali 反编译确认：
1. `ThunderstrikeController.getUpdateState()` 检查 USB 连接，USB 连接存在则返回 `DEVICE_UNSUPPORTED`
2. `ThunderstrikeController.getUpdater()` 始终返回 `BtUpdater`（蓝牙 SPP），无 USB 路径
3. HID 报告数据区仅 30 字节，固件几百 KB，SPP 支持 1024 字节/块大传输

## 固件包格式（.blkz）

`.blkz` 是 ZIP 归档（无压缩），包含：

| 文件 | 说明 |
|------|------|
| `manifest.xml` | 固件清单（版本号、配件类型、指纹） |
| `manifest.xml.sig` | 清单的 RSA 签名 |
| `thunderstrike.ota` | 固件二进制数据 |
| `thunderstrike.ota.sig` | 固件的 RSA 签名 |

### blkz 目录规范

1. `blkz/` 目录始终和程序放在一起
2. 文件名规范：`设备代号_版本号.blkz`，如 `Thunderstrike_0x010E.blkz`
3. 多语言固件：`设备代号_locale_版本号.blkz`
4. 刷机时先将 `.blkz` 解压到同名子目录
5. 版本号从 `manifest.xml` 读取，不依赖文件名

### 固件校验

- `.sig` 签名文件不发送给手柄（原版用 RSA 验签，本工具用 MD5 校验）
- 内置已知固件 MD5，防止刷错固件导致变砖

## 升级时间参数

来自 `ThunderstrikeController.smali` 的 `getUpgradeTimings()`：

| 参数 | 值 | 说明 |
|------|------|------|
| EraseMs | 0x960 = 2400ms | SQIF 擦除时间 |
| BytesPerSec | 0x6784 = 26484 | 估计写入速度 |
| ApplyDelayMs | 0x157C = 5500ms | Apply OTA 延迟 |
| ReadyTimeMs | 0x251C = 9500ms | 设备就绪等待 |
| ReconnectMs | 0 | 重连延迟（非 locale 版本为 0） |

## 固件版本

| 版本 | 代号 | 说明 |
|------|------|------|
| v1.14 | 0x010E | 不能XY |
| v1.18 | 0x0112 | 推荐版本 |
| v1.33 | 0x0121 | 不能A+B |
| v1.36 | 0x0124 | 不能A+B |

### 固件提取位置

手柄固件 `.blkz` 包位于 SHIELD TV 恢复镜像的 ext4 分区中：

```
nv-recovery-image-shield-2017-atv-9.2.1\vendor.ext4\oem\firmware\
```

解压恢复镜像后，在 `vendor.ext4` 分区的 `oem/firmware/` 目录下找到固件包。

实测：V1.33 → V1.18 降级成功，CSR 芯片版本同步降级（v0.18 → v0.17），热词引擎同步降级（v1.05 → v1.04）。

## 界面层架构

界面只有一套（原生 GUI），与业务层解耦：

| 包 | 技术 | 说明 |
|---|---|---|
| `src/gui/` | MyGo 原生自绘 UI（`github.com/egoist/mygo v0.7.1`，版本锁定） | 窗口 780×440，深色单屏，页面状态机 + 5 个视图文件 |

- `src/core/` 定义 `DeviceScanner`、`FirmwareScanner`、`Flasher`、`GamepadMonitor` 四个接口与全部数据类型，`gui` 只依赖 `core`，不碰业务实现。
- 业务实现集中在 `src/cmd/adapter.go`（界面适配器）与 `src/cmd/gamepad_monitor.go`。
- 所有业务包位于 `src/` 下，main.go 留在仓库根目录（`build.bat` 依旧在根目录执行 `go build .`）。
- **线程模型**：界面状态只在主线程读写。后台 goroutine（设备扫描、HID 查询、刷写回调）统一通过 `Model.post` → `mygo.Window.Update` 投递修改，禁止直接改字段；`post` 在窗口尚未建立时同步执行，便于单元测试。
- **Go 版本**：mygo v0.7.1 要求 Go ≥ 1.27.3（`go.mod` 已声明）。低于该版本时 `GOTOOLCHAIN=auto` 会自动下载工具链，首次构建需要联网。
- **DPI**：窗口尺寸按 DIP 计，实际物理像素 = 尺寸 × 系统缩放比例（本机 150% 时 780×440 约合 1170×660）；内容超出可视高度时由 `ui.Scroll` 滚动。
- **布局要点**：`Spacer` 本身即吸收剩余空间，不要再叠加 `Grow`；根容器必须用 `c.Size()` 显式设置宽高，否则内部 `FillWidth` 缺少参照会导致横向溢出。

### 视觉规范

对齐 NVIDIA 官方品牌（只有绿 `#76B900`、黑、白三色）：

- **深色为唯一形态**，不跟随系统主题：背景 `#0F0F0F`、面板 `#171717`、描边 `#2A2A2A`。
- **单一品牌绿**：强调、成功、在线、进度全部用 `#76B900`，不再另设成功色，避免一个绿兼五种语义。
- **斜切是唯一的装饰语言**（NVIDIA 标识体系的倾斜感）：页头 logo 用 `Skew(-8, 0)`；纯色块可直接斜，带文字的容器需再 `Skew(+8, 0)` 抵消文字倾斜。**内置按钮不吃 Skew**，主按钮保持直角以保证键盘焦点与 hover 反馈。
- **数字承载信息**：版本号 / 电量 32px，刷写百分比 56px，其余信息 10–12px 灰字。信息层级靠字号拉开，不靠颜色。
- **自绘标题栏**：`TitleBarStyle: mygo.TitleBarHidden` + `TitleBarHeight = 48`。系统标题栏是浅色的，会与深色界面割裂；隐藏后页头用 `c.TitleBar().Right` 让出窗口控件宽度，并调用 `DragWindow()` 作为拖拽区。
- **版式常量**：`headerHeight 48 / footerHeight 40 / contentPadding 14` 统一定义在 `theme.go`，视图计算可用高度时引用 `bodyHeight(c)`。
- **交互简化**：只有一台手柄时扫描后直接进设备信息页，省掉列表这一屏；多设备才出列表。

### 手柄输入监听

`src/hid/input_monitor.go` 负责把「手柄被按了」变成界面反馈（右上角指示器 + 底部状态栏）。三个关键点：

- 手柄注册了**两个** HID 集合（蓝牙 HID 服务路径 `hid#{00001124-...}` 与物理设备路径 `hid#vid_0955&pid_7214#...`，均 UsagePage=0x0001/Usage=0x0005，输入报告 233 字节）。按键上报可能落在任一个，所以 `findThunderstrikeHidPaths()` 返回全部路径，监听器为每个集合起一个阻塞读循环。
- 每个句柄**阻塞读，不做超时取消**。早期实现每 500ms `CancelIoEx` 强杀句柄，导致手柄静止时一直在「打开-取消-断开-重连」循环，按键落在重连空窗期就丢。现在只有 `Close()` 才打断读。
- 设备不在线时每 2 秒重试，手柄休眠或被刷写重启后无需重启程序；`onReady` 回调让界面区分「等待手柄 / 手柄就绪 / 手柄活动」三态。

诊断命令：

```
go test ./src/hid -run TestListHidCollections -v   # 列出手柄所有 HID 集合
```

### GUI 视图测试

`src/gui/views_test.go` 用 `ui.NewTester` 在内存中渲染，不需要打开窗口：

```
go test ./src/gui
```

- `TestSnapshots` 把 9 种页面状态渲染为 PNG，输出到 `.opencode/temp/renders/`（可用 `GUI_SNAPSHOT_DIR` 覆盖），用于人工核对布局。
- `TestClickDeviceOpensDetail`、`TestClickFirmwareOpensLanguages` 用 `Tester.ClickAt` 验证点击链路。

## 项目结构

```
ThunderstrikeControllerTool/
├── main.go                         # 程序入口
├── src/                            # 全部业务与界面代码（import 前缀 src/）
│   ├── core/                       # 界面层共用类型与接口（不依赖任何 UI）
│   │   ├── types.go                # 设备/固件/刷写数据结构 + 步骤与语言表
│   │   └── interfaces.go           # 四个业务接口 + Services 聚合
│   ├── gui/                        # 原生 GUI 层（MyGo 自绘）
│   │   ├── run.go                  # 窗口、页面状态机、异步流程
│   │   ├── theme.go                # 主题、徽章、卡片、进度条等通用组件
│   │   ├── keys.go                 # 键盘快捷键与页面返回/刷新逻辑
│   │   ├── view_devices.go         # 扫描状态 / 设备列表
│   │   ├── view_detail.go          # 设备信息与电量
│   │   ├── view_firmware.go        # 固件列表与语言选择
│   │   ├── view_flash.go           # 刷写进度与结果
│   │   └── views_test.go           # 渲染快照 + 点击链路测试
│   ├── cmd/                        # 业务逻辑层
│   │   ├── adapter.go              # 界面适配器（设备扫描、HID 查询、刷写）
│   │   ├── flash_core.go           # 刷写核心逻辑
│   │   ├── bt_discovery_windows.go # WMI 查询蓝牙 SPP COM 口
│   │   ├── helpers.go              # 辅助函数（HID 查询、序列号提取、电池查询）
│   │   └── gamepad_monitor.go      # 手柄输入监听适配器
│   ├── hid/                        # HID 协议
│   │   ├── commands.go             # 159 个命令枚举 + 协议常量
│   │   ├── windows_client.go       # Windows HID API 客户端 + BatteryInfo 解析
│   │   ├── input_monitor.go        # HID 输入监听（多集合阻塞读 + 自动重连）
│   │   ├── input_monitor_test.go   # 监听器就绪/关闭测试
│   │   └── hid_probe_test.go       # 诊断：列出所有 HID 集合
│   ├── spp/                        # 蓝牙 SPP 协议栈
│   │   ├── protocol.go             # 命令码、状态码、超时、UpgradeTimings
│   │   ├── client.go               # SPP 客户端（WriteCommand/ReadResponse/Connect）
│   │   ├── flasher.go              # 刷写流程（NOP→ERASE→WRITE→VALIDATE→APPLY）
│   │   └── hidraw_client.go        # [遗留] HID-over-SPP 客户端（已被 HID API 替代）
│   ├── firmware/                   # 固件解析
│   │   ├── blkz.go                 # .blkz 包解析+解压
│   │   ├── blkz_test.go            # blkz 解析测试
│   │   ├── manifest.go             # manifest.xml 解析
│   │   └── checksum.go             # MD5 校验（内置 6 个已知固件哈希）
│   └── logger/                     # 刷写日志记录
│       └── logger.go               # 时间戳日志文件
├── Release/                        # 编译打包目录
│   ├── Release.zip                 # 包含了手柄固件+刷写工具
│   ├── tsct.exe                    # 编译产物
│   ├── blkz/                       # 固件包目录（6 个 .blkz）
│   └── logs/                       # 刷写日志目录
```

## 逆向分析来源

逆向分析所用 APK 来自 SHIELD TV 2017 恢复镜像：

```
nv-recovery-image-shield-2017-atv-9.2.1\vendor.ext4\app\NvAccessories\NvAccessories.apk
```

| smali 文件 | 提取的信息 |
|-----------|-----------|
| `Bt_Ops.smali` | HID 命令格式、`requestData()`/`onReport()` 方法、`requestReboot()`、电池响应解析 |
| `Bt_Ops$BtHidrawCommand.smali` | 159 个 HID 命令枚举及 ordinal（0x00–0x9E） |
| `Bt_Ops$Callbacks.smali` | 回调接口（`onBatteryLevelChange(IIZIII)` 等签名） |
| `BatteryLevel.smali` | 电池百分比封装、`fromClassic()`/`fromPercent()` |
| `BatteryLevel$Classic.smali` | 8 级电池分级枚举（UNKNOWN/RESERVE/EMPTY/LOW/GOOD_40/60/80/FULL） |
| `ThunderstrikeController.smali` | Report 长度（30B）、升级时间参数 |
| `BtUpdater.smali` | SPP 命令/状态枚举、刷写流程（`sendUpdateCore`） |
| `BtUpdater$Command.smali` | SPP 命令包格式：`[cmd][2B LE len][data]` |
| `BtUpdater$Response.smali` | SPP 响应包格式：`[cmd][status][len][payload]` |
| `FirmwareFile.smali` | `.blkz` 格式、`manifest.xml` 结构 |

### 内核源码交叉验证

SHIELD TV 2017 内核源码关键发现：
- `TS_HOSTCMD_REPORT_SIZE = 33` — 确认 HID 报告总长度
- `USB_DEVICE_ID_NVIDIA_THUNDERSTRIKE = 0x7214` — 确认 PID
- ICM-20628 6 轴 IMU（加速度计+陀螺仪）
- 音频支持 ADPCM（Report ID=30）和 mSBC（Report ID=0xF7/0xFA/0xFB）
