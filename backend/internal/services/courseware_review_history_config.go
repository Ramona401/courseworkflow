package services

// courseware_review_history_config.go
//
// R-03历史详情中的R-02不可变审核配置读取。
//
// 只允许读取正式feedback关联的不可变AI session。
// 禁止读取当前审核配置、当前教案状态或浏览器提交值补齐历史。
//
// “是否实际使用教案资料”与“配置是否允许使用教案资料”严格分离：
//   - cwAIReviewUsesLessonMaterials只表示配置允许；
//   - 本文件从不可变context_manifest_json读取实际available/used/included事实。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

type cwReviewHistoryMaterialManifest struct {
	LessonMaterialUsage struct {
		LessonContentIncluded   *bool `json:"lesson_content_included"`
		CourseOutlineIncluded   *bool `json:"course_outline_included"`
		AlignmentReportIncluded *bool `json:"alignment_report_included"`
	} `json:"lesson_material_usage"`

	LessonPlan struct {
		Available *bool `json:"available"`
		Used      *bool `json:"used"`
	} `json:"lesson_plan"`

	CourseOutline struct {
		Available *bool `json:"available"`
		Used      *bool `json:"used"`
	} `json:"course_outline"`

	AlignmentReport struct {
		Available *bool `json:"available"`
		Used      *bool `json:"used"`
	} `json:"alignment_report"`
}

func emptyCWReviewHistoryConfig(
	reason string,
) models.CoursewareReviewHistoryConfig {
	return models.CoursewareReviewHistoryConfig{
		Available:         false,
		Dimensions:        []string{},
		UnavailableReason: strings.TrimSpace(reason),
	}
}

func buildCWReviewHistoryConfig(
	ctx context.Context,
	review *models.CoursewareReview,
	feedback *models.CoursewareReviewFeedback,
) (
	models.CoursewareReviewHistoryConfig,
	error,
) {
	if review == nil ||
		feedback == nil {
		return models.CoursewareReviewHistoryConfig{},
			errors.New(
				"课件审核历史配置关系不完整",
			)
	}

	if feedback.AIReviewSessionID == nil ||
		strings.TrimSpace(
			*feedback.AIReviewSessionID,
		) == "" {
		return emptyCWReviewHistoryConfig(
			models.CWReviewHistoryConfigUnavailableNoAI,
		), nil
	}

	sessionID :=
		strings.TrimSpace(
			*feedback.AIReviewSessionID,
		)

	session, err :=
		repository.GetCoursewareAIReviewSessionByID(
			ctx,
			sessionID,
		)
	if err != nil {
		return models.CoursewareReviewHistoryConfig{},
			fmt.Errorf(
				"读取历史AI审核会话失败: %w",
				err,
			)
	}

	if session == nil {
		return emptyCWReviewHistoryConfig(
			models.CWReviewHistoryConfigUnavailableSession,
		), nil
	}

	if strings.TrimSpace(session.ID) != sessionID ||
		strings.TrimSpace(session.CoursewareID) !=
			strings.TrimSpace(review.CoursewareID) ||
		strings.TrimSpace(session.ReviewerID) !=
			strings.TrimSpace(review.ReviewerID) ||
		session.ReviewLevel != review.ReviewLevel ||
		session.Status != models.CWAIReviewStatusDone {
		return models.CoursewareReviewHistoryConfig{},
			errors.New(
				"历史AI审核会话与正式审核记录关系异常",
			)
	}

	config, configErr :=
		cwAIReviewConfigFromSession(
			session,
		)
	if configErr != nil {
		// R-02上线前无法按当前不可变协议恢复的会话，
		// 明确标记不可用，绝不从当前配置补齐。
		return emptyCWReviewHistoryConfig(
			models.CWReviewHistoryConfigUnavailableLegacy,
		), nil
	}

	return models.CoursewareReviewHistoryConfig{
		Available:     true,
		SchemaVersion: config.SchemaVersion,
		Dimensions: append(
			[]string{},
			config.ReviewDimensions...,
		),
		CustomFocus: config.CustomDimensionDescription,

		LessonReferenceMode: config.LessonReferenceMode,

		LessonMaterialsUsed: cwReviewHistoryActualLessonMaterialUsage(
			session,
			config,
		),
	}, nil
}

// cwReviewHistoryActualLessonMaterialUsage 返回本次审核是否实际带入教案类材料。
//
// 不能把“允许使用”直接当作“实际使用”。
func cwReviewHistoryActualLessonMaterialUsage(
	session *models.CoursewareAIReviewSession,
	config *CWAIReviewConfigSnapshot,
) *bool {
	if config == nil {
		return nil
	}

	if config.LessonReferenceMode ==
		models.CWAIReviewLessonReferenceNoLesson {
		value := false
		return &value
	}

	if session == nil ||
		strings.TrimSpace(
			session.ContextManifestJSON,
		) == "" {
		return nil
	}

	var manifest cwReviewHistoryMaterialManifest

	if err := json.Unmarshal(
		[]byte(
			session.ContextManifestJSON,
		),
		&manifest,
	); err != nil {
		return nil
	}

	hasFact := false
	used := false

	includedFacts :=
		[]*bool{
			manifest.LessonMaterialUsage.
				LessonContentIncluded,
			manifest.LessonMaterialUsage.
				CourseOutlineIncluded,
			manifest.LessonMaterialUsage.
				AlignmentReportIncluded,
		}

	for _, value := range includedFacts {
		if value == nil {
			continue
		}

		hasFact = true

		if *value {
			used = true
		}
	}

	materialFacts :=
		[][2]*bool{
			{
				manifest.LessonPlan.Available,
				manifest.LessonPlan.Used,
			},
			{
				manifest.CourseOutline.Available,
				manifest.CourseOutline.Used,
			},
			{
				manifest.AlignmentReport.Available,
				manifest.AlignmentReport.Used,
			},
		}

	for _, pair := range materialFacts {
		available := pair[0]
		usedFlag := pair[1]

		if available != nil ||
			usedFlag != nil {
			hasFact = true
		}

		if available != nil &&
			usedFlag != nil &&
			*available &&
			*usedFlag {
			used = true
		}
	}

	if !hasFact {
		return nil
	}

	return &used
}
