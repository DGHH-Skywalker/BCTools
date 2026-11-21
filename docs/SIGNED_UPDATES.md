# BCTools 签名更新发布

GitHub Actions 的 Build 流程在 Windows 上测试、编译和检查安装包。Release 流程复用同一个工作流，从版本标签生成 Release，并上传 EXE、安装包、校验值及 `BCTools-UpdateTargets.zip`。`v6.0.0-beta1` 使用预发行标记，Windows 数字文件版本从 `versioninfo.json` 读取。

## 配置源

在 BCTools 仓库配置 Actions 变量。生产客户端信任的根需要先由维护者生成和签名。根 JSON 包含公钥及签名，私钥由发布者管理。

| 变量 | 用途 |
| --- | --- |
| `BCTOOLS_UPDATE_REPO` | 更新仓库，格式 `OWNER/REPO` |
| `BCTOOLS_TUF_ROOT_JSON` | 完整、已签名的 TUF 根元数据 |
| `BCTOOLS_UPDATE_BRANCH` | 发布分支，默认为 `update-feed` |
| `BCTOOLS_UPDATE_MIRRORS` | 维护者 HTTPS 镜像基础地址的 JSON 数组 |

以 [TUF-on-CI](https://github.com/theupdateframework/tuf-on-ci) 配置更新仓库，将构建产物解压后的 `targets/` 提交给它签名。发布分支应提供 `metadata/` 和 `targets/`。镜像复制同样的目录结构，先同步文件和带版本元数据，最后同步 timestamp。为了避免 Git 单文件大小上限，可以由维护者在镜像端提供较大的 targets 文件。

`catalog.json` 同样是 TUF 目标。首版接受 API 版本 1 的网页和插件，EXE 更新还需兼容数据格式 2。每次发布使用新版本名。公告与模板资源是 JSON，示例格式如下。

```json
{
  "announcement": "本周歌单发布通知",
  "templates": {"常规海报": "今晚的广播时间"},
  "config": {"themeColor": "#0086C3", "importXlsxEnabled": "true"}
}
```

Web 包同时提供 `main/index.html`、`um-react/index.html` 及对应静态资源。客户端以不同 URL 发布新旧版本，当前版本加载后确认生效。内嵌音乐解锁由 BCTools 提供版本化资源，独立运行的 um-react 仍使用自身的 Service Worker。签名插件包格式见 `GoServer/internal/plugins/example-layout.json`，包含 `manifest.json` 和可选的 `plugin.wasm`。

直连和公共代理用于获取公开内容。镜像的实际覆盖和下载速度需在目标中国大陆网络验收。客户端依据同一份 TUF 元数据校验来自各源的文件。
