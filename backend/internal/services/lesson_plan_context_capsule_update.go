package services

// lesson_plan_context_capsule_update.go — 备课核心共识胶囊旁路更新编排
//
// 性能原则：
//   - 本模块永远不进入教师主回复的流式关键路径；
//   - 当前轮先使用上一版active胶囊和教师当前消息立即生成回复；
//   - 主回复完成后，旁路形成下一版胶囊，供下一轮读取；
//   - 更新失败只记录日志，不回滚已经完成的主回复；
//   - 已有active胶囊不会因为一次旁路失败而被降级。
//
// 记忆原则：
//   - 课程大纲active知识脉络和课本是课程事实来源；
//   - 教师自然语言负责形成、修正、否定、搁置和恢复教学共识；
//   - AI自己的回复只帮助理解对话，不自动成为教师共识；
//   - 已纠正内容进入负向记忆，后续不得重新提出或再次确认；
//   - 阶段变化只改变运行时注意力，不重写稳定课程记忆。
//
// JSON容错原则：
//   - 首次输出先执行确定性、安全的本地解析与尾逗号清理；
//   - 不擅自补造缺失字段，不对截断JSON进行猜测性闭合；
//   - 首次解析仍失败时，使用原始输入独立重试一次；
//   - 第二次仍失败则保留上一版active胶囊，不影响主回复。
//
// 撰写进度原则：
//   - teaching_consensus只保存教师已经确认的教学决定；
//   - 当前AI刚生成的详细教案正文只进入“已生成待确认”；
//   - 教师下一轮明确确认后，上一版待确认范围才转为已确认；
//   - 稳定教学共识不变但撰写进度真实变化时，也生成新胶囊版本。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	aiClient "tedna/internal/ai"
	"tedna/internal/models"
	"tedna/internal/repository"
)

const (
	lessonPlanContextCapsuleMaxTokens = 5200
	lessonPlanContextCapsuleTimeout   = 75 * time.Second
)

const lessonPlanContextCapsuleSystemPrompt = `你是“备课核心共识胶囊更新器”。

你不会直接回复教师。你只在主回复完成后，旁路维护下一轮要使用的核心记忆。

你会收到：
1. 当前胶囊；
2. 教师本轮明确表达；
3. AI本轮回复，仅用于理解对话结果，AI回复本身不是教师确认；
4. 课程大纲active知识脉络、课本页和前序阶段正式产出等来源证据。

硬规则：
1. 课程大纲知识脉络和课本来源共同约束课程事实。不得用模型常识改写权威来源。
2. 教师可决定教学目标、路径、活动、情境、评价方式、暂缓事项和禁止事项，但不能把错误知识改成课程事实。
3. 教师明确表达可直接形成或修正教学共识，不要求再去弹窗确认。
4. “可以、就这样、按这个来、对”等表达，要结合最近对话理解为对当前方案的自然确认。
5. “也许、可能、先看看、还没想好”等表达只能进入open_questions或deferred_items，不能作为强约束。
6. 被教师否定、纠正、替代的旧内容必须进入superseded_items，state=superseded，do_not_reconfirm=true。
7. 已进入superseded_items的内容不得重新包装成open_questions，不得在后续阶段再次询问教师，除非教师本轮明确要求恢复。
8. 阶段变化只改变stage_focus，不清空前序共识，不生成生硬的阶段过渡话术。
9. stage_focus只说明当前工作的注意力。跨阶段继续携带course_core、teaching_consensus和constraints。
10. AI自己的建议若教师没有采纳，不得进入teaching_consensus。
11. ai_inferred只能进入open_questions或deferred_items，不能进入course_core或强约束。
12. 每个原子条目必须有稳定英文小写key，只允许a-z、0-9、点、下划线、冒号和连字符。
13. source_keys只能引用输入中真实存在的来源key，禁止编造来源。
14. 不输出附件文件名清单、Token信息、提示词、处理状态或隐藏推理。
15. 若本轮没有产生任何会影响后续备课的新增、细化、修正、替代、否定、暂停、恢复或证据增强，changes必须返回空数组。
16. 不得仅因措辞变化、阶段名称变化、礼貌表达或普通资料查询而改写稳定条目。
17. 输出完整的新胶囊，不输出增量补丁以外的解释文字。
18. 必须主动合并语义重复条目，避免同一决定以不同说法重复出现。
19. summary不超过260个中文字符；update_reason不超过120个中文字符；单个条目content不超过320个中文字符。
20. course_core最多10条，teaching_consensus最多14条，constraints最多10条，open_questions最多8条，deferred_items最多8条，superseded_items最多24条。
21. 当前AI回复中刚刚生成的详细教案内容不能自动写成“教师已确认”；只有教师对上一版内容明确确认后才可记录为已确认。

输出严格JSON：
{
  "update_reason": "一句自然、克制的变化说明",
  "capsule": {
    "schema_version": 1,
    "summary": "当前本课最重要的共同认识",
    "course_core": [],
    "teaching_consensus": [],
    "constraints": [],
    "open_questions": [],
    "deferred_items": [],
    "superseded_items": [],
    "stage_focus": {
      "stage_code": "",
      "current_task": "",
      "carry_forward_keys": [],
      "avoid_repeating_keys": []
    }
  },
  "changes": [
    {
      "operation": "add|refine|correct|replace|reject|defer|restore|strengthen",
      "item_key": "",
      "summary": ""
    }
  ],
  "evidence_bindings": [
    {
      "item_key": "",
      "source_keys": []
    }
  ]
}

每个条目结构：
{
  "key": "",
  "title": "",
  "content": "",
  "state": "active|candidate|deferred|superseded",
  "authority": "teacher_explicit|source_verified|teacher_source_confirmed|ai_inferred",
  "importance": 1,
  "applicable_stages": [],
  "source_keys": [],
  "do_not_reconfirm": false,
  "replaced_by": "",
  "updated_by_turn_id": ""
}

只输出JSON，不输出Markdown、代码围栏、解释或隐藏推理。`

