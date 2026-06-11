package services

// courseware_font_service.go — 课件字体方案服务（字体F1新建）
//
// 完全复刻背景图库的「快照 + 确定性注入 + 零token秒换」架构：
//   1. 方案列表：5套系统预设常量（models.CWFontSchemes），不查库
//   2. 课件选择/清除字体：方案code写 coursewares.font_scheme + 秒换全部已生成页
//      —— 秒换 = 纯字符串替换带 TEDNA-TPL-FONT 标记的<style>块，零AI调用零token
//   3. 注入块内容 = @font-face声明（自托管woff2，OFL协议） + 全元素正文字体强制(!important)
//      + h1-h6标题字体覆盖 + code/pre等宽豁免 + --cw-font-* CSS变量覆盖（供var()引用方）
//   4. 生成链路注入：attachUserBackground 顺带把 font_scheme 挂进 tplInfo（见
//      courseware_background_service.go 字体F1插入段）；applyTemplateBackground 包装函数
//      （见 courseware_gen_helpers.go 末尾）让封面预览/批量/重生/微调所有路径统一过
//      applyFontInjection，确定性注入不依赖AI是否采纳
//   5. 离线ZIP：注入块内 url() 指向公网woff2，courseware_export_assets.go 既有的
//      <style>内url()扫描会自动把字体文件打包并改写为相对路径（批次F3验证）
//   6. 优先级（两级）：老师字体选择 > 模板自带--cw-font-*变量/AI自定。未选=不注入，零回归

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// CWFontBaseURL 系统字体文件公网基地址（导出供handler返回给前端做预览加载）
// 用绝对URL而非相对路径：离线ZIP导出与将来edu平台运行时均可正确取到字体文件
const CWFontBaseURL = "https://workflow.pkuailab.com/uploads/courseware-assets/fonts/system/"

// cwFontInjectedStyleRe 匹配已注入的字体<style>块（TEDNA-TPL-FONT 标记块）
// 注入的CSS内容不含 "<" 字符，[^<]* 可安全匹配到块尾；与背景块 TEDNA-TPL-BG 标记互不干扰
var cwFontInjectedStyleRe = regexp.MustCompile("(?s)<style>/\\* TEDNA-TPL-FONT[^<]*</style>")

// cwFontCJKFallback 系统级中文兜底字体栈（自有字体之后统一追加，未加载完成时的过渡显示）
const cwFontCJKFallback = ",'PingFang SC','Microsoft YaHei',sans-serif"

// buildFontCSS 把字体方案构建为完整注入CSS（@font-face + 强制规则）
func buildFontCSS(scheme *models.CWFontScheme) string {
	if scheme == nil {
		return ""
	}
	var sb strings.Builder
	// 1. @font-face 声明（font-display:swap：字体未下载完先用兜底字体显示，到货后换上）
	for _, f := range scheme.Faces {
		weight := f.Weight
		if weight == "" {
			weight = "400"
		}
		sb.WriteString(fmt.Sprintf(
			"@font-face{font-family:'%s';src:url('%s%s') format('woff2');font-weight:%s;font-style:normal;font-display:swap}",
			f.Family, CWFontBaseURL, f.File, weight))
	}
	heading := scheme.HeadingFamily + cwFontCJKFallback
	body := scheme.BodyFamily + cwFontCJKFallback
	// 2. 正文：全元素强制（!important 压过AI写的内联font-family与片段内<style>）
	sb.WriteString(".cw-page,.cw-page *{font-family:" + body + " !important}")
	// 3. 标题：h1-h6 用标题字体（写在正文规则之后，同优先级按后者生效）
	sb.WriteString(".cw-page h1,.cw-page h2,.cw-page h3,.cw-page h4,.cw-page h5,.cw-page h6{font-family:" + heading + " !important}")
	// 4. 代码块豁免：保持等宽字体（信息科技课代码示例不被破坏）
	sb.WriteString(".cw-page code,.cw-page pre,.cw-page code *,.cw-page pre *{font-family:ui-monospace,'Cascadia Code',Consolas,monospace !important}")
	// 5. CSS变量覆盖：使用 var(--cw-font-*) 的元素（含模板片段自带:root）也换上新字体。
	//    注入块在文档中位于片段<style>之前，须加 !important 才能赢下同选择器的后者
	sb.WriteString(":root{--cw-font-heading:" + heading + " !important;--cw-font-body:" + body + " !important}")
	return sb.String()
}

