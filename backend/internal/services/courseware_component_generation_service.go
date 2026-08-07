package services

// courseware_component_generation_service.go — 课件组件正式生成运行时。
//
// 所有页面生成路径最终都通过matchComponentsForPage进入本文件：
//   - 封面预览；
//   - 批量生成；
//   - 全自动装配；
//   - 单页重新生成。
//
// 运行时规则：
//   1. 不信任调用方传入的subject、grade或教育域；
//   2. 根据page.courseware_id重新读取正式课件记录；
//   3. 只接受k12、vocational、adult具体课件快照域；
//   4. 只匹配课件同域或common组件；
//   5. mixed、common、空值、非法域和读取失败均fail-closed；
//   6. 候选仍执行既有互动代码兼容过滤，最终最多注入2个组件。
//
// 每页重新读取课件快照是有意设计：
// 异步任务启动后即使内存Actor或旧对象长期存活，组件匹配仍以数据库中
// 当前正式coursewares.education_domain快照为准，不发生异步漂移。

import (
	"context"
	"errors"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

var ErrCWComponentRuntimeCoursewareRequired = errors.New(
	"课件组件运行时缺少有效课件快照",
)

// resolveCoursewareComponentRuntimeDomain 解析具体课件的组件运行域。
func resolveCoursewareComponentRuntimeDomain(
	courseware *models.Courseware,
) (string, error) {
	if courseware == nil ||
		strings.TrimSpace(courseware.ID) == "" {
		return "",
			ErrCWComponentRuntimeCoursewareRequired
	}

	domain := strings.ToLower(
		strings.TrimSpace(
			courseware.EducationDomain,
		),
	)

	if !models.IsTeachingEducationDomain(
		domain,
	) {
		return "",
			ErrCWComponentEducationDomainInvalid
	}

	return domain, nil
}

// unwrapMatchedCWComponentResources 把域感知仓储结果转换为既有生成提示词结构。
//
// 教育域已经在仓储层完成过滤；生成提示词不需要重复暴露该字段。
func unwrapMatchedCWComponentResources(
	resources []*models.MatchedCWComponentResource,
) []*models.MatchedCWComponent {
	result := make(
		[]*models.MatchedCWComponent,
		0,
		len(resources),
	)

	for _, resource := range resources {
		if resource == nil ||
			resource.MatchedCWComponent == nil {
			continue
		}

		result = append(
			result,
			resource.MatchedCWComponent,
		)
	}

	return result
}

// matchCoursewareComponentsForPageSnapshot 为页面匹配同域或common组件。
func (s *CoursewareGenService) matchCoursewareComponentsForPageSnapshot(
	ctx context.Context,
	page *models.CoursewarePage,
) []*models.MatchedCWComponent {
	if page == nil ||
		strings.TrimSpace(
			page.CoursewareID,
		) == "" {
		cwGenLog.Warn(
			"组件匹配缺少课件页面归属，已fail-closed",
		)
		return nil
	}

	// 每次匹配都重新读取正式课件。
	//
	// 这一步既取得权威subject/grade，也取得创建时固化的教育域快照，
	// 不使用调用方长期持有的课件对象或请求参数。
	courseware, err :=
		repository.GetCoursewareByID(
			ctx,
			page.CoursewareID,
		)
	if err != nil {
		cwGenLog.Warn(
			"组件匹配重新读取课件失败，已fail-closed",
			"courseware_id", page.CoursewareID,
			"page_num", page.PageNumber,
			"error", err,
		)
		return nil
	}

	currentDomain, err :=
		resolveCoursewareComponentRuntimeDomain(
			courseware,
		)
	if err != nil {
		cwGenLog.Error(
			"课件组件运行教育域异常，已fail-closed",
			"courseware_id", page.CoursewareID,
			"page_num", page.PageNumber,
			"education_domain",
			courseware.EducationDomain,
			"error", err,
		)
		return nil
	}

	request := &models.MatchCWComponentsRequest{
		SubjectScope: strings.TrimSpace(
			courseware.Subject,
		),
		GradeScope: strings.TrimSpace(
			courseware.Grade,
		),
		InteractionLevel:
			cwInteractionLevelForPlan(
				page.InteractionType,
				page.IdxInteractionLevel,
				page.EstimatedComplexity,
			),
		VisualFormat:
			cwVisualFormatForMatch(page),

		// 先放大候选池，再按实际互动代码结构过滤，最终只注入Top 2。
		Limit: 8,
	}

	resources, err :=
		repository.
			MatchCWComponentsForEducationDomain(
				ctx,
				request,
				currentDomain,
			)
	if err != nil {
		cwGenLog.Warn(
			"按课件快照域匹配组件失败",
			"courseware_id", page.CoursewareID,
			"page_num", page.PageNumber,
			"education_domain", currentDomain,
			"interaction_type",
			page.InteractionType,
			"visual_format",
			page.VisualFormat,
			"error", err,
		)
		return nil
	}

	matched :=
		unwrapMatchedCWComponentResources(
			resources,
		)

	filtered :=
		filterCWComponentsForInteraction(
			matched,
			page.InteractionType,
		)

	if len(filtered) == 0 {
		interactionType :=
			normalizeCWInteractionType(
				page.InteractionType,
			)

		if len(matched) > 0 &&
			interactionType != "" &&
			interactionType != "static" {
			cwGenLog.Info(
				"同域候选组件均与方案互动类型冲突，本页不注入组件",
				"courseware_id",
				page.CoursewareID,
				"page_num",
				page.PageNumber,
				"education_domain",
				currentDomain,
				"interaction_type",
				page.InteractionType,
				"candidate_count",
				len(matched),
			)
		}

		return nil
	}

	if len(filtered) > 2 {
		filtered = filtered[:2]
	}

	return filtered
}
