package services

// courseware_assistant_context_digest.go
//
// 本文件负责把课件页面转换为教学智能体可使用的确定性摘要。
//
// 复用已有课件AI审核静态提取器，获得：
//   - 可见文字近似值；
//   - 页面互动契约结果；
//   - 静态事件入口；
//   - 可达函数；
//   - 状态变量；
//   - DOM目标；
//   - CSS显隐规则；
//   - 答案暴露风险；
//   - 人工复核标记。
//
// 安全边界：
//   - 不执行页面JavaScript；
//   - 不返回完整页面HTML；
//   - 不返回完整教案；
//   - 浏览器预览只返回互动数量和风险摘要；
//   - 所有截断均按rune处理，避免破坏中文字符。

import (
	"strings"

	"tedna/internal/models"
)

const (
	coursewareAssistantContextTitleMaxRunes      = 300
	coursewareAssistantContextPurposeMaxRunes    = 2000
	coursewareAssistantContextSummaryMaxRunes    = 4000
	coursewareAssistantContextVisualMaxRunes     = 500
	coursewareAssistantContextMediaMaxRunes      = 3000
	coursewareAssistantContextPageIndexMaxRunes  = 5000
	coursewareAssistantAdjacentPurposeMaxRunes   = 1000
	coursewareAssistantAdjacentSummaryMaxRunes   = 1800
	coursewareAssistantLessonPreviewMaxRunes     = 2000
	coursewareAssistantLessonExcerptHardMaxRunes = 12000
)

// buildCoursewareAssistantCurrentPageSnapshot 构建当前页发布快照。
func buildCoursewareAssistantCurrentPageSnapshot(
	page *models.CoursewarePage,
	config models.CoursewareAssistantContextConfig,
) (
	models.AssistantDeploymentPageContextSnapshot,
	string,
) {
	if page == nil {
		return models.AssistantDeploymentPageContextSnapshot{},
			coursewareAssistantSHA256String("")
	}

	digest :=
		BuildCWAIReviewPageDigest(page)

	snapshot :=
		models.AssistantDeploymentPageContextSnapshot{
			PageID:     strings.TrimSpace(page.ID),
			PageNumber: page.PageNumber,
			Title: coursewareAssistantTruncateRunes(
				page.Title,
				coursewareAssistantContextTitleMaxRunes,
			),
			InteractionEvidence: emptyCoursewareAssistantInteractionEvidence(
				"",
			),
		}

	if config.IncludePagePlan {
		snapshot.Purpose =
			coursewareAssistantTruncateRunes(
				digest.Purpose,
				coursewareAssistantContextPurposeMaxRunes,
			)
		snapshot.ContentSummary =
			coursewareAssistantTruncateRunes(
				digest.ContentSummary,
				coursewareAssistantContextSummaryMaxRunes,
			)
		snapshot.InteractionType =
			coursewareAssistantTruncateRunes(
				digest.InteractionType,
				100,
			)
		snapshot.VisualFormat =
			coursewareAssistantTruncateRunes(
				digest.VisualFormat,
				coursewareAssistantContextVisualMaxRunes,
			)
		snapshot.MediaRequirements =
			coursewareAssistantTruncateRunes(
				digest.MediaRequirements,
				coursewareAssistantContextMediaMaxRunes,
			)
		snapshot.PageIndex =
			coursewareAssistantTruncateRunes(
				digest.PageIndex,
				coursewareAssistantContextPageIndexMaxRunes,
			)
	}

	if config.IncludeVisibleText {
		snapshot.VisibleText =
			coursewareAssistantTruncateRunes(
				digest.VisibleText,
				cwAIReviewVisibleTextMaxRunes,
			)
	}

	if config.IncludeInteractionEvidence {
		snapshot.InteractionType =
			coursewareAssistantTruncateRunes(
				digest.InteractionType,
				100,
			)
		snapshot.InteractionEvidence =
			normalizeCoursewareAssistantInteractionEvidence(
				digest.Interaction,
			)
	}

	return snapshot,
		digest.HTMLHash
}