// buildFontStyleTag 构建注入用<style>标签；方案为nil返回空串（语义=移除字体块）
// 标记文本 TEDNA-TPL-FONT 与 cwFontInjectedStyleRe 配套，保证秒换/注入幂等互认
func buildFontStyleTag(scheme *models.CWFontScheme) string {
	css := buildFontCSS(scheme)
	if css == "" {
		return ""
	}
	return "<style>/* TEDNA-TPL-FONT 课件字体方案注入 */" + css + "</style>"
}

// swapInjectedFont 在已生成页面HTML上替换/注入/移除字体块（核心秒换原语，镜像 swapInjectedBackground）
//   - 页面已有 TEDNA-TPL-FONT 块：整块替换为新方案（scheme为nil=移除块）
//   - 页面没有该块且新方案非nil：走画布闸门后注入到根容器开标签之后
//
// 返回 (新HTML, 是否发生变化)
func swapInjectedFont(html string, scheme *models.CWFontScheme) (string, bool) {
	newTag := buildFontStyleTag(scheme)
	if cwFontInjectedStyleRe.MatchString(html) {
		out := cwFontInjectedStyleRe.ReplaceAllLiteralString(html, newTag)
		return out, out != html
	}
	if newTag == "" {
		return html, false
	}
	out := normalizeRootCanvas(html)
	trimmed := strings.TrimSpace(out)
	if !strings.HasPrefix(strings.ToLower(trimmed), "<div") {
		return html, false
	}
	gt := strings.Index(trimmed, ">")
	if gt < 0 {
		return html, false
	}
	return trimmed[:gt+1] + newTag + trimmed[gt+1:], true
}

// applyFontInjection 生成链路的字体确定性注入（挂CoursewareGenService，与背景注入同位面）
// 调用点：courseware_gen_helpers.go 末尾的 applyTemplateBackground 包装函数。
// 未选字体（FontSchemeCode为空）或已含标记块时原样返回，零回归。
func (s *CoursewareGenService) applyFontInjection(html string, tplInfo *cwTemplateInfo) string {
	if tplInfo == nil || tplInfo.FontSchemeCode == "" || strings.TrimSpace(html) == "" {
		return html
	}
	if strings.Contains(html, "TEDNA-TPL-FONT") {
		return html // 已注入过，幂等跳过
	}
	scheme := models.LookupCWFontScheme(tplInfo.FontSchemeCode)
	if scheme == nil {
		cwGenLog.Warn("课件存的字体方案code无效，跳过字体注入", "code", tplInfo.FontSchemeCode)
		return html
	}
	out, changed := swapInjectedFont(html, scheme)
	if changed {
		cwGenLog.Info("课件字体方案已注入", "scheme", scheme.Code)
	}
	return out
}

// ==================== 字体方案业务服务 ====================

// CoursewareFontService 课件字体方案服务（无状态，无外部依赖）
type CoursewareFontService struct{}

// NewCoursewareFontService 创建字体方案服务
func NewCoursewareFontService() *CoursewareFontService {
	return &CoursewareFontService{}
}

// ListSchemes 返回5套系统预设方案（常量，不查库）
func (s *CoursewareFontService) ListSchemes() []*models.CWFontScheme {
	return models.CWFontSchemes
}

// GetSelection 查询课件当前的字体选择
func (s *CoursewareFontService) GetSelection(ctx context.Context, coursewareID string) (*models.CoursewareFontSelection, error) {
	code, err := repository.GetCoursewareFontScheme(ctx, coursewareID)
	if err != nil {
		return nil, err
	}
	return &models.CoursewareFontSelection{FontScheme: code}, nil
}

