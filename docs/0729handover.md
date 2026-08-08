# 0729 交接文档

> 面向下一个 Agent 的交接。记录当前项目状态、本轮（0729）改动、技术细节、构建/测试/运行方式与待办。旧版 `HANDOVER.md` / `0726handover.md` / `0727handover.md` / `0728handover.md` 已全部删除，本文为唯一交接文档。
> 最后更新：2026-07-29

---

## 一、项目概述

**小播点歌工具（BCTools）**：面向校园广播站的本地桌面工具。Vue 3 前端（`Web/`）+ Go 后端（`GoServer/`），由 `build.py` 打包成单文件 exe，运行后自动开浏览器访问 `http://localhost:1743/`。

核心功能：导入 NCM/MP3/FLAC/WAV/MP4 并转码、NCM 解密（um-react 前端解密 + 后端暂存跨页导入）、宿舍歌单/播音歌单按周管理、按日期/时段分配、导出歌单图片、文件整理（复制/移动 + 缺失歌曲静音占位）、快照恢复、手机端局域网点歌、移动热点。

- **端口**：固定 `1743`，单实例（`main.go` 启动时强制占用，重复启动会被拦截）。
- **数据目录**：`%APPDATA%\BroadcastTool\`（`songs/` `temp/` `snapshots/` `decrypt-staging/` `logs/app.log` `data.json`）。
- **版本**：`5.5.0.0`（唯一源 `GoServer/internal/version/version.go` 的 `const Version`，`versioninfo.json` 同步）。
- **鉴权**：无 session/JWT 中间件，API 对可信局域网开放（`main.go` 内有 `SECURITY NOTICE`）。`/api/auth/verify` 后端仍在，但前端 `api/auth.ts` 已删，目前无页面调用。

---

## 二、本轮（0729）改动总览

本轮做了三件事：① um-react 解密改后端暂存跨页导入；② 删除整个备份模块；③ 修复宿舍歌单页面。**全部未提交**（工作区改动，用户习惯不频繁提交）。

### 2.1 um-react 解密：后端暂存跨页导入（保证局域网/移动端兼容）

**背景**：um-react 是挂在 `/um-react/` 的 React SPA，在浏览器内解密 NCM 生成 blob URL。原先用 `postMessage` 跨页传给 SongImport，但**跨设备（手机/局域网其他机器）访问时 `window` 不共享，postMessage 失效**。

**方案**：改用后端暂存中转。

| 文件 | 说明 |
| --- | --- |
| `GoServer/handlers/decrypt.go`（新增） | `DecryptHandler`：`POST /api/decrypt/stage`（multipart `file`，存为 `<stageId><ext>` + `.meta` 旁车 JSON `{filename,size,ext}`，返回 `{stageId,filename,size}`）；`GET /stage/{id}`（meta）；`GET /stage/{id}/file`（原始字节，`Content-Disposition: attachment`）；`DELETE /stage/{id}`（204）。`isValidStageID` 校验裸 UUID（仅 hex+连字符，防穿越）；`cleanupStale` 清理 >24h 文件。 |
| `GoServer/paths/paths.go` | 新增 `GetDecryptStagingDir(appDataDir) = appDataDir/decrypt-staging`。 |
| `GoServer/routes/routes.go` | `HandlerSet` 增加 `Decrypt *handlers.DecryptHandler`；新增 `/decrypt` 路由组。 |
| `GoServer/main.go` | `Decrypt: handlers.NewDecryptHandler(appDataDir)`。 |
| `Web/src/api/decrypt.ts`（新增） | `stageDecrypted(file,filename)` POST `/decrypt/stage`（`timeout:0` 不限）；`fetchStageMeta`；`fetchStageFile`（`responseType:blob`）；`deleteStage`。 |
| `scripts/um-react-bridge.js`（重写） | 由 `build.py` **非破坏性**注入 um-react 的 `index.html`。`MutationObserver` 监听 `[data-testid="file-row"]`，找到 `[data-testid="audio-download"]` 锚点（`href=blobUrl`、`download=fileName`），注入「导入」按钮。点击：`fetch(blobUrl)->blob->POST /api/decrypt/stage`（FormData）-> `{stageId}` -> `window.location.href = '/#/song/import?stage=' + encodeURIComponent(stageId)`。常量 `STAGE_URL/IMPORT_URL/LOGO_URL`。 |
| `Web/src/views/SongImport.vue` | `onMounted` 先 `fetchSongs`+`fetchSettings`，再检查 `route.query.stage` -> `openImportFromStage`。`confirmImportFromStage`：`fetchStageFile`->blob->`new File`-> 设 `selectedDate` + `handleFiles({fileList:[file]})` -> `deleteStage` -> `router.replace({query:{}})`。新增 `n-modal`（preset card）含 `n-date-picker`（`type="date"`，**不要** `value-format`；`importDate` 为 `ref<number|null>(dayjs().startOf("day").valueOf())`，确认时 `dayjs(importDate.value).format("YYYY-MM-DD")` 转给 `selectedDate`）。 |

> ⚠️ `n-date-picker` 类型陷阱：`value-format="yyyy-MM-dd"` 会让 `v-model` 变 string，与 naive-ui 的 `Value` 类型冲突（`vue-tsc` 报错，且 `components.d.ts` 重新生成后才暴露）。统一用**时间戳 number**，确认时再 `dayjs().format()`。

### 2.2 删除整个备份模块

保留快照（`snapshotstore`）与 `repo.go` 内部的防损坏备份，仅删除用户 facing 的「自动备份/立即备份」。

- **删除**：`GoServer/handlers/sync.go`（`SyncHandler/HandleBackup`）、`GoServer/store/backupstore/`（整目录）、`Web/src/api/sync.ts`（`backup()`）。
- **routes**：移除 `Sync *handlers.SyncHandler` 字段与 `/sync` 路由组。
- **main.go**：移除 `backupstore` 导入、`backupStore` 初始化、`Sync` handler 装配。
- **models/settings.go**：`Settings`/`PublicSettings`/`UpdateSettingsRequest` 移除 `AutoBackupPath`/`AutoBackupEnabled`。
- **store/settingstore/settingstore.go**：`GetSettings` 公开映射与 `UpdateSettings` 分支移除 autoBackup。
- **internal/repo/data.go**：`DefaultSettings()` 移除 autoBackup 默认值。
- **models/request.go**：移除未用的 `BackupResponse`。
- **前端**：`Settings.vue` 重写（删备份 UI 卡片、`backupNow/browsePath/saveBackupSetting`、`useMessage`/`selectDir` 导入）；`stores/settings.ts` 移除 autoBackup refs；`api/types.ts` 移除 `autoBackupPath/autoBackupEnabled`；`Guide.vue` 删「备份与恢复」折叠项；`locales/{zh-CN,en}.json` 删 `settings.backup*`/`guide.backup*`，`softwareIntro` 去掉「备份恢复」。
- **验证**：`/api/sync/backup` -> 404；`/api/snapshots` -> 200；设置返回无 autoBackup。

### 2.3 修复宿舍歌单页面（宿舍歌单 = `DormManage.vue` + `DormGrid.vue`）

**背景**：宿舍页在上一轮被重构成**按周卡片网格**（仿播音页 `BroadcastGrid`），新增 `DormGrid.vue` + `TimeSlotModal.vue`，但从未提交/构建/测试，用户反馈「页面坏了」。

**根因**：`DormGrid.addSong` 创建宿舍歌曲时传 `title: ""`，而后端对 dorm 类型**强制要求非空标题**（`handlers/songs.go` HandleCreate + `songstore.go` AddSong 都有校验 -> `400 歌名不能为空`）。导致每个时段的「添加歌曲」按钮**静默失败**（前端 `catch` 只 `console.error`），看起来页面不能用。

**修复（用户明确要求：默认不要填充"新歌曲"三字）**：
- 后端：移除 dorm 非空标题校验 —— `handlers/songs.go` HandleCreate（原 ~98 行）与 `songstore.go` AddSong（原 ~148 行）。现在 dorm **允许空标题创建**，与 broadcast 一致。（xlsx 批量导入路径 `HandleImport` 仍拒绝空标题行，那是另一条流程，保持不变。）
- 前端：`DormGrid.addSong` 用 `title: ""`（空），用户在内联输入框填写，placeholder 为「歌名」。
- `DormManage.onMounted`：去掉重复的 `songsStore.fetchSongs("dorm")`（仿 `BroadcastEdit`：父组件只 `fetchSettings`，子组件 `DormGrid` 自己 `onMounted` 拉歌曲）。

**DormGrid 结构**：props `weekDates:string[]`；`sectionsForDate(date)` 按 `settingsStore.timeSlots`（按 `dayIndex` 过滤）分组 + 未分配时段区（`timeSlotId` 为空或不匹配的歌曲）；每首歌内联 `n-input` 编辑标题（`@blur saveTitle`）、`duplicateWarnings` 黄字、试听（`player.play`，当前曲切换暂停）、删除。每个 slot 区渲染为一个 `n-timeline-item`（时段时间进原生 `:time` 属性，即时间轴节点的 time；未分配时段用 `type="warning"` 黄点区分），「添加歌曲」按钮与歌曲列表放在 timeline-item 内容插槽内。响应式 1/2/4 列（PC ≥1200px 一行四个；移动 1、平板 2）。

**DormManage 结构**：周导航（上周/下周/本周 + 「刷新」按钮调 `sortSongs("dorm")`+`fetchSongs`）、`<DormGrid :week-dates="weekDates" />`、右下角 `SettingsFab` 打开 `TimeSlotModal`。

**设计偏离说明**：0728 交接曾记录宿舍页用 `SongTable`（平铺表格）。本轮改为卡片网格 `DormGrid`，`SongTable.vue` 已删除。**用户已确认采用卡片网格 + 空标题添加**。卡片网格相比旧 `SongTable` **暂缺**：备注(remark)编辑、改日期、跨时段重新分配、xlsx 导入/导出（导出在独立的「歌单导出」`/export` 页）。用户尚未要求补这些。

### 2.4 新增测试

| 文件 | 内容 |
| --- | --- |
| `Web/src/components/dorm/DormGrid.test.ts` | 3 项：渲染周卡片/歌曲、按时段+未分配分组、点击「添加歌曲」断言 `createSong` 以 `title:""` 调用。 |
| `Web/src/views/DormManage.test.ts` | 1 项：在完整 naive-ui providers（NConfigProvider>NMessageProvider>NDialogProvider>NNotificationProvider）下挂载整页，渲染当前周歌曲、无运行时错误。 |
| `GoServer/store/songstore/songstore_test.go` | `TestDormEmptyTitleAllowed`：dorm 空标题创建应成功。 |

vitest 配置（`vitest.config.ts`）：`environment:"jsdom"`、`globals:true`、alias `@`。`@vue/test-utils` + `jsdom` 可用，无 headless 浏览器。测试中注册 naive-ui 组件用 `global.components:{...}` 或 `global.plugins:[naive]`（默认 default export 即可 `app.use`）。

---

## 三、技术栈与架构

| 层 | 技术 |
| --- | --- |
| 后端 | Go 1.22+，chi v5 路由，`embed.FS` 静态服务 + SPA fallback，Excelize，bcrypt，unlock-music/um-react 解密 |
| 前端 | Vue 3 + TS + Vite，naive-ui（`unplugin-vue-components`+`NaiveUiResolver` 自动导入，`components.d.ts` 构建时再生成），Pinia，vue-router（`createWebHashHistory`），Axios，dayjs(+isoWeek) |
| 工具 | ffmpeg/ffprobe（转码/元数据，精简版内嵌进 exe），modern-screenshot（PNG），JSZip，DOMPurify |
| 打包 | Go `embed` 嵌入前端产物；单文件 exe 分发 |

**目录**（关键）：
```
GoServer/
  main.go                 入口：目录初始化、ffmpeg 查找、单实例端口 1743、路由、静态+SPA、自动开浏览器
  handlers/               songs/files/settings/snapshot/decrypt/auth/update/system/network
  routes/routes.go        路由注册（/api 下）
  store/songstore/        歌曲 CRUD + 时段冲突 + 查重
  store/settingstore/     设置读写
  store/snapshotstore/    每日快照
  internal/repo/          data.json 持久化（原子写 + RWMutex + 防损坏备份）
  internal/version/       版本号唯一源
  internal/binembed/      内嵌 ffmpeg/ffprobe
  paths/ paths.go         AppData 子目录
  converter/              ffmpeg.go/ffprobe.go/process.go（ProcessFile/StashFile/NCM 解密 60s 超时）
  hotspot/                Windows 热点（WinRT + netsh 并行）
