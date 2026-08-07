package repository

// courseware_style_session_repo.go — AI美术风格工作室会话仓储
//
// 本文件负责：
//   - 创建新活动会话并自动归档旧活动会话；
//   - 查询当前活动会话及历史会话；
//   - 保存IAOCI草稿和预览状态；
//   - 确认风格时原子更新会话与coursewares正式锚点；
//   - 依靠现有数据库触发器在同一事务中同步@ANCHOR图片索引；
//   - 在仓储层再次校验图片资产属于同一课件；
//   - 在同一确认事务内校验图片必须来自当前会话。
//
// 确认图片来源规则：
//   - 当前会话状态为generated的三类预览图可以确认；
//   - style_character模式可确认当前会话reference_asset_id；
//   - style_only和inspiration不能直接确认原始参考图；
//   - 其它同课件图片一律不能借用；
//   - 请求模式与生成预览时保存的模式不一致时，旧预览必须重新生成。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

var (
	ErrCoursewareStyleSessionNotFound = errors.New("课件风格会话不存在")

	ErrCoursewareStyleSessionNotEditable = errors.New("课件风格会话不可继续编辑")

	ErrCoursewareStyleAssetInvalid = errors.New("风格图片不存在、不是图片或不属于当前课件")

	ErrCoursewareStyleConfirmAssetInvalid = errors.New(
		"确认图片必须是当前会话已生成的预览图；只有固定主体模式可以使用当前参考图",
	)

	ErrCoursewareStylePreviewModeStale = errors.New(
		"参考图模式已经改变，请按当前模式重新生成预览后再确认",
	)
)

const coursewareStyleSessionSelectColumns = `
id,
courseware_id,
user_id,
status,
reference_mode,
reference_asset_id,
confirmed_asset_id,
style_aoci_text,
style_summary,
version,
confirmed_at,
created_at,
updated_at`

