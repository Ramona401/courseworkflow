package integration

// auth_test.go — 认证与权限集成测试
//
// 测试范围（15个用例）：
//   1. 正常登录 → 返回 token + 用户信息
//   2. 错误密码登录 → 401
//   3. 不存在的用户登录 → 401
//   4. 禁用用户登录 → 403
//   5. 空用户名/密码登录 → 400
//   6. 带 Token 访问 /auth/me → 返回用户信息
//   7. 无 Token 访问受保护接口 → 401
//   8. 无效 Token 访问受保护接口 → 401
//   9. viewer 访问 admin 接口 → 403
//   10. operator 访问 admin 接口 → 403
//   11. admin 访问 admin 接口 → 200
//   12. 登出接口 → 正常返回
//   13. 不同角色都能成功登录
//   14. 登录后 login_count 递增
//   15. 健康检查接口（公开，不需要认证）

import (
	"context"
	"net/http"
	"os"
	"testing"

	"tedna/internal/database"
)

// ==================== 包级别测试入口 ====================

// TestMain 包级别测试入口
// 负责安全检查 + 运行所有测试 + 清理资源
func TestMain(m *testing.M) {
	// 安全检查：确保测试配置指向 tedna_test
	cfg := testConfig()
	if cfg.DBName != "tedna_test" {
		panic("安全检查失败：测试数据库必须为 tedna_test，实际为 " + cfg.DBName)
	}

	// 运行所有测试
	exitCode := m.Run()

	// 清理：关闭数据库连接池（如果 SetupTestServer 已创建）
	if database.DB != nil {
		database.DB.Close()
	}

	os.Exit(exitCode)
}

// ==================== 登录测试 ====================

// TestLogin_Success 正常登录：admin用户 → 返回token + 用户信息
func TestLogin_Success(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	loginBody := map[string]string{
		"username": SeedAdminUsername,
		"password": SeedAdminPassword,
	}

	resp, apiResp := DoPost(t, server.URL+"/api/v1/auth/login", loginBody, "")

	// 断言 HTTP 200
	AssertHTTPStatus(t, resp, http.StatusOK)
	// 断言业务码 0（成功）
	AssertAPICode(t, apiResp, 0)

	// 解析响应数据
	var loginData struct {
		Token string `json:"token"`
		User  struct {
			ID       string `json:"id"`
			Username string `json:"username"`
			Role     string `json:"role"`
			Status   string `json:"status"`
		} `json:"user"`
	}
	ParseData(t, apiResp, &loginData)

	// 断言 token 不为空
	if loginData.Token == "" {
		t.Error("登录成功但 token 为空")
	}

	// 断言用户信息正确
	if loginData.User.ID != SeedAdminID {
		t.Errorf("用户ID不匹配: 期望 %s, 实际 %s", SeedAdminID, loginData.User.ID)
	}
	if loginData.User.Username != SeedAdminUsername {
		t.Errorf("用户名不匹配: 期望 %s, 实际 %s", SeedAdminUsername, loginData.User.Username)
	}
	if loginData.User.Role != SeedAdminRole {
		t.Errorf("角色不匹配: 期望 %s, 实际 %s", SeedAdminRole, loginData.User.Role)
	}
	if loginData.User.Status != "active" {
		t.Errorf("状态不匹配: 期望 active, 实际 %s", loginData.User.Status)
	}
}

// TestLogin_WrongPassword 错误密码 → 401
func TestLogin_WrongPassword(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	loginBody := map[string]string{
		"username": SeedAdminUsername,
		"password": "wrong_password_12345",
	}

	resp, apiResp := DoPost(t, server.URL+"/api/v1/auth/login", loginBody, "")

	AssertHTTPStatus(t, resp, http.StatusUnauthorized)
	if apiResp.Code == 0 {
		t.Error("错误密码登录不应返回 code=0")
	}
}

// TestLogin_NonExistentUser 不存在的用户 → 401
func TestLogin_NonExistentUser(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	loginBody := map[string]string{
		"username": "nobody_exists_here",
		"password": "any_password",
	}

	resp, _ := DoPost(t, server.URL+"/api/v1/auth/login", loginBody, "")
	AssertHTTPStatus(t, resp, http.StatusUnauthorized)
}

// TestLogin_DisabledUser 禁用用户登录 → 403
func TestLogin_DisabledUser(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	loginBody := map[string]string{
		"username": SeedDisabledUsername,
		"password": SeedDisabledPassword,
	}

	resp, _ := DoPost(t, server.URL+"/api/v1/auth/login", loginBody, "")
	AssertHTTPStatus(t, resp, http.StatusForbidden)
}

// TestLogin_EmptyFields 空用户名/密码 → 400
func TestLogin_EmptyFields(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	// 空用户名
	resp1, _ := DoPost(t, server.URL+"/api/v1/auth/login", map[string]string{
		"username": "",
		"password": "some_password",
	}, "")
	AssertHTTPStatus(t, resp1, http.StatusBadRequest)

	// 空密码
	resp2, _ := DoPost(t, server.URL+"/api/v1/auth/login", map[string]string{
		"username": "admin",
		"password": "",
	}, "")
	AssertHTTPStatus(t, resp2, http.StatusBadRequest)
}

