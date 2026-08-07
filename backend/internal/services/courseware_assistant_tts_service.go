package services

// courseware_assistant_tts_service.go
//
// 老师端课件教学智能体回答朗读服务。
//
// 安全与计费边界：
//   - 只接受已经通过教师JWT认证的当前教师ID；
//   - 部署必须同时匹配deployment_id与owner_user_id，管理员不能代替其他教师调用；
//   - 只允许当前可运行的active部署；
//   - 课件、页面、学校和付费用户全部来自数据库部署记录，不信任请求正文；
//   - 豆包调用前按字符数预留TTS积分，成功后按实际字符数结算；
//   - operation_id与部署、文本、音色和语速共同形成幂等键；
//   - 已结算音频在私有缓存中短时保留，支持响应丢失后的无重复计费恢复；
//   - 供应商结果不确定或供应商已成功但本地处理失败时保留reserved事实，禁止误释放；
//   - 本服务不写课件素材，不修改教学智能体会话和消息记录。
//
// 音色策略：
//   - 中文或中英混合回答默认使用vivi 2.0；
//   - 纯英文或英文显著占优的回答使用Tim；
//   - 两种音色均走现有Seed TTS 2.0正式通道。

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"

	"tedna/internal/ai"
	"tedna/internal/config"
	"tedna/internal/models"
	"tedna/internal/repository"
)

const (
	coursewareAssistantTTSBillingNode = "courseware_assistant_tts"
	coursewareAssistantTTSSceneCode   = "courseware_assistant_tts"
	coursewareAssistantTTSProvider    = "volcengine"

	coursewareAssistantTTSChineseVoice = "zh_female_vv_uranus_bigtts"
	coursewareAssistantTTSEnglishVoice = "en_male_tim_uranus_bigtts"

	coursewareAssistantTTSDefaultSpeed = 1.0
	coursewareAssistantTTSMaxRunes     = 4000
	coursewareAssistantTTSCacheTTL     = 30 * time.Minute

	coursewareAssistantTTSCacheDir = "/www/wwwroot/tedna/private/assistant-tts-cache"
)

var (
	ErrCoursewareAssistantTTSInvalidRequest = errors.New(
		"课件教学智能体朗读请求无效",
	)
	ErrCoursewareAssistantTTSUnavailable = errors.New(
		"课件教学智能体朗读服务不可用",
	)
	ErrCoursewareAssistantTTSInProgress = errors.New(
		"课件教学智能体朗读任务正在处理",
	)
	ErrCoursewareAssistantTTSIdentityMismatch = errors.New(
		"课件教学智能体朗读任务身份不匹配",
	)
	ErrCoursewareAssistantTTSTerminal = errors.New(
		"课件教学智能体朗读任务已经结束",
	)
	ErrCoursewareAssistantTTSOutputMissing = errors.New(
		"课件教学智能体朗读音频缓存不存在",
	)
	ErrCoursewareAssistantTTSPending = errors.New(
		"课件教学智能体朗读结果等待恢复",
	)
	ErrCoursewareAssistantTTSSynthesisFailed = errors.New(
		"课件教学智能体豆包朗读合成失败",
	)
)

// CoursewareAssistantTTSService 负责教师部署授权、豆包合成、短时缓存和媒体积分结算。
type CoursewareAssistantTTSService struct {
	cfg                 *config.Config
	mediaBillingService *MediaBillingService
}

// CoursewareAssistantTTSResult 是Handler写入浏览器的音频结果。
type CoursewareAssistantTTSResult struct {
	Audio       []byte
	Voice       string
	Language    string
	ModelUsed   string
	Duration    float64
	FileSize    int64
	CacheHit    bool
	OperationID string
}

// NewCoursewareAssistantTTSService 创建老师端教学智能体朗读服务。
func NewCoursewareAssistantTTSService(
	cfg *config.Config,
) *CoursewareAssistantTTSService {
	return &CoursewareAssistantTTSService{
		cfg:                 cfg,
		mediaBillingService: NewMediaBillingService(),
	}
}

