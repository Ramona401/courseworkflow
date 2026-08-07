package models

// lesson_plan_context_capsule.go — 备课核心共识胶囊领域协议
//
// 产品边界：
//   - 胶囊不是附件挂载清单，也不是要求教师维护的表单；
//   - 教师通过自然语言形成、修正、否定和恢复备课共识；
//   - 课本核心与课程大纲知识脉络属于权威课程来源；
//   - 已被纠正、否定或替代的内容进入负向记忆，后续不得重复确认；
//   - 普通AI对话只读取active短版胶囊，必要时才沿证据路由回溯原文；
//   - 教师端只展示安全、轻量的“本课共识”，不展示提示词、Token和机械文件状态。

import "time"

const (
	LessonPlanContextCapsuleSchemaVersion = 1

	LessonPlanContextCapsuleStatusActive = "active"
	LessonPlanContextCapsuleStatusStale  = "stale"
	LessonPlanContextCapsuleStatusFailed = "failed"

	LessonPlanContextCapsuleItemStateActive     = "active"
	LessonPlanContextCapsuleItemStateCandidate  = "candidate"
	LessonPlanContextCapsuleItemStateDeferred   = "deferred"
	LessonPlanContextCapsuleItemStateSuperseded = "superseded"

	LessonPlanContextCapsuleAuthorityTeacherExplicit       = "teacher_explicit"
	LessonPlanContextCapsuleAuthoritySourceVerified        = "source_verified"
	LessonPlanContextCapsuleAuthorityTeacherSourceConfirmed = "teacher_source_confirmed"
	LessonPlanContextCapsuleAuthorityAIInferred            = "ai_inferred"

	LessonPlanContextCapsuleSourceTextbookPage     = "textbook_page"
	LessonPlanContextCapsuleSourceCourseOutline    = "course_outline"
	LessonPlanContextCapsuleSourceTeacherMessage   = "teacher_message"
	LessonPlanContextCapsuleSourceStageOutput      = "stage_output"
	LessonPlanContextCapsuleSourceUnitPlan         = "unit_plan"
	LessonPlanContextCapsuleSourceClassProfile     = "class_profile"
	LessonPlanContextCapsuleSourceReferenceMaterial = "reference_material"
	LessonPlanContextCapsuleSourceSystem           = "system"

	LessonPlanContextCapsuleChangeAdd       = "add"
	LessonPlanContextCapsuleChangeRefine    = "refine"
	LessonPlanContextCapsuleChangeCorrect   = "correct"
	LessonPlanContextCapsuleChangeReplace   = "replace"
	LessonPlanContextCapsuleChangeReject    = "reject"
	LessonPlanContextCapsuleChangeDefer     = "defer"
	LessonPlanContextCapsuleChangeRestore   = "restore"
	LessonPlanContextCapsuleChangeStrengthen = "strengthen"
)

// LessonPlanContextCapsule 是数据库中每个教案当前唯一的胶囊快照。
type LessonPlanContextCapsule struct {
	ID               string     `json:"id"`
	LessonPlanID     string     `json:"lesson_plan_id"`
	Status           string     `json:"status"`
	Version          int        `json:"version"`
	SchemaVersion    int        `json:"schema_version"`
	CurrentStageCode string     `json:"current_stage_code"`
	CapsuleJSON      string     `json:"capsule_json"`
	DisplayJSON      string     `json:"display_json"`
	ContextText      string     `json:"context_text"`
	SourceManifest   string     `json:"source_manifest"`
	SourceHash       string     `json:"source_hash"`
	LastTurnID       string     `json:"last_turn_id"`
	LastUpdateReason string     `json:"last_update_reason"`
	ErrorMessage     string     `json:"error_message"`
	GeneratedAt      *time.Time `json:"generated_at,omitempty"`
	CreatedAt        *time.Time `json:"created_at,omitempty"`
	UpdatedAt        *time.Time `json:"updated_at,omitempty"`
}

// IsActiveUsable 判断胶囊是否可进入AI运行上下文。
func (capsule *LessonPlanContextCapsule) IsActiveUsable() bool {
	return capsule != nil &&
		capsule.Status == LessonPlanContextCapsuleStatusActive &&
		capsule.Version >= 1 &&
		capsule.SchemaVersion >= 1 &&
		capsule.LessonPlanID != "" &&
		capsule.CapsuleJSON != "" &&
		capsule.CapsuleJSON != "{}" &&
		capsule.SourceManifest != "" &&
		capsule.SourceManifest != "{}" &&
		capsule.ContextText != "" &&
		len(capsule.SourceHash) == 64 &&
		capsule.GeneratedAt != nil
}

