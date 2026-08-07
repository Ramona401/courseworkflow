package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tedna/internal/models"
)

func TestCWAIReviewSessionViewIncludesOnlySafeConfig(
	t *testing.T,
) {
	session := &models.CoursewareAIReviewSession{
		ID:           "session-test",
		CoursewareID: "courseware-test",
		ReviewerID:   "reviewer-test",

		ReviewConfigSchemaVersion:  models.CWAIReviewConfigSchemaVersion,
		ReviewDimensionsJSON:       `["teaching_logic","custom"]`,
		CustomDimensionDescription: "重点检查校本案例",
		LessonReferenceMode:        models.CWAIReviewLessonReferenceNoLesson,
		ReviewConfigHash:           strings.Repeat("b", 64),

		SystemPromptSnapshot:    "绝密系统提示词",
		AssistantPromptSnapshot: "绝密助手提示词",
		BaselineJSON:            `{"lesson_plan":{"content":"绝密教案正文"}}`,
		PageIndexJSON:           `[{"visible_text":"绝密页面正文"}]`,
		ContinuityLedgerJSON:    `{"secret":"绝密连续性账本"}`,
		FinalReportJSON:         "{}",
	}

	view :=
		buildCoursewareAIReviewSessionView(
			session,
		)

	encoded, err := json.Marshal(view)
	if err != nil {
		t.Fatalf("序列化安全响应失败: %v", err)
	}

	body := string(encoded)

	required := []string{
		`"review_config"`,
		`"review_dimensions":["teaching_logic","custom"]`,
		`"lesson_reference_mode":"no_lesson"`,
		`"uses_lesson_materials":false`,
		`"review_config_hash":"` +
			strings.Repeat("b", 64) +
			`"`,
	}

	for _, marker := range required {
		if !strings.Contains(body, marker) {
			t.Fatalf(
				"安全响应缺少配置字段: %s",
				marker,
			)
		}
	}

	forbidden := []string{
		"绝密系统提示词",
		"绝密助手提示词",
		"绝密教案正文",
		"绝密页面正文",
		"绝密连续性账本",
		"system_prompt_snapshot",
		"assistant_prompt_snapshot",
		"baseline_json",
		"page_index_json",
		"continuity_ledger_json",
	}

	for _, marker := range forbidden {
		if strings.Contains(body, marker) {
			t.Fatalf(
				"浏览器安全响应泄露内部字段: %s",
				marker,
			)
		}
	}
}

func TestCWAIReviewPrepareRequestStrictJSON(
	t *testing.T,
) {
	t.Run(
		"valid configured request",
		func(t *testing.T) {
			recorder :=
				httptest.NewRecorder()

			request :=
				httptest.NewRequest(
					http.MethodPost,
					"/api/v1/courseware-ai-reviews",
					strings.NewReader(`{
						"courseware_id": "cw-1",
						"review_level": 1,
						"assistant_id": "",
						"review_dimensions": ["teaching_logic"],
						"custom_dimension_description": "",
						"lesson_reference_mode": "no_lesson"
					}`),
				)

			var target PrepareRequest

			if !decodeCoursewareAIReviewPrepareRequest(
				recorder,
				request,
				&target,
			) {
				t.Fatalf(
					"合法配置请求不应被拒绝，响应=%s",
					recorder.Body.String(),
				)
			}

			if target.ReviewDimensions == nil ||
				len(*target.ReviewDimensions) != 1 {
				t.Fatalf(
					"审核维度未正确解码: %+v",
					target.ReviewDimensions,
				)
			}

			if target.LessonReferenceMode == nil ||
				*target.LessonReferenceMode !=
					"no_lesson" {
				t.Fatalf(
					"教案模式未正确解码: %+v",
					target.LessonReferenceMode,
				)
			}
		},
	)

	testCases := []struct {
		name string
		body string
	}{
		{
			name: "unknown field",
			body: `{
				"courseware_id": "cw-1",
				"review_level": 1,
				"unexpected": true
			}`,
		},
		{
			name: "trailing json",
			body: `{
				"courseware_id": "cw-1",
				"review_level": 1
			} {}`,
		},
	}

	for _, testCase := range testCases {
		t.Run(
			testCase.name,
			func(t *testing.T) {
				recorder :=
					httptest.NewRecorder()

				request :=
					httptest.NewRequest(
						http.MethodPost,
						"/api/v1/courseware-ai-reviews",
						strings.NewReader(
							testCase.body,
						),
					)

				var target PrepareRequest

				if decodeCoursewareAIReviewPrepareRequest(
					recorder,
					request,
					&target,
				) {
					t.Fatalf(
						"严格JSON请求应被拒绝",
					)
				}

				if recorder.Code !=
					http.StatusBadRequest {
					t.Fatalf(
						"错误HTTP状态: got=%d want=%d",
						recorder.Code,
						http.StatusBadRequest,
					)
				}
			},
		)
	}
}
