package services

// review_v2_service.go — 多级审核核心业务逻辑
//
// v127.2 修复：
//   - GetPendingReviews admin使用ListPendingReviewsL1All全量查（不限教研组角色）
//   - GetReviewStats admin传isAdmin=true看全局统计
//   - 新增 GetReviewedRecords 已审核记录列表
//
// v168改动（第二批治本·功能A·正文产出硬门控 切片A）：
//   新增 ErrReviewContentEmpty 错误，并在 ReviewL1/ReviewL2 的 approved 分支
//   增加教案正文非空硬校验。这是堵住"无正文教案走到 approved"根洞的关键：
//     - 校验只针对 approved（通过）决策；revision（退回）完全不受影响——
//       空正文教案应当被退回让作者去补正文，而不是被通过。
//     - 校验放在写审核记录（CreateReviewV2）之前，确保被拒绝的 approved
//       不会留下任何审核记录，状态机保持干净。
//   历史坏数据（《春》《英语》等）正是因为审核端零正文校验才进入 approved，
//   此改动从审核侧彻底封堵该路径。
//
// 阶段5 收尾改动（教案审核通知接线）：
//   ReviewL1/ReviewL2 的 approved/revision 分支尾部，旁路发"审核结果"通知给作者。
//   - ReviewL1 approved：仅在【直接终审】子路径（needL2=false）发"审核通过"——
//     进 L2 时教案尚未真正通过，不发"通过"给作者，避免误导。
//   - ReviewL1 revision / ReviewL2 approved / ReviewL2 revision：各自分支尾部发。
//   通知逻辑在 lesson_plan_review_notify.go 的包级函数 notifyLPAuthorReviewResult，
//   全程 best-effort 异步、写失败仅记日志，绝不阻断审核主业务。
//   提交审核通知（发 L1 审核员）不在本文件，在 lesson_plan_service.SubmitForReview。
//
// ★ 方案B（2026-07-03，学校管理员审核盲区修复，与课件侧同批同口径）★
//   问题：senior_operator 若不兼任教研组 lead/backbone，看不到本校 L1 待审教案（教案先落 L1，
//   未过 L1 不进 L2 列表）；组无组长/骨干时教案卡死在 L1。产品决策：本校 senior 对本校 L1
//   "可见 + 可审（兜底代审）"。落地三处：
//     1. GetPendingReviews senior 分支：L1 从"组内口径"升级为"本校全量口径"
//        （新增 repository.ListPendingReviewsL1BySchool，按教研组所属学校过滤）；
//        学校解析失败降级回老口径（组内 L1，无 L2），行为与改造前一致。
//     2. GetReviewStats senior：L1/L2 统一按本校口径（repo 层 GetReviewStats 签名加 schoolID，
//        同时改为 fail-closed 堵住历史统计泄漏——旧实现校管统计卡显示全系统待审数）。
//     3. ReviewL1 权限阶梯：admin → 本校 senior（GetSchoolByAdminUserID 与 resolveSchoolID 比对）
//        → 教案绑定教研组的 lead/backbone。
//   职责分离说明：组长在岗仍应由组长完成 L1，senior 定位为"兜底代审"；开启 l2_enabled 的
//   学校 senior 若代审 L1，L2 仍由其复审（L2 本就归 senior），属提前介入，管理上可接受。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== 错误常量 ====================

var (
	ErrReviewNotSubmitted    = errors.New("只有已提交评审的教案可以审核")
	ErrReviewNotL2Status     = errors.New("该教案不在L2待审核状态")
	ErrReviewNoPermission    = errors.New("您没有审核此教案的权限")
	ErrReviewInvalidDecision = errors.New("审核决策无效，可选值：approved/revision")
	ErrReviewPlanNotFound    = errors.New("教案不存在")
	// v168新增：教案正文为空时禁止审核通过（功能A硬门控·审核侧）
	ErrReviewContentEmpty = errors.New("教案正文为空，无法审核通过；请退回修改，由作者补全正文后再提交")
)

var reviewLog = logger.WithModule("review_v2")

// ReviewV2Service 多级审核服务
type ReviewV2Service struct {
	compService *ComponentService
}

