package integration

// prompt_test.go — 提示词管理集成测试
//
// 测试范围（12个用例）：
//
// 提示词CRUD（5个）：
//   1. 列表查询（初始无数据）→ 空列表
//   2. 创建提示词（通过Update接口创建首个版本）→ 成功
//   3. 按key查询 → 返回正确内容
//   4. 更新提示词 → 版本号递增为2
//   5. 更新内容为空 → 400
//
// 版本管理（4个）：
//   6. 版本历史 → 按版本号倒序返回
//   7. 回滚到旧版本 → 旧版本变为当前
//   8. 回滚已生效版本 → 400（已经是当前版本）
//   9. 查询无效key → 400
//
// 权限控制（3个）：
//   10. 非admin查询提示词列表 → 403
//   11. 非admin更新提示词 → 403
//   12. 未登录访问 → 401
//
// 注意：
//   - 提示词表初始为空（CleanAndSeed不插入提示词种子数据）
//   - 使用 prompt_a 作为测试key（属于ValidPromptKeys）
//   - Update接口会自动创建新版本（如果key不存在则创建v1）

import (
	"context"
	"net/http"
	"testing"

	"tedna/internal/database"
)

// ==================== 辅助函数 ====================

// createTestPromptViaAPI 通过Update接口创建测试提示词（首个版本）
// 返回创建的提示词ID
func createTestPromptViaAPI(t *testing.T, serverURL string, token string, key string, content string) string {
	t.Helper()

	body := map[string]interface{}{
		"content": content,
	}

	resp, apiResp := DoPut(t, serverURL+"/api/v1/prompts/"+key, body, token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	var promptData struct {
		ID      string `json:"id"`
		Version int    `json:"version"`
	}
	ParseData(t, apiResp, &promptData)

	if promptData.ID == "" {
		t.Fatal("创建提示词成功但ID为空")
	}

	return promptData.ID
}

// ==================== 提示词CRUD测试 ====================

// TestPrompt_ListEmpty 列表查询（初始无数据）→ 空列表
func TestPrompt_ListEmpty(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	resp, apiResp := DoGet(t, server.URL+"/api/v1/prompts", token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	var listData struct {
		Total int `json:"total"`
	}
	ParseData(t, apiResp, &listData)

	if listData.Total != 0 {
		t.Errorf("初始提示词列表应为空，实际 total=%d", listData.Total)
	}
}

// TestPrompt_CreateViaUpdate 通过Update接口创建首个版本 → 成功
func TestPrompt_CreateViaUpdate(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	promptID := createTestPromptViaAPI(t, server.URL, token, "prompt_a", "这是Scanner扫描定位提示词的测试内容V1")

	// 验证DB中存在
	var count int
	err := database.DB.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM prompts WHERE id = $1 AND is_current = true`, promptID,
	).Scan(&count)
	if err != nil {
		t.Fatalf("查询提示词失败: %v", err)
	}
	if count != 1 {
		t.Error("创建提示词后DB中应有1条is_current=true记录")
	}

	// 验证版本号为1
	var version int
	err = database.DB.QueryRow(context.Background(),
		`SELECT version FROM prompts WHERE id = $1`, promptID,
	).Scan(&version)
	if err != nil {
		t.Fatalf("查询提示词版本失败: %v", err)
	}
	if version != 1 {
		t.Errorf("首个版本号应为1，实际 %d", version)
	}
}

// TestPrompt_GetByKey 按key查询 → 返回正确内容
func TestPrompt_GetByKey(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	createTestPromptViaAPI(t, server.URL, token, "prompt_b", "Evaluator评估打分提示词内容")

	// 按key查询
	resp, apiResp := DoGet(t, server.URL+"/api/v1/prompts/prompt_b", token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	var detail struct {
		PromptKey  string `json:"prompt_key"`
		PromptName string `json:"prompt_name"`
		Content    string `json:"content"`
		Version    int    `json:"version"`
		IsCurrent  bool   `json:"is_current"`
	}
	ParseData(t, apiResp, &detail)

	if detail.PromptKey != "prompt_b" {
		t.Errorf("prompt_key不匹配: 期望 prompt_b, 实际 %s", detail.PromptKey)
	}
	if detail.Content != "Evaluator评估打分提示词内容" {
		t.Errorf("内容不匹配: 期望 Evaluator评估打分提示词内容, 实际 %s", detail.Content)
	}
	if !detail.IsCurrent {
		t.Error("查询到的版本 is_current 应为 true")
	}
	if detail.PromptName == "" {
		t.Error("prompt_name 不应为空（应有中文名映射）")
	}
}

// TestPrompt_UpdateVersion 更新提示词 → 版本号递增为2
func TestPrompt_UpdateVersion(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	// 创建V1
	createTestPromptViaAPI(t, server.URL, token, "prompt_c", "Translator V1内容")

	// 更新为V2
	body := map[string]interface{}{
		"content": "Translator V2更新后内容",
	}
	resp, apiResp := DoPut(t, server.URL+"/api/v1/prompts/prompt_c", body, token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	var updated struct {
		Version   int    `json:"version"`
		Content   string `json:"content"`
		IsCurrent bool   `json:"is_current"`
	}
	ParseData(t, apiResp, &updated)

	if updated.Version != 2 {
		t.Errorf("更新后版本号应为2，实际 %d", updated.Version)
	}
	if updated.Content != "Translator V2更新后内容" {
		t.Errorf("更新后内容不匹配")
	}

	// 验证DB中V1不再是current
	var v1Current bool
	err := database.DB.QueryRow(context.Background(),
		`SELECT is_current FROM prompts WHERE prompt_key = 'prompt_c' AND version = 1`,
	).Scan(&v1Current)
	if err != nil {
		t.Fatalf("查询V1状态失败: %v", err)
	}
	if v1Current {
		t.Error("V1更新后 is_current 应为 false")
	}
}

// TestPrompt_UpdateEmptyContent 更新内容为空 → 400
func TestPrompt_UpdateEmptyContent(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	body := map[string]interface{}{
		"content": "   ",
	}
	resp, _ := DoPut(t, server.URL+"/api/v1/prompts/prompt_a", body, token)
	AssertHTTPStatus(t, resp, http.StatusBadRequest)
}

// ==================== 版本管理测试 ====================

// TestPrompt_VersionHistory 版本历史 → 按版本号倒序返回
func TestPrompt_VersionHistory(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	// 创建V1 + V2
	createTestPromptViaAPI(t, server.URL, token, "prompt_d", "Reviewer V1")
	body := map[string]interface{}{"content": "Reviewer V2"}
	DoPut(t, server.URL+"/api/v1/prompts/prompt_d", body, token)

	// 查询版本历史
	resp, apiResp := DoGet(t, server.URL+"/api/v1/prompts/prompt_d/versions", token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	var history struct {
		PromptKey string `json:"prompt_key"`
		Versions  []struct {
			Version   int  `json:"version"`
			IsCurrent bool `json:"is_current"`
		} `json:"versions"`
		Total int `json:"total"`
	}
	ParseData(t, apiResp, &history)

	if history.Total != 2 {
		t.Errorf("应有2个版本，实际 %d", history.Total)
	}
	if len(history.Versions) >= 2 {
		// 第一个应是V2（倒序）
		if history.Versions[0].Version != 2 {
			t.Errorf("第一个版本应为V2，实际 V%d", history.Versions[0].Version)
		}
		if !history.Versions[0].IsCurrent {
			t.Error("V2应为当前版本")
		}
		if history.Versions[1].IsCurrent {
			t.Error("V1不应为当前版本")
		}
	}
}

// TestPrompt_Rollback 回滚到旧版本 → 旧版本变为当前
func TestPrompt_Rollback(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	// 创建V1
	v1ID := createTestPromptViaAPI(t, server.URL, token, "prompt_e", "Meta V1")

	// 创建V2
	body := map[string]interface{}{"content": "Meta V2"}
	DoPut(t, server.URL+"/api/v1/prompts/prompt_e", body, token)

	// 回滚到V1
	rollbackBody := map[string]interface{}{
		"version_id": v1ID,
	}
	resp, apiResp := DoPost(t, server.URL+"/api/v1/prompts/prompt_e/rollback", rollbackBody, token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	// 验证当前版本是V1内容
	var current struct {
		Content string `json:"content"`
		Version int    `json:"version"`
	}
	ParseData(t, apiResp, &current)

	if current.Content != "Meta V1" {
		t.Errorf("回滚后当前内容应为 Meta V1，实际 %s", current.Content)
	}
}

// TestPrompt_RollbackAlreadyCurrent 回滚已生效版本 → 400
func TestPrompt_RollbackAlreadyCurrent(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	// 创建V1（当前就是V1）
	v1ID := createTestPromptViaAPI(t, server.URL, token, "prompt_f", "Generator V1")

	// 尝试回滚到当前版本 → 400
	rollbackBody := map[string]interface{}{
		"version_id": v1ID,
	}
	resp, _ := DoPost(t, server.URL+"/api/v1/prompts/prompt_f/rollback", rollbackBody, token)
	AssertHTTPStatus(t, resp, http.StatusBadRequest)
}

// TestPrompt_GetInvalidKey 查询无效key → 400
func TestPrompt_GetInvalidKey(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	// 使用无效key
	resp, _ := DoGet(t, server.URL+"/api/v1/prompts/invalid_key_xyz", token)
	AssertHTTPStatus(t, resp, http.StatusBadRequest)
}

// ==================== 权限控制测试 ====================

// TestPrompt_ListForbiddenNonAdmin 非admin查询提示词列表 → 403
func TestPrompt_ListForbiddenNonAdmin(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	opToken := LoginAsOperator(t, server.URL)

	resp, _ := DoGet(t, server.URL+"/api/v1/prompts", opToken)
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("非admin查询提示词列表期望403，实际 %d", resp.StatusCode)
	}
}

// TestPrompt_UpdateForbiddenNonAdmin 非admin更新提示词 → 403
func TestPrompt_UpdateForbiddenNonAdmin(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	opToken := LoginAsOperator(t, server.URL)

	body := map[string]interface{}{
		"content": "尝试非法更新",
	}
	resp, _ := DoPut(t, server.URL+"/api/v1/prompts/prompt_a", body, opToken)
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("非admin更新提示词期望403，实际 %d", resp.StatusCode)
	}
}

// TestPrompt_ListUnauthorized 未登录访问 → 401
func TestPrompt_ListUnauthorized(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	resp, _ := DoGet(t, server.URL+"/api/v1/prompts", "")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("未登录访问提示词列表期望401，实际 %d", resp.StatusCode)
	}
}
