package services

// lesson_plan_knowledge_anchor.go — 教学分析完成后的课程锚点提取
//
// 本文件只回答：“教师在教学分析阶段最终确认，这一课教什么？”
//
// 硬规则：
//   - 不读取课程大纲补写课程锚点；
//   - 不根据课题名称或模型记忆猜课文；
//   - 不把未被教师选择的备选方案当作结论；
//   - source_evidence和knowledge_points[].evidence必须是分析对话原文摘录；
//   - 后端会逐条验证证据是否真实存在于分析对话；
//   - AI首次违反结构或证据协议时只允许一次受控重试，二次仍失败必须继续拒绝推进。

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	aiClient "tedna/internal/ai"
	"tedna/internal/models"
	"tedna/internal/repository"
)

var (
	ErrLessonPlanKnowledgeAnchorsIncomplete = errors.New(
		"教学分析尚未形成可确认的课程锚点",
	)
	ErrLessonPlanKnowledgeAnchorExtractionUnavailable = errors.New(
		"课程锚点提取暂时不可用",
	)
)

const (
	lessonPlanKnowledgeAnchorMaxTranscriptRunes = 60000
	lessonPlanKnowledgeAnchorMaxTokens          = 3600
)

type lessonPlanKnowledgeAnchorAIResult struct {
	Ready          bool                              `json:"ready"`
	MissingFields  []string                          `json:"missing_fields"`
	AmbiguityNotes []string                          `json:"ambiguity_notes"`
	Anchors        models.LessonPlanKnowledgeAnchors `json:"anchors"`
}

type lessonPlanKnowledgeAnchorExtraction struct {
	Source             *repository.LessonPlanKnowledgeLineageSource
	Anchors            *models.LessonPlanKnowledgeAnchors
	AnchorJSON         string
	AnchorHash         string
	AnalysisSourceHash string
	ModelUsed          string
	TokensUsed         int
}

// extractConfirmedLessonPlanKnowledgeAnchors 从确认后的教学分析对话提取课程锚点。
func (s *WorkshopStageService) extractConfirmedLessonPlanKnowledgeAnchors(
	ctx context.Context,
	lessonPlanID string,
) (*lessonPlanKnowledgeAnchorExtraction, error) {
	source, err := repository.LoadLessonPlanKnowledgeLineageSource(
		ctx,
		lessonPlanID,
		"analyze",
	)
	if err != nil {
		return nil, err
	}

	messages, err := repository.GetCurrentStageMessages(ctx, lessonPlanID)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: 读取教学分析对话失败: %v",
			ErrLessonPlanKnowledgeAnchorExtractionUnavailable,
			err,
		)
	}

	transcript, teacherMessageCount, err :=
		buildConfirmedKnowledgeAnchorTranscript(messages)
	if err != nil {
		return nil, err
	}
	if teacherMessageCount == 0 {
		return nil, fmt.Errorf(
			"%w: 教师尚未在教学分析阶段提供实际确认信息",
			ErrLessonPlanKnowledgeAnchorsIncomplete,
		)
	}

	// 这里只把已经通过确定性前置校验的可信来源交给AI。
	// 首轮若仅因为JSON协议、核心字段或原文证据校验失败，
	// 由独立辅助模块基于同一份输入执行且仅执行一次受控重试。
	parsed, modelUsed, tokensUsed, err :=
		s.extractLessonPlanKnowledgeAnchorAIWithRetry(
			ctx,
			source,
			transcript,
		)
	if err != nil {
		return nil, err
	}

	anchorBytes, err := json.Marshal(&parsed.Anchors)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: 序列化确认课程锚点失败: %v",
			ErrLessonPlanKnowledgeAnchorExtractionUnavailable,
			err,
		)
	}

	anchorJSON := string(anchorBytes)
	analysisSourceHash := source.AnalysisSourceHash()

	if len(analysisSourceHash) != 64 {
		return nil, fmt.Errorf(
			"%w: 教学分析来源哈希生成失败",
			ErrLessonPlanKnowledgeAnchorExtractionUnavailable,
		)
	}

	return &lessonPlanKnowledgeAnchorExtraction{
		Source:             source,
		Anchors:            &parsed.Anchors,
		AnchorJSON:         anchorJSON,
		AnchorHash:         hashLessonPlanKnowledgePayload(anchorJSON),
		AnalysisSourceHash: analysisSourceHash,
		ModelUsed:          modelUsed,
		TokensUsed:         tokensUsed,
	}, nil
}

