package services

import (
	"testing"

	"tedna/internal/models"
)

// TestNormalizeStartRecipeSelection 锁定配方三态及旧客户端兼容契约。
// 本测试不访问数据库、不调用AI，可独立稳定运行。
func TestNormalizeStartRecipeSelection(t *testing.T) {
	tests := []struct {
		name         string
		mode         models.RecipeSelectionMode
		recipeID     string
		wantMode     models.RecipeSelectionMode
		wantRecipeID string
	}{
		{
			name:         "旧客户端无模式无ID继续自动选择",
			mode:         "",
			recipeID:     "",
			wantMode:     models.RecipeSelectionModeAuto,
			wantRecipeID: "",
		},
		{
			name:         "旧客户端带ID视为老师明确选择",
			mode:         "",
			recipeID:     "  recipe-old  ",
			wantMode:     models.RecipeSelectionModeSelected,
			wantRecipeID: "recipe-old",
		},
		{
			name:         "auto清除前端残留ID",
			mode:         models.RecipeSelectionModeAuto,
			recipeID:     "stale-recipe",
			wantMode:     models.RecipeSelectionModeAuto,
			wantRecipeID: "",
		},
		{
			name:         "selected保留并清洗老师选择的ID",
			mode:         models.RecipeSelectionModeSelected,
			recipeID:     "  recipe-selected  ",
			wantMode:     models.RecipeSelectionModeSelected,
			wantRecipeID: "recipe-selected",
		},
		{
			name:         "selected缺少ID时fail-closed为none",
			mode:         models.RecipeSelectionModeSelected,
			recipeID:     "",
			wantMode:     models.RecipeSelectionModeNone,
			wantRecipeID: "",
		},
		{
			name:         "none即使误带ID也明确清除",
			mode:         models.RecipeSelectionModeNone,
			recipeID:     "must-not-use",
			wantMode:     models.RecipeSelectionModeNone,
			wantRecipeID: "",
		},
		{
			name:         "未知模式带ID按旧显式选择兼容",
			mode:         models.RecipeSelectionMode("unknown"),
			recipeID:     "recipe-compatible",
			wantMode:     models.RecipeSelectionModeSelected,
			wantRecipeID: "recipe-compatible",
		},
		{
			name:         "未知模式无ID按旧自动选择兼容",
			mode:         models.RecipeSelectionMode("unknown"),
			recipeID:     "",
			wantMode:     models.RecipeSelectionModeAuto,
			wantRecipeID: "",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			request := &models.StartConversationRequest{
				RecipeMode: testCase.mode,
				RecipeID:   testCase.recipeID,
			}

			gotMode := normalizeStartRecipeSelection(request)

			if gotMode != testCase.wantMode {
				t.Fatalf(
					"返回模式不一致：got=%q want=%q",
					gotMode,
					testCase.wantMode,
				)
			}

			if request.RecipeMode != testCase.wantMode {
				t.Fatalf(
					"请求内模式未规范化：got=%q want=%q",
					request.RecipeMode,
					testCase.wantMode,
				)
			}

			if request.RecipeID != testCase.wantRecipeID {
				t.Fatalf(
					"请求内配方ID不一致：got=%q want=%q",
					request.RecipeID,
					testCase.wantRecipeID,
				)
			}
		})
	}
}
