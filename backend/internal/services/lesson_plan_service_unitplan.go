package services

// lesson_plan_service_unitplan.go — 教案挂载/解除单元方案业务逻辑（大单元备课·前端入口）
//
// 老师在备课时（起步首屏或对话中途）把一份 active 单元方案挂到本教案上，
// 后端注入层（workshop_stage_service.go）下一轮对话自动重读 lesson_plans.unit_plan_id，
// 注入单元方案上下文（五阶段全程 + 课程大纲让位）。引擎零改动，写列即生效。
//
// 校验规则（与 UpdateLessonPlanTextbooks / UpdateLessonPlan 完全同款，勿单边改动）：
//   - 仅作者本人可操作（ErrLPNotAuthor）
//   - 可编辑状态白名单：draft / published_personal / revision / approved / published_shared；
//     submitted（评审中锁定）与 developing/completed（课件关联锁定）拒绝（ErrLPCannotEdit）
//
// 关于"为什么不在这里校验单元方案是否 active / 是否可见"：
//   - 挂载本身只写 lesson_plans.unit_plan_id 这一列，不做单元方案存在性/可见性/状态校验。
//   - 注入层已经焊死"只注入 status==active 的单元方案"，草稿/已归档挂上去也不会注入，
//     不会造成数据泄漏或错误注入。
//   - 前端挂载选择器只列 active（ListMountableUnitPlans），正常路径不会挂到非 active 的。
//   - 这与 unit_plan_id 列"无外键约束、被挂方案删除时本列不连删"的设计一脉相承
//     （镜像 teacher_assistant_prefs.assistant_id），保持 service 层职责单一：只管写列。

import (
	"context"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// UpdateLessonPlanUnitPlan 挂载或解除教案关联的单元方案
//
// unitPlanID 传空串 "" → 解除挂载（unit_plan_id 置 NULL，合法操作，等于"取消大单元绑定"）。
// unitPlanID 传非空     → 挂载该单元方案（写入 unit_plan_id）。
//
// 归属与状态校验与 UpdateLessonPlanTextbooks 完全一致：仅作者本人、且教案处于可编辑状态。
func (s *LessonPlanService) UpdateLessonPlanUnitPlan(ctx context.Context, id string, callerID string, unitPlanID string) error {
	lp, err := repository.GetLessonPlanByID(ctx, id)
	if err != nil {
		return s.mapNotFoundErr(err)
	}
	if lp.AuthorID != callerID {
		return ErrLPNotAuthor
	}
	// 可编辑状态白名单（与 UpdateLessonPlan / UpdateLessonPlanTextbooks 保持完全一致，勿单边改动）
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

	// 空串归一为 nil → repo 层写 NULL（解除挂载）；非空 → 传指针写入。
	trimmed := strings.TrimSpace(unitPlanID)
	var idPtr *string
	if trimmed != "" {
		idPtr = &trimmed
	}

	if err := repository.UpdateLessonPlanUnitPlanID(ctx, id, idPtr); err != nil {
		lpLog.Error("更新教案单元方案关联失败", "plan_id", id, "error", err)
		return err
	}
	if idPtr == nil {
		lpLog.Info("教案已解除单元方案关联", "plan_id", id, "caller", callerID)
	} else {
		lpLog.Info("教案已挂载单元方案", "plan_id", id, "unit_plan_id", trimmed, "caller", callerID)
	}
	return nil
}
