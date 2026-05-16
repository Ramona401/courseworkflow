package services

// inspection_service.go — 区域抽查业务逻辑
//
// v127 新增（多级审核体系 · 区域抽查）：
//   - 抽查记录管理（查看、分配、提交结果）
//   - 自动抽样（按学校配置的比例随机抽取已发布教案）
//   - 撤回教案（抽查发现问题，将教案从 published_shared 退回 revision）
//   - 区域教研员管辖分配管理
//
// 设计原则：
//   - L3 抽查是非阻塞的：L2通过后教案立即可发布，抽查是事后行为
//   - 抽查按比例抽样（默认20%），可手动触发也可定时触发
//   - 发现问题可撤回已发布教案，教案状态回到 revision

import (
	"context"
	"errors"
	"fmt"
	"time"

	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== 错误常量 ====================

var (
	ErrInspectionNotFound     = errors.New("抽查记录不存在")
	ErrInspectionNotAssigned  = errors.New("该抽查记录未分配审查员")
	ErrInspectionAlreadyDone  = errors.New("该抽查记录已完成")
	ErrInspectionNoPermission = errors.New("您没有操作此抽查记录的权限")
	ErrInspectionInvalidDecision = errors.New("抽查决策无效，可选值：passed/revoked")
)

var inspLog = logger.WithModule("inspection")

// InspectionService 抽查服务
type InspectionService struct{}

// NewInspectionService 创建抽查服务实例
func NewInspectionService() *InspectionService {
	return &InspectionService{}
}

// ==================== 抽查列表与详情 ====================

// ListInspections 获取抽查列表
func (s *InspectionService) ListInspections(ctx context.Context, inspectorID string, status string, limit int, offset int) (*models.InspectionListResponse, error) {
	items, total, err := repository.ListInspections(ctx, inspectorID, status, limit, offset)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*models.InspectionListItem{}
	}
	return &models.InspectionListResponse{Items: items, Total: total}, nil
}

// GetInspection 获取抽查详情
func (s *InspectionService) GetInspection(ctx context.Context, id string) (*models.InspectionRecord, error) {
	record, err := repository.GetInspectionByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrInspectionNotFound) {
			return nil, ErrInspectionNotFound
		}
		return nil, err
	}
	return record, nil
}

// ==================== 提交抽查结果 ====================

// ReviewInspection 提交抽查结果
//
// passed: 抽查通过，教案保持当前状态
// revoked: 发现问题，将教案从 published_shared 退回 revision
func (s *InspectionService) ReviewInspection(ctx context.Context, id string, inspectorID string, req *models.InspectionReviewRequest) error {
	// 1. 查询抽查记录
	record, err := repository.GetInspectionByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrInspectionNotFound) {
			return ErrInspectionNotFound
		}
		return err
	}

	// 2. 状态校验
	if record.Status == models.InspectionStatusPassed || record.Status == models.InspectionStatusRevoked {
		return ErrInspectionAlreadyDone
	}

	// 3. 权限校验：必须是分配的审查员
	if record.InspectorID == nil || *record.InspectorID != inspectorID {
		return ErrInspectionNoPermission
	}

	// 4. 决策校验
	if req.Decision != "passed" && req.Decision != "revoked" {
		return ErrInspectionInvalidDecision
	}

	// 5. 更新抽查记录
	newStatus := models.InspectionStatusPassed
	if req.Decision == "revoked" {
		newStatus = models.InspectionStatusRevoked
	}
	if err := repository.UpdateInspectionStatus(ctx, id, newStatus, req.Comment); err != nil {
		return err
	}

	// 6. 如果撤回，需要更新教案状态
	if req.Decision == "revoked" {
		// 将教案状态改为 under_inspection → revision
		_ = repository.UpdateLessonPlanStatus(ctx, record.LessonPlanID, models.LPStatusRevision)
		_ = repository.UpdateLessonPlanReviewLevel(ctx, record.LessonPlanID, 0, nil)

		// 写入L3审核记录
		reviewRecord := &models.ReviewV2{
			LessonPlanID: record.LessonPlanID,
			ReviewLevel:  models.ReviewLevelL3,
			ReviewerID:   inspectorID,
			Decision:     models.ReviewDecisionRevoked,
			Comment:      req.Comment,
			ReviewRound:  1,
		}
		_ = repository.CreateReviewV2(ctx, reviewRecord)

		inspLog.Info("抽查撤回教案", "inspection_id", id, "plan_id", record.LessonPlanID, "inspector", inspectorID)
	} else {
		inspLog.Info("抽查通过", "inspection_id", id, "plan_id", record.LessonPlanID, "inspector", inspectorID)
	}

	return nil
}

