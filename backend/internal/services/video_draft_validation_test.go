package services

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func validVideoDraftInputForTest() *VideoDraftSaveInput {
	return &VideoDraftSaveInput{
		Name: "测试草稿",
		ClipsData: json.RawMessage(`[
			{
				"id":"asset-1",
				"url":"/uploads/courseware-assets/cw-1/videos/a.mp4",
				"duration":10,
				"trimStart":0,
				"trimEnd":10,
				"transition":"none",
				"transDur":0.5,
				"audioMuted":false,
				"audioVolume":1
			}
		]`),
		ClipCount: 1,
	}
}

func TestNormalizeVideoDraftInputValid(
	t *testing.T,
) {
	name, clipsJSON, count, err :=
		normalizeVideoDraftInput(
			validVideoDraftInputForTest(),
		)
	if err != nil {
		t.Fatalf(
			"合法草稿不应失败: %v",
			err,
		)
	}

	if name != "测试草稿" {
		t.Fatalf(
			"名称不一致: %q",
			name,
		)
	}

	if count != 1 {
		t.Fatalf(
			"片段数不一致: %d",
			count,
		)
	}

	if strings.Contains(
		clipsJSON,
		"\n",
	) {
		t.Fatal(
			"规范化JSON不应包含换行",
		)
	}
}

func TestNormalizeVideoDraftInputRejectsCountMismatch(
	t *testing.T,
) {
	input := validVideoDraftInputForTest()
	input.ClipCount = 2

	_, _, _, err := normalizeVideoDraftInput(input)
	if !errors.Is(
		err,
		ErrVideoDraftInputInvalid,
	) {
		t.Fatalf(
			"数量不一致应返回输入错误: %v",
			err,
		)
	}
}

func TestNormalizeVideoDraftInputRejectsDuplicateIDs(
	t *testing.T,
) {
	input := validVideoDraftInputForTest()
	input.ClipsData = json.RawMessage(`[
		{
			"id":"asset-1",
			"url":"/uploads/a.mp4",
			"duration":10,
			"trimStart":0,
			"trimEnd":10,
			"transition":"none",
			"transDur":0.5
		},
		{
			"id":"asset-1",
			"url":"/uploads/b.mp4",
			"duration":8,
			"trimStart":0,
			"trimEnd":8,
			"transition":"none",
			"transDur":0.5
		}
	]`)
	input.ClipCount = 2

	_, _, _, err := normalizeVideoDraftInput(input)
	if !errors.Is(
		err,
		ErrVideoDraftInputInvalid,
	) {
		t.Fatalf(
			"重复ID应返回输入错误: %v",
			err,
		)
	}
}

func TestNormalizeVideoDraftInputRejectsInvalidTrim(
	t *testing.T,
) {
	input := validVideoDraftInputForTest()
	input.ClipsData = json.RawMessage(`[
		{
			"id":"asset-1",
			"url":"/uploads/a.mp4",
			"duration":10,
			"trimStart":8,
			"trimEnd":7,
			"transition":"none",
			"transDur":0.5
		}
	]`)

	_, _, _, err := normalizeVideoDraftInput(input)
	if !errors.Is(
		err,
		ErrVideoDraftInputInvalid,
	) {
		t.Fatalf(
			"非法裁剪区间应返回输入错误: %v",
			err,
		)
	}
}

func TestNormalizeVideoDraftInputRejectsLongName(
	t *testing.T,
) {
	input := validVideoDraftInputForTest()
	input.Name = strings.Repeat(
		"草",
		videoDraftMaxNameRunes+1,
	)

	_, _, _, err := normalizeVideoDraftInput(input)
	if !errors.Is(
		err,
		ErrVideoDraftInputInvalid,
	) {
		t.Fatalf(
			"超长名称应返回输入错误: %v",
			err,
		)
	}
}
