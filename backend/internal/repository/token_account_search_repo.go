package repository

// token_account_search_repo.go — 积分账户列表用户名搜索
//
// 本文件独立提供管理页账户搜索，避免继续扩大token_account_repo.go。
//
// 修复目标：
//   1. 个人积分账户可按账户显示名或users.username搜索；
//   2. 个人账户列表名称显示为“显示名 (@登录用户名)”；
//   3. 区域和学校账户显示名称保持不变；
//   4. 完整保留TokenScope的owner_id白名单；
//   5. nil白名单表示管理员不过滤，空切片表示fail-closed空集。

import (
	"context"
	"fmt"
	"strings"

	"tedna/internal/database"
	"tedna/internal/models"
)

// ListTokenAccountsSearchable 查询支持登录用户名搜索的积分账户列表。
//
// keyword匹配：
//   - token_accounts.display_name；
//   - 个人账户对应users.username；
//   - token_accounts.owner_id。
//
// 个人账户的DisplayName返回“显示名 (@登录用户名)”。
// 本函数只改变响应展示文本，不修改数据库中的原始display_name。
func ListTokenAccountsSearchable(
	ctx context.Context,
	accountType string,
	parentAccountID string,
	status string,
	keyword string,
	ownerIDs []string,
	limit int,
	offset int,
) ([]*models.TokenAccountListItem, int, error) {
	where := "1=1"
	args := make([]interface{}, 0, 8)
	argIndex := 1

	accountType = strings.TrimSpace(accountType)
	parentAccountID = strings.TrimSpace(parentAccountID)
	status = strings.TrimSpace(status)
	keyword = strings.TrimSpace(keyword)

	if accountType != "" {
		where += fmt.Sprintf(
			" AND ta.account_type = $%d",
			argIndex,
		)
		args = append(args, accountType)
		argIndex++
	}

	if parentAccountID != "" {
		where += fmt.Sprintf(
			" AND ta.parent_account_id = $%d",
			argIndex,
		)
		args = append(args, parentAccountID)
		argIndex++
	}

	if status != "" {
		where += fmt.Sprintf(
			" AND ta.status = $%d",
			argIndex,
		)
		args = append(args, status)
		argIndex++
	}

	if keyword != "" {
		where += fmt.Sprintf(
			` AND (
				ta.display_name ILIKE $%d
				OR COALESCE(account_user.username, '') ILIKE $%d
				OR ta.owner_id::text ILIKE $%d
			)`,
			argIndex,
			argIndex,
			argIndex,
		)
		args = append(
			args,
			"%"+keyword+"%",
		)
		argIndex++
	}

	if ownerIDs != nil {
		if len(ownerIDs) == 0 {
			where += " AND 1=0"
		} else {
			where += fmt.Sprintf(
				" AND ta.owner_id = ANY($%d)",
				argIndex,
			)
			args = append(args, ownerIDs)
			argIndex++

			// 非管理员范围查询不展示上级region账户。
			where += " AND ta.account_type <> 'region'"
		}
	}

	var total int

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM token_accounts ta
		LEFT JOIN users account_user
		  ON ta.account_type = 'personal'
		 AND account_user.id = ta.owner_id
		WHERE %s
	`, where)

	if err := database.DB.QueryRow(
		ctx,
		countQuery,
		args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf(
			"统计可搜索积分账户数失败: %w",
			err,
		)
	}

	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	listQuery := fmt.Sprintf(`
		SELECT
			ta.id,
			ta.account_type,
			ta.owner_id,
			CASE
				WHEN ta.account_type = 'personal'
				 AND COALESCE(account_user.username, '') <> ''
				THEN CONCAT(
					ta.display_name,
					' (@',
					account_user.username,
					')'
				)
				ELSE ta.display_name
			END AS effective_display_name,
			ta.balance,
			ta.frozen_amount,
			ta.total_consumed,
			ta.total_quota,
			ta.monthly_quota,
			ta.status,
			ta.expires_at,
			ta.created_at,
			(
				SELECT COUNT(*)
				FROM token_accounts child_account
				WHERE child_account.parent_account_id = ta.id
			) AS child_count
		FROM token_accounts ta
		LEFT JOIN users account_user
		  ON ta.account_type = 'personal'
		 AND account_user.id = ta.owner_id
		WHERE %s
		ORDER BY
			ta.account_type ASC,
			ta.display_name ASC,
			ta.id ASC
		LIMIT $%d
		OFFSET $%d
	`,
		where,
		argIndex,
		argIndex+1,
	)

	listArgs := append(
		args,
		limit,
		offset,
	)

	rows, err := database.DB.Query(
		ctx,
		listQuery,
		listArgs...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf(
			"查询可搜索积分账户列表失败: %w",
			err,
		)
	}
	defer rows.Close()

	items := make(
		[]*models.TokenAccountListItem,
		0,
	)

	for rows.Next() {
		item := &models.TokenAccountListItem{}

		if err := rows.Scan(
			&item.ID,
			&item.AccountType,
			&item.OwnerID,
			&item.DisplayName,
			&item.Balance,
			&item.FrozenAmount,
			&item.TotalConsumed,
			&item.TotalQuota,
			&item.MonthlyQuota,
			&item.Status,
			&item.ExpiresAt,
			&item.CreatedAt,
			&item.ChildCount,
		); err != nil {
			return nil, 0, fmt.Errorf(
				"扫描可搜索积分账户行失败: %w",
				err,
			)
		}

		item.AccountTypeName = models.AccountTypeNameMap[item.AccountType]
		item.StatusName = models.AccountStatusNameMap[item.Status]
		item.AvailableBalance = item.Balance - item.FrozenAmount

		if item.TotalQuota > 0 {
			item.UsagePercent =
				item.TotalConsumed *
					100.0 /
					item.TotalQuota
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf(
			"遍历可搜索积分账户列表失败: %w",
			err,
		)
	}

	return items, total, nil
}
