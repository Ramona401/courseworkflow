package models

// notification.go — 通用通知中心数据模型（审核协作机制 阶段5·站内信载体）
//
// 设计定位（一句话）：一张通用 notifications 表承载任意业务事件 → 给指定用户推一条站内信。
//   - 不做实时推送（无 WebSocket/SSE），前端轮询未读数，与系统现有范式一致。
//   - 业务语义解耦：一条通知 = 给谁 + 什么类型 + 指向哪个业务对象 + 一句话文案 + 已读否。
//     业务模块只管"发生了什么事、该通知谁"，通知中心只管"存、查、标已读"。
//   - 新增业务接通知 = 加一个 type 常量 + 调一次 EmitNotification，零改表。
//
// 关联：被 notification_repo.go 读写 ← notification_service.go ← notification_handler.go。
//
// 软关联设计：entity_id/actor_id 无外键（镜像 teacher_assistant_prefs.assistant_id、
//   courseware_annotations.page_number 的一贯做法），业务对象删除时通知不连删不报错；
//   recipient_id 是唯一硬外键（人删了通知确实该删，CASCADE）。

import "time"

// ==================== 通知类型常量 ====================
//
// 命名规范：{业务前缀}_{动作}。前缀对齐业务线便于将来按类型筛选。
// 阶段5 实际接线 7 个（cw_collab_* 4 个 + cw_review_* 3 个）。
// 阶段5 收尾（教案审核接线）：lp_review_* 3 个接线（提交/通过/退回），镜像课件审核。
const (
	// —— 集体备课（阶段4，阶段5b 接线）——
	NotifCWCollabInvited = "cw_collab_invited" // 被拉入集体备课     收件人=被邀请人 actor=作者
	NotifCWCollabRemoved = "cw_collab_removed" // 被移出集体备课     收件人=被移除人 actor=作者
	NotifCWCollabStarted = "cw_collab_started" // 集体备课已发起     收件人=参与者   actor=作者
	NotifCWCollabEnded   = "cw_collab_ended"   // 集体备课已结束     收件人=参与者   actor=作者

	// —— 课件多级审核（阶段3，阶段5c 接线）——
	NotifCWReviewSubmitted = "cw_review_submitted" // 有新待审课件   收件人=审核员   actor=提交作者
	NotifCWReviewApproved  = "cw_review_approved"  // 课件审核通过   收件人=作者     actor=审核员
	NotifCWReviewRevision  = "cw_review_revision"  // 课件被退回     收件人=作者     actor=审核员

	// —— 教案多级审核（阶段3 既有功能，阶段5 收尾接线）——
	// 镜像课件审核三事件，由 lesson_plan_review_notify.go 旁路发送：
	//   submitted → 通知作者所属教研组的 L1 审核员（lead/backbone）
	//   approved  → 通知作者「审核通过」
	//   revision  → 通知作者「被退回」（审核意见进 body）
	NotifLPReviewSubmitted = "lp_review_submitted" // 有新待审教案   收件人=L1审核员 actor=提交作者
	NotifLPReviewApproved  = "lp_review_approved"  // 教案审核通过   收件人=作者     actor=审核员
	NotifLPReviewRevision  = "lp_review_revision"  // 教案被退回     收件人=作者     actor=审核员

	// —— Token 积分自动分配（自动补/月底补足机制，2026-07-04 新增）——
	// 由 token_auto_alloc_service.go 旁路发送（走通知旁路范式，best-effort）：
	//   学校积分池余额 < 本校当月消费的 10%（快见底）时，通知该校所在区域的区域管理员，
	//   提醒尽快给学校池充值，否则老师用完积分将无法自动补齐。
	//   收件人=区域管理员   actor=系统（无 actor_name）   entity=学校账户
	//   同校每天最多 1 条（去重键控制，见 token_auto_alloc_service.go）。
	NotifTokenSchoolPoolLow = "token_school_pool_low" // 学校积分池不足   收件人=区域管理员 actor=系统

	// —— 以下为预留命名，尚未接线 ——
	NotifLPInteraction = "lp_interaction" // 教案被点赞/收藏（预留）
	NotifCWInspection  = "cw_inspection"  // 课件被抽查（预留，阶段6）
)

// ==================== 业务对象类型常量 ====================
// entity_type 标识通知指向的业务对象类别，配合 entity_id 软关联。
const (
	NotifEntityCourseware   = "courseware"    // 课件
	NotifEntityLessonPlan   = "lesson_plan"   // 教案
	NotifEntityInspection   = "inspection"    // 抽查记录
	NotifEntityTokenAccount = "token_account" // 积分账户（自动分配·学校池不足通知指向学校账户）
)

// ==================== 实体 ====================

// Notification 对应 notifications 表一行，是一条站内信的事件快照。
//
// actor_name/link 为冗余字段（写入时一次性算好），列表渲染零 JOIN。
// 代价是触发人改名后旧通知名不刷新，但通知是"事件快照"语义，旧名反而正确。
type Notification struct {
	ID          string     `json:"id"`
	RecipientID string     `json:"recipient_id"` // 收件人（硬外键 users CASCADE）
	Type        string     `json:"type"`         // 事件类型常量
	Title       string     `json:"title"`        // 一句话标题
	Body        string     `json:"body"`         // 详情（可空）
	EntityType  string     `json:"entity_type"`  // 业务对象类型（可空）
	EntityID    string     `json:"entity_id"`    // 业务对象ID（软关联，可空）
	ActorID     string     `json:"actor_id"`     // 触发人ID（可空）
	ActorName   string     `json:"actor_name"`   // 冗余触发人显示名
	Link        string     `json:"link"`         // 前端点击跳转路径
	IsRead      bool       `json:"is_read"`
	ReadAt      *time.Time `json:"read_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

// ==================== 请求/响应 ====================

// NotificationListResponse 通知列表分页响应。
// Notifications 非 nil 保证（service 层兜底空切片防前端 .map 崩）。
type NotificationListResponse struct {
	Notifications []*Notification `json:"notifications"`
	Total         int             `json:"total"`
	UnreadCount   int             `json:"unread_count"` // 顺带回未读总数供前端刷红点
}

// UnreadCountResponse 未读数响应（顶栏红点轮询用，极轻）。
type UnreadCountResponse struct {
	UnreadCount int `json:"unread_count"`
}

// EmitNotificationInput 写入一条通知的入参（供 service.EmitNotification 使用）。
//
// 字段语义见 Notification。RecipientID/Type/Title 为必填，其余可空。
type EmitNotificationInput struct {
	RecipientID string
	Type        string
	Title       string
	Body        string
	EntityType  string
	EntityID    string
	ActorID     string
	ActorName   string
	Link        string
}
