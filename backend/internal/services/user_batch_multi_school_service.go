package services

// user_batch_multi_school_service.go — 跨区域多校批量建用户业务逻辑（本次新增）
//
// ========================= 这是什么 / 为什么独立成文件 =========================
//
// 场景：admin 要把一个区域下【多所学校】的老师，汇总成一张 Excel 一次性导入。
//   一张表里每一行老师各自带 school_id（前端由"学校名→ID"反查填入），同一批跨多校。
//
// 与既有【单校批量】(user_batch_service.go 的 BatchCreateUsers) 的本质区别——三条：
//   1. 单校：整批一个 school_id；本文件：每行各自的 school_id（跨校）。
//   2. 单校：整批回滚（一行错全不建）；本文件：逐行成败（能建的建好、建不了的列出来）。
//   3. 单校：用户名重复直接判失败；本文件：重名【自动改名】(teacher01→teacher01_2…)并建成功，
//      回显实际用户名，由 admin 通知到人、老师自己再改（与系统操作员确认的"路线A"）。
//
// 故【完全独立新写】，不改、不复用 BatchCreateUsers——老的单校批量(admin 单校 / senior 本校)
//   继续好用，与本跨校路径并存互不影响。少量校验逻辑重复(几十行)属有意取舍，换取安全隔离。
//
// ========================= 核心实现决策（已与系统操作员逐条确认） =========================
//
// ① 逐行成败 = 每行一个独立小事务：
//    遍历每一行 → 各自 Begin 一个小事务 → 事务内"建用户(users) + 入校(school_members)" →
//    成功 Commit、失败 Rollback 这一行 → 继续下一行。
//    这是"能建的建好、建不了的跳过"的唯一正确做法（大事务一错全回滚 ≠ 逐行成败）。
//    性能：2000 人 = 2000 个小事务，bcrypt 哈希是耗时大头(每次几十ms)，整体约 1-2 分钟，可接受。
//
// ② 重名自动改名(路线A)：
//    对每行先拿 Excel 填的 username 当候选，CheckUsernameExists 探测：
//      - 不存在 → 直接用它；
//      - 存在   → 依次试 username_2、username_3…，第一个不存在的拿去建；
//      - 试到 _maxRenameTry(50) 仍全占 → 该行判失败"用户名冲突过多，请手动改名"(极端，几乎不触发)。
//    并发兜底：探测说"不存在"但真建时撞唯一约束(IsUniqueViolation) → 该行 Rollback + 判失败
//      "用户名已被占用"，不无限重试(简单可靠)。
//    结果每行回显 final_username：没改名=原名，改了名=xxx_N，并标 renamed=true 供前端高亮提示。
//    跨校撞名天然处理：A校teacher01建成teacher01,B校teacher01自动变teacher01_2(username全局唯一)。
//
// ③ school_id 有效性：一次性批量校验。
//    建之前把表里去重后的所有 school_id 传 repository.ListExistingActiveSchoolIDs 拿"有效集合"，
//    逐行用内存集合 O(1) 判定。挡住前端伪造/失效 school_id 往不存在学校塞人，且零逐行查库开销。
//
// ④ 上限 + 超时：
//    行数 > maxMultiSchoolRows(2000) → 整体拒绝(返回 error，handler 转 400)。
//    ctx 超时(handler 给 5 分钟) → 停止处理，已建成的保留，剩余未处理行记"超时未处理"。
//
// ========================= 返回语义 =========================
//   - 不像单校那样有"整批回滚"。本方法【总是】尽力逐行处理，返回 created + failures 两份明细。
//   - 返回的 error 仅用于"系统级/前置异常"(行数超限、role 非法、批量校验学校查库失败)——
//     这类情况一行都不建、直接整体失败由 handler 转 400/500。
//   - 进入逐行处理后，单行失败只进 failures，不返 error(error 恒为 nil)。
//
// B4 修复（跨校批量建的用户也需个人积分账户，否则其 AI 消费无痕，admin 看不到）：
//   每行 createOneUserInSchoolTx 成功(该行已独立 Commit)后，best-effort 调
//   ensurePersonalTokenAccount 为该用户补开余额 0 的个人积分账户（幂等、失败仅 Warn，
//   不影响该行"建成功"的判定，也不影响其它行）。逐行补比批量补更贴合本文件的逐行事务模型。

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

// ==================== 常量 ====================

