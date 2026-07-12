package repository

// school_membership_repo.go — 用户↔学校归属治理·数据访问层（批A新增）
//
// 归属三规则（与 services/organization_membership_service.go 对应）：
//   R1 加组 ⇒ 自动入校（既有：organization_service.AddGroupMember 内实现）
//   R2 退组 ⇒ 只退组，永不碰 school_members（既有行为，保持）
//   R3 退校 ⇒ 强制退出该校全部教研组 + 删除校籍行（本文件的事务实现）
//
// 本文件只提供 R3 的事务原语 RemoveUserFromSchoolTx：
//   在【单个数据库事务】内完成——
//     ① 校验目标组织存在且 type='school'（不存在返回 ErrOrgNotFound）；
//     ② 收集该用户在该校的全部教研组ID（供调用方写审计明细）；
//     ③ 删除这些教研组成员行；
//     ④ 删除 school_members 校籍行；
//   任一步失败整体回滚，绝不出现"组退了校没退"或"校退了组还在"的中间态。
//
// 设计说明：
//   - 组织存在性校验用 COALESCE 子查询（恒返回一行），避免依赖具体 pgx 版本的
//     ErrNoRows 判定，空串即视为学校不存在。
//   - 返回值同时报告"删了几个组 + 校籍行是否真的删了"，二者皆零由 service 层
//     判定为"本就不是该校成员"（ErrMemberNotFound 语义）。

import (
	"context"
	"fmt"

	"tedna/internal/database"
)

// SchoolRemovalTxResult 移出本校事务的执行结果
type SchoolRemovalTxResult struct {
	SchoolName       string   // 学校名称（审计与响应文案用，事务内查出）
	RemovedGroupIDs  []string // 连带退出的教研组ID清单（可能为空）
	SchoolRowRemoved bool     // school_members 校籍行是否被删除（false=本无校籍）
}

// RemoveUserFromSchoolTx 单事务执行"移出本校"（R3）
//
// 入参：targetUserID 被移出的用户，schoolID 目标学校。
// 返回：执行结果 + error。学校不存在返回 ErrOrgNotFound（organization_repo.go 定义）。
// 注意：本函数不做权限校验与审计——权限在 handler 层、审计由 handler 按结果写入。
func RemoveUserFromSchoolTx(ctx context.Context, targetUserID string, schoolID string) (*SchoolRemovalTxResult, error) {
	if targetUserID == "" || schoolID == "" {
		return nil, fmt.Errorf("缺少用户ID或学校ID")
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("开启事务失败: %w", err)
	}
	// 提交成功后再 Rollback 是无害空操作（pgx 返回已关闭错误，忽略即可）
	defer func() { _ = tx.Rollback(ctx) }()

	result := &SchoolRemovalTxResult{RemovedGroupIDs: []string{}}

	// ① 校验学校存在（COALESCE 恒返回一行，空串=不存在，规避 ErrNoRows 版本差异）
	var schoolName string
	if err := tx.QueryRow(ctx,
		`SELECT COALESCE((SELECT name FROM organizations WHERE id = $1 AND type = 'school'), '')`,
		schoolID,
	).Scan(&schoolName); err != nil {
		return nil, fmt.Errorf("查询学校信息失败: %w", err)
	}
	if schoolName == "" {
		return nil, ErrOrgNotFound
	}
	result.SchoolName = schoolName

	// ② 收集该用户在该校的全部教研组ID（审计明细用）
	rows, err := tx.Query(ctx, `
		SELECT tgm.group_id
		FROM teaching_group_members tgm
		JOIN teaching_groups tg ON tg.id = tgm.group_id
		WHERE tgm.user_id = $1 AND tg.school_id = $2
	`, targetUserID, schoolID)
	if err != nil {
		return nil, fmt.Errorf("查询该校教研组归属失败: %w", err)
	}
	for rows.Next() {
		var gid string
		if scanErr := rows.Scan(&gid); scanErr != nil {
			rows.Close()
			return nil, fmt.Errorf("扫描教研组ID失败: %w", scanErr)
		}
		result.RemovedGroupIDs = append(result.RemovedGroupIDs, gid)
	}
	rows.Close()
	if rows.Err() != nil {
		return nil, fmt.Errorf("遍历教研组归属失败: %w", rows.Err())
	}

	// ③ 删除该校全部教研组成员行（R3 连带退组）
	if len(result.RemovedGroupIDs) > 0 {
		if _, err := tx.Exec(ctx, `
			DELETE FROM teaching_group_members
			WHERE user_id = $1
			  AND group_id IN (SELECT id FROM teaching_groups WHERE school_id = $2)
		`, targetUserID, schoolID); err != nil {
			return nil, fmt.Errorf("退出该校教研组失败: %w", err)
		}
	}

	// ④ 删除校籍行（无行受影响=本无校籍，非错误，由 SchoolRowRemoved 报告）
	ct, err := tx.Exec(ctx,
		`DELETE FROM school_members WHERE user_id = $1 AND school_id = $2`,
		targetUserID, schoolID,
	)
	if err != nil {
		return nil, fmt.Errorf("删除学校成员记录失败: %w", err)
	}
	result.SchoolRowRemoved = ct.RowsAffected() > 0

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("提交事务失败: %w", err)
	}
	return result, nil
}
