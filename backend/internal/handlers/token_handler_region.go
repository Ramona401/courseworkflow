package handlers

// token_handler_region.go — "我管辖的区域账户"查询端点（T1 区域分配入口 batch 新增）
//
// 背景与动机：
//   v172.1 起，非 admin 的账户列表在 SQL 层无条件排除一切 region 账户（防跨级泄漏），
//   region_admin 连自己管辖的区域账户也看不见——账户都不可见，前端自然没有
//   "从区域账户向学校分配"的入口，导致 admin 只能替区域管理员逐校分配。
//   而后端分配链路（tokenSourceAllowed + GetAllocatableTargets + AllocateTokens）
//   早已按 AllowedRegionOwnerIDs 放行 region_admin 从自己区域账户分配，仅缺发现入口。
//
// 本文件外科手术式补齐该入口（独立文件，token_handler.go 已近千行不再增重）：
//   GET /api/v1/tokens/my-region-accounts
//     - 登录即可（authMW），数据完全由 TokenScope.AllowedRegionOwnerIDs 驱动：
//       仅 region_admin 该白名单非空；admin 为 nil（admin 在账户列表本就能看到全部
//       region 账户，无需本入口）；senior/operator/viewer 也为 nil —— 一律返回
//       空列表，fail-closed，绝不放大可见范围。
//     - 仅返回"请求者自己管辖的区域"的积分账户；其余 region 账户依旧完全不可见，
//       v172.1 的账户列表/概览统计"排除 region"防线一字不动。
//     - 某管辖区域尚未创建积分账户时跳过该区域并计入 missing_accounts，
//       供前端提示"请联系系统管理员为区域创建积分账户"。
//
// 返回结构：{ items: TokenAccountListItem[], total: number, missing_accounts: number }
//   items 复用账户列表项结构（含 available_balance），前端卡片直接消费，
//   并把 item.id 作为 fromAccountId 交给现有 AllocateModal + getAllocatableTargets，
//   分配链路零改动（后端校验已全放行，越权测试见 region_admin 区域分配 batch）。

import (
	"errors"
	"net/http"

	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// GetMyRegionAccounts 查询"我管辖的区域账户"列表
// GET /api/v1/tokens/my-region-accounts
//
// 数据流：
//   resolveScope（积分系统唯一权限决策点，fail-closed）
//     → 遍历 scope.AllowedRegionOwnerIDs（仅 region_admin 非空）
//     → 逐区域 GetAccountByOwner(region, ownerID) 取积分账户
//     → 组装 TokenAccountListItem（与 ListTokenAccounts 展示口径一致）
//
// 安全边界：
//   - 白名单来源是 ResolveTokenScope 的 region_admin 分支
//     （organization_admins 双来源管辖解析，与分配来源校验同源），
//     本端点不接收任何前端传入的区域/账户参数，无伪造面。
//   - 单区域查询失败只跳过该区域，不整体失败、不泄漏错误细节。
func (h *TokenHandler) GetMyRegionAccounts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.JSON(w, http.StatusMethodNotAllowed, -1, "仅支持GET请求", nil)
		return
	}

	// 解析请求者数据范围（未认证/异常 → AllowedRegionOwnerIDs 为空 → 空列表）
	scope := h.resolveScope(r)

	// items 恒为非 nil 切片，保证 JSON 序列化为 [] 而非 null（前端 .map 安全）
	items := []*models.TokenAccountListItem{}
	missing := 0

	if scope != nil {
		// 遍历"可作分配来源的区域账户 owner 白名单"：
		//   仅 region_admin 非空（= 其管辖的区域组织ID）；
		//   admin/senior/operator/viewer 均为 nil，循环体不执行，天然返回空列表。
		for _, ownerID := range scope.AllowedRegionOwnerIDs {
			acc, err := h.tokenService.GetAccountByOwner(r.Context(), models.AccountTypeRegion, ownerID)
			if err != nil {
				if errors.Is(err, repository.ErrTokenAccountNotFound) {
					// 该区域尚未创建积分账户 → 计数提示，不整体失败
					missing++
					continue
				}
				// 其他查询错误：跳过该区域（单区域失败不阻断其余区域展示）
				continue
			}

			// 组装列表项（字段口径与 ListTokenAccounts 一致，前端表格/卡片可直接复用）
			item := &models.TokenAccountListItem{
				ID:               acc.ID,
				AccountType:      acc.AccountType,
				AccountTypeName:  models.AccountTypeNameMap[acc.AccountType],
				OwnerID:          acc.OwnerID,
				DisplayName:      acc.DisplayName,
				Balance:          acc.Balance,
				FrozenAmount:     acc.FrozenAmount,
				AvailableBalance: acc.Balance - acc.FrozenAmount,
				TotalConsumed:    acc.TotalConsumed,
				TotalQuota:       acc.TotalQuota,
				MonthlyQuota:     acc.MonthlyQuota,
				Status:           acc.Status,
				StatusName:       models.AccountStatusNameMap[acc.Status],
				ExpiresAt:        acc.ExpiresAt,
				CreatedAt:        acc.CreatedAt,
			}
			if item.TotalQuota > 0 {
				item.UsagePercent = item.TotalConsumed * 100.0 / item.TotalQuota
			}
			items = append(items, item)
		}
	}

	utils.JSON(w, http.StatusOK, 0, "", map[string]interface{}{
		"items":            items,
		"total":            len(items),
		"missing_accounts": missing,
	})
}
