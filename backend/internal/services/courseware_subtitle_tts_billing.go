package services

// courseware_subtitle_tts_billing.go — 字幕TTS逐段积分计费与文件恢复
//
// 每个字幕分段是一次独立供应商调用：
//   1. 供应商调用前按真正提交的Unicode字符数预留积分；
//   2. 明确合成失败时释放该段预留；
//   3. 合成成功后按同一字符数结算；
//   4. 音频文件使用operation_id、subtitle_id和segment_id生成确定性名称；
//   5. 供应商成功但结算或字幕JSON回写失败时保留音频文件；
//   6. 同一幂等键重试直接恢复原音频，不重复调用供应商；
//   7. 同一operation_id不得改换文本、音色、语速、字幕段或模型。

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"tedna/internal/ai"
	"tedna/internal/models"
)

const (
	coursewareTTSBillingNodeCode = "courseware_tts_segment"
	coursewareTTSBillingProvider = "volcengine"
	coursewareTTSBillingVariant  = "default"
	coursewareTTSBillingTimeout  = 8 * time.Second
)

var (
	ErrCoursewareTTSBillingInProgress = errors.New(
		"字幕配音任务正在处理中",
	)
	ErrCoursewareTTSBillingTerminal = errors.New(
		"字幕配音计费任务已经终态",
	)
	ErrCoursewareTTSBillingIdentityMismatch = errors.New(
		"字幕配音计费身份不匹配",
	)
	ErrCoursewareTTSBillingOutputMissing = errors.New(
		"字幕配音已经结算但音频文件不存在",
	)
	ErrCoursewareTTSBillingPending = errors.New(
		"字幕配音计费状态等待恢复",
	)
)

// coursewareTTSSegmentBillingInput 保存一个字幕段的不可变业务身份。
type coursewareTTSSegmentBillingInput struct {
        UserID        string
        SchoolID      *string
        CoursewareID  string
        SubtitleID    string
        SegmentID     string
        OperationID   string
        Text          string
        Voice         string
        Speed         float64
        ModelName     string
        OutputDir     string
        CharacterCount int
        TraceContext  *ai.TraceContext
}

// coursewareTTSSegmentBillingResult 是可写回字幕JSON的稳定结果。
type coursewareTTSSegmentBillingResult struct {
	AudioFilePath string
	AudioURL      string
	Duration      float64
	ModelUsed     string
	FileSize      int64
	CharacterCount int
	GeneratedAt   string
}

