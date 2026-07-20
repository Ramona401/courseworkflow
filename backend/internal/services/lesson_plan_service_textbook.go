package services

// lesson_plan_service_textbook.go — 教案课本关联业务逻辑（迭代3.5 A2-2 新增）
//
// 对话模式「课本中途挂载」：老师在备课对话进行中随时关联/更换课本页面，
// 下一轮对话引擎自动重读 textbook_page_ids 拼 OCR 原文。
//
// 校验规则：
//   - 仅作者本人可操作（ErrLPNotAuthor）
//   - 可编辑状态白名单：draft / published_personal / revision / approved / published_shared
//   - submitted（评审中锁定）与 developing/completed（课件关联锁定）拒绝（ErrLPCannotEdit）
//   - 上下文15：调用方实时教育域必须为k12
//   - 上下文15：教案创建快照域必须为k12
//   - 上下文15：所有课本ID必须存在、active、无重复，并与教案学科和年级一致
//   - 传空数组表示解除全部关联，但仍须经过用户域、教案域、作者和状态校验

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== 上下文15专用错误 ====================

var (
	// ErrLPTextbookEducationDomainDenied 表示用户域或教案快照域不允许使用K12课本模块。
	ErrLPTextbookEducationDomainDenied = errors.New("当前教育域不允许关联课本")

	// ErrLPTextbookSelectionInvalid 表示课本ID不存在、已归档、重复或属性与教案不一致。
	ErrLPTextbookSelectionInvalid = errors.New("课本页面选择无效")
)

// normalizeLessonPlanTextbookPageIDs 清理并校验课本页面ID列表。
//
// 规则：
//   - 空切片是合法的解除挂载操作；
//   - 每个ID必须为非空字符串；
//   - 不允许重复ID，防止通过重复项制造数量与实际资源不一致；
//   - 保持原始顺序，确保后续提示词按老师选择顺序读取。
func normalizeLessonPlanTextbookPageIDs(pageIDs []string) ([]string, error) {
	if pageIDs == nil {
		return []string{}, nil
	}

	normalized := make([]string, 0, len(pageIDs))
	seen := make(map[string]struct{}, len(pageIDs))
	for _, rawID := range pageIDs {
		id := strings.TrimSpace(rawID)
		if id == "" {
			return nil, ErrLPTextbookSelectionInvalid
		}
		if _, exists := seen[id]; exists {
			return nil, ErrLPTextbookSelectionInvalid
		}
		seen[id] = struct{}{}
		normalized = append(normalized, id)
	}
	return normalized, nil
}

// textbookPageMatchesLessonPlan 验证课本页面属性是否与目标教案一致。
//
// 当前课本表没有education_domain列，因为该模块本身就是K12专属资源；
// 因此资源属性验证由以下三层组成：
//   - Repository只在显式k12参数下返回active页面；
//   - 页面学科必须与教案学科一致；
//   - 页面年级范围必须与教案年级一致。
func textbookPageMatchesLessonPlan(page *models.TextbookPage, lessonPlan *models.LessonPlan) bool {
	if page == nil || lessonPlan == nil {
		return false
	}
	if strings.TrimSpace(page.Status) != "active" {
		return false
	}
	if strings.TrimSpace(page.Subject) != strings.TrimSpace(lessonPlan.Subject) {
		return false
	}
	if strings.TrimSpace(page.GradeRange) != strings.TrimSpace(lessonPlan.Grade) {
		return false
	}
	return true
}

// UpdateLessonPlanTextbooks 更新教案关联的课本页面ID列表。
//
// pageIDs传空切片即解除全部课本关联（合法操作）。
// 序列化为JSON数组字符串后写入jsonb列（与CreateLessonPlan的写入格式一致）。
func (s *LessonPlanService) UpdateLessonPlanTextbooks(ctx context.Context, id string, callerID string, pageIDs []string) error {
	// 第一层：实时解析操作者教育域，禁止依赖JWT或前端参数。
	actorContext, err := resolveTextbookActorEducationContext(ctx, callerID)
	if err != nil {
		return fmt.Errorf("解析课本挂载操作者教育域失败: %w", err)
	}
	if !textbookActorCanUseK12Module(actorContext) {
		return ErrLPTextbookEducationDomainDenied
	}

	// 第二层：读取教案正式数据库快照并校验作者与状态。
	lp, err := repository.GetLessonPlanByID(ctx, id)
	if err != nil {
		return s.mapNotFoundErr(err)
	}
	if lp.AuthorID != callerID {
		return ErrLPNotAuthor
	}

	// 课本能力只属于K12教案，教案创建快照域是运行时唯一依据。
	if strings.ToLower(strings.TrimSpace(lp.EducationDomain)) != models.EducationDomainK12 {
		return ErrLPTextbookEducationDomainDenied
	}

	// 可编辑状态白名单（与UpdateLessonPlan保持完全一致，勿单边改动）
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

	normalizedPageIDs, err := normalizeLessonPlanTextbookPageIDs(pageIDs)
	if err != nil {
		return err
	}

	// 空数组表示解除全部关联。
	//
	// 即使是解除挂载也必须在完成用户域、教案域、作者与状态校验后才能执行，
	// 从而满足“非K12挂载和解除挂载均返回403”的正式规则。
	if len(normalizedPageIDs) > 0 {
		pages, err := repository.GetTextbookPagesByIDsForEducationDomain(
			ctx,
			normalizedPageIDs,
			models.EducationDomainK12,
		)
		if err != nil {
			if errors.Is(err, repository.ErrTextbookEducationDomainUnsupported) {
				return ErrLPTextbookEducationDomainDenied
			}
			return fmt.Errorf("校验课本页面失败: %w", err)
		}

		// 查询结果数量必须与请求唯一ID数量完全一致。
		//
		// 任何不存在、已归档或伪造ID都会被Repository的active条件排除，
		// 数量不一致即整体拒绝，不允许部分挂载。
		if len(pages) != len(normalizedPageIDs) {
			return ErrLPTextbookSelectionInvalid
		}

		for _, page := range pages {
			if !textbookPageMatchesLessonPlan(page, lp) {
				return ErrLPTextbookSelectionInvalid
			}
		}
	}

	b, err := json.Marshal(normalizedPageIDs)
	if err != nil {
		return fmt.Errorf("序列化课本页面ID失败: %w", err)
	}

	if err := repository.UpdateLessonPlanTextbookPages(ctx, id, string(b)); err != nil {
		lpLog.Error("更新教案课本关联失败", "plan_id", id, "error", err)
		return err
	}
	lpLog.Info(
		"教案课本关联已更新",
		"plan_id", id,
		"page_count", len(normalizedPageIDs),
		"caller", callerID,
		"education_domain", models.EducationDomainK12,
	)
	return nil
}