// NewReviewV2Service 创建多级审核服务实例
func NewReviewV2Service(compService *ComponentService) *ReviewV2Service {
	return &ReviewV2Service{compService: compService}
}

// ==================== L1 教研组审核 ====================

// ReviewL1 L1 教研组审核决策。
//
// 权限阶梯（方案B，2026-07-03）：
//   admin 直放 → 本校 senior_operator 兜底可审（组长缺位时流程不死锁）
//   → 教案绑定教研组的 lead/backbone。
// 角色经 FindUserByID 现查（沿用 v127.3 既有模式，避免改 handler 签名与路由）。
func (s *ReviewV2Service) ReviewL1(ctx context.Context, planID string, reviewerID string, req *models.ReviewDecisionV2Request) error {
	lp, err := repository.GetLessonPlanByID(ctx, planID)
	if err != nil {
		if errors.Is(err, repository.ErrLessonPlanNotFound) {
			return ErrReviewPlanNotFound
		}
		return err
	}

	if lp.Status != models.LPStatusSubmitted || lp.ReviewLevel != 0 {
		return ErrReviewNotSubmitted
	}

	// 权限阶梯：admin → 本校 senior 兜底 → 教案绑定教研组的 lead/backbone
	reviewerRole := ""
	if u, uErr := repository.FindUserByID(ctx, reviewerID); uErr == nil {
		reviewerRole = u.Role
	}
	allowed := reviewerRole == models.RoleAdmin
	if !allowed && reviewerRole == models.RoleSeniorOperator {
		// 方案B：学校管理员对"审核学校=本校"的教案享有 L1 兜底审核权。
		// 教案学校口径经 resolveSchoolID：优先 review_school_id（提交时写入），
		// 回退教案 school_id / 教研组所属学校（与 L1 通过时"是否进 L2"的判定同一条解析链）。
		// 学校反查失败 / 解析为空 / 不匹配均不放行（fail-closed），继续落组内判定。
		if school, sErr := repository.GetSchoolByAdminUserID(ctx, reviewerID); sErr == nil && school != nil {
			planSchoolID := s.resolveSchoolID(ctx, lp)
			allowed = planSchoolID != "" && planSchoolID == school.ID
		}
	}
	if !allowed {
		// 组内 lead/backbone 判定（教案 L1 必须绑定教研组）
		if lp.GroupID == nil || *lp.GroupID == "" {
			return ErrReviewNoPermission
		}
		hasPermission, permErr := repository.IsGroupLeadOrBackbone(ctx, *lp.GroupID, reviewerID)
		if permErr != nil {
			return fmt.Errorf("校验审核权限失败: %w", permErr)
		}
		allowed = hasPermission
	}
	if !allowed {
		return ErrReviewNoPermission
	}

	if req.Decision != models.ReviewDecisionApproved && req.Decision != models.ReviewDecisionRevision {
		return ErrReviewInvalidDecision
	}

	// v168：approved 决策正文非空硬校验（在写审核记录之前拦截，被拒不留脏记录）
	// revision（退回）不校验——空正文本就该退回让作者补全。
	if req.Decision == models.ReviewDecisionApproved && strings.TrimSpace(lp.ContentMarkdown) == "" {
		reviewLog.Info("L1审核通过被拦截：教案正文为空", "plan_id", planID, "reviewer", reviewerID)
		return ErrReviewContentEmpty
	}

	existingCount, _ := repository.CountReviewsV2ByPlanAndLevel(ctx, planID, models.ReviewLevelL1)
	round := existingCount + 1

	review := &models.ReviewV2{
		LessonPlanID: planID,
		ReviewLevel:  models.ReviewLevelL1,
		ReviewerID:   reviewerID,
		Decision:     req.Decision,
		Score:        req.Score,
		Comment:      req.Comment,
		Dimensions:   req.Dimensions,
		ReviewRound:  round,
	}
	if err := repository.CreateReviewV2(ctx, review); err != nil {
		reviewLog.Error("创建L1审核记录失败", "plan_id", planID, "error", err)
		return err
	}

	s.syncLegacyReview(ctx, planID, reviewerID, req, round)

	switch req.Decision {
	case models.ReviewDecisionApproved:
		schoolID := s.resolveSchoolID(ctx, lp)
		var schoolIDPtr *string
		if schoolID != "" {
			schoolIDPtr = &schoolID
		}

		needL2 := false
		if schoolID != "" {
			cfg, cfgErr := repository.GetReviewFlowConfig(ctx, schoolID)
			if cfgErr == nil && cfg.L2Enabled {
				needL2 = true
			}
		}

		if needL2 {
			_ = repository.UpdateLessonPlanReviewLevel(ctx, planID, models.ReviewLevelL1, schoolIDPtr)
			reviewLog.Info("L1审核通过，进入L2待审核",
				"plan_id", planID, "school_id", schoolID, "round", round)
			// 进 L2 时教案尚未真正通过，不发"审核通过"给作者，避免误导；
			// L2 审核员是否提醒由其主动看审核中心列表（与课件审核一致）。
		} else {
			_ = repository.UpdateLessonPlanStatus(ctx, planID, models.LPStatusApproved)
			_ = repository.UpdateLessonPlanReviewLevel(ctx, planID, models.ReviewLevelL1, schoolIDPtr)
			reviewLog.Info("L1审核通过，直接终审", "plan_id", planID, "round", round)
			s.triggerAutoExtractIfEligible(ctx, lp, reviewerID)
			// 阶段5：L1 直接终审通过 → 通知作者"审核通过了"（best-effort 旁路）
			notifyLPAuthorReviewResult(ctx, lp, reviewerID, models.ReviewDecisionApproved, req.Comment)
		}

	case models.ReviewDecisionRevision:
		_ = repository.UpdateLessonPlanStatus(ctx, planID, models.LPStatusRevision)
		_ = repository.UpdateLessonPlanReviewLevel(ctx, planID, 0, nil)
		if restoreErr := repository.RestoreArchivedAnnotationsForLatestRound(ctx, planID); restoreErr != nil {
			reviewLog.Error("恢复归档批注失败（不影响退回）", "plan_id", planID, "error", restoreErr)
		}
		reviewLog.Info("L1审核退回", "plan_id", planID, "round", round)
		// 阶段5：L1 退回 → 通知作者"被退回"，审核意见进 body（best-effort 旁路）
		notifyLPAuthorReviewResult(ctx, lp, reviewerID, models.ReviewDecisionRevision, req.Comment)
	}

	return nil
}

