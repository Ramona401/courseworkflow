package integration

// test_helper.go — 集成测试基础设施
//
// 职责：
//   1. 连接 tedna_test 测试数据库（与生产 tedna 完全隔离）
//   2. 清空所有表数据 + 插入种子数据（每个测试用例前调用）
//   3. 启动真实 HTTP 服务（httptest.NewServer）
//   4. 提供便捷的 HTTP 请求辅助函数（带/不带 Token）
//   5. 提供 JSON 解析辅助函数
//
// 设计原则：
//   - 测试数据库名固定为 tedna_test，绝不连接生产库
//   - 每个 TestXxx 函数开头调用 CleanAndSeed() 保证数据干净
//   - 种子数据使用固定 UUID，方便断言
//   - HTTP 辅助函数自动处理 JSON 编解码
//
// v98变更：种子阶段增加skippable字段，与生产数据一致（write/revise=false，其余=true）

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"tedna/internal/config"
	"tedna/internal/database"
	"tedna/internal/repository"
	"tedna/internal/routes"
	"tedna/internal/utils"
)

// ==================== 固定种子数据常量 ====================

const (
	// 管理员种子用户（与生产种子一致）
	SeedAdminID       = "00000000-0000-0000-0000-000000000001"
	SeedAdminUsername  = "admin"
	SeedAdminPassword  = "admin123"
	SeedAdminRole      = "admin"

	// 操作员种子用户
	SeedOperatorID       = "00000000-0000-0000-0000-000000000002"
	SeedOperatorUsername  = "operator1"
	SeedOperatorPassword  = "operator123"
	SeedOperatorRole      = "operator"

	// 高级操作员种子用户
	SeedSeniorID       = "00000000-0000-0000-0000-000000000003"
	SeedSeniorUsername  = "senior1"
	SeedSeniorPassword  = "senior123"
	SeedSeniorRole      = "senior_operator"

	// 只读用户种子
	SeedViewerID       = "00000000-0000-0000-0000-000000000004"
	SeedViewerUsername  = "viewer1"
	SeedViewerPassword  = "viewer123"
	SeedViewerRole      = "viewer"

	// 被禁用的用户种子
	SeedDisabledID       = "00000000-0000-0000-0000-000000000005"
	SeedDisabledUsername  = "disabled1"
	SeedDisabledPassword  = "disabled123"
	SeedDisabledRole      = "viewer"
)

// ==================== 测试配置 ====================

// testConfig 返回指向 tedna_test 数据库的测试配置
// 关键安全措施：DB_NAME 硬编码为 "tedna_test"，不从环境变量读取，防止误连生产库
func testConfig() *config.Config {
	return &config.Config{
		DBHost:     envOrDefault("TEST_DB_HOST", "127.0.0.1"),
		DBPort:     envOrDefault("TEST_DB_PORT", "5432"),
		DBUser:     envOrDefault("TEST_DB_USER", "tedna_user"),
		DBPassword: envOrDefault("TEST_DB_PASSWORD", "9fIbnkYABWXt3VGPv8Pn"),
		DBName:     "tedna_test", // 硬编码！绝不允许改为 tedna
		Port:       "0",          // httptest 自动分配端口
		JWTSecret:  "test-jwt-secret-for-integration-tests-only",
		AESKey:     "c94985251907d9a973ee517d048d8430",
		GinMode:           "test",
		DisableSchedulers: true, // v142: 测试环境禁用调度器防goroutine泄漏
	}
}

// envOrDefault 读取环境变量，不存在则返回默认值
func envOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// ==================== 数据库初始化 ====================

// initTestDB 初始化测试数据库连接
// 直接设置 database.DB 全局变量，使所有 repository 层代码指向测试库
func initTestDB(t *testing.T, cfg *config.Config) {
	t.Helper()

	// 构建 DSN（与 database.Init 逻辑一致，但指向 tedna_test）
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.DBUser, cfg.DBPassword,
		cfg.DBHost, cfg.DBPort, cfg.DBName,
	)

	// 二次确认数据库名（防御性编程）
	if cfg.DBName != "tedna_test" {
		t.Fatalf("安全检查失败：测试数据库名必须为 tedna_test，实际为 %s", cfg.DBName)
	}

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("解析测试数据库DSN失败: %v", err)
	}

	// 测试环境使用较小的连接池
	poolConfig.MaxConns = 10
	poolConfig.MinConns = 2
	poolConfig.MaxConnLifetime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		t.Fatalf("连接测试数据库失败: %v", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		t.Fatalf("Ping测试数据库失败: %v", err)
	}

	// 设置全局 DB，所有 repository 层自动使用测试库
	database.DB = pool

	// 注册清理：测试结束后关闭连接池
	t.Cleanup(func() {
		pool.Close()
	})
}

// ==================== 数据清理与种子 ====================

