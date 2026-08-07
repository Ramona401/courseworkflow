package services

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"tedna/internal/models"
)

// newTestCWAIReviewSession 创建不访问数据库的R-02测试会话。
func newTestCWAIReviewSession(
	t *testing.T,
	dimensions []string,
	mode string,
) *models.CoursewareAIReviewSession {
	t.Helper()

	encoded, err := json.Marshal(dimensions)
	if err != nil {
		t.Fatalf("序列化测试维度失败: %v", err)
	}

	customDescription := ""
	for _, dimension := range dimensions {
		if dimension == models.CWAIReviewDimensionCustom {
			customDescription = "重点检查本校实验器材适配"
			break
		}
	}

	return &models.CoursewareAIReviewSession{
		ID:           "session-test",
		CoursewareID: "courseware-test",
		ReviewerID:   "reviewer-test",

		ReviewLevel:     models.ReviewLevelL1,
		EducationDomain: "k12",
		Subject:         "物理",
		Grade:           "八年级",

		ReviewConfigSchemaVersion:  models.CWAIReviewConfigSchemaVersion,
		ReviewDimensionsJSON:       string(encoded),
		CustomDimensionDescription: customDescription,
		LessonReferenceMode:        mode,
		ReviewConfigHash: strings.Repeat(
			"a",
			64,
		),

		SystemPromptSnapshot: "平台审核系统规则",
		BaselineJSON:         "{}",
		ContextManifestJSON:  "{}",
		ContinuityLedgerJSON: "{}",
	}
}

func TestCWAIReviewConfigDefaultCompatibility(
	t *testing.T,
) {
	config, err :=
		NormalizeCWAIReviewConfig(nil)
	if err != nil {
		t.Fatalf("兼容默认配置不应失败: %v", err)
	}

	expected :=
		models.CoursewareAIReviewDefaultDimensions()

	if len(config.ReviewDimensions) !=
		len(expected) {
		t.Fatalf(
			"默认维度数量错误: got=%d want=%d",
			len(config.ReviewDimensions),
			len(expected),
		)
	}

	for index := range expected {
		if config.ReviewDimensions[index] !=
			expected[index] {
			t.Fatalf(
				"默认维度顺序错误: got=%v want=%v",
				config.ReviewDimensions,
				expected,
			)
		}
	}

	if config.LessonReferenceMode !=
		models.CWAIReviewLessonReferenceCurrentCompatible {
		t.Fatalf(
			"默认教案模式错误: %s",
			config.LessonReferenceMode,
		)
	}
}

func TestCWAIReviewConfigRejectsInvalidIntent(
	t *testing.T,
) {
	custom := models.CWAIReviewDimensionCustom
	customDimensions := []string{custom}
	emptyDimensions := []string{}
	duplicateDimensions := []string{
		models.CWAIReviewDimensionTeachingLogic,
		models.CWAIReviewDimensionTeachingLogic,
	}
	unknownDimensions := []string{
		"unknown_dimension",
	}
	description := "额外要求"
	emptyMode := ""

	testCases := []struct {
		name  string
		input *CWAIReviewConfigInput
	}{
		{
			name: "empty dimensions",
			input: &CWAIReviewConfigInput{
				ReviewDimensions: &emptyDimensions,
			},
		},
		{
			name: "duplicate dimensions",
			input: &CWAIReviewConfigInput{
				ReviewDimensions: &duplicateDimensions,
			},
		},
		{
			name: "unknown dimensions",
			input: &CWAIReviewConfigInput{
				ReviewDimensions: &unknownDimensions,
			},
		},
		{
			name: "custom without description",
			input: &CWAIReviewConfigInput{
				ReviewDimensions: &customDimensions,
			},
		},
		{
			name: "description without custom",
			input: &CWAIReviewConfigInput{
				CustomDimensionDescription: &description,
			},
		},
		{
			name: "empty lesson mode",
			input: &CWAIReviewConfigInput{
				LessonReferenceMode: &emptyMode,
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(
			testCase.name,
			func(t *testing.T) {
				_, err :=
					NormalizeCWAIReviewConfig(
						testCase.input,
					)

				if !errors.Is(
					err,
					ErrCWAIReviewConfigInvalid,
				) {
					t.Fatalf(
						"应返回配置错误，实际为: %v",
						err,
					)
				}
			},
		)
	}
}

func TestCWAIReviewConfigNormalizesFixedOrder(
	t *testing.T,
) {
	dimensions := []string{
		models.CWAIReviewDimensionCustom,
		models.CWAIReviewDimensionTeachingLogic,
		models.CWAIReviewDimensionPageReadability,
	}

	description := "重点检查本校实验器材适配"
	mode :=
		models.CWAIReviewLessonReferenceNoLesson

	config, err :=
		NormalizeCWAIReviewConfig(
			&CWAIReviewConfigInput{
				ReviewDimensions:           &dimensions,
				CustomDimensionDescription: &description,
				LessonReferenceMode:        &mode,
			},
		)
	if err != nil {
		t.Fatalf("规范化配置失败: %v", err)
	}

	expected := []string{
		models.CWAIReviewDimensionTeachingLogic,
		models.CWAIReviewDimensionPageReadability,
		models.CWAIReviewDimensionCustom,
	}

	for index := range expected {
		if config.ReviewDimensions[index] !=
			expected[index] {
			t.Fatalf(
				"固定顺序错误: got=%v want=%v",
				config.ReviewDimensions,
				expected,
			)
		}
	}

	if config.CustomDimensionDescription !=
		description {
		t.Fatalf(
			"自定义说明错误: %q",
			config.CustomDimensionDescription,
		)
	}

	if config.LessonReferenceMode != mode {
		t.Fatalf(
			"教案模式错误: %s",
			config.LessonReferenceMode,
		)
	}
}