func scanCoursewareStyleSession(
	scanner interface {
		Scan(dest ...interface{}) error
	},
) (*models.CoursewareStyleSession, error) {
	item := &models.CoursewareStyleSession{}

	err := scanner.Scan(
		&item.ID,
		&item.CoursewareID,
		&item.UserID,
		&item.Status,
		&item.ReferenceMode,
		&item.ReferenceAssetID,
		&item.ConfirmedAssetID,
		&item.StyleAOCIText,
		&item.StyleSummary,
		&item.Version,
		&item.ConfirmedAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return item, nil
}

// CreateCoursewareStyleSession 创建新的活动会话。
//
// 同一课件的创建过程由coursewares行锁串行化。
// 已有draft或previewing会话会先归档，再创建新会话。
func CreateCoursewareStyleSession(
	ctx context.Context,
	item *models.CoursewareStyleSession,
) error {
	if err := validateCoursewareStyleSessionCreate(
		item,
	); err != nil {
		return err
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf(
			"开启风格会话事务失败: %w",
			err,
		)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err := lockCoursewareStyleOwnerTx(
		ctx,
		tx,
		item.CoursewareID,
		item.UserID,
	); err != nil {
		return err
	}

	if err := validateCoursewareStyleAssetTx(
		ctx,
		tx,
		item.CoursewareID,
		item.ReferenceAssetID,
	); err != nil {
		return err
	}

	if _, err := tx.Exec(
		ctx,
		`UPDATE courseware_style_sessions
SET status = $1,
        updated_at = now()
WHERE courseware_id = $2
  AND status IN ($3, $4)`,
		models.CWStyleSessionStatusArchived,
		item.CoursewareID,
		models.CWStyleSessionStatusDraft,
		models.CWStyleSessionStatusPreviewing,
	); err != nil {
		return fmt.Errorf(
			"归档旧风格会话失败: %w",
			err,
		)
	}

	err = tx.QueryRow(
		ctx,
		`INSERT INTO courseware_style_sessions (
courseware_id,
user_id,
status,
reference_mode,
reference_asset_id,
confirmed_asset_id,
style_aoci_text,
style_summary,
version
)
VALUES ($1, $2, $3, $4, $5, NULL, '', '', 1)
RETURNING `+coursewareStyleSessionSelectColumns,
		item.CoursewareID,
		item.UserID,
		item.Status,
		item.ReferenceMode,
		nullableCoursewareStyleString(
			item.ReferenceAssetID,
		),
	).Scan(
		&item.ID,
		&item.CoursewareID,
		&item.UserID,
		&item.Status,
		&item.ReferenceMode,
		&item.ReferenceAssetID,
		&item.ConfirmedAssetID,
		&item.StyleAOCIText,
		&item.StyleSummary,
		&item.Version,
		&item.ConfirmedAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf(
			"创建风格会话失败: %w",
			err,
		)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf(
			"提交风格会话事务失败: %w",
			err,
		)
	}

	return nil
}

// GetCoursewareStyleSessionByID 按课件、用户和会话ID读取。
func GetCoursewareStyleSessionByID(
	ctx context.Context,
	coursewareID string,
	sessionID string,
	userID string,
) (*models.CoursewareStyleSession, error) {
	item, err := scanCoursewareStyleSession(
		database.DB.QueryRow(
			ctx,
			`SELECT `+
				coursewareStyleSessionSelectColumns+
				` FROM courseware_style_sessions
WHERE id = $1
  AND courseware_id = $2
  AND user_id = $3`,
			sessionID,
			coursewareID,
			userID,
		),
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCoursewareStyleSessionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf(
			"读取风格会话失败: %w",
			err,
		)
	}

	return item, nil
}

// GetActiveCoursewareStyleSession 读取当前活动会话。
func GetActiveCoursewareStyleSession(
	ctx context.Context,
	coursewareID string,
	userID string,
) (*models.CoursewareStyleSession, error) {
	item, err := scanCoursewareStyleSession(
		database.DB.QueryRow(
			ctx,
			`SELECT `+
				coursewareStyleSessionSelectColumns+
				` FROM courseware_style_sessions
WHERE courseware_id = $1
  AND user_id = $2
  AND status IN ($3, $4)
ORDER BY updated_at DESC
LIMIT 1`,
			coursewareID,
			userID,
			models.CWStyleSessionStatusDraft,
			models.CWStyleSessionStatusPreviewing,
		),
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCoursewareStyleSessionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf(
			"读取当前风格会话失败: %w",
			err,
		)
	}

	return item, nil
}

// ListCoursewareStyleSessions 返回当前课件的会话历史。
func ListCoursewareStyleSessions(
	ctx context.Context,
	coursewareID string,
	userID string,
) ([]*models.CoursewareStyleSession, error) {
	rows, err := database.DB.Query(
		ctx,
		`SELECT `+
			coursewareStyleSessionSelectColumns+
			` FROM courseware_style_sessions
WHERE courseware_id = $1
  AND user_id = $2
ORDER BY created_at DESC`,
		coursewareID,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"查询风格会话历史失败: %w",
			err,
		)
	}
	defer rows.Close()

	items := make(
		[]*models.CoursewareStyleSession,
		0,
	)

	for rows.Next() {
		item, scanErr :=
			scanCoursewareStyleSession(rows)
		if scanErr != nil {
			return nil, fmt.Errorf(
				"扫描风格会话失败: %w",
				scanErr,
			)
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"遍历风格会话失败: %w",
			err,
		)
	}

	return items, nil
}

// SaveCoursewareStyleSessionDraft 保存当前IAOCI草稿。
func SaveCoursewareStyleSessionDraft(
	ctx context.Context,
	coursewareID string,
	sessionID string,
	userID string,
	status string,
	referenceMode string,
	referenceAssetID *string,
	styleAOCIText string,
	styleSummary string,
) (*models.CoursewareStyleSession, error) {
	if status != models.CWStyleSessionStatusDraft &&
		status != models.CWStyleSessionStatusPreviewing {
		return nil, fmt.Errorf(
			"风格草稿状态不合法: %s",
			status,
		)
	}

	if !models.IsValidCWStyleReferenceMode(
		referenceMode,
	) {
		return nil, fmt.Errorf(
			"参考图模式不合法: %s",
			referenceMode,
		)
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"开启风格草稿事务失败: %w",
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
		referenceAssetID,
	); err != nil {
		return nil, err
	}

	item, err := scanCoursewareStyleSession(
		tx.QueryRow(
			ctx,
			`UPDATE courseware_style_sessions
SET status = $1,
        reference_mode = $2,
        reference_asset_id = COALESCE($3, reference_asset_id),
        style_aoci_text = $4,
        style_summary = $5,
        version = version + 1,
        updated_at = now()
WHERE id = $6
  AND courseware_id = $7
  AND user_id = $8
RETURNING `+coursewareStyleSessionSelectColumns,
			status,
			referenceMode,
			nullableCoursewareStyleString(
				referenceAssetID,
			),
			strings.TrimSpace(styleAOCIText),
			strings.TrimSpace(styleSummary),
			sessionID,
			coursewareID,
			userID,
		),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"保存风格草稿失败: %w",
			err,
		)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf(
			"提交风格草稿事务失败: %w",
			err,
		)
	}

	return item, nil
}

