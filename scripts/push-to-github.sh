#!/bin/bash
# 一键把本地仓库推送到 GitHub 公开仓库 DGHH-Skywalker/BCTOOLS
# 用法：在仓库根目录下执行 `bash scripts/push-to-github.sh`
set -e

cd "$(dirname "$0")/.."

# 1. 把 um-react 从 submodule 链接换成真实源码（如果之前 commit 还是 gitlink）
git ls-tree HEAD Web/um-react | grep -q "160000 commit" && {
  echo "[1/4] 把 Web/um-react 从 submodule 链接换成真实源码..."
  git rm --cached -rf Web/um-react
  git update-index --replace --add Web/um-react
  git add Web/um-react
  git commit -m "feat: Web/um-react 从子模块改为内嵌源码"
} || echo "[1/4] um-react 已经是真实源码，跳过"

# 2. 创建公开仓库（如已存在会报错，跳过）
echo "[2/4] 创建公开仓库 DGHH-Skywalker/BCTOOLS..."
gh repo create BCTOOLS --public \
  --description "小播点歌工具 — 揭阳一中广播站歌单管理工具（Vue 3 + Go + um-react）" \
  --source . \
  --remote origin \
  --push || {
  echo "  仓库可能已存在，尝试仅 push..."
  git remote add origin https://github.com/DGHH-Skywalker/BCTOOLS.git 2>/dev/null || true
  git push -u origin master
}

# 3. 校验
echo "[3/4] 验证远端..."
git remote -v
git ls-remote origin HEAD 2>&1 | head -3 || true

echo "[4/4] 完成！仓库地址: https://github.com/DGHH-Skywalker/BCTOOLS"
