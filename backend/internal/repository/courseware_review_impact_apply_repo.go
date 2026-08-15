package repository

// courseware_review_impact_apply_repo.go
//
// R-07结构化影响方案Atomic Apply事务总控。
//
// 绝对边界：
//   1. 浏览器只提供plan_id、version、selected operation IDs；
//   2. 事务开始后重新FOR UPDATE锁plan；
//   3. 重新读取可信assistant消息并复核source_message_hash；
//   4. selected ID必须来自数据库冻结的operations_json；
//   5. cancel_relation先锁关系行，保持既有取消路径的锁序；
//   6. 所有选中operation完成precondition复核后才允许第一笔业务写入；
//   7. 任一失败整个事务回滚，plan继续保持draft/version=1；
//   8. 所有动作成功后才迁移plan到applied/version=2并追加applied事件。

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

var (
	ErrCoursewareReviewImpactSelectionInvalid = errors.New(
		"课件审核影响方案选中操作无效",
	)

	ErrCoursewareReviewImpactOperationUnsupported = errors.New(
		"课件审核影响方案包含尚未支持的操作",
	)
)

type cwReviewImpactItemPrecondition struct {
	ItemID      string `json:"item_id"`
	Status      string `json:"status"`
	Fingerprint string `json:"fingerprint"`
}

type cwReviewImpactGroupPrecondition struct {
	GroupID string `json:"group_id"`
	Status  string `json:"status"`
	Version int    `json:"version"`
}

type cwReviewImpactMemberPrecondition struct {
	MemberID string `json:"member_id"`
	GroupID  string `json:"group_id"`
	ItemID   string `json:"item_id"`
	Status   string `json:"status"`
	Version  int    `json:"version"`
}

type cwReviewImpactPagePrecondition struct {
	Scope      string `json:"scope"`
	PageID     string `json:"page_id,omitempty"`
	PageNumber int    `json:"page_number,omitempty"`
	Title      string `json:"title,omitempty"`
	HTMLHash   string `json:"html_hash,omitempty"`
}

// ApplyCoursewareReviewImpactPlan 是R-07教师最终确认的唯一Repository事务入口。
func ApplyCoursewareReviewImpactPlan(
	ctx context.Context,
	planID string,
	sessionID string,
	expectedVersion int,
	actorID string,
	selectedOperationIDs []string,
) (*models.CoursewareReviewImpactPlan, error) {
	planID = strings.TrimSpace(planID)
	sessionID = strings.TrimSpace(sessionID)
	actorID = strings.TrimSpace(actorID)

	if planID == "" ||
		sessionID == "" ||
		actorID == "" ||
		expectedVersion != 1 {
		return nil, ErrCoursewareReviewImpactSelectionInvalid
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"开始应用课件审核影响方案事务失败: %w",
			err,
		)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	plan, err := lockCoursewareReviewImpactPlanTx(
		ctx,
		tx,
		planID,
		sessionID,
		actorID,
	)
	if err != nil {
		return nil, err
	}

	if err := ensureCoursewareReviewImpactPlanDraftVersion(
		plan,
		expectedVersion,
	); err != nil {
		return nil, err
	}

	if err := verifyCoursewareReviewImpactSourceMessageTx(
		ctx,
		tx,
		plan,
	); err != nil {
		return nil, err
	}

	operations, err := parseCoursewareReviewImpactOperations(
		plan.OperationsJSON,
	)
	if err != nil {
		return nil, err
	}

	selectedOperations, normalizedSelectedIDs, err :=
		selectCoursewareReviewImpactOperations(
			operations,
			selectedOperationIDs,
		)
	if err != nil {
		return nil, err
	}

	groupOperations := make(
		[]models.CoursewareReviewImpactOperation,
		0,
		len(selectedOperations),
	)
	itemRelationOperations := make(
		[]models.CoursewareReviewImpactOperation,
		0,
		len(selectedOperations),
	)

	for _, operation := range selectedOperations {
		switch {
		case isCoursewareReviewImpactGroupOperation(
			operation.OperationType,
		):
			groupOperations = append(
				groupOperations,
				operation,
			)

		case isCoursewareReviewImpactItemRelationOperation(
			operation.OperationType,
		):
			itemRelationOperations = append(
				itemRelationOperations,
				operation,
			)

		default:
			return nil,
				ErrCoursewareReviewImpactOperationUnsupported
		}
	}

	// cancel_relation的既有正式仓储锁序是relation -> endpoint items。
	// 在任何group/item锁之前先锁住这些关系行，避免与既有取消入口形成反向锁环。
	prelockedCancelRelations, err :=
		prelockCoursewareReviewImpactCancelRelationsTx(
			ctx,
			tx,
			plan,
			itemRelationOperations,
			actorID,
		)
	if err != nil {
		return nil, err
	}

	preparedGroupOperations, err :=
		prevalidateCoursewareReviewImpactGroupOperationsTx(
			ctx,
			tx,
			plan,
			groupOperations,
			actorID,
		)
	if err != nil {
		return nil, err
	}

	preparedItemRelationOperations, err :=
		prevalidateCoursewareReviewImpactItemRelationOperationsTx(
			ctx,
			tx,
			plan,
			itemRelationOperations,
			actorID,
			prelockedCancelRelations,
		)
	if err != nil {
		return nil, err
	}

	// 到这里为止没有业务数据写入。
	// 所有selected operation均已重新读取并验证当前状态。

	if err := applyCoursewareReviewImpactGroupOperationsTx(
		ctx,
		tx,
		plan,
		preparedGroupOperations,
		actorID,
	); err != nil {
		return nil, err
	}

	if err := applyCoursewareReviewImpactItemRelationOperationsTx(
		ctx,
		tx,
		plan,
		preparedItemRelationOperations,
		actorID,
	); err != nil {
		return nil, err
	}

	selectedJSONBytes, err := json.Marshal(
		normalizedSelectedIDs,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"序列化影响方案最终选择失败: %w",
			err,
		)
	}

	appliedPlan, err := markCoursewareReviewImpactPlanAppliedTx(
		ctx,
		tx,
		plan,
		expectedVersion,
		string(selectedJSONBytes),
		actorID,
	)
	if err != nil {
		return nil, err
	}

	metadataJSONBytes, err := json.Marshal(
		map[string]interface{}{
			"selected_operation_count": len(normalizedSelectedIDs),
		},
	)
	if err != nil {
		return nil, fmt.Errorf(
			"序列化影响方案应用事件元数据失败: %w",
			err,
		)
	}

	if err := insertCoursewareReviewImpactPlanEventTx(
		ctx,
		tx,
		appliedPlan,
		models.CWReviewImpactPlanEventApplied,
		actorID,
		string(selectedJSONBytes),
		string(metadataJSONBytes),
	); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf(
			"提交课件审核影响方案原子应用事务失败: %w",
			err,
		)
	}

	return appliedPlan, nil
}

