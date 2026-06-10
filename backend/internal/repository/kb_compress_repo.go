package repository

// kb_compress_repo.go — 知识库压缩任务/单元数据访问层
//
// 对应两表：
//   kb_compress_jobs  压缩任务（一次上传一条）
//   kb_compress_items 压缩单元（中间真相层核心，半成品全程驻留）
//
// JSONB 列读写遵循同包 courseware_suggestion_repo.go 范式：
//   读 SELECT COALESCE(列::text,'')；写 nullIfEmptyJSON(空串→NULL)。
// draft_rounds 列 NOT NULL DEFAULT '[]'，故写时空串兜底为 '[]' 而非 NULL。
//
// pgx/v5 写法、错误风格对齐 curriculum_repo.go / organization_admin_repo.go。

import (
	"context"
	"fmt"

	"tedna/internal/database"
	"tedna/internal/models"
)

// ==================== Job（任务）====================

// jobSelectColumns 任务表统一列清单（16列）
const jobSelectColumns = `id, kind, batch_tag, COALESCE(source_file,''), compress_mode,
COALESCE(subject,''), COALESCE(publisher,''), grade_num, COALESCE(semester,''), unit_number,
status, total_items, done_items, created_by, created_at, updated_at`

// scanKBJob 统一扫描一行任务（与 jobSelectColumns 顺序严格对齐）
func scanKBJob(row interface {
	Scan(dest ...interface{}) error
}) (*models.KBCompressJob, error) {
	j := &models.KBCompressJob{}
	err := row.Scan(
		&j.ID, &j.Kind, &j.BatchTag, &j.SourceFile, &j.CompressMode,
		&j.Subject, &j.Publisher, &j.GradeNum, &j.Semester, &j.UnitNumber,
		&j.Status, &j.TotalItems, &j.DoneItems, &j.CreatedBy, &j.CreatedAt, &j.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return j, nil
}

// CreateKBJob 创建压缩任务，返回新任务 id
func CreateKBJob(ctx context.Context, job *models.KBCompressJob) (string, error) {
	var id string
	err := database.DB.QueryRow(ctx, `
		INSERT INTO kb_compress_jobs
		  (kind, batch_tag, source_file, compress_mode, subject, publisher, grade_num, semester, unit_number, status, total_items, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		RETURNING id
	`,
		job.Kind, job.BatchTag, nullIfEmptyStr(job.SourceFile), job.CompressMode,
		nullIfEmptyStr(job.Subject), nullIfEmptyStr(job.Publisher), job.GradeNum,
		nullIfEmptyStr(job.Semester), job.UnitNumber, job.Status, job.TotalItems, job.CreatedBy,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("创建压缩任务失败: %w", err)
	}
	return id, nil
}

// GetKBJobByID 按 id 查任务
func GetKBJobByID(ctx context.Context, id string) (*models.KBCompressJob, error) {
	sql := `SELECT ` + jobSelectColumns + ` FROM kb_compress_jobs WHERE id = $1`
	return scanKBJob(database.DB.QueryRow(ctx, sql, id))
}

// ListKBJobsByKind 按种类列出任务（curriculum/textbook），可选 batchTag 过滤；按创建时间倒序
func ListKBJobsByKind(ctx context.Context, kind string, batchTag string) ([]*models.KBCompressJob, error) {
	var rows interface {
		Next() bool
		Scan(dest ...interface{}) error
		Close()
	}
	var err error
	if batchTag != "" {
		rows, err = database.DB.Query(ctx,
			`SELECT `+jobSelectColumns+` FROM kb_compress_jobs WHERE kind = $1 AND batch_tag = $2 ORDER BY created_at DESC`,
			kind, batchTag)
	} else {
		rows, err = database.DB.Query(ctx,
			`SELECT `+jobSelectColumns+` FROM kb_compress_jobs WHERE kind = $1 ORDER BY created_at DESC`,
			kind)
	}
	if err != nil {
		return nil, fmt.Errorf("查询压缩任务列表失败: %w", err)
	}
	defer rows.Close()

	jobs := []*models.KBCompressJob{}
	for rows.Next() {
		j, scanErr := scanKBJob(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("扫描压缩任务行失败: %w", scanErr)
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}

// UpdateKBJobStatus 更新任务状态
func UpdateKBJobStatus(ctx context.Context, id string, status string) error {
	_, err := database.DB.Exec(ctx,
		`UPDATE kb_compress_jobs SET status = $1, updated_at = now() WHERE id = $2`,
		status, id)
	if err != nil {
		return fmt.Errorf("更新任务状态失败: %w", err)
	}
	return nil
}

// UpdateKBJobProgress 更新任务进度（done_items + 可选 status）
func UpdateKBJobProgress(ctx context.Context, id string, doneItems int, status string) error {
	_, err := database.DB.Exec(ctx,
		`UPDATE kb_compress_jobs SET done_items = $1, status = $2, updated_at = now() WHERE id = $3`,
		doneItems, status, id)
	if err != nil {
		return fmt.Errorf("更新任务进度失败: %w", err)
	}
	return nil
}

// ==================== Item（单元）====================

// itemSelectColumns 单元表统一列清单（21列；JSONB 用 ::text 读）
const itemSelectColumns = `id, job_id, kind, seq, COALESCE(source_excerpt,''), COALESCE(page_label,''),
COALESCE(draft_rounds::text,'[]'), COALESCE(confidence,''), COALESCE(arbitration::text,''),
COALESCE(final_line,''), review_status, reviewer_id, reviewed_at, COALESCE(review_note,''),
attempt_count, COALESCE(last_error,''), tokens_total, committed, committed_ref, created_at, updated_at`

// scanKBItem 统一扫描一行单元（与 itemSelectColumns 顺序严格对齐）
func scanKBItem(row interface {
	Scan(dest ...interface{}) error
}) (*models.KBCompressItem, error) {
	it := &models.KBCompressItem{}
	err := row.Scan(
		&it.ID, &it.JobID, &it.Kind, &it.Seq, &it.SourceExcerpt, &it.PageLabel,
		&it.DraftRounds, &it.Confidence, &it.Arbitration,
		&it.FinalLine, &it.ReviewStatus, &it.ReviewerID, &it.ReviewedAt, &it.ReviewNote,
		&it.AttemptCount, &it.LastError, &it.TokensTotal, &it.Committed, &it.CommittedRef,
		&it.CreatedAt, &it.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return it, nil
}

// CreateKBItem 创建一个待压缩单元，返回新单元 id（seq 由调用方递增传入）
func CreateKBItem(ctx context.Context, item *models.KBCompressItem) (string, error) {
	var id string
	err := database.DB.QueryRow(ctx, `
		INSERT INTO kb_compress_items (job_id, kind, seq, source_excerpt, page_label, review_status)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING id
	`,
		item.JobID, item.Kind, item.Seq, nullIfEmptyStr(item.SourceExcerpt),
		nullIfEmptyStr(item.PageLabel),
		defaultStr(item.ReviewStatus, models.KBReviewStatusPending),
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("创建压缩单元失败: %w", err)
	}
	return id, nil
}

// GetKBItemByID 按 id 查单元
func GetKBItemByID(ctx context.Context, id string) (*models.KBCompressItem, error) {
	sql := `SELECT ` + itemSelectColumns + ` FROM kb_compress_items WHERE id = $1`
	return scanKBItem(database.DB.QueryRow(ctx, sql, id))
}

// ListKBItemsByJob 列出某任务的全部单元（按 seq 升序）
func ListKBItemsByJob(ctx context.Context, jobID string) ([]*models.KBCompressItem, error) {
	sql := `SELECT ` + itemSelectColumns + ` FROM kb_compress_items WHERE job_id = $1 ORDER BY seq ASC`
	rows, err := database.DB.Query(ctx, sql, jobID)
	if err != nil {
		return nil, fmt.Errorf("查询任务单元列表失败: %w", err)
	}
	defer rows.Close()

	items := []*models.KBCompressItem{}
	for rows.Next() {
		it, scanErr := scanKBItem(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("扫描压缩单元行失败: %w", scanErr)
		}
		items = append(items, it)
	}
	return items, nil
}

// ListKBItemsByJobAndReviewStatus 列出某任务下指定审核状态的单元（审核队列用）
func ListKBItemsByJobAndReviewStatus(ctx context.Context, jobID string, reviewStatus string) ([]*models.KBCompressItem, error) {
	sql := `SELECT ` + itemSelectColumns + ` FROM kb_compress_items
	        WHERE job_id = $1 AND review_status = $2 ORDER BY seq ASC`
	rows, err := database.DB.Query(ctx, sql, jobID, reviewStatus)
	if err != nil {
		return nil, fmt.Errorf("按审核状态查询单元失败: %w", err)
	}
	defer rows.Close()

	items := []*models.KBCompressItem{}
	for rows.Next() {
		it, scanErr := scanKBItem(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("扫描压缩单元行失败: %w", scanErr)
		}
		items = append(items, it)
	}
	return items, nil
}

// UpdateKBItemCompressResult 写入压缩+仲裁结果（多轮草稿/置信/仲裁/审核状态/成本/错误）
// draftRoundsJSON 为多轮草稿 JSON（空串兜底为 '[]'）；arbitrationJSON 空串写 NULL。
func UpdateKBItemCompressResult(
	ctx context.Context, id string,
	draftRoundsJSON string, confidence string, arbitrationJSON string,
	finalLine string, reviewStatus string,
	attemptCount int, lastError string, tokensTotal int64,
) error {
	dr := draftRoundsJSON
	if dr == "" {
		dr = "[]"
	}
	_, err := database.DB.Exec(ctx, `
		UPDATE kb_compress_items SET
		  draft_rounds = $1::jsonb,
		  confidence = $2,
		  arbitration = $3,
		  final_line = $4,
		  review_status = $5,
		  attempt_count = $6,
		  last_error = $7,
		  tokens_total = $8,
		  updated_at = now()
		WHERE id = $9
	`,
		dr, nullIfEmptyStr(confidence), nullIfEmptyJSON(arbitrationJSON),
		nullIfEmptyStr(finalLine), reviewStatus,
		attemptCount, nullIfEmptyStr(lastError), tokensTotal, id,
	)
	if err != nil {
		return fmt.Errorf("更新单元压缩结果失败: %w", err)
	}
	return nil
}

// UpdateKBItemReview 写入人工审核结果（审核状态 + 最终采纳行 + 审核人/时间/意见）
func UpdateKBItemReview(
	ctx context.Context, id string,
	reviewStatus string, finalLine string, reviewerID string, reviewNote string,
) error {
	_, err := database.DB.Exec(ctx, `
		UPDATE kb_compress_items SET
		  review_status = $1,
		  final_line = $2,
		  reviewer_id = $3,
		  reviewed_at = now(),
		  review_note = $4,
		  updated_at = now()
		WHERE id = $5
	`,
		reviewStatus, nullIfEmptyStr(finalLine), nullIfEmptyStr(reviewerID),
		nullIfEmptyStr(reviewNote), id,
	)
	if err != nil {
		return fmt.Errorf("更新单元审核结果失败: %w", err)
	}
	return nil
}

// MarkKBItemCommitted 标记单元已 commit 到目标表，回填目标记录 id
func MarkKBItemCommitted(ctx context.Context, id string, committedRef string) error {
	_, err := database.DB.Exec(ctx, `
		UPDATE kb_compress_items SET committed = true, committed_ref = $1, updated_at = now()
		WHERE id = $2
	`, nullIfEmptyStr(committedRef), id)
	if err != nil {
		return fmt.Errorf("标记单元已入库失败: %w", err)
	}
	return nil
}

// ==================== 小工具（仅本包内复用；nullIfEmptyJSON 已在 courseware_asset_repo.go）====================

// nullIfEmptyStr 空串转 nil（写普通可空列用），非空原样返回
func nullIfEmptyStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// defaultStr 空串时返回默认值
func defaultStr(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
