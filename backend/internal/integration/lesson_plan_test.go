package integration

// lesson_plan_test.go — 教案生命周期集成测试
//
// 测试范围（20个用例）：
//
// 教案CRUD（7个）：
//   1. 创建教案 → 成功返回教案对象
//   2. 缺少必填字段创建 → 400
//   3. 获取教案详情 → 返回完整信息
//   4. 获取不存在的教案 → 404
//   5. 更新教案内容 → 成功
//   6. 非作者更新教案 → 403
//   7. 删除教案 → 成功+确认DB已删除
//
// 教案状态流转（8个）：
//   8. draft → published_personal（个人发布）
//   9. draft → submitted（提交评审，需指定group）
//   10. submitted → approved（评审通过）
//   11. submitted → revision（退回修改）
//   12. approved → published_shared（共享发布）
//   13. revision → published_personal（退回后可再次个人发布）
//   14. 非法状态转换 → 失败
//   15. 非作者操作 → 403
//
// Fork测试（2个）：
//   16. Fork已发布教案 → 成功+fork_count递增
//   17. Fork草稿教案 → 失败
//
// 列表与筛选（3个）：
//   18. 按作者筛选教案列表
//   19. 按状态筛选教案列表
//   20. 教案列表分页

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"tedna/internal/database"
)

// ==================== 辅助函数 ====================

// createTestLessonPlan 通过API创建测试教案，返回教案ID
func createTestLessonPlan(t *testing.T, serverURL string, token string, subject string, grade string, topic string) string {
	t.Helper()

	body := map[string]interface{}{
		"title":            fmt.Sprintf("%s %s — %s", grade, subject, topic),
		"subject":          subject,
		"grade":            grade,
		"topic":            topic,
		"duration_minutes": 45,
	}

	resp, apiResp := DoPost(t, serverURL+"/api/v1/lesson-plans/plans", body, token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	var planData struct {
		ID string `json:"id"`
	}
	ParseData(t, apiResp, &planData)

	if planData.ID == "" {
		t.Fatal("创建教案成功但ID为空")
	}

	return planData.ID
}

// createTestOrganization 通过DB直接创建测试组织，返回组织ID
func createTestOrganization(t *testing.T, name string, orgType string) string {
	t.Helper()
	var id string
	err := database.DB.QueryRow(context.Background(),
		`INSERT INTO organizations (name, type, status, created_at, updated_at)
		 VALUES ($1, $2, 'active', now(), now())
		 RETURNING id`,
		name, orgType,
	).Scan(&id)
	if err != nil {
		t.Fatalf("创建测试组织失败: %v", err)
	}
	return id
}

// createTestTeachingGroup 通过DB直接创建测试教研组，返回教研组ID
func createTestTeachingGroup(t *testing.T, schoolID string, name string, subject string) string {
	t.Helper()
	var id string
	err := database.DB.QueryRow(context.Background(),
		`INSERT INTO teaching_groups (name, school_id, subject, status, created_at, updated_at)
		 VALUES ($1, $2, $3, 'active', now(), now())
		 RETURNING id`,
		name, schoolID, subject,
	).Scan(&id)
	if err != nil {
		t.Fatalf("创建测试教研组失败: %v", err)
	}
	return id
}

// updateLessonPlanStatusDirectly 通过DB直接更新教案状态（绕过业务校验，用于测试前置条件设置）
func updateLessonPlanStatusDirectly(t *testing.T, planID string, status string) {
	t.Helper()
	_, err := database.DB.Exec(context.Background(),
		`UPDATE lesson_plans SET status = $1, updated_at = now() WHERE id = $2`,
		status, planID,
	)
	if err != nil {
		t.Fatalf("直接更新教案状态失败: %v", err)
	}
}

// getLessonPlanStatusFromDB 从DB直接读取教案状态
func getLessonPlanStatusFromDB(t *testing.T, planID string) string {
	t.Helper()
	var status string
	err := database.DB.QueryRow(context.Background(),
		`SELECT status FROM lesson_plans WHERE id = $1`, planID,
	).Scan(&status)
	if err != nil {
		t.Fatalf("查询教案状态失败: %v", err)
	}
	return status
}

// ==================== 教案CRUD测试 ====================

// TestLessonPlan_Create 创建教案 → 成功返回教案对象
func TestLessonPlan_Create(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)
	planID := createTestLessonPlan(t, server.URL, token, "数学", "七年级", "一元一次方程")

	// 验证DB中教案记录存在
	status := getLessonPlanStatusFromDB(t, planID)
	if status != "draft" {
		t.Errorf("新建教案状态应为draft，实际为 %s", status)
	}
}

