package services

// notification_service.go — 通用通知中心业务层
//
// 对外两类能力：
//   1. 写入（供任意业务模块调用，best-effort 异步旁路）：
//        EmitNotification       单条
//        EmitNotificationBatch  一事件多收件人
//      —— 铁律：通知写入永远是旁路，写失败仅 logger.Warn 绝不阻断主业务。
//         集体备课拉人成功了、通知没写进去，业务照样成功，绝不能因通知失败回滚 AddCollabMember。
//         镜像 audit_repo.WriteAuditLog 的"go func + 失败仅记日志不返回"范式。
//   2. 读取/标已读（供 notification_handler 调用）：List / CountUnread / MarkRead / MarkAllRead。
//
// 包级单例 GlobalNotificationService：跨 service 调用入口（courseware_collab_service、
//   courseware_review_service 等直接 services.GlobalNotificationService.EmitNotification(...)），
//   无需在各处构造依赖，与 GlobalCWSSEHub / GlobalKBSSEHub 的包级单例范式一致。

import (
	"context"
	"time"

	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// NotificationService 通知中心服务（无状态，仅承载方法）。
type NotificationService struct{}

// GlobalNotificationService 包级单例，供跨 service 旁路发通知。
var GlobalNotificationService = &NotificationService{}

var notifSvcLog = logger.WithModule("notification-svc")

// NewNotificationService 构造（handler 注册用；与单例指向同一无状态实现）。
func NewNotificationService() *NotificationService {
	return &NotificationService{}
}

// ==================== 写入（best-effort 异步）====================

// EmitNotification 发一条通知。异步执行，立即返回，绝不阻塞主业务流。
//
// 写入失败仅 Warn 不返回错误——调用方（如 AddCollabMember）无需也不应处理通知写入结果。
// 入参校验：recipient/type/title 任一为空则跳过（不发空通知，仅 Warn）。
func (s *NotificationService) EmitNotification(in models.EmitNotificationInput) {
	if in.RecipientID == "" || in.Type == "" || in.Title == "" {
		notifSvcLog.Warn("通知入参不完整已跳过", "recipient", in.RecipientID, "type", in.Type)
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := repository.CreateNotification(ctx, in); err != nil {
			notifSvcLog.Warn("通知写入失败(已忽略不影响主流程)",
				"recipient", in.RecipientID, "type", in.Type, "error", err)
		}
	}()
}

// EmitNotificationBatch 给多个收件人发同一事件（共享 type/title/body/entity/actor）。
// 同样异步 best-effort。recipientIDs 为空直接返回。
func (s *NotificationService) EmitNotificationBatch(recipientIDs []string, in models.EmitNotificationInput) {
	if len(recipientIDs) == 0 || in.Type == "" || in.Title == "" {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		n, err := repository.BatchCreateNotifications(ctx, recipientIDs, in)
		if err != nil {
			notifSvcLog.Warn("批量通知写入失败(已忽略不影响主流程)",
				"count", len(recipientIDs), "type", in.Type, "error", err)
			return
		}
		notifSvcLog.Info("批量通知已发", "written", n, "type", in.Type)
	}()
}

// ==================== 读取/标已读 ====================

// List 拉某用户通知列表（同步，handler 直接调）。
func (s *NotificationService) List(ctx context.Context, recipientID string, unreadOnly bool, limit, offset int) (*models.NotificationListResponse, error) {
	items, total, unread, err := repository.ListNotifications(ctx, recipientID, unreadOnly, limit, offset)
	if err != nil {
		return nil, err
	}
	return &models.NotificationListResponse{
		Notifications: items, // repo 已保证非 nil
		Total:         total,
		UnreadCount:   unread,
	}, nil
}

// CountUnread 查未读数（顶栏红点）。
func (s *NotificationService) CountUnread(ctx context.Context, recipientID string) (int, error) {
	return repository.CountUnread(ctx, recipientID)
}

// MarkRead 标单条已读（强制按 recipientID 防越权，repo 内已收口）。
func (s *NotificationService) MarkRead(ctx context.Context, notificationID, recipientID string) error {
	return repository.MarkRead(ctx, notificationID, recipientID)
}

// MarkAllRead 全部标已读，返回受影响数。
func (s *NotificationService) MarkAllRead(ctx context.Context, recipientID string) (int, error) {
	return repository.MarkAllRead(ctx, recipientID)
}
