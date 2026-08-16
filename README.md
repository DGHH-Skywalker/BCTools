# 小播点歌工具（BCTools）

面向校园广播站的本地点歌与歌单管理工具。单文件 `.exe`，双击即用，数据全部留在本机。

![版本](https://img.shields.io/badge/version-5.5.0.0-0086C3)
![平台](https://img.shields.io/badge/platform-Windows%2010%2F11-blue)
![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)
![Vue](https://img.shields.io/badge/Vue-3-4FC08D?logo=vue.js&logoColor=white)

## 这个工具解决什么问题

学校广播站每周要排一份宿舍播放歌单：哪天几点放哪首歌，然后把音频按播放顺序
命名成 `01.mp3`、`02.mp3` 拷到 SD 卡插进播放器。手工做这件事很容易出错——
漏了一个时段，后面所有曲序就整体错位。

BCTools 把这套流程做成了：

- **排歌单**：按「周 × 时段」的网格拖拽安排歌曲，自动查重（默认回溯 30 天）
- **导入音频**：拖进来自动转码成 MP3，支持网易云 `.ncm` 等加密格式
- **换卡导出**：一键把整周歌曲按序号复制到 SD 卡，**空时段自动生成静音占位**，
  保证曲序与贴出的歌单严格对应
- **出图**：把歌单渲染成海报或表格图片，直接发群里
- **手机点歌**：局域网内扫码，同学用手机就能提交

## 快速开始

到 [Releases](../../releases) 下载：

| 文件 | 说明 |
|------|------|
| `BCTools-Setup.exe` | 安装程序（推荐） |
| `bctools.exe` | 免安装单文件版 |

运行后自动打开浏览器访问 `http://localhost:1743/`，托盘会出现图标：
**左键**打开界面，**右键**菜单可打开界面或退出后端。

> 内置精简版 ffmpeg/ffprobe，无需另装。首次运行会释放到
> `%APPDATA%\BroadcastTool\bin`。

## 功能一览

| 页面 | 作用 |
|------|------|
| 宿舍歌单 `/dorm/manage` | 按周 × 时段拖拽排歌，第 8 张「待分配」卡片收纳未分配时段的歌 |
| 时段配置 `/dorm/manage/slots` | 自定义每天的播放时间点 |
| 播音歌单 `/broadcast` | 午间/午后播音歌单，支持 Excel 列映射导入 |
| 歌单导出 `/export` | 导出海报图/表格图，可设背景、模糊、文案、假期角标 |
| 换卡工具 `/organize` | 按序号复制整周音频到 SD 卡，空位生成静音占位 |
| 音乐解锁 `/um-react` | 内置 [Unlock Music](https://git.unlock-music.dev/um/web)，解密后一键导入 |
| 手机点歌 `/mobile` | 显示局域网二维码 |

## 技术栈

- **前端**：Vue 3 + TypeScript + Vite + Naive UI + Pinia + vue-router（hash 模式）
- **后端**：Go + chi/v5 + `embed.FS`（前端与 ffmpeg 全部嵌进单个 exe）
- **托盘**：`fyne.io/systray`（纯 Go，无需 cgo）
- **音频**：内嵌精简版 ffmpeg/ffprobe；`.ncm` 等解密复用
  [unlock-music](https://git.unlock-music.dev/um/cli) 解码器
- **安装程序**：C++ Win32/GDI+ 手写（`installer/`）

## 从源码构建

需要：Go 1.25+、Node.js 18+、Python 3.10+，Windows。
构建安装程序还需 MinGW-w64（`g++` 在 PATH 中）。

```bash
git clone --recursive <repo-url> BCTools
cd BCTools
python build.py
```

产物输出到 `dist/`：

| 产物 | 说明 |
|------|------|
| `bctools.exe` | 主程序（单文件，内嵌前端 + ffmpeg） |
| `bctool_dev.exe` | 开发模式启动器（替换已运行的后端） |
| `bctools_test.exe` | 一次性测试版：端口 712，数据写 `%TEMP%`，退出即清 |
| `BCTools-Setup.exe` | 安装程序 |

常用参数：

```bash
python build.py --skip-front      # 跳过前端构建
python build.py --skip-um-react   # 跳过 um-react 子模块
python build.py --skip-installer  # 跳过安装程序
python build.py --clean           # 清理产物
```

> `um-react` 是 Git submodule。忘记 `--recursive` 的话补一句
> `git submodule update --init --recursive`。

## 开发

```bash
# 前端
cd Web
npm install --legacy-peer-deps
npm run dev          # 开发服务器，API 代理到 :1743
npm run type-check
npm run test

# 后端
cd GoServer
go vet ./...
go test ./...
```

## 目录结构

```
BCTools/
├── GoServer/            Go 后端
│   ├── handlers/        HTTP 接口（含 lifecycle：health/watch/shutdown）
│   ├── routes/          路由注册
│   ├── services/        业务编排（导入、换卡、设置更新）
│   ├── store/           领域数据访问
│   ├── internal/repo/   data.json 原子写入与并发控制
│   ├── converter/       音频转码 / 解密 / 格式嗅探
│   ├── platform/        系统交互（隐藏窗口执行命令、打开浏览器）
│   ├── tray/            任务栏托盘控件
│   └── main.go
├── Web/                 Vue 3 前端
│   ├── src/{views,components,stores,composables,api,utils,styles,constants}
│   └── um-react/        Unlock Music 子模块
├── installer/           C++ Win32/GDI+ 安装/卸载程序
├── docs/                设计与交接文档
└── build.py             一键构建脚本
```

API 文档见 [`GoServer/docs/api.md`](GoServer/docs/api.md)。

## 数据目录

所有数据留在本机 `%APPDATA%\BroadcastTool\`：

```
├── data.json               主数据（歌单 + 设置）
├── deleted_songs_log.json  删除日志
├── songs/                  歌曲文件
├── snapshots/              每日快照备份
├── decrypt-staging/        um-react 解密暂存
├── temp/                   临时目录（启动时清空）
├── logs/                   滚动日志
└── bin/                    运行时释放的 ffmpeg/ffprobe
```

## 已知限制

- **无鉴权**：不实现后端 session/JWT，任何能访问该端口的设备都能调用管理接口。
  **只在可信局域网内运行**。`POST /api/shutdown` 是例外，仅接受本机请求。
- `data.json` 全量读写，当前数据量下足够；极大数据量需改造。
- 前端 `en.json` 保留结构但未启用，默认 locale 为 `zh-CN`。
- 仅面向 Windows 10/11；其他平台可编译，但托盘等功能为空实现。

## 许可证

代码以 [MIT](LICENSE) 发布。

**注意**：仓库中的中文字体（江西拙楷、方正颜宋等）与图片素材**不在 MIT 范围内**，
版权归各自作者，商用请自行确认授权。内嵌 ffmpeg 遵循其自身许可（LGPL/GPL）；
`.ncm` 解密能力来自 unlock-music，请仅用于解锁自己已购买的音乐。
