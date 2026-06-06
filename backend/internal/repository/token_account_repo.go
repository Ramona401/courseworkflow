package repository

// token_account_repo.go — Token积分系统数据访问层（账户+采购+预警）
//
// v128 新增（阶段C · Token/积分系统）
// v172 新增（积分管理三级数据权限隔离）：账户/概览 owner_id 白名单
// v172.1 修复（跨级泄漏）：scoped 非admin 排除 region 账户
// v172.2 新增（采购记录隔离）：ListTokenPurchases 增加 ownerIDs 白名单
//   - 采购记录无 owner 维度，需 JOIN token_accounts 取 owner_id 过滤
//   - 白名单语义同其它查询：nil=不过滤(admin)；空切片=匹配空集(fail-closed)；非空=ANY 且排除 region
//
// 对应数据库表：token_accounts / token_purchases / token_alert_configs

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"tedna/internal/database"
	"tedna/internal/models"
)

// ==================== 错误常量 ====================

var (
	ErrTokenAccountNotFound  = errors.New("积分账户不存在")
	ErrInsufficientBalance   = errors.New("积分余额不足")
	ErrAccountSuspended      = errors.New("积分账户已冻结")
	ErrDuplicateAccount      = errors.New("该实体已存在同类型账户")
	ErrTokenPurchaseNotFound = errors.New("采购记录不存在")
)

// ==================== 账户 CRUD ====================

// CreateTokenAccount 创建积分账户
func CreateTokenAccount(ctx context.Context, acc *models.TokenAccount) error {
	query := `
		INSERT INTO token_accounts
			(account_type, owner_id, parent_account_id, display_name,
			 balance, frozen_amount, total_consumed, total_quota,
			 monthly_quota, expires_at, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at, updated_at
	`
	err := database.DB.QueryRow(ctx, query,
		acc.AccountType, acc.OwnerID, acc.ParentAccountID, acc.DisplayName,
		acc.Balance, acc.FrozenAmount, acc.TotalConsumed, acc.TotalQuota,
		acc.MonthlyQuota, acc.ExpiresAt, acc.Status,
	).Scan(&acc.ID, &acc.CreatedAt, &acc.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrDuplicateAccount
		}
		return fmt.Errorf("创建积分账户失败: %w", err)
	}
	return nil
}

// GetTokenAccountByID 根据ID获取积分账户
func GetTokenAccountByID(ctx context.Context, id string) (*models.TokenAccount, error) {
	acc := &models.TokenAccount{}
	err := database.DB.QueryRow(ctx, `
		SELECT id, account_type, owner_id, parent_account_id, display_name,
		       balance, frozen_amount, total_consumed, total_quota,
		       monthly_quota, expires_at, status, created_at, updated_at
		FROM token_accounts WHERE id = $1
	`, id).Scan(
		&acc.ID, &acc.AccountType, &acc.OwnerID, &acc.ParentAccountID, &acc.DisplayName,
		&acc.Balance, &acc.FrozenAmount, &acc.TotalConsumed, &acc.TotalQuota,
		&acc.MonthlyQuota, &acc.ExpiresAt, &acc.Status, &acc.CreatedAt, &acc.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTokenAccountNotFound
		}
		return nil, fmt.Errorf("查询积分账户失败: %w", err)
	}
	return acc, nil
}

// GetTokenAccountByOwner 根据实体类型+实体ID获取账户
func GetTokenAccountByOwner(ctx context.Context, accountType string, ownerID string) (*models.TokenAccount, error) {
	acc := &models.TokenAccount{}
	err := database.DB.QueryRow(ctx, `
		SELECT id, account_type, owner_id, parent_account_id, display_name,
		       balance, frozen_amount, total_consumed, total_quota,
		       monthly_quota, expires_at, status, created_at, updated_at
		FROM token_accounts WHERE account_type = $1 AND owner_id = $2
	`, accountType, ownerID).Scan(
		&acc.ID, &acc.AccountType, &acc.OwnerID, &acc.ParentAccountID, &acc.DisplayName,
		&acc.Balance, &acc.FrozenAmount, &acc.TotalConsumed, &acc.TotalQuota,
		&acc.MonthlyQuota, &acc.ExpiresAt, &acc.Status, &acc.CreatedAt, &acc.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTokenAccountNotFound
		}
		return nil, fmt.Errorf("查询积分账户失败: %w", err)
	}
	return acc, nil
}

