package services

import (
	"errors"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== 提示词业务逻辑层 ====================

// 错误常量
var (
	ErrPromptKeyRequired   = errors.New("提示词标识不能为空")
	ErrInvalidPromptKey    = errors.New("无效的提示词标识")
	ErrPromptContentEmpty  = errors.New("提示词内容不能为空")
	ErrPromptNotFound      = errors.New("提示词不存在")
	ErrVersionNotFound     = errors.New("目标版本不存在")
	ErrAlreadyCurrent      = errors.New("该版本已经是当前生效版本")
)

// PromptService 提示词服务
type PromptService struct{}

// NewPromptService 创建提示词服务实例
func NewPromptService() *PromptService {
	return &PromptService{}
}

// ListCurrentPrompts 获取所有槽位的当前生效版本列表
func (s *PromptService) ListCurrentPrompts() (*models.PromptListResponse, error) {
	// 查询所有 is_current=true 的提示词
	prompts, err := repository.GetCurrentPrompts()
	if err != nil {
		return nil, err
	}

	// 转换为响应格式，附加中文名
	var responses []models.PromptResponse
	for _, p := range prompts {
		responses = append(responses, toPromptResponse(p))
	}

	return &models.PromptListResponse{
		Prompts: responses,
		Total:   len(responses),
	}, nil
}

// GetPromptByKey 获取指定槽位的当前生效版本
func (s *PromptService) GetPromptByKey(key string) (*models.PromptResponse, error) {
	// 校验 prompt_key
	if strings.TrimSpace(key) == "" {
		return nil, ErrPromptKeyRequired
	}
	if !models.IsValidPromptKey(key) {
		return nil, ErrInvalidPromptKey
	}

	// 查询当前生效版本
	prompt, err := repository.GetCurrentPromptByKey(key)
	if err != nil {
		return nil, ErrPromptNotFound
	}

	resp := toPromptResponse(*prompt)
	return &resp, nil
}

// UpdatePrompt 更新提示词内容（创建新版本）
// 不会覆盖旧版本，而是创建新版本并标记为当前生效
func (s *PromptService) UpdatePrompt(key string, content string, userID string) (*models.PromptResponse, error) {
	// 校验 prompt_key
	if strings.TrimSpace(key) == "" {
		return nil, ErrPromptKeyRequired
	}
	if !models.IsValidPromptKey(key) {
		return nil, ErrInvalidPromptKey
	}

	// 校验内容不能为空（允许空白字符作为内容，但不允许纯空）
	if strings.TrimSpace(content) == "" {
		return nil, ErrPromptContentEmpty
	}

	// 获取当前最大版本号
	maxVersion, err := repository.GetMaxVersion(key)
	if err != nil {
		return nil, err
	}

	// 创建新版本（版本号 = 最大版本号 + 1）
	newVersion := maxVersion + 1
	prompt, err := repository.CreatePromptVersion(key, content, newVersion, userID)
	if err != nil {
		return nil, err
	}

	resp := toPromptResponse(*prompt)
	return &resp, nil
}

// GetVersionHistory 获取指定槽位的版本历史
func (s *PromptService) GetVersionHistory(key string) (*models.PromptVersionListResponse, error) {
	// 校验 prompt_key
	if strings.TrimSpace(key) == "" {
		return nil, ErrPromptKeyRequired
	}
	if !models.IsValidPromptKey(key) {
		return nil, ErrInvalidPromptKey
	}

	// 查询所有版本（按版本号倒序）
	versions, err := repository.GetPromptVersions(key)
	if err != nil {
		return nil, err
	}

	// 转换为版本响应格式
	var versionResponses []models.PromptVersionResponse
	for _, v := range versions {
		versionResponses = append(versionResponses, models.PromptVersionResponse{
			ID:         v.ID,
			Version:    v.Version,
			Content:    v.Content,
			ContentLen: len([]rune(v.Content)),
			IsCurrent:  v.IsCurrent,
			CreatedBy:  v.CreatedBy,
			CreatedAt:  v.CreatedAt,
		})
	}

	return &models.PromptVersionListResponse{
		PromptKey:  key,
		PromptName: models.PromptNameMap[key],
		Versions:   versionResponses,
		Total:      len(versionResponses),
	}, nil
}

// RollbackToVersion 回滚到指定版本
// 将目标版本设为当前生效版本，其他版本设为非当前
func (s *PromptService) RollbackToVersion(key string, versionID string) (*models.PromptResponse, error) {
	// 校验 prompt_key
	if strings.TrimSpace(key) == "" {
		return nil, ErrPromptKeyRequired
	}
	if !models.IsValidPromptKey(key) {
		return nil, ErrInvalidPromptKey
	}

	// 校验 versionID 不能为空
	if strings.TrimSpace(versionID) == "" {
		return nil, ErrVersionNotFound
	}

	// 先查询目标版本，确认存在且属于该 prompt_key
	targetVersion, err := repository.GetPromptByID(versionID)
	if err != nil {
		return nil, ErrVersionNotFound
	}

	// 确认 prompt_key 匹配
	if targetVersion.PromptKey != key {
		return nil, ErrVersionNotFound
	}

	// 检查是否已经是当前版本
	if targetVersion.IsCurrent {
		return nil, ErrAlreadyCurrent
	}

	// 执行回滚（事务操作）
	err = repository.RollbackPromptVersion(key, versionID)
	if err != nil {
		return nil, err
	}

	// 查询回滚后的当前版本返回
	prompt, err := repository.GetCurrentPromptByKey(key)
	if err != nil {
		return nil, err
	}

	resp := toPromptResponse(*prompt)
	return &resp, nil
}

// toPromptResponse 将 Prompt 模型转为前端响应格式
func toPromptResponse(p models.Prompt) models.PromptResponse {
	return models.PromptResponse{
		ID:         p.ID,
		PromptKey:  p.PromptKey,
		PromptName: models.PromptNameMap[p.PromptKey],
		Content:    p.Content,
		Version:    p.Version,
		ContentLen: len([]rune(p.Content)),
		IsCurrent:  p.IsCurrent,
		CreatedBy:  p.CreatedBy,
		CreatedAt:  p.CreatedAt,
	}
}
