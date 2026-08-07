package integration

// assistant_runtime_fixture.go
//
// 为教学智能体真实仓储集成测试建立最小确定性数据库夹具。
//
// 夹具包含：
//   - 一个mixed区域；
//   - 一个K12学校；
//   - operator与学校的校籍；
//   - 一份preview课件和一个稳定页面；
//   - 一个可用个人AI助手；
//   - 一个活动教学智能体插槽；
//   - operator的个人Token账户。
//
// 部署和运行会话记录构造位于assistant_runtime_fixture_records.go。

import (
	"context"
	"encoding/json"
	"testing"

	"tedna/internal/database"
	"tedna/internal/models"
)

// 教学智能体夹具固定标识。
const (
	AssistantFixtureRegionID =
		"10000000-0000-4000-8000-000000000001"
	AssistantFixtureSchoolID =
		"10000000-0000-4000-8000-000000000002"
	AssistantFixtureCoursewareID =
		"10000000-0000-4000-8000-000000000003"
	AssistantFixturePageID =
		"10000000-0000-4000-8000-000000000004"
	AssistantFixtureAssistantID =
		"10000000-0000-4000-8000-000000000005"
	AssistantFixtureSlotID =
		"10000000-0000-4000-8000-000000000006"
	AssistantFixtureTokenAccountID =
		"10000000-0000-4000-8000-000000000007"

	AssistantFixtureSessionID =
		"20000000-0000-4000-8000-000000000001"
	AssistantFixtureSuccessTurnID =
		"20000000-0000-4000-8000-000000000002"
	AssistantFixtureFailureTurnID =
		"20000000-0000-4000-8000-000000000003"

	AssistantFixtureOrigin =
		"https://course.example"
)

// AssistantRuntimeFixture 保存夹具和不可变快照。
type AssistantRuntimeFixture struct {
	RegionID       string
	SchoolID       string
	CoursewareID   string
	PageID         string
	AssistantID    string
	SlotID         string
	TokenAccountID string

	TeachingPlanJSON    string
	ContextSnapshotJSON string
}