// ==================== L2 学校审核 ====================

func (s *ReviewV2Service) ReviewL2(ctx context.Context, planID string, reviewerID string, reviewerRole string, req *models.ReviewDecisionV2Request) error {
	lp, err := repository.GetLessonPlanByID(ctx, planID)
	if err != nil {
		if errors.Is(err, repository.ErrLessonPlanNotFound) {
			return ErrReviewPlanNotFound
		}
		return err
	}

	if lp.Status != models.LPStatusSubmitted || lp.ReviewLevel != models.ReviewLevelL1 {
		return ErrReviewNotL2Status
	}

	if reviewerRole != models.RoleSeniorOperator && reviewerRole != models.RoleAdmin {
		return ErrReviewNoPermission
	}
	if reviewerRole == models.RoleSeniorOperator {
		school, err := repository.GetSchoolByAdminUserID(ctx, reviewerID)
		if err != nil {
			return ErrReviewNoPermission
		}
		if lp.ReviewSchoolID == nil || *lp.ReviewSchoolID != school.ID {
			return ErrReviewNoPermission
		}
	}

	if req.Decision != models.ReviewDecisionApproved && req.Decision != models.ReviewDecisionRevision {
		return ErrReviewInvalidDecision
	}

	// v168：approved 决策正文非空硬校验（在写审核记录之前拦截，被拒不留脏记录）
	// revision（退回）不校验——空正文本就该退回让作者补全。
	if req.Decision == models.ReviewDecisionApproved && strings.TrimSpace(lp.ContentMarkdown) == "" {
		reviewLog.Info("L2审核通过被拦截：教案正文为空", "plan_id", planID, "reviewer", reviewerID)
		return ErrReviewContentEmpty
	}

	existingCount, _ := repository.CountReviewsV2ByPlanAndLevel(ctx, planID, models.ReviewLevelL2)
	round := existingCount + 1

	review := &models.ReviewV2{
		LessonPlanID: planID,
		ReviewLevel:  models.ReviewLevelL2,
		ReviewerID:   reviewerID,
		Decision:     req.Decision,
		Score:        req.Score,
		Comment:      req.Comment,
		Dimensions:   req.Dimensions,
		ReviewRound:  round,
	}
	if err := repository.CreateReviewV2(ctx, review); err != nil {
		reviewLog.Error("创建L2审核记录失败", "plan_id", planID, "error", err)
		return err
	}

	s.syncLegacyReview(ctx, planID, reviewerID, req, round)

	switch req.Decision {
	case models.ReviewDecisionApproved:
		_ = repository.UpdateLessonPlanStatus(ctx, planID, models.LPStatusApproved)
		_ = repository.UpdateLessonPlanReviewLevel(ctx, planID, models.ReviewLevelL2, nil)
		reviewLog.Info("L2审核通过", "plan_id", planID, "round", round)
		s.triggerAutoExtractIfEligible(ctx, lp, reviewerID)
		// 阶段5：L2 终审通过 → 通知作者"审核通过了"（best-effort 旁路）
		notifyLPAuthorReviewResult(ctx, lp, reviewerID, models.ReviewDecisionApproved, req.Comment)

	case models.ReviewDecisionRevision:
		_ = repository.UpdateLessonPlanStatus(ctx, planID, models.LPStatusRevision)
		_ = repository.UpdateLessonPlanReviewLevel(ctx, planID, 0, nil)
		if restoreErr := repository.RestoreArchivedAnnotationsForLatestRound(ctx, planID); restoreErr != nil {
			reviewLog.Error("恢复归档批注失败（不影响退回）", "plan_id", planID, "error", restoreErr)
		}
		reviewLog.Info("L2审核退回", "plan_id", planID, "round", round)
		// 阶段5：L2 退回 → 通知作者"被退回"，审核意见进 body（best-effort 旁路）
		notifyLPAuthorReviewResult(ctx, lp, reviewerID, models.ReviewDecisionRevision, req.Comment)
	}

	return nil
}

