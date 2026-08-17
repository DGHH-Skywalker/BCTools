#!/bin/bash
# 跑迁移端到端测试（Windows 路径用 JSON 文件传，避免 bash 转义坑）
set -e
PORT=1799
TARGET='C:\Users\pc\Desktop\mig_full'
BACKUP_BASE='C:\Users\pc\Desktop'

# 1. 导出
echo "=== 1. export files ==="
curl -sS -X POST http://localhost:$PORT/api/migration/export \
  -H "Content-Type: application/json" \
  -d "{\"scope\":\"files\",\"targetDir\":\"$TARGET\"}"
echo ""

# 2. 找子目录
sleep 1
BACKUP=$(ls "$BACKUP_BASE/mig_full" 2>/dev/null | head -1)
if [ -z "$BACKUP" ]; then
  echo "ERROR: no backup folder found"
  exit 1
fi
echo "backup folder: $BACKUP"

# 3. merge
echo "=== 2. import (merge) ==="
cat > /tmp/req.json <<EOF
{"backupDir":"$BACKUP_BASE\\mig_full\\$BACKUP","mode":"merge"}
EOF
curl -sS -X POST http://localhost:$PORT/api/migration/import \
  -H "Content-Type: application/json" \
  --data @/tmp/req.json
echo ""

# 4. merge 再次
echo "=== 3. import again (merge) ==="
curl -sS -X POST http://localhost:$PORT/api/migration/import \
  -H "Content-Type: application/json" \
  --data @/tmp/req.json
echo ""

# 5. replace
echo "=== 4. import (replace) ==="
cat > /tmp/req2.json <<EOF
{"backupDir":"$BACKUP_BASE\\mig_full\\$BACKUP","mode":"replace"}
EOF
curl -sS -X POST http://localhost:$PORT/api/migration/import \
  -H "Content-Type: application/json" \
  --data @/tmp/req2.json
echo ""

# 6. invalid
echo "=== 5. invalid dir ==="
cat > /tmp/req3.json <<EOF
{"backupDir":"$BACKUP_BASE\\nonexistent_xyz","mode":"merge"}
EOF
curl -sS -X POST http://localhost:$PORT/api/migration/import \
  -H "Content-Type: application/json" \
  --data @/tmp/req3.json
echo ""

# 7. bad mode
echo "=== 6. bad mode ==="
cat > /tmp/req4.json <<EOF
{"backupDir":"$BACKUP_BASE\\mig_full\\$BACKUP","mode":"invalid"}
EOF
curl -sS -X POST http://localhost:$PORT/api/migration/import \
  -H "Content-Type: application/json" \
  --data @/tmp/req4.json
echo ""

# 8. missing data.json
mkdir -p "$BACKUP_BASE/empty_dir"
echo "=== 7. missing data.json ==="
cat > /tmp/req5.json <<EOF
{"backupDir":"$BACKUP_BASE\\empty_dir","mode":"merge"}
EOF
curl -sS -X POST http://localhost:$PORT/api/migration/import \
  -H "Content-Type: application/json" \
  --data @/tmp/req5.json
echo ""

# 9. JSON export
echo "=== 8. JSON export ==="
curl -sS -X POST http://localhost:$PORT/api/migration/export \
  -H "Content-Type: application/json" \
  -d '{"scope":"json"}' -o /tmp/test_json_export.json -w " HTTP=%{http_code} SIZE=%{size_download}\n"
head -c 200 /tmp/test_json_export.json
echo ""
