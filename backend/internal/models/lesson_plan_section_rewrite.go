package models

// lesson_plan_section_rewrite.go — 教案目录与段落AI修改的数据协议。
//
// 设计原则：
//   1. 教案正文仍然以lesson_plans.content_markdown作为唯一事实源。
//   2. 目录和段落结构由代码确定性解析，不写入数据库，不调用AI生成目录。
//   3. AI修改采用“生成预览”和“确认应用”两阶段，生成预览绝不修改正文。
//   4. 确认应用必须同时校验教案版本与段落正文哈希，防止旧页面覆盖新正文。
//   5. 只替换标题下方的直属正文，不修改标题行，也不自动覆盖子标题结构。

// LessonPlanSectionLocator 是浏览器定位教案段落时提交的最小定位信息。
//
// HeadingText保存Markdown中的原始标题行去除首尾空白后的文本，
// 例如“## 教学目标”“一、教学过程”“**板书设计**”。
//
// Occurrence用于区分正文中重复出现的相同标题，从1开始计数。
// 后端始终重新解析数据库正式正文，不能直接相信浏览器提交的段落内容。
type LessonPlanSectionLocator struct {
	HeadingText string `json:"heading_text"`
	Occurrence  int    `json:"occurrence"`
}

// LessonPlanDocumentSection 是确定性解析得到的教案目录节点和可编辑正文范围。
//
// StartOffset、ContentStartOffset和EndOffset都是UTF-8字符串的字节偏移，
// 仅用于后端内部精确切片。浏览器不应自行提交这些偏移作为授权依据。
//
// BodyMarkdown只包含当前标题到下一个标题之间的直属正文，
// 不包含标题行，也不吞并后续子标题，避免一次修改意外覆盖整个章节结构。
type LessonPlanDocumentSection struct {
	ID                 string                   `json:"id"`
	Title              string                   `json:"title"`
	HeadingText        string                   `json:"heading_text"`
	Level              int                      `json:"level"`
	HeadingPath        []string                 `json:"heading_path"`
	Occurrence         int                      `json:"occurrence"`
	StartOffset        int                      `json:"-"`
	ContentStartOffset int                      `json:"-"`
	EndOffset          int                      `json:"-"`
	BodyMarkdown       string                   `json:"body_markdown"`
	SectionHash        string                   `json:"section_hash"`
	Locator            LessonPlanSectionLocator `json:"locator"`
}

// GenerateLessonPlanSectionRewriteRequest 请求AI生成某个教案段落的修改预览。
//
// BaseVersion必须等于数据库当前教案版本；版本不一致时必须刷新后重试。
// Instruction是老师本次修改要求，不能包含身份、学校或教育域等授权字段。
type GenerateLessonPlanSectionRewriteRequest struct {
	BaseVersion int                      `json:"base_version"`
	Locator     LessonPlanSectionLocator `json:"locator"`
	Instruction string                   `json:"instruction"`
}

// LessonPlanSectionRewritePreview 是AI生成完成后的预览结果。
//
// ReplacementMarkdown只包含可替换进当前标题下方的正文，
// 不含标题行。老师确认前不会写入数据库。
type LessonPlanSectionRewritePreview struct {
	BaseVersion        int                       `json:"base_version"`
	Section            LessonPlanDocumentSection `json:"section"`
	ReplacementMarkdown string                    `json:"replacement_markdown"`
}

// ApplyLessonPlanSectionRewriteRequest 是老师确认采用AI建议时的请求。
//
// SectionHash必须使用生成预览接口返回的服务端哈希，
// 不能由浏览器使用当前显示文本重新猜测。
type ApplyLessonPlanSectionRewriteRequest struct {
	BaseVersion        int                      `json:"base_version"`
	Locator            LessonPlanSectionLocator `json:"locator"`
	SectionHash        string                   `json:"section_hash"`
	ReplacementMarkdown string                   `json:"replacement_markdown"`
}

// LessonPlanSectionRewriteApplyResponse 是段落修改原子写入后的结果。
//
// Changed=false表示替换后的完整正文与数据库当前正文完全相同，
// 此时不会新增版本快照，也不会递增版本号。
type LessonPlanSectionRewriteApplyResponse struct {
	Changed         bool   `json:"changed"`
	CurrentVersion  int    `json:"current_version"`
	ContentMarkdown string `json:"content_markdown"`
}