// LessonPlanContextCapsuleVersion 是不可变历史版本。
type LessonPlanContextCapsuleVersion struct {
	ID               string     `json:"id"`
	LessonPlanID     string     `json:"lesson_plan_id"`
	Version          int        `json:"version"`
	SchemaVersion    int        `json:"schema_version"`
	CurrentStageCode string     `json:"current_stage_code"`
	CapsuleJSON      string     `json:"capsule_json"`
	DisplayJSON      string     `json:"display_json"`
	ContextText      string     `json:"context_text"`
	SourceManifest   string     `json:"source_manifest"`
	SourceHash       string     `json:"source_hash"`
	LastTurnID       string     `json:"last_turn_id"`
	UpdateReason     string     `json:"update_reason"`
	CreatedAt        *time.Time `json:"created_at,omitempty"`
}

// LessonPlanContextCapsuleEvidence 是一个胶囊原子条目的原文召回路由。
type LessonPlanContextCapsuleEvidence struct {
	ID             string                 `json:"id"`
	LessonPlanID   string                 `json:"lesson_plan_id"`
	CapsuleVersion int                    `json:"capsule_version"`
	ItemKey        string                 `json:"item_key"`
	SourceType     string                 `json:"source_type"`
	SourceID       string                 `json:"source_id"`
	SourceTitle    string                 `json:"source_title"`
	Locator        map[string]interface{} `json:"locator"`
	SourceHash     string                 `json:"source_hash"`
	ExcerptHash    string                 `json:"excerpt_hash"`
	EvidenceExcerpt string                `json:"evidence_excerpt"`
	Authority      string                 `json:"authority"`
	CreatedAt      *time.Time             `json:"created_at,omitempty"`
}

// LessonPlanContextCapsuleItem 是胶囊中的稳定原子记忆。
//
// DoNotReconfirm用于已经纠正、否定或明确确认过的内容。除非教师主动恢复，
// 后续AI不得把同一问题重新包装成待确认问题。
type LessonPlanContextCapsuleItem struct {
	Key                string   `json:"key"`
	Title              string   `json:"title"`
	Content            string   `json:"content"`
	State              string   `json:"state"`
	Authority          string   `json:"authority"`
	Importance         int      `json:"importance"`
	ApplicableStages   []string `json:"applicable_stages"`
	SourceKeys         []string `json:"source_keys"`
	DoNotReconfirm     bool     `json:"do_not_reconfirm"`
	ReplacedBy         string   `json:"replaced_by,omitempty"`
	UpdatedByTurnID    string   `json:"updated_by_turn_id,omitempty"`
}

// LessonPlanContextCapsuleStageFocus 只收窄当前注意力，不切断跨阶段记忆。
type LessonPlanContextCapsuleStageFocus struct {
	StageCode       string   `json:"stage_code"`
	CurrentTask     string   `json:"current_task"`
	CarryForwardKeys []string `json:"carry_forward_keys"`
	AvoidRepeatingKeys []string `json:"avoid_repeating_keys"`
}

// LessonPlanContextCapsuleDocument 是后端正式使用的结构化胶囊。
type LessonPlanContextCapsuleDocument struct {
	SchemaVersion     int                            `json:"schema_version"`
	Summary           string                         `json:"summary"`
	CourseCore        []LessonPlanContextCapsuleItem `json:"course_core"`
	TeachingConsensus []LessonPlanContextCapsuleItem `json:"teaching_consensus"`
	Constraints       []LessonPlanContextCapsuleItem `json:"constraints"`
	OpenQuestions     []LessonPlanContextCapsuleItem `json:"open_questions"`
	DeferredItems     []LessonPlanContextCapsuleItem `json:"deferred_items"`
	SupersededItems   []LessonPlanContextCapsuleItem `json:"superseded_items"`
	StageFocus        LessonPlanContextCapsuleStageFocus `json:"stage_focus"`
}

// LessonPlanContextCapsuleChange 描述本轮发生的最小差异。
type LessonPlanContextCapsuleChange struct {
	Operation string `json:"operation"`
	ItemKey   string `json:"item_key"`
	Summary   string `json:"summary"`
}

