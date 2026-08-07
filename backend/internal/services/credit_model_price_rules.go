package services

// credit_model_price_rules.go — 文本模型实际价格与安全兜底规则。
//
// 设计目的：
//   1. 兼容聚合网关返回的带供应商前缀、版本日期和不同词序的模型名；
//   2. 修正旧代码把所有Opus都按15/75美元每百万Token计价的问题；
//   3. 支持Gemini Pro超过200K输入Token后的价格阶梯；
//   4. 支持Claude Sonnet 5推广价到期后自动切换标准价；
//   5. 即使数据库暂时没有版本化模型记录，也不再套用过时价格。
//
// 数据库中的模型价格仍用于后台展示和手工维护。
// 本文件的规则只对明确识别的模型应用精确覆盖。

import (
	"strings"
	"time"

	"tedna/internal/models"
)

const (
	// Qwen3.7-Max当前五折价格为人民币6元/18元每百万Token。
	// 按系统当前7.2人民币/美元汇率转换。
	qwen37InputUSDPerMillion  = 6.0 / 7.2
	qwen37OutputUSDPerMillion = 18.0 / 7.2
)

// estimateModelPrice 在数据库没有精确模型名时使用实际模型规则。
func (service *CreditPolicyService) estimateModelPrice(
	modelName string,
	inputTokens int,
) *models.ModelPrice {
	return resolveActualModelPrice(
		modelName,
		inputTokens,
		time.Now(),
		nil,
	)
}

// applyActualModelPrice 对明确识别的模型应用当前官方价格和阶梯。
//
// configured不为空时保留数据库记录的ID、显示名等字段，
// 只覆盖供应商和输入/输出单价。
func (service *CreditPolicyService) applyActualModelPrice(
	modelName string,
	inputTokens int,
	configured *models.ModelPrice,
) *models.ModelPrice {
	return resolveActualModelPrice(
		modelName,
		inputTokens,
		time.Now(),
		configured,
	)
}

