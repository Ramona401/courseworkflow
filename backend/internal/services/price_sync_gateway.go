package services

// price_sync_gateway.go — New API兼容价格接口读取与安全解析。
//
// 支持四类价格来源：
//   - main_gateway：默认从api_base_url推导/api/pricing；
//   - domestic_gateway：必须显式配置price_sync_domestic_pricing_url；
//   - media_gateway：必须显式配置price_sync_media_pricing_url；
//   - tts_gateway：必须显式配置price_sync_tts_pricing_url。
//
// 独立供应商的“推理地址”不是价格接口，本文件不会再把它们直接拼成
// /api/pricing。只有明确配置的New API兼容价格URL才会被读取。
//
// 主聚合网关可在价格URL与api_base_url同源时携带已有Bearer Key；
// 其它来源默认不携带业务密钥，避免把推理密钥发送到错误域名。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

const priceSyncGatewayBodyLimit = 8 * 1024 * 1024

var (
	ErrPriceSyncSourceUnavailable = errors.New("价格同步来源不可用")
	ErrPriceSyncResponseInvalid   = errors.New("价格同步响应无效")
)

type priceSyncSourceConfig struct {
	Source     string
	PricingURL string
	BearerKey  string
}

type newAPIVendor struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type newAPIEndpoint struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

type newAPIPricingEnvelope struct {
	Success           *bool                     `json:"success"`
	Message           string                    `json:"message"`
	Data              json.RawMessage           `json:"data"`
	Vendors           []newAPIVendor            `json:"vendors"`
	GroupRatio        map[string]float64        `json:"group_ratio"`
	SupportedEndpoint map[string]newAPIEndpoint `json:"supported_endpoint"`
	AutoGroups        []string                  `json:"auto_groups"`
}

type newAPINestedPricing struct {
	Data              []json.RawMessage         `json:"data"`
	Models            []json.RawMessage         `json:"models"`
	Items             []json.RawMessage         `json:"items"`
	Pricing           []json.RawMessage         `json:"pricing"`
	Vendors           []newAPIVendor            `json:"vendors"`
	GroupRatio        map[string]float64        `json:"group_ratio"`
	SupportedEndpoint map[string]newAPIEndpoint `json:"supported_endpoint"`
}

type newAPIPricingRow struct {
	ModelName              string   `json:"model_name"`
	EnableGroup            []string `json:"enable_group"`
	ModelRatio             float64  `json:"model_ratio"`
	CompletionRatio        float64  `json:"completion_ratio"`
	ModelPrice             float64  `json:"model_price"`
	QuotaType              int      `json:"quota_type"`
	Description            string   `json:"description"`
	VendorID               int      `json:"vendor_id"`
	SupportedEndpointTypes []int    `json:"supported_endpoint_types"`
}

// NewAPIPricingClient 管理价格接口读取。
type NewAPIPricingClient struct {
	aesKey string
}

// NewNewAPIPricingClient 创建价格客户端。
func NewNewAPIPricingClient(aesKey string) *NewAPIPricingClient {
	return &NewAPIPricingClient{aesKey: strings.TrimSpace(aesKey)}
}

// Fetch 根据价格来源读取并规范化价格目录。
func (client *NewAPIPricingClient) Fetch(
	ctx context.Context,
	source string,
) (*models.GatewayPriceCatalog, error) {
	sourceConfig, err := client.loadSourceConfig(source)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		sourceConfig.PricingURL,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("创建价格接口请求失败: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "TE-DNA-Price-Sync/1.0")
	if sourceConfig.BearerKey != "" {
		request.Header.Set("Authorization", "Bearer "+sourceConfig.BearerKey)
	}

	httpClient := &http.Client{Timeout: 30 * time.Second}
	response, err := httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: %s",
			ErrPriceSyncSourceUnavailable,
			err.Error(),
		)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(
		io.LimitReader(response.Body, priceSyncGatewayBodyLimit+1),
	)
	if err != nil {
		return nil, fmt.Errorf("读取价格接口响应失败: %w", err)
	}
	if len(body) > priceSyncGatewayBodyLimit {
		return nil, fmt.Errorf(
			"%w: 响应超过8MB限制",
			ErrPriceSyncResponseInvalid,
		)
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"%w: HTTP %d: %s",
			ErrPriceSyncSourceUnavailable,
			response.StatusCode,
			priceSyncSafeText(string(body), 300),
		)
	}

	return parseNewAPIPricingResponse(
		sourceConfig.Source,
		sourceConfig.PricingURL,
		body,
	)
}

