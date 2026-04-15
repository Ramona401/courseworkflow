package repository

import (
	"context"
	"errors"
	"fmt"

	"tedna/internal/database"
	"tedna/internal/models"
)

// ==================== 提示词数据访问层 ====================

// 错误常量
var (
	ErrPromptNotFound = errors.New("提示词不存在")
)

// GetCurrentPrompts 获取所有槽位的当前生效版本
// 返回 is_current=true 的记录，按 prompt_key 排序
func GetCurrentPrompts() ([]models.Prompt, error) {
	query := `
		SELECT id, prompt_key, content, version, is_current, created_by, created_at
		FROM prompts
		WHERE is_current = true
		ORDER BY prompt_key ASC
	`

	rows, err := database.DB.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("查询当前提示词失败: %w", err)
	}
	defer rows.Close()

	var prompts []models.Prompt
	for rows.Next() {
		var p models.Prompt
		err := rows.Scan(&p.ID, &p.PromptKey, &p.Content, &p.Version, &p.IsCurrent, &p.CreatedBy, &p.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("扫描提示词行失败: %w", err)
		}
		prompts = append(prompts, p)
	}

	return prompts, nil
}

// GetCurrentPromptByKey 获取指定槽位的当前生效版本
func GetCurrentPromptByKey(key string) (*models.Prompt, error) {
	query := `
		SELECT id, prompt_key, content, version, is_current, created_by, created_at
		FROM prompts
		WHERE prompt_key = $1 AND is_current = true
		LIMIT 1
	`

	var p models.Prompt
	err := database.DB.QueryRow(context.Background(), query, key).Scan(
		&p.ID, &p.PromptKey, &p.Content, &p.Version, &p.IsCurrent, &p.CreatedBy, &p.CreatedAt,
	)
	if err != nil {
		return nil, ErrPromptNotFound
	}

	return &p, nil
}

// GetPromptVersions 获取指定槽位的所有版本历史（按版本号倒序）
func GetPromptVersions(key string) ([]models.Prompt, error) {
	query := `
		SELECT id, prompt_key, content, version, is_current, created_by, created_at
		FROM prompts
		WHERE prompt_key = $1
		ORDER BY version DESC
	`

	rows, err := database.DB.Query(context.Background(), query, key)
	if err != nil {
		return nil, fmt.Errorf("查询提示词版本历史失败: %w", err)
	}
	defer rows.Close()

	var prompts []models.Prompt
	for rows.Next() {
		var p models.Prompt
		err := rows.Scan(&p.ID, &p.PromptKey, &p.Content, &p.Version, &p.IsCurrent, &p.CreatedBy, &p.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("扫描提示词版本行失败: %w", err)
		}
		prompts = append(prompts, p)
	}

	return prompts, nil
}

// GetPromptByID 根据ID获取指定版本记录
func GetPromptByID(id string) (*models.Prompt, error) {
	query := `
		SELECT id, prompt_key, content, version, is_current, created_by, created_at
		FROM prompts
		WHERE id = $1
	`

	var p models.Prompt
	err := database.DB.QueryRow(context.Background(), query, id).Scan(
		&p.ID, &p.PromptKey, &p.Content, &p.Version, &p.IsCurrent, &p.CreatedBy, &p.CreatedAt,
	)
	if err != nil {
		return nil, ErrPromptNotFound
	}

	return &p, nil
}

// CreatePromptVersion 创建新版本（事务操作）
// 1. 将该 prompt_key 的所有旧版本 is_current 设为 false
// 2. 插入新版本记录，is_current = true
func CreatePromptVersion(key string, content string, version int, userID string) (*models.Prompt, error) {
	ctx := context.Background()
	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("开启事务失败: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// 步骤1：将该 prompt_key 的所有现有版本标记为非当前
	_, err = tx.Exec(ctx,
		`UPDATE prompts SET is_current = false WHERE prompt_key = $1 AND is_current = true`,
		key,
	)
	if err != nil {
		return nil, fmt.Errorf("更新旧版本状态失败: %w", err)
	}

	// 步骤2：插入新版本记录
	insertQuery := `
		INSERT INTO prompts (id, prompt_key, content, version, is_current, created_by, created_at)
		VALUES (gen_random_uuid(), $1, $2, $3, true, $4, now())
		RETURNING id, prompt_key, content, version, is_current, created_by, created_at
	`

	var p models.Prompt
	err = tx.QueryRow(ctx, insertQuery, key, content, version, userID).Scan(
		&p.ID, &p.PromptKey, &p.Content, &p.Version, &p.IsCurrent, &p.CreatedBy, &p.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("插入新版本失败: %w", err)
	}

	// 提交事务
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("提交事务失败: %w", err)
	}

	return &p, nil
}

// RollbackPromptVersion 回滚到指定版本（事务操作）
// 1. 将该 prompt_key 的所有版本 is_current 设为 false
// 2. 将目标版本 is_current 设为 true
func RollbackPromptVersion(key string, targetID string) error {
	ctx := context.Background()
	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// 步骤1：将该 prompt_key 的所有版本标记为非当前
	_, err = tx.Exec(ctx,
		`UPDATE prompts SET is_current = false WHERE prompt_key = $1`,
		key,
	)
	if err != nil {
		return fmt.Errorf("重置版本状态失败: %w", err)
	}

	// 步骤2：将目标版本标记为当前
	result, err := tx.Exec(ctx,
		`UPDATE prompts SET is_current = true WHERE id = $1 AND prompt_key = $2`,
		targetID, key,
	)
	if err != nil {
		return fmt.Errorf("设置目标版本失败: %w", err)
	}

	// 验证确实更新了一条记录
	if result.RowsAffected() == 0 {
		return ErrPromptNotFound
	}

	// 提交事务
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	return nil
}

// GetMaxVersion 获取指定槽位的最大版本号
func GetMaxVersion(key string) (int, error) {
	query := `SELECT COALESCE(MAX(version), 0) FROM prompts WHERE prompt_key = $1`

	var maxVersion int
	err := database.DB.QueryRow(context.Background(), query, key).Scan(&maxVersion)
	if err != nil {
		return 0, fmt.Errorf("查询最大版本号失败: %w", err)
	}

	return maxVersion, nil
}
