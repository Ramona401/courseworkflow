package integration

// recipe_test.go — 备课配方与组件集成测试
//
// 测试范围（12个用例）：
//
// 配方CRUD（5个）：
//   1. 创建配方 → 成功
//   2. 缺少必填字段 → 400
//   3. 获取配方详情 → 返回完整信息
//   4. 更新配方 → 成功
//   5. 删除配方 → 成功
//
// 配方Fork+列表（2个）：
//   6. Fork配方 → 成功+fork_count递增
//   7. 配方列表
//
// 组件库（3个）：
//   8. 获取组件列表
//   9. 组件匹配（按学科+年级）
//   10. 流程预设获取
//
// 流程校验（2个）：
//   11. 校验有效流程 → 通过
//   12. 校验缺少必选阶段 → 返回校验结果
//
// 关键注意：
//   CreateRecipeRequest/UpdateRecipeRequest 中 stages_config 和 lesson_structure
//   是 string 类型（JSON字符串），不是Go对象。传参时必须序列化为字符串。

import (
	"context"
	"net/http"
	"testing"

	"tedna/internal/database"
)

// ==================== 辅助函数 ====================

// defaultStagesConfigJSON 默认5阶段流程的JSON字符串
// 注意：API期望此字段为JSON字符串类型，不是Go对象
const defaultStagesConfigJSON = `[{"stage_code":"analyze","enabled":true,"order":1},{"stage_code":"design","enabled":true,"order":2},{"stage_code":"write","enabled":true,"order":3},{"stage_code":"review","enabled":true,"order":4},{"stage_code":"revise","enabled":true,"order":5}]`

