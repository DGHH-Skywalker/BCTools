# BCTools

小播点歌工具，用于校园广播站的宿舍歌单、播音歌单、音频整理和 SD 卡导出。

[![Release](https://img.shields.io/github/v/release/DGHH-Skywalker/BCTools?include_prereleases&label=Release)](https://github.com/DGHH-Skywalker/BCTools/releases)
![Platform](https://img.shields.io/badge/platform-Windows%20x64-blue)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

> [!IMPORTANT]
> v6.0.0 目前处于 Beta。新版使用拆分后的数据布局，升级前请退出旧版并完整备份 `BctoolData`。需要稳定版时，请使用 v5.8.0.0。

## 下载与启动

从 [Releases](https://github.com/DGHH-Skywalker/BCTools/releases) 下载 Windows x64 版本：

| 文件 | 用途 |
| --- | --- |
| `BCTools-Setup.exe` | 安装程序，支持当前用户或所有用户安装 |
| `bctools.exe` | 便携版，无需安装 |
| `SHA256SUMS.txt` | 下载文件校验值 |
| `BCTools-UpdateTargets.zip` | 维护者发布签名更新时使用的目标文件 |
| `FFmpeg-Sources.tar.gz` | 内置 FFmpeg、LAME 的对应源码、构建脚本和许可证 |

BCTools 有两种使用方式：

```powershell
# 默认浏览器，地址为 http://localhost:1743/
.\bctools.exe

# 独立桌面窗口，需要 Microsoft Edge WebView2 Runtime
.\bctools.exe desktop
```

两种模式共用同一个 Go 后端和数据目录，局域网设备仍可通过 `1743` 端口访问。重复启动会唤回已有实例；从浏览器模式进入桌面模式时，由原后端完成交接。托盘菜单可重新打开界面或退出程序。

```powershell
# 关闭正在运行的实例
.\bctools.exe --shutdown
```

## 功能

- **宿舍歌单**：按周和时段安排歌曲，支持拖拽排序、待分配歌曲和重复提示。
- **播音歌单**：管理午间、午后栏目，支持 Excel 批量导入；覆盖导入前会创建可恢复的快照。
- **音频导入**：常见音频统一转为 MP3；支持 NCM，以及 KGG、KGM、KGMA、VPR 等酷狗加密格式。酷狗取钥数据库可自动查找，也可在设置中手动指定。
- **歌曲文件导出**：按周和播放顺序写入 SD 卡。空时段生成 17.43 秒静音占位，同一时段的多首歌按顺序合并。
- **歌单图片**：导出简约表格或海报图片，可配置背景、模糊、文案和徽章。
- **手机点歌**：通过局域网二维码打开点歌页面。
- **音乐解锁**：内嵌 um-react，解密结果可回传主程序导入。
- **数据维护**：按保留期清理音频和歌曲信息，清理内容先进入七天回收区。
- **扩展与更新**：支持页面、布局和 WASM 命令插件；提供基于 TUF 的签名更新机制。

## 数据与升级

正式版的数据保存在程序所在目录的 `BctoolData`：

```text
BctoolData/
├── data.json                    # 存储版本、设置、下一歌曲 ID 和周目录索引
├── songs-without-audio.json     # 没有对应音频的歌曲记录
├── MusicFiles/
│   ├── _staging/                # 待归档音频
│   └── <YYYY年第N周>/            # 周音频、songs.json 和导出索引
├── snapshots/                   # 覆盖导入等操作前的快照
├── recycle/                     # 保留期清理的七天回收区
├── plugins/                     # 插件包与启用状态
├── updates/                     # 已验证的更新文件和状态
├── kugou/                       # 经验证的酷狗数据库副本
├── decrypt-staging/             # um-react 回传暂存
├── deleted_songs_log.json
├── logs/
├── temp/
└── bin/                         # 运行时释放的 FFmpeg、ffprobe
```

对于仍使用 `%APPDATA%\BroadcastTool` 的旧版，程序会将数据迁移到当前程序旁的 `BctoolData`，原目录保留。单体 `data.json` 还会升级为拆分布局；迁移前的 `data.v1-backup.json` 按保留策略暂存七天。

迁移数据时复制整个 `BctoolData`。降级到旧版前，请退出 Beta，并恢复升级前备份的完整数据目录；旧版无法直接读取新版拆分布局。

## 安全与使用边界

- 仅提供 Windows 10/11 x64 构建。桌面模式依赖 WebView2，浏览器模式不要求安装额外运行时。
- 旧版管理接口没有完整的会话鉴权，能访问 `1743` 端口的设备可能调用歌单、设置、快照和文件操作。请在可信局域网内使用，不要将端口暴露到公网。
- 保留期、插件、酷狗数据库和更新等新增管理操作仅接受本机请求。
- 当前 Beta 未预置生产更新仓库和签名根，更新请从 GitHub Releases 下载。签名更新的配置方法见 [docs/SIGNED_UPDATES.md](docs/SIGNED_UPDATES.md)。

## 开发

```bash
git clone https://github.com/DGHH-Skywalker/BCTools.git
cd BCTools
npm ci
npm --prefix Web ci --legacy-peer-deps
corepack pnpm@10.15.1 --dir Web/um-react install --frozen-lockfile

npm run dev
npm run test
npm run type-check
```

完整 Windows 构建：

```bash
npm run build
```

环境要求、构建步骤和发布流程见 [DEVELOPER.md](DEVELOPER.md)；常见工具链问题见 [TOOLS.md](TOOLS.md)。

## 项目结构

| 路径 | 内容 |
| --- | --- |
| `Web/` | Vue 3、TypeScript、Vite 主界面 |
| `Web/um-react/` | 内嵌的 Unlock Music React 源码 |
| `GoServer/` | Go 后端、浏览器/桌面运行时、存储、插件和更新 |
| `GoServer/internal/contracts/` | OpenAPI 契约及生成的 Go 类型 |
| `installer/` | Inno Setup 安装脚本；旧 C++ 安装器仅作历史对照 |
| `scripts/` | 构建、契约生成和发布辅助脚本 |
| `docs/` | 当前运维文档与版本历史记录 |

接口说明见 [GoServer/docs/api.md](GoServer/docs/api.md)，运行时可读取 `/api/v1/openapi.json`。

## 许可证

项目代码以 [MIT](LICENSE) 发布。字体、图片、FFmpeg、LAME、Unlock Music 及其他第三方组件遵循各自许可证，不因仓库主许可证而改变。

请仅处理自己有权使用的音频文件。
