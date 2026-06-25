package services

// token_service.go — Token积分系统核心业务逻辑
//
// v128 新增（阶段C · Token/积分系统）
// v129 改造（积分机制融合 · 对齐AOCI精确积分计算）
// v172 改造（积分管理三级数据权限隔离）：TokenScope + ResolveTokenScope + 各 Scoped 方法
// v172.2 新增（采购记录隔离）：ListPurchases 旧签名委托 nil；新增 ListPurchasesScoped
//
// 积分越权修复 新增：
//   ResolveTokenScope 的 senior_operator 分支，在取到本校成员后，剔除其中混入的
//   特权角色用户(admin/region_admin) —— 修复"学校管理员越权看到/分配系统管理员账户"。
//   根因：admin 因 migration/group_member 被登记进 school_members，ListSchoolMemberIDs
//   无差别返回其 user_id，使 admin 的个人账户落进 senior 白名单。
//   修法：用 repository.ListPrivilegedUserIDs 做差集，从 memberIDs 减掉特权用户，
//   再组装 OwnerIDs/UserIDs。admin 本身（admin 分支 IsAdmin=true）看全部，不受影响。
//
// 究极彻底版·批次1 新增（账户选择/搜索基础设施）：
//   - ListAccountsScoped 增加 keyword 参数，透传给 repository.ListTokenAccounts 做模糊搜索。
//   - ListAccounts 旧签名补一个空 keyword 委托，保持存量调用兼容（签名不变行为不变）。
//   - 新增 ListChildAccountsScoped：从 scope 取 OwnerIDs，调 repository.ListChildAccountsScoped
//     拿"某账户的合法下级"，供究极版分配弹窗使用（只显示合法下级，防越权）。
//
// 究极彻底版·A 新增（分配记录 total 精确）：
//   - ListAllocationsScoped 增加 excludeMonthly 参数，透传给 repository.ListTokenAllocations，
//     为 true 时排除月度自充值，使 items 与 total 一致（根治 P3 前端过滤的条数不一致）。
//   - ListAllocations 旧签名补 false 委托（保持兼容，不排除）。
//
// region_admin 区域分配 batch（高风险项，独立迭代，单独测越权）：
//   ResolveTokenScope 新增 region_admin 分支，赋予区域管理员"看管辖区域内学校账户、
//   并从自己的区域账户向这些学校账户分配积分"的能力。
//   设计要点（与 data_scope.go 已验证链路完全对齐）：
//     1. 复用 ListRegionIDsByAdmin（以 organization_admins 为管辖权威）查出管辖的区域；
//        每个区域再 ListDescendantSchoolIDs 递归取树下所有学校。
//     2. OwnerIDs（账户列表/概览/分配记录/采购 owner_id 白名单）= 管辖区域内所有学校的
//        owner_id（= 学校组织ID，与 senior 用 school.ID 同口径）。**不含成员 user_id**——
//        因为 region_admin 只分配到学校账户这一层（Q1 决策），不下到个人。
//     3. 新增字段 AllowedRegionOwnerIDs = 管辖的区域账户 owner_id（= 区域组织ID）。
//        专供"分配来源校验"放行 region_admin 自己的区域账户，而账户列表/概览统计的
//        SQL 一字不动仍排除 region —— 故 region_admin 在列表里看不到任何 region 账户，
//        仅在"选分配来源"这一动作里能用到自己的区域账户。这把 v172.1 的跨级泄漏防线
//        完整保留（外科手术式精准放行，详见 token_handler.tokenSourceAllowed）。
//
//   消费明细开放 新增（本次）：
//     原 region_admin 分支的 UserIDs 为空切片（消费流水查询收窄为空集，看不到任何消费）。
//     本次将 UserIDs 改为"管辖区域内所有学校的成员 user_id（去重，且剔除特权账户）"，
//     使区域管理员能查看辖区内所有老师的 AI 消费明细（消费流水 Tab 不再为空）。
//     - 成员来源：对管辖的每个学校调 ListSchoolMemberIDs 汇总去重（与 data_scope 同源链路，
//       天然只含本辖区学校成员，不跨区域，隔离性已验证）。
//     - 剔除特权：与 senior 分支一致，用 ListPrivilegedUserIDs 从成员里剔除 admin/region_admin，
//       防止某学校 school_members 混入 admin 时区域管理员越权看到特权账户消费。
//       fail-closed 策略同 senior：查特权名单失败时记 Warn 但继续用原成员（不瘫痪查询）。
//     - 注意：UserIDs 仅用于消费流水维度，不参与 OwnerIDs（账户/分配/采购仍只到学校一层），
//       两个白名单各自独立，开放消费明细不影响分配边界（仍 region→school 一层）。
//
// 权限范围设计（fail-closed，任何不确定一律收窄为空集，绝不放大为全量）：
//   - admin           → 看全部（白名单 nil）
//   - region_admin    → 管辖区域内学校账户 + 汇总统计；可从自己的区域账户向学校账户分配；
//                       可查辖区所有老师消费明细；
//                       OwnerIDs = 管辖学校 owner_id；AllowedRegionOwnerIDs = 管辖区域 owner_id；
//                       UserIDs = 辖区学校成员(剔除特权)；未任命/查询失败 → 空集
//   - senior_operator → 仅本校：账户/概览/采购 owner_id ∈ {本校成员user_id ∪ 学校组织ID}；
//                       消费流水 user_id ∈ 本校成员user_id；
//                       本校成员已剔除 admin/region_admin（防越权看上级账户）；
//                       未绑定学校 / 查询失败 → 空集（看不到任何数据 + 上层给提示）
//   - operator/viewer → 仅本人：账户/概览/采购 owner_id = 自己；消费流水 user_id = 自己
//
// 核心流程：采购 → 区域账户充值 → 分配到学校账户 → 分配到个人账户 → AI调用消费

