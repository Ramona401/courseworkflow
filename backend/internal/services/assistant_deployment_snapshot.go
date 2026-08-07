package services

// assistant_deployment_snapshot.go
//
// 本文件把经过发布校验的课件、页面插槽、可选AI助手、页面上下文和运行策略
// 确定性装配为assistant_deployment_versions不可变记录。
//
// 快照分层：
//   - assistant_prompt_snapshot保存系统默认教学风格，或教师主动选择的助手提示词；
//   - assistant_id允许为空，表示使用系统默认页面教学风格；
//   - teaching_plan_json保存教师确认的名称、欢迎语、角色、目标和问题链；
//   - context_snapshot_json保存受限页面教学上下文；
//   - courseware_snapshot_json保存最小课件事实和本次发布策略审计快照；
//   - page_html_hash只保存当前完整HTML的哈希，不保存完整HTML。
//
// 本文件不调用AI、不写数据库，也不把任何敏感快照转换为浏览器响应。

import (
	"encoding/json"
	"fmt"
	"strings"

	"tedna/internal/models"
)

// assistantDeploymentDefaultPromptSnapshot 是没有选择已有助手时冻结的默认教学风格。
//
// 具体教学方式、问题链、提示阶梯和知识边界仍由运行时共同规则、
// TeachingPlanJSON和ContextSnapshotJSON决定。本提示只提供稳定基础风格。
const assistantDeploymentDefaultPromptSnapshot = `你是TE-DNA课件当前页面的教学智能体。

请严格执行本次发布快照中的教学方式、教学目标、互动步骤、提示阶梯、学习困难应对和完成标准。

你的职责是帮助学生在当前页面真实参与学习，通过简短问题、反馈和由弱到强的支架推进理解。不要连续讲授长篇结论，不要替学生完成其应独立完成的任务，不要直接公布当前任务答案。

当教学方案和页面上下文已经提供具体规则时，以这些不可变发布数据为准。`

// assistantDeploymentCoursewareSnapshotEnvelope 是courseware_snapshot_json协议。
//
// Policy用于审计“这个版本发布时采用什么运行边界”；运行时仍以部署主表的
// 当前可变额度和来源策略作为实时闸门。
type assistantDeploymentCoursewareSnapshotEnvelope struct {
	Version    string                                       `json:"version"`
	Courseware models.AssistantDeploymentCoursewareSnapshot `json:"courseware"`
	Policy     assistantDeploymentPolicySnapshot            `json:"deployment_policy"`
}

// buildAssistantDeploymentVersionRecord 纯内存生成完整版本记录。
func buildAssistantDeploymentVersionRecord(
	courseware *models.Courseware,
	slot *models.CoursewareAssistantSlotView,
	assistant *models.AIAssistant,
	contextResult *CoursewareAssistantContextBuildResult,
	policy *assistantDeploymentNormalizedPolicy,
	createdBy string,
) (
	*models.AssistantDeploymentVersion,
	error,
) {
	if courseware == nil ||
		slot == nil ||
		contextResult == nil ||
		policy == nil ||
		strings.TrimSpace(
			createdBy,
		) == "" {
		return nil,
			ErrAssistantDeploymentSnapshotInvalid
	}

	if strings.TrimSpace(
		contextResult.SnapshotHash,
	) == "" ||
		strings.TrimSpace(
			contextResult.PageHTMLHash,
		) == "" ||
		strings.TrimSpace(
			contextResult.SnapshotJSON,
		) == "" {
		return nil,
			ErrAssistantDeploymentSnapshotInvalid
	}

	currentPage :=
		contextResult.Snapshot.CurrentPage

	if strings.TrimSpace(
		currentPage.PageID,
	) !=
		strings.TrimSpace(
			slot.PageID,
		) ||
		strings.TrimSpace(
			slot.CoursewareID,
		) !=
			strings.TrimSpace(
				courseware.ID,
			) {
		return nil,
			ErrAssistantDeploymentSnapshotInvalid
	}

	assistantID,
		promptSnapshot,
		err :=
		resolveAssistantDeploymentPromptSnapshot(
			assistant,
		)
	if err != nil {
		return nil, err
	}

	teachingPlan :=
		models.AssistantDeploymentTeachingPlanSnapshot{
			Version: models.AssistantDeploymentSnapshotVersion,
			Title: strings.TrimSpace(
				slot.Title,
			),
			WelcomeMessage: strings.TrimSpace(
				slot.WelcomeMessage,
			),
			TeachingRole: strings.TrimSpace(
				slot.TeachingRole,
			),
			LearningObjective: strings.TrimSpace(
				slot.LearningObjective,
			),
			DisplayMode: strings.TrimSpace(
				slot.DisplayMode,
			),
			DisplayPosition: strings.TrimSpace(
				slot.DisplayPosition,
			),
			GuidancePlan: slot.GuidancePlan,
		}

	teachingPlanJSON, err :=
		marshalAssistantDeploymentSnapshotValue(
			"教学方案",
			teachingPlan,
		)
	if err != nil {
		return nil, err
	}

	coursewareSnapshot :=
		models.AssistantDeploymentCoursewareSnapshot{
			CoursewareID: strings.TrimSpace(
				courseware.ID,
			),
			PageID: strings.TrimSpace(
				currentPage.PageID,
			),
			PageNumber: currentPage.PageNumber,
			CoursewareTitle: strings.TrimSpace(
				courseware.Title,
			),
			PageTitle: strings.TrimSpace(
				currentPage.Title,
			),
			Subject: strings.TrimSpace(
				courseware.Subject,
			),
			Grade: strings.TrimSpace(
				courseware.Grade,
			),
			EducationDomain: strings.ToLower(
				strings.TrimSpace(
					courseware.EducationDomain,
				),
			),
		}

	policySnapshot :=
		assistantDeploymentPolicySnapshot{
			AccessMode:       models.AssistantDeploymentAccessOriginAllowlist,
			DailyCallLimit:   policy.DailyCallLimit,
			SessionTurnLimit: policy.PerSessionTurnLimit,
			AllowedOrigins: append(
				[]string{},
				policy.AllowedOrigins...,
			),
			ValidUntil: policy.ValidUntil,
		}

	coursewareEnvelope :=
		assistantDeploymentCoursewareSnapshotEnvelope{
			Version:    models.AssistantDeploymentSnapshotVersion,
			Courseware: coursewareSnapshot,
			Policy:     policySnapshot,
		}

	coursewareSnapshotJSON, err :=
		marshalAssistantDeploymentSnapshotValue(
			"课件和发布策略",
			coursewareEnvelope,
		)
	if err != nil {
		return nil, err
	}

	version :=
		&models.AssistantDeploymentVersion{
			AssistantID:             assistantID,
			AssistantPromptSnapshot: promptSnapshot,
			AssistantPromptHash: coursewareAssistantSHA256String(
				promptSnapshot,
			),
			TeachingPlanJSON:    teachingPlanJSON,
			ContextSnapshotJSON: contextResult.SnapshotJSON,
			ContextSnapshotHash: strings.TrimSpace(
				contextResult.SnapshotHash,
			),
			PageHTMLHash: strings.TrimSpace(
				contextResult.PageHTMLHash,
			),
			CoursewareSnapshotJSON: coursewareSnapshotJSON,
			CreatedBy: strings.TrimSpace(
				createdBy,
			),
		}

	if err :=
		validateAssistantDeploymentVersionSnapshot(
			version,
		); err != nil {
		return nil, err
	}

	return version, nil
}

