package integration

// security_scope_test.go — 组织管理员端点与组织数据范围安全测试
//
// 从security_test.go拆出组织权限相关用例，保持单文件低于600行。
// 本文件只包含：
//   - 多管理员端点越权防护；
//   - senior/operator组织与教研组列表数据范围隔离。

import (
	"context"
	"net/http"
	"testing"

	"tedna/internal/database"
)

// ==================== 迭代一 Phase 5 追加：多管理员端点越权防护 ====================

// TestSecurity_OperatorCannotAppointAdmin operator 任命组织管理员 → 被拒绝（非 admin/region_admin）
func TestSecurity_OperatorCannotAppointAdmin(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	adminToken := LoginAsAdmin(t, server.URL)
	regionID := createRegionViaAPI(t, server.URL, adminToken, "越权测试区A")
	schoolID := createSchoolViaAPI(t, server.URL, adminToken, "越权测试校A", regionID)

	opToken := LoginAsOperator(t, server.URL)
	adminsURL := server.URL + "/api/v1/lesson-plans/organizations/" + schoolID + "/admins"
	resp, _ := DoPost(t, adminsURL, map[string]interface{}{"user_id": SeedViewerID, "role_type": "school_admin"}, opToken)
	if resp.StatusCode == http.StatusOK {
		t.Error("operator 不应能任命组织管理员")
	}
}

// TestSecurity_SeniorCannotAppointAdmin senior_operator 任命组织管理员 → 被拒绝（防权限自举）
func TestSecurity_SeniorCannotAppointAdmin(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	adminToken := LoginAsAdmin(t, server.URL)
	regionID := createRegionViaAPI(t, server.URL, adminToken, "越权测试区B")
	schoolID := createSchoolViaAPI(t, server.URL, adminToken, "越权测试校B", regionID)

	seniorToken := LoginAsSenior(t, server.URL)
	adminsURL := server.URL + "/api/v1/lesson-plans/organizations/" + schoolID + "/admins"
	resp, _ := DoPost(t, adminsURL, map[string]interface{}{"user_id": SeedViewerID, "role_type": "school_admin"}, seniorToken)
	if resp.StatusCode == http.StatusOK {
		t.Error("senior_operator 不应能任命组织管理员（防权限自举）")
	}
}

// TestSecurity_ViewerCannotListOrgAdmins viewer 列出组织管理员 → 被拒绝
func TestSecurity_ViewerCannotListOrgAdmins(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	adminToken := LoginAsAdmin(t, server.URL)
	regionID := createRegionViaAPI(t, server.URL, adminToken, "越权测试区C")
	schoolID := createSchoolViaAPI(t, server.URL, adminToken, "越权测试校C", regionID)

	viewerToken := LoginAsViewer(t, server.URL)
	adminsURL := server.URL + "/api/v1/lesson-plans/organizations/" + schoolID + "/admins"
	resp, _ := DoGet(t, adminsURL, viewerToken)
	if resp.StatusCode == http.StatusOK {
		t.Error("viewer 不应能列出组织管理员")
	}
}

// ==================== Phase 6 验收期追加：组织/教研组列表数据范围隔离 ====================
//
// 背景：ListOrganizations / ListTeachingGroups 此前无数据范围收窄，任何能进 /admin 的角色
//   （含 senior_operator）调 GET /lesson-plans/organizations 都能拿到全库所有区域/学校——真越权。
//   修复后接入 services.ResolveDataScope：admin 全量；senior 仅本校+父区域；
//   region_admin 仅辖区；operator/viewer/Blocked 空集。以下用例锁死该隔离，防回归。
//
// 绑校方式：createSchoolViaAPI 建校后，SQL 设 organizations.admin_user_id = senior，
//   使 GetSchoolByAdminUserID（ResolveDataScope 的 senior 分支依据）能反查到本校。

// bindSchoolAdmin 将指定用户设为某学校的管理员（organizations.admin_user_id）
// 用于构造 senior_operator 的本校归属（ResolveDataScope senior 分支按 admin_user_id 反查）
func bindSchoolAdmin(t *testing.T, schoolID string, userID string) {
	t.Helper()
	_, err := database.DB.Exec(context.Background(),
		`UPDATE organizations SET admin_user_id = $1 WHERE id = $2`,
		userID, schoolID,
	)
	if err != nil {
		t.Fatalf("绑定学校管理员失败: %v", err)
	}
}

