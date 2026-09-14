# 导出歌曲文件 + 本地化数据存储重构

## 目标（已与用户确认的决策）

1. Organize 页面改名「导出歌曲文件」；隐藏（弃用不删除）SD 卡目录选择组件；点导出按钮后端直接把该周文件夹拷贝到桌面。
2. 所有数据迁移到 `exe同目录\BctoolData\`，拷贝这一个目录即可完整迁移：
   - `BctoolData\MusicFiles\<YYYY年第N周>\NN.mp3` 存歌曲（周文件夹 = SD 最终形态）
   - data.json、temp、snapshots、logs、bin、decrypt-staging、deleted_songs_log.json 全部移入 BctoolData
3. 周文件夹编号沿用 SD 时段位置编号：空时段存 17.43s 静音占位（ID3 标签 title/artist/album 写「空音频」）；同时段多首合并成一个 MP3。
4. 合并文件的还原信息（每首歌的字节区间 + 原始 ID3 标签）写入**现有 data.json**，不新建 JSON 文件——data.json 即歌曲索引。
5. 已有 %APPDATA%\BroadcastTool 数据首次启动自动迁移（原目录保留作备份）。
6. 数据迁移功能前后端一起移除。

## 后端改动

### A. paths.go — 数据根目录

- `GetAppDataDir()` → 改名 `GetDataRootDir()`：`filepath.Join(GetExecutableDir(), "BctoolData")`。
- dev（`go run`，exe 在 `%TEMP%\go-build*`）回退到工作目录（`GoServer\BctoolData`，加 .gitignore），避免每次构建数据被清。
- 一次性测试版（ephemeral）逻辑不变。
- 新增 `GetMusicFilesDir(root)` = `root\MusicFiles`；新增 `GetStagingDir` = `root\MusicFiles\_staging`（上传后未入周文件夹的临时音频）。
- 周文件夹名：`fmt.Sprintf("%d年第%d周", isoYear, isoWeek)`，用 `time.ISOWeek()`。

### B. data.json 结构扩展（repo/data.go + models）

```go
type Data struct {
    ...现有字段...
    // 周文件夹索引：key = 周名，value = 编号("05") -> 该编号文件的信息
    Weeks map[string]map[string]WeekFile `json:"weeks,omitempty"`
}

type WeekFile struct {
    Kind    string       `json:"kind"`              // "silent" | "song" | "merged"
    SongIDs []int64      `json:"songIds,omitempty"`
    Parts   []MergedPart `json:"parts,omitempty"`   // merged 时的还原信息
}

