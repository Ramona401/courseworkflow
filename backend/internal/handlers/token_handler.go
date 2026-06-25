package handlers

// token_handler.go — Token积分系统HTTP处理器
//
// v128 新增（阶段C · Token/积分系统）
// v172 改造（积分管理三级数据权限隔离）：概览/账户/分配/消费 接入 TokenScope
// v172.2 新增：
//   - ListPurchases 接入 TokenScope（采购记录按角色范围过滤，兑现"本人采购记录"）
//   - GetAccountDetail 增加 region 硬拒绝：非admin 永不可查看 region 账户详情，
//     在 owner 白名单判断之前先拒，堵住"region 账户 owner_id 与某 user_id 碰撞 → 详情越权"漏洞
//
// 究极彻底版·批次1 新增（账户选择/搜索基础设施）：
//   - ListAccounts 读取 keyword query 参数，传给 ListAccountsScoped 做账户名模糊搜索。
//   - 新增 GetAllocatableTargets 端点：给定来源账户ID，返回其"合法下级账户"。
//     用于究极版分配弹窗——前端打开分配弹窗时拿该账户的合法下级，根本列不出非下级账户。
//
// 究极彻底版·A 新增（分配记录 total 精确）：
//   - ListAllocations 读取 exclude_monthly query 参数（"true" 时排除月度自充值），
//     传给 ListAllocationsScoped，使 items 与 total 一致。前端分配记录Tab 传 true。
//
// region_admin 区域分配 batch 新增（高风险项，独立迭代，单独测越权）：
//   引入统一的"分配来源校验"辅助函数 tokenSourceAllowed，取代原先散落的
//   "region 硬拒 + tokenOwnerInScope" 两段逻辑，使 GetAllocatableTargets 与
//   AllocateTokens 用同一口径判断"某账户能否作为分配来源"：
//     - admin                  → 任意账户均可作来源
//     - region 账户            → 仅当其 owner_id ∈ scope.AllowedRegionOwnerIDs（即请求者
//                                自己管辖的区域账户）时放行；其余 region 账户一律拒（堵跨区域、
//                                堵 senior 碰 region）。列表/统计 SQL 仍排除 region 不受影响。
//     - school/personal 账户   → 仅当其 owner_id ∈ scope.OwnerIDs 时放行
//   据此：region_admin 可从自己的区域账户向管辖学校账户分配；senior 仍只能从本校账户分配；
//   任何人都不能从非自己管辖的区域/学校账户分配。
//
// 权限设计（路由层 + 数据层双控）：
//   - 概览/账户列表/分配记录/消费流水/采购记录：登录即可访问（authMW），数据由 TokenScope 收窄
//   - 创建账户/状态/预警配置：admin
//   - 充值（采购POST）：admin
//   - 积分分配 / 可分配下级查询：admin + senior_operator + region_admin（来源账户范围校验在 handler）
//   - 我的积分：登录即可

import (
        "encoding/json"
        "errors"
        "net/http"
        "strconv"

        "tedna/internal/middleware"
        "tedna/internal/models"
        "tedna/internal/repository"
        "tedna/internal/services"
        "tedna/internal/utils"
)

// TokenHandler Token积分系统HTTP处理器
type TokenHandler struct {
        tokenService *services.TokenService
}

// NewTokenHandler 创建TokenHandler实例
func NewTokenHandler(tokenService *services.TokenService) *TokenHandler {
        return &TokenHandler{tokenService: tokenService}
}

// resolveScope 从请求上下文解析当前请求者的数据可见范围（统一入口，fail-closed）
func (h *TokenHandler) resolveScope(r *http.Request) *services.TokenScope {
        claims, ok := middleware.GetClaims(r.Context())
        if !ok || claims == nil {
                return h.tokenService.ResolveTokenScope(r.Context(), "", "")
        }
        return h.tokenService.ResolveTokenScope(r.Context(), claims.Role, claims.UserID)
}

// ==================== 概览统计 ====================

// GetOverviewStats 获取Token系统概览统计（按角色范围统计）
// GET /api/v1/tokens/overview
func (h *TokenHandler) GetOverviewStats(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodGet {
                utils.JSON(w, http.StatusMethodNotAllowed, -1, "仅支持GET请求", nil)
                return
        }
        scope := h.resolveScope(r)
        stats, err := h.tokenService.GetOverviewStatsScoped(r.Context(), scope)
        if err != nil {
                utils.JSON(w, http.StatusInternalServerError, -1, "获取统计失败", nil)
                return
        }
        utils.JSON(w, http.StatusOK, 0, "", stats)
}

