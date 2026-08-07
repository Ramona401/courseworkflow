package services

// price_sync_service.go — 文本及媒体价格同步业务编排。
//
// 流程：查询目标→按sync_source拉价格→精确匹配→安全校验→保存预览→
// 管理员或调度器原子应用。本文件只处理价格，不参与媒体业务结算。

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"tedna/internal/config"
	"tedna/internal/models"
	"tedna/internal/repository"
)

const (
	defaultPriceSyncGroup            = "default"
	defaultPriceSyncMaxChangePercent = 50.0
	priceSyncCompareTolerance        = 0.000000001
)

type PriceSyncService struct {
	gateway *NewAPIPricingClient
}

func NewPriceSyncService(cfg *config.Config) *PriceSyncService {
	aesKey := ""
	if cfg != nil {
		aesKey = cfg.GetAESKey()
	}
	return &PriceSyncService{gateway: NewNewAPIPricingClient(aesKey)}
}

// Preview只生成同步预览，不修改正式价格。
func (service *PriceSyncService) Preview(
	ctx context.Context,
	request *models.PriceSyncPreviewRequest,
	actorID string,
) (*models.PriceSyncPreviewResponse, error) {
	normalized := normalizePriceSyncPreviewRequest(request)
	schedulerOnly := normalized.TriggerType == models.PriceSyncTriggerScheduler

	var textTargets []models.TextPriceSyncTarget
	var mediaTargets []models.MediaPriceSyncTarget
	var err error

	if normalized.IncludeText {
		textTargets, err = repository.ListTextPriceSyncTargets(ctx, schedulerOnly)
		if err != nil {
			return nil, err
		}
	}
	if normalized.IncludeMedia {
		mediaTargets, err = repository.ListMediaPriceSyncTargets(ctx, schedulerOnly)
		if err != nil {
			return nil, err
		}
	}

	type sourceResult struct {
		catalog *models.GatewayPriceCatalog
		err     error
	}
	sources := collectPriceSyncSources(textTargets, mediaTargets)
	sourceResults := make(map[string]sourceResult, len(sources))
	sourceURLs := make([]string, 0, len(sources))

	for _, source := range sources {
		catalog, fetchErr := service.gateway.Fetch(ctx, source)
		sourceResults[source] = sourceResult{catalog: catalog, err: fetchErr}
		if catalog != nil && strings.TrimSpace(catalog.PricingURL) != "" {
			sourceURLs = append(sourceURLs, catalog.PricingURL)
		}
	}

	items := make([]models.PriceSyncItem, 0, len(textTargets)+len(mediaTargets))
	for _, target := range textTargets {
		source := normalizeTextPriceSyncSource(target.SyncSource)
		result := sourceResults[source]
		items = append(items, buildTextPriceSyncItem(
			target, source, result.catalog, result.err,
			normalized.Group, normalized.MaxChangePercent,
		))
	}
	for _, target := range mediaTargets {
		source := normalizeMediaPriceSyncSource(target)
		result := sourceResults[source]
		items = append(items, buildMediaPriceSyncItem(
			target, source, result.catalog, result.err,
			normalized.Group, normalized.MaxChangePercent,
		))
	}

	summary := summarizePriceSyncItems(items)
	var createdBy *string
	if actorID = strings.TrimSpace(actorID); actorID != "" {
		createdBy = &actorID
	}

	sort.Strings(sourceURLs)
	run := &models.PriceSyncRun{
		TriggerType:   normalized.TriggerType,
		Status:        models.PriceSyncRunPreviewed,
		SourceKind:    "multi",
		SourceBaseURL: strings.Join(uniquePriceSyncStrings(sourceURLs), ", "),
		PreviewOnly:   true,
		CreatedBy:     createdBy,
	}
	createdRun, err := repository.CreatePriceSyncPreview(ctx, run, items, summary)
	if err != nil {
		return nil, err
	}
	storedItems, err := repository.ListPriceSyncItems(ctx, createdRun.ID)
	if err != nil {
		return nil, err
	}
	return &models.PriceSyncPreviewResponse{
		Run: createdRun, Items: storedItems, Summary: summary,
	}, nil
}

// Apply保留旧调用方式：应用本批全部update明细。
func (service *PriceSyncService) Apply(
	ctx context.Context,
	runID string,
	actorID string,
) (*models.PriceSyncApplyResponse, error) {
	return service.ApplySelected(ctx, runID, nil, actorID)
}

