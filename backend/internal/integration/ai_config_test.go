package integration

// ai_config_test.go — AI配置管理集成测试
//
// 测试范围（4个用例）：
//
//   1. 获取全局AI配置 → 返回配置结构
//   2. 获取所有场景配置 → 返回场景列表
//   3. 非admin不能访问AI配置 → 403
//   4. 获取管理统计数据 → 返回统计结构
//
// 注意：
//   - TestConnection 和 ListModels 需要真实的AI API连接，不纳入集成测试
//   - UpdateGlobalConfig 涉及AES加密的API Key，需要seed AI配置行才能完整测试
//     这里只测读取接口以验证路由+权限+基本数据流通畅

import (
	"net/http"
	"testing"
)

// ==================== 全局配置测试 ====================

// TestAIConfig_GetGlobal 获取全局AI配置 → 返回配置结构
func TestAIConfig_GetGlobal(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	resp, apiResp := DoGet(t, server.URL+"/api/v1/ai-config/global", token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	// 全局配置应返回JSON数据（即使ai_configs表为空也不应出错）
	if apiResp.Data == nil {
		t.Error("全局配置响应Data不应为nil")
	}
}

// ==================== 场景配置测试 ====================

// TestAIConfig_GetScenes 获取所有场景配置 → 返回场景列表
func TestAIConfig_GetScenes(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	resp, apiResp := DoGet(t, server.URL+"/api/v1/ai-config/scenes", token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	// 场景配置表可能为空（测试环境不seed ai_scene_configs），但接口应正常返回
	if apiResp.Data == nil {
		t.Error("场景配置响应Data不应为nil")
	}
}

// ==================== 权限测试 ====================

// TestAIConfig_NonAdminForbidden 非admin不能访问AI配置 → 403
func TestAIConfig_NonAdminForbidden(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	viewerToken := LoginAsViewer(t, server.URL)

	// viewer访问全局配置 → 403
	resp1, _ := DoGet(t, server.URL+"/api/v1/ai-config/global", viewerToken)
	AssertHTTPStatus(t, resp1, http.StatusForbidden)

	// viewer访问场景配置 → 403
	resp2, _ := DoGet(t, server.URL+"/api/v1/ai-config/scenes", viewerToken)
	AssertHTTPStatus(t, resp2, http.StatusForbidden)

	operatorToken := LoginAsOperator(t, server.URL)

	// operator访问全局配置 → 403
	resp3, _ := DoGet(t, server.URL+"/api/v1/ai-config/global", operatorToken)
	AssertHTTPStatus(t, resp3, http.StatusForbidden)
}

// ==================== 管理统计测试 ====================

// TestAdmin_GetStats 获取管理统计数据 → 返回统计结构
func TestAdmin_GetStats(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	resp, apiResp := DoGet(t, server.URL+"/api/v1/admin/stats", token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	if apiResp.Data == nil {
		t.Error("统计数据响应Data不应为nil")
	}
}
