# CLAUDE.md — 小播点歌工具

本文件面向 Claude / Agent，说明项目结构、关键入口、常见修改方式与约束。

## 快速导航

| 目标 | 入口 |
|------|------|
| 后端主程序 | `GoServer/main.go` |
| 路由注册 | `GoServer/routes/routes.go` |
| 托盘控件 | `GoServer/tray/`（Windows systray，左键开页/右键菜单） |
| 进程生命周期 | `GoServer/handlers/lifecycle.go`（health / watch / shutdown） |
| 前端入口 | `Web/src/main.ts` |
| 前端路由 | `Web/src/router/index.ts` |
| 整库一键 dev / build | 根 `package.json`（`npm run dev` / `npm run build`） |
| 完整发布构建（含安装程序） | `npm run build` |
| 版本唯一源 | `GoServer/internal/version/version.go` |
| 数据持久化 | `GoServer/internal/repo/repo.go` |
| 安装程序源码 | `installer/`（C++ Win32/GDI+） |

## 目录速查

```
BCTools/
├── GoServer/           Go 后端（handlers / services / store / internal）
├── Web/                Vue 3 前端（views / components / composables / stores）
│   └── um-react/       Unlock Music 源码（5.6.0 起直接放进仓库，原是 git submodule）
├── installer/          C++ Win32/GDI+ 原生安装/卸载程序
├── scripts/            跨子项目的 Node 工具脚本（build-web / build-um-react / inject-ffmpeg / clean-all）
├── dist/               构建产物输出目录（不进入 git；用 `npm run build` 重新生成）
├── package.json        整库 dev / build 入口
├── DEVELOPER.md        开发者上手文档（环境、构建、约束、常见问题）
├── CLAUDE.md           本文件：给 Agent 的工作约束
```

## 架构约定

### 后端分层

```
main.go
  ↓ handlers   HTTP 请求解析、参数校验、响应封装
  ↓ services   业务逻辑编排（ Organize / Import / SettingsUpdate ）
  ↓ store      领域数据访问（songstore / settingstore / snapshotstore / deletedlogstore）
  ↓ internal/repo  data.json 原子写入、并发控制、备份
```

- **新增 API**：在 `GoServer/handlers/` 写 handler，在 `GoServer/routes/routes.go` 注册。
- **复杂业务**：优先抽到 `GoServer/services/`，保持 handler 简短。
- **数据访问**：通过 `store/*` 包操作；`internal/repo` 负责底层 JSON 持久化。
- **系统交互**：执行外部命令（隐藏控制台窗口）、打开浏览器、查询 Windows 版本，统一用 `GoServer/platform/`。
- **音频转码**：`converter/process.go` 的 `toMP3()` 会先嗅探格式（`converter/sniff.go`），解密结果已是 MP3 时直接搬运，不调 ffmpeg。重编码一首 4 分钟的歌约 4.8s，搬运 ~1ms，且能避免二次有损转码。不要改回无条件 `ConvertToMP3`。
- **um-react 导入**：走 `POST /api/decrypt/stage/:id/import` 就地入库，**不要**改回「下载回浏览器再上传」——那样一首 12MB 的歌要在本机跑三趟共 35.6MB（~263ms vs ~67ms）。已是 MP3 时用 rename，并优先用暂存 meta 的文件名当标题以跳过 ffprobe（一次约 88ms）。
- **换卡导出**：编号锚定「时段位置」不是歌曲（见 `Web/src/utils/exportEntries.ts`），空时段生成静音占位，否则整天空着会让后面曲序整体前移。同一时段多首歌合并成一个 MP3（`converter/merge.go`）——内置精简版 ffmpeg **没有** concat demuxer/滤镜/PCM muxer，所以用字节拼接 + 剥除后续文件的 ID3 标签，不要改成 ffmpeg concat。
- **托盘**：5.6.0 起**只保留右键菜单**（左键单击不再打开浏览器），由 `tray/dpi_windows.go` 在 init() 声明 Per-Monitor V2 DPI awareness 让菜单不糊在高分屏。`Config.OnOpen` 字段已删除。
- **进程退出**：Ctrl-C、前端 `POST /api/shutdown` 两条路径都汇聚到 `main.go` 里同一个 `sync.Once` 保护的 shutdown 函数（5.6.0 起托盘右键直接调 OnExit）。`tray.Run` 占用主 goroutine（systray 要求 LockOSThread），不要把它挪到子 goroutine。

