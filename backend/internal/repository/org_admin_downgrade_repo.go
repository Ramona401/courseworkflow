package repository

// org_admin_downgrade_repo.go — 末任命自动降级·数据访问层（归属治理批C新增）
//
// 提供 CountUserAdminBindings：统计某用户当前持有的组织管辖绑定总数（双来源）。
// 用途：移除某人的组织管理员任命后，判断其是否已"任命归零"——归零且身份为
// 任命制身份(senior_operator/region_admin)时，由 service 层自动降级为骨干教师，
// 维持不变式「管理身份 ⇔ 存在任命」。
//
// 双来源与管辖解析链(ListRegionIDsByAdmin/GetSchoolByAdminUserID)口径一致：
//   ① organization_admins 表（多管理员任命，权威来源）；
//   ② organizations.admin_user_id 单字段（历史主管理员指针，B2 起与①同等有效）。
// 同一组织两处同时指向同一人时会计 2——本计数仅用于"是否为 0"的判定，
// 重复计数不影响结论，无需去重。

import (
	"context"
	"fmt"

	"tedna/internal/database"
)

// CountUserAdminBindings 统计某用户的组织管辖绑定总数（organization_admins ∪ admin_user_id）
// userID 为空直接返回 0（防御，不发 SQL）。
func CountUserAdminBindings(ctx context.Context, userID string) (int, error) {
	if userID == "" {
		return 0, nil
	}
	var n int
	err := database.DB.QueryRow(ctx, `
		SELECT (SELECT COUNT(*) FROM organization_admins WHERE user_id = $1)
		     + (SELECT COUNT(*) FROM organizations WHERE admin_user_id = $1)
	`, userID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("统计用户管辖绑定失败: %w", err)
	}
	return n, nil
}
