# Thunderstrike Controller Tool

> NVIDIA SHIELD TV 2017 Thunderstrike 手柄命令行工具，
> 支持设备信息读取和固件刷写/降级（蓝牙 SPP）。
> 仅支持 Windows，仅通过蓝牙连接。

## 项目背景

NVIDIA SHIELD TV 2017（中国版）配套的 **Thunderstrike** 手柄基于 CSR 蓝牙芯片。

- **旧固件** `v1.14`（`0x010e`）— 从 ROM 中提取
- **新固件** `v1.33`（`0x0121`）— 当前手柄安装版本
- **locale 固件** `v1.36`（`0x0124`）— 多语言版本

Android 系统的 `BtUpdater` 在升级时检查固件版本号，拒绝降级。
但 SPP 协议本身不检查版本号——只要签名匹配即可刷写。
本工具直接通过 SPP 协议与手柄通信，绕过 Android 端的降级限制。

## 当前状态

| 功能 | 状态 | 说明 |
|------|------|------|
| 交互模式 | ✅ 已实现 | 扫描蓝牙 SPP → 探测手柄 → 显示信息 → 刷写固件 |
| `scan` — 扫描蓝牙串口 | ✅ 已实现 | 列出系统所有 COM 串口 |
| `flash` — 刷写固件 | ✅ 已实现 | SPP 协议完整实现，`--dry-run` 可验证固件包 |
| 设备信息读取 | ✅ 已实现 | 通过蓝牙 SPP 读取版本/MAC/序列号/传输方式 |
| 固件列表 | ✅ 已实现 | 扫描 blkz/ 目录，显示版本/大小/MD5 校验状态 |
| 交互模式刷写 | ✅ 已实现 | 选择设备 → 选择固件 → MD5 校验 → 刷写 |
| 多语言固件 | ✅ 已实现 | 支持 locale 版本的 .ota 选择 |
| 蓝牙设备探测 | ✅ 已实现 | 扫描时对每个 COM 口发送 NOP，识别手柄 |

> **USB 支持已移除**：原版应用不支持 USB 刷固件，本工具仅通过蓝牙 SPP 通信。

## 系统要求

- **操作系统**：Windows 10/11
- **蓝牙**：支持蓝牙 4.0+ 的适配器（笔记本内置即可）
- **Go 版本**：1.21+（仅编译时需要，已附带编译好的 `tsct.exe`）

## 安装

### 直接使用

已附带编译好的 `tsct.exe`，直接运行即可。

### 从源码编译

```powershell
cd C:\Users\letmu\Downloads\Souce\ThunderstrikeControllerTool
$env:GOPROXY = "https://goproxy.cn,direct"
go mod tidy
go build -o tsct.exe .
```

## 使用方法

### 交互模式（推荐）

直接运行程序（不带参数）：

```
tsct
```

流程：
1. 扫描蓝牙 SPP 串口 → 对每个 COM 口发送 NOP 探测，识别手柄
2. 列出设备：`Thunderstrike (蓝牙) COM3` 或 `蓝牙串口 (SPP) COM4`
3. 选择设备编号 → 显示设备信息（版本/MAC/序列号/传输方式）
4. 提示「是否刷写固件?」→ 输入 `y`
5. 列出 blkz/ 目录下的固件包 → 选择固件
6. MD5 安全校验 → 多语言固件选择语言
7. 确认后开始刷写（NOP → ERASE → WRITE → VALIDATE → APPLY）

### 命令行模式

**扫描蓝牙串口：**
```
tsct scan
```

**直接刷写固件：**
```powershell
tsct flash --port COM3 -f firmware.blkz
```

**试运行（仅解析固件，不刷写）：**
```powershell
tsct flash -f firmware.blkz --dry-run
```

## 通信协议

### 蓝牙 SPP（设备信息查询 + 固件刷写）

本工具仅通过蓝牙 SPP 通信。SPP 连接同时支持两种协议：

