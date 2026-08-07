package repository

// courseware_review_instruction_version_repo.go
//
// 课件审核整改指令不可变版本仓储。
//
// 核心约束：
//   1. 所有读取都绑定整改项参与者，越权与不存在统一按未找到处理；
//   2. 保存并确认在单事务中锁定整改项、校验预期当前版本和页面快照；
//   3. 浏览器不能提交创建者、确认者、页面哈希、版本号或来源；
//   4. 旧版本先变为superseded，再创建连续的新confirmed版本；
//   5. 页面变化或删除时先提交stale或orphaned状态，再返回稳定业务错误；
//   6. 正式交付后、页面应用后或非授权参与者均不能创建新版本；
//   7. 数据库触发器承担不可变、唯一当前版本和引用一致性的最终防线。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"tedna/internal/database"
	"tedna/internal/models"
)

var (
	ErrCoursewareReviewInstructionVersionNotFound       = errors.New("课件整改指令版本不存在")
	ErrCoursewareReviewInstructionVersionConflict       = errors.New("课件整改指令版本已变化，请刷新后重试")
	ErrCoursewareReviewInstructionVersionNotConfirmable = errors.New("当前整改项不能保存新的确认指令版本")
	ErrCoursewareReviewInstructionVersionPageStale      = errors.New("整改项对应页面已经变化，旧指令版本不能继续确认")
	ErrCoursewareReviewInstructionVersionPageOrphaned   = errors.New("整改项对应页面已经删除，旧指令版本不能继续确认")
)

const cwReviewInstructionVersionSelectColumns = `
	version.id,
	version.item_id,
	version.version_no,
	version.content,
	version.content_hash,
	version.source_type,
	version.created_by,
	version.created_at,
	COALESCE(version.confirmed_by::text, ''),
	version.confirmed_at,
	version.page_snapshot_hash,
	version.status`

// ConfirmCoursewareReviewInstructionVersionInput 是仓储可信确认输入。
type ConfirmCoursewareReviewInstructionVersionInput struct {
	ItemID                   string
	ActorID                  string
	Instruction              string
	ExpectedCurrentVersionID string
	SourceType               string
}

func scanCoursewareReviewInstructionVersion(row interface {
	Scan(dest ...interface{}) error
}) (*models.CoursewareReviewInstructionVersion, error) {
	version := &models.CoursewareReviewInstructionVersion{}
	var confirmedBy string

	err := row.Scan(
		&version.ID,
		&version.ItemID,
		&version.VersionNo,
		&version.Content,
		&version.ContentHash,
		&version.SourceType,
		&version.CreatedBy,
		&version.CreatedAt,
		&confirmedBy,
		&version.ConfirmedAt,
		&version.PageSnapshotHash,
		&version.Status,
	)
	if err != nil {
		return nil, err
	}
	if confirmedBy != "" {
		version.ConfirmedBy = &confirmedBy
	}

	return version, nil
}

// ListCoursewareReviewInstructionVersions 按版本号倒序返回历史及当前引用ID。
//
// 正式项交付作者后，作者只能读取delivered_instruction_version_id指向的版本，
// 不暴露审核员交付前的其他候选或历史版本。
func ListCoursewareReviewInstructionVersions(
	ctx context.Context,
	itemID string,
	participantID string,
) ([]*models.CoursewareReviewInstructionVersion, string, error) {
	itemID = strings.TrimSpace(itemID)
	participantID = strings.TrimSpace(participantID)

	var (
		currentVersionID   string
		itemSourceType     string
		createdBy          string
		ownerID            string
		deliveredVersionID string
	)

	err := database.DB.QueryRow(
		ctx,
		`SELECT
			COALESCE(current_instruction_version_id::text, ''),
			source_type,
			created_by,
			owner_id,
			COALESCE(delivered_instruction_version_id::text, '')
		 FROM courseware_review_items
		 WHERE id = $1
		   AND (created_by = $2 OR owner_id = $2)`,
		itemID,
		participantID,
	).Scan(
		&currentVersionID,
		&itemSourceType,
		&createdBy,
		&ownerID,
		&deliveredVersionID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, "", ErrCoursewareReviewInstructionVersionNotFound
		}
		return nil, "", fmt.Errorf("查询整改项当前指令版本失败: %w", err)
	}

	authorRestricted := itemSourceType == models.CWReviewItemSourceFormal &&
		participantID == ownerID &&
		participantID != createdBy

	if authorRestricted {
		if deliveredVersionID == "" {
			return nil, "", ErrCoursewareReviewInstructionVersionNotFound
		}
		currentVersionID = deliveredVersionID
	}

	rows, err := database.DB.Query(
		ctx,
		`SELECT `+cwReviewInstructionVersionSelectColumns+`
		 FROM courseware_review_instruction_versions AS version
		 JOIN courseware_review_items AS item
		   ON item.id = version.item_id
		 WHERE version.item_id = $1
		   AND (item.created_by = $2 OR item.owner_id = $2)
		   AND (
				NOT $3
				OR version.id = NULLIF($4, '')::uuid
		   )
		 ORDER BY version.version_no DESC`,
		itemID,
		participantID,
		authorRestricted,
		deliveredVersionID,
	)
	if err != nil {
		return nil, "", fmt.Errorf("查询课件整改指令版本列表失败: %w", err)
	}
	defer rows.Close()

	versions := make([]*models.CoursewareReviewInstructionVersion, 0)

	for rows.Next() {
		version, scanErr := scanCoursewareReviewInstructionVersion(rows)
		if scanErr != nil {
			return nil, "", fmt.Errorf("扫描课件整改指令版本失败: %w", scanErr)
		}
		versions = append(versions, version)
	}

	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("遍历课件整改指令版本失败: %w", err)
	}

	return versions, currentVersionID, nil
}

