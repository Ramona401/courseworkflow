package repository

// audit_repo.go — 审计日志数据访问层
//
// 写入 audit_logs 表，记录关键业务操作。
// v52强化：支持用户名模糊搜索 + 日期范围过滤 + 分页多条件查询。

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"tedna/internal/database"
	"tedna/internal/logger"
)

var auditLog = logger.WithModule("audit")

// ==================== 操作类型常量 ====================

const (
	ActionLogin           = "user.login"
	ActionLogout          = "user.logout"
	ActionSubmitFinalize  = "pipeline.submit_finalize"
	ActionConfirmFinalize = "pipeline.confirm_finalize"
	ActionRejectFinalize  = "pipeline.reject_finalize"
	ActionDirectFinalize  = "pipeline.direct_finalize"
	ActionMarkPassed      = "pipeline.mark_passed"
	ActionVerify          = "pipeline.verify"
)

// ==================== 写入 ====================

// WriteAuditLog 异步写入审计日志，永不阻塞主流程
func WriteAuditLog(userID, action string, detail map[string]interface{}, ip string) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		detailJSON, err := json.Marshal(detail)
		if err != nil {
			auditLog.Error("审计日志序列化失败", "user_id", userID, "action", action, "error", err)
			return
		}

		_, err = database.DB.Exec(ctx, `
			INSERT INTO audit_logs (user_id, action, detail, ip)
			VALUES ($1, $2, $3, $4)
		`, userID, action, string(detailJSON), ip)

		if err != nil {
			auditLog.Error("审计日志写入失败", "user_id", userID, "action", action, "error", err)
		}
	}()
}

// ==================== 查询（供管理中心使用）====================

// AuditLogItem 审计日志列表项
type AuditLogItem struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	Action      string    `json:"action"`
	ActionName  string    `json:"action_name"` // 操作中文名
	Detail      string    `json:"detail"`      // JSONB原始字符串
	IP          string    `json:"ip"`
	CreatedAt   time.Time `json:"created_at"`
}

// AuditLogListResult 审计日志分页结果
type AuditLogListResult struct {
	Logs  []AuditLogItem `json:"logs"`
	Total int            `json:"total"`
}

// AuditLogQueryParams 审计日志查询参数（v52新增结构体，支持更多过滤维度）
type AuditLogQueryParams struct {
	UserID    string // 按用户ID精确过滤（旧接口向后兼容）
	Username  string // 按用户名/显示名模糊搜索（新增）
	Action    string // 按操作类型精确过滤
	StartDate string // 开始日期，格式 yyyy-MM-dd，含当天（新增）
	EndDate   string // 结束日期，格式 yyyy-MM-dd，含当天（新增）
	Page      int
	PageSize  int
}

// actionNameMap 操作类型→中文名映射
var actionNameMap = map[string]string{
	"user.login":                "用户登录",
	"user.logout":               "用户登出",
	"pipeline.submit_finalize":  "提交定稿",
	"pipeline.confirm_finalize": "确认定稿",
	"pipeline.reject_finalize":  "退回重审",
	"pipeline.direct_finalize":  "直接定稿",
	"pipeline.mark_passed":      "快捷通过",
	"pipeline.verify":           "触发验收",
	"admin.user_create":         "创建用户",
	"admin.user_status":         "状态变更",
	"admin.user_reset_password": "重置密码",
}