import (
        "context"
        "errors"
        "fmt"
        "time"

        "tedna/internal/logger"
        "tedna/internal/models"
        "tedna/internal/repository"
)

// tokenLog 模块级结构化日志器
var tokenLog = logger.WithModule("services.token")

// ==================== 错误常量 ====================

var (
        ErrTokenInvalidAccountType = errors.New("无效的账户类型")
        ErrTokenInvalidAmount      = errors.New("积分数量必须大于0")
        ErrTokenSelfAllocate       = errors.New("不能分配给自己")
        ErrTokenNotParentChild     = errors.New("只能向下级账户分配积分")
        ErrTokenAccountNotActive   = errors.New("账户不在活跃状态")
)

// ==================== TokenService 结构体 ====================

// TokenService Token积分系统核心服务
type TokenService struct{}

// NewTokenService 创建TokenService实例
func NewTokenService() *TokenService {
        return &TokenService{}
}

// ==================== v172 数据权限范围（TokenScope）====================

// TokenScope 描述当前请求者对积分数据的可见范围
//
// 白名单字段语义（与 repository 层完全一致）：
//   - nil        → 不过滤（看全部，仅 admin）
//   - 非nil空切片 → 匹配空集（fail-closed，看不到任何数据）
//   - 非空        → 仅匹配名单内
//
// 查询维度复用两个白名单：
//   - OwnerIDs：账户列表 / 概览统计 / 分配记录 / 采购记录（按 token_accounts.owner_id）
//   - UserIDs ：消费流水（按 token_consumption_logs.user_id）
//
// region_admin 区域分配 batch 新增：
//   - AllowedRegionOwnerIDs：该请求者"可作为分配来源的区域账户" owner_id 白名单。
//     仅 region_admin 非空（= 其管辖的区域组织ID）。专供 token_handler.tokenSourceAllowed
//     在"分配来源校验"时放行 region_admin 自己的区域账户，**不参与任何列表/统计查询**，
//     从而完整保留 v172.1 "scoped 非admin 列表/统计排除 region" 的跨级泄漏防线。
//     语义：nil/空 → 没有任何区域账户可作来源；非空 → 仅名单内的区域账户可作来源。
type TokenScope struct {
        Role                  string   // 请求者角色
        IsAdmin               bool     // 是否系统管理员（看全部）
        OwnerIDs              []string // 账户/概览/分配/采购 owner_id 白名单
        UserIDs               []string // 消费流水 user_id 白名单
        AllowedRegionOwnerIDs []string // 可作分配来源的区域账户 owner_id 白名单（仅 region_admin 非空）
        Blocked               bool     // 是否被收窄为空集（如 senior 未绑定学校）
        BlockedReason         string   // 收窄原因（供上层提示）
}

