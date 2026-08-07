package services

// lesson_plan_version_service.go — 教案正文版本历史业务层
//
// 权限规则：
//   - 版本列表、详情和恢复目前只允许教案作者本人操作；
//   - submitted、developing、completed等锁定状态不允许恢复；
//   - 普通教案恢复仍保存当前正文快照并递增版本；
//   - Word保真教案必须同时恢复对应的不可变DOCX、结构、图片和语义正文；
//   - 当前Word即使已经stale，也只能通过可信历史Word快照恢复为active；
//   - 恢复不会回退教案审核或发布状态。

import (
	"context"
	"errors"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// ErrLPVersionNotFound 教案历史版本不存在。
var ErrLPVersionNotFound = errors.New(
	"教案历史版本不存在",
)

// ListContentVersions 查询作者自己的教案版本历史。
func (s *LessonPlanService) ListContentVersions(
	ctx context.Context,
	planID string,
	callerID string,
	limit int,
	offset int,
) (*models.LessonPlanContentVersionListResponse, error) {
	plan, err :=
		repository.GetLessonPlanByID(
			ctx,
			planID,
		)
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
		if errors.Is(
			err,
			repository.ErrLessonPlanNotFound,
		) {
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
	plan, err :=
		repository.GetLessonPlanByID(
			ctx,
			planID,
		)
	if err != nil {
		return nil, s.mapNotFoundErr(err)
	}
	if plan.AuthorID != callerID {
		return nil, ErrLPNotAuthor
	}

	version, err :=
		repository.GetLessonPlanContentVersion(
			ctx,
			planID,
			versionID,
		)
	if err != nil {
		if errors.Is(
			err,
			repository.ErrLessonPlanVersionNotFound,
		) {
			return nil, ErrLPVersionNotFound
		}
		return nil, err
	}

	return version, nil
}

// RestoreContentVersion 恢复指定历史版本。
//
// 普通教案：
//
//	当前v6 → 恢复历史v3，会保存v6快照并生成新的v7。
//
// Word保真教案：
//
//	同时选择与历史v3正文对应的不可变DOCX，复制为新的Word版本，
//	再在同一数据库事务中生成新的正文版本和Word版本。
//	因此页面正文、下载DOCX、图片、表格和原版式恢复到同一版本边界。
func (s *LessonPlanService) RestoreContentVersion(
	ctx context.Context,
	planID string,
	versionID string,
	callerID string,
) (*models.LessonPlanContentRestoreResponse, error) {
	plan, err :=
		repository.GetLessonPlanByID(
			ctx,
			planID,
		)
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

	target, err :=
		repository.GetLessonPlanContentVersion(
			ctx,
			planID,
			versionID,
		)
	if err != nil {
		if errors.Is(
			err,
			repository.ErrLessonPlanVersionNotFound,
		) {
			return nil, ErrLPVersionNotFound
		}
		return nil, err
	}

	result, err :=
		RestoreLessonPlanContentVersionPreservingWord(
			ctx,
			plan,
			target,
			callerID,
		)
	if err != nil {
		return nil, s.mapNotFoundErr(err)
	}

	refreshed, err :=
		repository.GetLessonPlanByID(
			ctx,
			planID,
		)
	if err != nil {
		return nil, s.mapNotFoundErr(err)
	}

	lpLog.Info(
		"恢复教案正文历史版本",
		"plan_id", planID,
		"restored_from_version", target.VersionNumber,
		"current_version", refreshed.Version,
		"changed", result.Changed,
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
