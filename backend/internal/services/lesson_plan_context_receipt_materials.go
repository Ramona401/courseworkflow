package services

// lesson_plan_context_receipt_materials.go
// 备课材料与知识脉络上下文回执。
//
// 本文件负责：
//   - 课本、单元方案、班级学情的实际读取状态；
//   - 原始课程大纲是否因教师明确查询而读取；
//   - active知识脉络是否真正进入本轮提示词；
//   - 组件回执通用辅助函数。
//
// 严格边界：
//   - UseRawCourseOutline=false时，禁止调用课程大纲正文读取仓储；
//   - CourseOutline和KnowledgeLineage必须分别记录；
//   - publisher-only旧关联不能被回执重新解释成唯一大纲；
//   - 回执不得写数据库，也不得触发AI调用。

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// buildTextbookReceipt 构建课本页面回执。
func (s *WorkshopStageService) buildTextbookReceipt(
	ctx context.Context,
	lessonPlan *models.LessonPlan,
) *models.MaterialContextReceipt {
	if lessonPlan == nil {
		return &models.MaterialContextReceipt{
			Status: models.ContextReceiptUnavailable,
			Reason: "教案数据为空，无法确认课本页面",
		}
	}

	raw := strings.TrimSpace(
		lessonPlan.TextbookPageIDs,
	)
	if raw == "" || raw == "[]" {
		return &models.MaterialContextReceipt{
			Status: models.ContextReceiptNotLinked,
			Reason: "本轮没有关联课本页面",
		}
	}

	var ids []string
	if err := json.Unmarshal(
		[]byte(raw),
		&ids,
	); err != nil || len(ids) == 0 {
		return &models.MaterialContextReceipt{
			Status: models.ContextReceiptUnavailable,
			Reason: "课本页面关联数据无法解析",
		}
	}

	if s.textbookService == nil {
		return &models.MaterialContextReceipt{
			Status: models.ContextReceiptUnavailable,
			Count: len(ids),
			Reason: "课本服务当前不可用，本轮未读取课本原文",
		}
	}

	pages, err := repository.GetTextbookPagesByIDs(
		ctx,
		ids,
	)
	if err != nil {
		return &models.MaterialContextReceipt{
			Status: models.ContextReceiptUnavailable,
			Count: len(ids),
			Reason: "关联课本页面读取失败",
		}
	}

	readable := 0
	titles := make(
		[]string,
		0,
		len(pages),
	)

	for _, page := range pages {
		if page == nil {
			continue
		}

		if strings.TrimSpace(
			page.OCRText,
		) != "" {
			readable++
		}

		label := strings.TrimSpace(
			page.Chapter,
		)
		if label == "" {
			label = strings.TrimSpace(
				page.TextbookName,
			)
		}
		if label != "" {
			titles = appendUniqueReceiptTitle(
				titles,
				label,
			)
		}
	}

	// 回执禁止再次调用BuildTextbookContext。
	// 该函数会递增课本使用次数，重复调用会造成同一轮重复计数。
	if len(pages) == 0 {
		return &models.MaterialContextReceipt{
			Status: models.ContextReceiptUnavailable,
			Count: len(ids),
			ReadableCount: readable,
			UnreadableCount: maxReceiptInt(
				0,
				len(ids)-readable,
			),
			Titles: limitReceiptTitles(
				titles,
				5,
			),
			Reason: "课本已关联，但没有读取到可用页面",
		}
	}

	return &models.MaterialContextReceipt{
		Status: models.ContextReceiptLoaded,
		Count: len(ids),
		ReadableCount: readable,
		UnreadableCount: maxReceiptInt(
			0,
			len(ids)-readable,
		),
		Titles: limitReceiptTitles(
			titles,
			5,
		),
	}
}