// TestLessonPlan_CreateMissingFields 缺少必填字段 → 400
func TestLessonPlan_CreateMissingFields(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	// 缺少subject
	body := map[string]interface{}{
		"title": "测试教案",
		"grade": "七年级",
		"topic": "测试课题",
	}
	resp, _ := DoPost(t, server.URL+"/api/v1/lesson-plans/plans", body, token)
	AssertHTTPStatus(t, resp, http.StatusBadRequest)
}

// TestLessonPlan_GetDetail 获取教案详情 → 返回完整信息
func TestLessonPlan_GetDetail(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)
	planID := createTestLessonPlan(t, server.URL, token, "语文", "八年级", "岳阳楼记")

	resp, apiResp := DoGet(t, server.URL+"/api/v1/lesson-plans/plans/"+planID, token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	var detail struct {
		ID      string `json:"id"`
		Title   string `json:"title"`
		Subject string `json:"subject"`
		Grade   string `json:"grade"`
		Topic   string `json:"topic"`
		Status  string `json:"status"`
	}
	ParseData(t, apiResp, &detail)

	if detail.ID != planID {
		t.Errorf("教案ID不匹配: 期望 %s, 实际 %s", planID, detail.ID)
	}
	if detail.Subject != "语文" {
		t.Errorf("学科不匹配: 期望 语文, 实际 %s", detail.Subject)
	}
	if detail.Status != "draft" {
		t.Errorf("状态不匹配: 期望 draft, 实际 %s", detail.Status)
	}
}

// TestLessonPlan_GetNotFound 获取不存在的教案 → 404
func TestLessonPlan_GetNotFound(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	resp, _ := DoGet(t, server.URL+"/api/v1/lesson-plans/plans/00000000-0000-0000-0000-999999999999", token)
	AssertHTTPStatus(t, resp, http.StatusNotFound)
}

// TestLessonPlan_Update 更新教案内容 → 成功
func TestLessonPlan_Update(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)
	planID := createTestLessonPlan(t, server.URL, token, "英语", "九年级", "Unit 1 Reading")

	updateBody := map[string]interface{}{
		"title":            "Updated Title",
		"content_markdown": "# 更新后的教案内容\n\n这是更新后的教案。",
		"duration_minutes": 40,
	}

	resp, apiResp := DoPut(t, server.URL+"/api/v1/lesson-plans/plans/"+planID, updateBody, token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	// 验证DB中内容已更新
	var title string
	err := database.DB.QueryRow(context.Background(),
		`SELECT title FROM lesson_plans WHERE id = $1`, planID,
	).Scan(&title)
	if err != nil {
		t.Fatalf("查询教案失败: %v", err)
	}
	if title != "Updated Title" {
		t.Errorf("标题未更新: 期望 Updated Title, 实际 %s", title)
	}
}

// TestLessonPlan_UpdateByNonAuthor 非作者更新教案 → 403
func TestLessonPlan_UpdateByNonAuthor(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	// admin创建教案
	adminToken := LoginAsAdmin(t, server.URL)
	planID := createTestLessonPlan(t, server.URL, adminToken, "物理", "八年级", "力学基础")

	// operator尝试更新
	operatorToken := LoginAsOperator(t, server.URL)
	updateBody := map[string]interface{}{
		"title": "Hacked Title",
	}

	resp, _ := DoPut(t, server.URL+"/api/v1/lesson-plans/plans/"+planID, updateBody, operatorToken)
	AssertHTTPStatus(t, resp, http.StatusForbidden)
}