// ==================== 账户管理 ====================

// CreateAccount 创建积分账户
// POST /api/v1/tokens/accounts
func (h *TokenHandler) CreateAccount(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
                utils.JSON(w, http.StatusMethodNotAllowed, -1, "仅支持POST请求", nil)
                return
        }
        var req models.CreateTokenAccountRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
                utils.JSON(w, http.StatusBadRequest, -1, "请求体解析失败", nil)
                return
        }
        if req.OwnerID == "" {
                utils.JSON(w, http.StatusBadRequest, -1, "owner_id不能为空", nil)
                return
        }
        acc, err := h.tokenService.CreateAccount(r.Context(), &req)
        if err != nil {
                handleTokenError(w, err)
                return
        }
        utils.JSON(w, http.StatusOK, 0, "创建成功", acc)
}

// ListAccounts 查询账户列表（按角色范围过滤 + 批次1 keyword 搜索）
// GET /api/v1/tokens/accounts?type=&parent_id=&status=&keyword=&limit=&offset=
func (h *TokenHandler) ListAccounts(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodGet {
                utils.JSON(w, http.StatusMethodNotAllowed, -1, "仅支持GET请求", nil)
                return
        }
        q := r.URL.Query()
        accountType := q.Get("type")
        parentID := q.Get("parent_id")
        status := q.Get("status")
        keyword := q.Get("keyword") // 批次1：账户名模糊搜索
        limit, _ := strconv.Atoi(q.Get("limit"))
        offset, _ := strconv.Atoi(q.Get("offset"))

        scope := h.resolveScope(r)

        items, total, err := h.tokenService.ListAccountsScoped(r.Context(), accountType, parentID, status, keyword, scope, limit, offset)
        if err != nil {
                utils.JSON(w, http.StatusInternalServerError, -1, "查询失败", nil)
                return
        }
        resp := map[string]interface{}{
                "items": items,
                "total": total,
        }
        if scope != nil && scope.Blocked {
                resp["scope_blocked"] = true
                resp["scope_message"] = scope.BlockedReason
        }
        utils.JSON(w, http.StatusOK, 0, "", resp)
}

// GetAllocatableTargets 查询某来源账户的"合法下级账户"（究极版分配弹窗用）
// GET /api/v1/tokens/accounts/{id}/allocatable-targets
//
// 防越权三重防护：
//   ① 先校验来源账户可作分配来源（tokenSourceAllowed）；不可 → 403
//   ② ListChildAccountsScoped 对下级再做 scope 白名单二次收窄
//   ③ 真正 allocate 时 service 层父子校验兜底
// 路由放行 admin + senior_operator + region_admin。
func (h *TokenHandler) GetAllocatableTargets(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodGet {
                utils.JSON(w, http.StatusMethodNotAllowed, -1, "仅支持GET请求", nil)
                return
        }
        fromAccountID := extractTokenMiddleID(r.URL.Path, "/allocatable-targets")
        if fromAccountID == "" {
                utils.JSON(w, http.StatusBadRequest, -1, "缺少来源账户ID", nil)
                return
        }

        scope := h.resolveScope(r)

        // ① 校验来源账户可作分配来源（admin 放行；region 账户须在自己管辖区域内；
        //    school/personal 须 owner 命中 OwnerIDs 白名单）
        fromAcc, accErr := h.tokenService.GetAccount(r.Context(), fromAccountID)
        if accErr != nil {
                handleTokenError(w, accErr)
                return
        }
        if !tokenSourceAllowed(scope, fromAcc.AccountType, fromAcc.OwnerID) {
                utils.JSON(w, http.StatusForbidden, -1, "无权从该账户分配积分", nil)
                return
        }

        // ② 取该账户的合法下级（下级再被 scope 白名单收窄）
        items, err := h.tokenService.ListChildAccountsScoped(r.Context(), fromAccountID, scope)
        if err != nil {
                utils.JSON(w, http.StatusInternalServerError, -1, "查询可分配下级失败", nil)
                return
        }
        utils.JSON(w, http.StatusOK, 0, "", map[string]interface{}{
                "items": items,
                "total": len(items),
        })
}