// buildCoursewareAssistantAdjacentPageSnapshot 构建相邻页最小摘要。
func buildCoursewareAssistantAdjacentPageSnapshot(
	page *models.CoursewarePage,
) *models.AssistantDeploymentAdjacentPageSnapshot {
	if page == nil {
		return nil
	}

	return &models.AssistantDeploymentAdjacentPageSnapshot{
		PageID:     strings.TrimSpace(page.ID),
		PageNumber: page.PageNumber,
		Title: coursewareAssistantTruncateRunes(
			page.Title,
			coursewareAssistantContextTitleMaxRunes,
		),
		Purpose: coursewareAssistantTruncateRunes(
			page.Purpose,
			coursewareAssistantAdjacentPurposeMaxRunes,
		),
		ContentSummary: coursewareAssistantTruncateRunes(
			page.ContentSummary,
			coursewareAssistantAdjacentSummaryMaxRunes,
		),
	}
}

// buildCoursewareAssistantLessonExcerpt 构建当前页相关教案片段。
//
// 复用已有按页相关性算法；配置值是上限而不是必须填满的目标长度。
// 既有算法本身可能为了聚焦当前页而返回短于配置上限的内容。
func buildCoursewareAssistantLessonExcerpt(
	lessonContent string,
	page *models.CoursewarePage,
	maxRunes int,
) string {
	if strings.TrimSpace(lessonContent) == "" ||
		page == nil {
		return ""
	}

	if maxRunes <= 0 {
		return ""
	}

	if maxRunes >
		coursewareAssistantLessonExcerptHardMaxRunes {
		maxRunes =
			coursewareAssistantLessonExcerptHardMaxRunes
	}

	relevant :=
		extractPageRelevantLessonSection(
			lessonContent,
			page,
		)

	return coursewareAssistantTruncateRunes(
		relevant,
		maxRunes,
	)
}

// buildCoursewareAssistantContextPreview 构建教师端安全预览。
//
// 预览不包含完整互动代码证据，也不会包含超出限制的教案片段。
func buildCoursewareAssistantContextPreview(
	snapshot models.AssistantDeploymentContextSnapshot,
	config models.CoursewareAssistantContextConfig,
) models.CoursewareAssistantContextPreview {
	preview :=
		models.CoursewareAssistantContextPreview{
			CurrentPage: coursewareAssistantPagePreviewFromCurrent(
				snapshot.CurrentPage,
			),
			Interaction: models.CoursewareAssistantInteractionPreview{
				DeclaredType: snapshot.CurrentPage.
					InteractionEvidence.
					DeclaredType,
				ContractOK: snapshot.CurrentPage.
					InteractionEvidence.
					ContractOK,
				EventCount: len(
					snapshot.CurrentPage.
						InteractionEvidence.
						Events,
				),
				DOMTargetCount: len(
					snapshot.CurrentPage.
						InteractionEvidence.
						DOMTargets,
				),
				RiskFlags: coursewareAssistantCopyStrings(
					snapshot.CurrentPage.
						InteractionEvidence.
						RiskFlags,
				),
				ManualReviewRequired: snapshot.CurrentPage.
					InteractionEvidence.
					ManualReviewRequired,
			},
			ContextConfig: config,
		}

	if snapshot.PreviousPage != nil {
		preview.PreviousPage =
			coursewareAssistantPagePreviewFromAdjacent(
				snapshot.PreviousPage,
			)
	}

	if snapshot.NextPage != nil {
		preview.NextPage =
			coursewareAssistantPagePreviewFromAdjacent(
				snapshot.NextPage,
			)
	}

	if snapshot.LessonPlan != nil {
		preview.LessonPlan =
			&models.CoursewareAssistantLessonPlanPreview{
				LessonPlanID: snapshot.LessonPlan.LessonPlanID,
				Title:        snapshot.LessonPlan.Title,
				ExcerptPreview: coursewareAssistantTruncateRunes(
					snapshot.LessonPlan.
						RelevantExcerpt,
					coursewareAssistantLessonPreviewMaxRunes,
				),
				CharacterCount: len(
					[]rune(
						snapshot.LessonPlan.
							RelevantExcerpt,
					),
				),
			}
	}

	if preview.Interaction.RiskFlags == nil {
		preview.Interaction.RiskFlags =
			[]string{}
	}

	return preview
}

