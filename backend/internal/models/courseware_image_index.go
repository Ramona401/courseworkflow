package models

// courseware_image_index.go — 课件图片 IAOCI 索引与图片关系模型
//
// IAOCI 协议格式：
//
//	IK:@I-3F8A91C4D2E7|IV:1|IT:I|UR:KN|CT:0|SB:O|AR:H|RC:0
//	[F]本图焦点和教学用途
//	[L]本图构图、镜头和留白
//	[A]艺术媒介、材质、色彩、光影和渲染质量
//	[C]固定人物、动物或标志性物体；无则写Ø
//	[S]本图环境；默认只作用于本图
//	[E]尺寸、画幅、文字和质量要求
//	[R]0
//	[N]明确禁止项
//
// 设计原则：
//   - JSON只能作为HTTP传输外壳，不能成为图片语义索引本体；
//   - F和L由本图片槽位决定，优先级高于课程锚点；
//   - 课程锚点默认只继承A；
//   - C仅在本图明确出现对应固定主体时继承；
//   - S和L只有R关系明确授权时才能跨图继承；
//   - R关系使用稳定image_key，不依赖会变化的page_number。

import (
	"strings"
	"time"
)

// IAOCI索引类型。
const (
	CWImageIndexTypeAnchor = "A"
	CWImageIndexTypeImage  = "I"
	CWImageIndexTypeVideo  = "V"
)

// IAOCI教学用途。
const (
	CWImageUsageCover     = "CV"
	CWImageUsageKnowledge = "KN"
	CWImageUsageStory     = "ST"
	CWImageUsageExperiment = "EX"
	CWImageUsageDiagram   = "DG"
	CWImageUsageBackground = "BG"
)

// IAOCI主体类别。
const (
	CWImageSubjectNone   = "N"
	CWImageSubjectPerson = "P"
	CWImageSubjectAnimal = "A"
	CWImageSubjectObject = "O"
	CWImageSubjectMixed  = "M"
)

// IAOCI画幅类别。
const (
	CWImageAspectHorizontal = "H"
	CWImageAspectVertical   = "V"
	CWImageAspectSquare     = "Q"
	CWImageAspectFlexible   = "F"
)

// 图片索引生产状态。
const (
	CWImageIndexStatusPlanned    = "planned"
	CWImageIndexStatusGenerating = "generating"
	CWImageIndexStatusGenerated  = "generated"
	CWImageIndexStatusFailed     = "failed"
	CWImageIndexStatusStale      = "stale"
)

// 图片关系类型。
const (
	CWImageRelationContinue = ">"
	CWImageRelationSameView = "="
	CWImageRelationParallel = "~"
	CWImageRelationContrast = "<>"
	CWImageRelationDetail   = "^"
)

// 图片关系可继承维度。
const (
	CWImageInheritArt       = "A"
	CWImageInheritCharacter = "C"
	CWImageInheritScene     = "S"
	CWImageInheritObject    = "O"
	CWImageInheritLayout    = "L"
)

