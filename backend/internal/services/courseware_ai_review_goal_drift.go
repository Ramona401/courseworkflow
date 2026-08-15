package services

// courseware_ai_review_goal_drift.go
//
// R-01.1 修改要求目标漂移保护的后端人工拆项服务。
//
// 用户明确选择“创建新改进项”后：
//   - 当前问题完全不变；
//   - 当前确认要求和不可变历史完全不变；
//   - 新文字成为同一课件、同一AI审核会话下的独立整改项；
//   - 页级问题使用请求发生时的当前页面快照，而不是复制已经变化的旧哈希；
//   - 新问题从detected开始，后续仍需走正常的人工确认流程；
//   - 不调用AI、不修改页面、不提交审核决定。

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"tedna/internal/models"
	"tedna/internal/repository"
)

const cwReviewGoalDriftTitleMaxRunes = 60

var (
	ErrCWReviewGoalDriftContentInvalid = errors.New(
		"新的独立改进内容不能为空或过长",
	)

	ErrCWReviewGoalDriftForbidden = errors.New(
		"当前身份不能从这条问题创建新的独立改进项",
	)

	ErrCWReviewGoalDriftUnavailable = errors.New(
		"当前问题已经不能继续拆分新的改进项",
	)
)

// CreateCWReviewGoalDriftItem 从当前未交付问题明确创建一条独立问题。
func (s *CoursewareAIReviewService) CreateCWReviewGoalDriftItem(
	ctx context.Context,
	sourceItemID string,
	content string,
	actor *CoursewareActorContext,
) (*models.CoursewareReviewItem, error) {
	if s == nil {
		return nil, errors.New("课件AI审核服务未初始化")
	}

	content = strings.TrimSpace(content)
	if content == "" ||
		utf8.RuneCountInString(content) > cwReviewItemMaxInstructionRunes {
		return nil, ErrCWReviewGoalDriftContentInvalid
	}

	sourceItem, _, err := loadAuthorizedCWReviewItem(
		ctx,
		sourceItemID,
		actor,
	)
	if err != nil {
		return nil, err
	}

	if actor == nil || strings.TrimSpace(actor.UserID) == "" {
		return nil, ErrCWAIReviewActorRequired
	}

	switch sourceItem.SourceType {
	case models.CWReviewItemSourceFormal:
		if actor.UserID != sourceItem.CreatedBy {
			return nil, ErrCWReviewGoalDriftForbidden
		}

	case models.CWReviewItemSourceSelf:
		if actor.UserID != sourceItem.OwnerID {
			return nil, ErrCWReviewGoalDriftForbidden
		}

	default:
		return nil, ErrCWReviewGoalDriftForbidden
	}

	// 已进入正式交付历史后，必须保持不可变。
	if sourceItem.CoursewareReviewID != nil ||
		sourceItem.FeedbackID != nil ||
		sourceItem.DeliveredInstructionVersionID != nil {
		return nil, ErrCWReviewGoalDriftUnavailable
	}

	if err := ensureCWReviewItemActionable(sourceItem); err != nil {
		return nil, err
	}

	page, err := ensureCWReviewItemFresh(
		ctx,
		sourceItem,
		actor.UserID,
	)
	if err != nil {
		return nil, err
	}

	sourceFindingID, err := newCWReviewGoalDriftFindingID()
	if err != nil {
		return nil, err
	}

	newItem, err := buildCWReviewGoalDriftItem(
		sourceItem,
		page,
		content,
		actor.UserID,
		sourceFindingID,
	)
	if err != nil {
		return nil, err
	}

	return repository.CreateCoursewareReviewGoalDriftItem(
		ctx,
		sourceItem.ID,
		actor.UserID,
		newItem,
	)
}

func newCWReviewGoalDriftFindingID() (string, error) {
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("生成目标漂移人工拆项来源ID失败: %w", err)
	}

	return "goal_drift_" + hex.EncodeToString(randomBytes), nil
}

func cwGoalDriftTeacherTitle(content string) string {
	normalized := strings.Join(
		strings.Fields(
			strings.TrimSpace(content),
		),
		" ",
	)

	if normalized == "" {
		return "需要单独处理的新问题"
	}

	runes := []rune(normalized)
	if len(runes) <= cwReviewGoalDriftTitleMaxRunes {
		return normalized
	}

	return string(runes[:cwReviewGoalDriftTitleMaxRunes]) + "…"
}