// scheduleLessonPlanContextCapsuleUpdate 在主回复完成后旁路启动更新。
//
// 自动阶段触发语、空消息和纯礼貌消息不产生胶囊AI调用。
// “好、好的、可以、继续”等短确认不能跳过，因为它们可能是在自然确认上一轮方案。
func (s *LessonPlanGenService) scheduleLessonPlanContextCapsuleUpdate(
	lessonPlan *models.LessonPlan,
	teacherMessage *models.ConversationMessage,
	assistantMessage *models.ConversationMessage,
	turnID string,
) {
	if s == nil ||
		s.cfg == nil ||
		lessonPlan == nil ||
		teacherMessage == nil ||
		shouldSkipLessonPlanContextCapsuleUpdate(
			teacherMessage.Content,
		) {
		return
	}

	lessonPlanCopy := *lessonPlan
	teacherCopy := *teacherMessage

	var assistantCopy *models.ConversationMessage
	if assistantMessage != nil {
		value := *assistantMessage
		assistantCopy = &value
	}

	go func() {
		ctx, cancel := context.WithTimeout(
			context.Background(),
			lessonPlanContextCapsuleTimeout,
		)
		defer cancel()

		capsule, changed, err := s.updateLessonPlanContextCapsule(
			ctx,
			&lessonPlanCopy,
			&teacherCopy,
			assistantCopy,
			turnID,
		)
		if err != nil {
			lpGenLog.Warn(
				"备课核心共识胶囊旁路更新失败",
				"plan_id", lessonPlanCopy.ID,
				"stage", lessonPlanCopy.CurrentStage,
				"turn_id", turnID,
				"error", err,
			)

			errorContext, errorCancel := context.WithTimeout(
				context.Background(),
				5*time.Second,
			)
			defer errorCancel()

			_ = repository.RecordLessonPlanContextCapsuleUpdateError(
				errorContext,
				lessonPlanCopy.ID,
				err.Error(),
			)
			return
		}

		if capsule == nil {
			return
		}

		if changed {
			broadcastLessonPlanContextCapsuleUpdate(
				lessonPlanCopy.ID,
				turnID,
				capsule,
			)
		}

		lpGenLog.Info(
			"备课核心共识胶囊旁路更新完成",
			"plan_id", lessonPlanCopy.ID,
			"stage", lessonPlanCopy.CurrentStage,
			"turn_id", turnID,
			"version", capsule.Version,
			"changed", changed,
			"context_runes", len([]rune(capsule.ContextText)),
		)
	}()
}