// ApplySelected只应用管理员勾选项；itemIDs为空表示应用全部update。
func (service *PriceSyncService) ApplySelected(
	ctx context.Context,
	runID string,
	itemIDs []string,
	actorID string,
) (*models.PriceSyncApplyResponse, error) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return nil, fmt.Errorf("价格同步批次ID不能为空")
	}
	run, items, summary, err := repository.ApplyPriceSyncRun(
		ctx, runID, strings.TrimSpace(actorID), normalizePriceSyncItemIDs(itemIDs),
	)
	if err != nil {
		return nil, err
	}
	return &models.PriceSyncApplyResponse{Run: run, Items: items, Summary: summary}, nil
}

func (service *PriceSyncService) GetRunDetail(
	ctx context.Context,
	runID string,
) (*models.PriceSyncPreviewResponse, error) {
	run, err := repository.GetPriceSyncRun(ctx, strings.TrimSpace(runID))
	if err != nil {
		return nil, err
	}
	items, err := repository.ListPriceSyncItems(ctx, run.ID)
	if err != nil {
		return nil, err
	}
	summary := summarizePriceSyncItems(items)
	if len(run.Summary) > 0 {
		_ = json.Unmarshal(run.Summary, &summary)
	}
	return &models.PriceSyncPreviewResponse{Run: run, Items: items, Summary: summary}, nil
}

func (service *PriceSyncService) ListRuns(
	ctx context.Context,
	limit int,
) ([]models.PriceSyncRun, error) {
	return repository.ListPriceSyncRuns(ctx, limit)
}

func normalizePriceSyncPreviewRequest(
	request *models.PriceSyncPreviewRequest,
) models.PriceSyncPreviewRequest {
	result := models.PriceSyncPreviewRequest{
		TriggerType: models.PriceSyncTriggerManual,
		Group:       loadPriceSyncStringConfig("price_sync_group", defaultPriceSyncGroup),
		IncludeText: true, IncludeMedia: true,
		MaxChangePercent: loadPriceSyncFloatConfig(
			"price_sync_max_change_percent",
			defaultPriceSyncMaxChangePercent,
		),
	}
	if request == nil {
		return result
	}
	if request.TriggerType == models.PriceSyncTriggerScheduler {
		result.TriggerType = models.PriceSyncTriggerScheduler
	}
	if strings.TrimSpace(request.Group) != "" {
		result.Group = strings.TrimSpace(request.Group)
	}
	if request.IncludeText || request.IncludeMedia {
		result.IncludeText, result.IncludeMedia =
			request.IncludeText, request.IncludeMedia
	}
	if request.MaxChangePercent > 0 {
		result.MaxChangePercent = request.MaxChangePercent
	}
	return result
}

func buildTextPriceSyncItem(
	target models.TextPriceSyncTarget,
	source string,
	catalog *models.GatewayPriceCatalog,
	sourceErr error,
	group string,
	maxChangePercent float64,
) models.PriceSyncItem {
	syncModel := strings.TrimSpace(target.SyncModelName)
	if syncModel == "" {
		syncModel = strings.TrimSpace(target.ModelName)
	}
	item := models.PriceSyncItem{
		TargetKind: models.PriceSyncTargetText, TargetID: target.ID,
		Provider: target.Provider, ModelName: syncModel, SyncSource: source,
		OldInputUSD: target.CostPer1kInput, OldOutputUSD: target.CostPer1kOutput,
		Action: models.PriceSyncActionSkipped,
	}

	if sourceErr != nil || catalog == nil {
		item.Reason = "价格来源不可用：" + priceSyncServiceSafeError(sourceErr)
		return item
	}
	price, found := findGatewayPrice(catalog, syncModel)
	if !found {
		item.Reason = "上游不存在精确匹配模型"
		return item
	}
	item.SourcePayload = price.RawPayload

	switch {
	case price.QuotaType != 0:
		item.Reason = "上游为固定按次价格，不能转换为文本Token单价"
		return item
	case isUnconfiguredNewAPIRatio(price):
		item.Reason = "上游返回未配置倍率默认值37.5，已安全跳过"
		return item
	case price.ModelRatio <= 0:
		item.Reason = "上游输入倍率无效"
		return item
	}

	groupRatio, err := resolveGatewayGroupRatio(catalog, price, group)
	if err != nil {
		item.Reason = err.Error()
		return item
	}
	completionRatio := price.CompletionRatio
	if completionRatio <= 0 {
		completionRatio = 1
	}

	// New API中1美元=500000配额点，因此每百万输入Token美元价为
	// 2×模型倍率×分组倍率；本地按每1K Token保存，再除以1000。
	item.NewInputUSD = 2 * price.ModelRatio * groupRatio / 1000
	item.NewOutputUSD = item.NewInputUSD * completionRatio

	if exceedsPriceSyncChangeLimit(
		item.OldInputUSD, item.NewInputUSD, maxChangePercent,
	) || exceedsPriceSyncChangeLimit(
		item.OldOutputUSD, item.NewOutputUSD, maxChangePercent,
	) {
		item.Reason = fmt.Sprintf("价格变化超过安全上限%.2f%%", maxChangePercent)
		return item
	}
	if nearlyEqualPriceSync(item.OldInputUSD, item.NewInputUSD) &&
		nearlyEqualPriceSync(item.OldOutputUSD, item.NewOutputUSD) {
		item.Action, item.Reason =
			models.PriceSyncActionUnchanged, "本地价格与上游一致"
		return item
	}

	item.Action = models.PriceSyncActionUpdate
	if item.OldInputUSD <= 0 && item.OldOutputUSD <= 0 {
		item.Reason = "发现可信文本价格，可初始化当前零价格"
	} else {
		item.Reason = "发现可信文本价格变化"
	}
	return item
}