// SelectFont 课件选用某字体方案：写code快照 + 秒换全部已生成页
func (s *CoursewareFontService) SelectFont(ctx context.Context, coursewareID string, userID string, schemeCode string) (*models.FontSelectionResult, error) {
	// 1. 课件归属与状态校验（口径与背景图库完全一致）
	cw, err := repository.GetCoursewareByID(ctx, coursewareID)
	if err != nil {
		return nil, fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != userID {
		return nil, fmt.Errorf("无权操作此课件")
	}
	if cw.Status == models.CoursewareStatusInPipeline {
		return nil, fmt.Errorf("已提交审核的课件不允许修改字体")
	}
	// 2. 方案校验（系统预设常量内查找）
	scheme := models.LookupCWFontScheme(schemeCode)
	if scheme == nil {
		return nil, fmt.Errorf("字体方案不存在: %s", schemeCode)
	}
	// 3. code快照写入课件列
	if err := repository.UpdateCoursewareFontScheme(ctx, coursewareID, scheme.Code); err != nil {
		return nil, err
	}
	// 4. 秒换全部已生成页
	swapped, swapErr := s.swapGeneratedPages(ctx, coursewareID, scheme)
	if swapErr != nil {
		cwGenLog.Warn("字体秒换部分失败（选择已生效，未换成功的页将在重生时获得新字体）", "error", swapErr, "courseware_id", coursewareID)
	}
	cwGenLog.Info("课件字体已选用", "courseware_id", coursewareID, "scheme", scheme.Name, "swapped_pages", swapped)
	return &models.FontSelectionResult{FontScheme: scheme.Code, SwappedPages: swapped}, nil
}

// ClearFont 清除课件字体选择：列置空串 + 已生成页移除字体注入块（回退模板自带字体）
func (s *CoursewareFontService) ClearFont(ctx context.Context, coursewareID string, userID string) (*models.FontSelectionResult, error) {
	cw, err := repository.GetCoursewareByID(ctx, coursewareID)
	if err != nil {
		return nil, fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != userID {
		return nil, fmt.Errorf("无权操作此课件")
	}
	if cw.Status == models.CoursewareStatusInPipeline {
		return nil, fmt.Errorf("已提交审核的课件不允许修改字体")
	}
	if err := repository.UpdateCoursewareFontScheme(ctx, coursewareID, ""); err != nil {
		return nil, err
	}
	// scheme传nil = 移除字体块（与背景"回退模板声明"不同：字体回退即页面自带的--cw-font-*变量，删块即恢复）
	swapped, swapErr := s.swapGeneratedPages(ctx, coursewareID, nil)
	if swapErr != nil {
		cwGenLog.Warn("字体清除秒换部分失败", "error", swapErr, "courseware_id", coursewareID)
	}
	cwGenLog.Info("课件字体已清除", "courseware_id", coursewareID, "swapped_pages", swapped)
	return &models.FontSelectionResult{FontScheme: "", SwappedPages: swapped}, nil
}

// swapGeneratedPages 对课件全部已生成HTML的页执行字体秒换；scheme为nil=移除字体块
func (s *CoursewareFontService) swapGeneratedPages(ctx context.Context, coursewareID string, scheme *models.CWFontScheme) (int, error) {
	pages, err := repository.ListCoursewarePages(ctx, coursewareID)
	if err != nil {
		return 0, err
	}
	swapped := 0
	var firstErr error
	for _, p := range pages {
		if strings.TrimSpace(p.HTMLContent) == "" {
			continue // 尚未生成的页：生成时经 applyFontInjection 按课件级选择注入
		}
		newHTML, changed := swapInjectedFont(p.HTMLContent, scheme)
		if !changed {
			continue
		}
		if uErr := repository.UpdateCWPageHTMLOnly(ctx, p.ID, newHTML); uErr != nil {
			if firstErr == nil {
				firstErr = uErr
			}
			continue
		}
		swapped++
	}
	return swapped, firstErr
}