// ==================== Token 验证测试 ====================

// TestAuthMe_WithValidToken 带有效 Token 访问 /auth/me → 返回用户信息
func TestAuthMe_WithValidToken(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	// 先登录获取 token
	token := LoginAsAdmin(t, server.URL)

	// 用 token 访问 /auth/me
	resp, apiResp := DoGet(t, server.URL+"/api/v1/auth/me", token)

	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	var userData struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		Role     string `json:"role"`
	}
	ParseData(t, apiResp, &userData)

	if userData.ID != SeedAdminID {
		t.Errorf("用户ID不匹配: 期望 %s, 实际 %s", SeedAdminID, userData.ID)
	}
	if userData.Username != SeedAdminUsername {
		t.Errorf("用户名不匹配: 期望 %s, 实际 %s", SeedAdminUsername, userData.Username)
	}
}

// TestAuthMe_NoToken 无 Token 访问受保护接口 → 401
func TestAuthMe_NoToken(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	resp, _ := DoGet(t, server.URL+"/api/v1/auth/me", "")
	AssertHTTPStatus(t, resp, http.StatusUnauthorized)
}

// TestAuthMe_InvalidToken 无效 Token → 401
func TestAuthMe_InvalidToken(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	resp, _ := DoGet(t, server.URL+"/api/v1/auth/me", "this.is.not.a.valid.jwt.token")
	AssertHTTPStatus(t, resp, http.StatusUnauthorized)
}

// ==================== 权限测试 ====================

// TestRBAC_ViewerCannotAccessAdmin viewer 访问 admin 接口 → 403
func TestRBAC_ViewerCannotAccessAdmin(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsViewer(t, server.URL)

	// viewer 尝试访问 admin 专属接口：用户列表
	resp, _ := DoGet(t, server.URL+"/api/v1/admin/users", token)
	AssertHTTPStatus(t, resp, http.StatusForbidden)
}

// TestRBAC_OperatorCannotAccessAdmin operator 访问 admin 接口 → 403
func TestRBAC_OperatorCannotAccessAdmin(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsOperator(t, server.URL)

	// operator 尝试访问 admin 专属接口：用户列表
	resp, _ := DoGet(t, server.URL+"/api/v1/admin/users", token)
	AssertHTTPStatus(t, resp, http.StatusForbidden)
}

// TestRBAC_AdminCanAccessAdmin admin 访问 admin 接口 → 200
func TestRBAC_AdminCanAccessAdmin(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	// admin 访问 admin 专属接口：用户列表
	resp, apiResp := DoGet(t, server.URL+"/api/v1/admin/users", token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)
}

// ==================== 登出测试 ====================

// TestLogout_Success 登出接口正常返回
func TestLogout_Success(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	resp, apiResp := DoPost(t, server.URL+"/api/v1/auth/logout", nil, token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)
}

// ==================== 多角色登录测试 ====================

// TestLogin_AllRoles 所有角色都能成功登录
func TestLogin_AllRoles(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	roles := []struct {
		username string
		password string
		role     string
	}{
		{SeedAdminUsername, SeedAdminPassword, SeedAdminRole},
		{SeedOperatorUsername, SeedOperatorPassword, SeedOperatorRole},
		{SeedSeniorUsername, SeedSeniorPassword, SeedSeniorRole},
		{SeedViewerUsername, SeedViewerPassword, SeedViewerRole},
	}

	for _, r := range roles {
		t.Run("role_"+r.role, func(t *testing.T) {
			token := LoginAs(t, server.URL, r.username, r.password)
			if token == "" {
				t.Errorf("角色 %s 登录失败", r.role)
			}
		})
	}
}

// ==================== 登录计数测试 ====================

// TestLogin_IncreasesLoginCount 登录后 login_count 应该递增
func TestLogin_IncreasesLoginCount(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	// 第一次登录
	LoginAsAdmin(t, server.URL)

	// 查数据库确认 login_count
	var count1 int
	err := database.DB.QueryRow(context.Background(),
		"SELECT login_count FROM users WHERE id = $1", SeedAdminID,
	).Scan(&count1)
	if err != nil {
		t.Fatalf("查询login_count失败: %v", err)
	}
	if count1 != 1 {
		t.Errorf("第一次登录后 login_count 应为1, 实际 %d", count1)
	}

	// 第二次登录
	LoginAsAdmin(t, server.URL)

	var count2 int
	err = database.DB.QueryRow(context.Background(),
		"SELECT login_count FROM users WHERE id = $1", SeedAdminID,
	).Scan(&count2)
	if err != nil {
		t.Fatalf("查询login_count失败: %v", err)
	}
	if count2 != 2 {
		t.Errorf("第二次登录后 login_count 应为2, 实际 %d", count2)
	}
}

// ==================== 健康检查测试 ====================

// TestHealthCheck 健康检查接口（公开，不需认证）
func TestHealthCheck(t *testing.T) {
	server, _ := SetupTestServer(t)
	// 健康检查不需要 CleanAndSeed（不涉及业务数据）

	resp, _ := DoGet(t, server.URL+"/api/v1/health", "")
	AssertHTTPStatus(t, resp, http.StatusOK)
}
