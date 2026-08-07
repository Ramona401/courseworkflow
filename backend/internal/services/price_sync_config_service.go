package services

// price_sync_config_service.go — 价格同步管理配置业务层。
//
// 本文件负责：
//   - 读取和原子保存全局同步配置；
//   - 管理每个文本或媒体价格目标的同步来源和开关；
//   - 校验价格接口URL、同步间隔、安全阈值和来源类型。
//
// 本文件不修改正式价格，不参与图片、视频或TTS业务结算。

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

const (
	priceSyncEnabledKey          = "price_sync_enabled"
	priceSyncAutoApplyKey        = "price_sync_auto_apply"
	priceSyncGroupKey            = "price_sync_group"
	priceSyncIntervalHoursKey    = "price_sync_interval_hours"
	priceSyncMaxChangePercentKey = "price_sync_max_change_percent"

	priceSyncMainPricingURLKey     = "price_sync_main_pricing_url"
	priceSyncDomesticPricingURLKey = "price_sync_domestic_pricing_url"
	priceSyncMediaPricingURLKey    = "price_sync_media_pricing_url"
	priceSyncTTSPricingURLKey      = "price_sync_tts_pricing_url"
)

// GetManagementState 返回全局配置和全部价格同步目标。
func (service *PriceSyncService) GetManagementState(
	ctx context.Context,
) (*models.PriceSyncSettingsResponse, error) {
	settings, err := service.GetSettings(ctx)
	if err != nil {
		return nil, err
	}

	textTargets, mediaTargets, err :=
		repository.ListPriceSyncTargetConfigs(ctx)
	if err != nil {
		return nil, err
	}

	return &models.PriceSyncSettingsResponse{
		Settings:     *settings,
		TextTargets:  textTargets,
		MediaTargets: mediaTargets,
	}, nil
}

// GetSettings 读取价格同步全局配置。
func (service *PriceSyncService) GetSettings(
	ctx context.Context,
) (*models.PriceSyncSettings, error) {
	enabledRaw, err := readPriceSyncConfig(
		priceSyncEnabledKey,
		"false",
	)
	if err != nil {
		return nil, err
	}

	autoApplyRaw, err := readPriceSyncConfig(
		priceSyncAutoApplyKey,
		"false",
	)
	if err != nil {
		return nil, err
	}

	group, err := readPriceSyncConfig(
		priceSyncGroupKey,
		defaultPriceSyncGroup,
	)
	if err != nil {
		return nil, err
	}

	intervalRaw, err := readPriceSyncConfig(
		priceSyncIntervalHoursKey,
		"24",
	)
	if err != nil {
		return nil, err
	}

	maxChangeRaw, err := readPriceSyncConfig(
		priceSyncMaxChangePercentKey,
		"50",
	)
	if err != nil {
		return nil, err
	}

	mainURL, err := readPriceSyncConfig(
		priceSyncMainPricingURLKey,
		"",
	)
	if err != nil {
		return nil, err
	}

	domesticURL, err := readPriceSyncConfig(
		priceSyncDomesticPricingURLKey,
		"",
	)
	if err != nil {
		return nil, err
	}

	mediaURL, err := readPriceSyncConfig(
		priceSyncMediaPricingURLKey,
		"",
	)
	if err != nil {
		return nil, err
	}

	ttsURL, err := readPriceSyncConfig(
		priceSyncTTSPricingURLKey,
		"",
	)
	if err != nil {
		return nil, err
	}

	enabled, err := parsePriceSyncBool(
		priceSyncEnabledKey,
		enabledRaw,
	)
	if err != nil {
		return nil, err
	}

	autoApply, err := parsePriceSyncBool(
		priceSyncAutoApplyKey,
		autoApplyRaw,
	)
	if err != nil {
		return nil, err
	}

	intervalHours, err := strconv.Atoi(
		strings.TrimSpace(intervalRaw),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"%s配置不是整数",
			priceSyncIntervalHoursKey,
		)
	}

	maxChangePercent, err := strconv.ParseFloat(
		strings.TrimSpace(maxChangeRaw),
		64,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"%s配置不是数字",
			priceSyncMaxChangePercentKey,
		)
	}

	settings := &models.PriceSyncSettings{
		Enabled:          enabled,
		AutoApply:        autoApply,
		Group:            strings.TrimSpace(group),
		IntervalHours:    intervalHours,
		MaxChangePercent: maxChangePercent,

		MainPricingURL:     strings.TrimSpace(mainURL),
		DomesticPricingURL: strings.TrimSpace(domesticURL),
		MediaPricingURL:    strings.TrimSpace(mediaURL),
		TTSPricingURL:      strings.TrimSpace(ttsURL),
	}

	if err := validatePriceSyncSettings(settings); err != nil {
		return nil, err
	}

	return settings, nil
}

