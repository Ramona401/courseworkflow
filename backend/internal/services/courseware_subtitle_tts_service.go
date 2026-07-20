package services

// courseware_subtitle_tts_service.go — 作者专属字幕TTS生成
//
// 每次产生外部费用前后均重新执行作者控制授权和字幕版本校验。
// 最终使用updated_at乐观锁写库；失权、冲突或写库失败时清理本轮生成音频。

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

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
) (
	*models.GenerateTTSResponse,
	error,
) {
	voice = strings.TrimSpace(voice)
	if voice == "" {
		return nil,
			fmt.Errorf(
				"%w: voice不能为空",
				ErrCoursewareSubtitleInputInvalid,
			)
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

	expectedUpdatedAt := *subtitle.UpdatedAt

	var segments []models.SubtitleSegment

	if err := json.Unmarshal(
		[]byte(subtitle.Segments),
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
			fmt.Errorf("字幕为空，无法生成配音")
	}

	targetSet := make(
		map[string]bool,
		len(segmentIDs),
	)
	for _, id := range segmentIDs {
		targetSet[id] = true
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

	outputDir := filepath.Join(
		CWAssetUploadDir,
		coursewareID,
		"tts",
	)
	if err := os.MkdirAll(
		outputDir,
		0755,
	); err != nil {
		return nil,
			fmt.Errorf(
				"创建TTS输出目录失败: %w",
				err,
			)
	}

	generatedFiles := make(
		[]string,
		0,
		len(segments),
	)
	keepGeneratedFiles := false

	defer func() {
		if !keepGeneratedFiles {
			cleanupCoursewareSubtitleFiles(
				generatedFiles,
			)
		}
	}()

	userID := scopedActor.UserID

	traceContext := &ai.TraceContext{
		UserID:    &userID,
		SchoolID:  schoolIDPtr(scopedActor.SchoolID),
		SceneCode: "courseware_subtitle_tts",
	}

	successCount := 0
	failCount := 0
	errorMessages := make([]string, 0)

	for index := range segments {
		segment := &segments[index]

		if strings.TrimSpace(segment.Text) == "" {
			continue
		}

		if len(targetSet) > 0 &&
			!targetSet[segment.ID] {
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
			return nil, err
		}

		userID = scopedActor.UserID
		traceContext.UserID = &userID
		traceContext.SchoolID =
			schoolIDPtr(
				scopedActor.SchoolID,
			)

		segmentToken :=
			coursewareSubtitleSafeFileToken(
				segment.ID,
			)

		outputName := fmt.Sprintf(
			"tts_%d_%d_%s",
			time.Now().UnixNano(),
			index+1,
			segmentToken,
		)

		expectedPath := filepath.Join(
			outputDir,
			outputName+".mp3",
		)

		result, ttsErr :=
			ai.SynthesizeSpeech(
				ctx,
				ttsConfig,
				segment.Text,
				voice,
				speed,
				outputDir,
				outputName,
				traceContext,
			)
		if ttsErr != nil {
			_ = os.Remove(expectedPath)

			failCount++
			errorMessages = append(
				errorMessages,
				fmt.Sprintf(
					"第%d条(%s): %s",
					index+1,
					subtitleTruncate(
						segment.Text,
						15,
					),
					ttsErr.Error(),
				),
			)
			continue
		}

		if result == nil {
			_ = os.Remove(expectedPath)

			failCount++
			errorMessages = append(
				errorMessages,
				fmt.Sprintf(
					"第%d条: TTS未返回结果",
					index+1,
				),
			)
			continue
		}

		resultPath := strings.TrimSpace(
			result.AudioFilePath,
		)
		if resultPath == "" {
			resultPath = expectedPath
		}

		generatedFiles = append(
			generatedFiles,
			resultPath,
		)

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
			return nil, err
		}

		segment.TTSAudioURL =
			CWAssetURLPrefix +
				filepath.Join(
					coursewareID,
					"tts",
					outputName+".mp3",
				)
		segment.TTSVoice = voice
		segment.TTSDuration = result.Duration
		segment.TTSGeneratedAt =
			time.Now().Format(time.RFC3339)

		successCount++
	}

	if successCount == 0 {
		keepGeneratedFiles = true

		return &models.GenerateTTSResponse{
			SubtitleID:   subtitleID,
			SuccessCount: 0,
			FailCount:    failCount,
			TotalCount:   failCount,
			Segments:     subtitle.Segments,
			Errors:       errorMessages,
			Message: fmt.Sprintf(
				"TTS配音完成：成功0条，失败%d条",
				failCount,
			),
		}, nil
	}

	if _, _, err =
		reloadOwnedCoursewareSubtitleMediaInputs(
			ctx,
			coursewareID,
			subtitleID,
			scopedActor,
			subtitle,
		); err != nil {
		return nil, err
	}

	updatedJSON, err := json.Marshal(segments)
	if err != nil {
		return nil,
			fmt.Errorf(
				"序列化更新后的字幕失败: %w",
				err,
			)
	}

	updated, err :=
		repository.
			UpdateCoursewareSubtitleSegmentsIfUnchanged(
				ctx,
				coursewareID,
				subtitleID,
				expectedUpdatedAt,
				string(updatedJSON),
			)
	if err != nil {
		return nil, err
	}
	if !updated {
		return nil,
			ErrCoursewareSubtitleMutationConflict
	}

	keepGeneratedFiles = true

	return &models.GenerateTTSResponse{
		SubtitleID:   subtitleID,
		SuccessCount: successCount,
		FailCount:    failCount,
		TotalCount: successCount +
			failCount,
		Segments: string(updatedJSON),
		Errors:   errorMessages,
		Message: fmt.Sprintf(
			"TTS配音完成：成功%d条，失败%d条",
			successCount,
			failCount,
		),
	}, nil
}
