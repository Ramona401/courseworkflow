package services

// permission_matrix_test.go — 权限矩阵 CanAccess 单元测试（迭代一 Phase 5）
//
// 测试目标：把 permission_matrix.go 那张权威矩阵的"契约"锁死，防止后续误改。
//   - 逐角色验证矩阵注释表里"被授予"的组合返回 true；
//   - 验证 fail-closed：未登记组合、空参数一律返回 false；
//   - 重点覆盖本迭代新增资源（ResourceOrgAdmin 任命、ResourceBatchUser 批量建号）的边界，
//     这些正是 Phase 5 两个新端点（多管理员任命 / 批量建用户）的门禁依据。
//
// 纯单元：CanAccess 零 I/O，本测试不连数据库、不起 server，进 RUN_TESTS 第一步（非 integration）。

import (
        "testing"

        "tedna/internal/models"
)

// TestCanAccess_EmptyParams 空参数一律拒绝（fail-closed 边界）
func TestCanAccess_EmptyParams(t *testing.T) {
        cases := []struct {
                role, resource, action string
        }{
                {"", ResourceUser, ActionView},
                {models.RoleAdmin, "", ActionView},
                {models.RoleAdmin, ResourceUser, ""},
                {"", "", ""},
        }
        for _, c := range cases {
                if CanAccess(c.role, c.resource, c.action) {
                        t.Errorf("空参数应拒绝: role=%q resource=%q action=%q 返回了 true", c.role, c.resource, c.action)
                }
        }
}

// TestCanAccess_AdminAllGranted admin 对所有资源所有操作都放行
func TestCanAccess_AdminAllGranted(t *testing.T) {
        allResources := []string{
                ResourceOrgRegion, ResourceOrgSchool, ResourceUser, ResourceOrgAdmin,
                ResourceLessonPlan, ResourceSystemRole, ResourceToken, ResourceBatchUser,
        }
        allActions := []string{ActionView, ActionCreate, ActionEdit, ActionDelete, ActionAssign}
        for _, r := range allResources {
                for _, a := range allActions {
                        if !CanAccess(models.RoleAdmin, r, a) {
                                t.Errorf("admin 应放行所有组合，但 resource=%s action=%s 被拒绝", r, a)
                        }
                }
        }
}

// TestCanAccess_RegionAdmin region_admin 的授予与拒绝边界
func TestCanAccess_RegionAdmin(t *testing.T) {
        // 应放行的组合
        granted := []struct{ resource, action string }{
                {ResourceOrgRegion, ActionView},
                {ResourceOrgRegion, ActionEdit},
                {ResourceOrgSchool, ActionView},
                {ResourceOrgSchool, ActionCreate},
                {ResourceOrgSchool, ActionEdit},
                {ResourceOrgAdmin, ActionView},   // 可查看组织管理员
                {ResourceOrgAdmin, ActionAssign},  // 可任命本区域校管（Phase 5 多管理员端点门禁）
                {ResourceLessonPlan, ActionView},
                {ResourceToken, ActionView},
                {ResourceToken, ActionAssign},
        }
        for _, g := range granted {
                if !CanAccess(models.RoleRegionAdmin, g.resource, g.action) {
                        t.Errorf("region_admin 应放行 resource=%s action=%s，实际被拒绝", g.resource, g.action)
                }
        }

        // 应拒绝的组合（fail-closed）
        denied := []struct{ resource, action string }{
                {ResourceOrgRegion, ActionDelete},  // 删除区域是 admin 专属
                {ResourceOrgRegion, ActionCreate},  // 不能建区域
                {ResourceUser, ActionCreate},       // 不直接建老师（经校管）
                {ResourceSystemRole, ActionEdit},   // 不能改系统角色
                {ResourceBatchUser, ActionCreate},  // 不能批量建号
                {ResourceOrgSchool, ActionDelete},  // 不能删学校
        }
        for _, d := range denied {
                if CanAccess(models.RoleRegionAdmin, d.resource, d.action) {
                        t.Errorf("region_admin 应拒绝 resource=%s action=%s，实际放行了", d.resource, d.action)
                }
        }
}

