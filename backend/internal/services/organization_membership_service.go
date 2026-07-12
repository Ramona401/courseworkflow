package services

// organization_membership_service.go — 用户↔学校归属治理·业务层（批A新增）
//
// 归属三规则（本系统"用户属于哪所学校"的唯一事实源是 school_members）：
//   R1 加组 ⇒ 自动入校（既有：AddGroupMember 内 source=group_member 自动写校籍）
//   R2 退组 ⇒ 只退组，永不碰校籍（既有行为，保持——简单可预测，无隐式副作用）
//   R3 退校 ⇒ 强制退出该校全部教研组（本文件实现）——否则教研组兜底链路会把
//      用户"算回"本校，退校等于没退（lichao01 事件的机制修复）
//   区域归属为派生：用户属于哪个区域 = 其学校的父区域，无独立"退区域"操作。
//
// 与管辖轴（organization_admins / organizations.admin_user_id）完全解耦：
//   移出本校不动任何管理员任命；撤销管理员也不动校籍（管理员≠成员）。
//
// 方法挂在 OrganizationService 上（接收者定义于 organization_service.go，
// 与 organization_admin_service.go 同一"分文件补充方法"模式）。

import (
	"context"
	"errors"
	"fmt"

	"tedna/internal/repository"
)

// SchoolMemberRemovalResult 移出本校的业务结果（供 handler 组装审计与响应文案）
type SchoolMemberRemovalResult struct {
	SchoolName       string   // 学校名
	RemovedGroupIDs  []string // 连带退出的教研组ID清单
	SchoolRowRemoved bool     // 校籍行是否被删除
}

// RemoveUserFromSchool 将用户移出某学校（R3：单事务连带退出该校全部教研组）
//
// 错误约定：
//   - 学校不存在              → ErrOrgNotFound
//   - 目标用户不存在          → 普通 error（handler 转 400）
//   - 既无校籍也不在该校任何组 → ErrMemberNotFound（"本就不是该校成员"）
// 权限校验在 handler 层完成（admin 任意校 / senior 仅本校且目标须教师级），
// 审计由 handler 按返回的明细写入 audit_logs。
func (s *OrganizationService) RemoveUserFromSchool(ctx context.Context, schoolID string, targetUserID string) (*SchoolMemberRemovalResult, error) {
	if schoolID == "" || targetUserID == "" {
		return nil, fmt.Errorf("缺少学校ID或用户ID")
	}

	// 目标用户必须存在（防对已删除/伪造ID操作产生迷惑性"成功"）
	if _, uErr := repository.FindUserByID(ctx, targetUserID); uErr != nil {
		return nil, fmt.Errorf("目标用户不存在")
	}

	// 单事务执行：校验学校 → 收集该校组归属 → 退组 → 删校籍
	txRes, err := repository.RemoveUserFromSchoolTx(ctx, targetUserID, schoolID)
	if err != nil {
		if errors.Is(err, repository.ErrOrgNotFound) {
			return nil, ErrOrgNotFound
		}
		orgLog.Error("移出本校失败", "school_id", schoolID, "target", targetUserID, "error", err)
		return nil, err
	}

	// 两处都没删到 = 本就不是该校成员（既无校籍也不在该校任何教研组）
	if !txRes.SchoolRowRemoved && len(txRes.RemovedGroupIDs) == 0 {
		return nil, ErrMemberNotFound
	}

	orgLog.Info("移出本校成功",
		"school_id", schoolID, "school_name", txRes.SchoolName, "target", targetUserID,
		"removed_group_count", len(txRes.RemovedGroupIDs), "school_row_removed", txRes.SchoolRowRemoved)

	return &SchoolMemberRemovalResult{
		SchoolName:       txRes.SchoolName,
		RemovedGroupIDs:  txRes.RemovedGroupIDs,
		SchoolRowRemoved: txRes.SchoolRowRemoved,
	}, nil
}
