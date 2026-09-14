# 本机工具链备忘（Windows / 2026-08）

仓库 `E:\BCTools3` 现在能在一台**裸 Windows** 上用 `npm run build` 端到端出 `dist/bctools.exe` + `dist/BCTools-Setup.exe`。本文件记录装在哪、用什么镜像、踩过的坑与对应处理。

## 1. 工具链位置

| 工具 | 位置 | 来源 | 备注 |
|------|------|------|------|
| Node.js 24.19 | `C:\Program Files\nodejs\` | 预装 | — |
| npm | 同上 | 预装 | 镜像见下 |
| pnpm 10.15.1 | `C:\Users\Administrator\AppData\Roaming\npm\pnpm.ps1` | `npm i -g pnpm@10.15.1` | **必须 10.x**；11+ 不读 `package.json#pnpm.overrides`（警告：`The "pnpm" field in package.json is no longer read`），会让 `um-react` 构建时拿到新版破坏性依赖 |
| Python 3.7.6 | `C:\Users\Administrator\anaconda3\` | 预装 | — |
| Go 1.25.0 | `E:\Tools\go\` | `https://golang.google.cn/dl/go1.25.0.windows-amd64.zip` | 写 `GOROOT=E:\Tools\go`、`GOPATH=E:\Tools\go-gopath` |
| MinGW-w64 14.2 (UCRT, posix-seh) | `E:\Tools\mingw64\` | `https://ghfast.top/https://github.com/niXman/mingw-builds-binaries/releases/download/14.2.0-rt_v12-rev2/x86_64-14.2.0-release-posix-seh-ucrt-rt_v12-rev2.7z` | 仅 `BCTools-Setup.exe` 用，`g++` + `mingw32-make` 即可 |
| 7-Zip 解压器 | `E:\Tools\7zr.exe` | `https://www.7-zip.org/a/7zr.exe` | 仅用于解压上面的 .7z；PowerShell 5.1 不带 7z |
| MinGit 2.50 | `E:\Tools\mingit\cmd\git.exe` | `https://npmmirror.com/mirrors/git-for-windows/v2.50.1.windows.1/MinGit-2.50.1-64-bit.zip` | 给 pnpm `simple-git-hooks` + Vite 用，`simple-git-hooks prepare` 没有 git 会报 `'git' is not recognized` |
| ffmpeg 精简版 | `E:\BCTools3\GoServer\ffmpeg_bctools_mini\dist\` | 仓库自带 | 不用动 |

> `E:\Tools\go\bin;E:\Tools\mingw64\bin;E:\Tools\mingit\cmd` 三个目录已加入 **User** 环境 `PATH`（`[Environment]::SetEnvironmentVariable('PATH', ..., 'User')`），新 shell 自动可用。

## 2. 镜像源

| 工具 | 源 | 配置 |
|------|----|------|
| npm | `https://registry.npmmirror.com` | `npm config get registry` 默认就是这台机器上 |
| pnpm | `https://registry.npmmirror.com` | `pnpm config set registry https://registry.npmmirror.com` |
| Go module | `https://goproxy.cn,direct` | `GOPROXY=https://goproxy.cn,direct`、`GOSUMDB=off`（`go env -w` 持久化） |
| MinGW 镜像 | `https://ghfast.top/https://github.com/...` | ghfast.top 是 GitHub 加速；直连 github.com 在国内慢 |
| 7-Zip | `https://www.7-zip.org/a/7zr.exe` | 主源，国外但文件小；不行就 `cnblogs` / 阿里云的镜像源搜 `7zr.exe` |

## 3. 关键环境变量（已持久化到 User）

```text
PATH = E:\Tools\go\bin;E:\Tools\mingw64\bin;E:\Tools\mingit\cmd;<原 PATH>
GOROOT = E:\Tools\go
GOPATH = E:\Tools\go-gopath
GO111MODULE = on
GOPROXY = https://goproxy.cn,direct
GOSUMDB = off
```

新 PowerShell 窗口 / VS Code 终端开起来就有；当前会话跑下面这条强制刷新：

```powershell
$env:PATH = [System.Environment]::GetEnvironmentVariable('PATH', 'User')
```

## 4. 端到端构建（裸机步骤）

```powershell
# 一次性：装好 pnpm 10、Go 1.25、MinGW、MinGit
npm i -g pnpm@10.15.1

cd E:\BCTools3
npm install                                     # 根 + 子项目 deps（npmmirror 镜像）
Set-Location Web\um-react
pnpm install                                    # pnpm 10，会读 package.json#pnpm.overrides
Set-Location E:\BCTools3
npm run build                                   # 完整链路
```

`npm run build` 顺序：

1. `build:web` — Vite 构建主前端 + 字体子集化
2. `build:copy` — 复制 `Web/dist` → `GoServer/embed/dist`
3. `build:um-react` — Vite 构建 Unlock Music → `GoServer/embed/um-react`
4. `build:inject-ffmpeg` — 复制精简版 ffmpeg/ffprobe → `GoServer/internal/binembed/bin`
5. `build:server` — `go build` 出 `dist/bctools.exe`（嵌入前端 + um-react + ffmpeg）
6. `build:installer` — MinGW 编译安装程序 → `dist/BCTools-Setup.exe`

