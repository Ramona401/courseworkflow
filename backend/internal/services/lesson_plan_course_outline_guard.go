package services

// lesson_plan_course_outline_guard.go — 教案唯一课程大纲ID统一硬闸
//
// 正式规则：
//   1. 开始备课时，唯一ID必须在任何教案INSERT前完成作者可见性与同域校验；
//   2. 自动匹配候选仍由候选API要求具体年级完全相等；
//   3. 教师手动选择时允许大纲年级与教案年级或学段存在交集；
//   4. 手动选择不能放宽教育域、用户可见范围、active状态或学科一致性；
//   5. 运行时只信任lesson_plans.education_domain与唯一大纲快照；
//   6. publisher-only存量教案不再模糊匹配，也不注入任何大纲；
//   7. 运行时最多返回一份大纲，绝不跨出版社、册次或学制兜底。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

var (
	ErrOutlineExactSelectionInvalid = errors.New(
		"教案课程大纲唯一挂载无效，请重新选择",
	)

	ErrOutlineExactSelectionUnavailable = errors.New(
		"已关联的课程大纲当前不可用，请重新选择",
	)

	ErrOutlineExactSelectionForbidden = errors.New(
		"当前已无权读取关联的课程大纲",
	)
)

var (
	lessonPlanCourseOutlineSnapshotReader = repository.GetLessonPlanCourseOutlineSnapshot

	activeCourseOutlineExactReader = repository.GetActiveCourseOutlineByIDAndEducationDomain

	visibleCourseOutlineReader = func(
		ctx context.Context,
		userID string,
		outlineID string,
	) (
		*models.CourseOutline,
		string,
		error,
	) {
		return NewCourseOutlineService().GetOutline(
			ctx,
			userID,
			outlineID,
		)
	}
)

// 旧出版社挂载兼容链暂时保留，直到前后端完成唯一ID切换。
var courseOutlineListActiveByDomain = repository.ListActiveOutlinesBySubjectAndEducationDomain

func resolveLessonPlanCourseOutlineSnapshotDomain(
	lessonPlan *models.LessonPlan,
) (string, error) {
	if lessonPlan == nil {
		return "",
			ErrOutlineEducationDomainRequired
	}

	domain := strings.ToLower(
		strings.TrimSpace(
			lessonPlan.EducationDomain,
		),
	)

	if !models.IsTeachingEducationDomain(
		domain,
	) {
		return "",
			ErrOutlineEducationDomainRequired
	}

	return domain, nil
}

// ValidateStartConversationCourseOutline
// 在教案INSERT前校验唯一课程大纲ID。
//
// 空ID表示明确不挂载；非空ID必须同时满足：
//   - 当前作者仍可见；
//   - 与创建教育域一致；
//   - active；
//   - 学科完全相等；
//   - 年级或学段存在交集；
//   - 学制字段合法。
//
// 自动匹配是否成功不在本函数判断；自动候选API只返回具体年级完全相等候选。
// 本函数同时承担手动选择的最终业务校验，因此允许具体年级选择覆盖自己的学段大纲。
func ValidateStartConversationCourseOutline(
	ctx context.Context,
	educationDomain string,
	authorID string,
	req *models.StartConversationRequest,
) error {
	if req == nil {
		return errors.New(
			"开始备课请求不能为空",
		)
	}

	outlineID := strings.TrimSpace(
		req.CourseOutlineID,
	)

	if outlineID == "" {
		req.CourseOutlineID = ""
		return nil
	}

	domain := strings.ToLower(
		strings.TrimSpace(
			educationDomain,
		),
	)

	if !models.IsTeachingEducationDomain(
		domain,
	) {
		return ErrOutlineEducationDomainRequired
	}

	if strings.TrimSpace(authorID) == "" {
		return ErrOutlineExactSelectionForbidden
	}

	visibleOutline, visibleDomain, err :=
		visibleCourseOutlineReader(
			ctx,
			authorID,
			outlineID,
		)
	if err != nil {
		if errors.Is(
			err,
			repository.ErrCourseOutlineNotFound,
		) {
			return ErrOutlineExactSelectionForbidden
		}

		return fmt.Errorf(
			"%w: 校验课程大纲可见性失败: %v",
			ErrOutlineEducationDomainResolveFailed,
			err,
		)
	}

	if visibleOutline == nil ||
		strings.TrimSpace(
			visibleOutline.ID,
		) != outlineID ||
		strings.ToLower(
			strings.TrimSpace(
				visibleDomain,
			),
		) != domain {
		return ErrOutlineExactSelectionForbidden
	}

	outline, err :=
		activeCourseOutlineExactReader(
			ctx,
			outlineID,
			domain,
		)
	if err != nil {
		if errors.Is(
			err,
			repository.ErrCourseOutlineNotFound,
		) {
			return ErrOutlineExactSelectionUnavailable
		}

		return fmt.Errorf(
			"%w: 读取唯一课程大纲失败: %v",
			ErrOutlineEducationDomainResolveFailed,
			err,
		)
	}

	if outline == nil ||
		strings.TrimSpace(
			outline.ID,
		) != outlineID ||
		strings.TrimSpace(
			outline.Subject,
		) != strings.TrimSpace(
			req.Subject,
		) ||
		!courseOutlineGradesMatch(
			outline.Grade,
			req.Grade,
		) ||
		!models.IsValidCourseOutlineSchoolSystem(
			strings.TrimSpace(
				outline.SchoolSystem,
			),
		) {
		return ErrOutlineExactSelectionInvalid
	}

	req.CourseOutlineID = strings.TrimSpace(
		outline.ID,
	)

	return nil
}

