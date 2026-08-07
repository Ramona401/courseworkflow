package routes

import "testing"

// TestCoursewareAssistantRouteKeepsKnownAssistantActions
// 确认修复普通页面动作下沉时，不会破坏已登记的教学智能体页面动作。
func TestCoursewareAssistantRouteKeepsKnownAssistantActions(
	t *testing.T,
) {
	t.Parallel()

	tests := []struct {
		path string
		kind string
	}{
		{
			path: "/api/v1/coursewares/courseware-1/pages/page-id-1/assistant-slot",
			kind: coursewareAssistantRoutePageSlot,
		},
		{
			path: "/api/v1/coursewares/courseware-1/pages/page-id-1/assistant-context",
			kind: coursewareAssistantRoutePageContext,
		},
		{
			path: "/api/v1/coursewares/courseware-1/pages/page-id-1/assistant-plan",
			kind: coursewareAssistantRoutePagePlan,
		},
		{
			path: "/api/v1/coursewares/courseware-1/pages/page-id-1/assistant-deployment",
			kind: coursewareAssistantRoutePublishDeployment,
		},
	}

	for _, item := range tests {
		item := item

		t.Run(
			item.kind,
			func(t *testing.T) {
				t.Parallel()

				matched :=
					matchCoursewareAssistantRoute(
						item.path,
					)

				if !matched.Matched {
					t.Fatalf(
						"已登记的教学智能体动作没有被匹配: path=%s",
						item.path,
					)
				}

				if matched.Kind != item.kind {
					t.Fatalf(
						"教学智能体动作匹配类型错误: path=%s got=%s want=%s",
						item.path,
						matched.Kind,
						item.kind,
					)
				}
			},
		)
	}
}

// TestCoursewareAssistantRouteRejectsUnknownReservedAction
// 未登记但使用assistant-保留前缀的动作仍必须fail-closed。
func TestCoursewareAssistantRouteRejectsUnknownReservedAction(
	t *testing.T,
) {
	t.Parallel()

	path :=
		"/api/v1/coursewares/courseware-1/pages/page-id-1/assistant-unknown"

	matched :=
		matchCoursewareAssistantRoute(
			path,
		)

	if !matched.Matched {
		t.Fatalf(
			"未知assistant-保留动作被错误下沉: path=%s",
			path,
		)
	}

	if matched.Kind !=
		coursewareAssistantRouteInvalid {
		t.Fatalf(
			"未知assistant-保留动作没有标记为invalid: got=%s",
			matched.Kind,
		)
	}
}
