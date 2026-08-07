package services

// assistant_deployment_service_test.go
//
// 本测试只验证纯内存发布策略、生产状态和不可变快照装配。
// 不连接数据库、不调用AI，也不创建真实部署。

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"tedna/internal/models"
)

// TestAssistantDeploymentServiceNormalizesOrigins 验证精确Origin规范化。
func TestAssistantDeploymentServiceNormalizesOrigins(t *testing.T) {
	origins, err := normalizeAssistantDeploymentAllowedOrigins(
		[]string{
			"https://Course.Example/",
			"https://course.example:443",
			"http://127.0.0.1:5173/",
		},
	)
	if err != nil {
		t.Fatalf("合法来源规范化失败: %v", err)
	}

	expected := []string{
		"http://127.0.0.1:5173",
		"https://course.example",
	}
	if len(origins) != len(expected) {
		t.Fatalf("去重后来源数量错误: %#v", origins)
	}
	for index := range expected {
		if origins[index] != expected[index] {
			t.Fatalf(
				"来源规范化错误: expected=%s actual=%s",
				expected[index],
				origins[index],
			)
		}
	}
}

// TestAssistantDeploymentServiceRejectsUnsafeOrigins 验证不安全来源全部拒绝。
func TestAssistantDeploymentServiceRejectsUnsafeOrigins(t *testing.T) {
	invalid := []string{
		"http://course.example",
		"https://*.example.com",
		"https://course.example/path",
		"https://course.example?token=x",
		"https://user:pass@course.example",
		"javascript:alert(1)",
	}

	for _, candidate := range invalid {
		_, err := normalizeAssistantDeploymentAllowedOrigins(
			[]string{candidate},
		)
		if !errors.Is(err, ErrAssistantDeploymentOriginInvalid) {
			t.Fatalf(
				"不安全来源应被拒绝: origin=%s error=%v",
				candidate,
				err,
			)
		}
	}
}

// TestAssistantDeploymentServiceValidatesPolicy 验证额度和有效期。
func TestAssistantDeploymentServiceValidatesPolicy(t *testing.T) {
	now := time.Date(
		2026,
		time.July,
		26,
		12,
		0,
		0,
		0,
		time.UTC,
	)
	future := now.Add(24 * time.Hour)

	policy, err := normalizeAssistantDeploymentPolicy(
		100,
		12,
		[]string{"https://course.example"},
		&future,
		now,
	)
	if err != nil {
		t.Fatalf("合法部署策略校验失败: %v", err)
	}
	if policy.DailyCallLimit != 100 ||
		policy.PerSessionTurnLimit != 12 ||
		policy.AllowedOriginsJSON != `["https://course.example"]` {
		t.Fatalf("部署策略规范化错误: %#v", policy)
	}

	past := now.Add(-time.Minute)
	_, err = normalizeAssistantDeploymentPolicy(
		100,
		12,
		[]string{"https://course.example"},
		&past,
		now,
	)
	if !errors.Is(err, ErrAssistantDeploymentPolicyInvalid) {
		t.Fatalf("过期策略应被拒绝: %v", err)
	}
}

// TestAssistantDeploymentServicePublishableCourseware 验证生产状态和审核锁。
func TestAssistantDeploymentServicePublishableCourseware(t *testing.T) {
	valid := &models.Courseware{
		EducationDomain: models.EducationDomainK12,
		Status:          "preview",
		PublishState:    models.CWPublishPrivate,
	}
	if err := validateAssistantDeploymentPublishableCourseware(valid); err != nil {
		t.Fatalf("preview课件应允许发布: %v", err)
	}

	valid.Status = "confirmed"
	if err := validateAssistantDeploymentPublishableCourseware(valid); err != nil {
		t.Fatalf("confirmed课件应允许发布: %v", err)
	}

	valid.Status = models.CoursewareStatusInPipeline
	if !errors.Is(
		validateAssistantDeploymentPublishableCourseware(valid),
		ErrAssistantDeploymentCoursewareNotPublishable,
	) {
		t.Fatal("in_pipeline课件必须拒绝发布")
	}

	valid.Status = "preview"
	valid.PublishState = models.CWPublishSubmitted
	if !errors.Is(
		validateAssistantDeploymentPublishableCourseware(valid),
		ErrAssistantDeploymentCoursewareNotPublishable,
	) {
		t.Fatal("submitted课件必须拒绝发布")
	}
}