// ==================== 审核历史查询 ====================

func (s *ReviewV2Service) GetReviewHistory(ctx context.Context, planID string) (*models.ReviewHistoryResponse, error) {
	reviews, err := repository.ListReviewsV2ByPlan(ctx, planID)
	if err != nil {
		return nil, err
	}
	if reviews == nil {
		reviews = []*models.ReviewV2ListItem{}
	}

	lp, err := repository.GetLessonPlanByID(ctx, planID)
	currentLevel := 0
	if err == nil {
		currentLevel = lp.ReviewLevel
	}

	return &models.ReviewHistoryResponse{
		Reviews:      reviews,
		Total:        len(reviews),
		CurrentLevel: currentLevel,
	}, nil
}

// ==================== 待审核列表 ====================

// GetPendingReviews 获取当前用户的待审核列表
//
// v127.2 修复：admin 使用 ListPendingReviewsL1All 全量查所有L1待审核
// 方案B（2026-07-03）：senior_operator 的 L1 待审升级为"本校全量口径"（详见分支内注释）
func (s *ReviewV2Service) GetPendingReviews(ctx context.Context, userID string, userRole string, limit int, offset int) (*models.PendingReviewListResponse, error) {
	if limit <= 0 {
		limit = 100
	}

	switch userRole {
	case models.RoleOperator, models.RoleViewer:
		items, total, err := repository.ListPendingReviewsL1(ctx, userID, limit, offset)
		if err != nil {
			return nil, err
		}
		return &models.PendingReviewListResponse{Items: items, Total: total}, nil

	case models.RoleSeniorOperator:
		// 方案B（2026-07-03）：学校管理员的 L1 待审从"组内口径"升级为"本校全量口径"
		//（ListPendingReviewsL1BySchool，按教研组所属学校过滤）。效果：不兼任组长/骨干的
		// 校管也能看到本校全部 L1 待审教案，并可兜底代审（见 ReviewL1 权限阶梯）。
		schoolID := ""
		if school, err := repository.GetSchoolByAdminUserID(ctx, userID); err == nil && school != nil {
			schoolID = school.ID
		}
		if schoolID != "" {
			l1Items, _, _ := repository.ListPendingReviewsL1BySchool(ctx, schoolID, 100, 0)
			l2Items, _, _ := repository.ListPendingReviewsL2(ctx, schoolID, 100, 0)
			allItems := append(l1Items, l2Items...)
			return &models.PendingReviewListResponse{Items: allItems, Total: len(allItems)}, nil
		}
		// 学校解析失败（senior 未绑校等异常）：降级回老口径——仅组内 L1，无 L2（行为与改造前一致）
		l1Items, _, _ := repository.ListPendingReviewsL1(ctx, userID, 100, 0)
		if l1Items == nil {
			l1Items = []*models.PendingReviewItem{}
		}
		return &models.PendingReviewListResponse{Items: l1Items, Total: len(l1Items)}, nil

	case models.RoleAdmin:
		// admin全量查L1（不限教研组角色） + L2（全部学校）
		l1Items, _, _ := repository.ListPendingReviewsL1All(ctx, 100, 0)
		l2Items, _, _ := repository.ListPendingReviewsL2(ctx, "", 100, 0)
		allItems := append(l1Items, l2Items...)
		return &models.PendingReviewListResponse{Items: allItems, Total: len(allItems)}, nil

	default:
		return &models.PendingReviewListResponse{Items: []*models.PendingReviewItem{}, Total: 0}, nil
	}
}

