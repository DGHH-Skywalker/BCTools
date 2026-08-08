# 小播点歌工具（BCTools）

面向校园广播站的本地点歌与歌单管理工具。

## 技术栈

- **前端**：Vue 3 + TypeScript + Vite + naive-ui + Pinia + vue-router（hash 模式）
- **后端**：Go 1.22+ + chi/v5 + embed.FS
- **工具链**：ffmpeg/ffprobe（精简版内嵌）、modern-screenshot、JSZip
- **安装程序**：C++ Win32/GDI+ 原生安装/卸载程序（`installer/`）

## 目录结构

```
BCTools/
├── GoServer/           # Go 后端源码
│   ├── handlers/       # HTTP 接口处理
│   ├── routes/         # 路由注册
│   ├── services/       # 业务逻辑服务层
│   ├── store/          # 数据访问层
│   ├── internal/       # 内部工具（repo、version、binembed）
│   ├── converter/      # 音频转码/NCM 解密
│   ├── launcher/       # 单实例/进程管理
│   └── main.go         # 后端入口
├── Web/                # Vue 3 前端源码
│   ├── src/
│   │   ├── views/      # 页面
│   │   ├── components/ # 组件
│   │   ├── stores/     # Pinia 状态
│   │   ├── composables/# 组合式函数
│   │   ├── api/        # API 客户端
│   │   ├── styles/     # 全局样式与 Design Token
│   │   └── constants/  # 颜色/间距/字体/布局常量
│   └── um-react/       # Unlock Music 子模块（Git submodule）
├── installer/          # C++ Win32/GDI+ 安装程序
├── dist/               # 构建产物输出目录
├── docs/               # 交接与设计文档
└── build.py            # 一键构建脚本
```

## 构建要求

- Go 1.22+
- Node.js 18+ + npm
- Windows（目标运行平台）
- MinGW-w64（用于编译安装程序）

## 常用命令

```bash
# 完整构建（前端 + um-react + 后端 + 安装程序）
python build.py

# 跳过部分步骤
python build.py --skip-front
python build.py --skip-back
python build.py --skip-um-react

# 清理构建产物
python build.py --clean

# 前端开发
npm run dev
npm run type-check
npm run test
npm run lint
npm run format

# 后端开发（在 GoServer/ 目录下）
go test ./...
go vet ./...
```

## 运行方式

构建完成后，产物位于 `dist/`：

- `dist/bctools.exe`：主程序（单文件 exe，已内嵌 ffmpeg/ffprobe + 前端）
- `dist/bctool_dev.exe`：开发模式启动器
- `dist/BCTools-Setup.exe`：安装程序（构建安装程序时需要 MinGW-w64）

双击 `bctools.exe` 后会自动打开浏览器访问 `http://localhost:1743/`。

## 数据目录

应用运行时会使用 `%APPDATA%\BroadcastTool\`：

```
%APPDATA%\BroadcastTool\
├── data.json              # 主数据文件
├── deleted_songs_log.json # 删除日志
├── songs\                 # 持久化歌曲文件
├── temp\                  # 临时目录（启动时清空）
├── snapshots\             # 每日快照备份
├── decrypt-staging\      # um-react 解密暂存
├── logs\                  # 滚动日志
└── bin\                   # 运行时释放的 ffmpeg/ffprobe
```

## 开发约定

- 前端样式优先使用 `Web/src/styles/variables.css` 中的 CSS 变量和 `Web/src/constants/` 中的 Design Token。
- 后端 handler 只负责 HTTP 编排，复杂业务逻辑下沉到 `GoServer/services/`。
- 端口固定 `1743`，单实例运行；重复启动会被拦截。
- 无后端 session 鉴权，API 面向可信局域网开放。

## 许可证

本项目为闭源内部工具，商用字体与资源需自行确认授权。
