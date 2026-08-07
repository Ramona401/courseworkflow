package services

// assistant_runtime_chat_test.go
//
// 仅验证不连接数据库的运行聊天纯逻辑：不可变快照解析、消息历史裁剪、
// 苏格拉底安全规则和稳定失败码。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"tedna/internal/models"
)

// TestAssistantRuntimeChatBuildsImmutableMessages 验证正式消息装配。
func TestAssistantRuntimeChatBuildsImmutableMessages(
	t *testing.T,
) {
	authorization,
		claim :=
		assistantRuntimeChatTestFixture(
			t,
		)

	messages, err :=
		buildAssistantRuntimeChatMessages(
			authorization,
			claim,
			"我觉得两个三角形可以拼成一个平行四边形。",
		)
	if err != nil {
		t.Fatalf(
			"构造运行聊天消息失败: %v",
			err,
		)
	}

	if len(messages) != 4 {
		t.Fatalf(
			"消息数量错误: %d",
			len(messages),
		)
	}

	if messages[0].Role !=
		"system" ||
		!strings.Contains(
			messages[0].Content,
			"禁止直接泄露最终答案",
		) ||
		!strings.Contains(
			messages[0].Content,
			"不得声称看见学生点击",
		) ||
		!strings.Contains(
			messages[0].Content,
			"发布时冻结的页面上下文",
		) {
		t.Fatalf(
			"系统提示缺少运行安全规则: %s",
			messages[0].Content,
		)
	}

	if messages[1].Role !=
		"user" ||
		messages[2].Role !=
			"assistant" {
		t.Fatalf(
			"历史消息角色映射错误: %#v",
			messages,
		)
	}

	if messages[3].Role !=
		"user" ||
		messages[3].Content !=
			"我觉得两个三角形可以拼成一个平行四边形。" {
		t.Fatalf(
			"当前学生消息错误: %#v",
			messages[3],
		)
	}
}

// TestAssistantRuntimeChatRejectsAnswerLeakSnapshot 验证答案泄露策略fail-closed。
func TestAssistantRuntimeChatRejectsAnswerLeakSnapshot(
	t *testing.T,
) {
	authorization,
		claim :=
		assistantRuntimeChatTestFixture(
			t,
		)

	var plan models.
		AssistantDeploymentTeachingPlanSnapshot

	if err :=
		json.Unmarshal(
			[]byte(
				authorization.Version.
					TeachingPlanJSON,
			),
			&plan,
		); err != nil {
		t.Fatal(err)
	}

	plan.GuidancePlan.
		AnswerLeakPolicy.
		DirectAnswerAllowed = true

	encoded, err :=
		json.Marshal(
			plan,
		)
	if err != nil {
		t.Fatal(err)
	}

	authorization.Version.
		TeachingPlanJSON =
		string(encoded)

	_, err =
		buildAssistantRuntimeChatMessages(
			authorization,
			claim,
			"请直接告诉我答案。",
		)

	if !errors.Is(
		err,
		ErrAssistantRuntimeChatSnapshotInvalid,
	) {
		t.Fatalf(
			"允许直接答案的快照应被拒绝: %v",
			err,
		)
	}
}

// TestAssistantRuntimeChatLimitsRecentHistory 验证只选择最近12条正式消息。
func TestAssistantRuntimeChatLimitsRecentHistory(
	t *testing.T,
) {
	authorization,
		claim :=
		assistantRuntimeChatTestFixture(
			t,
		)

	claim.Messages =
		make(
			[]models.AssistantRuntimeMessage,
			0,
			20,
		)

	for index := 0; index < 20; index++ {
		role :=
			models.
				AssistantRuntimeMessageRoleStudent

		if index%2 == 1 {
			role =
				models.
					AssistantRuntimeMessageRoleAssistant
		}

		claim.Messages =
			append(
				claim.Messages,
				models.AssistantRuntimeMessage{
					Role: role,
					Content: fmt.Sprintf(
						"历史消息-%02d",
						index,
					),
				},
			)
	}

	messages, err :=
		buildAssistantRuntimeChatMessages(
			authorization,
			claim,
			"继续。",
		)
	if err != nil {
		t.Fatalf(
			"构造消息失败: %v",
			err,
		)
	}

	if len(messages) !=
		assistantRuntimeHistoryMaxMessages+
			2 {
		t.Fatalf(
			"历史消息上限未生效: %d",
			len(messages),
		)
	}

	if messages[1].Content !=
		"历史消息-08" ||
		messages[len(messages)-2].Content !=
			"历史消息-19" {
		t.Fatalf(
			"没有保留最近历史: first=%s last=%s",
			messages[1].Content,
			messages[len(messages)-2].Content,
		)
	}
}

