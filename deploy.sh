#!/bin/bash
# ============================================================================
# TE-DNA 2.0 一键部署脚本（日常迭代版 v3.3）
# ----------------------------------------------------------------------------
# 适用环境：阿里云 ECS Ubuntu 24.04 (47.86.248.255)
# 架构：Go 后端 (:8080, systemd) + React 前端 (Nginx 静态) + PostgreSQL 16
# 域名：https://workflow.pkuailab.com
# ----------------------------------------------------------------------------
# v3.3 变更(2026-07-12)：新增可选前端体积与懒加载防回归检查。
#   - 使用 RUN_BUNDLE_CHECK=1 显式开启，默认快速部署行为不变。
#   - 开启后在正式Vite构建前执行 npm run check:bundle。
#   - 检查课件工坊与学科工具体积预算、两级懒加载边界、
#     11个工具弹窗动态入口及地理/生命科学/物理/化学模板隔离。
#   - 检查脚本使用Vite内存构建，不覆盖正式frontend/dist。
#   - 任一硬性检查失败都会阻断部署，避免大型模板重新回流主资源。
# ----------------------------------------------------------------------------
# v3.2 变更(2026-07-04)：编译步骤末尾新增“清理 server.new.* 中转遗留”逻辑。
#   背景：编译产物先写到 server.new.<时间戳> 再原子 mv 成 server；若部署中途
#   某步失败触发 set -e 提前退出，该中转文件不会被 mv 掉，日积月累堆成几十个
#   十几 M 的二进制垃圾（本次清理掉 37 个），既占磁盘又污染代码索引工具。
#   本版在编译成功后清理历史遗留的 server.new.*（严格排除本次刚编译的产物，
#   保证第 5 步 mv 不受影响）。
# ----------------------------------------------------------------------------
# v3.1 变更(2026-07-02)：集成测试调用前 source backend/.env（set -a 自动导出），
#   修复此前 go test 进程拿不到数据库连接等环境变量导致集成测试整批失败的问题。
# ----------------------------------------------------------------------------
# 默认行为（快速模式，预计 1-2 分钟完成）：
#   ✅ 数据库备份        ✅ Go 编译         ✅ 前端 Vite 构建
#   ✅ 健康检查+自动回滚  ✅ 端点冒烟测试    ✅ 二进制原子替换
#   ❌ 跳过 golangci-lint ❌ 跳过 ESLint    ❌ 跳过单元+集成测试
#   ❌ 不跑 npm audit（npmmirror 环境不支持）
#   ❌ 默认不跑前端体积防回归检查（RUN_BUNDLE_CHECK=1时开启并阻断）
#
# 使用方法：
#   bash /www/wwwroot/tedna/deploy.sh                     # 日常快速部署
#   SKIP_BACKEND=1   bash /www/wwwroot/tedna/deploy.sh    # 仅前端更新
#   SKIP_FRONTEND=1  bash /www/wwwroot/tedna/deploy.sh    # 仅后端更新
#   RUN_LINT=1       bash /www/wwwroot/tedna/deploy.sh    # 额外跑 lint（不阻塞）
#   RUN_TESTS=1      bash /www/wwwroot/tedna/deploy.sh    # 额外跑测试（阻塞）
#   RUN_BUNDLE_CHECK=1 bash /www/wwwroot/tedna/deploy.sh    # 额外跑前端体积与懒加载检查（阻塞）
#   RUN_BUNDLE_CHECK=1 SKIP_BACKEND=1 bash /www/wwwroot/tedna/deploy.sh  # 纯前端部署并执行体积检查
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

# 上下文15：课本图片从公开uploads目录迁移到项目私有目录。
#
# 私有目录位于systemd现有ReadWritePaths覆盖范围内，
# 新后端只从该目录读写，Nginx不再直接暴露课本原图。
TEXTBOOK_PUBLIC_DIR="$PROJECT_ROOT/uploads/textbooks"
TEXTBOOK_PRIVATE_DIR="$PROJECT_ROOT/private/textbooks"
TEXTBOOK_MIGRATION_ACTIVE=0
TEXTBOOK_PROBE_PATH=""

# 开始计时
START_TS=$(date +%s)

# ============================================================================
# 上下文15：课本图片私有迁移辅助函数
# ============================================================================

