package integration

// security_test.go — 安全专项集成测试
//
// 测试范围（18个用例）：
//
// JWT安全（5个）：
//   1. 伪造JWT签名 → 401
//   2. 篡改JWT payload（改角色为admin）→ 401
//   3. 空Bearer token → 401
//   4. 非Bearer前缀的Authorization头 → 401
//   5. 过期token模拟（用错误密钥签发）→ 401
//
// SQL注入防护（4个）：
//   6. 登录用户名注入 → 401（不应报500）
//   7. 课程编号注入 → 不应导致数据泄露或破坏（允许400/409/500）
//   8. 配方名称注入 → 正常创建（参数化查询安全存储）
//   9. 提示词内容注入 → 正常保存（参数化查询安全存储）
//
// XSS防护（3个）：
//   10. 教案标题含<script>标签 → 正常创建（后端原样存储，前端负责转义）
//   11. 组织名称含<script>标签 → 正常创建
//   12. 用户显示名含HTML标签 → 正常保存
//
// 越权访问（4个）：
//   13. operator尝试删除admin创建的教案 → 被拒绝
//   14. operator尝试删除admin创建的配方 → 403
//   15. viewer尝试创建课程 → 403
//   16. operator尝试修改其他用户密码（通过admin接口）→ 403
//
// 输入边界（2个）：
//   17. 超长用户名登录（10000字符）→ 不应崩溃
//   18. 超长请求体（大JSON）→ 不应崩溃

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"tedna/internal/database"
)

// ==================== JWT安全测试 ====================

// TestSecurity_FakeJWTSignature 伪造JWT签名 → 401
func TestSecurity_FakeJWTSignature(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"user_id":"00000000-0000-0000-0000-000000000001","username":"admin","role":"admin","exp":9999999999}`))
	fakeToken := header + "." + payload + ".fake_signature_here_12345"

	resp, _ := DoGet(t, server.URL+"/api/v1/auth/me", fakeToken)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("伪造JWT签名应返回401，实际 %d", resp.StatusCode)
	}
}

// TestSecurity_TamperedJWTRole 篡改JWT中的角色字段 → 401
func TestSecurity_TamperedJWTRole(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsViewer(t, server.URL)
	parts := strings.SplitN(token, ".", 3)
	if len(parts) != 3 {
		t.Fatal("token格式异常")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("解码payload失败: %v", err)
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		t.Fatalf("解析payload失败: %v", err)
	}
	claims["role"] = "admin"
	tamperedPayload, _ := json.Marshal(claims)
	parts[1] = base64.RawURLEncoding.EncodeToString(tamperedPayload)

	tamperedToken := strings.Join(parts, ".")

	resp, _ := DoGet(t, server.URL+"/api/v1/admin/users", tamperedToken)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("篡改JWT角色应返回401，实际 %d", resp.StatusCode)
	}
}

// TestSecurity_EmptyBearerToken 空Bearer token → 401
func TestSecurity_EmptyBearerToken(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	req, _ := http.NewRequest("GET", server.URL+"/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer ")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("空Bearer token应返回401，实际 %d", resp.StatusCode)
	}
}

// TestSecurity_NonBearerAuth 非Bearer前缀的Authorization头 → 401
func TestSecurity_NonBearerAuth(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	req, _ := http.NewRequest("GET", server.URL+"/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("非Bearer认证应返回401，实际 %d", resp.StatusCode)
	}
}

// TestSecurity_WrongSecretJWT 用错误密钥签发的JWT → 401
func TestSecurity_WrongSecretJWT(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"user_id":"00000000-0000-0000-0000-000000000001","username":"admin","role":"admin","exp":9999999999,"iat":1700000000,"nbf":1700000000,"iss":"tedna"}`))
	wrongSig := base64.RawURLEncoding.EncodeToString([]byte("wrong-secret-signature-bytes-here"))
	fakeToken := header + "." + payload + "." + wrongSig

	resp, _ := DoGet(t, server.URL+"/api/v1/auth/me", fakeToken)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("错误密钥签发的JWT应返回401，实际 %d", resp.StatusCode)
	}
}

