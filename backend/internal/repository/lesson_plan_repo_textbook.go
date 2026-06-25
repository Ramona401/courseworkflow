package repository

// lesson_plan_repo_textbook.go — 教案课本关联数据访问（迭代3.5 A2-2 新增）
//
// 背景：对话模式「课本中途挂载」能力。引擎侧 LoadStagePromptContextV2 每轮对话
// 都会重新读取 lesson_plans.textbook_page_ids 并拼接课本 OCR 原文进系统提示词，
// 因此只需更新该列，下一轮对话即自动携带课本上下文——引擎零改动。
//
// 独立成文件的原因：lesson_plan_repo.go 已较大，按模块化纪律新功能落新文件。

import (
	"context"
	"fmt"
	"time"

	"tedna/internal/database"
)

// UpdateLessonPlanTextbookPages 更新教案的课本页面ID列表（jsonb 列，传 JSON 数组字符串）
//
// 范式与 UpdateLessonPlanStatus 等单列更新完全一致：
//   - 空字符串防御性归一为 "[]"（jsonb 列不接受空串）
//   - RowsAffected()==0 → ErrLessonPlanNotFound
func UpdateLessonPlanTextbookPages(ctx context.Context, id string, textbookPageIDs string) error {
	// 防御性编程：空字符串不是有效JSON，PostgreSQL的jsonb列会报错
	if textbookPageIDs == "" {
		textbookPageIDs = "[]"
	}
	now := time.Now()
	result, err := database.DB.Exec(ctx,
		`UPDATE lesson_plans SET textbook_page_ids = $1, updated_at = $2 WHERE id = $3`,
		textbookPageIDs, now, id,
	)
	if err != nil {
		return fmt.Errorf("更新教案课本关联失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrLessonPlanNotFound
	}
	return nil
}
