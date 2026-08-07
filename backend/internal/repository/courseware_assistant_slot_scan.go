package repository

// courseware_assistant_slot_scan.go
//
// 本文件集中处理课件教学智能体插槽的：
//   1. 查询列定义；
//   2. 数据库行扫描；
//   3. JSONB与明确Go协议之间的转换；
//   4. PostgreSQL约束错误到稳定仓储错误的映射。
//
// 安全边界：
//   - 查询只关联助手名称和启用状态；
//   - 不读取ai_assistants.full_prompt；
//   - 数据库原始JSON不能直接返回浏览器；
//   - 数据库中出现无法解析的JSON时明确报错，不静默返回空方案。

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"tedna/internal/models"
)

var (
	// ErrCoursewareAssistantSlotNotFound 表示指定课件、页面和插槽组合不存在。
	ErrCoursewareAssistantSlotNotFound = errors.New(
		"课件教学智能体插槽不存在",
	)

	// ErrCoursewareAssistantSlotAlreadyExists 表示同一稳定页面已存在插槽。
	ErrCoursewareAssistantSlotAlreadyExists = errors.New(
		"当前课件页面已经存在教学智能体插槽",
	)

	// ErrCoursewareAssistantSlotPageNotFound 表示目标页面不存在、
	// 不属于当前课件，或所属课件已经进入回收站。
	ErrCoursewareAssistantSlotPageNotFound = errors.New(
		"课件教学智能体插槽目标页面不存在",
	)

	// ErrCoursewareAssistantSlotCreatorNotFound 表示审计创建者不存在。
	ErrCoursewareAssistantSlotCreatorNotFound = errors.New(
		"课件教学智能体插槽创建者不存在",
	)

	// ErrCoursewareAssistantSlotInvalidInput 表示仓储调用参数缺失。
	ErrCoursewareAssistantSlotInvalidInput = errors.New(
		"课件教学智能体插槽参数无效",
	)

	// ErrCoursewareAssistantSlotStoredJSON 表示数据库中的结构化方案损坏。
	ErrCoursewareAssistantSlotStoredJSON = errors.New(
		"课件教学智能体插槽结构化数据无法解析",
	)
)

// coursewareAssistantSlotSelectColumns 是插槽稳定查询列。
//
// 末尾只读取助手名称和启用状态，不读取助手完整提示词。
const coursewareAssistantSlotSelectColumns = `
	slot.id,
	slot.courseware_id,
	slot.page_id,
	slot.assistant_id,
	slot.created_by,
	slot.display_mode,
	slot.display_position,
	slot.title,
	slot.welcome_message,
	slot.teaching_role,
	slot.learning_objective,
	slot.guidance_plan_json::text,
	slot.context_config_json::text,
	slot.status,
	slot.created_at,
	slot.updated_at,
	COALESCE(assistant.name, ''),
	COALESCE(assistant.is_active, false)`

// coursewareAssistantSlotReadJoins 同时保证：
//   - 课件尚未进入回收站；
//   - page_id确实属于courseware_id；
//   - 助手删除后仍能读取插槽，名称回退为空。
const coursewareAssistantSlotReadJoins = `
	JOIN coursewares AS courseware
		ON courseware.id = slot.courseware_id
		AND courseware.deleted_at IS NULL
	JOIN courseware_pages AS page
		ON page.id = slot.page_id
		AND page.courseware_id = slot.courseware_id
	LEFT JOIN ai_assistants AS assistant
		ON assistant.id = slot.assistant_id`

// coursewareAssistantSlotScanner 兼容pgx.Row和pgx.Rows。
type coursewareAssistantSlotScanner interface {
	Scan(dest ...interface{}) error
}