#### HID-over-SPP（设备信息查询）

与 USB HID 相同的报告格式，通过 SPP 串口传输：

```
请求: [0x04] [cmd_ordinal] [transaction_id] [data...] [zero-padded to 33 bytes]
响应: [0x03] [cmd_ordinal] [transaction_id] [response_data...]
```

- 请求前缀 = `0x04`（Output Report）
- 响应前缀 = `0x03`（Input Report）
- Report 长度 = 33 字节（`mReportLen = getHidrawReportLength() + 3 = 30 + 3`）
- 最大响应 = 65 字节（`MAX_RESPONSE_SIZE = 0x41`）
- Transaction ID 自增匹配，解决缓冲区残留问题
- 设备对不支持的指令返回 `0xFF` 作为 cmd_ordinal（CMD_ERROR）

**支持的查询指令：**

| 指令 | Code | 说明 |
|------|------|------|
| VERSION | 0x01 | 固件版本 + CSR 芯片版本 + 热词引擎版本 |
| BOARD_INFO | 0x10 | 硬件版本 + 序列号 |
| MAC_ADDRESS | 0x0B | MAC 地址 |
| SET_TRANSPORT | 0x49 | 传输方式（0x02=wireless） |

#### SPP 刷写协议（固件刷写）

**命令格式**（PC → 手柄）：
```
[1 byte CommandCode] [2 bytes LE length] [N bytes data]
```

**响应格式**（手柄 → PC）：
```
[1 byte CommandCode] [1 byte StatusCode] [1 byte payload_len] [N bytes payload]
```

**命令序列**（基于 smali 逆向确认）：

| 步骤 | 命令 | Code | 超时 | 说明 |
|------|------|------|------|------|
| 1a | NOP | 0x00 | 5s | WriteCommand（唤醒设备） |
| 1b | NOP | 0x00 | 5s | ExecuteCommand（握手确认） |
| 2 | ERASE_SQIF | 0x01 | 20s | 擦除 SQIF 闪存，参数 `[0x00]` |
| 3 | WRITE_SQIF | 0x02 | 2s/块 | 分块写入，1024 字节/块，每块等 ACK |
| 4 | VALIDATE_SQIF | 0x03 | 20s | 验证写入数据 |
| 5 | APPLY_OTA | 0x05 | — | 应用固件，参数 `[0x00]`，发送后 sleep(1s) |
| 6 | 等待重启 | — | — | close() → 设备自动重启 |

**状态码**：

| 状态 | Code | 说明 |
|------|------|------|
| COMPLETED_OK | 0x00 | 成功 |
| IN_PROGRESS | 0x01 | 执行中（自动重试，ReadResponse 中跳过） |
| FAILED | 0x02 | 失败 |
| TIMEOUT | 0x03 | 超时 |
| FAILED_LOW_BATTERY | 0x06 | 电量不足 |
| FAILED_SIGNATURE | 0x07 | 签名验证失败 |
| FAILED_FLASH_ERROR | 0x08 | 闪存错误 |

> **注意**：`ReadResponse` 只跳过 `STATUS_CMD_IN_PROGRESS`（与 smali 一致），不跳过 `STATUS_DIAG_IN_PROGRESS`。

### NOP 握手细节

原版 smali 中 NOP 握手发送**两次** NOP：
1. `WriteCommand(NOP)` — 唤醒设备（不读响应）
2. `ExecuteCommand(NOP)` — 再次发送 NOP + 读响应确认

本工具的 `Client.Connect()` 已实现此行为。

### APPLY_OTA 后的处理

原版应用在 APPLY_OTA 后：close → 等待断开(30s) → 重连延迟 → 等待重连(120s) → 等待就绪(30s)。

PC 端无法自动重连蓝牙，本工具在 APPLY_OTA 后直接关闭串口返回，提示用户手柄会自动重启。

## 固件包格式（.blkz）