Web/src/
  api/                    client(axios) + songs/files/settings/decrypt/snapshots/...
  stores/                 songs.ts/settings.ts/export.ts
  views/                  Home/SongImport/DormManage/BroadcastEdit/Export/Organize/Mobile/Decrypt/Settings/Guide/Error
  components/             dorm/(DormGrid,TimeSlotModal) broadcast/(BroadcastGrid,BroadcastColumnMapModal) common/(SettingsFab,...) calendar/ song-import/
  composables/            useAudioPlayer/useFileProcessor/useAppConfig/useExportImage
  utils/songSimilarity.ts normalizeTitle/levenshtein/isSimilar/findSimilarSongs
  i18n/index.ts           useI18n（t/weekdayName/weekdayShortName/weekdayLabel/dayIndexFromDate）
scripts/um-react-bridge.js  um-react 非破坏性桥接注入
build.py                  一键构建（corepack pnpm + go）
```

---

## 四、数据模型

```go
type Song struct {
    ID         int64   `json:"id"`
    Date       string  `json:"date"`           // YYYY-MM-DD
    Weekday    string  `json:"weekday"`        // 后端由 date 计算中文星期
    Title      string  `json:"title"`          // dorm/broadcast 均允许空
    Artist     string  `json:"artist"`
    Remark     string  `json:"remark"`
    FilePath   string  `json:"filePath"`       // songs/<uuid>.mp3
    TimeSlotID *string `json:"timeSlotId"`     // dorm 用；broadcast 不用
    Period     string  `json:"period"`         // broadcast 用："noon"|"afternoon"
    CreatedAt  time.Time `json:"createdAt"`
}
```

- **dorm**：`date` + 可选 `timeSlotId`（卡片网格按时段分组）。同一天同一 `timeSlotId` 仅允许一首（`ErrSlotConflict` -> 400）。
- **broadcast**：`date` + `period`（noon/afternoon）。允许同天同时段多首。
- 两者均按 **ISO 周** 过滤展示；均允许空标题（本轮放开 dorm）。
- `data.json`：`{dormSongs, broadcastSongs, settings}`。原子写（`tmp`+`os.Rename`），`sync.RWMutex`。每日首次写生成 `snapshots/snapshot-YYYY-MM-DD.json`，保留最近 30 个。
- 默认时段：周一~周六 06:20/06:35/13:50/18:30，周日 18:30，共 25 个 UUID。
- 查重：`duplicateCheckDays`（1~365，默认 30），`findSimilarSongs(title,date,allSongs,days)` 在过去 N 天找相似标题（归一化 + 精确/包含/Levenshtein），**排除同日**（不自匹配）。DormGrid 与 ImportProcessingGrid 显示黄字警告。

---

## 五、API 路由（当前，均在 `/api` 下）

```
/songs      GET /(type,dates) · GET /{id} · POST / · PUT /{id} · DELETE /{id} · POST /sort?type=dorm · POST /import?type · GET /export?type&dates
/files      POST /process · POST /stash · POST /organize · POST /select-dir · GET /browse?dir · GET /stream?file · GET /silent · DELETE /source
/auth       POST /verify                  # 后端仍在，前端已不调用
/settings   GET · PUT
/snapshots  GET / · POST /restore
/check-update GET
/system/status GET
/network    GET /info · GET /qr
/hotspot    POST /start · GET /status
/decrypt    POST /stage · GET /stage/{id} · GET /stage/{id}/file · DELETE /stage/{id}
/um-react/  React SPA（非 /api，build.py 注入 bridge）
```
**已删除**：`/sync`（备份模块）。`/songs/sort` 仅支持 `type=dorm`。

---

## 六、构建 / 测试 / 运行

```bash
# 前端
cd WebUI
corepack pnpm run type-check      # vue-tsc --noEmit
corepack pnpm run test            # vitest run（jsdom）
corepack pnpm run build           # vite build