// executeBilledCoursewareTTSSegment 执行一个字幕段的预留、合成、结算或恢复。
func (s *CoursewareSubtitleService) executeBilledCoursewareTTSSegment(
	ctx context.Context,
	cfg *ai.TTSConfig,
	input *coursewareTTSSegmentBillingInput,
) (*coursewareTTSSegmentBillingResult, error) {
	normalized, err :=
		normalizeCoursewareTTSSegmentBillingInput(
			input,
		)
	if err != nil {
		return nil, err
	}

	if cfg == nil {
		return nil,
			ErrMediaBillingInvalidRequest
	}

	idempotencyKey :=
		coursewareTTSSegmentIdempotencyKey(
			normalized.OperationID,
			normalized.SubtitleID,
			normalized.SegmentID,
		)

	requestFingerprint :=
		coursewareTTSSegmentFingerprint(
			normalized,
		)

	outputName :=
		coursewareTTSSegmentOutputName(
			normalized.OperationID,
			normalized.SubtitleID,
			normalized.SegmentID,
		)

	outputPath :=
		filepath.Join(
			normalized.OutputDir,
			outputName+".mp3",
		)

	audioURL :=
		CWAssetURLPrefix +
			filepath.ToSlash(
				filepath.Join(
					normalized.CoursewareID,
					"tts",
					outputName+".mp3",
				),
			)

	billingService :=
		NewMediaBillingService()

	billing, err :=
		billingService.Reserve(
			ctx,
			&models.MediaBillingReserveRequest{
				UserID:          normalized.UserID,
				SchoolID:        normalized.SchoolID,
				BillingCategory: models.BillingCategoryTTS,
				BillingNodeCode:
					coursewareTTSBillingNodeCode,
				MediaType:
					models.MediaTypeTTS,
				Provider:
					coursewareTTSBillingProvider,
				ModelName:
					normalized.ModelName,
				Variant:
					coursewareTTSBillingVariant,
				MediaUnit:
					models.MediaUnitCharacter,
				EstimatedQuantity:
					float64(
						normalized.CharacterCount,
					),
				IdempotencyKey:
					idempotencyKey,
				CoursewareID:
					coursewareTTSStringPointer(
						normalized.CoursewareID,
					),
				Metadata:
					map[string]interface{}{
						"subtitle_id":
							normalized.SubtitleID,
						"segment_id":
							normalized.SegmentID,
						"operation_id":
							normalized.OperationID,
						"request_fingerprint":
							requestFingerprint,
						"voice":
							normalized.Voice,
						"speed":
							normalized.Speed,
						"text_character_count":
							normalized.CharacterCount,
						"output_name":
							outputName,
					},
			},
		)
	if err != nil {
		return nil, err
	}

	if billing == nil {
		return nil,
			ErrMediaBillingInvalidRequest
	}

	if !billing.ReservationCreated {
		return recoverBilledCoursewareTTSSegment(
			billing,
			normalized,
			requestFingerprint,
			outputPath,
			audioURL,
		)
	}

	// 极少数情况下，计费记录重建前已经留下同名完整文件。
	// 确定性文件存在时优先按真实字符数结算，不再调用供应商。
	if recovered, ok :=
		loadCoursewareTTSSegmentFileResult(
			outputPath,
			audioURL,
			normalized.ModelName,
			normalized.CharacterCount,
		); ok {
		if err :=
			settleCoursewareTTSSegmentBilling(
				billing,
				recovered,
				requestFingerprint,
			); err != nil {
			return nil,
				fmt.Errorf(
					"%w: %v",
					ErrCoursewareTTSBillingPending,
					err,
				)
		}

		return recovered, nil
	}

	result, synthErr :=
		ai.SynthesizeSpeech(
			ctx,
			cfg,
			normalized.Text,
			normalized.Voice,
			normalized.Speed,
			normalized.OutputDir,
			outputName,
			normalized.TraceContext,
		)
        if synthErr != nil {
                _ = os.Remove(outputPath)

                switch {
                case ai.DidTTSSynthesisProviderSucceed(
                        synthErr,
                ):
                        // 供应商已经明确完成合成，本地文件失败不能释放预留。
                        missingResult :=
                                &coursewareTTSSegmentBillingResult{
                                        AudioFilePath:
                                                outputPath,
                                        AudioURL:
                                                audioURL,
                                        Duration:
                                                0,
                                        ModelUsed:
                                                normalized.ModelName,
                                        FileSize:
                                                0,
                                        CharacterCount:
                                                normalized.CharacterCount,
                                        GeneratedAt:
                                                time.Now().
                                                        Format(
                                                                time.RFC3339,
                                                        ),
                                }

                        if err :=
                                settleCoursewareTTSSegmentBilling(
                                        billing,
                                        missingResult,
                                        requestFingerprint,
                                ); err != nil {
                                return nil,
                                        fmt.Errorf(
                                                "%w: 供应商已成功但结算失败: %v",
                                                ErrCoursewareTTSBillingPending,
                                                err,
                                        )
                        }

                        return nil,
                                ErrCoursewareTTSBillingOutputMissing

                case ai.IsTTSSynthesisUncertain(
                        synthErr,
                ):
                        // 同步接口没有可查询任务ID。
                        // 请求结果不确定时保持reserved并禁止自动重复供应商调用。
                        return nil,
                                fmt.Errorf(
                                        "%w: %v",
                                        ErrCoursewareTTSBillingPending,
                                        synthErr,
                                )

                default:
                        releaseErr :=
                                releaseCoursewareTTSSegmentBilling(
                                        billing,
                                        "tts_provider_failed",
                                        map[string]interface{}{
                                                "provider_error":
                                                        truncateMediaBillingReason(
                                                                synthErr.Error(),
                                                        ),
                                        },
                                )

                        if releaseErr != nil {
                                return nil,
                                        fmt.Errorf(
                                                "%w: 合成失败且积分预留释放失败: %v",
                                                ErrCoursewareTTSBillingPending,
                                                releaseErr,
                                        )
                        }

                        return nil, synthErr
                }
        }
	if result == nil {
		releaseErr :=
			releaseCoursewareTTSSegmentBilling(
				billing,
				"tts_provider_result_missing",
				nil,
			)

		if releaseErr != nil {
			return nil,
				fmt.Errorf(
					"%w: TTS空结果且积分预留释放失败: %v",
					ErrCoursewareTTSBillingPending,
					releaseErr,
				)
		}

		return nil,
			fmt.Errorf("TTS未返回结果")
	}

	resultPath :=
		strings.TrimSpace(
			result.AudioFilePath,
		)

	if resultPath == "" {
		resultPath = outputPath
	}

	fileResult, ok :=
		loadCoursewareTTSSegmentFileResult(
			resultPath,
			audioURL,
			result.ModelUsed,
			normalized.CharacterCount,
		)

	if !ok {
		// 供应商已经返回成功结果，此时真实成本已经产生。
		// 即使本地文件缺失，也按真正提交的字符数结算。
		missingResult :=
			&coursewareTTSSegmentBillingResult{
				AudioFilePath:
					resultPath,
				AudioURL:
					audioURL,
				Duration:
					result.Duration,
				ModelUsed:
					result.ModelUsed,
				FileSize:
					result.FileSize,
				CharacterCount:
					normalized.CharacterCount,
				GeneratedAt:
					time.Now().
						Format(
							time.RFC3339,
						),
			}

		if err :=
			settleCoursewareTTSSegmentBilling(
				billing,
				missingResult,
				requestFingerprint,
			); err != nil {
			return nil,
				fmt.Errorf(
					"%w: %v",
					ErrCoursewareTTSBillingPending,
					err,
				)
		}

		return nil,
			ErrCoursewareTTSBillingOutputMissing
	}

	if fileResult.Duration <= 0 {
		fileResult.Duration =
			result.Duration
	}

	if strings.TrimSpace(
		fileResult.ModelUsed,
	) == "" {
		fileResult.ModelUsed =
			result.ModelUsed
	}

	if err :=
		settleCoursewareTTSSegmentBilling(
			billing,
			fileResult,
			requestFingerprint,
		); err != nil {
		// 音频已经形成，不能删除。
		// 同一operation_id重试会从确定性文件补结算。
		return nil,
			fmt.Errorf(
				"%w: %v",
				ErrCoursewareTTSBillingPending,
				err,
			)
	}

	return fileResult, nil
}

