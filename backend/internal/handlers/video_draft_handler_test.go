package handlers

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseVideoDraftPath(
	t *testing.T,
) {
	tests := []struct {
		name      string
		path      string
		wantCW    string
		wantDraft string
		wantRoute videoDraftRouteKind
	}{
		{
			name:      "集合路径",
			path:      "/api/v1/coursewares/cw-1/video-drafts",
			wantCW:    "cw-1",
			wantRoute: videoDraftRouteCollection,
		},
		{
			name:      "集合路径尾斜杠",
			path:      "/api/v1/coursewares/cw-1/video-drafts/",
			wantCW:    "cw-1",
			wantRoute: videoDraftRouteCollection,
		},
		{
			name:      "单条路径",
			path:      "/api/v1/coursewares/cw-1/video-drafts/draft-1",
			wantCW:    "cw-1",
			wantDraft: "draft-1",
			wantRoute: videoDraftRouteItem,
		},
		{
			name:      "缺少课件ID",
			path:      "/api/v1/coursewares/video-drafts",
			wantRoute: videoDraftRouteInvalid,
		},
		{
			name:      "额外子路径",
			path:      "/api/v1/coursewares/cw-1/video-drafts/draft-1/extra",
			wantRoute: videoDraftRouteInvalid,
		},
		{
			name:      "相似但非法路径",
			path:      "/api/v1/coursewares/cw-1/my-video-drafts",
			wantRoute: videoDraftRouteInvalid,
		},
	}

	for _, testCase := range tests {
		t.Run(
			testCase.name,
			func(t *testing.T) {
				gotCW, gotDraft, gotRoute :=
					parseVideoDraftPath(
						testCase.path,
					)

				if gotCW != testCase.wantCW {
					t.Fatalf(
						"courseware_id不一致: got=%q want=%q",
						gotCW,
						testCase.wantCW,
					)
				}

				if gotDraft != testCase.wantDraft {
					t.Fatalf(
						"draft_id不一致: got=%q want=%q",
						gotDraft,
						testCase.wantDraft,
					)
				}

				if gotRoute != testCase.wantRoute {
					t.Fatalf(
						"路由类型不一致: got=%v want=%v",
						gotRoute,
						testCase.wantRoute,
					)
				}
			},
		)
	}
}

func TestDecodeVideoDraftSaveInput(
	t *testing.T,
) {
	t.Run(
		"合法请求",
		func(t *testing.T) {
			request := httptest.NewRequest(
				"POST",
				"/api/v1/coursewares/cw-1/video-drafts",
				strings.NewReader(
					`{
						"name":"测试草稿",
						"clips_data":[
							{
								"id":"asset-1",
								"url":"/uploads/video.mp4",
								"duration":10,
								"trimStart":0,
								"trimEnd":10,
								"transition":"none",
								"transDur":0.5
							}
						],
						"clip_count":1
					}`,
				),
			)

			recorder := httptest.NewRecorder()

			input, err := decodeVideoDraftSaveInput(
				recorder,
				request,
			)
			if err != nil {
				t.Fatalf(
					"不期望解析失败: %v",
					err,
				)
			}

			if input.Name != "测试草稿" ||
				input.ClipCount != 1 {
				t.Fatalf(
					"解析结果异常: %+v",
					input,
				)
			}
		},
	)

	t.Run(
		"拒绝未知字段",
		func(t *testing.T) {
			request := httptest.NewRequest(
				"POST",
				"/api/v1/coursewares/cw-1/video-drafts",
				strings.NewReader(
					`{
						"name":"测试",
						"clips_data":[],
						"clip_count":0,
						"unexpected":true
					}`,
				),
			)

			recorder := httptest.NewRecorder()

			if _, err := decodeVideoDraftSaveInput(
				recorder,
				request,
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
				"/api/v1/coursewares/cw-1/video-drafts",
				strings.NewReader(
					`{"name":"a","clips_data":[],"clip_count":0}
					 {"name":"b","clips_data":[],"clip_count":0}`,
				),
			)

			recorder := httptest.NewRecorder()

			if _, err := decodeVideoDraftSaveInput(
				recorder,
				request,
			); err == nil {
				t.Fatal(
					"多个JSON值应被拒绝",
				)
			}
		},
	)
}