// LessonPlanContextCapsuleEvidenceBinding 是AI对来源键的显式绑定。
type LessonPlanContextCapsuleEvidenceBinding struct {
	ItemKey    string   `json:"item_key"`
	SourceKeys []string `json:"source_keys"`
}

// LessonPlanContextCapsuleAIResult 是旁路更新器要求AI返回的完整协议。
type LessonPlanContextCapsuleAIResult struct {
	UpdateReason     string                                    `json:"update_reason"`
	Capsule          LessonPlanContextCapsuleDocument          `json:"capsule"`
	Changes          []LessonPlanContextCapsuleChange          `json:"changes"`
	EvidenceBindings []LessonPlanContextCapsuleEvidenceBinding `json:"evidence_bindings"`
}

// LessonPlanContextCapsuleDisplaySection 是教师端渐进披露的一个区域。
type LessonPlanContextCapsuleDisplaySection struct {
	Key      string   `json:"key"`
	Title    string   `json:"title"`
	Items    []string `json:"items"`
	Emphasis string   `json:"emphasis,omitempty"`
}

// LessonPlanContextCapsuleDisplayView 是教师端安全视图。
//
// 该结构不包含附件文件名清单、字符数、Token、内部状态码或完整来源正文。
type LessonPlanContextCapsuleDisplayView struct {
	StateLabel   string                                   `json:"state_label"`
	Headline     string                                   `json:"headline"`
	Summary      string                                   `json:"summary"`
	RecentChange string                                   `json:"recent_change,omitempty"`
	Sections     []LessonPlanContextCapsuleDisplaySection `json:"sections"`
}

// LessonPlanContextCapsuleSourceRef 是source_manifest中的轻量来源快照。
type LessonPlanContextCapsuleSourceRef struct {
	Key        string                 `json:"key"`
	SourceType string                 `json:"source_type"`
	SourceID   string                 `json:"source_id"`
	Title      string                 `json:"title"`
	Locator    map[string]interface{} `json:"locator"`
	SourceHash string                 `json:"source_hash"`
}

// LessonPlanContextCapsuleSourceManifest 记录本版本依赖的来源，不保存全文。
type LessonPlanContextCapsuleSourceManifest struct {
	LessonPlanID     string                              `json:"lesson_plan_id"`
	Subject          string                              `json:"subject"`
	Grade            string                              `json:"grade"`
	Topic            string                              `json:"topic"`
	EducationDomain  string                              `json:"education_domain"`
	CurrentStageCode string                              `json:"current_stage_code"`
	Sources          []LessonPlanContextCapsuleSourceRef `json:"sources"`
}

// IsValidLessonPlanContextCapsuleStatus 校验数据库状态。
func IsValidLessonPlanContextCapsuleStatus(value string) bool {
	switch value {
	case LessonPlanContextCapsuleStatusActive,
		LessonPlanContextCapsuleStatusStale,
		LessonPlanContextCapsuleStatusFailed:
		return true
	default:
		return false
	}
}

// IsValidLessonPlanContextCapsuleAuthority 校验证据可信等级。
func IsValidLessonPlanContextCapsuleAuthority(value string) bool {
	switch value {
	case LessonPlanContextCapsuleAuthorityTeacherExplicit,
		LessonPlanContextCapsuleAuthoritySourceVerified,
		LessonPlanContextCapsuleAuthorityTeacherSourceConfirmed,
		LessonPlanContextCapsuleAuthorityAIInferred:
		return true
	default:
		return false
	}
}

// IsValidLessonPlanContextCapsuleSourceType 校验证据来源类型。
func IsValidLessonPlanContextCapsuleSourceType(value string) bool {
	switch value {
	case LessonPlanContextCapsuleSourceTextbookPage,
		LessonPlanContextCapsuleSourceCourseOutline,
		LessonPlanContextCapsuleSourceTeacherMessage,
		LessonPlanContextCapsuleSourceStageOutput,
		LessonPlanContextCapsuleSourceUnitPlan,
		LessonPlanContextCapsuleSourceClassProfile,
		LessonPlanContextCapsuleSourceReferenceMaterial,
		LessonPlanContextCapsuleSourceSystem:
		return true
	default:
		return false
	}
}
