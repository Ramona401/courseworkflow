#!/bin/bash
# 验证前端入口权限、服务状态和公开端点。

set -Eeuo pipefail

DIST="/www/wwwroot/tedna/frontend/dist"
INDEX_FILE="$DIST/index.html"
PUBLIC_URL="https://workflow.pkuailab.com"

test -d "$DIST"
test -f "$INDEX_FILE"

echo "=========文件权限========="

stat \
  -c '%A | %U:%G | %a | %n' \
  "$DIST" \
  "$INDEX_FILE"

sudo -u www-data test -r "$INDEX_FILE"

echo ""
echo "=========服务状态========="

systemctl is-active nginx
systemctl is-active tedna
systemctl is-active postgresql

echo ""
echo "=========公开端点========="

HOME_STATUS=$(
  curl \
    --silent \
    --output /dev/null \
    --write-out '%{http_code}' \
    --max-time 8 \
    "$PUBLIC_URL/"
)

API_STATUS=$(
  curl \
    --silent \
    --output /dev/null \
    --write-out '%{http_code}' \
    --max-time 8 \
    "$PUBLIC_URL/api/v1/health"
)

echo "首页：$HOME_STATUS"
echo "API：$API_STATUS"

test "$HOME_STATUS" = "200"
test "$API_STATUS" = "200"

echo "✅ 首页和API均正常"
