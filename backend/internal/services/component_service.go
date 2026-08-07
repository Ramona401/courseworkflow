package services

// component_service.go — 组件库共享服务底座。
//
// 本文件只保留仍被正式运行链使用的共享能力：
//   - 组件业务错误常量；
//   - ComponentService依赖与构造函数；
//   - 教师画像到组件匹配标签的转换；
//   - 组件使用、选中和质量分统计。
//
// 组件管理、匹配和萃取分别位于：
//   - component_management_service.go
//   - component_match_domain_service.go
//   - component_extraction_creation_service.go
//   - component_extraction_review_service.go
//
// 所有旧无教育域CRUD和匹配兼容方法已经删除，避免未来新调用方
// 再次绕过可信Actor或教案education_domain快照。

import (
	"context"
	"errors"

	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== 错误常量 ====================

var (
	ErrComponentLibTypeRequired = errors.New(
		"组件库类型不能为空",
	)

	ErrComponentLibTypeInvalid = errors.New(
		"无效的组件库类型",
	)

	ErrComponentLabelRequired = errors.New(
		"组件展示标签不能为空",
	)

	ErrComponentNotFound = errors.New(
		"组件不存在",
	)

	ErrComponentReviewInvalid = errors.New(
		"审核决策无效，可选值：approved/rejected",
	)

	ErrExtractionNotFound = errors.New(
		"萃取记录不存在",
	)

	ErrExtractionDecisionInvalid = errors.New(
		"萃取决策无效，可选值：confirmed/rejected",
	)
)

// ComponentService 组件库共享服务。
//
// cfg目前由自动萃取AI调用使用；管理与匹配方法不再依赖旧无域Repository。
type ComponentService struct {
	cfg interface {
		GetAESKey() string
	}
}

var compLog = logger.WithModule(
	"component",
)

// NewComponentService 创建组件服务。
func NewComponentService(
	cfg interface {
		GetAESKey() string
	},
) *ComponentService {
	return &ComponentService{
		cfg: cfg,
	}
}

// buildProfileTags 从教学画像生成匹配标签。
//
// 本函数由域感知画像推荐链复用。标签只参与同域候选集排序，
// 不得改变教育域过滤结果。
func buildProfileTags(
	profile *models.TeachingProfile,
) []string {
	if profile == nil {
		return nil
	}

	tags := make([]string, 0)

	if profile.TeachingStyle != "" {
		tags = append(
			tags,
			"style:"+profile.TeachingStyle,
		)
	}

	if profile.AICollaboration != "" {
		tags = append(
			tags,
			"collab:"+profile.AICollaboration,
		)
	}

	for index, priority := range profile.Priorities {
		if index >= 4 {
			break
		}

		tags = append(
			tags,
			"priority:"+priority,
		)
	}

	return tags
}

// ==================== 组件统计 ====================

// RecordComponentUsage 记录组件使用次数。
func (s *ComponentService) RecordComponentUsage(
	ctx context.Context,
	componentID string,
) error {
	return repository.IncrementComponentUsage(
		ctx,
		componentID,
	)
}

// RecordComponentSelect 记录组件被选中并异步刷新质量分。
func (s *ComponentService) RecordComponentSelect(
	ctx context.Context,
	componentID string,
) error {
	if err := repository.IncrementComponentSelect(
		ctx,
		componentID,
	); err != nil {
		return err
	}

	go func() {
		backgroundContext := context.Background()

		if err := s.RefreshQualityScore(
			backgroundContext,
			componentID,
		); err != nil {
			compLog.Warn(
				"异步刷新质量分失败",
				"component_id",
				componentID,
				"error",
				err,
			)
		}
	}()

	return nil
}

// RefreshQualityScore 刷新组件质量分。
func (s *ComponentService) RefreshQualityScore(
	ctx context.Context,
	componentID string,
) error {
	averageScore, err :=
		repository.GetComponentLinkedPlanAvgScore(
			ctx,
			componentID,
		)

	if err != nil {
		averageScore = 0
	}

	return repository.UpdateComponentQualityScore(
		ctx,
		componentID,
		averageScore,
	)
}
