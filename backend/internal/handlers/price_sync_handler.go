package handlers

// price_sync_handler.go — 价格同步执行与历史查询HTTP处理器。
//
// 本文件负责：
//   - 拉取价格并生成不可变预览；
//   - 按管理员选择应用预览结果；
//   - 查询同步批次和同步明细历史。
//
// 全局配置和单模型同步配置位于price_sync_config_handler.go。
// 本模块不参与图片、视频或TTS业务调用及结算。

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// PriceSyncHandler 价格同步HTTP处理器。
type PriceSyncHandler struct {
	service *services.PriceSyncService
}

// NewPriceSyncHandler 创建价格同步处理器。
func NewPriceSyncHandler(
	service *services.PriceSyncService,
) *PriceSyncHandler {
	return &PriceSyncHandler{
		service: service,
	}
}

// Preview 拉取上游价格并生成不可变预览。
//
// POST /api/v1/tokens/price-sync/preview
//
// 空请求体等价于使用服务端当前配置。
// 手动HTTP入口始终覆盖trigger_type，禁止伪造调度器批次。
func (handler *PriceSyncHandler) Preview(
	writer http.ResponseWriter,
	request *http.Request,
) {
	if !handler.ensureService(writer) {
		return
	}

	var input models.PriceSyncPreviewRequest

	err := json.NewDecoder(request.Body).Decode(&input)
	if err != nil && !errors.Is(err, io.EOF) {
		utils.JSON(
			writer,
			http.StatusBadRequest,
			-1,
			"请求体解析失败",
			nil,
		)
		return
	}

	input.TriggerType = models.PriceSyncTriggerManual

	claims, ok := middleware.GetClaims(request.Context())
	if !ok || claims == nil {
		utils.JSON(
			writer,
			http.StatusUnauthorized,
			-1,
			"未认证",
			nil,
		)
		return
	}

	result, err := handler.service.Preview(
		request.Context(),
		&input,
		claims.UserID,
	)
	if err != nil {
		utils.JSON(
			writer,
			http.StatusBadGateway,
			-1,
			err.Error(),
			nil,
		)
		return
	}

	utils.JSON(
		writer,
		http.StatusOK,
		0,
		"价格同步预览已生成，尚未修改正式价格",
		result,
	)
}

// Apply 应用指定预览批次。
//
// POST /api/v1/tokens/price-sync/apply
//
// 必须明确指定apply_all=true，或者提交至少一个item_id。
// 禁止把空选择静默解释为应用全部。
func (handler *PriceSyncHandler) Apply(
	writer http.ResponseWriter,
	request *http.Request,
) {
	if !handler.ensureService(writer) {
		return
	}

	var input models.PriceSyncApplyRequest

	if err := json.NewDecoder(
		request.Body,
	).Decode(&input); err != nil {
		utils.JSON(
			writer,
			http.StatusBadRequest,
			-1,
			"请求体解析失败",
			nil,
		)
		return
	}

	input.RunID = strings.TrimSpace(input.RunID)

	if input.RunID == "" {
		utils.JSON(
			writer,
			http.StatusBadRequest,
			-1,
			"缺少价格同步批次ID",
			nil,
		)
		return
	}

	if !input.ApplyAll && len(input.ItemIDs) == 0 {
		utils.JSON(
			writer,
			http.StatusBadRequest,
			-1,
			"请选择至少一条价格变化，或明确设置apply_all=true",
			nil,
		)
		return
	}

	claims, ok := middleware.GetClaims(request.Context())
	if !ok || claims == nil {
		utils.JSON(
			writer,
			http.StatusUnauthorized,
			-1,
			"未认证",
			nil,
		)
		return
	}

	itemIDs := input.ItemIDs
	if input.ApplyAll {
		itemIDs = nil
	}

	result, err := handler.service.ApplySelected(
		request.Context(),
		input.RunID,
		itemIDs,
		claims.UserID,
	)

	switch {
	case errors.Is(
		err,
		repository.ErrPriceSyncRunNotFound,
	):
		utils.JSON(
			writer,
			http.StatusNotFound,
			-1,
			"价格同步批次不存在",
			nil,
		)
		return

	case errors.Is(
		err,
		repository.ErrPriceSyncRunNotApplicable,
	):
		utils.JSON(
			writer,
			http.StatusConflict,
			-1,
			"该价格同步批次已经应用或不可再次应用",
			nil,
		)
		return

	case errors.Is(
		err,
		repository.ErrPriceSyncItemNotFound,
	):
		utils.JSON(
			writer,
			http.StatusBadRequest,
			-1,
			"选择的价格同步明细不属于当前批次",
			nil,
		)
		return

	case err != nil:
		utils.JSON(
			writer,
			http.StatusBadRequest,
			-1,
			err.Error(),
			nil,
		)
		return
	}

	utils.JSON(
		writer,
		http.StatusOK,
		0,
		"价格同步批次已应用",
		result,
	)
}