// UpdateSettings 原子保存价格同步全局配置。
func (service *PriceSyncService) UpdateSettings(
	ctx context.Context,
	request *models.UpdatePriceSyncSettingsRequest,
	updatedBy string,
) (*models.PriceSyncSettings, error) {
	if request == nil {
		return nil, fmt.Errorf("价格同步配置请求不能为空")
	}

	current, err := service.GetSettings(ctx)
	if err != nil {
		return nil, err
	}

	merged := *current

	if request.Enabled != nil {
		merged.Enabled = *request.Enabled
	}
	if request.AutoApply != nil {
		merged.AutoApply = *request.AutoApply
	}
	if request.Group != nil {
		merged.Group = strings.TrimSpace(*request.Group)
	}
	if request.IntervalHours != nil {
		merged.IntervalHours = *request.IntervalHours
	}
	if request.MaxChangePercent != nil {
		merged.MaxChangePercent = *request.MaxChangePercent
	}
	if request.MainPricingURL != nil {
		merged.MainPricingURL = strings.TrimSpace(
			*request.MainPricingURL,
		)
	}
	if request.DomesticPricingURL != nil {
		merged.DomesticPricingURL = strings.TrimSpace(
			*request.DomesticPricingURL,
		)
	}
	if request.MediaPricingURL != nil {
		merged.MediaPricingURL = strings.TrimSpace(
			*request.MediaPricingURL,
		)
	}
	if request.TTSPricingURL != nil {
		merged.TTSPricingURL = strings.TrimSpace(
			*request.TTSPricingURL,
		)
	}

	if err := validatePriceSyncSettings(&merged); err != nil {
		return nil, err
	}

	updates := []repository.ConfigValueUpdate{
		{
			Key:         priceSyncEnabledKey,
			Value:       strconv.FormatBool(merged.Enabled),
			Description: "是否启用模型价格定时同步",
		},
		{
			Key:         priceSyncAutoApplyKey,
			Value:       strconv.FormatBool(merged.AutoApply),
			Description: "定时同步后是否自动应用可信价格变化",
		},
		{
			Key:         priceSyncGroupKey,
			Value:       merged.Group,
			Description: "聚合网关价格同步使用的计费分组",
		},
		{
			Key: priceSyncIntervalHoursKey,
			Value: strconv.Itoa(
				merged.IntervalHours,
			),
			Description: "价格定时同步间隔小时数",
		},
		{
			Key: priceSyncMaxChangePercentKey,
			Value: strconv.FormatFloat(
				merged.MaxChangePercent,
				'f',
				-1,
				64,
			),
			Description: "单次价格变化安全上限百分比",
		},
		{
			Key:         priceSyncMainPricingURLKey,
			Value:       merged.MainPricingURL,
			Description: "主聚合网关价格接口，留空时自动推导",
		},
		{
			Key:         priceSyncDomesticPricingURLKey,
			Value:       merged.DomesticPricingURL,
			Description: "境内文本模型价格接口",
		},
		{
			Key:         priceSyncMediaPricingURLKey,
			Value:       merged.MediaPricingURL,
			Description: "图片和视频模型价格接口",
		},
		{
			Key:         priceSyncTTSPricingURLKey,
			Value:       merged.TTSPricingURL,
			Description: "TTS模型价格接口",
		},
	}

	if err := repository.UpsertConfigValues(
		updates,
		strings.TrimSpace(updatedBy),
	); err != nil {
		return nil, err
	}

	return service.GetSettings(ctx)
}