// Synthesize 为当前教师本人拥有的部署生成豆包朗读音频。
func (s *CoursewareAssistantTTSService) Synthesize(
	ctx context.Context,
	deploymentID string,
	teacherUserID string,
	text string,
	operationID string,
) (
	*CoursewareAssistantTTSResult,
	error,
) {
	if ctx == nil {
		ctx = context.Background()
	}

	if s == nil ||
		s.cfg == nil ||
		s.mediaBillingService == nil {
		return nil, ErrCoursewareAssistantTTSUnavailable
	}

	deploymentID = strings.TrimSpace(deploymentID)
	teacherUserID = strings.TrimSpace(teacherUserID)
	text = normalizeCoursewareAssistantTTSText(text)
	operationID = strings.TrimSpace(operationID)

	if deploymentID == "" ||
		teacherUserID == "" ||
		text == "" {
		return nil, ErrCoursewareAssistantTTSInvalidRequest
	}

	if _, err := uuid.Parse(operationID); err != nil {
		return nil, ErrCoursewareAssistantTTSInvalidRequest
	}

	textRunes := utf8.RuneCountInString(text)
	if textRunes <= 0 ||
		textRunes > coursewareAssistantTTSMaxRunes {
		return nil, ErrCoursewareAssistantTTSInvalidRequest
	}

	deployment, err := repository.GetAssistantDeploymentForOwner(
		ctx,
		deploymentID,
		teacherUserID,
	)
	if err != nil {
		return nil, err
	}

	if err := validateAssistantRuntimeDeploymentForSessionStart(
		deployment,
		time.Now().UTC(),
	); err != nil {
		return nil, err
	}

	voice, language := selectCoursewareAssistantTTSVoice(text)
	speed := coursewareAssistantTTSDefaultSpeed
	schoolID := coursewareAssistantTTSSchoolID(
		deployment.SchoolID,
	)

	ttsConfig, err := ai.GetTTSConfig(s.cfg.GetAESKey())
	if err != nil {
		return nil, fmt.Errorf(
			"%w: %v",
			ErrCoursewareAssistantTTSUnavailable,
			err,
		)
	}

	modelName := ai.ResolveTTSResourceID(voice)
	if strings.TrimSpace(modelName) == "" {
		return nil, ErrCoursewareAssistantTTSUnavailable
	}

	if err := ensureCoursewareAssistantTTSCacheDir(); err != nil {
		return nil, fmt.Errorf(
			"%w: %v",
			ErrCoursewareAssistantTTSUnavailable,
			err,
		)
	}

	cleanupCoursewareAssistantTTSCache()

	textHash := coursewareAssistantTTSTextHash(text)
	idempotencyKey := coursewareAssistantTTSIdempotencyKey(
		operationID,
	)
	cachePath := coursewareAssistantTTSCachePath(idempotencyKey)

	metadata := map[string]interface{}{
		"operation_id":  operationID,
		"deployment_id": deployment.ID,
		"text_hash":     textHash,
		"voice":         voice,
		"language":      language,
		"speed":         speed,
		"cache_path":    cachePath,
	}

	billing, err := s.mediaBillingService.Reserve(
		ctx,
		&models.MediaBillingReserveRequest{
			UserID:            deployment.OwnerUserID,
			SchoolID:          schoolID,
			BillingCategory:   models.BillingCategoryTTS,
			BillingNodeCode:   coursewareAssistantTTSBillingNode,
			SceneCode:         coursewareAssistantTTSSceneCode,
			MediaType:         models.MediaTypeTTS,
			Provider:          coursewareAssistantTTSProvider,
			ModelName:         modelName,
			Variant:           "default",
			MediaUnit:         models.MediaUnitCharacter,
			EstimatedQuantity: float64(textRunes),
			IdempotencyKey:    idempotencyKey,
			CoursewareID:      coursewareAssistantTTSStringPointer(deployment.CoursewareID),
			PageID:            coursewareAssistantTTSStringPointer(deployment.PageID),
			Metadata:          metadata,
		},
	)
	if err != nil {
		return nil, err
	}

	if !billing.ReservationCreated {
		return s.recoverCoursewareAssistantTTS(
			ctx,
			billing,
			cachePath,
			operationID,
			voice,
			language,
			modelName,
			textRunes,
			deployment.ID,
			textHash,
			speed,
		)
	}

	userID := deployment.OwnerUserID
	traceContext := &ai.TraceContext{
		UserID:    &userID,
		SchoolID:  schoolID,
		SceneCode: coursewareAssistantTTSSceneCode,
	}

	outputName := strings.TrimSuffix(
		filepath.Base(cachePath),
		filepath.Ext(cachePath),
	)

	startedAt := time.Now()

	synthesisResult, synthesisErr := ai.SynthesizeSpeech(
		ctx,
		ttsConfig,
		text,
		voice,
		speed,
		coursewareAssistantTTSCacheDir,
		outputName,
		traceContext,
	)

	latencyMS := int(time.Since(startedAt).Milliseconds())
	if latencyMS < 0 {
		latencyMS = 0
	}

	if synthesisErr != nil {
		return nil, s.handleCoursewareAssistantTTSSynthesisError(
			ctx,
			idempotencyKey,
			synthesisErr,
			latencyMS,
			metadata,
		)
	}

	if synthesisResult == nil ||
		strings.TrimSpace(synthesisResult.AudioFilePath) == "" {
		return nil, s.preserveCoursewareAssistantTTSPending(
			ctx,
			idempotencyKey,
			"tts_result_missing",
			errors.New("TTS未返回音频文件"),
			metadata,
		)
	}

	if synthesisResult.AudioFilePath != cachePath {
		if err := os.Rename(
			synthesisResult.AudioFilePath,
			cachePath,
		); err != nil {
			return nil, s.preserveCoursewareAssistantTTSPending(
				ctx,
				idempotencyKey,
				"tts_cache_rename_failed",
				err,
				metadata,
			)
		}
	}

	_ = os.Chmod(cachePath, 0o600)

	audio, err := os.ReadFile(cachePath)
	if err != nil ||
		len(audio) < 100 {
		if err == nil {
			err = errors.New("TTS音频文件异常小")
		}

		return nil, s.preserveCoursewareAssistantTTSPending(
			ctx,
			idempotencyKey,
			"tts_cache_read_failed",
			err,
			metadata,
		)
	}

	settled, err := s.mediaBillingService.Settle(
		ctx,
		&models.MediaBillingSettleRequest{
			IdempotencyKey: idempotencyKey,
			ActualQuantity: float64(textRunes),
			LatencyMs:      latencyMS,
			Metadata: map[string]interface{}{
				"operation_id": operationID,
				"voice":        voice,
				"language":     language,
				"file_size":    len(audio),
				"duration":     synthesisResult.Duration,
				"cache_path":   cachePath,
			},
		},
	)
	if err != nil ||
		settled == nil {
		if err == nil {
			err = errors.New("TTS积分结算未返回结果")
		}

		return nil, s.preserveCoursewareAssistantTTSPending(
			ctx,
			idempotencyKey,
			"tts_settlement_failed",
			err,
			metadata,
		)
	}

	return &CoursewareAssistantTTSResult{
		Audio:       audio,
		Voice:       voice,
		Language:    language,
		ModelUsed:   modelName,
		Duration:    synthesisResult.Duration,
		FileSize:    int64(len(audio)),
		CacheHit:    false,
		OperationID: operationID,
	}, nil
}

