package models

// courseware_font.go — 课件字体方案数据模型（字体F1新建）
//
// 产品口径：
//   - 5套系统预设字体方案，一套 = 标题字体 + 正文字体的固定搭配（防乱配）
//   - 全部为 SIL OFL 开源协议字体，可自由商用、可随离线ZIP分发，零版权风险
//   - 硬编码常量不建表（无个人上传入口；将来开放个人字体再建表）
//   - 课件选中后只把方案code写入 coursewares.font_scheme 列；
//     注入页面的 @font-face 与字体规则随 html_content 持久化（HTML本身即快照）
//   - 字体文件自托管于 /uploads/courseware-assets/fonts/system/
//     （Nginx 已有公开 alias 映射 + 30天缓存，与背景图共用基础设施）
//
// 字体清单（均为单字重Regular，加粗由浏览器合成；Poppins另含真600字重）：
//   noto-sans-sc.woff2     思源黑体（Noto Sans SC）       OFL
//   noto-serif-sc.woff2    思源宋体（Noto Serif SC）      OFL
//   lxgw-wenkai.woff2      霞鹜文楷（LXGW WenKai）        OFL
//   smiley-sans.woff2      得意黑（Smiley Sans）          OFL
//   poppins-regular.woff2  Poppins 400（单层a几何无衬线） OFL
//   poppins-semibold.woff2 Poppins 600                    OFL

// CWFontFace 一条 @font-face 声明所需的数据
type CWFontFace struct {
	Family string `json:"family"` // CSS font-family 内部别名（如 TednaSans）
	File   string `json:"file"`   // woff2 文件名（位于系统字体目录下）
	Weight string `json:"weight"` // font-weight（"400"/"600"），空按400处理
}

// CWFontScheme 字体方案：标题字体 + 正文字体的成对搭配
// HeadingFamily/BodyFamily 为自有字体的 font-family 栈（含内部回退，如得意黑标题
// 回退思源黑体兜住生僻字），系统级中文兜底栈由服务层 buildFontCSS 统一追加。
type CWFontScheme struct {
	Code          string       `json:"code"`           // 方案code（存库值）
	Name          string       `json:"name"`           // 方案名（前端展示）
	Description   string       `json:"description"`    // 一句话描述
	HeadingLabel  string       `json:"heading_label"`  // 标题字体中文名
	BodyLabel     string       `json:"body_label"`     // 正文字体中文名
	HeadingFamily string       `json:"heading_family"` // 标题 font-family 栈
	BodyFamily    string       `json:"body_family"`    // 正文 font-family 栈
	Faces         []CWFontFace `json:"faces"`          // 需注入的 @font-face 列表（前端预览也用它现场加载）
}

// 六条 @font-face 基础数据（被各方案组合复用）
var (
	cwFaceSans   = CWFontFace{Family: "TednaSans", File: "noto-sans-sc.woff2", Weight: "400"}
	cwFaceSerif  = CWFontFace{Family: "TednaSerif", File: "noto-serif-sc.woff2", Weight: "400"}
	cwFaceKai    = CWFontFace{Family: "TednaKai", File: "lxgw-wenkai.woff2", Weight: "400"}
	cwFaceSmiley = CWFontFace{Family: "TednaSmiley", File: "smiley-sans.woff2", Weight: "400"}
	cwFacePop400 = CWFontFace{Family: "TednaPoppins", File: "poppins-regular.woff2", Weight: "400"}
	cwFacePop600 = CWFontFace{Family: "TednaPoppins", File: "poppins-semibold.woff2", Weight: "600"}
	cwFaceMarker = CWFontFace{Family: "TednaMarker", File: "lxgw-marker.woff2", Weight: "400"} // F4: 霞鹜漫黑 OFL
)

// CWFontSchemes 6套系统预设字体方案（顺序即前端展示顺序；F4新增漫趣马克。朱雀仿宋待其正式版后追加）
var CWFontSchemes = []*CWFontScheme{
	{
		Code: "modern", Name: "现代清爽",
		Description:  "思源黑体通铺，中性清晰，全学段通用首选",
		HeadingLabel: "思源黑体", BodyLabel: "思源黑体",
		HeadingFamily: "'TednaSans'", BodyFamily: "'TednaSans'",
		Faces: []CWFontFace{cwFaceSans},
	},
	{
		Code: "serif", Name: "书卷雅致",
		Description:  "思源宋体标题配思源黑体正文，书卷气，适合语文、历史与高学段",
		HeadingLabel: "思源宋体", BodyLabel: "思源黑体",
		HeadingFamily: "'TednaSerif'", BodyFamily: "'TednaSans'",
		Faces: []CWFontFace{cwFaceSerif, cwFaceSans},
	},
	{
		Code: "wenkai", Name: "温暖手写",
		Description:  "霞鹜文楷楷书手写感，温暖亲切，小学语文与低学段最佳",
		HeadingLabel: "霞鹜文楷", BodyLabel: "霞鹜文楷",
		HeadingFamily: "'TednaKai'", BodyFamily: "'TednaKai'",
		Faces: []CWFontFace{cwFaceKai},
	},
	{
		Code: "smiley", Name: "活力标题",
		Description:  "得意黑倾斜标题配思源黑体正文，活泼有冲击力，适合趣味课堂",
		HeadingLabel: "得意黑", BodyLabel: "思源黑体",
		HeadingFamily: "'TednaSmiley','TednaSans'", BodyFamily: "'TednaSans'",
		Faces: []CWFontFace{cwFaceSmiley, cwFaceSans},
	},
	{
		Code: "english", Name: "英语教学",
		Description:  "Poppins几何无衬线，小写a为单层手写字形（类Century Gothic），英语启蒙不教错字形；中文自动回退思源黑体",
		HeadingLabel: "Poppins", BodyLabel: "Poppins",
		HeadingFamily: "'TednaPoppins','TednaSans'", BodyFamily: "'TednaPoppins','TednaSans'",
		Faces: []CWFontFace{cwFacePop400, cwFacePop600, cwFaceSans},
	},
	{
		Code: "marker", Name: "漫趣马克",
		Description:  "霞鹜漫黑马克笔手绘感标题配思源黑体正文，童趣十足，适合低年级课堂",
		HeadingLabel: "霞鹜漫黑", BodyLabel: "思源黑体",
		HeadingFamily: "'TednaMarker','TednaSans'", BodyFamily: "'TednaSans'",
		Faces: []CWFontFace{cwFaceMarker, cwFaceSans},
	},
}

// LookupCWFontScheme 按code查找方案；不存在返回nil
func LookupCWFontScheme(code string) *CWFontScheme {
	for _, s := range CWFontSchemes {
		if s.Code == code {
			return s
		}
	}
	return nil
}

// SelectCoursewareFontRequest 课件选择/清除字体请求
//   - SchemeCode 非空：选用该方案（code写入课件 + 秒换全部已生成页）
//   - Clear=true：清除选择（列置空串 + 已生成页移除字体注入块，回退模板自带字体变量）
type SelectCoursewareFontRequest struct {
	SchemeCode string `json:"scheme_code"`
	Clear      bool   `json:"clear"`
}

// CoursewareFontSelection 课件当前字体选择（GET 返回；空串=未选）
type CoursewareFontSelection struct {
	FontScheme string `json:"font_scheme"`
}

// FontSelectionResult 选择/清除字体的执行结果
type FontSelectionResult struct {
	FontScheme   string `json:"font_scheme"`   // 生效后的方案code（清除后为空串）
	SwappedPages int    `json:"swapped_pages"` // 被秒换字体的已生成页数（零token字符串操作）
}
