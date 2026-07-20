package services

import (
	"errors"
	"strings"
	"testing"

	"tedna/internal/models"
)

func newPageVersionPolicyCourseware() *models.Courseware {
	return &models.Courseware{
		ID:              "cw-page-version-policy",
		UserID:          "owner-1",
		EducationDomain: models.EducationDomainK12,
		Status:          models.CoursewareStatusPreview,
		PublishState:    models.CWPublishPrivate,
	}
}

func TestValidateCoursewarePageVersionPath(
	t *testing.T,
) {
	page := &models.CoursewarePage{
		ID:           "page-1",
		CoursewareID: "cw-1",
	}
	version := &models.CoursewarePageVersion{
		ID:           "version-1",
		PageID:       "page-1",
		CoursewareID: "cw-1",
	}

	if err := validateCoursewarePageVersionPath(
		"cw-1",
		page,
		version,
	); err != nil {
		t.Fatalf(
			"合法三层归属不应报错：%v",
			err,
		)
	}

	tests := []struct {
		name       string
		courseware string
		page       *models.CoursewarePage
		version    *models.CoursewarePageVersion
	}{
		{
			name:       "页面不属于路径课件",
			courseware: "cw-2",
			page:       page,
			version:    version,
		},
		{
			name:       "版本不属于路径页面",
			courseware: "cw-1",
			page:       page,
			version: &models.CoursewarePageVersion{
				ID:           "version-2",
				PageID:       "page-2",
				CoursewareID: "cw-1",
			},
		},
		{
			name:       "版本不属于路径课件",
			courseware: "cw-1",
			page:       page,
			version: &models.CoursewarePageVersion{
				ID:           "version-3",
				PageID:       "page-1",
				CoursewareID: "cw-2",
			},
		},
		{
			name:       "空页面拒绝",
			courseware: "cw-1",
			page:       nil,
			version:    version,
		},
		{
			name:       "空版本拒绝",
			courseware: "cw-1",
			page:       page,
			version:    nil,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			err := validateCoursewarePageVersionPath(
				testCase.courseware,
				testCase.page,
				testCase.version,
			)

			if !errors.Is(
				err,
				ErrCoursewarePageVersionNotFound,
			) {
				t.Fatalf(
					"期望错误%v，实际%v",
					ErrCoursewarePageVersionNotFound,
					err,
				)
			}
		})
	}
}

func TestValidateCoursewarePageMutationState(
	t *testing.T,
) {
	t.Run("预览私有态允许覆盖", func(t *testing.T) {
		courseware :=
			newPageVersionPolicyCourseware()

		if err := validateCoursewarePageMutationState(
			courseware,
		); err != nil {
			t.Fatalf(
				"预览私有态不应报错：%v",
				err,
			)
		}
	})

	t.Run("generating状态拒绝", func(t *testing.T) {
		courseware :=
			newPageVersionPolicyCourseware()
		courseware.Status =
			models.CoursewareStatusGenerating

		err := validateCoursewarePageMutationState(
			courseware,
		)
		if !errors.Is(
			err,
			ErrCoursewarePageMutationConflict,
		) {
			t.Fatalf(
				"期望状态冲突，实际%v",
				err,
			)
		}
	})

	t.Run("in_pipeline状态拒绝", func(t *testing.T) {
		courseware :=
			newPageVersionPolicyCourseware()
		courseware.Status =
			models.CoursewareStatusInPipeline

		err := validateCoursewarePageMutationState(
			courseware,
		)
		if !errors.Is(
			err,
			ErrCoursewarePageMutationConflict,
		) {
			t.Fatalf(
				"期望状态冲突，实际%v",
				err,
			)
		}
	})

	t.Run("submitted发布态拒绝", func(t *testing.T) {
		courseware :=
			newPageVersionPolicyCourseware()
		courseware.PublishState =
			models.CWPublishSubmitted

		err := validateCoursewarePageMutationState(
			courseware,
		)
		if !errors.Is(
			err,
			ErrCoursewarePageMutationConflict,
		) {
			t.Fatalf(
				"期望状态冲突，实际%v",
				err,
			)
		}
	})

	t.Run("批量生成运行锁拒绝", func(t *testing.T) {
		courseware :=
			newPageVersionPolicyCourseware()
		courseware.ID =
			"cw-page-version-gen-lock"

		cwGenRunning.Store(
			courseware.ID,
			struct{}{},
		)
		defer cwGenRunning.Delete(
			courseware.ID,
		)

		err := validateCoursewarePageMutationState(
			courseware,
		)
		if !errors.Is(
			err,
			ErrCoursewarePageMutationConflict,
		) {
			t.Fatalf(
				"期望生成锁冲突，实际%v",
				err,
			)
		}
	})

	t.Run("自动装配运行锁拒绝", func(t *testing.T) {
		courseware :=
			newPageVersionPolicyCourseware()
		courseware.ID =
			"cw-page-version-assembly-lock"

		cwAssemblyRunning.Store(
			courseware.ID,
			struct{}{},
		)
		defer cwAssemblyRunning.Delete(
			courseware.ID,
		)

		err := validateCoursewarePageMutationState(
			courseware,
		)
		if !errors.Is(
			err,
			ErrCoursewarePageMutationConflict,
		) {
			t.Fatalf(
				"期望装配锁冲突，实际%v",
				err,
			)
		}
	})
}