// GetCoursewareReviewInstructionVersion 读取属于指定整改项的一条版本。
func GetCoursewareReviewInstructionVersion(
	ctx context.Context,
	itemID string,
	versionID string,
	participantID string,
) (*models.CoursewareReviewInstructionVersion, error) {
	version, err := scanCoursewareReviewInstructionVersion(
		database.DB.QueryRow(
			ctx,
			`SELECT `+cwReviewInstructionVersionSelectColumns+`
			 FROM courseware_review_instruction_versions AS version
			 JOIN courseware_review_items AS item
			   ON item.id = version.item_id
			 WHERE version.id = $1
			   AND version.item_id = $2
			   AND (item.created_by = $3 OR item.owner_id = $3)
			   AND (
					item.source_type <> 'formal'
					OR item.created_by = $3
					OR version.id =
						item.delivered_instruction_version_id
			   )`,
			strings.TrimSpace(versionID),
			strings.TrimSpace(itemID),
			strings.TrimSpace(participantID),
		),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCoursewareReviewInstructionVersionNotFound
		}
		return nil, fmt.Errorf("查询课件整改指令版本失败: %w", err)
	}

	return version, nil
}

// GetCurrentCoursewareReviewInstructionVersion 读取整改项当前版本。
// 尚未确认过任何版本时返回nil、nil。
func GetCurrentCoursewareReviewInstructionVersion(
	ctx context.Context,
	itemID string,
	participantID string,
) (*models.CoursewareReviewInstructionVersion, error) {
	itemID = strings.TrimSpace(itemID)
	participantID = strings.TrimSpace(participantID)

	var currentVersionID string

	err := database.DB.QueryRow(
		ctx,
		`SELECT CASE
			WHEN source_type = 'formal'
			 AND owner_id = $2
			 AND created_by <> $2
				THEN COALESCE(
					delivered_instruction_version_id::text,
					''
				)
			ELSE COALESCE(
				current_instruction_version_id::text,
				''
			)
		 END
		 FROM courseware_review_items
		 WHERE id = $1
		   AND (created_by = $2 OR owner_id = $2)`,
		itemID,
		participantID,
	).Scan(&currentVersionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCoursewareReviewInstructionVersionNotFound
		}
		return nil, fmt.Errorf("查询整改项当前指令版本引用失败: %w", err)
	}

	if currentVersionID == "" {
		return nil, nil
	}

	return GetCoursewareReviewInstructionVersion(
		ctx,
		itemID,
		currentVersionID,
		participantID,
	)
}

