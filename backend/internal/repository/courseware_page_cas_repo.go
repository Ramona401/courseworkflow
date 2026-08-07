package repository

// courseware_page_cas_repo.go
//
// 课件页面“比较并交换”写回仓储。
//
// 一个事务内完成：
//   1. 获取页面版本序列advisory lock；
//   2. FOR UPDATE锁定稳定page_id所属页面；
//   3. 比较预期页码、HTML哈希和页面生成元数据；
//   4. 创建当前旧页面完整历史版本；
//   5. 裁剪到最近20个版本；
//   6. 更新新HTML和生成元数据；
//   7. 一并提交。
//
// 锁顺序与CreatePageVersion保持一致：先advisory lock，再锁页面行，
// 避免普通版本创建与CAS写回互相持有不同锁而形成死锁。
//
// CAS冲突发生在版本插入之前；事务后半段失败时版本和页面更新整体回滚。

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
)

var (
	// ErrCoursewarePageCASConflict 表示页面已不再等于调用方读取的基线。
	ErrCoursewarePageCASConflict = errors.New(
		"课件页面已发生变化",
	)

	// ErrCoursewarePageCASInputInvalid 表示服务层传入的CAS参数不完整。
	ErrCoursewarePageCASInputInvalid = errors.New(
		"课件页面CAS写入参数无效",
	)
)

// CoursewarePageCASWriteInput 是页面原子写回输入。
type CoursewarePageCASWriteInput struct {
	PageID       string
	CoursewareID string
	PageNumber   int

	ExpectedHTMLHash            string
	ExpectedPlaceholderMap      string
	ExpectedMatchedComponentIDs string
	ExpectedPageStatus          string

	NewHTMLContent         string
	NewPlaceholderMap      string
	NewMatchedComponentIDs string
	NewPageStatus          string

	VersionSource string
	VersionNote   string
}

// CoursewarePageCASWriteResult 返回本次原子写回创建的历史版本。
//
// 旧页面HTML为空时不创建版本，此时VersionID为空、VersionNo为0。
type CoursewarePageCASWriteResult struct {
	VersionID string
	VersionNo int
}

func hashCoursewarePageHTML(htmlContent string) string {
	sum := sha256.Sum256([]byte(htmlContent))
	return fmt.Sprintf("%x", sum[:])
}

func validateCoursewarePageCASWriteInput(
	input *CoursewarePageCASWriteInput,
) error {
	if input == nil {
		return ErrCoursewarePageCASInputInvalid
	}

	if strings.TrimSpace(input.PageID) == "" ||
		strings.TrimSpace(input.CoursewareID) == "" ||
		input.PageNumber <= 0 {
		return ErrCoursewarePageCASInputInvalid
	}

	if len(strings.TrimSpace(input.ExpectedHTMLHash)) != 64 ||
		strings.TrimSpace(input.ExpectedPageStatus) == "" {
		return ErrCoursewarePageCASInputInvalid
	}

	if strings.TrimSpace(input.NewHTMLContent) == "" ||
		strings.TrimSpace(input.NewPageStatus) == "" ||
		strings.TrimSpace(input.VersionSource) == "" {
		return ErrCoursewarePageCASInputInvalid
	}

	return nil
}