# prepare_textbook_private_storage 将当前公开目录中的全部课本图片
# 增量复制到私有目录。
#
# 部署前执行一次，后端重启后再执行一次：
#   - 第一次复制绝大部分存量文件；
#   - 第二次捕获旧后端优雅退出期间刚完成的上传。
#
# 公开目录此时仍保留，因此新后端健康检查失败并自动回滚旧二进制时，
# 旧版本仍能继续访问原图片，不会出现数据路径断裂。
prepare_textbook_private_storage() {
    mkdir -p "$PROJECT_ROOT/private"
    mkdir -p "$TEXTBOOK_PRIVATE_DIR"

    chmod 700 "$PROJECT_ROOT/private"
    chmod 700 "$TEXTBOOK_PRIVATE_DIR"

    if [ -d "$TEXTBOOK_PUBLIC_DIR" ]; then
        cp -a "$TEXTBOOK_PUBLIC_DIR/." "$TEXTBOOK_PRIVATE_DIR/"
    fi

    # 私有目录中所有文件和子目录只允许root访问。
    chmod -R go-rwx "$TEXTBOOK_PRIVATE_DIR"

    # 记录一条真实旧文件相对路径，供部署后的公开直链冒烟检查使用。
    TEXTBOOK_PROBE_PATH=$(python3 - "$TEXTBOOK_PUBLIC_DIR" <<'PYTEXTBOOKPROBE'
import sys
from pathlib import Path
from urllib.parse import quote

public_dir = Path(sys.argv[1])

if not public_dir.exists():
    print("")
    raise SystemExit(0)

for path in sorted(public_dir.rglob("*")):
    if path.is_file() and not path.is_symlink():
        relative = path.relative_to(public_dir).as_posix()
        print(quote(relative, safe="/"))
        raise SystemExit(0)

print("")
PYTEXTBOOKPROBE
)
}

# verify_textbook_private_storage 对公开目录中的每一个普通文件做逐文件校验：
#   - 私有目录必须存在同名文件；
#   - 文件大小必须一致；
#   - SHA-256必须一致；
#   - 两侧都禁止出现符号链接。
#
# 私有目录允许存在额外文件，因为新后端启动后可能已经收到新上传。
verify_textbook_private_storage() {
    python3 - "$TEXTBOOK_PUBLIC_DIR" "$TEXTBOOK_PRIVATE_DIR" <<'PYTEXTBOOKVERIFY'
import hashlib
import os
import sys
from pathlib import Path

public_dir = Path(sys.argv[1])
private_dir = Path(sys.argv[2])

if not private_dir.exists():
    raise SystemExit("课本私有目录不存在")

if not public_dir.exists():
    print("       ✅ 公开课本目录不存在，无存量文件需要校验")
    raise SystemExit(0)

checked_files = 0
checked_bytes = 0

def sha256_file(path: Path) -> str:
    digest = hashlib.sha256()

    with path.open("rb") as source:
        while True:
            chunk = source.read(1024 * 1024)
            if not chunk:
                break
            digest.update(chunk)

    return digest.hexdigest()

for root, directory_names, file_names in os.walk(
    public_dir,
    followlinks=False,
):
    root_path = Path(root)

    for directory_name in directory_names:
        directory_path = root_path / directory_name

        if directory_path.is_symlink():
            raise SystemExit(
                f"公开课本目录存在符号链接目录：{directory_path}"
            )

    for file_name in file_names:
        public_path = root_path / file_name

        if public_path.is_symlink():
            raise SystemExit(
                f"公开课本目录存在符号链接文件：{public_path}"
            )

        if not public_path.is_file():
            raise SystemExit(
                f"公开课本目录存在非普通文件：{public_path}"
            )

        relative_path = public_path.relative_to(public_dir)
        private_path = private_dir / relative_path

        if private_path.is_symlink():
            raise SystemExit(
                f"私有课本目录存在符号链接文件：{private_path}"
            )

        if not private_path.is_file():
            raise SystemExit(
                f"私有目录缺少课本文件：{relative_path}"
            )

        public_size = public_path.stat().st_size
        private_size = private_path.stat().st_size

        if public_size != private_size:
            raise SystemExit(
                f"课本文件大小不一致：{relative_path}"
            )

        if sha256_file(public_path) != sha256_file(private_path):
            raise SystemExit(
                f"课本文件校验和不一致：{relative_path}"
            )

        checked_files += 1
        checked_bytes += public_size

print(
    "       ✅ 课本图片迁移校验通过 "
    f"({checked_files}个文件，{checked_bytes}字节)"
)
PYTEXTBOOKVERIFY
}

# retire_public_textbook_storage 只在新后端健康检查成功后执行。
#
# 此时私有副本已完成两轮复制和逐文件SHA-256校验，
# 可以安全删除公开目录中的重复文件，并创建一个空的700权限目录。
#
# 即使Nginx仍配置了/uploads静态映射，该目录也没有任何课本文件，
# 旧的伪造或缓存URL因此无法再读取原图。
retire_public_textbook_storage() {
    verify_textbook_private_storage

    rm -rf "$TEXTBOOK_PUBLIC_DIR"
    mkdir -p "$TEXTBOOK_PUBLIC_DIR"
    chmod 700 "$TEXTBOOK_PUBLIC_DIR"

    echo "   ✅ 公开课本目录已清空并降为700权限"
    echo "   ✅ 课本原图现仅保存在: $TEXTBOOK_PRIVATE_DIR"
}


