package repository

// courseware_subtitle_repo.go — 课件字幕轨数据访问
//
// v0.42.8 新增：CRUD + 按 scope 查询 + 按课件查询
// 表: courseware_subtitles (UNIQUE: courseware_id + scope_type + scope_id + language)

import (
	"context"
	"fmt"
	"time"

	"tedna/internal/database"
	"tedna/internal/models"
)

// ==================== 创建/更新（UPSERT） ====================

// UpsertCoursewareSubtitle 创建或更新字幕轨
// 按 UNIQUE(courseware_id, scope_type, scope_id, language) 做 UPSERT
func UpsertCoursewareSubtitle(ctx context.Context, sub *models.CoursewareSubtitle) error {
	query := `
		INSERT INTO courseware_subtitles (
			courseware_id, scope_type, scope_id, language,
			segments, style_config, tts_config, created_by, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (courseware_id, scope_type, scope_id, language)
		DO UPDATE SET
			segments = EXCLUDED.segments,
			style_config = EXCLUDED.style_config,
			tts_config = EXCLUDED.tts_config,
			updated_at = EXCLUDED.updated_at
		RETURNING id, created_at, updated_at
	`
	now := time.Now()
	return database.DB.QueryRow(ctx, query,
		sub.CoursewareID,
		sub.ScopeType,
		sub.ScopeID,
		sub.Language,
		sub.Segments,
		sub.StyleConfig,
		sub.TTSConfig,
		sub.CreatedBy,
		now,
	).Scan(&sub.ID, &sub.CreatedAt, &sub.UpdatedAt)
}

// ==================== 查询 ====================

// 统一的 SELECT 列（保持一致性）
const subtitleSelectColumns = `
	id, courseware_id, scope_type, scope_id, language,
	segments, style_config, tts_config,
	created_by, created_at, updated_at
`

// scanSubtitle 统一扫描行到模型
func scanSubtitle(scanner interface{ Scan(dest ...interface{}) error }) (*models.CoursewareSubtitle, error) {
	s := &models.CoursewareSubtitle{}
	err := scanner.Scan(
		&s.ID, &s.CoursewareID, &s.ScopeType, &s.ScopeID, &s.Language,
		&s.Segments, &s.StyleConfig, &s.TTSConfig,
		&s.CreatedBy, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// GetCoursewareSubtitleByID 按 ID 查询单条字幕轨
func GetCoursewareSubtitleByID(ctx context.Context, id string) (*models.CoursewareSubtitle, error) {
	query := fmt.Sprintf("SELECT %s FROM courseware_subtitles WHERE id = $1", subtitleSelectColumns)
	row := database.DB.QueryRow(ctx, query, id)
	return scanSubtitle(row)
}

// ListCoursewareSubtitles 按课件+范围查询字幕轨列表
// scopeType 和 scopeID 可选筛选
func ListCoursewareSubtitles(ctx context.Context, coursewareID, scopeType, scopeID string) ([]*models.CoursewareSubtitle, error) {
	query := fmt.Sprintf("SELECT %s FROM courseware_subtitles WHERE courseware_id = $1", subtitleSelectColumns)
	args := []interface{}{coursewareID}
	argIdx := 2

	if scopeType != "" {
		query += fmt.Sprintf(" AND scope_type = $%d", argIdx)
		args = append(args, scopeType)
		argIdx++
	}
	if scopeID != "" {
		query += fmt.Sprintf(" AND scope_id = $%d", argIdx)
		args = append(args, scopeID)
		argIdx++
	}

	query += " ORDER BY language ASC, created_at ASC"

	rows, err := database.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询字幕列表失败: %w", err)
	}
	defer rows.Close()

	var results []*models.CoursewareSubtitle
	for rows.Next() {
		s, err := scanSubtitle(rows)
		if err != nil {
			return nil, fmt.Errorf("扫描字幕行失败: %w", err)
		}
		results = append(results, s)
	}
	return results, nil
}

// ==================== 删除 ====================

// DeleteCoursewareSubtitle 按 ID 删除字幕轨
func DeleteCoursewareSubtitle(ctx context.Context, id string) error {
	tag, err := database.DB.Exec(ctx, "DELETE FROM courseware_subtitles WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("删除字幕失败: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("字幕不存在: %s", id)
	}
	return nil
}

// DeleteCoursewareSubtitlesByScope 按范围批量删除（如删除某草稿的所有字幕）
func DeleteCoursewareSubtitlesByScope(ctx context.Context, coursewareID, scopeType, scopeID string) error {
	_, err := database.DB.Exec(ctx,
		"DELETE FROM courseware_subtitles WHERE courseware_id = $1 AND scope_type = $2 AND scope_id = $3",
		coursewareID, scopeType, scopeID,
	)
	return err
}
