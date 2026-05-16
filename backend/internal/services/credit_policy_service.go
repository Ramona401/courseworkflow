package services

// credit_policy_service.go — 积分策略业务逻辑 + 积分计算引擎
//
// v129 新增（积分机制融合 · 对齐AOCI精确积分计算）：
//   - 策略管理：系统策略/学校策略 CRUD
//   - 积分计算引擎：CalculateCredits（核心！）
//   - 模型单价管理：CRUD
//   - 模拟计算：Simulate
//   - 模型积分预览：GetModelPreviews
//
// 积分计算公式（对齐AOCI的credit_policy.go）：
//   cost_usd = (input_tokens/1000 × cost_per_1k_input) + (output_tokens/1000 × cost_per_1k_output)
//   credits  = cost_usd × exchange_rate × multiplier
//
// 策略查询链（对齐AOCI的getPolicyForModel）：
//   学校策略(school, schoolID) → 系统策略(system, NULL) → 默认兜底(7.0 × 1.0)

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== CreditPolicyService 结构体 ====================

// CreditPolicyService 积分策略服务（对齐AOCI的credit_policy.go service）
type CreditPolicyService struct{}

// NewCreditPolicyService 创建CreditPolicyService实例
func NewCreditPolicyService() *CreditPolicyService {
	return &CreditPolicyService{}
}

// ==================== 核心：积分计算引擎 ====================

// CalculateCredits 根据真实token消耗计算积分（核心方法）
//
// 对齐AOCI: credit_policy.go → CalculateCreditsForCall
//
// 计算步骤：
//   1. 查模型单价 → 计算美元成本
//   2. 查策略（学校→系统→兜底）→ 获取汇率×倍率
//   3. 积分 = 美元成本 × 汇率 × 倍率
//
// 参数：
//   modelUsed    — AI实际使用的模型名称（来自API响应）
//   inputTokens  — 输入token数（来自API响应的usage.prompt_tokens）
//   outputTokens — 输出token数（来自API响应的usage.completion_tokens）
//   totalTokens  — 总token数（来自API响应的usage.total_tokens，当分开的不可用时兜底）
//   schoolID     — 用户所属学校ID（可为nil，用于查学校策略）
//   latencyMs    — 调用耗时（毫秒）
func (s *CreditPolicyService) CalculateCredits(
	ctx context.Context,
	modelUsed string,
	inputTokens int,
	outputTokens int,
	totalTokens int,
	schoolID *string,
	latencyMs int64,
) *models.CreditCalculation {

	// 如果没有分开的token统计，用总token数按6:4比例估算
	if inputTokens == 0 && outputTokens == 0 && totalTokens > 0 {
		inputTokens = totalTokens * 6 / 10
		outputTokens = totalTokens - inputTokens
	}

	// token数全部为0，返回零消费
	if inputTokens == 0 && outputTokens == 0 {
		return &models.CreditCalculation{
			ModelName:   modelUsed,
			LatencyMs:   latencyMs,
		}
	}

	// 1. 查模型单价
	price, err := repository.GetModelPriceByName(ctx, modelUsed)
	if err != nil {
		// 模型单价未配置，尝试模糊匹配
		price = s.estimateModelPrice(modelUsed)
		if price == nil {
			log.Printf("[CreditPolicy] 模型 %s 未配置单价且无法估算，积分为0", modelUsed)
			return &models.CreditCalculation{
				InputTokens:  inputTokens,
				OutputTokens: outputTokens,
				ModelName:    modelUsed,
				LatencyMs:    latencyMs,
			}
		}
	}

	// 2. 计算美元成本
	costUSD := price.CalculateCostUSD(inputTokens, outputTokens)

	// 3. 查策略（学校→系统→兜底）
	policy := s.GetEffectivePolicy(ctx, schoolID)

	// 4. 计算积分
	credits := policy.CalculateCredits(costUSD)

	return &models.CreditCalculation{
		InputTokens:     inputTokens,
		OutputTokens:    outputTokens,
		ModelName:       modelUsed,
		Provider:        price.Provider,
		CostUSD:         costUSD,
		ExchangeRate:    policy.ExchangeRate,
		Multiplier:      policy.Multiplier,
		CreditsConsumed: credits,
		LatencyMs:       latencyMs,
	}
}

