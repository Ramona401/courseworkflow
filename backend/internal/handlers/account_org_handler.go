package handlers

/*
 * account_org_handler.go — 个人中心"我的组织归属"处理器
 *
 * 提供：
 *   GET /api/v1/account/organization  查询当前登录用户的完整组织归属
 *
 * 用途（测试反馈 7-1 #2）：
 *   让每个用户在个人中心看到自己所属的区域 / 学校 / 教研组，以及在各教研组里的
 *   职位角色（组长 / 骨干 / 普通成员），解决"平台用户看不到自己的角色、对权限层级茫然"。
 *
 * 纯只读、登录即可、只查自己（userID 取自 JWT，不接受前端传入他人ID），无越权风险。
 * 数据由 repository.GetUserOrganizationProfile 聚合，本 handler 仅做鉴权与响应封装。
 *
 * 独立成文件，不改动 account_handler.go 的既有职责。
 */

import (
	"net/http"

	"tedna/internal/middleware"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// AccountOrgHandler 个人组织归属处理器（无依赖，纯查询）
type AccountOrgHandler struct{}

// NewAccountOrgHandler 创建实例
func NewAccountOrgHandler() *AccountOrgHandler {
	return &AccountOrgHandler{}
}

// GetMyOrganization GET /api/v1/account/organization
//
// 返回当前登录用户的组织归属聚合结果：
//   { "schools": [...], "groups": [...] }
// schools = 直接归属的学校（含区域、是否本校管理员）
// groups  = 所在教研组（含我在各组的角色 member/backbone/lead）
func (h *AccountOrgHandler) GetMyOrganization(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}

	// 取当前登录用户（只查自己，userID 来自 JWT）
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, utils.MsgUnauthorized)
		return
	}

	profile, err := repository.GetUserOrganizationProfile(r.Context(), claims.UserID)
	if err != nil {
		utils.InternalError(w, "获取组织归属信息失败")
		return
	}

	utils.Success(w, profile)
}