// scanCoursewareAssistantSlotView 扫描并解码一个浏览器安全插槽视图。
func scanCoursewareAssistantSlotView(
	scanner coursewareAssistantSlotScanner,
) (
	*models.CoursewareAssistantSlotView,
	error,
) {
	row := &models.CoursewareAssistantSlot{}
	assistantName := ""
	assistantActive := false

	err := scanner.Scan(
		&row.ID,
		&row.CoursewareID,
		&row.PageID,
		&row.AssistantID,
		&row.CreatedBy,
		&row.DisplayMode,
		&row.DisplayPosition,
		&row.Title,
		&row.WelcomeMessage,
		&row.TeachingRole,
		&row.LearningObjective,
		&row.GuidancePlanJSON,
		&row.ContextConfigJSON,
		&row.Status,
		&row.CreatedAt,
		&row.UpdatedAt,
		&assistantName,
		&assistantActive,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCoursewareAssistantSlotNotFound
		}

		return nil, fmt.Errorf(
			"扫描课件教学智能体插槽失败: %w",
			err,
		)
	}

	guidancePlan := models.CoursewareAssistantGuidancePlan{}
	if err := json.Unmarshal(
		[]byte(row.GuidancePlanJSON),
		&guidancePlan,
	); err != nil {
		return nil, fmt.Errorf(
			"%w: guidance_plan_json: %v",
			ErrCoursewareAssistantSlotStoredJSON,
			err,
		)
	}

	contextConfig := models.CoursewareAssistantContextConfig{}
	if err := json.Unmarshal(
		[]byte(row.ContextConfigJSON),
		&contextConfig,
	); err != nil {
		return nil, fmt.Errorf(
			"%w: context_config_json: %v",
			ErrCoursewareAssistantSlotStoredJSON,
			err,
		)
	}

	normalizeCoursewareAssistantGuidancePlan(
		&guidancePlan,
	)

	return &models.CoursewareAssistantSlotView{
		ID:                row.ID,
		CoursewareID:      row.CoursewareID,
		PageID:            row.PageID,
		AssistantID:       row.AssistantID,
		AssistantName:     assistantName,
		AssistantActive:   assistantActive,
		DisplayMode:       row.DisplayMode,
		DisplayPosition:   row.DisplayPosition,
		Title:             row.Title,
		WelcomeMessage:    row.WelcomeMessage,
		TeachingRole:      row.TeachingRole,
		LearningObjective: row.LearningObjective,
		GuidancePlan:      guidancePlan,
		ContextConfig:     contextConfig,
		Status:            row.Status,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}, nil
}

// marshalCoursewareAssistantSlotPayload 将明确协议编码为JSONB文本。
func marshalCoursewareAssistantSlotPayload(
	guidancePlan models.CoursewareAssistantGuidancePlan,
	contextConfig models.CoursewareAssistantContextConfig,
) (
	string,
	string,
	error,
) {
	guidanceJSON, err := json.Marshal(
		guidancePlan,
	)
	if err != nil {
		return "", "", fmt.Errorf(
			"序列化课件教学智能体方案失败: %w",
			err,
		)
	}

	contextJSON, err := json.Marshal(
		contextConfig,
	)
	if err != nil {
		return "", "", fmt.Errorf(
			"序列化课件教学智能体上下文配置失败: %w",
			err,
		)
	}

	return string(guidanceJSON),
		string(contextJSON),
		nil
}

// normalizeCoursewareAssistantGuidancePlan 保证数组字段稳定返回[]而不是null。
func normalizeCoursewareAssistantGuidancePlan(
	plan *models.CoursewareAssistantGuidancePlan,
) {
	if plan == nil {
		return
	}

	if plan.GuidingPrinciples == nil {
		plan.GuidingPrinciples = []string{}
	}
	if plan.QuestionChain == nil {
		plan.QuestionChain =
			[]models.CoursewareAssistantQuestionStep{}
	}
	if plan.MisconceptionBranches == nil {
		plan.MisconceptionBranches =
			[]models.CoursewareAssistantMisconceptionBranch{}
	}
	if plan.ForbiddenBehaviors == nil {
		plan.ForbiddenBehaviors = []string{}
	}
	if plan.CompletionCriteria == nil {
		plan.CompletionCriteria = []string{}
	}
	if plan.AnswerLeakPolicy.ProhibitedBehaviors == nil {
		plan.AnswerLeakPolicy.ProhibitedBehaviors =
			[]string{}
	}

	for index := range plan.QuestionChain {
		step := &plan.QuestionChain[index]

		if step.ExpectedSignals == nil {
			step.ExpectedSignals = []string{}
		}
		if step.HintLadder == nil {
			step.HintLadder = []string{}
		}
		if step.MisconceptionBranchIDs == nil {
			step.MisconceptionBranchIDs = []string{}
		}
	}

	for index := range plan.MisconceptionBranches {
		branch := &plan.MisconceptionBranches[index]

		if branch.MatchSignals == nil {
			branch.MatchSignals = []string{}
		}
	}
}

// mapCoursewareAssistantSlotWriteError 将约束错误转换为稳定仓储错误。
func mapCoursewareAssistantSlotWriteError(
	err error,
) error {
	if err == nil {
		return nil
	}

	pgError := &pgconn.PgError{}
	if !errors.As(err, &pgError) {
		return err
	}

	switch pgError.ConstraintName {
	case "uq_courseware_assistant_slots_courseware_page":
		return ErrCoursewareAssistantSlotAlreadyExists

	case "fk_courseware_assistant_slots_courseware",
		"fk_courseware_assistant_slots_page":
		return ErrCoursewareAssistantSlotPageNotFound

	case "fk_courseware_assistant_slots_assistant":
		return ErrAIAssistantNotFound

	case "fk_courseware_assistant_slots_creator":
		return ErrCoursewareAssistantSlotCreatorNotFound
	}

	return err
}
