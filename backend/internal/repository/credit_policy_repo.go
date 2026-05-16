package repository

// credit_policy_repo.go — 积分策略 + 模型单价数据访问层
//
// v129 新增（积分机制融合 · 对齐AOCI精确积分计算）：
//   - 策略 CRUD（系统级/学校级，UPSERT按scope+scope_id唯一键）
//   - 模型单价 CRUD
//
// 对应数据库表：
//   token_credit_policies — 积分策略
//   token_model_prices    — 模型单价

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
	ErrCreditPolicyNotFound = errors.New("积分策略不存在")
	ErrModelPriceNotFound   = errors.New("模型单价不存在")
	ErrModelPriceDuplicate  = errors.New("模型单价已存在")
)

// ==================== 积分策略 ====================

// GetSystemCreditPolicy 获取系统级策略
func GetSystemCreditPolicy(ctx context.Context) (*models.CreditPolicy, error) {
	return getCreditPolicyByScope(ctx, models.PolicyScopeSystem, nil)
}

// GetSchoolCreditPolicy 获取学校级策略
func GetSchoolCreditPolicy(ctx context.Context, schoolID string) (*models.CreditPolicy, error) {
	return getCreditPolicyByScope(ctx, models.PolicyScopeSchool, &schoolID)
}

// getCreditPolicyByScope 按scope+scope_id查询策略（内部函数）
func getCreditPolicyByScope(ctx context.Context, scope string, scopeID *string) (*models.CreditPolicy, error) {
	var p models.CreditPolicy
	var query string
	var args []interface{}

	if scopeID == nil {
		query = `SELECT id, scope, scope_id, exchange_rate, multiplier, description,
		                updated_by, created_at, updated_at
		         FROM token_credit_policies WHERE scope = $1 AND scope_id IS NULL LIMIT 1`
		args = []interface{}{scope}
	} else {
		query = `SELECT id, scope, scope_id, exchange_rate, multiplier, description,
		                updated_by, created_at, updated_at
		         FROM token_credit_policies WHERE scope = $1 AND scope_id = $2 LIMIT 1`
		args = []interface{}{scope, *scopeID}
	}

	err := database.DB.QueryRow(ctx, query, args...).Scan(
		&p.ID, &p.Scope, &p.ScopeID, &p.ExchangeRate, &p.Multiplier,
		&p.Description, &p.UpdatedBy, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCreditPolicyNotFound
		}
		return nil, fmt.Errorf("查询积分策略失败: %w", err)
	}
	return &p, nil
}

// UpsertCreditPolicy 创建或更新策略（按scope+scope_id唯一键UPSERT）
func UpsertCreditPolicy(ctx context.Context, scope string, scopeID *string,
	exchangeRate float64, multiplier float64, description string, updatedBy *string) (*models.CreditPolicy, error) {

	if exchangeRate <= 0 {
		return nil, fmt.Errorf("汇率必须大于0")
	}
	if multiplier <= 0 {
		return nil, fmt.Errorf("倍率必须大于0")
	}

	var p models.CreditPolicy
	err := database.DB.QueryRow(ctx, `
		INSERT INTO token_credit_policies (scope, scope_id, exchange_rate, multiplier, description, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (scope, scope_id) DO UPDATE SET
		  exchange_rate = EXCLUDED.exchange_rate,
		  multiplier = EXCLUDED.multiplier,
		  description = EXCLUDED.description,
		  updated_by = EXCLUDED.updated_by,
		  updated_at = NOW()
		RETURNING id, scope, scope_id, exchange_rate, multiplier, description, updated_by, created_at, updated_at
	`, scope, scopeID, exchangeRate, multiplier, description, updatedBy,
	).Scan(&p.ID, &p.Scope, &p.ScopeID, &p.ExchangeRate, &p.Multiplier,
		&p.Description, &p.UpdatedBy, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("更新积分策略失败: %w", err)
	}
	return &p, nil
}