// CleanAndSeed 清空所有表数据并插入种子数据
// 每个 TestXxx 函数开头调用，保证测试数据隔离
func CleanAndSeed(t *testing.T) {
	t.Helper()
	ctx := context.Background()

	// 按外键依赖顺序清空（先子表后父表）
	tables := []string{
		"workshop_stage_outputs",
		"workshop_stages",
		"recipe_usage_log",
		"teaching_recipes",
		"component_extractions",
		"lesson_plan_reviews",
		"lesson_plans",
		"prompt_templates",
		"teaching_group_members",
		"teaching_groups",
		"organizations",
		"generated_pages",
		"pipeline_indexes",
		"eval_rounds",
		"pipeline_steps",
		"pipelines",
		"ai_call_traces",
		"course_indexes",
		"courses",
		"role_permissions",
		"roles",
		"textbook_pages",
		"user_course_assignments",
		"audit_logs",
		"ai_scene_configs",
		"ai_configs",
		"external_data_configs",
		"prompts",
		"users",
	}

	for _, table := range tables {
		_, err := database.DB.Exec(ctx, fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			t.Fatalf("清空表 %s 失败: %v", table, err)
		}
	}

	// 插入种子用户
	seedUsers := []struct {
		id, username, displayName, password, role, status string
	}{
		{SeedAdminID, SeedAdminUsername, "管理员", SeedAdminPassword, SeedAdminRole, "active"},
		{SeedOperatorID, SeedOperatorUsername, "操作员1", SeedOperatorPassword, SeedOperatorRole, "active"},
		{SeedSeniorID, SeedSeniorUsername, "高级操作员1", SeedSeniorPassword, SeedSeniorRole, "active"},
		{SeedViewerID, SeedViewerUsername, "查看者1", SeedViewerPassword, SeedViewerRole, "active"},
		{SeedDisabledID, SeedDisabledUsername, "禁用用户1", SeedDisabledPassword, SeedDisabledRole, "disabled"},
	}

	for _, u := range seedUsers {
		hash, err := utils.HashPassword(u.password)
		if err != nil {
			t.Fatalf("哈希密码失败 (user=%s): %v", u.username, err)
		}
		_, err = database.DB.Exec(ctx,
			`INSERT INTO users (id, username, display_name, password_hash, role, status, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, now(), now())`,
			u.id, u.username, u.displayName, hash, u.role, u.status,
		)
		if err != nil {
			t.Fatalf("插入种子用户失败 (user=%s): %v", u.username, err)
		}
	}

	// 插入种子备课工坊阶段（5个系统默认阶段，很多服务依赖它们）
	// v98修复：增加skippable字段，与生产数据一致（write/revise=false，其余=true）
	seedStages := []struct {
		code, name string
		order      int
		aiRole     string
		skippable  bool
	}{
		{"analyze", "教学分析", 1, "教学分析师", true},
		{"design", "教学设计", 2, "教学设计师", true},
		{"write", "教案撰写", 3, "教案撰写专家", false},
		{"review", "AI评审", 4, "教案评审专家", true},
		{"revise", "修订定稿", 5, "教案修订专家", false},
	}

	for _, s := range seedStages {
		_, err := database.DB.Exec(ctx,
			`INSERT INTO workshop_stages (id, stage_code, stage_name, stage_order, source, ai_role, system_prompt, skippable, status, created_at, updated_at)
			 VALUES (gen_random_uuid(), $1, $2, $3, 'system', $4, '', $5, 'active', now(), now())`,
			s.code, s.name, s.order, s.aiRole, s.skippable,
		)
		if err != nil {
			t.Fatalf("插入种子阶段失败 (stage=%s): %v", s.code, err)
		}
	}
}

// ==================== HTTP 服务启动 ====================

// SetupTestServer 初始化测试数据库 + 启动 HTTP 测试服务器
// 返回 httptest.Server 和测试配置
// 调用方在测试结束后 server.Close()
func SetupTestServer(t *testing.T) (*httptest.Server, *config.Config) {
	t.Helper()

	cfg := testConfig()
	initTestDB(t, cfg)

	// 启动 AI 调用追踪异步写入器（routes.Setup 中会调用）
	repository.InitTraceWriter()

	// 使用与生产相同的路由注册逻辑
	handler := routes.Setup(cfg)

	// 创建测试 HTTP 服务器
	server := httptest.NewServer(handler)

	t.Cleanup(func() {
		server.Close()
	})

	return server, cfg
}

// ==================== HTTP 请求辅助函数 ====================

