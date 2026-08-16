# BroadcastTool API 契约

> 基线路径：`/api`（前端必须使用相对路径，禁止硬编码 localhost:1743）

## 通用错误响应

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "用户可理解的错误描述"
  }
}
```

错误码：
- `VALIDATION_ERROR` - 输入校验失败
- `NOT_FOUND` - 资源不存在
- `INTERNAL_ERROR` - 服务器内部错误
- `UNAUTHORIZED` - 未认证
- `FORBIDDEN` - 禁止访问
- `FILE_ERROR` - 文件操作失败
- `CONVERSION_ERROR` - 音频转换失败
- `DATA_CORRUPTED` - 数据损坏
- `UNSUPPORTED_ENCRYPTED_FORMAT` - 不支持的加密格式
- `ENCRYPTED_FILE_INVALID` - 加密文件无效
- `ENCRYPTION_DECRYPT_FAILED` - 解密失败

## 健康检查

`GET /api/health`

响应：
```json
{ "status": "ok" }
```

## 生命周期

托盘控件（任务栏图标）通过这组端点与前端页面协作退出。

### 等待关闭通知

`GET /api/lifecycle/watch`

长轮询：请求会一直挂起，直到后端准备退出才返回。前端据此在后端退出前尝试
`window.close()` 自行关闭页面（浏览器常拦截脚本关闭标签页，因此是 best-effort）。
客户端断开时服务端直接放弃该请求，不泄漏 goroutine。

响应（仅在后端即将退出时）：
```json
{ "closing": true }
```

### 请求关闭后端

`POST /api/shutdown`

**仅接受来自本机（loopback）的请求**，局域网内其他设备调用返回 403，
避免任何能访问该端口的设备都能关掉服务。

响应：
```json
{ "status": "closing" }
```

错误：
- `403 FORBIDDEN` - 非本机请求
- `501 NOT_IMPLEMENTED` - 当前运行模式未接入关闭回调

### 关闭浏览器（兼容保留）

`POST /api/close-browser`

仅记录日志，不会退出后端。同样限制为本机请求。

## 歌曲

### 模型

```typescript
interface Song {
  id: number
  date: string        // YYYY-MM-DD
  weekday: string     // 计算字段，如"周一"
  title: string
  artist: string
  remark: string
  filePath: string
  timeSlotId: string | null
  createdAt: string   // RFC3339
}
```

### 列表/查询

`GET /api/songs?type=dorm|broadcast&dates=2026-07-20,2026-07-21`

响应：`Song[]`

### 获取单个

`GET /api/songs/:id`

响应：`Song`

### 创建

`POST /api/songs`

请求体：
```json
{
  "type": "dorm",
  "date": "2026-07-25",
  "title": "歌曲名",
  "artist": "歌手",
  "remark": "备注",
  "filePath": "temp/xxx.mp3",
  "timeSlotId": null
}
```

响应：`Song` (201)

### 更新

`PUT /api/songs/:id`

请求体只包含需要更新的字段。`timeSlotId: null` 表示取消分配。

响应：`Song`

### 删除

`DELETE /api/songs/:id`

响应：`204 No Content`

### 导入 XLSX

`POST /api/songs/import?type=dorm|broadcast`

上传字段：`file`（xlsx）

响应：
```json
{
  "inserted": 10,
  "skipped": 0,
  "errors": [{ "row": 3, "message": "日期格式错误" }]
}
```

### 导出 XLSX

`GET /api/songs/export?type=dorm|broadcast&dates=...`

响应：Blob，Content-Type `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`

## 文件处理

### 处理/转码

`POST /api/files/process`

上传字段：`file`

支持：ncm、mp3、flac、wav、mp4 等。NCM 由后端 unlock-music 库解密并转码为 MP3。

响应：
```json
{
  "tempFileName": "uuid.mp3",
  "title": "歌名",
  "artist": "歌手"
}
```

### MP3 暂存

`POST /api/files/stash`

上传字段：`file`（mp3）

不转码，仅提取标签。

响应：同 `/api/files/process`

### 文件整理

`POST /api/files/organize`

请求体：
```json
{
  "entries": [
    { "source": "temp/xxx.mp3", "targetName": "01.mp3" }
  ],
  "targetDir": "D:/target",
  "mode": "copy",
  "confirm": false
}
```

`source` 为空时生成静音占位文件。

响应：
```json
{
  "successful": [{ "source": "...", "target": "..." }],
  "failed": [{ "source": "...", "reason": "..." }],
  "confirmNeeded": true,
  "existingFiles": ["01.mp3"]
}
```

### 选择文件夹

`POST /api/files/select-dir`

响应：
```json
{ "path": "D:/selected" }
```

用户取消时返回 `""`。

### 浏览目录

`GET /api/files/browse?dir=...`

响应：
```json
{
  "path": "D:/dir",
  "dirs": ["sub"],
  "files": ["a.mp3"]
}
```

## 设置

### 获取

`GET /api/settings`

响应（不含 `adminPasswordHash`）：
```json
{
  "timeSlots": [...],
  "allowTemplateJS": false,
  "autoBackupPath": "",
  "autoBackupEnabled": false,
  "silentPlaceholderDuration": 30,
  "adminPasswordHint": "",
  "locale": "zh-CN",
  "version": "5.5.0.0",
  "downloadUrl": ""
}
```

### 更新

`PUT /api/settings`

部分更新。删除时段时若影响已分配歌曲，首次返回：
```json
{ "confirmNeeded": true, "affectedCount": 3 }
```

再次发送 `confirmed: true` 执行删除。

## 认证

`POST /api/auth/verify`

请求体：
```json
{ "password": "admin" }
```

响应：
```json
{ "success": true, "hint": "" }
```

## 备份与快照

### 立即备份

`POST /api/sync/backup`

响应：
```json
{ "success": true, "path": "..." }
```

### 快照列表

`GET /api/snapshots`

响应：
```json
[
  { "filename": "snapshot-2026-07-25.json", "time": "...", "size": 1234 }
]
```

### 恢复快照

`POST /api/snapshots/restore?filename=snapshot-2026-07-25.json`

响应：
```json
{ "success": true }
```

## 检查更新

`GET /api/check-update`

响应：
```json
{
  "hasUpdate": false,
  "latestVersion": "5.5.0.0",
  "downloadUrl": ""
}
```

## 解密暂存（um-react 桥接）

um-react 在前端完成解密后，把音频交给主应用导入。两个页面可能在**不同设备**上
（手机解密、电脑导入），因此交接走后端暂存区而非 postMessage。

### 上传暂存

`POST /api/decrypt/stage`（multipart，字段名 `file`）

响应：
```json
{ "stageId": "uuid", "filename": "李克勤 - 红日.mp3", "size": 12451935 }
```

### 查询暂存元信息

`GET /api/decrypt/stage/{id}`

```json
{ "filename": "李克勤 - 红日.mp3", "size": 12451935, "ext": ".mp3", "imported": false }
```

### 就地导入歌库（推荐）

`POST /api/decrypt/stage/{id}/import`

服务端直接把暂存文件转入歌库，响应结构与 `/api/files/stash` 一致：

```json
{ "tempFileName": "uuid.mp3", "title": "李克勤 - 红日", "artist": "" }
```

**为什么要有这个端点**：此前主应用得先 `GET .../file` 把音频下载回浏览器，
再 `POST /api/files/stash` 原样传回去。一首 11.88 MB 的歌要在本机 HTTP 上跑三趟
共 35.6 MB。实测（真实 `.ncm`，Python 直连、排除 curl 进程启动开销）：

| | 耗时 | 传输量 |
|---|---|---|
| 旧：下载回来再上传 | ~263 ms | 35.6 MB |
| 新：就地导入 | **~67 ms** | **11.9 MB** |

已是 MP3 时用 rename 搬运，并优先采用暂存 meta 里的文件名作标题，
从而跳过 ffprobe（一次约 88 ms，比复制 12 MB 还贵 5 倍）。
非 MP3（flac/m4a 等）仍走 `StashFile` 转码。

### 下载暂存文件

`GET /api/decrypt/stage/{id}/file`

仍保留，供需要拿到原始字节的场景使用；导入流程已不再需要它。

### 标记已导入 / 删除暂存

`POST /api/decrypt/stage/{id}/imported`：置 `imported=true`。um-react 轮询到该标志后
用自身的删除逻辑移除卡片，再调用下面的删除接口。

`DELETE /api/decrypt/stage/{id}`：清理暂存文件。
