package services

// component_runtime_domain_service.go — 教案运行时组件教育域辅助。
//
// 本文件统一承载提示词与推荐链的运行时域上下文：
//   - 使用lesson_plans.education_domain创建时快照；
//   - 只允许k12、vocational、adult作为运行时当前域；
//   - 历史ID只加载同域或common；
//   - 自动匹配只查询同域或common；
//   - invalid、common、mixed和空值全部fail-closed。
//
// Context只用于同一调用栈内向既有提示词函数传递教案快照域，
// 不接受HTTP参数，也不从登录Actor教育域推断。

import (
	"context"
	"encoding/json"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

type lessonComponentDomainContextKey struct{}

// withLessonComponentDomain 把具体教案快照域写入当前调用栈Context。
func withLessonComponentDomain(
	ctx context.Context,
	educationDomain string,
) context.Context {
	educationDomain = strings.ToLower(
		strings.TrimSpace(educationDomain),
	)

	return context.WithValue(
		ctx,
		lessonComponentDomainContextKey{},
		educationDomain,
	)
}

// lessonComponentDomainFromContext 读取并严格校验运行时教案域。
func lessonComponentDomainFromContext(
	ctx context.Context,
) (string, bool) {
	if ctx == nil {
		return "", false
	}

	value := ctx.Value(
		lessonComponentDomainContextKey{},
	)

	educationDomain, ok := value.(string)
	if !ok {
		return "", false
	}

	educationDomain = strings.ToLower(
		strings.TrimSpace(educationDomain),
	)

	if !models.IsTeachingEducationDomain(
		educationDomain,
	) {
		return "", false
	}

	return educationDomain, true
}

// parseRuntimeStageTypes 解析阶段组件类型并去重。
//
// 非法JSON、空值或未知library_type全部fail-closed。
func parseRuntimeStageTypes(
	raw string,
) ([]string, bool) {
	raw = strings.TrimSpace(raw)

	if raw == "" || raw == "[]" {
		return []string{}, true
	}

	var values []string

	if err := json.Unmarshal(
		[]byte(raw),
		&values,
	); err != nil {
		return nil, false
	}

	result := make(
		[]string,
		0,
		len(values),
	)

	seen := make(
		map[string]bool,
		len(values),
	)

	for _, rawValue := range values {
		libraryType := strings.TrimSpace(
			rawValue,
		)

		if libraryType == "" ||
			!models.IsValidLibraryType(
				libraryType,
			) {
			return nil, false
		}

		if seen[libraryType] {
			continue
		}

		seen[libraryType] = true

		result = append(
			result,
			libraryType,
		)
	}

	return result, true
}

// loadHistoricalComponentGroupsForRuntime 安全加载历史组件ID。
func loadHistoricalComponentGroupsForRuntime(
	ctx context.Context,
	componentIDs []string,
) ([]*models.MatchedComponentGroup, error) {
	educationDomain, ok :=
		lessonComponentDomainFromContext(ctx)

	if !ok {
		return nil,
			ErrComponentEducationDomainInvalid
	}

	return LoadHistoricalLessonComponentGroups(
		ctx,
		componentIDs,
		educationDomain,
	)
}

// matchStageComponentsForRuntime 执行域感知自动匹配。
func matchStageComponentsForRuntime(
	ctx context.Context,
	request *models.MatchComponentsRequest,
) ([]*models.MatchedComponentGroup, error) {
	educationDomain, ok :=
		lessonComponentDomainFromContext(ctx)

	if !ok {
		return nil,
			ErrComponentEducationDomainInvalid
	}

	return repository.MatchComponentsForEducationDomain(
		ctx,
		request,
		educationDomain,
	)
}

// filterComponentGroupsByLibraryTypes 按阶段类型过滤历史分组。
//
// 保留原分组顺序和组内顺序，不修改历史JSON。
func filterComponentGroupsByLibraryTypes(
	groups []*models.MatchedComponentGroup,
	allowedTypes []string,
) []*models.MatchedComponentGroup {
	if len(groups) == 0 ||
		len(allowedTypes) == 0 {
		return []*models.MatchedComponentGroup{}
	}

	allowed := make(
		map[string]bool,
		len(allowedTypes),
	)

	for _, libraryType := range allowedTypes {
		libraryType = strings.TrimSpace(
			libraryType,
		)

		if libraryType != "" {
			allowed[libraryType] = true
		}
	}

	result := make(
		[]*models.MatchedComponentGroup,
		0,
		len(groups),
	)

	for _, group := range groups {
		if group == nil ||
			!allowed[group.LibraryType] ||
			len(group.Components) == 0 {
			continue
		}

		result = append(
			result,
			group,
		)
	}

	return result
}

// countRuntimeComponents 统计实际加载的组件数量。
func countRuntimeComponents(
	groups []*models.MatchedComponentGroup,
) int {
	total := 0

	for _, group := range groups {
		if group == nil {
			continue
		}

		total += len(group.Components)
	}

	return total
}

// AutoMatchStageComponentsForRuntime 域感知自动匹配提示词组件。
func AutoMatchStageComponentsForRuntime(
	ctx context.Context,
	componentTypesJSON string,
	subject string,
	grade string,
	stageCode string,
) string {
	stageTypes, ok :=
		parseRuntimeStageTypes(
			componentTypesJSON,
		)

	if !ok || len(stageTypes) == 0 {
		return ""
	}

	request := &models.MatchComponentsRequest{
		Subject: strings.TrimSpace(
			subject,
		),
		GradeRange: utils.NormalizeGradeToNumber(
			grade,
		),
		LibraryTypes: stageTypes,
		Limit:        2,
	}

	if timings, exists :=
		stageTimingMap[stageCode]; exists {
		request.StageTiming = timings
	}

	groups, err :=
		matchStageComponentsForRuntime(
			ctx,
			request,
		)

	if err != nil ||
		len(groups) == 0 {
		return ""
	}

	var builder strings.Builder

	builder.WriteString(
		"=== 本阶段参考资料(系统自动匹配)===\n",
	)

	builder.WriteString(
		"以下是根据教案教育域、学科和学习层级自动匹配的教学参考组件，请在本阶段工作中适当参考:\n",
	)

	for _, group := range groups {
		if group == nil {
			continue
		}

		builder.WriteString(
			"\n【" +
				group.LibraryName +
				"】\n",
		)

		for _, component := range group.Components {
			if component == nil {
				continue
			}

			if component.ComponentIndex != "" {
				builder.WriteString(
					utils.FormatIndexForPrompt(
						component.ComponentIndex,
						component.DisplayLabel,
					),
				)
				builder.WriteString("\n")
				continue
			}

			builder.WriteString(
				"▸ " +
					component.DisplayLabel +
					"\n",
			)

			if component.DesignLogic != "" {
				builder.WriteString(
					"  设计逻辑:" +
						component.DesignLogic +
						"\n",
				)
			}
		}
	}

	return builder.String()
}

// RerankedStageComponentsForRuntime 域感知知识型技能精排。
//
// 候选池从源头只包含教案同域或common，精排不会扩大教育域范围。
func RerankedStageComponentsForRuntime(
	ctx context.Context,
	componentTypesJSON string,
	subject string,
	grade string,
	stageCode string,
	recentUserText string,
) string {
	stageTypes, ok :=
		parseRuntimeStageTypes(
			componentTypesJSON,
		)

	if !ok || len(stageTypes) == 0 {
		return ""
	}

	request := &models.MatchComponentsRequest{
		Subject: strings.TrimSpace(
			subject,
		),
		GradeRange: utils.NormalizeGradeToNumber(
			grade,
		),
		LibraryTypes: stageTypes,
		Limit:        skillRouterCandidateLimitPerType,
	}

	if timings, exists :=
		stageTimingMap[stageCode]; exists {
		request.StageTiming = timings
	}

	groups, err :=
		matchStageComponentsForRuntime(
			ctx,
			request,
		)

	if err != nil ||
		len(groups) == 0 {
		return ""
	}

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
		return ""
	}

	keywords := extractRerankKeywords(
		recentUserText,
	)

	if skillRouterRerankEnabled &&
		len(keywords) > 0 {
		for _, candidate := range candidates {
			candidate.score =
				scoreCandidateLexical(
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

	return formatRerankedComponents(
		candidates[:topN],
	)
}

// sortRuntimeCandidates 按精排得分和原始顺序稳定排序。
func sortRuntimeCandidates(
	candidates []*rerankCandidate,
) {
	for left := 1; left < len(candidates); left++ {
		current := candidates[left]
		right := left - 1

		for right >= 0 {
			previous := candidates[right]

			scoreHigher :=
				current.score > previous.score

			sameScoreEarlier :=
				current.score == previous.score &&
					current.originalRank <
						previous.originalRank

			shouldMove :=
				scoreHigher ||
					sameScoreEarlier
			if !shouldMove {
				break
			}

			candidates[right+1] =
				previous

			right--
		}

		candidates[right+1] = current
	}
}
