package services

import (
	"testing"

	"tedna/internal/models"
)

func TestValidateCoursewareAssetForCourseware(
	t *testing.T,
) {
	t.Parallel()

	asset := &models.CoursewareAsset{
		ID:           "asset-1",
		CoursewareID: "courseware-1",
		AssetType:    models.CWAssetTypeAudio,
	}

	tests := []struct {
		name         string
		coursewareID string
		asset        *models.CoursewareAsset
		wantError    bool
	}{
		{
			name:         "same courseware allowed",
			coursewareID: "courseware-1",
			asset:        asset,
			wantError:    false,
		},
		{
			name:         "nil asset rejected",
			coursewareID: "courseware-1",
			asset:        nil,
			wantError:    true,
		},
		{
			name:         "cross courseware rejected",
			coursewareID: "courseware-2",
			asset:        asset,
			wantError:    true,
		},
	}

	for _, testCase := range tests {
		testCase := testCase

		t.Run(
			testCase.name,
			func(t *testing.T) {
				t.Parallel()

				err :=
					validateCoursewareAssetForCourseware(
						testCase.coursewareID,
						testCase.asset,
					)

				if testCase.wantError &&
					err == nil {
					t.Fatal(
						"expected validation error",
					)
				}

				if !testCase.wantError &&
					err != nil {
					t.Fatalf(
						"unexpected error: %v",
						err,
					)
				}
			},
		)
	}
}
