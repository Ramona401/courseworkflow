#!/bin/bash
# ============================================================================
# TE-DNA 2.0 一键部署脚本（日常迭代版 v3）
# ----------------------------------------------------------------------------
# 适用环境：阿里云 ECS Ubuntu 24.04 (47.86.248.255)
# 架构：Go 后端 (:8080, systemd) + React 前端 (Nginx 静态) + PostgreSQL 16
# 域名：https://workflow.pkuailab.com
# ----------------------------------------------------------------------------
# 默认行为（快速模式，预计 1-2 分钟完成）：
#   ✅ 数据库备份        ✅ Go 编译         ✅ 前端 Vite 构建
#   ✅ 健康检查+自动回滚  ✅ 端点冒烟测试    ✅ 二进制原子替换
#   ❌ 跳过 golangci-lint ❌ 跳过 ESLint    ❌ 跳过单元+集成测试
#   ❌ 不跑 npm audit（npmmirror 环境不支持）
#
# 使用方法：
#   bash /www/wwwroot/tedna/deploy.sh                     # 日常快速部署
#   SKIP_BACKEND=1   bash /www/wwwroot/tedna/deploy.sh    # 仅前端更新
#   SKIP_FRONTEND=1  bash /www/wwwroot/tedna/deploy.sh    # 仅后端更新
#   RUN_LINT=1       bash /www/wwwroot/tedna/deploy.sh    # 额外跑 lint（不阻塞）
#   RUN_TESTS=1      bash /www/wwwroot/tedna/deploy.sh    # 额外跑测试（阻塞）
#   RUN_LINT=1 RUN_TESTS=1 bash /www/wwwroot/tedna/deploy.sh  # 发版前完整验证
#
# 其他开关：
#   SKIP_BACKUP=1    跳过数据库备份（极端紧急时）
# ============================================================================