// ListRuns 查询最近同步批次。
//
// GET /api/v1/tokens/price-sync/runs?limit=20
func (handler *PriceSyncHandler) ListRuns(
	writer http.ResponseWriter,
	request *http.Request,
) {
	if !handler.ensureService(writer) {
		return
	}

	limit := 20

	rawLimit := strings.TrimSpace(
		request.URL.Query().Get("limit"),
	)
	if rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed <= 0 {
			utils.JSON(
				writer,
				http.StatusBadRequest,
				-1,
				"limit必须为正整数",
				nil,
			)
			return
		}

		limit = parsed
	}

	items, err := handler.service.ListRuns(
		request.Context(),
		limit,
	)
	if err != nil {
		utils.JSON(
			writer,
			http.StatusInternalServerError,
			-1,
			err.Error(),
			nil,
		)
		return
	}

	utils.JSON(
		writer,
		http.StatusOK,
		0,
		"",
		items,
	)
}

// GetRunDetail 查询指定批次及全部明细。
//
// GET /api/v1/tokens/price-sync/runs/{id}
func (handler *PriceSyncHandler) GetRunDetail(
	writer http.ResponseWriter,
	request *http.Request,
) {
	if !handler.ensureService(writer) {
		return
	}

	runID := extractPriceSyncRunID(request.URL.Path)
	if runID == "" {
		utils.JSON(
			writer,
			http.StatusBadRequest,
			-1,
			"缺少价格同步批次ID",
			nil,
		)
		return
	}

	result, err := handler.service.GetRunDetail(
		request.Context(),
		runID,
	)

	if errors.Is(
		err,
		repository.ErrPriceSyncRunNotFound,
	) {
		utils.JSON(
			writer,
			http.StatusNotFound,
			-1,
			"价格同步批次不存在",
			nil,
		)
		return
	}

	if err != nil {
		utils.JSON(
			writer,
			http.StatusInternalServerError,
			-1,
			err.Error(),
			nil,
		)
		return
	}

	utils.JSON(
		writer,
		http.StatusOK,
		0,
		"",
		result,
	)
}

// ensureService 确保路由已经注入价格同步服务。
func (handler *PriceSyncHandler) ensureService(
	writer http.ResponseWriter,
) bool {
	if handler != nil && handler.service != nil {
		return true
	}

	utils.JSON(
		writer,
		http.StatusServiceUnavailable,
		-1,
		"价格同步服务未初始化",
		nil,
	)

	return false
}

// extractPriceSyncRunID 从批次详情路径提取唯一批次ID。
func extractPriceSyncRunID(path string) string {
	const prefix = "/api/v1/tokens/price-sync/runs/"

	if !strings.HasPrefix(path, prefix) {
		return ""
	}

	value := strings.Trim(
		strings.TrimSpace(
			strings.TrimPrefix(path, prefix),
		),
		"/",
	)

	if value == "" || strings.Contains(value, "/") {
		return ""
	}

	return value
}