### 前端分层

```
Web/src/
  views/        页面（每个路由一个）
  components/   可复用组件
  stores/       Pinia 状态
  composables/  与 UI 解耦的逻辑
  api/          axios 客户端 + 按领域拆分的 API 方法（零散端点收在 misc.ts）
  utils/        无状态工具（datetime / persist / playlistTable）
  styles/       CSS 变量、全局样式、导出样式
  constants/    Design Token（颜色、间距、字体、布局）
```

- **新增页面**：`views/` 创建组件，`router/index.ts` 加路由。
- **样式**：优先使用 `variables.css` 变量和 `constants/`；避免在业务逻辑中写死颜色/像素。
- **日期时间**：统一从 `utils/datetime.ts` 取 `dayjs`（isoWeek 插件在那里注册一次），不要在各文件重复 `dayjs.extend`。
- **localStorage**：统一走 `utils/persist.ts`，key 带 `bctools.` 前缀并保留旧 key 兼容读取。
- **中文分词**：`segmentit` 词典约 1.25 MB（gzip），已改为动态 `import()`。需要分词前先 `await ensureSegmenter()`；未就绪时 `segmentChineseTitle` 原样返回标题。不要改回静态 import。
- **导出图片**：样式定义在 `Web/src/styles/export.css`，业务逻辑在 `Web/src/composables/useExportImage.ts`。

## 关键配置

- 监听端口：`1743`（`GoServer/main.go`）
- 数据目录：`%APPDATA%\BroadcastTool\`（`GoServer/paths/paths.go`）
- 默认主题色：`#0086C3`（`Web/src/constants/colors.ts` / `Web/src/styles/variables.css`）
- 路由模式：`createWebHashHistory`（不依赖服务端路由）

## 常见修改入口

### 修改主题色/字体/间距

1. `Web/src/constants/colors.ts` / `spacing.ts` / `typography.ts`
2. `Web/src/styles/variables.css`
3. `Web/src/composables/useTheme.ts`（主题色动态计算）

### 修改导出图片样式

1. `Web/src/styles/export.css`
2. `Web/src/composables/useExportImage.ts`（不要改业务结构）

### 修改宿舍/播音歌单展示

- 宿舍：`Web/src/views/DormManage.vue`、`Web/src/components/dorm/DormGrid.vue`
- 播音：`Web/src/views/BroadcastEdit.vue`、`Web/src/components/broadcast/BroadcastGrid.vue`

### 修改导入/导出行为

- 后端：
  - 导入：`GoServer/services/import_service.go`
  - 导出：`GoServer/handlers/songs.go` `HandleExport`
- 前端：
  - 导入：`Web/src/views/SongImport.vue`、`Web/src/composables/useFileProcessor.ts`
  - 导出：`Web/src/views/Export.vue`、`Web/src/composables/useExportImage.ts`

## 整库 dev / build

仓库根 `package.json` 提供一条龙入口，覆盖前端 + Go 后端。

```bash
# 一次性安装根级 devDependencies（concurrently、rimraf）
npm install

# 开发：vite 热更新 + go run 后端（--dev 会顶替已运行的后端）并行
npm run dev

# 构建：并行准备 Vue、um-react、FFmpeg，再产出主程序与安装包
npm run build

# 单独运行子步骤
npm run dev:web              # 仅 vite
npm run dev:server           # 仅 go run
npm run build:web            # 仅 vite build + 字体精简
npm run build:copy           # 复制 Web/dist → GoServer/embed/dist
npm run build:um-react       # 仅 um-react 构建与嵌入
npm run build:server         # 仅 go build
npm run build:installer      # 仅安装程序
npm run build:clean          # 清理所有构建产物
```

## 测试

```bash
# 整库
npm run type-check           # 前端 vue-tsc + 后端 go vet
npm run test                 # 前端 vitest + 后端 go test

# 单独
cd Web
npm run type-check
npm run test

cd GoServer
go test ./...
```

## 已知限制

- 无 session/JWT 鉴权，API 对局域网开放。
- `data.json` 全量读写，当前场景下足够；数据量极大时需考虑迁移。
- 前端 `en.json` 保留结构但未启用，默认 locale 为 `zh-CN`。

## 构建产物

运行 `npm run build` 后产物输出到 `dist/`：

- `bctools.exe`
- `BCTools-Setup.exe`（需要 MinGW-w64）
