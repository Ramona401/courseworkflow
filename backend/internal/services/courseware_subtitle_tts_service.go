package services

// courseware_subtitle_tts_service.go — 作者专属字幕TTS逐段计费生成
//
// 每个目标字幕段独立执行：
//   作者授权与字幕版本复验 → 积分预留 → TTS合成/文件恢复 → 积分结算。
//
// 已经产生供应商费用的音频文件永不因最终字幕JSON写入失败而删除。
// 批量操作中断、网络响应丢失或乐观锁冲突后，使用同一operation_id重试，
// 会从逐段计费记录和确定性文件恢复，不重复调用供应商。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"tedna/internal/ai"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// GenerateTTS 为课件作者批量生成字幕配音。
func (s *CoursewareSubtitleService) GenerateTTS(
	ctx context.Context,
	coursewareID string,
	subtitleID string,
	actor *CoursewareActorContext,
	voice string,
	speed float64,
	segmentIDs []string,
	operationID string,
) (
	*models.GenerateTTSResponse,
	error,
) {
	voice =
		strings.TrimSpace(
			voice,
		)

	operationID =
		strings.TrimSpace(
			operationID,
		)

	if voice == "" {
		return nil,
			fmt.Errorf(
				"%w: voice不能为空",
				ErrCoursewareSubtitleInputInvalid,
			)
	}

	if _, err :=
		uuid.Parse(
			operationID,
		); err != nil {
		return nil,
			fmt.Errorf(
				"%w: operation_id无效",
				ErrCoursewareSubtitleInputInvalid,
			)
	}

	if speed <= 0 {
		speed = 1
	}

	scopedActor,
		subtitle,
		err :=
		loadOwnedCoursewareSubtitleMediaInputs(
			ctx,
			coursewareID,
			subtitleID,
			actor,
		)

	if err != nil {
		return nil, err
	}

	if subtitle.UpdatedAt == nil {
		return nil,
			ErrCoursewareSubtitleMutationConflict
	}

	expectedUpdatedAt :=
		*subtitle.UpdatedAt

	var segments []models.SubtitleSegment

	if err :=
		json.Unmarshal(
			[]byte(
				subtitle.Segments,
			),
			&segments,
		); err != nil {
		return nil,
			fmt.Errorf(
				"解析字幕片段失败: %w",
				err,
			)
	}

	if len(segments) == 0 {
		return nil,
			fmt.Errorf(
				"字幕为空，无法生成配音",
			)
	}

	targetSet :=
		make(
			map[string]bool,
			len(segmentIDs),
		)

	for _, rawID := range segmentIDs {
		id :=
			strings.TrimSpace(
				rawID,
			)

		if id != "" {
			targetSet[id] = true
		}
	}

	ttsConfig, err :=
		ai.GetTTSConfig(
			s.cfg.GetAESKey(),
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"加载TTS配置失败: %w",
				err,
			)
	}

	// v3供应商按实际音色决定Resource ID。
	// 必须先确定真实价格身份，再进行积分预留，禁止使用退役别名调用供应商。
	if ttsConfig.Provider == ai.TTSProviderVolcanoV3 {
		ttsConfig.Model =
			ai.ResolveTTSResourceID(
				voice,
			)
	}

	if strings.TrimSpace(
		ttsConfig.Model,
	) == "" {
		return nil,
			fmt.Errorf(
				"TTS计费模型身份为空",
			)
	}

	outputDir :=
		filepath.Join(
			CWAssetUploadDir,
			coursewareID,
			"tts",
		)

	if err :=
		os.MkdirAll(
			outputDir,
			0755,
		); err != nil {
		return nil,
			fmt.Errorf(
				"创建TTS输出目录失败: %w",
				err,
			)
	}

	userID :=
		scopedActor.UserID

	traceContext :=
		&ai.TraceContext{
			UserID: &userID,
			SchoolID: schoolIDPtr(
				scopedActor.SchoolID,
			),
			SceneCode: "courseware_subtitle_tts",
		}

	successCount := 0
	failCount := 0
	errorMessages :=
		make(
			[]string,
			0,
		)

	var fatalErr error

	for index := range segments {
		segment :=
			&segments[index]

		text :=
			strings.TrimSpace(
				segment.Text,
			)

		if text == "" {
			continue
		}

		segmentID :=
			strings.TrimSpace(
				segment.ID,
			)

		if len(targetSet) > 0 &&
			!targetSet[segmentID] {
			continue
		}

		if segmentID == "" {
			failCount++

			errorMessages =
				append(
					errorMessages,
					fmt.Sprintf(
						"第%d条缺少稳定字幕ID",
						index+1,
					),
				)

			continue
		}

		scopedActor,
			_,
			err =
			reloadOwnedCoursewareSubtitleMediaInputs(
				ctx,
				coursewareID,
				subtitleID,
				scopedActor,
				subtitle,
			)

		if err != nil {
			fatalErr = err
			break
		}

		userID =
			scopedActor.UserID

		traceContext.UserID =
			&userID

		traceContext.SchoolID =
			schoolIDPtr(
				scopedActor.SchoolID,
			)

		result, ttsErr :=
			s.executeBilledCoursewareTTSSegment(
				ctx,
				ttsConfig,
				&coursewareTTSSegmentBillingInput{
					UserID: userID,
					SchoolID: schoolIDPtr(
						scopedActor.SchoolID,
					),
					CoursewareID: coursewareID,
					SubtitleID:   subtitleID,
					SegmentID:    segmentID,
					OperationID:  operationID,
					Text:         text,
					Voice:        voice,
					Speed:        speed,
					ModelName:    ttsConfig.Model,
					OutputDir:    outputDir,
					TraceContext: traceContext,
				},
			)

		if ttsErr != nil {
			if isCoursewareTTSFatalError(
				ttsErr,
			) {
				fatalErr = ttsErr
				break
			}

			failCount++

			errorMessages =
				append(
					errorMessages,
					fmt.Sprintf(
						"第%d条(%s): %s",
						index+1,
						subtitleTruncate(
							text,
							15,
						),
						ttsErr.Error(),
					),
				)

			continue
		}

		if result == nil {
			failCount++

			errorMessages =
				append(
					errorMessages,
					fmt.Sprintf(
						"第%d条: TTS未返回结果",
						index+1,
					),
				)

			continue
		}

		// 每次真实费用形成后再次授权和复验字幕版本。
		scopedActor,
			_,
			err =
			reloadOwnedCoursewareSubtitleMediaInputs(
				ctx,
				coursewareID,
				subtitleID,
				scopedActor,
				subtitle,
			)

		if err != nil {
			fatalErr = err
			break
		}

		segment.TTSAudioURL =
			result.AudioURL
		segment.TTSVoice =
			voice
		segment.TTSDuration =
			result.Duration
		segment.TTSGeneratedAt =
			result.GeneratedAt

		successCount++
	}

	updatedSegments :=
		subtitle.Segments

	if successCount > 0 {
		// 写库前再次确认作者权限和原始字幕版本未变化。
		if _, _, reloadErr :=
			reloadOwnedCoursewareSubtitleMediaInputs(
				ctx,
				coursewareID,
				subtitleID,
				scopedActor,
				subtitle,
			); reloadErr != nil {
			return nil, reloadErr
		}

		updatedJSON, marshalErr :=
			json.Marshal(
				segments,
			)

		if marshalErr != nil {
			return nil,
				fmt.Errorf(
					"序列化更新后的字幕失败: %w",
					marshalErr,
				)
		}

		updated, updateErr :=
			repository.
				UpdateCoursewareSubtitleSegmentsIfUnchanged(
					ctx,
					coursewareID,
					subtitleID,
					expectedUpdatedAt,
					string(
						updatedJSON,
					),
				)

		if updateErr != nil {
			return nil, updateErr
		}

		if !updated {
			// 已结算音频和确定性文件继续保留。
			// 同一operation_id在最新字幕版本上重试即可回写。
			return nil,
				ErrCoursewareSubtitleMutationConflict
		}

		updatedSegments =
			string(
				updatedJSON,
			)
	}

	if fatalErr != nil {
		return nil, fatalErr
	}

	return &models.GenerateTTSResponse{
		SubtitleID:   subtitleID,
		SuccessCount: successCount,
		FailCount:    failCount,
		TotalCount: successCount +
			failCount,
		Segments: updatedSegments,
		Errors:   errorMessages,
		Message: fmt.Sprintf(
			"TTS配音完成：成功%d条，失败%d条",
			successCount,
			failCount,
		),
	}, nil
}

// isCoursewareTTSFatalError 判断是否应中止整批并保留operation_id恢复。
func isCoursewareTTSFatalError(
	err error,
) bool {
	switch {
	case err == nil:
		return false

	case errors.Is(
		err,
		ErrMediaBillingPriceNotConfigured,
	),
		errors.Is(
			err,
			repository.ErrInsufficientBalance,
		),
		errors.Is(
			err,
			repository.ErrTokenAccountNotFound,
		),
		errors.Is(
			err,
			repository.ErrAccountSuspended,
		),
		errors.Is(
			err,
			ErrCoursewareTTSBillingInProgress,
		),
		errors.Is(
			err,
			ErrCoursewareTTSBillingTerminal,
		),
		errors.Is(
			err,
			ErrCoursewareTTSBillingIdentityMismatch,
		),
		errors.Is(
			err,
			ErrCoursewareTTSBillingOutputMissing,
		),
		errors.Is(
			err,
			ErrCoursewareTTSBillingPending,
		):
		return true

	default:
		return false
	}
}
