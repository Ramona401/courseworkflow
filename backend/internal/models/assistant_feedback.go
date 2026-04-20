package models

// assistant_feedback.go — AI 助手反馈数据模型
//
// 对应数据库表:assistant_feedback(2026-04-20 P1 收尾引入)
//
// 业务目标:
//   - 老师每条 AI 回复末尾点 👍/👎,可选填一句话评论
//   - 记录完整上下文(scene/stage/lesson_plan/ai_response_preview),
//     为 P2 数据飞轮提供真实信号
//   - 零强制、零打扰:comment 可空,stage_code 可空
//
// 关联关系:
//   assistant_id → ai_assistants.id (CASCADE)
//   user_id      → users.id         (CASCADE)
//   lesson_plan_id → lesson_plans.id (SET NULL,教案删了反馈仍保留)
//   trace_id 不加外键(ai_call_traces 未来归档清理)

import "time"

// ==================== 反馈倾向常量 ====================

const (
	FeedbackRatingUp   = "up"   // 👍 赞
	FeedbackRatingDown = "down" // 👎 踩
)

// ValidFeedbackRatings 有效的 rating 值(与数据库 CHECK 约束对齐)
var ValidFeedbackRatings = []string{
	FeedbackRatingUp,
	FeedbackRatingDown,
}

// IsValidFeedbackRating 校验 rating 是否有效
func IsValidFeedbackRating(r string) bool {
	for _, v := range ValidFeedbackRatings {
		if v == r {
			return true
		}
	}
	return false
}

// ==================== 数据库实体 ====================

// AssistantFeedback AI 助手反馈主实体(对应 assistant_feedback 表)
//
// 所有 UUID 字段使用 string 表示,与项目内其他 UUID 实体保持一致
// 可空外键字段使用 *string 指针,便于区分"未填写"和"空字符串"
type AssistantFeedback struct {
	ID string `json:"id"`

	// 反馈核心
	AssistantID string  `json:"assistant_id"`
	UserID      string  `json:"user_id"`
	Rating      string  `json:"rating"`  // up | down
	Comment     *string `json:"comment"` // 可选评论,NULL 时为 nil

	// 上下文锚点
	SceneCode         string  `json:"scene_code"`            // 必填,6 个场景之一
	LessonPlanID      *string `json:"lesson_plan_id"`        // 工坊/评审场景填写
	StageCode         *string `json:"stage_code"`            // 仅工坊场景填写
	AIResponsePreview *string `json:"ai_response_preview"`   // AI 回复前 300 字

	// 追溯(可选)
	TraceID *string `json:"trace_id"`

	// 审计
	CreatedAt *time.Time `json:"created_at"`
}

// ==================== 列表项(带展示辅助字段) ====================

// AssistantFeedbackListItem 反馈列表项,附带助手名和用户名(便于后台查看)
type AssistantFeedbackListItem struct {
	ID                string     `json:"id"`
	AssistantID       string     `json:"assistant_id"`
	AssistantName     string     `json:"assistant_name"` // 助手名(LEFT JOIN ai_assistants)
	UserID            string     `json:"user_id"`
	UserName          string     `json:"user_name"` // 用户显示名(LEFT JOIN users)
	Rating            string     `json:"rating"`
	Comment           *string    `json:"comment"`
	SceneCode         string     `json:"scene_code"`
	LessonPlanID      *string    `json:"lesson_plan_id"`
	StageCode         *string    `json:"stage_code"`
	AIResponsePreview *string    `json:"ai_response_preview"`
	CreatedAt         *time.Time `json:"created_at"`
}

// ==================== 请求/响应结构 ====================

// CreateFeedbackRequest 创建反馈请求
//
// 字段设计原则:
//   - 前端能提供什么就传什么,可空字段不强制
//   - rating 和 scene_code 是必填项(前端按钮场景本来就知道)
//   - user_id 从 JWT 解析,不接受前端传
type CreateFeedbackRequest struct {
	AssistantID       string  `json:"assistant_id"`
	Rating            string  `json:"rating"`
	Comment           *string `json:"comment"`
	SceneCode         string  `json:"scene_code"`
	LessonPlanID      *string `json:"lesson_plan_id"`
	StageCode         *string `json:"stage_code"`
	AIResponsePreview *string `json:"ai_response_preview"`
	TraceID           *string `json:"trace_id"`
}

// FeedbackStatsResponse 单个助手的反馈统计响应
//
// 用途:展示在 AI 管理中心或教研员后台,让管理员看到某助手的受欢迎程度
// 计算方式:
//   up_count   — rating='up'  的总数
//   down_count — rating='down' 的总数
//   up_ratio   — up / (up + down),保留 2 位小数,无数据时为 0
type FeedbackStatsResponse struct {
	AssistantID string  `json:"assistant_id"`
	UpCount     int     `json:"up_count"`
	DownCount   int     `json:"down_count"`
	TotalCount  int     `json:"total_count"`
	UpRatio     float64 `json:"up_ratio"` // 0.00 ~ 1.00
}

// FeedbackListResponse 列表分页响应
type FeedbackListResponse struct {
	Feedbacks []*AssistantFeedbackListItem `json:"feedbacks"`
	Total     int                          `json:"total"`
}

// ==================== 列表查询参数 ====================

// ListFeedbackParams 反馈列表查询参数(仅 admin 可查全部)
//
// 过滤维度:
//   - AssistantID:查某个具体助手的所有反馈
//   - UserID:     查某个用户的所有反馈(自查)
//   - Rating:     筛选 up / down
//   - SceneCode:  按场景筛选
//   - StartDate/EndDate: 时间范围
//
// 分页:Page 从 1 开始,PageSize 默认 20,上限 100
type ListFeedbackParams struct {
	AssistantID string
	UserID      string
	Rating      string
	SceneCode   string
	StartDate   string // yyyy-MM-dd
	EndDate     string // yyyy-MM-dd
	Page        int
	PageSize    int
}