// ArchiveCoursewareStyleSession 归档未确认会话。
func ArchiveCoursewareStyleSession(
	ctx context.Context,
	coursewareID string,
	sessionID string,
	userID string,
) error {
	tag, err := database.DB.Exec(
		ctx,
		`UPDATE courseware_style_sessions
SET status = $1,
        updated_at = now()
WHERE id = $2
  AND courseware_id = $3
  AND user_id = $4
  AND status IN ($5, $6)`,
		models.CWStyleSessionStatusArchived,
		sessionID,
		coursewareID,
		userID,
		models.CWStyleSessionStatusDraft,
		models.CWStyleSessionStatusPreviewing,
	)
	if err != nil {
		return fmt.Errorf(
			"归档风格会话失败: %w",
			err,
		)
	}
	if tag.RowsAffected() != 1 {
		return ErrCoursewareStyleSessionNotEditable
	}

	return nil
}

// ConfirmCoursewareStyleSession 保留旧内部调用签名。
//
// 旧调用不显式传模式时，事务内读取会话当前模式。
func ConfirmCoursewareStyleSession(
	ctx context.Context,
	coursewareID string,
	sessionID string,
	userID string,
	confirmedAssetID string,
	styleAOCIText string,
	styleSummary string,
) (*models.CoursewareStyleSession, error) {
	return ConfirmCoursewareStyleSessionWithMode(
		ctx,
		coursewareID,
		sessionID,
		userID,
		confirmedAssetID,
		"",
		styleAOCIText,
		styleSummary,
	)
}