const (
	// maxMultiSchoolRows 跨校批量单次最大行数(超时保护配套：再多一次性事务循环时间不可控)
	maxMultiSchoolRows = 2000
	// maxRenameTry 重名自动改名最多尝试的序号(xxx_2 ~ xxx_51)，兜底防极端死循环
	maxRenameTry = 50
)

// ==================== 请求 / 响应结构 ====================

// MultiSchoolUserItem 跨校批量的单个条目——每行【自带 school_id】(与单校 BatchUserItem 的区别)
type MultiSchoolUserItem struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
	SchoolID    string `json:"school_id"` // 该老师所属学校(前端由"学校名→ID"反查填入)
}

// MultiSchoolBatchRequest 跨校批量建用户请求
//   - Role  ：批次级统一角色(仅 operator/viewer，符合"角色批次"原则)
//   - Source：写入 school_members.source 的来源标记(handler 固定传跨校专用值)
//   - Users ：用户条目数组(每行自带 school_id)
type MultiSchoolBatchRequest struct {
	Role   string                `json:"role"`
	Source string                `json:"source"`
	Users  []MultiSchoolUserItem `json:"users"`
}

// MultiSchoolCreatedItem 成功建成的单行明细(含改名回显)
type MultiSchoolCreatedItem struct {
	Index            int    `json:"index"`             // 行号(1-based，对齐 admin 看到的表格行)
	OriginalUsername string `json:"original_username"` // Excel 里填的原始用户名
	FinalUsername    string `json:"final_username"`    // 实际建成的用户名(改名则为 xxx_N)
	Renamed          bool   `json:"renamed"`           // 是否发生了自动改名
	SchoolID         string `json:"school_id"`         // 入校的学校ID
	DisplayName      string `json:"display_name"`      // 教师姓名(供前端通知清单展示)
}

// MultiSchoolFailureItem 失败的单行明细
type MultiSchoolFailureItem struct {
	Index    int    `json:"index"`     // 行号(1-based)
	Username string `json:"username"`  // 该行原始用户名(去空格后)
	SchoolID string `json:"school_id"` // 该行学校ID(供 admin 定位是哪个学校的行有问题)
	Reason   string `json:"reason"`    // 失败原因(中文)
}

// MultiSchoolBatchResult 跨校批量结果
type MultiSchoolBatchResult struct {
	TotalCount   int                      `json:"total_count"`   // 请求条目总数
	CreatedCount int                      `json:"created_count"` // 成功建成数
	FailedCount  int                      `json:"failed_count"`  // 失败数
	Created      []MultiSchoolCreatedItem `json:"created"`       // 成功明细(含改名清单，admin 据此通知)
	Failures     []MultiSchoolFailureItem `json:"failures"`      // 失败明细
}

// ==================== 跨校批量建用户 ====================