// ResolveTokenScope 根据角色与用户ID解析数据可见范围（积分系统唯一权限决策点，fail-closed）
func (s *TokenService) ResolveTokenScope(ctx context.Context, role string, userID string) *TokenScope {
        if role == "" || userID == "" {
                return &TokenScope{
                        Role:          role,
                        IsAdmin:       false,
                        OwnerIDs:      []string{},
                        UserIDs:       []string{},
                        Blocked:       true,
                        BlockedReason: "未认证",
                }
        }

        switch role {
        case models.RoleAdmin:
                return &TokenScope{
                        Role:     role,
                        IsAdmin:  true,
                        OwnerIDs: nil,
                        UserIDs:  nil,
                }

        case models.RoleRegionAdmin:
                // region_admin 区域分配 batch：复用 data_scope 已验证链路。
                // 1. 查该用户管辖的所有区域（以 organization_admins.role_type='region_admin' 为权威）。
                regionIDs, rErr := repository.ListRegionIDsByAdmin(ctx, userID)
                if rErr != nil {
                        tokenLog.Warn("查询区域管理员管辖区域失败，收窄为空集", "user", userID, "error", rErr)
                        return &TokenScope{
                                Role:          role,
                                IsAdmin:       false,
                                OwnerIDs:      []string{},
                                UserIDs:       []string{},
                                Blocked:       true,
                                BlockedReason: "查询管辖区域失败",
                        }
                }
                if len(regionIDs) == 0 {
                        // 不是任何区域的管理员 → 收窄空集（fail-closed）
                        return &TokenScope{
                                Role:          role,
                                IsAdmin:       false,
                                OwnerIDs:      []string{},
                                UserIDs:       []string{},
                                Blocked:       true,
                                BlockedReason: "您尚未被任命为任何区域的管理员",
                        }
                }

                // 2. 对每个区域递归查出树下所有学校，汇总去重。
                //    学校组织ID 即学校账户的 owner_id（与 senior 用 school.ID 同口径）。
                schoolIDSet := make(map[string]struct{})
                for _, regionID := range regionIDs {
                        schoolIDs, sErr := repository.ListDescendantSchoolIDs(ctx, regionID)
                        if sErr != nil {
                                tokenLog.Warn("递归查询区域树下学校失败，收窄为空集", "region", regionID, "error", sErr)
                                return &TokenScope{
                                        Role:          role,
                                        IsAdmin:       false,
                                        OwnerIDs:      []string{},
                                        UserIDs:       []string{},
                                        Blocked:       true,
                                        BlockedReason: "查询区域树下学校失败",
                                }
                        }
                        for _, sid := range schoolIDs {
                                schoolIDSet[sid] = struct{}{}
                        }
                }

                // 3. OwnerIDs = 管辖区域内所有学校的 owner_id（学校组织ID）。
                //    注意：不含成员 user_id（Q1 决策——只分配到学校一层，不下到个人）；
                //    也不含区域 owner_id（区域账户由 AllowedRegionOwnerIDs 单独承载，
                //    列表/统计 SQL 仍排除 region，故区域账户不会出现在账户列表/概览中）。
                ownerIDs := make([]string, 0, len(schoolIDSet))
                for sid := range schoolIDSet {
                        ownerIDs = append(ownerIDs, sid)
                }

                // 4. AllowedRegionOwnerIDs = 管辖的区域账户 owner_id（区域组织ID）。
                //    专供分配来源校验放行，不参与列表/统计。
                allowedRegionOwnerIDs := make([]string, 0, len(regionIDs))
                allowedRegionOwnerIDs = append(allowedRegionOwnerIDs, regionIDs...)

                // 5. 消费明细开放：UserIDs = 管辖区域内所有学校的成员 user_id（去重 + 剔除特权）。
                //    使区域管理员能查看辖区内所有老师的 AI 消费明细（消费流水维度）。
                //    成员来源与 OwnerIDs 同源（schoolIDSet），天然只含本辖区学校成员不跨区域。
                memberIDSet := make(map[string]struct{})
                for sid := range schoolIDSet {
                        memberIDs, mErr := repository.ListSchoolMemberIDs(ctx, sid)
                        if mErr != nil {
                                tokenLog.Warn("查询区域树下学校成员失败，该校成员本轮跳过（消费明细不含该校）",
                                        "school", sid, "error", mErr)
                                continue // 单校失败不整体瘫痪，跳过该校成员继续汇总
                        }
                        for _, uid := range memberIDs {
                                memberIDSet[uid] = struct{}{}
                        }
                }

                // 剔除混入的特权账户(admin/region_admin)，与 senior 分支一致。
                // fail-closed 策略同 senior：查特权名单失败记 Warn 但继续用原成员（不瘫痪查询）。
                privilegedIDs, pErr := repository.ListPrivilegedUserIDs(ctx)
                if pErr != nil {
                        tokenLog.Warn("查询特权用户列表失败，区域消费明细本轮不剔除（继续用原成员列表）",
                                "user", userID, "error", pErr)
                } else if len(privilegedIDs) > 0 {
                        for _, id := range privilegedIDs {
                                delete(memberIDSet, id) // 从成员集合中移除特权用户
                        }
                }

                userIDs := make([]string, 0, len(memberIDSet))
                for uid := range memberIDSet {
                        userIDs = append(userIDs, uid)
                }

                return &TokenScope{
                        Role:                  role,
                        IsAdmin:               false,
                        OwnerIDs:              ownerIDs,
                        UserIDs:               userIDs,
                        AllowedRegionOwnerIDs: allowedRegionOwnerIDs,
                }

        case models.RoleSeniorOperator:
                school, err := repository.GetSchoolByAdminUserID(ctx, userID)
                if err != nil || school == nil || school.ID == "" {
                        return &TokenScope{
                                Role:          role,
                                IsAdmin:       false,
                                OwnerIDs:      []string{},
                                UserIDs:       []string{},
                                Blocked:       true,
                                BlockedReason: "您尚未绑定学校，请联系系统管理员",
                        }
                }
                memberIDs, mErr := repository.ListSchoolMemberIDs(ctx, school.ID)
                if mErr != nil {
                        tokenLog.Warn("查询本校成员失败，收窄为空集", "school", school.ID, "error", mErr)
                        return &TokenScope{
                                Role:          role,
                                IsAdmin:       false,
                                OwnerIDs:      []string{},
                                UserIDs:       []string{},
                                Blocked:       true,
                                BlockedReason: "查询本校成员失败",
                        }
                }

                // 积分越权修复：剔除混入本校成员的特权角色用户(admin/region_admin)。
                // 根因——admin 因 migration/group_member 被登记进 school_members，
                // 若不剔除，admin 的个人账户会落进 senior 白名单导致越权（看到/分配系统管理员账户）。
                // fail-closed 策略（选项甲）：查特权名单失败时记 Warn 但继续用原 memberIDs，
                //   不让 senior 整体瘫痪（该查询极简单几乎不会失败，且账户列表层另有
                //   account_type<>'region' 等防线）。安全要求绝对时可改为收窄空集。
                privilegedIDs, pErr := repository.ListPrivilegedUserIDs(ctx)
                if pErr != nil {
                        tokenLog.Warn("查询特权用户列表失败，本轮不剔除（继续用原成员列表）",
                                "school", school.ID, "error", pErr)
                } else if len(privilegedIDs) > 0 {
                        privilegedSet := make(map[string]struct{}, len(privilegedIDs))
                        for _, id := range privilegedIDs {
                                privilegedSet[id] = struct{}{}
                        }
                        filtered := make([]string, 0, len(memberIDs))
                        for _, id := range memberIDs {
                                if _, isPrivileged := privilegedSet[id]; isPrivileged {
                                        continue // 跳过特权用户，不纳入 senior 可见范围
                                }
                                filtered = append(filtered, id)
                        }
                        memberIDs = filtered
                }

                ownerIDs := make([]string, 0, len(memberIDs)+1)
                ownerIDs = append(ownerIDs, memberIDs...)
                ownerIDs = append(ownerIDs, school.ID)
                userIDs := make([]string, 0, len(memberIDs))
                userIDs = append(userIDs, memberIDs...)

                return &TokenScope{
                        Role:     role,
                        IsAdmin:  false,
                        OwnerIDs: ownerIDs,
                        UserIDs:  userIDs,
                }

        default:
                return &TokenScope{
                        Role:     role,
                        IsAdmin:  false,
                        OwnerIDs: []string{userID},
                        UserIDs:  []string{userID},
                }
        }
}