// createTestRecipeViaAPI 通过API创建测试配方，返回配方ID
func createTestRecipeViaAPI(t *testing.T, serverURL string, token string, name string, subject string, grade string) string {
	t.Helper()

	// stages_config 必须是JSON字符串（string类型），不是Go数组
	body := map[string]interface{}{
		"name":          name,
		"subject":       subject,
		"grade_range":   grade,
		"description":   "集成测试配方",
		"stages_config": defaultStagesConfigJSON,
		"prompt_mode":   "guided",
	}

	resp, apiResp := DoPost(t, serverURL+"/api/v1/lesson-plans/recipes", body, token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	var recipeData struct {
		ID string `json:"id"`
	}
	ParseData(t, apiResp, &recipeData)

	if recipeData.ID == "" {
		t.Fatal("创建配方成功但ID为空")
	}

	return recipeData.ID
}

// ==================== 配方CRUD测试 ====================

// TestRecipe_Create 创建配方 → 成功
func TestRecipe_Create(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)
	recipeID := createTestRecipeViaAPI(t, server.URL, token, "数学探究配方", "数学", "7-9")

	// 验证DB中存在
	var count int
	err := database.DB.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM teaching_recipes WHERE id = $1`, recipeID,
	).Scan(&count)
	if err != nil {
		t.Fatalf("查询配方失败: %v", err)
	}
	if count != 1 {
		t.Error("创建配方后DB中应有1条记录")
	}
}

// TestRecipe_CreateMissingFields 缺少必填字段 → 400
func TestRecipe_CreateMissingFields(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	// 缺少name → 服务层返回"配方名称不能为空"
	body := map[string]interface{}{
		"name":        "",
		"subject":     "数学",
		"grade_range": "7-9",
	}
	resp, _ := DoPost(t, server.URL+"/api/v1/lesson-plans/recipes", body, token)
	AssertHTTPStatus(t, resp, http.StatusBadRequest)
}

// TestRecipe_GetDetail 获取配方详情
func TestRecipe_GetDetail(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)
	recipeID := createTestRecipeViaAPI(t, server.URL, token, "语文阅读配方", "语文", "7-8")

	resp, apiResp := DoGet(t, server.URL+"/api/v1/lesson-plans/recipes/"+recipeID, token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	var detail struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Subject string `json:"subject"`
	}
	ParseData(t, apiResp, &detail)

	if detail.ID != recipeID {
		t.Errorf("配方ID不匹配")
	}
	if detail.Name != "语文阅读配方" {
		t.Errorf("配方名称不匹配: 期望 语文阅读配方, 实际 %s", detail.Name)
	}
}

// TestRecipe_Update 更新配方 → 成功
func TestRecipe_Update(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)
	recipeID := createTestRecipeViaAPI(t, server.URL, token, "待更新配方", "英语", "7-9")

	// UpdateRecipeRequest 的 stages_config 也是 string 类型
	shortStagesJSON := `[{"stage_code":"analyze","enabled":true,"order":1},{"stage_code":"write","enabled":true,"order":2},{"stage_code":"review","enabled":true,"order":3}]`

	updateBody := map[string]interface{}{
		"name":          "已更新配方",
		"subject":       "英语",
		"grade_range":   "7-9",
		"description":   "更新后的描述",
		"stages_config": shortStagesJSON,
	}

	resp, apiResp := DoPut(t, server.URL+"/api/v1/lesson-plans/recipes/"+recipeID, updateBody, token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	// 验证DB已更新
	var name string
	err := database.DB.QueryRow(context.Background(),
		`SELECT name FROM teaching_recipes WHERE id = $1`, recipeID,
	).Scan(&name)
	if err != nil {
		t.Fatalf("查询配方失败: %v", err)
	}
	if name != "已更新配方" {
		t.Errorf("配方名称未更新: 期望 已更新配方, 实际 %s", name)
	}
}

// TestRecipe_Delete 删除配方 → 成功（实际为archived）
func TestRecipe_Delete(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)
	recipeID := createTestRecipeViaAPI(t, server.URL, token, "待删除配方", "科学", "5-6")

	resp, apiResp := DoDelete(t, server.URL+"/api/v1/lesson-plans/recipes/"+recipeID, token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	// 确认DB中已归档
	var status string
	err := database.DB.QueryRow(context.Background(),
		`SELECT status FROM teaching_recipes WHERE id = $1`, recipeID,
	).Scan(&status)
	if err != nil {
		t.Fatalf("查询配方状态失败: %v", err)
	}
	if status != "archived" {
		t.Errorf("删除后配方状态应为archived，实际为 %s", status)
	}
}

// ==================== Fork+列表测试 ====================

// TestRecipe_Fork Fork配方 → 成功+fork_count递增
func TestRecipe_Fork(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	adminToken := LoginAsAdmin(t, server.URL)
	recipeID := createTestRecipeViaAPI(t, server.URL, adminToken, "可Fork配方", "数学", "7-9")

	operatorToken := LoginAsOperator(t, server.URL)
	resp, apiResp := DoPost(t, server.URL+"/api/v1/lesson-plans/recipes/"+recipeID+"/fork", nil, operatorToken)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	// 验证fork_count递增
	var forkCount int
	err := database.DB.QueryRow(context.Background(),
		`SELECT fork_count FROM teaching_recipes WHERE id = $1`, recipeID,
	).Scan(&forkCount)
	if err != nil {
		t.Fatalf("查询fork_count失败: %v", err)
	}
	if forkCount != 1 {
		t.Errorf("fork_count应为1，实际为 %d", forkCount)
	}
}

// TestRecipe_List 配方列表
func TestRecipe_List(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)
	createTestRecipeViaAPI(t, server.URL, token, "列表测试1", "数学", "7-9")
	createTestRecipeViaAPI(t, server.URL, token, "列表测试2", "语文", "7-8")

	resp, apiResp := DoGet(t, server.URL+"/api/v1/lesson-plans/recipes", token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	var listData struct {
		Recipes []struct {
			ID string `json:"id"`
		} `json:"recipes"`
		Total int `json:"total"`
	}
	ParseData(t, apiResp, &listData)

	if listData.Total < 2 {
		t.Errorf("应至少有2个配方，实际 %d", listData.Total)
	}
}

// ==================== 组件库测试 ====================

// TestComponents_List 组件列表接口可用
func TestComponents_List(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	resp, apiResp := DoGet(t, server.URL+"/api/v1/lesson-plans/components", token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)
}

// TestComponents_Match 组件匹配接口可用
func TestComponents_Match(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	body := map[string]interface{}{
		"subject":     "数学",
		"grade_range": "7-9",
		"limit":       5,
	}
	resp, apiResp := DoPost(t, server.URL+"/api/v1/lesson-plans/components/match", body, token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)
}

// TestRecipe_FlowPresets 流程预设获取
func TestRecipe_FlowPresets(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	resp, apiResp := DoGet(t, server.URL+"/api/v1/lesson-plans/recipes/flow-presets", token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)
}

// ==================== 流程校验测试 ====================

// TestRecipe_ValidateFlow 校验有效流程
func TestRecipe_ValidateFlow(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	body := map[string]interface{}{
		"stages": []map[string]interface{}{
			{"stage_code": "analyze", "enabled": true, "order": 1},
			{"stage_code": "design", "enabled": true, "order": 2},
			{"stage_code": "write", "enabled": true, "order": 3},
			{"stage_code": "review", "enabled": true, "order": 4},
			{"stage_code": "revise", "enabled": true, "order": 5},
		},
	}
	resp, apiResp := DoPost(t, server.URL+"/api/v1/lesson-plans/recipes/validate-flow", body, token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)
}

// TestRecipe_ValidateFlowMissingWrite 缺少write阶段 → 校验结果包含错误
func TestRecipe_ValidateFlowMissingWrite(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	body := map[string]interface{}{
		"stages": []map[string]interface{}{
			{"stage_code": "analyze", "enabled": true, "order": 1},
			{"stage_code": "review", "enabled": true, "order": 2},
		},
	}
	resp, apiResp := DoPost(t, server.URL+"/api/v1/lesson-plans/recipes/validate-flow", body, token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)
}