func (client *NewAPIPricingClient) loadSourceConfig(
	source string,
) (*priceSyncSourceConfig, error) {
	source = strings.TrimSpace(source)
	if source == "" {
		source = models.PriceSyncSourceMainGateway
	}

	switch source {
	case models.PriceSyncSourceMainGateway:
		return client.loadMainGatewaySource()
	case models.PriceSyncSourceDomesticGateway:
		return loadExplicitPriceSource(
			source,
			"price_sync_domestic_pricing_url",
		)
	case models.PriceSyncSourceMediaGateway:
		return loadExplicitPriceSource(
			source,
			"price_sync_media_pricing_url",
		)
	case models.PriceSyncSourceTTSGateway:
		return loadExplicitPriceSource(
			source,
			"price_sync_tts_pricing_url",
		)
	default:
		return nil, fmt.Errorf(
			"%w: 未知来源%s",
			ErrPriceSyncSourceUnavailable,
			source,
		)
	}
}

func (client *NewAPIPricingClient) loadMainGatewaySource() (*priceSyncSourceConfig, error) {
	baseURL, err := repository.GetConfigValue("api_base_url")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(baseURL) == "" {
		return nil, fmt.Errorf(
			"%w: api_base_url未配置",
			ErrPriceSyncSourceUnavailable,
		)
	}

	explicitURL, err := repository.GetConfigValue(
		"price_sync_main_pricing_url",
	)
	if err != nil {
		return nil, err
	}

	pricingURL := strings.TrimSpace(explicitURL)
	if pricingURL == "" {
		rootURL, normalizeErr := normalizePriceSyncGatewayRoot(baseURL)
		if normalizeErr != nil {
			return nil, normalizeErr
		}
		pricingURL = strings.TrimRight(rootURL, "/") + "/api/pricing"
	} else {
		pricingURL, err = normalizePriceSyncPricingURL(pricingURL)
		if err != nil {
			return nil, err
		}
	}

	result := &priceSyncSourceConfig{
		Source:     models.PriceSyncSourceMainGateway,
		PricingURL: pricingURL,
	}

	// 只有价格URL与业务网关同源时才携带已有API Key。
	// 自定义到其它域名时保持匿名，避免泄露业务密钥。
	if !samePriceSyncOrigin(pricingURL, baseURL) {
		return result, nil
	}

	encryptedKey, keyErr := repository.GetConfigValue("api_key_enc")
	if keyErr != nil || encryptedKey == "" || client.aesKey == "" {
		return result, nil
	}

	plaintext, decryptErr := utils.DecryptAES(encryptedKey, client.aesKey)
	if decryptErr == nil {
		result.BearerKey = strings.TrimSpace(plaintext)
	}
	return result, nil
}

func loadExplicitPriceSource(
	source string,
	configKey string,
) (*priceSyncSourceConfig, error) {
	pricingURL, err := repository.GetConfigValue(configKey)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(pricingURL) == "" {
		return nil, fmt.Errorf(
			"%w: %s未配置；当前只支持New API兼容JSON价格接口",
			ErrPriceSyncSourceUnavailable,
			configKey,
		)
	}

	normalized, err := normalizePriceSyncPricingURL(pricingURL)
	if err != nil {
		return nil, err
	}
	return &priceSyncSourceConfig{
		Source:     source,
		PricingURL: normalized,
	}, nil
}