// BatchCreateUsersMultiSchool 跨区域多校批量建用户（逐行成败 + 重名自动改名 + 超时保护）
//
// 流程：
//   1. 前置校验(任一不过 → 返回 error，一行不建)：
//      行数 1~2000；角色合法且在 operator/viewer 白名单。
//   2. 收集去重 school_id，一次性 ListExistingActiveSchoolIDs 拿有效集合。
//   3. 逐行处理(每行独立小事务)：
//      a. ctx 超时检查 → 超时则剩余行全部记"超时未处理"并结束；
//      b. 字段自检(用户名/姓名非空、密码≥6、school_id 非空且在有效集合内) → 不过则记 failure 跳过；
//      c. 重名探测改名 → 拿到一个可用 username(原名或 xxx_N)；
//      d. 开小事务建用户 + 入校 → 成功 Commit 记 created、失败 Rollback 记 failure；
//         成功后 B4：best-effort 为该用户补开个人积分账户。
//   4. 返回 created + failures 明细(error 恒为 nil，单行问题不抛 error)。
//
// 注意：本方法不做"批内用户名查重"——跨校汇总表里重名是常态(各校各自命名)，
//   重名交给"自动改名"处理，而非判失败。这是与单校批量(批内查重判失败)的关键差异。
func (s *UserService) BatchCreateUsersMultiSchool(ctx context.Context, req *MultiSchoolBatchRequest) (*MultiSchoolBatchResult, error) {
	total := len(req.Users)
	result := &MultiSchoolBatchResult{
		TotalCount: total,
		Created:    []MultiSchoolCreatedItem{},
		Failures:   []MultiSchoolFailureItem{},
	}

	// ---------- 1. 前置校验 ----------
	if total == 0 {
		return result, fmt.Errorf("用户列表为空")
	}
	if total > maxMultiSchoolRows {
		return result, fmt.Errorf("单次最多导入 %d 人，当前 %d 人，请分批导入", maxMultiSchoolRows, total)
	}
	if !models.IsValidRole(req.Role) {
		return result, fmt.Errorf("无效的批次角色: %s", req.Role)
	}
	if !models.IsSchoolAdminCreatableRole(req.Role) {
		return result, fmt.Errorf("仅可批量创建骨干教师(operator)或普通教师(viewer)账号")
	}

	// ---------- 2. 一次性批量校验 school_id 有效性 ----------
	// 收集去重的 school_id(只收非空)，一次查库拿"真实存在的 active 学校"集合
	idSet := make(map[string]struct{})
	for _, u := range req.Users {
		sid := strings.TrimSpace(u.SchoolID)
		if sid != "" {
			idSet[sid] = struct{}{}
		}
	}
	distinctIDs := make([]string, 0, len(idSet))
	for sid := range idSet {
		distinctIDs = append(distinctIDs, sid)
	}
	validSchools, err := repository.ListExistingActiveSchoolIDs(ctx, distinctIDs)
	if err != nil {
		// 批量校验学校查库失败属系统级异常，整体中止(尚未建任何人)
		return result, fmt.Errorf("校验学校列表失败: %w", err)
	}

	source := req.Source
	if source == "" {
		source = "admin_multi_school_batch_create"
	}

	// ---------- 3. 逐行处理(每行独立小事务) ----------
	for i, item := range req.Users {
		idx := i + 1 // 行号 1-based

		// 3a. 超时检查：ctx 被取消/超时 → 剩余所有行记"超时未处理"并结束
		if ctxErr := ctx.Err(); ctxErr != nil {
			for j := i; j < total; j++ {
				rest := req.Users[j]
				result.Failures = append(result.Failures, MultiSchoolFailureItem{
					Index:    j + 1,
					Username: strings.TrimSpace(rest.Username),
					SchoolID: strings.TrimSpace(rest.SchoolID),
					Reason:   "处理超时未完成(请将剩余未建成的老师单独再导一次)",
				})
			}
			userLog.Error("跨校批量建用户处理超时",
				"processed", i, "remaining", total-i, "error", ctxErr)
			break
		}

		username := strings.TrimSpace(item.Username)
		displayName := strings.TrimSpace(item.DisplayName)
		password := item.Password
		schoolID := strings.TrimSpace(item.SchoolID)

		// 3b. 字段自检
		if username == "" {
			result.Failures = append(result.Failures, MultiSchoolFailureItem{Index: idx, Username: username, SchoolID: schoolID, Reason: "用户名不能为空"})
			continue
		}
		if displayName == "" {
			result.Failures = append(result.Failures, MultiSchoolFailureItem{Index: idx, Username: username, SchoolID: schoolID, Reason: "教师姓名不能为空"})
			continue
		}
		if len(password) < 6 {
			result.Failures = append(result.Failures, MultiSchoolFailureItem{Index: idx, Username: username, SchoolID: schoolID, Reason: "初始密码至少6位"})
			continue
		}
		if schoolID == "" {
			result.Failures = append(result.Failures, MultiSchoolFailureItem{Index: idx, Username: username, SchoolID: schoolID, Reason: "未指定所属学校(请在表内选择学校)"})
			continue
		}
		if _, ok := validSchools[schoolID]; !ok {
			result.Failures = append(result.Failures, MultiSchoolFailureItem{Index: idx, Username: username, SchoolID: schoolID, Reason: "所属学校无效或已停用(请重新选择学校)"})
			continue
		}

		// 3c. 重名探测改名：拿到一个可用 username
		finalUsername, renamed, renameErr := s.resolveAvailableUsername(ctx, username)
		if renameErr != nil {
			result.Failures = append(result.Failures, MultiSchoolFailureItem{Index: idx, Username: username, SchoolID: schoolID, Reason: renameErr.Error()})
			continue
		}

		// 3d. 开小事务：建用户 + 入校
		passwordHash, hErr := utils.HashPassword(password)
		if hErr != nil {
			result.Failures = append(result.Failures, MultiSchoolFailureItem{Index: idx, Username: username, SchoolID: schoolID, Reason: "密码加密失败"})
			continue
		}

		user := &models.User{
			ID:           uuid.New().String(),
			Username:     finalUsername,
			DisplayName:  displayName,
			PasswordHash: passwordHash,
			Role:         req.Role,
			Status:       models.StatusActive,
		}

		if ok := s.createOneUserInSchoolTx(ctx, user, schoolID, source); !ok {
			// 事务内失败(含并发撞唯一约束) → 该行判失败，继续下一行(不影响别行)
			result.Failures = append(result.Failures, MultiSchoolFailureItem{Index: idx, Username: username, SchoolID: schoolID, Reason: "用户名已被占用或创建失败(请稍后单独重试该行)"})
			continue
		}

		// B4：该行已独立 Commit 成功 → best-effort 补开个人积分账户
		//   （幂等、失败仅 Warn；不影响该行"建成功"判定，也不影响其它行）
		ensurePersonalTokenAccount(ctx, user.ID, displayName)

		// 成功：记 created(含改名回显)
		result.Created = append(result.Created, MultiSchoolCreatedItem{
			Index:            idx,
			OriginalUsername: username,
			FinalUsername:    finalUsername,
			Renamed:          renamed,
			SchoolID:         schoolID,
			DisplayName:      displayName,
		})
	}

	result.CreatedCount = len(result.Created)
	result.FailedCount = len(result.Failures)
	userLog.Info("跨校批量建用户完成",
		"total", total, "created", result.CreatedCount, "failed", result.FailedCount,
		"role", req.Role, "source", source)
	return result, nil
}

