package handlers

// token_response_privacy.go — 积分中心财务成本响应隔离
//
// 财务成本、真实模型和供应商只向真正的超级管理员返回。
// TokenScope.IsAdmin仅代表admin角色，不能作为成本权限判断依据。

import (
        "net/http"
        "strings"
        "time"

        "tedna/internal/middleware"
        "tedna/internal/models"
)

type tokenPublicConsumptionItem struct {
        ID              string     `json:"id"`
        AccountName     string     `json:"account_name"`
        UserName        string     `json:"user_name"`
        Amount          float64    `json:"amount"`
        BalanceBefore   float64    `json:"balance_before"`
        BalanceAfter    float64    `json:"balance_after"`
        SceneCode       string     `json:"scene_code"`
        Memo            string     `json:"memo"`
        CreatedAt       *time.Time `json:"created_at"`
        CreditsConsumed float64    `json:"credits_consumed"`
        LatencyMs       int        `json:"latency_ms"`

        // 安全业务字段，不包含真实供应商、模型、Token或美元成本。
        BusinessName  string  `json:"business_name,omitempty"`
        MediaType     string  `json:"media_type,omitempty"`
        UsageQuantity float64 `json:"usage_quantity"`
        UsageUnit     string  `json:"usage_unit"`
}

type tokenPublicConsumptionListResponse struct {
	Items        []*tokenPublicConsumptionItem `json:"items"`
	Total        int                           `json:"total"`
	ScopeBlocked bool                          `json:"scope_blocked,omitempty"`
	ScopeMessage string                        `json:"scope_message,omitempty"`
}

type tokenPublicConsumptionSummaryRow struct {
	Key     string  `json:"key"`
	Label   string  `json:"label"`
	Credits float64 `json:"credits"`
	Calls   int     `json:"calls"`
	Percent float64 `json:"percent"`
}

type tokenPublicConsumptionSummaryResponse struct {
	Dimension    string                              `json:"dimension"`
	From         string                              `json:"from"`
	To           string                              `json:"to"`
	TotalCredits float64                             `json:"total_credits"`
	TotalCalls   int                                 `json:"total_calls"`
	Rows         []*tokenPublicConsumptionSummaryRow `json:"rows"`
	ScopeBlocked bool                                `json:"scope_blocked,omitempty"`
	ScopeMessage string                              `json:"scope_message,omitempty"`
}

func tokenCanViewInternalCostDetails(
	r *http.Request,
) bool {
	if r == nil {
		return false
	}

	claims, ok :=
		middleware.GetClaims(
			r.Context(),
		)

	return ok &&
		claims != nil &&
		claims.Role ==
			models.RoleAdmin &&
		claims.IsSuper
}

func buildTokenPublicConsumptionItems(
        items []*models.ConsumptionListItem,
) []*tokenPublicConsumptionItem {
        result :=
                make(
                        []*tokenPublicConsumptionItem,
                        0,
                        len(items),
                )

        for _, item :=
                range items {
                if item == nil {
                        continue
                }

                usageQuantity,
                        usageUnit :=
                        buildTokenPublicUsage(
                                item,
                        )

                result =
                        append(
                                result,
                                &tokenPublicConsumptionItem{
                                        ID:
                                                item.ID,
                                        AccountName:
                                                item.AccountName,
                                        UserName:
                                                item.UserName,
                                        Amount:
                                                item.Amount,
                                        BalanceBefore:
                                                item.BalanceBefore,
                                        BalanceAfter:
                                                item.BalanceAfter,
                                        SceneCode:
                                                item.SceneCode,
                                        Memo:
                                                item.Memo,
                                        CreatedAt:
                                                item.CreatedAt,
                                        CreditsConsumed:
                                                item.CreditsConsumed,
                                        LatencyMs:
                                                item.LatencyMs,
                                        BusinessName:
                                                strings.TrimSpace(
                                                        item.BillingNodeName,
                                                ),
                                        MediaType:
                                                strings.TrimSpace(
                                                        item.MediaType,
                                                ),
                                        UsageQuantity:
                                                usageQuantity,
                                        UsageUnit:
                                                usageUnit,
                                },
                        )
        }

        return result
}

// buildTokenPublicUsage 把内部媒体计量转换成普通用户可理解的业务用量。
//
// 视频内部按供应商token结算，但普通用户只看到“1个视频”，
// 不暴露供应商专用计量。
func buildTokenPublicUsage(
        item *models.ConsumptionListItem,
) (
        float64,
        string,
) {
        if item == nil {
                return 1,
                        "request"
        }

        quantity :=
                item.MediaQuantity

        switch strings.TrimSpace(
                item.MediaType,
        ) {
        case models.MediaTypeImage:
                if quantity <= 0 {
                        quantity = 1
                }

                return quantity,
                        "image"

        case models.MediaTypeVideo:
                return 1,
                        "video"

        case models.MediaTypeTTS:
                if quantity < 0 {
                        quantity = 0
                }

                return quantity,
                        "character"

        case models.MediaTypeASR:
                if quantity < 0 {
                        quantity = 0
                }

                return quantity,
                        "audio_second"

        default:
                return 1,
                        "request"
        }
}

func buildTokenPublicConsumptionSummary(
	response *models.ConsumptionSummaryResponse,
) *tokenPublicConsumptionSummaryResponse {
	publicResponse :=
		&tokenPublicConsumptionSummaryResponse{
			Rows: []*tokenPublicConsumptionSummaryRow{},
		}

	if response == nil {
		return publicResponse
	}

	publicResponse.Dimension =
		response.Dimension

	publicResponse.From =
		response.From

	publicResponse.To =
		response.To

	publicResponse.TotalCredits =
		response.TotalCredits

	publicResponse.TotalCalls =
		response.TotalCalls

	publicResponse.ScopeBlocked =
		response.ScopeBlocked

	publicResponse.ScopeMessage =
		response.ScopeMessage

	for _, row :=
		range response.Rows {
		if row == nil {
			continue
		}

		publicResponse.Rows =
			append(
				publicResponse.Rows,
				&tokenPublicConsumptionSummaryRow{
					Key:     row.Key,
					Label:   row.Label,
					Credits: row.Credits,
					Calls:   row.Calls,
					Percent: row.Percent,
				},
			)
	}

	return publicResponse
}