// TestAssistantRuntimeChatFailureCodes 验证公开运行流水错误码稳定。
func TestAssistantRuntimeChatFailureCodes(
	t *testing.T,
) {
	if code :=
		assistantRuntimeChatFailureCode(
			context.Canceled,
		); code !=
		"client_cancelled" {
		t.Fatalf(
			"取消错误码错误: %s",
			code,
		)
	}

	if code :=
		assistantRuntimeChatFailureCode(
			context.DeadlineExceeded,
		); code !=
		"runtime_timeout" {
		t.Fatalf(
			"超时错误码错误: %s",
			code,
		)
	}

	if code :=
		assistantRuntimeChatFailureCode(
			errors.New(
				"upstream",
			),
		); code !=
		"ai_stream_failed" {
		t.Fatalf(
			"普通流错误码错误: %s",
			code,
		)
	}
}

// assistantRuntimeChatTestFixture 构造纯内存不可变版本和会话历史。
func assistantRuntimeChatTestFixture(
	t *testing.T,
) (
	*AssistantRuntimeAuthorization,
	*models.AssistantRuntimeTurnClaim,
) {
	t.Helper()

	pageID :=
		"55555555-5555-5555-5555-555555555555"

	deploymentID :=
		"66666666-6666-6666-6666-666666666666"

	plan :=
		models.AssistantDeploymentTeachingPlanSnapshot{
			Version: models.
				AssistantDeploymentSnapshotVersion,
			Title:             "面积探究伙伴",
			WelcomeMessage:    "先观察，再说说你的发现。",
			TeachingRole:      "通过逐层提问支持学生自主推导。",
			LearningObjective: "解释三角形面积公式中除以二的来源。",
			DisplayMode: models.
				CoursewareAssistantDisplayModeFloating,
			DisplayPosition: models.
				CoursewareAssistantPositionBottomRight,
			GuidancePlan: models.CoursewareAssistantGuidancePlan{
				Version: models.
					CoursewareAssistantProtocolVersion,
				QuestionChain: []models.CoursewareAssistantQuestionStep{
					{
						ID:     "q1",
						Prompt: "两个相同三角形可以拼成什么图形？",
					},
				},
				AnswerLeakPolicy: models.CoursewareAssistantAnswerLeakPolicy{
					DirectAnswerAllowed: false,
					RequireStudentTry:   true,
					MaximumHintLevel:    3,
				},
			},
		}

	contextSnapshot :=
		models.AssistantDeploymentContextSnapshot{
			Version: models.
				AssistantDeploymentSnapshotVersion,
			CurrentPage: models.AssistantDeploymentPageContextSnapshot{
				PageID:      pageID,
				PageNumber:  3,
				Title:       "拼接与转化",
				VisibleText: "拖动两个相同三角形，观察拼成的图形。",
			},
		}

	planJSON, err :=
		json.Marshal(
			plan,
		)
	if err != nil {
		t.Fatal(err)
	}

	contextJSON, err :=
		json.Marshal(
			contextSnapshot,
		)
	if err != nil {
		t.Fatal(err)
	}

	authorization :=
		&AssistantRuntimeAuthorization{
			Session: &models.AssistantRuntimeSession{
				ID:                "77777777-7777-7777-7777-777777777777",
				DeploymentID:      deploymentID,
				DeploymentVersion: 1,
			},
			Deployment: &models.AssistantDeployment{
				ID:             deploymentID,
				PageID:         pageID,
				CurrentVersion: 1,
			},
			Version: &models.AssistantDeploymentVersion{
				DeploymentID:            deploymentID,
				Version:                 1,
				AssistantPromptSnapshot: "你是一位耐心但不直接给答案的数学探究伙伴。",
				TeachingPlanJSON: string(
					planJSON,
				),
				ContextSnapshotJSON: string(
					contextJSON,
				),
			},
		}

	claim :=
		&models.AssistantRuntimeTurnClaim{
			TurnID: "88888888-8888-8888-8888-888888888888",
			SessionID: authorization.Session.
				ID,
			DeploymentID:      deploymentID,
			DeploymentVersion: 1,
			Messages: []models.AssistantRuntimeMessage{
				{
					Role: models.
						AssistantRuntimeMessageRoleStudent,
					Content: "我不知道从哪里开始。",
				},
				{
					Role: models.
						AssistantRuntimeMessageRoleAssistant,
					Content: "先观察页面上有几个相同的三角形？",
				},
			},
		}

	return authorization,
		claim
}
