package integration

// account_admin_test.go — 账户自助操作 + 用户管理(admin) + 审计日志 集成测试
//
// 测试范围（13个用例）：
//
// 账户自助操作（4个）：
//   1. 获取个人信息 → 返回当前用户详情
//   2. 修改显示名称 → 成功
//   3. 修改密码 → 成功（可用新密码重新登录）
//   4. 修改密码 — 旧密码错误 → 400
//
// 用户管理-admin（6个）：
//   5. admin创建用户 → 成功
//   6. 创建用户缺少必填字段 → 400
//   7. 创建用户名重复 → 400
//   8. admin修改用户状态（禁用/启用）→ 成功+禁用后无法登录
//   9. admin重置密码 → 成功（用新密码登录）
//   10. admin编辑用户角色+显示名 → 成功
//
// 审计日志（3个）：
//   11. 创建用户后审计日志出现 admin.user_create 记录
//   12. 审计日志支持分页查询
//   13. 非admin不能查看审计日志 → 403

import (
	"net/http"
	"testing"
	"time"
)

// ==================== 账户自助操作测试 ====================

// TestAccount_GetProfile 获取个人信息 → 返回当前用户详情
func TestAccount_GetProfile(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsOperator(t, server.URL)

	resp, apiResp := DoGet(t, server.URL+"/api/v1/account/profile", token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	var profile struct {
		ID          string `json:"id"`
		Username    string `json:"username"`
		DisplayName string `json:"display_name"`
		Role        string `json:"role"`
		Status      string `json:"status"`
	}
	ParseData(t, apiResp, &profile)

	if profile.ID != SeedOperatorID {
		t.Errorf("用户ID不匹配: 期望 %s, 实际 %s", SeedOperatorID, profile.ID)
	}
	if profile.Username != SeedOperatorUsername {
		t.Errorf("用户名不匹配: 期望 %s, 实际 %s", SeedOperatorUsername, profile.Username)
	}
	if profile.Role != SeedOperatorRole {
		t.Errorf("角色不匹配: 期望 %s, 实际 %s", SeedOperatorRole, profile.Role)
	}
}

// TestAccount_UpdateDisplayName 修改显示名称 → 成功
func TestAccount_UpdateDisplayName(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsViewer(t, server.URL)

	body := map[string]interface{}{
		"display_name": "新的显示名称",
	}
	resp, apiResp := DoPut(t, server.URL+"/api/v1/account/profile", body, token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	// 再次获取个人信息验证
	resp2, apiResp2 := DoGet(t, server.URL+"/api/v1/account/profile", token)
	AssertHTTPStatus(t, resp2, http.StatusOK)

	var profile struct {
		DisplayName string `json:"display_name"`
	}
	ParseData(t, apiResp2, &profile)
	if profile.DisplayName != "新的显示名称" {
		t.Errorf("显示名称未更新: 期望 新的显示名称, 实际 %s", profile.DisplayName)
	}
}

// TestAccount_ChangePassword 修改密码 → 成功（用新密码重新登录）
func TestAccount_ChangePassword(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	// 先用旧密码登录
	token := LoginAsOperator(t, server.URL)

	// 修改密码
	body := map[string]interface{}{
		"old_password": SeedOperatorPassword,
		"new_password": "new_password_123",
	}
	resp, apiResp := DoPut(t, server.URL+"/api/v1/account/password", body, token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	// 用新密码登录 → 应成功
	newToken := LoginAs(t, server.URL, SeedOperatorUsername, "new_password_123")
	if newToken == "" {
		t.Error("用新密码登录失败")
	}

	// 用旧密码登录 → 应失败
	loginBody := map[string]string{
		"username": SeedOperatorUsername,
		"password": SeedOperatorPassword,
	}
	respOld, _ := DoPost(t, server.URL+"/api/v1/auth/login", loginBody, "")
	AssertHTTPStatus(t, respOld, http.StatusUnauthorized)
}

// TestAccount_ChangePasswordWrongOld 修改密码 — 旧密码错误 → 400
func TestAccount_ChangePasswordWrongOld(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsOperator(t, server.URL)

	body := map[string]interface{}{
		"old_password": "completely_wrong_password",
		"new_password": "new_password_456",
	}
	resp, _ := DoPut(t, server.URL+"/api/v1/account/password", body, token)
	AssertHTTPStatus(t, resp, http.StatusBadRequest)
}

// ==================== 用户管理测试（admin操作） ====================

// TestAdminUser_Create admin创建用户 → 成功
func TestAdminUser_Create(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	body := map[string]interface{}{
		"username":     "new_teacher",
		"display_name": "新教师",
		"password":     "teacher123",
		"role":         "operator",
	}
	resp, apiResp := DoPost(t, server.URL+"/api/v1/admin/users", body, token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	var userData struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		Role     string `json:"role"`
		Status   string `json:"status"`
	}
	ParseData(t, apiResp, &userData)

	if userData.Username != "new_teacher" {
		t.Errorf("用户名不匹配: 期望 new_teacher, 实际 %s", userData.Username)
	}
	if userData.Role != "operator" {
		t.Errorf("角色不匹配: 期望 operator, 实际 %s", userData.Role)
	}
	if userData.Status != "active" {
		t.Errorf("新用户状态应为active, 实际 %s", userData.Status)
	}

	// 新创建的用户可以登录
	newToken := LoginAs(t, server.URL, "new_teacher", "teacher123")
	if newToken == "" {
		t.Error("新创建的用户登录失败")
	}
}

// TestAdminUser_CreateMissingFields 创建用户缺少必填字段 → 400
func TestAdminUser_CreateMissingFields(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	// 缺少username
	body := map[string]interface{}{
		"username":     "",
		"display_name": "测试",
		"password":     "123456",
		"role":         "viewer",
	}
	resp, _ := DoPost(t, server.URL+"/api/v1/admin/users", body, token)
	AssertHTTPStatus(t, resp, http.StatusBadRequest)

	// 密码太短
	body2 := map[string]interface{}{
		"username":     "short_pwd_user",
		"display_name": "测试",
		"password":     "123",
		"role":         "viewer",
	}
	resp2, _ := DoPost(t, server.URL+"/api/v1/admin/users", body2, token)
	AssertHTTPStatus(t, resp2, http.StatusBadRequest)
}

// TestAdminUser_CreateDuplicate 创建重复用户名 → 400
func TestAdminUser_CreateDuplicate(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	// 尝试创建已存在的admin用户名
	body := map[string]interface{}{
		"username":     SeedAdminUsername,
		"display_name": "重复管理员",
		"password":     "123456",
		"role":         "viewer",
	}
	resp, _ := DoPost(t, server.URL+"/api/v1/admin/users", body, token)
	AssertHTTPStatus(t, resp, http.StatusBadRequest)
}

// TestAdminUser_DisableAndEnable admin禁用用户 → 该用户无法登录 → 重新启用 → 可登录
func TestAdminUser_DisableAndEnable(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	// 禁用operator1
	disableBody := map[string]interface{}{
		"status": "disabled",
	}
	resp, apiResp := DoPut(t, server.URL+"/api/v1/admin/users/"+SeedOperatorID+"/status", disableBody, token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	// operator1尝试登录 → 应失败(403)
	loginBody := map[string]string{
		"username": SeedOperatorUsername,
		"password": SeedOperatorPassword,
	}
	respLogin, _ := DoPost(t, server.URL+"/api/v1/auth/login", loginBody, "")
	AssertHTTPStatus(t, respLogin, http.StatusForbidden)

	// 重新启用operator1
	enableBody := map[string]interface{}{
		"status": "active",
	}
	resp2, apiResp2 := DoPut(t, server.URL+"/api/v1/admin/users/"+SeedOperatorID+"/status", enableBody, token)
	AssertHTTPStatus(t, resp2, http.StatusOK)
	AssertAPICode(t, apiResp2, 0)

	// operator1再次登录 → 应成功
	newToken := LoginAs(t, server.URL, SeedOperatorUsername, SeedOperatorPassword)
	if newToken == "" {
		t.Error("重新启用后operator1登录失败")
	}
}

// TestAdminUser_ResetPassword admin重置密码 → 成功
func TestAdminUser_ResetPassword(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	// 重置viewer1的密码
	body := map[string]interface{}{
		"new_password": "reset_password_789",
	}
	resp, apiResp := DoPut(t, server.URL+"/api/v1/admin/users/"+SeedViewerID+"/password", body, token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	// 用新密码登录 → 应成功
	newToken := LoginAs(t, server.URL, SeedViewerUsername, "reset_password_789")
	if newToken == "" {
		t.Error("重置密码后用新密码登录失败")
	}
}

// TestAdminUser_Update admin编辑用户角色+显示名 → 成功
func TestAdminUser_Update(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	// 将viewer1升级为operator
	body := map[string]interface{}{
		"display_name": "升级后的操作员",
		"role":         "operator",
	}
	resp, apiResp := DoPut(t, server.URL+"/api/v1/admin/users/"+SeedViewerID, body, token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	var userData struct {
		Role        string `json:"role"`
		DisplayName string `json:"display_name"`
	}
	ParseData(t, apiResp, &userData)

	if userData.Role != "operator" {
		t.Errorf("角色未更新: 期望 operator, 实际 %s", userData.Role)
	}
	if userData.DisplayName != "升级后的操作员" {
		t.Errorf("显示名称未更新: 期望 升级后的操作员, 实际 %s", userData.DisplayName)
	}
}

// ==================== 审计日志测试 ====================

// TestAuditLog_UserCreateGeneratesLog 创建用户后审计日志出现 admin.user_create 记录
func TestAuditLog_UserCreateGeneratesLog(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	// 创建一个新用户（会触发审计日志写入）
	body := map[string]interface{}{
		"username":     "audit_test_user",
		"display_name": "审计测试用户",
		"password":     "audit123",
		"role":         "viewer",
	}
	resp, _ := DoPost(t, server.URL+"/api/v1/admin/users", body, token)
	AssertHTTPStatus(t, resp, http.StatusOK)

	// 审计日志是异步写入的，等一小段时间
	time.Sleep(500 * time.Millisecond)

	// 查询审计日志
	respLogs, apiRespLogs := DoGet(t, server.URL+"/api/v1/admin/audit-logs?action=admin.user_create", token)
	AssertHTTPStatus(t, respLogs, http.StatusOK)
	AssertAPICode(t, apiRespLogs, 0)

	var logData struct {
		Logs []struct {
			Action string `json:"action"`
			UserID string `json:"user_id"`
		} `json:"logs"`
		Total int `json:"total"`
	}
	ParseData(t, apiRespLogs, &logData)

	if logData.Total < 1 {
		t.Error("创建用户后应至少有1条 admin.user_create 审计日志")
	}
}

// TestAuditLog_Pagination 审计日志分页查询
func TestAuditLog_Pagination(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	// 查询第1页，每页2条
	resp, apiResp := DoGet(t, server.URL+"/api/v1/admin/audit-logs?page=1&page_size=2", token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	var logData struct {
		Logs  []interface{} `json:"logs"`
		Total int           `json:"total"`
	}
	ParseData(t, apiResp, &logData)

	// 分页应正常工作（即使日志为空也应返回正确结构）
	if logData.Logs == nil {
		t.Error("审计日志logs字段不应为nil")
	}
}

// TestAuditLog_NonAdminForbidden 非admin不能查看审计日志 → 403
func TestAuditLog_NonAdminForbidden(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsViewer(t, server.URL)

	resp, _ := DoGet(t, server.URL+"/api/v1/admin/audit-logs", token)
	AssertHTTPStatus(t, resp, http.StatusForbidden)
}
