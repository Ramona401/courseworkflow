package models

import (
	"time"

	"github.com/google/uuid"
)

// TrashItem 回收站通用列表项（教案和课件共用的轻量结构）
type TrashItem struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	Type      string    `json:"type"`      // "lesson_plan" 或 "courseware"
	Subject   string    `json:"subject"`
	Grade     string    `json:"grade"`
	DeletedAt time.Time `json:"deleted_at"`
	CreatedAt time.Time `json:"created_at"`
	// 过期剩余天数（前端展示"还剩X天"）
	DaysLeft  int       `json:"days_left"`
}
