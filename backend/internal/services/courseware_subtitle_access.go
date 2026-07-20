package services

// courseware_subtitle_access.go — 课件字幕查看、文本编辑与Scope归属治理
//
// 权限语义：
//
//   - 列表读取、SRT导出：课件可信查看权；
//   - 字幕文本新增、修改、删除：教研微调权；
//   - TTS生成、字幕烧录：由作者专属控制链单独治理。
//
// Scope兼容规则：
//
//   - page：必须提供当前课件中的真实page_id；
//   - video_asset：必须提供当前课件中的真实视频asset_id；
//   - editor_draft：现有前端允许不传scope_id，因此继续兼容空值；
//   - editor_draft提供scope_id时，草稿必须属于当前课件和当前操作者；
//   - editor_draft列表和删除继续按个人草稿边界收窄，不能串读或删除他人字幕。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

var (
	ErrCoursewareSubtitleNotFound = errors.New(
		"字幕不存在",
	)
	ErrCoursewareSubtitleInputInvalid = errors.New(
		"字幕参数无效",
	)
	ErrCoursewareSubtitleScopeInvalid = errors.New(
		"字幕范围类型无效",
	)
	ErrCoursewareSubtitleScopeTargetRequired = errors.New(
		"字幕范围目标不能为空",
	)
	ErrCoursewareSubtitleScopeTargetMismatch = errors.New(
		"字幕范围目标不属于当前课件",
	)
	ErrCoursewareSubtitleMutationConflict = errors.New(
		"字幕已发生变化，请刷新后重试",
	)
)

// normalizeCoursewareSubtitleScope 规范化并校验字幕Scope形状。
func normalizeCoursewareSubtitleScope(
	scopeType string,
	scopeID *string,
) (
	string,
	*string,
	error,
) {
	normalizedType := strings.ToLower(
		strings.TrimSpace(scopeType),
	)

	var normalizedID *string

	if scopeID != nil {
		value := strings.TrimSpace(
			*scopeID,
		)
		if value != "" {
			normalizedID = &value
		}
	}

	switch normalizedType {
	case models.SubScopeVideoAsset,
		models.SubScopePage:
		if normalizedID == nil {
			return "",
				nil,
				ErrCoursewareSubtitleScopeTargetRequired
		}

	case models.SubScopeEditorDraft:
		// 兼容现有视频编辑器以courseware级字幕轨运行，
		// 尚未保存的编辑会话仍可不传scope_id。

	default:
		return "",
			nil,
			ErrCoursewareSubtitleScopeInvalid
	}

	return normalizedType,
		normalizedID,
		nil
}

// validateCoursewareSubtitleScopeTarget 校验Scope目标的真实课件归属。
func validateCoursewareSubtitleScopeTarget(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	scopeType string,
	scopeID *string,
) error {
	if actor == nil ||
		strings.TrimSpace(actor.UserID) == "" {
		return ErrCoursewareActorRequired
	}

	switch scopeType {
	case models.SubScopePage:
		if scopeID == nil {
			return ErrCoursewareSubtitleScopeTargetRequired
		}

		pages, err :=
			repository.ListCoursewarePages(
				ctx,
				coursewareID,
			)
		if err != nil {
			return fmt.Errorf(
				"读取课件页面失败: %w",
				err,
			)
		}

		for _, page := range pages {
			if page == nil {
				continue
			}

			if page.ID == *scopeID &&
				page.CoursewareID ==
					coursewareID {
				return nil
			}
		}

		return ErrCoursewareSubtitleScopeTargetMismatch

	case models.SubScopeVideoAsset:
		if scopeID == nil {
			return ErrCoursewareSubtitleScopeTargetRequired
		}

		asset, err :=
			repository.GetCWAssetByID(
				ctx,
				*scopeID,
			)
		if err != nil {
			return fmt.Errorf(
				"%w: %v",
				ErrCoursewareSubtitleScopeTargetMismatch,
				err,
			)
		}

		if asset.CoursewareID !=
			coursewareID ||
			asset.AssetType !=
				models.CWAssetTypeVideo {
			return ErrCoursewareSubtitleScopeTargetMismatch
		}

		return nil

	case models.SubScopeEditorDraft:
		if scopeID == nil {
			return nil
		}

		_, err :=
			repository.GetVideoDraftForCoursewareUser(
				ctx,
				coursewareID,
				*scopeID,
				actor.UserID,
			)
		if err != nil {
			if errors.Is(
				err,
				repository.ErrVideoDraftNotFound,
			) {
				return ErrCoursewareSubtitleScopeTargetMismatch
			}

			return fmt.Errorf(
				"读取视频草稿失败: %w",
				err,
			)
		}

		return nil

	default:
		return ErrCoursewareSubtitleScopeInvalid
	}
}

