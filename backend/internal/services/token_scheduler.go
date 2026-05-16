package services

// token_scheduler.go — Token积分系统定时任务
//
// v128 新增（阶段C · Token/积分系统）：
//   - 月度自动充值：每月1号凌晨4:00为配置了monthly_quota的账户自动充值
//   - 预警检查：每天凌晨5:00检查余额低于阈值的账户
//
// 设计原则：
//   与 verify_batch.go / course_nightly.go 模式一致
//   使用 time.Timer 定时触发，单goroutine执行

import (
	"context"
	"log"
	"time"

	"tedna/internal/database"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== 月度自动充值 ====================

// StartMonthlyQuotaScheduler 启动月度自动充值定时任务
// 每月1号凌晨4:00执行
func (s *TokenService) StartMonthlyQuotaScheduler() {
	go func() {
		for {
			now := time.Now()
			// 计算下一个月1号凌晨4:00
			nextMonth := time.Date(now.Year(), now.Month()+1, 1, 4, 0, 0, 0, now.Location())
			sleepDuration := nextMonth.Sub(now)

			log.Printf("[Token定时] 月度自动充值 下次执行: %s（%s后）",
				nextMonth.Format("2006-01-02 15:04:05"), sleepDuration.Round(time.Minute))

			time.Sleep(sleepDuration)
			s.doMonthlyQuotaRefill()
		}
	}()
}

// doMonthlyQuotaRefill 执行月度自动充值
func (s *TokenService) doMonthlyQuotaRefill() {
	ctx := context.Background()
	log.Printf("[Token定时] 开始月度自动充值")

	// 查询所有配置了月度配额且活跃的账户
	rows, err := database.DB.Query(ctx, `
		SELECT id, display_name, monthly_quota, account_type
		FROM token_accounts
		WHERE monthly_quota > 0 AND status = 'active'
	`)
	if err != nil {
		log.Printf("[Token定时] 查询月度配额账户失败: %v", err)
		return
	}
	defer rows.Close()

	var success, failed int
	for rows.Next() {
		var id, displayName, accountType string
		var monthlyQuota float64
		if err := rows.Scan(&id, &displayName, &monthlyQuota, &accountType); err != nil {
			failed++
			continue
		}

		// 增加余额
		if err := repository.AddBalance(ctx, id, monthlyQuota); err != nil {
			log.Printf("[Token定时] 账户 %s(%s) 充值失败: %v", displayName, id, err)
			failed++
			continue
		}

		// 记录分配流水
		alloc := &models.TokenAllocation{
			FromAccountID:  id, // 自充值，from=to
			ToAccountID:    id,
			Amount:         monthlyQuota,
			AllocationType: models.AllocationTypeMonthly,
			Memo:           "月度自动充值",
			OperatorID:     "00000000-0000-0000-0000-000000000000", // 系统操作
		}
		_ = repository.CreateTokenAllocation(ctx, alloc)

		success++
		log.Printf("[Token定时] 账户 %s(%s) 月度充值 +%d 积分", displayName, id, monthlyQuota)
	}

	log.Printf("[Token定时] 月度自动充值完成: 成功=%d 失败=%d", success, failed)
}

// ==================== 预警检查 ====================

// StartAlertCheckScheduler 启动预警检查定时任务
// 每天凌晨5:00执行
func (s *TokenService) StartAlertCheckScheduler() {
	go func() {
		for {
			now := time.Now()
			// 计算今天/明天凌晨5:00
			next := time.Date(now.Year(), now.Month(), now.Day(), 5, 0, 0, 0, now.Location())
			if next.Before(now) {
				next = next.Add(24 * time.Hour)
			}
			sleepDuration := next.Sub(now)

			log.Printf("[Token定时] 预警检查 下次执行: %s（%s后）",
				next.Format("2006-01-02 15:04:05"), sleepDuration.Round(time.Minute))

			time.Sleep(sleepDuration)
			s.doAlertCheck()
		}
	}()
}

// doAlertCheck 执行预警检查
func (s *TokenService) doAlertCheck() {
	ctx := context.Background()
	log.Printf("[Token定时] 开始预警检查")

	// 查询所有启用了预警的配置
	rows, err := database.DB.Query(ctx, `
		SELECT ac.id, ac.account_id, ac.warn_threshold, ac.urgent_threshold,
		       ta.display_name, ta.balance, ta.frozen_amount, ta.total_quota
		FROM token_alert_configs ac
		JOIN token_accounts ta ON ta.id = ac.account_id
		WHERE ac.is_enabled = TRUE AND ta.status = 'active' AND ta.total_quota > 0
	`)
	if err != nil {
		log.Printf("[Token定时] 查询预警配置失败: %v", err)
		return
	}
	defer rows.Close()

	var warnCount, urgentCount int
	now := time.Now()

	for rows.Next() {
		var configID, accountID, displayName string
		var warnThreshold, urgentThreshold int
		var balance, frozenAmount, totalQuota float64

		if err := rows.Scan(&configID, &accountID, &warnThreshold, &urgentThreshold,
			&displayName, &balance, &frozenAmount, &totalQuota); err != nil {
			continue
		}

		// 计算已使用比例
		available := balance - frozenAmount
		usedPercent := 100.0 - (float64(available)*100.0)/float64(totalQuota)

		if usedPercent >= float64(urgentThreshold) {
			urgentCount++
			log.Printf("[Token预警-紧急] 账户 %s: 已用%.1f%% (阈值%d%%), 可用余额=%.2f",
				displayName, usedPercent, urgentThreshold, available)
			// 更新上次紧急预警时间
			_, _ = database.DB.Exec(ctx,
				`UPDATE token_alert_configs SET last_urgent_at = $1, updated_at = $1 WHERE id = $2`,
				now, configID)
		} else if usedPercent >= float64(warnThreshold) {
			warnCount++
			log.Printf("[Token预警-警告] 账户 %s: 已用%.1f%% (阈值%d%%), 可用余额=%.2f",
				displayName, usedPercent, warnThreshold, available)
			// 更新上次预警时间
			_, _ = database.DB.Exec(ctx,
				`UPDATE token_alert_configs SET last_warn_at = $1, updated_at = $1 WHERE id = $2`,
				now, configID)
		}
	}

	log.Printf("[Token定时] 预警检查完成: 警告=%d 紧急=%d", warnCount, urgentCount)
}
