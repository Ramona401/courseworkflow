package integration

// organization_test.go — 组织管理与教研组集成测试
//
// 测试范围（8个用例）：
//
// 组织CRUD（4个）：
//   1. 创建区域组织 → 成功
//   2. 创建学校组织（挂载在区域下） → 成功
//   3. 更新组织名称 → 成功
//   4. 删除组织 → 成功；有子组织时删除失败
//
// 教研组CRUD（2个）：
//   5. 创建教研组（挂载在学校下） → 成功
//   6. 教研组缺少必填字段 → 400
//
// 教研组成员管理（2个）：
//   7. 添加成员+重复添加 → 成功/400
//   8. 移除成员+获取我的教研组

import (
	"context"
	"net/http"
	"testing"

	"tedna/internal/database"
)

// ==================== 辅助函数 ====================

// createRegionViaAPI 通过API创建区域组织，返回组织ID
func createRegionViaAPI(t *testing.T, serverURL string, token string, name string) string {
	t.Helper()
	body := map[string]interface{}{
		"name": name,
		"type": "region",
	}
	resp, apiResp := DoPost(t, serverURL+"/api/v1/lesson-plans/organizations", body, token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	var data struct {
		ID string `json:"id"`
	}
	ParseData(t, apiResp, &data)
	if data.ID == "" {
		t.Fatal("创建区域组织成功但ID为空")
	}
	return data.ID
}

// createSchoolViaAPI 通过API创建学校组织，返回组织ID
func createSchoolViaAPI(t *testing.T, serverURL string, token string, name string, parentID string) string {
	t.Helper()
	body := map[string]interface{}{
		"name":      name,
		"type":      "school",
		"parent_id": parentID,
	}
	resp, apiResp := DoPost(t, serverURL+"/api/v1/lesson-plans/organizations", body, token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	var data struct {
		ID string `json:"id"`
	}
	ParseData(t, apiResp, &data)
	if data.ID == "" {
		t.Fatal("创建学校组织成功但ID为空")
	}
	return data.ID
}

// createTeachingGroupViaAPI 通过API创建教研组，返回教研组ID
func createTeachingGroupViaAPI(t *testing.T, serverURL string, token string, name string, schoolID string, subject string) string {
	t.Helper()
	body := map[string]interface{}{
		"name":      name,
		"school_id": schoolID,
		"subject":   subject,
	}
	resp, apiResp := DoPost(t, serverURL+"/api/v1/lesson-plans/teaching-groups", body, token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	var data struct {
		ID string `json:"id"`
	}
	ParseData(t, apiResp, &data)
	if data.ID == "" {
		t.Fatal("创建教研组成功但ID为空")
	}
	return data.ID
}

// ==================== 组织CRUD测试 ====================

// TestOrg_CreateRegion 创建区域组织 → 成功
func TestOrg_CreateRegion(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)
	regionID := createRegionViaAPI(t, server.URL, token, "海淀区教育局")

	// 验证DB中存在
	var name, orgType string
	err := database.DB.QueryRow(context.Background(),
		`SELECT name, type FROM organizations WHERE id = $1`, regionID,
	).Scan(&name, &orgType)
	if err != nil {
		t.Fatalf("查询组织失败: %v", err)
	}
	if name != "海淀区教育局" {
		t.Errorf("组织名称不匹配: 期望 海淀区教育局, 实际 %s", name)
	}
	if orgType != "region" {
		t.Errorf("组织类型不匹配: 期望 region, 实际 %s", orgType)
	}
}

// TestOrg_CreateSchoolUnderRegion 在区域下创建学校 → 成功
func TestOrg_CreateSchoolUnderRegion(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	// 先创建区域
	regionID := createRegionViaAPI(t, server.URL, token, "朝阳区教育局")

	// 再创建学校（挂载在区域下）
	schoolID := createSchoolViaAPI(t, server.URL, token, "朝阳实验中学", regionID)

	// 验证DB中父级ID正确
	var parentID *string
	err := database.DB.QueryRow(context.Background(),
		`SELECT parent_id FROM organizations WHERE id = $1`, schoolID,
	).Scan(&parentID)
	if err != nil {
		t.Fatalf("查询学校失败: %v", err)
	}
	if parentID == nil || *parentID != regionID {
		t.Error("学校的parent_id应指向区域组织")
	}

	// 验证组织列表可以按type筛选
	resp, apiResp := DoGet(t, server.URL+"/api/v1/lesson-plans/organizations?type=school", token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)
}

// TestOrg_UpdateAndDelete 更新组织名称 + 删除组织（含有子组织删除失败校验）
func TestOrg_UpdateAndDelete(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	// 创建区域
	regionID := createRegionViaAPI(t, server.URL, token, "待更新区域")

	// 更新名称
	updateBody := map[string]interface{}{
		"name": "已更新区域",
	}
	resp, apiResp := DoPut(t, server.URL+"/api/v1/lesson-plans/organizations/"+regionID, updateBody, token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	// 验证DB名称已更新
	var name string
	err := database.DB.QueryRow(context.Background(),
		`SELECT name FROM organizations WHERE id = $1`, regionID,
	).Scan(&name)
	if err != nil {
		t.Fatalf("查询组织失败: %v", err)
	}
	if name != "已更新区域" {
		t.Errorf("组织名称未更新: 期望 已更新区域, 实际 %s", name)
	}

	// 在区域下创建学校（使删除失败）
	createSchoolViaAPI(t, server.URL, token, "子学校", regionID)

	// 尝试删除有子组织的区域 → 应失败(400)
	respDel, _ := DoDelete(t, server.URL+"/api/v1/lesson-plans/organizations/"+regionID, token)
	AssertHTTPStatus(t, respDel, http.StatusBadRequest)

	// 创建一个无子组织的区域并删除 → 应成功
	emptyRegionID := createRegionViaAPI(t, server.URL, token, "待删除空区域")
	respDel2, apiResp2 := DoDelete(t, server.URL+"/api/v1/lesson-plans/organizations/"+emptyRegionID, token)
	AssertHTTPStatus(t, respDel2, http.StatusOK)
	AssertAPICode(t, apiResp2, 0)
}

// TestOrg_NonAdminCannotCreate 非admin用户不能创建组织 → 403
func TestOrg_NonAdminCannotCreate(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsOperator(t, server.URL)

	body := map[string]interface{}{
		"name": "非法组织",
		"type": "region",
	}
	resp, _ := DoPost(t, server.URL+"/api/v1/lesson-plans/organizations", body, token)
	AssertHTTPStatus(t, resp, http.StatusForbidden)
}

// ==================== 教研组CRUD测试 ====================

// TestGroup_Create 创建教研组 → 成功
func TestGroup_Create(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	// 先创建区域和学校
	regionID := createRegionViaAPI(t, server.URL, token, "西城区")
	schoolID := createSchoolViaAPI(t, server.URL, token, "西城实验小学", regionID)

	// 创建教研组
	groupID := createTeachingGroupViaAPI(t, server.URL, token, "数学教研组", schoolID, "数学")

	// 验证DB
	var subject, schoolRef string
	err := database.DB.QueryRow(context.Background(),
		`SELECT subject, school_id FROM teaching_groups WHERE id = $1`, groupID,
	).Scan(&subject, &schoolRef)
	if err != nil {
		t.Fatalf("查询教研组失败: %v", err)
	}
	if subject != "数学" {
		t.Errorf("教研组学科不匹配: 期望 数学, 实际 %s", subject)
	}
	if schoolRef != schoolID {
		t.Error("教研组的school_id不匹配")
	}

	// 获取教研组详情
	respDetail, apiRespDetail := DoGet(t, server.URL+"/api/v1/lesson-plans/teaching-groups/"+groupID, token)
	AssertHTTPStatus(t, respDetail, http.StatusOK)
	AssertAPICode(t, apiRespDetail, 0)
}

// TestGroup_CreateMissingFields 缺少必填字段创建教研组 → 400
func TestGroup_CreateMissingFields(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	// 缺少school_id
	body := map[string]interface{}{
		"name":    "无学校教研组",
		"subject": "数学",
	}
	resp, _ := DoPost(t, server.URL+"/api/v1/lesson-plans/teaching-groups", body, token)
	AssertHTTPStatus(t, resp, http.StatusBadRequest)

	// 缺少subject
	regionID := createRegionViaAPI(t, server.URL, token, "测试区域")
	schoolID := createSchoolViaAPI(t, server.URL, token, "测试学校", regionID)

	body2 := map[string]interface{}{
		"name":      "无学科教研组",
		"school_id": schoolID,
		"subject":   "",
	}
	resp2, _ := DoPost(t, server.URL+"/api/v1/lesson-plans/teaching-groups", body2, token)
	AssertHTTPStatus(t, resp2, http.StatusBadRequest)
}

// ==================== 成员管理测试 ====================

// TestGroupMember_AddAndDuplicate 添加成员+重复添加
func TestGroupMember_AddAndDuplicate(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	token := LoginAsAdmin(t, server.URL)

	// 创建组织结构
	regionID := createRegionViaAPI(t, server.URL, token, "丰台区")
	schoolID := createSchoolViaAPI(t, server.URL, token, "丰台一中", regionID)
	groupID := createTeachingGroupViaAPI(t, server.URL, token, "语文组", schoolID, "语文")

	// 添加operator1为成员
	addBody := map[string]interface{}{
		"user_id": SeedOperatorID,
		"role":    "member",
	}
	resp, apiResp := DoPost(t, server.URL+"/api/v1/lesson-plans/teaching-groups/"+groupID+"/members", addBody, token)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	// 验证DB中成员存在
	var memberCount int
	err := database.DB.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM teaching_group_members WHERE group_id = $1 AND user_id = $2`,
		groupID, SeedOperatorID,
	).Scan(&memberCount)
	if err != nil {
		t.Fatalf("查询成员失败: %v", err)
	}
	if memberCount != 1 {
		t.Errorf("成员应存在1条记录，实际 %d", memberCount)
	}

	// 重复添加同一用户 → 应失败(400)
	resp2, _ := DoPost(t, server.URL+"/api/v1/lesson-plans/teaching-groups/"+groupID+"/members", addBody, token)
	AssertHTTPStatus(t, resp2, http.StatusBadRequest)
}

// TestGroupMember_RemoveAndMyGroups 移除成员 + 获取我的教研组列表
func TestGroupMember_RemoveAndMyGroups(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	adminToken := LoginAsAdmin(t, server.URL)

	// 创建组织结构+教研组
	regionID := createRegionViaAPI(t, server.URL, adminToken, "东城区")
	schoolID := createSchoolViaAPI(t, server.URL, adminToken, "东城二中", regionID)
	groupID := createTeachingGroupViaAPI(t, server.URL, adminToken, "英语组", schoolID, "英语")

	// 添加viewer1为成员
	addBody := map[string]interface{}{
		"user_id": SeedViewerID,
		"role":    "member",
	}
	resp, _ := DoPost(t, server.URL+"/api/v1/lesson-plans/teaching-groups/"+groupID+"/members", addBody, adminToken)
	AssertHTTPStatus(t, resp, http.StatusOK)

	// viewer1查看自己所属教研组
	viewerToken := LoginAsViewer(t, server.URL)
	respGroups, apiRespGroups := DoGet(t, server.URL+"/api/v1/lesson-plans/my-groups", viewerToken)
	AssertHTTPStatus(t, respGroups, http.StatusOK)
	AssertAPICode(t, apiRespGroups, 0)

	var myGroups []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	ParseData(t, apiRespGroups, &myGroups)
	if len(myGroups) < 1 {
		t.Error("viewer1应至少属于1个教研组")
	}

	// admin移除viewer1
	respRemove, apiRespRemove := DoDelete(t, server.URL+"/api/v1/lesson-plans/teaching-groups/"+groupID+"/members/"+SeedViewerID, adminToken)
	AssertHTTPStatus(t, respRemove, http.StatusOK)
	AssertAPICode(t, apiRespRemove, 0)

	// 再次查看 → 应为空
	respGroups2, apiRespGroups2 := DoGet(t, server.URL+"/api/v1/lesson-plans/my-groups", viewerToken)
	AssertHTTPStatus(t, respGroups2, http.StatusOK)
	AssertAPICode(t, apiRespGroups2, 0)

	var myGroups2 []struct {
		ID string `json:"id"`
	}
	ParseData(t, apiRespGroups2, &myGroups2)
	if len(myGroups2) != 0 {
		t.Errorf("移除后viewer1不应属于任何教研组，实际 %d 个", len(myGroups2))
	}
}