const lessonPlanKnowledgeAnchorSystemPrompt = `你是“教学分析课程锚点提取器”。

你的唯一任务是从教师已经完成的教学分析对话中，提取本课真正敲定的课程锚点。

绝对禁止：
1. 不得根据课题名称、常识或模型记忆猜课文内容。
2. 不得读取、假设或引用课程大纲内容。
3. 不得把AI提出过但教师没有选择的备选方案当作正式结论。
4. 不得为了返回ready=true而补写缺失目标、知识点或学习深度。
5. 不得把教学活动、课堂形式或教学策略误当成知识点。
6. 不得用“了解、掌握、提升能力”等模糊词代替具体学习深度。
7. source_evidence和knowledge_points[].evidence只能填写分析对话中的原文短句摘录，不能概括、改写或伪造。
8. 当教师只说“可以、同意、按这个来”时，证据应同时引用此前被确认的AI方案短句与教师确认短句。

只有以下内容全部明确时ready才可以为true：
- lesson_object：具体课文、章节、知识主题或明确课时对象；
- teaching_scope：本课实际教学范围；
- source_evidence：对话中的真实原文摘录；
- teaching_objectives：已经确认的教学目标；
- knowledge_points：已经确认的知识点，每项有expected_depth和原文证据；
- learning_depth：整节课总体达到的学习深度。

严格输出一个JSON对象：
{
  "ready": true或false,
  "missing_fields": ["尚未确定的必需字段"],
  "ambiguity_notes": ["仍有多个方案或表述冲突的地方"],
  "anchors": {
    "lesson_object": "",
    "teaching_scope": "",
    "source_evidence": ["分析对话中的原文短句"],
    "teaching_objectives": [],
    "knowledge_points": [
      {
        "name": "",
        "description": "",
        "expected_depth": "",
        "evidence": ["分析对话中的原文短句"],
        "excluded_depth": ""
      }
    ],
    "learning_depth": "",
    "excluded_content": [],
    "teacher_confirmed": false
  }
}

teacher_confirmed始终输出false，由后端根据教师完成阶段事件写入。
只输出JSON，不输出解释、Markdown、代码围栏或隐藏推理。`

func buildKnowledgeAnchorIncompleteError(
	missingFields []string,
	ambiguityNotes []string,
) error {
	missing := normalizeKnowledgeStringList(missingFields, 12, 180)
	ambiguities := normalizeKnowledgeStringList(ambiguityNotes, 12, 180)
	details := append(missing, ambiguities...)

	if len(details) == 0 {
		details = []string{
			"需要明确课文或教学范围、来源证据、教学目标、知识点及学习深度",
		}
	}

	return fmt.Errorf(
		"%w: %s",
		ErrLessonPlanKnowledgeAnchorsIncomplete,
		strings.Join(details, "；"),
	)
}

// validateKnowledgeAnchorEvidenceAgainstTranscript 验证每条证据都真实存在于分析对话。
func validateKnowledgeAnchorEvidenceAgainstTranscript(
	anchors *models.LessonPlanKnowledgeAnchors,
	transcript string,
) error {
	if anchors == nil {
		return fmt.Errorf(
			"%w: 课程锚点为空",
			ErrLessonPlanKnowledgeAnchorsIncomplete,
		)
	}

	for _, evidence := range anchors.SourceEvidence {
		if !containsKnowledgeEvidenceFragment(transcript, evidence) {
			return fmt.Errorf(
				"%w: 来源证据无法在教学分析对话中找到：%s",
				ErrLessonPlanKnowledgeAnchorsIncomplete,
				evidence,
			)
		}
	}

	for _, point := range anchors.KnowledgePoints {
		for _, evidence := range point.Evidence {
			if !containsKnowledgeEvidenceFragment(transcript, evidence) {
				return fmt.Errorf(
					"%w: 知识点“%s”的证据无法在教学分析对话中找到：%s",
					ErrLessonPlanKnowledgeAnchorsIncomplete,
					point.Name,
					evidence,
				)
			}
		}
	}

	return nil
}

// containsKnowledgeEvidenceFragment 使用规范化空白后的精确子串进行证据验证。
func containsKnowledgeEvidenceFragment(source string, fragment string) bool {
	normalizedSource := normalizeKnowledgeEvidenceSearchText(source)
	normalizedFragment := normalizeKnowledgeEvidenceSearchText(fragment)

	return len([]rune(normalizedFragment)) >= 4 &&
		strings.Contains(normalizedSource, normalizedFragment)
}

func normalizeKnowledgeEvidenceSearchText(value string) string {
	return strings.ToLower(
		strings.Join(
			strings.Fields(strings.TrimSpace(value)),
			"",
		),
	)
}