// ListAuditLogs 分页查询审计日志（支持多维过滤）
//
//   - params.UserID    非空则精确匹配 user_id（旧逻辑）
//   - params.Username  非空则对 username/display_name 做 ILIKE 模糊搜索
//   - params.Action    非空则精确匹配 action
//   - params.StartDate 非空则过滤 created_at >= 当天零点
//   - params.EndDate   非空则过滤 created_at <  次日零点
//   - params.Page      从1开始；params.PageSize 默认20，上限100
func ListAuditLogs(ctx context.Context, params AuditLogQueryParams) (*AuditLogListResult, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 20
	}
	offset := (params.Page - 1) * params.PageSize

	// ---- 动态构建 WHERE 子句 ----
	where := "WHERE 1=1"
	args := []interface{}{}
	idx := 1

	// 按用户ID精确过滤（旧接口保留）
	if params.UserID != "" {
		where += fmt.Sprintf(" AND al.user_id = $%d", idx)
		args = append(args, params.UserID)
		idx++
	}

	// 按用户名/显示名模糊搜索（ILIKE，同一参数复用）
	if params.Username != "" {
		where += fmt.Sprintf(
			" AND (u.username ILIKE $%d OR u.display_name ILIKE $%d)",
			idx, idx,
		)
		args = append(args, "%"+params.Username+"%")
		idx++
	}

	// 按操作类型精确过滤
	if params.Action != "" {
		where += fmt.Sprintf(" AND al.action = $%d", idx)
		args = append(args, params.Action)
		idx++
	}

	// 开始日期：created_at >= StartDate 当天零点
	if params.StartDate != "" {
		where += fmt.Sprintf(" AND al.created_at >= $%d::date", idx)
		args = append(args, params.StartDate)
		idx++
	}

	// 结束日期：created_at < EndDate 次日零点（含当天）
	if params.EndDate != "" {
		where += fmt.Sprintf(" AND al.created_at < ($%d::date + INTERVAL '1 day')", idx)
		args = append(args, params.EndDate)
		idx++
	}

	// ---- 查总数 ----
	countSQL := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM audit_logs al
		LEFT JOIN users u ON u.id = al.user_id
		%s`, where)

	var total int
	if err := database.DB.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("统计审计日志失败: %w", err)
	}

	// ---- 查数据（带分页）----
	dataArgs := append(args, params.PageSize, offset)
	dataSQL := fmt.Sprintf(`
		SELECT al.id, al.user_id,
		       COALESCE(u.username, '已删除用户') AS username,
		       COALESCE(u.display_name, '')       AS display_name,
		       al.action,
		       COALESCE(al.detail::text, '{}')    AS detail,
		       COALESCE(al.ip, '')                AS ip,
		       al.created_at
		FROM audit_logs al
		LEFT JOIN users u ON u.id = al.user_id
		%s
		ORDER BY al.created_at DESC
		LIMIT $%d OFFSET $%d`, where, idx, idx+1)

	rows, err := database.DB.Query(ctx, dataSQL, dataArgs...)
	if err != nil {
		return nil, fmt.Errorf("查询审计日志失败: %w", err)
	}
	defer rows.Close()

	var logs []AuditLogItem
	for rows.Next() {
		var item AuditLogItem
		if err := rows.Scan(
			&item.ID, &item.UserID, &item.Username, &item.DisplayName,
			&item.Action, &item.Detail, &item.IP, &item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("扫描审计日志行失败: %w", err)
		}
		// 填充中文操作名
		if name, ok := actionNameMap[item.Action]; ok {
			item.ActionName = name
		} else {
			item.ActionName = item.Action
		}
		logs = append(logs, item)
	}

	if logs == nil {
		logs = []AuditLogItem{}
	}
	return &AuditLogListResult{Logs: logs, Total: total}, nil
}

// ==================== 工具函数 ====================

// GetClientIP 从 RemoteAddr 提取客户端 IP
func GetClientIP(remoteAddr string) string {
	if len(remoteAddr) > 0 && remoteAddr[0] == '[' {
		for i, c := range remoteAddr {
			if c == ']' && i > 1 {
				return remoteAddr[1:i]
			}
		}
	}
	for i := len(remoteAddr) - 1; i >= 0; i-- {
		if remoteAddr[i] == ':' {
			return remoteAddr[:i]
		}
	}
	return remoteAddr
}

// QueryAuditLogs 兼容旧接口（保留，防止编译错误）
func QueryAuditLogs(ctx context.Context, userID string, action string, limit int) ([]map[string]interface{}, error) {
	result, err := ListAuditLogs(ctx, AuditLogQueryParams{
		UserID:   userID,
		Action:   action,
		Page:     1,
		PageSize: limit,
	})
	if err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, 0, len(result.Logs))
	for _, l := range result.Logs {
		out = append(out, map[string]interface{}{
			"id":         l.ID,
			"user_id":    l.UserID,
			"username":   l.Username,
			"action":     l.Action,
			"detail":     l.Detail,
			"ip":         l.IP,
			"created_at": l.CreatedAt,
		})
	}
	return out, nil
}
