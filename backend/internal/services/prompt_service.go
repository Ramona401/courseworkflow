package services

import (
	"errors"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== 提示词业务逻辑层 ====================
//
// v2 治理改造（保持原 handler 接口契约不变，仅换内部实现）：
//   1. key 合法性校验：从「models 写死 9 个白名单」改为「调 repository.PromptKeyExists
//      查 DB 是否存在该 key」。使课件/知识库那 19 个已入库但旧代码拒绝写入的 key
//      恢复可编辑，且未来新 key 自动纳管，无需再改代码。
//   2. toPromptResponse 转换时统一填充 Category（危险分档）/ PromptName（中文名）/
//      Description（用途说明），供前端红/橙/绿三色标记与差异化二次确认文案。
//   本文件对外契约（PromptService 结构体 + NewPromptService + 5 个方法名 + 错误常量
//   + toPromptResponse）与改造前完全一致，handler 层零改动。

// 业务错误常量（与 handler handlePromptError 的错误映射一一对应，勿改名）
var (
	ErrPromptKeyRequired  = errors.New("提示词标识不能为空")
	ErrPromptContentEmpty = errors.New("提示词内容不能为空")
	ErrPromptNotFound     = errors.New("提示词不存在")
	ErrVersionNotFound    = errors.New("目标版本不存在")
	ErrAlreadyCurrent     = errors.New("该版本已经是当前生效版本")
	// v2 动态校验新增：key 在 prompts 表中不存在（非法或误拼）。
	// 命名与 handler handlePromptError 中的引用保持一致（ErrInvalidPromptKey → 映射 400）。
	ErrInvalidPromptKey = errors.New("无效的提示词标识：该标识在提示词库中不存在")
)

// PromptService 提示词业务服务（无状态结构体）
type PromptService struct{}

// NewPromptService 构造提示词业务服务
func NewPromptService() *PromptService {
	return &PromptService{}
}

// isValidPromptKey 动态校验 key 是否存在于 prompts 表（任意版本即算存在）。
//   替代原写死的 9 个白名单，DB 查询异常时 fail-closed 返回 false（保守拒绝，防脏写）。
func (s *PromptService) isValidPromptKey(key string) bool {
	if key == "" {
		return false
	}
	exists, err := repository.PromptKeyExists(key)
	if err != nil {
		return false
	}
	return exists
}

// ListCurrentPrompts 获取所有槽位的当前生效版本列表。
// v2：返回 prompts 表全部 is_current=true 记录（不再受 9 个白名单限制），
//     每条经 toPromptResponse 附带危险分档/中文名/描述。
func (s *PromptService) ListCurrentPrompts() (*models.PromptListResponse, error) {
	prompts, err := repository.GetCurrentPrompts()
	if err != nil {
		return nil, err
	}

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
	if key == "" {
		return nil, ErrPromptKeyRequired
	}
	// v2 动态校验：key 必须在库中存在
	if !s.isValidPromptKey(key) {
		return nil, ErrInvalidPromptKey
	}

	prompt, err := repository.GetCurrentPromptByKey(key)
	if err != nil {
		return nil, ErrPromptNotFound
	}

	resp := toPromptResponse(*prompt)
	return &resp, nil
}

// UpdatePrompt 更新提示词（创建新版本）。
// 校验：key 非空 + key 在库存在 + 内容非空；版本号 = 当前最大版本号 + 1。
func (s *PromptService) UpdatePrompt(key string, content string, userID string) (*models.PromptResponse, error) {
	if key == "" {
		return nil, ErrPromptKeyRequired
	}
	// v2 动态校验：key 必须在库中存在（取代原写死白名单）
	if !s.isValidPromptKey(key) {
		return nil, ErrInvalidPromptKey
	}
	if content == "" {
		return nil, ErrPromptContentEmpty
	}

	// 计算新版本号 = 当前最大版本号 + 1
	maxVersion, err := repository.GetMaxVersion(key)
	if err != nil {
		return nil, err
	}
	newVersion := maxVersion + 1

	prompt, err := repository.CreatePromptVersion(key, content, newVersion, userID)
	if err != nil {
		return nil, err
	}

	resp := toPromptResponse(*prompt)
	return &resp, nil
}

// GetVersionHistory 获取指定槽位的版本历史（按版本号倒序）
func (s *PromptService) GetVersionHistory(key string) (*models.PromptVersionListResponse, error) {
	if key == "" {
		return nil, ErrPromptKeyRequired
	}
	// v2 动态校验：key 必须在库中存在
	if !s.isValidPromptKey(key) {
		return nil, ErrInvalidPromptKey
	}

	prompts, err := repository.GetPromptVersions(key)
	if err != nil {
		return nil, err
	}

	var versions []models.PromptVersionResponse
	for _, p := range prompts {
		versions = append(versions, models.PromptVersionResponse{
			ID:         p.ID,
			Version:    p.Version,
			Content:    p.Content,
			ContentLen: len([]rune(p.Content)),
			IsCurrent:  p.IsCurrent,
			CreatedBy:  p.CreatedBy,
			CreatedAt:  p.CreatedAt,
		})
	}

	return &models.PromptVersionListResponse{
		PromptKey:  key,
		PromptName: models.GetPromptName(key),
		Versions:   versions,
		Total:      len(versions),
	}, nil
}

// RollbackToVersion 回滚指定槽位到某个历史版本
func (s *PromptService) RollbackToVersion(key string, versionID string) (*models.PromptResponse, error) {
	if key == "" {
		return nil, ErrPromptKeyRequired
	}
	// v2 动态校验：key 必须在库中存在
	if !s.isValidPromptKey(key) {
		return nil, ErrInvalidPromptKey
	}

	// 校验目标版本存在且属于该 key
	target, err := repository.GetPromptByID(versionID)
	if err != nil {
		return nil, ErrVersionNotFound
	}
	if target.PromptKey != key {
		return nil, ErrVersionNotFound
	}
	// 目标已是当前生效版本 → 无需回滚
	if target.IsCurrent {
		return nil, ErrAlreadyCurrent
	}

	// 执行回滚
	if err := repository.RollbackPromptVersion(key, versionID); err != nil {
		return nil, err
	}

	// 返回回滚后的当前版本
	prompt, err := repository.GetCurrentPromptByKey(key)
	if err != nil {
		return nil, ErrPromptNotFound
	}

	resp := toPromptResponse(*prompt)
	return &resp, nil
}

// toPromptResponse 将 Prompt 模型转为前端响应格式。
// v2：统一附加危险分档（Category）、中文名（PromptName）、用途说明（Description），
//     三者均来自 models 层映射，未登记 key 有合理兜底（分档→mid / 名→key / 描述→通用文案）。
func toPromptResponse(p models.Prompt) models.PromptResponse {
	return models.PromptResponse{
		ID:          p.ID,
		PromptKey:   p.PromptKey,
		PromptName:  models.GetPromptName(p.PromptKey),
		Category:    models.GetPromptCategory(p.PromptKey),
		Description: models.GetPromptDescription(p.PromptKey),
		Content:     p.Content,
		Version:     p.Version,
		ContentLen:  len([]rune(p.Content)),
		IsCurrent:   p.IsCurrent,
		CreatedBy:   p.CreatedBy,
		CreatedAt:   p.CreatedAt,
	}
}
