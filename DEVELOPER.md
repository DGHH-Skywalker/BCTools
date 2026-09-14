# DEVELOPER.md — 小播点歌工具

本文档面向**接手本仓库的开发者**：从 clone 到本地跑起来，再到打 release
产物，所有命令都在这里。

## 0. 项目是什么

「小播点歌工具」是揭阳一中广播站的一体化歌单管理工具，三个组成部分：

| 子项目 | 技术栈 | 角色 |
|------|------|------|
| `Web/` | Vue 3 + Vite + TypeScript + Naive UI | 主前端：歌单 / 点歌 / 导出 / 设置 / 数据迁移 |
| `Web/um-react/` | React + Vite（[Unlock Music](https://git.um-react.app/um/um-react) 源码直接放进仓库） | ncm / qmc / kgm 浏览器端解密器，嵌进主程序 |
| `GoServer/` | Go 1.22+（chi router、fyne systray、go:embed） | 单文件 exe：HTTP API + 文件转码 + 系统托盘 |
| `installer/` | C++ Win32 / GDI+（MinGW-w64 编译） | 安装 / 卸载程序，可选 |
| `scripts/` | Node（.cjs） | 构建辅助：um-react、注入 ffmpeg、清理等 |

构建产物：
- `dist/bctools.exe` — 单文件主程序（含 Vue 前端、um-react、ffmpeg/ffprobe）
- `dist/BCTools-Setup.exe` — Windows 安装包（可选，要 MinGW）

---

## 1. 环境要求

| 工具 | 最低版本 | 检查 | 说明 |
|------|--------|------|------|
| Git | 2.30+ | `git --version` | clone 用 |
| Node.js | 24+ | `node --version` | 前端 / 构建脚本 |
| Go | 1.22+ | `go version` | 后端（推荐 1.24+） |
| pnpm | 9+ | `pnpm --version` 或 `corepack pnpm --version` | um-react 依赖（**用 corepack 调即可，无需全局装**） |
| MinGW-w64 | 8+ | `g++ --version` | **可选**：仅在跑 `npm run build:installer` 时需要 |

无管理员权限也能跑：用 `corepack pnpm` 代替全局 pnpm；Go 装到用户目录也行。

---

## 2. Clone 与首次安装

```bash
git clone <repo-url> bctools
cd bctools

# 根目录 devDependencies（concurrently 等）— npm run dev / build 入口会自动跑，
# 也可以手动先装：
npm install

# 前端依赖
npm --prefix Web install --legacy-peer-deps

# um-react 依赖（用 corepack pnpm，无管理员权限也行）
corepack pnpm --version                    # 先确认 corepack 调得到
corepack pnpm --dir Web/um-react install --no-frozen-lockfile
```

> `npm run dev` / `npm run build` 都会在执行前自动检查根 `node_modules`；
> 若缺 `concurrently` 等会先跑一次 `npm install`。前端的 `Web/node_modules`
> 跟 um-react 的 `Web/um-react/node_modules` 不会被自动安装，需要手动跑上面的命令。

---

## 3. 开发

```bash
# 跑前端 + 后端（Vite 热更新 + go run 后端，并行）
npm run dev
```

行为：
- `Web/` 走 Vite dev server，端口 5173；自动把 `/api/*` 代理到 `:1743`
- `GoServer/` 走 `go run . --dev`，端口 1743；`--dev` 标志会顶替已运行的生产版后端
- 任一进程退出，另一个会被 concurrently `--kill-others-on-fail` 一起杀掉

单独跑子任务：

```bash
npm run dev:web         # 仅 vite
npm run dev:server      # 仅 go run --dev
```

---

## 4. 构建

### 4.1 完整构建

```bash
npm run build
```

构建会先并行准备 Vue、um-react 和 FFmpeg，再依次编译 Go 主程序与安装包：

| 步骤 | 脚本 | 说明 |
|------|------|------|
| 并行 1 | `build:web-and-copy` | `Web/`：类型检查、Vite 构建、复制到 `GoServer/embed/dist` |
| 并行 2 | `build:um-react` | 构建、注入桥接脚本并复制到 `GoServer/embed/um-react/` |
| 并行 3 | `build:inject-ffmpeg` | 将精简版 ffmpeg/ffprobe 放入 Go 的嵌入目录 |
| 收敛 1 | `build:server` | 编译 `dist/bctools.exe`（GUI 应用，启动不闪黑框） |
| 收敛 2 | `build:installer` | 编译 `dist/BCTools-Setup.exe`；缺少 MinGW 时会跳过并提示 |

产物：
- `dist/bctools.exe` ~45 MB（自带前端 + um-react + ffmpeg/ffprobe，GUI 应用）
- `dist/BCTools-Setup.exe` ~57 MB（仅在装了 MinGW 时生成）

> 5.6.0 起**不再生成 `bctool_dev.exe`**——所有 dev 模式都走 `npm run dev`（go run）。Release 包只需要主 exe 和 setup。

### 4.2 启动不闪黑框
所有产物都用 `-H windowsgui`（Go）+ `-mwindows`（g++）编译：直接双击 `dist/bctools.exe` 不会出现控制台窗口。**别**用 `go run` 启动生产 exe——dev 模式本来就要看日志。

### 4.3 安装程序（必选，5.6.0 起）

```bash
# 5.6.0 起 npm run build 默认就会跑安装程序构建；
# 没装 MinGW-w64 也会跳过并打 warning，不影响 bctools.exe 主产物。
npm run build               # 已经包含 build:installer
npm run build:installer     # 想单独跑也行
```

需要 MinGW-w64 提供 `g++` 和 `make`（Windows 上常用 `mingw32-make`）。
脚本会自动从常见路径找 MinGW，没装就跳过。

### 4.4 单独跑某一步

```bash
npm run build:web           # 仅前端
npm run build:um-react      # 仅 um-react（含复制到 embed）
npm run build:copy          # 仅把 Web/dist 复制到 GoServer/embed/dist
npm run build:inject-ffmpeg # 仅复制 ffmpeg/ffprobe
npm run build:server-only   # 仅 go build（输出 dist/bctools.exe）
npm run build:installer     # 仅安装程序
npm run build:clean         # 清理所有构建产物
```

### 4.5 清理

```bash
npm run build:clean
# 删除：Web/dist, GoServer/embed/dist, GoServer/embed/um-react,
#       GoServer/internal/binembed/bin, dist/, installer/obj, installer/setup.exe
```

### 4.6 数据迁移（5.6.0 完整版）

设置页 → 「数据迁移」(`/settings/migration`)。两种导出 + 两种导入，路径输入框可粘贴。

**导出：**
- **小清单（不带歌）**：下载单个 JSON，含所有元数据（歌名、时段、栏目映射）。包小可微信传；歌曲需在新机器另行导入。
- **完整搬家（推荐）**：在选定文件夹下新建 `xxxx年xx月xx日广播站点歌工具数据备份N/`，内含 data.json + 所有歌曲 mp3 + 快照 + 删除日志。换电脑时首选。

**导入（两种方式，二选一）：**
- **从文件夹装回（推荐）**：选完整的备份目录；歌曲文件会一起回来。
- **从 JSON 文件装回**：选之前下载的清单 JSON；只回歌名与设置，不带音频。

**模式：**
- `merge`（默认）：追加新歌，时段按 ID 合并（自定义的不会被覆盖）
- `replace`：先清空本机 dorm / broadcast 歌单与时段，再整体装入

后端：`POST /api/migration/export`、`POST /api/migration/import`。

---

## 5. 测试 / 类型检查

```bash
npm run type-check     # 前端 vue-tsc + 后端 go vet
npm run test           # 前端 vitest + 后端 go test
npm run lint           # 前端 eslint
```

单独跑：

```bash
npm --prefix Web run type-check
npm --prefix Web run test

go -C GoServer test ./...
go -C GoServer vet ./...
```

---

## 6. 目录速查

```
bctools/
├── Web/
│   ├── src/
│   │   ├── views/             每个路由一个页面
│   │   ├── components/       可复用组件
│   │   ├── stores/           Pinia 状态
│   │   ├── composables/      与 UI 解耦的逻辑
│   │   ├── api/              axios 客户端 + 按领域拆分
│   │   ├── utils/            无状态工具（datetime / persist / playlistTable）
│   │   ├── constants/        Design Token（colors.ts）
│   │   ├── i18n/             i18n 实现（默认 zh-CN）
│   │   └── locales/          zh-CN.json / en.json
│   ├── um-react/              Unlock Music 源码（已从子模块改为内嵌）
│   ├── scripts/               subset-fonts.cjs / um-react-bridge.js
│   └── package.json
├── GoServer/
│   ├── main.go                主程序入口（systray + 关闭流程）
│   ├── routes/                chi 路由表
│   ├── handlers/              HTTP handler（每个领域一个文件）
│   ├── services/              业务逻辑（organize / import / migration / settings）
│   ├── store/                 领域数据访问（song / setting / snapshot / deletedlog）
│   ├── internal/
│   │   ├── repo/              data.json 原子读写
│   │   ├── binembed/          ffmpeg/ffprobe 嵌入与释放
│   │   └── version/           单一版本号来源
│   ├── converter/             音频转码（FFmpeg 包装 / 合并 / 静音占位）
│   ├── embed/
│   │   ├── dist/              嵌入的 Vue 前端产物（go:embed）
│   │   └── um-react/          嵌入的 um-react 产物（go:embed）
│   ├── ffmpeg_bctools_mini/   项目内置精简版 ffmpeg/ffprobe 源
│   ├── tray/                  Windows systray 封装（含 DPI 感知）
│   ├── assets/                图标等静态资源
│   └── cmd/devlauncher/       开发模式启动器
├── installer/
│   ├── src/                   C++ 源码（Win32 / GDI+）
│   ├── res/                   图标 / 字体 / 待打包 exe（构建期填入）
│   └── Makefile
├── scripts/                   Node 构建辅助（build-web / build-um-react / inject-ffmpeg / clean-all）
├── docs/                      设计与历史交接文档
├── package.json               根 npm 入口（dev / build 链）
├── DEVELOPER.md               ← 你正在看的文档
├── CLAUDE.md                  Claude/Agent 工作约束（项目内）
└── package.json               唯一的开发与构建入口
```

---

## 7. 关键约束（修改前必读）

这些是 5.6.0 沉淀下来的「改了会炸」的不成文规定，新人务必读完：

- **音频转码**：`converter/process.go` 的 `toMP3()` 已实现「已是 MP3 就跳过重编码」。一首歌重编码 ~4.8s，搬运 ~1ms，且能避免二次有损。**不要**改回无条件 `ConvertToMP3`。
- **um-react 导入**：走 `POST /api/decrypt/stage/:id/import` **就地入库**到 `%APPDATA%\BroadcastTool\songs`，不要改回「下载回浏览器再上传」——一首 12MB 的歌要少跑 ~200ms。
- **换卡导出（organize）**：编号锚定「时段位置」不是歌曲（见 `Web/src/utils/exportEntries.ts`）。空时段必须生成静音占位，否则整天空着会让后面曲序整体前移。同一时段多首歌合并成一个 MP3：`converter/merge.go` 用字节拼接 + 剥 ID3 标签，**不要**改成 ffmpeg concat（精简版 ffmpeg 没那个 demuxer）。
- **进程退出**：托盘 / Ctrl-C / 前端 `POST /api/shutdown` 三条路径都汇聚到 `main.go` 里同一个 `sync.Once` 保护的 shutdown 函数。`tray.Run` 占用主 goroutine（systray 要求 LockOSThread），**不要**把它挪到子 goroutine。
- **segmentit**：词典约 1.25 MB（gzip），动态 `import()`，未就绪时 `segmentChineseTitle` 原样返回标题。**不要**改回静态 import。
- **托盘 DPI**：Win32 弹菜单默认是 Win95 API 不跟随系统缩放。`tray/dpi_windows.go` 在 init() 里声明 Per-Monitor V2 DPI awareness，右键菜单在 2K/4K 屏上才不糊。**不要**删这个文件。
- **文件夹选择器**：`GoServer/handlers/selectdir_windows.go` 用 `IFileOpenDialog + FOS_PICKFOLDERS`（Vista+），不跟随 `SHBrowseForFolderW`（Win95）。**不要**回退。

---

## 8. 常见问题

**Q：`go run . -- --dev` 在 npm run dev 下不生效？**
A：`go run` 接收参数用 `--` 分隔。Windows Git Bash + npm 链里这层转义有时候会丢。可以用环境变量：
在 PowerShell 跑 `$env:BCTOOLS_DEV=1; npm run dev:server` 临时替代（需要在 main.go 加上 env 路径）。
当前默认 `--dev` 是通过的；遇到再排查。

**Q：精简版 ffmpeg 在哪？**
A：`GoServer/ffmpeg_bctools_mini/`，由单独的 `build.sh` 生成。仓库里只放编译好的 `dist/ffmpeg.exe` / `ffprobe.exe`。本仓库不维护精简版 ffmpeg 的源码（那是另一个项目）。缺失时 `npm run build` 会在 `build:inject-ffmpeg` 阶段中止，避免运行时回退到系统完整版。

**Q：构建时 Go 报 `undefined: services`？**
A：检查 `GoServer/main.go` 顶部 imports 是否含 `broadcast-tool/services`。5.6.0 后已加。

**Q：Vue 端 npm run test 出现 `No match found for location with path ""`？**
A：单测里 `<router-link>` 没有显式 path 时会触发该警告。已在 `DormManage.test.ts` 加了 stub router，不影响功能。

---

## 9. 发版流程

```bash
# 1. 改版本号（仅一处即可，Go ldflags 会覆盖）
#    GoServer/internal/version/version.go
#    Web/package.json
#    package.json

# 2. 构建
npm run build
# 可选：npm run build:installer（需要 MinGW）

# 3. 产物
ls dist/
#   bctools.exe            单文件主程序
#   bctool_dev.exe         开发模式启动器
#   BCTools-Setup.exe      安装包（如有）

# 4. 打 tag + 推 GitHub
git tag v5.6.0.0
git push origin v5.6.0.0
# 在 GitHub Releases 里把 dist/ 下的 exe 作为 release assets 上传
```

精简版 ffmpeg 缺失时 `npm run build` 会显式报错中止——这是故意的：避免运行时悄悄回退到系统完整版 ffmpeg（可能引入不必要的编解码器体积 / 协议支持差异）。
