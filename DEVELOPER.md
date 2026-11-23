# BCTools 开发指南

本文面向开发和发布维护者。日常使用与数据备份见 [README.md](README.md)，构建环境排障见 [TOOLS.md](TOOLS.md)。

## 1. 组成与运行方式

| 路径 | 技术与职责 |
| --- | --- |
| `Web/` | Vue 3、TypeScript、Vite、Naive UI；主界面与浏览器端交互 |
| `Web/um-react/` | React、Vite；内嵌 Unlock Music，源码已直接纳入仓库 |
| `GoServer/` | Go、chi、Wails v3；HTTP API、音频处理、存储、桌面窗口、托盘、插件和更新 |
| `installer/` | Inno Setup 6；当前安装与卸载流程 |
| `scripts/` | Node.js 构建、契约生成、更新打包和清理脚本 |

无参数启动时，程序监听 `0.0.0.0:1743`，打开默认浏览器并显示系统托盘。`bctools.exe desktop` 使用 Wails v3 和 WebView2 打开桌面窗口，同时尽量保留 LAN 监听；端口被占用不妨碍桌面窗口使用本机功能。

两种模式共享同一后端和数据目录。`GoServer/launcher/` 负责实例占用、状态查询、浏览器与桌面模式交接；不要让两个后端同时写同一个 `BctoolData`。

## 2. 环境准备

| 工具 | 版本 | 用途 |
| --- | --- | --- |
| Git | 当前稳定版 | 获取源码；um-react 的依赖安装脚本也会调用 |
| Node.js | 24，CI 使用 24.7.0 | 根脚本与 Vue 构建 |
| Go | 1.26.11 | 后端、测试与版本资源 |
| pnpm | 10.15.1，通过 Corepack 使用 | um-react 依赖与构建 |
| Inno Setup | 6 | Windows 安装包 |
| MSYS2 UCRT64 | 当前版本 | 仅在重新构建精简 FFmpeg 时需要 |

获取源码并安装依赖：

```bash
git clone https://github.com/DGHH-Skywalker/BCTools.git
cd BCTools
npm ci
npm --prefix Web ci --legacy-peer-deps
corepack pnpm@10.15.1 --dir Web/um-react install --frozen-lockfile
```

um-react 已内嵌源码，不需要初始化 Git submodule。根目录的 `predev`、`prebuild` 会检查根级依赖，Vue 和 um-react 的依赖仍需按上面命令安装。

## 3. 本地开发

```bash
npm run dev
```

该命令并行启动：

- Vite：`http://localhost:5173`，将 API 代理到 `1743`；
- Go：`go -C GoServer run . --dev`，必要时接替已有开发后端。

也可分别运行：

```bash
npm run dev:web
npm run dev:server
```

`go run` 的开发数据默认写入 `GoServer/BctoolData`，不会使用发布版程序旁的目录。开发前如需保留真实数据，先备份。

## 4. 检查与测试

```bash
npm run type-check
npm run test
npm run lint
```

这些入口分别覆盖 Vue 类型检查与 Go vet、Vue 与 Go 测试、Vue ESLint。um-react 使用独立工具链：

```bash
corepack pnpm@10.15.1 --dir Web/um-react exec tsc -p tsconfig.prod.json --noEmit
corepack pnpm@10.15.1 --dir Web/um-react test
```

修改 API 契约后重新生成类型，并确认生成文件没有漏提交：

```bash
npm run generate:api
git diff -- GoServer/internal/contracts/types.gen.go Web/src/api/generated/contracts.ts
```

CI 还会执行生成一致性检查、Go 测试、两个前端的检查与测试。

## 5. 构建

先准备 `GoServer/ffmpeg_bctools_mini/dist/ffmpeg.exe` 和 `ffprobe.exe`。重新构建它们的方法见 [FFmpeg 构建说明](GoServer/ffmpeg_bctools_mini/README.md)；CI 会独立编译并缓存。

完整构建：

```bash
npm run build
```

流程如下：

1. 并行构建 Vue、um-react，并准备内嵌 FFmpeg；
2. 将前端产物复制到 Go 的嵌入目录；
3. 生成 Windows 版本资源并编译 `dist/bctools.exe`；
4. 调用 Inno Setup 生成 `dist/BCTools-Setup.exe`。

常用子步骤：

```bash
npm run build:web
npm run build:copy
npm run build:um-react
npm run build:inject-ffmpeg
npm run build:server-only
npm run build:installer
npm run build:clean
```

`ISCC.exe` 不在常见安装路径时，通过 `ISCC_PATH` 指定。主程序使用 `-H windowsgui`，正常启动不显示控制台窗口。

GitHub Actions 还会生成以下发布文件，本地 `npm run build` 不负责全部生成：

- `BCTools-UpdateTargets.zip`：由 `scripts/package-update-feed.ps1` 生成；
- `FFmpeg-Sources.tar.gz`：FFmpeg 构建开启源码打包后生成；
- `SHA256SUMS.txt`：发布工作流统一计算。

## 6. 当前架构

### 请求链路

```text
Browser / WebView2
        ↓
server.WithUpdateAssets
        ↓
chi router
  /api       兼容原有接口
  /api/v1    同一组核心接口 + OpenAPI、保留期、插件、酷狗和签名更新
        ↓
handlers → services / store → internal/repo
```

- `GoServer/main.go`：启动、实例交接、依赖组装和退出。
- `GoServer/desktop_windows.go`：WebView2 窗口、桌面保存对话框和桌面专用接口。
- `GoServer/routes/routes.go`：路由及本机请求边界。
- `GoServer/server/`：内嵌前端、um-react 和版本化更新资源的静态服务。
- `GoServer/handlers/`：请求解析与响应。
- `GoServer/services/`：导入、周音频库、旧版迁移、设置和更新日志。
- `GoServer/store/`：歌曲、设置、快照与删除日志的数据访问。