echo "========= TE-DNA 2.0 部署开始 ========="
echo "时间:     $(date '+%Y-%m-%d %H:%M:%S')"
echo "操作员:   $(whoami)@$(hostname)"
echo "提交版本: $(cd $PROJECT_ROOT && git rev-parse --short HEAD 2>/dev/null || echo '非 git 仓库')"
echo "时间戳:   $TIMESTAMP"
echo "模式:     $([ "$RUN_LINT" = "1" ] && echo -n '含Lint ' )$([ "$RUN_TESTS" = "1" ] && echo -n '含测试 ' )$([ "$RUN_BUNDLE_CHECK" = "1" ] && echo -n '含体积检查 ' )$([ "$RUN_LINT" != "1" ] && [ "$RUN_TESTS" != "1" ] && [ "$RUN_BUNDLE_CHECK" != "1" ] && echo -n '快速模式 ')"
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

    # ---- 必做：go vet 静态检查（阻断，捕获 printf/锁拷贝/不可达代码等编译期查不出的问题）----
    echo "   2.1.1 go vet 静态检查"
    VET_LOG="$DEPLOY_LOG_DIR/go-vet_${TIMESTAMP}.log"
    if go vet ./... > "$VET_LOG" 2>&1; then
        echo "       ✅ go vet 0 问题"
    else
        echo "       ❌ go vet 发现问题（已阻断部署）:"
        cat "$VET_LOG"
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
            # ---- 修复(2026-07-02)：集成测试前加载 backend/.env 环境变量 ----
            # 说明：单元测试不连库可裸跑，但集成测试进程要读数据库连接串、密钥等配置；
            #       此前未 source .env 导致 go test 拿不到环境变量，集成测试整批失败。
            # set -a 使 source 进来的变量自动 export 传给子进程(go test)，用完 set +a 关闭。
            set -a
            . "$BACKEND_DIR/.env"
            set +a
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

    # ---- v3.2 新增：清理 server.new.* 中转遗留 ----
    # 背景：server.new.<时间戳> 是编译产物的临时中转文件，正常会在第 5 步被 mv 成
    #   server；但若部署中途失败(健康检查超时/前端构建失败等)触发 set -e 提前退出，
    #   该中转文件就残留在磁盘上，日积月累堆成几十个十几 M 的二进制垃圾，既占磁盘
    #   又撑爆代码索引工具(单文件超限)。此处清理历史遗留，保留 0 份。
    # 安全：严格用 grep -v 排除本次刚编译好的 BIN_TMP，绝不删除本次产物，
    #   保证第 5 步的 mv "$BIN_TMP" "$BIN_PATH" 不受影响。
    ls -t "${BIN_PATH}.new."* 2>/dev/null | grep -vF "$BIN_TMP" | xargs -r rm -f
    STALE_KEPT=$(ls "${BIN_PATH}.new."* 2>/dev/null | grep -vF "$BIN_TMP" | wc -l)
    echo "       ✅ 已清理 server.new.* 中转遗留（残留 $STALE_KEPT 份）"
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

    # ---- 可选：前端体积与懒加载防回归检查（阻断）----
    # 默认关闭；发版前或重要前端改造时使用 RUN_BUNDLE_CHECK=1 显式开启。
    # check:bundle 内部执行 TypeScript 编译和 Vite 内存生产构建，
    # 不写入正式 dist；任一硬性体积预算或动态加载边界失败均阻断部署。
    if [ "$RUN_BUNDLE_CHECK" = "1" ]; then
        echo "   3.2 前端体积与懒加载防回归检查（RUN_BUNDLE_CHECK=1，失败会阻断部署）"
        BUNDLE_CHECK_LOG="$DEPLOY_LOG_DIR/bundle-check_${TIMESTAMP}.log"

        if npm run check:bundle > "$BUNDLE_CHECK_LOG" 2>&1; then
            echo "       ✅ 前端体积与懒加载防回归检查通过"
        else
            echo "       ❌ 前端体积与懒加载防回归检查失败（已阻断部署）:"
            cat "$BUNDLE_CHECK_LOG"
            false
        fi
    else
        echo "   3.2 ⏭ 跳过前端体积防回归检查（默认，可用 RUN_BUNDLE_CHECK=1 开启）"
    fi

    # ---- 可选：ESLint ----
    if [ "$RUN_LINT" = "1" ]; then
        echo "   3.3 ESLint 检查（RUN_LINT=1）"
        ESLINT_LOG="$DEPLOY_LOG_DIR/eslint_${TIMESTAMP}.log"
        if npm run lint > "$ESLINT_LOG" 2>&1; then
            echo "       ✅ ESLint 通过"
        else
            echo "       ⚠ ESLint 有问题（不阻塞部署）:"
            tail -15 "$ESLINT_LOG"
            echo "       ...完整日志: $ESLINT_LOG"
        fi
    else
        echo "   3.3 ⏭ 跳过 ESLint（默认）"
    fi

    # ---- 必做：Vite 构建 ----
    echo "   3.4 Vite 生产构建"
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
# 3.5 课本图片私有迁移预同步
# ============================================================================
#
# 只有本次同时发布新后端时才迁移。
# 纯前端部署不能提前退役旧后端仍在使用的公开目录。
if [ "$SKIP_BACKEND" = "1" ]; then
    echo ""
    echo "3.5 ⏭ 跳过课本图片私有迁移（本次未发布后端）"
