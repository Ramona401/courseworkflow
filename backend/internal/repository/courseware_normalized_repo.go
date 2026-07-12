package repository

// courseware_normalized_repo.go — 教案规整缓存表 courseware_normalized_lessons 数据访问
//
// 【表职责】一课件一条规整结果（UNIQUE courseware_id），随课件删除 CASCADE 清理。
//   覆盖 lesson_plan 与 doc_upload 两种有原文的来源；其余来源不产生本表记录。
//   范式对齐 courseware_alignment_repo.go（同为课件级缓存表的三态 UPSERT 风格）。
//
// 【提供的能力】
//   - GetNormalizedByCoursewareID：按课件取规整结果（供注入层"先查规整、无则退原文"）
//   - UpsertGeneratingNormalized ：占位为 generating（供规整开始前落一条，便于前端/排查看进度）
//   - UpsertDoneNormalized       ：写入规整成功结果（status=done + 正文 + 统计）
//   - MarkNormalizedFailed       ：标记规整失败（status=failed + 错误原因，不阻断下游）
//
// 所有 UPSERT 均 ON CONFLICT(courseware_id) DO UPDATE，重算即覆盖。

import (
	"context"
	"fmt"

	"tedna/internal/database"
	"tedna/internal/models"
)

// GetNormalizedByCoursewareID 按课件ID取规整缓存记录。
//   无记录时返回 (nil, nil)，供注入层判断"尚无规整结果 → 退回原文"，非错误。
//   仅在 DB 异常时返回 error。
func GetNormalizedByCoursewareID(ctx context.Context, coursewareID string) (*models.CoursewareNormalizedLesson, error) {
	row := database.DB.QueryRow(ctx,
		`SELECT id, courseware_id, source_type, source_ref, normalized_content,
		        status, error_message, model_used, tokens_used,
		        raw_char_count, norm_char_count, created_at, updated_at
		   FROM courseware_normalized_lessons
		  WHERE courseware_id = $1`, coursewareID)

	n := &models.CoursewareNormalizedLesson{}
	err := row.Scan(
		&n.ID, &n.CoursewareID, &n.SourceType, &n.SourceRef, &n.NormalizedContent,
		&n.Status, &n.ErrorMessage, &n.ModelUsed, &n.TokensUsed,
		&n.RawCharCount, &n.NormCharCount, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		// pgx 在无行时返回 ErrNoRows；本层统一转 (nil, nil) 表示"无缓存"
		return nil, nil
	}
	return n, nil
}

// UpsertGeneratingNormalized 在规整开始前落一条 generating 占位记录（便于排查/看进度）。
//   已有记录则更新为 generating 并清空旧的成功正文与错误（重算语义）。
func UpsertGeneratingNormalized(ctx context.Context, coursewareID, sourceType, sourceRef string, rawCharCount int) error {
	_, err := database.DB.Exec(ctx,
		`INSERT INTO courseware_normalized_lessons
		   (id, courseware_id, source_type, source_ref, normalized_content,
		    status, error_message, model_used, tokens_used, raw_char_count, norm_char_count, updated_at)
		 VALUES
		   (gen_random_uuid(), $1, $2, $3, '',
		    'generating', '', '', 0, $4, 0, now())
		 ON CONFLICT (courseware_id) DO UPDATE
		 SET source_type     = EXCLUDED.source_type,
		     source_ref      = EXCLUDED.source_ref,
		     status          = 'generating',
		     error_message   = '',
		     raw_char_count  = EXCLUDED.raw_char_count,
		     updated_at      = now()`,
		coursewareID, sourceType, sourceRef, rawCharCount)
	if err != nil {
		return fmt.Errorf("写入规整占位记录失败: %w", err)
	}
	return nil
}

// UpsertDoneNormalized 写入规整成功结果（status=done）。
func UpsertDoneNormalized(ctx context.Context, coursewareID, sourceType, sourceRef, normalizedContent, modelUsed string, tokensUsed, rawCharCount, normCharCount int) error {
	_, err := database.DB.Exec(ctx,
		`INSERT INTO courseware_normalized_lessons
		   (id, courseware_id, source_type, source_ref, normalized_content,
		    status, error_message, model_used, tokens_used, raw_char_count, norm_char_count, updated_at)
		 VALUES
		   (gen_random_uuid(), $1, $2, $3, $4,
		    'done', '', $5, $6, $7, $8, now())
		 ON CONFLICT (courseware_id) DO UPDATE
		 SET source_type        = EXCLUDED.source_type,
		     source_ref         = EXCLUDED.source_ref,
		     normalized_content = EXCLUDED.normalized_content,
		     status             = 'done',
		     error_message      = '',
		     model_used         = EXCLUDED.model_used,
		     tokens_used        = EXCLUDED.tokens_used,
		     raw_char_count     = EXCLUDED.raw_char_count,
		     norm_char_count    = EXCLUDED.norm_char_count,
		     updated_at         = now()`,
		coursewareID, sourceType, sourceRef, normalizedContent, modelUsed, tokensUsed, rawCharCount, normCharCount)
	if err != nil {
		return fmt.Errorf("写入规整成功结果失败: %w", err)
	}
	return nil
}

// MarkNormalizedFailed 标记规整失败（status=failed + 错误原因）。
//   规整失败不阻断下游生成，注入层遇 failed/无正文时自动退回原文。
func MarkNormalizedFailed(ctx context.Context, coursewareID, sourceType, sourceRef, errMsg string, rawCharCount int) error {
	// 错误信息防超长（text 列无长度限制，但截断避免异常长响应塞爆）
	if len([]rune(errMsg)) > 1000 {
		errMsg = string([]rune(errMsg)[:1000]) + "...(截断)"
	}
	_, err := database.DB.Exec(ctx,
		`INSERT INTO courseware_normalized_lessons
		   (id, courseware_id, source_type, source_ref, normalized_content,
		    status, error_message, model_used, tokens_used, raw_char_count, norm_char_count, updated_at)
		 VALUES
		   (gen_random_uuid(), $1, $2, $3, '',
		    'failed', $4, '', 0, $5, 0, now())
		 ON CONFLICT (courseware_id) DO UPDATE
		 SET source_type    = EXCLUDED.source_type,
		     source_ref     = EXCLUDED.source_ref,
		     status         = 'failed',
		     error_message  = EXCLUDED.error_message,
		     raw_char_count = EXCLUDED.raw_char_count,
		     updated_at     = now()`,
		coursewareID, sourceType, sourceRef, errMsg, rawCharCount)
	if err != nil {
		return fmt.Errorf("标记规整失败记录失败: %w", err)
	}
	return nil
}