// TestLessonPlan_Delete 删除教案 → 成功
func TestLessonPlan_Delete(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)
	planID := createTestLessonPlan(t, server.URL, token, "化学", "九年级", "化学反应")

	// 删除
	resp, apiResp := DoDelete(t, server.URL+"/api/v1/lesson-plans/plans/"+planID, token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	// 确认DB中已删除
	var count int
	err := database.DB.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM lesson_plans WHERE id = $1`, planID,
	).Scan(&count)
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if count != 0 {
		t.Error("删除后教案记录仍存在")
	}
}

// ==================== 教案状态流转测试 ====================

// TestLessonPlan_PublishPersonal draft → published_personal
func TestLessonPlan_PublishPersonal(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)
	planID := createTestLessonPlan(t, server.URL, token, "历史", "七年级", "秦朝统一")

	resp, apiResp := DoPost(t, server.URL+"/api/v1/lesson-plans/plans/"+planID+"/publish-personal", nil, token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	status := getLessonPlanStatusFromDB(t, planID)
	if status != "published_personal" {
		t.Errorf("个人发布后状态应为published_personal，实际为 %s", status)
	}
}

// TestLessonPlan_SubmitForReview draft → submitted（需指定group）
func TestLessonPlan_SubmitForReview(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)
	planID := createTestLessonPlan(t, server.URL, token, "地理", "七年级", "地球运动")

	// 先创建组织和教研组
	schoolID := createTestOrganization(t, "测试学校", "school")
	groupID := createTestTeachingGroup(t, schoolID, "地理教研组", "地理")

	submitBody := map[string]interface{}{
		"group_id": groupID,
	}
	resp, apiResp := DoPost(t, server.URL+"/api/v1/lesson-plans/plans/"+planID+"/submit-review", submitBody, token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	status := getLessonPlanStatusFromDB(t, planID)
	if status != "submitted" {
		t.Errorf("提交评审后状态应为submitted，实际为 %s", status)
	}
}

// TestLessonPlan_ReviewApproved submitted → approved
func TestLessonPlan_ReviewApproved(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)
	planID := createTestLessonPlan(t, server.URL, token, "美术", "五年级", "色彩构成")

	// 直接设置为submitted状态（绕过需要group的提交流程）
	updateLessonPlanStatusDirectly(t, planID, "submitted")

	// admin评审通过
	reviewBody := map[string]interface{}{
		"decision": "approved",
		"score":    8.5,
		"comments": "教案质量优秀",
	}
	resp, apiResp := DoPost(t, server.URL+"/api/v1/lesson-plans/plans/"+planID+"/review", reviewBody, token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	status := getLessonPlanStatusFromDB(t, planID)
	if status != "approved" {
		t.Errorf("评审通过后状态应为approved，实际为 %s", status)
	}

	// 验证评审记录存在
	var reviewCount int
	err := database.DB.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM lesson_plan_reviews WHERE lesson_plan_id = $1`, planID,
	).Scan(&reviewCount)
	if err != nil {
		t.Fatalf("查询评审记录失败: %v", err)
	}
	if reviewCount != 1 {
		t.Errorf("评审记录数应为1，实际为 %d", reviewCount)
	}
}

// TestLessonPlan_ReviewRevision submitted → revision（退回修改）
func TestLessonPlan_ReviewRevision(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)
	planID := createTestLessonPlan(t, server.URL, token, "音乐", "三年级", "节奏练习")

	updateLessonPlanStatusDirectly(t, planID, "submitted")

	reviewBody := map[string]interface{}{
		"decision": "revision",
		"score":    6.0,
		"comments": "需要补充教学活动设计",
	}
	resp, apiResp := DoPost(t, server.URL+"/api/v1/lesson-plans/plans/"+planID+"/review", reviewBody, token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	status := getLessonPlanStatusFromDB(t, planID)
	if status != "revision" {
		t.Errorf("退回后状态应为revision，实际为 %s", status)
	}
}

