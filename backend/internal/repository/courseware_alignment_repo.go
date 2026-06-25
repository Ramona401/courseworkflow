package repository

// courseware_alignment_repo.go — 课件↔教案对齐报告数据访问
//
// 操作 courseware_alignment_reports 表（一课件一报告，UNIQUE(courseware_id)）。
// 报告由对齐校验服务异步生成：先 UpsertGeneratingReport 占位(status=generating)，
// AI 返回后 UpsertDoneReport 覆盖为完整结果；失败则 MarkReportFailed。
//
// report_json 为 jsonb 列：写入用 nullIfEmptyJSON（空串转 NULL，非空原样写）；
// 读出用 ::text 取原始 JSON 文本透传前端，不在后端反序列化（前端直接解析渲染）。

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

// 对齐报告统一 SELECT 列（report_json 以 ::text 读出原始 JSON 文本，空则返 '{}'）
const cwAlignmentSelectColumns = `
	id, courseware_id, COALESCE(lesson_plan_id::text, ''), overall, summary,
	COALESCE(report_json::text, '{}'), status, error_message,
	model_used, tokens_used, page_count, created_at, updated_at`

// scanCWAlignmentReport 统一扫描一行对齐报告
func scanCWAlignmentReport(row interface {
	Scan(dest ...interface{}) error
}) (*models.CoursewareAlignmentReport, error) {
	var r models.CoursewareAlignmentReport
	var lessonPlanID string
	if err := row.Scan(
		&r.ID, &r.CoursewareID, &lessonPlanID, &r.Overall, &r.Summary,
		&r.ReportJSON, &r.Status, &r.ErrorMessage,
		&r.ModelUsed, &r.TokensUsed, &r.PageCount, &r.CreatedAt, &r.UpdatedAt,
	); err != nil {
		return nil, err
	}
	// lesson_plan_id 空串归一为 nil 指针
	if lessonPlanID != "" {
		r.LessonPlanID = &lessonPlanID
	}
	return &r, nil
}

// GetAlignmentReportByCoursewareID 按课件ID取报告；无记录返回 (nil, nil)（非错误）
func GetAlignmentReportByCoursewareID(ctx context.Context, coursewareID string) (*models.CoursewareAlignmentReport, error) {
	row := database.DB.QueryRow(ctx,
		`SELECT `+cwAlignmentSelectColumns+`
		 FROM courseware_alignment_reports WHERE courseware_id = $1`, coursewareID)
	r, err := scanCWAlignmentReport(row)
	if err != nil {
		// 无记录：pgx 返回 ErrNoRows，统一翻译为 (nil, nil) 供上层判断"无报告"
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r, nil
}

// UpsertGeneratingReport 校验开始时占位：写入/覆盖为 status=generating
// 让前端进 Step1 立刻能看到"校验中"状态，而非空白。
func UpsertGeneratingReport(ctx context.Context, coursewareID string, lessonPlanID *string, pageCount int) error {
	_, err := database.DB.Exec(ctx, `
		INSERT INTO courseware_alignment_reports
			(courseware_id, lesson_plan_id, overall, summary, report_json,
			 status, error_message, model_used, tokens_used, page_count, created_at, updated_at)
		VALUES ($1, $2, 'aligned', '', '{}'::jsonb, 'generating', '', '', 0, $3, now(), now())
		ON CONFLICT (courseware_id) DO UPDATE SET
			lesson_plan_id = EXCLUDED.lesson_plan_id,
			status         = 'generating',
			error_message  = '',
			page_count     = EXCLUDED.page_count,
			updated_at     = now()`,
		coursewareID, lessonPlanID, pageCount)
	return err
}

// UpsertDoneReport 校验成功：覆盖为完整结果（status=done）
func UpsertDoneReport(ctx context.Context, coursewareID string, lessonPlanID *string,
	overall, summary, reportJSON, modelUsed string, tokensUsed, pageCount int) error {
	_, err := database.DB.Exec(ctx, `
		INSERT INTO courseware_alignment_reports
			(courseware_id, lesson_plan_id, overall, summary, report_json,
			 status, error_message, model_used, tokens_used, page_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, 'done', '', $6, $7, $8, now(), now())
		ON CONFLICT (courseware_id) DO UPDATE SET
			lesson_plan_id = EXCLUDED.lesson_plan_id,
			overall        = EXCLUDED.overall,
			summary        = EXCLUDED.summary,
			report_json    = EXCLUDED.report_json,
			status         = 'done',
			error_message  = '',
			model_used     = EXCLUDED.model_used,
			tokens_used    = EXCLUDED.tokens_used,
			page_count     = EXCLUDED.page_count,
			updated_at     = now()`,
		coursewareID, lessonPlanID, overall, summary, nullIfEmptyJSON(reportJSON),
		modelUsed, tokensUsed, pageCount)
	return err
}

// MarkReportFailed 校验失败：把报告标记为 failed 并记错误原因
// 若此前无记录（极少数情况），用 UPSERT 兜底建一条 failed 记录。
func MarkReportFailed(ctx context.Context, coursewareID string, lessonPlanID *string, errMsg string) error {
	_, err := database.DB.Exec(ctx, `
		INSERT INTO courseware_alignment_reports
			(courseware_id, lesson_plan_id, overall, summary, report_json,
			 status, error_message, model_used, tokens_used, page_count, created_at, updated_at)
		VALUES ($1, $2, 'failed', '', '{}'::jsonb, 'failed', $3, '', 0, 0, now(), now())
		ON CONFLICT (courseware_id) DO UPDATE SET
			status        = 'failed',
			overall       = 'failed',
			error_message = EXCLUDED.error_message,
			updated_at    = now()`,
		coursewareID, lessonPlanID, errMsg)
	return err
}
