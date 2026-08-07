package repository

// user_token_account_tx_repo.go — 新建教学账号的个人积分账户事务闭环
//
// 本文件只承载“用户建号事务”需要的积分账户写入能力，避免继续扩张
// token_account_repo.go。调用方负责事务生命周期，本文件不自行提交或回滚。
//
// 核心规则：
//   1. 个人账户必须挂到所选学校的 active 学校积分账户；
//   2. 学校必须真实存在、启用且 education_domain 为具体教学域；
//   3. 用户、校籍和个人账户由 UserService 放在同一事务中创建；
//   4. 学校账户缺失时 fail-closed，整笔建号回滚，禁止产生半成品账号。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

var (
	// ErrSchoolTokenAccountNotFound 表示学校组织存在，但积分账户树尚未初始化。
	//
	// Service收到该错误后应返回明确业务提示，并回滚用户与校籍事务。
	ErrSchoolTokenAccountNotFound = errors.New("学校积分账户不存在或不可用")
)

// CreatePersonalTokenAccountForSchoolUserTx 在调用方事务内创建学校教师的个人积分账户。
//
// 返回新个人账户ID。函数执行时会通过 organizations + token_accounts 联表并加
// FOR SHARE 行锁，重新确认学校和父账户的最终状态，避免事务外预校验后的竞态。
//
// 参数：
//   - userID：刚在同一事务创建的用户ID；
//   - displayName：个人账户显示名，空值回退“个人积分账户”；
//   - schoolID：school_members 同事务写入的学校ID；
//   - initialCredits：新用户初始积分，不得为负数。
func CreatePersonalTokenAccountForSchoolUserTx(
	ctx context.Context,
	tx pgx.Tx,
	userID string,
	displayName string,
	schoolID string,
	initialCredits float64,
) (string, error) {
	userID = strings.TrimSpace(userID)
	schoolID = strings.TrimSpace(schoolID)
	displayName = strings.TrimSpace(displayName)

	if userID == "" || schoolID == "" {
		return "", fmt.Errorf("创建个人积分账户失败: userID或schoolID为空")
	}
	if initialCredits < 0 {
		return "", fmt.Errorf("创建个人积分账户失败: 初始积分不能为负数")
	}
	if displayName == "" {
		displayName = "个人积分账户"
	}

	var parentAccountID string

	err := tx.QueryRow(ctx, `
		SELECT ta.id::text
		FROM organizations school
		JOIN token_accounts ta
		  ON ta.account_type = 'school'
		 AND ta.owner_id = school.id
		 AND ta.status = 'active'
		WHERE school.id = $1
		  AND school.type = 'school'
		  AND school.status = 'active'
		  AND school.education_domain IN ('k12', 'vocational', 'adult')
		FOR SHARE OF school, ta
	`, schoolID).Scan(&parentAccountID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrSchoolTokenAccountNotFound
		}
		return "", fmt.Errorf("锁定学校积分账户失败: %w", err)
	}

	var accountID string

	err = tx.QueryRow(ctx, `
		INSERT INTO token_accounts (
			account_type,
			owner_id,
			parent_account_id,
			display_name,
			balance,
			frozen_amount,
			total_consumed,
			total_quota,
			monthly_quota,
			expires_at,
			status
		)
		VALUES (
			'personal',
			$1,
			$2,
			$3,
			$4,
			0,
			0,
			0,
			0,
			NULL,
			'active'
		)
		RETURNING id::text
	`,
		userID,
		parentAccountID,
		displayName,
		initialCredits,
	).Scan(&accountID)
	if err != nil {
		return "", fmt.Errorf("事务内创建个人积分账户失败: %w", err)
	}

	return accountID, nil
}