func normalizePriceSyncGatewayRoot(baseURL string) (string, error) {
	parsed, err := parsePriceSyncHTTPURL(baseURL)
	if err != nil {
		return "", err
	}

	path := strings.TrimRight(parsed.Path, "/")
	for _, suffix := range []string{
		"/compatible-mode/v1",
		"/openai/v1",
		"/api/v1",
		"/v1",
	} {
		if strings.HasSuffix(path, suffix) {
			path = strings.TrimSuffix(path, suffix)
			break
		}
	}

	parsed.Path = strings.TrimRight(path, "/")
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return strings.TrimRight(parsed.String(), "/"), nil
}

func normalizePriceSyncPricingURL(rawURL string) (string, error) {
	parsed, err := parsePriceSyncHTTPURL(rawURL)
	if err != nil {
		return "", err
	}
	parsed.Fragment = ""
	return parsed.String(), nil
}

func parsePriceSyncHTTPURL(rawURL string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil ||
		parsed.Scheme == "" ||
		parsed.Host == "" ||
		(parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, fmt.Errorf(
			"%w: 价格接口地址非法",
			ErrPriceSyncSourceUnavailable,
		)
	}
	return parsed, nil
}

func samePriceSyncOrigin(left string, right string) bool {
	leftURL, leftErr := parsePriceSyncHTTPURL(left)
	rightURL, rightErr := parsePriceSyncHTTPURL(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return strings.EqualFold(leftURL.Scheme, rightURL.Scheme) &&
		strings.EqualFold(leftURL.Host, rightURL.Host)
}

func parseNewAPIPricingResponse(
	source string,
	pricingURL string,
	body []byte,
) (*models.GatewayPriceCatalog, error) {
	var envelope newAPIPricingEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf(
			"%w: %s",
			ErrPriceSyncResponseInvalid,
			err.Error(),
		)
	}
	if envelope.Success != nil && !*envelope.Success {
		message := strings.TrimSpace(envelope.Message)
		if message == "" {
			message = "价格接口返回失败"
		}
		return nil, fmt.Errorf(
			"%w: %s",
			ErrPriceSyncResponseInvalid,
			message,
		)
	}

	rows := make([]json.RawMessage, 0)
	vendors := envelope.Vendors
	groupRatios := envelope.GroupRatio
	supportedEndpoints := envelope.SupportedEndpoint

	if len(envelope.Data) > 0 && string(envelope.Data) != "null" {
		if err := json.Unmarshal(envelope.Data, &rows); err != nil {
			var nested newAPINestedPricing
			if nestedErr := json.Unmarshal(envelope.Data, &nested); nestedErr != nil {
				return nil, fmt.Errorf(
					"%w: data字段结构无法识别",
					ErrPriceSyncResponseInvalid,
				)
			}
			switch {
			case len(nested.Data) > 0:
				rows = nested.Data
			case len(nested.Models) > 0:
				rows = nested.Models
			case len(nested.Items) > 0:
				rows = nested.Items
			default:
				rows = nested.Pricing
			}
			if len(vendors) == 0 {
				vendors = nested.Vendors
			}
			if len(groupRatios) == 0 {
				groupRatios = nested.GroupRatio
			}
			if len(supportedEndpoints) == 0 {
				supportedEndpoints = nested.SupportedEndpoint
			}
		}
	}

	if len(rows) == 0 {
		message := strings.TrimSpace(envelope.Message)
		if message == "" {
			message = "价格列表为空"
		}
		return nil, fmt.Errorf(
			"%w: %s",
			ErrPriceSyncResponseInvalid,
			message,
		)
	}

	vendorNames := make(map[int]string, len(vendors))
	for _, vendor := range vendors {
		vendorNames[vendor.ID] = strings.TrimSpace(vendor.Name)
	}

	prices := make([]models.GatewayPrice, 0, len(rows))
	for _, rawRow := range rows {
		var row newAPIPricingRow
		if err := json.Unmarshal(rawRow, &row); err != nil {
			continue
		}

		modelName := strings.TrimSpace(row.ModelName)
		if modelName == "" {
			continue
		}

		provider := vendorNames[row.VendorID]
		if provider == "" {
			provider = inferPriceSyncProvider(modelName)
		}

		endpointPaths := make([]string, 0)
		for _, endpointType := range row.SupportedEndpointTypes {
			endpoint, exists :=
				supportedEndpoints[strconv.Itoa(endpointType)]
			if !exists || strings.TrimSpace(endpoint.Path) == "" {
				continue
			}
			endpointPaths = append(
				endpointPaths,
				strings.TrimSpace(endpoint.Path),
			)
		}

		prices = append(prices, models.GatewayPrice{
			ModelName:              modelName,
			Provider:               strings.ToLower(strings.TrimSpace(provider)),
			Description:            strings.TrimSpace(row.Description),
			QuotaType:              row.QuotaType,
			ModelRatio:             row.ModelRatio,
			CompletionRatio:        row.CompletionRatio,
			ModelPriceUSD:          row.ModelPrice,
			EnableGroups:           append([]string(nil), row.EnableGroup...),
			SupportedEndpointPaths: endpointPaths,
			UnitPricesUSD:          extractGatewayUnitPrices(rawRow),
			RawPayload:             append(json.RawMessage(nil), rawRow...),
		})
	}

	if len(prices) == 0 {
		return nil, fmt.Errorf(
			"%w: 没有合法模型价格",
			ErrPriceSyncResponseInvalid,
		)
	}
	if groupRatios == nil {
		groupRatios = map[string]float64{}
	}

	return &models.GatewayPriceCatalog{
		Source:      source,
		PricingURL:  pricingURL,
		GroupRatios: groupRatios,
		Prices:      prices,
	}, nil
}