// TestAssistantDeploymentServiceBuildsImmutableSnapshot 验证完整快照内容和隔离。
func TestAssistantDeploymentServiceBuildsImmutableSnapshot(t *testing.T) {
	courseware := &models.Courseware{
		ID:              "11111111-1111-1111-1111-111111111111",
		UserID:          "22222222-2222-2222-2222-222222222222",
		Title:           "三角形面积探究",
		Subject:         "数学",
		Grade:           "五年级",
		EducationDomain: models.EducationDomainK12,
	}

	assistantID := "33333333-3333-3333-3333-333333333333"
	slot := &models.CoursewareAssistantSlotView{
		ID:                "44444444-4444-4444-4444-444444444444",
		CoursewareID:      courseware.ID,
		PageID:            "55555555-5555-5555-5555-555555555555",
		AssistantID:       &assistantID,
		DisplayMode:       models.CoursewareAssistantDisplayModeFloating,
		DisplayPosition:   models.CoursewareAssistantPositionBottomRight,
		Title:             "面积探究伙伴",
		WelcomeMessage:    "先观察，再说说你的发现。",
		TeachingRole:      "通过逐层提问支持学生自主推导。",
		LearningObjective: "学生能够解释公式中除以二的来源。",
		Status:            models.CoursewareAssistantSlotStatusActive,
		GuidancePlan: models.CoursewareAssistantGuidancePlan{
			Version: models.CoursewareAssistantProtocolVersion,
			AnswerLeakPolicy: models.CoursewareAssistantAnswerLeakPolicy{
				DirectAnswerAllowed: false,
				RequireStudentTry:   true,
				MaximumHintLevel:    3,
			},
		},
	}

	assistant := &models.AIAssistant{
		ID:         assistantID,
		FullPrompt: "这是只能保存在后端版本表中的完整助手提示词。",
	}

	snapshot := models.AssistantDeploymentContextSnapshot{
		Version: models.AssistantDeploymentSnapshotVersion,
		CurrentPage: models.AssistantDeploymentPageContextSnapshot{
			PageID:      slot.PageID,
			PageNumber:  3,
			Title:       "拼接与转化",
			VisibleText: "拖动两个相同三角形，观察拼成的图形。",
			InteractionEvidence: emptyCoursewareAssistantInteractionEvidence(
				"drag",
			),
		},
	}
	snapshotJSON, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("构造测试上下文失败: %v", err)
	}

	contextResult := &CoursewareAssistantContextBuildResult{
		Snapshot:     snapshot,
		SnapshotJSON: string(snapshotJSON),
		SnapshotHash: strings.Repeat("a", 64),
		PageHTMLHash: strings.Repeat("b", 64),
	}

	policy := &assistantDeploymentNormalizedPolicy{
		DailyCallLimit:      200,
		PerSessionTurnLimit: 10,
		AllowedOrigins:      []string{"https://course.example"},
		AllowedOriginsJSON:  `["https://course.example"]`,
	}

	version, err := buildAssistantDeploymentVersionRecord(
		courseware,
		slot,
		assistant,
		contextResult,
		policy,
		courseware.UserID,
	)
	if err != nil {
		t.Fatalf("不可变快照装配失败: %v", err)
	}

	if version.AssistantPromptSnapshot != assistant.FullPrompt {
		t.Fatal("完整助手提示词没有被固化到后端版本快照")
	}
	if len(version.AssistantPromptHash) != 64 ||
		len(version.ContextSnapshotHash) != 64 ||
		len(version.PageHTMLHash) != 64 {
		t.Fatal("版本哈希长度不完整")
	}
	if strings.Contains(
		version.TeachingPlanJSON+version.CoursewareSnapshotJSON,
		assistant.FullPrompt,
	) {
		t.Fatal("提示词不得混入教学方案或课件策略JSON")
	}
	if !strings.Contains(
		version.CoursewareSnapshotJSON,
		`"deployment_policy"`,
	) || !strings.Contains(
		version.CoursewareSnapshotJSON,
		`"https://course.example"`,
	) {
		t.Fatalf(
			"课件快照缺少发布策略审计字段: %s",
			version.CoursewareSnapshotJSON,
		)
	}
}