func (s *CoursewareAssistantTTSService) recoverCoursewareAssistantTTS(
	ctx context.Context,
	billing *models.TokenMediaBilling,
	cachePath string,
	operationID string,
	voice string,
	language string,
	modelName string,
	textRunes int,
	deploymentID string,
	textHash string,
	speed float64,
) (
	*CoursewareAssistantTTSResult,
	error,
) {
	if billing == nil {
		return nil, ErrCoursewareAssistantTTSUnavailable
	}

	if !coursewareAssistantTTSBillingIdentityMatches(
		billing,
		operationID,
		deploymentID,
		textHash,
		voice,
		speed,
	) {
		return nil, ErrCoursewareAssistantTTSIdentityMismatch
	}

	switch billing.Status {
	case models.MediaBillingStatusSettled:
		audio, err := os.ReadFile(cachePath)
		if err != nil ||
			len(audio) < 100 {
			return nil, ErrCoursewareAssistantTTSOutputMissing
		}

		return &CoursewareAssistantTTSResult{
			Audio:       audio,
			Voice:       voice,
			Language:    language,
			ModelUsed:   modelName,
			FileSize:    int64(len(audio)),
			CacheHit:    true,
			OperationID: operationID,
		}, nil

	case models.MediaBillingStatusReserved:
		audio, err := os.ReadFile(cachePath)
		if err != nil ||
			len(audio) < 100 {
			return nil, ErrCoursewareAssistantTTSInProgress
		}

		settled, settleErr := s.mediaBillingService.Settle(
			ctx,
			&models.MediaBillingSettleRequest{
				IdempotencyKey: billing.IdempotencyKey,
				ActualQuantity: float64(textRunes),
				LatencyMs:      0,
				Metadata: map[string]interface{}{
					"operation_id": operationID,
					"voice":        voice,
					"language":     language,
					"file_size":    len(audio),
					"cache_path":   cachePath,
					"recovered":    true,
				},
			},
		)
		if settleErr != nil ||
			settled == nil {
			return nil, ErrCoursewareAssistantTTSPending
		}

		return &CoursewareAssistantTTSResult{
			Audio:       audio,
			Voice:       voice,
			Language:    language,
			ModelUsed:   modelName,
			FileSize:    int64(len(audio)),
			CacheHit:    true,
			OperationID: operationID,
		}, nil

	case models.MediaBillingStatusFailed,
		models.MediaBillingStatusCancelled:
		return nil, ErrCoursewareAssistantTTSTerminal

	default:
		return nil, ErrCoursewareAssistantTTSUnavailable
	}
}