// recoverBilledCoursewareTTSSegment 恢复已有逐段计费记录。
func recoverBilledCoursewareTTSSegment(
	billing *models.TokenMediaBilling,
	input *coursewareTTSSegmentBillingInput,
	requestFingerprint string,
	outputPath string,
	audioURL string,
) (*coursewareTTSSegmentBillingResult, error) {
	if billing == nil ||
		input == nil {
		return nil,
			ErrMediaBillingInvalidRequest
	}

	if err :=
		validateCoursewareTTSSegmentBillingIdentity(
			billing,
			input,
			requestFingerprint,
		); err != nil {
		return nil, err
	}

	switch billing.Status {
	case models.MediaBillingStatusFailed,
		models.MediaBillingStatusCancelled:
		return nil,
			ErrCoursewareTTSBillingTerminal

	case models.MediaBillingStatusReserved,
		models.MediaBillingStatusSettled:

	default:
		return nil,
			ErrCoursewareTTSBillingTerminal
	}

	recovered, ok :=
		loadCoursewareTTSSegmentFileResult(
			outputPath,
			audioURL,
			billing.ModelName,
			input.CharacterCount,
		)

	if !ok {
		if billing.Status ==
			models.MediaBillingStatusSettled {
			return nil,
				ErrCoursewareTTSBillingOutputMissing
		}

		return nil,
			ErrCoursewareTTSBillingInProgress
	}

	if billing.Status ==
		models.MediaBillingStatusReserved {
		if err :=
			settleCoursewareTTSSegmentBilling(
				billing,
				recovered,
				requestFingerprint,
			); err != nil {
			return nil,
				fmt.Errorf(
					"%w: %v",
					ErrCoursewareTTSBillingPending,
					err,
				)
		}
	}

	return recovered, nil
}