// ListSubtitles 按可信课件查看权读取字幕列表。
func (s *CoursewareSubtitleService) ListSubtitles(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	scopeType string,
	scopeID string,
) ([]*models.CoursewareSubtitle, error) {
	courseware, err :=
		(&CoursewareService{}).
			LoadCoursewareForView(
				ctx,
				coursewareID,
				actor,
			)
	if err != nil {
		return nil, err
	}

	normalizedType :=
		strings.ToLower(
			strings.TrimSpace(
				scopeType,
			),
		)
	normalizedID :=
		strings.TrimSpace(
			scopeID,
		)

	if normalizedType != "" {
		switch normalizedType {
		case models.SubScopeVideoAsset,
			models.SubScopeEditorDraft,
			models.SubScopePage:
		default:
			return nil,
				ErrCoursewareSubtitleScopeInvalid
		}
	}

	var items []*models.CoursewareSubtitle

	if normalizedType ==
		models.SubScopeEditorDraft {
		items, err =
			repository.ListCoursewareEditorDraftSubtitles(
				ctx,
				coursewareID,
				actor.UserID,
				normalizedID,
			)
	} else {
		items, err =
			repository.ListCoursewareSubtitles(
				ctx,
				coursewareID,
				normalizedType,
				normalizedID,
			)
	}
	if err != nil {
		return nil, err
	}

	// 未指定scope_type时会读取课件全部字幕，此时仍必须剔除
	// 其他用户的editor_draft个人字幕。
	if normalizedType == "" {
		items, err =
			filterCoursewareSubtitlesForActor(
				ctx,
				courseware,
				actor,
				items,
			)
		if err != nil {
			return nil, err
		}
	}

	if items == nil {
		items =
			[]*models.CoursewareSubtitle{}
	}

	return items, nil
}

// UpsertSubtitle 在教研微调权限下新增或更新字幕文本。
func (s *CoursewareSubtitleService) UpsertSubtitle(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	req *models.UpsertSubtitleRequest,
) (*models.CoursewareSubtitle, error) {
	_, scopedActor, err :=
		(&CoursewareService{}).
			LoadCoursewareForRefine(
				ctx,
				coursewareID,
				actor,
			)
	if err != nil {
		return nil, err
	}

	if req == nil {
		return nil,
			ErrCoursewareSubtitleInputInvalid
	}

	language :=
		strings.TrimSpace(
			req.Language,
		)
	segments :=
		strings.TrimSpace(
			req.Segments,
		)

	if language == "" ||
		segments == "" {
		return nil,
			fmt.Errorf(
				"%w: language和segments不能为空",
				ErrCoursewareSubtitleInputInvalid,
			)
	}

	var parsedSegments []models.SubtitleSegment

	if err := json.Unmarshal(
		[]byte(segments),
		&parsedSegments,
	); err != nil {
		return nil,
			fmt.Errorf(
				"%w: segments不是合法JSON数组",
				ErrCoursewareSubtitleInputInvalid,
			)
	}

	scopeType, scopeID, err :=
		normalizeCoursewareSubtitleScope(
			req.ScopeType,
			req.ScopeID,
		)
	if err != nil {
		return nil, err
	}

	if err :=
		validateCoursewareSubtitleScopeTarget(
			ctx,
			coursewareID,
			scopedActor,
			scopeType,
			scopeID,
		); err != nil {
		return nil, err
	}

	// 正式写字幕前再次授权，并重新验证Scope目标。
	_, latestActor, err :=
		(&CoursewareService{}).
			LoadCoursewareForRefine(
				ctx,
				coursewareID,
				scopedActor,
			)
	if err != nil {
		return nil, err
	}

	if err :=
		validateCoursewareSubtitleScopeTarget(
			ctx,
			coursewareID,
			latestActor,
			scopeType,
			scopeID,
		); err != nil {
		return nil, err
	}

	createdBy := latestActor.UserID

	subtitle := &models.CoursewareSubtitle{
		CoursewareID: coursewareID,
		ScopeType:    scopeType,
		ScopeID:      scopeID,
		Language:     language,
		Segments:     segments,
		StyleConfig:  req.StyleConfig,
		TTSConfig:    req.TTSConfig,
		CreatedBy:    &createdBy,
	}

	if scopeType ==
		models.SubScopeEditorDraft &&
		scopeID != nil {
		err = repository.
			UpsertCoursewareSubtitleForEditorDraft(
				ctx,
				subtitle,
				latestActor.UserID,
			)
	} else {
		err = repository.UpsertCoursewareSubtitle(
			ctx,
			subtitle,
		)
	}
	if err != nil {
		if errors.Is(
			err,
			repository.ErrVideoDraftNotFound,
		) {
			return nil,
				ErrCoursewareSubtitleScopeTargetMismatch
		}

		return nil,
			fmt.Errorf(
				"保存字幕失败: %w",
				err,
			)
	}

	return subtitle, nil
}

