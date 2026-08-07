package repository

// courseware_style_confirm_guard_test.go — 风格确认来源纯规则定向测试

import (
	"errors"
	"testing"

	"tedna/internal/models"
)

func TestValidateCoursewareStyleConfirmSelection(
	t *testing.T,
) {
	referenceAssetID :=
		"11111111-1111-1111-1111-111111111111"

	previewAssetID :=
		"22222222-2222-2222-2222-222222222222"

	otherAssetID :=
		"33333333-3333-3333-3333-333333333333"

	testCases := []struct {
		name string

		referenceMode       string
		storedReferenceMode string

		storedReferenceAssetID *string
		confirmedAssetID       string

		generatedPreview bool

		wantError error
	}{
		{
			name:                   "同模式generated预览允许确认",
			referenceMode:          models.CWStyleReferenceModeStyleOnly,
			storedReferenceMode:    models.CWStyleReferenceModeStyleOnly,
			storedReferenceAssetID: &referenceAssetID,
			confirmedAssetID:       previewAssetID,
			generatedPreview:       true,
		},
		{
			name:                   "模式不一致的generated预览拒绝确认",
			referenceMode:          models.CWStyleReferenceModeCharacter,
			storedReferenceMode:    models.CWStyleReferenceModeStyleOnly,
			storedReferenceAssetID: &referenceAssetID,
			confirmedAssetID:       previewAssetID,
			generatedPreview:       true,
			wantError:              ErrCoursewareStylePreviewModeStale,
		},
		{
			name:                   "固定主体同模式允许当前参考图",
			referenceMode:          models.CWStyleReferenceModeCharacter,
			storedReferenceMode:    models.CWStyleReferenceModeCharacter,
			storedReferenceAssetID: &referenceAssetID,
			confirmedAssetID:       referenceAssetID,
		},
		{
			name:                   "临时切到固定主体不能绕过会话模式",
			referenceMode:          models.CWStyleReferenceModeCharacter,
			storedReferenceMode:    models.CWStyleReferenceModeStyleOnly,
			storedReferenceAssetID: &referenceAssetID,
			confirmedAssetID:       referenceAssetID,
			wantError:              ErrCoursewareStylePreviewModeStale,
		},
		{
			name:                   "只提取风格不能直接确认原始参考图",
			referenceMode:          models.CWStyleReferenceModeStyleOnly,
			storedReferenceMode:    models.CWStyleReferenceModeStyleOnly,
			storedReferenceAssetID: &referenceAssetID,
			confirmedAssetID:       referenceAssetID,
			wantError:              ErrCoursewareStyleConfirmAssetInvalid,
		},
		{
			name:                   "抽象灵感不能直接确认原始参考图",
			referenceMode:          models.CWStyleReferenceModeInspiration,
			storedReferenceMode:    models.CWStyleReferenceModeInspiration,
			storedReferenceAssetID: &referenceAssetID,
			confirmedAssetID:       referenceAssetID,
			wantError:              ErrCoursewareStyleConfirmAssetInvalid,
		},
		{
			name:                   "同课件其它图片不能借用为锚点",
			referenceMode:          models.CWStyleReferenceModeCharacter,
			storedReferenceMode:    models.CWStyleReferenceModeCharacter,
			storedReferenceAssetID: &referenceAssetID,
			confirmedAssetID:       otherAssetID,
			wantError:              ErrCoursewareStyleConfirmAssetInvalid,
		},
		{
			name:                   "没有参考图时固定主体不能确认任意图片",
			referenceMode:          models.CWStyleReferenceModeCharacter,
			storedReferenceMode:    models.CWStyleReferenceModeCharacter,
			storedReferenceAssetID: nil,
			confirmedAssetID:       otherAssetID,
			wantError:              ErrCoursewareStyleConfirmAssetInvalid,
		},
	}

	for _, testCase := range testCases {
		t.Run(
			testCase.name,
			func(
				t *testing.T,
			) {
				err :=
					validateCoursewareStyleConfirmSelection(
						testCase.referenceMode,
						testCase.storedReferenceMode,
						testCase.storedReferenceAssetID,
						testCase.confirmedAssetID,
						testCase.generatedPreview,
					)

				if testCase.wantError == nil {
					if err != nil {
						t.Fatalf(
							"预期允许确认，实际返回错误：%v",
							err,
						)
					}

					return
				}

				if !errors.Is(
					err,
					testCase.wantError,
				) {
					t.Fatalf(
						"错误类型不符：got=%v want=%v",
						err,
						testCase.wantError,
					)
				}
			},
		)
	}
}