func normalizeCoursewareTTSSegmentBillingInput(
	input *coursewareTTSSegmentBillingInput,
) (*coursewareTTSSegmentBillingInput, error) {
	if input == nil {
		return nil,
			ErrMediaBillingInvalidRequest
	}

	normalized := *input

	normalized.UserID =
		strings.TrimSpace(
			normalized.UserID,
		)
	normalized.CoursewareID =
		strings.TrimSpace(
			normalized.CoursewareID,
		)
	normalized.SubtitleID =
		strings.TrimSpace(
			normalized.SubtitleID,
		)
	normalized.SegmentID =
		strings.TrimSpace(
			normalized.SegmentID,
		)
	normalized.OperationID =
		strings.TrimSpace(
			normalized.OperationID,
		)
	normalized.Text =
		strings.TrimSpace(
			normalized.Text,
		)
	normalized.Voice =
		strings.TrimSpace(
			normalized.Voice,
		)
	normalized.ModelName =
		strings.TrimSpace(
			normalized.ModelName,
		)
	normalized.OutputDir =
		strings.TrimSpace(
			normalized.OutputDir,
		)

	if normalized.Speed <= 0 {
		normalized.Speed = 1
	}

	if _, err :=
		uuid.Parse(
			normalized.OperationID,
		); err != nil {
		return nil,
			ErrMediaBillingInvalidRequest
	}

	normalized.CharacterCount =
		utf8.RuneCountInString(
			normalized.Text,
		)

	if normalized.UserID == "" ||
		normalized.CoursewareID == "" ||
		normalized.SubtitleID == "" ||
		normalized.SegmentID == "" ||
		normalized.Text == "" ||
		normalized.Voice == "" ||
		normalized.ModelName == "" ||
		normalized.OutputDir == "" ||
		normalized.CharacterCount <= 0 {
		return nil,
			ErrMediaBillingInvalidRequest
	}

	return &normalized, nil
}

func validateCoursewareTTSSegmentBillingIdentity(
	billing *models.TokenMediaBilling,
	input *coursewareTTSSegmentBillingInput,
	requestFingerprint string,
) error {
	if billing == nil ||
		input == nil {
		return ErrMediaBillingInvalidRequest
	}

	billingCoursewareID := ""

	if billing.CoursewareID != nil {
		billingCoursewareID =
			strings.TrimSpace(
				*billing.CoursewareID,
			)
	}

	metadata :=
		coursewareTTSBillingMetadataMap(
			billing.Metadata,
		)

	if strings.TrimSpace(
		billing.UserID,
	) != input.UserID ||
		strings.TrimSpace(
			billing.BillingNodeCode,
		) != coursewareTTSBillingNodeCode ||
		strings.TrimSpace(
			billing.MediaType,
		) != models.MediaTypeTTS ||
		strings.TrimSpace(
			billing.ModelName,
		) != input.ModelName ||
		billingCoursewareID !=
			input.CoursewareID ||
		coursewareTTSMetadataText(
			metadata,
			"request_fingerprint",
		) != requestFingerprint {
		return ErrCoursewareTTSBillingIdentityMismatch
	}

	return nil
}

func settleCoursewareTTSSegmentBilling(
	billing *models.TokenMediaBilling,
	result *coursewareTTSSegmentBillingResult,
	requestFingerprint string,
) error {
	if billing == nil ||
		result == nil ||
		result.CharacterCount <= 0 {
		return ErrMediaBillingInvalidRequest
	}

	if billing.Status ==
		models.MediaBillingStatusSettled {
		return nil
	}

	if billing.Status !=
		models.MediaBillingStatusReserved {
		return ErrCoursewareTTSBillingTerminal
	}

	latencyMS :=
		int(
			time.Since(
				billing.CreatedAt,
			).Milliseconds(),
		)

	if latencyMS < 0 {
		latencyMS = 0
	}

	settleCtx, cancel :=
		context.WithTimeout(
			context.Background(),
			coursewareTTSBillingTimeout,
		)
	defer cancel()

	_, err :=
		NewMediaBillingService().
			Settle(
				settleCtx,
				&models.MediaBillingSettleRequest{
					IdempotencyKey:
						billing.IdempotencyKey,
					ActualQuantity:
						float64(
							result.CharacterCount,
						),
					LatencyMs:
						latencyMS,
					Metadata:
						map[string]interface{}{
							"provider_succeeded":
								true,
							"request_fingerprint":
								requestFingerprint,
							"audio_url":
								result.AudioURL,
							"audio_file_path":
								result.AudioFilePath,
							"audio_duration":
								result.Duration,
							"file_size":
								result.FileSize,
							"model_used":
								result.ModelUsed,
							"actual_character_count":
								result.CharacterCount,
						},
				},
			)

	return err
}

