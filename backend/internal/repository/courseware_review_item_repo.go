package repository

// courseware_review_item_repo.go
//
// 课件AI审核整改项的基础创建、读取、列表和正式反馈绑定仓储。
//
// 核心约束：
//   1. 一条跨多页finding由Service拆成多个页级整改项；
//   2. page_id是稳定定位依据，页码只作为审核时快照；
//   3. 创建正式整改项时先处于未绑定状态，人工提交审核决定后，
//      在同一事务中绑定courseware_review_id和feedback_id；
//   4. 正式反馈绑定必须同时冻结仍有效的current_instruction_version_id；
//   5. confirmed_instruction仅作为当前版本正文的兼容快照；
//   6. 作者重新提交和人工确认解决的字段由专用状态仓储维护；
//   7. 仓储不决定教育域和页面修改权限，Service仍需重新授权。
//
// 状态迁移、忽略、恢复、重新提交和解决确认位于状态仓储。
// 独立讨论消息位于courseware_review_item_message_repo.go。

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
	// ErrCoursewareReviewItemNotFound 合并不存在和不满足记录访问边界两种情况。
	ErrCoursewareReviewItemNotFound = errors.New("课件审核整改项不存在")

	// ErrCoursewareReviewItemConflict 表示状态、绑定关系或参与者权限已经变化。
	ErrCoursewareReviewItemConflict = errors.New("课件审核整改项状态已变化，请刷新后重试")
)

const cwReviewItemSelectColumns = `
	id,
	courseware_id,
	source_session_id,
	source_finding_id,
	origin_type,
	COALESCE(source_global_message_id::text, ''),
	COALESCE(courseware_review_id::text, ''),
	COALESCE(feedback_id::text, ''),
	source_type,
	review_level,
	review_round,
	created_by,
	owner_id,
	COALESCE(page_id::text, ''),
	page_number_snapshot,
	page_title_snapshot,
	page_html_hash,
	page_updated_at_snapshot,
	severity,
	dimension,
	title,
	description,
	COALESCE(evidence_json::text, '{}'),
	original_suggestion,
	confirmed_instruction,
	COALESCE(current_instruction_version_id::text, ''),
	COALESCE(delivered_instruction_version_id::text, ''),
	COALESCE(applied_instruction_version_id::text, ''),
	status,
	applied_page_hash,
	resubmitted_at,
	resubmitted_review_level,
	resubmitted_review_round,
	COALESCE(resolved_by::text, ''),
	COALESCE(resolved_review_id::text, ''),
	resolved_review_level,
	resolved_review_round,
	resolution_note,
	created_at,
	updated_at,
	confirmed_at,
	applied_at,
	resolved_at`

func scanCoursewareReviewItem(row interface {
	Scan(dest ...interface{}) error
}) (*models.CoursewareReviewItem, error) {
	item := &models.CoursewareReviewItem{}

	var (
		reviewID                      string
		feedbackID                    string
		pageID                        string
		sourceGlobalMessageID         string
		currentInstructionVersionID   string
		deliveredInstructionVersionID string
		appliedInstructionVersionID   string
		resolvedBy                    string
		resolvedReviewID              string
	)

	err := row.Scan(
		&item.ID,
		&item.CoursewareID,
		&item.SourceSessionID,
		&item.SourceFindingID,
		&item.OriginType,
		&sourceGlobalMessageID,
		&reviewID,
		&feedbackID,
		&item.SourceType,
		&item.ReviewLevel,
		&item.ReviewRound,
		&item.CreatedBy,
		&item.OwnerID,
		&pageID,
		&item.PageNumberSnapshot,
		&item.PageTitleSnapshot,
		&item.PageHTMLHash,
		&item.PageUpdatedAtSnapshot,
		&item.Severity,
		&item.Dimension,
		&item.Title,
		&item.Description,
		&item.EvidenceJSON,
		&item.OriginalSuggestion,
		&item.ConfirmedInstruction,
		&currentInstructionVersionID,
		&deliveredInstructionVersionID,
		&appliedInstructionVersionID,
		&item.Status,
		&item.AppliedPageHash,
		&item.ResubmittedAt,
		&item.ResubmittedReviewLevel,
		&item.ResubmittedReviewRound,
		&resolvedBy,
		&resolvedReviewID,
		&item.ResolvedReviewLevel,
		&item.ResolvedReviewRound,
		&item.ResolutionNote,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.ConfirmedAt,
		&item.AppliedAt,
		&item.ResolvedAt,
	)
	if err != nil {
		return nil, err
	}

	if sourceGlobalMessageID != "" {
		item.SourceGlobalMessageID = &sourceGlobalMessageID
	}
	if reviewID != "" {
		item.CoursewareReviewID = &reviewID
	}
	if feedbackID != "" {
		item.FeedbackID = &feedbackID
	}
	if pageID != "" {
		item.PageID = &pageID
	}
	if currentInstructionVersionID != "" {
		item.CurrentInstructionVersionID =
			&currentInstructionVersionID
	}
	if deliveredInstructionVersionID != "" {
		item.DeliveredInstructionVersionID =
			&deliveredInstructionVersionID
	}
	if appliedInstructionVersionID != "" {
		item.AppliedInstructionVersionID =
			&appliedInstructionVersionID
	}
	if resolvedBy != "" {
		item.ResolvedBy = &resolvedBy
	}
	if resolvedReviewID != "" {
		item.ResolvedReviewID = &resolvedReviewID
	}

	return item, nil
}

