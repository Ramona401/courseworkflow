package services

// user_batch_service.go — 批量建用户业务逻辑（迭代一 Phase 3.2，P0-18）
//
// 设计要点(已与系统操作员确认)：
//   - 方案1 整批回滚：批内任一用户失败(重名/校验不过/DB错)，整批不创建，回滚到操作前状态，
//     返回精确到行号的失败原因，让操作员改完重传。最干净、对非开发操作员最不易出错。
//   - 方案X 批次级统一角色：整批共用一个 role(请求级)，每行只填 用户名/显示名/密码。
//     典型场景"导入一批同类教师"；需要混合角色时分两批导入即可。
//   - 与单建一致的事务化：整批包进【一个大事务】，每个用户在事务内"建 users + 入校"，
//     全成功才 Commit。这是 CreateUserWithSchool 单事务语义在批量场景的自然延伸。
//
// 命名约定(避免与 pipeline_review_batch.go 的 BatchCreateResult/BatchStartResult 等撞名)：
//   本文件类型一律带 "User(s)" 前缀——BatchCreateUsersRequest / BatchUserItem /
//   BatchCreateUserFailure / BatchCreateUsersResult，与 Pipeline 批量类型在同包内并存不冲突。
//
// 校验分两阶段(全部在写库前完成，尽早暴露问题、避免开了事务又回滚的浪费)：
//   阶段一(批内自检)：角色合法、条目非空、每行字段必填+密码长度、批内用户名不得重复。
//   阶段二(批与库比对)：逐个 CheckUsernameExists，与库中现有用户名查重。
//   两阶段任一发现问题 → 直接返回结果(成功数0 + 全部失败明细)，根本不开事务。
//
// 并发兜底(方案A)：阶段二查重通过后，事务内 INSERT 仍可能撞唯一约束(23505)——
//   此时整批回滚并把该行标记为"用户名已存在"。
//
// B4 修复（批量建的用户也需个人积分账户，否则其 AI 消费无痕，admin 看不到）：
//   大事务 Commit 成功后，收集本批所有新用户 ID，逐个 best-effort 调 ensurePersonalTokenAccount
//   补开余额 0 的个人积分账户（幂等、失败仅 Warn，不影响已成功的批量结果）。
//   放在 Commit 之后而非事务内——建账户失败绝不该回滚已建好的整批用户。

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"tedna/internal/database"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// ==================== 请求 / 响应结构 ====================

// BatchUserItem 批量建用户的单个条目(不含角色——角色在批次级统一指定)
type BatchUserItem struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
}

// BatchCreateUsersRequest 批量建用户请求
//   - Role     ：批次级统一角色，整批所有用户共用
//   - SchoolID ：批次级统一入校学校(空则只建用户不入校，与单建语义一致)
//   - Source   ：写入 school_members.source 的来源标记(仅 SchoolID 非空时有意义)
//   - Users    ：用户条目数组
type BatchCreateUsersRequest struct {
	Role     string          `json:"role"`
	SchoolID string          `json:"school_id"`
	Source   string          `json:"source"`
	Users    []BatchUserItem `json:"users"`
}

// BatchCreateUserFailure 单行失败明细(行号从1开始，对齐操作员看到的表格行)
type BatchCreateUserFailure struct {
	Index    int    `json:"index"`    // 行号(1-based)
	Username string `json:"username"` // 该行用户名(去空格后)
	Reason   string `json:"reason"`   // 失败原因(中文)
}

// BatchCreateUsersResult 批量建用户结果
//   - Success      ：整批是否全部成功并已提交(方案1：要么全成功，要么 false 且无任何创建)
//   - CreatedCount ：成功创建数(Success=true 时等于条目数；失败时恒为0)
//   - TotalCount   ：请求条目总数
//   - Failures     ：失败明细(Success=true 时为空数组)
type BatchCreateUsersResult struct {
	Success      bool                     `json:"success"`
	CreatedCount int                      `json:"created_count"`
	TotalCount   int                      `json:"total_count"`
	Failures     []BatchCreateUserFailure `json:"failures"`
}

