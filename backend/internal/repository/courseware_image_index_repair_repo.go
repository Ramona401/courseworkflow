package repository

// courseware_image_index_repair_repo.go — 图片索引安全重试的最小CAS写边界
//
// 本文件只服务“明确失败后的同槽位安全重试”：更新generation_prompt、清空失败状态并递增version。
// 稳定image_key、IAOCI语义本体、R关系和槽位身份一律不在此修改。version递增后，媒体计费自然获得
// 新的幂等键，避免复用已经failed/cancelled的旧计费终态。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
)

var ErrCoursewareImageIndexVersionConflict = errors.New("课件图片索引版本已经变化")

// UpdateCoursewareImageIndexGenerationPromptForRetry 为一次明确可重试的图片失败更新生成提示词。
func UpdateCoursewareImageIndexGenerationPromptForRetry(
	ctx context.Context,
	id string,
	expectedVersion int,
	generationPrompt string,
) (
	int,
	error,
) {
	id = strings.TrimSpace(id)
	generationPrompt = strings.TrimSpace(generationPrompt)

	if id == "" ||
		expectedVersion < 1 ||
		generationPrompt == "" {
		return 0, fmt.Errorf(
			"图片索引重试参数不完整",
		)
	}

	var newVersion int

	err := database.DB.QueryRow(
		ctx,
		`UPDATE courseware_image_indexes
SET
        generation_prompt = $1,
        asset_id = NULL,
        status = 'planned',
        last_error = '',
        version = version + 1,
        updated_at = now()
WHERE id = $2
  AND version = $3
RETURNING version`,
		generationPrompt,
		id,
		expectedVersion,
	).Scan(&newVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		current, currentErr :=
			GetCoursewareImageIndexByID(
				ctx,
				id,
			)
		if currentErr != nil {
			if errors.Is(
				currentErr,
				ErrCoursewareImageIndexNotFound,
			) {
				return 0,
					ErrCoursewareImageIndexNotFound
			}
			return 0, currentErr
		}
		if current.Version != expectedVersion {
			return 0,
				ErrCoursewareImageIndexVersionConflict
		}
		return 0,
			ErrCoursewareImageIndexNotFound
	}
	if err != nil {
		return 0, fmt.Errorf(
			"更新图片重试提示词失败: %w",
			err,
		)
	}

	return newVersion, nil
}