// ==================== 审核统计 ====================

// GetReviewStats 获取审核统计
//
// 方案B（2026-07-03）：按角色装配白名单，口径与各角色待审列表严格一致——
//   senior_operator → schoolID 本校（L1/L2 统一，与其列表新口径一致）；
//                     学校解析失败且 level=L1 时降级组内口径（与列表降级路径一致）；
//   其余非 admin 角色 → L1 传 lead/backbone 教研组（与 ListPendingReviewsL1 同口径），L2 无口径计 0。
// repo 层 GetReviewStats 已改为 fail-closed：白名单皆空时计 0，绝不退化全局。
func (s *ReviewV2Service) GetReviewStats(ctx context.Context, reviewerID string, userRole string, level int) (*models.ReviewStatsResponse, error) {
	isAdmin := userRole == models.RoleAdmin

	var groupIDs []string
	schoolID := ""
	if !isAdmin {
		switch userRole {
		case models.RoleSeniorOperator:
			// 方案B：L1、L2 统一按本校口径计数（与其待审列表新口径严格一致）
			if school, err := repository.GetSchoolByAdminUserID(ctx, reviewerID); err == nil && school != nil {
				schoolID = school.ID
			} else if level == models.ReviewLevelL1 {
				// 学校解析失败降级：回老口径（lead/backbone 组内），与列表降级路径一致
				groupIDs, _ = repository.GetUserLeadOrBackboneGroupIDs(ctx, reviewerID)
			}
		default:
			// v127.3: 非admin的L1统计需要传入教研组ID列表（与列表查询一致）；L2 无口径计 0
			if level == models.ReviewLevelL1 {
				groupIDs, _ = repository.GetUserLeadOrBackboneGroupIDs(ctx, reviewerID)
			}
		}
	}

	return repository.GetReviewStats(ctx, reviewerID, level, isAdmin, groupIDs, schoolID)
}

// ==================== 已审核记录列表（v127.2新增） ====================

// GetReviewedRecords 获取已审核记录列表
func (s *ReviewV2Service) GetReviewedRecords(ctx context.Context, reviewerID string, userRole string, level int, decision string, limit int, offset int) (*models.ReviewedListResponse, error) {
	isAdmin := userRole == models.RoleAdmin
	items, total, err := repository.ListReviewedRecords(ctx, reviewerID, level, decision, isAdmin, limit, offset)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*models.ReviewedListItem{}
	}
	return &models.ReviewedListResponse{Items: items, Total: total}, nil
}

