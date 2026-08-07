#!/usr/bin/env python3
"""
修复最近一份成功导入、正文仍含图片占位符的Word保真教案。

默认仅干跑。必须显式传入 --apply 才会写文件和数据库。
"""

from __future__ import annotations

import argparse
import json
import subprocess
import sys
import uuid
from pathlib import Path

from repair_recent_word_import_images_lib import (
    build_repair_plan,
    cleanup_asset_files,
    decode_base64_text,
    file_sha256,
    resolve_safe_docx,
    sql_text,
    write_asset_files,
)


PROJECT_ROOT = Path("/www/wwwroot/tedna")
WORD_ROOT = (
    PROJECT_ROOT
    / "private/lesson-plan-word"
)
ASSET_ROOT = (
    PROJECT_ROOT
    / "uploads/lesson-plans"
)


def run_psql(
    sql: str,
) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        [
            "sudo",
            "-u",
            "postgres",
            "psql",
            "-d",
            "tedna",
            "-v",
            "ON_ERROR_STOP=1",
            "-tA",
            "-F",
            "\t",
            "-c",
            sql,
        ],
        text=True,
        capture_output=True,
        check=False,
    )


def run_psql_script(
    sql: str,
) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        [
            "sudo",
            "-u",
            "postgres",
            "psql",
            "-d",
            "tedna",
            "-v",
            "ON_ERROR_STOP=1",
        ],
        input=sql,
        text=True,
        capture_output=True,
        check=False,
    )


def load_candidate(lesson_plan_id: str) -> dict[str, object]:
    lesson_plan_id = str(uuid.UUID(lesson_plan_id))
    query = f"""
SELECT
    word_document.id::text,
    word_document.lesson_plan_id::text,
    word_document.current_storage_key,
    word_document.current_file_sha256,
    word_document.version::text,
    replace(
        encode(
            convert_to(
                word_document.structure_json::text,
                'UTF8'
            ),
            'base64'
        ),
        E'\n',
        ''
    ),
    replace(
        encode(
            convert_to(
                word_document.semantic_markdown,
                'UTF8'
            ),
            'base64'
        ),
        E'\n',
        ''
    ),
    lesson_plan.author_id::text,
    lesson_plan.education_domain,
    word_document.status,
    (
        SELECT COUNT(*)
        FROM lesson_plan_assets asset
        WHERE asset.lesson_plan_id =
            word_document.lesson_plan_id
    )::text,
    (
        SELECT COUNT(*)
        FROM lesson_plan_word_document_versions version_snapshot
        WHERE version_snapshot.lesson_plan_id =
            word_document.lesson_plan_id
    )::text
FROM lesson_plan_word_documents word_document
INNER JOIN lesson_plans lesson_plan
    ON lesson_plan.id =
       word_document.lesson_plan_id
WHERE lesson_plan.deleted_at IS NULL
  AND word_document.lesson_plan_id = '{lesson_plan_id}'::uuid
  AND word_document.status = 'active'
  AND word_document.semantic_markdown
      LIKE '%[图片：%'
  AND lesson_plan.content_markdown =
      word_document.semantic_markdown
  AND word_document.created_at >=
      TIMESTAMPTZ
      '2026-08-05 09:30:00+08'
ORDER BY word_document.created_at DESC
LIMIT 1;
"""

    result = run_psql(query)

    if result.returncode != 0:
        raise RuntimeError(
            "查询待修复Word教案失败：\n"
            + result.stdout
            + result.stderr
        )

    lines = [
        line
        for line in result.stdout.splitlines()
        if line.strip()
    ]

    if not lines:
        raise RuntimeError(
            "未发现满足安全条件的近期图片占位符教案"
        )

    if len(lines) != 1:
        raise RuntimeError(
            "发现多份候选教案，拒绝自动选择："
            f"{len(lines)}"
        )

    fields = lines[0].split("\t")

    if len(fields) != 12:
        raise RuntimeError(
            "候选教案字段数量异常："
            f"{len(fields)}"
        )

    (
        word_document_id,
        lesson_plan_id,
        storage_key,
        current_file_sha256,
        version_text,
        structure_base64,
        semantic_base64,
        author_id,
        education_domain,
        status,
        asset_count_text,
        version_count_text,
    ) = fields

    for identifier in (
        word_document_id,
        lesson_plan_id,
        author_id,
    ):
        uuid.UUID(identifier)

    if education_domain not in {
        "k12",
        "vocational",
        "adult",
    }:
        raise RuntimeError(
            f"候选教案教育域无效：{education_domain}"
        )

    if status != "active":
        raise RuntimeError(
            f"候选Word文档状态不是active：{status}"
        )

    if (
        len(current_file_sha256) != 64
        or any(
            character not in "0123456789abcdef"
            for character in current_file_sha256
        )
    ):
        raise RuntimeError(
            "候选Word文件哈希无效"
        )

    return {
        "word_document_id":
            word_document_id,
        "lesson_plan_id":
            lesson_plan_id,
        "storage_key":
            storage_key,
        "current_file_sha256":
            current_file_sha256,
        "version":
            int(version_text),
        "structure_json":
            decode_base64_text(
                structure_base64,
            ),
        "semantic_markdown":
            decode_base64_text(
                semantic_base64,
            ),
        "author_id":
            author_id,
        "education_domain":
            education_domain,
        "asset_count":
            int(asset_count_text),
        "version_count":
            int(version_count_text),
    }


