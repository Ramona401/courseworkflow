package services

// courseware_annotation_service.go — 课件页级批注可信Actor治理
//
// 权限语义：
//
//   - 列表：合法课件查看者；
//   - 创建：课件作者或进行中的合法集体备课参与者；
//   - 解决、删除：课件作者或批注创建者，且操作者当前仍有课件查看权；
//   - admin不因平台角色自动进入普通教研批注写通道。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

var (
	ErrCoursewareAnnotationNotFound = errors.New(
		"批注不存在",
	)
	ErrCoursewareAnnotationPageNotFound = errors.New(
		"课件页面不存在",
	)
	ErrCoursewareAnnotationInputInvalid = errors.New(
		"批注参数无效",
	)
	ErrCoursewareAnnotationManageDenied = errors.New(
		"无权操作此批注",
	)
	ErrCoursewareAnnotationMutationConflict = errors.New(
		"批注已发生变化，请刷新后重试",
	)
)

var cwAnnotationLog = cwServiceLog

// coursewareAnnotationManagePolicyAllows 是不访问数据库的批注管理纯策略。
//
// 不自动放行admin。只有课件作者或批注创建者本人可以解决、重开或删除批注。
func coursewareAnnotationManagePolicyAllows(
	courseware *models.Courseware,
	annotation *models.CoursewareAnnotation,
	actor *CoursewareActorContext,
) bool {
	if courseware == nil ||
		annotation == nil ||
		actor == nil ||
		strings.TrimSpace(actor.UserID) == "" {
		return false
	}

	if annotation.CoursewareID !=
		courseware.ID {
		return false
	}

	return courseware.UserID ==
		actor.UserID ||
		annotation.ReviewerID ==
			actor.UserID
}

// coursewareAnnotationPageIDEqual 比较可空稳定页面ID。
func coursewareAnnotationPageIDEqual(
	before *string,
	after *string,
) bool {
	if before == nil ||
		after == nil {
		return before == nil &&
			after == nil
	}

	return *before == *after
}

// coursewareAnnotationRevisionUnchanged 判断批注是否仍为同一数据库版本。
//
// PageNumber是通过稳定PageID动态解析的当前页码，页面重排时可能变化，
// 因此版本一致性比较稳定PageID和创建时页码快照，不比较动态当前页码。
func coursewareAnnotationRevisionUnchanged(
	before *models.CoursewareAnnotation,
	after *models.CoursewareAnnotation,
) bool {
	if before == nil ||
		after == nil {
		return false
	}

	return before.ID == after.ID &&
		before.CoursewareID ==
			after.CoursewareID &&
		coursewareAnnotationPageIDEqual(
			before.PageID,
			after.PageID,
		) &&
		before.PageNumberSnapshot ==
			after.PageNumberSnapshot &&
		before.ReviewerID ==
			after.ReviewerID &&
		before.UpdatedAt.Equal(
			after.UpdatedAt,
		)
}

// LoadCoursewareAnnotationForManage 加载批注、验证课件查看权和批注管理身份。
func (s *CoursewareService) LoadCoursewareAnnotationForManage(
	ctx context.Context,
	annotationID string,
	actor *CoursewareActorContext,
) (
	*models.Courseware,
	*models.CoursewareAnnotation,
	*CoursewareActorContext,
	error,
) {
	annotation, err :=
		repository.GetCWAnnotationByID(
			ctx,
			annotationID,
		)
	if err != nil {
		if errors.Is(
			err,
			repository.ErrCWAnnotationNotFound,
		) {
			return nil,
				nil,
				nil,
				ErrCoursewareAnnotationNotFound
		}

		return nil, nil, nil, err
	}

	courseware, err :=
		s.LoadCoursewareForView(
			ctx,
			annotation.CoursewareID,
			actor,
		)
	if err != nil {
		return nil, nil, nil, err
	}

	if !coursewareAnnotationManagePolicyAllows(
		courseware,
		annotation,
		actor,
	) {
		return nil,
			nil,
			nil,
			ErrCoursewareAnnotationManageDenied
	}

	return courseware,
		annotation,
		scopeAuthorizedCoursewareActor(
			actor,
			courseware,
		),
		nil
}