// ==================== 账户管理 ====================

// CreateAccount 创建积分账户
func (s *TokenService) CreateAccount(ctx context.Context, req *models.CreateTokenAccountRequest) (*models.TokenAccount, error) {
        if req.AccountType != models.AccountTypeRegion &&
                req.AccountType != models.AccountTypeSchool &&
                req.AccountType != models.AccountTypePersonal {
                return nil, ErrTokenInvalidAccountType
        }

        if req.DisplayName == "" {
                return nil, fmt.Errorf("账户名称不能为空")
        }

        acc := &models.TokenAccount{
                AccountType:     req.AccountType,
                OwnerID:         req.OwnerID,
                ParentAccountID: req.ParentAccountID,
                DisplayName:     req.DisplayName,
                Balance:         0,
                FrozenAmount:    0,
                TotalConsumed:   0,
                TotalQuota:      0,
                MonthlyQuota:    req.MonthlyQuota,
                Status:          models.AccountStatusActive,
        }

        if err := repository.CreateTokenAccount(ctx, acc); err != nil {
                if errors.Is(err, repository.ErrDuplicateAccount) {
                        return nil, err
                }
                return nil, fmt.Errorf("创建账户失败: %w", err)
        }

        tokenLog.Info("创建账户成功",
                "type", acc.AccountType,
                "owner", acc.OwnerID,
                "name", acc.DisplayName)
        return acc, nil
}

