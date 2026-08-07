package models

// lesson_plan_knowledge_lineage.go — 教案课程锚点与知识脉络模型
//
// 核心边界：
//   - 先确认课文、教学范围、目标和知识点，再提取知识脉络；
//   - 当前知识节点必须逐项指向教师确认的知识点；
//   - 课程大纲证据存在时必须保存真实原文摘录；
//   - 大纲没有直接证据时必须写EvidenceGap，不能补猜；
//   - 后续阶段只使用active短版快照，不重复注入大纲全文。

import (
	"strings"
	"time"
)

const (
	LessonPlanKnowledgeLineageStatusGenerating = "generating"
	LessonPlanKnowledgeLineageStatusActive     = "active"
	LessonPlanKnowledgeLineageStatusStale      = "stale"
	LessonPlanKnowledgeLineageStatusFailed     = "failed"
)

// LessonPlanKnowledgePoint 是教师确认的单个本课知识点。
type LessonPlanKnowledgePoint struct {
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	ExpectedDepth string   `json:"expected_depth"`
	Evidence      []string `json:"evidence"`
	ExcludedDepth string   `json:"excluded_depth,omitempty"`
}

// LessonPlanKnowledgeAnchors 是知识脉络生成之前必须具备的课程锚点。
type LessonPlanKnowledgeAnchors struct {
	LessonObject       string                     `json:"lesson_object"`
	TeachingScope      string                     `json:"teaching_scope"`
	SourceEvidence     []string                   `json:"source_evidence"`
	TeachingObjectives []string                   `json:"teaching_objectives"`
	KnowledgePoints    []LessonPlanKnowledgePoint `json:"knowledge_points"`
	LearningDepth      string                     `json:"learning_depth"`
	ExcludedContent    []string                   `json:"excluded_content"`
	TeacherConfirmed   bool                       `json:"teacher_confirmed"`
}

// HasConfirmedCore 判断课程锚点是否具备生成知识脉络的最小可靠内容。
func (anchors *LessonPlanKnowledgeAnchors) HasConfirmedCore() bool {
	if anchors == nil || !anchors.TeacherConfirmed ||
		strings.TrimSpace(anchors.LessonObject) == "" ||
		strings.TrimSpace(anchors.TeachingScope) == "" ||
		strings.TrimSpace(anchors.LearningDepth) == "" ||
		countNonEmptyKnowledgeStrings(anchors.SourceEvidence) == 0 ||
		countNonEmptyKnowledgeStrings(anchors.TeachingObjectives) == 0 {
		return false
	}

	for _, point := range anchors.KnowledgePoints {
		if strings.TrimSpace(point.Name) != "" &&
			strings.TrimSpace(point.ExpectedDepth) != "" &&
			countNonEmptyKnowledgeStrings(point.Evidence) > 0 {
			return true
		}
	}

	return false
}

// LessonPlanKnowledgeLineageNode 描述知识脉络中的一个节点。
//
// AnchorKnowledgePoint：
//   - current_knowledge中必填；
//   - 必须原样复制教师确认的knowledge_points[].name；
//   - 前置和后续节点可以为空。
//
// OutlineEvidence与EvidenceGap二选一：
//   - 有大纲原文依据时保存原文短句；
//   - 无明确依据时保存证据缺口；
//   - 不得两者都为空。
type LessonPlanKnowledgeLineageNode struct {
	AnchorKnowledgePoint string   `json:"anchor_knowledge_point,omitempty"`
	Name                 string   `json:"name"`
	ExpectedMastery      string   `json:"expected_mastery"`
	Connection           string   `json:"connection"`
	OutlineEvidence      []string `json:"outline_evidence"`
	EvidenceGap          string   `json:"evidence_gap,omitempty"`
	TeachingBoundary     string   `json:"teaching_boundary,omitempty"`
}

// LessonPlanKnowledgeMisconception 是与确认知识点直接相关的认知断层或误区。
type LessonPlanKnowledgeMisconception struct {
	Misconception string `json:"misconception"`
	Cause         string `json:"cause"`
	DiagnosticCue string `json:"diagnostic_cue"`
	Correction    string `json:"correction"`
}

// LessonPlanKnowledgeAssessmentEvidence 描述真正理解知识点的可观察证据。
type LessonPlanKnowledgeAssessmentEvidence struct {
	Evidence      string `json:"evidence"`
	Meaning       string `json:"meaning"`
	FalsePositive string `json:"false_positive,omitempty"`
}

