package routes

import "testing"

func TestCoursewareComicStylePreviewRouteMatching(
	t *testing.T,
) {
	cases :=
		[]struct {
			name         string
			path         string
			expectedKind string
		}{
			{
				name:         "generate style preview",
				path:         "/api/v1/coursewares/courseware-1/comic-projects/project-1/generate-style-preview",
				expectedKind: coursewareComicRouteGenerateStylePreview,
			},
			{
				name:         "confirm style preview",
				path:         "/api/v1/coursewares/courseware-2/comic-projects/project-2/confirm-style-preview",
				expectedKind: coursewareComicRouteConfirmStylePreview,
			},
			{
				name:         "confirm style preview trailing slash",
				path:         "/api/v1/coursewares/courseware-3/comic-projects/project-3/confirm-style-preview/",
				expectedKind: coursewareComicRouteConfirmStylePreview,
			},
		}

	for _, testCase := range cases {
		testCase :=
			testCase

		t.Run(
			testCase.name,
			func(t *testing.T) {
				result :=
					matchCoursewareComicRoute(
						testCase.path,
					)

				if !result.Matched {
					t.Fatal(
						"样张路由未匹配",
					)
				}

				if result.Kind !=
					testCase.expectedKind {
					t.Fatalf(
						"路由类型错误：得到%q，期望%q",
						result.Kind,
						testCase.expectedKind,
					)
				}

				if result.CoursewareID == "" ||
					result.ProjectID == "" {
					t.Fatalf(
						"路由ID解析失败：%+v",
						result,
					)
				}
			},
		)
	}
}
