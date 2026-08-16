# 托盘控件 + 项目结构/性能优化

## 已完成

`bctools.exe` (PID 23484) 已终止，端口 1743 释放。

## 目标

1. 任务栏托盘控件：左键打开前端网页，右键菜单可打开界面或退出后端（退出前尝试关闭浏览器页）。
2. 收敛零散文件到中等大小的聚合文件。
3. 修复已确认的性能问题。

## 调研结论（已验证，非推测）

| 结论 | 验证方式 |
|------|---------|
| `fyne.io/systray@v1.12.2` 可在 `CGO_ENABLED=0` + `GOOS=windows` + `-H windowsgui` 下纯 Go 编译 | 在 `/tmp/systray-spike` 实际编译并运行通过 |
| systray 与 `http.Server` 可共存；`Quit()` 能让 `Run()` 返回，`onExit` 正常执行 | 同一 spike 实际运行，输出 `onExit ran` / `Run() returned` |
| `SetOnTapped`（左键）/ `SetOnSecondaryTapped`（右键）为 v1.12.2 公开 API；右键不设回调时默认弹菜单 | 读 `systray.go:162-167`、`systray_windows.go:1131-1147` |
| systray 在 `init()` 里 `runtime.LockOSThread()`，`Run()` 必须在主 goroutine | 读 `systray.go:36-38` |
| 托盘图标已有现成资源 `GoServer/assets/icon-sys.ico`（58 KB） | 文件存在，且 Windows 下 `SetIcon` 要求 .ico |
| 前端有个 **3.6 MB / gzip 1.28 MB** 的产物块，17.9% 是中文字符，含单个 2.63 M 字符的字符串字面量 —— 即 segmentit 分词词典 | `node` 探测 chunk 的 CJK 密度与最大字面量 |
| 该词典**不在首屏**，是 `/export` 与 `/organize` 的路由懒加载块（`index.html` 只预载 `index-*.js`） | 读 `dist/index.html` 与 chunk 互相引用关系 |
| 词典在 `utils/playlistTable.ts:9` 模块顶层 `useDefault(new Segment())` 无条件实例化 | 读源码 |
| `api/snapshots.ts`、`api/update.ts` 零引用（死代码）；`components/ui/WarningText.vue` 零引用 | 全仓 grep |
| 后端基线：`go test ./...` 全绿；前端基线：3 个文件 / 9 个测试全绿 | 实际运行 |

## 实施计划

### A. 托盘控件（后端）

新建 `GoServer/tray/tray.go`（聚合文件，含 Windows 与非 Windows 两份实现）：

- `tray_windows.go`：`Run(cfg Config)` 封装 systray，图标用 `//go:embed icon-sys.ico`（从 assets 复制一份进包，避免跨包 embed 限制）。
  - 左键 → `SetOnTapped` → 打开 `http://localhost:<port>/`
  - 右键 → 默认弹菜单（不设 `SetOnSecondaryTapped`，让 systray 自己弹）
  - 菜单：`打开界面` / 分隔符 / `退出程序`
- `tray_other.go`：非 Windows 空实现，保持 `go build` 跨平台可过。

`main.go` 生命周期改造（这是**唯一有风险的改动**，因为 systray 必须占用主 goroutine）：

```
现在:  runBackend() 在主 goroutine 阻塞等 <-quit
改为:  runBackend() 启动 HTTP 后返回一个 shutdown 函数
       主 goroutine 交给 tray.Run()
       托盘「退出」→ 关浏览器页 → shutdown() → systray.Quit()
       SIGINT/SIGTERM 仍然有效（走同一条 shutdown 路径，用 sync.Once 防重复）
```

「退出前关闭浏览器页」按你的选择实现：新增 `POST /api/shutdown`（仅 localhost）；前端 `AppHeartbeat` 已有 30s 轮询，改为在收到关闭信号时调用 `window.close()`。
说明：浏览器普遍拦截脚本关闭非脚本打开的标签页，因此这是 **best-effort**——关不掉时退出流程照常继续，不会卡住。

### B. 文件收敛（保守，不动分层）

| 动作 | 前 | 后 |
|------|---|---|
| 删死代码 | `api/snapshots.ts`、`api/update.ts`、`components/ui/WarningText.vue` | 删除 |
| 合并微型 api 模块 | `api/{network,settings,system}.ts` 各 1 个函数 | 并入 `api/misc.ts` |
| 合并平台小包 | `browser/`、`runner/`(4 文件共 60 行) | 并入 `platform/`，`runner.Command` → `platform.Command` |
| 合并 UI 微组件 | `components/ui/PageContainer.vue`、`common/SettingsFab.vue` | 归入 `components/ui/`（SettingsFab 移入） |
| 统一时间工具 | 12 处各自 `dayjs.extend(isoWeek)` | 新建 `utils/datetime.ts` 单点 extend + 导出周工具 |

`api/client.ts` 保持独立（是 axios 实例，不是零散函数）。`constants/` 已有 `index.ts` 桶文件，不动。

### C. 性能修复（全部可测量）

1. **segmentit 懒加载**（最大项）：`playlistTable.ts` 的 `new Segment()` 从模块顶层改为首次调用 `segmentChineseTitle` 时惰性构造。
   - 效果：`/export`、`/organize` 首次进入不再同步解析 2.6 M 字符词典；仅在真正导出图片时付这个代价。
   - 注意：`segmentChineseTitle` 目前是同步函数，且被 `buildDormTableHTML` → `DormPlaylistTable` 的 `computed` 同步调用。因此**只做惰性实例化（仍同步）**，不改成 async，避免把渲染路径改成异步引发连锁改动。
2. **backend 减少重复遍历**：`GetSongsByType` 先 `cloneSongsWithWeekday` 全量克隆再按 dates 过滤 —— 改为先过滤再克隆，避免为丢弃的歌计算 weekday。
3. `CalcWeekday` 每次调用都新建 `weekdays` slice → 提为包级 `var`。

以上 3 项都不改变外部行为，靠现有测试兜底。

### D. 状态/配置标准化（轻量）

- `stores/export.ts` 的 localStorage 读写 + `i18n` 的 `locale` key 散落 → 集中到 `utils/persist.ts`，统一 key 前缀 `bctools.` 并保留旧 key 迁移读取（否则用户已存的导出设置会丢）。
- 不合并 Pinia store、不重排 views/components 目录（按你选的保守范围）。

## 验证

1. `cd GoServer && go vet ./... && go test ./...`（须与基线一样全绿）
2. `cd Web && npm run type-check && npm run test`（须仍是 9 个测试全绿）
3. `python build.py` 完整构建
4. 实测：产物 chunk 大小对比；启动新 exe 确认托盘图标出现、左键开页、右键菜单两项可用、退出后端后端口 1743 释放

## 不做

- 不加开机自启（未要求）
- 不动 installer C++ 源码
- 不引入鉴权（已知限制，超出本次范围）
- 不把 `useAudioPlayer` 等模块级单例迁进 Pinia（属激进重构）
