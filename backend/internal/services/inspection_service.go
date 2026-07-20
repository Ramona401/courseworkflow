package services

// inspection_service.go
//
// 区域抽查业务服务。
//
// 上下文 6：抽查候选改为同域候选。
// 候选教案必须满足：
//   - review_school_id 指向真实 active 学校；
//   - 学校教育域是 k12、vocational 或 adult；
//   - 教案教育域快照与学校教育域完全一致；
//   - 教案未软删除。
//
// 抽查记录列表、详情归属、结果提交和教研员分配逻辑保持不变。

import (
	"context"
	"errors"
	"fmt"
	"time"

	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
)

var (
	ErrInspectionNotFound = errors.New(
		"抽查记录不存在",
	)
	ErrInspectionNotAssigned = errors.New(
		"该抽查记录未分配审查员",
	)
	ErrInspectionAlreadyDone = errors.New(
		"该抽查记录已完成",
	)
	ErrInspectionNoPermission = errors.New(
		"您没有操作此抽查记录的权限",
	)
	ErrInspectionInvalidDecision = errors.New(
		"抽查决策无效，可选值：passed/revoked",
	)
)

var inspLog = logger.WithModule("inspection")

// InspectionService 抽查服务。
type InspectionService struct{}

// NewInspectionService 创建抽查服务。
func NewInspectionService() *InspectionService {
	return &InspectionService{}
}

// ListInspections 获取抽查列表。
func (s *InspectionService) ListInspections(
	ctx context.Context,
	inspectorID string,
	status string,
	limit int,
	offset int,
) (*models.InspectionListResponse, error) {
	items, total, err :=
		repository.ListInspections(
			ctx,
			inspectorID,
			status,
			limit,
			offset,
		)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items =
			[]*models.InspectionListItem{}
	}

	return &models.InspectionListResponse{
		Items: items,
		Total: total,
	}, nil
}

// GetInspection 获取抽查详情。
func (s *InspectionService) GetInspection(
	ctx context.Context,
	id string,
	callerRole string,
	callerID string,
) (*models.InspectionRecord, error) {
	record, err :=
		repository.GetInspectionByID(
			ctx,
			id,
		)
	if err != nil {
		if errors.Is(
			err,
			repository.ErrInspectionNotFound,
		) {
			return nil, ErrInspectionNotFound
		}
		return nil, err
	}

	if callerRole != models.RoleAdmin {
		if record.InspectorID == nil ||
			*record.InspectorID != callerID {
			inspLog.Warn(
				"抽查详情越权拦截",
				"inspection_id",
				id,
				"caller",
				callerID,
				"caller_role",
				callerRole,
			)
			return nil, ErrInspectionNoPermission
		}
	}

	return record, nil
}

// ReviewInspection 提交抽查结果。
func (s *InspectionService) ReviewInspection(
	ctx context.Context,
	id string,
	inspectorID string,
	req *models.InspectionReviewRequest,
) error {
	record, err :=
		repository.GetInspectionByID(
			ctx,
			id,
		)
	if err != nil {
		if errors.Is(
			err,
			repository.ErrInspectionNotFound,
		) {
			return ErrInspectionNotFound
		}
		return err
	}

	if record.Status ==
		models.InspectionStatusPassed ||
		record.Status ==
			models.InspectionStatusRevoked {
		return ErrInspectionAlreadyDone
	}

	if record.InspectorID == nil ||
		*record.InspectorID != inspectorID {
		return ErrInspectionNoPermission
	}

	if req == nil ||
		(req.Decision != "passed" &&
			req.Decision != "revoked") {
		return ErrInspectionInvalidDecision
	}

	newStatus :=
		models.InspectionStatusPassed
	if req.Decision == "revoked" {
		newStatus =
			models.InspectionStatusRevoked
	}

	if err := repository.UpdateInspectionStatus(
		ctx,
		id,
		newStatus,
		req.Comment,
	); err != nil {
		return err
	}

	if req.Decision == "revoked" {
		_ = repository.UpdateLessonPlanStatus(
			ctx,
			record.LessonPlanID,
			models.LPStatusRevision,
		)
		_ = repository.UpdateLessonPlanReviewLevel(
			ctx,
			record.LessonPlanID,
			0,
			nil,
		)

		reviewRecord := &models.ReviewV2{
			LessonPlanID: record.LessonPlanID,
			ReviewLevel:  models.ReviewLevelL3,
			ReviewerID:   inspectorID,
			Decision:     models.ReviewDecisionRevoked,
			Comment:      req.Comment,
			ReviewRound:  1,
		}
		_ = repository.CreateReviewV2(
			ctx,
			reviewRecord,
		)

		inspLog.Info(
			"抽查撤回教案",
			"inspection_id",
			id,
			"plan_id",
			record.LessonPlanID,
			"inspector",
			inspectorID,
		)
	} else {
		inspLog.Info(
			"抽查通过",
			"inspection_id",
			id,
			"plan_id",
			record.LessonPlanID,
			"inspector",
			inspectorID,
		)
	}

	return nil
}