// GetAccount 获取账户详情（含预警配置和子账户）
func (s *TokenService) GetAccount(ctx context.Context, accountID string) (*models.TokenAccountDetail, error) {
        acc, err := repository.GetTokenAccountByID(ctx, accountID)
        if err != nil {
                return nil, err
        }

        detail := &models.TokenAccountDetail{
                TokenAccount:     *acc,
                AccountTypeName:  models.AccountTypeNameMap[acc.AccountType],
                StatusName:       models.AccountStatusNameMap[acc.Status],
                AvailableBalance: acc.Balance - acc.FrozenAmount,
        }
        if acc.TotalQuota > 0 {
                detail.UsagePercent = float64(acc.TotalConsumed) * 100.0 / float64(acc.TotalQuota)
        }

        alertCfg, _ := repository.GetTokenAlertConfig(ctx, accountID)
        detail.AlertConfig = alertCfg

        children, _ := repository.ListChildAccounts(ctx, accountID)
        detail.ChildAccounts = children

        return detail, nil
}

// GetAccountByOwner 根据实体类型+实体ID获取账户
func (s *TokenService) GetAccountByOwner(ctx context.Context, accountType string, ownerID string) (*models.TokenAccount, error) {
        return repository.GetTokenAccountByOwner(ctx, accountType, ownerID)
}

// ListAccounts 查询账户列表（旧签名，无范围限制，委托 nil 白名单；保留兼容存量调用）
// 批次1：补一个空 keyword 透传，签名保持不变，行为不变。
func (s *TokenService) ListAccounts(ctx context.Context, accountType string, parentAccountID string, status string, limit int, offset int) ([]*models.TokenAccountListItem, int, error) {
        return repository.ListTokenAccounts(ctx, accountType, parentAccountID, status, "", nil, limit, offset)
}

// ListAccountsScoped 查询账户列表（v172 范围感知 + 批次1 keyword 模糊搜索）
func (s *TokenService) ListAccountsScoped(ctx context.Context, accountType string, parentAccountID string, status string, keyword string, scope *TokenScope, limit int, offset int) ([]*models.TokenAccountListItem, int, error) {
        var ownerIDs []string
        if scope != nil {
                ownerIDs = scope.OwnerIDs
        }
        return repository.ListTokenAccounts(ctx, accountType, parentAccountID, status, keyword, ownerIDs, limit, offset)
}