// GetAccountDetail 获取账户详情
// GET /api/v1/tokens/accounts/{id}
//
// v172.2：非admin 永不可查看 region 账户详情（在 owner 白名单判断之前先拒），
// 堵住 region 账户 owner_id 与某 user_id 碰撞导致的详情越权。
//
// region_admin 区域分配 batch 说明：此处仍对所有非admin 硬拒 region 账户详情，
//   region_admin 也不例外——本轮只放开"从区域账户分配"的来源校验，不放开"查看区域账户详情"。
//   区域账户详情能力留待后续迭代（与洞③创建账户、未来区域账户管理一并设计），
//   避免本 batch 越权面外溢。
func (h *TokenHandler) GetAccountDetail(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodGet {
                utils.JSON(w, http.StatusMethodNotAllowed, -1, "仅支持GET请求", nil)
                return
        }
        accountID := extractTokenPathID(r.URL.Path)
        if accountID == "" {
                utils.JSON(w, http.StatusBadRequest, -1, "缺少账户ID", nil)
                return
        }

        detail, err := h.tokenService.GetAccount(r.Context(), accountID)
        if err != nil {
                handleTokenError(w, err)
                return
        }
        scope := h.resolveScope(r)
        if scope != nil && !scope.IsAdmin {
                // v172.2：region 账户对任何非admin 一律不可见（硬拒绝，优先于 owner 白名单）
                if detail.AccountType == models.AccountTypeRegion {
                        utils.JSON(w, http.StatusForbidden, -1, "无权查看该账户", nil)
                        return
                }
                if !tokenOwnerInScope(scope, detail.OwnerID) {
                        utils.JSON(w, http.StatusForbidden, -1, "无权查看该账户", nil)
                        return
                }
        }
        utils.JSON(w, http.StatusOK, 0, "", detail)
}

// UpdateAccountStatus 更新账户状态
// PUT /api/v1/tokens/accounts/{id}/status
func (h *TokenHandler) UpdateAccountStatus(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPut {
                utils.JSON(w, http.StatusMethodNotAllowed, -1, "仅支持PUT请求", nil)
                return
        }
        accountID := extractTokenMiddleID(r.URL.Path, "/status")
        if accountID == "" {
                utils.JSON(w, http.StatusBadRequest, -1, "缺少账户ID", nil)
                return
        }
        var req struct {
                Status string `json:"status"`
        }
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
                utils.JSON(w, http.StatusBadRequest, -1, "请求体解析失败", nil)
                return
        }
        if err := h.tokenService.UpdateAccountStatus(r.Context(), accountID, req.Status); err != nil {
                handleTokenError(w, err)
                return
        }
        utils.JSON(w, http.StatusOK, 0, "状态更新成功", nil)
}

// ==================== 积分分配 ====================

// AllocateTokens 分配积分
// POST /api/v1/tokens/accounts/{id}/allocate
//
// region_admin 区域分配 batch：senior 与 region_admin 均须通过 tokenSourceAllowed 校验
// "来源账户可作分配来源"。admin 不校验（service 层父子关系兜底）。
func (h *TokenHandler) AllocateTokens(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
                utils.JSON(w, http.StatusMethodNotAllowed, -1, "仅支持POST请求", nil)
                return
        }
        fromAccountID := extractTokenMiddleID(r.URL.Path, "/allocate")
        if fromAccountID == "" {
                utils.JSON(w, http.StatusBadRequest, -1, "缺少来源账户ID", nil)
                return
        }
        claims, ok := middleware.GetClaims(r.Context())
        if !ok || claims == nil {
                utils.JSON(w, http.StatusUnauthorized, -1, "未认证", nil)
                return
        }

        // senior / region_admin：校验"来源账户可作分配来源"（admin 跳过，由 service 父子校验兜底）。
        if claims.Role == models.RoleSeniorOperator || claims.Role == models.RoleRegionAdmin {
                scope := h.resolveScope(r)
                fromAcc, accErr := h.tokenService.GetAccount(r.Context(), fromAccountID)
                if accErr != nil {
                        handleTokenError(w, accErr)
                        return
                }
                if scope != nil && scope.Blocked {
                        utils.JSON(w, http.StatusForbidden, -1, "无权从该账户分配积分", nil)
                        return
                }
                if !tokenSourceAllowed(scope, fromAcc.AccountType, fromAcc.OwnerID) {
                        utils.JSON(w, http.StatusForbidden, -1, "无权从该账户分配积分", nil)
                        return
                }
        }

        var req models.AllocateTokensRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
                utils.JSON(w, http.StatusBadRequest, -1, "请求体解析失败", nil)
                return
        }
        if err := h.tokenService.AllocateTokens(r.Context(), fromAccountID, &req, claims.UserID); err != nil {
                handleTokenError(w, err)
                return
        }
        utils.JSON(w, http.StatusOK, 0, "分配成功", nil)
}

