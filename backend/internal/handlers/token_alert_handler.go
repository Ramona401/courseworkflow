package handlers

// token_alert_handler.go — Token积分账户预警HTTP处理器
//
// 路由层已将读取和更新均限制为超级管理员。

import (
	"encoding/json"
	"net/http"

	"tedna/internal/models"
	"tedna/internal/utils"
)

// GetAlertConfig 获取账户预警配置。
func (h *TokenHandler) GetAlertConfig(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		utils.JSON(
			w,
			http.StatusMethodNotAllowed,
			-1,
			"仅支持GET请求",
			nil,
		)
		return
	}

	accountID := extractTokenMiddleID(
		r.URL.Path,
		"/alert-config",
	)
	if accountID == "" {
		utils.JSON(
			w,
			http.StatusBadRequest,
			-1,
			"缺少账户ID",
			nil,
		)
		return
	}

	config, err := h.tokenService.GetAlertConfig(
		r.Context(),
		accountID,
	)
	if err != nil {
		utils.JSON(
			w,
			http.StatusInternalServerError,
			-1,
			"查询失败",
			nil,
		)
		return
	}

	utils.JSON(
		w,
		http.StatusOK,
		0,
		"",
		config,
	)
}

// UpdateAlertConfig 更新账户预警配置。
func (h *TokenHandler) UpdateAlertConfig(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPut {
		utils.JSON(
			w,
			http.StatusMethodNotAllowed,
			-1,
			"仅支持PUT请求",
			nil,
		)
		return
	}

	accountID := extractTokenMiddleID(
		r.URL.Path,
		"/alert-config",
	)
	if accountID == "" {
		utils.JSON(
			w,
			http.StatusBadRequest,
			-1,
			"缺少账户ID",
			nil,
		)
		return
	}

	var request models.UpdateAlertConfigRequest

	if err := json.NewDecoder(
		r.Body,
	).Decode(
		&request,
	); err != nil {
		utils.JSON(
			w,
			http.StatusBadRequest,
			-1,
			"请求体解析失败",
			nil,
		)
		return
	}

	if err := h.tokenService.UpdateAlertConfig(
		r.Context(),
		accountID,
		&request,
	); err != nil {
		utils.JSON(
			w,
			http.StatusBadRequest,
			-1,
			err.Error(),
			nil,
		)
		return
	}

	utils.JSON(
		w,
		http.StatusOK,
		0,
		"更新成功",
		nil,
	)
}
