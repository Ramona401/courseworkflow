package models

// courseware_background.go — 课件背景图库数据模型（批次1新建）
//
// 对应表 courseware_background_sets：一集 = 头图(cover) + 内页图(content) 两张，
// scope 区分系统图库(system, admin维护)与个人图库(personal, 老师AI生成/上传产物)。
// 课件选中某集后，把两个公网URL快照复制进 coursewares.cover_bg_url / content_bg_url，
// 生成与秒换均只读快照列——集后续被删被改不影响已选课件。

import "time"

// CoursewareBackgroundSet 背景图集（对应 courseware_background_sets 表）
type CoursewareBackgroundSet struct {
	ID               string     `json:"id"`
	Name             string     `json:"name"`              // 集名称
	Description      string     `json:"description"`       // 描述
	StyleCategory    string     `json:"style_category"`    // 风格类别（可选，供前端筛选）
	Scope            string     `json:"scope"`             // system=系统图库 / personal=个人
	UserID           *string    `json:"user_id"`           // personal集归属人；system为nil
	CoverOssURL      string     `json:"cover_oss_url"`     // 头图本地路径（/uploads/，可空）
	CoverPublicURL   string     `json:"cover_public_url"`  // 头图OSS公网URL（注入用）
	ContentOssURL    string     `json:"content_oss_url"`   // 内页图本地路径（可空）
	ContentPublicURL string     `json:"content_public_url"`// 内页图OSS公网URL（注入用）
	Status           string     `json:"status"`            // active/archived
	SortOrder        int        `json:"sort_order"`
	CreatedAt        *time.Time `json:"created_at"`
	UpdatedAt        *time.Time `json:"updated_at"`
}

// 背景图集 scope / status 常量
const (
	CWBgScopeSystem   = "system"
	CWBgScopePersonal = "personal"
	CWBgStatusActive  = "active"
)

// SelectCoursewareBackgroundRequest 课件选择/清除背景请求
//   - SetID 非空：选用该图集（两URL快照写入课件 + 秒换全部已生成页）
//   - Clear=true：清除选择（两列置NULL + 已生成页回退到模板自带背景或无背景）
type SelectCoursewareBackgroundRequest struct {
	SetID string `json:"set_id"`
	Clear bool   `json:"clear"`
}

// CoursewareBackgroundSelection 课件当前背景选择（GET 返回）
type CoursewareBackgroundSelection struct {
	CoverBgURL   string `json:"cover_bg_url"`
	ContentBgURL string `json:"content_bg_url"`
}

// BackgroundSelectionResult 选择/清除背景的执行结果
type BackgroundSelectionResult struct {
	CoverBgURL   string `json:"cover_bg_url"`   // 生效后的头图URL（清除后为空串）
	ContentBgURL string `json:"content_bg_url"` // 生效后的内页图URL
	SwappedPages int    `json:"swapped_pages"`  // 被秒换背景的已生成页数（零token字符串操作）
}