// normalizeLessonPlanCourseOutlineMount
// 旧publisher-only端点的过渡校验。
// 新前端不得再调用该路径。
func normalizeLessonPlanCourseOutlineMount(
	ctx context.Context,
	lessonPlan *models.LessonPlan,
	liveEducationDomain string,
	publisher *string,
) (*string, error) {
	snapshotDomain, err :=
		resolveLessonPlanCourseOutlineSnapshotDomain(
			lessonPlan,
		)
	if err != nil {
		return nil, err
	}

	liveDomain := strings.ToLower(
		strings.TrimSpace(
			liveEducationDomain,
		),
	)

	if !models.IsTeachingEducationDomain(
		liveDomain,
	) {
		return nil,
			ErrOutlineEducationDomainRequired
	}

	if liveDomain != snapshotDomain {
		return nil,
			ErrOutlineEducationDomainMismatch
	}

	if publisher == nil {
		return nil, nil
	}

	normalized, err :=
		normalizeCourseOutlinePublisherForDomain(
			snapshotDomain,
			*publisher,
		)
	if err != nil {
		return nil, err
	}

	if snapshotDomain !=
		models.EducationDomainK12 {
		return &normalized, nil
	}

	candidates, err :=
		courseOutlineListActiveByDomain(
			ctx,
			lessonPlan.Subject,
			snapshotDomain,
		)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: 查询K12课程大纲候选失败: %v",
			ErrOutlineEducationDomainResolveFailed,
			err,
		)
	}

	hits := MatchOutlinesByPublisher(
		lessonPlan.Grade,
		normalized,
		candidates,
	)

	if len(hits) == 0 {
		return nil,
			ErrOutlinePublisherUnavailable
	}

	return &normalized, nil
}

