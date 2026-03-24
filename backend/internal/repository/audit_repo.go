package repository

// audit_repo.go — 审计日志数据访问层
//
// 写入 audit_logs 表，记录关键业务操作：
//   - 用户登录 / 登出
//   - Pipeline 提交定稿 / 确认定稿 / 退回重审 / 直接定稿 / 快捷通过 / 触发验收
//
// 设计原则：
//   1. WriteAuditLog 永不返回 error 给调用方（审计失败不影响主流程）
//   2. 失败时只记录 slog 日志，不 panic
//   3. 使用独立 context（不依赖 HTTP request context，防止请求结束后 context 取消）
//   4. detail 字段为 JSONB，存储操作相关的业务参数

import (
	"context"
	"encoding/json"
	"time"

	"tedna/internal/database"
	"tedna/internal/logger"
)

// auditLog 模块日志
var auditLog = logger.WithModule("audit")

// AuditAction 审计操作类型常量
// 与 audit_logs.action 字段对应（varchar100）
const (
	// 认证类
	ActionLogin  = "user.login"   // 用户登录成功
	ActionLogout = "user.logout"  // 用户登出

	// Pipeline 二级审批类
	ActionSubmitFinalize  = "pipeline.submit_finalize"   // 提交定稿申请
	ActionConfirmFinalize = "pipeline.confirm_finalize"  // 确认定稿
	ActionRejectFinalize  = "pipeline.reject_finalize"   // 退回重审
	ActionDirectFinalize  = "pipeline.direct_finalize"   // 直接定稿（admin）
	ActionMarkPassed      = "pipeline.mark_passed"       // 快捷通过
	ActionVerify          = "pipeline.verify"            // 触发验收
)

// WriteAuditLog 写入一条审计日志（异步，永不阻塞主流程）
//
// 参数：
//   - userID   : 操作者 UUID（对应 users.id）
//   - action   : 操作类型（使用上方 Action* 常量）
//   - detail   : 任意 map，序列化为 JSONB 存入 detail 字段
//   - ip       : 客户端 IP（可从 r.RemoteAddr 提取）
//
// 示例：
//   repository.WriteAuditLog(userID, repository.ActionConfirmFinalize,
//       map[string]interface{}{"pipeline_id": id}, ip)
func WriteAuditLog(userID, action string, detail map[string]interface{}, ip string) {
	// 异步写入：不阻塞 HTTP handler 响应
	go func() {
		// 使用独立 context，5s 超时，不依赖 HTTP request context
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// 序列化 detail 为 JSON
		detailJSON, err := json.Marshal(detail)
		if err != nil {
			auditLog.Error("审计日志序列化失败",
				"user_id", userID,
				"action", action,
				"error", err,
			)
			return
		}

		// 插入 audit_logs 表
		_, err = database.DB.Exec(ctx, `
			INSERT INTO audit_logs (user_id, action, detail, ip)
			VALUES ($1, $2, $3, $4)
		`, userID, action, string(detailJSON), ip)

		if err != nil {
			// 审计写入失败只记录日志，不影响主流程
			auditLog.Error("审计日志写入失败",
				"user_id", userID,
				"action", action,
				"error", err,
			)
			return
		}

		// DEBUG：正常写入不需要 INFO 级别（避免日志量翻倍）
		auditLog.Debug("审计日志已写入",
			"user_id", userID,
			"action", action,
			"ip", ip,
		)
	}()
}

// GetClientIP 从 RemoteAddr 提取客户端 IP（去掉端口号）
// r.RemoteAddr 格式为 "ip:port" 或 "[::1]:port"
func GetClientIP(remoteAddr string) string {
	// 处理 IPv6 格式 [::1]:port
	if len(remoteAddr) > 0 && remoteAddr[0] == '[' {
		end := 0
		for i, c := range remoteAddr {
			if c == ']' {
				end = i
				break
			}
		}
		if end > 1 {
			return remoteAddr[1:end]
		}
	}
	// 处理 IPv4 格式 ip:port
	for i := len(remoteAddr) - 1; i >= 0; i-- {
		if remoteAddr[i] == ':' {
			return remoteAddr[:i]
		}
	}
	return remoteAddr
}

// QueryAuditLogs 查询审计日志（供后续管理界面使用）
// limit 最多返回条数，userID 为空则查全部
func QueryAuditLogs(ctx context.Context, userID string, action string, limit int) ([]map[string]interface{}, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	var rows interface{}
	var err error

	query := `
		SELECT al.id, al.user_id, u.username, al.action, al.detail, al.ip, al.created_at
		FROM audit_logs al
		LEFT JOIN users u ON u.id = al.user_id
		WHERE 1=1
	`
	args := []interface{}{}
	argIdx := 1

	if userID != "" {
		query += " AND al.user_id = $" + string(rune('0'+argIdx))
		args = append(args, userID)
		argIdx++
	}
	if action != "" {
		query += " AND al.action = $" + string(rune('0'+argIdx))
		args = append(args, action)
		argIdx++
	}
	query += " ORDER BY al.created_at DESC LIMIT $" + string(rune('0'+argIdx))
	args = append(args, limit)

	_ = rows
	_ = err

	// 简化版：直接返回原始查询（供后续扩展）
	sqlRows, err := database.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer sqlRows.Close()

	result := []map[string]interface{}{}
	for sqlRows.Next() {
		var id, uid, username, act, detail, ip string
		var createdAt time.Time
		if err := sqlRows.Scan(&id, &uid, &username, &act, &detail, &ip, &createdAt); err != nil {
			continue
		}
		result = append(result, map[string]interface{}{
			"id":         id,
			"user_id":    uid,
			"username":   username,
			"action":     act,
			"detail":     detail,
			"ip":         ip,
			"created_at": createdAt,
		})
	}
	return result, nil
}