// ==================== 分配审查员 ====================

// AssignInspector 分配审查员到抽查记录
func (s *InspectionService) AssignInspector(ctx context.Context, id string, inspectorID string) error {
	record, err := repository.GetInspectionByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrInspectionNotFound) {
			return ErrInspectionNotFound
		}
		return err
	}

	// 只允许在 pending 状态分配
	if record.Status != models.InspectionStatusPending {
		return errors.New("只有待分配状态的抽查记录可以分配审查员")
	}

	return repository.AssignInspector(ctx, id, inspectorID)
}

// ==================== 手动触发抽样 ====================

// BatchSample 手动触发抽样
// 从指定学校（或全部学校）的已发布教案中按比例随机抽取
func (s *InspectionService) BatchSample(ctx context.Context, req *models.BatchSampleRequest) (int, error) {
	batchID := fmt.Sprintf("manual_%s", time.Now().Format("20060102_150405"))

	if req.SchoolID != "" {
		// 指定学校
		sampleRate := req.SampleRate
		if sampleRate <= 0 {
			// 从配置读取
			cfg, err := repository.GetReviewFlowConfig(ctx, req.SchoolID)
			if err == nil {
				sampleRate = cfg.L3SampleRate
			} else {
				sampleRate = 0.20
			}
		}
		count, err := repository.SamplePublishedPlansForInspection(ctx, req.SchoolID, sampleRate, batchID)
		if err != nil {
			return 0, fmt.Errorf("抽样失败: %w", err)
		}
		inspLog.Info("手动抽样完成", "school_id", req.SchoolID, "sample_rate", sampleRate, "count", count, "batch", batchID)
		return count, nil
	}

	// 全部学校：遍历所有有 review_flow_configs 的学校
	// 简化实现：查所有 published_shared 教案的 review_school_id 去重
	totalSampled := 0
	sampleRate := req.SampleRate
	if sampleRate <= 0 {
		sampleRate = 0.20
	}

	// 获取所有有已发布教案的学校ID
	rows, err := repository.QueryDistinctReviewSchoolIDs(ctx)
	if err != nil {
		return 0, err
	}
	for _, schoolID := range rows {
		count, err := repository.SamplePublishedPlansForInspection(ctx, schoolID, sampleRate, batchID)
		if err != nil {
			inspLog.Error("学校抽样失败", "school_id", schoolID, "error", err)
			continue
		}
		totalSampled += count
	}

	inspLog.Info("全量抽样完成", "total_sampled", totalSampled, "batch", batchID)
	return totalSampled, nil
}

// ==================== 抽查统计 ====================

// GetInspectionStats 获取抽查统计
func (s *InspectionService) GetInspectionStats(ctx context.Context, inspectorID string) (*models.InspectionStatsResponse, error) {
	return repository.GetInspectionStats(ctx, inspectorID)
}

// ==================== 区域教研员管理 ====================

// ListDistrictInspectors 获取区域教研员列表
func (s *InspectionService) ListDistrictInspectors(ctx context.Context, regionID string) ([]*models.DistrictInspectorListItem, error) {
	return repository.ListDistrictInspectors(ctx, regionID)
}

// CreateDistrictInspector 分配教研员到区域
func (s *InspectionService) CreateDistrictInspector(ctx context.Context, req *models.CreateDistrictInspectorRequest) (*models.DistrictInspectorAssignment, error) {
	if req.InspectorID == "" || req.RegionID == "" {
		return nil, errors.New("教研员ID和区域ID不能为空")
	}
	assign := &models.DistrictInspectorAssignment{
		InspectorID: req.InspectorID,
		RegionID:    req.RegionID,
	}
	if err := repository.CreateDistrictInspectorAssignment(ctx, assign); err != nil {
		return nil, err
	}
	return assign, nil
}

// DeleteDistrictInspector 取消区域教研员分配
func (s *InspectionService) DeleteDistrictInspector(ctx context.Context, id string) error {
	return repository.DeleteDistrictInspectorAssignment(ctx, id)
}
