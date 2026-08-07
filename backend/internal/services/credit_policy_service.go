package services

// credit_policy_service.go — 积分策略业务逻辑与文本积分计算引擎。
//
// 核心公式：
//   cost_usd =
//     input_tokens / 1000 × input_price +
//     output_tokens / 1000 × output_price
//
//   credits =
//     cost_usd × exchange_rate × multiplier
//
// 价格查询链：
//   1. token_model_prices精确模型名；
//   2. 对明确识别的版本化模型应用实际价格及阶梯规则；
//   3. 数据库无精确记录时使用实际模型兜底规则；
//   4. 完全未知模型使用中档安全兜底并记录告警。
//
// 价格修正只影响后续调用，不重新计算历史消费记录。

import (
	"context"
	"errors"
	"fmt"

	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
)

var cpLog = logger.WithModule(
	"credit_policy",
)

// CreditPolicyService 提供积分策略与模型价格能力。
type CreditPolicyService struct{}

// NewCreditPolicyService 创建积分策略服务。
func NewCreditPolicyService() *CreditPolicyService {
	return &CreditPolicyService{}
}

// CalculateCredits 根据真实Token消耗计算积分。
func (service *CreditPolicyService) CalculateCredits(
	ctx context.Context,
	modelUsed string,
	inputTokens int,
	outputTokens int,
	totalTokens int,
	schoolID *string,
	latencyMs int64,
) *models.CreditCalculation {
	// 部分兼容接口只返回total_tokens。
	// 无输入输出拆分时继续使用原系统6:4估算。
	if inputTokens == 0 &&
		outputTokens == 0 &&
		totalTokens > 0 {
		inputTokens =
			totalTokens * 6 / 10
		outputTokens =
			totalTokens - inputTokens
	}

	if inputTokens == 0 &&
		outputTokens == 0 {
		return &models.CreditCalculation{
			ModelName: modelUsed,
			LatencyMs: latencyMs,
		}
	}

	// 先查数据库中的精确模型名。
	price, err :=
		repository.GetModelPriceByName(
			ctx,
			modelUsed,
		)

	if err != nil {
		if !errors.Is(
			err,
			repository.ErrModelPriceNotFound,
		) {
			cpLog.Warn(
				"查询模型单价失败，使用实际模型兜底",
				"model",
				modelUsed,
				"error",
				err,
			)
		}

		price = service.estimateModelPrice(
			modelUsed,
			inputTokens,
		)
	} else {
		// 即使数据库中已有基础价，
		// 仍需处理Sonnet 5日期切换和Gemini Pro阶梯价。
		price = service.applyActualModelPrice(
			modelUsed,
			inputTokens,
			price,
		)
	}

	if price == nil {
		cpLog.Warn(
			"模型无可用单价，积分为0",
			"model",
			modelUsed,
		)

		return &models.CreditCalculation{
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
			ModelName:    modelUsed,
			LatencyMs:    latencyMs,
		}
	}

	costUSD := price.CalculateCostUSD(
		inputTokens,
		outputTokens,
	)

	policy := service.GetEffectivePolicy(
		ctx,
		schoolID,
	)

	credits :=
		policy.CalculateCredits(costUSD)

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

// GetEffectivePolicy 按学校、系统、默认顺序取得有效策略。
func (service *CreditPolicyService) GetEffectivePolicy(
	ctx context.Context,
	schoolID *string,
) *models.CreditPolicy {
	if schoolID != nil &&
		*schoolID != "" {
		policy, err :=
			repository.GetSchoolCreditPolicy(
				ctx,
				*schoolID,
			)

		if err == nil &&
			policy != nil {
			return policy
		}
	}

	policy, err :=
		repository.GetSystemCreditPolicy(ctx)

	if err == nil &&
		policy != nil {
		return policy
	}

	return &models.CreditPolicy{
		ExchangeRate: models.DefaultExchangeRate,
		Multiplier:   models.DefaultMultiplier,
	}
}

// GetSystemPolicy 获取系统级策略。
func (service *CreditPolicyService) GetSystemPolicy(
	ctx context.Context,
) (*models.CreditPolicy, error) {
	return repository.GetSystemCreditPolicy(
		ctx,
	)
}

// UpdateSystemPolicy 更新系统级策略。
func (service *CreditPolicyService) UpdateSystemPolicy(
	ctx context.Context,
	request *models.UpdateCreditPolicyRequest,
	updatedBy string,
) (*models.CreditPolicy, error) {
	current, _ :=
		repository.GetSystemCreditPolicy(
			ctx,
		)

	exchangeRate :=
		models.DefaultExchangeRate

	multiplier :=
		models.DefaultMultiplier

	description := ""

	if current != nil {
		exchangeRate =
			current.ExchangeRate
		multiplier =
			current.Multiplier
		description =
			current.Description
	}

	if request.ExchangeRate != nil {
		exchangeRate =
			*request.ExchangeRate
	}

	if request.Multiplier != nil {
		multiplier =
			*request.Multiplier
	}

	if request.Description != nil {
		description =
			*request.Description
	}

	return repository.UpsertCreditPolicy(
		ctx,
		models.PolicyScopeSystem,
		nil,
		exchangeRate,
		multiplier,
		description,
		&updatedBy,
	)
}

// GetSchoolPolicy 获取学校级策略。
func (service *CreditPolicyService) GetSchoolPolicy(
	ctx context.Context,
	schoolID string,
) (*models.CreditPolicy, error) {
	return repository.GetSchoolCreditPolicy(
		ctx,
		schoolID,
	)
}

// UpdateSchoolPolicy 更新学校级策略。
func (service *CreditPolicyService) UpdateSchoolPolicy(
	ctx context.Context,
	schoolID string,
	request *models.UpdateCreditPolicyRequest,
	updatedBy string,
) (*models.CreditPolicy, error) {
	current, _ :=
		repository.GetSchoolCreditPolicy(
			ctx,
			schoolID,
		)

	exchangeRate :=
		models.DefaultExchangeRate

	multiplier :=
		models.DefaultMultiplier

	description := ""

	if current != nil {
		exchangeRate =
			current.ExchangeRate
		multiplier =
			current.Multiplier
		description =
			current.Description
	}

	if request.ExchangeRate != nil {
		exchangeRate =
			*request.ExchangeRate
	}

	if request.Multiplier != nil {
		multiplier =
			*request.Multiplier
	}

	if request.Description != nil {
		description =
			*request.Description
	}

	return repository.UpsertCreditPolicy(
		ctx,
		models.PolicyScopeSchool,
		&schoolID,
		exchangeRate,
		multiplier,
		description,
		&updatedBy,
	)
}

// DeleteSchoolPolicy 删除学校级策略。
func (service *CreditPolicyService) DeleteSchoolPolicy(
	ctx context.Context,
	schoolID string,
) error {
	return repository.DeleteSchoolCreditPolicy(
		ctx,
		schoolID,
	)
}

// ListPolicies 列出全部积分策略。
func (service *CreditPolicyService) ListPolicies(
	ctx context.Context,
) ([]*models.CreditPolicyListItem, error) {
	return repository.ListCreditPolicies(ctx)
}

// ListModelPrices 列出文本模型单价。
func (service *CreditPolicyService) ListModelPrices(
	ctx context.Context,
	includeInactive bool,
) ([]models.ModelPrice, error) {
	return repository.ListModelPrices(
		ctx,
		includeInactive,
	)
}

// CreateModelPrice 创建文本模型单价。
func (service *CreditPolicyService) CreateModelPrice(
	ctx context.Context,
	request *models.CreateModelPriceRequest,
	updatedBy string,
) (*models.ModelPrice, error) {
	if request.ModelName == "" {
		return nil,
			fmt.Errorf("模型名称不能为空")
	}

	if request.Provider == "" {
		return nil,
			fmt.Errorf("供应商不能为空")
	}

	price := &models.ModelPrice{
		ModelName:       request.ModelName,
		Provider:        request.Provider,
		CostPer1kInput:  request.CostPer1kInput,
		CostPer1kOutput: request.CostPer1kOutput,
		DisplayName:     request.DisplayName,
		IsActive:        true,
		UpdatedBy:       &updatedBy,
	}

	if err :=
		repository.CreateModelPrice(
			ctx,
			price,
		); err != nil {
		if errors.Is(
			err,
			repository.ErrModelPriceDuplicate,
		) {
			return nil, err
		}

		return nil,
			fmt.Errorf(
				"创建模型单价失败: %w",
				err,
			)
	}

	return price, nil
}

// UpdateModelPrice 更新文本模型单价。
func (service *CreditPolicyService) UpdateModelPrice(
	ctx context.Context,
	id string,
	request *models.UpdateModelPriceRequest,
	updatedBy string,
) (*models.ModelPrice, error) {
	return repository.UpdateModelPrice(
		ctx,
		id,
		request.CostPer1kInput,
		request.CostPer1kOutput,
		request.DisplayName,
		request.IsActive,
		&updatedBy,
	)
}

// DeleteModelPrice 删除文本模型单价。
func (service *CreditPolicyService) DeleteModelPrice(
	ctx context.Context,
	id string,
) error {
	return repository.DeleteModelPrice(
		ctx,
		id,
	)
}

// Simulate 模拟一次积分计算。
func (service *CreditPolicyService) Simulate(
	ctx context.Context,
	request *models.SimulateCreditRequest,
) (*models.CreditCalculation, error) {
	if request.ModelName == "" {
		return nil,
			fmt.Errorf("模型名称不能为空")
	}

	if request.InputTokens <= 0 &&
		request.OutputTokens <= 0 {
		return nil,
			fmt.Errorf(
				"输入/输出token数至少填一个",
			)
	}

	calculation :=
		service.CalculateCredits(
			ctx,
			request.ModelName,
			request.InputTokens,
			request.OutputTokens,
			0,
			request.SchoolID,
			0,
		)

	return calculation, nil
}

// GetModelPreviews 返回当前基础价格的积分预览。
//
// 阶梯模型在真实调用时仍会根据input_tokens应用对应档位。
func (service *CreditPolicyService) GetModelPreviews(
	ctx context.Context,
) ([]models.ModelPricePreview, error) {
	prices, err :=
		repository.ListModelPrices(
			ctx,
			false,
		)

	if err != nil {
		return nil, err
	}

	policy :=
		service.GetEffectivePolicy(
			ctx,
			nil,
		)

	previews :=
		make(
			[]models.ModelPricePreview,
			0,
			len(prices),
		)

	for _, price := range prices {
		previews = append(
			previews,
			models.ModelPricePreview{
				ModelName:       price.ModelName,
				Provider:        price.Provider,
				DisplayName:     price.DisplayName,
				CostPer1kInput:  price.CostPer1kInput,
				CostPer1kOutput: price.CostPer1kOutput,
				CreditsPer1kInput: price.CostPer1kInput *
					policy.ExchangeRate *
					policy.Multiplier,
				CreditsPer1kOutput: price.CostPer1kOutput *
					policy.ExchangeRate *
					policy.Multiplier,
			},
		)
	}

	return previews, nil
}