// broadcastLessonPlanContextCapsuleUpdate 广播新的教师端安全视图。
//
// 这是非终态SSE事件。连接继续保持，前端只更新环境式胶囊浮层。
func broadcastLessonPlanContextCapsuleUpdate(
	lessonPlanID string,
	turnID string,
	capsule *models.LessonPlanContextCapsule,
) {
	if capsule == nil ||
		strings.TrimSpace(capsule.DisplayJSON) == "" {
		return
	}

	display := &models.LessonPlanContextCapsuleDisplayView{}
	if err := json.Unmarshal(
		[]byte(capsule.DisplayJSON),
		display,
	); err != nil {
		lpGenLog.Warn(
			"解析胶囊教师端安全视图失败，跳过SSE广播",
			"plan_id", lessonPlanID,
			"capsule_version", capsule.Version,
			"error", err,
		)
		return
	}

	GlobalLPSSEHub.Broadcast(
		lessonPlanID,
		models.LPSSEEvent{
			EventType:    models.LPSSEContextCapsule,
			PlanID:       lessonPlanID,
			ClientTurnID: strings.TrimSpace(turnID),
			ContextCapsule: &models.LessonPlanContextCapsuleEventData{
				Version: capsule.Version,
				Status:  capsule.Status,
				Display: display,
			},
		},
	)
}

