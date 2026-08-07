package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

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
//
// 【健壮性说明】ai_configs 是一张键值对表，除 AI 配置本身外，其它功能模块
// （如积分自动分配开关 token_auto_allocation_enabled）也会借用本表存开关，
// 这些外部写入可能不填 description 列导致 DB 存 NULL。而 AIConfig.Description
// 字段是值类型 string，无法容纳 NULL，直接 Scan 会报
// "cannot scan NULL into *string" 使整批查询崩溃、AI 配置页面读不出来。
// 因此 SELECT 层用 COALESCE(description, ”) 把 NULL 归一为空串，
// 无论该列存 NULL 还是空串都能安全扫入 string，永不再崩。
func GetAllConfigs() ([]*models.AIConfig, error) {
	ctx := context.Background()
	rows, err := database.DB.Query(ctx,
		`SELECT id, config_key, config_value, COALESCE(description, ''), updated_by, updated_at
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
// SELECT 同样用 COALESCE(description, ”) 防 NULL 扫描崩溃（理由同 GetAllConfigs）
func GetConfigByKey(key string) (*models.AIConfig, error) {
	ctx := context.Background()
	c := &models.AIConfig{}
	err := database.DB.QueryRow(ctx,
		`SELECT id, config_key, config_value, COALESCE(description, ''), updated_by, updated_at
                 FROM ai_configs WHERE config_key = $1`, key).Scan(
		&c.ID, &c.ConfigKey, &c.ConfigValue, &c.Description, &c.UpdatedBy, &c.UpdatedAt)
	if err != nil {
		return nil, ErrConfigNotFound
	}
	return c, nil
}

// GetConfigValue 读取单个配置值。
//
// 与GetConfigByKey不同，本方法会区分“键不存在”和数据库基础设施错误：
//   - 键不存在：返回空字符串和nil；
//   - 查询失败：返回带上下文的错误。
//
// 动态配置处理器在组装候选配置时使用本方法，不能把数据库错误误判成未配置。
func GetConfigValue(key string) (string, error) {
	ctx := context.Background()
	var value string

	err := database.DB.QueryRow(
		ctx,
		`SELECT config_value
		 FROM ai_configs
		 WHERE config_key = $1`,
		strings.TrimSpace(key),
	).Scan(&value)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("读取配置 %s 失败: %w", key, err)
	}

	return strings.TrimSpace(value), nil
}

// ConfigValueUpdate 是一次原子批量配置写入中的单项。
type ConfigValueUpdate struct {
	Key         string
	Value       string
	Description string
}

// UpsertConfigValues 在单个数据库事务中插入或更新多项配置。
//
// ASR等配置由APP ID、加密Token、资源ID和接口地址共同组成，
// 不能逐项提交后留下半套新配置。因此保存时必须全部成功或全部回滚。
func UpsertConfigValues(
	updates []ConfigValueUpdate,
	updatedBy string,
) error {
	if len(updates) == 0 {
		return nil
	}

	ctx := context.Background()
	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("开始配置批量写入事务失败: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	for _, update := range updates {
		key := strings.TrimSpace(update.Key)
		if key == "" {
			return fmt.Errorf("配置键不能为空")
		}

		_, err = tx.Exec(
			ctx,
			`INSERT INTO ai_configs (
			     id,
			     config_key,
			     config_value,
			     description,
			     updated_by,
			     updated_at
			 )
			 VALUES (
			     gen_random_uuid(),
			     $1,
			     $2,
			     $3,
			     $4,
			     NOW()
			 )
			 ON CONFLICT (config_key) DO UPDATE
			 SET config_value = EXCLUDED.config_value,
			     description = CASE
			         WHEN EXCLUDED.description = ''
			         THEN ai_configs.description
			         ELSE EXCLUDED.description
			     END,
			     updated_by = EXCLUDED.updated_by,
			     updated_at = NOW()`,
			key,
			update.Value,
			strings.TrimSpace(update.Description),
			strings.TrimSpace(updatedBy),
		)
		if err != nil {
			return fmt.Errorf("写入配置 %s 失败: %w", key, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("提交配置批量写入事务失败: %w", err)
	}

	return nil
}

// UpdateConfigValue 更新单条配置的值（仅UPDATE，键不存在返回ErrConfigNotFound）
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

// UpsertConfigValue 插入或更新单条配置（S-V1.5新增）
// 与UpdateConfigValue的区别：键不存在时自动INSERT，供动态新增配置键（如TTS provider族键）使用。
// id列用gen_random_uuid()生成（PostgreSQL 13+内置函数，本库PG16）。
func UpsertConfigValue(key string, value string, description string, updatedBy string) error {
	ctx := context.Background()
	_, err := database.DB.Exec(ctx,
		`INSERT INTO ai_configs (id, config_key, config_value, description, updated_by, updated_at)
                 VALUES (gen_random_uuid(), $1, $2, $3, $4, NOW())
                 ON CONFLICT (config_key) DO UPDATE
                 SET config_value = EXCLUDED.config_value,
                     updated_by = EXCLUDED.updated_by,
                     updated_at = NOW()`,
		key, value, description, updatedBy)
	if err != nil {
		return fmt.Errorf("写入配置 %s 失败: %w", key, err)
	}
	return nil
}

// ==================== 场景配置数据访问 ====================

// GetAllSceneConfigs 获取所有场景配置（v85：新增fallback_models列）
func GetAllSceneConfigs() ([]*models.AISceneConfig, error) {
	ctx := context.Background()
	rows, err := database.DB.Query(ctx,
		`SELECT id, scene_code, model, temperature, max_tokens,
                        system_prompt_id, is_active, updated_by, updated_at,
                        fallback_models
                 FROM ai_scene_configs ORDER BY scene_code`)
	if err != nil {
		return nil, fmt.Errorf("查询场景配置失败: %w", err)
	}
	defer rows.Close()

	var scenes []*models.AISceneConfig
	for rows.Next() {
		s := &models.AISceneConfig{}
		var fallbackRaw []byte // JSONB原始字节
		err := rows.Scan(&s.ID, &s.SceneCode, &s.Model, &s.Temperature,
			&s.MaxTokens, &s.SystemPromptID, &s.IsActive, &s.UpdatedBy, &s.UpdatedAt,
			&fallbackRaw)
		if err != nil {
			return nil, fmt.Errorf("扫描场景配置行失败: %w", err)
		}
		// 解析JSONB为字符串切片
		s.FallbackModels = models.ParseFallbackModels(fallbackRaw)
		scenes = append(scenes, s)
	}
	return scenes, nil
}

// GetSceneConfigByCode 根据场景代码获取单条配置（v85：新增fallback_models列）
func GetSceneConfigByCode(code string) (*models.AISceneConfig, error) {
	ctx := context.Background()
	s := &models.AISceneConfig{}
	var fallbackRaw []byte // JSONB原始字节
	err := database.DB.QueryRow(ctx,
		`SELECT id, scene_code, model, temperature, max_tokens,
                        system_prompt_id, is_active, updated_by, updated_at,
                        fallback_models
                 FROM ai_scene_configs WHERE scene_code = $1`, code).Scan(
		&s.ID, &s.SceneCode, &s.Model, &s.Temperature,
		&s.MaxTokens, &s.SystemPromptID, &s.IsActive, &s.UpdatedBy, &s.UpdatedAt,
		&fallbackRaw)
	if err != nil {
		return nil, ErrSceneNotFound
	}
	// 解析JSONB为字符串切片
	s.FallbackModels = models.ParseFallbackModels(fallbackRaw)
	return s, nil
}

// UpdateSceneConfig 更新场景配置（v85：新增fallback_models列）
func UpdateSceneConfig(code string, req *models.UpdateSceneConfigRequest, updatedBy string) error {
	ctx := context.Background()

	// 将fallback_models序列化为JSONB
	fallbackJSON, err := json.Marshal(req.FallbackModels)
	if err != nil {
		fallbackJSON = []byte("[]")
	}
	// nil切片序列化为"[]"确保数据库不存null
	if req.FallbackModels == nil {
		fallbackJSON = []byte("[]")
	}

	cmdTag, err := database.DB.Exec(ctx,
		`UPDATE ai_scene_configs
                 SET model = $1, temperature = $2, max_tokens = $3,
                     system_prompt_id = $4, is_active = $5,
                     updated_by = $6, updated_at = NOW(),
                     fallback_models = $7
                 WHERE scene_code = $8`,
		req.Model, req.Temperature, req.MaxTokens,
		req.SystemPromptID, req.IsActive,
		updatedBy, fallbackJSON, code)
	if err != nil {
		return fmt.Errorf("更新场景配置 %s 失败: %w", code, err)
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrSceneNotFound
	}
	return nil
}
