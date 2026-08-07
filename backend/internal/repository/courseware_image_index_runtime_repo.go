package repository

// courseware_image_index_runtime_repo.go — 图片IAOCI运行期辅助仓储
//
// 当前只提供页面槽位变更后的陈旧索引处理。
// 页面重新生成HTML后，如果某些旧placeholder_id已经不存在，不能继续把旧索引
// 当作当前图片计划使用；但为保留审计记录，不物理删除，统一标记为stale。

import (
	"context"
	"fmt"

	"tedna/internal/database"
	"tedna/internal/models"
)

// MarkPageImageIndexesStaleExcept 把不在当前占位列表中的页面图片索引标为stale。
//
// allowedPlaceholderIDs为空时，表示本页当前没有有效图片占位，全部页面图片索引标为stale。
// 课程级@ANCHOR不受影响。
func MarkPageImageIndexesStaleExcept(
	ctx context.Context,
	pageID string,
	allowedPlaceholderIDs []string,
) error {
	if len(allowedPlaceholderIDs) == 0 {
		_, err := database.DB.Exec(
			ctx,
			`UPDATE courseware_image_indexes
SET status = $1,
	last_error = '页面当前无有效图片占位',
	updated_at = now()
WHERE page_id = $2
  AND index_type IN ('I', 'V')
  AND status <> $1`,
			models.CWImageIndexStatusStale,
			pageID,
		)
		if err != nil {
			return fmt.Errorf(
				"标记页面图片索引过期失败: %w",
				err,
			)
		}

		return nil
	}

	_, err := database.DB.Exec(
		ctx,
		`UPDATE courseware_image_indexes
SET status = $1,
	last_error = '页面HTML已不再包含该图片占位',
	updated_at = now()
WHERE page_id = $2
  AND index_type IN ('I', 'V')
  AND NOT (placeholder_id = ANY($3::text[]))
  AND status <> $1`,
		models.CWImageIndexStatusStale,
		pageID,
		allowedPlaceholderIDs,
	)
	if err != nil {
		return fmt.Errorf(
			"标记陈旧图片索引失败: %w",
			err,
		)
	}

	return nil
}