新增扩展接口时，同时核对 OpenAPI 契约、路由权限和前端生成类型。保留期、插件管理、酷狗数据库选择、更新检查与应用等操作有本机请求限制，不要为了联调直接移除。

### 存储布局

当前存储版本为 2：

- `data.json` 保存存储头、设置、下一歌曲 ID 和周目录索引；
- `songs-without-audio.json` 保存没有对应音频的歌曲；
- `MusicFiles/<YYYY年第N周>/songs.json` 保存有音频歌曲及周文件索引；
- `.storage-transaction.json` 记录文件事务，启动时重放未完成操作；
- `recycle/<id>/` 保存可恢复的清理批次。

`internal/repo` 在内存中汇总各领域数据，写入时拆分并提交文件事务。不要绕过它直接改写多个存储文件。歌曲 ID 由仓储层分配，批量导入和普通新增都必须遵守同一规则。

覆盖导入前先保存与本次替换对应的快照；备份失败时取消导入。恢复快照和回收记录时，保留文件名与目录边界校验。

默认保留当前及前三个 ISO 周的音频、过去两年的歌曲信息和未来排期。设置页可调整保留期；清理先进入七天回收区。

### 音频处理

- 已是 MP3 的输入直接搬运，避免二次有损转码；其他格式再调用内嵌 FFmpeg。
- NCM 由 Unlock Music 解码器处理；KGG、KGM、KGMA、VPR 通过酷狗数据库取钥。
- 酷狗数据库经验证后复制到 `BctoolData/kugou`，源文件暂时缺失时可沿用上一次可用副本。
- um-react 回传使用 `POST /api/v1/decrypt/stage/{id}/import` 就地入库，不再下载回浏览器后重复上传。
- SD 卡编号以时段位置为锚点。空时段保留编号并生成 17.43 秒静音占位。
- 同一时段多首歌在导出时合并，歌库内仍分别保存。合并采用 MP3 帧拼接并处理 ID3 标签；精简 FFmpeg 不包含完整的 concat 组件。
- 合并音频的单曲试听按原始文件区间读取，保留 Range 与 Seek 处理。

### 插件与更新

插件 ZIP 包含 `manifest.json` 和可选的 `plugin.wasm`。当前权限为 `ui.contribute`、`ui.layout`、`commands.run`、`playlist.read`；WASM 有内存和执行时间限制。插件接口只暴露约定能力，不能直接访问宿主存储。

更新管理器基于 TUF 校验元数据和目标文件，支持资源、网页、插件和 Windows 核心程序。网页更新需确认生效，并保留上一版本用于回滚；核心更新区分便携版与已安装版。发布配置见 [SIGNED_UPDATES.md](docs/SIGNED_UPDATES.md)。

### 安装与数据

当前安装入口是 `installer/BCTools.iss`。`installer/src/` 和旧 Makefile 保留作历史对照，不参与根构建或 CI。

安装器只拥有程序文件和快捷方式，不拥有 `BctoolData`。升级、重装和普通卸载保留数据；交互卸载可由用户选择删除，静默卸载只有显式传入 `/REMOVEUSERDATA` 才删除。

## 7. 目录速查

```text
BCTools/
├── Web/
│   ├── src/                     Vue 页面、组件、状态、API 与工具
│   └── um-react/                Unlock Music React 源码
├── GoServer/
│   ├── main.go                  进程入口与依赖组装
│   ├── routes/                  路由与本机访问控制
│   ├── handlers/                HTTP handler
│   ├── services/                业务编排与迁移
│   ├── store/                   领域数据访问
│   ├── internal/repo/           拆分存储、事务与保留期
│   ├── internal/contracts/      OpenAPI 与生成类型
│   ├── internal/plugins/        插件包、权限和 WASM 运行时
│   ├── internal/updates/        TUF 更新与核心程序交接
│   ├── internal/kugoudb/        酷狗数据库发现、验证与副本
│   ├── converter/               解密、转码、合并与静音
│   ├── launcher/                单实例与模式交接
│   ├── server/                  前端和更新资源服务
│   └── ffmpeg_bctools_mini/     精简 FFmpeg 构建
├── installer/                   Inno Setup 与旧安装器源码
├── scripts/                     构建、测试和发布脚本
└── docs/                        运维文档与历史版本记录
```

## 8. 发布

发布前同步以下版本信息：

- 根 `package.json` 与锁文件；
- `Web/package.json` 与锁文件；
- `GoServer/internal/version/version.go`；
- `GoServer/versioninfo.json` 中的 Windows 数字文件版本和产品版本。

应用显示版本与 Windows 四段数字文件版本是两套字段，发布工作流都会检查。

1. 添加与根 `package.json` 对应的 `docs/releases/v<version>.md`；
2. 推送 `v<version>` 标签；
3. Release 工作流校验标签，复用 Build 工作流，核对校验值并发布附件；
4. 版本号包含预发行标识时，GitHub Release 自动标记为 prerelease。

发布签名更新还需配置更新仓库和信任根，并将更新目标交给 TUF-on-CI 签名。不要仅上传 EXE 就认为客户端已能收到更新。

## 9. 文档维护

README、开发指南和运维文档描述当前版本。`docs/releases/` 与 `.claude/plans/` 是历史记录，保留当时的版本语境。修改架构、命令、数据布局或发布流程时，应同步检查当前文档和中英文界面文案。