// coursewareAssistantPagePreviewFromCurrent 转换当前页安全预览。
func coursewareAssistantPagePreviewFromCurrent(
	page models.AssistantDeploymentPageContextSnapshot,
) models.CoursewareAssistantPagePreview {
	return models.CoursewareAssistantPagePreview{
		PageID:         page.PageID,
		PageNumber:     page.PageNumber,
		Title:          page.Title,
		Purpose:        page.Purpose,
		ContentSummary: page.ContentSummary,
		VisibleText:    page.VisibleText,
	}
}

// coursewareAssistantPagePreviewFromAdjacent 转换相邻页安全预览。
func coursewareAssistantPagePreviewFromAdjacent(
	page *models.AssistantDeploymentAdjacentPageSnapshot,
) *models.CoursewareAssistantPagePreview {
	if page == nil {
		return nil
	}

	return &models.CoursewareAssistantPagePreview{
		PageID:         page.PageID,
		PageNumber:     page.PageNumber,
		Title:          page.Title,
		Purpose:        page.Purpose,
		ContentSummary: page.ContentSummary,
	}
}

// normalizeCoursewareAssistantInteractionEvidence 复制并稳定所有数组字段。
func normalizeCoursewareAssistantInteractionEvidence(
	source models.CWAIReviewInteractionEvidence,
) models.CWAIReviewInteractionEvidence {
	result := source

	result.DeclaredType =
		strings.TrimSpace(result.DeclaredType)
	result.ContractReason =
		strings.TrimSpace(result.ContractReason)
	result.ContractDetail =
		strings.TrimSpace(result.ContractDetail)

	result.Events =
		append(
			[]models.CWAIReviewInteractionEvent{},
			source.Events...,
		)
	result.ReachableFunctions =
		append(
			[]models.CWAIReviewReachableFunction{},
			source.ReachableFunctions...,
		)
	result.StateVariables =
		coursewareAssistantCopyStrings(
			source.StateVariables,
		)
	result.DOMTargets =
		coursewareAssistantCopyStrings(
			source.DOMTargets,
		)
	result.CSSStateRules =
		coursewareAssistantCopyStrings(
			source.CSSStateRules,
		)
	result.InitialExposureSignals =
		coursewareAssistantCopyStrings(
			source.InitialExposureSignals,
		)
	result.RiskFlags =
		coursewareAssistantCopyStrings(
			source.RiskFlags,
		)

	return result
}

// emptyCoursewareAssistantInteractionEvidence 返回稳定的空互动证据。
func emptyCoursewareAssistantInteractionEvidence(
	declaredType string,
) models.CWAIReviewInteractionEvidence {
	return models.CWAIReviewInteractionEvidence{
		DeclaredType:           strings.TrimSpace(declaredType),
		Events:                 []models.CWAIReviewInteractionEvent{},
		ReachableFunctions:     []models.CWAIReviewReachableFunction{},
		StateVariables:         []string{},
		DOMTargets:             []string{},
		CSSStateRules:          []string{},
		InitialExposureSignals: []string{},
		RiskFlags:              []string{},
	}
}

// coursewareAssistantCopyStrings 复制切片并稳定返回[]而不是null。
func coursewareAssistantCopyStrings(
	values []string,
) []string {
	result := make(
		[]string,
		0,
		len(values),
	)

	for _, value := range values {
		normalized :=
			strings.TrimSpace(value)

		if normalized != "" {
			result = append(
				result,
				normalized,
			)
		}
	}

	return result
}

// coursewareAssistantTruncateRunes 按Unicode字符安全截断。
func coursewareAssistantTruncateRunes(
	value string,
	maxRunes int,
) string {
	value =
		strings.TrimSpace(value)

	if value == "" ||
		maxRunes <= 0 {
		return ""
	}

	runes :=
		[]rune(value)

	if len(runes) <= maxRunes {
		return value
	}

	return string(
		runes[:maxRunes],
	)
}
