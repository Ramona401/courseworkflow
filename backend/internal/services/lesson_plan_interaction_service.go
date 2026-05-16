package services

// lesson_plan_interaction_service.go — 教案互动（点赞/收藏）业务逻辑层
//
// 职责：
//   - Toggle 点赞/收藏（参数校验 + 教案存在性检查）
//   - 查询教案互动统计
//   - 查询用户收藏列表

import (
	"context"
	"errors"

	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== 错误常量 ====================

var (
	ErrInvalidInteractionType = errors.New("互动类型无效，可选值：like / favorite")
)

// LessonPlanInteractionService 教案互动服务
type LessonPlanInteractionService struct{}

var lpInterLog = logger.WithModule("lesson_plan_interaction")

// NewLessonPlanInteractionService 创建教案互动服务实例
func NewLessonPlanInteractionService() *LessonPlanInteractionService {
	return &LessonPlanInteractionService{}
}

// ==================== Toggle 互动 ====================

// ToggleInteraction 切换点赞/收藏状态
// 返回切换后的状态 + 最新计数
func (s *LessonPlanInteractionService) ToggleInteraction(ctx context.Context, userID, planID, interactionType string) (*models.ToggleInteractionResponse, error) {
	// 校验互动类型
	if interactionType != models.InteractionTypeLike && interactionType != models.InteractionTypeFavorite {
		return nil, ErrInvalidInteractionType
	}

	// 校验教案存在
	_, err := repository.GetLessonPlanByID(ctx, planID)
	if err != nil {
		if errors.Is(err, repository.ErrLessonPlanNotFound) {
			return nil, ErrLPNotFound
		}
		return nil, err
	}

	// Toggle
	active, err := repository.ToggleInteraction(ctx, userID, planID, interactionType)
	if err != nil {
		lpInterLog.Error("切换互动状态失败", "user_id", userID, "plan_id", planID, "type", interactionType, "error", err)
		return nil, err
	}

	// 查询最新计数
	newCount, err := repository.GetInteractionCount(ctx, planID, interactionType)
	if err != nil {
		lpInterLog.Error("查询互动计数失败", "plan_id", planID, "type", interactionType, "error", err)
		// 计数查询失败不阻断，返回 0
		newCount = 0
	}

	action := "取消"
	if active {
		action = "添加"
	}
	lpInterLog.Info("互动操作完成", "user_id", userID, "plan_id", planID, "type", interactionType, "action", action, "new_count", newCount)

	return &models.ToggleInteractionResponse{
		Active:   active,
		NewCount: newCount,
	}, nil
}

// ==================== 查询互动统计 ====================

// GetInteractionCounts 查询教案的互动统计（含当前用户状态）
func (s *LessonPlanInteractionService) GetInteractionCounts(ctx context.Context, planID, currentUserID string) (*models.InteractionCounts, error) {
	return repository.GetInteractionCounts(ctx, planID, currentUserID)
}

// ==================== 收藏列表 ====================

// ListMyFavorites 查询当前用户的收藏列表
func (s *LessonPlanInteractionService) ListMyFavorites(ctx context.Context, userID string, limit, offset int) (*models.FavoriteListResponse, error) {
	items, total, err := repository.ListUserFavorites(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	return &models.FavoriteListResponse{Items: items, Total: total}, nil
}