func verifyCoursewareReviewImpactSourceMessageTx(
	ctx context.Context,
	tx pgx.Tx,
	plan *models.CoursewareReviewImpactPlan,
) error {
	if plan == nil {
		return ErrCoursewareReviewImpactPlanNotFound
	}

	var trustedHash string

	err := tx.QueryRow(
		ctx,
		`SELECT
			public.build_cw_review_impact_message_hash(
				message.content,
				message.citations_json
			)
		 FROM courseware_ai_review_messages AS message
		 WHERE message.id = $1
		   AND message.session_id = $2
		   AND message.review_item_id IS NULL
		   AND message.role = 'assistant'
		 FOR SHARE`,
		plan.SourceMessageID,
		plan.SourceSessionID,
	).Scan(&trustedHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrCoursewareReviewImpactPlanConflict
		}

		return fmt.Errorf(
			"重新读取影响方案可信来源消息失败: %w",
			err,
		)
	}

	if strings.TrimSpace(trustedHash) == "" ||
		trustedHash != plan.SourceMessageHash {
		return ErrCoursewareReviewImpactPlanConflict
	}

	return nil
}

func parseCoursewareReviewImpactOperations(
	raw string,
) ([]models.CoursewareReviewImpactOperation, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, ErrCoursewareReviewImpactSelectionInvalid
	}

	var operations []models.CoursewareReviewImpactOperation
	if err := json.Unmarshal(
		[]byte(raw),
		&operations,
	); err != nil {
		return nil, fmt.Errorf(
			"解析数据库冻结影响方案操作失败: %w",
			err,
		)
	}

	if len(operations) == 0 || len(operations) > 100 {
		return nil, ErrCoursewareReviewImpactSelectionInvalid
	}

	seen := make(
		map[string]struct{},
		len(operations),
	)

	for index := range operations {
		operation := &operations[index]

		operation.OperationID = strings.TrimSpace(
			operation.OperationID,
		)
		operation.OperationType = strings.TrimSpace(
			operation.OperationType,
		)
		operation.Summary = strings.TrimSpace(
			operation.Summary,
		)

		if operation.OperationID == "" ||
			!models.IsCWReviewImpactOperationType(
				operation.OperationType,
			) ||
			operation.Summary == "" ||
			operation.Payload == nil ||
			operation.Preconditions == nil {
			return nil,
				ErrCoursewareReviewImpactSelectionInvalid
		}

		if _, exists := seen[operation.OperationID]; exists {
			return nil,
				ErrCoursewareReviewImpactSelectionInvalid
		}

		seen[operation.OperationID] = struct{}{}
	}

	return operations, nil
}