// CoursewareImageIndex 对应 courseware_image_indexes。
type CoursewareImageIndex struct {
	ID            string  `json:"id"`
	CoursewareID  string  `json:"courseware_id"`
	PageID        *string `json:"page_id"`
	PlaceholderID string  `json:"placeholder_id"`
	ImageKey      string  `json:"image_key"`
	SlotOrder     int     `json:"slot_order"`

	IndexVersion    int    `json:"index_version"`
	IndexType       string `json:"index_type"`
	UsageRole       string `json:"usage_role"`
	ContinuityLevel int    `json:"continuity_level"`
	SubjectType     string `json:"subject_type"`
	AspectRatio     string `json:"aspect_ratio"`
	RelationCount   string `json:"relation_count"`

	FocusText     string `json:"focus_text"`
	LayoutText    string `json:"layout_text"`
	ArtText       string `json:"art_text"`
	CharacterText string `json:"character_text"`
	SceneText     string `json:"scene_text"`
	ExportText    string `json:"export_text"`
	NegativeText  string `json:"negative_text"`

	AOCIText         string  `json:"aoci_text"`
	GenerationPrompt string  `json:"generation_prompt"`
	AssetID          *string `json:"asset_id"`
	Status           string  `json:"status"`
	LastError        string  `json:"last_error"`
	Version          int     `json:"version"`

	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

// CoursewareImageRelationSpec 是IAOCI中一条[R]关系的业务输入。
// TargetImageKey使用稳定图片键，不使用页码。
type CoursewareImageRelationSpec struct {
	TargetImageKey string `json:"target_image_key"`
	RelationCode   string `json:"relation_code"`
	InheritMask    string `json:"inherit_mask"`
	SemanticNote   string `json:"semantic_note"`
}

// CoursewareImageRelation 对应 courseware_image_relations。
type CoursewareImageRelation struct {
	ID                 string `json:"id"`
	CoursewareID       string `json:"courseware_id"`
	SourceImageIndexID string `json:"source_image_index_id"`
	TargetImageIndexID string `json:"target_image_index_id"`
	SourceImageKey     string `json:"source_image_key"`
	TargetImageKey     string `json:"target_image_key"`
	RelationCode       string `json:"relation_code"`
	InheritMask        string `json:"inherit_mask"`
	SemanticNote       string `json:"semantic_note"`
	CreatedAt          *time.Time `json:"created_at"`
}

// ImageAOCI 是解析后的IAOCI协议对象。
// RawText只用于诊断，正式保存前应使用工具层重新格式化。
type ImageAOCI struct {
	ImageKey       string `json:"image_key"`
	IndexVersion   int    `json:"index_version"`
	IndexType      string `json:"index_type"`
	UsageRole      string `json:"usage_role"`
	ContinuityLevel int   `json:"continuity_level"`
	SubjectType    string `json:"subject_type"`
	AspectRatio    string `json:"aspect_ratio"`
	RelationCount  string `json:"relation_count"`

	FocusText     string `json:"focus_text"`
	LayoutText    string `json:"layout_text"`
	ArtText       string `json:"art_text"`
	CharacterText string `json:"character_text"`
	SceneText     string `json:"scene_text"`
	ExportText    string `json:"export_text"`
	NegativeText  string `json:"negative_text"`

	Relations []CoursewareImageRelationSpec `json:"relations"`
	RawText   string                        `json:"-"`
}

// IsValidCWImageIndexType 校验IAOCI索引类型。
func IsValidCWImageIndexType(value string) bool {
	switch value {
	case CWImageIndexTypeAnchor, CWImageIndexTypeImage, CWImageIndexTypeVideo:
		return true
	default:
		return false
	}
}

// IsValidCWImageUsageRole 校验图片教学用途。
func IsValidCWImageUsageRole(value string) bool {
	switch value {
	case CWImageUsageCover,
		CWImageUsageKnowledge,
		CWImageUsageStory,
		CWImageUsageExperiment,
		CWImageUsageDiagram,
		CWImageUsageBackground:
		return true
	default:
		return false
	}
}

// IsValidCWImageSubjectType 校验主体类别。
func IsValidCWImageSubjectType(value string) bool {
	switch value {
	case CWImageSubjectNone,
		CWImageSubjectPerson,
		CWImageSubjectAnimal,
		CWImageSubjectObject,
		CWImageSubjectMixed:
		return true
	default:
		return false
	}
}

// IsValidCWImageAspectRatio 校验画幅类别。
func IsValidCWImageAspectRatio(value string) bool {
	switch value {
	case CWImageAspectHorizontal,
		CWImageAspectVertical,
		CWImageAspectSquare,
		CWImageAspectFlexible:
		return true
	default:
		return false
	}
}

// IsValidCWImageIndexStatus 校验索引生产状态。
func IsValidCWImageIndexStatus(value string) bool {
	switch value {
	case CWImageIndexStatusPlanned,
		CWImageIndexStatusGenerating,
		CWImageIndexStatusGenerated,
		CWImageIndexStatusFailed,
		CWImageIndexStatusStale:
		return true
	default:
		return false
	}
}

// IsValidCWImageRelationCode 校验图片关系类型。
func IsValidCWImageRelationCode(value string) bool {
	switch value {
	case CWImageRelationContinue,
		CWImageRelationSameView,
		CWImageRelationParallel,
		CWImageRelationContrast,
		CWImageRelationDetail:
		return true
	default:
		return false
	}
}

// NormalizeCWImageInheritMask 校验并规范化继承掩码。
// 输出顺序固定为A、C、S、O、L，并自动去重。
func NormalizeCWImageInheritMask(mask string) (string, bool) {
	mask = strings.ToUpper(strings.TrimSpace(mask))
	if mask == "" {
		return "", false
	}

	allowed := map[rune]bool{
		'A': true,
		'C': true,
		'S': true,
		'O': true,
		'L': true,
	}
	selected := make(map[rune]bool, 5)

	for _, code := range mask {
		if !allowed[code] {
			return "", false
		}
		selected[code] = true
	}

	var builder strings.Builder
	for _, code := range []rune{'A', 'C', 'S', 'O', 'L'} {
		if selected[code] {
			builder.WriteRune(code)
		}
	}

	normalized := builder.String()
	return normalized, normalized != ""
}

// CWImageRelationCountCode 根据实际关系数量返回RC字段。
func CWImageRelationCountCode(total int) string {
	switch {
	case total <= 0:
		return "0"
	case total == 1:
		return "1"
	default:
		return "M"
	}
}