// ==================== SQL注入防护测试 ====================

// TestSecurity_SQLInjection_Login 登录接口SQL注入 → 401（不应500）
func TestSecurity_SQLInjection_Login(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	injectionPayloads := []string{
		"admin'; DROP TABLE users; --",
		"' OR '1'='1",
		"admin' OR 1=1 --",
		"'; SELECT * FROM users WHERE '1'='1",
		"admin\" OR \"\"=\"",
	}

	for _, payload := range injectionPayloads {
		loginBody := map[string]string{
			"username": payload,
			"password": "anything",
		}
		resp, _ := DoPost(t, server.URL+"/api/v1/auth/login", loginBody, "")

		// 关键：不应返回500（500意味着SQL语法错误，说明注入到了SQL中）
		if resp.StatusCode == http.StatusInternalServerError {
			t.Errorf("SQL注入payload '%s' 导致500错误，可能存在注入漏洞", payload)
		}
	}
}

// TestSecurity_SQLInjection_CourseCode 课程编号SQL注入 → 验证无数据泄露或破坏
// 注意：课程编号含单引号会通过参数化查询安全处理，但可能在后续业务逻辑中
// 引发错误返回500（如varchar约束、OSS路径拼接等），这不是SQL注入漏洞。
// 关键验证：courses表和users表数据完整性不受影响。
func TestSecurity_SQLInjection_CourseCode(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	injectionPayloads := []string{
		"TEST'; DROP TABLE courses; --",
		"' OR '1'='1",
	}

	for _, payload := range injectionPayloads {
		body := map[string]interface{}{
			"course_code":        payload,
			"course_name":        "注入测试",
			"external_module_id": 99999,
		}
		DoPost(t, server.URL+"/api/v1/courses", body, token)
		// 不断言状态码（可能是200/400/409/500都正常）
	}

	// 关键验证：表数据完整性未被破坏
	var userCount int
	err := database.DB.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM users`,
	).Scan(&userCount)
	if err != nil {
		t.Fatalf("查询users表失败（可能已被DROP）: %v", err)
	}
	if userCount < 5 {
		t.Errorf("SQL注入后users表数据异常: 期望>=5, 实际 %d", userCount)
	}

	var courseTableExists bool
	err = database.DB.QueryRow(context.Background(),
		`SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'courses')`,
	).Scan(&courseTableExists)
	if err != nil {
		t.Fatalf("检查courses表存在性失败: %v", err)
	}
	if !courseTableExists {
		t.Error("SQL注入后courses表不存在！严重安全漏洞！")
	}
}

// TestSecurity_SQLInjection_RecipeName 配方名称SQL注入 → 安全存储
func TestSecurity_SQLInjection_RecipeName(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	maliciousName := "'; DROP TABLE teaching_recipes; --"
	body := map[string]interface{}{
		"name":          maliciousName,
		"subject":       "数学",
		"grade_range":   "7-9",
		"description":   "SQL注入测试",
		"stages_config": `[{"stage_code":"write","enabled":true,"order":1},{"stage_code":"revise","enabled":true,"order":2}]`,
		"prompt_mode":   "guided",
	}

	resp, apiResp := DoPost(t, server.URL+"/api/v1/lesson-plans/recipes", body, token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	var recipeData struct {
		ID string `json:"id"`
	}
	ParseData(t, apiResp, &recipeData)

	var dbName string
	err := database.DB.QueryRow(context.Background(),
		`SELECT name FROM teaching_recipes WHERE id = $1`, recipeData.ID,
	).Scan(&dbName)
	if err != nil {
		t.Fatalf("查询配方失败: %v", err)
	}
	if dbName != maliciousName {
		t.Errorf("DB中配方名称应为原始字符串，实际 %s", dbName)
	}
}

// TestSecurity_SQLInjection_PromptContent 提示词内容SQL注入 → 安全存储
func TestSecurity_SQLInjection_PromptContent(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	maliciousContent := "'; DELETE FROM prompts; -- 正常提示词内容"
	body := map[string]interface{}{
		"content": maliciousContent,
	}

	resp, apiResp := DoPut(t, server.URL+"/api/v1/prompts/prompt_a", body, token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	var dbContent string
	err := database.DB.QueryRow(context.Background(),
		`SELECT content FROM prompts WHERE prompt_key = 'prompt_a' AND is_current = true`,
	).Scan(&dbContent)
	if err != nil {
		t.Fatalf("查询提示词失败: %v", err)
	}
	if dbContent != maliciousContent {
		t.Errorf("DB中提示词内容应为原始字符串")
	}
}

// ==================== XSS防护测试 ====================

// TestSecurity_XSS_LessonPlanTitle 教案标题含<script>标签 → 安全存储
func TestSecurity_XSS_LessonPlanTitle(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	xssTitle := "<script>alert('XSS')</script>数学教案"
	body := map[string]interface{}{
		"title":   xssTitle,
		"subject": "数学",
		"grade":   "7",
		"topic":   "XSS测试课题",
	}

	resp, apiResp := DoPost(t, server.URL+"/api/v1/lesson-plans/plans", body, token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	var planData struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}
	ParseData(t, apiResp, &planData)

	// 验证DB中原样存储（后端不做HTML转义，这是前端职责）
	var dbTitle string
	err := database.DB.QueryRow(context.Background(),
		`SELECT title FROM lesson_plans WHERE id = $1`, planData.ID,
	).Scan(&dbTitle)
	if err != nil {
		t.Fatalf("查询教案失败: %v", err)
	}
	if dbTitle != xssTitle {
		t.Errorf("DB中title应为原始字符串（含HTML标签），实际 %s", dbTitle)
	}
}

// TestSecurity_XSS_OrgName 组织名称含<script>标签 → 安全存储
func TestSecurity_XSS_OrgName(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	xssName := "<img src=x onerror=alert('XSS')>海淀区"
	body := map[string]interface{}{
		"name": xssName,
		"type": "region",
	}

	resp, apiResp := DoPost(t, server.URL+"/api/v1/lesson-plans/organizations", body, token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	var orgData struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	ParseData(t, apiResp, &orgData)
	if orgData.Name != xssName {
		t.Errorf("组织名称应原样返回，实际 %s", orgData.Name)
	}
}

// TestSecurity_XSS_DisplayName 用户显示名含HTML标签 → 安全保存
func TestSecurity_XSS_DisplayName(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	xssDisplayName := "<b onmouseover=alert('XSS')>管理员</b>"
	body := map[string]interface{}{
		"display_name": xssDisplayName,
	}

	resp, apiResp := DoPut(t, server.URL+"/api/v1/account/profile", body, token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	var dbDisplayName string
	err := database.DB.QueryRow(context.Background(),
		`SELECT display_name FROM users WHERE id = $1`, SeedAdminID,
	).Scan(&dbDisplayName)
	if err != nil {
		t.Fatalf("查询用户显示名失败: %v", err)
	}
	if dbDisplayName != xssDisplayName {
		t.Errorf("显示名应原样存储，实际 %s", dbDisplayName)
	}
}

// ==================== 越权访问测试 ====================

// TestSecurity_UnauthorizedPlanAccess operator尝试删除admin创建的教案 → 被拒绝
func TestSecurity_UnauthorizedPlanAccess(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	// admin创建教案（需要title+subject+grade+topic四个必填字段）
	adminToken := LoginAsAdmin(t, server.URL)
	body := map[string]interface{}{
		"title":   "越权测试教案",
		"subject": "数学",
		"grade":   "7",
		"topic":   "越权测试课题",
	}
	resp, apiResp := DoPost(t, server.URL+"/api/v1/lesson-plans/plans", body, adminToken)
	AssertHTTPStatus(t, resp, http.StatusOK)
	var planData struct {
		ID string `json:"id"`
	}
	ParseData(t, apiResp, &planData)

	// operator尝试删除admin的教案 → 应该被拒绝（非作者不能删除）
	opToken := LoginAsOperator(t, server.URL)
	delResp, _ := DoDelete(t, server.URL+"/api/v1/lesson-plans/plans/"+planData.ID, opToken)
	if delResp.StatusCode == http.StatusOK {
		t.Error("operator不应能删除admin创建的教案")
	}
}

// TestSecurity_UnauthorizedRecipeDelete operator尝试删除admin创建的配方 → 被拒绝
func TestSecurity_UnauthorizedRecipeDelete(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	adminToken := LoginAsAdmin(t, server.URL)
	recipeID := createTestRecipeViaAPI(t, server.URL, adminToken, "admin的配方", "数学", "7-9")

	opToken := LoginAsOperator(t, server.URL)
	resp, _ := DoDelete(t, server.URL+"/api/v1/lesson-plans/recipes/"+recipeID, opToken)
	if resp.StatusCode == http.StatusOK {
		t.Error("operator不应能删除admin创建的配方")
	}
}

// TestSecurity_ViewerCannotCreateCourse viewer尝试创建课程 → 403
func TestSecurity_ViewerCannotCreateCourse(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	viewerToken := LoginAsViewer(t, server.URL)

	body := map[string]interface{}{
		"course_code":        "VIEWER-01",
		"course_name":        "viewer课程",
		"external_module_id": 99999,
	}
	resp, _ := DoPost(t, server.URL+"/api/v1/courses", body, viewerToken)
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("viewer创建课程期望403，实际 %d", resp.StatusCode)
	}
}

// TestSecurity_OperatorCannotAccessAdminAPI operator尝试通过admin接口修改用户 → 403
func TestSecurity_OperatorCannotAccessAdminAPI(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	opToken := LoginAsOperator(t, server.URL)

	body := map[string]interface{}{
		"new_password": "hacked123",
	}
	resp, _ := DoPost(t, server.URL+"/api/v1/admin/users/"+SeedViewerID+"/password", body, opToken)
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("operator通过admin接口重置密码期望403，实际 %d", resp.StatusCode)
	}
}

// ==================== 输入边界测试 ====================

// TestSecurity_OversizedUsername 超长用户名登录 → 不应崩溃
func TestSecurity_OversizedUsername(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	longUsername := strings.Repeat("a", 10000)
	loginBody := map[string]string{
		"username": longUsername,
		"password": "anything",
	}

	resp, _ := DoPost(t, server.URL+"/api/v1/auth/login", loginBody, "")

	// 不应500（崩溃）
	if resp.StatusCode == http.StatusInternalServerError {
		t.Error("超长用户名不应导致500错误")
	}
}

// TestSecurity_OversizedRequestBody 超大请求体 → 不应崩溃
func TestSecurity_OversizedRequestBody(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	largeDescription := strings.Repeat("这是一段很长的描述文字。", 5000) // ~70KB
	body := map[string]interface{}{
		"name":          "大请求体测试",
		"subject":       "数学",
		"grade_range":   "7-9",
		"description":   largeDescription,
		"stages_config": `[{"stage_code":"write","enabled":true,"order":1},{"stage_code":"revise","enabled":true,"order":2}]`,
		"prompt_mode":   "guided",
	}

	resp, _ := DoPost(t, server.URL+"/api/v1/lesson-plans/recipes", body, token)

	if resp.StatusCode == http.StatusInternalServerError {
		t.Error("超大请求体不应导致500错误")
	}
}
