package services

// lesson_plan_service_courseoutline.go — 教案设置/清除「课程大纲教材版本」业务逻辑（教材版本增强）
//
// 镜像 lesson_plan_service_classprofile.go 的作者+可编辑状态校验，但有一处关键差异：
//   班级学情用「空串=解除」；本模块的教材版本【空串是有意义的版本值（通用/不限版本）】，
//   故不能用空串表达解除。改用 *string 三态：
//     publisher == nil       → 解除关联（course_outline_publisher 置 NULL，注入层不注入大纲）
//     publisher 指向 ""       → 老师选了"通用/不限版本"（只注入 publisher 为空串的大纲）
//     publisher 指向 "人教版" → 选了具名版本（只注入该版本大纲，零跨版本兜底）
//
// 不校验该版本是否真有对应大纲（交注入层据实匹配，匹配不到则不注入，提示老师联系管理员上传）。

import (
	"context"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// UpdateLessonPlanCourseOutlinePublisher 设置/清除教案在备课首屏选定的课程大纲教材版本
//
// publisher 三态（见文件头）：nil=解除、指向空串=通用版、指向具名=该版本。
// 校验与 UpdateLessonPlanClassProfile 完全一致（作者本人 + 可编辑状态白名单）。
// 写列即生效——注入层下一轮 analyze/design 阶段重读 lesson_plans.course_outline_publisher。
func (s *LessonPlanService) UpdateLessonPlanCourseOutlinePublisher(ctx context.Context, id string, callerID string, publisher *string) error {
	lp, err := repository.GetLessonPlanByID(ctx, id)
	if err != nil {
		return s.mapNotFoundErr(err)
	}
	if lp.AuthorID != callerID {
		return ErrLPNotAuthor
	}
	// 可编辑状态白名单（与 UpdateLessonPlanClassProfile / UpdateLessonPlanUnitPlan 保持完全一致，勿单边改动）
	editableStatuses := map[string]bool{
		models.LPStatusDraft:             true,
		models.LPStatusPublishedPersonal: true,
		models.LPStatusRevision:          true,
		models.LPStatusApproved:          true,
		models.LPStatusPublishedShared:   true,
	}
	if !editableStatuses[lp.Status] {
		return ErrLPCannotEdit
	}

	// 版本三态处理（关键差异：空串是有效版本值=通用版，不可归一为 nil）：
	//   publisher == nil          → 直接写 NULL（解除关联）
	//   publisher 指向任意字符串    → TrimSpace 后写入（含空串=通用版）
	var pubPtr *string
	if publisher != nil {
		trimmed := strings.TrimSpace(*publisher)
		pubPtr = &trimmed
	}

	if err := repository.UpdateLessonPlanCourseOutlinePublisher(ctx, id, pubPtr); err != nil {
		lpLog.Error("更新教案课程大纲版本失败", "plan_id", id, "error", err)
		return err
	}
	if pubPtr == nil {
		lpLog.Info("教案已解除课程大纲版本关联（不注入大纲）", "plan_id", id, "caller", callerID)
	} else {
		lpLog.Info("教案已设定课程大纲教材版本", "plan_id", id, "publisher", *pubPtr, "caller", callerID)
	}
	return nil
}
