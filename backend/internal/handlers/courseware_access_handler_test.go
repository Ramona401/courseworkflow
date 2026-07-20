package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"tedna/internal/services"
)

func TestHandleCoursewareAccessError(
	t *testing.T,
) {
	t.Parallel()

	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{
			name:       "课件不存在映射404",
			err:        services.ErrCoursewareAccessNotFound,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "无查看权映射403",
			err:        services.ErrCoursewareViewDenied,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "跨教育域映射403",
			err:        services.ErrCoursewareEducationDomainMismatch,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "Actor缺失映射403",
			err:        services.ErrCoursewareActorRequired,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "资源教育域异常映射500",
			err:        services.ErrCoursewareEducationDomainInvalid,
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "运行教育域异常映射500",
			err:        services.ErrCoursewareRuntimeDomainRequired,
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "其它错误映射500",
			err:        errors.New("database unavailable"),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			recorder := httptest.NewRecorder()

			handleCoursewareAccessError(
				recorder,
				test.err,
				"测试访问失败",
			)

			if recorder.Code != test.wantStatus {
				t.Fatalf(
					"期望HTTP %d，实际HTTP %d",
					test.wantStatus,
					recorder.Code,
				)
			}
		})
	}
}
