package services

import (
	"testing"
	"time"

	"tedna/internal/models"
)

func subtitleScopeID(
	value string,
) *string {
	return &value
}

func TestNormalizeCoursewareSubtitleScope(
	t *testing.T,
) {
	tests := []struct {
		name      string
		scopeType string
		scopeID   *string
		wantType  string
		wantID    string
		wantError bool
	}{
		{
			name:      "视频资产必须有ID",
			scopeType: models.SubScopeVideoAsset,
			wantError: true,
		},
		{
			name:      "页面必须有ID",
			scopeType: models.SubScopePage,
			wantError: true,
		},
		{
			name:      "视频资产正常",
			scopeType: models.SubScopeVideoAsset,
			scopeID:   subtitleScopeID("asset-1"),
			wantType:  models.SubScopeVideoAsset,
			wantID:    "asset-1",
		},
		{
			name:      "页面正常",
			scopeType: models.SubScopePage,
			scopeID:   subtitleScopeID("page-1"),
			wantType:  models.SubScopePage,
			wantID:    "page-1",
		},
		{
			name:      "编辑器草稿兼容空ID",
			scopeType: models.SubScopeEditorDraft,
			wantType:  models.SubScopeEditorDraft,
		},
		{
			name:      "编辑器草稿带ID",
			scopeType: models.SubScopeEditorDraft,
			scopeID:   subtitleScopeID("draft-1"),
			wantType:  models.SubScopeEditorDraft,
			wantID:    "draft-1",
		},
		{
			name:      "非法类型拒绝",
			scopeType: "unknown",
			wantError: true,
		},
	}

	for _, testCase := range tests {
		t.Run(
			testCase.name,
			func(t *testing.T) {
				gotType,
					gotID,
					err :=
					normalizeCoursewareSubtitleScope(
						testCase.scopeType,
						testCase.scopeID,
					)

				if testCase.wantError {
					if err == nil {
						t.Fatal(
							"期望错误，实际为nil",
						)
					}
					return
				}

				if err != nil {
					t.Fatalf(
						"不期望错误，实际=%v",
						err,
					)
				}

				if gotType !=
					testCase.wantType {
					t.Fatalf(
						"scope_type不一致: got=%s want=%s",
						gotType,
						testCase.wantType,
					)
				}

				gotIDValue := ""
				if gotID != nil {
					gotIDValue = *gotID
				}

				if gotIDValue !=
					testCase.wantID {
					t.Fatalf(
						"scope_id不一致: got=%s want=%s",
						gotIDValue,
						testCase.wantID,
					)
				}
			},
		)
	}
}

func TestCoursewareSubtitleRevisionUnchanged(
	t *testing.T,
) {
	now := time.Now()
	same := now
	later := now.Add(time.Second)

	base := &models.CoursewareSubtitle{
		ID:           "subtitle-1",
		CoursewareID: "courseware-1",
		UpdatedAt:    &now,
	}

	if !coursewareSubtitleRevisionUnchanged(
		base,
		&models.CoursewareSubtitle{
			ID:           "subtitle-1",
			CoursewareID: "courseware-1",
			UpdatedAt:    &same,
		},
	) {
		t.Fatal(
			"相同版本应判定为未变化",
		)
	}

	if coursewareSubtitleRevisionUnchanged(
		base,
		&models.CoursewareSubtitle{
			ID:           "subtitle-1",
			CoursewareID: "courseware-1",
			UpdatedAt:    &later,
		},
	) {
		t.Fatal(
			"更新时间变化应判定为冲突",
		)
	}

	if coursewareSubtitleRevisionUnchanged(
		base,
		&models.CoursewareSubtitle{
			ID:           "subtitle-2",
			CoursewareID: "courseware-1",
			UpdatedAt:    &same,
		},
	) {
		t.Fatal(
			"字幕ID变化应判定为冲突",
		)
	}
}
