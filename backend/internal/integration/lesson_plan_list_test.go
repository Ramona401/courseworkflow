package integration

// lesson_plan_list_test.go — 教案列表、筛选与分页集成测试
//
// 本文件从教案生命周期测试中拆出列表类用例，保持单文件职责清晰且低于600行。
// 共覆盖：按作者筛选、按状态筛选、分页。

import (
	"fmt"
	"net/http"
	"testing"
)

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
