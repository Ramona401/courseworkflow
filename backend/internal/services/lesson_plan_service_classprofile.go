package services

// lesson_plan_service_classprofile.go — 教案挂载/解除班级学情卡业务逻辑（差异化教学·前端入口）
//
// 老师在备课时（起步首屏或对话中途）把一张 active 班级学情卡挂到本教案上，
// 后端注入层（workshop_stage_service.go）下一轮对话自动重读 lesson_plans.class_profile_id，
// 在 analyze/design/write 三阶段注入该班级卡的四大段群体学情内容（差异化教学设计）。
// 引擎零改动，写列即生效（与单元方案挂载 UpdateLessonPlanUnitPlan 完全同款机制）。
//
// 校验规则（与 UpdateLessonPlanUnitPlan / UpdateLessonPlanTextbooks / UpdateLessonPlan 完全同款，勿单边改动）：
//   - 仅作者本人可操作（ErrLPNotAuthor）
//   - 可编辑状态白名单：draft / published_personal / revision / approved / published_shared；
//     submitted（评审中锁定）与 developing/completed（课件关联锁定）拒绝（ErrLPCannotEdit）
//
// 关于"为什么不在这里校验班级卡是否 active / 是否归属本人"：
//   - 挂载本身只写 lesson_plans.class_profile_id 这一列，不做班级卡存在性/归属/状态校验。
//   - 注入层已经焊死"只注入 status==active 且 created_by==本人 的班级卡"，
//     草稿/已归档/他人的卡挂上去也不会注入，不会造成数据泄漏或错误注入。
//   - 前端挂载选择器只列本人 active 班级卡（getClassProfiles），正常路径不会挂到非法卡。
//   - 这与 class_profile_id 列"无外键约束、被挂卡删除时本列不连删"的设计一脉相承
//     （镜像 unit_plan_id / teacher_assistant_prefs.assistant_id），保持 service 层职责单一：只管写列。
//
// 合规红线复述：注入链路只用班级卡四大段"群体结论"（匿名、无个人身份信息）；
// 学生个体明细（class_students）永不进注入链路。本文件只写一个关联ID，更不触碰个体明细。

import (
	"context"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// UpdateLessonPlanClassProfile 挂载或解除教案关联的班级学情卡
//
// classProfileID 传空串 "" → 解除挂载（class_profile_id 置 NULL，合法操作，等于"取消班级关联"）。
// classProfileID 传非空     → 挂载该班级学情卡（写入 class_profile_id）。
//
// 归属与状态校验与 UpdateLessonPlanUnitPlan 完全一致：仅作者本人、且教案处于可编辑状态。
func (s *LessonPlanService) UpdateLessonPlanClassProfile(ctx context.Context, id string, callerID string, classProfileID string) error {
	lp, err := repository.GetLessonPlanByID(ctx, id)
	if err != nil {
		return s.mapNotFoundErr(err)
	}
	if lp.AuthorID != callerID {
		return ErrLPNotAuthor
	}
	// 可编辑状态白名单（与 UpdateLessonPlanUnitPlan / UpdateLessonPlan / UpdateLessonPlanTextbooks 保持完全一致，勿单边改动）
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
	trimmed := strings.TrimSpace(classProfileID)
	var idPtr *string
	if trimmed != "" {
		idPtr = &trimmed
	}

	if err := repository.UpdateLessonPlanClassProfileID(ctx, id, idPtr); err != nil {
		lpLog.Error("更新教案班级学情关联失败", "plan_id", id, "error", err)
		return err
	}
	if idPtr == nil {
		lpLog.Info("教案已解除班级学情关联", "plan_id", id, "caller", callerID)
	} else {
		lpLog.Info("教案已挂载班级学情卡", "plan_id", id, "class_profile_id", trimmed, "caller", callerID)
	}
	return nil
}