// ==================== 审核流程配置 ====================

func (s *ReviewV2Service) GetReviewFlowConfig(ctx context.Context, schoolID string) (*models.ReviewFlowConfigResponse, error) {
	cfg, err := repository.GetReviewFlowConfig(ctx, schoolID)
	if err != nil {
		if errors.Is(err, repository.ErrReviewConfigNotFound) {
			school, _ := repository.GetOrganizationByID(ctx, schoolID)
			schoolName := ""
			if school != nil {
				schoolName = school.Name
			}
			return &models.ReviewFlowConfigResponse{
				SchoolID:              schoolID,
				SchoolName:            schoolName,
				L2Enabled:             false,
				L3SampleRate:          0.20,
				AutoPublishOnApproved: false,
			}, nil
		}
		return nil, err
	}

	school, _ := repository.GetOrganizationByID(ctx, cfg.SchoolID)
	schoolName := ""
	if school != nil {
		schoolName = school.Name
	}

	return &models.ReviewFlowConfigResponse{
		SchoolID:              cfg.SchoolID,
		SchoolName:            schoolName,
		L2Enabled:             cfg.L2Enabled,
		L3SampleRate:          cfg.L3SampleRate,
		AutoPublishOnApproved: cfg.AutoPublishOnApproved,
	}, nil
}

func (s *ReviewV2Service) UpdateReviewFlowConfig(ctx context.Context, schoolID string, req *models.UpdateReviewFlowConfigRequest, updatedBy string) error {
	if req.L3SampleRate < 0 || req.L3SampleRate > 1.0 {
		return errors.New("抽查比例必须在 0.00 - 1.00 之间")
	}
	return repository.UpsertReviewFlowConfig(ctx, schoolID, req, updatedBy)
}

// ==================== 内部辅助方法 ====================

// resolveSchoolID 解析教案归属学校ID：优先 review_school_id（提交时写入），
// 回退教案 school_id，再回退教案绑定教研组的所属学校。
// 被"L1 通过是否进 L2"与"方案B senior 本校 L1 权限判定"共用，保证两处口径一致。
func (s *ReviewV2Service) resolveSchoolID(ctx context.Context, lp *models.LessonPlan) string {
	if lp.ReviewSchoolID != nil && *lp.ReviewSchoolID != "" {
		return *lp.ReviewSchoolID
	}
	if lp.SchoolID != nil && *lp.SchoolID != "" {
		return *lp.SchoolID
	}
	if lp.GroupID != nil && *lp.GroupID != "" {
		group, err := repository.GetTeachingGroupByID(ctx, *lp.GroupID)
		if err == nil {
			return group.SchoolID
		}
	}
	return ""
}

func (s *ReviewV2Service) syncLegacyReview(ctx context.Context, planID string, reviewerID string, req *models.ReviewDecisionV2Request, round int) {
	legacyReview := &models.LessonPlanReview{
		LessonPlanID: planID,
		ReviewerID:   reviewerID,
		Decision:     req.Decision,
		Score:        req.Score,
		Comments:     req.Comment,
		Dimensions:   req.Dimensions,
		Round:        round,
	}
	if err := repository.CreateLessonPlanReview(ctx, legacyReview); err != nil {
		reviewLog.Error("同步旧版审核记录失败（不影响主流程）", "plan_id", planID, "error", err)
	}
}

func (s *ReviewV2Service) triggerAutoExtractIfEligible(ctx context.Context, lp *models.LessonPlan, reviewerID string) {
	if lp.AIReviewScore != nil && *lp.AIReviewScore >= 8.5 && s.compService != nil {
		planContent := lp.ContentMarkdown
		subject := lp.Subject
		grade := lp.Grade
		go func() {
			bgCtx := context.Background()
			reviewLog.Info("触发通道二自动萃取", "plan_id", lp.ID, "ai_score", *lp.AIReviewScore)
			if err := s.compService.AutoExtractFromLessonPlan(
				bgCtx, lp.ID, planContent, subject, grade, reviewerID,
			); err != nil {
				reviewLog.Error("通道二自动萃取失败", "plan_id", lp.ID, "error", err)
			}
		}()
	}
}
