# 构建工具与排障

本文记录 Windows 构建所需的工具版本和可复现的排障方法。具体路径以本机安装位置为准，不要把某一台电脑上的盘符或用户目录写进构建脚本。

## 工具版本

| 工具 | 项目使用版本 | 检查命令 |
| --- | --- | --- |
| Node.js | 24，CI 为 24.7.0 | `node --version` |
| npm | Node.js 自带 | `npm --version` |
| pnpm | 10.15.1 | `corepack pnpm@10.15.1 --version` |
| Go | 1.26.11 | `go version` |
| Git | 当前稳定版 | `git --version` |
| Inno Setup | 6 | 检查 `ISCC.exe`，或运行 `npm run build:installer` |
| MSYS2 UCRT64 | 当前版本 | 仅重新构建 FFmpeg 时需要 |

安装包已经改用 Inno Setup。构建主程序和安装包不需要 MinGW；`installer/src/` 中的旧 C++ 安装器不参与当前构建。

## 依赖安装

在仓库根目录执行：

```powershell
npm ci
npm --prefix Web ci --legacy-peer-deps
corepack pnpm@10.15.1 --dir Web/um-react install --frozen-lockfile
```

网络环境需要镜像时，可分别为 npm、pnpm 和 Go 配置镜像。Go 校验数据库保持默认开启，不要为了绕过网络问题关闭 `GOSUMDB`。

```powershell
npm config set registry https://registry.npmmirror.com
corepack pnpm@10.15.1 config set registry https://registry.npmmirror.com
go env -w GOPROXY=https://goproxy.cn,direct
```

镜像只影响依赖下载，不改变锁文件要求的版本和完整性校验。

## 构建入口

```powershell
npm run dev
npm run test
npm run type-check
npm run build
```

完整构建前，确认以下文件存在：

```text
GoServer/ffmpeg_bctools_mini/dist/ffmpeg.exe
GoServer/ffmpeg_bctools_mini/dist/ffprobe.exe
```

缺少它们时，`build:inject-ffmpeg` 会失败。可按 [FFmpeg 构建说明](GoServer/ffmpeg_bctools_mini/README.md) 重新构建。

## 常见问题

### pnpm 安装后依赖版本异常

um-react 在 `package.json` 中使用 `pnpm.overrides` 和补丁。请通过 Corepack 固定到 pnpm 10.15.1，不要换成其他主版本。

```powershell
Remove-Item Web/um-react/node_modules -Recurse -Force
corepack pnpm@10.15.1 --dir Web/um-react install --frozen-lockfile
```

如果之前从另一台电脑复制过 `node_modules`，先删除再安装。依赖目录包含本机路径和平台二进制，不适合直接迁移。

### `@swc/core` 或 top-level-await 构建报错

um-react 当前通过 pnpm override 固定 `@swc/core`，并将 Vite 构建目标设为 `es2022`。升级 Vite、`vite-plugin-top-level-await` 或 SWC 时，应一起检查这两项配置，不要只删除锁文件重试。

### `simple-git-hooks` 提示找不到 Git

安装 Git 并确保 `git.exe` 在 `PATH` 中，然后重新运行 pnpm 安装。该步骤来自 um-react 的 `prepare` 脚本。

### 找不到 `ISCC.exe`

安装 Inno Setup 6。使用非默认安装位置时，设置 `ISCC_PATH` 为 `ISCC.exe` 的完整路径：

```powershell
$env:ISCC_PATH = 'D:\Tools\Inno Setup 6\ISCC.exe'
npm run build:installer
```

### 桌面模式提示缺少 WebView2

安装 Microsoft Edge WebView2 Runtime，或直接使用无参数的浏览器模式：

```powershell
.\bctools.exe
```

### 1743 端口被占用

浏览器模式需要监听 `1743`。先尝试正常交接或关闭现有实例：

```powershell
.\bctools.exe --shutdown
```

桌面模式在端口不可用时仍可打开本机窗口，但 LAN 访问不可用。排查占用进程前，不要直接结束正在写数据的其他 BCTools 实例。

### 测试提示缺少 Go 嵌入文件

根目录的 `test` 和 `type-check` 会自动准备 Go embed 占位文件。单独运行 Go 工具前，可先执行：

```powershell
node scripts/prepare-go-embed-stubs.cjs
go -C GoServer test ./...
```

占位文件只用于测试和检查，不是可发布的前端产物。

## FFmpeg 源码包

日常应用构建只复制已有 FFmpeg 产物。需要重编译时，在 MSYS2 UCRT64 中运行：

```bash
cd GoServer/ffmpeg_bctools_mini
bash build.sh
```

发布工作流设置 `BCTOOLS_PACKAGE_FFMPEG_SOURCE=1`，同时生成 `FFmpeg-Sources.tar.gz`。分发二进制时应一并保留对应源码包和许可证。

## 提交前检查

```powershell
npm run type-check
npm run test
npm run lint
corepack pnpm@10.15.1 --dir Web/um-react exec tsc -p tsconfig.prod.json --noEmit
corepack pnpm@10.15.1 --dir Web/um-react test
```

修改 API 契约时再运行 `npm run generate:api`，并提交两个生成类型文件。构建产物、依赖目录、运行数据和本地密钥不进入 Git。
