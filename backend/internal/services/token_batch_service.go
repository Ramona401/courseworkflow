package services

// token_batch_service.go — Token积分批量分配业务逻辑（一键分配 batch 新增）
//
// 背景：区域管理员向辖区学校分配积分（如郯城教育局下 54 所学校）、学校管理员向
// 全校老师分配积分时，逐个打开分配弹窗操作极其低效。本文件提供"一键批量分配"：
// 选中多个目标账户 + 每户定额，一次请求完成全部分配。
//
// 设计决策：
//   1. 每户定额（amount_each）而非总额均分——每个目标分到同样金额，总额=每户×N，
//      最直观可预期，不存在除不尽的尾数分配纠纷。
//   2. 预检 + 逐笔执行 + 收集成败（镜像跨校批量建用户 BatchCreateUsersMultiSchool 哲学）：
//      - 预检（整体拦截，一笔不分）：金额>0 / 目标数 1~maxBatchAllocateTargets /
//        目标ID去重防重复分配 / 来源账户可用余额 ≥ 每户×N（余额不足时明确告知缺口）。
//      - 逐笔执行：每笔完整复用既有 AllocateTokens（来源/目标状态校验、父子关系校验、
//        扣减、回滚补偿、分配流水一样不少），单笔失败只进 failures 不回滚已成功笔——
//        每笔分配本就相互独立，失败笔的钱仍留在来源账户可单独重试，无需整体事务。
//      - 返回 BatchAllocateResult{成功数/失败数/实际分出总额/失败明细}，前端逐条展示。
//   3. 不在本层做权限判断——"来源账户可作分配来源"由 handler 层 tokenSourceAllowed
//      校验（与单笔分配同一口径），本层只管业务执行；"目标是否合法下级"由
//      AllocateTokens 内的父子关系校验兜底（非下级的目标该笔自然失败进 failures）。
//
// 权限口径（handler + service 双控，与单笔 /allocate 完全一致）：
//   路由层 admin + senior_operator + region_admin；
//   handler 层 senior/region 须过 tokenSourceAllowed（admin 跳过由父子校验兜底）。