// TestLessonPlan_PublishShared approved → published_shared
func TestLessonPlan_PublishShared(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)
	planID := createTestLessonPlan(t, server.URL, token, "体育", "四年级", "篮球基础")

	updateLessonPlanStatusDirectly(t, planID, "approved")

	resp, apiResp := DoPost(t, server.URL+"/api/v1/lesson-plans/plans/"+planID+"/publish-shared", nil, token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	status := getLessonPlanStatusFromDB(t, planID)
	if status != "published_shared" {
		t.Errorf("共享发布后状态应为published_shared，实际为 %s", status)
	}
}

// TestLessonPlan_RevisionToPublishPersonal revision → published_personal
func TestLessonPlan_RevisionToPublishPersonal(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)
	planID := createTestLessonPlan(t, server.URL, token, "道德与法治", "六年级", "公民意识")

	updateLessonPlanStatusDirectly(t, planID, "revision")

	resp, apiResp := DoPost(t, server.URL+"/api/v1/lesson-plans/plans/"+planID+"/publish-personal", nil, token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	status := getLessonPlanStatusFromDB(t, planID)
	if status != "published_personal" {
		t.Errorf("退回后个人发布状态应为published_personal，实际为 %s", status)
	}
}

// TestLessonPlan_InvalidStateTransition 非法状态转换 → 失败
func TestLessonPlan_InvalidStateTransition(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)
	planID := createTestLessonPlan(t, server.URL, token, "科学", "五年级", "光合作用")

	// draft状态不能直接共享发布（需要先approved）
	resp, _ := DoPost(t, server.URL+"/api/v1/lesson-plans/plans/"+planID+"/publish-shared", nil, token)
	if resp.StatusCode == http.StatusOK {
		t.Error("draft状态不应该能直接共享发布")
	}
}

// TestLessonPlan_NonAuthorPublish 非作者操作 → 403
func TestLessonPlan_NonAuthorPublish(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	// admin创建教案
	adminToken := LoginAsAdmin(t, server.URL)
	planID := createTestLessonPlan(t, server.URL, adminToken, "AI", "七年级", "机器学习入门")

	// operator尝试个人发布
	operatorToken := LoginAsOperator(t, server.URL)
	resp, _ := DoPost(t, server.URL+"/api/v1/lesson-plans/plans/"+planID+"/publish-personal", nil, operatorToken)
	AssertHTTPStatus(t, resp, http.StatusForbidden)
}

// ==================== Fork测试 ====================

// TestLessonPlan_ForkPublished Fork已发布教案 → 成功+fork_count递增
func TestLessonPlan_ForkPublished(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	// admin创建并设为approved（可被fork的状态）
	adminToken := LoginAsAdmin(t, server.URL)
	planID := createTestLessonPlan(t, server.URL, adminToken, "数学", "八年级", "二次函数")
	updateLessonPlanStatusDirectly(t, planID, "approved")

	// operator fork该教案
	operatorToken := LoginAsOperator(t, server.URL)
	resp, apiResp := DoPost(t, server.URL+"/api/v1/lesson-plans/plans/"+planID+"/fork", nil, operatorToken)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	// 验证新教案已创建
	var newPlan struct {
		ID     string  `json:"id"`
		Status string  `json:"status"`
		ForkedFrom *string `json:"forked_from"`
	}
	ParseData(t, apiResp, &newPlan)

	if newPlan.ID == "" {
		t.Error("Fork成功但新教案ID为空")
	}
	if newPlan.Status != "draft" {
		t.Errorf("Fork的教案状态应为draft，实际为 %s", newPlan.Status)
	}

	// 验证原教案fork_count递增
	var forkCount int
	err := database.DB.QueryRow(context.Background(),
		`SELECT fork_count FROM lesson_plans WHERE id = $1`, planID,
	).Scan(&forkCount)
	if err != nil {
		t.Fatalf("查询fork_count失败: %v", err)
	}
	if forkCount != 1 {
		t.Errorf("fork_count应为1，实际为 %d", forkCount)
	}
}

// TestLessonPlan_ForkDraft Fork草稿教案 → 失败
func TestLessonPlan_ForkDraft(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	// admin创建草稿教案
	adminToken := LoginAsAdmin(t, server.URL)
	planID := createTestLessonPlan(t, server.URL, adminToken, "语文", "七年级", "背影")

	// operator尝试fork草稿
	operatorToken := LoginAsOperator(t, server.URL)
	resp, _ := DoPost(t, server.URL+"/api/v1/lesson-plans/plans/"+planID+"/fork", nil, operatorToken)
	if resp.StatusCode == http.StatusOK {
		t.Error("草稿教案不应该能被fork")
	}
}

