package repository

// notification_repo.go — 通用通知中心数据访问层
//
// 操作 notifications 表：单条/批量写入 + 按收件人列表 + 未读计数 + 标已读。
// 所有读查询强制按 recipient_id 过滤（数据隔离在 service/handler 层收口，本层忠实执行）。
//
// 写入语义：best-effort，由 service 层包在异步 goroutine 内调用（镜像 audit_repo 范式），
//   本层只负责正确的 SQL，不决定同步/异步。

import (
	"context"
	"fmt"
	"strings"
	"time"

	"tedna/internal/database"
	"tedna/internal/models"
)

// notifSelectColumns 统一列清单（单条与列表共用，顺序与 scanNotification 严格对齐）。
const notifSelectColumns = `
	id, recipient_id, type, title,
	COALESCE(body, '')          AS body,
	COALESCE(entity_type, '')   AS entity_type,
	COALESCE(entity_id::text,'') AS entity_id,
	COALESCE(actor_id::text, '') AS actor_id,
	COALESCE(actor_name, '')    AS actor_name,
	COALESCE(link, '')          AS link,
	is_read, read_at, created_at`

// scanNotification 统一扫描一行（列顺序必须与 notifSelectColumns 一致）。
func scanNotification(row interface {
	Scan(dest ...interface{}) error
}) (*models.Notification, error) {
	var n models.Notification
	if err := row.Scan(
		&n.ID, &n.RecipientID, &n.Type, &n.Title,
		&n.Body, &n.EntityType, &n.EntityID, &n.ActorID,
		&n.ActorName, &n.Link, &n.IsRead, &n.ReadAt, &n.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &n, nil
}

// ==================== 写入 ====================

// CreateNotification 写入单条通知，返回新行 id。
//
// entity_id/actor_id 为软关联 UUID 列：空串经 nullIfEmptyUUID 写 NULL（非法 UUID 字符串
// 会被 PG 拒绝，故空值必须落 NULL 而非空串）。其余空串列直接写空串。
func CreateNotification(ctx context.Context, in models.EmitNotificationInput) (string, error) {
	var id string
	err := database.DB.QueryRow(ctx, `
		INSERT INTO notifications
			(recipient_id, type, title, body, entity_type, entity_id, actor_id, actor_name, link)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id`,
		in.RecipientID, in.Type, in.Title, in.Body,
		in.EntityType, nullIfEmptyUUID(in.EntityID),
		nullIfEmptyUUID(in.ActorID), in.ActorName, in.Link,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("写入通知失败: %w", err)
	}
	return id, nil
}

// BatchCreateNotifications 一次给多个收件人写同一事件（共享 type/title/body/entity/actor，仅收件人不同）。
//
// 用一条多 VALUES INSERT 避免循环单插，供"结束集体备课通知全体参与者""审核退回通知作者+链"等场景。
// recipientIDs 为空直接返回不拼 SQL。返回写入行数。
func BatchCreateNotifications(ctx context.Context, recipientIDs []string, in models.EmitNotificationInput) (int, error) {
	if len(recipientIDs) == 0 {
		return 0, nil
	}

	// 共享字段从 $1 起占位，每个收件人复用这些 + 各自的 recipient_id。
	// 占位布局：$1=type $2=title $3=body $4=entity_type $5=entity_id $6=actor_id $7=actor_name $8=link
	//           $9.. = 各 recipient_id
	args := []interface{}{
		in.Type, in.Title, in.Body, in.EntityType,
		nullIfEmptyUUID(in.EntityID), nullIfEmptyUUID(in.ActorID),
		in.ActorName, in.Link,
	}
	valueRows := make([]string, 0, len(recipientIDs))
	for i, rid := range recipientIDs {
		if strings.TrimSpace(rid) == "" {
			continue
		}
		args = append(args, rid)
		// recipient_id 是本行第 9+i 个参数
		valueRows = append(valueRows, fmt.Sprintf(
			"($%d, $1, $2, $3, $4, $5, $6, $7, $8)", 9+i))
	}
	if len(valueRows) == 0 {
		return 0, nil
	}

	sql := `INSERT INTO notifications
		(recipient_id, type, title, body, entity_type, entity_id, actor_id, actor_name, link)
		VALUES ` + strings.Join(valueRows, ", ")

	tag, err := database.DB.Exec(ctx, sql, args...)
	if err != nil {
		return 0, fmt.Errorf("批量写入通知失败: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

// ==================== 查询 ====================

// ListNotifications 按收件人分页拉通知，unreadOnly=true 时只返未读。
// 同时返回该收件人的未读总数（无论 unreadOnly 与否，红点口径一致）与匹配总数。
func ListNotifications(ctx context.Context, recipientID string, unreadOnly bool, limit, offset int) ([]*models.Notification, int, int, error) {
	if limit < 1 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	where := "WHERE recipient_id = $1"
	args := []interface{}{recipientID}
	if unreadOnly {
		where += " AND is_read = false"
	}

	// 匹配总数
	var total int
	if err := database.DB.QueryRow(ctx,
		"SELECT COUNT(*) FROM notifications "+where, args...).Scan(&total); err != nil {
		return nil, 0, 0, fmt.Errorf("统计通知失败: %w", err)
	}

	// 未读总数（红点用，独立于 unreadOnly 过滤）
	var unread int
	if err := database.DB.QueryRow(ctx,
		"SELECT COUNT(*) FROM notifications WHERE recipient_id = $1 AND is_read = false",
		recipientID).Scan(&unread); err != nil {
		return nil, 0, 0, fmt.Errorf("统计未读通知失败: %w", err)
	}

	// 列表数据
	dataArgs := append(args, limit, offset)
	dataSQL := fmt.Sprintf(`SELECT %s FROM notifications %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, notifSelectColumns, where, len(args)+1, len(args)+2)

	rows, err := database.DB.Query(ctx, dataSQL, dataArgs...)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("查询通知失败: %w", err)
	}
	defer rows.Close()

	list := make([]*models.Notification, 0)
	for rows.Next() {
		n, err := scanNotification(rows)
		if err != nil {
			return nil, 0, 0, fmt.Errorf("扫描通知行失败: %w", err)
		}
		list = append(list, n)
	}
	return list, total, unread, nil
}

// CountUnread 仅查某收件人未读数（顶栏红点轮询专用，极轻）。
func CountUnread(ctx context.Context, recipientID string) (int, error) {
	var n int
	err := database.DB.QueryRow(ctx,
		"SELECT COUNT(*) FROM notifications WHERE recipient_id = $1 AND is_read = false",
		recipientID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("统计未读数失败: %w", err)
	}
	return n, nil
}

// ==================== 标已读 ====================

// MarkRead 标单条已读。强制带 recipient_id 防越权改他人通知。
// RowsAffected==0 表示该通知不存在或不属于此人，归 ErrNotificationNotFound。
func MarkRead(ctx context.Context, notificationID, recipientID string) error {
	tag, err := database.DB.Exec(ctx, `
		UPDATE notifications
		SET is_read = true, read_at = $1
		WHERE id = $2 AND recipient_id = $3 AND is_read = false`,
		time.Now(), notificationID, recipientID)
	if err != nil {
		return fmt.Errorf("标记已读失败: %w", err)
	}
	if tag.RowsAffected() == 0 {
		// 可能已是已读状态或不属于此人；为幂等不报错，但区分"完全不存在"。
		var exists bool
		_ = database.DB.QueryRow(ctx,
			"SELECT EXISTS(SELECT 1 FROM notifications WHERE id = $1 AND recipient_id = $2)",
			notificationID, recipientID).Scan(&exists)
		if !exists {
			return ErrNotificationNotFound
		}
		// 存在但已读：幂等放行
	}
	return nil
}

// MarkAllRead 把某收件人全部未读标已读，返回受影响行数。
func MarkAllRead(ctx context.Context, recipientID string) (int, error) {
	tag, err := database.DB.Exec(ctx, `
		UPDATE notifications
		SET is_read = true, read_at = $1
		WHERE recipient_id = $2 AND is_read = false`,
		time.Now(), recipientID)
	if err != nil {
		return 0, fmt.Errorf("全部标记已读失败: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

// ==================== 错误常量与辅助 ====================

// ErrNotificationNotFound 通知不存在或不属于当前用户。
var ErrNotificationNotFound = fmt.Errorf("通知不存在")

// nullIfEmptyUUID 空串/纯空白 → nil（落 SQL NULL），否则原样返回。
// 用于 entity_id/actor_id 两个软关联 UUID 列：空值必须 NULL 而非空串（PG uuid 列不接受空串）。
func nullIfEmptyUUID(s string) interface{} {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}
