package services

// unit_plan_course_outline_prompt.go
//
// 单元方案课程大纲运行时提示词硬闸。
//
// 单元方案表没有独立education_domain列，因此每一轮都从正式归属解析：
//   - system维持K12全局资源语义；
//   - school读取学校组织教育域；
//   - group读取教研组所属学校教育域。
//
// 解析完成后：
//   - 只查询同教育域课程大纲；
//   - K12允许空出版社或具名出版社；
//   - vocational/adult只允许空出版社普通大纲；
//   - 非K12具名出版社、非法归属和数据库故障均向上传递。

import (
	"context"
	"fmt"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// buildSystemPromptForUnitPlan
// 按单元方案正式归属构建系统提示词。
func (s *UnitPlanService) buildSystemPromptForUnitPlan(
	ctx context.Context,
	plan *models.UnitPlan,
) (string, bool, error) {
	base := ""

	prompt, promptErr :=
		repository.GetCurrentPromptByKey(
			unitDesignPromptKey,
		)
	if promptErr == nil && prompt != nil {
		base = prompt.Content
	} else {
		unitPlanLog.Warn(
			"取单元设计提示词失败，使用空骨架",
			"error",
			promptErr,
		)
	}

	if plan == nil {
		return base,
			false,
			ErrUnitPlanFieldRequired
	}

	if plan.CourseOutlinePublisher == nil {
		return base, false, nil
	}

	educationDomain, err :=
		resolveCourseOutlineResourceDomain(
			ctx,
			plan.Scope,
			plan.ScopeTargetID,
		)
	if err != nil {
		return "",
			false,
			err
	}

	publisher, err :=
		normalizeCourseOutlinePublisherForDomain(
			educationDomain,
			*plan.CourseOutlinePublisher,
		)
	if err != nil {
		return "",
			false,
			err
	}

	candidates, err :=
		repository.
			ListActiveOutlinesBySubjectAndEducationDomain(
				ctx,
				strings.TrimSpace(
					plan.Subject,
				),
				educationDomain,
			)
	if err != nil {
		return "",
			false,
			fmt.Errorf(
				"%w: 查询单元方案课程大纲失败: %v",
				ErrOutlineEducationDomainResolveFailed,
				err,
			)
	}

	hits := MatchOutlinesByPublisher(
		plan.Grade,
		publisher,
		candidates,
	)
	if len(hits) == 0 {
		unitPlanLog.Info(
			"当前教育域下没有匹配课程大纲，跳过注入",
			"unit_plan_id",
			plan.ID,
			"subject",
			plan.Subject,
			"grade",
			plan.Grade,
			"education_domain",
			educationDomain,
			"has_named_publisher",
			publisher != "",
		)

		return base, false, nil
	}

	outlineContext :=
		BuildCourseOutlinesContext(
			hits,
		)
	if strings.TrimSpace(
		outlineContext,
	) == "" {
		return base, false, nil
	}

	base += outlineContext

	unitPlanLog.Info(
		"单元方案已通过教育域硬闸注入课程大纲",
		"unit_plan_id",
		plan.ID,
		"subject",
		plan.Subject,
		"grade",
		plan.Grade,
		"education_domain",
		educationDomain,
		"has_named_publisher",
		publisher != "",
		"count",
		len(hits),
	)

	return base, true, nil
}
