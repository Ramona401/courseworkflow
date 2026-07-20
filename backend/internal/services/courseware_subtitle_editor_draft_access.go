package services

// courseware_subtitle_editor_draft_access.go — 编辑器草稿字幕个人归属辅助
//
// 本文件从courseware_subtitle_access.go拆分，专门处理editor_draft：
//   - 真实draft_id字幕按courseware_id + draft_id + user_id复合归属；
//   - 历史空scope_id字幕按created_by归属；
//   - created_by为空的最早期遗留字幕只允许课件作者本人继续管理；
//   - 通用字幕列表未指定scope_type时，剔除他人的个人草稿字幕。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// validateCoursewareEditorDraftSubtitleOwner 校验editor_draft字幕属于当前用户。
func validateCoursewareEditorDraftSubtitleOwner(
	ctx context.Context,
	courseware *models.Courseware,
	actor *CoursewareActorContext,
	subtitle *models.CoursewareSubtitle,
) error {
	if subtitle == nil ||
		subtitle.ScopeType !=
			models.SubScopeEditorDraft {
		return nil
	}

	if actor == nil ||
		strings.TrimSpace(actor.UserID) == "" {
		return ErrCoursewareActorRequired
	}

	if subtitle.ScopeID != nil {
		draftID := strings.TrimSpace(
			*subtitle.ScopeID,
		)
		if draftID != "" {
			_, err :=
				repository.GetVideoDraftForCoursewareUser(
					ctx,
					subtitle.CoursewareID,
					draftID,
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
		}
	}

	createdBy := ""
	if subtitle.CreatedBy != nil {
		createdBy = strings.TrimSpace(
			*subtitle.CreatedBy,
		)
	}

	if createdBy == actor.UserID {
		return nil
	}

	if createdBy == "" &&
		courseware != nil &&
		courseware.UserID == actor.UserID {
		return nil
	}

	return ErrCoursewareSubtitleScopeTargetMismatch
}

// filterCoursewareSubtitlesForActor 从通用列表中剔除他人的editor_draft字幕。
func filterCoursewareSubtitlesForActor(
	ctx context.Context,
	courseware *models.Courseware,
	actor *CoursewareActorContext,
	items []*models.CoursewareSubtitle,
) (
	[]*models.CoursewareSubtitle,
	error,
) {
	filtered := make(
		[]*models.CoursewareSubtitle,
		0,
		len(items),
	)

	for _, item := range items {
		if item == nil {
			continue
		}

		if item.ScopeType !=
			models.SubScopeEditorDraft {
			filtered = append(
				filtered,
				item,
			)
			continue
		}

		err := validateCoursewareEditorDraftSubtitleOwner(
			ctx,
			courseware,
			actor,
			item,
		)
		if err == nil {
			filtered = append(
				filtered,
				item,
			)
			continue
		}

		if errors.Is(
			err,
			ErrCoursewareSubtitleScopeTargetMismatch,
		) {
			continue
		}

		return nil, err
	}

	return filtered, nil
}
