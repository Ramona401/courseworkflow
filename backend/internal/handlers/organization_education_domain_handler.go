package handlers

// organization_education_domain_handler.go — 组织教育域只读HTTP处理器
//
// 路由：
//   GET /api/v1/admin/organization-education-domains
//       admin可查看全部组织创建时确定的教育域；
//
//   PUT /api/v1/admin/organization-education-domains/{id}
//       保留旧路由兼容性；
//       真实组织统一返回409 Conflict；
//       不解析请求中的education_domain；
//       不调用更新Repository；
//       不写审计日志。
//
// 学校教育域不可变的最终保护同时位于数据库触发器。

import (
	"errors"
	"net/http"
	"strings"

	"tedna/internal/repository"
	"tedna/internal/services"
	"tedna/internal/utils"
)

type OrganizationEducationDomainHandler struct {
	service *services.OrganizationEducationDomainService
}

func NewOrganizationEducationDomainHandler(
	service *services.OrganizationEducationDomainService,
) *OrganizationEducationDomainHandler {
	return &OrganizationEducationDomainHandler{
		service: service,
	}
}

// List GET /api/v1/admin/organization-education-domains。
func (h *OrganizationEducationDomainHandler) List(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			utils.MsgMethodGetOnly,
		)
		return
	}

	result, err := h.service.ListOrganizations(r.Context())
	if err != nil {
		utils.InternalError(w, "查询组织教育域失败: "+err.Error())
		return
	}

	utils.Success(w, result)
}

// Update PUT /api/v1/admin/organization-education-domains/{id}。
//
// 此方法只作为旧客户端的明确兼容响应，不执行更新。
func (h *OrganizationEducationDomainHandler) Update(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPut {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			utils.MsgMethodPutOnly,
		)
		return
	}

	organizationID := extractOrganizationEducationDomainID(
		r.URL.Path,
	)
	if organizationID == "" {
		utils.BadRequest(w, "缺少组织ID")
		return
	}

	err := h.service.RejectUpdate(
		r.Context(),
		organizationID,
	)
	if err != nil {
		switch {
		case errors.Is(
			err,
			repository.ErrOrganizationEducationDomainNotFound,
		):
			utils.Fail(w, http.StatusNotFound, "组织不存在")

		case errors.Is(
			err,
			services.ErrOrganizationDomainImmutable,
		):
			utils.Fail(
				w,
				http.StatusConflict,
				"组织教育域在创建后不可修改",
			)

		default:
			utils.InternalError(
				w,
				"校验组织教育域失败: "+err.Error(),
			)
		}
		return
	}

	// RejectUpdate对合法组织必须返回不可变错误。
	// 如果未来实现意外返回nil，仍然以409 fail-closed，绝不放行修改。
	utils.Fail(
		w,
		http.StatusConflict,
		"组织教育域在创建后不可修改",
	)
}

func extractOrganizationEducationDomainID(path string) string {
	const prefix = "/api/v1/admin/organization-education-domains/"

	if !strings.HasPrefix(path, prefix) {
		return ""
	}

	id := strings.TrimPrefix(path, prefix)
	id = strings.Trim(id, "/")

	if id == "" || strings.Contains(id, "/") {
		return ""
	}

	return id
}
