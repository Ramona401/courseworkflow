package services

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func validVideoDraftInput() *VideoDraftSaveInput {
	return &VideoDraftSaveInput{
		Name: "  第一版草稿  ",
		ClipsData: json.RawMessage(
			`[
				{
					"id":"asset-1",
					"url":"/uploads/courseware-assets/video-1.mp4",
					"label":"片段一",
					"duration":12.5,
					"trimStart":1,
					"trimEnd":10,
					"transition":"fade",
					"transDur":0.5,
					"audioVolume":0.8
				},
				{
					"id":"asset-2",
					"url":"https://example.test/video-2.mp4",
					"label":"片段二",
					"duration":8,
					"trimStart":0,
					"trimEnd":8,
					"transition":"none",
					"transDur":0
				}
			]`,
		),
		ClipCount: 2,
	}
}

func TestNormalizeVideoDraftInput(
	t *testing.T,
) {
	t.Run(
		"合法草稿",
		func(t *testing.T) {
			name, clipsJSON, clipCount, err :=
				normalizeVideoDraftInput(
					validVideoDraftInput(),
				)

			if err != nil {
				t.Fatalf(
					"不期望错误: %v",
					err,
				)
			}

			if name != "第一版草稿" {
				t.Fatalf(
					"名称未正确规范化: %q",
					name,
				)
			}

			if clipCount != 2 {
				t.Fatalf(
					"片段数不一致: %d",
					clipCount,
				)
			}

			if strings.Contains(clipsJSON, "\n") {
				t.Fatal(
					"保存JSON应已压缩",
				)
			}
		},
	)

	tests := []struct {
		name   string
		mutate func(*VideoDraftSaveInput)
	}{
		{
			name: "片段数不一致",
			mutate: func(input *VideoDraftSaveInput) {
				input.ClipCount = 1
			},
		},
		{
			name: "clips_data不是数组",
			mutate: func(input *VideoDraftSaveInput) {
				input.ClipsData = json.RawMessage(
					`{"id":"asset-1"}`,
				)
				input.ClipCount = 1
			},
		},
		{
			name: "缺少片段ID",
			mutate: func(input *VideoDraftSaveInput) {
				input.ClipsData = json.RawMessage(
					`[{
						"id":"",
						"url":"/video.mp4",
						"duration":10,
						"trimStart":0,
						"trimEnd":10,
						"transition":"none",
						"transDur":0
					}]`,
				)
				input.ClipCount = 1
			},
		},
		{
			name: "片段ID重复",
			mutate: func(input *VideoDraftSaveInput) {
				input.ClipsData = json.RawMessage(
					`[
						{
							"id":"asset-1",
							"url":"/one.mp4",
							"duration":10,
							"trimStart":0,
							"trimEnd":10,
							"transition":"none",
							"transDur":0
						},
						{
							"id":"asset-1",
							"url":"/two.mp4",
							"duration":10,
							"trimStart":0,
							"trimEnd":10,
							"transition":"none",
							"transDur":0
						}
					]`,
				)
				input.ClipCount = 2
			},
		},
		{
			name: "裁剪终点超过视频时长",
			mutate: func(input *VideoDraftSaveInput) {
				input.ClipsData = json.RawMessage(
					`[{
						"id":"asset-1",
						"url":"/video.mp4",
						"duration":10,
						"trimStart":0,
						"trimEnd":20,
						"transition":"none",
						"transDur":0
					}]`,
				)
				input.ClipCount = 1
			},
		},
		{
			name: "音量越界",
			mutate: func(input *VideoDraftSaveInput) {
				input.ClipsData = json.RawMessage(
					`[{
						"id":"asset-1",
						"url":"/video.mp4",
						"duration":10,
						"trimStart":0,
						"trimEnd":10,
						"transition":"none",
						"transDur":0,
						"audioVolume":1.5
					}]`,
				)
				input.ClipCount = 1
			},
		},
		{
			name: "草稿名称过长",
			mutate: func(input *VideoDraftSaveInput) {
				input.Name = strings.Repeat(
					"名",
					videoDraftMaxNameRunes+1,
				)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(
			testCase.name,
			func(t *testing.T) {
				input := validVideoDraftInput()
				testCase.mutate(input)

				_, _, _, err :=
					normalizeVideoDraftInput(input)

				if !errors.Is(
					err,
					ErrVideoDraftInputInvalid,
				) {
					t.Fatalf(
						"期望参数错误，实际=%v",
						err,
					)
				}
			},
		)
	}
}
