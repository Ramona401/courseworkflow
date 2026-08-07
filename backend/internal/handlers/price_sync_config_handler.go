package handlers

// price_sync_config_handler.go — 价格同步全局配置与目标配置接口。
//
// 本文件只管理：
//   - 定时同步总开关、自动应用开关及同步周期；
//   - 四类价格接口地址；
//   - 单个模型的价格来源、上游模型名和自动同步开关。
//
// 配置更新不直接修改正式价格。
// 正式价格只能经过预览及应用流程变更。

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// GetSettings 返回全局配置及全部同步目标。
//
// GET /api/v1/tokens/price-sync/settings
func (handler *PriceSyncHandler) GetSettings(
	writer http.ResponseWriter,
	request *http.Request,
) {
	if !handler.ensureService(writer) {
		return
	}

	result, err := handler.service.GetManagementState(
		request.Context(),
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
		result,
	)
}

// UpdateSettings 更新全局价格同步配置。
//
// PUT /api/v1/tokens/price-sync/settings
func (handler *PriceSyncHandler) UpdateSettings(
	writer http.ResponseWriter,
	request *http.Request,
) {
	if !handler.ensureService(writer) {
		return
	}

	var input models.UpdatePriceSyncSettingsRequest

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

	result, err := handler.service.UpdateSettings(
		request.Context(),
		&input,
		claims.UserID,
	)
	if err != nil {
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
		"价格同步配置已更新",
		result,
	)
}

// UpdateTarget 更新单个价格同步目标。
//
// PUT /api/v1/tokens/price-sync/targets/{kind}/{id}
func (handler *PriceSyncHandler) UpdateTarget(
	writer http.ResponseWriter,
	request *http.Request,
) {
	if !handler.ensureService(writer) {
		return
	}

	targetKind, targetID :=
		extractPriceSyncTargetPath(request.URL.Path)

	if targetKind == "" || targetID == "" {
		utils.JSON(
			writer,
			http.StatusBadRequest,
			-1,
			"价格同步目标路径无效",
			nil,
		)
		return
	}

	var input models.UpdatePriceSyncTargetRequest

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

	result, err := handler.service.UpdateTarget(
		request.Context(),
		targetKind,
		targetID,
		&input,
		claims.UserID,
	)

	if errors.Is(
		err,
		repository.ErrPriceSyncTargetNotFound,
	) {
		utils.JSON(
			writer,
			http.StatusNotFound,
			-1,
			"价格同步目标不存在",
			nil,
		)
		return
	}

	if err != nil {
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
		"价格同步目标已更新",
		result,
	)
}

// extractPriceSyncTargetPath 从目标配置路径读取类型和ID。
func extractPriceSyncTargetPath(
	path string,
) (string, string) {
	const prefix = "/api/v1/tokens/price-sync/targets/"

	if !strings.HasPrefix(path, prefix) {
		return "", ""
	}

	value := strings.Trim(
		strings.TrimSpace(
			strings.TrimPrefix(path, prefix),
		),
		"/",
	)

	parts := strings.Split(value, "/")
	if len(parts) != 2 {
		return "", ""
	}

	targetKind := strings.TrimSpace(parts[0])
	targetID := strings.TrimSpace(parts[1])

	if targetKind != models.PriceSyncTargetText &&
		targetKind != models.PriceSyncTargetMedia {
		return "", ""
	}

	if targetID == "" {
		return "", ""
	}

	return targetKind, targetID
}
