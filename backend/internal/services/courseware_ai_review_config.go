package services

// courseware_ai_review_config.go
//
// R-02 课件AI审核配置的服务端协议与规范化逻辑。
//
// 安全边界：
//   - 浏览器只提交选择意图，不能提交配置哈希、课件身份或材料正文；
//   - 字段缺失表示旧客户端，使用与历史行为一致的兼容预设；
//   - 明确提交空维度数组必须拒绝，不能退化成默认配置；
//   - 维度去空、去重、合法性和固定顺序全部由后端裁决；
//   - no_lesson 是真实材料隔离模式，不只是提示词文案；
//   - 配置最终由数据库触发器计算哈希并禁止创建后修改。

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"tedna/internal/models"
)

// ErrCWAIReviewConfigInvalid 表示启动配置为空、冲突或包含非法值。
var ErrCWAIReviewConfigInvalid = errors.New("课件AI审核启动配置无效")

// CWAIReviewConfigInput 是浏览器选择意图的服务层输入。
//
// 指针用于区分：
//   - nil：旧客户端未提交该字段，使用兼容默认值；
//   - 非nil空数组：用户明确没有选择任何维度，必须拒绝。
type CWAIReviewConfigInput struct {
	ReviewDimensions           *[]string
	CustomDimensionDescription *string
	LessonReferenceMode        *string
}

// CWAIReviewConfigSnapshot 是后端规范化后的会话配置。
type CWAIReviewConfigSnapshot struct {
	SchemaVersion              int
	ReviewDimensions           []string
	ReviewDimensionsJSON       string
	CustomDimensionDescription string
	LessonReferenceMode        string
}

// NormalizeCWAIReviewConfig 规范化并校验审核启动配置。
func NormalizeCWAIReviewConfig(
	input *CWAIReviewConfigInput,
) (*CWAIReviewConfigSnapshot, error) {
	dimensions := models.CoursewareAIReviewDefaultDimensions()
	customDescription := ""
	referenceMode := models.CWAIReviewLessonReferenceCurrentCompatible

	if input != nil {
		if input.ReviewDimensions != nil {
			normalized, err := normalizeCWAIReviewDimensions(*input.ReviewDimensions)
			if err != nil {
				return nil, err
			}
			dimensions = normalized
		}

		if input.CustomDimensionDescription != nil {
			customDescription = strings.TrimSpace(*input.CustomDimensionDescription)
		}

		if input.LessonReferenceMode != nil {
			referenceMode = strings.ToLower(strings.TrimSpace(*input.LessonReferenceMode))
			if referenceMode == "" {
				return nil, fmt.Errorf("%w：请选择教案参考模式", ErrCWAIReviewConfigInvalid)
			}
		}
	}

	if !models.IsCWAIReviewLessonReferenceMode(referenceMode) {
		return nil, fmt.Errorf("%w：教案参考模式不受支持", ErrCWAIReviewConfigInvalid)
	}

	hasCustom := false
	for _, dimension := range dimensions {
		if dimension == models.CWAIReviewDimensionCustom {
			hasCustom = true
			break
		}
	}

	if hasCustom && customDescription == "" {
		return nil, fmt.Errorf("%w：选择自定义审核维度时必须填写说明", ErrCWAIReviewConfigInvalid)
	}
	if !hasCustom && customDescription != "" {
		return nil, fmt.Errorf("%w：未选择自定义审核维度时不能填写自定义说明", ErrCWAIReviewConfigInvalid)
	}

	encodedDimensions, err := json.Marshal(dimensions)
	if err != nil {
		return nil, fmt.Errorf("序列化课件AI审核维度失败: %w", err)
	}

	return &CWAIReviewConfigSnapshot{
		SchemaVersion:              models.CWAIReviewConfigSchemaVersion,
		ReviewDimensions:           append([]string{}, dimensions...),
		ReviewDimensionsJSON:       string(encodedDimensions),
		CustomDimensionDescription: customDescription,
		LessonReferenceMode:        referenceMode,
	}, nil
}

