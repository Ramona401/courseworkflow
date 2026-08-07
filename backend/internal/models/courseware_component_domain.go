package models

// courseware_component_domain.go — 课件组件教育域协议扩展。
//
// 课件风格模板不属于本轮改造范围，因此本文件只扩展
// courseware_components资源的请求与响应协议，不给CoursewareTemplate
// 增加education_domain，也不改变模板的system/school/group/personal范围模型。
//
// 这里采用嵌入既有结构体的方式：
//   1. 保持现有CoursewareComponent和模板协议稳定；
//   2. 新版组件API显式返回education_domain；
//   3. 旧内部调用可以在后续批次逐步退出；
//   4. 避免模板模型被错误卷入组件教育域改造。

// CWComponentResource 是带教育域快照的课件组件完整记录。
//
// EducationDomain允许：
//   - k12
//   - vocational
//   - adult
//   - common
//
// mixed只表示跨域管理Actor，绝不能写入组件资源。
type CWComponentResource struct {
	*CoursewareComponent

	EducationDomain string `json:"education_domain"`
}

// CWComponentDomainListItem 是带教育域的组件列表项。
type CWComponentDomainListItem struct {
	*CWComponentListItem

	EducationDomain string `json:"education_domain"`
}

// CWComponentDomainListResponse 是新版域感知组件列表响应。
type CWComponentDomainListResponse struct {
	Components []*CWComponentDomainListItem `json:"components"`
	Total      int                          `json:"total"`
}

// MatchedCWComponentResource 是运行时匹配返回的域感知组件。
//
// 返回education_domain便于日志、测试和管理页面核验匹配结果；
// 前端不能据此自行决定授权，最终过滤始终在后端完成。
type MatchedCWComponentResource struct {
	*MatchedCWComponent

	EducationDomain string `json:"education_domain"`
}

// CreateCWComponentDomainRequest 创建课件组件请求。
//
// 普通教学Actor即使伪造EducationDomain也不会获得跨域能力；
// 当前组件写接口仍保持admin专属，mixed管理员必须显式选择资源域。
type CreateCWComponentDomainRequest struct {
	CreateCWComponentRequest

	EducationDomain string `json:"education_domain,omitempty"`
}

// MatchCWComponentsDomainRequest 域感知组件匹配请求。
//
// 普通教学Actor提交的EducationDomain会被忽略，始终使用可信Actor域；
// mixed管理Actor进行管理预览时必须明确选择k12、vocational或adult。
// common不能作为运行时当前教育域。
type MatchCWComponentsDomainRequest struct {
	MatchCWComponentsRequest

	EducationDomain string `json:"education_domain,omitempty"`
}
