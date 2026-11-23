# Git 文件范围

仓库保存从源码构建、测试和发布所需的文件，不保存本机运行状态。

## 应提交

- 应用源码：`GoServer/`、`Web/src/`、`Web/um-react/` 和安装脚本；
- 必需静态资源、测试样本、脚本、文档与 CI 工作流；
- 依赖清单及锁文件：`package.json`、`package-lock.json`、`go.mod`、`go.sum`、`pnpm-lock.yaml`；
- 可共享的编辑器配置与不含凭据的环境变量模板；
- OpenAPI 契约及其受控生成文件：`GoServer/internal/contracts/types.gen.go`、`Web/src/api/generated/contracts.ts`。

生成契约文件通过 `npm run generate:api` 更新，CI 会检查它们与契约是否一致。其他可重新生成的构建产物不进入 Git。

## 不提交

- `node_modules/`、包管理器缓存和本地依赖副本；
- `dist/`、前端嵌入产物、安装器中间文件、可执行文件、压缩包、覆盖率和测试报告；
- `BctoolData/`、歌曲与音频缓存、日志、临时文件、性能分析输出；
- `.env`、本地更新源配置、证书、签名私钥、更新日志私钥和其他凭据；
- 带个人路径或本机密钥的 IDE 配置。

提交前检查 `git status` 和暂存差异。发布附件由工作流生成，不需要把二进制提交到源码分支。