// ListChildAccountsScoped 查询某账户的合法下级（批次1 新增，究极版分配弹窗用）
//
// 从 scope 取 OwnerIDs 作为白名单，调 repository.ListChildAccountsScoped：
//   - admin（OwnerIDs=nil）→ 返回该账户全部下级
//   - 非admin → 下级还须在请求者范围内（fail-closed）
// 调用前 handler 须先校验"来源账户"在 scope 内（见 GetAllocatableTargets），双重防越权。
//
// region_admin 适配说明：region_admin 的 OwnerIDs 装的正是管辖学校的 owner_id，
//   故"region 账户的下级（学校账户）"会被该白名单正确收窄——只列出管辖区域内的学校，
//   非管辖学校即便是该区域账户的下级也不会列出（理论上不会跨区域，双保险）。
func (s *TokenService) ListChildAccountsScoped(ctx context.Context, parentAccountID string, scope *TokenScope) ([]*models.TokenAccountListItem, error) {
        var ownerIDs []string
        if scope != nil {
                ownerIDs = scope.OwnerIDs
        }
        return repository.ListChildAccountsScoped(ctx, parentAccountID, ownerIDs)
}

// UpdateAccountStatus 更新账户状态
func (s *TokenService) UpdateAccountStatus(ctx context.Context, accountID string, status string) error {
        if status != models.AccountStatusActive &&
                status != models.AccountStatusSuspended &&
                status != models.AccountStatusExpired {
                return fmt.Errorf("无效的账户状态: %s", status)
        }
        return repository.UpdateTokenAccountStatus(ctx, accountID, status)
}

// GetSchoolAccountByAdmin 根据学校管理员用户ID查找本校积分账户（保留兼容）
func (s *TokenService) GetSchoolAccountByAdmin(ctx context.Context, adminUserID string) (*models.TokenAccount, error) {
        school, err := repository.GetSchoolByAdminUserID(ctx, adminUserID)
        if err != nil {
                return nil, err
        }
        return repository.GetTokenAccountByOwner(ctx, models.AccountTypeSchool, school.ID)
}

// ==================== 积分分配 ====================

// AllocateTokens 从上级账户分配积分到下级账户
func (s *TokenService) AllocateTokens(ctx context.Context, fromAccountID string, req *models.AllocateTokensRequest, operatorID string) error {
        if req.Amount <= 0 {
                return ErrTokenInvalidAmount
        }
        if fromAccountID == req.ToAccountID {
                return ErrTokenSelfAllocate
        }

        fromAcc, err := repository.GetTokenAccountByID(ctx, fromAccountID)
        if err != nil {
                return fmt.Errorf("来源账户不存在: %w", err)
        }
        if fromAcc.Status != models.AccountStatusActive {
                return ErrTokenAccountNotActive
        }

        toAcc, err := repository.GetTokenAccountByID(ctx, req.ToAccountID)
        if err != nil {
                return fmt.Errorf("目标账户不存在: %w", err)
        }
        if toAcc.Status != models.AccountStatusActive {
                return ErrTokenAccountNotActive
        }

        validRelation := false
        if toAcc.ParentAccountID != nil && *toAcc.ParentAccountID == fromAccountID {
                validRelation = true
        }
        if !validRelation && toAcc.ParentAccountID != nil {
                parentAcc, parentErr := repository.GetTokenAccountByID(ctx, *toAcc.ParentAccountID)
                if parentErr == nil && parentAcc.ParentAccountID != nil && *parentAcc.ParentAccountID == fromAccountID {
                        validRelation = true
                }
        }
        if !validRelation {
                return ErrTokenNotParentChild
        }

        if err := repository.DeductBalanceForAllocation(ctx, fromAccountID, req.Amount); err != nil {
                return err
        }

        if err := repository.AddBalance(ctx, req.ToAccountID, req.Amount); err != nil {
                _ = repository.AddBalance(ctx, fromAccountID, req.Amount)
                return fmt.Errorf("增加下级余额失败: %w", err)
        }

        alloc := &models.TokenAllocation{
                FromAccountID:  fromAccountID,
                ToAccountID:    req.ToAccountID,
                Amount:         req.Amount,
                AllocationType: models.AllocationTypeManual,
                Memo:           req.Memo,
                OperatorID:     operatorID,
        }
        if err := repository.CreateTokenAllocation(ctx, alloc); err != nil {
                tokenLog.Warn("分配成功但记录流水失败",
                        "from", fromAccountID,
                        "to", req.ToAccountID,
                        "amount", req.Amount,
                        "error", err)
        }

        tokenLog.Info("分配成功",
                "from", fromAccountID,
                "to", req.ToAccountID,
                "amount", req.Amount,
                "operator", operatorID)
        return nil
}

