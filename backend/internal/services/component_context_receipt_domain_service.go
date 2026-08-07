package services

// component_context_receipt_domain_service.go — 组件上下文回执教育域链。
//
// 回执必须与正式提示词读取结果完全一致：
//
//   1. 使用lesson_plans.education_domain创建时快照。
//   2. 历史手动ID只加载同域或common，异域和失效ID静默过滤。
//   3. 手动历史ID全部失效时，不提前结束；继续回退到配方组件。
//   4. 配方历史ID全部失效时，继续回退到同域自动匹配。
//   5. 自动匹配和技能精排候选池从源头按教育域过滤。
//   6. Items和CandidateCount只记录真正读取的组件。
//   7. 被过滤组件的ID、标题和其它信息不进入回执。
//
// 本文件复用第5组A建立的运行时教育域辅助函数，避免提示词与回执
// 各维护一套教育域判定逻辑。

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"tedna/internal/models"
	"tedna/internal/utils"
)

// buildComponentsReceipt 构建与正式提示词一致的组件回执。
func (s *WorkshopStageService) buildComponentsReceipt(
	ctx context.Context,
	lessonPlan *models.LessonPlan,
	stage *models.WorkshopStage,
	recipe *models.TeachingRecipe,
	stageCode string,
	recentUserText string,
) *models.ComponentsContextReceipt {
	if lessonPlan == nil {
		return &models.ComponentsContextReceipt{
			Status: models.ContextReceiptUnavailable,
			Reason: "教案数据为空，无法确认专业组件",
		}
	}

	if stage == nil ||
		strings.TrimSpace(stage.ComponentTypes) == "" ||
		stage.ComponentTypes == "[]" {
		return &models.ComponentsContextReceipt{
			Status: models.ContextReceiptNotApplicable,
			Reason: "当前阶段没有配置专业组件类型",
		}
	}

	educationDomain := strings.ToLower(
		strings.TrimSpace(
			lessonPlan.EducationDomain,
		),
	)

	if !models.IsTeachingEducationDomain(
		educationDomain,
	) {
		return &models.ComponentsContextReceipt{
			Status: models.ContextReceiptUnavailable,
			Reason: "教案教育域快照无效，本轮未读取专业组件",
		}
	}

	stageTypes, ok := parseRuntimeStageTypes(
		stage.ComponentTypes,
	)
	if !ok || len(stageTypes) == 0 {
		return &models.ComponentsContextReceipt{
			Status: models.ContextReceiptUnavailable,
			Reason: "当前阶段组件类型配置无效，本轮未读取专业组件",
		}
	}

	runtimeContext := withLessonComponentDomain(
		ctx,
		educationDomain,
	)

	selectedIDs := s.getSelectedComponentIDsFromOutput(
		ctx,
		lessonPlan.ID,
		stageCode,
	)

	if len(selectedIDs) > 0 {
		groups, err := loadReceiptHistoricalComponentGroups(
			runtimeContext,
			selectedIDs,
			stageTypes,
		)

		if err == nil && countRuntimeComponents(groups) > 0 {
			return runtimeComponentReceiptFromGroups(
				groups,
				"manual",
				false,
			)
		}

		// 与正式提示词保持一致：
		// 历史选择全部失效时不把这些ID标记为已加载，
		// 也不提前结束，而是继续尝试配方与自动匹配。
		wsLog.Info(
			"上下文回执忽略不可用的历史手动组件并继续回退",
			"plan_id",
			lessonPlan.ID,
			"stage",
			stageCode,
			"submitted_count",
			len(NormalizeUniqueComponentIDs(selectedIDs)),
		)
	}

	if recipe != nil {
		var recipeIDs []string

		if err := json.Unmarshal(
			[]byte(recipe.ComponentIDs),
			&recipeIDs,
		); err == nil &&
			len(recipeIDs) > 0 {
			groups, loadErr :=
				loadReceiptHistoricalComponentGroups(
					runtimeContext,
					recipeIDs,
					stageTypes,
				)

			if loadErr == nil &&
				countRuntimeComponents(groups) > 0 {
				return runtimeComponentReceiptFromGroups(
					groups,
					"recipe",
					false,
				)
			}

			wsLog.Info(
				"上下文回执忽略不可用的历史配方组件并继续自动匹配",
				"plan_id",
				lessonPlan.ID,
				"stage",
				stageCode,
			)
		}
	}

	return buildAutoComponentsReceipt(
		runtimeContext,
		stage.ComponentTypes,
		lessonPlan.Subject,
		lessonPlan.Grade,
		stageCode,
		recentUserText,
	)
}

// loadReceiptHistoricalComponentGroups 加载并按阶段类型过滤历史组件。
func loadReceiptHistoricalComponentGroups(
	ctx context.Context,
	componentIDs []string,
	stageTypes []string,
) ([]*models.MatchedComponentGroup, error) {
	groups, err := loadHistoricalComponentGroupsForRuntime(
		ctx,
		componentIDs,
	)
	if err != nil {
		return nil, err
	}

	return filterComponentGroupsByLibraryTypes(
		groups,
		stageTypes,
	), nil
}

