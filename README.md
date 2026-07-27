# 广播站歌单与音频整理工具

一个集成 **NCM 解密、音频转码、歌单管理、图片导出与文件整理** 的桌面小工具。

双击 `dist/broadcast-tool.exe` 即可启动，程序会自动在浏览器中打开 `http://localhost:1743/`。

---

## 功能特性

- **歌曲导入**：支持 `ncm / mp3 / flac / wav / mp4` 等格式；`ncm` 文件自动解密并转码为 `mp3`。
- **歌单管理**：宿舍歌单、播音歌单独立维护，支持按日期查看与编辑。
- **时段配置**：高级设置中可自定义一周 7 天的播放时段。
- **歌单导出**：支持单张 PNG 图片导出与多日期批量打包下载。
- **文件整理**：按序号将音频文件复制/移动到目标文件夹，缺失歌曲自动生成静音占位文件。
- **自动备份**：可配置自动备份路径，支持一键手动备份与快照恢复。
- **双语界面**：简体中文 / English 可切换。

---

## 快速开始

### 直接运行

1. 进入 `dist/` 文件夹。
2. 双击 `broadcast-tool.exe`。
3. 等待浏览器自动打开 `http://localhost:1743/` 即可使用。

> `dist/` 中已经包含 `ffmpeg.exe` 与 `ffprobe.exe`，无需额外配置环境变量。

### 数据与配置位置

所有数据保存在用户配置目录：

```text
%APPDATA%\BroadcastTool
```

主要文件：

| 文件/目录 | 说明 |
| --- | --- |
| `data.json` | 歌曲与设置数据 |
| `temp/` | 上传与转换临时文件 |
| `snapshots/` | 每日自动快照 |
| `logs/app.log` | 运行日志 |
| `bin/` | 程序自动复制的 ffmpeg 目录 |

---

## 开发构建

### 环境要求

- Go 1.22+
- Node.js 18+
- Windows（当前版本依赖 Windows 文件夹选择对话框）

### 前端

```bash
cd WebUI
npm install
npm run type-check
npm run build
```

### 后端

```bash
cd GoSever
go mod tidy
go vet ./...
go test ./...
set CGO_ENABLED=0
go build -ldflags="-s -w" -o broadcast-tool.exe .
```

### 完整打包

```bash
cd WebUI && npm run build
cd ..
rm -rf GoSever/embed/dist
mkdir -p GoSever/embed/dist
cp -r WebUI/dist/* GoSever/embed/dist/
cd GoSever
go build -ldflags="-s -w" -o broadcast-tool.exe .
cd ..
mkdir -p dist
cp GoSever/broadcast-tool.exe dist/
# 确保 dist/ 中已包含 ffmpeg.exe 与 ffprobe.exe
```

---

## 项目结构

```text
.
├── GoSever/              Go 后端
│   ├── converter/        音频转换、NCM 解密、静音生成
│   ├── embed/dist/       嵌入的前端构建产物
│   ├── handlers/         HTTP 接口处理
│   ├── middleware/       CORS / 日志 / 恢复
│   ├── models/           数据模型
│   ├── paths/            应用目录路径
│   ├── response/         统一响应封装
│   ├── routes/           路由注册
│   ├── store/            数据持久化与快照
│   ├── validation/       参数校验
│   └── main.go           程序入口
├── WebUI/                Vue 3 + TypeScript 前端
│   ├── src/
│   │   ├── api/          接口请求
│   │   ├── components/   组件
│   │   ├── locales/      中英双语
│   │   ├── router/       路由
│   │   ├── stores/       Pinia 状态
│   │   ├── views/        页面
│   │   ├── App.vue
│   │   └── main.ts
│   └── dist/             前端构建产物
├── dist/                 最终分发包（exe + ffmpeg + ffprobe）
└── README.md
```

---

## 接口说明

后端默认监听 `0.0.0.0:1743`~`1749`，可用第一个未被占用的端口。

主要 API：

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/health` | 健康检查 |
| GET | `/api/settings` | 获取设置 |
| PUT | `/api/settings` | 更新设置 |
| GET | `/api/songs` | 获取歌曲列表 |
| POST | `/api/songs` | 创建歌曲 |
| PUT | `/api/songs/{id}` | 更新歌曲 |
| DELETE | `/api/songs/{id}` | 删除歌曲 |
| POST | `/api/files/process` | 上传并转换音频 |
| POST | `/api/files/stash` | 上传 MP3 并提取元数据 |
| POST | `/api/files/organize` | 整理文件到目标目录 |
| POST | `/api/sync/backup` | 手动备份 |

---

## 注意事项

1. **NCM 解密**：仅用于学习交流，请支持正版音乐。
2. **ffmpeg 查找顺序**：程序启动时会按 `应用目录/bin/ > exe 同目录 > 系统 PATH` 查找 `ffmpeg.exe` 和 `ffprobe.exe`。
3. **自动打开浏览器**：启动 500ms 后会调用系统默认浏览器打开首页；若失败可在日志中查看原因。
4. **路由刷新**：前端为单页应用（SPA），直接刷新 `/song/import`、`/broadcast` 等页面不会 404。
5. **高级设置密码**：若忘记密码，可手动编辑 `%APPDATA%\BroadcastTool\data.json`，删除 `adminPasswordHash` 字段后重启程序。

---

## 常见问题

**Q：启动后浏览器没有自动打开？**  
A：请检查是否有安全软件拦截，或手动访问 `http://localhost:1743/`。

**Q：NCM 转换卡住？**  
A：后端已对解密流程设置 60 秒超时保护，超时会返回错误；如频繁超时，请检查网络或更换 NCM 文件。

**Q：播音歌单输入框无法编辑？**  
A：当前版本已改为本地编辑 + 失焦自动保存，若仍有问题请刷新页面。

---

## 版本历史

### v5.5.0.0（当前）

- 所有页面标题统一为“江西拙楷 + 主题色”风格。
- 侧边栏菜单重命名：`歌曲导入` → `宿舍点歌`，`宿舍点歌` → `宿舍歌单`。
- 换卡工具改为按周多选卡片，复用歌单导出“简约模式”表格预览，并从缓存目录复制重命名文件到 SD 卡目录。
- 新增 `PageTitle` 组件与 `playlistTable` 工具，提升代码复用。
- 全项目版本号统一升级为 `v5.5.0.0`。

---

## License

仅供个人学习与交流使用。
