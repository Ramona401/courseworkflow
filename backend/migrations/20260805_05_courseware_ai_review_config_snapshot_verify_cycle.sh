#!/bin/bash
# ============================================================================
# TE-DNA 2.0：R-02数据库完整迁移周期验证
# 文件：20260805_05_courseware_ai_review_config_snapshot_verify_cycle.sh
# ----------------------------------------------------------------------------
# 安全规则：
#   - 启用严格错误退出和管道错误传播；
#   - 任一步失败立即停止；
#   - 验证失败时保留临时数据库；
#   - 只有正向迁移、完整验证、回滚、重新迁移和再次验证全部通过后，
#     才执行生产库只读验证；
#   - 本脚本不会重新迁移或回滚生产数据库。
# ============================================================================

set -Eeuo pipefail
umask 077

PROJECT_ROOT="/www/wwwroot/tedna"
MIGRATION_DIR="$PROJECT_ROOT/backend/migrations"
DB_NAME="tedna"

MIGRATION_FILE="$MIGRATION_DIR/20260805_05_courseware_ai_review_config_snapshot.sql"
VERIFY_FILE="$MIGRATION_DIR/20260805_05_courseware_ai_review_config_snapshot_verify.sql"
PRODUCTION_VERIFY_FILE="$MIGRATION_DIR/20260805_05_courseware_ai_review_config_snapshot_production_verify.sql"
ROLLBACK_FILE="$MIGRATION_DIR/20260805_05_courseware_ai_review_config_snapshot_rollback.sql"

DB_BACKUP_FILE="${1:-}"

if [ -z "$DB_BACKUP_FILE" ]; then
    echo "❌ 缺少迁移前数据库备份路径"
    echo "使用方式：$0 /完整路径/tedna_pre_r02_时间戳.sql.gz"
    exit 1
fi

if [ ! -s "$DB_BACKUP_FILE" ]; then
    echo "❌ 数据库备份不存在或为空：$DB_BACKUP_FILE"
    exit 1
fi

for required_file in \
    "$MIGRATION_FILE" \
    "$VERIFY_FILE" \
    "$PRODUCTION_VERIFY_FILE" \
    "$ROLLBACK_FILE"
do
    if [ ! -s "$required_file" ]; then
        echo "❌ 必需文件不存在或为空：$required_file"
        exit 1
    fi
done

TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
TEMP_DB="tedna_r02_reverify_${TIMESTAMP}"
TEMP_DB_CREATED=0
SUCCESS=0

on_exit() {
    exit_code=$?

    if [ "$SUCCESS" = "1" ]; then
        exit "$exit_code"
    fi

    echo ""
    echo "❌ R-02完整迁移周期验证未通过"
    echo "   失败退出码：$exit_code"

    if [ "$TEMP_DB_CREATED" = "1" ]; then
        echo "   临时数据库已保留：$TEMP_DB"
        echo "   请勿自行删除，先报告完整错误输出"
    fi

    exit "$exit_code"
}

trap on_exit EXIT

echo "=========R-02完整迁移周期重新验证开始========="
echo "数据库备份：$DB_BACKUP_FILE"
echo "临时数据库：$TEMP_DB"
echo "";

echo "1. 创建临时恢复数据库";
sudo -u postgres createdb "$TEMP_DB";
TEMP_DB_CREATED=1;
echo "✅ 临时数据库创建完成";

echo "";
echo "2. 恢复迁移前生产备份";
gzip -dc "$DB_BACKUP_FILE" |
    sudo -u postgres psql \
        -v ON_ERROR_STOP=1 \
        -d "$TEMP_DB";
echo "✅ 临时数据库恢复完成";

echo "";
echo "3. 临时库执行正向迁移";
sudo -u postgres psql \
    -v ON_ERROR_STOP=1 \
    -d "$TEMP_DB" \
    -f "$MIGRATION_FILE";
echo "✅ 临时库正向迁移成功";

echo "";
echo "4. 临时库执行完整验证";
sudo -u postgres psql \
    -v ON_ERROR_STOP=1 \
    -d "$TEMP_DB" \
    -f "$VERIFY_FILE";
echo "✅ 临时库首次完整验证通过";

echo "";
echo "5. 临时库执行独立回滚";
sudo -u postgres psql \
    -v ON_ERROR_STOP=1 \
    -d "$TEMP_DB" \
    -f "$ROLLBACK_FILE";
echo "✅ 临时库回滚成功";

echo "";
echo "6. 对账回滚后的字段和函数";
ROLLBACK_COLUMN_COUNT=$(
    sudo -u postgres psql \
        -v ON_ERROR_STOP=1 \
        -At \
        -d "$TEMP_DB" \
        -c "
            SELECT COUNT(*)
            FROM information_schema.columns
            WHERE table_schema = 'public'
              AND table_name =
                  'courseware_ai_review_sessions'
              AND column_name IN (
                  'review_config_schema_version',
                  'review_dimensions_json',
                  'custom_dimension_description',
                  'lesson_reference_mode',
                  'review_config_hash'
              );
        "
);

if [ "$ROLLBACK_COLUMN_COUNT" != "0" ]; then
    echo "❌ 回滚后仍残留R-02字段，数量：$ROLLBACK_COLUMN_COUNT";
    false;
fi;

ROLLBACK_FUNCTION_COUNT=$(
    sudo -u postgres psql \
        -v ON_ERROR_STOP=1 \
        -At \
        -d "$TEMP_DB" \
        -c "
            SELECT
                (
                    CASE
                        WHEN to_regprocedure(
                            'public.normalize_cw_ai_review_dimensions(jsonb)'
                        ) IS NULL
                        THEN 0
                        ELSE 1
                    END
                )
                +
                (
                    CASE
                        WHEN to_regprocedure(
                            'public.is_valid_cw_ai_review_dimensions(jsonb)'
                        ) IS NULL
                        THEN 0
                        ELSE 1
                    END
                )
                +
                (
                    CASE
                        WHEN to_regprocedure(
                            'public.build_cw_ai_review_config_hash(smallint,jsonb,text,text)'
                        ) IS NULL
                        THEN 0
                        ELSE 1
                    END
                );
        "
);

if [ "$ROLLBACK_FUNCTION_COUNT" != "0" ]; then
    echo "❌ 回滚后仍残留R-02函数，数量：$ROLLBACK_FUNCTION_COUNT";
    false;
fi;

echo "✅ 回滚结构对账通过";

echo "";
echo "7. 临时库重新执行正向迁移";
sudo -u postgres psql \
    -v ON_ERROR_STOP=1 \
    -d "$TEMP_DB" \
    -f "$MIGRATION_FILE";
echo "✅ 临时库重新迁移成功";

echo "";
echo "8. 临时库再次执行完整验证";
sudo -u postgres psql \
    -v ON_ERROR_STOP=1 \
    -d "$TEMP_DB" \
    -f "$VERIFY_FILE";
echo "✅ 临时库重新迁移验证通过";

echo "";
echo "9. 生产数据库执行只读验证";
sudo -u postgres psql \
    -v ON_ERROR_STOP=1 \
    -d "$DB_NAME" \
    -f "$PRODUCTION_VERIFY_FILE";
echo "✅ 生产数据库只读验证通过";

echo "";
echo "10. 删除已经完整验证通过的临时数据库";
sudo -u postgres dropdb "$TEMP_DB";
TEMP_DB_CREATED=0;
echo "✅ 临时数据库已删除";

SUCCESS=1;

echo "";
echo "=========R-02完整迁移周期重新验证全部通过=========";
echo "生产数据库未被重新迁移或回滚";
