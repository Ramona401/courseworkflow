#!/usr/bin/env python3
"""Word保真教案历史图片补偿辅助能力。"""

from __future__ import annotations

import base64
import copy
import hashlib
import json
import os
import posixpath
import stat
import uuid
import zipfile
from dataclasses import dataclass
from pathlib import Path, PurePosixPath
from typing import Any


MAX_ENTRIES = 2000
MAX_TOTAL_UNCOMPRESSED = 160 * 1024 * 1024
MAX_ENTRY_SIZE = 64 * 1024 * 1024
MAX_IMAGE_SIZE = 5 * 1024 * 1024
MAX_COMPRESSION_RATIO = 500


@dataclass
class ExtractedAsset:
    target: str
    original_name: str
    alt_text: str
    mime_type: str
    extension: str
    data: bytes
    stored_name: str
    relative_path: str
    full_path: Path
    url: str


@dataclass
class RepairPlan:
    document: dict[str, Any]
    structure_json: str
    semantic_markdown: str
    assets: list[ExtractedAsset]
    replacement_count: int
    referenced_relationship_count: int
    skipped_missing: int
    skipped_unsupported: int
    skipped_oversized: int


def decode_base64_text(value: str) -> str:
    return base64.b64decode(value).decode("utf-8")


def sql_text(value: str) -> str:
    encoded = base64.b64encode(
        value.encode("utf-8"),
    ).decode("ascii")

    return (
        "convert_from("
        f"decode('{encoded}', 'base64'),"
        "'UTF8'"
        ")"
    )


def file_sha256(path: Path) -> str:
    digest = hashlib.sha256()

    with path.open("rb") as stream:
        while True:
            chunk = stream.read(1024 * 1024)
            if not chunk:
                break
            digest.update(chunk)

    return digest.hexdigest()


def detect_image(
    data: bytes,
) -> tuple[str, str] | None:
    if data.startswith(b"\x89PNG\r\n\x1a\n"):
        return "image/png", ".png"

    if data.startswith(b"\xff\xd8\xff"):
        return "image/jpeg", ".jpg"

    if data.startswith(b"GIF87a") or data.startswith(b"GIF89a"):
        return "image/gif", ".gif"

    if (
        len(data) >= 12
        and data[:4] == b"RIFF"
        and data[8:12] == b"WEBP"
    ):
        return "image/webp", ".webp"

    return None


def sanitize_alt(value: str) -> str:
    value = (
        value
        .replace("[", "_")
        .replace("]", "_")
        .replace("\r", " ")
        .replace("\n", " ")
        .strip()
    )

    return value[:100] or "Word图片"


def render_text_run_markdown(
    run: dict[str, Any],
) -> str:
    text = str(run.get("text", ""))

    vertical_align = str(
        run.get("vertical_align", ""),
    ).lower()

    if vertical_align == "superscript":
        text = "^{" + text + "}"
    elif vertical_align == "subscript":
        text = "_{" + text + "}"

    if bool(run.get("bold")):
        text = "**" + text + "**"
    elif bool(run.get("italic")):
        text = "*" + text + "*"

    return text


def validate_storage_key(
    storage_key: str,
) -> tuple[str, ...]:
    storage_key = storage_key.strip()

    if (
        not storage_key
        or "\\" in storage_key
        or storage_key.startswith("/")
        or "//" in storage_key
    ):
        raise RuntimeError(
            f"私有DOCX存储键无效：{storage_key}"
        )

    path_value = PurePosixPath(storage_key)

    if (
        path_value.is_absolute()
        or "." in path_value.parts
        or ".." in path_value.parts
        or str(path_value) != storage_key
    ):
        raise RuntimeError(
            f"私有DOCX存储键包含越界片段：{storage_key}"
        )

    return path_value.parts


def resolve_safe_docx(
    word_root: Path,
    storage_key: str,
) -> Path:
    if (
        not word_root.exists()
        or not word_root.is_dir()
        or word_root.is_symlink()
    ):
        raise RuntimeError(
            f"Word私有根目录无效：{word_root}"
        )

    parts = validate_storage_key(storage_key)

    current = word_root

    for part in parts:
        current = current / part

        if current.is_symlink():
            raise RuntimeError(
                f"私有DOCX路径包含符号链接：{current}"
            )

    try:
        resolved_root = word_root.resolve(strict=True)
        resolved_file = current.resolve(strict=True)
    except FileNotFoundError as error:
        raise RuntimeError(
            f"私有DOCX不存在：{current}"
        ) from error

    if (
        resolved_file == resolved_root
        or resolved_root not in resolved_file.parents
        or not resolved_file.is_file()
    ):
        raise RuntimeError(
            f"私有DOCX路径越界或不是普通文件：{resolved_file}"
        )

    return resolved_file