// TestCanAccess_SeniorOperator senior_operator（学校管理员）的授予与拒绝边界
func TestCanAccess_SeniorOperator(t *testing.T) {
        granted := []struct{ resource, action string }{
                {ResourceOrgSchool, ActionView},
                {ResourceOrgSchool, ActionEdit},
                {ResourceUser, ActionView},
                {ResourceUser, ActionCreate},      // 本校建老师
                {ResourceUser, ActionEdit},
                {ResourceLessonPlan, ActionView},
                {ResourceToken, ActionView},
                {ResourceToken, ActionAssign},
                {ResourceBatchUser, ActionCreate},  // 本校批量建 operator/viewer（Phase 5 批量端点门禁）
        }
        for _, g := range granted {
                if !CanAccess(models.RoleSeniorOperator, g.resource, g.action) {
                        t.Errorf("senior_operator 应放行 resource=%s action=%s，实际被拒绝", g.resource, g.action)
                }
        }

        denied := []struct{ resource, action string }{
                {ResourceOrgAdmin, ActionAssign},   // 关键：学校管理员不能任命管理员（防权限自举）
                {ResourceOrgAdmin, ActionView},
                {ResourceSystemRole, ActionEdit},   // 不能改系统角色
                {ResourceUser, ActionDelete},       // 删除走禁用，不在此资源
                {ResourceOrgRegion, ActionView},    // 不能看区域组织
                {ResourceOrgSchool, ActionDelete},  // 不能删学校
        }
        for _, d := range denied {
                if CanAccess(models.RoleSeniorOperator, d.resource, d.action) {
                        t.Errorf("senior_operator 应拒绝 resource=%s action=%s，实际放行了", d.resource, d.action)
                }
        }
}

// TestCanAccess_OperatorViewer 普通教师只有 view 类权限
func TestCanAccess_OperatorViewer(t *testing.T) {
        for _, role := range []string{models.RoleOperator, models.RoleViewer} {
                // 应放行：教案/用户/积分的 view
                granted := []struct{ resource, action string }{
                        {ResourceLessonPlan, ActionView},
                        {ResourceUser, ActionView},
                        {ResourceToken, ActionView},
                }
                for _, g := range granted {
                        if !CanAccess(role, g.resource, g.action) {
                                t.Errorf("%s 应放行 resource=%s action=%s，实际被拒绝", role, g.resource, g.action)
                        }
                }

                // 应拒绝：任何写操作、任何管理类资源
                denied := []struct{ resource, action string }{
                        {ResourceLessonPlan, ActionCreate},
                        {ResourceLessonPlan, ActionDelete},
                        {ResourceUser, ActionCreate},
                        {ResourceOrgAdmin, ActionAssign},
                        {ResourceBatchUser, ActionCreate},
                        {ResourceOrgSchool, ActionView},
                        {ResourceSystemRole, ActionEdit},
                        {ResourceToken, ActionAssign},
                }
                for _, d := range denied {
                        if CanAccess(role, d.resource, d.action) {
                                t.Errorf("%s 应拒绝 resource=%s action=%s，实际放行了", role, d.resource, d.action)
                        }
                }
        }
}

// TestCanAccess_OrgAdminAssignMatrix 组织管理员任命资源的角色矩阵（Phase 5 多管理员端点门禁核心）
//   只有 admin（任何）与 region_admin（assign/view）被授予；其余角色全拒绝。
func TestCanAccess_OrgAdminAssign(t *testing.T) {
        // 放行
        if !CanAccess(models.RoleAdmin, ResourceOrgAdmin, ActionAssign) {
                t.Error("admin 应能任命组织管理员")
        }
        if !CanAccess(models.RoleRegionAdmin, ResourceOrgAdmin, ActionAssign) {
                t.Error("region_admin 应能任命（本区域校管）")
        }
        // 拒绝：senior_operator / operator / viewer / district_inspector 都不能任命
        for _, role := range []string{
                models.RoleSeniorOperator, models.RoleOperator,
                models.RoleViewer, models.RoleDistrictInspector,
        } {
                if CanAccess(role, ResourceOrgAdmin, ActionAssign) {
                        t.Errorf("%s 不应能任命组织管理员（防权限自举），实际放行了", role)
                }
        }
}

// TestCanAccess_UnknownRoleResource 完全未知的角色/资源 → fail-closed 拒绝
func TestCanAccess_UnknownRoleResource(t *testing.T) {
        if CanAccess("nonexistent_role", ResourceUser, ActionView) {
                t.Error("未知角色应拒绝")
        }
        if CanAccess(models.RoleAdmin, "nonexistent_resource", ActionView) {
                t.Error("未知资源应拒绝")
        }
        if CanAccess(models.RoleAdmin, ResourceUser, "nonexistent_action") {
                t.Error("未知操作应拒绝")
        }
}
