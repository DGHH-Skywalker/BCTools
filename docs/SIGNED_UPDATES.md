# 签名更新

BCTools v6 使用 TUF 校验更新元数据和目标文件。更新内容可以是公告与模板资源、主界面、um-react、插件或 Windows 核心程序。

当前仓库的 `GoServer/internal/updates/source.json` 仍是占位配置，未包含生产信任根。未配置更新源时，设置页不会执行在线更新，用户应从 GitHub Releases 下载新版本。

## 发布链路

1. GitHub Actions 的 Build 工作流完成测试、Windows 构建、安装冒烟和校验值生成。
2. `scripts/package-update-feed.ps1` 将需要签名的内容整理为 `BCTools-UpdateTargets.zip`。
3. 维护者将解压后的 `targets/` 提交给更新仓库，由 [TUF-on-CI](https://github.com/theupdateframework/tuf-on-ci) 完成签名。
4. 更新仓库的发布分支提供 `metadata/` 和 `targets/`；镜像保持相同目录结构。
5. 客户端刷新 TUF 元数据，读取 `catalog.json`，按兼容规则选择更新，再下载并校验目标文件。

普通 GitHub Release 与签名更新仓库是两个发布步骤。只发布 Release 附件不会自动让已配置客户端发现更新。

## 配置构建来源

在 BCTools 仓库设置以下 Actions 变量，构建时由 `scripts/configure-update-source.cjs` 写入客户端：

| 变量 | 内容 |
| --- | --- |
| `BCTOOLS_UPDATE_REPO` | 更新仓库，格式为 `OWNER/REPO` |
| `BCTOOLS_TUF_ROOT_JSON` | 完整且已签名的 TUF 根元数据 JSON |
| `BCTOOLS_UPDATE_BRANCH` | 发布分支，默认 `update-feed` |
| `BCTOOLS_UPDATE_MIRRORS` | HTTPS 镜像基础地址组成的 JSON 数组 |

根私钥和更新日志私钥分开保管，不进入源码、Actions 日志或 Release 附件。根元数据轮换按 TUF-on-CI 的流程执行。

## 更新目录

`catalog.json` 本身也是 TUF 目标。每个更新项声明 ID、类型、版本、目标路径和 API 版本；核心程序还需声明平台、架构及兼容的存储版本。

当前客户端使用：

- `apiVersion: 1`；
- 核心程序 `storageVersion: 2`；
- `kind` 为 `resource`、`web`、`plugin` 或 `core`；
- `core` 仅接受 Windows 且版本高于当前程序。

每次发布使用新的版本号。不要原地替换同一版本的目标内容。

## 目标格式

### 资源

资源 JSON 可包含公告、模板和配置：

```json
{
  "announcement": "本周歌单发布通知",
  "templates": {"常规海报": "今晚的广播时间"},
  "config": {"themeColor": "#0086C3", "importXlsxEnabled": "true"}
}
```

### 网页

网页 ZIP 必须同时包含：

```text
main/index.html
um-react/index.html
```

以及两者引用的静态资源。客户端按内容摘要保存版本，新版本加载后需确认生效；未确认或检查失败时可回到上一网页版本。内嵌 um-react 的版本化资源由 BCTools 提供，独立运行 um-react 时仍由其自身 Service Worker 管理。

### 插件

插件 ZIP 包含 `manifest.json` 和可选的 `plugin.wasm`。清单声明 `apiVersion`、权限、页面、布局和命令。示例见 `GoServer/internal/plugins/example-layout.json`。

### Windows 核心程序

便携版使用普通核心目标；已安装版本需要同时提供 `installerTarget`，由 Inno Setup 完成替换。客户端先下载并验证，再通过重启交接安装。发布前应分别验证便携版和安装版的更新、失败退出与回滚路径。

## 镜像与发布顺序

客户端可以尝试直连来源和维护者提供的 HTTPS 镜像。无论文件来自哪里，都必须通过同一份 TUF 元数据中的长度与 SHA-256 校验。

同步镜像时：

1. 先复制 `targets/` 和带版本的元数据；
2. 确认大文件完整可下载；
3. 最后更新 timestamp 元数据。

较大的目标文件可只放在维护者镜像，避免 Git 托管的单文件限制。镜像的访问速度和中国大陆网络可用性需要在目标网络中实测，不能由签名通过推断。

## 发布检查

- 标签、应用版本和 Windows 文件版本一致；
- Release 中的 `SHA256SUMS.txt` 校验通过；
- 更新仓库中的目标与本次构建一致；
- 根元数据、镜像地址和分支配置可用；
- 网页更新能确认，也能恢复上一版本；
- 便携版与安装版核心更新分别验收；
- 更新失败时不破坏现有程序与 `BctoolData`。
