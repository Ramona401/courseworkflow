package handlers

// token_consumption_handler.go — Token消费流水HTTP处理器
//
// 超级管理员返回完整内部成本字段；
// 其它角色只返回积分和业务使用信息，响应中不序列化：
//   - cost_usd、exchange_rate、multiplier；
//   - provider、model_name、model_used；
//   - input_tokens、output_tokens、tokens_used。

import (
	"net/http"
	"strconv"

	"tedna/internal/utils"
)

// ListConsumptionLogs 查询消费流水。
func (h *TokenHandler) ListConsumptionLogs(
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

	query :=
		r.URL.Query()

	accountID :=
		query.Get("account_id")

	userID :=
		query.Get("user_id")

	sceneCode :=
		query.Get("scene_code")

	limit, _ :=
		strconv.Atoi(
			query.Get("limit"),
		)

	offset, _ :=
		strconv.Atoi(
			query.Get("offset"),
		)

	scope :=
		h.resolveScope(r)

	items, total, err :=
		h.tokenService.ListConsumptionLogsScoped(
			r.Context(),
			accountID,
			userID,
			sceneCode,
			scope,
			limit,
			offset,
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

	if tokenCanViewInternalCostDetails(r) {
		response :=
			map[string]interface{}{
				"items": items,
				"total": total,
			}

		if scope != nil &&
			scope.Blocked {
			response["scope_blocked"] = true
			response["scope_message"] =
				scope.BlockedReason
		}

		utils.JSON(
			w,
			http.StatusOK,
			0,
			"",
			response,
		)
		return
	}

	publicResponse :=
		&tokenPublicConsumptionListResponse{
			Items: buildTokenPublicConsumptionItems(
				items,
			),
			Total: total,
		}

	if scope != nil &&
		scope.Blocked {
		publicResponse.ScopeBlocked = true
		publicResponse.ScopeMessage =
			scope.BlockedReason
	}

	utils.JSON(
		w,
		http.StatusOK,
		0,
		"",
		publicResponse,
	)
}
