# BCTools HTTP API

当前接口以 `/api/v1` 为开发入口。完整请求、响应和错误结构以 [OpenAPI 契约](../internal/contracts/openapi.json) 为准；运行中的实例可通过以下地址读取同一份契约：

```text
http://localhost:1743/api/v1/openapi.json
```

前端使用相对路径，不写死主机名或端口。`/api` 保留原有兼容路由，新能力只注册在 `/api/v1`。

## 路由范围

### `/api` 与 `/api/v1` 共用

| 分组 | 内容 |
| --- | --- |
| `/health`、`/lifecycle/*`、`/shutdown` | 健康检查与进程生命周期 |
| `/songs/*` | 宿舍、播音歌曲的查询、创建、更新、排序、导入和导出 |
| `/files/*` | 音频处理、暂存、周文件导出、试听、合并与目录选择 |
| `/settings`、`/auth/*` | 设置与高级设置密码验证 |
| `/snapshots/*` | 快照列表与恢复 |
| `/check-update`、`/update-log/*` | 兼容更新检查与更新日志展示 |
| `/system/status`、`/network/*` | 系统状态、局域网地址与二维码 |
| `/decrypt/stage/*` | um-react 解密文件的暂存、回传和清理 |

### 仅 `/api/v1`

| 分组 | 内容 |
| --- | --- |
| `/openapi.json` | 当前 OpenAPI 契约 |
| `/retention/*` | 保留期设置、立即清理与回收批次恢复 |
| `/plugins/*` | 插件查询、安装、启停、布局切换、卸载和 WASM 命令 |
| `/updates/*` | 签名更新状态、检查、应用、网页确认与回滚、核心程序重启 |
| `/decrypt/kugou/*` | 酷狗数据库状态、手动选择、恢复自动查找与取钥 |

桌面模式另提供 `/api/desktop/window` 和 `/api/desktop/save`，用于窗口管理与原生保存对话框。它们由桌面运行时处理，不属于 LAN 通用接口。

## 访问边界

- 旧 `/api` 及共用核心路由保留原有 LAN 行为，没有完整的会话鉴权。能访问端口的设备可能调用管理功能，只允许在可信网络中运行。
- 保留期设置与清理、插件写操作和命令、更新检查与应用、酷狗数据库配置等新增管理操作仅接受本机请求。
- `/shutdown`、`/close-browser` 和桌面专用操作有本机限制。
- 插件和更新的部分只读状态接口可用于页面展示，不代表对应写操作向 LAN 开放。

新增接口时先确定调用方是浏览器、本机桌面窗口还是局域网设备，再在 `GoServer/routes/routes.go` 中选择相应的访问控制。

## 契约与生成类型

契约源文件：

```text
GoServer/internal/contracts/openapi.json
```

修改契约后，从仓库根目录运行：

```bash
npm run generate:api
```

生成并提交：

- `GoServer/internal/contracts/types.gen.go`
- `Web/src/api/generated/contracts.ts`

不要手工修改生成文件。CI 会重新生成并检查差异。

## 业务约定

OpenAPI 描述数据格式，以下约束还需要结合后端实现与测试理解：

- 歌曲 ID 由服务端分配，批量导入和并发新增不能复用同一 ID。
- 覆盖导入先创建独立快照；备份失败时取消导入。
- 歌曲记录按是否有对应音频拆分存储，客户端不应假设 `data.json` 包含完整歌单。
- 周文件导出按时段位置编号。空时段生成静音占位，同一时段多首歌在导出时合并。
- um-react 回传优先使用暂存 ID 就地导入；下载暂存文件的接口保留给需要原始字节的场景。
- 保留期清理产生可恢复批次，恢复接口只接受有效且未过期的回收记录。
- 插件与更新接口只处理通过格式、权限、签名和完整性校验的内容。

实现入口见 `GoServer/routes/routes.go`、`GoServer/handlers/` 和 `GoServer/services/`。
