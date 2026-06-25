package repository

// model_alias_repo.go — 模型别名映射数据访问层（批三-2）
//
// 对应表：model_alias_rules。
// 核心：ResolveModelAlias —— 给真实模型名，返回业务别名（精确>前缀>兜底）。
// 风格对齐 school_model_policy_repo.go：database.DB、ctx、fmt.Errorf 包错。

import (
	"context"
	"fmt"
	"strings"

	"tedna/internal/database"
	"tedna/internal/models"
)

// ListModelAliasRules 列出全部规则（管理界面用，按 match_type、priority、pattern 排序便于阅读）
func ListModelAliasRules(ctx context.Context) ([]*models.ModelAliasRule, error) {
	query := `
		SELECT id, match_type, pattern, alias, priority, enabled, note, created_by, created_at, updated_at
		FROM model_alias_rules
		ORDER BY match_type ASC, priority DESC, length(pattern) DESC, pattern ASC
	`
	rows, err := database.DB.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("查询模型别名规则列表失败: %w", err)
	}
	defer rows.Close()

	items := []*models.ModelAliasRule{}
	for rows.Next() {
		r := &models.ModelAliasRule{}
		if err := rows.Scan(
			&r.ID, &r.MatchType, &r.Pattern, &r.Alias, &r.Priority,
			&r.Enabled, &r.Note, &r.CreatedBy, &r.CreatedAt, &r.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("扫描模型别名规则行失败: %w", err)
		}
		items = append(items, r)
	}
	return items, nil
}

// GetModelAliasRule 按 ID 查单条（不存在返回 nil, nil）
func GetModelAliasRule(ctx context.Context, id string) (*models.ModelAliasRule, error) {
	if id == "" {
		return nil, nil
	}
	r := &models.ModelAliasRule{}
	err := database.DB.QueryRow(ctx, `
		SELECT id, match_type, pattern, alias, priority, enabled, note, created_by, created_at, updated_at
		FROM model_alias_rules WHERE id = $1
	`, id).Scan(
		&r.ID, &r.MatchType, &r.Pattern, &r.Alias, &r.Priority,
		&r.Enabled, &r.Note, &r.CreatedBy, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("查询模型别名规则失败: %w", err)
	}
	return r, nil
}

// CreateModelAliasRule 新增规则。createdBy 可空（空串写 NULL）。
// 依赖唯一索引 (match_type, pattern) 防重复，重复时返回错误由 handler 转人话。
func CreateModelAliasRule(ctx context.Context, matchType, pattern, alias string, priority int, enabled bool, note, createdBy string) (string, error) {
	if pattern == "" || alias == "" {
		return "", fmt.Errorf("pattern 与 alias 不能为空")
	}
	if matchType != models.MatchTypeExact && matchType != models.MatchTypePrefix {
		return "", fmt.Errorf("match_type 非法（仅 exact/prefix）")
	}
	var createdByArg interface{}
	if createdBy == "" {
		createdByArg = nil
	} else {
		createdByArg = createdBy
	}
	var newID string
	err := database.DB.QueryRow(ctx, `
		INSERT INTO model_alias_rules (match_type, pattern, alias, priority, enabled, note, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, matchType, pattern, alias, priority, enabled, note, createdByArg).Scan(&newID)
	if err != nil {
		return "", fmt.Errorf("创建模型别名规则失败: %w", err)
	}
	return newID, nil
}

// UpdateModelAliasRule 更新规则（全字段覆盖）
func UpdateModelAliasRule(ctx context.Context, id, matchType, pattern, alias string, priority int, enabled bool, note string) error {
	if id == "" {
		return fmt.Errorf("id 为空")
	}
	if pattern == "" || alias == "" {
		return fmt.Errorf("pattern 与 alias 不能为空")
	}
	if matchType != models.MatchTypeExact && matchType != models.MatchTypePrefix {
		return fmt.Errorf("match_type 非法（仅 exact/prefix）")
	}
	ct, err := database.DB.Exec(ctx, `
		UPDATE model_alias_rules
		SET match_type=$2, pattern=$3, alias=$4, priority=$5, enabled=$6, note=$7, updated_at=now()
		WHERE id=$1
	`, id, matchType, pattern, alias, priority, enabled, note)
	if err != nil {
		return fmt.Errorf("更新模型别名规则失败: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("规则不存在")
	}
	return nil
}

// DeleteModelAliasRule 删除规则
func DeleteModelAliasRule(ctx context.Context, id string) error {
	_, err := database.DB.Exec(ctx, `DELETE FROM model_alias_rules WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("删除模型别名规则失败: %w", err)
	}
	return nil
}

// ResolveModelAlias 核心：给真实模型名，返回业务别名。
// 优先级：精确(exact) > 前缀(prefix，最长+最高优先) > 兜底(fallback)。
// 该函数供批三-3 老师侧渲染调用；批三-2 仅用于管理界面的「预览」端点自测。
//
// fallback 由调用方传入（来自 ai_configs.model_alias_fallback，缺失时用 DefaultModelAliasFallback）。
// 任何查询异常都返回 fallback（fail-safe：宁可显示兜底名，绝不暴露真实模型名）。
func ResolveModelAlias(ctx context.Context, modelName, fallback string) string {
	if fallback == "" {
		fallback = models.DefaultModelAliasFallback
	}
	trimmed := strings.TrimSpace(modelName)
	if trimmed == "" {
		return fallback
	}

	// 1) 精确匹配（启用规则中 pattern 完全相等，取 priority 最高）
	var exactAlias string
	err := database.DB.QueryRow(ctx, `
		SELECT alias FROM model_alias_rules
		WHERE enabled = true AND match_type = 'exact' AND pattern = $1
		ORDER BY priority DESC
		LIMIT 1
	`, trimmed).Scan(&exactAlias)
	if err == nil && exactAlias != "" {
		return exactAlias
	}
	// 精确未命中（no rows）继续前缀；其他错误也 fail-safe 往下走

	// 2) 前缀匹配（模型名以 pattern 开头；最长前缀+最高优先）
	//    用 $1 LIKE pattern||'%' 实现「模型名以 pattern 开头」，排序保证最长最优先。
	var prefixAlias string
	err = database.DB.QueryRow(ctx, `
		SELECT alias FROM model_alias_rules
		WHERE enabled = true AND match_type = 'prefix' AND $1 LIKE pattern || '%'
		ORDER BY priority DESC, length(pattern) DESC
		LIMIT 1
	`, trimmed).Scan(&prefixAlias)
	if err == nil && prefixAlias != "" {
		return prefixAlias
	}

	// 3) 兜底
	return fallback
}
