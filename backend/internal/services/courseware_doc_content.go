package services

// courseware_doc_content.go — doc_upload 课件"上传文档原文"的导出取数封装
//
// 【为什么有这个文件】
//   老师把 Word 教案上传到课件工坊（source_type=doc_upload）后，工坊 Step4/5 的
//   "原教案对照抽屉"（LessonPlanRefDrawer）需要展示原文供对照。但 doc_upload 课件
//   没有关联的 lesson_plan 记录，原文以 .docx 文件形式存在 DocUploadDir 下；
//   而读取 docx 的 readDocxFullText 与目录常量 DocUploadDir 都在 services 包内
//   （前者未导出），handler 层跨包调不到。本文件提供一个导出的薄封装，
//   供 courseware_index_handler.GetLessonPlanContent 的 doc_upload 分支调用。
//
// 【与 loadLessonPlanContextForGen 的区别】
//   - loadLessonPlanContextForGen 服务于"逐页生成注入"，带 8000 rune 截断且规整缓存优先；
//   - 本函数服务于"人看的对照抽屉"，返回 docx 完整原文、不截断、不走规整缓存——
//     老师对照时要看的是自己上传的原文，而非 AI 规整后的版本。
//
// 【安全边界】
//   best-effort：非 doc_upload 来源 / 路径为空 / 文件缺失 / 解析失败 / 内容为空，
//   一律返回空串，调用方据此落回"无对照内容"的原有行为，零回归、绝不报错中断。

import (
	"path/filepath"
	"strings"

	"tedna/internal/models"
)

// ExtractDocUploadFullText 读取 doc_upload 课件所上传的 docx 完整原文（纯文本）。
//
// 取数路径与 courseware_gen_lesson_context.go / courseware_index_refine.go 完全一致：
//   filepath.Join(DocUploadDir, cw.SourceFilePath) → readDocxFullText 标准库解析。
//
// 返回空串的所有情形（调用方据此判定"无原文可展示"）：
//   - cw 为 nil / 来源不是 doc_upload / SourceFilePath 为空；
//   - docx 文件缺失、解析失败、解析结果为空白。
func ExtractDocUploadFullText(cw *models.Courseware) string {
	if cw == nil || cw.SourceType != models.CWSourceDocUpload {
		return ""
	}
	if strings.TrimSpace(cw.SourceFilePath) == "" {
		return ""
	}
	docFullPath := filepath.Join(DocUploadDir, cw.SourceFilePath)
	text, err := readDocxFullText(docFullPath)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(text)
}