// UpdateCWPageHTMLWithVersionCAS 原子保存页面旧版并写入AI结果。
func UpdateCWPageHTMLWithVersionCAS(
	ctx context.Context,
	input *CoursewarePageCASWriteInput,
) (*CoursewarePageCASWriteResult, error) {
	if err := validateCoursewarePageCASWriteInput(input); err != nil {
		return nil, err
	}

	pageID := strings.TrimSpace(input.PageID)
	coursewareID := strings.TrimSpace(input.CoursewareID)
	expectedHash := strings.ToLower(
		strings.TrimSpace(input.ExpectedHTMLHash),
	)

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"开始课件页面CAS写回事务失败: %w",
			err,
		)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// 与CreatePageVersion统一锁顺序：
	// 页面版本序列锁必须先于页面行锁获取。
	if _, err := tx.Exec(
		ctx,
		`SELECT pg_advisory_xact_lock(hashtext($1))`,
		pageID,
	); err != nil {
		return nil, fmt.Errorf(
			"锁定页面版本序列失败: %w",
			err,
		)
	}

	var (
		currentPageNumber     int
		currentHTML           string
		currentPlaceholderMap string
		currentMatchedIDs     string
		currentPageStatus     string
	)

	err = tx.QueryRow(
		ctx,
		`SELECT
			page_number,
			COALESCE(html_content, ''),
			COALESCE(placeholder_map::text, ''),
			COALESCE(matched_component_ids::text, ''),
			status
		 FROM courseware_pages
		 WHERE id = $1
		   AND courseware_id = $2
		 FOR UPDATE`,
		pageID,
		coursewareID,
	).Scan(
		&currentPageNumber,
		&currentHTML,
		&currentPlaceholderMap,
		&currentMatchedIDs,
		&currentPageStatus,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCoursewarePageCASConflict
		}

		return nil, fmt.Errorf(
			"锁定待写回课件页面失败: %w",
			err,
		)
	}

	if currentPageNumber != input.PageNumber ||
		hashCoursewarePageHTML(currentHTML) != expectedHash ||
		currentPlaceholderMap != input.ExpectedPlaceholderMap ||
		currentMatchedIDs != input.ExpectedMatchedComponentIDs ||
		currentPageStatus != input.ExpectedPageStatus {
		return nil, ErrCoursewarePageCASConflict
	}

	result := &CoursewarePageCASWriteResult{}

	// 旧页面非空时，历史版本与页面写回必须同事务提交。
	if strings.TrimSpace(currentHTML) != "" {
		if err := tx.QueryRow(
			ctx,
			`SELECT COALESCE(MAX(version_no), 0) + 1
			 FROM courseware_page_versions
			 WHERE page_id = $1
			   AND courseware_id = $2`,
			pageID,
			coursewareID,
		).Scan(&result.VersionNo); err != nil {
			return nil, fmt.Errorf(
				"计算页面版本号失败: %w",
				err,
			)
		}

		if err := tx.QueryRow(
			ctx,
			`INSERT INTO courseware_page_versions (
				id,
				page_id,
				courseware_id,
				version_no,
				html_content,
				placeholder_map,
				matched_component_ids,
				page_status,
				metadata_snapshot_complete,
				source,
				note
			)
			VALUES (
				gen_random_uuid(),
				$1,
				$2,
				$3,
				$4,
				$5::jsonb,
				$6::jsonb,
				$7,
				true,
				$8,
				$9
			)
			RETURNING id`,
			pageID,
			coursewareID,
			result.VersionNo,
			currentHTML,
			nullIfEmpty(currentPlaceholderMap),
			nullIfEmpty(currentMatchedIDs),
			currentPageStatus,
			input.VersionSource,
			nullIfEmpty(input.VersionNote),
		).Scan(&result.VersionID); err != nil {
			return nil, fmt.Errorf(
				"写入页面CAS完整版本快照失败: %w",
				err,
			)
		}

		if _, err := tx.Exec(
			ctx,
			`DELETE FROM courseware_page_versions
			 WHERE page_id = $1
			   AND courseware_id = $2
			   AND id NOT IN (
					SELECT id
					FROM courseware_page_versions
					WHERE page_id = $1
					  AND courseware_id = $2
					ORDER BY version_no DESC
					LIMIT $3
			   )`,
			pageID,
			coursewareID,
			cwPageVersionMaxKeep,
		); err != nil {
			return nil, fmt.Errorf(
				"裁剪页面CAS历史版本失败: %w",
				err,
			)
		}
	}

	updateResult, err := tx.Exec(
		ctx,
		`UPDATE courseware_pages
		 SET
			html_content = $1,
			placeholder_map = $2::jsonb,
			matched_component_ids = $3::jsonb,
			status = $4,
			assembly_version = 0,
			layout_status = 'unchecked',
			layout_audit_json = '{}'::jsonb,
			layout_html_hash = NULL,
			layout_checked_at = NULL,
			updated_at = NOW()
		 WHERE id = $5
		   AND courseware_id = $6`,
		input.NewHTMLContent,
		nullIfEmpty(input.NewPlaceholderMap),
		nullIfEmpty(input.NewMatchedComponentIDs),
		input.NewPageStatus,
		pageID,
		coursewareID,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"写入课件页面CAS结果失败: %w",
			err,
		)
	}
	if updateResult.RowsAffected() != 1 {
		return nil, ErrCoursewarePageCASConflict
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf(
			"提交课件页面CAS写回事务失败: %w",
			err,
		)
	}

	return result, nil
}
