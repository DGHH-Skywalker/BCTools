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
  "version": "5.0.0.1",
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
  "latestVersion": "5.0.0.1",
  "downloadUrl": ""
}
```
