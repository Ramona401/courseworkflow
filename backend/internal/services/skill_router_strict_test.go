package services

import (
	"context"
	"testing"

	"tedna/internal/models"
)

type strictAssistantListerStub struct {
	response *models.AIAssistantListResponse
	err      error
}

func (stub *strictAssistantListerStub) ListAssistants(
	_ context.Context,
	_ *AssistantActorContext,
	_ string,
	_ string,
	_ string,
	_ bool,
) (*models.AIAssistantListResponse, error) {
	return stub.response, stub.err
}

func TestRouteDefaultAssistantRejectsBroadGradeFallback(
	t *testing.T,
) {
	oldFlag := skillRouterDefaultAssistantEnabled
	skillRouterDefaultAssistantEnabled = true
	defer func() {
		skillRouterDefaultAssistantEnabled = oldFlag
	}()

	actor := &AssistantActorContext{
		UserID: "teacher-1",
		Role:   models.RoleOperator,
	}

	lister := &strictAssistantListerStub{
		response: &models.AIAssistantListResponse{
			Assistants: []*models.AIAssistantListItem{
				{
					ID:         "assistant-high-school",
					Name:       "高中通用助手",
					Subject:    "语文",
					GradeRange: "高中",
					Scenes: []string{
						models.SceneWorkshopAnalyze,
					},
					Source: models.AssistantSourceSystem,
				},
			},
			Total: 1,
		},
	}

	got := RouteDefaultAssistant(
		context.Background(),
		lister,
		actor,
		"analyze",
		"语文",
		"高三",
	)

	if got != "" {
		t.Fatalf(
			"只有高中通用助手时应返回空：got=%q",
			got,
		)
	}
}

func TestRouteDefaultAssistantChoosesOnlyExactGrade(
	t *testing.T,
) {
	oldFlag := skillRouterDefaultAssistantEnabled
	skillRouterDefaultAssistantEnabled = true
	defer func() {
		skillRouterDefaultAssistantEnabled = oldFlag
	}()

	actor := &AssistantActorContext{
		UserID: "teacher-1",
		Role:   models.RoleOperator,
	}

	lister := &strictAssistantListerStub{
		response: &models.AIAssistantListResponse{
			Assistants: []*models.AIAssistantListItem{
				{
					ID:         "assistant-grade-11",
					Name:       "高二助手",
					Subject:    "语文",
					GradeRange: "高二",
					Scenes: []string{
						models.SceneWorkshopAnalyze,
					},
					Source: models.AssistantSourcePersonal,
				},
				{
					ID:         "assistant-grade-12",
					Name:       "高三助手",
					Subject:    "语文",
					GradeRange: "十二年级",
					Scenes: []string{
						models.SceneWorkshopAnalyze,
					},
					Source: models.AssistantSourceSystem,
				},
			},
			Total: 2,
		},
	}

	got := RouteDefaultAssistant(
		context.Background(),
		lister,
		actor,
		"analyze",
		"语文",
		"高三",
	)

	if got != "assistant-grade-12" {
		t.Fatalf(
			"应只选择高三助手：got=%q",
			got,
		)
	}
}