else
    echo ""
    echo "3.5 课本图片私有迁移预同步"

    prepare_textbook_private_storage
    verify_textbook_private_storage

    TEXTBOOK_MIGRATION_ACTIVE=1
    echo "   ✅ 私有目录预同步完成，公开目录暂时保留用于安全回滚"
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

    if [ "$TEXTBOOK_MIGRATION_ACTIVE" = "1" ]; then
        echo "   同步旧后端优雅退出后的课本图片增量..."

        # 旧进程停止接收新请求并完成在途请求后，
        # 再复制一次公开目录，捕获重启期间最后完成的上传。
        prepare_textbook_private_storage
        verify_textbook_private_storage

        echo "   ✅ 旧后端优雅退出后的课本图片增量同步完成"
    fi
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


# 新后端健康检查成功后，才正式退役公开课本目录。
#
# 如果健康检查失败，上面的自动回滚会在公开目录仍完整时恢复旧后端，
# 因此不会因文件迁移破坏回滚链。
if [ "$TEXTBOOK_MIGRATION_ACTIVE" = "1" ]; then
    echo ""
    echo "6.1 正式退役公开课本图片目录"
    retire_public_textbook_storage
fi

# ============================================================================
# 7. 端点冒烟验证
# ============================================================================
echo ""
echo "7. 端点冒烟验证"
GO_HEALTH=$(curl -so/dev/null -w%{http_code} "$HEALTH_URL")
NGINX_HTTPS=$(curl -so/dev/null -w%{http_code} --insecure "$PUBLIC_URL")
NGINX_API=$(curl -so/dev/null -w%{http_code} --insecure "$PUBLIC_URL/api/v1/health")

# 鉴权图片端点在没有JWT时必须返回401。
TEXTBOOK_IMAGE_UNAUTH=$(curl -so/dev/null -w%{http_code} --insecure     "$PUBLIC_URL/api/v1/lesson-plans/textbooks/00000000-0000-0000-0000-000000000000/image")

# 原公开/uploads直链不得再返回真实图片。
TEXTBOOK_DIRECT_PATH="${TEXTBOOK_PROBE_PATH:-__context15_direct_access_probe__.png}"
TEXTBOOK_DIRECT_STATUS=$(curl -so/dev/null -w%{http_code} --insecure     "$PUBLIC_URL/uploads/textbooks/$TEXTBOOK_DIRECT_PATH")
echo "   Go 直连 /health:     $GO_HEALTH"
echo "   Nginx HTTPS 首页:    $NGINX_HTTPS"
echo "   Nginx HTTPS /health: $NGINX_API"
echo "   课本图片无JWT访问:  $TEXTBOOK_IMAGE_UNAUTH"
echo "   旧课本公开直链:     $TEXTBOOK_DIRECT_STATUS"

if [ "$GO_HEALTH" != "200" ] || [ "$NGINX_HTTPS" != "200" ] || [ "$NGINX_API" != "200" ]; then
    echo "   ❌ 关键端点异常，请人工核查"
    false
fi

if [ "$TEXTBOOK_IMAGE_UNAUTH" != "401" ]; then
    echo "   ❌ 课本图片鉴权端点未正确拒绝无JWT请求"
    false
fi

if [ "$TEXTBOOK_DIRECT_STATUS" = "200" ]; then
    echo "   ❌ 旧/uploads课本直链仍可直接访问"
    false
fi

echo "   ✅ 所有关键端点正常"
echo "   ✅ 课本图片鉴权与公开直链关闭验证通过"

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