def build_asset_insert_sql(
    candidate: dict[str, object],
    repair_plan,
) -> str:
    statements: list[str] = []

    for asset in repair_plan.assets:
        statements.append(
            """
    INSERT INTO lesson_plan_assets (
        lesson_plan_id,
        uploader_id,
        asset_type,
        file_name,
        file_path,
        file_size,
        mime_type,
        alt_text,
        width,
        height
    )
    VALUES (
        '{lesson_plan_id}'::uuid,
        '{author_id}'::uuid,
        'image',
        {file_name},
        {file_path},
        {file_size},
        {mime_type},
        {alt_text},
        0,
        0
    );
""".format(
                lesson_plan_id=
                    candidate[
                        "lesson_plan_id"
                    ],
                author_id=
                    candidate[
                        "author_id"
                    ],
                file_name=
                    sql_text(
                        asset.original_name,
                    ),
                file_path=
                    sql_text(
                        asset.relative_path,
                    ),
                file_size=
                    len(asset.data),
                mime_type=
                    sql_text(
                        asset.mime_type,
                    ),
                alt_text=
                    sql_text(
                        asset.alt_text,
                    ),
            )
        )

    return "".join(statements)


def apply_database_repair(
    candidate: dict[str, object],
    repair_plan,
) -> None:
    structure_expression = sql_text(
        repair_plan.structure_json,
    )

    semantic_expression = sql_text(
        repair_plan.semantic_markdown,
    )

    old_semantic_expression = sql_text(
        str(
            candidate[
                "semantic_markdown"
            ],
        ),
    )

    storage_key_expression = sql_text(
        str(
            candidate[
                "storage_key"
            ],
        ),
    )

    asset_sql = build_asset_insert_sql(
        candidate,
        repair_plan,
    )

    sql = f"""
\\set ON_ERROR_STOP on

BEGIN;

DO $repair$
DECLARE
    stored_word_version INTEGER;
    stored_word_status TEXT;
    stored_word_storage_key TEXT;
    stored_word_file_sha256 TEXT;
    stored_word_semantic TEXT;
    stored_plan_content TEXT;
    stored_asset_count BIGINT;
    old_snapshot_count BIGINT;
    new_snapshot_count BIGINT;
BEGIN
    SELECT
        word_document.version,
        word_document.status,
        word_document.current_storage_key,
        word_document.current_file_sha256,
        word_document.semantic_markdown,
        lesson_plan.content_markdown
    INTO
        stored_word_version,
        stored_word_status,
        stored_word_storage_key,
        stored_word_file_sha256,
        stored_word_semantic,
        stored_plan_content
    FROM lesson_plan_word_documents word_document
    INNER JOIN lesson_plans lesson_plan
        ON lesson_plan.id =
           word_document.lesson_plan_id
    WHERE word_document.id =
        '{candidate["word_document_id"]}'::uuid
      AND word_document.lesson_plan_id =
        '{candidate["lesson_plan_id"]}'::uuid
      AND lesson_plan.author_id =
        '{candidate["author_id"]}'::uuid
      AND lesson_plan.education_domain =
        '{candidate["education_domain"]}'
      AND lesson_plan.deleted_at IS NULL
    FOR UPDATE OF
        word_document,
        lesson_plan;

    IF NOT FOUND THEN
        RAISE EXCEPTION
            '待修复Word教案不存在或权限快照不匹配';
    END IF;

    IF stored_word_version <>
           {candidate["version"]}
        OR stored_word_status <> 'active'
        OR stored_word_storage_key <>
           {storage_key_expression}
        OR stored_word_file_sha256 <>
           '{candidate["current_file_sha256"]}'
        OR stored_word_semantic
           IS DISTINCT FROM stored_plan_content
        OR stored_word_semantic
           IS DISTINCT FROM
           {old_semantic_expression}
        OR stored_word_semantic
           NOT LIKE '%[图片：%' THEN

        RAISE EXCEPTION
            '待修复Word教案已经发生变化，拒绝覆盖';
    END IF;

    SELECT COUNT(*)
    INTO stored_asset_count
    FROM lesson_plan_assets
    WHERE lesson_plan_id =
        '{candidate["lesson_plan_id"]}'::uuid;

    IF stored_asset_count <>
       {candidate["asset_count"]} THEN
        RAISE EXCEPTION
            '教案资产数量已经发生变化，拒绝覆盖';
    END IF;

    SELECT COUNT(*)
    INTO old_snapshot_count
    FROM lesson_plan_word_document_versions
    WHERE lesson_plan_id =
        '{candidate["lesson_plan_id"]}'::uuid
      AND version =
        {candidate["version"]}
      AND storage_key =
        {storage_key_expression}
      AND file_sha256 =
        '{candidate["current_file_sha256"]}';

    IF old_snapshot_count <> 1 THEN
        RAISE EXCEPTION
            '当前Word版本快照不完整，拒绝继续补偿';
    END IF;

{asset_sql}

    -- 先更新平台语义正文。
    -- stale触发器会临时把Word当前文档标记为stale。
    UPDATE lesson_plans
    SET
        content_markdown =
            {semantic_expression},
        updated_at = NOW()
    WHERE id =
        '{candidate["lesson_plan_id"]}'::uuid
      AND author_id =
        '{candidate["author_id"]}'::uuid
      AND deleted_at IS NULL
      AND content_markdown =
        {old_semantic_expression};

    IF NOT FOUND THEN
        RAISE EXCEPTION
            '更新教案图片语义正文失败';
    END IF;

    -- 递增Word版本会自动触发不可变版本快照。
    UPDATE lesson_plan_word_documents
    SET
        version = version + 1,
        status = 'active',
        structure_json =
            {structure_expression}::jsonb,
        semantic_markdown =
            {semantic_expression},
        semantic_markdown_hash =
            encode(
                digest(
                    convert_to(
                        {semantic_expression},
                        'UTF8'
                    ),
                    'sha256'
                ),
                'hex'
            ),
        structure_hash =
            encode(
                digest(
                    convert_to(
                        (
                            {structure_expression}::jsonb
                        )::text,
                        'UTF8'
                    ),
                    'sha256'
                ),
                'hex'
            ),
        last_change_source = 'system',
        last_changed_by =
            '{candidate["author_id"]}'::uuid,
        last_change_summary =
            '补全原Word图片资产和网页正文图片',
        error_message = '',
        generated_at = NOW()
    WHERE id =
        '{candidate["word_document_id"]}'::uuid
      AND version =
        {candidate["version"]};

    IF NOT FOUND THEN
        RAISE EXCEPTION
            '更新Word当前文档失败';
    END IF;

    SELECT COUNT(*)
    INTO new_snapshot_count
    FROM lesson_plan_word_document_versions
    WHERE lesson_plan_id =
        '{candidate["lesson_plan_id"]}'::uuid
      AND version =
        {candidate["version"] + 1};

    IF new_snapshot_count <> 1 THEN
        RAISE EXCEPTION
            '新Word不可变版本快照未自动生成';
    END IF;
END
$repair$;

COMMIT;
"""

    result = run_psql_script(sql)

    if result.returncode != 0:
        raise RuntimeError(
            "数据库图片补偿事务失败：\n"
            + result.stdout
            + result.stderr
        )


