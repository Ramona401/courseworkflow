package services

import (
	"testing"

	"tedna/internal/models"
)

func TestValidateVideoSourceFrameAsset(
	t *testing.T,
) {
	t.Parallel()

	validImage := &models.CoursewareAsset{
		ID:           "source-image-1",
		CoursewareID: "courseware-1",
		AssetType:    models.CWAssetTypeImage,
	}

	tests := []struct {
		name         string
		coursewareID string
		asset        *models.CoursewareAsset
		wantError    bool
	}{
		{
			name:         "same courseware image allowed",
			coursewareID: "courseware-1",
			asset:        validImage,
			wantError:    false,
		},
		{
			name:         "nil asset rejected",
			coursewareID: "courseware-1",
			asset:        nil,
			wantError:    true,
		},
		{
			name:         "cross courseware image rejected",
			coursewareID: "courseware-2",
			asset:        validImage,
			wantError:    true,
		},
		{
			name:         "video asset rejected",
			coursewareID: "courseware-1",
			asset: &models.CoursewareAsset{
				ID:           "source-video-1",
				CoursewareID: "courseware-1",
				AssetType:    models.CWAssetTypeVideo,
			},
			wantError: true,
		},
		{
			name:         "audio asset rejected",
			coursewareID: "courseware-1",
			asset: &models.CoursewareAsset{
				ID:           "source-audio-1",
				CoursewareID: "courseware-1",
				AssetType:    models.CWAssetTypeAudio,
			},
			wantError: true,
		},
	}

	for _, testCase := range tests {
		testCase := testCase

		t.Run(
			testCase.name,
			func(t *testing.T) {
				t.Parallel()

				err :=
					validateVideoSourceFrameAsset(
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