## 5. 已踩的坑与修复

### 5.1 Go 缺失 / `go.mod` 1.25.0

`go.mod` 写 `go 1.25.0`，没装 Go 直接 `go build` 报"未知命令"。

- **修**：从 `https://golang.google.cn/dl/`（Google 国内 CDN）拉 `go1.25.0.windows-amd64.zip`，解压到 `E:\Tools\go`。

### 5.2 Go 编译报 `possible misuse of unsafe.Pointer`

`go vet` 4 处 `unsafe.Pointer(uintptr)` 警告（`handlers/selectdir_windows.go` 里 4 行 COM vtable 偏移读写）。`go build` 默认 vet 集合不含 `unsafeptr`，**不影响 `go build`**；只有 `go vet ./...` 与 `npm run type-check` 会报。不修。

### 5.3 um-react 用 pnpm 11 装但 shim 写死旧路径

项目里残留的 `Web\um-react\node_modules` 是从 `D:\BCTools\...` 拷来的（`.bin\tsc.CMD` 里的 `NODE_PATH=...D:\BCTools\...` 露馅）。同时 pnpm 11 已不再读 `package.json#pnpm.overrides`，配合上 `packageManager: pnpm@10.15.1` 等于同时坏两处。

- **修**：
  1. `npm i -g pnpm@10.15.1` 装回 10.15.1
  2. `Remove-Item Web\um-react\node_modules -Recurse -Force` + `pnpm install` 干净重装
  3. 之后 `.bin\tsc.CMD` 里 `NODE_PATH` 自动改回 `E:\BCTools3\...`

### 5.4 `vite-plugin-top-level-await` 与 `@swc/core@1.16.1` 不兼容

um-react 用 `vite-plugin-top-level-await@1.6.0` 把 TLA 包成 `Promise.all([...]).then(...)`，内部走 `@swc/core` 的 `parseSync` / `printSync`。SWC 1.16 引入更严格的 `Module` 节点检查，插件的 `transformModule` 产出的 AST 节点会被报 `missing field 'type'`。

- **修**：在 `Web/um-react/package.json#pnpm.overrides` 加 `"@swc/core": "1.10.7"`，重装后 `node_modules\.pnpm\@swc+core@1.10.7\` 命中。

### 5.5 `vite-plugin-top-level-await` 与 esbuild 0.28（Vite 7 自带）不兼容

锁住 `@swc/core` 后又报错 `Transforming destructuring to the configured target environment ("chrome87", "edge88", "es2020", "firefox78", "safari14") is not supported yet`。原因：插件在 `generateBundle` 后用 vite 自带的 esbuild 把代码从 esnext 降回 `build.target`，esbuild 0.28 砍掉了 chrome87 等老目标。

- **修**：`Web/um-react/vite.config.ts` 的 `build` 加 `target: 'es2022'`。插件会把它存下来当后处理目标，0.28 仍兼容。

### 5.6 `simple-git-hooks prepare` 报 `'git' is not recognized`

`Web/um-react` 的 `prepare` 跑 `simple-git-hooks`，里头要 git；`pnpm install` 装完会跑一次。

- **修**：装 MinGit 2.50.1（轻量、~47 MB）到 `E:\Tools\mingit\cmd\` 并入 PATH。

### 5.7 MinGW 7z 解压

PowerShell 5.1 没 7z，且 `py7zr` 装不上（py7zr 依赖的 `inflate64` / `pyppmd` 旧版被国内镜像砍了）。

- **修**：直接下 `https://www.7-zip.org/a/7zr.exe`（~600 KB 单文件）当 7z 解压器。

## 6. 验证产物

```powershell
Get-ChildItem E:\BCTools3\dist
# Name                Length
# BCTools-Setup.exe   58536187   (57 MB 安装程序)
# bctools.exe         45730304   (45 MB 主程序，含 ffmpeg+ffprobe+前端+um-react)

# 主程序能起来（PowerShell 主机没 V2 DPI，会走 fallback；正常 Win10/11 桌面无此提示）
& E:\BCTools3\dist\bctools.exe --help
# Usage of E:\BCTools3\dist\bctools.exe:
#   -dev      开发模式：替换已有后端
```

## 7. 后续维护

- `um-react` 升级时若 vue-tsc 报错，先看 `Web\um-react\.bin\tsc.CMD` 的 `NODE_PATH` 是不是还指向本机 `E:\BCTools3\...`，是的话直接清 `node_modules` 重装。
- 装新版 Vite（>= 8）时如果 esbuild 跨过 0.29 砍更多目标，再调 `Web/um-react/vite.config.ts#build.target` 即可。
- Go 升 1.26+ 时同步更新 `go.mod` 第一行 `go 1.x.0`。
