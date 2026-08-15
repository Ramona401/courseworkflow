package services

// courseware_ai_review_impact_plan_contract.go
//
// R-07结构化影响方案的服务层协议校验。
//
// 本文件不执行AI调用，也不写数据库。
// 它只负责冻结操作协议前的严格结构校验和反序列化，供后续：
//   - AI Impact Plan生成服务；
//   - 浏览器安全Preview；
//   - Atomic Apply重新解析不可变operations_json。
// 共同复用。
//
// 重要：浏览器不能调用本层提交operations。
// operations只能由服务端AI生成或服务端确定性构造后进入本校验。

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"strings"

	"tedna/internal/models"
)

const (
	cwAIReviewImpactPlanSchemaVersion = 1
	cwAIReviewImpactPlanMaxOperations = 100
)

var (
	ErrCWAIReviewImpactPlanInvalid = errors.New(
		"课件审核影响方案结构无效",
	)

	ErrCWAIReviewImpactPlanNoOperations = errors.New(
		"课件审核影响方案没有可执行候选操作",
	)

	ErrCWAIReviewImpactPlanConflict = errors.New(
		"课件审核影响方案或目标状态已变化，请刷新后重新生成",
	)

	cwAIReviewImpactOperationIDPattern = regexp.MustCompile(
		`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`,
	)
)

// CWAIReviewImpactPlanRecord 是Service层计划+事件组合。
type CWAIReviewImpactPlanRecord struct {
	Plan *models.CoursewareReviewImpactPlan

	Operations []models.CoursewareReviewImpactOperation
	Events     []*models.CoursewareReviewImpactPlanEvent
}

// validateCWAIReviewImpactOperations 对服务端生成的操作执行协议级fail-closed校验。
//
// 这里只校验所有操作共有的结构；每一种operation_type的payload/preconditions
// 业务字段将在对应生成器和Apply执行器中继续做类型化校验。
func validateCWAIReviewImpactOperations(
	operations []models.CoursewareReviewImpactOperation,
) error {
	if len(operations) == 0 {
		return ErrCWAIReviewImpactPlanNoOperations
	}

	if len(operations) > cwAIReviewImpactPlanMaxOperations {
		return ErrCWAIReviewImpactPlanInvalid
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

		if !cwAIReviewImpactOperationIDPattern.MatchString(
			operation.OperationID,
		) {
			return ErrCWAIReviewImpactPlanInvalid
		}

		if _, exists := seen[operation.OperationID]; exists {
			return ErrCWAIReviewImpactPlanInvalid
		}

		seen[operation.OperationID] = struct{}{}

		if !models.IsCWReviewImpactOperationType(
			operation.OperationType,
		) {
			return ErrCWAIReviewImpactPlanInvalid
		}

		if operation.Summary == "" ||
			operation.Payload == nil ||
			operation.Preconditions == nil {
			return ErrCWAIReviewImpactPlanInvalid
		}
	}

	return nil
}

func marshalCWAIReviewImpactOperations(
	operations []models.CoursewareReviewImpactOperation,
) (string, error) {
	if err := validateCWAIReviewImpactOperations(
		operations,
	); err != nil {
		return "", err
	}

	raw, err := json.Marshal(operations)
	if err != nil {
		return "", ErrCWAIReviewImpactPlanInvalid
	}

	return string(raw), nil
}

// parseCWAIReviewImpactOperationsJSON 严格解析数据库冻结的operations_json。
//
// DisallowUnknownFields确保未来协议升级时，旧后端不会静默忽略未知顶层字段。
// payload和preconditions本身仍是对象，后续按operation_type继续严格解码。
func parseCWAIReviewImpactOperationsJSON(
	raw string,
) ([]models.CoursewareReviewImpactOperation, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, ErrCWAIReviewImpactPlanInvalid
	}

	var operationItems []json.RawMessage
	if err := json.Unmarshal(
		[]byte(raw),
		&operationItems,
	); err != nil {
		return nil, ErrCWAIReviewImpactPlanInvalid
	}

	operations := make(
		[]models.CoursewareReviewImpactOperation,
		0,
		len(operationItems),
	)

	for _, item := range operationItems {
		decoder := json.NewDecoder(bytes.NewReader(item))
		decoder.DisallowUnknownFields()

		var operation models.CoursewareReviewImpactOperation

		if err := decoder.Decode(&operation); err != nil {
			return nil, ErrCWAIReviewImpactPlanInvalid
		}

		var trailing json.RawMessage
		if err := decoder.Decode(&trailing); err != io.EOF {
			return nil, ErrCWAIReviewImpactPlanInvalid
		}

		operations = append(
			operations,
			operation,
		)
	}

	if err := validateCWAIReviewImpactOperations(
		operations,
	); err != nil {
		return nil, err
	}

	return operations, nil
}

func containsCWAIReviewImpactOperationID(
	operations []models.CoursewareReviewImpactOperation,
	operationID string,
) bool {
	operationID = strings.TrimSpace(operationID)

	for _, operation := range operations {
		if operation.OperationID == operationID {
			return true
		}
	}

	return false
}