import (
	"context"
	"fmt"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// maxBatchAllocateTargets 单次批量分配目标数上限（防误操作与超长请求；54所学校场景绰绰有余）
const maxBatchAllocateTargets = 500

// BatchAllocateRequest 批量分配请求体
type BatchAllocateRequest struct {
	ToAccountIDs []string `json:"to_account_ids"` // 目标账户ID列表（勾选的下级账户）
	AmountEach   float64  `json:"amount_each"`    // 每户分配金额（每个目标分同样金额）
	Memo         string   `json:"memo"`           // 备注（写入每笔分配流水）
}

// BatchAllocateFailure 单笔失败明细
type BatchAllocateFailure struct {
	ToAccountID string `json:"to_account_id"` // 失败的目标账户ID
	Reason      string `json:"reason"`        // 失败原因（中文，直接可展示）
}

// BatchAllocateResult 批量分配结果
type BatchAllocateResult struct {
	SuccessCount   int                    `json:"success_count"`   // 成功笔数
	FailCount      int                    `json:"fail_count"`      // 失败笔数
	TotalAllocated float64                `json:"total_allocated"` // 实际分出总额（=成功笔数×每户金额）
	Failures       []BatchAllocateFailure `json:"failures"`        // 失败明细（恒非nil，空数组=全部成功）
}

// BatchAllocateTokens 批量分配积分（预检整体拦截 + 逐笔执行收集成败）
//
// 返回 error 仅限"预检不通过/系统级异常"（此时一笔未分，handler 转 400/500）；
// 进入逐笔阶段后 error 恒为 nil，单笔失败只进 result.Failures。
func (s *TokenService) BatchAllocateTokens(ctx context.Context, fromAccountID string, req *BatchAllocateRequest, operatorID string) (*BatchAllocateResult, error) {
	// ==================== 阶段一：预检（任一不过整体拦截，一笔不分）====================

	// 金额校验：每户金额必须 > 0
	if req.AmountEach <= 0 {
		return nil, ErrTokenInvalidAmount
	}

	// 目标列表校验：非空 + 上限 + 去重（前端重复勾选/重放不会导致同一账户被分两次）
	if len(req.ToAccountIDs) == 0 {
		return nil, fmt.Errorf("请至少选择一个目标账户")
	}
	if len(req.ToAccountIDs) > maxBatchAllocateTargets {
		return nil, fmt.Errorf("单次批量分配目标不能超过 %d 个", maxBatchAllocateTargets)
	}
	seen := make(map[string]struct{}, len(req.ToAccountIDs))
	targets := make([]string, 0, len(req.ToAccountIDs))
	for _, id := range req.ToAccountIDs {
		if id == "" {
			continue // 空ID直接丢弃（防御性）
		}
		if id == fromAccountID {
			return nil, ErrTokenSelfAllocate // 目标含来源自身，整体拦截
		}
		if _, dup := seen[id]; dup {
			continue // 重复ID只保留一次
		}
		seen[id] = struct{}{}
		targets = append(targets, id)
	}
	if len(targets) == 0 {
		return nil, fmt.Errorf("请至少选择一个有效的目标账户")
	}

	// 来源账户校验：存在 + 活跃 + 可用余额足够覆盖全部目标
	fromAcc, err := repository.GetTokenAccountByID(ctx, fromAccountID)
	if err != nil {
		return nil, fmt.Errorf("来源账户不存在: %w", err)
	}
	if fromAcc.Status != models.AccountStatusActive {
		return nil, ErrTokenAccountNotActive
	}
	totalNeeded := req.AmountEach * float64(len(targets))
	available := fromAcc.Balance - fromAcc.FrozenAmount
	if available < totalNeeded {
		// 余额不足整体拦截并明确告知缺口（分配是明确的额度下发动作，不允许透支）
		return nil, fmt.Errorf("来源账户可用余额不足：需要 %.2f 积分（%d 个目标 × %.2f），当前可用 %.2f",
			totalNeeded, len(targets), req.AmountEach, available)
	}

	// ==================== 阶段二：逐笔执行（收集成败，不整体回滚）====================
	// 每笔完整复用 AllocateTokens：状态校验/父子关系校验/扣减/失败补偿/流水记录一样不少。
	// 并发执行期间余额被其他操作消耗导致后续笔余额不足时，该笔进 failures（钱不会超分）。
	result := &BatchAllocateResult{
		Failures: make([]BatchAllocateFailure, 0), // 恒非nil，保证JSON序列化为 [] 而非 null
	}
	for _, toID := range targets {
		allocReq := &models.AllocateTokensRequest{
			ToAccountID: toID,
			Amount:      req.AmountEach,
			Memo:        req.Memo,
		}
		if allocErr := s.AllocateTokens(ctx, fromAccountID, allocReq, operatorID); allocErr != nil {
			result.FailCount++
			result.Failures = append(result.Failures, BatchAllocateFailure{
				ToAccountID: toID,
				Reason:      batchAllocateReasonText(allocErr),
			})
			continue // 单笔失败不阻断后续目标
		}
		result.SuccessCount++
		result.TotalAllocated += req.AmountEach
	}

	tokenLog.Info("批量分配完成",
		"from", fromAccountID,
		"targets", len(targets),
		"amount_each", req.AmountEach,
		"success", result.SuccessCount,
		"fail", result.FailCount,
		"operator", operatorID)
	return result, nil
}

// batchAllocateReasonText 把单笔分配错误翻译为可直接展示的中文原因
//（与 handler 层 handleTokenError 的映射口径一致，但这里返回文本进 failures 而非HTTP响应）
func batchAllocateReasonText(err error) string {
	switch {
	case err == nil:
		return ""
	case err == ErrTokenInvalidAmount:
		return "积分数量必须大于0"
	case err == ErrTokenSelfAllocate:
		return "不能分配给自己"
	case err == ErrTokenNotParentChild:
		return "该账户不是来源账户的下级"
	case err == ErrTokenAccountNotActive:
		return "账户不在活跃状态"
	default:
		return err.Error()
	}
}
