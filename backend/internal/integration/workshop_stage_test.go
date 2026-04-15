package integration

// workshop_stage_test.go — 备课工坊阶段管理集成测试
//
// 测试范围（10个用例）：
//
// 系统阶段查询（3个）：
//   1. 获取默认阶段列表（全员可访问）→ 5个active阶段
//   2. Admin获取全部系统阶段（含disabled）→ 至少5个
//   3. 非admin获取全部系统阶段 → 403
//
// 系统阶段更新（4个）：
//   4. Admin更新系统阶段名称+AI角色 → 成功
//   5. 更新不存在的阶段 → 404
//   6. 缺少必填字段（stage_name为空）→ 400
//   7. 无效的gate_mode → 400
//
// 默认阶段数据验证（3个）：
//   8. 默认阶段顺序正确（analyze→design→write→review→revise）
//   9. write阶段不可跳过
//   10. 非登录用户获取默认阶段 → 401
//
// 注意：
//   - CleanAndSeed已插入5个系统默认阶段种子数据
//   - 默认阶段接口 /api/v1/lesson-plans/workshop/stages/defaults 全员可访问
//   - Admin阶段管理接口 /api/v1/admin/workshop-stages 仅admin可访问

import (
	"context"
	"net/http"
	"testing"

	"tedna/internal/database"
)

// ==================== 系统阶段查询测试 ====================

