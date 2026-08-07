package integration

// assistant_runtime_fixture_records.go
//
// 提供教学智能体部署、不可变版本和运行会话记录构造。
// 本文件不写基础组织、课件、页面或积分账户。

import (
	"context"
	"strings"
	"testing"
	"time"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// NewDeploymentRecords 构造首发仓储输入。
func (fixture *AssistantRuntimeFixture) NewDeploymentRecords() (
	*models.AssistantDeployment,
	*models.AssistantDeploymentVersion,
) {
	slotID := fixture.SlotID
	assistantID := fixture.AssistantID

	deployment := &models.AssistantDeployment{
		SlotID:          &slotID,
		CoursewareID:    fixture.CoursewareID,
		PageID:          fixture.PageID,
		OwnerUserID:     SeedOperatorID,
		SchoolID:        fixture.SchoolID,
		EducationDomain: "k12",
		AccessMode:
			models.AssistantDeploymentAccessOriginAllowlist,
		Status:
			models.AssistantDeploymentStatusActive,
		DailyCallLimit:      50,
		PerSessionTurnLimit: 5,
		AllowedOriginsJSON:
			`["https://course.example"]`,
	}

	version := &models.AssistantDeploymentVersion{
		AssistantID: &assistantID,
		AssistantPromptSnapshot:
			"你是一位耐心的数学探究伙伴，禁止直接给出最终答案。",
		AssistantPromptHash:
			strings.Repeat("a", 64),
		TeachingPlanJSON:
			fixture.TeachingPlanJSON,
		ContextSnapshotJSON:
			fixture.ContextSnapshotJSON,
		ContextSnapshotHash:
			strings.Repeat("b", 64),
		PageHTMLHash:
			strings.Repeat("c", 64),
		CoursewareSnapshotJSON:
			`{
				"version":"v1",
				"courseware_id":"10000000-0000-4000-8000-000000000003",
				"page_id":"10000000-0000-4000-8000-000000000004",
				"education_domain":"k12"
			}`,
		CreatedBy:
			SeedOperatorID,
	}

	return deployment,
		version
}

// CreateDeployment 创建合法部署和版本1。
func (fixture *AssistantRuntimeFixture) CreateDeployment(
	t *testing.T,
) (
	*models.AssistantDeployment,
	*models.AssistantDeploymentVersion,
) {
	t.Helper()

	deployment,
		version :=
		fixture.NewDeploymentRecords()

	if err := repository.CreateAssistantDeploymentWithFirstVersion(
		context.Background(),
		deployment,
		version,
	); err != nil {
		t.Fatalf(
			"创建教学智能体测试部署失败: %v",
			err,
		)
	}

	return deployment,
		version
}

// CreateRuntimeSession 创建绑定当前版本的短时运行会话。
func (fixture *AssistantRuntimeFixture) CreateRuntimeSession(
	t *testing.T,
	deployment *models.AssistantDeployment,
	sessionID string,
	jtiHash string,
) *models.AssistantRuntimeSession {
	t.Helper()

	expiresAt := time.Now().UTC().
		Add(10 * time.Minute)

	session, err := repository.CreateAssistantRuntimeSession(
		context.Background(),
		sessionID,
		&models.AssistantRuntimeSessionCreateInput{
			DeploymentID:
				deployment.ID,
			DeploymentVersion:
				deployment.CurrentVersion,
			TokenJTIHash:
				jtiHash,
			AnonymousClientHash:
				strings.Repeat("d", 64),
			OriginSnapshot:
				AssistantFixtureOrigin,
			IPHash:
				strings.Repeat("e", 64),
			SessionKind:
				models.AssistantRuntimeSessionKindExternal,
			MaxTurns:
				deployment.PerSessionTurnLimit,
			ExpiresAt:
				expiresAt,
		},
	)
	if err != nil {
		t.Fatalf(
			"创建教学智能体测试运行会话失败: %v",
			err,
		)
	}

	return session
}