// buildCWReviewGoalDriftItem 只构造新问题，不写库。
//
// source_goal_drift_item_id只保存在内部EvidenceJSON中。
// 浏览器响应继续经过BuildCWReviewItemTeacherEvidenceJSON，因此不会泄露该内部关联。
func buildCWReviewGoalDriftItem(
	sourceItem *models.CoursewareReviewItem,
	page *repository.CoursewareReviewPageSnapshot,
	content string,
	actorID string,
	sourceFindingID string,
) (*models.CoursewareReviewItem, error) {
	if sourceItem == nil ||
		strings.TrimSpace(actorID) == "" ||
		strings.TrimSpace(sourceFindingID) == "" ||
		strings.TrimSpace(content) == "" {
		return nil, ErrCWReviewGoalDriftContentInvalid
	}

	sourceTeacherView := BuildCWReviewItemTeacherView(sourceItem)

	teacherSnapshot := models.CWAIReviewTeacherViewSnapshot{
		TeacherTitle: cwGoalDriftTeacherTitle(content),
		WhatHappened: "教师在完善当前修改要求时，又发现了这一项需要单独处理的内容：" +
			strings.TrimSpace(content),
		TeachingImpact:  "这是一项独立的新问题，需要单独检查它是否会影响课堂呈现、理解或操作。",
		ImprovementGoal: strings.TrimSpace(content),
		AcceptanceChecks: []string{
			"打开对应页面，对照这条新的改进目标逐项检查。",
			"实际展示或操作相关内容一次，确认调整结果适合课堂使用。",
		},
		TeacherContext:      sourceTeacherView.TeacherContext,
		ManualCheckRequired: true,
	}

	evidence := map[string]interface{}{
		"origin_type":               models.CWReviewItemOriginGoalDriftManual,
		"source_goal_drift_item_id": sourceItem.ID,
		"scope":                     "courseware",
		"teacher_view_snapshot":     teacherSnapshot,
	}

	item := &models.CoursewareReviewItem{
		CoursewareID:    sourceItem.CoursewareID,
		SourceSessionID: sourceItem.SourceSessionID,
		SourceFindingID: strings.TrimSpace(sourceFindingID),

		OriginType: models.CWReviewItemOriginGoalDriftManual,

		SourceType:  sourceItem.SourceType,
		ReviewLevel: sourceItem.ReviewLevel,
		ReviewRound: sourceItem.ReviewRound,

		CreatedBy: strings.TrimSpace(actorID),
		OwnerID:   sourceItem.OwnerID,

		Severity:  sourceItem.Severity,
		Dimension: sourceItem.Dimension,

		Title:       teacherSnapshot.TeacherTitle,
		Description: teacherSnapshot.WhatHappened,

		OriginalSuggestion: teacherSnapshot.ImprovementGoal,
		Status:             models.CWReviewItemStatusDetected,
	}

	if page != nil {
		pageID := strings.TrimSpace(page.ID)

		item.PageID = &pageID
		item.PageNumberSnapshot = page.PageNumber
		item.PageTitleSnapshot = strings.TrimSpace(page.Title)
		item.PageHTMLHash = cwAIReviewHash(page.HTMLContent)
		item.PageUpdatedAtSnapshot = page.UpdatedAt

		evidence["scope"] = "page"
		evidence["page_id"] = pageID
		evidence["page_number_snapshot"] = page.PageNumber
	}

	evidenceJSON, err := json.Marshal(evidence)
	if err != nil {
		return nil, fmt.Errorf("序列化目标漂移教师快照失败: %w", err)
	}

	item.EvidenceJSON = string(evidenceJSON)

	// 再经过统一教师契约规范化，确保内部术语、空字段和检查项规则
	// 与其他AI问题和人工问题完全一致。
	normalizedTeacherView := BuildCWReviewItemTeacherView(item)

	item.Title = normalizedTeacherView.TeacherTitle
	item.Description = normalizedTeacherView.WhatHappened
	item.OriginalSuggestion = normalizedTeacherView.ImprovementGoal

	evidence["teacher_view_snapshot"] = normalizedTeacherView

	evidenceJSON, err = json.Marshal(evidence)
	if err != nil {
		return nil, fmt.Errorf("保存规范化目标漂移教师快照失败: %w", err)
	}

	item.EvidenceJSON = string(evidenceJSON)
	return item, nil
}