// TestWorkshopStage_GetDefaults 获取默认阶段列表 → 5个active阶段
func TestWorkshopStage_GetDefaults(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	// 默认阶段接口全员可访问，用viewer测试
	token := LoginAsViewer(t, server.URL)

	resp, apiResp := DoGet(t, server.URL+"/api/v1/lesson-plans/workshop/stages/defaults", token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	var stagesData struct {
		Stages []struct {
			StageCode  string `json:"stage_code"`
			StageName  string `json:"stage_name"`
			StageOrder int    `json:"stage_order"`
			Skippable  bool   `json:"skippable"`
		} `json:"stages"`
	}
	ParseData(t, apiResp, &stagesData)

	if len(stagesData.Stages) != 5 {
		t.Errorf("应有5个默认阶段，实际 %d", len(stagesData.Stages))
	}
}

// TestWorkshopStage_AdminListAll Admin获取全部系统阶段 → 至少5个
func TestWorkshopStage_AdminListAll(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	resp, apiResp := DoGet(t, server.URL+"/api/v1/admin/workshop-stages", token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	var stagesData struct {
		Stages []struct {
			StageCode string `json:"stage_code"`
			Status    string `json:"status"`
		} `json:"stages"`
	}
	ParseData(t, apiResp, &stagesData)

	if len(stagesData.Stages) < 5 {
		t.Errorf("admin应至少看到5个系统阶段，实际 %d", len(stagesData.Stages))
	}
}

// TestWorkshopStage_AdminListForbidden 非admin获取全部系统阶段 → 403
func TestWorkshopStage_AdminListForbidden(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	opToken := LoginAsOperator(t, server.URL)

	resp, _ := DoGet(t, server.URL+"/api/v1/admin/workshop-stages", opToken)
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("非admin获取全部系统阶段期望403，实际 %d", resp.StatusCode)
	}
}

// ==================== 系统阶段更新测试 ====================

// TestWorkshopStage_AdminUpdate Admin更新系统阶段 → 成功
func TestWorkshopStage_AdminUpdate(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	body := map[string]interface{}{
		"stage_name":      "教学分析（已更新）",
		"ai_role":         "高级教学分析师",
		"system_prompt":   "你是一位高级教学分析师。",
		"output_format":   "{}",
		"component_types": "[]",
		"gate_mode":       "suggest",
		"skippable":       true,
		"status":          "active",
	}

	resp, apiResp := DoPut(t, server.URL+"/api/v1/admin/workshop-stages/analyze", body, token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	// 验证DB已更新
	var name, aiRole string
	err := database.DB.QueryRow(context.Background(),
		`SELECT stage_name, ai_role FROM workshop_stages WHERE source = 'system' AND stage_code = 'analyze'`,
	).Scan(&name, &aiRole)
	if err != nil {
		t.Fatalf("查询阶段失败: %v", err)
	}
	if name != "教学分析（已更新）" {
		t.Errorf("阶段名称未更新: 期望 教学分析（已更新）, 实际 %s", name)
	}
	if aiRole != "高级教学分析师" {
		t.Errorf("AI角色未更新: 期望 高级教学分析师, 实际 %s", aiRole)
	}
}

// TestWorkshopStage_AdminUpdateNotFound 更新不存在的阶段 → 404
func TestWorkshopStage_AdminUpdateNotFound(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	body := map[string]interface{}{
		"stage_name":      "不存在阶段",
		"ai_role":         "测试角色",
		"system_prompt":   "测试",
		"output_format":   "{}",
		"component_types": "[]",
		"gate_mode":       "suggest",
		"skippable":       true,
		"status":          "active",
	}

	resp, _ := DoPut(t, server.URL+"/api/v1/admin/workshop-stages/nonexistent_stage", body, token)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("更新不存在的阶段期望404，实际 %d", resp.StatusCode)
	}
}

// TestWorkshopStage_AdminUpdateEmptyName 缺少必填字段 → 400
func TestWorkshopStage_AdminUpdateEmptyName(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	body := map[string]interface{}{
		"stage_name":      "",
		"ai_role":         "测试角色",
		"system_prompt":   "测试",
		"output_format":   "{}",
		"component_types": "[]",
		"gate_mode":       "suggest",
		"skippable":       true,
		"status":          "active",
	}

	resp, _ := DoPut(t, server.URL+"/api/v1/admin/workshop-stages/analyze", body, token)
	AssertHTTPStatus(t, resp, http.StatusBadRequest)
}

// TestWorkshopStage_AdminUpdateInvalidGateMode 无效gate_mode → 400
func TestWorkshopStage_AdminUpdateInvalidGateMode(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	body := map[string]interface{}{
		"stage_name":      "教学分析",
		"ai_role":         "教学分析师",
		"system_prompt":   "测试",
		"output_format":   "{}",
		"component_types": "[]",
		"gate_mode":       "invalid_mode",
		"skippable":       true,
		"status":          "active",
	}

	resp, _ := DoPut(t, server.URL+"/api/v1/admin/workshop-stages/analyze", body, token)
	AssertHTTPStatus(t, resp, http.StatusBadRequest)
}

// ==================== 默认阶段数据验证测试 ====================

// TestWorkshopStage_DefaultOrder 默认阶段顺序正确
func TestWorkshopStage_DefaultOrder(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	resp, apiResp := DoGet(t, server.URL+"/api/v1/lesson-plans/workshop/stages/defaults", token)
	AssertHTTPStatus(t, resp, http.StatusOK)

	var stagesData struct {
		Stages []struct {
			StageCode  string `json:"stage_code"`
			StageOrder int    `json:"stage_order"`
			Skippable  bool   `json:"skippable"`
		} `json:"stages"`
	}
	ParseData(t, apiResp, &stagesData)

	expectedOrder := []string{"analyze", "design", "write", "review", "revise"}
	if len(stagesData.Stages) != len(expectedOrder) {
		t.Fatalf("阶段数量不匹配: 期望 %d, 实际 %d", len(expectedOrder), len(stagesData.Stages))
	}

	for i, expected := range expectedOrder {
		if stagesData.Stages[i].StageCode != expected {
			t.Errorf("阶段[%d]顺序错误: 期望 %s, 实际 %s", i, expected, stagesData.Stages[i].StageCode)
		}
	}
}

// TestWorkshopStage_WriteNotSkippable write阶段不可跳过（种子数据验证）
func TestWorkshopStage_WriteNotSkippable(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	resp, apiResp := DoGet(t, server.URL+"/api/v1/lesson-plans/workshop/stages/defaults", token)
	AssertHTTPStatus(t, resp, http.StatusOK)

	var stagesData struct {
		Stages []struct {
			StageCode string `json:"stage_code"`
			Skippable bool   `json:"skippable"`
		} `json:"stages"`
	}
	ParseData(t, apiResp, &stagesData)

	for _, stage := range stagesData.Stages {
		if stage.StageCode == "write" && stage.Skippable {
			t.Error("write阶段不应该可跳过")
		}
	}
}

// TestWorkshopStage_DefaultsUnauthorized 未登录获取默认阶段 → 401
func TestWorkshopStage_DefaultsUnauthorized(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	resp, _ := DoGet(t, server.URL+"/api/v1/lesson-plans/workshop/stages/defaults", "")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("未登录获取默认阶段期望401，实际 %d", resp.StatusCode)
	}
}