// GetEffectivePolicy 获取有效策略（对齐AOCI的策略查询链）
// 查询链: 学校策略 → 系统策略 → 默认兜底(7.0 × 1.0)
func (s *CreditPolicyService) GetEffectivePolicy(ctx context.Context, schoolID *string) *models.CreditPolicy {
	// 1. 尝试学校级策略
	if schoolID != nil && *schoolID != "" {
		policy, err := repository.GetSchoolCreditPolicy(ctx, *schoolID)
		if err == nil && policy != nil {
			return policy
		}
	}
	// 2. 回退系统策略
	policy, err := repository.GetSystemCreditPolicy(ctx)
	if err == nil && policy != nil {
		return policy
	}
	// 3. 兜底默认
	return &models.CreditPolicy{
		ExchangeRate: models.DefaultExchangeRate,
		Multiplier:   models.DefaultMultiplier,
	}
}

// estimateModelPrice 未配置模型的兜底估价
// 按模型名称关键词推断供应商和价格区间
func (s *CreditPolicyService) estimateModelPrice(modelName string) *models.ModelPrice {
	nameLower := strings.ToLower(modelName)

	// Anthropic模型估价
	if strings.Contains(nameLower, "haiku") {
		return &models.ModelPrice{
			ModelName: modelName, Provider: "anthropic",
			CostPer1kInput: 0.0008, CostPer1kOutput: 0.004,
		}
	}
	if strings.Contains(nameLower, "opus") {
		return &models.ModelPrice{
			ModelName: modelName, Provider: "anthropic",
			CostPer1kInput: 0.015, CostPer1kOutput: 0.075,
		}
	}
	if strings.Contains(nameLower, "sonnet") || strings.Contains(nameLower, "claude") {
		return &models.ModelPrice{
			ModelName: modelName, Provider: "anthropic",
			CostPer1kInput: 0.003, CostPer1kOutput: 0.015,
		}
	}

	// Google模型估价
	if strings.Contains(nameLower, "gemini") {
		if strings.Contains(nameLower, "flash") {
			return &models.ModelPrice{
				ModelName: modelName, Provider: "google",
				CostPer1kInput: 0.00015, CostPer1kOutput: 0.0006,
			}
		}
		return &models.ModelPrice{
			ModelName: modelName, Provider: "google",
			CostPer1kInput: 0.00125, CostPer1kOutput: 0.01,
		}
	}

	// 完全未知的模型，按Sonnet价格兜底（中间档，不会过高或过低）
	return &models.ModelPrice{
		ModelName: modelName, Provider: "unknown",
		CostPer1kInput: 0.003, CostPer1kOutput: 0.015,
	}
}

// ==================== 策略管理 ====================

// GetSystemPolicy 获取系统级策略
func (s *CreditPolicyService) GetSystemPolicy(ctx context.Context) (*models.CreditPolicy, error) {
	return repository.GetSystemCreditPolicy(ctx)
}

// UpdateSystemPolicy 更新系统级策略
func (s *CreditPolicyService) UpdateSystemPolicy(ctx context.Context, req *models.UpdateCreditPolicyRequest, updatedBy string) (*models.CreditPolicy, error) {
	// 先获取当前策略
	current, _ := repository.GetSystemCreditPolicy(ctx)
	exchangeRate := models.DefaultExchangeRate
	multiplier := models.DefaultMultiplier
	description := ""
	if current != nil {
		exchangeRate = current.ExchangeRate
		multiplier = current.Multiplier
		description = current.Description
	}

	// 合并更新
	if req.ExchangeRate != nil {
		exchangeRate = *req.ExchangeRate
	}
	if req.Multiplier != nil {
		multiplier = *req.Multiplier
	}
	if req.Description != nil {
		description = *req.Description
	}

	return repository.UpsertCreditPolicy(ctx, models.PolicyScopeSystem, nil, exchangeRate, multiplier, description, &updatedBy)
}

// GetSchoolPolicy 获取学校级策略
func (s *CreditPolicyService) GetSchoolPolicy(ctx context.Context, schoolID string) (*models.CreditPolicy, error) {
	return repository.GetSchoolCreditPolicy(ctx, schoolID)
}

// UpdateSchoolPolicy 更新学校级策略
func (s *CreditPolicyService) UpdateSchoolPolicy(ctx context.Context, schoolID string, req *models.UpdateCreditPolicyRequest, updatedBy string) (*models.CreditPolicy, error) {
	// 先获取当前策略（可能不存在）
	current, _ := repository.GetSchoolCreditPolicy(ctx, schoolID)
	exchangeRate := models.DefaultExchangeRate
	multiplier := models.DefaultMultiplier
	description := ""
	if current != nil {
		exchangeRate = current.ExchangeRate
		multiplier = current.Multiplier
		description = current.Description
	}

	if req.ExchangeRate != nil {
		exchangeRate = *req.ExchangeRate
	}
	if req.Multiplier != nil {
		multiplier = *req.Multiplier
	}
	if req.Description != nil {
		description = *req.Description
	}

	return repository.UpsertCreditPolicy(ctx, models.PolicyScopeSchool, &schoolID, exchangeRate, multiplier, description, &updatedBy)
}