// ListAllocations 查询分配记录（旧签名，无范围限制，不排除 monthly；保留兼容）
func (s *TokenService) ListAllocations(ctx context.Context, fromAccountID string, toAccountID string, limit int, offset int) ([]*models.AllocationListItem, int, error) {
        return repository.ListTokenAllocations(ctx, fromAccountID, toAccountID, false, nil, limit, offset)
}

// ListAllocationsScoped 查询分配记录（v172 范围感知 + A excludeMonthly）
func (s *TokenService) ListAllocationsScoped(ctx context.Context, fromAccountID string, toAccountID string, excludeMonthly bool, scope *TokenScope, limit int, offset int) ([]*models.AllocationListItem, int, error) {
        var ownerIDs []string
        if scope != nil {
                ownerIDs = scope.OwnerIDs
        }
        return repository.ListTokenAllocations(ctx, fromAccountID, toAccountID, excludeMonthly, ownerIDs, limit, offset)
}

// ==================== 采购/充值 ====================

// PurchaseTokens 采购/充值积分
func (s *TokenService) PurchaseTokens(ctx context.Context, req *models.PurchaseTokensRequest, operatorID string) error {
        if req.Amount <= 0 {
                return ErrTokenInvalidAmount
        }

        acc, err := repository.GetTokenAccountByID(ctx, req.AccountID)
        if err != nil {
                return fmt.Errorf("目标账户不存在: %w", err)
        }
        if acc.Status != models.AccountStatusActive {
                return ErrTokenAccountNotActive
        }

        var validUntil *time.Time
        if req.ValidUntil != nil && *req.ValidUntil != "" {
                t, parseErr := time.Parse(time.RFC3339, *req.ValidUntil)
                if parseErr == nil {
                        validUntil = &t
                }
        }

        if err := repository.AddBalance(ctx, req.AccountID, req.Amount); err != nil {
                return err
        }

        purchase := &models.TokenPurchase{
                AccountID:    req.AccountID,
                Amount:       req.Amount,
                PurchaseType: req.PurchaseType,
                OrderNo:      req.OrderNo,
                Memo:         req.Memo,
                OperatorID:   operatorID,
                ValidUntil:   validUntil,
        }
        if err := repository.CreateTokenPurchase(ctx, purchase); err != nil {
                tokenLog.Warn("充值成功但记录采购流水失败",
                        "account", req.AccountID,
                        "amount", req.Amount,
                        "error", err)
        }

        tokenLog.Info("充值成功",
                "account", req.AccountID,
                "amount", req.Amount,
                "type", req.PurchaseType,
                "operator", operatorID)
        return nil
}

// ListPurchases 查询采购记录（旧签名，无范围限制，委托 nil 白名单；保留兼容）
func (s *TokenService) ListPurchases(ctx context.Context, accountID string, limit int, offset int) ([]*models.PurchaseListItem, int, error) {
        return repository.ListTokenPurchases(ctx, accountID, nil, limit, offset)
}

// ListPurchasesScoped 查询采购记录（v172.2 范围感知，按目标账户 owner 白名单）
func (s *TokenService) ListPurchasesScoped(ctx context.Context, accountID string, scope *TokenScope, limit int, offset int) ([]*models.PurchaseListItem, int, error) {
        var ownerIDs []string
        if scope != nil {
                ownerIDs = scope.OwnerIDs
        }
        return repository.ListTokenPurchases(ctx, accountID, ownerIDs, limit, offset)
}

// ==================== 消费流程（AI调用时使用）====================