// LessonPlanKnowledgeLineageSnapshot 是围绕确认课程锚点提取的知识脉络。
type LessonPlanKnowledgeLineageSnapshot struct {
	PrerequisiteKnowledge   []LessonPlanKnowledgeLineageNode        `json:"prerequisite_knowledge"`
	PrerequisiteEvidenceGap string                                  `json:"prerequisite_evidence_gap,omitempty"`
	CurrentKnowledge        []LessonPlanKnowledgeLineageNode        `json:"current_knowledge"`
	SubsequentKnowledge     []LessonPlanKnowledgeLineageNode        `json:"subsequent_knowledge"`
	SubsequentEvidenceGap   string                                  `json:"subsequent_evidence_gap,omitempty"`
	Misconceptions          []LessonPlanKnowledgeMisconception      `json:"misconceptions"`
	AssessmentEvidence      []LessonPlanKnowledgeAssessmentEvidence `json:"assessment_evidence"`
	CoherenceRules          []string                                `json:"coherence_rules"`
}

// HasUsableCore 判断脉络是否覆盖本课位置，并诚实处理前置与后续证据。
func (snapshot *LessonPlanKnowledgeLineageSnapshot) HasUsableCore() bool {
	if snapshot == nil || len(snapshot.CurrentKnowledge) == 0 ||
		countNonEmptyKnowledgeStrings(snapshot.CoherenceRules) == 0 {
		return false
	}

	hasPrerequisite := len(snapshot.PrerequisiteKnowledge) > 0 ||
		strings.TrimSpace(snapshot.PrerequisiteEvidenceGap) != ""

	hasSubsequent := len(snapshot.SubsequentKnowledge) > 0 ||
		strings.TrimSpace(snapshot.SubsequentEvidenceGap) != ""

	return hasPrerequisite && hasSubsequent
}

// LessonPlanKnowledgeLineage 是数据库中的知识脉络快照记录。
type LessonPlanKnowledgeLineage struct {
	ID                            string     `json:"id"`
	LessonPlanID                  string     `json:"lesson_plan_id"`
	CourseOutlineID               string     `json:"course_outline_id"`
	Status                        string     `json:"status"`
	AnchorSnapshot                string     `json:"anchor_snapshot"`
	LineageSnapshot               string     `json:"lineage_snapshot"`
	ContextText                   string     `json:"context_text"`
	AnchorHash                    string     `json:"anchor_hash"`
	OutlineHash                   string     `json:"outline_hash"`
	ConfirmedStageCode            string     `json:"confirmed_stage_code"`
	ConfirmedStageOutputUpdatedAt *time.Time `json:"confirmed_stage_output_updated_at,omitempty"`
	ModelUsed                     string     `json:"model_used"`
	TokensUsed                    int        `json:"tokens_used"`
	ErrorMessage                  string     `json:"error_message"`
	GeneratedAt                   *time.Time `json:"generated_at,omitempty"`
	CreatedAt                     *time.Time `json:"created_at,omitempty"`
	UpdatedAt                     *time.Time `json:"updated_at,omitempty"`
}

// IsActiveUsable 判断记录是否具备进入后续阶段提示词的基本条件。
func (lineage *LessonPlanKnowledgeLineage) IsActiveUsable() bool {
	return lineage != nil &&
		lineage.Status == LessonPlanKnowledgeLineageStatusActive &&
		strings.TrimSpace(lineage.LessonPlanID) != "" &&
		strings.TrimSpace(lineage.CourseOutlineID) != "" &&
		strings.TrimSpace(lineage.AnchorSnapshot) != "" &&
		strings.TrimSpace(lineage.AnchorSnapshot) != "{}" &&
		strings.TrimSpace(lineage.LineageSnapshot) != "" &&
		strings.TrimSpace(lineage.LineageSnapshot) != "{}" &&
		strings.TrimSpace(lineage.ContextText) != "" &&
		len(strings.TrimSpace(lineage.AnchorHash)) == 64 &&
		len(strings.TrimSpace(lineage.OutlineHash)) == 64 &&
		lineage.ConfirmedStageOutputUpdatedAt != nil &&
		lineage.GeneratedAt != nil
}

func countNonEmptyKnowledgeStrings(values []string) int {
	count := 0

	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			count++
		}
	}

	return count
}
