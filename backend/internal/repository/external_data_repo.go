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
	ErrEDConfigNotFound = errors.New("外部数据配置项不存在")
)

// ==================== 外部数据配置数据访问 ====================

// GetAllEDConfigs 获取所有外部数据配置项
func GetAllEDConfigs() ([]*models.ExternalDataConfig, error) {
	ctx := context.Background()
	rows, err := database.DB.Query(ctx,
		`SELECT id, config_key, config_value, description, updated_by, updated_at
		 FROM external_data_configs ORDER BY config_key`)
	if err != nil {
		return nil, fmt.Errorf("查询外部数据配置失败: %w", err)
	}
	defer rows.Close()

	var configs []*models.ExternalDataConfig
	for rows.Next() {
		c := &models.ExternalDataConfig{}
		err := rows.Scan(&c.ID, &c.ConfigKey, &c.ConfigValue, &c.Description, &c.UpdatedBy, &c.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("扫描外部数据配置行失败: %w", err)
		}
		configs = append(configs, c)
	}
	return configs, nil
}

// GetEDConfigByKey 根据键名获取单条外部数据配置
func GetEDConfigByKey(key string) (*models.ExternalDataConfig, error) {
	ctx := context.Background()
	c := &models.ExternalDataConfig{}
	err := database.DB.QueryRow(ctx,
		`SELECT id, config_key, config_value, description, updated_by, updated_at
		 FROM external_data_configs WHERE config_key = $1`, key).Scan(
		&c.ID, &c.ConfigKey, &c.ConfigValue, &c.Description, &c.UpdatedBy, &c.UpdatedAt)
	if err != nil {
		return nil, ErrEDConfigNotFound
	}
	return c, nil
}

// UpdateEDConfigValue 更新单条外部数据配置的值
func UpdateEDConfigValue(key string, value string, updatedBy string) error {
	ctx := context.Background()
	cmdTag, err := database.DB.Exec(ctx,
		`UPDATE external_data_configs SET config_value = $1, updated_by = $2, updated_at = NOW()
		 WHERE config_key = $3`, value, updatedBy, key)
	if err != nil {
		return fmt.Errorf("更新外部数据配置 %s 失败: %w", key, err)
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrEDConfigNotFound
	}
	return nil
}
