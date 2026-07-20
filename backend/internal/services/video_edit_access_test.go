package services

import (
	"errors"
	"math"
	"testing"

	"tedna/internal/models"
)

func videoEditTestStringPointer(
	value string,
) *string {
	return &value
}

func TestValidateVideoEditTimeRange(
	t *testing.T,
) {
	tests := []struct {
		name        string
		start       float64
		end         float64
		source      float64
		minDuration float64
		wantError   bool
	}{
		{
			name:        "合法区间",
			start:       1,
			end:         4,
			source:      10,
			minDuration: 1,
		},
		{
			name:        "负起点",
			start:       -1,
			end:         4,
			source:      10,
			minDuration: 1,
			wantError:   true,
		},
		{
			name:        "结束超过源时长",
			start:       1,
			end:         11,
			source:      10,
			minDuration: 1,
			wantError:   true,
		},
		{
			name:        "非有限数值",
			start:       math.NaN(),
			end:         4,
			source:      10,
			minDuration: 1,
			wantError:   true,
		},
	}

	for _, testCase := range tests {
		t.Run(
			testCase.name,
			func(t *testing.T) {
				err := validateVideoEditTimeRange(
					testCase.start,
					testCase.end,
					testCase.source,
					testCase.minDuration,
				)

				if testCase.wantError {
					if !errors.Is(
						err,
						ErrVideoEditInputInvalid,
					) {
						t.Fatalf(
							"期望输入错误，实际=%v",
							err,
						)
					}
					return
				}

				if err != nil {
					t.Fatalf(
						"合法区间不应失败: %v",
						err,
					)
				}
			},
		)
	}
}

func TestNormalizeVideoEditGain(
	t *testing.T,
) {
	gain, err :=
		normalizeVideoEditGain(0)
	if err != nil ||
		gain != 1 {
		t.Fatalf(
			"零值应规范为1: gain=%v err=%v",
			gain,
			err,
		)
	}

	if _, err :=
		normalizeVideoEditGain(
			math.Inf(1),
		); !errors.Is(
		err,
		ErrVideoEditInputInvalid,
	) {
		t.Fatalf(
			"无限增益应拒绝: %v",
			err,
		)
	}
}

func TestVideoEditAssetRevisionEqual(
	t *testing.T,
) {
	pageID := "page-1"

	base := &models.CoursewareAsset{
		ID:           "asset-1",
		CoursewareID: "courseware-1",
		PageID:       &pageID,
		AssetType:    models.CWAssetTypeVideo,
		OssURL:       "/uploads/courseware-assets/courseware-1/videos/a.mp4",
		MimeType:     "video/mp4",
		Status:       models.CWAssetStatusUploaded,
	}

	samePage := "page-1"
	same := *base
	same.PageID = &samePage

	if !videoEditAssetRevisionEqual(
		base,
		&same,
	) {
		t.Fatal(
			"相同正式资产版本应判定为一致",
		)
	}

	changed := same
	changed.OssURL =
		"/uploads/courseware-assets/courseware-1/videos/b.mp4"

	if videoEditAssetRevisionEqual(
		base,
		&changed,
	) {
		t.Fatal(
			"文件路径变化应判定为冲突",
		)
	}

	otherPage := "page-2"
	changedPage := same
	changedPage.PageID = &otherPage

	if videoEditAssetRevisionEqual(
		base,
		&changedPage,
	) {
		t.Fatal(
			"页面归属变化应判定为冲突",
		)
	}
}

func TestNormalizeTransition(
	t *testing.T,
) {
	if got := normalizeTransition(
		" wipeleft ",
	); got != "wipeleft" {
		t.Fatalf(
			"合法转场规范化异常: %q",
			got,
		)
	}

	if got := normalizeTransition(
		"unknown",
	); got != "fade" {
		t.Fatalf(
			"未知转场应降级为fade: %q",
			got,
		)
	}
}

func TestCloneVideoEditPageID(
	t *testing.T,
) {
	if cloneVideoEditPageID(nil) != nil {
		t.Fatal(
			"nil页面应保持nil",
		)
	}

	got := cloneVideoEditPageID(
		videoEditTestStringPointer(
			" page-1 ",
		),
	)
	if got == nil ||
		*got != "page-1" {
		t.Fatalf(
			"页面ID规范化异常: %v",
			got,
		)
	}
}
