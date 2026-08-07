package repository

// courseware_style_preview_repo.go — AI美术风格工作室预览仓储
//
// 本文件负责：
//   - 为人物、知识对象、教学图解三种固定类型创建或覆盖预览；
//   - 保存生成状态、错误和图片资产；
//   - 新风格版本形成后把旧预览标记为stale；
//   - 会话确认或归档后禁止继续更新预览；
//   - 再次校验预览图片属于同一课件。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

var ErrCoursewareStylePreviewNotFound = errors.New("课件风格预览不存在")

const coursewareStylePreviewSelectColumns = `
id,
session_id,
courseware_id,
preview_type,
asset_id,
generation_prompt,
status,
last_error,
version,
created_at,
updated_at`

func scanCoursewareStylePreview(
	scanner interface {
		Scan(dest ...interface{}) error
	},
) (*models.CoursewareStylePreview, error) {
	item := &models.CoursewareStylePreview{}

	err := scanner.Scan(
		&item.ID,
		&item.SessionID,
		&item.CoursewareID,
		&item.PreviewType,
		&item.AssetID,
		&item.GenerationPrompt,
		&item.Status,
		&item.LastError,
		&item.Version,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return item, nil
}

// UpsertCoursewareStylePreview 创建或覆盖指定类型预览。
func UpsertCoursewareStylePreview(
	ctx context.Context,
	userID string,
	item *models.CoursewareStylePreview,
) error {
	if err := validateCoursewareStylePreview(
		item,
	); err != nil {
		return err
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf(
			"开启风格预览事务失败: %w",
			err,
		)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err := lockEditableCoursewareStyleSessionTx(
		ctx,
		tx,
		item.CoursewareID,
		item.SessionID,
		userID,
	); err != nil {
		return err
	}

	if err := validateCoursewareStyleAssetTx(
		ctx,
		tx,
		item.CoursewareID,
		item.AssetID,
	); err != nil {
		return err
	}

	err = tx.QueryRow(
		ctx,
		`INSERT INTO courseware_style_previews (
session_id,
courseware_id,
preview_type,
asset_id,
generation_prompt,
status,
last_error,
version
)
VALUES ($1, $2, $3, $4, $5, $6, $7, 1)
ON CONFLICT (session_id, preview_type)
DO UPDATE SET
courseware_id = EXCLUDED.courseware_id,
asset_id = EXCLUDED.asset_id,
generation_prompt = EXCLUDED.generation_prompt,
status = EXCLUDED.status,
last_error = EXCLUDED.last_error,
version = courseware_style_previews.version + 1,
updated_at = now()
RETURNING id, version, created_at, updated_at`,
		item.SessionID,
		item.CoursewareID,
		item.PreviewType,
		nullableCoursewareStyleString(
			item.AssetID,
		),
		strings.TrimSpace(
			item.GenerationPrompt,
		),
		item.Status,
		strings.TrimSpace(item.LastError),
	).Scan(
		&item.ID,
		&item.Version,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf(
			"保存风格预览失败: %w",
			err,
		)
	}

	if _, err := tx.Exec(
		ctx,
		`UPDATE courseware_style_sessions
SET status = $1,
	updated_at = now()
WHERE id = $2
  AND courseware_id = $3
  AND user_id = $4`,
		models.CWStyleSessionStatusPreviewing,
		item.SessionID,
		item.CoursewareID,
		userID,
	); err != nil {
		return fmt.Errorf(
			"更新风格会话预览状态失败: %w",
			err,
		)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf(
			"提交风格预览事务失败: %w",
			err,
		)
	}

	return nil
}

// GetCoursewareStylePreviewByType 读取指定类型预览。
func GetCoursewareStylePreviewByType(
	ctx context.Context,
	coursewareID string,
	sessionID string,
	userID string,
	previewType string,
) (*models.CoursewareStylePreview, error) {
	if !models.IsValidCWStylePreviewType(
		previewType,
	) {
		return nil, fmt.Errorf(
			"风格预览类型不合法: %s",
			previewType,
		)
	}

	if _, err := GetCoursewareStyleSessionByID(
		ctx,
		coursewareID,
		sessionID,
		userID,
	); err != nil {
		return nil, err
	}

	item, err := scanCoursewareStylePreview(
		database.DB.QueryRow(
			ctx,
			`SELECT `+
				coursewareStylePreviewSelectColumns+
				` FROM courseware_style_previews
WHERE session_id = $1
  AND courseware_id = $2
  AND preview_type = $3`,
			sessionID,
			coursewareID,
			previewType,
		),
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCoursewareStylePreviewNotFound
	}
	if err != nil {
		return nil, fmt.Errorf(
			"读取风格预览失败: %w",
			err,
		)
	}

	return item, nil
}

// ListCoursewareStylePreviews 按固定类型顺序返回预览。
func ListCoursewareStylePreviews(
	ctx context.Context,
	coursewareID string,
	sessionID string,
	userID string,
) ([]*models.CoursewareStylePreview, error) {
	if _, err := GetCoursewareStyleSessionByID(
		ctx,
		coursewareID,
		sessionID,
		userID,
	); err != nil {
		return nil, err
	}

	rows, err := database.DB.Query(
		ctx,
		`SELECT `+
			coursewareStylePreviewSelectColumns+
			` FROM courseware_style_previews
WHERE session_id = $1
  AND courseware_id = $2
ORDER BY CASE preview_type
	WHEN 'character' THEN 1
	WHEN 'object' THEN 2
	WHEN 'diagram' THEN 3
	ELSE 99
END`,
		sessionID,
		coursewareID,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"查询风格预览失败: %w",
			err,
		)
	}
	defer rows.Close()

	items := make(
		[]*models.CoursewareStylePreview,
		0,
	)

	for rows.Next() {
		item, scanErr :=
			scanCoursewareStylePreview(rows)
		if scanErr != nil {
			return nil, fmt.Errorf(
				"扫描风格预览失败: %w",
				scanErr,
			)
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"遍历风格预览失败: %w",
			err,
		)
	}

	return items, nil
}

// UpdateCoursewareStylePreviewStatus 更新预览生成状态。
func UpdateCoursewareStylePreviewStatus(
	ctx context.Context,
	userID string,
	coursewareID string,
	sessionID string,
	previewType string,
	status string,
	assetID *string,
	lastError string,
) (*models.CoursewareStylePreview, error) {
	if !models.IsValidCWStylePreviewType(
		previewType,
	) {
		return nil, fmt.Errorf(
			"风格预览类型不合法: %s",
			previewType,
		)
	}
	if !models.IsValidCWStylePreviewStatus(
		status,
	) {
		return nil, fmt.Errorf(
			"风格预览状态不合法: %s",
			status,
		)
	}
	if status == models.CWStylePreviewStatusGenerated &&
		(assetID == nil ||
			strings.TrimSpace(*assetID) == "") {
		return nil, fmt.Errorf(
			"generated预览必须绑定图片资产",
		)
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"开启预览状态事务失败: %w",
			err,
		)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err := lockEditableCoursewareStyleSessionTx(
		ctx,
		tx,
		coursewareID,
		sessionID,
		userID,
	); err != nil {
		return nil, err
	}

	if err := validateCoursewareStyleAssetTx(
		ctx,
		tx,
		coursewareID,
		assetID,
	); err != nil {
		return nil, err
	}

	item, err := scanCoursewareStylePreview(
		tx.QueryRow(
			ctx,
			`UPDATE courseware_style_previews
SET status = $1,
	asset_id = $2,
	last_error = $3,
	version = version + 1,
	updated_at = now()
WHERE session_id = $4
  AND courseware_id = $5
  AND preview_type = $6
RETURNING `+coursewareStylePreviewSelectColumns,
			status,
			nullableCoursewareStyleString(
				assetID,
			),
			strings.TrimSpace(lastError),
			sessionID,
			coursewareID,
			previewType,
		),
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCoursewareStylePreviewNotFound
	}
	if err != nil {
		return nil, fmt.Errorf(
			"更新风格预览状态失败: %w",
			err,
		)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf(
			"提交预览状态事务失败: %w",
			err,
		)
	}

	return item, nil
}

// MarkCoursewareStylePreviewsStale 把当前会话预览全部标记为过期。
func MarkCoursewareStylePreviewsStale(
	ctx context.Context,
	userID string,
	coursewareID string,
	sessionID string,
) error {
	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf(
			"开启预览过期事务失败: %w",
			err,
		)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err := lockEditableCoursewareStyleSessionTx(
		ctx,
		tx,
		coursewareID,
		sessionID,
		userID,
	); err != nil {
		return err
	}

	if _, err := tx.Exec(
		ctx,
		`UPDATE courseware_style_previews
SET status = $1,
	last_error = '',
	version = version + 1,
	updated_at = now()
WHERE session_id = $2
  AND courseware_id = $3
  AND status <> $1`,
		models.CWStylePreviewStatusStale,
		sessionID,
		coursewareID,
	); err != nil {
		return fmt.Errorf(
			"标记风格预览过期失败: %w",
			err,
		)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf(
			"提交预览过期事务失败: %w",
			err,
		)
	}

	return nil
}

func validateCoursewareStylePreview(
	item *models.CoursewareStylePreview,
) error {
	if item == nil {
		return fmt.Errorf("风格预览对象为空")
	}

	item.SessionID =
		strings.TrimSpace(item.SessionID)
	item.CoursewareID =
		strings.TrimSpace(item.CoursewareID)
	item.PreviewType =
		strings.TrimSpace(item.PreviewType)
	item.GenerationPrompt =
		strings.TrimSpace(item.GenerationPrompt)
	item.LastError =
		strings.TrimSpace(item.LastError)

	if item.SessionID == "" ||
		item.CoursewareID == "" {
		return fmt.Errorf(
			"风格预览会话ID或课件ID为空",
		)
	}

	if !models.IsValidCWStylePreviewType(
		item.PreviewType,
	) {
		return fmt.Errorf(
			"风格预览类型不合法: %s",
			item.PreviewType,
		)
	}

	if item.Status == "" {
		item.Status =
			models.CWStylePreviewStatusPending
	}

	if !models.IsValidCWStylePreviewStatus(
		item.Status,
	) {
		return fmt.Errorf(
			"风格预览状态不合法: %s",
			item.Status,
		)
	}

	if item.Status ==
		models.CWStylePreviewStatusGenerated &&
		(item.AssetID == nil ||
			strings.TrimSpace(*item.AssetID) == "") {
		return fmt.Errorf(
			"generated预览必须绑定图片资产",
		)
	}

	item.Version = 1

	return nil
}
