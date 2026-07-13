package services

// lesson_plan_recipe_selection.go — 开始备课时的配方三态解释与回执恢复
//
// 本文件集中管理配方选择的三态语义，避免三态逻辑继续堆入
// lesson_plan_gen_service.go和上下文回执大文件。
//
// 三态定义：
//   - auto：平台根据学校默认、教研组共享和学科规则自动选择；
//   - selected：老师明确选择了一个recipe_id；
//   - none：老师明确不使用配方，只使用系统阶段骨架。
//
// 兼容规则：
//   - 旧客户端没有recipe_mode但带recipe_id，视为selected；
//   - 旧客户端没有recipe_mode也没有recipe_id，视为auto。
//
// 持久化方式：
// 配方选择方式写入第一条AI开场消息的metadata，并随conversation_log持久化。
// 这样不增加数据库字段，也能供实时SSE、断线补齐、历史恢复和每轮回执共用。

import (
	"context"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

const recipeSelectionModeMetadataKey = "recipe_selection_mode"

// normalizeStartRecipeSelection 统一解释开始备课请求中的配方模式。
//
// 本函数会同时规范化req.RecipeMode和req.RecipeID，调用方只需根据返回值决定
// 是否执行自动解析，后续创建教案和初始化阶段仍复用既有RecipeID链路。
func normalizeStartRecipeSelection(
	req *models.StartConversationRequest,
) models.RecipeSelectionMode {
	if req == nil {
		return models.RecipeSelectionModeAuto
	}

	rawMode := models.RecipeSelectionMode(
		strings.TrimSpace(string(req.RecipeMode)),
	)
	recipeID := strings.TrimSpace(req.RecipeID)

	switch rawMode {
	case models.RecipeSelectionModeAuto:
		// auto模式的唯一数据来源是后端自动解析，清除前端可能残留的旧ID。
		req.RecipeID = ""
		req.RecipeMode = models.RecipeSelectionModeAuto
		return models.RecipeSelectionModeAuto

	case models.RecipeSelectionModeSelected:
		if recipeID == "" {
			// selected却没有目标ID属于不完整请求。
			// 为避免静默自动挂载另一个配方，按fail-closed原则降级为none。
			req.RecipeID = ""
			req.RecipeMode = models.RecipeSelectionModeNone
			lpGenLog.Warn(
				"开始备课请求声明selected但没有recipe_id，已按明确不使用处理",
				"recipe_mode", rawMode,
			)
			return models.RecipeSelectionModeNone
		}

		req.RecipeID = recipeID
		req.RecipeMode = models.RecipeSelectionModeSelected
		return models.RecipeSelectionModeSelected

	case models.RecipeSelectionModeNone:
		// none拥有最高否决权，即使请求误带recipe_id也必须清除。
		req.RecipeID = ""
		req.RecipeMode = models.RecipeSelectionModeNone
		return models.RecipeSelectionModeNone

	case "":
		// 旧客户端兼容路径。
		if recipeID != "" {
			req.RecipeID = recipeID
			req.RecipeMode = models.RecipeSelectionModeSelected
			return models.RecipeSelectionModeSelected
		}

		req.RecipeID = ""
		req.RecipeMode = models.RecipeSelectionModeAuto
		return models.RecipeSelectionModeAuto

	default:
		// 未知值不直接报错，避免新旧前端版本交错时阻断建会话。
		// 有recipe_id按旧显式选择解释，没有则按旧自动选择解释。
		lpGenLog.Warn(
			"开始备课请求包含未知recipe_mode，已按兼容规则解释",
			"recipe_mode", rawMode,
			"has_recipe_id", recipeID != "",
		)

		if recipeID != "" {
			req.RecipeID = recipeID
			req.RecipeMode = models.RecipeSelectionModeSelected
			return models.RecipeSelectionModeSelected
		}

		req.RecipeID = ""
		req.RecipeMode = models.RecipeSelectionModeAuto
		return models.RecipeSelectionModeAuto
	}
}

// resolveRecipeSelectionModeForReceipt 从完整conversation_log恢复本会话的配方选择方式。
//
// 新会话会从开场消息metadata准确读出auto/selected/none。
// 存量历史教案没有该metadata时采用兼容推断：
//   - 已有关联recipe_id：视为selected；
//   - 没有关联recipe_id：视为auto。
//
// 存量会话的自动匹配与老师选择无法从旧数据完全区分，因此只做保守推断，
// 不影响实际提示词装配和教案内容。
func resolveRecipeSelectionModeForReceipt(
	ctx context.Context,
	lp *models.LessonPlan,
) models.RecipeSelectionMode {
	if lp == nil {
		return models.RecipeSelectionModeAuto
	}

	messages, err := repository.GetConversationLog(ctx, lp.ID)
	if err == nil {
		for _, message := range messages {
			if message == nil || message.Metadata == nil {
				continue
			}

			rawValue, exists := message.Metadata[recipeSelectionModeMetadataKey]
			if !exists {
				continue
			}

			modeText, ok := rawValue.(string)
			if !ok {
				continue
			}

			mode := models.RecipeSelectionMode(strings.TrimSpace(modeText))
			switch mode {
			case models.RecipeSelectionModeAuto,
				models.RecipeSelectionModeSelected,
				models.RecipeSelectionModeNone:
				return mode
			}
		}
	}

	if lp.RecipeID != nil && strings.TrimSpace(*lp.RecipeID) != "" {
		return models.RecipeSelectionModeSelected
	}

	return models.RecipeSelectionModeAuto
}