func buildConfirmedKnowledgeAnchorTranscript(
	messages []*models.ConversationMessage,
) (string, int, error) {
	var builder strings.Builder
	teacherMessageCount := 0

	for _, message := range messages {
		if message == nil {
			continue
		}

		content := strings.TrimSpace(message.Content)
		if content == "" ||
			string(message.Role) == "system" ||
			isStageAutoTriggerContent(content) {
			continue
		}

		role := "AI助手"
		if message.Role == models.ConvRoleUser {
			role = "教师"
			teacherMessageCount++
		}

		builder.WriteString(role)
		builder.WriteString("：")
		builder.WriteString(content)
		builder.WriteString("\n\n")
	}

	transcript := strings.TrimSpace(builder.String())

	if len([]rune(transcript)) > lessonPlanKnowledgeAnchorMaxTranscriptRunes {
		return "", 0, fmt.Errorf(
			"%w: 教学分析对话过长，无法在不截断确认依据的情况下安全提取，请先形成明确分析定稿",
			ErrLessonPlanKnowledgeAnchorsIncomplete,
		)
	}

	return transcript, teacherMessageCount, nil
}

func normalizeLessonPlanKnowledgeAnchors(
	anchors *models.LessonPlanKnowledgeAnchors,
) {
	if anchors == nil {
		return
	}

	anchors.LessonObject = normalizeKnowledgeText(anchors.LessonObject, 500)
	anchors.TeachingScope = normalizeKnowledgeText(anchors.TeachingScope, 800)
	anchors.LearningDepth = normalizeKnowledgeText(anchors.LearningDepth, 800)
	anchors.SourceEvidence = normalizeKnowledgeStringList(
		anchors.SourceEvidence,
		12,
		300,
	)
	anchors.TeachingObjectives = normalizeKnowledgeStringList(
		anchors.TeachingObjectives,
		12,
		400,
	)
	anchors.ExcludedContent = normalizeKnowledgeStringList(
		anchors.ExcludedContent,
		12,
		300,
	)

	points := make(
		[]models.LessonPlanKnowledgePoint,
		0,
		len(anchors.KnowledgePoints),
	)
	seenNames := make(map[string]struct{})

	for _, point := range anchors.KnowledgePoints {
		point.Name = normalizeKnowledgeText(point.Name, 200)
		point.Description = normalizeKnowledgeText(point.Description, 500)
		point.ExpectedDepth = normalizeKnowledgeText(point.ExpectedDepth, 500)
		point.ExcludedDepth = normalizeKnowledgeText(point.ExcludedDepth, 400)
		point.Evidence = normalizeKnowledgeStringList(point.Evidence, 8, 300)

		nameKey := normalizeKnowledgeEvidenceSearchText(point.Name)
		if point.Name == "" || point.ExpectedDepth == "" ||
			len(point.Evidence) == 0 || nameKey == "" {
			continue
		}
		if _, exists := seenNames[nameKey]; exists {
			continue
		}

		seenNames[nameKey] = struct{}{}
		points = append(points, point)

		if len(points) >= 16 {
			break
		}
	}

	anchors.KnowledgePoints = points
}

func normalizeKnowledgeStringList(
	values []string,
	limit int,
	maxRunes int,
) []string {
	output := make([]string, 0, len(values))

	for _, value := range values {
		value = normalizeKnowledgeText(value, maxRunes)
		if value == "" {
			continue
		}

		duplicate := false
		for _, current := range output {
			if current == value {
				duplicate = true
				break
			}
		}
		if duplicate {
			continue
		}

		output = append(output, value)
		if len(output) >= limit {
			break
		}
	}

	return output
}

func normalizeKnowledgeText(value string, maxRunes int) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	runes := []rune(value)
	if maxRunes > 0 && len(runes) > maxRunes {
		return string(runes[:maxRunes]) + "…"
	}

	return value
}

func hashLessonPlanKnowledgePayload(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func buildLessonPlanKnowledgeTraceContext(
	ctx context.Context,
	lessonPlanID string,
	authorID string,
) *aiClient.TraceContext {
	trace := &aiClient.TraceContext{
		SceneCode: models.SceneLessonPlanHarness,
	}

	lessonPlanID = strings.TrimSpace(lessonPlanID)
	if lessonPlanID != "" {
		trace.LessonPlanID = &lessonPlanID
	}

	// 隐藏提取不携带UserID，避免重复消耗教师个人积分。
	// SchoolID继续保留，用于学校授权的模型分流。
	schoolID, err := repository.GetSchoolIDByUserID(
		ctx,
		strings.TrimSpace(authorID),
	)
	if err == nil {
		schoolID = strings.TrimSpace(schoolID)
		if schoolID != "" {
			trace.SchoolID = &schoolID
		}
	}

	return trace
}