// resolveActualModelPrice 是可测试的实际价格解析入口。
func resolveActualModelPrice(
	modelName string,
	inputTokens int,
	now time.Time,
	configured *models.ModelPrice,
) *models.ModelPrice {
	normalized := normalizeActualModelName(modelName)

	setPrice := func(
		provider string,
		inputUSDPerMillion float64,
		outputUSDPerMillion float64,
	) *models.ModelPrice {
		result := models.ModelPrice{
			ModelName: modelName,
			Provider:  provider,
			IsActive:  true,
		}

		if configured != nil {
			result = *configured
			result.ModelName = modelName
		}

		result.Provider = provider
		result.CostPer1kInput =
			inputUSDPerMillion / 1000.0
		result.CostPer1kOutput =
			outputUSDPerMillion / 1000.0

		return &result
	}

	// ==================== 千问 ====================

	if strings.Contains(
		normalized,
		"qwen3-7-max",
	) {
		return setPrice(
			"qwen",
			qwen37InputUSDPerMillion,
			qwen37OutputUSDPerMillion,
		)
	}

	// ==================== Google Gemini ====================

	// Flash-Lite必须放在Flash规则之前，避免被宽匹配覆盖。
	if strings.Contains(
		normalized,
		"gemini-3-5-flash-lite",
	) {
		return setPrice(
			"google",
			0.30,
			2.50,
		)
	}

	if strings.Contains(
		normalized,
		"gemini-3-5-flash",
	) {
		return setPrice(
			"google",
			1.50,
			9.00,
		)
	}

	if strings.Contains(
		normalized,
		"gemini-3-6-flash",
	) {
		return setPrice(
			"google",
			1.50,
			7.50,
		)
	}

	if strings.Contains(
		normalized,
		"gemini-3-1-flash-lite",
	) {
		return setPrice(
			"google",
			0.25,
			1.50,
		)
	}

	if strings.Contains(
		normalized,
		"gemini-3-1-pro-preview",
	) {
		if inputTokens > 200000 {
			return setPrice(
				"google",
				4.00,
				18.00,
			)
		}

		return setPrice(
			"google",
			2.00,
			12.00,
		)
	}

	if strings.Contains(
		normalized,
		"gemini-2-5-pro",
	) {
		if inputTokens > 200000 {
			return setPrice(
				"google",
				2.50,
				15.00,
			)
		}

		return setPrice(
			"google",
			1.25,
			10.00,
		)
	}

	if strings.Contains(
		normalized,
		"gemini-2-5-flash",
	) {
		return setPrice(
			"google",
			0.30,
			2.50,
		)
	}

	if strings.Contains(
		normalized,
		"gemini-2-0-flash",
	) {
		return setPrice(
			"google",
			0.10,
			0.40,
		)
	}

	// ==================== Anthropic Claude ====================

	// 旧Claude Opus 4版本仍保持15/75美元，必须先于新版Opus匹配。
	if containsActualModelFragment(
		normalized,
		"claude-opus-4-20250514",
		"claude-4-opus-20250514",
	) {
		return setPrice(
			"anthropic",
			15.00,
			75.00,
		)
	}

	if containsActualModelFragment(
		normalized,
		"claude-opus-4-8",
		"claude-4-8-opus",
		"claude-opus-4-7",
		"claude-4-7-opus",
		"claude-opus-4-6",
		"claude-4-6-opus",
		"claude-opus-4-5",
		"claude-4-5-opus",
		"claude-opus-5",
		"claude-5-opus",
	) {
		return setPrice(
			"anthropic",
			5.00,
			25.00,
		)
	}

	if strings.Contains(
		normalized,
		"claude-sonnet-5",
	) {
		promotionCutoff := time.Date(
			2026,
			time.September,
			1,
			0,
			0,
			0,
			0,
			time.FixedZone(
				"UTC+8",
				8*60*60,
			),
		)

		if now.Before(promotionCutoff) {
			return setPrice(
				"anthropic",
				2.00,
				10.00,
			)
		}

		return setPrice(
			"anthropic",
			3.00,
			15.00,
		)
	}

	if containsActualModelFragment(
		normalized,
		"claude-sonnet-4-6",
		"claude-4-6-sonnet",
		"claude-sonnet-4-5",
		"claude-4-5-sonnet",
		"claude-sonnet-4-20250514",
	) {
		return setPrice(
			"anthropic",
			3.00,
			15.00,
		)
	}

	// 旧本地claude-haiku-4-20250514沿用原0.8/4价格。
	if strings.Contains(
		normalized,
		"claude-haiku-4-20250514",
	) {
		return setPrice(
			"anthropic",
			0.80,
			4.00,
		)
	}

	if containsActualModelFragment(
		normalized,
		"claude-haiku-4-5",
		"claude-4-5-haiku",
	) {
		return setPrice(
			"anthropic",
			1.00,
			5.00,
		)
	}

	// ==================== 安全通用兜底 ====================

	// 数据库已配置但不属于上述明确模型时，保持管理员配置。
	if configured != nil {
		result := *configured
		return &result
	}

	// 供应商家族可识别但具体版本未知时使用当前中档标准，
	// 避免未知模型静默变成0积分。
	switch {
	case strings.Contains(normalized, "opus"):
		return setPrice(
			"anthropic",
			5.00,
			25.00,
		)

	case strings.Contains(normalized, "haiku"):
		return setPrice(
			"anthropic",
			1.00,
			5.00,
		)

	case strings.Contains(normalized, "sonnet"),
		strings.Contains(normalized, "claude"):
		return setPrice(
			"anthropic",
			3.00,
			15.00,
		)

	case strings.Contains(normalized, "gemini"):
		return setPrice(
			"google",
			1.50,
			9.00,
		)
	}

	// 完全未知模型维持原系统的中档安全兜底，
	// 同时由调用方记录结构化告警，等待管理员补充精确价格。
	return setPrice(
		"unknown",
		3.00,
		15.00,
	)
}

func normalizeActualModelName(
	modelName string,
) string {
	replacer := strings.NewReplacer(
		"_",
		"-",
		".",
		"-",
	)

	return replacer.Replace(
		strings.ToLower(
			strings.TrimSpace(modelName),
		),
	)
}

func containsActualModelFragment(
	modelName string,
	fragments ...string,
) bool {
	for _, fragment := range fragments {
		if strings.Contains(
			modelName,
			fragment,
		) {
			return true
		}
	}

	return false
}
