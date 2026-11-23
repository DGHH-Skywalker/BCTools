# 更新日志签名

更新日志用于向用户展示本次版本说明。它与 [TUF 签名更新](SIGNED_UPDATES.md) 是两套机制：这里只决定展示哪份日志，不负责下载或替换程序。

## 加载规则

后端内嵌 `GoServer/services/update_log_default.json`。启动时可以额外读取：

```text
BctoolData/update-log.json
BctoolData/update-log.sig
```

外部日志只有在原始 JSON 字节通过 Ed25519 验证，并通过结构、版本、发布时间、有效期、标题、正文和下载地址校验后才会使用。文件缺失、损坏、过期或校验失败时，回退到内嵌日志。

外部日志不能早于内嵌版本或其发布时间，有效期最长 90 天。签名后的 JSON 不能再改动，包括换行和空白。

## 密钥

应用只编译 `GoServer/services/update_log.go` 中的公钥。对应私钥保存在仓库外受限的发布位置。

不要将私钥放入源码树、程序目录、`BctoolData`、CI 日志或 Release 附件。需要轮换时，使用 `scripts/generate-update-log-key.cjs` 生成到受控位置，更新 `updateLogPublicKeyB64`，再保管好新私钥。

## 签名

对即将复制到 `BctoolData` 的原始 JSON 文件签名：

```powershell
$env:BCTOOLS_UPDATE_LOG_PRIVATE_KEY = (Get-Content '<secure-key-path>' -Raw).Trim()
go -C GoServer run ./cmd/sign-update-log ../update-log.json ../update-log.sig
Remove-Item Env:BCTOOLS_UPDATE_LOG_PRIVATE_KEY
```

命令在仓库根目录执行，输出 `update-log.json` 对应的 `update-log.sig`。签名后如需修改正文，必须重新签名。

发布前至少验证：有效日志可展示、错误签名被拒绝、过期日志回退、缺失文件不影响启动。私钥不得为了测试而复制进应用目录。