// buildUnitPlanReceipt 构建单元方案回执。
func buildUnitPlanReceipt(
	ctx context.Context,
	lessonPlan *models.LessonPlan,
) *models.MaterialContextReceipt {
	if lessonPlan == nil {
		return &models.MaterialContextReceipt{
			Status: models.ContextReceiptUnavailable,
			Reason: "教案数据为空，无法确认单元方案",
		}
	}

	if lessonPlan.UnitPlanID == nil ||
		strings.TrimSpace(
			*lessonPlan.UnitPlanID,
		) == "" {
		return &models.MaterialContextReceipt{
			Status: models.ContextReceiptNotLinked,
			Reason: "本轮没有关联单元方案",
		}
	}

	id := strings.TrimSpace(
		*lessonPlan.UnitPlanID,
	)

	unitPlan, err := repository.GetUnitPlanByID(
		ctx,
		id,
	)
	if err != nil {
		return &models.MaterialContextReceipt{
			Status: models.ContextReceiptUnavailable,
			ID: id,
			Reason: "已关联单元方案，但当前无法读取",
		}
	}

	name := strings.TrimSpace(
		unitPlan.UnitTheme,
	)
	if name == "" {
		name = strings.TrimSpace(
			unitPlan.Unit,
		)
	}

	if unitPlan.Status !=
		models.UnitPlanStatusActive {
		return &models.MaterialContextReceipt{
			Status: models.ContextReceiptUnavailable,
			ID: unitPlan.ID,
			Name: name,
			Reason: "单元方案不是已启用状态，本轮未读取",
		}
	}

	if strings.TrimSpace(
		BuildUnitPlanContext(
			unitPlan,
		),
	) == "" {
		return &models.MaterialContextReceipt{
			Status: models.ContextReceiptUnavailable,
			ID: unitPlan.ID,
			Name: name,
			Reason: "单元方案没有可注入内容",
		}
	}

	return &models.MaterialContextReceipt{
		Status: models.ContextReceiptLoaded,
		ID: unitPlan.ID,
		Name: name,
	}
}

// buildCourseOutlineReceipt 构建原始课程大纲回执。
//
// 只有turnPlan.UseRawCourseOutline=true时才读取唯一大纲。
// 普通正式教案、评审和修订不允许通过回执隐藏读取大纲全文。
func buildCourseOutlineReceipt(
	ctx context.Context,
	lessonPlan *models.LessonPlan,
	turnPlan *lessonPlanTurnContextPlan,
) *models.MaterialContextReceipt {
	if lessonPlan == nil {
		return &models.MaterialContextReceipt{
			Status: models.ContextReceiptUnavailable,
			Reason: "教案数据为空，无法确认课程大纲",
		}
	}

	hasExactOutline :=
		lessonPlan.CourseOutlineID != nil &&
			strings.TrimSpace(
				*lessonPlan.CourseOutlineID,
			) != ""

	hasLegacyPublisher :=
		lessonPlan.CourseOutlinePublisher != nil

	if !hasExactOutline &&
		!hasLegacyPublisher {
		return &models.MaterialContextReceipt{
			Status: models.ContextReceiptNotLinked,
			Reason: "本轮没有关联课程大纲",
		}
	}

	if turnPlan == nil ||
		!turnPlan.UseRawCourseOutline {
		reason :=
			"课程大纲来源已关联，但本轮没有明确查询原始大纲，未读取全文"

		if !hasExactOutline &&
			hasLegacyPublisher {
			reason =
				"当前仅保留旧出版社关联，没有唯一课程大纲ID；本轮未读取原始大纲"
		}

		return &models.MaterialContextReceipt{
			Status: models.ContextReceiptDeferred,
			Reason: reason,
		}
	}

	if !hasExactOutline {
		return &models.MaterialContextReceipt{
			Status: models.ContextReceiptUnavailable,
			Reason: "旧出版社关联没有唯一课程大纲ID，无法安全读取原始大纲",
		}
	}

	hits, err := ResolveLessonPlanCourseOutlines(
		ctx,
		lessonPlan,
	)
	if err != nil {
		switch {
		case errors.Is(
			err,
			ErrOutlineExactSelectionForbidden,
		),
			errors.Is(
				err,
				ErrOutlineEducationDomainRequired,
			),
			errors.Is(
				err,
				ErrOutlineEducationDomainMismatch,
			):
			return &models.MaterialContextReceipt{
				Status: models.ContextReceiptForbidden,
				ID: strings.TrimSpace(
					*lessonPlan.CourseOutlineID,
				),
				Reason: "已关联课程大纲当前无权读取或教育域不一致",
			}

		default:
			return &models.MaterialContextReceipt{
				Status: models.ContextReceiptUnavailable,
				ID: strings.TrimSpace(
					*lessonPlan.CourseOutlineID,
				),
				Reason: "唯一课程大纲当前无法读取",
			}
		}
	}

	if len(hits) != 1 ||
		hits[0] == nil {
		return &models.MaterialContextReceipt{
			Status: models.ContextReceiptUnavailable,
			ID: strings.TrimSpace(
				*lessonPlan.CourseOutlineID,
			),
			Reason: "唯一课程大纲关联没有解析到一份可用记录",
		}
	}

	outline := hits[0]
	content := strings.TrimSpace(
		outline.Content,
	)
	if content == "" {
		return &models.MaterialContextReceipt{
			Status: models.ContextReceiptUnavailable,
			ID: outline.ID,
			Name: outline.Title,
			Reason: "唯一课程大纲正文为空",
		}
	}

	titles := make([]string, 0, 1)
	if strings.TrimSpace(
		outline.Title,
	) != "" {
		titles = append(
			titles,
			strings.TrimSpace(
				outline.Title,
			),
		)
	}

	return &models.MaterialContextReceipt{
		Status: models.ContextReceiptLoaded,
		ID: outline.ID,
		Name: outline.Title,
		Count: 1,
		CharacterCount: len(
			[]rune(content),
		),
		Titles: titles,
		Reason: "老师本轮明确查询课程大纲原文或版本要求，已读取唯一挂载大纲",
	}
}

