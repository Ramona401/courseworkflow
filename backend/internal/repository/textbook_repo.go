package repository

// textbook_repo.go — 课本页面图片数据访问层
//
// 迭代7新增：课本图片CRUD+搜索+OCR缓存回填+共享+使用计数

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"tedna/internal/database"
	"tedna/internal/models"
)

// ==================== 错误常量 ====================

var (
	ErrTextbookNotFound = errors.New("课本页面不存在")
)

// ==================== 创建 ====================

// CreateTextbookPage 创建课本页面记录
func CreateTextbookPage(ctx context.Context, page *models.TextbookPage) error {
	query := `
		INSERT INTO textbook_pages (
			subject, grade_range, textbook_name, chapter, page_number,
			file_name, file_path, file_size, mime_type,
			description, tags, scope, scope_ref_id, uploaded_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id, created_at, updated_at
	`
	tags := page.Tags
	if tags == "" {
		tags = "[]"
	}
	scope := page.Scope
	if scope == "" {
		scope = models.TextbookScopePersonal
	}

	err := database.DB.QueryRow(ctx, query,
		page.Subject, page.GradeRange, page.TextbookName, page.Chapter, page.PageNumber,
		page.FileName, page.FilePath, page.FileSize, page.MimeType,
		page.Description, tags, scope, page.ScopeRefID, page.UploadedBy,
	).Scan(&page.ID, &page.CreatedAt, &page.UpdatedAt)
	if err != nil {
		return fmt.Errorf("创建课本页面记录失败: %w", err)
	}
	return nil
}

// ==================== 查询 ====================

// GetTextbookPageByID 根据ID查询课本页面完整信息
func GetTextbookPageByID(ctx context.Context, id string) (*models.TextbookPage, error) {
	p := &models.TextbookPage{}
	query := `
		SELECT id, subject, grade_range, textbook_name, chapter, page_number,
		       file_name, file_path, file_size, mime_type,
		       COALESCE(ocr_text, ''), COALESCE(ocr_model, ''), ocr_at,
		       COALESCE(description, ''), COALESCE(tags::text, '[]'),
		       scope, scope_ref_id, uploaded_by, usage_count, status,
		       created_at, updated_at
		FROM textbook_pages WHERE id = $1
	`
	err := database.DB.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.Subject, &p.GradeRange, &p.TextbookName, &p.Chapter, &p.PageNumber,
		&p.FileName, &p.FilePath, &p.FileSize, &p.MimeType,
		&p.OCRText, &p.OCRModel, &p.OCRAt,
		&p.Description, &p.Tags,
		&p.Scope, &p.ScopeRefID, &p.UploadedBy, &p.UsageCount, &p.Status,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTextbookNotFound
		}
		return nil, fmt.Errorf("查询课本页面失败: %w", err)
	}
	return p, nil
}

// ==================== 列表查询 ====================