// normalizeCWAIReviewDimensions 校验浏览器提交的维度，并按平台固定顺序输出。
func normalizeCWAIReviewDimensions(input []string) ([]string, error) {
	if len(input) == 0 {
		return nil, fmt.Errorf("%w：请至少选择一个审核维度", ErrCWAIReviewConfigInvalid)
	}

	selected := make(map[string]bool, len(input))

	for _, raw := range input {
		dimension := strings.ToLower(strings.TrimSpace(raw))
		if dimension == "" {
			return nil, fmt.Errorf("%w：审核维度不能为空", ErrCWAIReviewConfigInvalid)
		}
		if !models.IsCWAIReviewDimension(dimension) {
			return nil, fmt.Errorf("%w：存在不受支持的审核维度", ErrCWAIReviewConfigInvalid)
		}
		if selected[dimension] {
			return nil, fmt.Errorf("%w：审核维度不能重复", ErrCWAIReviewConfigInvalid)
		}

		selected[dimension] = true
	}

	result := make([]string, 0, len(selected))
	for _, dimension := range models.CoursewareAIReviewAllDimensions() {
		if selected[dimension] {
			result = append(result, dimension)
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("%w：请至少选择一个审核维度", ErrCWAIReviewConfigInvalid)
	}

	return result, nil
}

// cwAIReviewConfigFromSession 从数据库会话中重建并复核规范化配置。
//
// 数据库已经有约束和不可变触发器，本函数仍保留服务层防御，避免未来绕过
// 正式仓储的内部代码把异常配置带入真实AI输入。
func cwAIReviewConfigFromSession(
	session *models.CoursewareAIReviewSession,
) (*CWAIReviewConfigSnapshot, error) {
	if session == nil {
		return nil, fmt.Errorf("%w：缺少审核会话", ErrCWAIReviewConfigInvalid)
	}
	if session.ReviewConfigSchemaVersion != models.CWAIReviewConfigSchemaVersion {
		return nil, fmt.Errorf("%w：审核配置协议版本不受支持", ErrCWAIReviewConfigInvalid)
	}

	var dimensions []string
	if err := json.Unmarshal([]byte(session.ReviewDimensionsJSON), &dimensions); err != nil {
		return nil, fmt.Errorf("%w：审核维度快照无法解析", ErrCWAIReviewConfigInvalid)
	}

	customDescription := session.CustomDimensionDescription
	referenceMode := session.LessonReferenceMode

	return NormalizeCWAIReviewConfig(
		&CWAIReviewConfigInput{
			ReviewDimensions:           &dimensions,
			CustomDimensionDescription: &customDescription,
			LessonReferenceMode:        &referenceMode,
		},
	)
}

// cwAIReviewUsesLessonMaterials 判断本次会话是否允许读取教案类材料。
func cwAIReviewUsesLessonMaterials(config *CWAIReviewConfigSnapshot) bool {
	return config != nil &&
		config.LessonReferenceMode != models.CWAIReviewLessonReferenceNoLesson
}

// cwAIReviewConfigManifest 构造可安全写入会话内部清单和AI输入的配置事实。
func cwAIReviewConfigManifest(config *CWAIReviewConfigSnapshot) map[string]interface{} {
	if config == nil {
		return map[string]interface{}{}
	}

	dimensions := make([]map[string]string, 0, len(config.ReviewDimensions))
	for _, code := range config.ReviewDimensions {
		dimensions = append(
			dimensions,
			map[string]string{
				"code":  code,
				"label": cwAIReviewDimensionLabel(code),
			},
		)
	}

	return map[string]interface{}{
		"schema_version":               config.SchemaVersion,
		"review_dimensions":            append([]string{}, config.ReviewDimensions...),
		"review_dimension_items":       dimensions,
		"custom_dimension_description": config.CustomDimensionDescription,
		"lesson_reference_mode":        config.LessonReferenceMode,
		"lesson_reference_label":       cwAIReviewLessonReferenceLabel(config.LessonReferenceMode),
		"uses_lesson_materials":        cwAIReviewUsesLessonMaterials(config),
	}
}

func cwAIReviewDimensionLabel(code string) string {
	switch code {
	case models.CWAIReviewDimensionTeachingLogic:
		return "教学逻辑"
	case models.CWAIReviewDimensionTechnicalImplementation:
		return "技术实现"
	case models.CWAIReviewDimensionInteractionExperience:
		return "交互体验"
	case models.CWAIReviewDimensionLessonAlignment:
		return "教案一致性"
	case models.CWAIReviewDimensionAuthenticity:
		return "真实性"
	case models.CWAIReviewDimensionKnowledgeAccuracy:
		return "知识严谨性"
	case models.CWAIReviewDimensionPageReadability:
		return "页面可读性"
	case models.CWAIReviewDimensionOperationalUsability:
		return "操作可用性"
	case models.CWAIReviewDimensionCustom:
		return "自定义维度"
	default:
		return code
	}
}

func cwAIReviewLessonReferenceLabel(mode string) string {
	switch mode {
	case models.CWAIReviewLessonReferenceCurrentCompatible:
		return "现行兼容"
	case models.CWAIReviewLessonReferenceStrictAlignment:
		return "严格一致"
	case models.CWAIReviewLessonReferenceLessonIntent:
		return "参考教案意图"
	case models.CWAIReviewLessonReferenceNoLesson:
		return "不使用教案"
	default:
		return mode
	}
}
