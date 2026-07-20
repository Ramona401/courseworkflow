package handlers

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExtractVideoEditCoursewareID(
	t *testing.T,
) {
	tests := []struct {
		name   string
		path   string
		suffix string
		want   string
	}{
		{
			name:   "标准路径",
			path:   "/api/v1/coursewares/cw-1/videos/trim",
			suffix: "/videos/trim",
			want:   "cw-1",
		},
		{
			name:   "尾斜杠",
			path:   "/api/v1/coursewares/cw-1/videos/trim/",
			suffix: "/videos/trim",
			want:   "cw-1",
		},
		{
			name:   "缺少课件ID",
			path:   "/api/v1/coursewares/videos/trim",
			suffix: "/videos/trim",
			want:   "",
		},
		{
			name:   "额外子路径",
			path:   "/api/v1/coursewares/cw-1/extra/videos/trim",
			suffix: "/videos/trim",
			want:   "",
		},
		{
			name:   "错误动作",
			path:   "/api/v1/coursewares/cw-1/videos/mute",
			suffix: "/videos/trim",
			want:   "",
		},
	}

	for _, testCase := range tests {
		t.Run(
			testCase.name,
			func(t *testing.T) {
				got :=
					extractVideoEditCoursewareID(
						testCase.path,
						testCase.suffix,
					)
				if got != testCase.want {
					t.Fatalf(
						"课件ID不一致: got=%q want=%q",
						got,
						testCase.want,
					)
				}
			},
		)
	}
}

func TestDecodeVideoEditJSON(
	t *testing.T,
) {
	t.Run(
		"合法单对象",
		func(t *testing.T) {
			request := httptest.NewRequest(
				"POST",
				"/api/v1/coursewares/cw-1/videos/trim",
				strings.NewReader(
					`{"asset_id":"asset-1","start_sec":0,"end_sec":3}`,
				),
			)
			recorder := httptest.NewRecorder()

			var body struct {
				AssetID  string  `json:"asset_id"`
				StartSec float64 `json:"start_sec"`
				EndSec   float64 `json:"end_sec"`
			}

			if err := decodeVideoEditJSON(
				recorder,
				request,
				&body,
			); err != nil {
				t.Fatalf(
					"合法正文不应失败: %v",
					err,
				)
			}

			if body.AssetID != "asset-1" ||
				body.EndSec != 3 {
				t.Fatalf(
					"解析结果异常: %+v",
					body,
				)
			}
		},
	)

	t.Run(
		"拒绝未知字段",
		func(t *testing.T) {
			request := httptest.NewRequest(
				"POST",
				"/api/v1/coursewares/cw-1/videos/mute",
				strings.NewReader(
					`{"asset_id":"asset-1","unexpected":true}`,
				),
			)
			recorder := httptest.NewRecorder()

			var body struct {
				AssetID string `json:"asset_id"`
			}

			if err := decodeVideoEditJSON(
				recorder,
				request,
				&body,
			); err == nil {
				t.Fatal(
					"未知字段应被拒绝",
				)
			}
		},
	)

	t.Run(
		"拒绝多个JSON值",
		func(t *testing.T) {
			request := httptest.NewRequest(
				"POST",
				"/api/v1/coursewares/cw-1/videos/mute",
				strings.NewReader(
					`{"asset_id":"asset-1"} {"asset_id":"asset-2"}`,
				),
			)
			recorder := httptest.NewRecorder()

			var body struct {
				AssetID string `json:"asset_id"`
			}

			if err := decodeVideoEditJSON(
				recorder,
				request,
				&body,
			); err == nil {
				t.Fatal(
					"多个JSON值应被拒绝",
				)
			}
		},
	)
}