// TestSecurity_SeniorOrgListScopedToOwnSchool
// 学校管理员调组织列表，只能看到本校 + 本校所属父区域，看不到其他区域/学校（防跨域越权）
func TestSecurity_SeniorOrgListScopedToOwnSchool(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	adminToken := LoginAsAdmin(t, server.URL)

	// 区A + 校A（绑给 senior）
	regionAID := createRegionViaAPI(t, server.URL, adminToken, "隔离测试-区A")
	schoolAID := createSchoolViaAPI(t, server.URL, adminToken, "隔离测试-校A", regionAID)
	bindSchoolAdmin(t, schoolAID, SeedSeniorID)

	// 区B + 校B（与 senior 无关，应不可见）
	regionBID := createRegionViaAPI(t, server.URL, adminToken, "隔离测试-区B")
	schoolBID := createSchoolViaAPI(t, server.URL, adminToken, "隔离测试-校B", regionBID)

	// senior 登录调组织列表
	seniorToken := LoginAsSenior(t, server.URL)
	resp, apiResp := DoGet(t, server.URL+"/api/v1/lesson-plans/organizations", seniorToken)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	var listData struct {
		Organizations []struct {
			ID string `json:"id"`
		} `json:"organizations"`
		Total int `json:"total"`
	}
	ParseData(t, apiResp, &listData)

	// 收集可见组织ID集合
	visible := map[string]bool{}
	for _, o := range listData.Organizations {
		visible[o.ID] = true
	}

	// 应可见：本校A + 父区域A
	if !visible[schoolAID] {
		t.Error("senior 应能看到本校A")
	}
	if !visible[regionAID] {
		t.Error("senior 应能看到本校所属父区域A（只读上级）")
	}
	// 不应可见：区B、校B
	if visible[regionBID] {
		t.Error("越权：senior 不应看到其他区域B")
	}
	if visible[schoolBID] {
		t.Error("越权：senior 不应看到其他学校B")
	}
}

// TestSecurity_SeniorGroupListRejectsOtherSchool
// 学校管理员传他校 school_id 查教研组 → 拿空集；传本校 → 正常（防跨校查教研组）
func TestSecurity_SeniorGroupListRejectsOtherSchool(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	adminToken := LoginAsAdmin(t, server.URL)

	// 校A 绑 senior，校A 下建一个教研组
	regionAID := createRegionViaAPI(t, server.URL, adminToken, "教研组隔离-区A")
	schoolAID := createSchoolViaAPI(t, server.URL, adminToken, "教研组隔离-校A", regionAID)
	bindSchoolAdmin(t, schoolAID, SeedSeniorID)
	createTeachingGroupViaAPI(t, server.URL, adminToken, "校A数学组", schoolAID, "数学")

	// 校B（他校）下也建一个教研组
	regionBID := createRegionViaAPI(t, server.URL, adminToken, "教研组隔离-区B")
	schoolBID := createSchoolViaAPI(t, server.URL, adminToken, "教研组隔离-校B", regionBID)
	createTeachingGroupViaAPI(t, server.URL, adminToken, "校B语文组", schoolBID, "语文")

	seniorToken := LoginAsSenior(t, server.URL)

	// 传他校B的 school_id → 应拿空集（越权拦截）
	respB, apiB := DoGet(t, server.URL+"/api/v1/lesson-plans/teaching-groups?school_id="+schoolBID, seniorToken)
	AssertHTTPStatus(t, respB, http.StatusOK)
	AssertAPICode(t, apiB, 0)
	var groupsB struct {
		Groups []struct {
			ID string `json:"id"`
		} `json:"groups"`
		Total int `json:"total"`
	}
	ParseData(t, apiB, &groupsB)
	if groupsB.Total != 0 || len(groupsB.Groups) != 0 {
		t.Errorf("越权：senior 传他校 school_id 应拿空集，实际 %d 个教研组", groupsB.Total)
	}

	// 传本校A的 school_id → 应正常拿到本校教研组
	respA, apiA := DoGet(t, server.URL+"/api/v1/lesson-plans/teaching-groups?school_id="+schoolAID, seniorToken)
	AssertHTTPStatus(t, respA, http.StatusOK)
	AssertAPICode(t, apiA, 0)
	var groupsA struct {
		Groups []struct {
			ID string `json:"id"`
		} `json:"groups"`
		Total int `json:"total"`
	}
	ParseData(t, apiA, &groupsA)
	if groupsA.Total < 1 {
		t.Error("senior 查本校 school_id 应能拿到本校教研组（不应被自我拦截）")
	}
}

// TestSecurity_OperatorOrgListEmpty
// 普通骨干教师（operator）调组织列表 → 空集（operator scope 为空切片，fail-closed）
func TestSecurity_OperatorOrgListEmpty(t *testing.T) {
	server, _ := SetupTestServer(t)
	CleanAndSeed(t)

	adminToken := LoginAsAdmin(t, server.URL)
	// 建若干组织，确认 operator 一个都看不到
	regionID := createRegionViaAPI(t, server.URL, adminToken, "operator隔离-区")
	createSchoolViaAPI(t, server.URL, adminToken, "operator隔离-校", regionID)

	opToken := LoginAsOperator(t, server.URL)
	resp, apiResp := DoGet(t, server.URL+"/api/v1/lesson-plans/organizations", opToken)
	AssertHTTPStatus(t, resp, http.StatusOK)
	AssertAPICode(t, apiResp, 0)

	var listData struct {
		Organizations []struct {
			ID string `json:"id"`
		} `json:"organizations"`
		Total int `json:"total"`
	}
	ParseData(t, apiResp, &listData)
	if listData.Total != 0 || len(listData.Organizations) != 0 {
		t.Errorf("越权：operator 不应看到任何组织，实际 %d 个", listData.Total)
	}
}

// TODO(Phase 7): 补 inspection 详情越权防回归测试
//   GetInspection 已修（service 层归属校验：非 admin 必须是被分配审查员，否则 403）。
//   测试搭建需造 2 个 district_inspector + 已发布教案 + 抽样 + 分配两条分属不同 inspector 的记录，
//   链路较长且依赖 inspection 种子。留待 Phase 7 给 inspection 补功能测试时复用场景一并写。