// runtimeComponentReceiptFromGroups 使用实际加载数量生成回执。
//
// CandidateCount不再使用原始历史ID数量，避免异域、停用、未审核或不存在的
// ID被计入“候选/已读取”统计。
func runtimeComponentReceiptFromGroups(
	groups []*models.MatchedComponentGroup,
	mode string,
	reranked bool,
) *models.ComponentsContextReceipt {
	return componentReceiptFromGroups(
		groups,
		mode,
		countRuntimeComponents(groups),
		reranked,
	)
}

// buildAutoComponentsReceipt 构建域感知自动匹配或精排回执。
func buildAutoComponentsReceipt(
	ctx context.Context,
	componentTypesJSON string,
	subject string,
	grade string,
	stageCode string,
	recentUserText string,
) *models.ComponentsContextReceipt {
	if _, ok := lessonComponentDomainFromContext(ctx); !ok {
		return &models.ComponentsContextReceipt{
			Status:        models.ContextReceiptUnavailable,
			SelectionMode: "auto",
			Reason:        "教案教育域快照无效，本轮未读取专业组件",
		}
	}

	stageTypes, ok := parseRuntimeStageTypes(
		componentTypesJSON,
	)
	if !ok || len(stageTypes) == 0 {
		return &models.ComponentsContextReceipt{
			Status: models.ContextReceiptNotApplicable,
			Reason: "当前阶段没有可匹配的组件类型",
		}
	}

	useRerank := skillRouterRerankEnabled &&
		strings.TrimSpace(recentUserText) != ""

	limit := 2
	if useRerank {
		limit = skillRouterCandidateLimitPerType
	}

	request := &models.MatchComponentsRequest{
		Subject: strings.TrimSpace(
			subject,
		),
		GradeRange: utils.NormalizeGradeToNumber(
			grade,
		),
		LibraryTypes: stageTypes,
		Limit:        limit,
	}

	if timings, exists := stageTimingMap[stageCode]; exists {
		request.StageTiming = timings
	}

	groups, err := matchStageComponentsForRuntime(
		ctx,
		request,
	)
	if err != nil {
		reason := "专业组件候选读取失败"

		if errors.Is(
			err,
			ErrComponentEducationDomainInvalid,
		) {
			reason = "教案教育域快照无效，本轮未读取专业组件"
		}

		return &models.ComponentsContextReceipt{
			Status:        models.ContextReceiptUnavailable,
			SelectionMode: "auto",
			Reason:        reason,
		}
	}

	if len(groups) == 0 {
		return &models.ComponentsContextReceipt{
			Status:        models.ContextReceiptNotFound,
			SelectionMode: "auto",
			Reason:        "本轮没有匹配到可用专业组件",
		}
	}

	if !useRerank {
		return runtimeComponentReceiptFromGroups(
			groups,
			"auto",
			false,
		)
	}

	return buildRuntimeRerankedReceipt(
		groups,
		recentUserText,
	)
}

// buildRuntimeRerankedReceipt 对同域候选执行与正式提示词一致的精排。
func buildRuntimeRerankedReceipt(
	groups []*models.MatchedComponentGroup,
	recentUserText string,
) *models.ComponentsContextReceipt {
	candidates := make(
		[]*rerankCandidate,
		0,
	)

	originalRank := 0

	for _, group := range groups {
		if group == nil {
			continue
		}

		for _, component := range group.Components {
			if component == nil {
				continue
			}

			candidates = append(
				candidates,
				&rerankCandidate{
					libraryType:  group.LibraryType,
					libraryName:  group.LibraryName,
					component:    component,
					originalRank: originalRank,
				},
			)

			originalRank++
		}
	}

	if len(candidates) == 0 {
		return &models.ComponentsContextReceipt{
			Status:        models.ContextReceiptNotFound,
			SelectionMode: "reranked",
			Reason:        "本轮没有匹配到可用专业组件",
		}
	}

	keywords := extractRerankKeywords(
		recentUserText,
	)

	if len(keywords) > 0 {
		for _, candidate := range candidates {
			candidate.score = scoreCandidateLexical(
				candidate.component,
				keywords,
			)
		}

		sortRuntimeCandidates(
			candidates,
		)
	}

	topN := skillRouterTopN
	if topN > len(candidates) {
		topN = len(candidates)
	}

	chosen := candidates[:topN]

	items := make(
		[]models.ComponentContextReceiptItem,
		0,
		len(chosen),
	)

	for _, candidate := range chosen {
		items = append(
			items,
			models.ComponentContextReceiptItem{
				ID:           candidate.component.ID,
				LibraryType:  candidate.libraryType,
				LibraryName:  candidate.libraryName,
				DisplayLabel: candidate.component.DisplayLabel,
				QualityScore: candidate.component.QualityScore,
			},
		)
	}

	return &models.ComponentsContextReceipt{
		Status:         models.ContextReceiptLoaded,
		SelectionMode:  "reranked",
		CandidateCount: len(candidates),
		Reranked:       len(keywords) > 0,
		Items:          items,
	}
}