// resolveAvailableUsername 重名探测改名：返回一个当前库中不存在的可用用户名。
//
// 逻辑：
//   - 先探测原始 base：不存在 → 返回 (base, false, nil)；
//   - 存在 → 依次试 base_2、base_3…base_(maxRenameTry+1)，第一个不存在的 → 返回 (该名, true, nil)；
//   - 全部占用 → 返回 error "用户名冲突过多，请手动改名"(极端情况)。
//
// 注意：这里只是"探测"，存在探测通过后到真正 INSERT 之间被并发抢占的窗口，
//   由 createOneUserInSchoolTx 的唯一约束兜底(撞约束则该行判失败)，故本函数不保证 100% 可建。
func (s *UserService) resolveAvailableUsername(ctx context.Context, base string) (string, bool, error) {
	// 先试原名
	exists, err := repository.CheckUsernameExists(ctx, base)
	if err != nil {
		return "", false, fmt.Errorf("用户名查重失败")
	}
	if !exists {
		return base, false, nil
	}
	// 原名被占，依次试 base_2 ... base_(maxRenameTry+1)
	for n := 2; n <= maxRenameTry+1; n++ {
		candidate := fmt.Sprintf("%s_%d", base, n)
		exists, err := repository.CheckUsernameExists(ctx, candidate)
		if err != nil {
			return "", false, fmt.Errorf("用户名查重失败")
		}
		if !exists {
			return candidate, true, nil
		}
	}
	return "", false, fmt.Errorf("用户名「%s」冲突过多，请手动改名后单独导入", base)
}

// createOneUserInSchoolTx 在一个独立小事务内"建用户 + 入校"，成功返回 true。
//
// 失败(含并发撞唯一约束 23505)即 Rollback 并返回 false——调用方据此把该行记入 failures，
// 绝不影响其它行(逐行成败的事务隔离单位就在这里)。
func (s *UserService) createOneUserInSchoolTx(ctx context.Context, user *models.User, schoolID, source string) bool {
	tx, err := database.DB.Begin(ctx)
	if err != nil {
		userLog.Error("跨校批量-开启单行事务失败", "username", user.Username, "error", err)
		return false
	}
	defer func() { _ = tx.Rollback(ctx) }() // 已 Commit 后再 Rollback 是 no-op，安全

	// 建用户(撞 users_username_key 唯一约束 → IsUniqueViolation → 视为该行失败)
	if cErr := repository.CreateUserTx(ctx, tx, user); cErr != nil {
		if repository.IsUniqueViolation(cErr) {
			userLog.Info("跨校批量-用户名并发撞约束，该行跳过", "username", user.Username)
		} else {
			userLog.Error("跨校批量-建用户失败", "username", user.Username, "error", cErr)
		}
		return false
	}

	// 入校
	if aErr := repository.AddSchoolMemberTx(ctx, tx, schoolID, user.ID, source); aErr != nil {
		userLog.Error("跨校批量-入校失败", "username", user.Username, "school_id", schoolID, "error", aErr)
		return false
	}

	// 提交
	if cErr := tx.Commit(ctx); cErr != nil {
		userLog.Error("跨校批量-提交单行事务失败", "username", user.Username, "error", cErr)
		return false
	}
	return true
}
