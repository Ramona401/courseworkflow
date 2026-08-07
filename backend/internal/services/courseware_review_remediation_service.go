package services

// courseware_review_remediation_service.go
//
// 课件作者整改中心的只读聚合服务。
//
// 返回内容：
//   1. 正式人工审核提交时生成的不可变整体反馈快照；
//   2. 作者自己的AI自审整改项；
//   3. 已随正式审核结果交付作者的页级整改项。
//
// 权限边界：
//   - 复用ListCWOwnerReviewItems执行作者身份和教育域校验；
//   - 未绑定feedback_id的正式整改项仍不会向作者开放；
//   - Service返回数据库业务模型，HTTP层负责剥离内部AI会话ID和创建者ID。

import (
	"context"
	"errors"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// CWOwnerReviewRemediationBundle 是作者整改中心的聚合结果。
type CWOwnerReviewRemediationBundle struct {
	Feedbacks []*models.CoursewareReviewFeedback `json:"feedbacks"`
	Items     []*models.CoursewareReviewItem     `json:"items"`
}

// GetCWOwnerReviewRemediation 返回作者可见的正式反馈和整改项。
func (s *CoursewareAIReviewService) GetCWOwnerReviewRemediation(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
) (*CWOwnerReviewRemediationBundle, error) {
	if s == nil {
		return nil, errors.New(
			"课件AI审核服务未初始化",
		)
	}

	coursewareID = strings.TrimSpace(
		coursewareID,
	)
	if coursewareID == "" {
		return nil, ErrCWAIReviewCoursewareNotFound
	}

	// 先复用现有作者整改项入口完成：
	//   - 登录校验；
	//   - 课件存在性；
	//   - 教育域校验；
	//   - 作者本人校验；
	//   - 正式整改项交付状态过滤。
	items, err := s.ListCWOwnerReviewItems(
		ctx,
		coursewareID,
		actor,
	)
	if err != nil {
		return nil, err
	}

	feedbacks, err :=
		repository.ListCoursewareReviewFeedbackByCourseware(
			ctx,
			coursewareID,
		)
	if err != nil {
		return nil, err
	}

	if feedbacks == nil {
		feedbacks =
			[]*models.CoursewareReviewFeedback{}
	}
	if items == nil {
		items =
			[]*models.CoursewareReviewItem{}
	}

	return &CWOwnerReviewRemediationBundle{
		Feedbacks: feedbacks,
		Items:     items,
	}, nil
}