// DeleteSchoolCreditPolicy 删除学校级策略（系统策略不可删）
func DeleteSchoolCreditPolicy(ctx context.Context, schoolID string) error {
	result, err := database.DB.Exec(ctx,
		`DELETE FROM token_credit_policies WHERE scope = 'school' AND scope_id = $1`,
		schoolID,
	)
	if err != nil {
		return fmt.Errorf("删除学校策略失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrCreditPolicyNotFound
	}
	return nil
}

// ListCreditPolicies 列出所有策略（含学校名称）
func ListCreditPolicies(ctx context.Context) ([]*models.CreditPolicyListItem, error) {
	rows, err := database.DB.Query(ctx, `
		SELECT p.id, p.scope, p.scope_id, p.exchange_rate, p.multiplier,
		       p.description, p.updated_by, p.created_at, p.updated_at,
		       COALESCE(o.name, '') AS school_name
		FROM token_credit_policies p
		LEFT JOIN organizations o ON o.id = p.scope_id AND p.scope = 'school'
		ORDER BY p.scope ASC, p.created_at ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("查询策略列表失败: %w", err)
	}
	defer rows.Close()

	var items []*models.CreditPolicyListItem
	for rows.Next() {
		item := &models.CreditPolicyListItem{}
		err := rows.Scan(
			&item.ID, &item.Scope, &item.ScopeID, &item.ExchangeRate, &item.Multiplier,
			&item.Description, &item.UpdatedBy, &item.CreatedAt, &item.UpdatedAt,
			&item.SchoolName,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描策略行失败: %w", err)
		}
		item.EffectiveRateValue = item.ExchangeRate * item.Multiplier
		items = append(items, item)
	}
	return items, nil
}

// ==================== 模型单价 ====================

// GetModelPriceByName 按模型名称查单价（仅查活跃的）
func GetModelPriceByName(ctx context.Context, modelName string) (*models.ModelPrice, error) {
	var mp models.ModelPrice
	err := database.DB.QueryRow(ctx, `
		SELECT id, model_name, provider, cost_per_1k_input, cost_per_1k_output,
		       display_name, is_active, updated_by, created_at, updated_at
		FROM token_model_prices WHERE model_name = $1 AND is_active = true LIMIT 1
	`, modelName).Scan(
		&mp.ID, &mp.ModelName, &mp.Provider, &mp.CostPer1kInput, &mp.CostPer1kOutput,
		&mp.DisplayName, &mp.IsActive, &mp.UpdatedBy, &mp.CreatedAt, &mp.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrModelPriceNotFound
		}
		return nil, fmt.Errorf("查询模型单价失败: %w", err)
	}
	return &mp, nil
}

// GetModelPriceByID 按ID查单价
func GetModelPriceByID(ctx context.Context, id string) (*models.ModelPrice, error) {
	var mp models.ModelPrice
	err := database.DB.QueryRow(ctx, `
		SELECT id, model_name, provider, cost_per_1k_input, cost_per_1k_output,
		       display_name, is_active, updated_by, created_at, updated_at
		FROM token_model_prices WHERE id = $1 LIMIT 1
	`, id).Scan(
		&mp.ID, &mp.ModelName, &mp.Provider, &mp.CostPer1kInput, &mp.CostPer1kOutput,
		&mp.DisplayName, &mp.IsActive, &mp.UpdatedBy, &mp.CreatedAt, &mp.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrModelPriceNotFound
		}
		return nil, fmt.Errorf("查询模型单价失败: %w", err)
	}
	return &mp, nil
}

// ListModelPrices 列出所有模型单价
func ListModelPrices(ctx context.Context, includeInactive bool) ([]models.ModelPrice, error) {
	query := `SELECT id, model_name, provider, cost_per_1k_input, cost_per_1k_output,
	                 display_name, is_active, updated_by, created_at, updated_at
	          FROM token_model_prices`
	if !includeInactive {
		query += ` WHERE is_active = true`
	}
	query += ` ORDER BY provider ASC, model_name ASC`

	rows, err := database.DB.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("查询模型单价列表失败: %w", err)
	}
	defer rows.Close()

	var items []models.ModelPrice
	for rows.Next() {
		var mp models.ModelPrice
		err := rows.Scan(
			&mp.ID, &mp.ModelName, &mp.Provider, &mp.CostPer1kInput, &mp.CostPer1kOutput,
			&mp.DisplayName, &mp.IsActive, &mp.UpdatedBy, &mp.CreatedAt, &mp.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描模型单价行失败: %w", err)
		}
		items = append(items, mp)
	}
	return items, nil
}

// CreateModelPrice 创建模型单价
func CreateModelPrice(ctx context.Context, mp *models.ModelPrice) error {
	err := database.DB.QueryRow(ctx, `
		INSERT INTO token_model_prices
			(model_name, provider, cost_per_1k_input, cost_per_1k_output, display_name, is_active, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`,
		mp.ModelName, mp.Provider, mp.CostPer1kInput, mp.CostPer1kOutput,
		mp.DisplayName, mp.IsActive, mp.UpdatedBy,
	).Scan(&mp.ID, &mp.CreatedAt, &mp.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrModelPriceDuplicate
		}
		return fmt.Errorf("创建模型单价失败: %w", err)
	}
	return nil
}

// UpdateModelPrice 更新模型单价
func UpdateModelPrice(ctx context.Context, id string, costIn *float64, costOut *float64,
	displayName *string, isActive *bool, updatedBy *string) (*models.ModelPrice, error) {

	now := time.Now()
	// 先获取当前值
	mp, err := GetModelPriceByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 合并更新字段
	if costIn != nil {
		mp.CostPer1kInput = *costIn
	}
	if costOut != nil {
		mp.CostPer1kOutput = *costOut
	}
	if displayName != nil {
		mp.DisplayName = *displayName
	}
	if isActive != nil {
		mp.IsActive = *isActive
	}

	_, err = database.DB.Exec(ctx, `
		UPDATE token_model_prices
		SET cost_per_1k_input = $1, cost_per_1k_output = $2, display_name = $3,
		    is_active = $4, updated_by = $5, updated_at = $6
		WHERE id = $7
	`, mp.CostPer1kInput, mp.CostPer1kOutput, mp.DisplayName,
		mp.IsActive, updatedBy, now, id)
	if err != nil {
		return nil, fmt.Errorf("更新模型单价失败: %w", err)
	}
	mp.UpdatedAt = now
	return mp, nil
}

// DeleteModelPrice 删除模型单价
func DeleteModelPrice(ctx context.Context, id string) error {
	result, err := database.DB.Exec(ctx,
		`DELETE FROM token_model_prices WHERE id = $1`, id,
	)
	if err != nil {
		return fmt.Errorf("删除模型单价失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrModelPriceNotFound
	}
	return nil
}
