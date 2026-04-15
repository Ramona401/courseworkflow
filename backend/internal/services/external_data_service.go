package services

import (
	"errors"
	"fmt"
	"strings"

	"tedna/internal/config"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// ==================== 错误常量 ====================

var (
	ErrEDConfigsRequired = errors.New("配置数据不能为空")
)

// ==================== ExternalDataService ====================

// ExternalDataService 外部数据配置业务逻辑层
type ExternalDataService struct {
	aesKey string // AES加密密钥（与AI配置共用同一密钥）
}

// NewExternalDataService 创建外部数据配置服务实例
func NewExternalDataService(cfg *config.Config) *ExternalDataService {
	return &ExternalDataService{
		aesKey: cfg.AESKey,
	}
}

// ==================== 配置读取 ====================

// GetAllConfigs 获取所有外部数据配置（敏感字段脱敏处理）
func (s *ExternalDataService) GetAllConfigs() (*models.ExternalDataConfigListResponse, error) {
	// 从数据库获取所有配置项
	configs, err := repository.GetAllEDConfigs()
	if err != nil {
		return nil, fmt.Errorf("获取外部数据配置失败: %w", err)
	}

	// 转换为前端响应格式（敏感字段脱敏）
	var items []*models.ExternalDataConfigItem
	for _, c := range configs {
		item := &models.ExternalDataConfigItem{
			ConfigKey:   c.ConfigKey,
			Description: c.Description,
			IsSensitive: models.IsSensitiveEDKey(c.ConfigKey),
			UpdatedAt:   c.UpdatedAt,
		}

		// 判断是否已配置（非占位符且非空）
		isPlaceholder := c.ConfigValue == "PLACEHOLDER_SET_IN_ADMIN" || c.ConfigValue == ""
		item.IsSet = !isPlaceholder

		// 敏感字段脱敏处理
		if item.IsSensitive {
			if isPlaceholder {
				item.ConfigValue = "未配置"
			} else {
				// 尝试解密后脱敏显示
				item.ConfigValue = s.maskSensitiveValue(c.ConfigValue)
			}
		} else {
			// 非敏感字段：占位符显示为空，其他显示原值
			if isPlaceholder {
				item.ConfigValue = ""
			} else {
				item.ConfigValue = c.ConfigValue
			}
		}

		items = append(items, item)
	}

	return &models.ExternalDataConfigListResponse{
		Configs: items,
		Total:   len(items),
	}, nil
}

// ==================== 配置更新 ====================

// UpdateConfigs 批量更新外部数据配置
// 规则：
//   - 非敏感字段：直接更新（空字符串也会写入，表示清空）
//   - 敏感字段：空字符串表示不修改，非空则AES加密后存储
func (s *ExternalDataService) UpdateConfigs(req *models.UpdateExternalDataConfigsRequest, userID string) error {
	if len(req.Configs) == 0 {
		return ErrEDConfigsRequired
	}

	// 逐项更新
	for key, value := range req.Configs {
		value = strings.TrimSpace(value)

		// 敏感字段特殊处理
		if models.IsSensitiveEDKey(key) {
			// 空字符串表示不修改敏感字段
			if value == "" {
				continue
			}
			// 非空：AES加密后存储
			encrypted, err := utils.EncryptAES(value, s.aesKey)
			if err != nil {
				return fmt.Errorf("加密配置 %s 失败: %w", key, err)
			}
			if err := repository.UpdateEDConfigValue(key, encrypted, userID); err != nil {
				return err
			}
		} else {
			// 非敏感字段：直接存储
			// 如果值为空，存为占位符（保持数据库一致性）
			storeValue := value
			if storeValue == "" {
				storeValue = "PLACEHOLDER_SET_IN_ADMIN"
			}
			if err := repository.UpdateEDConfigValue(key, storeValue, userID); err != nil {
				return err
			}
		}
	}

	return nil
}

// ==================== 内部工具方法 ====================

// maskSensitiveValue 敏感值脱敏处理
// 尝试解密AES密文，然后显示前4位***后4位
// 如果解密失败（可能是明文旧数据），直接对原值脱敏
func (s *ExternalDataService) maskSensitiveValue(encValue string) string {
	// 占位符直接返回
	if encValue == "" || encValue == "PLACEHOLDER_SET_IN_ADMIN" {
		return "未配置"
	}

	// 尝试解密
	plaintext, err := utils.DecryptAES(encValue, s.aesKey)
	if err != nil {
		// 解密失败，可能是明文旧数据，直接对原值脱敏
		plaintext = encValue
	}

	// 脱敏：前4位 + *** + 后4位
	if len(plaintext) <= 8 {
		// 太短，只显示前2位+***+后2位
		if len(plaintext) <= 4 {
			return "***"
		}
		return plaintext[:2] + "***" + plaintext[len(plaintext)-2:]
	}
	return plaintext[:4] + "***" + plaintext[len(plaintext)-4:]
}