`.blkz` 是 ZIP 归档（无压缩），包含：

| 文件 | 说明 |
|------|------|
| `manifest.xml` | 固件清单（版本号、配件类型、指纹） |
| `manifest.xml.sig` | 清单的 RSA 签名 |
| `thunderstrike.ota` | 固件二进制数据 |
| `thunderstrike.ota.sig` | 固件的 RSA 签名 |

### blkz 目录规范

1. `blkz/` 目录始终和程序放在一起，存放 `.blkz` 固件包
2. 文件名规范：`设备代号_版本号.blkz`，如 `Thunderstrike_0x010E.blkz`
3. 多语言固件：`设备代号_locale_版本号.blkz`，如 `Thunderstrike_locale_0x0124.blkz`
4. 刷机时先将 `.blkz` 解压到同名子目录，如 `blkz/Thunderstrike_0x010E.blkz` → `blkz/Thunderstrike_0x010E/`
5. 版本号从 `manifest.xml` 读取，不依赖文件名解析

### 固件校验

- **MD5 校验**：内置已知固件的 MD5 值，防止刷错固件导致变砖
- `.sig` 签名文件不发送给手柄，是 PC 端验证用的（原版用 RSA 验签）
- 本工具采用 MD5 校验方案（比 RSA 验签简单）

### 固件刷写流程

发送给手柄的是 `.blkz` 解压后的 `.ota` 文件，不是 `.blkz` 本身：

```
NOP(握手) → ERASE_SQIF(擦除) → WRITE_SQIF(循环写入.ota, 1024字节/块) → VALIDATE_SQIF(校验) → APPLY_OTA(应用)
```

## 固件降级原理

1. Android 的 `BtUpdater` 在升级前检查 `manifest.xml` 中的版本号
2. 如果新版本号 <= 当前版本号，Android 端拒绝升级
3. 但 SPP 协议的 `ERASE_SQIF` / `WRITE_SQIF` / `APPLY_OTA` **不检查版本号**
4. 只要在 `VALIDATE_SQIF` 阶段签名验证通过，即可刷写成功
5. 因此直接通过 SPP 刷写旧固件即可绕过降级限制

### 为什么不能通过 USB 刷固件？

从 smali 反编译确认，**原版应用不支持 USB 刷固件**：

1. `ThunderstrikeController.getUpdateState()` 检查 USB 连接，如果 USB 连接存在则返回 `DEVICE_UNSUPPORTED`
2. `ThunderstrikeController.getUpdater()` 始终返回 `BtUpdater`（蓝牙 SPP），没有 USB 路径的 updater
3. `BtBaseController.updateAccessory()` 调用 `getUpdater()` 来获取 updater 实例

USB HID 虽然定义了 OTA 命令（0x21-0x24），但上层应用从未在 USB 路径上使用它们。
HID 报告数据区仅 30 字节，而固件有几百 KB，SPP 协议支持 1024 字节/块的大数据传输。

## 项目结构