// ListAllocations 查询分配记录（按角色范围过滤 + A excludeMonthly）
// GET /api/v1/tokens/allocations?from_account_id=&to_account_id=&exclude_monthly=&limit=&offset=
func (h *TokenHandler) ListAllocations(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodGet {
                utils.JSON(w, http.StatusMethodNotAllowed, -1, "仅支持GET请求", nil)
                return
        }
        q := r.URL.Query()
        fromID := q.Get("from_account_id")
        toID := q.Get("to_account_id")
        excludeMonthly := q.Get("exclude_monthly") == "true" // A：分配记录Tab 传 true，排除月度自充值
        limit, _ := strconv.Atoi(q.Get("limit"))
        offset, _ := strconv.Atoi(q.Get("offset"))

        scope := h.resolveScope(r)

        items, total, err := h.tokenService.ListAllocationsScoped(r.Context(), fromID, toID, excludeMonthly, scope, limit, offset)
        if err != nil {
                utils.JSON(w, http.StatusInternalServerError, -1, "查询失败", nil)
                return
        }
        resp := map[string]interface{}{
                "items": items,
                "total": total,
        }
        if scope != nil && scope.Blocked {
                resp["scope_blocked"] = true
                resp["scope_message"] = scope.BlockedReason
        }
        utils.JSON(w, http.StatusOK, 0, "", resp)
}

// ==================== 采购/充值 ====================

// PurchaseTokens 采购/充值积分
// POST /api/v1/tokens/purchases
func (h *TokenHandler) PurchaseTokens(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
                utils.JSON(w, http.StatusMethodNotAllowed, -1, "仅支持POST请求", nil)
                return
        }
        claims, ok := middleware.GetClaims(r.Context())
        if !ok || claims == nil {
                utils.JSON(w, http.StatusUnauthorized, -1, "未认证", nil)
                return
        }
        var req models.PurchaseTokensRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
                utils.JSON(w, http.StatusBadRequest, -1, "请求体解析失败", nil)
                return
        }
        if req.AccountID == "" {
                utils.JSON(w, http.StatusBadRequest, -1, "account_id不能为空", nil)
                return
        }
        if err := h.tokenService.PurchaseTokens(r.Context(), &req, claims.UserID); err != nil {
                handleTokenError(w, err)
                return
        }
        utils.JSON(w, http.StatusOK, 0, "充值成功", nil)
}

// ListPurchases 查询采购记录（按角色范围过滤）
// GET /api/v1/tokens/purchases?account_id=&limit=&offset=
//
// v172.2：接入 TokenScope，兑现"本人/本校采购记录"。
//   - admin           → 全部采购记录
//   - senior_operator → 本校账户的采购记录（未绑定学校时为空集）
//   - operator/viewer → 仅自己账户的采购记录
func (h *TokenHandler) ListPurchases(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodGet {
                utils.JSON(w, http.StatusMethodNotAllowed, -1, "仅支持GET请求", nil)
                return
        }
        q := r.URL.Query()
        accountID := q.Get("account_id")
        limit, _ := strconv.Atoi(q.Get("limit"))
        offset, _ := strconv.Atoi(q.Get("offset"))

        scope := h.resolveScope(r)

        items, total, err := h.tokenService.ListPurchasesScoped(r.Context(), accountID, scope, limit, offset)
        if err != nil {
                utils.JSON(w, http.StatusInternalServerError, -1, "查询失败", nil)
                return
        }
        resp := map[string]interface{}{
                "items": items,
                "total": total,
        }
        if scope != nil && scope.Blocked {
                resp["scope_blocked"] = true
                resp["scope_message"] = scope.BlockedReason
        }
        utils.JSON(w, http.StatusOK, 0, "", resp)
}

// ==================== 消费流水 ====================

