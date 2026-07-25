# 广播站歌单工具 — 交接文档

> 本文档记录项目技术细节、当前状态与后续注意事项，供其他 Agent 继续开发使用。

---

## 1. 项目概述

一个面向校园广播站的本地桌面工具，核心功能：

- 导入 NCM / MP3 / FLAC / WAV / MP4 等音频
- NCM 自动解密并转码为 MP3
- 宿舍歌单、播音歌单管理
- 按日期/时段分配歌曲
- 导出歌单图片（PNG）与批量打包
- 文件整理（复制/移动到目标目录，缺失歌曲生成静音占位）
- 配置备份、快照恢复

运行时双击 `dist/broadcast-tool.exe`，后端启动 HTTP 服务并自动打开浏览器访问 `http://localhost:1743/`。

---

## 2. 技术栈

| 层级 | 技术 |
| --- | --- |
| 后端 | Go 1.22+，Chi 路由，Zap/Lumberjack 日志，Excelize 表格，bcrypt 密码，unlock-music CLI 库 |
| 前端 | Vue 3 + TypeScript + Vite，Naive UI，Pinia，Vue Router，Axios |
| 工具 | ffmpeg / ffprobe（音频转码与元数据），modern-screenshot（PNG 导出），JSZip（批量下载），DOMPurify（模板安全） |
| 打包 | Go `embed` 嵌入 `frontend/dist`，单文件 exe 分发 |

---

## 3. 目录结构

```text
.
├── backend/                 Go 后端
│   ├── main.go              入口：目录初始化、ffmpeg 查找、路由、静态文件、自动开浏览器
│   ├── embed/dist/          嵌入的前端构建产物（由 build 脚本复制）
│   ├── converter/           音频转换与 NCM 解密
│   │   ├── ffmpeg.go        ffmpeg 调用、并发控制
│   │   ├── ffprobe.go       元数据解析
│   │   ├── ncm.go           unlock-music 解密封装
│   │   └── process.go       ProcessFile / StashFile 流程、超时保护
│   ├── handlers/            HTTP handler
│   │   ├── files.go         文件上传/转换/整理/目录浏览
│   │   ├── songs.go         歌曲 CRUD / 导入导出
│   │   ├── settings.go      设置读写
│   │   ├── auth.go          密码校验
│   │   ├── sync.go          备份
│   │   ├── snapshot.go      快照列表/恢复
│   │   └── update.go        检查更新
│   ├── routes/routes.go     API 路由注册
│   ├── store/store.go       数据持久化、快照、默认时段生成
│   ├── models/models.go     数据结构与请求/响应模型
│   ├── response/response.go 统一 JSON 响应
│   ├── middleware/          CORS / 日志 / Recovery
│   ├── paths/paths.go       应用目录路径
│   └── validation/          参数校验
├── frontend/                Vue 前端
│   ├── public/
│   │   └── config.ini       运行时配置（主题色、语言切换、xlsx 开关）
│   ├── src/
│   │   ├── api/             axios 封装与接口函数
│   │   ├── stores/          Pinia 状态（songs / settings / conversion / export / error）
│   │   ├── views/           页面
│   │   │   ├── SongImport.vue    歌曲导入（原 DormImport，支持 dorm/broadcast）
│   │   │   ├── DormManage.vue    宿舍歌单管理
│   │   │   ├── BroadcastEdit.vue 播音歌单表格编辑
│   │   │   ├── Export.vue        歌单导出
│   │   │   ├── Organize.vue      文件整理
│   │   │   ├── Settings.vue      设置
│   │   │   ├── Advanced.vue      高级设置（时段配置，需密码）
│   │   │   └── ...
│   │   ├── components/      公共组件与业务组件
│   │   ├── composables/     useAppConfig / useAudioPreview
│   │   ├── locales/         zh-CN.json / en.json
│   │   ├── i18n/index.ts    简单 i18n 组合式函数
│   │   ├── router/index.ts  路由配置
│   │   ├── App.vue          根组件（含 n-message-provider 等）
│   │   └── main.ts
│   └── dist/                前端构建产物
├── dist/                    最终分发包（exe + ffmpeg + ffprobe）
├── README.md                用户说明文档
└── HANDOVER.md              本文档
```

---

## 4. 后端关键技术细节

### 4.1 启动流程（`main.go`）

1. 获取 `APPDATA/BroadcastTool` 作为数据目录。
2. 创建 `temp/`、`snapshots/`、`logs/`、`bin/`。
3. 初始化 Store（加载 `data.json`，不存在则生成默认 25 个时段）。
4. 查找 `ffmpeg.exe` / `ffprobe.exe`：顺序为 `APPDATA/BroadcastTool/bin/` > exe 同目录 > 系统 PATH。
5. 注册 Chi 路由与 API。
6. 注册 embed 静态文件服务 + SPA fallback。
7. 监听 `1743~1749`，启动后 500ms 自动调用 `cmd /c start` 打开浏览器。