func TestValidateCoursewarePageHTMLPayload(
	t *testing.T,
) {
	if err := validateCoursewarePageHTMLPayload(
		`<div class="cw-page">正常页面</div>`,
	); err != nil {
		t.Fatalf(
			"正常HTML不应报错：%v",
			err,
		)
	}

	for _, invalid := range []string{
		"",
		"   ",
	} {
		err := validateCoursewarePageHTMLPayload(
			invalid,
		)
		if !errors.Is(
			err,
			ErrCoursewarePageHTMLInvalid,
		) {
			t.Fatalf(
				"空HTML应返回非法错误，实际%v",
				err,
			)
		}
	}

	tooLarge := strings.Repeat(
		"x",
		CoursewarePageHTMLMaxBytes+1,
	)
	err := validateCoursewarePageHTMLPayload(
		tooLarge,
	)
	if !errors.Is(
		err,
		ErrCoursewarePageHTMLInvalid,
	) {
		t.Fatalf(
			"超限HTML应返回非法错误，实际%v",
			err,
		)
	}
}

func TestResolveCoursewarePageVersionRestoreMetadata(
	t *testing.T,
) {
	currentPage := &models.CoursewarePage{
		ID:                  "page-1",
		CoursewareID:        "cw-1",
		PlaceholderMap:      `{"current":true}`,
		MatchedComponentIDs: `["component-current"]`,
		Status:              models.CWPageStatusConfirmed,
	}

	t.Run("旧版本只恢复HTML并保留当前元数据", func(t *testing.T) {
		version := &models.CoursewarePageVersion{
			ID:                       "version-legacy",
			MetadataSnapshotComplete: false,
		}

		placeholderMap,
			matchedIDs,
			status,
			restored,
			err :=
			resolveCoursewarePageVersionRestoreMetadata(
				currentPage,
				version,
			)

		if err != nil {
			t.Fatalf("不期望错误：%v", err)
		}
		if restored {
			t.Fatal("旧版本不应声称恢复了历史元数据")
		}
		if placeholderMap != currentPage.PlaceholderMap ||
			matchedIDs != currentPage.MatchedComponentIDs ||
			status != currentPage.Status {
			t.Fatal("旧版本应保留当前页面元数据")
		}
	})

	t.Run("新完整版本恢复历史元数据", func(t *testing.T) {
		version := &models.CoursewarePageVersion{
			ID:                       "version-complete",
			PlaceholderMap:           `{"old":true}`,
			MatchedComponentIDs:      `["component-old"]`,
			PageStatus:               models.CWPageStatusGenerated,
			MetadataSnapshotComplete: true,
		}

		placeholderMap,
			matchedIDs,
			status,
			restored,
			err :=
			resolveCoursewarePageVersionRestoreMetadata(
				currentPage,
				version,
			)

		if err != nil {
			t.Fatalf("不期望错误：%v", err)
		}
		if !restored {
			t.Fatal("完整版本应恢复历史元数据")
		}
		if placeholderMap != version.PlaceholderMap ||
			matchedIDs != version.MatchedComponentIDs ||
			status != version.PageStatus {
			t.Fatal("完整版本恢复结果与快照不一致")
		}
	})

	t.Run("完整版本非法状态拒绝", func(t *testing.T) {
		version := &models.CoursewarePageVersion{
			ID:                       "version-invalid",
			PageStatus:               "invalid",
			MetadataSnapshotComplete: true,
		}

		_, _, _, _, err :=
			resolveCoursewarePageVersionRestoreMetadata(
				currentPage,
				version,
			)

		if !errors.Is(
			err,
			ErrCoursewarePageVersionSnapshotFailed,
		) {
			t.Fatalf(
				"期望快照错误，实际%v",
				err,
			)
		}
	})
}
