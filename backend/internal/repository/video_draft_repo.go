package repository

// video_draft_repo.go — 视频编辑器草稿数据访问(v0.42.5)
//
// 功能: 草稿CRUD + 数量统计 + 自动清理最旧记录
// 表: video_editor_drafts (courseware_id+user_id+created_at DESC索引)

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"tedna/internal/database"
)

// VideoDraftItem 视频编辑器草稿列表项
type VideoDraftItem struct {
	ID           string          `json:"id"`
	CoursewareID string          `json:"courseware_id"`
	UserID       string          `json:"user_id"`
	Name         string          `json:"name"`
	ClipsData    json.RawMessage `json:"clips_data"`
	ClipCount    int             `json:"clip_count"`
	CreatedAt    time.Time       `json:"created_at"`
}

// CreateVideoDraft 创建草稿记录
func CreateVideoDraft(ctx context.Context, coursewareID, userID, name, clipsJSON string, clipCount int) (string, time.Time, error) {
	var id string
	var createdAt time.Time
	sql := `INSERT INTO video_editor_drafts (id, courseware_id, user_id, name, clips_data, clip_count)
	        VALUES (gen_random_uuid(), $1, $2, $3, $4::jsonb, $5)
	        RETURNING id, created_at`
	err := database.DB.QueryRow(ctx, sql, coursewareID, userID, name, clipsJSON, clipCount).Scan(&id, &createdAt)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("创建草稿失败: %w", err)
	}
	return id, createdAt, nil
}

// ListVideoDrafts 获取指定课件+用户的草稿列表(最新在前,最多20条)
func ListVideoDrafts(ctx context.Context, coursewareID, userID string) ([]*VideoDraftItem, error) {
	sql := `SELECT id, courseware_id, user_id, COALESCE(name,''), clips_data, clip_count, created_at
	        FROM video_editor_drafts
	        WHERE courseware_id = $1 AND user_id = $2
	        ORDER BY created_at DESC LIMIT 20`
	rows, err := database.DB.Query(ctx, sql, coursewareID, userID)
	if err != nil {
		return nil, fmt.Errorf("查询草稿列表失败: %w", err)
	}
	defer rows.Close()

	var drafts []*VideoDraftItem
	for rows.Next() {
		d := &VideoDraftItem{}
		if err := rows.Scan(&d.ID, &d.CoursewareID, &d.UserID, &d.Name, &d.ClipsData, &d.ClipCount, &d.CreatedAt); err != nil {
			return nil, fmt.Errorf("扫描草稿行失败: %w", err)
		}
		drafts = append(drafts, d)
	}
	if drafts == nil {
		drafts = []*VideoDraftItem{}
	}
	return drafts, nil
}

// DeleteVideoDraft 删除草稿(限制只能删除自己的)
func DeleteVideoDraft(ctx context.Context, draftID, userID string) error {
	sql := `DELETE FROM video_editor_drafts WHERE id = $1 AND user_id = $2`
	tag, err := database.DB.Exec(ctx, sql, draftID, userID)
	if err != nil {
		return fmt.Errorf("删除草稿失败: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("草稿不存在或无权限删除")
	}
	return nil
}

// CountVideoDrafts 统计草稿数量
func CountVideoDrafts(ctx context.Context, coursewareID, userID string) (int, error) {
	var count int
	err := database.DB.QueryRow(ctx,
		`SELECT COUNT(*) FROM video_editor_drafts WHERE courseware_id = $1 AND user_id = $2`,
		coursewareID, userID).Scan(&count)
	return count, err
}

// DeleteOldestVideoDraft 删除最旧的一条草稿(超出限制时调用)
func DeleteOldestVideoDraft(ctx context.Context, coursewareID, userID string) error {
	sql := `DELETE FROM video_editor_drafts WHERE id = (
	            SELECT id FROM video_editor_drafts
	            WHERE courseware_id = $1 AND user_id = $2
	            ORDER BY created_at ASC LIMIT 1)`
	_, err := database.DB.Exec(ctx, sql, coursewareID, userID)
	return err
}
