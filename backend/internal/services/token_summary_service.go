package services

// token_summary_service.go — Token积分消费汇总报告业务逻辑
//
// 积分消费汇总报告 batch：
//   独立文件承载汇总业务逻辑，方法挂在 *TokenService 接收器上。
//
// 职责：
//   1. 接收 handler 解析好的维度/时间范围/下钻过滤，结合 TokenScope 决定 owner/user 白名单。
//   2. 调 repository.GetConsumptionSummary 做聚合。
//   3. scene 维度：把原始 scene_code 用 models.SceneNameMap 翻译为中文（取不到回退原码）。
//   4. 组装 ConsumptionSummaryResponse 返回。
//
// scope 收窄口径（与 ① region_admin 分配同一套 TokenScope，零越权）：
//   - admin           → OwnerIDs/UserIDs 皆 nil → 看全部
//   - region_admin    → UserIDs=辖区学校成员(剔除特权)，OwnerIDs=辖区学校owner → 看辖区
//   - senior_operator → UserIDs=本校成员，OwnerIDs={本校成员∪学校组织ID} → 看本校
//   - operator/viewer → UserIDs/OwnerIDs=[自己] → 只看自己
//   scope.Blocked（未绑校等）→ 返回空 rows + ScopeBlocked 提示。
//
// region/school 维度都按 OwnerIDs 收窄（repo 内分别取 grandparent/parent 的 owner）。
//
// rows:null→[] 修复（本次）：repo 返回 nil slice 时（越权/无数据），确保 resp.Rows 为
//   非 nil 空切片，避免前端 .map(null) 崩溃。

import (
	"context"
	"time"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// ConsumptionSummaryInput 汇总请求输入（handler 解析 query 后组装传入）
type ConsumptionSummaryInput struct {
	Dimension    string    // region/school/user/model/scene/time
	From         time.Time // 时间范围起（零值=不限）
	To           time.Time // 时间范围止（零值=不限，已由 handler 加到次日0点）
	UserFilter   string    // 下钻：单个 user_id（handler 已校验在 scope 内）
	SchoolMember []string  // 下钻：某学校成员 user_id 列表（handler 已确认在 scope 内）
}

// GetConsumptionSummary 获取消费汇总报告（范围感知 + scene 中文翻译）
func (s *TokenService) GetConsumptionSummary(ctx context.Context, in *ConsumptionSummaryInput, scope *TokenScope) (*models.ConsumptionSummaryResponse, error) {
	resp := &models.ConsumptionSummaryResponse{
		Dimension: in.Dimension,
		Rows:      []*models.ConsumptionSummaryRow{},
	}
	if !in.From.IsZero() {
		resp.From = in.From.Format("2006-01-02")
	}
	if !in.To.IsZero() {
		resp.To = in.To.AddDate(0, 0, -1).Format("2006-01-02")
	}

	// scope 被收窄为空集（如 senior 未绑校）→ 直接返回空 + 提示，不查库
	if scope != nil && scope.Blocked {
		resp.ScopeBlocked = true
		resp.ScopeMessage = scope.BlockedReason
		return resp, nil
	}

	// 组装 repo 参数：region/school 维度按 owner 收窄，其余维度按 user_id 收窄
	params := &repository.ConsumptionSummaryParams{
		Dimension:    in.Dimension,
		From:         in.From,
		To:           in.To,
		UserFilter:   in.UserFilter,
		SchoolMember: in.SchoolMember,
	}
	if scope != nil {
		if in.Dimension == models.SummaryDimRegion || in.Dimension == models.SummaryDimSchool {
			params.OwnerIDs = scope.OwnerIDs // region/school 维度按 owner 收窄
		} else {
			params.UserIDs = scope.UserIDs // 其余维度按 user_id 收窄
		}
	}

	rows, totalCredits, totalCostUSD, totalCalls, err := repository.GetConsumptionSummary(ctx, params)
	if err != nil {
		return nil, err
	}

	// scene 维度：把 scene_code 翻译为中文（取不到回退原码，如 courseware_media_prompt）
	if in.Dimension == models.SummaryDimScene {
		for _, row := range rows {
			if cn, ok := models.SceneNameMap[row.Key]; ok && cn != "" {
				row.Label = cn
			}
		}
	}

	// rows:null→[] 修复：repo 无数据时返回 nil slice，确保 resp.Rows 为非 nil 空切片
	if rows == nil {
		rows = []*models.ConsumptionSummaryRow{}
	}

	resp.Rows = rows
	resp.TotalCredits = totalCredits
	resp.TotalCostUSD = totalCostUSD
	resp.TotalCalls = totalCalls
	return resp, nil
}