// SeedAssistantRuntimeFixture 写入教学智能体最小依赖数据。
func SeedAssistantRuntimeFixture(
	t *testing.T,
) *AssistantRuntimeFixture {
	t.Helper()

	if database.DB == nil {
		t.Fatal(
			"测试数据库连接池未初始化",
		)
	}

	ctx := context.Background()

	teachingPlanJSON,
		contextSnapshotJSON,
		guidancePlanJSON :=
		buildAssistantRuntimeFixtureSnapshots(
			t,
		)

	_, err := database.DB.Exec(
		ctx,
		`
		INSERT INTO organizations (
			id,
			name,
			type,
			parent_id,
			admin_user_id,
			settings,
			status,
			education_domain,
			created_at,
			updated_at
		)
		VALUES (
			$1,
			'教学智能体测试区域',
			'region',
			NULL,
			$2,
			'{}'::jsonb,
			'active',
			'mixed',
			NOW(),
			NOW()
		)
		`,
		AssistantFixtureRegionID,
		SeedAdminID,
	)
	if err != nil {
		t.Fatalf(
			"插入教学智能体测试区域失败: %v",
			err,
		)
	}

	_, err = database.DB.Exec(
		ctx,
		`
		INSERT INTO organizations (
			id,
			name,
			type,
			parent_id,
			admin_user_id,
			settings,
			status,
			education_domain,
			created_at,
			updated_at
		)
		VALUES (
			$1,
			'教学智能体测试学校',
			'school',
			$2,
			$3,
			'{}'::jsonb,
			'active',
			'k12',
			NOW(),
			NOW()
		)
		`,
		AssistantFixtureSchoolID,
		AssistantFixtureRegionID,
		SeedAdminID,
	)
	if err != nil {
		t.Fatalf(
			"插入教学智能体测试学校失败: %v",
			err,
		)
	}

	_, err = database.DB.Exec(
		ctx,
		`
		INSERT INTO school_members (
			school_id,
			user_id,
			source,
			joined_at
		)
		VALUES (
			$1,
			$2,
			'manual',
			NOW()
		)
		`,
		AssistantFixtureSchoolID,
		SeedOperatorID,
	)
	if err != nil {
		t.Fatalf(
			"插入教学智能体测试校籍失败: %v",
			err,
		)
	}

	_, err = database.DB.Exec(
		ctx,
		`
		INSERT INTO coursewares (
			id,
			lesson_plan_id,
			user_id,
			title,
			subject,
			grade,
			status,
			page_count,
			source_type,
			publish_state,
			review_level,
			code_share_scope,
			collab_state,
			education_domain,
			created_at,
			updated_at
		)
		VALUES (
			$1,
			NULL,
			$2,
			'三角形面积探究',
			'数学',
			'七年级',
			'preview',
			1,
			'topic_direct',
			'private',
			0,
			'none',
			'idle',
			'k12',
			NOW(),
			NOW()
		)
		`,
		AssistantFixtureCoursewareID,
		SeedOperatorID,
	)
	if err != nil {
		t.Fatalf(
			"插入教学智能体测试课件失败: %v",
			err,
		)
	}

	_, err = database.DB.Exec(
		ctx,
		`
		INSERT INTO courseware_pages (
			id,
			courseware_id,
			page_number,
			title,
			purpose,
			content_summary,
			interaction_type,
			visual_format,
			media_requirements,
			estimated_complexity,
			html_content,
			status,
			page_index,
			idx_cognitive_level,
			idx_interaction_level,
			idx_visual_format,
			created_at,
			updated_at
		)
		VALUES (
			$1,
			$2,
			1,
			'拼接与转化',
			'通过拼接理解面积公式',
			'观察两个相同三角形能够拼成的图形',
			'drag',
			'interactive_diagram',
			'',
			3,
			'<div class="cw-page"><button id="try">开始拼接</button></div>',
			'generated',
			'',
			2,
			2,
			'interactive_diagram',
			NOW(),
			NOW()
		)
		`,
		AssistantFixturePageID,
		AssistantFixtureCoursewareID,
	)
	if err != nil {
		t.Fatalf(
			"插入教学智能体测试页面失败: %v",
			err,
		)
	}

	_, err = database.DB.Exec(
		ctx,
		`
		INSERT INTO ai_assistants (
			id,
			name,
			avatar_emoji,
			description,
			source,
			created_by,
			organization_id,
			group_id,
			full_prompt,
			knowledge_refs,
			subject,
			grade_range,
			scenes,
			is_default_for_scene,
			is_active,
			share_policy,
			education_domain,
			created_at,
			updated_at
		)
		VALUES (
			$1,
			'面积探究伙伴',
			'🧭',
			'只通过提问支持学生完成面积推导',
			'personal',
			$2,
			NULL,
			NULL,
			'你是一位耐心的数学探究伙伴，禁止直接给出最终答案。',
			'[]'::jsonb,
			'数学',
			'七年级',
			'["workshop_design"]'::jsonb,
			'[]'::jsonb,
			TRUE,
			'open',
			'k12',
			NOW(),
			NOW()
		)
		`,
		AssistantFixtureAssistantID,
		SeedOperatorID,
	)
	if err != nil {
		t.Fatalf(
			"插入教学智能体测试AI助手失败: %v",
			err,
		)
	}

	_, err = database.DB.Exec(
		ctx,
		`
		INSERT INTO courseware_assistant_slots (
			id,
			courseware_id,
			page_id,
			assistant_id,
			created_by,
			display_mode,
			display_position,
			title,
			welcome_message,
			teaching_role,
			learning_objective,
			guidance_plan_json,
			context_config_json,
			status,
			created_at,
			updated_at
		)
		VALUES (
			$1,
			$2,
			$3,
			$4,
			$5,
			'floating',
			'bottom_right',
			'面积探究伙伴',
			'先观察页面，再说说你的发现。',
			'通过逐层提问支持学生自主推导。',
			'解释三角形面积公式中除以二的来源。',
			$6::jsonb,
			'{
				"include_visible_text": true,
				"include_page_plan": true,
				"include_interaction_evidence": true,
				"include_adjacent_pages": true,
				"include_lesson_plan_excerpt": false,
				"max_lesson_plan_excerpt_chars": 0
			}'::jsonb,
			'active',
			NOW(),
			NOW()
		)
		`,
		AssistantFixtureSlotID,
		AssistantFixtureCoursewareID,
		AssistantFixturePageID,
		AssistantFixtureAssistantID,
		SeedOperatorID,
		guidancePlanJSON,
	)
	if err != nil {
		t.Fatalf(
			"插入教学智能体测试插槽失败: %v",
			err,
		)
	}

	_, err = database.DB.Exec(
		ctx,
		`
		INSERT INTO token_accounts (
			id,
			account_type,
			owner_id,
			parent_account_id,
			display_name,
			balance,
			frozen_amount,
			total_consumed,
			total_quota,
			monthly_quota,
			status,
			created_at,
			updated_at
		)
		VALUES (
			$1,
			'personal',
			$2,
			NULL,
			'operator1个人积分账户',
			1000,
			0,
			0,
			1000,
			1000,
			'active',
			NOW(),
			NOW()
		)
		`,
		AssistantFixtureTokenAccountID,
		SeedOperatorID,
	)
	if err != nil {
		t.Fatalf(
			"插入教学智能体测试积分账户失败: %v",
			err,
		)
	}

	return &AssistantRuntimeFixture{
		RegionID:       AssistantFixtureRegionID,
		SchoolID:       AssistantFixtureSchoolID,
		CoursewareID:   AssistantFixtureCoursewareID,
		PageID:         AssistantFixturePageID,
		AssistantID:    AssistantFixtureAssistantID,
		SlotID:         AssistantFixtureSlotID,
		TokenAccountID: AssistantFixtureTokenAccountID,

		TeachingPlanJSON:    teachingPlanJSON,
		ContextSnapshotJSON: contextSnapshotJSON,
	}
}