// updateLessonPlanContextCapsule 完成一次旁路更新。
func (s *LessonPlanGenService) updateLessonPlanContextCapsule(
	ctx context.Context,
	lessonPlan *models.LessonPlan,
	teacherMessage *models.ConversationMessage,
	assistantMessage *models.ConversationMessage,
	turnID string,
) (*models.LessonPlanContextCapsule, bool, error) {
	if lessonPlan == nil ||
		teacherMessage == nil {
		return nil, false, errors.New(
			"胶囊旁路更新输入不完整",
		)
	}

	current, err := repository.GetLessonPlanContextCapsule(
		ctx,
		lessonPlan.ID,
	)
	if err != nil {
		return nil, false, err
	}

	source, err := loadLessonPlanContextCapsuleSource(
		ctx,
		lessonPlan,
		teacherMessage,
		turnID,
	)
	if err != nil {
		return nil, false, err
	}

	effectiveConfig, err := aiClient.GetEffectiveConfig(
		s.cfg.GetAESKey(),
		models.SceneLessonPlanHarness,
		"",
		"",
		"",
	)
	if err != nil {
		return nil, false, fmt.Errorf(
			"加载胶囊更新模型失败: %w",
			err,
		)
	}

	effectiveConfig.Temperature = 0
	if effectiveConfig.MaxTokens <= 0 ||
		effectiveConfig.MaxTokens >
			lessonPlanContextCapsuleMaxTokens {
		effectiveConfig.MaxTokens =
			lessonPlanContextCapsuleMaxTokens
	}

	currentCapsule := map[string]interface{}{}
	if current != nil &&
		strings.TrimSpace(current.CapsuleJSON) != "" &&
		strings.TrimSpace(current.CapsuleJSON) != "{}" {
		currentCapsule =
			compactLessonPlanContextCapsuleForModel(
				current.CapsuleJSON,
			)
	}

	assistantContent := ""
	if assistantMessage != nil {
		assistantContent =
			compactLessonPlanContextCapsuleTextForModel(
				assistantMessage.Content,
				7000,
			)
	}

	payload := map[string]interface{}{
		"lesson_plan": map[string]string{
			"id":               lessonPlan.ID,
			"subject":          lessonPlan.Subject,
			"grade":            lessonPlan.Grade,
			"topic":            lessonPlan.Topic,
			"education_domain": lessonPlan.EducationDomain,
			"current_stage":    lessonPlan.CurrentStage,
		},
		"current_capsule": currentCapsule,
		"teacher_turn": map[string]string{
			"turn_id": strings.TrimSpace(
				turnID,
			),
			"message_id": strings.TrimSpace(
				teacherMessage.ID,
			),
			"content": compactLessonPlanContextCapsuleTextForModel(
				teacherMessage.Content,
				5000,
			),
		},
		"assistant_reply_for_dialogue_understanding_only": assistantContent,
		"authoritative_sources":                           source.Entries,
	}

	inputJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, false, fmt.Errorf(
			"序列化胶囊更新输入失败: %w",
			err,
		)
	}

	schoolID, _ := repository.GetSchoolIDByUserID(
		ctx,
		lessonPlan.AuthorID,
	)

	planID := lessonPlan.ID
	traceContext := &aiClient.TraceContext{
		SceneCode:    models.SceneLessonPlanHarness,
		LessonPlanID: &planID,
		UserID:       &lessonPlan.AuthorID,
		SchoolID:     schoolIDPtr(schoolID),
	}

	result, err := aiClient.CallAI(
		effectiveConfig,
		lessonPlanContextCapsuleSystemPrompt,
		string(inputJSON),
		traceContext,
	)
	if err != nil {
		return nil, false, fmt.Errorf(
			"胶囊旁路AI调用失败: %w",
			err,
		)
	}
	if result == nil ||
		strings.TrimSpace(result.Content) == "" {
		return nil, false, errors.New(
			"胶囊旁路AI没有返回内容",
		)
	}

	aiResult, parseErr :=
		parseLessonPlanContextCapsuleAIResult(
			result.Content,
		)

	if parseErr != nil {
		lpGenLog.Warn(
			"胶囊旁路AI首轮JSON解析失败，准备受控重试",
			"plan_id", lessonPlan.ID,
			"stage", lessonPlan.CurrentStage,
			"turn_id", turnID,
			"content_runes",
			len([]rune(result.Content)),
			"error", parseErr,
		)

		if contextErr := ctx.Err(); contextErr != nil {
			return nil, false, fmt.Errorf(
				"胶囊旁路AI首轮解析失败且任务已超时: %w",
				contextErr,
			)
		}

		retryResult, retryErr := aiClient.CallAI(
			effectiveConfig,
			lessonPlanContextCapsuleRetrySystemPrompt(),
			string(inputJSON),
			traceContext,
		)

		switch {
		case retryErr != nil:
			parseErr = fmt.Errorf(
				"胶囊旁路AI JSON受控重试失败: %w",
				retryErr,
			)

		case retryResult == nil ||
			strings.TrimSpace(retryResult.Content) == "":
			parseErr = errors.New(
				"胶囊旁路AI JSON受控重试没有返回内容",
			)

		default:
			aiResult, parseErr =
				parseLessonPlanContextCapsuleAIResult(
					retryResult.Content,
				)

			if parseErr == nil {
				lpGenLog.Info(
					"胶囊旁路AI JSON受控重试成功",
					"plan_id", lessonPlan.ID,
					"stage", lessonPlan.CurrentStage,
					"turn_id", turnID,
					"content_runes",
					len([]rune(retryResult.Content)),
				)
			}
		}

		if parseErr != nil {
			fallbackResult, fallbackErr :=
				buildLessonPlanContextCapsuleDeterministicFallback(
					current,
				)

			if fallbackErr != nil {
				return nil, false, fmt.Errorf(
					"胶囊旁路AI两次均未返回合法JSON: %v；确定性降级失败: %w",
					parseErr,
					fallbackErr,
				)
			}

			aiResult = fallbackResult

			lpGenLog.Warn(
				"胶囊旁路AI JSON失败，已启用当前active胶囊的确定性进度降级",
				"plan_id", lessonPlan.ID,
				"stage", lessonPlan.CurrentStage,
				"turn_id", turnID,
				"current_version", current.Version,
				"error", parseErr,
			)
		}
	}

	normalizeLessonPlanContextCapsuleDocument(
		&aiResult.Capsule,
		lessonPlan.CurrentStage,
		turnID,
	)

	deterministicChoiceChanged := false

	if current != nil {
		preserveLessonPlanCapsuleNegativeMemory(
			&aiResult.Capsule,
			current.CapsuleJSON,
			teacherMessage.Content,
		)

		deterministicChoiceChanged =
			applyLessonPlanCapsuleTeacherChoice(
				&aiResult.Capsule,
				current.CapsuleJSON,
				teacherMessage.Content,
				turnID,
			)

		if deterministicChoiceChanged &&
			strings.TrimSpace(aiResult.UpdateReason) == "" {
			aiResult.UpdateReason =
				"已按教师选择收拢当前教学方案"
		}
	}

	progressChanged :=
		reconcileLessonPlanContextCapsuleProgress(
			&aiResult.Capsule,
			lessonPlanContextCapsuleCurrentJSON(
				current,
			),
			teacherMessage.Content,
			lessonPlanContextCapsuleAssistantProgressText(
				assistantMessage,
			),
			lessonPlan.CurrentStage,
			turnID,
		)

	if progressChanged {
		aiResult.UpdateReason =
			lessonPlanContextCapsuleProgressRecentChange(
				&aiResult.Capsule,
			)
	}

	if current != nil &&
		len(aiResult.Changes) == 0 &&
		!deterministicChoiceChanged &&
		!progressChanged {
		return current, false, nil
	}

	if !lessonPlanContextCapsuleHasUsableCore(
		&aiResult.Capsule,
	) {
		return nil, false, errors.New(
			"胶囊结果没有可靠课程核心或教师共识",
		)
	}

	manifestJSON, err := json.Marshal(
		source.Manifest,
	)
	if err != nil {
		return nil, false, fmt.Errorf(
			"序列化胶囊来源清单失败: %w",
			err,
		)
	}

	capsuleJSON, err := json.Marshal(
		aiResult.Capsule,
	)
	if err != nil {
		return nil, false, fmt.Errorf(
			"序列化胶囊正文失败: %w",
			err,
		)
	}

	contextText := buildLessonPlanContextCapsuleContextText(
		&aiResult.Capsule,
	)
	if strings.TrimSpace(contextText) == "" {
		return nil, false, errors.New(
			"无法构建胶囊运行时短版上下文",
		)
	}

	displayView := buildLessonPlanContextCapsuleDisplayView(
		&aiResult.Capsule,
		aiResult.UpdateReason,
	)

	displayJSON, err := json.Marshal(
		displayView,
	)
	if err != nil {
		return nil, false, fmt.Errorf(
			"序列化胶囊教师端安全视图失败: %w",
			err,
		)
	}

	stableHash, err := hashLessonPlanContextCapsuleVersion(
		&aiResult.Capsule,
	)
	if err != nil {
		return nil, false, err
	}

	sourceHash :=
		hashLessonPlanContextCapsuleVersionWithProgress(
			stableHash,
			&aiResult.Capsule,
		)

	evidence := buildLessonPlanContextCapsuleEvidence(
		lessonPlan.ID,
		&aiResult.Capsule,
		source.Entries,
	)

	saved, changed, err :=
		repository.UpsertActiveLessonPlanContextCapsule(
			ctx,
			&repository.UpsertLessonPlanContextCapsuleInput{
				LessonPlanID:     lessonPlan.ID,
				SchemaVersion:    models.LessonPlanContextCapsuleSchemaVersion,
				CurrentStageCode: lessonPlan.CurrentStage,
				CapsuleJSON: string(
					capsuleJSON,
				),
				DisplayJSON: string(
					displayJSON,
				),
				ContextText: contextText,
				SourceManifest: string(
					manifestJSON,
				),
				SourceHash: sourceHash,
				LastTurnID: strings.TrimSpace(
					turnID,
				),
				UpdateReason: normalizeLessonPlanCapsuleText(
					aiResult.UpdateReason,
					500,
				),
				Evidence: evidence,
			},
		)
	if err != nil {
		return nil, false, err
	}

	return saved, changed, nil
}