func buildMediaPriceSyncItem(
	target models.MediaPriceSyncTarget,
	source string,
	catalog *models.GatewayPriceCatalog,
	sourceErr error,
	group string,
	maxChangePercent float64,
) models.PriceSyncItem {
	syncModel := strings.TrimSpace(target.SyncModelName)
	if syncModel == "" {
		syncModel = strings.TrimSpace(target.ModelName)
	}
	item := models.PriceSyncItem{
		TargetKind: models.PriceSyncTargetMedia, TargetID: target.ID,
		Provider: target.Provider, ModelName: syncModel, SyncSource: source,
		MediaType: target.MediaType, Variant: target.Variant,
		MediaUnit: target.MediaUnit, OldUnitCostUSD: target.UnitCostUSD,
		Action: models.PriceSyncActionSkipped,
	}

	if sourceErr != nil || catalog == nil {
		item.Reason = "价格来源不可用：" + priceSyncServiceSafeError(sourceErr)
		return item
	}
	price, found := findGatewayPrice(catalog, syncModel)
	if !found {
		item.Reason = "上游不存在精确匹配媒体模型"
		return item
	}
	item.SourcePayload = price.RawPayload

	groupRatio, err := resolveGatewayGroupRatio(catalog, price, group)
	if err != nil {
		item.Reason = err.Error()
		return item
	}
	unitCost, valid := resolveGatewayMediaUnitCost(price, target.MediaUnit)
	if !valid {
		item.Reason = "上游价格无法安全换算为本地媒体计量单位"
		return item
	}

	item.NewUnitCostUSD = unitCost * groupRatio
	if item.NewUnitCostUSD <= 0 {
		item.Reason = "上游媒体单价无效"
		return item
	}
	if exceedsPriceSyncChangeLimit(
		item.OldUnitCostUSD, item.NewUnitCostUSD, maxChangePercent,
	) {
		item.Reason = fmt.Sprintf("媒体价格变化超过安全上限%.2f%%", maxChangePercent)
		return item
	}
	if nearlyEqualPriceSync(item.OldUnitCostUSD, item.NewUnitCostUSD) {
		item.Action, item.Reason =
			models.PriceSyncActionUnchanged, "本地媒体价格与上游一致"
		return item
	}

	item.Action = models.PriceSyncActionUpdate
	if item.OldUnitCostUSD <= 0 {
		item.Reason = "发现可信媒体价格，可初始化当前零价格"
	} else {
		item.Reason = "发现可信媒体价格变化"
	}
	return item
}

func resolveGatewayMediaUnitCost(price models.GatewayPrice, unit string) (float64, bool) {
	unit = strings.TrimSpace(unit)
	if value, exists := price.UnitPricesUSD[unit]; exists && value > 0 {
		return value, true
	}
	if price.QuotaType != 1 || price.ModelPriceUSD <= 0 {
		return 0, false
	}
	if unit == models.MediaUnitRequest || unit == models.MediaUnitImage {
		return price.ModelPriceUSD, true
	}
	return 0, false
}

