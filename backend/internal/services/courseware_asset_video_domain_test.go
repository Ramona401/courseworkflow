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

func TestValidateCoursewareVideoBillingIdentity(
	t *testing.T,
) {
	t.Parallel()

	if coursewareVideoBillingVariant != "silent" {
		t.Fatalf(
			"expected silent billing variant, got %s",
			coursewareVideoBillingVariant,
		)
	}

	identity, err :=
		newCoursewareVideoBillingIdentity(
			"user-1",
			"school-1",
			"courseware-1",
			"page-1",
			1,
			"doubao-seedance-1-5-pro-251215",
			"11111111-1111-4111-8111-111111111111",
			"生成一个无声教学视频",
			"",
			"",
		)

	if err != nil {
		t.Fatalf(
			"create billing identity failed: %v",
			err,
		)
	}

	metadata :=
		coursewareVideoMetadataString(
			coursewareVideoBillingMetadata(
				identity,
				"task-1",
			),
		)

	valid :=
		&models.TokenMediaBilling{
			UserID:          identity.UserID,
			BillingNodeCode: coursewareVideoBillingNodeCode,
			MediaType:       models.MediaTypeVideo,
			Provider:        coursewareVideoBillingProvider,
			ModelName:       identity.ModelName,
			Variant:         coursewareVideoBillingVariant,
			MediaUnit:       models.MediaUnitProviderToken,
			CoursewareID: coursewareVideoStringPointer(
				identity.CoursewareID,
			),
			PageID: coursewareVideoStringPointer(
				identity.PageID,
			),
			Metadata: []byte(
				metadata,
			),
		}

	if err :=
		validateCoursewareVideoBillingIdentity(
			valid,
			identity,
		); err != nil {
		t.Fatalf(
			"valid billing identity rejected: %v",
			err,
		)
	}

	tests := []struct {
		name   string
		mutate func(*models.TokenMediaBilling)
	}{
		{
			name: "provider mismatch rejected",
			mutate: func(
				billing *models.TokenMediaBilling,
			) {
				billing.Provider = "other"
			},
		},
		{
			name: "model mismatch rejected",
			mutate: func(
				billing *models.TokenMediaBilling,
			) {
				billing.ModelName =
					"other-model"
			},
		},
		{
			name: "variant mismatch rejected",
			mutate: func(
				billing *models.TokenMediaBilling,
			) {
				billing.Variant = "audio"
			},
		},
		{
			name: "unit mismatch rejected",
			mutate: func(
				billing *models.TokenMediaBilling,
			) {
				billing.MediaUnit =
					models.MediaUnitSecond
			},
		},
	}

	for _, testCase := range tests {
		testCase := testCase

		t.Run(
			testCase.name,
			func(t *testing.T) {
				t.Parallel()

				candidate := *valid
				testCase.mutate(
					&candidate,
				)

				if err :=
					validateCoursewareVideoBillingIdentity(
						&candidate,
						identity,
					); err == nil {
					t.Fatal(
						"expected identity mismatch",
					)
				}
			},
		)
	}
}