// validateExactLessonPlanCourseOutlineSnapshot
// 复核数据库固化快照与当前正式大纲。
//
// grade不是独立快照字段，但当前教案grade仍必须与正式大纲grade或学段相交；
// 这使教师能够绑定学段大纲，同时防止教案后续被改成完全无关的年级。
func validateExactLessonPlanCourseOutlineSnapshot(
	lessonPlan *models.LessonPlan,
	snapshot *models.LessonPlanCourseOutlineSnapshot,
	outline *models.CourseOutline,
) error {
	if lessonPlan == nil ||
		snapshot == nil ||
		outline == nil ||
		snapshot.CourseOutlineID == nil ||
		strings.TrimSpace(
			*snapshot.CourseOutlineID,
		) == "" ||
		snapshot.CourseOutlinePublisher == nil ||
		snapshot.CourseOutlineVolume == nil ||
		strings.TrimSpace(
			*snapshot.CourseOutlineVolume,
		) == "" ||
		snapshot.SchoolSystem == nil ||
		!models.IsValidCourseOutlineSchoolSystem(
			strings.TrimSpace(
				*snapshot.SchoolSystem,
			),
		) {
		return ErrOutlineExactSelectionInvalid
	}

	if strings.TrimSpace(
		outline.ID,
	) != strings.TrimSpace(
		*snapshot.CourseOutlineID,
	) ||
		strings.TrimSpace(
			outline.Subject,
		) != strings.TrimSpace(
			lessonPlan.Subject,
		) ||
		!courseOutlineGradesMatch(
			outline.Grade,
			lessonPlan.Grade,
		) ||
		strings.TrimSpace(
			outline.Publisher,
		) != strings.TrimSpace(
			*snapshot.CourseOutlinePublisher,
		) ||
		strings.TrimSpace(
			outline.Volume,
		) != strings.TrimSpace(
			*snapshot.CourseOutlineVolume,
		) ||
		strings.TrimSpace(
			outline.SchoolSystem,
		) != strings.TrimSpace(
			*snapshot.SchoolSystem,
		) {
		return ErrOutlineExactSelectionInvalid
	}

	return nil
}

func resolveLessonPlanExactCourseOutline(
	ctx context.Context,
	lessonPlan *models.LessonPlan,
) (*models.CourseOutline, error) {
	snapshotDomain, err :=
		resolveLessonPlanCourseOutlineSnapshotDomain(
			lessonPlan,
		)
	if err != nil {
		return nil, err
	}

	if lessonPlan == nil ||
		strings.TrimSpace(
			lessonPlan.ID,
		) == "" {
		return nil,
			ErrOutlineExactSelectionInvalid
	}

	snapshot, err :=
		lessonPlanCourseOutlineSnapshotReader(
			ctx,
			lessonPlan.ID,
		)
	if err != nil {
		if errors.Is(
			err,
			repository.ErrLessonPlanNotFound,
		) {
			return nil,
				ErrOutlineExactSelectionUnavailable
		}

		return nil, fmt.Errorf(
			"%w: 读取教案课程大纲快照失败: %v",
			ErrOutlineEducationDomainResolveFailed,
			err,
		)
	}

	if snapshot == nil ||
		snapshot.CourseOutlineID == nil ||
		strings.TrimSpace(
			*snapshot.CourseOutlineID,
		) == "" {
		return nil, nil
	}

	visibleOutline, visibleDomain, err :=
		visibleCourseOutlineReader(
			ctx,
			lessonPlan.AuthorID,
			*snapshot.CourseOutlineID,
		)
	if err != nil {
		if errors.Is(
			err,
			repository.ErrCourseOutlineNotFound,
		) {
			return nil,
				ErrOutlineExactSelectionForbidden
		}

		return nil, fmt.Errorf(
			"%w: 复核课程大纲读取权限失败: %v",
			ErrOutlineEducationDomainResolveFailed,
			err,
		)
	}

	if visibleOutline == nil ||
		strings.TrimSpace(
			visibleOutline.ID,
		) != strings.TrimSpace(
			*snapshot.CourseOutlineID,
		) ||
		strings.ToLower(
			strings.TrimSpace(
				visibleDomain,
			),
		) != snapshotDomain {
		return nil,
			ErrOutlineExactSelectionForbidden
	}

	outline, err :=
		activeCourseOutlineExactReader(
			ctx,
			*snapshot.CourseOutlineID,
			snapshotDomain,
		)
	if err != nil {
		if errors.Is(
			err,
			repository.ErrCourseOutlineNotFound,
		) {
			return nil,
				ErrOutlineExactSelectionUnavailable
		}

		return nil, fmt.Errorf(
			"%w: 读取唯一课程大纲失败: %v",
			ErrOutlineEducationDomainResolveFailed,
			err,
		)
	}

	if err :=
		validateExactLessonPlanCourseOutlineSnapshot(
			lessonPlan,
			snapshot,
			outline,
		); err != nil {
		return nil, err
	}

	return outline, nil
}