# 后端
cd GoSever
go vet ./...
go test ./...

# 一键全量构建（前端+um-react+后端，产物到 dist/）
cd D:\BCTools
python build.py --no-mirror       # 无管理员/无镜像；内部用 corepack pnpm
```

- **产物**：`dist/bctools.exe`（release，~44MB，内嵌 ffmpeg/ffprobe + 前端）+ `dist/bctool_dev.exe`（~2MB，dev 启动器）。
- **运行**：双击 `dist/bctools.exe`（或 dev）。浏览器 `http://localhost:1743/`。
- **重启**（单实例，必须先杀旧）：
  ```bash
  taskkill //F //IM bctools.exe
  cmd //c start "" "D:\BCTools\dist\bctools.exe"
  ```
  > 不要用 `tasklist //FO CSV` 取 PID —— 中文表头会让 sed 截取出错（曾误杀 PID 1）。直接按映像名 `taskkill //F //IM bctools.exe`。

---

## 七、约束与踩坑（务必遵守）

1. **pnpm 走 corepack，无管理员权限**：一律 `corepack pnpm ...`，不要用裸 `pnpm`（会失败）。`build.py` 内部已处理。
2. **构建产物只进 `dist/`**，不要把 exe 散落别处。
3. **Windows 控制台默认 gbk**：读 `zh-CN.json` 等 utf-8 文件用 `python io.open(...,encoding='utf-8')`；`node -e` 输出中文到 gbk 控制台会报错退出（不影响数据，只是脚本失败）。
4. **单实例端口 1743**：新构建后必须先 `taskkill //F //IM bctools.exe` 再启动，否则旧进程仍服务旧 bundle。
5. **naive-ui 自动导入只在 vite 构建生效**；vitest 里不会自动解析 `n-*`，需在 `mount` 的 `global.components`/`global.plugins:[naive]` 注册（否则告警 "Failed to resolve component"，但内容仍渲染）。
6. **i18n 仅中文**：`currentLocale` 硬编码 `"zh-CN"`，不读写 localStorage；`useAppConfig` 强制 `languageSwitchEnabled=false`。`en.json` 结构保留但不用。新增文案只需保证 `zh-CN.json` 有键（`t()` 缺键返回 key 字符串本身，不崩溃）。
7. **无鉴权**：API 对局域网开放，勿引入依赖 session 的假设。
8. **当前已知缺 i18n 键**：`settings.advancedSettings`（`TimeSlotModal` 标题 fallback 到 key 字符串，纯展示问题）。