### 4.2 数据存储（`store/store.go`）

- 主文件：`%APPDATA%/BroadcastTool/data.json`
- 结构：`{ "dormSongs": [], "broadcastSongs": [], "settings": {...} }`
- 原子写入：`data.json.tmp` → `os.Rename`
- 锁：`sync.RWMutex`，GET 读锁，写操作写锁
- 快照：每天首次写入时生成 `snapshots/snapshot-YYYY-MM-DD.json`，保留最近 30 个
- 默认时段：周一~周六每天 06:20/06:35/13:50/18:30，周日 18:30，共 25 个 UUID（旧数据中的 18:45 时段不会自动删除）

### 4.3 模型（`models/models.go`）

```go
type Song struct {
    ID         int64   `json:"id"`
    Date       string  `json:"date"`
    Title      string  `json:"title"`
    Artist     string  `json:"artist"`
    Remark     string  `json:"remark"`
    FilePath   string  `json:"filePath"`
    TimeSlotID *string `json:"timeSlotId"`
    Weekday    string  `json:"weekday"`     // 后端由 date 计算中文星期
    CreatedAt  string  `json:"createdAt"`
}
```

```go
type Settings struct {
    TimeSlots                 []TimeSlot `json:"timeSlots"`
    AllowTemplateJS           bool       `json:"allowTemplateJS"`
    AutoBackupPath            string     `json:"autoBackupPath"`
    AutoBackupEnabled         bool       `json:"autoBackupEnabled"`
    SilentPlaceholderDuration int        `json:"silentPlaceholderDuration"`
    AdminPasswordHint         string     `json:"adminPasswordHint"`
    Locale                    string     `json:"locale"`
    Version                   string     `json:"version"`
    DownloadURL               string     `json:"downloadUrl"`
    AdminPasswordHash         string     `json:"-"` // 不序列化输出
}
```

### 4.4 文件转换流程

- **ProcessFile**：非 MP3 转码为 MP3；NCM 等加密格式先解密再转码。
- **StashFile**：MP3 仅提取元数据并保存；加密格式仍走解密+转码。
- 上传文件大小限制：`512 MB`（`ParseMultipartForm` + `MaxBytesReader`）。
- 并发控制：`FFMpegConverter.semaphore` 当前限制 **4** 个并发 ffmpeg 任务。
- **NCM 防卡死**：`process.go` 中 `DecryptToFile` 运行在独立 goroutine，外层 60 秒超时，超时后关闭源文件描述符强制中断解码。

### 4.5 API 路由

```text
GET    /api/health
GET    /api/settings
PUT    /api/settings

GET    /api/songs?type=dorm|broadcast&dates=...
POST   /api/songs
GET    /api/songs/{id}
PUT    /api/songs/{id}
DELETE /api/songs/{id}
POST   /api/songs/import?type=...
GET    /api/songs/export?type=...&dates=...

POST   /api/files/process
POST   /api/files/stash
POST   /api/files/organize
POST   /api/files/select-dir
GET    /api/files/browse?dir=...
GET    /api/files/preview?file=...

POST   /api/sync/backup
POST   /api/auth/verify

GET    /api/snapshots
POST   /api/snapshots/restore
GET    /api/check-update
```

### 4.6 静态文件与 SPA fallback

`main.go` 使用 `embed.FS` 嵌入 `backend/embed/dist`。所有非 `/api/` 请求尝试读取对应文件，失败则回退到 `index.html`，使前端路由刷新不 404。

### 4.7 安全说明

- 后端敏感接口目前没有真正的 session 鉴权（代码中有 `SECURITY NOTICE`）。
- 高级设置页面的密码验证仅由前端 `sessionStorage.setItem("advanced_authenticated", "1")` 控制，后端 `/api/auth/verify` 只校验密码 hash。
- 路径穿越防护：文件整理与目录浏览均校验绝对路径前缀，禁止 `..` 与系统敏感目录。

---

## 5. 前端关键技术细节

### 5.1 入口与全局 Provider

`App.vue` 已包裹：

```vue
<n-config-provider>
  <n-message-provider>
    <n-dialog-provider>
      <n-notification-provider>
        <router-view />
      </n-notification-provider>
    </n-dialog-provider>
  </n-message-provider>
</n-config-provider>
```

