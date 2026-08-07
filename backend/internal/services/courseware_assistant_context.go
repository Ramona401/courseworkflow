package services

// courseware_assistant_context.go
//
// 本文件从正式课件、稳定页面和来源教案中装配教学智能体上下文。
//
// 装配流程：
//   1. 使用可信Actor重新加载作者自己的正式课件；
//   2. 收敛到课件历史education_domain；
//   3. 重新读取课件全部正式页面并按页码稳定排序；
//   4. 使用稳定page_id定位当前页；
//   5. 静态提取当前页可见文字和互动证据；
//   6. 读取前后相邻页面的标题、目的和概要；
//   7. 对来源教案执行教育域校验并提取当前页相关片段；
//   8. 生成完整快照JSON、稳定内容哈希和教师端安全预览。
//
// 安全边界：
//   - 只有课件作者本人可以装配包含来源教案片段的上下文；
//   - 不执行页面JavaScript；
//   - 不把完整HTML写入上下文快照；
//   - 不把完整教案写入上下文快照或教师端预览；
//   - 不调用AI、不产生积分消费；
//   - 不写数据库。

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"tedna/internal/models"
	"tedna/internal/repository"
)

var (
	// ErrCoursewareAssistantContextPageNotFound 表示稳定页面不存在。
	ErrCoursewareAssistantContextPageNotFound = errors.New(
		"课件教学智能体上下文目标页面不存在",
	)

	// ErrCoursewareAssistantContextNoPages 表示课件没有正式页面。
	ErrCoursewareAssistantContextNoPages = errors.New(
		"课件没有可构建教学智能体上下文的页面",
	)

	// ErrCoursewareAssistantContextLessonMissing 表示教案来源记录缺失。
	ErrCoursewareAssistantContextLessonMissing = errors.New(
		"课件来源教案不存在",
	)

	// ErrCoursewareAssistantContextLessonDomainMismatch 表示课件和教案教育域不一致。
	ErrCoursewareAssistantContextLessonDomainMismatch = errors.New(
		"课件与来源教案的教育域不一致",
	)

	// ErrCoursewareAssistantContextBuildFailed 表示快照无法完成。
	ErrCoursewareAssistantContextBuildFailed = errors.New(
		"课件教学智能体上下文构建失败",
	)
)

// CoursewareAssistantContextBuildResult 是后续方案生成和发布服务使用的完整结果。
//
// SnapshotJSON可直接作为不可变版本context_snapshot_json的输入。
// Preview是教师端安全响应，不包含完整HTML或完整教案。
type CoursewareAssistantContextBuildResult struct {
	Snapshot     models.AssistantDeploymentContextSnapshot
	Preview      models.CoursewareAssistantContextPreview
	SnapshotJSON string
	SnapshotHash string
	PageHTMLHash string
}

// CoursewareAssistantContextService 是确定性上下文装配服务。
type CoursewareAssistantContextService struct {
	coursewareService *CoursewareService
}

// NewCoursewareAssistantContextService 创建默认上下文服务。
func NewCoursewareAssistantContextService() *CoursewareAssistantContextService {
	return &CoursewareAssistantContextService{
		coursewareService: NewCoursewareService(),
	}
}

// NewCoursewareAssistantContextServiceWithDependencies 创建可注入依赖的上下文服务。
func NewCoursewareAssistantContextServiceWithDependencies(
	coursewareService *CoursewareService,
) *CoursewareAssistantContextService {
	return &CoursewareAssistantContextService{
		coursewareService: coursewareService,
	}
}

// coursewareAssistantCoursewareService 返回可用课件服务。
func (s *CoursewareAssistantContextService) coursewareAssistantCoursewareService() *CoursewareService {
	if s != nil &&
		s.coursewareService != nil {
		return s.coursewareService
	}

	return NewCoursewareService()
}