func selectCoursewareReviewImpactOperations(
	operations []models.CoursewareReviewImpactOperation,
	selectedOperationIDs []string,
) (
	[]models.CoursewareReviewImpactOperation,
	[]string,
	error,
) {
	if len(selectedOperationIDs) == 0 ||
		len(selectedOperationIDs) > len(operations) {
		return nil, nil,
			ErrCoursewareReviewImpactSelectionInvalid
	}

	selectedSet := make(
		map[string]struct{},
		len(selectedOperationIDs),
	)

	for _, rawID := range selectedOperationIDs {
		operationID := strings.TrimSpace(rawID)
		if operationID == "" {
			return nil, nil,
				ErrCoursewareReviewImpactSelectionInvalid
		}

		if _, exists := selectedSet[operationID]; exists {
			return nil, nil,
				ErrCoursewareReviewImpactSelectionInvalid
		}

		selectedSet[operationID] = struct{}{}
	}

	selected := make(
		[]models.CoursewareReviewImpactOperation,
		0,
		len(selectedSet),
	)
	normalizedIDs := make(
		[]string,
		0,
		len(selectedSet),
	)

	for _, operation := range operations {
		if _, exists :=
			selectedSet[operation.OperationID]; !exists {
			continue
		}

		selected = append(selected, operation)
		normalizedIDs = append(
			normalizedIDs,
			operation.OperationID,
		)
		delete(selectedSet, operation.OperationID)
	}

	if len(selectedSet) != 0 ||
		len(selected) != len(selectedOperationIDs) {
		return nil, nil,
			ErrCoursewareReviewImpactSelectionInvalid
	}

	return selected, normalizedIDs, nil
}

func decodeCoursewareReviewImpactMap(
	value map[string]interface{},
	dest interface{},
) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return ErrCoursewareReviewImpactSelectionInvalid
	}

	decoder := json.NewDecoder(
		bytes.NewReader(raw),
	)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dest); err != nil {
		return ErrCoursewareReviewImpactSelectionInvalid
	}

	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		return ErrCoursewareReviewImpactSelectionInvalid
	}

	return nil
}

// lockCoursewareReviewImpactItemTx 锁定整改项，同时在同一事务中检查其原页面仍新鲜。
func lockCoursewareReviewImpactItemTx(
	ctx context.Context,
	tx pgx.Tx,
	itemID string,
) (*models.CoursewareReviewItem, error) {
	item, err := scanCoursewareReviewItem(
		tx.QueryRow(
			ctx,
			`SELECT `+cwReviewItemSelectColumns+`
			 FROM courseware_review_items
			 WHERE id = $1
			 FOR UPDATE`,
			strings.TrimSpace(itemID),
		),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCoursewareReviewItemConflict
		}

		return nil, fmt.Errorf(
			"锁定影响方案整改项失败: %w",
			err,
		)
	}

	if err := validateCoursewareReviewImpactItemPageFreshTx(
		ctx,
		tx,
		item,
	); err != nil {
		return nil, err
	}

	return item, nil
}

func validateCoursewareReviewImpactItemPrecondition(
	plan *models.CoursewareReviewImpactPlan,
	item *models.CoursewareReviewItem,
	precondition cwReviewImpactItemPrecondition,
	actorID string,
) error {
	if plan == nil || item == nil {
		return ErrCoursewareReviewImpactPlanConflict
	}

	if item.ID != strings.TrimSpace(
		precondition.ItemID,
	) ||
		item.CoursewareID != plan.CoursewareID ||
		item.SourceSessionID != plan.SourceSessionID ||
		item.Status != strings.TrimSpace(
			precondition.Status,
		) ||
		item.CoursewareReviewID != nil ||
		item.FeedbackID != nil ||
		item.DeliveredInstructionVersionID != nil ||
		item.AppliedInstructionVersionID != nil ||
		item.AppliedAt != nil {
		return ErrCoursewareReviewImpactPlanConflict
	}

	switch item.SourceType {
	case models.CWReviewItemSourceFormal:
		if item.CreatedBy != strings.TrimSpace(actorID) {
			return ErrCoursewareReviewImpactPlanConflict
		}

	case models.CWReviewItemSourceSelf:
		if item.OwnerID != strings.TrimSpace(actorID) {
			return ErrCoursewareReviewImpactPlanConflict
		}

	default:
		return ErrCoursewareReviewImpactPlanConflict
	}

	switch item.Status {
	case models.CWReviewItemStatusDetected,
		models.CWReviewItemStatusDiscussing,
		models.CWReviewItemStatusConfirmed:
	default:
		return ErrCoursewareReviewImpactPlanConflict
	}

	fingerprint, err :=
		coursewareReviewImpactItemFingerprint(item)
	if err != nil {
		return err
	}

	if fingerprint != strings.TrimSpace(
		precondition.Fingerprint,
	) {
		return ErrCoursewareReviewImpactPlanConflict
	}

	return nil
}