// AssignInspector 分配审查员。
func (s *InspectionService) AssignInspector(
	ctx context.Context,
	id string,
	inspectorID string,
) error {
	record, err :=
		repository.GetInspectionByID(
			ctx,
			id,
		)
	if err != nil {
		if errors.Is(
			err,
			repository.ErrInspectionNotFound,
		) {
			return ErrInspectionNotFound
		}
		return err
	}

	if record.Status !=
		models.InspectionStatusPending {
		return errors.New(
			"只有待分配状态的抽查记录可以分配审查员",
		)
	}

	return repository.AssignInspector(
		ctx,
		id,
		inspectorID,
	)
}

// BatchSample 手动触发同域抽样。
func (s *InspectionService) BatchSample(
	ctx context.Context,
	req *models.BatchSampleRequest,
) (int, error) {
	if req == nil {
		return 0, errors.New(
			"抽样请求不能为空",
		)
	}

	batchID := fmt.Sprintf(
		"manual_%s",
		time.Now().Format(
			"20060102_150405",
		),
	)

	if req.SchoolID != "" {
		sampleRate := req.SampleRate
		if sampleRate <= 0 {
			config, err :=
				repository.GetReviewFlowConfig(
					ctx,
					req.SchoolID,
				)
			if err == nil {
				sampleRate =
					config.L3SampleRate
			} else {
				sampleRate = 0.20
			}
		}

		count, err :=
			repository.SamplePublishedPlansForInspectionSameDomain(
				ctx,
				req.SchoolID,
				sampleRate,
				batchID,
			)
		if err != nil {
			return 0, fmt.Errorf(
				"抽样失败: %w",
				err,
			)
		}

		inspLog.Info(
			"手动同域抽样完成",
			"school_id",
			req.SchoolID,
			"sample_rate",
			sampleRate,
			"count",
			count,
			"batch",
			batchID,
		)
		return count, nil
	}

	totalSampled := 0
	sampleRate := req.SampleRate
	if sampleRate <= 0 {
		sampleRate = 0.20
	}

	schoolIDs, err :=
		repository.QueryDistinctSameDomainReviewSchoolIDs(
			ctx,
		)
	if err != nil {
		return 0, err
	}

	for _, schoolID := range schoolIDs {
		count, sampleErr :=
			repository.SamplePublishedPlansForInspectionSameDomain(
				ctx,
				schoolID,
				sampleRate,
				batchID,
			)
		if sampleErr != nil {
			inspLog.Error(
				"学校同域抽样失败",
				"school_id",
				schoolID,
				"error",
				sampleErr,
			)
			continue
		}
		totalSampled += count
	}

	inspLog.Info(
		"全量同域抽样完成",
		"total_sampled",
		totalSampled,
		"batch",
		batchID,
	)

	return totalSampled, nil
}

// GetInspectionStats 获取抽查统计。
func (s *InspectionService) GetInspectionStats(
	ctx context.Context,
	inspectorID string,
) (*models.InspectionStatsResponse, error) {
	return repository.GetInspectionStats(
		ctx,
		inspectorID,
	)
}

// ListDistrictInspectors 获取区域教研员列表。
func (s *InspectionService) ListDistrictInspectors(
	ctx context.Context,
	regionID string,
) ([]*models.DistrictInspectorListItem, error) {
	return repository.ListDistrictInspectors(
		ctx,
		regionID,
	)
}

// CreateDistrictInspector 分配区域教研员。
func (s *InspectionService) CreateDistrictInspector(
	ctx context.Context,
	req *models.CreateDistrictInspectorRequest,
) (*models.DistrictInspectorAssignment, error) {
	if req == nil ||
		req.InspectorID == "" ||
		req.RegionID == "" {
		return nil, errors.New(
			"教研员ID和区域ID不能为空",
		)
	}

	assignment :=
		&models.DistrictInspectorAssignment{
			InspectorID: req.InspectorID,
			RegionID:    req.RegionID,
		}

	if err :=
		repository.CreateDistrictInspectorAssignment(
			ctx,
			assignment,
		); err != nil {
		return nil, err
	}

	return assignment, nil
}

// DeleteDistrictInspector 取消区域教研员分配。
func (s *InspectionService) DeleteDistrictInspector(
	ctx context.Context,
	id string,
) error {
	return repository.DeleteDistrictInspectorAssignment(
		ctx,
		id,
	)
}
