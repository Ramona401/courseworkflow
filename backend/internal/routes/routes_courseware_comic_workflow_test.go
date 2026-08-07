package routes

import "testing"

func TestCoursewareComicWorkflowRouteMatching(
	t *testing.T,
) {
	cases := []struct {
		name         string
		path         string
		matched      bool
		kind         string
		coursewareID string
		projectID    string
	}{
		{
			name:         "confirm storyboard",
			path:         "/api/v1/coursewares/courseware-1/comic-projects/project-1/confirm-storyboard",
			matched:      true,
			kind:         coursewareComicRouteConfirmStoryboard,
			coursewareID: "courseware-1",
			projectID:    "project-1",
		},
		{
			name:         "style settings",
			path:         "/api/v1/coursewares/courseware-2/comic-projects/project-2/style-settings",
			matched:      true,
			kind:         coursewareComicRouteStyleSettings,
			coursewareID: "courseware-2",
			projectID:    "project-2",
		},
		{
			name:         "style settings trailing slash",
			path:         "/api/v1/coursewares/courseware-3/comic-projects/project-3/style-settings/",
			matched:      true,
			kind:         coursewareComicRouteStyleSettings,
			coursewareID: "courseware-3",
			projectID:    "project-3",
		},
		{
			name:         "existing plan route preserved",
			path:         "/api/v1/coursewares/courseware-4/comic-projects/project-4/plan",
			matched:      true,
			kind:         coursewareComicRoutePlan,
			coursewareID: "courseware-4",
			projectID:    "project-4",
		},
		{
			name:         "existing generate route preserved",
			path:         "/api/v1/coursewares/courseware-5/comic-projects/project-5/generate",
			matched:      true,
			kind:         coursewareComicRouteGenerate,
			coursewareID: "courseware-5",
			projectID:    "project-5",
		},
		{
			name:    "unknown comic action is invalid",
			path:    "/api/v1/coursewares/courseware-6/comic-projects/project-6/unknown-action",
			matched: true,
			kind:    coursewareComicRouteInvalid,
		},
		{
			name:    "extra reserved segment is invalid",
			path:    "/api/v1/coursewares/courseware-7/comic-projects/project-7/style-settings/extra",
			matched: true,
			kind:    coursewareComicRouteInvalid,
		},
		{
			name:    "non comic route remains unmatched",
			path:    "/api/v1/coursewares/courseware-8/pages",
			matched: false,
			kind:    "",
		},
	}

	for _, testCase := range cases {
		testCase := testCase

		t.Run(
			testCase.name,
			func(t *testing.T) {
				result :=
					matchCoursewareComicRoute(
						testCase.path,
					)

				if result.Matched !=
					testCase.matched {
					t.Fatalf(
						"匹配状态错误：得到%v，期望%v，结果=%+v",
						result.Matched,
						testCase.matched,
						result,
					)
				}

				if result.Kind !=
					testCase.kind {
					t.Fatalf(
						"路由类型错误：得到%q，期望%q",
						result.Kind,
						testCase.kind,
					)
				}

				if testCase.coursewareID != "" &&
					result.CoursewareID !=
						testCase.coursewareID {
					t.Fatalf(
						"课件ID错误：得到%q，期望%q",
						result.CoursewareID,
						testCase.coursewareID,
					)
				}

				if testCase.projectID != "" &&
					result.ProjectID !=
						testCase.projectID {
					t.Fatalf(
						"项目ID错误：得到%q，期望%q",
						result.ProjectID,
						testCase.projectID,
					)
				}
			},
		)
	}
}
