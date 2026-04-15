package integration

// external_data_test.go — 外部数据配置集成测试
//
// 测试范围（6个用例）：
//
// 配置读取（2个）：
//   1. 获取配置列表（初始空）→ 成功，total=0
//   2. 插入种子配置后获取列表 → 成功，包含预期配置项
//
// 配置更新（2个）：
//   3. 更新非敏感配置 → 成功，值被更新
//   4. 更新空configs → 400（配置数据不能为空）
//
// 权限控制（2个）：
//   5. 非admin读取配置 → 403
//   6. 未登录访问 → 401
//
// 注意：
//   - external_data_configs表在CleanAndSeed中被清空，初始无数据
//   - UpdateConfigs是UPDATE操作（非UPSERT），需要表中已有记录
//   - 测试更新前先通过DB直接INSERT种子配置数据

import (
	"context"
	"net/http"
	"testing"

	"tedna/internal/database"
)

// ==================== 辅助函数 ====================

// seedExternalDataConfigs 直接向DB插入外部数据配置种子数据
// 因为API只有UPDATE没有CREATE，需要通过DB直接插入
func seedExternalDataConfigs(t *testing.T) {
	t.Helper()
	ctx := context.Background()

	configs := []struct {
		key, value, description string
	}{
		{"oss_endpoint", "oss-cn-beijing.aliyuncs.com", "OSS Endpoint"},
		{"oss_bucket", "test-bucket", "OSS Bucket名称"},
		{"oss_access_key_id", "LTAI_test_key_id", "OSS AccessKey ID"},
		{"oss_access_key_enc", "PLACEHOLDER_SET_IN_ADMIN", "OSS AccessKey Secret（加密存储）"},
		{"oss_index_prefix", "indexes/", "OSS 索引文件路径前缀"},
		{"oss_html_prefix", "html/", "OSS HTML文件路径前缀"},
		{"push_api_url", "https://example.com/api/push", "推送API地址"},
		{"push_api_token", "PLACEHOLDER_SET_IN_ADMIN", "推送API Token（加密存储）"},
	}

	for _, c := range configs {
		_, err := database.DB.Exec(ctx,
			`INSERT INTO external_data_configs (id, config_key, config_value, description, updated_at)
			 VALUES (gen_random_uuid(), $1, $2, $3, now())`,
			c.key, c.value, c.description,
		)
		if err != nil {
			t.Fatalf("插入外部数据配置种子失败 (key=%s): %v", c.key, err)
		}
	}
}

// ==================== 配置读取测试 ====================

// TestExternalData_ListEmpty 获取配置列表（初始空）→ 成功，total=0
func TestExternalData_ListEmpty(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	resp, apiResp := DoGet(t, server.URL+"/api/v1/external-data/configs", token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	var listData struct {
		Configs []struct {
			ConfigKey string `json:"config_key"`
		} `json:"configs"`
		Total int `json:"total"`
	}
	ParseData(t, apiResp, &listData)

	if listData.Total != 0 {
		t.Errorf("初始外部数据配置应为空，实际 total=%d", listData.Total)
	}
}

// TestExternalData_ListWithSeed 插入种子配置后获取列表 → 包含预期配置项
func TestExternalData_ListWithSeed(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)
	seedExternalDataConfigs(t)

	token := LoginAsAdmin(t, server.URL)

	resp, apiResp := DoGet(t, server.URL+"/api/v1/external-data/configs", token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	var listData struct {
		Configs []struct {
			ConfigKey   string `json:"config_key"`
			IsSensitive bool   `json:"is_sensitive"`
			IsSet       bool   `json:"is_set"`
		} `json:"configs"`
		Total int `json:"total"`
	}
	ParseData(t, apiResp, &listData)

	if listData.Total != 8 {
		t.Errorf("应有8个配置项，实际 %d", listData.Total)
	}

	// 验证敏感字段标记
	sensitiveCount := 0
	setCount := 0
	for _, c := range listData.Configs {
		if c.IsSensitive {
			sensitiveCount++
		}
		if c.IsSet {
			setCount++
		}
	}
	if sensitiveCount != 2 {
		t.Errorf("应有2个敏感字段，实际 %d", sensitiveCount)
	}
	// oss_access_key_enc和push_api_token为PLACEHOLDER，IsSet=false
	// 其余6个非占位符，IsSet=true
	if setCount != 6 {
		t.Errorf("应有6个已配置(is_set=true)的配置项，实际 %d", setCount)
	}
}

// ==================== 配置更新测试 ====================

// TestExternalData_UpdateNonSensitive 更新非敏感配置 → 成功
func TestExternalData_UpdateNonSensitive(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)
	seedExternalDataConfigs(t)

	token := LoginAsAdmin(t, server.URL)

	body := map[string]interface{}{
		"configs": map[string]string{
			"oss_endpoint": "oss-cn-shanghai.aliyuncs.com",
			"oss_bucket":   "updated-bucket",
		},
	}

	resp, apiResp := DoPut(t, server.URL+"/api/v1/external-data/configs", body, token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	// 验证DB中的值已更新
	var value string
	err := database.DB.QueryRow(context.Background(),
		`SELECT config_value FROM external_data_configs WHERE config_key = 'oss_endpoint'`,
	).Scan(&value)
	if err != nil {
		t.Fatalf("查询配置失败: %v", err)
	}
	if value != "oss-cn-shanghai.aliyuncs.com" {
		t.Errorf("oss_endpoint未更新: 期望 oss-cn-shanghai.aliyuncs.com, 实际 %s", value)
	}
}

// TestExternalData_UpdateEmptyConfigs 更新空configs → 400
func TestExternalData_UpdateEmptyConfigs(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	body := map[string]interface{}{
		"configs": map[string]string{},
	}
	resp, _ := DoPut(t, server.URL+"/api/v1/external-data/configs", body, token)
	AssertHTTPStatus(t, resp, http.StatusBadRequest)
}

// ==================== 权限控制测试 ====================

// TestExternalData_ForbiddenNonAdmin 非admin读取配置 → 403
func TestExternalData_ForbiddenNonAdmin(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	opToken := LoginAsOperator(t, server.URL)

	resp, _ := DoGet(t, server.URL+"/api/v1/external-data/configs", opToken)
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("非admin读取外部数据配置期望403，实际 %d", resp.StatusCode)
	}
}

// TestExternalData_Unauthorized 未登录访问 → 401
func TestExternalData_Unauthorized(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	resp, _ := DoGet(t, server.URL+"/api/v1/external-data/configs", "")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("未登录访问外部数据配置期望401，实际 %d", resp.StatusCode)
	}
}
