package integration

// pipeline_test.go — Pipeline生命周期集成测试
//
// 测试范围（15个用例）：
//   CRUD(5) + 状态操作(4) + 权限控制(3) + 批量操作+Dashboard(3)
//
// 关键设计：
//   使用 repository.CreateCourse 而非直接SQL插入课程（确保与生产代码完全一致的数据结构）

import (
	"context"
	"net/http"
	"testing"

	"tedna/internal/database"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== 辅助函数 ====================

// createTestCourse 通过repository层创建测试课程记录（确保与生产代码一致）
func createTestCourse(t *testing.T, courseCode string, courseName string) {
	t.Helper()
	moduleID := 99999 // 测试用外部模块ID
	course := &models.Course{
		CourseCode:       courseCode,
		CourseName:       courseName,
		ExternalModuleID: &moduleID,
		Status:           "active",
	}
	if err := repository.CreateCourse(course); err != nil {
		t.Fatalf("创建测试课程失败 (code=%s): %v", courseCode, err)
	}
}

// createTestCourseWithIndex 创建课程+索引
func createTestCourseWithIndex(t *testing.T, courseCode string, courseName string) {
	t.Helper()
	createTestCourse(t, courseCode, courseName)

	// 获取course_id
	course, err := repository.GetCourseByCode(courseCode)
	if err != nil {
		t.Fatalf("查询课程失败: %v", err)
	}

	// 插入索引
	idx := &models.CourseIndex{
		CourseID:     course.ID,
		IndexContent: "测试索引内容-用于集成测试-" + courseCode + "-需要超过50字符才能通过最小长度校验-这是额外填充内容确保足够长",
		IndexHash:    "test_hash_" + courseCode,
		PageCount:    10,
		TotalLength:  5000,
	}
	if err := repository.UpsertCourseIndex(idx); err != nil {
		t.Fatalf("创建测试索引失败: %v", err)
	}
}

// createPipelineViaAPI 通过API创建Pipeline，返回Pipeline ID
func createPipelineViaAPI(t *testing.T, serverURL string, token string, courseCode string) string {
	t.Helper()

	body := map[string]interface{}{
		"course_code": courseCode,
	}

	resp, apiResp := DoPost(t, serverURL+"/api/v1/pipelines", body, token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	var pipelineData struct {
		ID string `json:"id"`
	}
	ParseData(t, apiResp, &pipelineData)

	if pipelineData.ID == "" {
		t.Fatal("创建Pipeline成功但ID为空")
	}

	return pipelineData.ID
}

// getPipelineStatusFromDB 从DB直接读取Pipeline状态
func getPipelineStatusFromDB(t *testing.T, pipelineID string) string {
	t.Helper()
	var status string
	err := database.DB.QueryRow(context.Background(),
		`SELECT status FROM pipelines WHERE id = $1`, pipelineID,
	).Scan(&status)
	if err != nil {
		t.Fatalf("查询Pipeline状态失败: %v", err)
	}
	return status
}

// setPipelineStatusDirectly 直接通过DB设置Pipeline状态
func setPipelineStatusDirectly(t *testing.T, pipelineID string, status string) {
	t.Helper()
	_, err := database.DB.Exec(context.Background(),
		`UPDATE pipelines SET status = $1, updated_at = now() WHERE id = $2`,
		status, pipelineID,
	)
	if err != nil {
		t.Fatalf("直接更新Pipeline状态失败: %v", err)
	}
}

// ==================== Pipeline CRUD测试 ====================

// TestPipeline_Create 创建Pipeline → 成功+8个步骤记录
func TestPipeline_Create(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)
	createTestCourse(t, "TEST-G7-01", "七年级数学上册")

	pipelineID := createPipelineViaAPI(t, server.URL, token, "TEST-G7-01")

	// 验证状态为pending
	status := getPipelineStatusFromDB(t, pipelineID)
	if status != "pending" {
		t.Errorf("新建Pipeline状态应为pending，实际为 %s", status)
	}

	// 验证8个步骤记录已创建
	var stepCount int
	err := database.DB.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM pipeline_steps WHERE pipeline_id = $1`, pipelineID,
	).Scan(&stepCount)
	if err != nil {
		t.Fatalf("查询步骤数失败: %v", err)
	}
	if stepCount != 8 {
		t.Errorf("Pipeline应有8个步骤，实际有 %d 个", stepCount)
	}
}

// TestPipeline_CreateMissingCourseCode 缺少课程编号 → 400
func TestPipeline_CreateMissingCourseCode(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	body := map[string]interface{}{
		"course_code": "",
	}
	resp, _ := DoPost(t, server.URL+"/api/v1/pipelines", body, token)
	AssertHTTPStatus(t, resp, http.StatusBadRequest)
}

// TestPipeline_CreateCourseNotFound 不存在的课程 → 404
func TestPipeline_CreateCourseNotFound(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	body := map[string]interface{}{
		"course_code": "NONEXISTENT-999",
	}
	resp, _ := DoPost(t, server.URL+"/api/v1/pipelines", body, token)
	AssertHTTPStatus(t, resp, http.StatusNotFound)
}

// TestPipeline_GetDetail 获取Pipeline详情
func TestPipeline_GetDetail(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)
	createTestCourse(t, "TEST-G7-02", "七年级语文上册")
	pipelineID := createPipelineViaAPI(t, server.URL, token, "TEST-G7-02")

	resp, apiResp := DoGet(t, server.URL+"/api/v1/pipelines/"+pipelineID, token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	var detail struct {
		ID          string `json:"id"`
		CourseCode  string `json:"course_code"`
		Status      string `json:"status"`
		CurrentStep string `json:"current_step"`
		Steps       []struct {
			StepName string `json:"step_name"`
		} `json:"steps"`
	}
	ParseData(t, apiResp, &detail)

	if detail.ID != pipelineID {
		t.Errorf("Pipeline ID不匹配")
	}
	if detail.Status != "pending" {
		t.Errorf("状态不匹配: 期望 pending, 实际 %s", detail.Status)
	}
	if len(detail.Steps) != 8 {
		t.Errorf("步骤数不匹配: 期望 8, 实际 %d", len(detail.Steps))
	}
}

// TestPipeline_Delete 删除Pipeline → 成功+级联删除
func TestPipeline_Delete(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)
	createTestCourse(t, "TEST-G7-03", "七年级英语上册")
	pipelineID := createPipelineViaAPI(t, server.URL, token, "TEST-G7-03")

	resp, apiResp := DoDelete(t, server.URL+"/api/v1/pipelines/"+pipelineID, token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	// 确认Pipeline和步骤都级联删除
	var pCount, sCount int
	err := database.DB.QueryRow(context.Background(), `SELECT COUNT(*) FROM pipelines WHERE id = $1`, pipelineID).Scan(&pCount)
	if err != nil {
		t.Fatalf("查询Pipeline数失败: %v", err)
	}
	err = database.DB.QueryRow(context.Background(), `SELECT COUNT(*) FROM pipeline_steps WHERE pipeline_id = $1`, pipelineID).Scan(&sCount)
	if err != nil {
		t.Fatalf("查询步骤数失败: %v", err)
	}
	if pCount != 0 {
		t.Error("删除后Pipeline记录仍存在")
	}
	if sCount != 0 {
		t.Errorf("删除后步骤记录仍存在，剩余 %d 条", sCount)
	}
}

// ==================== Pipeline状态操作测试 ====================

// TestPipeline_CancelPending 取消pending状态Pipeline → 成功
func TestPipeline_CancelPending(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)
	createTestCourse(t, "TEST-G7-04", "七年级历史上册")
	pipelineID := createPipelineViaAPI(t, server.URL, token, "TEST-G7-04")

	resp, apiResp := DoPost(t, server.URL+"/api/v1/pipelines/"+pipelineID+"/cancel", nil, token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	status := getPipelineStatusFromDB(t, pipelineID)
	if status != "cancelled" {
		t.Errorf("取消后状态应为cancelled，实际为 %s", status)
	}
}

// TestPipeline_CancelAlreadyCancelled 取消已取消的Pipeline → 409
func TestPipeline_CancelAlreadyCancelled(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)
	createTestCourse(t, "TEST-G7-05", "七年级地理上册")
	pipelineID := createPipelineViaAPI(t, server.URL, token, "TEST-G7-05")
	setPipelineStatusDirectly(t, pipelineID, "cancelled")

	resp, _ := DoPost(t, server.URL+"/api/v1/pipelines/"+pipelineID+"/cancel", nil, token)
	AssertHTTPStatus(t, resp, http.StatusConflict)
}

// TestPipeline_DeleteRunning 删除运行中Pipeline → 409
func TestPipeline_DeleteRunning(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)
	createTestCourse(t, "TEST-G7-06", "七年级科学上册")
	pipelineID := createPipelineViaAPI(t, server.URL, token, "TEST-G7-06")
	setPipelineStatusDirectly(t, pipelineID, "running")

	resp, _ := DoDelete(t, server.URL+"/api/v1/pipelines/"+pipelineID, token)
	AssertHTTPStatus(t, resp, http.StatusConflict)
}

// TestPipeline_ListByRole Pipeline列表
func TestPipeline_ListByRole(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	adminToken := LoginAsAdmin(t, server.URL)
	createTestCourse(t, "TEST-G7-07", "七年级数学下册")
	createPipelineViaAPI(t, server.URL, adminToken, "TEST-G7-07")

	resp, apiResp := DoGet(t, server.URL+"/api/v1/pipelines", adminToken)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	var listData struct {
		Total int `json:"total"`
	}
	ParseData(t, apiResp, &listData)

	if listData.Total < 1 {
		t.Errorf("admin应至少能看到1个Pipeline，实际 %d", listData.Total)
	}
}

// ==================== 权限控制测试 ====================

// TestPipeline_ViewerCannotCreate viewer不能创建Pipeline → 403
func TestPipeline_ViewerCannotCreate(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	viewerToken := LoginAsViewer(t, server.URL)
	createTestCourse(t, "TEST-G7-08", "七年级美术上册")

	body := map[string]interface{}{"course_code": "TEST-G7-08"}
	resp, _ := DoPost(t, server.URL+"/api/v1/pipelines", body, viewerToken)
	AssertHTTPStatus(t, resp, http.StatusForbidden)
}

// TestPipeline_ViewerCannotCancel viewer不能取消Pipeline → 403
func TestPipeline_ViewerCannotCancel(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	adminToken := LoginAsAdmin(t, server.URL)
	viewerToken := LoginAsViewer(t, server.URL)
	createTestCourse(t, "TEST-G7-09", "七年级音乐上册")
	pipelineID := createPipelineViaAPI(t, server.URL, adminToken, "TEST-G7-09")

	resp, _ := DoPost(t, server.URL+"/api/v1/pipelines/"+pipelineID+"/cancel", nil, viewerToken)
	AssertHTTPStatus(t, resp, http.StatusForbidden)
}

// TestPipeline_AdminCanDelete admin可以删除Pipeline → 200
func TestPipeline_AdminCanDelete(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)
	createTestCourse(t, "TEST-G7-10", "七年级体育上册")
	pipelineID := createPipelineViaAPI(t, server.URL, token, "TEST-G7-10")

	resp, apiResp := DoDelete(t, server.URL+"/api/v1/pipelines/"+pipelineID, token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)
}

// ==================== 批量操作+Dashboard测试 ====================

// TestPipeline_BatchCreate 批量创建Pipeline
func TestPipeline_BatchCreate(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)
	createTestCourse(t, "BATCH-01", "批量测试1")
	createTestCourse(t, "BATCH-02", "批量测试2")
	createTestCourse(t, "BATCH-03", "批量测试3")

	body := map[string]interface{}{
		"course_codes": []string{"BATCH-01", "BATCH-02", "BATCH-03"},
	}
	resp, apiResp := DoPost(t, server.URL+"/api/v1/pipelines/batch-create", body, token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	// 验证3个Pipeline已创建
	var pipelineCount int
	err := database.DB.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM pipelines WHERE course_code IN ('BATCH-01','BATCH-02','BATCH-03')`,
	).Scan(&pipelineCount)
	if err != nil {
		t.Fatalf("查询Pipeline数失败: %v", err)
	}
	if pipelineCount != 3 {
		t.Errorf("批量创建后应有3个Pipeline，实际 %d", pipelineCount)
	}
}