// BuildCoursewareAssistantContext 构建完整上下文快照和安全预览。
func (s *CoursewareAssistantContextService) BuildCoursewareAssistantContext(
	ctx context.Context,
	coursewareID string,
	pageID string,
	actor *CoursewareActorContext,
	config models.CoursewareAssistantContextConfig,
) (
	*CoursewareAssistantContextBuildResult,
	error,
) {
	coursewareID =
		strings.TrimSpace(coursewareID)
	pageID =
		strings.TrimSpace(pageID)

	if coursewareID == "" ||
		pageID == "" {
		return nil,
			ErrCoursewareAssistantInvalidRequest
	}

	if err := validateCoursewareAssistantContextConfig(
		&config,
	); err != nil {
		return nil, err
	}

	courseware,
		_,
		err :=
		s.coursewareAssistantCoursewareService().
			LoadCoursewareForOwnerRuntime(
				ctx,
				coursewareID,
				actor,
			)
	if err != nil {
		return nil,
			mapCoursewareAssistantWriteAccessError(
				err,
			)
	}

	pages, err :=
		loadSortedCoursewareAssistantPages(
			ctx,
			courseware.ID,
		)
	if err != nil {
		return nil, err
	}

	currentIndex :=
		findCoursewareAssistantPageIndex(
			pages,
			pageID,
		)
	if currentIndex < 0 {
		return nil,
			ErrCoursewareAssistantContextPageNotFound
	}

	currentPage :=
		pages[currentIndex]

	currentSnapshot,
		pageHTMLHash :=
		buildCoursewareAssistantCurrentPageSnapshot(
			currentPage,
			config,
		)

	snapshot :=
		models.AssistantDeploymentContextSnapshot{
			Version:     models.AssistantDeploymentSnapshotVersion,
			CurrentPage: currentSnapshot,
		}

	if config.IncludePreviousPageSummary &&
		currentIndex > 0 {
		snapshot.PreviousPage =
			buildCoursewareAssistantAdjacentPageSnapshot(
				pages[currentIndex-1],
			)
	}

	if config.IncludeNextPageSummary &&
		currentIndex+1 < len(pages) {
		snapshot.NextPage =
			buildCoursewareAssistantAdjacentPageSnapshot(
				pages[currentIndex+1],
			)
	}

	if config.IncludeLessonPlanExcerpt {
		lessonSnapshot, lessonErr :=
			loadCoursewareAssistantLessonContext(
				ctx,
				courseware,
				currentPage,
				config.MaxLessonPlanExcerptChars,
			)
		if lessonErr != nil {
			return nil, lessonErr
		}

		snapshot.LessonPlan =
			lessonSnapshot
	}

	generatedAt :=
		time.Now().UTC()
	snapshot.GeneratedAt =
		&generatedAt

	snapshotJSON, err :=
		marshalCoursewareAssistantContextSnapshot(
			snapshot,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"%w: %v",
				ErrCoursewareAssistantContextBuildFailed,
				err,
			)
	}

	snapshotHash, err :=
		hashCoursewareAssistantContextSnapshot(
			snapshot,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"%w: %v",
				ErrCoursewareAssistantContextBuildFailed,
				err,
			)
	}

	if pageHTMLHash == "" {
		pageHTMLHash =
			coursewareAssistantPageHTMLHash(
				currentPage.HTMLContent,
			)
	}

	preview :=
		buildCoursewareAssistantContextPreview(
			snapshot,
			config,
		)

	return &CoursewareAssistantContextBuildResult{
		Snapshot:     snapshot,
		Preview:      preview,
		SnapshotJSON: snapshotJSON,
		SnapshotHash: snapshotHash,
		PageHTMLHash: pageHTMLHash,
	}, nil
}

// BuildCoursewareAssistantContextPreview 只返回教师端安全预览。
func (s *CoursewareAssistantContextService) BuildCoursewareAssistantContextPreview(
	ctx context.Context,
	coursewareID string,
	pageID string,
	actor *CoursewareActorContext,
	config models.CoursewareAssistantContextConfig,
) (
	*models.CoursewareAssistantContextPreview,
	error,
) {
	result, err :=
		s.BuildCoursewareAssistantContext(
			ctx,
			coursewareID,
			pageID,
			actor,
			config,
		)
	if err != nil {
		return nil, err
	}

	return &result.Preview, nil
}