因此 `useMessage` 不应再报 “No outer provider”。若仍出现，检查是否在 provider 挂载前调用。

### 5.2 路由

使用 `createWebHashHistory`，路径：

```text
/#/home
/#/song/import
/#/dorm/manage
/#/broadcast
/#/export
/#/settings
/#/advanced   （独立 layout，需 sessionStorage 认证标记）
```

### 5.3 状态管理

- `songs.ts`：dormSongs / broadcastSongs，提供 `assignSong(songId, timeSlotId)`。
- `settings.ts`：时段、备份、语言等。
- `conversion.ts` / `export.ts`：导出相关状态（部分页面可能已改用本地状态）。

### 5.4 歌曲导入页（`SongImport.vue`）

当前实现要点：

- 选择日期后进入上传区，顶部可切换 dorm / broadcast。
- 支持多文件拖拽/点击上传。
- **前端并发处理**：最多 **3** 个文件同时调用后端接口，提升批量转换速度。
- 每个文件处理完成后自动创建歌曲（`POST /api/songs`）。
- **分配到时段**：宿舍歌单完成卡片下方会显示当前日期可用时段下拉框，选择后调用 `songsStore.assignSong`（`PUT /api/songs/{id}`）。
- 卡片上可直接编辑歌名，失焦自动保存。
- **当天已点歌曲**：进入日期后会同步显示当天已有歌曲，使用原生 `<audio controls>` 试听，不再显示删除按钮。
- 时段冲突时前端提示“该时段已有歌曲，请更换时段或先删除原歌曲”。

### 5.5 歌单表格组件（`SongTable.vue`）

`BroadcastEdit.vue` 与 `DormManage.vue` 共用此组件：

- 按 **ISO 周** 分页显示，支持“上一周/下一周/本周”，标题显示 `YYYY 年第 N 周`。
- 添加歌曲默认日期为当前周周一（若 `DormManage` 已选某天则使用该天）。
- 行内编辑日期、歌名、备注；宿舍歌单额外显示时段下拉框。
- 行内试听按钮（调用全局音频播放器）。
- 删除按钮直接删除，不再弹出确认框。
- 导入/导出 xlsx 按钮受 `config.ini` 中 `importXlsxEnabled` / `exportXlsxEnabled` 控制。

### 5.6 宿舍歌单页（`DormManage.vue`）

- 顶部复用 `CalendarGrid`，点击某天即选中该天，表格会过滤到该日期并切换到对应周。
- 提供“清除日期筛选”按钮恢复显示整周。
- 下方使用 `SongTable.vue` 组件。

### 5.7 导出页（`Export.vue`）

- 选中日期汇总为一张图。
- 宿舍：行=日期，列=时段。
- 播音：按日期聚合列表。

### 5.8 国际化

`i18n/index.ts` 提供 `useI18n()`，支持 `{count}` 插值。语言包在 `locales/*.json`，当前已维护中/英两套。新增 UI 文案时必须同时更新两个文件。

### 5.9 运行时配置（`frontend/public/config.ini`）

前端启动时会通过 `fetch('/config.ini')` 加载配置，目前支持：

```ini
[ui]
languageSwitchEnabled=false
themeColor=#66ccff

[features]
importXlsxEnabled=false
exportXlsxEnabled=false
```

- `languageSwitchEnabled`：控制顶部与设置页的语言切换是否显示。
- `themeColor`：主题色，会注入到 Naive UI `themeOverrides` 与 CSS 变量 `--theme-color`。
- `importXlsxEnabled` / `exportXlsxEnabled`：控制歌单表格页面的 xlsx 导入/导出按钮是否显示，默认关闭。

配置由 `src/composables/useAppConfig.ts` 读取，`App.vue` 在 `onMounted` 中加载并应用。

### 5.10 组件自动导入

项目使用 `unplugin-auto-import` 与 `unplugin-vue-components`，Vue API 与 Naive UI 组件通常无需手动 import。但有时 `vue-tsc` 对 render 函数里的类型推断较严，必要时显式 import（如 `NButton`）。

---

## 6. 构建与打包

### 6.1 开发

```bash
# 前端
cd frontend
npm install
npm run dev

# 后端
cd backend
go run .
```

### 6.2 生产构建

```bash
cd frontend
npm run type-check
npm run build

cd ..
rm -rf backend/embed/dist
mkdir -p backend/embed/dist
cp -r frontend/dist/* backend/embed/dist/

cd backend
go mod tidy
go vet ./...
go test ./...
set CGO_ENABLED=0
go build -ldflags="-s -w" -o broadcast-tool.exe .

cd ..
mkdir -p dist
cp backend/broadcast-tool.exe dist/
# 确保 dist/ffmpeg.exe 与 dist/ffprobe.exe 存在
```