// CreateCoursewareReviewItem 创建一条自审或正式审核整改项。
//
// 本入口自建事务，适合AI报告完成后将finding物化为整改项。
// 正式审核提交决定时的绑定操作使用独立Tx函数。
func CreateCoursewareReviewItem(
	ctx context.Context,
	item *models.CoursewareReviewItem,
) error {
	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("开始创建课件整改项事务失败: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err := CreateCoursewareReviewItemTx(ctx, tx, item); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("提交课件整改项事务失败: %w", err)
	}

	return nil
}

// CreateCoursewareReviewItemTx 在调用方事务中创建整改项。
//
// 新建整改项不得直接声明当前、交付或应用版本。
// 三个版本引用分别由明确确认、正式审核提交和页面应用事务形成。
func CreateCoursewareReviewItemTx(
	ctx context.Context,
	tx pgx.Tx,
	item *models.CoursewareReviewItem,
) error {
	if tx == nil {
		return errors.New("课件整改项事务不能为空")
	}
	if item == nil {
		return errors.New("课件整改项不能为空")
	}

	if strings.TrimSpace(item.OriginType) == "" {
		item.OriginType = models.CWReviewItemOriginAIFinding
	}
	if strings.TrimSpace(item.Status) == "" {
		item.Status = models.CWReviewItemStatusDetected
	}
	if strings.TrimSpace(item.Severity) == "" {
		item.Severity = models.CWReviewSeverityMedium
	}

	err := tx.QueryRow(
		ctx,
		`
		INSERT INTO courseware_review_items (
			courseware_id,
			source_session_id,
			source_finding_id,
			origin_type,
			source_global_message_id,
			courseware_review_id,
			feedback_id,
			source_type,
			review_level,
			review_round,
			created_by,
			owner_id,
			page_id,
			page_number_snapshot,
			page_title_snapshot,
			page_html_hash,
			page_updated_at_snapshot,
			severity,
			dimension,
			title,
			description,
			evidence_json,
			original_suggestion,
			confirmed_instruction,
			status,
			applied_page_hash,
			resubmitted_at,
			resubmitted_review_level,
			resubmitted_review_round,
			resolved_by,
			resolved_review_id,
			resolved_review_level,
			resolved_review_round,
			resolution_note,
			created_at,
			updated_at
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
			$21, $22::jsonb, $23, $24, $25, $26, $27, $28, $29, $30,
			$31, $32, $33, $34, NOW(), NOW()
		)
		RETURNING id, created_at, updated_at`,
		strings.TrimSpace(item.CoursewareID),
		strings.TrimSpace(item.SourceSessionID),
		strings.TrimSpace(item.SourceFindingID),
		strings.TrimSpace(item.OriginType),
		item.SourceGlobalMessageID,
		item.CoursewareReviewID,
		item.FeedbackID,
		strings.TrimSpace(item.SourceType),
		item.ReviewLevel,
		item.ReviewRound,
		strings.TrimSpace(item.CreatedBy),
		strings.TrimSpace(item.OwnerID),
		item.PageID,
		item.PageNumberSnapshot,
		strings.TrimSpace(item.PageTitleSnapshot),
		strings.TrimSpace(item.PageHTMLHash),
		item.PageUpdatedAtSnapshot,
		strings.TrimSpace(item.Severity),
		strings.TrimSpace(item.Dimension),
		strings.TrimSpace(item.Title),
		strings.TrimSpace(item.Description),
		cwAIReviewJSONOrDefault(item.EvidenceJSON, "{}"),
		strings.TrimSpace(item.OriginalSuggestion),
		strings.TrimSpace(item.ConfirmedInstruction),
		strings.TrimSpace(item.Status),
		strings.TrimSpace(item.AppliedPageHash),
		item.ResubmittedAt,
		item.ResubmittedReviewLevel,
		item.ResubmittedReviewRound,
		item.ResolvedBy,
		item.ResolvedReviewID,
		item.ResolvedReviewLevel,
		item.ResolvedReviewRound,
		strings.TrimSpace(item.ResolutionNote),
	).Scan(
		&item.ID,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("创建课件审核整改项失败: %w", err)
	}

	return nil
}