// buildKnowledgeLineageReceipt 构建active知识脉络回执。
func buildKnowledgeLineageReceipt(
	ctx context.Context,
	lessonPlan *models.LessonPlan,
	turnPlan *lessonPlanTurnContextPlan,
) *models.MaterialContextReceipt {
	if lessonPlan == nil {
		return &models.MaterialContextReceipt{
			Status: models.ContextReceiptUnavailable,
			Reason: "教案数据为空，无法确认知识脉络",
		}
	}

	if lessonPlan.CourseOutlineID == nil ||
		strings.TrimSpace(
			*lessonPlan.CourseOutlineID,
		) == "" {
		return &models.MaterialContextReceipt{
			Status: models.ContextReceiptNotLinked,
			Reason: "没有唯一课程大纲来源，不适用知识脉络快照",
		}
	}

	if turnPlan == nil ||
		!turnPlan.UseKnowledgeLineage {
		reason :=
			"知识脉络来源已关联，但本轮没有使用active快照"

		if strings.TrimSpace(
			lessonPlan.CurrentStage,
		) == "analyze" {
			reason =
				"教学分析尚未完成确认，当前不会提前生成或注入知识脉络"
		}

		if turnPlan != nil &&
			turnPlan.UseRawCourseOutline {
			reason =
				"本轮明确查询原始课程大纲，未同时注入知识脉络以避免混淆资料查询与正式课程锚点"
		}

		return &models.MaterialContextReceipt{
			Status: models.ContextReceiptDeferred,
			Reason: reason,
		}
	}

	lineage, err :=
		repository.GetActiveLessonPlanKnowledgeLineage(
			ctx,
			lessonPlan.ID,
		)
	if err != nil {
		return &models.MaterialContextReceipt{
			Status: models.ContextReceiptUnavailable,
			Reason: "active知识脉络当前读取失败",
		}
	}

	if lineage == nil ||
		!lineage.IsActiveUsable() ||
		strings.TrimSpace(
			lineage.ContextText,
		) == "" {
		return &models.MaterialContextReceipt{
			Status: models.ContextReceiptUnavailable,
			Reason: "没有与当前课程大纲绑定一致的active知识脉络",
		}
	}

	return &models.MaterialContextReceipt{
		Status: models.ContextReceiptLoaded,
		ID: lineage.ID,
		Name: "本课统一知识脉络",
		Count: 1,
		CharacterCount: len(
			[]rune(
				strings.TrimSpace(
					lineage.ContextText,
				),
			),
		),
		Reason: "本轮已读取教师确认后生成的active知识脉络短版上下文",
	}
}

