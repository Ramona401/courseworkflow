package routes

import "testing"

// TestCoursewareAssistantRouteFallsThroughForNonAssistantPageActions
// 防止教学智能体包装层再次拦截普通课件页面动作。
//
// 这些请求必须返回Matched=false，随后由课程图片、页面编辑或生成主路由处理。
func TestCoursewareAssistantRouteFallsThroughForNonAssistantPageActions(
	t *testing.T,
) {
	t.Parallel()

	paths := []string{
		"/api/v1/coursewares/courseware-1/pages/1/generate-image",
		"/api/v1/coursewares/courseware-1/pages/1/upload-image",
		"/api/v1/coursewares/courseware-1/pages/1/upload-video",
		"/api/v1/coursewares/courseware-1/pages/1/upload-audio",
		"/api/v1/coursewares/courseware-1/pages/1/insert-image",
		"/api/v1/coursewares/courseware-1/pages/1/suggest-image-prompt",
		"/api/v1/coursewares/courseware-1/pages/1/suggest-video-prompt",
		"/api/v1/coursewares/courseware-1/pages/1/generate-video",
		"/api/v1/coursewares/courseware-1/pages/1/save-html",
		"/api/v1/coursewares/courseware-1/pages/1/import-html",
		"/api/v1/coursewares/courseware-1/pages/1/regenerate",
		"/api/v1/coursewares/courseware-1/pages/1/refine",
		"/api/v1/coursewares/courseware-1/pages/1/rollback",
		"/api/v1/coursewares/courseware-1/pages/1/versions",
	}

	for _, path := range paths {
		path := path

		t.Run(
			path,
			func(t *testing.T) {
				t.Parallel()

				matched :=
					matchCoursewareAssistantRoute(
						path,
					)

				if matched.Matched {
					t.Fatalf(
						"普通课件页面动作被教学智能体包装层误拦截: path=%s kind=%s",
						path,
						matched.Kind,
					)
				}
			},
		)
	}
}