// buildLessonPlanContextCapsuleDeterministicFallback 复制当前active胶囊。
//
// 当胶囊模型连续两次未返回合法JSON时，稳定课程核心、教师共识、
// 约束和负向记忆全部沿用数据库当前版本。后续仍会执行本地确定性的
// 环节生成识别、教师确认识别、摘要生成、展示生成和进度哈希。
//
// 该降级不会根据损坏JSON猜测任何业务事实，也不会把当前AI回复
// 自动升级成教师确认。
func buildLessonPlanContextCapsuleDeterministicFallback(
	current *models.LessonPlanContextCapsule,
) (
	*models.LessonPlanContextCapsuleAIResult,
	error,
) {
	if current == nil {
		return nil, errors.New(
			"不存在可供确定性降级使用的active胶囊",
		)
	}

	currentJSON :=
		strings.TrimSpace(
			current.CapsuleJSON,
		)

	if currentJSON == "" ||
		currentJSON == "{}" {
		return nil, errors.New(
			"当前active胶囊正文为空",
		)
	}

	document :=
		&models.LessonPlanContextCapsuleDocument{}

	if err := json.Unmarshal(
		[]byte(currentJSON),
		document,
	); err != nil {
		return nil, fmt.Errorf(
			"解析当前active胶囊失败: %w",
			err,
		)
	}

	if !lessonPlanContextCapsuleHasUsableCore(
		document,
	) {
		return nil, errors.New(
			"当前active胶囊没有可靠课程核心或教师共识",
		)
	}

	result :=
		&models.LessonPlanContextCapsuleAIResult{}

	result.Capsule =
		*document

	return result, nil
}
