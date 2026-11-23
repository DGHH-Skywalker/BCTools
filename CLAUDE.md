# BCTools Agent 工作指南

修改前先读 [DEVELOPER.md](DEVELOPER.md)。README 面向使用者，开发指南说明当前架构；`docs/releases/` 和 `.claude/plans/` 只记录历史版本，不能作为现状依据。

## 常用入口

| 目标 | 位置 |
| --- | --- |
| 进程启动与依赖组装 | `GoServer/main.go` |
| 桌面窗口与桌面专用接口 | `GoServer/desktop_windows.go` |
| 单实例与浏览器/桌面交接 | `GoServer/launcher/` |
| 路由与本机访问控制 | `GoServer/routes/routes.go`、`GoServer/internal/securelocal/` |
| HTTP handler | `GoServer/handlers/` |
| 业务编排与旧版迁移 | `GoServer/services/` |
| 拆分存储、事务、保留期 | `GoServer/internal/repo/` |
| OpenAPI 契约 | `GoServer/internal/contracts/openapi.json` |
| 插件与 WASM | `GoServer/internal/plugins/` |
| TUF 更新 | `GoServer/internal/updates/` |
| 酷狗数据库 | `GoServer/internal/kugoudb/` |
| 前端入口与路由 | `Web/src/main.ts`、`Web/src/router/index.ts` |
| 根构建入口 | `package.json` |
| 当前安装脚本 | `installer/BCTools.iss` |

um-react 源码直接位于 `Web/um-react/`。旧 C++ 安装器仅保留在 `installer/src/` 作对照，不参与构建。

## 开发命令

```bash
npm run dev
npm run type-check
npm run test
npm run lint
npm run build
```

um-react 使用独立的 pnpm 工具链：

```bash
corepack pnpm@10.15.1 --dir Web/um-react exec tsc -p tsconfig.prod.json --noEmit
corepack pnpm@10.15.1 --dir Web/um-react test
```

修改 `GoServer/internal/contracts/openapi.json` 后运行 `npm run generate:api`，同时提交 Go 和 TypeScript 生成文件。不要手工编辑生成文件来绕过契约。

## 架构约束

### 进程与请求

- 一个数据目录只由一个后端实例持有。浏览器与桌面模式通过 `launcher` 交接，不能并行写同一目录。
- 浏览器模式监听 `1743`；桌面模式使用进程内 `http.Handler`，并尽量保留 LAN 监听。
- `/api` 保留兼容接口；`/api/v1` 复用核心接口，并增加 OpenAPI、保留期、插件、酷狗和签名更新。
- 新增管理操作应检查本机请求边界。旧 LAN 接口仍缺乏完整鉴权，不要把敏感能力继续暴露给局域网。
- 前端 API 使用相对路径，不写死 `localhost:1743`。

### 数据与文件

- 正式数据目录是 `<程序目录>/BctoolData`；`go run` 默认是 `GoServer/BctoolData`。
- 当前存储版本为 2。`data.json` 是存储头和索引，歌曲记录分布在 `songs-without-audio.json` 与周目录的 `songs.json`。
- 所有写操作通过 `internal/repo` 的事务布局提交。未完成事务由下一次启动重放，不要绕过仓储层手工拼写多个文件。
- 歌曲 ID 由仓储层统一分配。覆盖导入必须先创建独立快照，备份失败则取消。
- 保留期清理先进入 `recycle/`，恢复时保留路径边界校验。
- 修改数据布局时同步考虑旧版迁移、备份、恢复、降级说明和发布说明。

### 音频

- MP3 输入直接搬运，其他格式才重编码，避免二次有损。
- NCM 与酷狗加密格式走后端解密；酷狗数据库使用验证后的本机副本。
- um-react 回传通过暂存 ID 就地导入，避免同一音频在本机 HTTP 上重复传输。
- SD 卡编号按时段位置分配。空时段生成 17.43 秒静音占位，同时段多首歌按顺序合并。
- 精简 FFmpeg 没有完整 concat 组件，MP3 合并使用帧拼接和 ID3 处理。修改前阅读 `converter/` 与相关测试。
- 合并文件的单曲试听依赖原始文件区间和 Range/Seek，不要改成一次性加载整文件。

### 前端

- 新页面在 `views/` 创建并注册路由；可复用交互放入 `components/` 或 `composables/`。
- 日期统一使用 `utils/datetime.ts` 中配置好的 dayjs。
- 本地持久化统一走 `utils/persist.ts`，保留旧 key 的兼容读取。
- 中文分词词典保持动态加载，避免重新进入公共首屏包。
- 修改界面文案时同步检查 `Web/src/locales/zh-CN.json` 与 `en.json`，不写无法保证的测试结论或宣传语。

### 插件与更新

- 插件包只接受规定的清单和可选 WASM 文件，权限、内存、执行时间和返回大小均有限制。
- 插件通过宿主能力访问数据，不直接读写宿主存储。
- 更新必须通过 TUF 元数据和 SHA-256 校验，镜像切换不改变信任要求。
- 网页更新保留确认与回滚路径；核心更新区分便携版和已安装版。
- 私钥、签名种子和本地更新配置不得写入仓库、日志或 Release 附件。

### 安装与发布

- 当前安装器是 Inno Setup 6，根构建不编译旧 C++ 安装器。
- 升级和普通卸载默认保留 `BctoolData`；静默删除数据必须显式传入 `/REMOVEUSERDATA`。
- 发布前同步根/Web 清单与锁文件、Go 版本常量和 `versioninfo.json`。
- 标签必须等于 `v` + 根 `package.json` 版本，Release 正文来自对应的 `docs/releases/` 文件。

## 文档改动

- 当前架构、命令和数据布局变化时，同步更新 README、DEVELOPER 和相关专项文档。
- 历史 Release 说明保留原始语境；如需补充说明，明确标注记录版本。
- 文档中的命令应能从仓库根目录执行，链接使用仓库内真实路径。
- 删除过时说明，不保留互相矛盾的“新旧两套现状”。历史背景放入版本记录。
