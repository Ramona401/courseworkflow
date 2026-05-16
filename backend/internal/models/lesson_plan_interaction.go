package models

import "time"

// ==================== 教案互动模型（对应 lesson_plan_interactions 表） ====================

// 互动类型常量
const (
	InteractionTypeLike     = "like"     // 点赞
	InteractionTypeFavorite = "favorite" // 收藏
)

// LessonPlanInteraction 教案互动记录
type LessonPlanInteraction struct {
	ID              string    `json:"id"`               // UUID主键
	UserID          string    `json:"user_id"`           // 用户ID
	LessonPlanID    string    `json:"lesson_plan_id"`    // 教案ID
	InteractionType string    `json:"interaction_type"`  // 互动类型：like/favorite
	CreatedAt       time.Time `json:"created_at"`        // 创建时间
}

// ==================== 互动统计（嵌入教案列表/详情响应） ====================

// InteractionCounts 教案互动计数
type InteractionCounts struct {
	LikeCount     int  `json:"like_count"`      // 点赞总数
	FavoriteCount int  `json:"favorite_count"`   // 收藏总数
	IsLiked       bool `json:"is_liked"`         // 当前用户是否已点赞
	IsFavorited   bool `json:"is_favorited"`     // 当前用户是否已收藏
}

// ==================== 请求/响应结构体 ====================

// ToggleInteractionRequest 切换互动状态请求
type ToggleInteractionRequest struct {
	InteractionType string `json:"interaction_type"` // like 或 favorite
}

// ToggleInteractionResponse 切换互动状态响应
type ToggleInteractionResponse struct {
	Active    bool `json:"active"`     // 切换后的状态（true=已点赞/已收藏，false=已取消）
	NewCount  int  `json:"new_count"`  // 切换后该类型的最新计数
}

// FavoriteListItem 收藏列表条目（包含教案摘要信息）
type FavoriteListItem struct {
	InteractionID string     `json:"interaction_id"`   // 互动记录ID
	LessonPlanID  string     `json:"lesson_plan_id"`   // 教案ID
	Title         string     `json:"title"`             // 教案标题
	Subject       string     `json:"subject"`           // 学科
	Grade         string     `json:"grade"`             // 年级
	Topic         string     `json:"topic"`             // 课题
	AuthorName    string     `json:"author_name"`       // 作者名
	AIReviewScore *float64   `json:"ai_review_score"`   // AI评分
	Status        string     `json:"status"`            // 教案状态
	StatusName    string     `json:"status_name"`       // 教案状态中文名
	LikeCount     int        `json:"like_count"`        // 点赞数
	FavoriteCount int        `json:"favorite_count"`    // 收藏数
	FavoritedAt   *time.Time `json:"favorited_at"`      // 收藏时间
}

// FavoriteListResponse 收藏列表响应
type FavoriteListResponse struct {
	Items []*FavoriteListItem `json:"items"`
	Total int                 `json:"total"`
}