def validate_zip(
    archive: zipfile.ZipFile,
) -> dict[str, zipfile.ZipInfo]:
    infos = archive.infolist()

    if not infos or len(infos) > MAX_ENTRIES:
        raise RuntimeError("DOCX条目数量异常")

    entries: dict[str, zipfile.ZipInfo] = {}
    total_uncompressed = 0

    for info in infos:
        raw_name = info.filename

        if "\\" in raw_name:
            raise RuntimeError(
                f"DOCX包含反斜线路径：{raw_name}"
            )

        name = raw_name
        clean_input = name.rstrip("/")
        normalized = posixpath.normpath(clean_input)

        if (
            not name
            or not clean_input
            or "\x00" in name
            or name.startswith("/")
            or normalized in {".", ".."}
            or normalized.startswith("../")
            or normalized != clean_input
        ):
            raise RuntimeError(
                f"DOCX包含不安全路径：{name}"
            )

        unix_mode = info.external_attr >> 16

        if stat.S_IFMT(unix_mode) == stat.S_IFLNK:
            raise RuntimeError(
                f"DOCX包含符号链接：{name}"
            )

        if info.flag_bits & 0x1:
            raise RuntimeError(
                f"DOCX包含加密部件：{name}"
            )

        if name in entries:
            raise RuntimeError(
                f"DOCX包含重复部件：{name}"
            )

        if info.file_size > MAX_ENTRY_SIZE:
            raise RuntimeError(
                f"DOCX部件过大：{name}"
            )

        if (
            info.compress_size > 0
            and info.file_size > 1024 * 1024
            and (
                info.file_size
                // info.compress_size
            ) > MAX_COMPRESSION_RATIO
        ):
            raise RuntimeError(
                f"DOCX压缩比异常：{name}"
            )

        total_uncompressed += info.file_size

        if total_uncompressed > MAX_TOTAL_UNCOMPRESSED:
            raise RuntimeError(
                "DOCX解压后总体积过大"
            )

        entries[name] = info

    for required_name in (
        "[Content_Types].xml",
        "word/document.xml",
    ):
        if required_name not in entries:
            raise RuntimeError(
                f"DOCX缺少必要部件：{required_name}"
            )

    return entries


def normalize_media_target(target: str) -> str:
    target = target.strip()

    if (
        not target
        or "\\" in target
        or target.startswith("/")
        or "//" in target
    ):
        raise RuntimeError(
            f"Word媒体路径无效：{target}"
        )

    normalized = posixpath.normpath(target)

    if (
        normalized in {".", ".."}
        or normalized.startswith("../")
        or normalized != target
        or not normalized.startswith("word/media/")
    ):
        raise RuntimeError(
            f"Word媒体路径越界：{target}"
        )

    return normalized


def build_table_semantic_markdown(
    document: dict[str, Any],
    table_index: int,
    block_map: dict[str, dict[str, Any]],
    depth: int,
) -> str:
    tables = document.get("tables", [])

    if (
        not isinstance(tables, list)
        or table_index < 0
        or table_index >= len(tables)
        or depth > 6
    ):
        return ""

    table = tables[table_index]

    if not isinstance(table, dict):
        return ""

    parts: list[str] = []

    for row in table.get("rows", []):
        if not isinstance(row, dict):
            continue

        cells = row.get("cells", [])

        if not isinstance(cells, list):
            continue

        cell_markdown: list[str] = []

        for cell in cells:
            if not isinstance(cell, dict):
                cell_markdown.append("")
                continue

            paragraphs: list[str] = []

            for block_id in cell.get("block_ids", []):
                block = block_map.get(str(block_id))

                if not isinstance(block, dict):
                    continue

                markdown = str(
                    block.get("markdown", ""),
                ).strip()

                if markdown:
                    paragraphs.append(markdown)

            for nested_index in cell.get(
                "nested_table_indices",
                [],
            ):
                try:
                    nested_table_index = int(nested_index)
                except (TypeError, ValueError):
                    continue

                nested_markdown = (
                    build_table_semantic_markdown(
                        document,
                        nested_table_index,
                        block_map,
                        depth + 1,
                    )
                )

                if nested_markdown:
                    paragraphs.append(nested_markdown)

            cell_markdown.append(
                "\n".join(paragraphs),
            )

        if len(cell_markdown) == 1:
            content = cell_markdown[0].strip()

            if content:
                parts.append(content)

            continue

        if len(cell_markdown) == 2:
            label = cell_markdown[0].strip()
            content = cell_markdown[1].strip()

            if label and len(label) <= 40:
                parts.append("## " + label)

                if content:
                    parts.append(content)

                continue

        row_lines: list[str] = []

        for cell_index, content in enumerate(
            cell_markdown,
            start=1,
        ):
            content = content.strip()

            if content:
                row_lines.append(
                    f"- 第{cell_index}列：{content}"
                )

        if row_lines:
            table_number = int(
                table.get("index", table_index),
            ) + 1

            row_number = int(
                row.get("index", 0),
            ) + 1

            parts.append(
                f"### 表格{table_number} · 第{row_number}行"
            )

            parts.append(
                "\n".join(row_lines),
            )

    return "\n\n".join(parts)


