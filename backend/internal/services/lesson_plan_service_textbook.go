package services

// lesson_plan_service_textbook.go — 教案课本关联业务逻辑（迭代3.5 A2-2 新增）
//
// 对话模式「课本中途挂载」：老师在备课对话进行中随时关联/更换课本页面，
// 下一轮对话引擎自动重读 textbook_page_ids 拼 OCR 原文（引擎零改动）。
//
// 校验规则（与 UpdateLessonPlan 完全同款）：
//   - 仅作者本人可操作（ErrLPNotAuthor）
//   - 可编辑状态白名单：draft / published_personal / revision / approved / published_shared；
//     submitted（评审中锁定）与 developing/completed（课件关联锁定）拒绝（ErrLPCannotEdit）

import (
	"context"
	"encoding/json"
	"fmt"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// UpdateLessonPlanTextbooks 更新教案关联的课本页面ID列表
//
// pageIDs 传空切片即解除全部课本关联（合法操作）。
// 序列化为 JSON 数组字符串后写入 jsonb 列（与 CreateLessonPlan 的写入格式一致）。
func (s *LessonPlanService) UpdateLessonPlanTextbooks(ctx context.Context, id string, callerID string, pageIDs []string) error {
	lp, err := repository.GetLessonPlanByID(ctx, id)
	if err != nil {
		return s.mapNotFoundErr(err)
	}
	if lp.AuthorID != callerID {
		return ErrLPNotAuthor
	}
	// 可编辑状态白名单（与 UpdateLessonPlan 保持完全一致，勿单边改动）
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

	// nil 归一为空数组，保证序列化结果是 "[]" 而非 "null"
	if pageIDs == nil {
		pageIDs = []string{}
	}
	b, err := json.Marshal(pageIDs)
	if err != nil {
		return fmt.Errorf("序列化课本页面ID失败: %w", err)
	}

	if err := repository.UpdateLessonPlanTextbookPages(ctx, id, string(b)); err != nil {
		lpLog.Error("更新教案课本关联失败", "plan_id", id, "error", err)
		return err
	}
	lpLog.Info("教案课本关联已更新", "plan_id", id, "page_count", len(pageIDs), "caller", callerID)
	return nil
}