// ==================== 列表与筛选测试 ====================

// TestLessonPlan_ListByAuthor 按作者筛选教案列表
func TestLessonPlan_ListByAuthor(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	adminToken := LoginAsAdmin(t, server.URL)
	operatorToken := LoginAsOperator(t, server.URL)

	// admin创建2个教案
	createTestLessonPlan(t, server.URL, adminToken, "数学", "七年级", "方程1")
	createTestLessonPlan(t, server.URL, adminToken, "数学", "七年级", "方程2")

	// operator创建1个教案
	createTestLessonPlan(t, server.URL, operatorToken, "语文", "八年级", "古诗1")

	// 按admin筛选
	resp, apiResp := DoGet(t, server.URL+"/api/v1/lesson-plans/plans?author_id="+SeedAdminID, adminToken)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	var listData struct {
		LessonPlans []struct {
			ID string `json:"id"`
		} `json:"lesson_plans"`
		Total int `json:"total"`
	}
	ParseData(t, apiResp, &listData)

	if listData.Total != 2 {
		t.Errorf("admin的教案数应为2，实际为 %d", listData.Total)
	}
}

// TestLessonPlan_ListByStatus 按状态筛选教案列表
func TestLessonPlan_ListByStatus(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	// 创建3个教案，1个发布
	plan1 := createTestLessonPlan(t, server.URL, token, "数学", "七年级", "几何1")
	createTestLessonPlan(t, server.URL, token, "数学", "七年级", "几何2")
	createTestLessonPlan(t, server.URL, token, "数学", "七年级", "几何3")

	// 第一个设为published_personal
	DoPost(t, server.URL+"/api/v1/lesson-plans/plans/"+plan1+"/publish-personal", nil, token)

	// 按draft状态筛选
	resp, apiResp := DoGet(t, server.URL+"/api/v1/lesson-plans/plans?status=draft&author_id="+SeedAdminID, token)
	AssertHTTPStatus(t, resp, http.StatusOK)

	var listData struct {
		Total int `json:"total"`
	}
	ParseData(t, apiResp, &listData)

	if listData.Total != 2 {
		t.Errorf("draft状态教案数应为2，实际为 %d", listData.Total)
	}
}

// TestLessonPlan_ListPagination 教案列表分页
func TestLessonPlan_ListPagination(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	// 创建5个教案
	for i := 1; i <= 5; i++ {
		createTestLessonPlan(t, server.URL, token, "数学", "七年级", fmt.Sprintf("分页测试课题%d", i))
	}

	// 请求前2个
	resp, apiResp := DoGet(t, server.URL+"/api/v1/lesson-plans/plans?author_id="+SeedAdminID+"&limit=2&offset=0", token)
	AssertHTTPStatus(t, resp, http.StatusOK)

	var page1 struct {
		LessonPlans []struct {
			ID string `json:"id"`
		} `json:"lesson_plans"`
		Total int `json:"total"`
	}
	ParseData(t, apiResp, &page1)

	if page1.Total != 5 {
		t.Errorf("总数应为5，实际为 %d", page1.Total)
	}
	if len(page1.LessonPlans) != 2 {
		t.Errorf("第一页应返回2条，实际返回 %d 条", len(page1.LessonPlans))
	}

	// 请求第2页
	resp2, apiResp2 := DoGet(t, server.URL+"/api/v1/lesson-plans/plans?author_id="+SeedAdminID+"&limit=2&offset=2", token)
	AssertHTTPStatus(t, resp2, http.StatusOK)

	var page2 struct {
		LessonPlans []struct {
			ID string `json:"id"`
		} `json:"lesson_plans"`
	}
	ParseData(t, apiResp2, &page2)

	if len(page2.LessonPlans) != 2 {
		t.Errorf("第二页应返回2条，实际返回 %d 条", len(page2.LessonPlans))
	}
}