func (s *CoursewareAssistantTTSService) handleCoursewareAssistantTTSSynthesisError(
	ctx context.Context,
	idempotencyKey string,
	synthesisErr error,
	latencyMS int,
	metadata map[string]interface{},
) error {
	if ai.IsTTSSynthesisUncertain(synthesisErr) ||
		ai.DidTTSSynthesisProviderSucceed(synthesisErr) {
		return s.preserveCoursewareAssistantTTSPending(
			ctx,
			idempotencyKey,
			"tts_provider_result_uncertain",
			synthesisErr,
			metadata,
		)
	}

	_, releaseErr := s.mediaBillingService.Release(
		ctx,
		&models.MediaBillingReleaseRequest{
			IdempotencyKey: idempotencyKey,
			Status:         models.MediaBillingStatusFailed,
			FailureReason:  "courseware_assistant_tts_synthesis_failed",
			Metadata: map[string]interface{}{
				"latency_ms": latencyMS,
			},
		},
	)
	if releaseErr != nil {
		return errors.Join(
			fmt.Errorf("%w: %v", ErrCoursewareAssistantTTSSynthesisFailed, synthesisErr),
			releaseErr,
		)
	}

	return fmt.Errorf(
		"%w: %v",
		ErrCoursewareAssistantTTSSynthesisFailed,
		synthesisErr,
	)
}

func (s *CoursewareAssistantTTSService) preserveCoursewareAssistantTTSPending(
	ctx context.Context,
	idempotencyKey string,
	reason string,
	cause error,
	metadata map[string]interface{},
) error {
	pendingMetadata := map[string]interface{}{
		"pending_reason": reason,
	}

	for key, value := range metadata {
		pendingMetadata[key] = value
	}

	if cause != nil {
		pendingMetadata["pending_error"] = truncateCoursewareAssistantTTSError(
			cause.Error(),
		)
	}

	_, _ = s.mediaBillingService.AnnotateReserved(
		ctx,
		&models.MediaBillingAnnotateRequest{
			IdempotencyKey: idempotencyKey,
			Metadata:       pendingMetadata,
		},
	)

	return ErrCoursewareAssistantTTSPending
}