// ListConsumptionLogs 查询消费流水（按角色范围过滤）
// GET /api/v1/tokens/consumption?account_id=&user_id=&scene_code=&limit=&offset=
func (h *TokenHandler) ListConsumptionLogs(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodGet {
                utils.JSON(w, http.StatusMethodNotAllowed, -1, "仅支持GET请求", nil)
                return
        }
        q := r.URL.Query()
        accountID := q.Get("account_id")
        userID := q.Get("user_id")
        sceneCode := q.Get("scene_code")
        limit, _ := strconv.Atoi(q.Get("limit"))
        offset, _ := strconv.Atoi(q.Get("offset"))

        scope := h.resolveScope(r)

        items, total, err := h.tokenService.ListConsumptionLogsScoped(r.Context(), accountID, userID, sceneCode, scope, limit, offset)
        if err != nil {
                utils.JSON(w, http.StatusInternalServerError, -1, "查询失败", nil)
                return
        }
        resp := map[string]interface{}{
                "items": items,
                "total": total,
        }
        if scope != nil && scope.Blocked {
                resp["scope_blocked"] = true
                resp["scope_message"] = scope.BlockedReason
        }
        utils.JSON(w, http.StatusOK, 0, "", resp)
}

// ==================== 我的积分 ====================

// GetMyTokenAccount 获取当前用户的积分账户
// GET /api/v1/tokens/my-account
func (h *TokenHandler) GetMyTokenAccount(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodGet {
                utils.JSON(w, http.StatusMethodNotAllowed, -1, "仅支持GET请求", nil)
                return
        }
        claims, ok := middleware.GetClaims(r.Context())
        if !ok || claims == nil {
                utils.JSON(w, http.StatusUnauthorized, -1, "未认证", nil)
                return
        }
        acc, err := h.tokenService.GetAccountByOwner(r.Context(), models.AccountTypePersonal, claims.UserID)
        if err != nil {
                if errors.Is(err, repository.ErrTokenAccountNotFound) {
                        utils.JSON(w, http.StatusOK, 0, "", map[string]interface{}{
                                "has_account": false,
                                "message":     "暂未开通积分账户",
                        })
                        return
                }
                utils.JSON(w, http.StatusInternalServerError, -1, "查询失败", nil)
                return
        }
        utils.JSON(w, http.StatusOK, 0, "", map[string]interface{}{
                "has_account":       true,
                "account":           acc,
                "available_balance": acc.Balance - acc.FrozenAmount,
        })
}

// ==================== 预警配置 ====================

// GetAlertConfig 获取预警配置
// GET /api/v1/tokens/accounts/{id}/alert-config
func (h *TokenHandler) GetAlertConfig(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodGet {
                utils.JSON(w, http.StatusMethodNotAllowed, -1, "仅支持GET请求", nil)
                return
        }
        accountID := extractTokenMiddleID(r.URL.Path, "/alert-config")
        if accountID == "" {
                utils.JSON(w, http.StatusBadRequest, -1, "缺少账户ID", nil)
                return
        }
        cfg, err := h.tokenService.GetAlertConfig(r.Context(), accountID)
        if err != nil {
                utils.JSON(w, http.StatusInternalServerError, -1, "查询失败", nil)
                return
        }
        utils.JSON(w, http.StatusOK, 0, "", cfg)
}

// UpdateAlertConfig 更新预警配置
// PUT /api/v1/tokens/accounts/{id}/alert-config
func (h *TokenHandler) UpdateAlertConfig(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPut {
                utils.JSON(w, http.StatusMethodNotAllowed, -1, "仅支持PUT请求", nil)
                return
        }
        accountID := extractTokenMiddleID(r.URL.Path, "/alert-config")
        if accountID == "" {
                utils.JSON(w, http.StatusBadRequest, -1, "缺少账户ID", nil)
                return
        }
        var req models.UpdateAlertConfigRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
                utils.JSON(w, http.StatusBadRequest, -1, "请求体解析失败", nil)
                return
        }
        if err := h.tokenService.UpdateAlertConfig(r.Context(), accountID, &req); err != nil {
                utils.JSON(w, http.StatusBadRequest, -1, err.Error(), nil)
                return
        }
        utils.JSON(w, http.StatusOK, 0, "更新成功", nil)
}

// ==================== 辅助函数 ====================

