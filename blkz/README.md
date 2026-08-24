# 固件名称规范

固件名称要规范，涉及到验证解压和刷机。  
程序记录了几个版本的固件 MD5 值。  
MD5 值不对或未知的版本需要谨慎刷机。

## 规范

手柄代号_固件版本号.blkz

### 标准版

| 文件名 | 版本 | 说明 |
|--------|------|------|
| Thunderstrike_0x010E.blkz | v1.14 | 旧版，从 SHIELD TV ROM 提取 |
| Thunderstrike_0x0112.blkz | v1.18 | 中间版本 |
| Thunderstrike_0x0124.blkz | v1.36 | 新版 |

### 多语言版

| 文件名 | 版本 | 说明 |
|--------|------|------|
| Thunderstrike_locale_0x0124.blkz | v1.36 | 15 种语言：da_dk, de_de, en_au, en_gb, en_ie, es_es, es_us, fi_fi, fr_fr, it_it, ja_jp, ko_kr, nb_no, nl_nl, sv_se |

## 解压

刷机时程序会自动将 `.blkz` 解压到同名子目录，如：

```
Thunderstrike_0x0112.blkz → Thunderstrike_0x0112/
  ├── manifest.xml
  ├── manifest.xml.sig
  ├── thunderstrike.ota
  └── thunderstrike.ota.sig
```

多语言版解压后包含多个语言的 `.ota` 文件：

```
Thunderstrike_locale_0x0124.blkz → Thunderstrike_locale_0x0124/
  ├── manifest.xml
  ├── manifest.xml.sig
  ├── thunderstrike_da_dk.ota + .sig
  ├── thunderstrike_de_de.ota + .sig
  └── ... (共 15 种语言)
```