// DeleteSubtitle 在教研微调权限下删除当前课件的字幕。
func (s *CoursewareSubtitleService) DeleteSubtitle(
	ctx context.Context,
	coursewareID string,
	subtitleID string,
	actor *CoursewareActorContext,
) error {
	courseware, scopedActor, err :=
		(&CoursewareService{}).
			LoadCoursewareForRefine(
				ctx,
				coursewareID,
				actor,
			)
	if err != nil {
		return err
	}

	subtitle, err :=
		repository.GetCoursewareSubtitleForCourseware(
			ctx,
			coursewareID,
			subtitleID,
		)
	if err != nil {
		return fmt.Errorf(
			"%w: %v",
			ErrCoursewareSubtitleNotFound,
			err,
		)
	}

	if err :=
		validateCoursewareEditorDraftSubtitleOwner(
			ctx,
			courseware,
			scopedActor,
			subtitle,
		); err != nil {
		return err
	}

	// 正式删除前重新授权并重新绑定课件、字幕与个人草稿归属。
	latestCourseware, latestActor, err :=
		(&CoursewareService{}).
			LoadCoursewareForRefine(
				ctx,
				coursewareID,
				scopedActor,
			)
	if err != nil {
		return err
	}

	latestSubtitle, err :=
		repository.GetCoursewareSubtitleForCourseware(
			ctx,
			coursewareID,
			subtitleID,
		)
	if err != nil {
		return fmt.Errorf(
			"%w: %v",
			ErrCoursewareSubtitleNotFound,
			err,
		)
	}

	if err :=
		validateCoursewareEditorDraftSubtitleOwner(
			ctx,
			latestCourseware,
			latestActor,
			latestSubtitle,
		); err != nil {
		return err
	}

	return repository.
		DeleteCoursewareSubtitleForCourseware(
			ctx,
			coursewareID,
			subtitleID,
		)
}

// coursewareSubtitleRevisionUnchanged 判断字幕是否仍是同一数据库版本。
//
// TTS和字幕烧录使用本函数阻止长任务覆盖并发修改。
func coursewareSubtitleRevisionUnchanged(
	before *models.CoursewareSubtitle,
	after *models.CoursewareSubtitle,
) bool {
	if before == nil ||
		after == nil ||
		before.ID != after.ID ||
		before.CoursewareID !=
			after.CoursewareID {
		return false
	}

	switch {
	case before.UpdatedAt == nil &&
		after.UpdatedAt == nil:
		return true

	case before.UpdatedAt == nil ||
		after.UpdatedAt == nil:
		return false

	default:
		return before.UpdatedAt.Equal(
			*after.UpdatedAt,
		)
	}
}