// ListTokenAccounts 查询账户列表（v172 owner_id 白名单 + v172.1 scoped 排除 region）
func ListTokenAccounts(ctx context.Context, accountType string, parentAccountID string, status string, ownerIDs []string, limit int, offset int) ([]*models.TokenAccountListItem, int, error) {
	where := "1=1"
	args := []interface{}{}
	argIdx := 1

	if accountType != "" {
		where += fmt.Sprintf(" AND ta.account_type = $%d", argIdx)
		args = append(args, accountType)
		argIdx++
	}
	if parentAccountID != "" {
		where += fmt.Sprintf(" AND ta.parent_account_id = $%d", argIdx)
		args = append(args, parentAccountID)
		argIdx++
	}
	if status != "" {
		where += fmt.Sprintf(" AND ta.status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}

	if ownerIDs != nil {
		if len(ownerIDs) == 0 {
			where += " AND 1=0"
		} else {
			where += fmt.Sprintf(" AND ta.owner_id = ANY($%d)", argIdx)
			args = append(args, ownerIDs)
			argIdx++
			where += " AND ta.account_type <> 'region'"
		}
	}

	var total int
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM token_accounts ta WHERE %s`, where)
	if err := database.DB.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计积分账户数失败: %w", err)
	}

	if limit <= 0 {
		limit = 50
	}

	listQuery := fmt.Sprintf(`
		SELECT ta.id, ta.account_type, ta.owner_id, ta.display_name,
		       ta.balance, ta.frozen_amount, ta.total_consumed, ta.total_quota,
		       ta.monthly_quota, ta.status, ta.expires_at, ta.created_at,
		       (SELECT COUNT(*) FROM token_accounts sub WHERE sub.parent_account_id = ta.id) AS child_count
		FROM token_accounts ta
		WHERE %s
		ORDER BY ta.account_type ASC, ta.display_name ASC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := database.DB.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询积分账户列表失败: %w", err)
	}
	defer rows.Close()

	var items []*models.TokenAccountListItem
	for rows.Next() {
		item := &models.TokenAccountListItem{}
		err := rows.Scan(
			&item.ID, &item.AccountType, &item.OwnerID, &item.DisplayName,
			&item.Balance, &item.FrozenAmount, &item.TotalConsumed, &item.TotalQuota,
			&item.MonthlyQuota, &item.Status, &item.ExpiresAt, &item.CreatedAt,
			&item.ChildCount,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("扫描积分账户行失败: %w", err)
		}
		item.AccountTypeName = models.AccountTypeNameMap[item.AccountType]
		item.StatusName = models.AccountStatusNameMap[item.Status]
		item.AvailableBalance = item.Balance - item.FrozenAmount
		if item.TotalQuota > 0 {
			item.UsagePercent = float64(item.TotalConsumed) * 100.0 / float64(item.TotalQuota)
		}
		items = append(items, item)
	}
	return items, total, nil
}

// ListChildAccounts 获取某账户的所有子账户
func ListChildAccounts(ctx context.Context, parentID string) ([]*models.TokenAccountListItem, error) {
	rows, err := database.DB.Query(ctx, `
		SELECT ta.id, ta.account_type, ta.owner_id, ta.display_name,
		       ta.balance, ta.frozen_amount, ta.total_consumed, ta.total_quota,
		       ta.monthly_quota, ta.status, ta.expires_at, ta.created_at,
		       (SELECT COUNT(*) FROM token_accounts sub WHERE sub.parent_account_id = ta.id) AS child_count
		FROM token_accounts ta
		WHERE ta.parent_account_id = $1
		ORDER BY ta.display_name ASC
	`, parentID)
	if err != nil {
		return nil, fmt.Errorf("查询子账户列表失败: %w", err)
	}
	defer rows.Close()

	var items []*models.TokenAccountListItem
	for rows.Next() {
		item := &models.TokenAccountListItem{}
		err := rows.Scan(
			&item.ID, &item.AccountType, &item.OwnerID, &item.DisplayName,
			&item.Balance, &item.FrozenAmount, &item.TotalConsumed, &item.TotalQuota,
			&item.MonthlyQuota, &item.Status, &item.ExpiresAt, &item.CreatedAt,
			&item.ChildCount,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描子账户行失败: %w", err)
		}
		item.AccountTypeName = models.AccountTypeNameMap[item.AccountType]
		item.StatusName = models.AccountStatusNameMap[item.Status]
		item.AvailableBalance = item.Balance - item.FrozenAmount
		if item.TotalQuota > 0 {
			item.UsagePercent = float64(item.TotalConsumed) * 100.0 / float64(item.TotalQuota)
		}
		items = append(items, item)
	}
	return items, nil
}

// ==================== 余额操作（事务安全）====================

// FreezeTokens 冻结积分（AI调用开始前调用，事务内 SELECT FOR UPDATE）
func FreezeTokens(ctx context.Context, accountID string, amount float64) (*models.TokenAccount, error) {
	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("开启事务失败: %w", err)
	}
	defer tx.Rollback(ctx)

	acc := &models.TokenAccount{}
	err = tx.QueryRow(ctx, `
		SELECT id, balance, frozen_amount, status
		FROM token_accounts WHERE id = $1 FOR UPDATE
	`, accountID).Scan(&acc.ID, &acc.Balance, &acc.FrozenAmount, &acc.Status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTokenAccountNotFound
		}
		return nil, fmt.Errorf("锁定账户失败: %w", err)
	}

	if acc.Status != models.AccountStatusActive {
		return nil, ErrAccountSuspended
	}

	available := acc.Balance - acc.FrozenAmount
	if available < amount {
		return nil, ErrInsufficientBalance
	}

	now := time.Now()
	_, err = tx.Exec(ctx, `
		UPDATE token_accounts
		SET frozen_amount = frozen_amount + $1, updated_at = $2
		WHERE id = $3
	`, amount, now, accountID)
	if err != nil {
		return nil, fmt.Errorf("冻结积分失败: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("提交冻结事务失败: %w", err)
	}

	acc.FrozenAmount += amount
	return acc, nil
}