---

## 八、待办 / 已知问题（供后续）

> 0729 后续批次：#1/#2/#5 已完成；#3 用户尚未要求、暂缓；#4 需真实 NCM 文件，无法本地验证。

1. ✅ **导出工具崩溃 bug**（已完成）：`useExportImage.ts` 加固--`slotsForDate` 对 `timeSlots` 做数组兜底；新增 `periodOf(s)` 安全推断广播时段（`period` 优先，`createdAt || date` 兜底，NaN 归下午）替换 V2 内联 `dayjs(s.createdAt).hour()`；`buildDormPosterTable` 在无时段时返回提示而非退化网格；`generateImage` 在 `rawHtml` 为空时抛 `export.noData`；`domToPng` 包 try/catch（记日志后重抛）。`Export.vue` 导出 dorm 前校验 `timeSlots` 为空则提示 `export.noSlots`，catch 改为展示实际错误信息。新增 i18n `export.noData`/`export.noSlots`。数据加载由 `Export.vue onMounted` 的 `Promise.all` 保证。
2. ✅ **软件指南独立页 `/guide`**（已完成）：`Guide.vue` 重写为主题色标题 + 返回按钮 + `n-card`/`n-collapse` 完整指南页，内容按当前功能刷新（导入/宿舍/播音/导出/换卡/手机点歌/解密/密码重置 8 节，原「分配歌曲」等过时描述已替换）；`Settings.vue`「关于软件」卡片新增「软件指南」按钮（`about.guideButton`）跳 `/guide`；侧边栏不加入口（`AppLayout` 菜单不含 `/guide`）。新增/更新 `guide.*` i18n（zh-CN + en 同步）。
3. ⏸ **宿舍卡片网格功能缺口**（暂缓，用户尚未要求）：备注(remark)编辑、改日期、跨时段重新分配、xlsx 导入/导出（导出在 `/export` 页）。本轮未做，待用户提出。
4. ⏸ **NCM 真实文件未实测**（需真实文件，无法本地验证）：仅 60s 超时防卡死，不保证所有 NCM 解密成功。
5. ✅ **dorm 空标题导出**（已完成）：`useExportImage.ts` 新增 `displayTitle(song)`（空标题->`common.unnamed`「（未命名）」），用于宿舍/播音的简约表格与海报各构建函数；`playlistTable.ts buildDormTableHTML` 增加 `emptyTitleText` 选项由调用方传入。新增 i18n `common.unnamed`。

---

## 九、Git 状态

- 当前分支 `master`（主分支 `main`，PR 目标）。
- **大量未提交改动**（本轮 um-react 暂存、备份删除、宿舍修复均未 commit）。用户倾向不频繁提交；如需提交请先确认。
- 关键删除（相对 HEAD）：`SongTable.vue`、`sync.go`、`backupstore/`、`api/sync.ts`、`api/auth.ts`、`About.vue`、`Advanced.vue`。
- 关键新增（未跟踪）：`handlers/decrypt.go`、`api/decrypt.ts`、`components/dorm/`、`scripts/um-react-bridge.js`、`handlers/network.go`、`handlers/system.go`、`hotspot/`、`internal/`、`server/`、`runner/`、`launcher/`、`browser/`、`autostart/`、`network/` 等。

---

## 十、记忆库指针

`C:\Users\pc\.claude\projects\D--BCTools\memory\`：
- `build-output-location.md` — 产物统一输出 `dist/`。
- `pnpm-via-corepack-no-admin.md` — 无管理员，用 `corepack pnpm`。
- `umreact-import-backend-staging.md` — um-react 导入走后端暂存（bridge->/api/decrypt/stage->SongImport?stage=）。
