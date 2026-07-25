# 更新日志 / Changelog

> 本文件记录 **广播站歌单与音频整理工具（BroadcastTool）** 的版本迭代历史与后续发布约定。

---

## 版本命名规则

本项目采用 **四段式版本号**：

```text
v{主版本}.{次版本}.{修订号}.{构建号}
```

示例：`v5.0.0.1`

| 段位 | 名称 | 递增场景 |
| --- | --- | --- |
| 第 1 段 | 主版本（Major） | 架构重构、不兼容变更、重大功能改版 |
| 第 2 段 | 次版本（Minor） | 新增功能模块、较大交互调整 |
| 第 3 段 | 修订号（Patch） | Bug 修复、性能优化、细节改进 |
| 第 4 段 | 构建号（Build） | 同一修订下的打包迭代、配置更新、文档补齐 |

### 迭代示例

- `v5.0.0.1` → `v5.0.0.2`：同 Patch 下的构建迭代
- `v5.0.0.2` → `v5.0.1.0`：修复若干 Bug 后发布新修订
- `v5.0.1.0` → `v5.1.0.0`：新增功能模块
- `v5.1.0.0` → `v6.0.0.0`：重大架构升级或不兼容变更

---

## 当前版本

### v5.0.0.1（当前）

- **统一全项目版本标识**：将代码、构建脚本、Windows 资源文件、API 文档中的版本号统一为 `v5.0.0.1`。
- **建立 Git 版本管理**：初始化 Git 仓库，配置 `.gitignore`，纳入源码与文档的版本控制。
- **新增 `update.md`**：建立更新日志与版本命名规范，便于后续迭代追溯。

> 注：此前项目中存在多个版本号不一致的情况（如 `v1.0.0`、`v2.0`、`v2.1` 等），自 `v5.0.0.1` 起统一按本规范维护。

---

## 历史版本（据现有文档整理）

> 以下为根据 `README.md` 与 `HANDOVER.md` 整理的已知版本信息，可能不完整，仅供参考。

### v2.1

- 网站图标使用 `logo.png` 作为 favicon。
- 程序图标使用 `icon.ico` + `resource.syso` 嵌入 exe。
- 隐藏控制台窗口（`-H windowsgui`）。
- 关闭页面时前端调用 `/api/shutdown`，后端自动保存并退出。
- `dist/` 打包：`broadcast-tool.exe` + `ffmpeg.exe` + `ffprobe.exe`。

### v2.0

- 构建脚本 `build.py` 版本标识为 `v2.0`。
- 一键构建前后端并输出单文件 exe。

### v1.0.0

- 项目初始版本号，曾用于：
  - 后端启动日志
  - Windows 可执行文件版本信息（`backend/versioninfo.json`）
  - 前端 `package.json`
  - 默认设置字段 `version`
  - API 文档示例

---

## 待办 / 后续方向

详见 `HANDOVER.md` 第 7 节「当前已知问题与 TODO」。主要方向包括：

- [ ] 后端敏感接口鉴权（JWT / Session）
- [ ] NCM 解密真实文件实测与方案优化
- [ ] 转换流程性能优化（pipe、减少临时文件）
- [ ] 文件整理目标文件名规则扩展
- [ ] 前端业务单元测试补充

---

## 维护建议

1. **每次发布前**：
   - 确认 `backend/main.go`、`backend/versioninfo.json`、`frontend/package.json`、
     `frontend/package-lock.json`、`backend/store/models.go`、
     `frontend/src/stores/settings.ts`、`contracts/api.md` 中的版本号一致。
   - 在 `update.md` 顶部新增版本条目。
   - 使用 `python build.py` 重新构建并校验 `dist/broadcast-tool.exe`。

2. **提交规范**：
   - 每次文件修改前执行 `git add -A && git commit -m "checkpoint: <简述>"`。
   - 禁止 `git push --force` 到 `main` / `master`。
   - 修改完成后使用 `git diff HEAD~1` 向协作者展示变更摘要。

---

*最后更新：2026-07-25*