func normalizeCoursewareAssistantTTSText(text string) string {
	replacer := strings.NewReplacer(
		"\r\n", "\n",
		"\r", "\n",
		"```", " ",
		"`", " ",
		"**", " ",
		"__", " ",
		"##", " ",
		"#", " ",
		">", " ",
		"|", " ",
		"~", " ",
	)

	text = replacer.Replace(text)

	lines := strings.Split(text, "\n")
	parts := make([]string, 0, len(lines))

	for _, line := range lines {
		normalized := strings.Join(
			strings.Fields(line),
			" ",
		)
		if normalized != "" {
			parts = append(parts, normalized)
		}
	}

	return strings.TrimSpace(strings.Join(parts, " "))
}

func selectCoursewareAssistantTTSVoice(text string) (
	voice string,
	language string,
) {
	hanCount := 0
	latinCount := 0

	for _, value := range text {
		switch {
		case unicode.Is(unicode.Han, value):
			hanCount++

		case unicode.IsLetter(value) &&
			value <= unicode.MaxASCII:
			latinCount++
		}
	}

	if hanCount == 0 &&
		latinCount > 0 {
		return coursewareAssistantTTSEnglishVoice, "en-US"
	}

	if latinCount >= 4 &&
		latinCount > hanCount*2 {
		return coursewareAssistantTTSEnglishVoice, "en-US"
	}

	return coursewareAssistantTTSChineseVoice, "zh-CN"
}

func coursewareAssistantTTSTextHash(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

func coursewareAssistantTTSIdempotencyKey(
	operationID string,
) string {
	return "courseware_assistant_tts:" + operationID
}

func coursewareAssistantTTSBillingIdentityMatches(
	billing *models.TokenMediaBilling,
	operationID string,
	deploymentID string,
	textHash string,
	voice string,
	speed float64,
) bool {
	if billing == nil {
		return false
	}

	metadata := map[string]interface{}{}
	if len(billing.Metadata) == 0 ||
		json.Unmarshal(billing.Metadata, &metadata) != nil {
		return false
	}

	return strings.TrimSpace(
		fmt.Sprint(metadata["operation_id"]),
	) == operationID &&
		strings.TrimSpace(
			fmt.Sprint(metadata["deployment_id"]),
		) == deploymentID &&
		strings.TrimSpace(
			fmt.Sprint(metadata["text_hash"]),
		) == textHash &&
		strings.TrimSpace(
			fmt.Sprint(metadata["voice"]),
		) == voice &&
		strings.TrimSpace(
			fmt.Sprint(metadata["speed"]),
		) == fmt.Sprintf("%.0f", speed)
}

func coursewareAssistantTTSCachePath(
	idempotencyKey string,
) string {
	sum := sha256.Sum256([]byte(idempotencyKey))

	return filepath.Join(
		coursewareAssistantTTSCacheDir,
		hex.EncodeToString(sum[:])+".mp3",
	)
}

func ensureCoursewareAssistantTTSCacheDir() error {
	if err := os.MkdirAll(
		coursewareAssistantTTSCacheDir,
		0o700,
	); err != nil {
		return err
	}

	return os.Chmod(
		coursewareAssistantTTSCacheDir,
		0o700,
	)
}

func cleanupCoursewareAssistantTTSCache() {
	entries, err := os.ReadDir(
		coursewareAssistantTTSCacheDir,
	)
	if err != nil {
		return
	}

	cutoff := time.Now().Add(
		-coursewareAssistantTTSCacheTTL,
	)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		path := filepath.Join(
			coursewareAssistantTTSCacheDir,
			entry.Name(),
		)

		info, statErr := entry.Info()
		if statErr != nil ||
			info.ModTime().After(cutoff) {
			continue
		}

		_ = os.Remove(path)
	}
}

func coursewareAssistantTTSSchoolID(value interface{}) *string {
	switch typed := value.(type) {
	case string:
		return coursewareAssistantTTSStringPointer(typed)

	case *string:
		if typed == nil {
			return nil
		}

		return coursewareAssistantTTSStringPointer(*typed)

	default:
		return nil
	}
}

func coursewareAssistantTTSStringPointer(value string) *string {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return nil
	}

	return &normalized
}

func truncateCoursewareAssistantTTSError(
	value string,
) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)

	if len(runes) <= 500 {
		return value
	}

	return string(runes[:500])
}