// TestPipeline_BatchStart 批量启动Pipeline
func TestPipeline_BatchStart(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)
	createTestCourseWithIndex(t, "START-01", "启动测试1")
	createTestCourseWithIndex(t, "START-02", "启动测试2")

	id1 := createPipelineViaAPI(t, server.URL, token, "START-01")
	id2 := createPipelineViaAPI(t, server.URL, token, "START-02")

	body := map[string]interface{}{
		"pipeline_ids": []string{id1, id2},
	}
	resp, apiResp := DoPost(t, server.URL+"/api/v1/pipelines/batch-start", body, token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	// 批量启动后状态不再是pending（可能running或failed，因为没有真实AI）
	status1 := getPipelineStatusFromDB(t, id1)
	if status1 == "pending" {
		t.Errorf("批量启动后Pipeline1状态不应仍为pending，实际为 %s", status1)
	}
}

// TestPipeline_Dashboard Dashboard统计
func TestPipeline_Dashboard(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)
	createTestCourse(t, "DASH-01", "Dashboard测试1")
	createPipelineViaAPI(t, server.URL, token, "DASH-01")

	resp, apiResp := DoGet(t, server.URL+"/api/v1/dashboard/stats", token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	var stats struct {
		TotalCourses   int `json:"total_courses"`
		TotalPipelines int `json:"total_pipelines"`
	}
	ParseData(t, apiResp, &stats)

	if stats.TotalCourses < 1 {
		t.Errorf("课程数应>=1，实际 %d", stats.TotalCourses)
	}
	if stats.TotalPipelines < 1 {
		t.Errorf("Pipeline数应>=1，实际 %d", stats.TotalPipelines)
	}
}