// buildAssistantRuntimeFixtureSnapshots 构造正式可解析快照。
func buildAssistantRuntimeFixtureSnapshots(
	t *testing.T,
) (
	string,
	string,
	string,
) {
	t.Helper()

	plan := models.AssistantDeploymentTeachingPlanSnapshot{
		Version: models.AssistantDeploymentSnapshotVersion,
		Title:   "面积探究伙伴",
		WelcomeMessage:
			"先观察页面，再说说你的发现。",
		TeachingRole:
			"通过逐层提问支持学生自主推导。",
		LearningObjective:
			"解释三角形面积公式中除以二的来源。",
		DisplayMode:
			models.CoursewareAssistantDisplayModeFloating,
		DisplayPosition:
			models.CoursewareAssistantPositionBottomRight,
		GuidancePlan: models.CoursewareAssistantGuidancePlan{
			Version:
				models.CoursewareAssistantProtocolVersion,
			GuidingPrinciples: []string{
				"先让学生观察和尝试",
				"只给必要的分层提示",
			},
			QuestionChain:
				[]models.CoursewareAssistantQuestionStep{
					{
						ID: "q1",
						Prompt:
							"两个相同三角形可以拼成什么熟悉图形？",
						HintLadder: []string{
							"观察对应边的位置。",
							"尝试把一个三角形旋转后再拼。",
						},
					},
				},
			ForbiddenBehaviors: []string{
				"直接给出最终面积公式",
			},
			CompletionCriteria: []string{
				"学生能够说明为什么需要除以二",
			},
			AnswerLeakPolicy:
				models.CoursewareAssistantAnswerLeakPolicy{
					DirectAnswerAllowed: false,
					RequireStudentTry:   true,
					MaximumHintLevel:    3,
					ProhibitedBehaviors: []string{
						"直接公布答案",
					},
					SafeClosureGuidance:
						"学生仍未完成时总结已观察事实并建议重试。",
				},
		},
	}

	contextSnapshot := models.AssistantDeploymentContextSnapshot{
		Version:
			models.AssistantDeploymentSnapshotVersion,
		CurrentPage:
			models.AssistantDeploymentPageContextSnapshot{
				PageID:     AssistantFixturePageID,
				PageNumber: 1,
				Title:      "拼接与转化",
				Purpose:
					"通过拼接理解面积公式",
				ContentSummary:
					"观察两个相同三角形能够拼成的图形",
				VisibleText:
					"开始拼接",
			},
	}

	teachingPlanBytes, err := json.Marshal(
		plan,
	)
	if err != nil {
		t.Fatalf(
			"编码教学智能体方案快照失败: %v",
			err,
		)
	}

	contextBytes, err := json.Marshal(
		contextSnapshot,
	)
	if err != nil {
		t.Fatalf(
			"编码教学智能体上下文快照失败: %v",
			err,
		)
	}

	guidanceBytes, err := json.Marshal(
		plan.GuidancePlan,
	)
	if err != nil {
		t.Fatalf(
			"编码教学智能体引导方案失败: %v",
			err,
		)
	}

	return string(teachingPlanBytes),
		string(contextBytes),
		string(guidanceBytes)
}
