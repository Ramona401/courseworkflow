#!/bin/bash
# 恢复Vite生产目录的公开读取权限。
#
# 只处理前端dist目录，不修改Nginx配置，不重启后端，不重新构建。

set -Eeuo pipefail
umask 022

DIST="/www/wwwroot/tedna/frontend/dist"
INDEX_FILE="$DIST/index.html"

test -d "$DIST"
test -f "$INDEX_FILE"

chmod -R a+rX "$DIST"

sudo -u www-data test -r "$INDEX_FILE"

echo "✅ 前端dist权限已恢复"