// buildClassProfileReceipt 构建班级学情回执。
func buildClassProfileReceipt(
	ctx context.Context,
	lessonPlan *models.LessonPlan,
	stageCode string,
) *models.MaterialContextReceipt {
	if lessonPlan == nil {
		return &models.MaterialContextReceipt{
			Status: models.ContextReceiptUnavailable,
			Reason: "教案数据为空，无法确认班级学情",
		}
	}

	if lessonPlan.ClassProfileID == nil ||
		strings.TrimSpace(
			*lessonPlan.ClassProfileID,
		) == "" {
		return &models.MaterialContextReceipt{
			Status: models.ContextReceiptNotLinked,
			Reason: "本轮没有关联班级学情",
		}
	}

	id := strings.TrimSpace(
		*lessonPlan.ClassProfileID,
	)

	if stageCode != "analyze" &&
		stageCode != "design" &&
		stageCode != "write" {
		return &models.MaterialContextReceipt{
			Status: models.ContextReceiptNotApplicable,
			ID: id,
			Reason: "班级学情只在教学分析、教学设计和教案撰写阶段读取",
		}
	}

	classProfile, err :=
		repository.GetClassProfileByID(
			ctx,
			id,
		)
	if err != nil {
		return &models.MaterialContextReceipt{
			Status: models.ContextReceiptUnavailable,
			ID: id,
			Reason: "已关联班级学情，但当前无法读取",
		}
	}

	if classProfile.Status !=
		models.ClassProfileStatusActive {
		return &models.MaterialContextReceipt{
			Status: models.ContextReceiptUnavailable,
			ID: classProfile.ID,
			Name: classProfile.ClassName,
			Reason: "班级学情不是已启用状态，本轮未读取",
		}
	}

	if classProfile.CreatedBy !=
		lessonPlan.AuthorID {
		return &models.MaterialContextReceipt{
			Status: models.ContextReceiptForbidden,
			ID: classProfile.ID,
			Name: classProfile.ClassName,
			Reason: "班级学情归属与教案作者不一致，本轮未读取",
		}
	}

	if strings.TrimSpace(
		BuildClassProfileContext(
			classProfile,
		),
	) == "" {
		return &models.MaterialContextReceipt{
			Status: models.ContextReceiptUnavailable,
			ID: classProfile.ID,
			Name: classProfile.ClassName,
			Reason: "班级学情没有可注入内容",
		}
	}

	return &models.MaterialContextReceipt{
		Status: models.ContextReceiptLoaded,
		ID: classProfile.ID,
		Name: classProfile.ClassName,
	}
}

// componentReceiptFromGroups 使用实际加载组件构建回执。
func componentReceiptFromGroups(
	groups []*models.MatchedComponentGroup,
	mode string,
	candidateCount int,
	reranked bool,
) *models.ComponentsContextReceipt {
	items := make(
		[]models.ComponentContextReceiptItem,
		0,
	)

	for _, group := range groups {
		if group == nil {
			continue
		}

		for _, component := range group.Components {
			if component == nil {
				continue
			}

			items = append(
				items,
				models.ComponentContextReceiptItem{
					ID: component.ID,
					LibraryType: group.LibraryType,
					LibraryName: group.LibraryName,
					DisplayLabel: component.DisplayLabel,
					QualityScore: component.QualityScore,
				},
			)
		}
	}

	if len(items) == 0 {
		return &models.ComponentsContextReceipt{
			Status: models.ContextReceiptNotFound,
			SelectionMode: mode,
			Reason: "本轮没有实际读取专业组件",
		}
	}

	return &models.ComponentsContextReceipt{
		Status: models.ContextReceiptLoaded,
		SelectionMode: mode,
		CandidateCount: candidateCount,
		Reranked: reranked,
		Items: items,
	}
}

// countMatchedReceiptComponents 统计实际匹配组件数量。
func countMatchedReceiptComponents(
	groups []*models.MatchedComponentGroup,
) int {
	total := 0

	for _, group := range groups {
		if group == nil {
			continue
		}
		total += len(
			group.Components,
		)
	}

	return total
}

// appendUniqueReceiptTitle 向标题列表追加非空唯一值。
func appendUniqueReceiptTitle(
	items []string,
	value string,
) []string {
	value = strings.TrimSpace(
		value,
	)
	if value == "" {
		return items
	}

	for _, item := range items {
		if item == value {
			return items
		}
	}

	return append(items, value)
}

// limitReceiptTitles 限制回执标题数量。
func limitReceiptTitles(
	items []string,
	limit int,
) []string {
	if len(items) <= limit {
		return items
	}

	return items[:limit]
}

// maxReceiptInt 返回较大整数。
func maxReceiptInt(
	left int,
	right int,
) int {
	if left > right {
		return left
	}

	return right
}

