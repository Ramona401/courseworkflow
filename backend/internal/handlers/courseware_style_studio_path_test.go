package handlers

import "testing"

// TestParseCoursewareStyleStudioPath 验证全部公开路径都能稳定解析。
func TestParseCoursewareStyleStudioPath(
	t *testing.T,
) {
	const coursewareID = "11111111-1111-1111-1111-111111111111"

	const sessionID = "22222222-2222-2222-2222-222222222222"

	testCases := []struct {
		name string
		path string

		wantAction string

		wantCourseware string
		wantSession    string

		wantError bool
	}{
		{
			name: "当前活动会话",
			path: "/api/v1/coursewares/" +
				coursewareID +
				"/style-studio",
			wantAction:     coursewareStyleStudioActionActive,
			wantCourseware: coursewareID,
		},
		{
			name: "尾部斜杠",
			path: "/api/v1/coursewares/" +
				coursewareID +
				"/style-studio/",
			wantAction:     coursewareStyleStudioActionActive,
			wantCourseware: coursewareID,
		},
		{
			name: "创建会话",
			path: "/api/v1/coursewares/" +
				coursewareID +
				"/style-studio/sessions",
			wantAction:     coursewareStyleStudioActionSessions,
			wantCourseware: coursewareID,
		},
		{
			name: "上传参考图",
			path: "/api/v1/coursewares/" +
				coursewareID +
				"/style-studio/upload-reference",
			wantAction:     coursewareStyleStudioActionUploadReference,
			wantCourseware: coursewareID,
		},
		{
			name: "读取指定会话",
			path: "/api/v1/coursewares/" +
				coursewareID +
				"/style-studio/sessions/" +
				sessionID,
			wantAction:     coursewareStyleStudioActionSession,
			wantCourseware: coursewareID,
			wantSession:    sessionID,
		},
		{
			name: "发送消息",
			path: "/api/v1/coursewares/" +
				coursewareID +
				"/style-studio/sessions/" +
				sessionID +
				"/messages",
			wantAction:     coursewareStyleStudioActionMessages,
			wantCourseware: coursewareID,
			wantSession:    sessionID,
		},
		{
			name: "生成预览",
			path: "/api/v1/coursewares/" +
				coursewareID +
				"/style-studio/sessions/" +
				sessionID +
				"/previews",
			wantAction:     coursewareStyleStudioActionPreviews,
			wantCourseware: coursewareID,
			wantSession:    sessionID,
		},
		{
			name: "确认风格",
			path: "/api/v1/coursewares/" +
				coursewareID +
				"/style-studio/sessions/" +
				sessionID +
				"/confirm",
			wantAction:     coursewareStyleStudioActionConfirm,
			wantCourseware: coursewareID,
			wantSession:    sessionID,
		},
		{
			name: "错误前缀",
			path: "/api/v1/other/" +
				coursewareID +
				"/style-studio",
			wantError: true,
		},
		{
			name: "未知会话动作",
			path: "/api/v1/coursewares/" +
				coursewareID +
				"/style-studio/sessions/" +
				sessionID +
				"/unknown",
			wantError: true,
		},
		{
			name: "多余层级",
			path: "/api/v1/coursewares/" +
				coursewareID +
				"/style-studio/sessions/" +
				sessionID +
				"/messages/extra",
			wantError: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(
			testCase.name,
			func(t *testing.T) {
				result, err :=
					parseCoursewareStyleStudioPath(
						testCase.path,
					)

				if testCase.wantError {
					if err == nil {
						t.Fatalf(
							"预期路径解析失败，实际成功：%+v",
							result,
						)
					}

					return
				}

				if err != nil {
					t.Fatalf(
						"路径解析失败：%v",
						err,
					)
				}

				if result.CoursewareID !=
					testCase.wantCourseware {
					t.Fatalf(
						"课件ID错误：got=%s want=%s",
						result.CoursewareID,
						testCase.wantCourseware,
					)
				}

				if result.SessionID !=
					testCase.wantSession {
					t.Fatalf(
						"会话ID错误：got=%s want=%s",
						result.SessionID,
						testCase.wantSession,
					)
				}

				if result.Action !=
					testCase.wantAction {
					t.Fatalf(
						"操作类型错误：got=%s want=%s",
						result.Action,
						testCase.wantAction,
					)
				}
			},
		)
	}
}