// UpdateTarget 更新单个价格同步目标。
func (service *PriceSyncService) UpdateTarget(
	ctx context.Context,
	targetKind string,
	targetID string,
	request *models.UpdatePriceSyncTargetRequest,
	updatedBy string,
) (*models.PriceSyncTargetConfig, error) {
	if request == nil {
		return nil, fmt.Errorf("价格同步目标请求不能为空")
	}

	current, err := repository.GetPriceSyncTargetConfig(
		ctx,
		strings.TrimSpace(targetKind),
		strings.TrimSpace(targetID),
	)
	if err != nil {
		return nil, err
	}

	merged := *current

	if request.AutoSyncEnabled != nil {
		merged.AutoSyncEnabled =
			*request.AutoSyncEnabled
	}
	if request.SyncSource != nil {
		merged.SyncSource = strings.TrimSpace(
			*request.SyncSource,
		)
	}
	if request.SyncModelName != nil {
		merged.SyncModelName = strings.TrimSpace(
			*request.SyncModelName,
		)
	}

	if merged.SyncModelName == "" {
		return nil, fmt.Errorf("上游模型名不能为空")
	}

	if err := validatePriceSyncTargetSource(
		&merged,
	); err != nil {
		return nil, err
	}

	return repository.UpdatePriceSyncTargetConfig(
		ctx,
		&merged,
		strings.TrimSpace(updatedBy),
	)
}

func readPriceSyncConfig(
	key string,
	fallback string,
) (string, error) {
	value, err := repository.GetConfigValue(key)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(value) == "" {
		return fallback, nil
	}
	return strings.TrimSpace(value), nil
}

func parsePriceSyncBool(
	key string,
	value string,
) (bool, error) {
	parsed, err := strconv.ParseBool(
		strings.TrimSpace(value),
	)
	if err != nil {
		return false, fmt.Errorf(
			"%s配置不是布尔值",
			key,
		)
	}
	return parsed, nil
}

func validatePriceSyncSettings(
	settings *models.PriceSyncSettings,
) error {
	if settings == nil {
		return fmt.Errorf("价格同步配置不能为空")
	}

	settings.Group = strings.TrimSpace(settings.Group)

	if settings.Group == "" {
		return fmt.Errorf("价格同步分组不能为空")
	}
	if len([]rune(settings.Group)) > 64 {
		return fmt.Errorf("价格同步分组不能超过64个字符")
	}
	if settings.IntervalHours < 1 ||
		settings.IntervalHours > 720 {
		return fmt.Errorf("同步间隔必须在1至720小时之间")
	}
	if settings.MaxChangePercent < 1 ||
		settings.MaxChangePercent > 1000 {
		return fmt.Errorf("价格变化安全阈值必须在1%%至1000%%之间")
	}

	for _, item := range []struct {
		name  string
		value string
	}{
		{"主聚合网关价格接口", settings.MainPricingURL},
		{"境内文本价格接口", settings.DomesticPricingURL},
		{"图片视频价格接口", settings.MediaPricingURL},
		{"TTS价格接口", settings.TTSPricingURL},
	} {
		if err := validateOptionalPriceSyncURL(
			item.name,
			item.value,
		); err != nil {
			return err
		}
	}

	return nil
}

func validateOptionalPriceSyncURL(
	name string,
	value string,
) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	parsed, err := url.Parse(value)
	if err != nil ||
		parsed.Scheme == "" ||
		parsed.Host == "" ||
		(parsed.Scheme != "http" &&
			parsed.Scheme != "https") {
		return fmt.Errorf("%s必须是合法HTTP或HTTPS地址", name)
	}

	if parsed.User != nil {
		return fmt.Errorf("%s不能在URL中携带用户名或密码", name)
	}

	return nil
}

func validatePriceSyncTargetSource(
	target *models.PriceSyncTargetConfig,
) error {
	if target == nil {
		return fmt.Errorf("价格同步目标不能为空")
	}

	source := strings.TrimSpace(target.SyncSource)

	if target.TargetKind == models.PriceSyncTargetText {
		if source != models.PriceSyncSourceMainGateway &&
			source != models.PriceSyncSourceDomesticGateway {
			return fmt.Errorf(
				"文本价格来源只能是main_gateway或domestic_gateway",
			)
		}
		return nil
	}

	if target.TargetKind != models.PriceSyncTargetMedia {
		return fmt.Errorf("未知价格同步目标类型")
	}

	switch target.MediaType {
	case models.MediaTypeImage,
		models.MediaTypeVideo:
		if source != models.PriceSyncSourceMainGateway &&
			source != models.PriceSyncSourceMediaGateway {
			return fmt.Errorf(
				"图片或视频价格来源只能是main_gateway或media_gateway",
			)
		}

	case models.MediaTypeTTS:
		if source != models.PriceSyncSourceMainGateway &&
			source != models.PriceSyncSourceTTSGateway {
			return fmt.Errorf(
				"TTS价格来源只能是main_gateway或tts_gateway",
			)
		}

	default:
		return fmt.Errorf("当前媒体类型不支持价格同步")
	}

	return nil
}
