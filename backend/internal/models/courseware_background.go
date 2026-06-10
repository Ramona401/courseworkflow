package models

// courseware_background.go — 课件背景图库数据模型（批次1新建，批次3扩展生产入口）
//
// 对应表 courseware_background_sets：一集 = 头图(cover) + 内页图(content) 两张，
// scope 区分系统图库(system, admin维护)与个人图库(personal, 老师AI生成/上传产物)。
// 课件选中某集后，把两个公网URL快照复制进 coursewares.cover_bg_url / content_bg_url，
// 生成与秒换均只读快照列——集后续被删被改不影响已选课件。
//
// 批次3新增：AI生成图集请求 / 生产结果（含可选自动选中结果）/ archived状态 / 个人集上限常量。

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
	CWBgScopeSystem    = "system"
	CWBgScopePersonal  = "personal"
	CWBgStatusActive   = "active"
	CWBgStatusArchived = "archived" // 批次3：个人集删除=归档，不删OSS对象（已选课件URL快照不受影响）
)

// CWBgPersonalMaxSets 每人激活态个人背景图集上限（批次3：超出拒绝新建，提示先归档旧集）
const CWBgPersonalMaxSets = 20

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

// GenerateBackgroundSetRequest 批次3：AI生成一套背景请求
//   - CoursewareID 可选：非空时生成成功后自动为该课件选中此集（并秒换已生成页）
//   - CoverPrompt / ContentPrompt：封面/内页两张图的生成提示词（前端按课件主题+模板风格预填，可编辑）
//   - 内页提示词后端会强制追加"浅色低对比、适合做底纹"约束，保证内页可读性
type GenerateBackgroundSetRequest struct {
	CoursewareID  string `json:"courseware_id"`
	Name          string `json:"name"`
	CoverPrompt   string `json:"cover_prompt"`
	ContentPrompt string `json:"content_prompt"`
}

// BackgroundSetProduceResult 批次3：图集生产（AI生成/上传）结果
//   - Set：新建的个人图集
//   - Selection：带 courseware_id 时自动选中的执行结果；自动选中失败或未传课件时为nil
type BackgroundSetProduceResult struct {
	Set       *CoursewareBackgroundSet   `json:"set"`
	Selection *BackgroundSelectionResult `json:"selection,omitempty"`
}