// ==================== 批量建用户 ====================

// BatchCreateUsers 批量创建用户（整批回滚 + 批次级统一角色）
//
// 流程：
//   1. 批次级前置校验：用户列表非空、角色合法。
//   2. 阶段一 批内自检：逐行字段校验 + 批内用户名查重(本批内不得自相重复)。
//   3. 阶段二 与库比对：逐行 CheckUsernameExists。
//   4. 任一阶段有失败 → 不开事务，直接返回 Success=false + 全部失败明细。
//   5. 全部通过 → 开一个大事务，逐行"建 users + 入校"，全成功 Commit；
//      事务内任一行失败(含并发撞唯一约束) → 整批 Rollback，返回该行失败。
//   6. Commit 成功后 → B4：为本批所有新用户 best-effort 补开个人积分账户。
//
// 返回的 error 仅用于"系统级异常"(如开事务失败)；业务级失败(重名/校验)通过
// 结果的 Failures 表达，error 为 nil。这样 handler 能区分"系统炸了"与"数据有问题"。
func (s *UserService) BatchCreateUsers(ctx context.Context, req *BatchCreateUsersRequest) (*BatchCreateUsersResult, error) {
	total := len(req.Users)
	result := &BatchCreateUsersResult{
		Success:    false,
		TotalCount: total,
		Failures:   []BatchCreateUserFailure{},
	}

	// 1. 批次级前置校验
	if total == 0 {
		return result, fmt.Errorf("用户列表为空")
	}
	if !models.IsValidRole(req.Role) {
		return result, fmt.Errorf("无效的批次角色: %s", req.Role)
	}

	// 规范化每行字段(去空格)，并据此做两阶段查重；用 normalized 暂存修剪后的值供后续建用户复用
	type normItem struct {
		Username    string
		DisplayName string
		Password    string
	}
	normalized := make([]normItem, total)
	seenInBatch := make(map[string]int) // 批内用户名 → 首次出现行号(1-based)

	// 2. 阶段一：批内自检(字段 + 批内重名)
	for i, item := range req.Users {
		idx := i + 1
		u := strings.TrimSpace(item.Username)
		d := strings.TrimSpace(item.DisplayName)
		p := item.Password
		normalized[i] = normItem{Username: u, DisplayName: d, Password: p}

		if u == "" {
			result.Failures = append(result.Failures, BatchCreateUserFailure{Index: idx, Username: u, Reason: "用户名不能为空"})
			continue
		}
		if d == "" {
			result.Failures = append(result.Failures, BatchCreateUserFailure{Index: idx, Username: u, Reason: "显示名称不能为空"})
			continue
		}
		if len(p) < 6 {
			result.Failures = append(result.Failures, BatchCreateUserFailure{Index: idx, Username: u, Reason: "密码长度不能少于6位"})
			continue
		}
		if first, dup := seenInBatch[u]; dup {
			result.Failures = append(result.Failures, BatchCreateUserFailure{
				Index: idx, Username: u,
				Reason: fmt.Sprintf("本批内用户名重复(与第%d行相同)", first),
			})
			continue
		}
		seenInBatch[u] = idx
	}

	// 3. 阶段二：与库比对(仅对阶段一未失败的行做查重)
	//    用 failedIdx 标记阶段一已失败的行号，避免对它们重复查库/重复报错
	failedIdx := make(map[int]bool, len(result.Failures))
	for _, f := range result.Failures {
		failedIdx[f.Index] = true
	}
	for i := range normalized {
		idx := i + 1
		if failedIdx[idx] {
			continue
		}
		exists, err := repository.CheckUsernameExists(ctx, normalized[i].Username)
		if err != nil {
			// 查重本身出错视为系统级异常，整体中止(尚未开事务，无副作用)
			return result, fmt.Errorf("批量查重失败(第%d行 %s): %w", idx, normalized[i].Username, err)
		}
		if exists {
			result.Failures = append(result.Failures, BatchCreateUserFailure{
				Index: idx, Username: normalized[i].Username, Reason: "用户名已存在",
			})
		}
	}

	// 4. 任一阶段有失败 → 不开事务，整批不创建(方案1)
	if len(result.Failures) > 0 {
		userLog.Info("批量建用户因校验失败整批未创建",
			"total", total, "fail_count", len(result.Failures), "role", req.Role, "school_id", req.SchoolID)
		return result, nil
	}

	// 5. 全部通过 → 开一个大事务，逐行建用户+入校
	//    createdUserIDs 收集本批所有新用户 ID，供 Commit 后 B4 补开积分账户
	tx, err := database.DB.Begin(ctx)
	if err != nil {
		userLog.Error("批量建用户开启事务失败", "total", total, "error", err)
		return result, fmt.Errorf("开启批量事务失败: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	createdUserIDs := make([]string, 0, total)
	createdDisplayNames := make([]string, 0, total)

	for i := range normalized {
		idx := i + 1
		passwordHash, hErr := utils.HashPassword(normalized[i].Password)
		if hErr != nil {
			result.Failures = append(result.Failures, BatchCreateUserFailure{Index: idx, Username: normalized[i].Username, Reason: "密码加密失败"})
			return result, nil // 触发 defer 回滚，整批作废
		}

		user := &models.User{
			ID:           uuid.New().String(),
			Username:     normalized[i].Username,
			DisplayName:  normalized[i].DisplayName,
			PasswordHash: passwordHash,
			Role:         req.Role,
			Status:       models.StatusActive,
		}

		// 5a. 事务内建用户
		if cErr := repository.CreateUserTx(ctx, tx, user); cErr != nil {
			reason := "创建失败"
			if repository.IsUniqueViolation(cErr) {
				reason = "用户名已存在" // 并发兜底：查重后到建用户之间被别人抢先
			}
			result.Failures = append(result.Failures, BatchCreateUserFailure{Index: idx, Username: normalized[i].Username, Reason: reason})
			userLog.Error("批量建用户失败(整批将回滚)", "index", idx, "username", normalized[i].Username, "error", cErr)
			return result, nil // defer 回滚
		}

		// 5b. 事务内入校(仅当指定学校)
		if req.SchoolID != "" {
			if aErr := repository.AddSchoolMemberTx(ctx, tx, req.SchoolID, user.ID, req.Source); aErr != nil {
				result.Failures = append(result.Failures, BatchCreateUserFailure{Index: idx, Username: normalized[i].Username, Reason: "入校写入失败"})
				userLog.Error("批量入校失败(整批将回滚)", "index", idx, "username", normalized[i].Username, "school_id", req.SchoolID, "error", aErr)
				return result, nil // defer 回滚
			}
		}

		// 收集成功入事务的用户 ID/显示名（尚未 Commit，仅登记，待 Commit 成功后再补账户）
		createdUserIDs = append(createdUserIDs, user.ID)
		createdDisplayNames = append(createdDisplayNames, user.DisplayName)
	}

	// 6. 全部成功，提交
	if err := tx.Commit(ctx); err != nil {
		userLog.Error("批量建用户提交事务失败(整批将回滚)", "total", total, "error", err)
		return result, fmt.Errorf("提交批量事务失败: %w", err)
	}

	result.Success = true
	result.CreatedCount = total
	userLog.Info("批量建用户成功",
		"created", total, "role", req.Role, "school_id", req.SchoolID, "source", req.Source)

	// 7. B4：事务外 best-effort 为本批所有新用户补开个人积分账户
	//    （失败仅 Warn，绝不影响已成功的批量创建结果；ensurePersonalTokenAccount 定义于 user_service.go 同包）
	for i, uid := range createdUserIDs {
		name := ""
		if i < len(createdDisplayNames) {
			name = createdDisplayNames[i]
		}
		ensurePersonalTokenAccount(ctx, uid, name)
	}

	return result, nil
}
