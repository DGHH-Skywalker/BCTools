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
| 构建脚本 | `build.py` |
| 版本唯一源 | `GoServer/internal/version/version.go` |
| 数据持久化 | `GoServer/internal/repo/repo.go` |
| 安装程序源码 | `installer/`（C++ Win32/GDI+） |

## 目录速查

```
BCTools/
├── GoServer/    Go 后端（handlers / services / store / internal）
├── Web/         Vue 3 前端（views / components / composables / stores）
├── installer/   C++ Win32/GDI+ 原生安装/卸载程序
├── dist/        构建产物输出目录
├── docs/        设计/交接文档
└── build.py     一键构建脚本
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
- **进程退出**：托盘、Ctrl-C、前端 `POST /api/shutdown` 三条路径都汇聚到 `main.go` 里同一个 `sync.Once` 保护的 shutdown 函数。`tray.Run` 占用主 goroutine（systray 要求 LockOSThread），不要把它挪到子 goroutine。

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

## 测试

```bash
# 前端
cd Web
npm run type-check
npm run test

# 后端
cd GoServer
go test ./...
```

## 已知限制

- 无 session/JWT 鉴权，API 对局域网开放。
- `data.json` 全量读写，当前场景下足够；数据量极大时需考虑迁移。
- 前端 `en.json` 保留结构但未启用，默认 locale 为 `zh-CN`。

## 构建产物

运行 `python build.py` 后产物输出到 `dist/`：

- `bctools.exe`
- `bctool_dev.exe`
- `BCTools-Setup.exe`（需要 MinGW-w64）