def build_semantic_markdown(
    document: dict[str, Any],
) -> str:
    blocks = document.get("blocks", [])

    block_map = {
        str(block.get("id", "")): block
        for block in blocks
        if isinstance(block, dict)
        and str(block.get("id", "")).strip()
    }

    parts: list[str] = []

    for item in document.get("flow", []):
        if not isinstance(item, dict):
            continue

        kind = str(item.get("kind", ""))

        if kind == "block":
            block = block_map.get(
                str(item.get("block_id", "")),
            )

            if isinstance(block, dict):
                markdown = str(
                    block.get("markdown", ""),
                ).strip()

                if markdown:
                    parts.append(markdown)

            continue

        if kind == "table":
            try:
                table_index = int(
                    item.get("table_index", -1),
                )
            except (TypeError, ValueError):
                continue

            markdown = build_table_semantic_markdown(
                document,
                table_index,
                block_map,
                0,
            )

            if markdown:
                parts.append(markdown)

    return "\n\n".join(parts).strip()


def build_repair_plan(
    docx_path: Path,
    source_document: dict[str, Any],
    lesson_plan_id: str,
    asset_root: Path,
) -> RepairPlan:
    document = copy.deepcopy(source_document)

    media_by_relationship = {
        str(
            item.get("relationship_id", ""),
        ).strip(): item
        for item in document.get("media", [])
        if isinstance(item, dict)
        and str(
            item.get("relationship_id", ""),
        ).strip()
    }

    referenced_relationships = sorted({
        str(
            run.get("relationship_id", ""),
        ).strip()
        for block in document.get("blocks", [])
        if isinstance(block, dict)
        for run in block.get("runs", [])
        if isinstance(run, dict)
        and run.get("kind") == "image"
        and str(
            run.get("relationship_id", ""),
        ).strip()
    })

    if not referenced_relationships:
        raise RuntimeError(
            "Word结构中没有可补偿的图片引用"
        )

    relationship_urls: dict[str, str] = {}
    asset_by_target: dict[str, ExtractedAsset] = {}

    skipped_missing = 0
    skipped_unsupported = 0
    skipped_oversized = 0

    with zipfile.ZipFile(docx_path, "r") as archive:
        entries = validate_zip(archive)

        for relationship_id in referenced_relationships:
            media = media_by_relationship.get(
                relationship_id,
            )

            if not isinstance(media, dict):
                skipped_missing += 1
                continue

            if (
                bool(media.get("missing"))
                or str(
                    media.get("target_mode", ""),
                ).lower() == "external"
            ):
                skipped_missing += 1
                continue

            target = normalize_media_target(
                str(media.get("target", "")),
            )

            info = entries.get(target)

            if info is None:
                skipped_missing += 1
                continue

            if info.file_size > MAX_IMAGE_SIZE:
                skipped_oversized += 1
                continue

            existing_asset = asset_by_target.get(target)

            if existing_asset is not None:
                relationship_urls[relationship_id] = (
                    existing_asset.url
                )
                continue

            data = archive.read(target)
            detected = detect_image(data)

            if detected is None:
                skipped_unsupported += 1
                continue

            mime_type, extension = detected
            original_name = PurePosixPath(target).name
            alt_text = sanitize_alt(original_name)

            digest = hashlib.sha256(data).hexdigest()

            stored_name = (
                f"word_{digest[:16]}_"
                f"{uuid.uuid4()}{extension}"
            )

            relative_path = (
                f"{lesson_plan_id}/{stored_name}"
            )

            full_path = (
                asset_root
                / lesson_plan_id
                / stored_name
            )

            url = (
                "/uploads/lesson-plans/"
                + relative_path
            )

            asset = ExtractedAsset(
                target=target,
                original_name=original_name,
                alt_text=alt_text,
                mime_type=mime_type,
                extension=extension,
                data=data,
                stored_name=stored_name,
                relative_path=relative_path,
                full_path=full_path,
                url=url,
            )

            asset_by_target[target] = asset
            relationship_urls[relationship_id] = url

    replacement_count = 0

    for block in document.get("blocks", []):
        if not isinstance(block, dict):
            continue

        runs = block.get("runs", [])

        if (
            not isinstance(runs, list)
            or not any(
                isinstance(run, dict)
                and run.get("kind") == "image"
                for run in runs
            )
        ):
            continue

        markdown_parts: list[str] = []

        for run in runs:
            if not isinstance(run, dict):
                continue

            kind = str(run.get("kind", ""))

            if kind == "text":
                markdown_parts.append(
                    render_text_run_markdown(run),
                )
                continue

            if kind == "formula":
                formula_id = str(
                    run.get("formula_id", ""),
                ).upper()

                formula_text = str(
                    run.get("text", ""),
                )

                markdown_parts.append(
                    "{{"
                    + formula_id
                    + ":"
                    + formula_text
                    + "}}"
                )
                continue

            if kind != "image":
                continue

            relationship_id = str(
                run.get("relationship_id", ""),
            ).strip()

            target = str(
                run.get("media_target", ""),
            ).strip()

            label = (
                sanitize_alt(
                    PurePosixPath(target).name,
                )
                if target
                else "图片"
            )

            image_url = relationship_urls.get(
                relationship_id,
                "",
            )

            if not image_url:
                markdown_parts.append(
                    f"[图片：{label}]"
                )
                continue

            markdown_parts.append(
                f"![{label}]({image_url})"
            )

            replacement_count += 1

        block["markdown"] = "".join(
            markdown_parts,
        ).strip()

    semantic_markdown = build_semantic_markdown(
        document,
    )

    if (
        not semantic_markdown
        or replacement_count <= 0
        or "![" not in semantic_markdown
    ):
        raise RuntimeError(
            "未能从Word结构生成包含图片的语义正文"
        )

    structure_json = json.dumps(
        document,
        ensure_ascii=False,
        separators=(",", ":"),
    )

    return RepairPlan(
        document=document,
        structure_json=structure_json,
        semantic_markdown=semantic_markdown,
        assets=list(asset_by_target.values()),
        replacement_count=replacement_count,
        referenced_relationship_count=len(
            referenced_relationships,
        ),
        skipped_missing=skipped_missing,
        skipped_unsupported=skipped_unsupported,
        skipped_oversized=skipped_oversized,
    )