// ConfirmCoursewareReviewInstructionVersion 原子保存并确认一个新版本。
func ConfirmCoursewareReviewInstructionVersion(
	ctx context.Context,
	input *ConfirmCoursewareReviewInstructionVersionInput,
) (*models.CoursewareReviewInstructionVersion, error) {
	if input == nil {
		return nil, errors.New("课件整改指令版本确认输入不能为空")
	}

	itemID := strings.TrimSpace(input.ItemID)
	actorID := strings.TrimSpace(input.ActorID)
	instruction := strings.TrimSpace(input.Instruction)
	expectedCurrentVersionID := strings.TrimSpace(
		input.ExpectedCurrentVersionID,
	)
	sourceType := strings.TrimSpace(input.SourceType)

	if itemID == "" || actorID == "" {
		return nil, ErrCoursewareReviewInstructionVersionNotFound
	}
	if instruction == "" {
		return nil, errors.New("确认的课件修改指令不能为空")
	}
	if !models.IsCWReviewInstructionVersionSourceType(sourceType) {
		return nil, errors.New("课件整改指令版本来源无效")
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"开始课件整改指令版本确认事务失败: %w",
			err,
		)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var (
		coursewareID     string
		itemSourceType   string
		createdBy        string
		ownerID          string
		itemStatus       string
		pageID           string
		pageHTMLHash     string
		alreadyDelivered bool
		alreadyApplied   bool
		currentVersionID string
	)

	err = tx.QueryRow(
		ctx,
		`SELECT
			courseware_id,
			source_type,
			created_by,
			owner_id,
			status,
			COALESCE(page_id::text, ''),
			page_html_hash,
			(
				courseware_review_id IS NOT NULL
				OR feedback_id IS NOT NULL
			),
			(
				applied_instruction_version_id IS NOT NULL
				OR applied_at IS NOT NULL
			),
			COALESCE(current_instruction_version_id::text, '')
		 FROM courseware_review_items
		 WHERE id = $1
		   AND (created_by = $2 OR owner_id = $2)
		 FOR UPDATE`,
		itemID,
		actorID,
	).Scan(
		&coursewareID,
		&itemSourceType,
		&createdBy,
		&ownerID,
		&itemStatus,
		&pageID,
		&pageHTMLHash,
		&alreadyDelivered,
		&alreadyApplied,
		&currentVersionID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCoursewareReviewInstructionVersionNotFound
		}
		return nil, fmt.Errorf("锁定课件整改项失败: %w", err)
	}

	if currentVersionID != expectedCurrentVersionID {
		return nil, ErrCoursewareReviewInstructionVersionConflict
	}
	if alreadyDelivered || alreadyApplied {
		return nil, ErrCoursewareReviewInstructionVersionNotConfirmable
	}
	if !canConfirmCoursewareReviewInstructionVersion(
		itemSourceType,
		actorID,
		createdBy,
		ownerID,
	) {
		return nil, ErrCoursewareReviewInstructionVersionNotConfirmable
	}
	if !isCoursewareReviewInstructionItemStatusConfirmable(itemStatus) {
		return nil, ErrCoursewareReviewInstructionVersionNotConfirmable
	}

	pageSnapshotHash, err :=
		loadCoursewareReviewInstructionPageSnapshotTx(
			ctx,
			tx,
			itemID,
			coursewareID,
			pageID,
			pageHTMLHash,
		)
	if err != nil {
		return nil, err
	}

	if currentVersionID != "" {
		result, updateErr := tx.Exec(
			ctx,
			`UPDATE courseware_review_instruction_versions
			 SET status = 'superseded'
			 WHERE id = $1
			   AND item_id = $2
			   AND status = 'confirmed'`,
			currentVersionID,
			itemID,
		)
		if updateErr != nil {
			return nil,
				mapCoursewareReviewInstructionVersionWriteError(
					"替代旧课件整改指令版本失败",
					updateErr,
				)
		}
		if result.RowsAffected() != 1 {
			return nil,
				ErrCoursewareReviewInstructionVersionConflict
		}
	}

	var nextVersionNo int

	err = tx.QueryRow(
		ctx,
		`SELECT COALESCE(MAX(version_no), 0) + 1
		 FROM courseware_review_instruction_versions
		 WHERE item_id = $1`,
		itemID,
	).Scan(&nextVersionNo)
	if err != nil {
		return nil, fmt.Errorf(
			"计算课件整改指令下一版本号失败: %w",
			err,
		)
	}

	version, err := scanCoursewareReviewInstructionVersion(
		tx.QueryRow(
			ctx,
			`INSERT INTO
				courseware_review_instruction_versions AS version (
					item_id,
					version_no,
					content,
					content_hash,
					source_type,
					created_by,
					created_at,
					confirmed_by,
					confirmed_at,
					page_snapshot_hash,
					status
				)
			 VALUES (
					$1,
					$2,
					$3,
					encode(
						digest(
							convert_to($3, 'UTF8'),
							'sha256'
						),
						'hex'
					),
					$4,
					$5,
					NOW(),
					$5,
					NOW(),
					$6,
					'confirmed'
				)
			 RETURNING `+cwReviewInstructionVersionSelectColumns,
			itemID,
			nextVersionNo,
			instruction,
			sourceType,
			actorID,
			pageSnapshotHash,
		),
	)
	if err != nil {
		return nil,
			mapCoursewareReviewInstructionVersionWriteError(
				"创建课件整改指令版本失败",
				err,
			)
	}
	if version.ConfirmedAt == nil {
		return nil, errors.New("新指令版本缺少确认时间")
	}

	result, err := tx.Exec(
		ctx,
		`UPDATE courseware_review_items
		 SET
			status = 'confirmed',
			confirmed_instruction = $3,
			confirmed_at = $4,
			current_instruction_version_id = $5,
			updated_at = NOW()
		 WHERE id = $1
		   AND (created_by = $2 OR owner_id = $2)
		   AND current_instruction_version_id
				IS NOT DISTINCT FROM NULLIF($6, '')::uuid
		   AND status IN ('detected', 'discussing', 'confirmed')
		   AND courseware_review_id IS NULL
		   AND feedback_id IS NULL
		   AND applied_instruction_version_id IS NULL
		   AND applied_at IS NULL`,
		itemID,
		actorID,
		instruction,
		*version.ConfirmedAt,
		version.ID,
		expectedCurrentVersionID,
	)
	if err != nil {
		return nil,
			mapCoursewareReviewInstructionVersionWriteError(
				"绑定课件整改项当前指令版本失败",
				err,
			)
	}
	if result.RowsAffected() != 1 {
		return nil, ErrCoursewareReviewInstructionVersionConflict
	}

	if err := tx.Commit(ctx); err != nil {
		return nil,
			mapCoursewareReviewInstructionVersionWriteError(
				"提交课件整改指令版本确认事务失败",
				err,
			)
	}

	return version, nil
}