// ConfirmCoursewareStyleSessionWithMode 原子确认风格。
//
// 事务内执行：
//  1. 锁定课件和会话；
//  2. 读取会话当前模式和参考图；
//  3. 校验确认图片属于当前课件；
//  4. 校验确认图片来源属于当前会话；
//  5. 校验预览生成模式与确认模式一致；
//  6. 更新coursewares.style_anchor_asset_id/style_anchor_vaoci；
//  7. 由现有触发器同步@ANCHOR；
//  8. 将会话模式和状态原子更新为confirmed。
func ConfirmCoursewareStyleSessionWithMode(
	ctx context.Context,
	coursewareID string,
	sessionID string,
	userID string,
	confirmedAssetID string,
	referenceMode string,
	styleAOCIText string,
	styleSummary string,
) (*models.CoursewareStyleSession, error) {
	confirmedAssetID =
		strings.TrimSpace(confirmedAssetID)

	referenceMode =
		strings.TrimSpace(referenceMode)

	styleAOCIText =
		strings.TrimSpace(styleAOCIText)

	if confirmedAssetID == "" {
		return nil, fmt.Errorf(
			"确认风格必须选择一张图片",
		)
	}
	if styleAOCIText == "" {
		return nil, fmt.Errorf(
			"确认风格必须存在完整IAOCI",
		)
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"开启风格确认事务失败: %w",
			err,
		)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err := lockCoursewareStyleOwnerTx(
		ctx,
		tx,
		coursewareID,
		userID,
	); err != nil {
		return nil, err
	}

	if err := lockEditableCoursewareStyleSessionTx(
		ctx,
		tx,
		coursewareID,
		sessionID,
		userID,
	); err != nil {
		return nil, err
	}

	var storedReferenceMode string
	var storedReferenceAssetID *string

	if err := tx.QueryRow(
		ctx,
		`SELECT reference_mode, reference_asset_id
FROM courseware_style_sessions
WHERE id = $1
  AND courseware_id = $2
  AND user_id = $3`,
		sessionID,
		coursewareID,
		userID,
	).Scan(
		&storedReferenceMode,
		&storedReferenceAssetID,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil,
				ErrCoursewareStyleSessionNotFound
		}

		return nil, fmt.Errorf(
			"读取确认会话模式失败: %w",
			err,
		)
	}

	if referenceMode == "" {
		referenceMode =
			storedReferenceMode
	}

	if !models.IsValidCWStyleReferenceMode(
		referenceMode,
	) {
		return nil, fmt.Errorf(
			"参考图模式不合法: %s",
			referenceMode,
		)
	}

	assetIDPtr := &confirmedAssetID

	if err := validateCoursewareStyleAssetTx(
		ctx,
		tx,
		coursewareID,
		assetIDPtr,
	); err != nil {
		return nil, err
	}

	if err :=
		validateCoursewareStyleConfirmAssetTx(
			ctx,
			tx,
			coursewareID,
			sessionID,
			confirmedAssetID,
			referenceMode,
			storedReferenceMode,
			storedReferenceAssetID,
		); err != nil {
		return nil, err
	}

	tag, err := tx.Exec(
		ctx,
		`UPDATE coursewares
SET style_anchor_asset_id = $1,
        style_anchor_vaoci = $2,
        updated_at = now()
WHERE id = $3
  AND user_id = $4
  AND deleted_at IS NULL`,
		confirmedAssetID,
		styleAOCIText,
		coursewareID,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"写入课程正式风格锚点失败: %w",
			err,
		)
	}
	if tag.RowsAffected() != 1 {
		return nil,
			ErrCoursewareStyleSessionNotFound
	}

	item, err := scanCoursewareStyleSession(
		tx.QueryRow(
			ctx,
			`UPDATE courseware_style_sessions
SET status = $1,
        confirmed_asset_id = $2,
        reference_mode = $3,
        style_aoci_text = $4,
        style_summary = $5,
        version = version + 1,
        confirmed_at = now(),
        updated_at = now()
WHERE id = $6
  AND courseware_id = $7
  AND user_id = $8
RETURNING `+coursewareStyleSessionSelectColumns,
			models.CWStyleSessionStatusConfirmed,
			confirmedAssetID,
			referenceMode,
			styleAOCIText,
			strings.TrimSpace(styleSummary),
			sessionID,
			coursewareID,
			userID,
		),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"确认风格会话失败: %w",
			err,
		)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf(
			"提交风格确认事务失败: %w",
			err,
		)
	}

	return item, nil
}

// validateCoursewareStyleConfirmAssetTx 校验确认图属于当前会话。
//
// 当前会话预览图必须：
//   - session_id和courseware_id匹配；
//   - status为generated；
//   - asset_id等于确认图片；
//   - 生成预览时保存的reference_mode与本次确认模式一致。
//
// 当前参考图只有在style_character模式下才允许直接确认。
func validateCoursewareStyleConfirmAssetTx(
	ctx context.Context,
	tx pgx.Tx,
	coursewareID string,
	sessionID string,
	confirmedAssetID string,
	referenceMode string,
	storedReferenceMode string,
	storedReferenceAssetID *string,
) error {
	var generatedPreview bool

	err := tx.QueryRow(
		ctx,
		`SELECT EXISTS (
        SELECT 1
        FROM courseware_style_previews
        WHERE session_id = $1
          AND courseware_id = $2
          AND asset_id = $3
          AND status = $4
)`,
		sessionID,
		coursewareID,
		confirmedAssetID,
		models.CWStylePreviewStatusGenerated,
	).Scan(&generatedPreview)
	if err != nil {
		return fmt.Errorf(
			"校验确认预览来源失败: %w",
			err,
		)
	}

	return validateCoursewareStyleConfirmSelection(
		referenceMode,
		storedReferenceMode,
		storedReferenceAssetID,
		confirmedAssetID,
		generatedPreview,
	)
}

