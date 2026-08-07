package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"tedna/internal/services"
)

func TestCoursewareComicWorkflowHandlerErrorMapping(
	t *testing.T,
) {
	cases := []struct {
		name       string
		sourceErr  error
		statusCode int
	}{
		{
			name:       "invalid workflow request",
			sourceErr:  services.ErrCoursewareComicWorkflowInvalidRequest,
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "style instruction too long",
			sourceErr:  services.ErrCoursewareComicStyleInstructionTooLong,
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "narrative replan required",
			sourceErr:  services.ErrCoursewareComicNarrativeReplanRequired,
			statusCode: http.StatusConflict,
		},
	}

	for _, testCase := range cases {
		testCase := testCase

		t.Run(
			testCase.name,
			func(t *testing.T) {
				recorder :=
					httptest.NewRecorder()

				wrappedErr :=
					fmt.Errorf(
						"测试包装错误: %w",
						testCase.sourceErr,
					)

				writeCoursewareComicHandlerError(
					recorder,
					wrappedErr,
				)

				if recorder.Code !=
					testCase.statusCode {
					t.Fatalf(
						"HTTP状态码错误：得到%d，期望%d，响应=%s",
						recorder.Code,
						testCase.statusCode,
						recorder.Body.String(),
					)
				}

				if recorder.Body.Len() == 0 {
					t.Fatal(
						"错误响应正文不应为空",
					)
				}
			},
		)
	}
}
