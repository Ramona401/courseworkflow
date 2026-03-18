package repository

import (
	"context"
	"errors"
	"fmt"

	"tedna/internal/database"
	"tedna/internal/models"
)

// ==================== 错误常量 ====================

var (
	ErrConfigNotFound = errors.New("配置项不存在")
	ErrSceneNotFound  = errors.New("场景配置不存在")
)

// ==================== 全局配置数据访问 ====================

// GetAllConfigs 获取所有全局配置项
func GetAllConfigs() ([]*models.AIConfig, error) {
	ctx := context.Background()
	rows, err := database.DB.Query(ctx,
		`SELECT id, config_key, config_value, description, updated_by, updated_at
		 FROM ai_configs ORDER BY config_key`)
	if err != nil {
		return nil, fmt.Errorf("查询全局配置失败: %w", err)
	}
	defer rows.Close()

	var configs []*models.AIConfig
	for rows.Next() {
		c := &models.AIConfig{}
		err := rows.Scan(&c.ID, &c.ConfigKey, &c.ConfigValue, &c.Description, &c.UpdatedBy, &c.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("扫描全局配置行失败: %w", err)
		}
		configs = append(configs, c)
	}
	return configs, nil
}

// GetConfigByKey 根据键名获取单条配置
func GetConfigByKey(key string) (*models.AIConfig, error) {
	ctx := context.Background()
	c := &models.AIConfig{}
	err := database.DB.QueryRow(ctx,
		`SELECT id, config_key, config_value, description, updated_by, updated_at
		 FROM ai_configs WHERE config_key = $1`, key).Scan(
		&c.ID, &c.ConfigKey, &c.ConfigValue, &c.Description, &c.UpdatedBy, &c.UpdatedAt)
	if err != nil {
		return nil, ErrConfigNotFound
	}
	return c, nil
}

// UpdateConfigValue 更新单条配置的值
func UpdateConfigValue(key string, value string, updatedBy string) error {
	ctx := context.Background()
	cmdTag, err := database.DB.Exec(ctx,
		`UPDATE ai_configs SET config_value = $1, updated_by = $2, updated_at = NOW()
		 WHERE config_key = $3`, value, updatedBy, key)
	if err != nil {
		return fmt.Errorf("更新配置 %s 失败: %w", key, err)
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrConfigNotFound
	}
	return nil
}

// ==================== 场景配置数据访问 ====================

// GetAllSceneConfigs 获取所有场景配置
func GetAllSceneConfigs() ([]*models.AISceneConfig, error) {
	ctx := context.Background()
	rows, err := database.DB.Query(ctx,
		`SELECT id, scene_code, model, temperature, max_tokens,
		        system_prompt_id, is_active, updated_by, updated_at
		 FROM ai_scene_configs ORDER BY scene_code`)
	if err != nil {
		return nil, fmt.Errorf("查询场景配置失败: %w", err)
	}
	defer rows.Close()

	var scenes []*models.AISceneConfig
	for rows.Next() {
		s := &models.AISceneConfig{}
		err := rows.Scan(&s.ID, &s.SceneCode, &s.Model, &s.Temperature,
			&s.MaxTokens, &s.SystemPromptID, &s.IsActive, &s.UpdatedBy, &s.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("扫描场景配置行失败: %w", err)
		}
		scenes = append(scenes, s)
	}
	return scenes, nil
}

// GetSceneConfigByCode 根据场景代码获取单条配置
func GetSceneConfigByCode(code string) (*models.AISceneConfig, error) {
	ctx := context.Background()
	s := &models.AISceneConfig{}
	err := database.DB.QueryRow(ctx,
		`SELECT id, scene_code, model, temperature, max_tokens,
		        system_prompt_id, is_active, updated_by, updated_at
		 FROM ai_scene_configs WHERE scene_code = $1`, code).Scan(
		&s.ID, &s.SceneCode, &s.Model, &s.Temperature,
		&s.MaxTokens, &s.SystemPromptID, &s.IsActive, &s.UpdatedBy, &s.UpdatedAt)
	if err != nil {
		return nil, ErrSceneNotFound
	}
	return s, nil
}

// UpdateSceneConfig 更新场景配置
func UpdateSceneConfig(code string, req *models.UpdateSceneConfigRequest, updatedBy string) error {
	ctx := context.Background()
	cmdTag, err := database.DB.Exec(ctx,
		`UPDATE ai_scene_configs
		 SET model = $1, temperature = $2, max_tokens = $3,
		     system_prompt_id = $4, is_active = $5,
		     updated_by = $6, updated_at = NOW()
		 WHERE scene_code = $7`,
		req.Model, req.Temperature, req.MaxTokens,
		req.SystemPromptID, req.IsActive,
		updatedBy, code)
	if err != nil {
		return fmt.Errorf("更新场景配置 %s 失败: %w", code, err)
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrSceneNotFound
	}
	return nil
}
