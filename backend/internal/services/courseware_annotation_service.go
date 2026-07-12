package services

// courseware_annotation_service.go — 课件页级批注业务逻辑(阶段2)
//
// 权限模型(三档):
//   - 写/读批注:能"看到"该课件的人即可。复用阶段1的共享可见性
//     resolveSameOrgUserIDs(同 *CoursewareService,同包私有方法):
//       · 课件作者本人 → 无条件放行
//       · admin → 无条件放行
//       · 否则要求课件已共享(published_shared)且当前用户在"同校/同组作者白名单"内
//   - 删除/标记:批注作者本人 / 课件作者本人 / admin 三者之一(比写权限宽,
//     因为课件作者要能管理自己课件上别人留的批注)。
//
// 挂载点为 page_number,创建时校验该页号在课件里真实存在(防挂到不存在的页)。
// 本文件方法均挂在既有 *CoursewareService 上,共用 cwServiceLog,不碰原有方法。

import (
	"context"
	"fmt"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// cwAnnotationLog 模块日志(复用同包 cwServiceLog)
var cwAnnotationLog = cwServiceLog

// ==================== 内部:可见性裁决(能否看到该课件)====================

// canViewCourseware 裁决"当前用户能否看到该课件"(= 能否读写批注的前置)。
// 规则:作者本人 / admin → true;否则课件须已共享且用户在同校同组白名单内。
func (s *CoursewareService) canViewCourseware(ctx context.Context, cw *models.Courseware, userID string, role string) bool {
	if cw.UserID == userID {
		return true
	}
	if role == models.RoleAdmin {
		return true
	}
	// 非作者非admin:课件必须已共享,且用户在"能看到此作者共享课件"的白名单内
	if cw.PublishState != models.CWPublishPublishedShared {
		return false
	}
	visibleAuthorIDs := s.resolveSameOrgUserIDs(ctx, userID)
	for _, aid := range visibleAuthorIDs {
		if aid == cw.UserID {
			return true
		}
	}
	return false
}

// ==================== 创建批注 ====================

// CreateCWAnnotation 创建课件页级批注。
// 校验链:课件存在 → 当前用户可看到该课件 → 页号真实存在 → 内容非空 → 落库。
func (s *CoursewareService) CreateCWAnnotation(ctx context.Context, coursewareID string, userID string, role string, reviewerName string, req *models.CreateCWAnnotationRequest) (*models.CoursewareAnnotation, error) {
	cw, err := repository.GetCoursewareByID(ctx, coursewareID)
	if err != nil {
		return nil, fmt.Errorf("课件不存在: %w", err)
	}
	if !s.canViewCourseware(ctx, cw, userID, role) {
		return nil, fmt.Errorf("无权批注此课件(需作者本人或能看到该共享课件的成员)")
	}

	content := strings.TrimSpace(req.Content)
	if content == "" {
		return nil, fmt.Errorf("批注内容不能为空")
	}
	if req.PageNumber <= 0 {
		return nil, fmt.Errorf("页号无效")
	}

	// 校验页号真实存在(防止挂到不存在的页)
	if _, err := repository.GetCoursewarePageByNumber(ctx, coursewareID, req.PageNumber); err != nil {
		return nil, fmt.Errorf("第 %d 页不存在,无法添加批注", req.PageNumber)
	}

	a := &models.CoursewareAnnotation{
		CoursewareID: coursewareID,
		PageNumber:   req.PageNumber,
		ReviewerID:   userID,
		ReviewerName: reviewerName,
		Content:      content,
	}
	if err := repository.CreateCWAnnotation(ctx, a); err != nil {
		return nil, fmt.Errorf("创建批注失败: %w", err)
	}
	cwAnnotationLog.Info("课件批注创建",
		"courseware_id", coursewareID, "page", req.PageNumber, "reviewer", userID)
	return a, nil
}

// ==================== 列表 ====================

// ListCWAnnotations 列出课件全部批注(前端按 page_number 分组挂气泡)。
// 读权限同写权限:能看到该课件才能读批注。
func (s *CoursewareService) ListCWAnnotations(ctx context.Context, coursewareID string, userID string, role string) (*models.CWAnnotationListResponse, error) {
	cw, err := repository.GetCoursewareByID(ctx, coursewareID)
	if err != nil {
		return nil, fmt.Errorf("课件不存在: %w", err)
	}
	if !s.canViewCourseware(ctx, cw, userID, role) {
		return nil, fmt.Errorf("无权查看此课件的批注")
	}

	items, err := repository.ListCWAnnotationsByCoursewareID(ctx, coursewareID)
	if err != nil {
		return nil, err
	}
	return &models.CWAnnotationListResponse{
		Annotations: items,
		Total:       len(items),
	}, nil
}

// ==================== 标记状态(已处理 / 重新待处理)====================

// ResolveCWAnnotation 标记批注处理状态(resolved / pending)。
// 权限:批注作者本人 / 课件作者本人 / admin。
func (s *CoursewareService) ResolveCWAnnotation(ctx context.Context, annotationID string, userID string, role string, status string) error {
	if status != models.AnnotationStatusResolved && status != models.AnnotationStatusPending {
		return fmt.Errorf("无效的状态值(仅支持 resolved / pending)")
	}
	a, err := repository.GetCWAnnotationByID(ctx, annotationID)
	if err != nil {
		return err // ErrCWAnnotationNotFound 由 handler 映射 404
	}
	if err := s.ensureCanManageAnnotation(ctx, a, userID, role); err != nil {
		return err
	}
	if err := repository.UpdateCWAnnotationStatus(ctx, annotationID, status); err != nil {
		return err
	}
	cwAnnotationLog.Info("课件批注状态变更",
		"annotation_id", annotationID, "to", status, "operator", userID)
	return nil
}

// ==================== 删除 ====================

// DeleteCWAnnotation 删除批注。
// 权限:批注作者本人 / 课件作者本人 / admin。
func (s *CoursewareService) DeleteCWAnnotation(ctx context.Context, annotationID string, userID string, role string) error {
	a, err := repository.GetCWAnnotationByID(ctx, annotationID)
	if err != nil {
		return err // ErrCWAnnotationNotFound 由 handler 映射 404
	}
	if err := s.ensureCanManageAnnotation(ctx, a, userID, role); err != nil {
		return err
	}
	if err := repository.DeleteCWAnnotation(ctx, annotationID); err != nil {
		return err
	}
	cwAnnotationLog.Info("课件批注删除",
		"annotation_id", annotationID, "operator", userID)
	return nil
}

// ==================== 内部:管理权(删/标记)裁决 ====================

// ensureCanManageAnnotation 裁决"当前用户能否管理(删/标记)这条批注"。
// 放行:admin / 批注作者本人 / 课件作者本人。
func (s *CoursewareService) ensureCanManageAnnotation(ctx context.Context, a *models.CoursewareAnnotation, userID string, role string) error {
	if role == models.RoleAdmin {
		return nil
	}
	if a.ReviewerID == userID {
		return nil // 批注作者本人
	}
	// 课件作者本人(有权管理自己课件上的批注)
	cw, err := repository.GetCoursewareByID(ctx, a.CoursewareID)
	if err == nil && cw.UserID == userID {
		return nil
	}
	return fmt.Errorf("无权操作此批注(仅批注作者、课件作者或管理员可操作)")
}
