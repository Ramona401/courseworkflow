package models

// organization_education_domain.go — 组织教育域只读模型
//
// 本模块与Organization普通CRUD解耦：
//   - 既有组织接口负责名称、层级、管理员、Logo和门户板块；
//   - 本模块只负责查看组织创建时确定的教育域；
//   - 学校教育域创建后永久锁定；
//   - 区域教育域固定为mixed；
//   - 不再提供普通业务换域请求或换域结果DTO。

// OrganizationEducationDomainItem 组织教育域列表项。
type OrganizationEducationDomainItem struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Type            string  `json:"type"`
	ParentID        *string `json:"parent_id"`
	ParentName      string  `json:"parent_name"`
	EducationDomain string  `json:"education_domain"`
	Status          string  `json:"status"`
	GroupCount      int     `json:"group_count"`
	MemberCount     int     `json:"member_count"`
}

// OrganizationEducationDomainListResponse 组织教育域列表响应。
type OrganizationEducationDomainListResponse struct {
	Organizations []*OrganizationEducationDomainItem `json:"organizations"`
	Total         int                                `json:"total"`
}