// loadSortedCoursewareAssistantPages 读取并稳定排序正式页面。
func loadSortedCoursewareAssistantPages(
	ctx context.Context,
	coursewareID string,
) (
	[]*models.CoursewarePage,
	error,
) {
	pages, err :=
		repository.ListCoursewarePages(
			ctx,
			strings.TrimSpace(coursewareID),
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"读取课件教学智能体上下文页面失败: %w",
				err,
			)
	}

	validPages := make(
		[]*models.CoursewarePage,
		0,
		len(pages),
	)

	for _, page := range pages {
		if page == nil ||
			strings.TrimSpace(page.ID) == "" {
			continue
		}

		validPages = append(
			validPages,
			page,
		)
	}

	if len(validPages) == 0 {
		return nil,
			ErrCoursewareAssistantContextNoPages
	}

	sort.SliceStable(
		validPages,
		func(i int, j int) bool {
			if validPages[i].PageNumber ==
				validPages[j].PageNumber {
				return validPages[i].ID <
					validPages[j].ID
			}

			return validPages[i].PageNumber <
				validPages[j].PageNumber
		},
	)

	return validPages, nil
}

// findCoursewareAssistantPageIndex 按稳定页面ID定位页面。
func findCoursewareAssistantPageIndex(
	pages []*models.CoursewarePage,
	pageID string,
) int {
	pageID =
		strings.TrimSpace(pageID)

	for index, page := range pages {
		if page != nil &&
			strings.TrimSpace(page.ID) ==
				pageID {
			return index
		}
	}

	return -1
}

// loadCoursewareAssistantLessonContext 读取并校验来源教案相关片段。
func loadCoursewareAssistantLessonContext(
	ctx context.Context,
	courseware *models.Courseware,
	page *models.CoursewarePage,
	maxRunes int,
) (
	*models.AssistantDeploymentLessonPlanSnapshot,
	error,
) {
	if courseware == nil ||
		page == nil {
		return nil,
			ErrCoursewareAssistantContextBuildFailed
	}

	if courseware.SourceType !=
		models.CWSourceLessonPlan {
		return nil, nil
	}

	if courseware.LessonPlanID == nil ||
		strings.TrimSpace(
			*courseware.LessonPlanID,
		) == "" {
		return nil,
			ErrCoursewareAssistantContextLessonMissing
	}

	lessonPlan, err :=
		repository.GetLessonPlanByID(
			ctx,
			strings.TrimSpace(
				*courseware.LessonPlanID,
			),
		)
	if err != nil ||
		lessonPlan == nil {
		return nil,
			ErrCoursewareAssistantContextLessonMissing
	}

	coursewareDomain :=
		strings.ToLower(
			strings.TrimSpace(
				courseware.EducationDomain,
			),
		)
	lessonDomain :=
		strings.ToLower(
			strings.TrimSpace(
				lessonPlan.EducationDomain,
			),
		)

	if !models.IsTeachingEducationDomain(
		coursewareDomain,
	) ||
		!models.IsTeachingEducationDomain(
			lessonDomain,
		) ||
		coursewareDomain != lessonDomain {
		return nil,
			ErrCoursewareAssistantContextLessonDomainMismatch
	}

	lessonContent :=
		strings.TrimSpace(
			ExtractLessonPlanContentForCW(
				lessonPlan,
			),
		)

	relevantExcerpt :=
		buildCoursewareAssistantLessonExcerpt(
			lessonContent,
			page,
			maxRunes,
		)

	lessonID :=
		strings.TrimSpace(
			lessonPlan.ID,
		)

	return &models.AssistantDeploymentLessonPlanSnapshot{
		LessonPlanID: &lessonID,
		Title: coursewareAssistantTruncateRunes(
			lessonPlan.Title,
			500,
		),
		RelevantExcerpt: relevantExcerpt,
		ExcerptHash: coursewareAssistantSHA256String(
			relevantExcerpt,
		),
	}, nil
}
