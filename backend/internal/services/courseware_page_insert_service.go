package services

// courseware_page_insert_service.go — 课件页面指定位置插入服务。
//
// 本文件只负责“新增页面插入到第几页”的业务编排，避免继续扩张
// courseware_service_curriculum.go中的页面基础操作代码。
//
// 正式HTTP入口必须调用AddPageAtForActor：
//   可信Actor
//   → 重新加载课件
//   → 作者本人及教育域快照校验
//   → 审核写锁校验
//   → 仓储事务内校验插入位置
//   → 仓储事务原子插页
//   → 刷新全部页面导航栏中的当前页码和总页数。
//
// insertAt兼容规则：
//   - insertAt小于等于0：兼容旧客户端，在事务锁内读取真实页数并追加到最后；
//   - insertAt为1至count+1：插入指定位置；
//   - 超出范围：明确拒绝，不擅自修正到末尾。
//
// 并发原则：
//   服务层不在事务外预先计算“最后一页”，避免两个并发插页请求都使用旧页数。
//   真实插入位置由仓储层取得课件页码事务锁后确定。

import (
	"context"
	"fmt"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// AddPageAtForActor是课件新增页面正式安全入口。
//
// 它先复用课件作者控制面的完整鉴权与审核锁，之后才执行插页。
// 前端提交的用户ID、教育域或角色都不能替代可信Actor。
func (s *CoursewareService) AddPageAtForActor(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	request *models.AddCWPageRequest,
	insertAt int,
) (*models.CoursewarePage, error) {
	_, scopedActor, err :=
		s.loadOwnedCoursewareForControlMutation(
			ctx,
			coursewareID,
			actor,
		)
	if err != nil {
		return nil, err
	}

	return s.addPageAtPosition(
		ctx,
		coursewareID,
		scopedActor.UserID,
		request,
		insertAt,
	)
}

// addPageAtPosition校验页面方案并调用仓储事务完成原子插页。
//
// 插入页码范围由仓储事务根据锁内真实页数校验，
// 避免服务层读到旧页数后产生并发位置错误。
func (s *CoursewareService) addPageAtPosition(
	ctx context.Context,
	coursewareID string,
	userID string,
	request *models.AddCWPageRequest,
	insertAt int,
) (*models.CoursewarePage, error) {
	if request == nil {
		return nil, fmt.Errorf(
			"新增页面请求不能为空",
		)
	}

	if strings.TrimSpace(
		coursewareID,
	) == "" {
		return nil, fmt.Errorf(
			"缺少课件ID",
		)
	}

	if strings.TrimSpace(
		userID,
	) == "" {
		return nil, fmt.Errorf(
			"缺少操作用户",
		)
	}

	estimatedComplexity :=
		request.EstimatedComplexity

	if estimatedComplexity <= 0 {
		estimatedComplexity = 1
	}

	page := &models.CoursewarePage{
		CoursewareID:
			coursewareID,
		PageNumber:
			insertAt,
		Title:
			strings.TrimSpace(
				request.Title,
			),
		Purpose:
			strings.TrimSpace(
				request.Purpose,
			),
		ContentSummary:
			strings.TrimSpace(
				request.ContentSummary,
			),
		InteractionType:
			strings.TrimSpace(
				request.InteractionType,
			),
		VisualFormat:
			strings.TrimSpace(
				request.VisualFormat,
			),
		MediaRequirements:
			strings.TrimSpace(
				request.MediaRequirements,
			),
		EstimatedComplexity:
			estimatedComplexity,
		Status:
			models.CWPageStatusPending,
	}

	if err := repository.InsertCoursewarePageAtPosition(
		ctx,
		page,
		insertAt,
	); err != nil {
		if insertAt > 0 {
			return nil, fmt.Errorf(
				"插入第%d页失败: %w",
				insertAt,
				err,
			)
		}

		return nil, fmt.Errorf(
			"追加页面失败: %w",
			err,
		)
	}

	// 插页事务已经保证page_number与page_count正确。
	// 此处继续刷新所有已有HTML导航栏中的“当前页 / 总页数”。
	//
	// 导航刷新是插页后的增强动作：失败不回滚已经成功插入的页面，
	// 避免前端收到失败后重复提交并产生第二个页面。
	if resyncErr := s.ResyncCWPageNumbers(
		ctx,
		coursewareID,
	); resyncErr != nil {
		cwServiceLog.Warn(
			"指定位置插页成功，但导航页码刷新未完全成功",
			"courseware_id",
			coursewareID,
			"requested_insert_at",
			insertAt,
			"actual_insert_at",
			page.PageNumber,
			"page_id",
			page.ID,
			"user_id",
			userID,
			"error",
			resyncErr,
		)
	}

	totalPages, countErr :=
		repository.CountCoursewarePages(
			ctx,
			coursewareID,
		)

	if countErr != nil {
		cwServiceLog.Warn(
			"插页完成后读取总页数失败",
			"courseware_id",
			coursewareID,
			"page_id",
			page.ID,
			"error",
			countErr,
		)
	}

	cwServiceLog.Info(
		"课件页面已插入指定位置",
		"courseware_id",
		coursewareID,
		"requested_insert_at",
		insertAt,
		"actual_insert_at",
		page.PageNumber,
		"page_id",
		page.ID,
		"user_id",
		userID,
		"total_pages",
		totalPages,
	)

	return page, nil
}