func validateCoursewareStyleSessionCreate(
	item *models.CoursewareStyleSession,
) error {
	if item == nil {
		return fmt.Errorf("风格会话对象为空")
	}

	item.CoursewareID =
		strings.TrimSpace(item.CoursewareID)
	item.UserID =
		strings.TrimSpace(item.UserID)

	if item.CoursewareID == "" {
		return fmt.Errorf(
			"courseware_id不能为空",
		)
	}
	if item.UserID == "" {
		return fmt.Errorf(
			"user_id不能为空",
		)
	}

	if item.Status == "" {
		item.Status =
			models.CWStyleSessionStatusDraft
	}
	if item.Status !=
		models.CWStyleSessionStatusDraft {
		return fmt.Errorf(
			"新风格会话必须从draft开始",
		)
	}

	if item.ReferenceMode == "" {
		item.ReferenceMode =
			models.CWStyleReferenceModeStyleOnly
	}
	if !models.IsValidCWStyleReferenceMode(
		item.ReferenceMode,
	) {
		return fmt.Errorf(
			"参考图模式不合法: %s",
			item.ReferenceMode,
		)
	}

	item.Version = 1

	return nil
}

func lockCoursewareStyleOwnerTx(
	ctx context.Context,
	tx pgx.Tx,
	coursewareID string,
	userID string,
) error {
	var lockedID string

	err := tx.QueryRow(
		ctx,
		`SELECT id
FROM coursewares
WHERE id = $1
  AND user_id = $2
  AND deleted_at IS NULL
FOR UPDATE`,
		coursewareID,
		userID,
	).Scan(&lockedID)

	if errors.Is(err, pgx.ErrNoRows) {
		return ErrCoursewareStyleSessionNotFound
	}
	if err != nil {
		return fmt.Errorf(
			"锁定风格课件失败: %w",
			err,
		)
	}

	return nil
}

func lockEditableCoursewareStyleSessionTx(
	ctx context.Context,
	tx pgx.Tx,
	coursewareID string,
	sessionID string,
	userID string,
) error {
	var status string

	err := tx.QueryRow(
		ctx,
		`SELECT status
FROM courseware_style_sessions
WHERE id = $1
  AND courseware_id = $2
  AND user_id = $3
FOR UPDATE`,
		sessionID,
		coursewareID,
		userID,
	).Scan(&status)

	if errors.Is(err, pgx.ErrNoRows) {
		return ErrCoursewareStyleSessionNotFound
	}
	if err != nil {
		return fmt.Errorf(
			"锁定风格会话失败: %w",
			err,
		)
	}

	if !models.IsEditableCWStyleSessionStatus(
		status,
	) {
		return ErrCoursewareStyleSessionNotEditable
	}

	return nil
}

func validateCoursewareStyleAssetTx(
	ctx context.Context,
	tx pgx.Tx,
	coursewareID string,
	assetID *string,
) error {
	if assetID == nil ||
		strings.TrimSpace(*assetID) == "" {
		return nil
	}

	var valid bool

	err := tx.QueryRow(
		ctx,
		`SELECT EXISTS (
        SELECT 1
        FROM courseware_assets
        WHERE id = $1
          AND courseware_id = $2
          AND asset_type = 'image'
)`,
		strings.TrimSpace(*assetID),
		coursewareID,
	).Scan(&valid)
	if err != nil {
		return fmt.Errorf(
			"校验风格图片归属失败: %w",
			err,
		)
	}
	if !valid {
		return ErrCoursewareStyleAssetInvalid
	}

	return nil
}

func nullableCoursewareStyleString(
	value *string,
) interface{} {
	if value == nil ||
		strings.TrimSpace(*value) == "" {
		return nil
	}

	return strings.TrimSpace(*value)
}