// ConsumeTokens 消费积分（AI调用完成后调用）
func (s *TokenService) ConsumeTokens(ctx context.Context, req *models.TokenConsumeRequest) error {
        var creditCost float64
        var calc *models.CreditCalculation

        if req.Calculation != nil && req.Calculation.CreditsConsumed > 0 {
                creditCost = req.Calculation.CreditsConsumed
                calc = req.Calculation
        } else if req.TokensUsed > 0 {
                creditCost = float64(req.TokensUsed) / 1000.0
                if creditCost <= 0 {
                        return nil
                }
        } else {
                return nil
        }

        acc, err := repository.GetTokenAccountByOwner(ctx, models.AccountTypePersonal, req.UserID)
        if err != nil {
                if errors.Is(err, repository.ErrTokenAccountNotFound) {
                        return nil
                }
                return fmt.Errorf("查询用户积分账户失败: %w", err)
        }

        if acc.Status != models.AccountStatusActive {
                return nil
        }

        balanceBefore := acc.Balance

        if err := repository.DirectDeductBalance(ctx, acc.ID, creditCost); err != nil {
                tokenLog.Warn("消费扣减失败(不阻断AI)",
                        "account", acc.ID,
                        "amount", creditCost,
                        "error", err)
                return nil
        }

        consumeLog := &models.TokenConsumptionLog{
                AccountID:     acc.ID,
                UserID:        req.UserID,
                Amount:        creditCost,
                BalanceBefore: balanceBefore,
                BalanceAfter:  balanceBefore - creditCost,
                SceneCode:     req.SceneCode,
                ModelUsed:     req.ModelUsed,
                TokensUsed:    req.TokensUsed,
                LessonPlanID:  req.LessonPlanID,
                PipelineID:    req.PipelineID,
        }
        if calc != nil {
                consumeLog.InputTokens = calc.InputTokens
                consumeLog.OutputTokens = calc.OutputTokens
                consumeLog.ModelName = calc.ModelName
                consumeLog.Provider = calc.Provider
                consumeLog.CostUSD = calc.CostUSD
                consumeLog.ExchangeRate = calc.ExchangeRate
                consumeLog.Multiplier = calc.Multiplier
                consumeLog.CreditsConsumed = calc.CreditsConsumed
                consumeLog.LatencyMs = int(calc.LatencyMs)
        }
        if err := repository.CreateTokenConsumptionLog(ctx, consumeLog); err != nil {
                tokenLog.Warn("记录消费流水失败",
                        "account", acc.ID,
                        "amount", creditCost,
                        "error", err)
        }

        return nil
}

// ListConsumptionLogs 查询消费流水（旧签名，无范围限制；保留兼容）
func (s *TokenService) ListConsumptionLogs(ctx context.Context, accountID string, userID string, sceneCode string, limit int, offset int) ([]*models.ConsumptionListItem, int, error) {
        return repository.ListTokenConsumptionLogs(ctx, accountID, userID, sceneCode, nil, limit, offset)
}

// ListConsumptionLogsScoped 查询消费流水（v172 范围感知，按 user_id 白名单）
func (s *TokenService) ListConsumptionLogsScoped(ctx context.Context, accountID string, userID string, sceneCode string, scope *TokenScope, limit int, offset int) ([]*models.ConsumptionListItem, int, error) {
        var userIDs []string
        if scope != nil {
                userIDs = scope.UserIDs
        }
        return repository.ListTokenConsumptionLogs(ctx, accountID, userID, sceneCode, userIDs, limit, offset)
}

// ==================== 统计 ====================

// GetOverviewStats 获取概览统计（旧签名，全系统；保留兼容）
func (s *TokenService) GetOverviewStats(ctx context.Context) (*models.TokenOverviewStats, error) {
        return repository.GetTokenOverviewStatsScoped(ctx, nil)
}

// GetOverviewStatsScoped 获取概览统计（v172 范围感知，按 owner_id 白名单）
func (s *TokenService) GetOverviewStatsScoped(ctx context.Context, scope *TokenScope) (*models.TokenOverviewStats, error) {
        var ownerIDs []string
        if scope != nil {
                ownerIDs = scope.OwnerIDs
        }
        return repository.GetTokenOverviewStatsScoped(ctx, ownerIDs)
}

// ==================== 预警配置 ====================

// GetAlertConfig 获取预警配置
func (s *TokenService) GetAlertConfig(ctx context.Context, accountID string) (*models.TokenAlertConfig, error) {
        return repository.GetTokenAlertConfig(ctx, accountID)
}

// UpdateAlertConfig 更新预警配置
func (s *TokenService) UpdateAlertConfig(ctx context.Context, accountID string, req *models.UpdateAlertConfigRequest) error {
        if req.WarnThreshold <= 0 || req.WarnThreshold > 100 {
                return fmt.Errorf("预警阈值必须在1-100之间")
        }
        if req.UrgentThreshold <= 0 || req.UrgentThreshold > 100 {
                return fmt.Errorf("紧急阈值必须在1-100之间")
        }
        if req.UrgentThreshold <= req.WarnThreshold {
                return fmt.Errorf("紧急阈值必须大于预警阈值")
        }
        return repository.UpsertTokenAlertConfig(ctx, accountID, req)
}