// CreateCWAnnotation 创建页级批注。
func (s *CoursewareService) CreateCWAnnotation(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	reviewerName string,
	req *models.CreateCWAnnotationRequest,
) (
	*models.CoursewareAnnotation,
	error,
) {
	_, scopedActor, err :=
		s.LoadCoursewareForRefine(
			ctx,
			coursewareID,
			actor,
		)
	if err != nil {
		return nil, err
	}

	if req == nil {
		return nil,
			ErrCoursewareAnnotationInputInvalid
	}

	content := strings.TrimSpace(
		req.Content,
	)
	if content == "" {
		return nil,
			fmt.Errorf(
				"%w: 批注内容不能为空",
				ErrCoursewareAnnotationInputInvalid,
			)
	}
	if req.PageNumber <= 0 {
		return nil,
			fmt.Errorf(
				"%w: 页号无效",
				ErrCoursewareAnnotationInputInvalid,
			)
	}

	page, err :=
		repository.GetCoursewarePageByNumber(
			ctx,
			coursewareID,
			req.PageNumber,
		)
	if err != nil {
		return nil,
			ErrCoursewareAnnotationPageNotFound
	}

	// 正式写入前重新授权并重新绑定同一课件页面。
	_, latestActor, err :=
		s.LoadCoursewareForRefine(
			ctx,
			coursewareID,
			scopedActor,
		)
	if err != nil {
		return nil, err
	}

	latestPage, err :=
		repository.GetCoursewarePageByNumber(
			ctx,
			coursewareID,
			req.PageNumber,
		)
	if err != nil {
		return nil,
			ErrCoursewareAnnotationPageNotFound
	}

	if page.ID != latestPage.ID ||
		latestPage.CoursewareID !=
			coursewareID {
		return nil,
			ErrCoursewareAnnotationMutationConflict
	}

	reviewerName =
		strings.TrimSpace(
			reviewerName,
		)
	if reviewerName == "" {
		reviewerName =
			latestActor.UserID
	}

	annotation := &models.CoursewareAnnotation{
		CoursewareID: coursewareID,
		PageNumber:   req.PageNumber,
		ReviewerID:   latestActor.UserID,
		ReviewerName: reviewerName,
		Content:      content,
	}

	if err :=
		repository.CreateCWAnnotation(
			ctx,
			annotation,
		); err != nil {
		if errors.Is(
			err,
			repository.ErrCWAnnotationPageNotFound,
		) {
			return nil,
				ErrCoursewareAnnotationPageNotFound
		}

		return nil,
			fmt.Errorf(
				"创建批注失败: %w",
				err,
			)
	}

	cwAnnotationLog.Info(
		"课件批注创建",
		"courseware_id", coursewareID,
		"page_id", latestPage.ID,
		"page_number", annotation.PageNumber,
		"page_number_snapshot",
		annotation.PageNumberSnapshot,
		"reviewer", latestActor.UserID,
	)

	return annotation, nil
}

// ListCWAnnotations 按可信查看权读取课件全部批注。
func (s *CoursewareService) ListCWAnnotations(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
) (
	*models.CWAnnotationListResponse,
	error,
) {
	if _, err :=
		s.LoadCoursewareForView(
			ctx,
			coursewareID,
			actor,
		); err != nil {
		return nil, err
	}

	annotations, err :=
		repository.
			ListCWAnnotationsByCoursewareID(
				ctx,
				coursewareID,
			)
	if err != nil {
		return nil, err
	}

	if annotations == nil {
		annotations =
			[]*models.CoursewareAnnotation{}
	}

	return &models.CWAnnotationListResponse{
		Annotations: annotations,
		Total:       len(annotations),
	}, nil
}

// ResolveCWAnnotation 标记批注已处理或重新待处理。
func (s *CoursewareService) ResolveCWAnnotation(
	ctx context.Context,
	annotationID string,
	actor *CoursewareActorContext,
	status string,
) error {
	status = strings.TrimSpace(
		status,
	)
	if status !=
		models.AnnotationStatusResolved &&
		status !=
			models.AnnotationStatusPending {
		return fmt.Errorf(
			"%w: 仅支持resolved或pending",
			ErrCoursewareAnnotationInputInvalid,
		)
	}

	_, annotation, scopedActor, err :=
		s.LoadCoursewareAnnotationForManage(
			ctx,
			annotationID,
			actor,
		)
	if err != nil {
		return err
	}

	// 正式更新前重新加载正式批注和课件并再次授权。
	_, latestAnnotation, _, err :=
		s.LoadCoursewareAnnotationForManage(
			ctx,
			annotationID,
			scopedActor,
		)
	if err != nil {
		return err
	}

	if !coursewareAnnotationRevisionUnchanged(
		annotation,
		latestAnnotation,
	) {
		return ErrCoursewareAnnotationMutationConflict
	}

	updated, err :=
		repository.
			UpdateCWAnnotationStatusIfUnchanged(
				ctx,
				annotation.CoursewareID,
				annotation.ID,
				annotation.UpdatedAt,
				status,
			)
	if err != nil {
		return err
	}
	if !updated {
		return ErrCoursewareAnnotationMutationConflict
	}

	cwAnnotationLog.Info(
		"课件批注状态变更",
		"annotation_id", annotation.ID,
		"to", status,
		"operator", scopedActor.UserID,
	)

	return nil
}

// DeleteCWAnnotation 删除课件页级批注。
func (s *CoursewareService) DeleteCWAnnotation(
	ctx context.Context,
	annotationID string,
	actor *CoursewareActorContext,
) error {
	_, annotation, scopedActor, err :=
		s.LoadCoursewareAnnotationForManage(
			ctx,
			annotationID,
			actor,
		)
	if err != nil {
		return err
	}

	// 正式删除前重新加载正式批注和课件并再次授权。
	_, latestAnnotation, _, err :=
		s.LoadCoursewareAnnotationForManage(
			ctx,
			annotationID,
			scopedActor,
		)
	if err != nil {
		return err
	}

	if !coursewareAnnotationRevisionUnchanged(
		annotation,
		latestAnnotation,
	) {
		return ErrCoursewareAnnotationMutationConflict
	}

	deleted, err :=
		repository.DeleteCWAnnotationIfUnchanged(
			ctx,
			annotation.CoursewareID,
			annotation.ID,
			annotation.UpdatedAt,
		)
	if err != nil {
		return err
	}
	if !deleted {
		return ErrCoursewareAnnotationMutationConflict
	}

	cwAnnotationLog.Info(
		"课件批注删除",
		"annotation_id", annotation.ID,
		"operator", scopedActor.UserID,
	)

	return nil
}