// tokenOwnerInScope 判断某账户 owner_id 是否在范围白名单内
// admin(OwnerIDs==nil) 恒为 true；空切片恒为 false；否则成员判定
func tokenOwnerInScope(scope *services.TokenScope, ownerID string) bool {
        if scope == nil {
                return false
        }
        if scope.OwnerIDs == nil {
                return true // admin
        }
        for _, id := range scope.OwnerIDs {
                if id == ownerID {
                        return true
                }
        }
        return false
}

// tokenSourceAllowed 判断某账户能否作为"分配来源"（region_admin 区域分配 batch 新增，统一口径）
//
// 判定规则（fail-closed）：
//   - scope==nil                 → false（未认证/异常，绝不放行）
//   - scope.IsAdmin（OwnerIDs nil）→ true（admin 任意账户可作来源，service 层父子兜底）
//   - region 账户                → 仅当 ownerID ∈ scope.AllowedRegionOwnerIDs 时 true
//                                  （即请求者自己管辖的区域账户；其余 region 一律 false，
//                                   堵跨区域、堵 senior 碰 region；列表/统计 SQL 仍排除 region）
//   - school/personal 账户       → 仅当 ownerID ∈ scope.OwnerIDs 时 true（复用 tokenOwnerInScope）
//
// 与列表/统计查询的关系：本函数只用于"分配来源"这一动作的鉴权，AllowedRegionOwnerIDs
// 不参与任何列表/统计 SQL，故 v172.1 "scoped 非admin 排除 region" 防线完整保留。
func tokenSourceAllowed(scope *services.TokenScope, accountType string, ownerID string) bool {
        if scope == nil {
                return false
        }
        if scope.IsAdmin {
                return true
        }
        if accountType == models.AccountTypeRegion {
                // 仅放行请求者自己管辖的区域账户作来源
                for _, id := range scope.AllowedRegionOwnerIDs {
                        if id == ownerID {
                                return true
                        }
                }
                return false
        }
        // school / personal：走常规 owner 白名单
        return tokenOwnerInScope(scope, ownerID)
}

// extractTokenPathID 从路径末尾提取ID
// /api/v1/tokens/accounts/{id} → id
func extractTokenPathID(path string) string {
        for i := len(path) - 1; i >= 0; i-- {
                if path[i] == '/' {
                        id := path[i+1:]
                        for len(id) > 0 && id[len(id)-1] == '/' {
                                id = id[:len(id)-1]
                        }
                        if len(id) > 0 {
                                return id
                        }
                }
        }
        return ""
}

// extractTokenMiddleID 从路径中提取中间的ID
// /api/v1/tokens/accounts/{id}/suffix → id
func extractTokenMiddleID(path string, suffix string) string {
        if len(path) >= len(suffix) && path[len(path)-len(suffix):] == suffix {
                path = path[:len(path)-len(suffix)]
        }
        return extractTokenPathID(path)
}

// handleTokenError 统一Token错误响应
func handleTokenError(w http.ResponseWriter, err error) {
        switch {
        case errors.Is(err, repository.ErrTokenAccountNotFound):
                utils.JSON(w, http.StatusNotFound, -1, "账户不存在", nil)
        case errors.Is(err, repository.ErrInsufficientBalance):
                utils.JSON(w, http.StatusBadRequest, -1, "积分余额不足", nil)
        case errors.Is(err, repository.ErrAccountSuspended):
                utils.JSON(w, http.StatusBadRequest, -1, "账户已冻结", nil)
        case errors.Is(err, repository.ErrDuplicateAccount):
                utils.JSON(w, http.StatusConflict, -1, "该实体已存在同类型账户", nil)
        case errors.Is(err, services.ErrTokenInvalidAmount):
                utils.JSON(w, http.StatusBadRequest, -1, "积分数量必须大于0", nil)
        case errors.Is(err, services.ErrTokenSelfAllocate):
                utils.JSON(w, http.StatusBadRequest, -1, "不能分配给自己", nil)
        case errors.Is(err, services.ErrTokenNotParentChild):
                utils.JSON(w, http.StatusBadRequest, -1, "只能向下级账户分配积分", nil)
        case errors.Is(err, services.ErrTokenAccountNotActive):
                utils.JSON(w, http.StatusBadRequest, -1, "账户不在活跃状态", nil)
        default:
                utils.JSON(w, http.StatusInternalServerError, -1, err.Error(), nil)
        }
}