// GetCoursewareReviewItemByID 按ID读取整改项。
func GetCoursewareReviewItemByID(
	ctx context.Context,
	itemID string,
) (*models.CoursewareReviewItem, error) {
	item, err := scanCoursewareReviewItem(
		database.DB.QueryRow(
			ctx,
			`SELECT `+cwReviewItemSelectColumns+`
			 FROM courseware_review_items
			 WHERE id = $1`,
			strings.TrimSpace(itemID),
		),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCoursewareReviewItemNotFound
		}
		return nil, fmt.Errorf("查询课件审核整改项失败: %w", err)
	}

	return item, nil
}

// GetCoursewareReviewItemForParticipant 只允许创建者或课件作者读取。
//
// Service仍需先重新读取课件并执行教育域和资源访问校验，
// 本函数只提供最小的记录级参与者边界。
func GetCoursewareReviewItemForParticipant(
	ctx context.Context,
	itemID string,
	participantID string,
) (*models.CoursewareReviewItem, error) {
	item, err := scanCoursewareReviewItem(
		database.DB.QueryRow(
			ctx,
			`SELECT `+cwReviewItemSelectColumns+`
			 FROM courseware_review_items
			 WHERE id = $1
			   AND (created_by = $2 OR owner_id = $2)`,
			strings.TrimSpace(itemID),
			strings.TrimSpace(participantID),
		),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCoursewareReviewItemNotFound
		}
		return nil, fmt.Errorf("查询参与者课件整改项失败: %w", err)
	}

	return item, nil
}

// ListCoursewareReviewItemsForOwner 查询作者需要处理的全部整改项。
func ListCoursewareReviewItemsForOwner(
	ctx context.Context,
	coursewareID string,
	ownerID string,
) ([]*models.CoursewareReviewItem, error) {
	return listCoursewareReviewItems(
		ctx,
		`WHERE courseware_id = $1 AND owner_id = $2`,
		strings.TrimSpace(coursewareID),
		strings.TrimSpace(ownerID),
	)
}

// ListCoursewareReviewItemsBySessionForCreator 查询创建者在一次AI会话中物化的全部整改项。
func ListCoursewareReviewItemsBySessionForCreator(
	ctx context.Context,
	sessionID string,
	creatorID string,
) ([]*models.CoursewareReviewItem, error) {
	return listCoursewareReviewItems(
		ctx,
		`WHERE source_session_id = $1 AND created_by = $2`,
		strings.TrimSpace(sessionID),
		strings.TrimSpace(creatorID),
	)
}

// ListCoursewareReviewItemsByFeedback 查询一次正式审核反馈交付的整改项。
func ListCoursewareReviewItemsByFeedback(
	ctx context.Context,
	feedbackID string,
) ([]*models.CoursewareReviewItem, error) {
	return listCoursewareReviewItems(
		ctx,
		`WHERE feedback_id = $1`,
		strings.TrimSpace(feedbackID),
	)
}