// ListTextbookPages 查询课本页面列表（支持多条件筛选）
// 查询逻辑：我上传的 OR 共享给我可见的
func ListTextbookPages(ctx context.Context, callerID string, subject string, gradeRange string, textbookName string, scope string, limit int, offset int) ([]*models.TextbookListItem, int, error) {
	if limit <= 0 {
		limit = 50
	}

	// 构建WHERE：我上传的 OR 共享的（scope为group/school/public）
	where := " WHERE tp.status = 'active' AND (tp.uploaded_by = $1 OR tp.scope IN ('group','school','public'))"
	args := []interface{}{callerID}
	argIdx := 2

	if subject != "" {
		where += fmt.Sprintf(" AND tp.subject = $%d", argIdx)
		args = append(args, subject)
		argIdx++
	}
	if gradeRange != "" {
		where += fmt.Sprintf(" AND tp.grade_range = $%d", argIdx)
		args = append(args, gradeRange)
		argIdx++
	}
	if textbookName != "" {
		where += fmt.Sprintf(" AND tp.textbook_name ILIKE '%%' || $%d || '%%'", argIdx)
		args = append(args, textbookName)
		argIdx++
	}
	if scope != "" {
		where += fmt.Sprintf(" AND tp.scope = $%d", argIdx)
		args = append(args, scope)
		argIdx++
	}

	// 查总数
	var total int
	countQuery := "SELECT COUNT(*) FROM textbook_pages tp" + where
	if err := database.DB.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("查询课本页面总数失败: %w", err)
	}

	// 查列表
	listQuery := fmt.Sprintf(`
		SELECT tp.id, tp.subject, tp.grade_range, tp.textbook_name, tp.chapter, tp.page_number,
		       tp.file_name, tp.file_size, tp.mime_type,
		       (tp.ocr_text IS NOT NULL AND tp.ocr_text != ''),
		       COALESCE(tp.description, ''), tp.scope,
		       tp.uploaded_by, COALESCE(u.display_name, u.username, ''),
		       tp.usage_count, tp.file_path, tp.created_at
		FROM textbook_pages tp
		LEFT JOIN users u ON u.id = tp.uploaded_by
		%s
		ORDER BY tp.textbook_name ASC, tp.page_number ASC, tp.created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := database.DB.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询课本页面列表失败: %w", err)
	}
	defer rows.Close()

	var items []*models.TextbookListItem
	for rows.Next() {
		item := &models.TextbookListItem{}
		var filePath string
		if err := rows.Scan(
			&item.ID, &item.Subject, &item.GradeRange, &item.TextbookName, &item.Chapter, &item.PageNumber,
			&item.FileName, &item.FileSize, &item.MimeType,
			&item.HasOCR, &item.Description, &item.Scope,
			&item.UploadedBy, &item.UploaderName,
			&item.UsageCount, &filePath, &item.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("扫描课本页面行失败: %w", err)
		}
		item.ScopeName = models.TextbookScopeNameMap[item.Scope]
		item.ImageURL = "/uploads/textbooks/" + filePath
		items = append(items, item)
	}
	if items == nil {
		items = []*models.TextbookListItem{}
	}
	return items, total, nil
}

// ==================== 更新 ====================

// UpdateTextbookPage 更新课本页面元数据
func UpdateTextbookPage(ctx context.Context, id string, req *models.UpdateTextbookRequest) error {
	tags := req.Tags
	if tags == "" {
		tags = "[]"
	}
	scope := req.Scope
	if scope == "" {
		scope = models.TextbookScopePersonal
	}
	var scopeRefID *string
	if req.ScopeRefID != "" {
		scopeRefID = &req.ScopeRefID
	}

	now := time.Now()
	result, err := database.DB.Exec(ctx, `
		UPDATE textbook_pages
		SET chapter = $1, page_number = $2, description = $3,
		    tags = $4, scope = $5, scope_ref_id = $6, updated_at = $7
		WHERE id = $8 AND status = 'active'
	`, req.Chapter, req.PageNumber, req.Description, tags, scope, scopeRefID, now, id)
	if err != nil {
		return fmt.Errorf("更新课本页面失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrTextbookNotFound
	}
	return nil
}

// ==================== 删除 ====================

// DeleteTextbookPage 软删除课本页面
func DeleteTextbookPage(ctx context.Context, id string) error {
	now := time.Now()
	result, err := database.DB.Exec(ctx,
		`UPDATE textbook_pages SET status = 'archived', updated_at = $1 WHERE id = $2`,
		now, id,
	)
	if err != nil {
		return fmt.Errorf("删除课本页面失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrTextbookNotFound
	}
	return nil
}

// ==================== OCR缓存回填 ====================

// UpdateTextbookOCR 回填AI识别的文字内容
func UpdateTextbookOCR(ctx context.Context, id string, ocrText string, ocrModel string) error {
	now := time.Now()
	result, err := database.DB.Exec(ctx, `
		UPDATE textbook_pages
		SET ocr_text = $1, ocr_model = $2, ocr_at = $3, updated_at = $3
		WHERE id = $4
	`, ocrText, ocrModel, now, id)
	if err != nil {
		return fmt.Errorf("更新OCR结果失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrTextbookNotFound
	}
	return nil
}

// ==================== 使用计数 ====================

// IncrementTextbookUsage 递增课本页面使用次数
func IncrementTextbookUsage(ctx context.Context, id string) error {
	_, err := database.DB.Exec(ctx,
		`UPDATE textbook_pages SET usage_count = usage_count + 1 WHERE id = $1`, id,
	)
	return err
}

// ==================== 批量查询（备课时用）====================

// GetTextbookPagesByIDs 批量按ID查询课本页面（备课时加载关联的课本图片）
func GetTextbookPagesByIDs(ctx context.Context, ids []string) ([]*models.TextbookPage, error) {
	if len(ids) == 0 {
		return []*models.TextbookPage{}, nil
	}
	query := `
		SELECT id, subject, grade_range, textbook_name, chapter, page_number,
		       file_name, file_path, file_size, mime_type,
		       COALESCE(ocr_text, ''), COALESCE(ocr_model, ''), ocr_at,
		       COALESCE(description, ''), COALESCE(tags::text, '[]'),
		       scope, scope_ref_id, uploaded_by, usage_count, status,
		       created_at, updated_at
		FROM textbook_pages
		WHERE id = ANY($1) AND status = 'active'
		ORDER BY textbook_name ASC, page_number ASC
	`
	rows, err := database.DB.Query(ctx, query, ids)
	if err != nil {
		return nil, fmt.Errorf("批量查询课本页面失败: %w", err)
	}
	defer rows.Close()

	var items []*models.TextbookPage
	for rows.Next() {
		p := &models.TextbookPage{}
		if err := rows.Scan(
			&p.ID, &p.Subject, &p.GradeRange, &p.TextbookName, &p.Chapter, &p.PageNumber,
			&p.FileName, &p.FilePath, &p.FileSize, &p.MimeType,
			&p.OCRText, &p.OCRModel, &p.OCRAt,
			&p.Description, &p.Tags,
			&p.Scope, &p.ScopeRefID, &p.UploadedBy, &p.UsageCount, &p.Status,
			&p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("扫描课本页面行失败: %w", err)
		}
		items = append(items, p)
	}
	if items == nil {
		items = []*models.TextbookPage{}
	}
	return items, nil
}