```
ThunderstrikeControllerTool/
├── go.mod                          # Go 模块定义
├── main.go                         # 程序入口
├── tsct.exe                        # 编译好的可执行文件
├── README.md
├── blkz/                            # 固件包目录
│   ├── Thunderstrike_0x010E.blkz   # v1.14 标准固件
│   ├── Thunderstrike_locale_0x0124.blkz  # v1.36 多语言固件
│   └── README.md                   # 固件目录说明
├── cmd/                            # CLI 命令
│   ├── root.go                     # 根命令注册
│   ├── scan.go                     # scan — 扫描蓝牙 SPP 串口
│   ├── interactive.go              # 交互模式（扫描+探测+选择+信息）
│   ├── interactive_flash.go        # 交互模式刷写流程
│   ├── bt_info.go                  # 蓝牙设备信息读取
│   ├── blkz_list.go                # 固件列表显示
│   ├── flash.go                    # flash — 刷写命令（命令行模式）
│   └── flash_core.go               # 刷写核心逻辑（共用）
├── hid/                            # HID 协议定义
│   └── commands.go                 # 命令常量 + ErrCmdError + FormatMacAddress
├── spp/                            # 蓝牙 SPP 协议栈
│   ├── protocol.go                 # 命令码、状态码、超时、UpgradeTimings
│   ├── client.go                   # SPP 刷写客户端（WriteCommand/ReadResponse/Connect）
│   ├── flasher.go                  # 完整刷写流程（NOP→ERASE→WRITE→VALIDATE→APPLY）
│   └── hidraw_client.go            # HID-over-SPP 客户端（设备信息查询 + ProbeController）
├── firmware/                       # 固件解析
│   ├── blkz.go                     # .blkz 包解析+解压
│   ├── manifest.go                 # manifest.xml 解析
│   └── checksum.go                 # MD5 校验
└── decompiled/                     # APK 逆向分析源码
    ├── NvAccessories/              # 主应用 APK（smali）
    └── NvShieldTech/               # Shield 技术库（smali）
```

## 逆向分析来源

协议信息来自对以下 APK 的逆向分析（反编译文件位于 `decompiled/` 目录）：

| 来源 | 提取的信息 |
|------|-----------|
| `Bt_Ops.smali` | HID 命令格式、`requestData()` 方法、指令分类、响应解析逻辑 |
| `Bt_Ops$BtHidrawCommand.smali` | 159 个 HID 命令枚举及 ordinal（0x00–0x9E） |
| `ThunderstrikeController.smali` | Report 长度（30B）、升级时间参数 |
| `BtUpdater.smali` | SPP 命令/状态枚举、刷写流程（sendUpdateCore） |
| `BtUpdater$Command.smali` | SPP 命令包格式：`[cmd][2B LE len][data]` |
| `BtUpdater$Response.smali` | SPP 响应包格式：`[cmd][status][len][payload]` |
| `FirmwareFile.smali` | `.blkz` 格式、`manifest.xml` 结构 |

### SHIELD TV 2017 内核源码（交叉验证）

WSL 中保存了 SHIELD TV 2017 的完整内核源码：
```
\\wsl.localhost\Ubuntu-22.04\home\leon\SHIELDTV2017\
```

关键发现：
- `TS_HOSTCMD_REPORT_SIZE = 33` — 确认 HID 报告总长度
- `USB_DEVICE_ID_NVIDIA_THUNDERSTRIKE = 0x7214` — 确认 PID
- ICM-20628 6 轴 IMU（加速度计+陀螺仪）
- 音频支持 ADPCM（Report ID=30）和 mSBC（Report ID=0xF7/0xFA/0xFB）

## 注意事项

⚠️ **刷写固件有风险，请确保：**
- 电池电量充足（> 20%）
- 刷写过程中不要断开蓝牙连接
- 手柄休眠后会断开 SPP 连接，刷写前请操作手柄保持唤醒
- 保留一份已知良好的固件包以备恢复

⚠️ **蓝牙连接说明：**
- 手柄需要先与电脑蓝牙配对，配对后 Windows 会分配 SPP COM 端口
- 蓝牙列表中 COM3 和 COM4 都出现时，可能没有真正连接的设备
- 扫描时会自动探测每个 COM 口，标记为 `Thunderstrike (蓝牙)` 的是手柄

### 固件版本说明

| 版本 | 代号 | 说明 |
|------|------|------|
| v1.14 | 0x010E | 旧版，从 SHIELD TV SW 6.1.0 中国版 ROM 提取 |
| v1.33 | 0x0121 | 当前手柄安装版本 |
| v1.36 | 0x0124 | 多语言（locale）版本 |

SPP 协议不检查版本号，只要签名匹配即可刷写。

## 许可证

本项目仅供学习和研究使用。
NVIDIA、SHIELD、Thunderstrike 是 NVIDIA Corporation 的商标。