// 分组倍率缺失时严格跳过，不默认猜测1.0。
func resolveGatewayGroupRatio(
	catalog *models.GatewayPriceCatalog,
	price models.GatewayPrice,
	group string,
) (float64, error) {
	group = strings.TrimSpace(group)
	if group == "" {
		group = defaultPriceSyncGroup
	}
	if len(price.EnableGroups) > 0 &&
		!containsPriceSyncString(price.EnableGroups, group) {
		return 0, fmt.Errorf("模型未对分组%s开放", group)
	}
	if catalog == nil || catalog.GroupRatios == nil {
		return 0, fmt.Errorf("上游未返回分组倍率")
	}
	ratio, exists := catalog.GroupRatios[group]
	if !exists {
		return 0, fmt.Errorf("上游未返回分组%s倍率", group)
	}
	if ratio <= 0 {
		return 0, fmt.Errorf("分组%s倍率无效", group)
	}
	return ratio, nil
}

func isUnconfiguredNewAPIRatio(price models.GatewayPrice) bool {
	return math.Abs(price.ModelRatio-37.5) < priceSyncCompareTolerance &&
		len(price.EnableGroups) == 0 && price.ModelPriceUSD <= 0
}

func findGatewayPrice(
	catalog *models.GatewayPriceCatalog,
	modelName string,
) (models.GatewayPrice, bool) {
	if catalog == nil {
		return models.GatewayPrice{}, false
	}
	modelName = strings.TrimSpace(modelName)
	for _, price := range catalog.Prices {
		if strings.TrimSpace(price.ModelName) == modelName {
			return price, true
		}
	}
	return models.GatewayPrice{}, false
}

func collectPriceSyncSources(
	textTargets []models.TextPriceSyncTarget,
	mediaTargets []models.MediaPriceSyncTarget,
) []string {
	sourceSet := map[string]bool{}
	for _, target := range textTargets {
		sourceSet[normalizeTextPriceSyncSource(target.SyncSource)] = true
	}
	for _, target := range mediaTargets {
		sourceSet[normalizeMediaPriceSyncSource(target)] = true
	}
	result := make([]string, 0, len(sourceSet))
	for source := range sourceSet {
		if source != "" {
			result = append(result, source)
		}
	}
	sort.Strings(result)
	return result
}

func normalizeTextPriceSyncSource(source string) string {
	if source = strings.TrimSpace(source); source != "" {
		return source
	}
	return models.PriceSyncSourceMainGateway
}

func normalizeMediaPriceSyncSource(target models.MediaPriceSyncTarget) string {
	if source := strings.TrimSpace(target.SyncSource); source != "" {
		return source
	}
	switch target.MediaType {
	case models.MediaTypeImage, models.MediaTypeVideo:
		return models.PriceSyncSourceMediaGateway
	case models.MediaTypeTTS:
		return models.PriceSyncSourceTTSGateway
	default:
		return models.PriceSyncSourceMainGateway
	}
}

func summarizePriceSyncItems(items []models.PriceSyncItem) models.PriceSyncSummary {
	summary := models.PriceSyncSummary{TotalCount: len(items)}
	for _, item := range items {
		switch item.Action {
		case models.PriceSyncActionUpdate:
			summary.UpdateCount++
		case models.PriceSyncActionUnchanged:
			summary.UnchangedCount++
		case models.PriceSyncActionSkipped:
			summary.SkippedCount++
		case models.PriceSyncActionApplied:
			summary.AppliedCount++
		case models.PriceSyncActionStale:
			summary.StaleCount++
		}
	}
	return summary
}

// 旧价格为0代表尚未初始化；只要来源可信，就允许生成更新预览。
func exceedsPriceSyncChangeLimit(oldValue, newValue, maxPercent float64) bool {
	if maxPercent <= 0 || oldValue <= 0 {
		return false
	}
	return math.Abs(newValue-oldValue)/oldValue*100 > maxPercent
}

func nearlyEqualPriceSync(left, right float64) bool {
	return math.Abs(left-right) <= priceSyncCompareTolerance
}

func containsPriceSyncString(values []string, target string) bool {
	target = strings.TrimSpace(target)
	for _, value := range values {
		if strings.TrimSpace(value) == target {
			return true
		}
	}
	return false
}

func uniquePriceSyncStrings(values []string) []string {
	result := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}

func normalizePriceSyncItemIDs(values []string) []string {
	return uniquePriceSyncStrings(values)
}

func loadPriceSyncStringConfig(key, fallback string) string {
	value, err := repository.GetConfigValue(key)
	if err != nil || strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func loadPriceSyncFloatConfig(key string, fallback float64) float64 {
	value, err := repository.GetConfigValue(key)
	if err != nil {
		return fallback
	}
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func priceSyncServiceSafeError(err error) string {
	if err == nil {
		return "未知错误"
	}
	runes := []rune(strings.TrimSpace(err.Error()))
	if len(runes) <= 300 {
		return string(runes)
	}
	return string(runes[:300]) + "..."
}