// resolveAssistantDeploymentPromptSnapshot 解析可选助手和默认风格。
func resolveAssistantDeploymentPromptSnapshot(
	assistant *models.AIAssistant,
) (
	*string,
	string,
	error,
) {
	if assistant == nil {
		prompt :=
			strings.TrimSpace(
				assistantDeploymentDefaultPromptSnapshot,
			)

		if prompt == "" {
			return nil, "",
				ErrAssistantDeploymentSnapshotInvalid
		}

		return nil, prompt, nil
	}

	assistantID :=
		strings.TrimSpace(
			assistant.ID,
		)

	prompt :=
		strings.TrimSpace(
			assistant.FullPrompt,
		)

	if assistantID == "" ||
		prompt == "" {
		return nil, "",
			ErrAssistantDeploymentAssistantPromptRequired
	}

	return &assistantID,
		prompt,
		nil
}

// marshalAssistantDeploymentSnapshotValue 使用结构体固定字段顺序序列化。
func marshalAssistantDeploymentSnapshotValue(
	label string,
	value interface{},
) (
	string,
	error,
) {
	encoded, err :=
		json.Marshal(value)
	if err != nil {
		return "",
			fmt.Errorf(
				"%w: 序列化%s快照失败: %v",
				ErrAssistantDeploymentSnapshotInvalid,
				label,
				err,
			)
	}

	if !json.Valid(encoded) {
		return "",
			fmt.Errorf(
				"%w: %s快照不是合法JSON",
				ErrAssistantDeploymentSnapshotInvalid,
				label,
			)
	}

	return string(encoded), nil
}

// validateAssistantDeploymentVersionSnapshot 复核所有不可变快照字段。
func validateAssistantDeploymentVersionSnapshot(
	version *models.AssistantDeploymentVersion,
) error {
	if version == nil {
		return ErrAssistantDeploymentSnapshotInvalid
	}

	if version.AssistantID != nil &&
		strings.TrimSpace(
			*version.AssistantID,
		) == "" {
		return ErrAssistantDeploymentSnapshotInvalid
	}

	if strings.TrimSpace(
		version.AssistantPromptSnapshot,
	) == "" ||
		len(
			strings.TrimSpace(
				version.AssistantPromptHash,
			),
		) != 64 ||
		len(
			strings.TrimSpace(
				version.ContextSnapshotHash,
			),
		) != 64 ||
		len(
			strings.TrimSpace(
				version.PageHTMLHash,
			),
		) != 64 ||
		strings.TrimSpace(
			version.CreatedBy,
		) == "" {
		return ErrAssistantDeploymentSnapshotInvalid
	}

	jsonFields :=
		[]string{
			version.TeachingPlanJSON,
			version.ContextSnapshotJSON,
			version.CoursewareSnapshotJSON,
		}

	for _, raw := range jsonFields {
		if !json.Valid(
			[]byte(
				strings.TrimSpace(
					raw,
				),
			),
		) {
			return ErrAssistantDeploymentSnapshotInvalid
		}
	}

	return nil
}