func releaseCoursewareTTSSegmentBilling(
	billing *models.TokenMediaBilling,
	reason string,
	metadata map[string]interface{},
) error {
	if billing == nil {
		return ErrMediaBillingInvalidRequest
	}

	if billing.Status ==
		models.MediaBillingStatusFailed ||
		billing.Status ==
			models.MediaBillingStatusCancelled {
		return nil
	}

	if billing.Status !=
		models.MediaBillingStatusReserved {
		return ErrCoursewareTTSBillingTerminal
	}

	releaseCtx, cancel :=
		context.WithTimeout(
			context.Background(),
			coursewareTTSBillingTimeout,
		)
	defer cancel()

	_, err :=
		NewMediaBillingService().
			Release(
				releaseCtx,
				&models.MediaBillingReleaseRequest{
					IdempotencyKey:
						billing.IdempotencyKey,
					Status:
						models.MediaBillingStatusFailed,
					FailureReason:
						truncateMediaBillingReason(
							reason,
						),
					Metadata:
						metadata,
				},
			)

	return err
}

func loadCoursewareTTSSegmentFileResult(
	path string,
	audioURL string,
	modelUsed string,
	characterCount int,
) (*coursewareTTSSegmentBillingResult, bool) {
	path =
		strings.TrimSpace(
			path,
		)

	if path == "" ||
		characterCount <= 0 {
		return nil, false
	}

	stat, err :=
		os.Stat(
			path,
		)

	if err != nil ||
		stat.IsDir() ||
		stat.Size() < 100 {
		return nil, false
	}

	duration :=
		probeCoursewareTTSAudioDuration(
			path,
		)

	return &coursewareTTSSegmentBillingResult{
		AudioFilePath:
			path,
		AudioURL:
			strings.TrimSpace(
				audioURL,
			),
		Duration:
			duration,
		ModelUsed:
			strings.TrimSpace(
				modelUsed,
			),
		FileSize:
			stat.Size(),
		CharacterCount:
			characterCount,
		GeneratedAt:
			stat.ModTime().
				Format(
					time.RFC3339,
				),
	}, true
}

func probeCoursewareTTSAudioDuration(
	path string,
) float64 {
	command :=
		exec.Command(
			"ffprobe",
			"-v",
			"quiet",
			"-print_format",
			"json",
			"-show_format",
			path,
		)

	output, err :=
		command.Output()

	if err != nil {
		return 0
	}

	var probe struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}

	if err :=
		json.Unmarshal(
			output,
			&probe,
		); err != nil {
		return 0
	}

	duration, err :=
		strconv.ParseFloat(
			probe.Format.Duration,
			64,
		)

	if err != nil ||
		duration < 0 {
		return 0
	}

	return duration
}

func coursewareTTSSegmentIdempotencyKey(
	operationID string,
	subtitleID string,
	segmentID string,
) string {
	return fmt.Sprintf(
		"courseware-tts:%s:%s",
		strings.TrimSpace(
			operationID,
		),
		coursewareTTSDigest(
			subtitleID,
			segmentID,
		),
	)
}

func coursewareTTSSegmentOutputName(
	operationID string,
	subtitleID string,
	segmentID string,
) string {
	return "tts_" +
		coursewareTTSDigest(
			operationID,
			subtitleID,
			segmentID,
		)
}

func coursewareTTSSegmentFingerprint(
	input *coursewareTTSSegmentBillingInput,
) string {
	return coursewareTTSDigest(
		input.CoursewareID,
		input.SubtitleID,
		input.SegmentID,
		input.Text,
		input.Voice,
		strconv.FormatFloat(
			input.Speed,
			'f',
			4,
			64,
		),
		input.ModelName,
	)
}

func coursewareTTSDigest(
	values ...string,
) string {
	normalized :=
		make(
			[]string,
			0,
			len(values),
		)

	for _, value :=
		range values {
		normalized =
			append(
				normalized,
				strings.TrimSpace(
					value,
				),
			)
	}

	sum :=
		sha256.Sum256(
			[]byte(
				strings.Join(
					normalized,
					"\x1f",
				),
			),
		)

	return hex.EncodeToString(
		sum[:12],
	)
}

func coursewareTTSStringPointer(
	value string,
) *string {
	normalized :=
		strings.TrimSpace(
			value,
		)

	if normalized == "" {
		return nil
	}

	return &normalized
}

func coursewareTTSBillingMetadataMap(
	raw json.RawMessage,
) map[string]interface{} {
	result :=
		map[string]interface{}{}

	if len(raw) > 0 {
		_ =
			json.Unmarshal(
				raw,
				&result,
			)
	}

	return result
}

func coursewareTTSMetadataText(
	metadata map[string]interface{},
	key string,
) string {
	value, exists :=
		metadata[key]

	if !exists {
		return ""
	}

	text, ok :=
		value.(string)

	if !ok {
		return ""
	}

	return strings.TrimSpace(
		text,
	)
}