func listCoursewareReviewItems(
	ctx context.Context,
	whereClause string,
	args ...interface{},
) ([]*models.CoursewareReviewItem, error) {
	rows, err := database.DB.Query(
		ctx,
		`SELECT `+cwReviewItemSelectColumns+`
		 FROM courseware_review_items
		 `+whereClause+`
		 ORDER BY
			CASE severity
				WHEN 'critical' THEN 1
				WHEN 'high' THEN 2
				WHEN 'medium' THEN 3
				WHEN 'low' THEN 4
				ELSE 5
			END,
			page_number_snapshot ASC,
			created_at ASC`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("查询课件审核整改项列表失败: %w", err)
	}
	defer rows.Close()

	items := make(
		[]*models.CoursewareReviewItem,
		0,
	)

	for rows.Next() {
		item, scanErr :=
			scanCoursewareReviewItem(
				rows,
			)
		if scanErr != nil {
			return nil, fmt.Errorf(
				"扫描课件审核整改项失败: %w",
				scanErr,
			)
		}

		items = append(
			items,
			item,
		)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"遍历课件审核整改项失败: %w",
			err,
		)
	}

	return items, nil
}

// AttachCoursewareReviewItemsToFeedbackTx 将审核员选中的正式整改项
// 和其当前确认版本绑定到人工审核记录。
//
// 事务层要求：
//  1. 记录属于当前课件、会话、创建者和正式审核来源；
//  2. 状态仍为confirmed，兼容指令快照非空；
//  3. current_instruction_version_id存在且属于本整改项；
//  4. 当前版本状态仍为confirmed，版本正文与兼容快照一致；
//  5. 整改项尚未绑定其他审核、反馈或交付版本。
//
// UPDATE在同一语句中将delivered_instruction_version_id冻结为当前版本。
// 返回数量必须与请求问题数量完全相等，否则上层事务整体回滚。
func AttachCoursewareReviewItemsToFeedbackTx(
	ctx context.Context,
	tx pgx.Tx,
	itemIDs []string,
	coursewareID string,
	sessionID string,
	creatorID string,
	reviewID string,
	feedbackID string,
	reviewLevel int,
	reviewRound int,
) (int64, error) {
	if tx == nil {
		return 0, errors.New("课件整改项绑定事务不能为空")
	}
	if len(itemIDs) == 0 {
		return 0, nil
	}

	result, err := tx.Exec(
		ctx,
		`
		UPDATE courseware_review_items
		SET
			courseware_review_id = $5,
			feedback_id = $6,
			delivered_instruction_version_id =
				current_instruction_version_id,
			review_level = $7,
			review_round = $8,
			updated_at = NOW()
		WHERE id::text = ANY($1::text[])
		  AND courseware_id = $2
		  AND source_session_id = $3
		  AND created_by = $4
		  AND source_type = 'formal'
		  AND status = 'confirmed'
		  AND BTRIM(confirmed_instruction) <> ''
		  AND current_instruction_version_id IS NOT NULL
		  AND delivered_instruction_version_id IS NULL
		  AND courseware_review_id IS NULL
		  AND feedback_id IS NULL
		  AND EXISTS (
			SELECT 1
			FROM courseware_review_instruction_versions AS version
			WHERE version.id =
				courseware_review_items.current_instruction_version_id
			  AND version.item_id =
				courseware_review_items.id
			  AND version.status = 'confirmed'
			  AND BTRIM(version.content) =
				BTRIM(
					courseware_review_items.confirmed_instruction
				)
		  )`,
		itemIDs,
		strings.TrimSpace(coursewareID),
		strings.TrimSpace(sessionID),
		strings.TrimSpace(creatorID),
		strings.TrimSpace(reviewID),
		strings.TrimSpace(feedbackID),
		reviewLevel,
		reviewRound,
	)
	if err != nil {
		return 0, fmt.Errorf(
			"绑定正式课件整改项及确认版本失败: %w",
			err,
		)
	}

	return result.RowsAffected(), nil
}