// ResolveLessonPlanCourseOutlines
// 返回零份或唯一一份已绑定课程大纲。
func ResolveLessonPlanCourseOutlines(
	ctx context.Context,
	lessonPlan *models.LessonPlan,
) ([]*models.CourseOutline, error) {
	outline, err :=
		resolveLessonPlanExactCourseOutline(
			ctx,
			lessonPlan,
		)
	if err != nil {
		return nil, err
	}

	if outline == nil {
		return []*models.CourseOutline{},
			nil
	}

	return []*models.CourseOutline{
		outline,
	}, nil
}

// BuildLessonPlanCourseOutlineContext
// 构建教案本轮所需的课程层级上下文。
//
// 教案对话链存在lessonPlanTurnContextPlan时：
//   - 普通备课、正式教案、评审和修订只读取active短版知识脉络；
//   - 该路径不会读取课程大纲正文，避免每轮重复加载全文；
//   - analyze尚未完成确认时不注入，也不凭课题提前提取；
//   - analyze之后缺少active快照时fail-closed，要求回到教学分析；
//   - 只有老师明确询问课程大纲原文或版本要求时，才读取原始大纲全文。
//
// 不存在单轮计划的其它调用方（如课程大纲管理或课件独立审核）
// 保持原有唯一大纲全文读取行为，避免影响非教案对话场景。
func BuildLessonPlanCourseOutlineContext(
	ctx context.Context,
	lessonPlan *models.LessonPlan,
) (
	string,
	[]*models.CourseOutline,
	error,
) {
	turnPlan :=
		lessonPlanTurnContextPlanFromContext(
			ctx,
		)

	if turnPlan != nil &&
		turnPlan.UseKnowledgeLineage &&
		!turnPlan.UseRawCourseOutline {
		if lessonPlan == nil ||
			strings.TrimSpace(
				lessonPlan.ID,
			) == "" {
			return "", nil,
				ErrOutlineExactSelectionInvalid
		}

		lineage, lineageErr :=
			repository.GetActiveLessonPlanKnowledgeLineage(
				ctx,
				lessonPlan.ID,
			)
		if lineageErr != nil {
			return "", nil,
				fmt.Errorf(
					"读取教案active知识脉络失败: %w",
					lineageErr,
				)
		}

		if lineage != nil &&
			lineage.IsActiveUsable() &&
			strings.TrimSpace(
				lineage.ContextText,
			) != "" {
			return "\n\n" +
					strings.TrimSpace(
						lineage.ContextText,
					) +
					"\n",
				[]*models.CourseOutline{},
				nil
		}

		// 教学分析阶段尚未完成确认时，不提前读取或注入大纲全文。
		if strings.TrimSpace(
			lessonPlan.CurrentStage,
		) == "analyze" {
			turnPlan.UseKnowledgeLineage = false

			if !turnPlan.UseRawCourseOutline {
				turnPlan.UseCourseOutline = false
			}

			// 本轮若没有其它正式证据，也不必启动多证据Harness。
			if !turnPlan.UseTextbook &&
				!turnPlan.UseRefMaterial &&
				!turnPlan.UseUnitPlan &&
				!turnPlan.UseRawCourseOutline &&
				!turnPlan.UseClassProfile {
				turnPlan.BlockingEvidenceHarness = false
			}

			return "",
				[]*models.CourseOutline{},
				nil
		}

		return "", nil,
			ErrLessonPlanKnowledgeLineageAnalyzeRequired
	}

	// 原始大纲查询或非教案对话调用方继续走唯一精确大纲硬闸。
	hits, err :=
		ResolveLessonPlanCourseOutlines(
			ctx,
			lessonPlan,
		)
	if err != nil {
		return "", nil, err
	}

	if len(hits) == 0 {
		return "", hits, nil
	}

	if len(hits) != 1 ||
		hits[0] == nil {
		return "", nil,
			ErrOutlineExactSelectionInvalid
	}

	return BuildCourseOutlineContext(
			hits[0],
		),
		hits,
		nil
}