// APIResponse 统一 API 响应结构（与 utils.Response 对应）
type APIResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// DoRequest 发送 HTTP 请求并返回响应
// method: GET/POST/PUT/DELETE
// url: 完整URL（含 server.URL 前缀）
// body: 请求体（nil 表示无 body）
// token: JWT token（空字符串表示不带认证头）
func DoRequest(t *testing.T, method, url string, body interface{}, token string) (*http.Response, *APIResponse) {
	t.Helper()

	var bodyReader io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("序列化请求体失败: %v", err)
		}
		bodyReader = bytes.NewReader(jsonBytes)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		t.Fatalf("创建HTTP请求失败: %v", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("发送HTTP请求失败: %v", err)
	}

	// 读取并解析响应体
	respBody, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		t.Fatalf("读取响应体失败: %v", err)
	}

	var apiResp APIResponse
	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, &apiResp); err != nil {
			// 非JSON响应（如SSE流），不解析
			apiResp = APIResponse{Code: -999, Message: string(respBody)}
		}
	}

	return resp, &apiResp
}

// DoGet 发送 GET 请求
func DoGet(t *testing.T, url string, token string) (*http.Response, *APIResponse) {
	t.Helper()
	return DoRequest(t, http.MethodGet, url, nil, token)
}

// DoPost 发送 POST 请求
func DoPost(t *testing.T, url string, body interface{}, token string) (*http.Response, *APIResponse) {
	t.Helper()
	return DoRequest(t, http.MethodPost, url, body, token)
}

// DoPut 发送 PUT 请求
func DoPut(t *testing.T, url string, body interface{}, token string) (*http.Response, *APIResponse) {
	t.Helper()
	return DoRequest(t, http.MethodPut, url, body, token)
}

// DoDelete 发送 DELETE 请求
func DoDelete(t *testing.T, url string, token string) (*http.Response, *APIResponse) {
	t.Helper()
	return DoRequest(t, http.MethodDelete, url, nil, token)
}

// ==================== 登录辅助函数 ====================

// LoginAs 使用指定用户名密码登录，返回 JWT token
// 登录失败时直接 t.Fatal
func LoginAs(t *testing.T, serverURL, username, password string) string {
	t.Helper()

	loginBody := map[string]string{
		"username": username,
		"password": password,
	}

	resp, apiResp := DoPost(t, serverURL+"/api/v1/auth/login", loginBody, "")

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("登录失败 (user=%s): HTTP %d, message=%s", username, resp.StatusCode, apiResp.Message)
	}

	if apiResp.Code != 0 {
		t.Fatalf("登录失败 (user=%s): code=%d, message=%s", username, apiResp.Code, apiResp.Message)
	}

	// 从响应中提取 token
	var loginData struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(apiResp.Data, &loginData); err != nil {
		t.Fatalf("解析登录响应失败 (user=%s): %v", username, err)
	}

	if loginData.Token == "" {
		t.Fatalf("登录成功但 token 为空 (user=%s)", username)
	}

	return loginData.Token
}

// LoginAsAdmin 以管理员身份登录
func LoginAsAdmin(t *testing.T, serverURL string) string {
	t.Helper()
	return LoginAs(t, serverURL, SeedAdminUsername, SeedAdminPassword)
}

// LoginAsOperator 以操作员身份登录
func LoginAsOperator(t *testing.T, serverURL string) string {
	t.Helper()
	return LoginAs(t, serverURL, SeedOperatorUsername, SeedOperatorPassword)
}

// LoginAsSenior 以高级操作员身份登录
func LoginAsSenior(t *testing.T, serverURL string) string {
	t.Helper()
	return LoginAs(t, serverURL, SeedSeniorUsername, SeedSeniorPassword)
}

// LoginAsViewer 以只读用户身份登录
func LoginAsViewer(t *testing.T, serverURL string) string {
	t.Helper()
	return LoginAs(t, serverURL, SeedViewerUsername, SeedViewerPassword)
}

// ==================== JSON 解析辅助函数 ====================

// ParseData 将 APIResponse.Data 解析到目标结构体
func ParseData(t *testing.T, apiResp *APIResponse, target interface{}) {
	t.Helper()
	if apiResp.Data == nil {
		t.Fatal("API响应Data为nil")
	}
	if err := json.Unmarshal(apiResp.Data, target); err != nil {
		t.Fatalf("解析API响应Data失败: %v, raw=%s", err, string(apiResp.Data))
	}
}

// ==================== 断言辅助函数 ====================

// AssertHTTPStatus 断言 HTTP 状态码
func AssertHTTPStatus(t *testing.T, resp *http.Response, expected int) {
	t.Helper()
	if resp.StatusCode != expected {
		t.Errorf("期望HTTP状态码 %d，实际 %d", expected, resp.StatusCode)
	}
}

// AssertAPICode 断言 API 业务码
func AssertAPICode(t *testing.T, apiResp *APIResponse, expected int) {
	t.Helper()
	if apiResp.Code != expected {
		t.Errorf("期望API Code %d，实际 %d (message=%s)", expected, apiResp.Code, apiResp.Message)
	}
}