func validateCoursewareReviewImpactItemPageFreshTx(
	ctx context.Context,
	tx pgx.Tx,
	item *models.CoursewareReviewItem,
) error {
	if item == nil {
		return ErrCoursewareReviewImpactPlanConflict
	}

	if item.PageID == nil ||
		strings.TrimSpace(*item.PageID) == "" {
		if item.PageNumberSnapshot != 0 {
			return ErrCoursewareReviewImpactPlanConflict
		}

		return nil
	}

	var (
		pageNumber int
		title      string
		htmlHash   string
		updatedAt  time.Time
	)

	err := tx.QueryRow(
		ctx,
		`SELECT
			page_number,
			COALESCE(title, ''),
			encode(
				digest(
					convert_to(COALESCE(html_content, ''), 'UTF8'),
					'sha256'
				),
				'hex'
			),
			updated_at
		 FROM courseware_pages
		 WHERE id = $1
		   AND courseware_id = $2
		 FOR SHARE`,
		strings.TrimSpace(*item.PageID),
		item.CoursewareID,
	).Scan(
		&pageNumber,
		&title,
		&htmlHash,
		&updatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrCoursewareReviewImpactPlanConflict
		}

		return fmt.Errorf(
			"重新读取影响方案整改项页面失败: %w",
			err,
		)
	}

	if pageNumber != item.PageNumberSnapshot ||
		strings.TrimSpace(title) !=
			strings.TrimSpace(item.PageTitleSnapshot) ||
		strings.TrimSpace(htmlHash) !=
			strings.TrimSpace(item.PageHTMLHash) ||
		item.PageUpdatedAtSnapshot == nil ||
		!item.PageUpdatedAtSnapshot.Equal(updatedAt) {
		return ErrCoursewareReviewImpactPlanConflict
	}

	return nil
}

func validateCoursewareReviewImpactCreateItemPageTx(
	ctx context.Context,
	tx pgx.Tx,
	coursewareID string,
	payloadPageID string,
	precondition cwReviewImpactPagePrecondition,
) (*time.Time, error) {
	payloadPageID = strings.TrimSpace(payloadPageID)
	precondition.Scope = strings.TrimSpace(
		precondition.Scope,
	)
	precondition.PageID = strings.TrimSpace(
		precondition.PageID,
	)
	precondition.Title = strings.TrimSpace(
		precondition.Title,
	)
	precondition.HTMLHash = strings.TrimSpace(
		precondition.HTMLHash,
	)

	if precondition.Scope == "global" {
		if payloadPageID != "" ||
			precondition.PageID != "" {
			return nil,
				ErrCoursewareReviewImpactSelectionInvalid
		}

		return nil, nil
	}

	if precondition.Scope != "page" ||
		payloadPageID == "" ||
		payloadPageID != precondition.PageID ||
		precondition.PageNumber <= 0 ||
		precondition.HTMLHash == "" {
		return nil,
			ErrCoursewareReviewImpactSelectionInvalid
	}

	var (
		currentPageNumber int
		currentTitle      string
		currentHTMLHash   string
		currentUpdatedAt  time.Time
	)

	err := tx.QueryRow(
		ctx,
		`SELECT
			page_number,
			COALESCE(title, ''),
			encode(
				digest(
					convert_to(COALESCE(html_content, ''), 'UTF8'),
					'sha256'
				),
				'hex'
			),
			updated_at
		 FROM courseware_pages
		 WHERE id = $1
		   AND courseware_id = $2
		 FOR SHARE`,
		precondition.PageID,
		strings.TrimSpace(coursewareID),
	).Scan(
		&currentPageNumber,
		&currentTitle,
		&currentHTMLHash,
		&currentUpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil,
				ErrCoursewareReviewImpactPlanConflict
		}

		return nil, fmt.Errorf(
			"重新读取影响方案新增问题页面失败: %w",
			err,
		)
	}

	if currentPageNumber != precondition.PageNumber ||
		strings.TrimSpace(currentTitle) !=
			precondition.Title ||
		strings.TrimSpace(currentHTMLHash) !=
			precondition.HTMLHash {
		return nil,
			ErrCoursewareReviewImpactPlanConflict
	}

	return &currentUpdatedAt, nil
}

func coursewareReviewImpactItemFingerprint(
	item *models.CoursewareReviewItem,
) (string, error) {
	raw, err := json.Marshal(item)
	if err != nil {
		return "", fmt.Errorf(
			"生成影响方案整改项当前指纹失败: %w",
			err,
		)
	}

	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func sortedCoursewareReviewImpactKeys(
	values map[string]struct{},
) []string {
	result := make(
		[]string,
		0,
		len(values),
	)

	for value := range values {
		result = append(result, value)
	}

	sort.Strings(result)

	return result
}