// DeleteSchoolPolicy 删除学校级策略
func (s *CreditPolicyService) DeleteSchoolPolicy(ctx context.Context, schoolID string) error {
	return repository.DeleteSchoolCreditPolicy(ctx, schoolID)
}

// ListPolicies 列出所有策略
func (s *CreditPolicyService) ListPolicies(ctx context.Context) ([]*models.CreditPolicyListItem, error) {
	return repository.ListCreditPolicies(ctx)
}

// ==================== 模型单价管理 ====================

// ListModelPrices 列出模型单价
func (s *CreditPolicyService) ListModelPrices(ctx context.Context, includeInactive bool) ([]models.ModelPrice, error) {
	return repository.ListModelPrices(ctx, includeInactive)
}

// CreateModelPrice 创建模型单价
func (s *CreditPolicyService) CreateModelPrice(ctx context.Context, req *models.CreateModelPriceRequest, updatedBy string) (*models.ModelPrice, error) {
	if req.ModelName == "" {
		return nil, fmt.Errorf("模型名称不能为空")
	}
	if req.Provider == "" {
		return nil, fmt.Errorf("供应商不能为空")
	}

	mp := &models.ModelPrice{
		ModelName:       req.ModelName,
		Provider:        req.Provider,
		CostPer1kInput:  req.CostPer1kInput,
		CostPer1kOutput: req.CostPer1kOutput,
		DisplayName:     req.DisplayName,
		IsActive:        true,
		UpdatedBy:       &updatedBy,
	}
	if err := repository.CreateModelPrice(ctx, mp); err != nil {
		if errors.Is(err, repository.ErrModelPriceDuplicate) {
			return nil, err
		}
		return nil, fmt.Errorf("创建模型单价失败: %w", err)
	}
	return mp, nil
}

// UpdateModelPrice 更新模型单价
func (s *CreditPolicyService) UpdateModelPrice(ctx context.Context, id string, req *models.UpdateModelPriceRequest, updatedBy string) (*models.ModelPrice, error) {
	return repository.UpdateModelPrice(ctx, id, req.CostPer1kInput, req.CostPer1kOutput,
		req.DisplayName, req.IsActive, &updatedBy)
}

// DeleteModelPrice 删除模型单价
func (s *CreditPolicyService) DeleteModelPrice(ctx context.Context, id string) error {
	return repository.DeleteModelPrice(ctx, id)
}

// ==================== 模拟计算 + 预览 ====================

// Simulate 模拟积分计算（供前端预估）
func (s *CreditPolicyService) Simulate(ctx context.Context, req *models.SimulateCreditRequest) (*models.CreditCalculation, error) {
	if req.ModelName == "" {
		return nil, fmt.Errorf("模型名称不能为空")
	}
	if req.InputTokens <= 0 && req.OutputTokens <= 0 {
		return nil, fmt.Errorf("输入/输出token数至少填一个")
	}

	calc := s.CalculateCredits(ctx, req.ModelName, req.InputTokens, req.OutputTokens, 0, req.SchoolID, 0)
	return calc, nil
}

// GetModelPreviews 获取所有活跃模型的积分预览（按系统策略计算）
func (s *CreditPolicyService) GetModelPreviews(ctx context.Context) ([]models.ModelPricePreview, error) {
	prices, err := repository.ListModelPrices(ctx, false) // 仅活跃模型
	if err != nil {
		return nil, err
	}

	policy := s.GetEffectivePolicy(ctx, nil) // 系统策略

	var previews []models.ModelPricePreview
	for _, p := range prices {
		previews = append(previews, models.ModelPricePreview{
			ModelName:          p.ModelName,
			Provider:           p.Provider,
			DisplayName:        p.DisplayName,
			CostPer1kInput:     p.CostPer1kInput,
			CostPer1kOutput:    p.CostPer1kOutput,
			CreditsPer1kInput:  p.CostPer1kInput * policy.ExchangeRate * policy.Multiplier,
			CreditsPer1kOutput: p.CostPer1kOutput * policy.ExchangeRate * policy.Multiplier,
		})
	}
	return previews, nil
}