type MergedPart struct {
    SongID int64  `json:"songId"`
    ID3v2  string `json:"id3v2,omitempty"`  // 原始 ID3v2 头（base64）
    ID3v1  string `json:"id3v1,omitempty"`  // 原始 ID3v1 尾（base64）
    Offset int64  `json:"offset"`           // 裸音频帧在合并文件中的区间
    Length int64  `json:"length"`
}
```

还原算法：`part.ID3v2 + merged[Offset:Offset+Length] + part.ID3v1`（与 merge.go 的剥标签逻辑对称）。

### C. 新服务 services/musiclibrary.go — 周文件夹维护引擎

编号算法：把 `Web/src/utils/exportEntries.ts` 的逻辑移植到 Go（遍历周一~周日 × 当天时段按 order 排序，seq++）。

- `SyncWeek(year, week)`：按 dorm 歌曲与 timeSlots 计算期望布局，与 `data.json` 的 Weeks 索引对比，仅对差异执行：生成静音 / 搬运源文件 / 重新合并 / 重命名 / 删除，最后更新索引（rename 走临时名防冲突）。
- `PlaceSong(song)`：新增歌曲后落位（staging 文件 → 周文件夹该时段编号；时段已有歌则并入合并）。
- `RemoveSong(song)`：删除歌曲后维护时段文件（最后一首 → 静音占位；多首 → 用 Parts 从现有合并文件抽出剩余成员重组）。
- `MoveSong(old, new)`：改日期/时段后重新落位两个受影响时段。
- `ResyncAll()`：timeSlots 配置变化后全量重排（编号整体平移 → rename 级联；被删时段的歌曲变为未分配，其音频抽到 `_staging\<songID>.mp3`）。
- 静音占位生成时 ffmpeg 加 `-metadata title=空音频 -metadata artist=空音频 -metadata album=空音频`。
- 合并文件成员的单独试听：`ExtractPart` 到 temp 再流式返回。

### D. handlers 接线

- `files.go` HandleProcess/HandleStash、`decrypt.go` HandleStageImport：产出文件写入 `_staging\`（仍用 uuid 名，仅暂存用）；`validateSourcePath` 的裸文件名解析改为 `_staging` → temp。
- `songs.go` HandleCreate / HandleUpdate / HandleDelete：成功后调 musiclibrary 的 Place/Move/Remove，统一更新该时段所有歌曲的 FilePath（同时段歌曲共享 `周名/NN.mp3`）。
- `settings.go` 时段配置更新后调 `ResyncAll()`；恢复已删歌曲走 Place 逻辑。
- `files.go` HandleStream：新增可选 `songId` 参数——若该歌属于合并文件，抽出自己的 Part 播放（前端播放按钮传 songId）。
- **新端点** `POST /api/files/export-week` `{year, week, confirm}`：先 `SyncWeek` 保证最终形态，再把 `MusicFiles\<周名>` 整目录复制到桌面（`%USERPROFILE%\Desktop`，不存在则试 `OneDrive\Desktop`）；桌面已存在同名文件夹时返回 confirmNeeded + 已有文件列表（沿用现有覆盖确认弹窗）。
- 旧 `/api/files/organize`、`/select-dir`、`/merge`、`/silent` 端点保留不删（弃用）。

### E. 启动自动迁移（main.go，repo 初始化后、conv 初始化后）

`%APPDATA%\BroadcastTool\data.json` 存在且 `BctoolData\data.json` 不存在时：
- 拷贝 data.json、deleted_songs_log.json、snapshots\（logs/bin/temp 不迁）。
- 旧 uuid 歌曲文件以「源文件」身份走 musiclibrary 重建所有周文件夹（复用同步引擎，源解析指向旧 songs 目录），生成新编号、静音占位、合并与 Parts。
- 旧 %APPDATA% 目录原样保留作备份；迁移写日志。

### F. 移除数据迁移功能

- 后端：删 `handlers/migration.go`、`services/migration_service.go`、`models/migration.go`、routes 注册与 HandlerSet.Migration。
- 前端：删 Settings.vue 的迁移按钮、路由 `/settings/migration`、`DataMigration.vue`、`api/migration.ts`、locale `migration.*` / `settings.dataMigration`（en.json 同步，跑 `npm run prune:i18n`）。

## 前端改动

- **Organize.vue**：
  - SD 目录选择行用 `v-if="false"`（或常量开关）隐藏，browse/targetDir 相关代码保留。
  - 按钮改「导出歌曲文件」，直接调 `exportWeek(year, week)`；去掉 copyMode 限制；保留覆盖确认弹窗（后端 confirmNeeded）。
  - doFrontendCopy/FSA 路径保留代码但不可达（弃用）。
  - 播放表格里同时段多首歌时 per-song 试听走 stream+songId。
- `api/files.ts`：新增 `exportWeek`；`useAudioPlayer`/`AudioPlayButton` stream 调用加 songId。
- 文案：nav/页面标题「换卡工具」→「导出歌曲文件」（zh-CN.json、en.json、router meta），organize.* 相关文案同步改。
- router 路由路径 `/organize` 保留（书签兼容），仅改标题。

## 安装程序（installer）

- 卸载逻辑（UninstallerCore.cpp）：删安装目录时**跳过 `BctoolData`**（用户数据），其余照旧；%APPDATA%\BroadcastTool 清理逻辑保留（兜底清旧版残留）。

## 版本

- 版本号 5.6.1 → 5.7.0（version.go、package.json、versioninfo.json、installer/src/version.h，含 4 处）。

## 测试

- Go：编号移植逻辑单测（对齐 exportEntries.test.ts 用例）、MergedPart 抽取/重组 round-trip、周名 ISO week 计算、迁移逻辑。
- Web：`npm run type-check`、`npm run test`；`go test ./...`。

## 实施顺序

1. paths 数据根目录 + main.go 接线（含 dev 回退）
2. musiclibrary 引擎 + data.json 扩展 + 单测
3. handlers 接线（上传/增删改/时段/流式/导出端点）
4. 启动自动迁移
5. 前端 Organize 页改造 + 文案 + 播放 songId
6. 移除迁移功能（前后端）
7. 卸载程序保留 BctoolData + 版本号
8. 全量 type-check / test / build