(
set -e

# ============================================================================
# 路径常量
# ============================================================================
PROJECT_ROOT="/www/wwwroot/tedna"
BACKEND_DIR="$PROJECT_ROOT/backend"
FRONTEND_DIR="$PROJECT_ROOT/frontend"
FRONTEND_DIST="$FRONTEND_DIR/dist"
BIN_PATH="$BACKEND_DIR/server"
SERVICE_NAME="tedna"
DB_NAME="tedna"
DB_TEST_NAME="tedna_test"
HEALTH_URL="http://127.0.0.1:8080/api/v1/health"
PUBLIC_URL="https://workflow.pkuailab.com"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
DEPLOY_LOG_DIR="$PROJECT_ROOT/deploy-logs"
DB_BACKUP_DIR="$PROJECT_ROOT/db-backups"
mkdir -p "$DEPLOY_LOG_DIR" "$DB_BACKUP_DIR"
OLD_BIN_BACKUP=""
DB_BACKUP_FILE=""
BACKUP_SIZE=""

# 开始计时
START_TS=$(date +%s)

echo "========= TE-DNA 2.0 部署开始 ========="
echo "时间:     $(date '+%Y-%m-%d %H:%M:%S')"
echo "操作员:   $(whoami)@$(hostname)"
echo "提交版本: $(cd $PROJECT_ROOT && git rev-parse --short HEAD 2>/dev/null || echo '非 git 仓库')"
echo "时间戳:   $TIMESTAMP"
echo "模式:     $([ "$RUN_LINT" = "1" ] && echo -n '含Lint ' )$([ "$RUN_TESTS" = "1" ] && echo -n '含测试 ' )$([ "$RUN_LINT" != "1" ] && [ "$RUN_TESTS" != "1" ] && echo -n '快速模式 ')"
echo ""

# ============================================================================
# 0. 前置环境检查
# ============================================================================
echo "0. 前置环境检查"

systemctl is-active postgresql > /dev/null 2>&1 || { echo "   ❌ PostgreSQL 未运行"; false; }
echo "   ✅ PostgreSQL 16 运行中"

sudo -u postgres psql -d "$DB_NAME" -c "SELECT 1" > /dev/null 2>&1 || { echo "   ❌ 数据库 $DB_NAME 不可访问"; false; }
TABLE_COUNT=$(sudo -u postgres psql -d "$DB_NAME" -t -c "SELECT count(*) FROM information_schema.tables WHERE table_schema='public'" | tr -d ' ')
echo "   ✅ 数据库 $DB_NAME 可访问 ($TABLE_COUNT 张表)"

systemctl is-active nginx > /dev/null 2>&1 || { echo "   ⚠ Nginx 未运行，正在启动..."; systemctl start nginx; }
echo "   ✅ Nginx 运行中"

systemctl list-unit-files | grep -q "^${SERVICE_NAME}.service" || { echo "   ❌ systemd 服务 ${SERVICE_NAME}.service 不存在"; false; }
echo "   ✅ systemd 服务 $SERVICE_NAME 已注册"

command -v go > /dev/null || { echo "   ❌ Go 未安装"; false; }
command -v node > /dev/null || { echo "   ❌ Node.js 未安装"; false; }
[ -f "$BACKEND_DIR/go.mod" ] || { echo "   ❌ 后端 go.mod 不存在"; false; }
[ -f "$FRONTEND_DIR/package.json" ] || { echo "   ❌ 前端 package.json 不存在"; false; }
[ -f "$BACKEND_DIR/.env" ] || { echo "   ❌ 后端 .env 文件不存在"; false; }
echo "   ✅ 项目结构完整"

# ============================================================================
# 1. 数据库备份
# ============================================================================
if [ "$SKIP_BACKUP" = "1" ]; then
    echo ""
    echo "1. ⏭ 跳过数据库备份（SKIP_BACKUP=1）"
else
    echo ""
    echo "1. 数据库自动备份"
    DB_BACKUP_FILE="$DB_BACKUP_DIR/${DB_NAME}_${TIMESTAMP}.sql.gz"
    sudo -u postgres pg_dump "$DB_NAME" 2>/dev/null | gzip > "$DB_BACKUP_FILE" || { echo "   ❌ 数据库备份失败"; false; }
    BACKUP_SIZE=$(ls -lh "$DB_BACKUP_FILE" | awk '{print $5}')
    echo "   ✅ 备份完成: ${DB_NAME}_${TIMESTAMP}.sql.gz ($BACKUP_SIZE)"

    # 只保留最近 10 份
    ls -t "$DB_BACKUP_DIR"/${DB_NAME}_*.sql.gz 2>/dev/null | tail -n +11 | xargs -r rm -f
    KEPT=$(ls "$DB_BACKUP_DIR"/${DB_NAME}_*.sql.gz 2>/dev/null | wc -l)
    echo "   ✅ 历史备份保留 $KEPT 份"
fi

# ============================================================================
# 2. 后端：依赖 + [可选 lint] + [可选 测试] + 编译
# ============================================================================
if [ "$SKIP_BACKEND" = "1" ]; then
    echo ""
    echo "2. ⏭ 跳过后端构建（SKIP_BACKEND=1）"
else
    echo ""
    echo "2. 后端构建"
    cd "$BACKEND_DIR"

    echo "   2.1 同步 Go 依赖"
    GO_MOD_LOG="$DEPLOY_LOG_DIR/go-mod_${TIMESTAMP}.log"
    if go mod download > "$GO_MOD_LOG" 2>&1; then
        echo "       ✅ 依赖同步完成"
    else
        echo "       ❌ go mod download 失败:"
        cat "$GO_MOD_LOG"
        false
    fi

    # ---- 可选：golangci-lint ----
    if [ "$RUN_LINT" = "1" ]; then
        echo "   2.2 golangci-lint 检查（RUN_LINT=1）"
        LINT_LOG="$DEPLOY_LOG_DIR/golangci-lint_${TIMESTAMP}.log"
        if golangci-lint run ./... > "$LINT_LOG" 2>&1; then
            echo "       ✅ golangci-lint 0 问题"
        else
            echo "       ⚠ golangci-lint 发现问题（不阻塞部署）:"
            tail -20 "$LINT_LOG"
            echo "       ...完整日志: $LINT_LOG"
        fi
    else
        echo "   2.2 ⏭ 跳过 golangci-lint（默认）"
    fi

    # ---- 可选：测试 ----
    if [ "$RUN_TESTS" = "1" ]; then
        echo "   2.3 运行后端测试（RUN_TESTS=1，失败会阻塞部署）"
        UNIT_LOG="$DEPLOY_LOG_DIR/go-unit-test_${TIMESTAMP}.log"
        echo "       运行单元测试..."
        if go test -count=1 -timeout 5m $(go list ./... | grep -v '/integration') > "$UNIT_LOG" 2>&1; then
            UNIT_PASS=$(grep -c "^ok" "$UNIT_LOG" || true)
            echo "       ✅ 单元测试通过 ($UNIT_PASS 个包)"
        else
            echo "       ❌ 单元测试失败:"
            tail -40 "$UNIT_LOG"
            false
        fi

        INTEG_LOG="$DEPLOY_LOG_DIR/go-integ-test_${TIMESTAMP}.log"
        if sudo -u postgres psql -lqt | cut -d \| -f 1 | grep -qw "$DB_TEST_NAME"; then
            echo "       运行集成测试（真实数据库 $DB_TEST_NAME）..."
            if go test -count=1 -timeout 10m ./internal/integration/... > "$INTEG_LOG" 2>&1; then
                INTEG_PASS=$(grep -c "^ok" "$INTEG_LOG" || true)
                echo "       ✅ 集成测试通过 ($INTEG_PASS 个包)"
            else
                echo "       ❌ 集成测试失败:"
                tail -40 "$INTEG_LOG"
                false
            fi
        else
            echo "       ⚠ 数据库 $DB_TEST_NAME 不存在，跳过集成测试"
        fi
    else
        echo "   2.3 ⏭ 跳过测试（默认，可用 RUN_TESTS=1 开启）"
    fi

    # ---- 必做：Go 编译 ----
    echo "   2.4 编译 Go 二进制（生产模式 -s -w）"
    BUILD_LOG="$DEPLOY_LOG_DIR/go-build_${TIMESTAMP}.log"

    if [ -f "$BIN_PATH" ]; then
        OLD_BIN_BACKUP="${BIN_PATH}.backup.${TIMESTAMP}"
        cp "$BIN_PATH" "$OLD_BIN_BACKUP"
        echo "       ✅ 旧二进制已备份: server.backup.${TIMESTAMP}"
    fi

    GIT_COMMIT=$(cd "$PROJECT_ROOT" && git rev-parse --short HEAD 2>/dev/null || echo "unknown")
    BUILD_TIME=$(date '+%Y-%m-%dT%H:%M:%S%z')
    LDFLAGS="-s -w -X main.GitCommit=$GIT_COMMIT -X main.BuildTime=$BUILD_TIME"

    # 编译到临时路径，成功后再原子替换
    BIN_TMP="${BIN_PATH}.new.${TIMESTAMP}"
    if CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "$LDFLAGS" -o "$BIN_TMP" ./cmd/server > "$BUILD_LOG" 2>&1; then
        BIN_SIZE=$(ls -lh "$BIN_TMP" | awk '{print $5}')
        echo "       ✅ 编译完成 ($BIN_SIZE, commit=$GIT_COMMIT)"
    else
        echo "       ❌ Go 编译失败:"
        cat "$BUILD_LOG"
        rm -f "$BIN_TMP"
        false
    fi

    # 清理：最近 5 份旧二进制备份
    ls -t "${BIN_PATH}.backup."* 2>/dev/null | tail -n +6 | xargs -r rm -f
fi

# ============================================================================
# 3. 前端：依赖 + [可选 lint] + 构建（不跑 npm audit）
# ============================================================================
if [ "$SKIP_FRONTEND" = "1" ]; then
    echo ""
    echo "3. ⏭ 跳过前端构建（SKIP_FRONTEND=1）"
else
    echo ""
    echo "3. 前端构建"
    cd "$FRONTEND_DIR"

    echo "   3.1 检查 npm 依赖"
    if [ ! -d "node_modules" ] || [ "package.json" -nt "node_modules" ]; then
        echo "       依赖需要更新，执行 npm ci..."
        NPM_INSTALL_LOG="$DEPLOY_LOG_DIR/npm-install_${TIMESTAMP}.log"
        if npm ci > "$NPM_INSTALL_LOG" 2>&1; then
            echo "       ✅ npm ci 完成"
        else
            echo "       ❌ npm ci 失败:"
            tail -30 "$NPM_INSTALL_LOG"
            false
        fi
    else
        echo "       ✅ node_modules 已是最新"
    fi

    # ---- 可选：ESLint ----
    if [ "$RUN_LINT" = "1" ]; then
        echo "   3.2 ESLint 检查（RUN_LINT=1）"
        ESLINT_LOG="$DEPLOY_LOG_DIR/eslint_${TIMESTAMP}.log"
        if npm run lint > "$ESLINT_LOG" 2>&1; then
            echo "       ✅ ESLint 通过"
        else
            echo "       ⚠ ESLint 有问题（不阻塞部署）:"
            tail -15 "$ESLINT_LOG"
            echo "       ...完整日志: $ESLINT_LOG"
        fi
    else
        echo "   3.2 ⏭ 跳过 ESLint（默认）"
    fi

    # ---- 必做：Vite 构建 ----
    echo "   3.3 Vite 生产构建"
    if [ -d "$FRONTEND_DIST" ]; then
        DIST_BACKUP="$FRONTEND_DIR/dist.backup.${TIMESTAMP}"
        cp -r "$FRONTEND_DIST" "$DIST_BACKUP"
        echo "       ✅ 旧 dist 已备份"
    fi
    VITE_LOG="$DEPLOY_LOG_DIR/vite-build_${TIMESTAMP}.log"
    if npm run build > "$VITE_LOG" 2>&1; then
        ASSET_COUNT=$(ls "$FRONTEND_DIST/assets/" 2>/dev/null | wc -l)
        DIST_SIZE=$(du -sh "$FRONTEND_DIST" | awk '{print $1}')
        echo "       ✅ 前端构建成功 ($ASSET_COUNT 个资源, $DIST_SIZE)"
    else
        echo "       ❌ 前端构建失败:"
        tail -30 "$VITE_LOG"
        false
    fi
    [ -f "$FRONTEND_DIST/index.html" ] || { echo "   ❌ 构建产物 index.html 缺失"; false; }

    # 清理：最近 3 份旧 dist 备份
    ls -td "$FRONTEND_DIR/dist.backup."* 2>/dev/null | tail -n +4 | xargs -r rm -rf
fi

# ============================================================================
# 4. Nginx 配置校验与重载
# ============================================================================
echo ""
echo "4. Nginx 配置校验"
if nginx -t 2>&1 | tail -2 | grep -q "successful"; then
    systemctl reload nginx
    echo "   ✅ nginx -t 通过，已 reload（前端静态即时生效）"
else
    echo "   ❌ Nginx 配置语法错误:"
    nginx -t
    false
fi

# ============================================================================
# 5. 原子替换二进制 + 重启 systemd
# ============================================================================
if [ "$SKIP_BACKEND" = "1" ]; then
    echo ""
    echo "5. ⏭ 跳过后端重启（SKIP_BACKEND=1）"
else
    echo ""
    echo "5. 替换二进制并重启 systemd"

    BIN_TMP="${BIN_PATH}.new.${TIMESTAMP}"
    if [ ! -f "$BIN_TMP" ]; then
        echo "   ❌ 编译产物 $BIN_TMP 不存在"
        false
    fi

    mv "$BIN_TMP" "$BIN_PATH"
    chmod +x "$BIN_PATH"
    echo "   ✅ 二进制已原子替换"

    systemctl restart "$SERVICE_NAME"
    echo "   ✅ systemctl restart $SERVICE_NAME 已发送"
fi

# ============================================================================
# 6. 健康检查 + 自动回滚
# ============================================================================
echo ""
echo -n "6. 等待服务就绪"
OK=0
for i in $(seq 1 30); do
    sleep 1
    echo -n "."
    if curl -sf "$HEALTH_URL" > /dev/null 2>&1; then
        OK=1
        break
    fi
done
echo ""

if [ "$OK" = "1" ]; then
    echo "   ✅ 服务就绪 (耗时 ${i}s)"
else
    echo "   ❌ 服务启动超时（30s），最近日志:"
    journalctl -u "$SERVICE_NAME" --no-pager -n 30

    if [ -n "$OLD_BIN_BACKUP" ] && [ -f "$OLD_BIN_BACKUP" ]; then
        echo ""
        echo "   ⚠ 自动回滚到上一版二进制..."
        cp "$OLD_BIN_BACKUP" "$BIN_PATH"
        systemctl restart "$SERVICE_NAME"
        sleep 3
        if curl -sf "$HEALTH_URL" > /dev/null 2>&1; then
            echo "   ✅ 已回滚到上一版本，服务恢复正常"
        else
            echo "   ❌ 回滚后仍异常，请人工介入: journalctl -u $SERVICE_NAME"
        fi
    fi
    false
fi

# ============================================================================
# 7. 端点冒烟验证
# ============================================================================
echo ""
echo "7. 端点冒烟验证"
GO_HEALTH=$(curl -so/dev/null -w%{http_code} "$HEALTH_URL")
NGINX_HTTPS=$(curl -so/dev/null -w%{http_code} --insecure "$PUBLIC_URL")
NGINX_API=$(curl -so/dev/null -w%{http_code} --insecure "$PUBLIC_URL/api/v1/health")
echo "   Go 直连 /health:     $GO_HEALTH"
echo "   Nginx HTTPS 首页:    $NGINX_HTTPS"
echo "   Nginx HTTPS /health: $NGINX_API"

if [ "$GO_HEALTH" != "200" ] || [ "$NGINX_HTTPS" != "200" ] || [ "$NGINX_API" != "200" ]; then
    echo "   ❌ 关键端点异常，请人工核查"
    false
fi
echo "   ✅ 所有关键端点正常"

# ============================================================================
# 8. 部署统计
# ============================================================================
END_TS=$(date +%s)
ELAPSED=$((END_TS - START_TS))

echo ""
echo "8. 部署统计"
echo "   耗时:        ${ELAPSED}s"
echo "   后端二进制:  $(ls -lh $BIN_PATH | awk '{print $5}')"
if [ -d "$FRONTEND_DIST" ]; then
    echo "   前端 dist:   $(du -sh $FRONTEND_DIST | awk '{print $1}') ($(ls $FRONTEND_DIST/assets/ 2>/dev/null | wc -l) 个资源)"
fi
[ -n "$DB_BACKUP_FILE" ] && echo "   数据库备份:  ${DB_NAME}_${TIMESTAMP}.sql.gz ($BACKUP_SIZE)"

# 运行版本（从 /health 提取）
RUNNING_VERSION=$(curl -sf "$HEALTH_URL" | grep -oP '"version":"[^"]+"' || echo "")
RUNNING_UPTIME=$(curl -sf "$HEALTH_URL" | grep -oP '"uptime":"[^"]+"' || echo "")
[ -n "$RUNNING_VERSION" ] && echo "   运行版本:    $RUNNING_VERSION"
[ -n "$RUNNING_UPTIME" ] && echo "   服务运行:    $RUNNING_UPTIME"

echo ""
echo "========= ✅ 部署完成 $(date '+%H:%M:%S') (${ELAPSED}s) ========="
echo ""
echo "🌐 $PUBLIC_URL"
echo ""
echo "📋 运维命令:"
echo "   journalctl -u $SERVICE_NAME -f        # 实时日志"
echo "   systemctl status $SERVICE_NAME        # 服务状态"
if [ -n "$OLD_BIN_BACKUP" ]; then
    echo ""
    echo "🔙 手动回滚（如需）:"
    echo "   cp $OLD_BIN_BACKUP $BIN_PATH && systemctl restart $SERVICE_NAME"
fi
echo "================================================================"
)