def verify_repair(
    candidate: dict[str, object],
    repair_plan,
) -> None:
    query = f"""
SELECT
    word_document.version::text,
    word_document.status,
    (
        word_document.semantic_markdown =
        lesson_plan.content_markdown
    )::text,
    (
        (
            char_length(
                lesson_plan.content_markdown
            )
            -
            char_length(
                replace(
                    lesson_plan.content_markdown,
                    '[图片：',
                    ''
                )
            )
        )
        /
        char_length('[图片：')
    )::text,
    (
        (
            char_length(
                lesson_plan.content_markdown
            )
            -
            char_length(
                replace(
                    lesson_plan.content_markdown,
                    '![',
                    ''
                )
            )
        )
        /
        char_length('![')
    )::text,
    (
        SELECT COUNT(*)
        FROM lesson_plan_assets asset
        WHERE asset.lesson_plan_id =
            word_document.lesson_plan_id
    )::text,
    (
        SELECT COUNT(*)
        FROM lesson_plan_word_document_versions version_snapshot
        WHERE version_snapshot.lesson_plan_id =
            word_document.lesson_plan_id
    )::text,
    (
        SELECT COUNT(*)
        FROM lesson_plan_word_document_versions version_snapshot
        WHERE version_snapshot.lesson_plan_id =
                word_document.lesson_plan_id
          AND version_snapshot.version =
                word_document.version
          AND version_snapshot.storage_key =
                word_document.current_storage_key
          AND version_snapshot.file_sha256 =
                word_document.current_file_sha256
          AND version_snapshot.structure_json =
                word_document.structure_json
          AND version_snapshot.semantic_markdown =
                word_document.semantic_markdown
          AND version_snapshot.semantic_markdown_hash =
                word_document.semantic_markdown_hash
          AND version_snapshot.structure_hash =
                word_document.structure_hash
    )::text
FROM lesson_plan_word_documents word_document
INNER JOIN lesson_plans lesson_plan
    ON lesson_plan.id =
       word_document.lesson_plan_id
WHERE word_document.id =
    '{candidate["word_document_id"]}'::uuid;
"""

    result = run_psql(query)

    if result.returncode != 0:
        raise RuntimeError(
            "补偿后数据库验证失败：\n"
            + result.stdout
            + result.stderr
        )

    fields = result.stdout.strip().split("\t")

    if len(fields) != 8:
        raise RuntimeError(
            "补偿后验证字段数量异常："
            f"{len(fields)}"
        )

    (
        version_text,
        status,
        synchronized_text,
        placeholder_count_text,
        markdown_image_count_text,
        asset_count_text,
        version_count_text,
        snapshot_match_count_text,
    ) = fields

    expected_version = int(
        candidate["version"],
    ) + 1

    expected_asset_count = int(
        candidate["asset_count"],
    ) + len(repair_plan.assets)

    expected_version_count = int(
        candidate["version_count"],
    ) + 1

    if int(version_text) != expected_version:
        raise RuntimeError(
            "Word当前版本号验证失败"
        )

    if status != "active":
        raise RuntimeError(
            f"Word当前状态验证失败：{status}"
        )

    if synchronized_text != "true":
        raise RuntimeError(
            "平台正文与Word语义正文仍未同步"
        )

    if int(asset_count_text) != expected_asset_count:
        raise RuntimeError(
            "教案图片资产数量验证失败"
        )

    if int(version_count_text) != expected_version_count:
        raise RuntimeError(
            "Word不可变版本数量验证失败"
        )

    if int(snapshot_match_count_text) != 1:
        raise RuntimeError(
            "Word最新不可变快照内容不匹配"
        )

    if int(markdown_image_count_text) < (
        repair_plan.replacement_count
    ):
        raise RuntimeError(
            "正文Markdown图片数量少于替换数量"
        )

    print(
        "   补偿后验证通过："
        f"Word版本={version_text}，"
        f"状态={status}，"
        f"图片资产总数={asset_count_text}，"
        f"Markdown图片={markdown_image_count_text}，"
        f"剩余占位符={placeholder_count_text}，"
        f"Word快照总数={version_count_text}"
    )


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--lesson-plan-id", required=True)
    parser.add_argument(
        "--apply",
        action="store_true",
        help="实际写入图片文件和数据库",
    )

    args = parser.parse_args()

    candidate = load_candidate(args.lesson_plan_id)

    structure_document = json.loads(
        str(candidate["structure_json"]),
    )

    docx_path = resolve_safe_docx(
        WORD_ROOT,
        str(candidate["storage_key"]),
    )

    actual_file_sha256 = file_sha256(
        docx_path,
    )

    if actual_file_sha256 != str(
        candidate["current_file_sha256"],
    ):
        raise RuntimeError(
            "私有DOCX文件哈希与数据库快照不一致"
        )

    repair_plan = build_repair_plan(
        docx_path,
        structure_document,
        str(candidate["lesson_plan_id"]),
        ASSET_ROOT,
    )

    print(
        "   候选教案："
        f"{candidate['lesson_plan_id']}"
    )

    print(
        "   Word当前版本："
        f"{candidate['version']}"
    )

    print(
        "   图片分析："
        f"引用关系={repair_plan.referenced_relationship_count}，"
        f"唯一图片资产={len(repair_plan.assets)}，"
        f"正文替换={repair_plan.replacement_count}，"
        f"缺失或外部={repair_plan.skipped_missing}，"
        f"不支持格式={repair_plan.skipped_unsupported}，"
        f"超过5MB={repair_plan.skipped_oversized}"
    )

    if not args.apply:
        print(
            "   干跑完成：未创建文件，未修改数据库"
        )
        return 0

    digest_check = run_psql(
        """
SELECT
    encode(
        digest(
            convert_to(
                'word-image-repair-check',
                'UTF8'
            ),
            'sha256'
        ),
        'hex'
    );
"""
    )

    if digest_check.returncode != 0:
        raise RuntimeError(
            "数据库缺少SHA-256摘要能力："
            + digest_check.stderr
        )

    created_paths: list[Path] = []
    database_committed = False

    try:
        created_paths = write_asset_files(
            ASSET_ROOT,
            str(candidate["lesson_plan_id"]),
            repair_plan.assets,
        )

        apply_database_repair(
            candidate,
            repair_plan,
        )

        database_committed = True

        verify_repair(
            candidate,
            repair_plan,
        )

        current_docx_hash = file_sha256(
            docx_path,
        )

        if current_docx_hash != actual_file_sha256:
            raise RuntimeError(
                "补偿过程中原始DOCX文件发生变化"
            )

        for asset in repair_plan.assets:
            if (
                not asset.full_path.is_file()
                or asset.full_path.stat().st_size
                   != len(asset.data)
            ):
                raise RuntimeError(
                    f"补偿图片文件验证失败：{asset.full_path}"
                )

        print(
            "   当前Word教案图片补偿成功："
            f"{candidate['lesson_plan_id']}"
        )

        return 0

    except BaseException:
        if not database_committed:
            cleanup_asset_files(
                created_paths,
            )
        else:
            print(
                "数据库事务已经提交；"
                "为避免资产记录指向缺失文件，"
                "异常后保留已创建图片文件",
                file=sys.stderr,
            )

        raise


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except BaseException as error:
        print(
            f"错误：{error}",
            file=sys.stderr,
        )
        raise