// DeductTokens 扣减积分（AI调用完成后调用，释放冻结并扣减余额）
func DeductTokens(ctx context.Context, accountID string, frozenAmount float64, actualAmount float64) error {
	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	defer tx.Rollback(ctx)

	now := time.Now()
	result, err := tx.Exec(ctx, `
		UPDATE token_accounts
		SET frozen_amount = frozen_amount - $1,
		    balance = balance - $2,
		    total_consumed = total_consumed + $2,
		    updated_at = $3
		WHERE id = $4 AND frozen_amount >= $1
	`, frozenAmount, actualAmount, now, accountID)
	if err != nil {
		return fmt.Errorf("扣减积分失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("扣减积分失败：账户不存在或冻结额度不一致")
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("提交扣减事务失败: %w", err)
	}
	return nil
}

// UnfreezeTokens 释放冻结额度（AI调用失败时调用，不扣减余额）
func UnfreezeTokens(ctx context.Context, accountID string, amount float64) error {
	now := time.Now()
	result, err := database.DB.Exec(ctx, `
		UPDATE token_accounts
		SET frozen_amount = GREATEST(frozen_amount - $1, 0), updated_at = $2
		WHERE id = $3
	`, amount, now, accountID)
	if err != nil {
		return fmt.Errorf("释放冻结失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrTokenAccountNotFound
	}
	return nil
}

// AddBalance 增加账户余额（采购/充值/分配接收时调用）
func AddBalance(ctx context.Context, accountID string, amount float64) error {
	now := time.Now()
	result, err := database.DB.Exec(ctx, `
		UPDATE token_accounts
		SET balance = balance + $1, total_quota = total_quota + $1, updated_at = $2
		WHERE id = $3
	`, amount, now, accountID)
	if err != nil {
		return fmt.Errorf("增加余额失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrTokenAccountNotFound
	}
	return nil
}

// DirectDeductBalance 直接扣减余额（消费用，允许透支，不动total_quota）
func DirectDeductBalance(ctx context.Context, accountID string, amount float64) error {
	now := time.Now()
	result, err := database.DB.Exec(ctx, `
		UPDATE token_accounts
		SET balance = balance - $1,
		    total_consumed = total_consumed + $1,
		    updated_at = $2
		WHERE id = $3 AND status = 'active'
	`, amount, now, accountID)
	if err != nil {
		return fmt.Errorf("直接扣减余额失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrTokenAccountNotFound
	}
	return nil
}

// DeductBalanceForAllocation 分配时扣减上级账户余额（不增加total_consumed）
func DeductBalanceForAllocation(ctx context.Context, accountID string, amount float64) error {
	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	defer tx.Rollback(ctx)

	var balance float64
	var frozenAmount float64
	err = tx.QueryRow(ctx, `
		SELECT balance, frozen_amount FROM token_accounts
		WHERE id = $1 FOR UPDATE
	`, accountID).Scan(&balance, &frozenAmount)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrTokenAccountNotFound
		}
		return fmt.Errorf("锁定上级账户失败: %w", err)
	}

	available := balance - frozenAmount
	if available < amount {
		return ErrInsufficientBalance
	}

	now := time.Now()
	_, err = tx.Exec(ctx, `
		UPDATE token_accounts SET balance = balance - $1, updated_at = $2 WHERE id = $3
	`, amount, now, accountID)
	if err != nil {
		return fmt.Errorf("扣减上级余额失败: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("提交分配扣减事务失败: %w", err)
	}
	return nil
}

// UpdateTokenAccountStatus 更新账户状态
func UpdateTokenAccountStatus(ctx context.Context, accountID string, status string) error {
	now := time.Now()
	result, err := database.DB.Exec(ctx, `
		UPDATE token_accounts SET status = $1, updated_at = $2 WHERE id = $3
	`, status, now, accountID)
	if err != nil {
		return fmt.Errorf("更新账户状态失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrTokenAccountNotFound
	}
	return nil
}

// ==================== 采购/充值记录 ====================

// CreateTokenPurchase 创建采购/充值记录
func CreateTokenPurchase(ctx context.Context, purchase *models.TokenPurchase) error {
	err := database.DB.QueryRow(ctx, `
		INSERT INTO token_purchases
			(account_id, amount, purchase_type, order_no, memo, operator_id, valid_until)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`,
		purchase.AccountID, purchase.Amount, purchase.PurchaseType,
		purchase.OrderNo, purchase.Memo, purchase.OperatorID, purchase.ValidUntil,
	).Scan(&purchase.ID, &purchase.CreatedAt)
	if err != nil {
		return fmt.Errorf("创建采购记录失败: %w", err)
	}
	return nil
}

// ListTokenPurchases 查询采购记录列表（v172.2 增加 ownerIDs 白名单）
//
// 采购记录本身无 owner 维度，按目标账户 token_accounts.owner_id 过滤。
// 白名单语义（与其它查询一致）：
//   - ownerIDs == nil  → 不过滤（admin 看全部）
//   - ownerIDs 为空切片 → AND 1=0（fail-closed，返回空集）
//   - ownerIDs 非空     → AND ta.owner_id = ANY($n) 且 ta.account_type <> 'region'
//     （scoped 非admin 不应看到上级 region 账户的采购记录）
func ListTokenPurchases(ctx context.Context, accountID string, ownerIDs []string, limit int, offset int) ([]*models.PurchaseListItem, int, error) {
	// 采购记录需 JOIN token_accounts 取 owner，统一在 FROM 中 JOIN
	where := "1=1"
	args := []interface{}{}
	argIdx := 1

	if accountID != "" {
		where += fmt.Sprintf(" AND p.account_id = $%d", argIdx)
		args = append(args, accountID)
		argIdx++
	}

	// v172.2 owner_id 白名单（按目标账户 owner 过滤）
	if ownerIDs != nil {
		if len(ownerIDs) == 0 {
			where += " AND 1=0"
		} else {
			where += fmt.Sprintf(" AND ta.owner_id = ANY($%d)", argIdx)
			args = append(args, ownerIDs)
			argIdx++
			where += " AND ta.account_type <> 'region'"
		}
	}

	var total int
	// 统计也需 JOIN（owner 过滤依赖 ta）
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM token_purchases p
		LEFT JOIN token_accounts ta ON ta.id = p.account_id
		WHERE %s
	`, where)
	if err := database.DB.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计采购记录数失败: %w", err)
	}

	if limit <= 0 {
		limit = 50
	}

	listQuery := fmt.Sprintf(`
		SELECT p.id, COALESCE(ta.display_name, '') AS account_name,
		       p.amount, p.purchase_type, p.order_no, p.memo,
		       COALESCE(u.display_name, '') AS operator_name,
		       p.valid_until, p.created_at
		FROM token_purchases p
		LEFT JOIN token_accounts ta ON ta.id = p.account_id
		LEFT JOIN users u ON u.id = p.operator_id
		WHERE %s
		ORDER BY p.created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := database.DB.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询采购记录列表失败: %w", err)
	}
	defer rows.Close()

	var items []*models.PurchaseListItem
	for rows.Next() {
		item := &models.PurchaseListItem{}
		err := rows.Scan(
			&item.ID, &item.AccountName, &item.Amount, &item.PurchaseType,
			&item.OrderNo, &item.Memo, &item.OperatorName,
			&item.ValidUntil, &item.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("扫描采购记录行失败: %w", err)
		}
		items = append(items, item)
	}
	return items, total, nil
}

// ==================== 预警配置 ====================

// GetTokenAlertConfig 获取账户的预警配置
func GetTokenAlertConfig(ctx context.Context, accountID string) (*models.TokenAlertConfig, error) {
	cfg := &models.TokenAlertConfig{}
	err := database.DB.QueryRow(ctx, `
		SELECT id, account_id, warn_threshold, urgent_threshold,
		       is_enabled, last_warn_at, last_urgent_at, created_at, updated_at
		FROM token_alert_configs WHERE account_id = $1
	`, accountID).Scan(
		&cfg.ID, &cfg.AccountID, &cfg.WarnThreshold, &cfg.UrgentThreshold,
		&cfg.IsEnabled, &cfg.LastWarnAt, &cfg.LastUrgentAt, &cfg.CreatedAt, &cfg.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("查询预警配置失败: %w", err)
	}
	return cfg, nil
}

// UpsertTokenAlertConfig 创建或更新预警配置
func UpsertTokenAlertConfig(ctx context.Context, accountID string, req *models.UpdateAlertConfigRequest) error {
	now := time.Now()
	_, err := database.DB.Exec(ctx, `
		INSERT INTO token_alert_configs
			(account_id, warn_threshold, urgent_threshold, is_enabled, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (account_id) DO UPDATE SET
			warn_threshold = EXCLUDED.warn_threshold,
			urgent_threshold = EXCLUDED.urgent_threshold,
			is_enabled = EXCLUDED.is_enabled,
			updated_at = EXCLUDED.updated_at
	`, accountID, req.WarnThreshold, req.UrgentThreshold, req.IsEnabled, now)
	if err != nil {
		return fmt.Errorf("更新预警配置失败: %w", err)
	}
	return nil
}

// ==================== 概览统计 ====================

// GetTokenOverviewStats 获取Token系统概览统计（全系统，admin 用）
func GetTokenOverviewStats(ctx context.Context) (*models.TokenOverviewStats, error) {
	return GetTokenOverviewStatsScoped(ctx, nil)
}

// GetTokenOverviewStatsScoped 获取概览统计（v172 owner_id 白名单 + v172.1 scoped 排除 region）
func GetTokenOverviewStatsScoped(ctx context.Context, ownerIDs []string) (*models.TokenOverviewStats, error) {
	stats := &models.TokenOverviewStats{}

	if ownerIDs != nil && len(ownerIDs) == 0 {
		return stats, nil
	}

	scoped := ownerIDs != nil

	if scoped {
		_ = database.DB.QueryRow(ctx,
			`SELECT COUNT(*) FROM token_accounts
			 WHERE owner_id = ANY($1) AND account_type <> 'region'`, ownerIDs,
		).Scan(&stats.TotalAccounts)

		_ = database.DB.QueryRow(ctx,
			`SELECT COALESCE(SUM(balance),0), COALESCE(SUM(total_consumed),0), COALESCE(SUM(total_quota),0)
			 FROM token_accounts WHERE owner_id = ANY($1) AND account_type <> 'region'`, ownerIDs,
		).Scan(&stats.TotalBalance, &stats.TotalConsumed, &stats.TotalQuota)

		_ = database.DB.QueryRow(ctx,
			`SELECT COALESCE(SUM(cl.amount),0)
			 FROM token_consumption_logs cl
			 JOIN token_accounts ta ON ta.id = cl.account_id
			 WHERE cl.created_at >= CURRENT_DATE AND ta.owner_id = ANY($1) AND ta.account_type <> 'region'`, ownerIDs,
		).Scan(&stats.TodayConsumed)

		_ = database.DB.QueryRow(ctx,
			`SELECT COALESCE(SUM(cl.amount),0)
			 FROM token_consumption_logs cl
			 JOIN token_accounts ta ON ta.id = cl.account_id
			 WHERE cl.created_at >= date_trunc('month', CURRENT_DATE) AND ta.owner_id = ANY($1) AND ta.account_type <> 'region'`, ownerIDs,
		).Scan(&stats.MonthConsumed)

		_ = database.DB.QueryRow(ctx,
			`SELECT COUNT(*) FROM token_accounts
			 WHERE owner_id = ANY($1) AND account_type <> 'region' AND status = 'active' AND total_quota > 0
			       AND (balance - frozen_amount) < total_quota * 0.2`, ownerIDs,
		).Scan(&stats.LowBalanceCount)

		_ = database.DB.QueryRow(ctx,
			`SELECT COUNT(*) FROM token_accounts
			 WHERE owner_id = ANY($1) AND account_type <> 'region' AND status = 'active' AND expires_at IS NOT NULL
			       AND expires_at <= NOW() + INTERVAL '30 days'`, ownerIDs,
		).Scan(&stats.ExpiringSoonCount)

		return stats, nil
	}

	_ = database.DB.QueryRow(ctx,
		`SELECT COUNT(*) FROM token_accounts`,
	).Scan(&stats.TotalAccounts)

	_ = database.DB.QueryRow(ctx,
		`SELECT COALESCE(SUM(balance),0), COALESCE(SUM(total_consumed),0), COALESCE(SUM(total_quota),0)
		 FROM token_accounts`,
	).Scan(&stats.TotalBalance, &stats.TotalConsumed, &stats.TotalQuota)

	_ = database.DB.QueryRow(ctx,
		`SELECT COALESCE(SUM(amount),0) FROM token_consumption_logs
		 WHERE created_at >= CURRENT_DATE`,
	).Scan(&stats.TodayConsumed)

	_ = database.DB.QueryRow(ctx,
		`SELECT COALESCE(SUM(amount),0) FROM token_consumption_logs
		 WHERE created_at >= date_trunc('month', CURRENT_DATE)`,
	).Scan(&stats.MonthConsumed)

	_ = database.DB.QueryRow(ctx,
		`SELECT COUNT(*) FROM token_accounts
		 WHERE status = 'active' AND total_quota > 0
		       AND (balance - frozen_amount) < total_quota * 0.2`,
	).Scan(&stats.LowBalanceCount)

	_ = database.DB.QueryRow(ctx,
		`SELECT COUNT(*) FROM token_accounts
		 WHERE status = 'active' AND expires_at IS NOT NULL
		       AND expires_at <= NOW() + INTERVAL '30 days'`,
	).Scan(&stats.ExpiringSoonCount)

	return stats, nil
}

// ==================== 辅助函数 ====================

// isUniqueViolation 检查是否为唯一约束冲突错误
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	return fmt.Sprintf("%v", err) != "" &&
		(tokenContains(err.Error(), "unique") || tokenContains(err.Error(), "duplicate") ||
			tokenContains(err.Error(), "23505"))
}

// tokenContains 字符串包含检查（辅助函数）
func tokenContains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