func canConfirmCoursewareReviewInstructionVersion(
	itemSourceType string,
	actorID string,
	createdBy string,
	ownerID string,
) bool {
	switch itemSourceType {
	case models.CWReviewItemSourceFormal:
		return actorID == createdBy
	case models.CWReviewItemSourceSelf:
		return actorID == ownerID
	default:
		return false
	}
}

func isCoursewareReviewInstructionItemStatusConfirmable(
	status string,
) bool {
	switch status {
	case models.CWReviewItemStatusDetected,
		models.CWReviewItemStatusDiscussing,
		models.CWReviewItemStatusConfirmed:
		return true
	default:
		return false
	}
}

func loadCoursewareReviewInstructionPageSnapshotTx(
	ctx context.Context,
	tx pgx.Tx,
	itemID string,
	coursewareID string,
	pageID string,
	pageHTMLHash string,
) (string, error) {
	if pageID == "" {
		return "", nil
	}

	var pageSnapshotHash string

	err := tx.QueryRow(
		ctx,
		`SELECT encode(
			digest(
				convert_to(
					COALESCE(html_content, ''),
					'UTF8'
				),
				'sha256'
			),
			'hex'
		 )
		 FROM courseware_pages
		 WHERE id = $1
		   AND courseware_id = $2
		 FOR SHARE`,
		pageID,
		coursewareID,
	).Scan(&pageSnapshotHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			if markErr :=
				markCoursewareReviewInstructionItemInvalidTx(
					ctx,
					tx,
					itemID,
					models.CWReviewItemStatusOrphaned,
				); markErr != nil {
				return "", markErr
			}

			if commitErr := tx.Commit(ctx); commitErr != nil {
				return "", fmt.Errorf(
					"提交页面删除状态失败: %w",
					commitErr,
				)
			}

			return "",
				ErrCoursewareReviewInstructionVersionPageOrphaned
		}

		return "", fmt.Errorf(
			"读取确认时页面快照失败: %w",
			err,
		)
	}

	if strings.TrimSpace(pageHTMLHash) == "" ||
		pageSnapshotHash == strings.TrimSpace(pageHTMLHash) {
		return pageSnapshotHash, nil
	}

	if err :=
		markCoursewareReviewInstructionItemInvalidTx(
			ctx,
			tx,
			itemID,
			models.CWReviewItemStatusStale,
		); err != nil {
		return "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf(
			"提交页面变化状态失败: %w",
			err,
		)
	}

	return "",
		ErrCoursewareReviewInstructionVersionPageStale
}

func markCoursewareReviewInstructionItemInvalidTx(
	ctx context.Context,
	tx pgx.Tx,
	itemID string,
	nextStatus string,
) error {
	result, err := tx.Exec(
		ctx,
		`UPDATE courseware_review_items
		 SET
			status = $2,
			updated_at = NOW()
		 WHERE id = $1
		   AND status IN ('detected', 'discussing', 'confirmed')`,
		itemID,
		nextStatus,
	)
	if err != nil {
		return mapCoursewareReviewInstructionVersionWriteError(
			"标记课件整改项页面失效状态失败",
			err,
		)
	}
	if result.RowsAffected() != 1 {
		return ErrCoursewareReviewInstructionVersionConflict
	}

	return nil
}

func mapCoursewareReviewInstructionVersionWriteError(
	message string,
	err error,
) error {
	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23503",
			"23505",
			"23514",
			"40001",
			"P0001":
			return ErrCoursewareReviewInstructionVersionConflict
		}
	}

	return fmt.Errorf("%s: %w", message, err)
}