def write_asset_files(
    asset_root: Path,
    lesson_plan_id: str,
    assets: list[ExtractedAsset],
) -> list[Path]:
    if (
        not asset_root.exists()
        or not asset_root.is_dir()
        or asset_root.is_symlink()
    ):
        raise RuntimeError(
            f"教案图片根目录无效：{asset_root}"
        )

    plan_directory = asset_root / lesson_plan_id

    if plan_directory.exists():
        if (
            not plan_directory.is_dir()
            or plan_directory.is_symlink()
        ):
            raise RuntimeError(
                f"教案图片目录无效：{plan_directory}"
            )
    else:
        plan_directory.mkdir(
            mode=0o755,
            parents=False,
            exist_ok=False,
        )

    os.chmod(plan_directory, 0o755)

    created: list[Path] = []

    try:
        for asset in assets:
            if asset.full_path.parent != plan_directory:
                raise RuntimeError(
                    f"图片目标路径越界：{asset.full_path}"
                )

            with asset.full_path.open("xb") as stream:
                stream.write(asset.data)
                stream.flush()
                os.fsync(stream.fileno())

            os.chmod(asset.full_path, 0o644)
            created.append(asset.full_path)

        return created
    except BaseException:
        cleanup_asset_files(created)

        try:
            plan_directory.rmdir()
        except OSError:
            pass

        raise


def cleanup_asset_files(
    paths: list[Path],
) -> None:
    parent_directories: set[Path] = set()

    for path in paths:
        parent_directories.add(path.parent)

        try:
            path.unlink()
        except FileNotFoundError:
            pass

    for directory in parent_directories:
        try:
            directory.rmdir()
        except OSError:
            pass
