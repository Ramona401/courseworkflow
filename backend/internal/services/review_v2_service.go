package services

// review_v2_service.go — 多级审核核心业务逻辑
//
// v127.2 修复：
//   - GetPendingReviews admin使用ListPendingReviewsL1All全量查（不限教研组角色）
//   - GetReviewStats admin传isAdmin=true看全局统计
//   - 新增 GetReviewedRecords 已审核记录列表

import (
	"context"
	"errors"
	"fmt"

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

	// v127.3: admin 跳过教研组角色校验，可直接审核任何L1教案
	isAdminUser := false
	if claims, claimsErr := repository.FindUserByID(ctx, reviewerID); claimsErr == nil && claims.Role == models.RoleAdmin {
		isAdminUser = true
	}

	if !isAdminUser {
		if lp.GroupID == nil || *lp.GroupID == "" {
			return ErrReviewNoPermission
		}
		hasPermission, err := repository.IsGroupLeadOrBackbone(ctx, *lp.GroupID, reviewerID)
		if err != nil {
			return fmt.Errorf("校验审核权限失败: %w", err)
		}
		if !hasPermission {
			return ErrReviewNoPermission
		}
	}

	if req.Decision != models.ReviewDecisionApproved && req.Decision != models.ReviewDecisionRevision {
		return ErrReviewInvalidDecision
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
		} else {
			_ = repository.UpdateLessonPlanStatus(ctx, planID, models.LPStatusApproved)
			_ = repository.UpdateLessonPlanReviewLevel(ctx, planID, models.ReviewLevelL1, schoolIDPtr)
			reviewLog.Info("L1审核通过，直接终审", "plan_id", planID, "round", round)
			s.triggerAutoExtractIfEligible(ctx, lp, reviewerID)
		}

	case models.ReviewDecisionRevision:
		_ = repository.UpdateLessonPlanStatus(ctx, planID, models.LPStatusRevision)
		_ = repository.UpdateLessonPlanReviewLevel(ctx, planID, 0, nil)
		if restoreErr := repository.RestoreArchivedAnnotationsForLatestRound(ctx, planID); restoreErr != nil {
			reviewLog.Error("恢复归档批注失败（不影响退回）", "plan_id", planID, "error", restoreErr)
		}
		reviewLog.Info("L1审核退回", "plan_id", planID, "round", round)
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

	case models.ReviewDecisionRevision:
		_ = repository.UpdateLessonPlanStatus(ctx, planID, models.LPStatusRevision)
		_ = repository.UpdateLessonPlanReviewLevel(ctx, planID, 0, nil)
		if restoreErr := repository.RestoreArchivedAnnotationsForLatestRound(ctx, planID); restoreErr != nil {
			reviewLog.Error("恢复归档批注失败（不影响退回）", "plan_id", planID, "error", restoreErr)
		}
		reviewLog.Info("L2审核退回", "plan_id", planID, "round", round)
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
		l1Items, _, _ := repository.ListPendingReviewsL1(ctx, userID, 100, 0)
		schoolID := ""
		school, err := repository.GetSchoolByAdminUserID(ctx, userID)
		if err == nil {
			schoolID = school.ID
		}
		var l2Items []*models.PendingReviewItem
		if schoolID != "" {
			l2Items, _, _ = repository.ListPendingReviewsL2(ctx, schoolID, 100, 0)
		}
		allItems := append(l1Items, l2Items...)
		return &models.PendingReviewListResponse{Items: allItems, Total: len(allItems)}, nil

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

func (s *ReviewV2Service) GetReviewStats(ctx context.Context, reviewerID string, userRole string, level int) (*models.ReviewStatsResponse, error) {
	isAdmin := userRole == models.RoleAdmin

	// v127.3: 非admin的L1统计需要传入教研组ID列表（与列表查询一致）
	var groupIDs []string
	if !isAdmin && level == models.ReviewLevelL1 {
		groupIDs, _ = repository.GetUserLeadOrBackboneGroupIDs(ctx, reviewerID)
	}

	return repository.GetReviewStats(ctx, reviewerID, level, isAdmin, groupIDs)
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
