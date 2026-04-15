package integration

// course_test.go — 课程管理集成测试
//
// 测试范围（10个用例）：
//
// 课程CRUD（5个）：
//   1. 创建课程（admin）→ 成功
//   2. 缺少必填字段（course_code为空）→ 400
//   3. 课程编号重复 → 409
//   4. 课程列表（admin）→ 返回所有课程
//   5. 课程列表（operator无分配）→ 返回空列表
//
// 索引相关（3个）：
//   6. 获取索引摘要（无索引）→ has_index=false
//   7. 获取完整索引（无索引）→ 500（课程尚未拉取索引）
//   8. 非admin获取完整索引 → 403
//
// 权限控制（2个）：
//   9. 非登录用户访问课程列表 → 401
//   10. operator创建课程 → 403
//
// 注意：
//   - FetchIndex/RegisterAndFetch/BatchRegisterAndFetch 依赖外部OSS服务，不纳入集成测试
//   - 课程创建需要 external_module_id > 0，使用虚拟值99999避免与真实数据冲突

import (
	"context"
	"net/http"
	"testing"

	"tedna/internal/database"
)

// ==================== 辅助函数 ====================

// createTestCourseViaAPI 通过API创建测试课程，返回课程ID
// 使用虚拟module_id避免与真实OSS数据冲突
func createTestCourseViaAPI(t *testing.T, serverURL string, token string, courseCode string, courseName string, moduleID int) string {
	t.Helper()

	body := map[string]interface{}{
		"course_code":        courseCode,
		"course_name":        courseName,
		"external_module_id": moduleID,
	}

	resp, apiResp := DoPost(t, serverURL+"/api/v1/courses", body, token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	var courseData struct {
		ID         string `json:"id"`
		CourseCode string `json:"course_code"`
	}
	ParseData(t, apiResp, &courseData)

	if courseData.ID == "" {
		t.Fatal("创建课程成功但ID为空")
	}

	return courseData.ID
}

// ==================== 课程CRUD测试 ====================

// TestCourse_Create 创建课程（admin）→ 成功
func TestCourse_Create(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)
	courseID := createTestCourseViaAPI(t, server.URL, token, "TEST-01", "集成测试课程01", 99901)

	// 验证DB中存在
	var count int
	err := database.DB.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM courses WHERE id = $1`, courseID,
	).Scan(&count)
	if err != nil {
		t.Fatalf("查询课程失败: %v", err)
	}
	if count != 1 {
		t.Error("创建课程后DB中应有1条记录")
	}

	// 验证字段正确性
	var code, name, status string
	err = database.DB.QueryRow(context.Background(),
		`SELECT course_code, course_name, status FROM courses WHERE id = $1`, courseID,
	).Scan(&code, &name, &status)
	if err != nil {
		t.Fatalf("查询课程字段失败: %v", err)
	}
	if code != "TEST-01" {
		t.Errorf("课程编号不匹配: 期望 TEST-01, 实际 %s", code)
	}
	if name != "集成测试课程01" {
		t.Errorf("课程名称不匹配: 期望 集成测试课程01, 实际 %s", name)
	}
	if status != "active" {
		t.Errorf("课程状态应为active，实际 %s", status)
	}
}

// TestCourse_CreateMissingFields 缺少必填字段 → 400
func TestCourse_CreateMissingFields(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	// course_code为空 → 服务层返回"课程编号不能为空"
	body := map[string]interface{}{
		"course_code":        "",
		"course_name":        "测试课程",
		"external_module_id": 99902,
	}
	resp, _ := DoPost(t, server.URL+"/api/v1/courses", body, token)
	AssertHTTPStatus(t, resp, http.StatusBadRequest)
}

// TestCourse_CreateDuplicate 课程编号重复 → 409
func TestCourse_CreateDuplicate(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	// 先创建一个课程
	createTestCourseViaAPI(t, server.URL, token, "DUP-01", "重复测试课程", 99903)

	// 再用相同编号创建 → 409 Conflict
	body := map[string]interface{}{
		"course_code":        "DUP-01",
		"course_name":        "重复测试课程2",
		"external_module_id": 99904,
	}
	resp, _ := DoPost(t, server.URL+"/api/v1/courses", body, token)
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("期望HTTP状态码 409，实际 %d", resp.StatusCode)
	}
}

// TestCourse_ListAdmin 课程列表（admin）→ 返回所有课程
func TestCourse_ListAdmin(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	// 创建2个课程
	createTestCourseViaAPI(t, server.URL, token, "LIST-01", "列表测试1", 99905)
	createTestCourseViaAPI(t, server.URL, token, "LIST-02", "列表测试2", 99906)

	// 查询列表
	resp, apiResp := DoGet(t, server.URL+"/api/v1/courses", token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	var listData struct {
		Courses []struct {
			ID         string `json:"id"`
			CourseCode string `json:"course_code"`
		} `json:"courses"`
		Total int `json:"total"`
	}
	ParseData(t, apiResp, &listData)

	if listData.Total < 2 {
		t.Errorf("admin应至少看到2个课程，实际 %d", listData.Total)
	}
}

// TestCourse_ListOperatorEmpty 课程列表（operator无分配）→ 返回空列表
func TestCourse_ListOperatorEmpty(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	adminToken := LoginAsAdmin(t, server.URL)

	// admin创建课程（operator未分配）
	createTestCourseViaAPI(t, server.URL, adminToken, "NOACCESS-01", "无权限课程", 99907)

	// operator查询 → 应返回空列表
	opToken := LoginAsOperator(t, server.URL)
	resp, apiResp := DoGet(t, server.URL+"/api/v1/courses", opToken)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	var listData struct {
		Total int `json:"total"`
	}
	ParseData(t, apiResp, &listData)

	if listData.Total != 0 {
		t.Errorf("operator无分配时应返回0个课程，实际 %d", listData.Total)
	}
}

// ==================== 索引相关测试 ====================

// TestCourse_IndexSummaryNoIndex 获取索引摘要（无索引）→ has_index=false
func TestCourse_IndexSummaryNoIndex(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	// 创建课程（不拉取索引）
	createTestCourseViaAPI(t, server.URL, token, "NOINDEX-01", "无索引课程", 99908)

	// 获取索引摘要
	resp, apiResp := DoGet(t, server.URL+"/api/v1/courses/NOINDEX-01/index-summary", token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	var summary struct {
		CourseCode string `json:"course_code"`
		HasIndex   bool   `json:"has_index"`
	}
	ParseData(t, apiResp, &summary)

	if summary.HasIndex {
		t.Error("未拉取索引的课程 has_index 应为 false")
	}
	if summary.CourseCode != "NOINDEX-01" {
		t.Errorf("课程编号不匹配: 期望 NOINDEX-01, 实际 %s", summary.CourseCode)
	}
}

// TestCourse_IndexFullNoIndex 获取完整索引（无索引）→ 500
func TestCourse_IndexFullNoIndex(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	// 创建课程（不拉取索引）
	createTestCourseViaAPI(t, server.URL, token, "NOINDEX-02", "无索引课程2", 99909)

	// 获取完整索引 → 应返回500（课程尚未拉取索引）
	resp, _ := DoGet(t, server.URL+"/api/v1/courses/NOINDEX-02/index", token)
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("无索引时获取完整索引期望500，实际 %d", resp.StatusCode)
	}
}

// TestCourse_IndexFullForbiddenNonAdmin 非admin获取完整索引 → 403
func TestCourse_IndexFullForbiddenNonAdmin(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	adminToken := LoginAsAdmin(t, server.URL)
	createTestCourseViaAPI(t, server.URL, adminToken, "FORBID-01", "禁止课程", 99910)

	// operator尝试获取完整索引 → 403
	opToken := LoginAsOperator(t, server.URL)
	resp, _ := DoGet(t, server.URL+"/api/v1/courses/FORBID-01/index", opToken)
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("非admin获取完整索引期望403，实际 %d", resp.StatusCode)
	}
}

// ==================== 权限控制测试 ====================

// TestCourse_ListUnauthorized 未登录用户访问课程列表 → 401
func TestCourse_ListUnauthorized(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	// 不带token访问
	resp, _ := DoGet(t, server.URL+"/api/v1/courses", "")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("未登录访问课程列表期望401，实际 %d", resp.StatusCode)
	}
}

// TestCourse_CreateForbiddenOperator operator创建课程 → 403
func TestCourse_CreateForbiddenOperator(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	opToken := LoginAsOperator(t, server.URL)

	body := map[string]interface{}{
		"course_code":        "OP-01",
		"course_name":        "operator课程",
		"external_module_id": 99911,
	}
	resp, _ := DoPost(t, server.URL+"/api/v1/courses", body, opToken)
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("operator创建课程期望403，实际 %d", resp.StatusCode)
	}
}
