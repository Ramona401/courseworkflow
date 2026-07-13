package services

// lesson_plan_version_service.go — 教案正文版本历史业务层
//
// 权限规则：
//   - 版本列表、详情和恢复目前只允许教案作者本人操作。
//   - submitted/developing/completed状态保持既有锁定规则，不允许恢复。
//   - 恢复历史版本时先由统一更新事务保存当前正文，恢复操作天然可逆。

import (
	"context"
	"errors"
	"fmt"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// ErrLPVersionNotFound 教案历史版本不存在。
var ErrLPVersionNotFound = errors.New("教案历史版本不存在")

// ListContentVersions 查询作者自己的教案版本历史。
func (s *LessonPlanService) ListContentVersions(
	ctx context.Context,
	planID string,
	callerID string,
	limit int,
	offset int,
) (*models.LessonPlanContentVersionListResponse, error) {
	plan, err := repository.GetLessonPlanByID(ctx, planID)
	if err != nil {
		return nil, s.mapNotFoundErr(err)
	}
	if plan.AuthorID != callerID {
		return nil, ErrLPNotAuthor
	}

	items, total, currentVersion, err :=
		repository.ListLessonPlanContentVersions(
			ctx,
			planID,
			limit,
			offset,
		)
	if err != nil {
		if errors.Is(err, repository.ErrLessonPlanNotFound) {
			return nil, ErrLPNotFound
		}
		return nil, err
	}

	return &models.LessonPlanContentVersionListResponse{
		Versions:       items,
		Total:          total,
		CurrentVersion: currentVersion,
	}, nil
}

// GetContentVersion 查询一个完整历史版本。
func (s *LessonPlanService) GetContentVersion(
	ctx context.Context,
	planID string,
	versionID string,
	callerID string,
) (*models.LessonPlanContentVersion, error) {
	plan, err := repository.GetLessonPlanByID(ctx, planID)
	if err != nil {
		return nil, s.mapNotFoundErr(err)
	}
	if plan.AuthorID != callerID {
		return nil, ErrLPNotAuthor
	}

	version, err := repository.GetLessonPlanContentVersion(
		ctx,
		planID,
		versionID,
	)
	if err != nil {
		if errors.Is(err, repository.ErrLessonPlanVersionNotFound) {
			return nil, ErrLPVersionNotFound
		}
		return nil, err
	}

	return version, nil
}

// RestoreContentVersion 恢复指定历史版本。
//
// UpdateLessonPlanContent在覆盖前会保存当前正文，因此：
//
//	当前v6 → 恢复历史v3
//
// 会先把当前v6保存为历史快照，再把v3正文写成新的v7。
// 老师之后仍可从历史记录恢复到刚才的v6，不会形成不可逆覆盖。
func (s *LessonPlanService) RestoreContentVersion(
	ctx context.Context,
	planID string,
	versionID string,
	callerID string,
) (*models.LessonPlanContentRestoreResponse, error) {
	plan, err := repository.GetLessonPlanByID(ctx, planID)
	if err != nil {
		return nil, s.mapNotFoundErr(err)
	}
	if plan.AuthorID != callerID {
		return nil, ErrLPNotAuthor
	}

	editableStatuses := map[string]bool{
		models.LPStatusDraft:             true,
		models.LPStatusPublishedPersonal: true,
		models.LPStatusRevision:          true,
		models.LPStatusApproved:          true,
		models.LPStatusPublishedShared:   true,
	}
	if !editableStatuses[plan.Status] {
		return nil, ErrLPCannotEdit
	}

	target, err := repository.GetLessonPlanContentVersion(
		ctx,
		planID,
		versionID,
	)
	if err != nil {
		if errors.Is(err, repository.ErrLessonPlanVersionNotFound) {
			return nil, ErrLPVersionNotFound
		}
		return nil, err
	}

	summary := fmt.Sprintf(
		"恢复历史版本 v%d",
		target.VersionNumber,
	)

	if err := repository.UpdateLessonPlanContent(
		ctx,
		planID,
		target.Title,
		target.ContentMarkdown,
		target.ContentStructured,
		target.DurationMinutes,
		models.LessonPlanVersionMeta{
			ChangeSource:  models.LPVersionSourceRestore,
			ChangedBy:     &callerID,
			ChangeSummary: summary,
		},
	); err != nil {
		return nil, s.mapNotFoundErr(err)
	}

	refreshed, err := repository.GetLessonPlanByID(ctx, planID)
	if err != nil {
		return nil, s.mapNotFoundErr(err)
	}

	lpLog.Info(
		"恢复教案正文历史版本",
		"plan_id", planID,
		"restored_from_version", target.VersionNumber,
		"current_version", refreshed.Version,
		"caller", callerID,
	)

	return &models.LessonPlanContentRestoreResponse{
		RestoredFromVersion: target.VersionNumber,
		CurrentVersion:      refreshed.Version,
		Title:               refreshed.Title,
		ContentMarkdown:     refreshed.ContentMarkdown,
		DurationMinutes:     refreshed.DurationMinutes,
	}, nil
}
