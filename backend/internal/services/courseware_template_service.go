package services

import (
	"context"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== 课件风格模板服务（薄代理层） ====================
// 将repository的模板查询方法暴露为services包级函数，供handler调用

// ListCWTemplates 获取风格模板列表
func ListCWTemplates(ctx context.Context, activeOnly bool) ([]*models.CoursewareTemplate, error) {
	return repository.ListCWTemplates(ctx, activeOnly)
}

// GetCWTemplateByID 获取风格模板详情
func GetCWTemplateByID(ctx context.Context, id string) (*models.CoursewareTemplate, error) {
	return repository.GetCWTemplateByID(ctx, id)
}