// extractGatewayUnitPrices读取可选扩展字段。
// 标准New API没有这些字段；媒体价格源可按相同外壳扩展它们。
func extractGatewayUnitPrices(raw json.RawMessage) map[string]float64 {
	var values map[string]interface{}
	if err := json.Unmarshal(raw, &values); err != nil {
		return map[string]float64{}
	}

	result := map[string]float64{}
	assign := func(unit string, keys ...string) {
		for _, key := range keys {
			value, ok := priceSyncNumber(values[key])
			if ok && value > 0 {
				result[unit] = value
				return
			}
		}
	}

	assign(models.MediaUnitImage, "price_per_image", "image_price")
	assign(models.MediaUnitRequest, "price_per_request", "request_price")
	assign(models.MediaUnitSecond, "price_per_second", "second_price")
	assign(
		models.MediaUnitAudioSecond,
		"price_per_audio_second",
		"audio_second_price",
	)
	assign(
		models.MediaUnitCharacter,
		"price_per_character",
		"character_price",
	)
	assign(
		models.MediaUnitProviderToken,
		"price_per_provider_token",
		"provider_token_price",
	)

	if value, ok := priceSyncNumber(values["price_per_1k_characters"]); ok && value > 0 {
		result[models.MediaUnitCharacter] = value / 1000
	}
	if value, ok := priceSyncNumber(values["price_per_10k_characters"]); ok && value > 0 {
		result[models.MediaUnitCharacter] = value / 10000
	}
	return result
}

func priceSyncNumber(value interface{}) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case json.Number:
		parsed, err := typed.Float64()
		return parsed, err == nil
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func inferPriceSyncProvider(modelName string) string {
	normalized := strings.ToLower(strings.TrimSpace(modelName))
	switch {
	case strings.Contains(normalized, "claude"),
		strings.Contains(normalized, "anthropic"):
		return "anthropic"
	case strings.Contains(normalized, "gemini"),
		strings.Contains(normalized, "google"):
		return "google"
	case strings.Contains(normalized, "qwen"),
		strings.Contains(normalized, "dashscope"):
		return "qwen"
	case strings.Contains(normalized, "glm"),
		strings.Contains(normalized, "zhipu"):
		return "zhipu"
	case strings.Contains(normalized, "doubao"),
		strings.Contains(normalized, "seedream"),
		strings.Contains(normalized, "seedance"),
		strings.Contains(normalized, "volc"):
		return "volcengine"
	default:
		return "unknown"
	}
}

func priceSyncSafeText(value string, maxRunes int) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes]) + "..."
}
