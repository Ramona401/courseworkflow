package services

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// TrashService 回收站服务
type TrashService struct{}

// NewTrashService 创建回收站服务
func NewTrashService() *TrashService {
	return &TrashService{}
}

// TrashListResponse 回收站列表响应
type TrashListResponse struct {
	LessonPlans []models.TrashItem `json:"lesson_plans"`
	Coursewares []models.TrashItem `json:"coursewares"`
	Total       int                `json:"total"`
}

// ListTrash 获取当前用户的回收站列表（教案+课件合并）
func (s *TrashService) ListTrash(userIDStr string) (*TrashListResponse, error) {
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("用户ID无效")
	}

	lpItems, err := repository.ListLessonPlanTrash(userID)
	if err != nil {
		return nil, fmt.Errorf("获取教案回收站失败: %w", err)
	}
	if lpItems == nil {
		lpItems = []models.TrashItem{}
	}

	cwItems, err := repository.ListCoursewareTrash(userID)
	if err != nil {
		return nil, fmt.Errorf("获取课件回收站失败: %w", err)
	}
	if cwItems == nil {
		cwItems = []models.TrashItem{}
	}

	return &TrashListResponse{
		LessonPlans: lpItems,
		Coursewares: cwItems,
		Total:       len(lpItems) + len(cwItems),
	}, nil
}

// RestoreItem 恢复回收站中的项目
func (s *TrashService) RestoreItem(itemIDStr string, itemType string, userIDStr string) error {
	itemID, err := uuid.Parse(itemIDStr)
	if err != nil {
		return fmt.Errorf("项目ID无效")
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return fmt.Errorf("用户ID无效")
	}

	switch itemType {
	case "lesson_plan":
		return repository.RestoreLessonPlan(itemID, userID)
	case "courseware":
		return repository.RestoreCourseware(itemID, userID)
	default:
		return fmt.Errorf("不支持的类型: %s", itemType)
	}
}

// PermanentDeleteItem 永久删除回收站中的项目
func (s *TrashService) PermanentDeleteItem(itemIDStr string, itemType string, userIDStr string) error {
	itemID, err := uuid.Parse(itemIDStr)
	if err != nil {
		return fmt.Errorf("项目ID无效")
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return fmt.Errorf("用户ID无效")
	}

	switch itemType {
	case "lesson_plan":
		return repository.PermanentDeleteLessonPlan(itemID, userID)
	case "courseware":
		return repository.PermanentDeleteCourseware(itemID, userID)
	default:
		return fmt.Errorf("不支持的类型: %s", itemType)
	}
}

// PurgeExpiredTrash 清理过期的回收站数据（定时任务调用）
func (s *TrashService) PurgeExpiredTrash(days int) {
	lpCount, err := repository.PurgeLessonPlanExpiredTrash(days)
	if err != nil {
		log.Printf("[TrashScheduler] 清理过期教案失败: %v", err)
	} else if lpCount > 0 {
		log.Printf("[TrashScheduler] 已清理 %d 条过期教案", lpCount)
	}

	cwCount, err := repository.PurgeCoursewareExpiredTrash(days)
	if err != nil {
		log.Printf("[TrashScheduler] 清理过期课件失败: %v", err)
	} else if cwCount > 0 {
		log.Printf("[TrashScheduler] 已清理 %d 条过期课件", cwCount)
	}

	if lpCount == 0 && cwCount == 0 {
		log.Printf("[TrashScheduler] 本次无过期回收站数据需要清理")
	}
}

// StartTrashScheduler 启动回收站定时清理任务（每天凌晨3点执行）
func (s *TrashService) StartTrashScheduler() {
	const retentionDays = 30

	go func() {
		log.Printf("[TrashScheduler] 回收站定时清理任务已启动，保留 %d 天", retentionDays)
		s.PurgeExpiredTrash(retentionDays)

		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day(), 3, 0, 0, 0, now.Location())
		if now.After(next) {
			next = next.Add(24 * time.Hour)
		}
		log.Printf("[TrashScheduler] 下次清理时间: %s", next.Format("2006-01-02 15:04:05"))
		time.Sleep(time.Until(next))
		s.PurgeExpiredTrash(retentionDays)

		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			log.Printf("[TrashScheduler] 开始每日回收站清理...")
			s.PurgeExpiredTrash(retentionDays)
		}
	}()
}