### 6.3 验证

启动 `dist/broadcast-tool.exe` 后：

```bash
curl http://localhost:1743/api/health
curl http://localhost:1743/api/settings | grep -o '"id"' | wc -l   # 应返回 25（新安装）
curl -F "file=@test.mp3" http://localhost:1743/api/files/process
```

---

## 7. 当前已知问题与 TODO

### 7.1 已修复/已优化

- [x] 播音歌单输入框无法编辑（本地状态 + 失焦保存）
- [x] “宿舍导入”改为“歌曲导入”并支持 dorm/broadcast 选择
- [x] 歌曲导入页增加“分配到时段”下拉框
- [x] 批量上传前端并发 + 后端 ffmpeg 并发提升转换速度
- [x] NCM 解密增加 60 秒超时保护
- [x] `n-message-provider` 等全局 Provider 已配置
- [x] 自动打开浏览器已实现
- [x] `dist/` 已包含 exe + ffmpeg + ffprobe
- [x] 新增“文件整理”页面，可按日期范围生成清单并复制/移动到目标目录
- [x] 歌单导出改为汇总一张图：宿舍 = 行是天、列是时段；播音 = 按日期聚合列表
- [x] 宿舍歌单页面改为与播音歌单一致的表格，复用 `SongTable.vue`
- [x] 播音歌单添加歌曲后立即刷新
- [x] 后端增加校验：同一天同时段只能有一首宿舍歌曲
- [x] 增加 `frontend/public/config.ini` 运行时配置（语言切换开关、主题色）
- [x] 有歌曲的日历日期用主题色高亮
- [x] 移除歌单/导入导出中的“歌手”字段，只保留歌名
- [x] 增加试听功能：`/api/files/preview?file=UUID.mp3` + 前端播放按钮
- [x] 歌曲导入页同步显示当天已点歌曲，使用原生 `<audio controls>` 试听
- [x] 时段冲突 400 错误提示改为更友好的文案
- [x] 移除默认的周日 18:45 时段
- [x] `config.ini` 增加 `importXlsxEnabled` / `exportXlsxEnabled` 开关，默认关闭
- [x] 侧边栏收起时不显示“站”字，去除菜单斜体
- [x] 宿舍歌单页增加日历，点击某天可选中当天
- [x] 宿舍/播音歌单表格改为按 ISO 周分页
- [x] 歌单表格删除按钮直接删除，不再确认
- [x] 清理无用文件：prompt txt、log、unused components/helpers

### 7.2 仍需关注

- [ ] **NCM 真实文件未实测**：超时保护只能避免无限挂起，不能保证所有 NCM 都能成功解密。若仍有失败，可改用 `github.com/keuin/ncm` 替换 unlock-music 的 NCM 分支，或引入 `unlock-music` 官方 CLI 子进程方案。
- [ ] **转换速度**：虽已并发，但大文件或加密文件仍需先解密再 ffmpeg 转码。若仍慢，可考虑：
  - 把解密输出直接通过 pipe 喂给 ffmpeg，避免中间 tmp 文件二次读写。
  - 对纯 MP3 保持当前 StashFile 逻辑（不解码）。
- [ ] **文件整理目标文件名**：当前自动生成 `序号_日期_时段.mp3`，如需更灵活的命名规则可扩展。
- [ ] **后端鉴权**：当前敏感接口无 session 鉴权，后续若开放局域网访问，应补 JWT/session 中间件。
- [ ] **压缩插件日志**：`vite-plugin-compression` 日志中会出现 `dist/D:/BCTools/...` 样式的绝对路径，但实际 gzip 文件仍在 `frontend/dist/assets/` 下，不影响功能。如介意可调整插件配置。
- [ ] **前端测试**：目前仅依赖 `type-check`，没有编写业务单元测试。

---

## 8. 常用调试

### 查看日志

```text
%APPDATA%\BroadcastTool\logs\app.log
```

### 重置高级设置密码

编辑 `%APPDATA%\BroadcastTool\data.json`，删除 `adminPasswordHash` 字段，重启程序。

### 端口冲突

程序会自动尝试 1743~1749。若全部被占用，修改 `main.go` 中 `findAvailablePort` 起始端口。

---

## 9. 交付清单

- `README.md`：面向用户的使用说明
- `HANDOVER.md`：本文档（面向开发者/Agent）
- `dist/broadcast-tool.exe`：最新构建的可执行文件
- `dist/ffmpeg.exe`、`dist/ffprobe.exe`：音频处理依赖
- 前端构建产物已嵌入 `backend/embed/dist/`
