package services

// courseware_background_service.go — 课件背景图库服务（批次1新建）
//
// 职责：
//   1. 图集列表（系统+个人）
//   2. 课件选择/清除背景：写URL快照两列 + 【秒换】全部已生成页的注入背景
//   3. 用户背景声明构建（统一蒙版方案）：封面=左浓右透方向蒙版；内页=奶白平铺蒙版
//   4. 三级优先级解析 resolveUserBgDecls + 生成上下文挂载 attachUserBackground
//
// ========== 双 cw-page 封面"换背景后文字全消失"根因修复（本次）==========
// 真凶：老封面页HTML是「导航栏div(第一个) + 内容div(第二个)」两个 class="cw-page"。
//   换背景路径B原本调 normalizeRootCanvas(html)，而该函数只认"第一个<div>"当根容器，
//   用 enforceCanvasDecls 把它的 style 强改成 width:1920px;height:1080px。
//   导航栏div本是顶栏(height≈64px)，被强撑成 height:1080px；它又带
//   position:absolute + z-index:100 + 不透明背景 var(--cw-bg)，于是变成
//   一整块满屏遮罩，把下面内容div的所有正文(标题/卡片/图)全盖住 → "内容都在却完全看不见"。
//
// 修复（最小、不动蒙版/字体/插图/数据形态）：
//   swapInjectedBackground 路径B——若HTML含 <!-- NAV_END --> (双cw-page结构)，
//   【跳过 normalizeRootCanvas】，直接把<style>注入到第一个div开标签之后即可：
//     · 不再强改任何div尺寸，导航栏保持原顶栏高度，不再变遮罩；
//     · <style>注入位置无所谓(CSS全局生效)，选择器 .cw-page:last-of-type 精准命中内容div。
//   单cw-page普通页维持原逻辑(过 normalizeRootCanvas 规范根容器)，零回归。
//   选择器统一为 .cw-page:last-of-type：双cw-page页只命中内容div(放过导航栏)，
//   单页first=last命中唯一div，与原 .cw-page{} 效果一致。

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"tedna/internal/config"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== 用户背景声明构建（统一蒙版，已实测可读参数） ====================

// buildCoverBgDecls 构建封面背景声明：方向性可读性蒙版（左浓右透）+ 用户头图
func buildCoverBgDecls(url string) string {
	return buildCoverBgDeclsWithOpacity(url, -1)
}

// buildCoverBgDeclsWithOpacity 带透明度参数的封面背景声明构建
// opacity < 0 或 > 1 按默认四档渐变；opacity == 0 无蒙版纯背景图；否则四档按比例缩放
func buildCoverBgDeclsWithOpacity(url string, opacity float64) string {
	if opacity < 0 || opacity > 1 {
		return "background:linear-gradient(105deg, rgba(255,253,248,0.94) 0%, rgba(255,253,248,0.80) 32%, rgba(255,253,248,0.34) 62%, rgba(255,253,248,0.08) 100%), url('" + url + "') center/cover no-repeat"
	}
	if opacity == 0 {
		return "background:url('" + url + "') center/cover no-repeat"
	}
	a0 := fmt.Sprintf("%.2f", 0.94*opacity)
	a1 := fmt.Sprintf("%.2f", 0.80*opacity)
	a2 := fmt.Sprintf("%.2f", 0.34*opacity)
	a3 := fmt.Sprintf("%.2f", 0.08*opacity)
	return "background:linear-gradient(105deg, rgba(255,253,248," + a0 + ") 0%, rgba(255,253,248," + a1 + ") 32%, rgba(255,253,248," + a2 + ") 62%, rgba(255,253,248," + a3 + ") 100%), url('" + url + "') center/cover no-repeat"
}

// buildContentBgDecls 构建内页背景声明：奶白半透明平铺蒙版 + 用户内页图（压成隐约肌理）
func buildContentBgDecls(url string) string {
	return buildContentBgDeclsWithOpacity(url, -1)
}

// buildContentBgDeclsWithOpacity 带透明度参数的内页背景声明构建
// opacity < 0 按默认0.86；opacity == 0 无蒙版纯背景图；否则用指定值
func buildContentBgDeclsWithOpacity(url string, opacity float64) string {
	if opacity < 0 {
		opacity = 0.86
	}
	if opacity == 0 {
		return "background:url('" + url + "') center/cover no-repeat"
	}
	a := fmt.Sprintf("%.2f", opacity)
	return "background:linear-gradient(rgba(255,253,250," + a + "), rgba(255,253,250," + a + ")), url('" + url + "') center/cover no-repeat"
}

// resolveUserBgDecls 三级优先级的第一级：课件级用户选择的背景（无则返回空串交给下一级）
func resolveUserBgDecls(tplInfo *cwTemplateInfo, pageNum int) string {
	if tplInfo == nil {
		return ""
	}
	if pageNum == 1 && tplInfo.CoverBgURL != "" {
		return buildCoverBgDecls(tplInfo.CoverBgURL)
	}
	if pageNum > 1 && tplInfo.ContentBgURL != "" {
		return buildContentBgDecls(tplInfo.ContentBgURL)
	}
	return ""
}

// resolvePageBgDecls 页级背景覆盖解析——优先级最高，覆盖课件级与模板级
// 如果该页设了专属背景(page_bg_url非空)或自定义蒙版(page_bg_mode!="default")，
// 按页级设置构建声明替代课件级，返回非空串；否则返回空串交给课件级/模板级。
func resolvePageBgDecls(pageBg *repository.PageBgSetting, pageNum int, courseLevelDecls string) string {
        if pageBg == nil {
                return ""
        }
        // 页级有专属背景图
        if pageBg.PageBgURL != "" {
                opacity := float64(-1) // 默认
                if pageBg.PageBgMode == "none" {
                        opacity = 0
                } else if pageBg.PageBgMode == "custom" && pageBg.PageBgOpacity != nil {
                        opacity = *pageBg.PageBgOpacity
                }
                if pageNum == 1 {
                        return buildCoverBgDeclsWithOpacity(pageBg.PageBgURL, opacity)
                }
                return buildContentBgDeclsWithOpacity(pageBg.PageBgURL, opacity)
        }
        // 页级没有专属图但改了蒙版模式（用课件级的图+页级的透明度）
        if pageBg.PageBgMode == "none" || pageBg.PageBgMode == "custom" {
                // 需要从课件级声明中提取背景图URL
                bgURL := extractBgURLFromDecls(courseLevelDecls)
                if bgURL == "" {
                        return "" // 课件级也没有图，交给模板级
                }
                opacity := float64(-1)
                if pageBg.PageBgMode == "none" {
                        opacity = 0
                } else if pageBg.PageBgOpacity != nil {
                        opacity = *pageBg.PageBgOpacity
                }
                if pageNum == 1 {
                        return buildCoverBgDeclsWithOpacity(bgURL, opacity)
                }
                return buildContentBgDeclsWithOpacity(bgURL, opacity)
        }
        return "" // default 模式，跟随课件级
}

// extractBgURLFromDecls 从背景声明中提取 url('...') 里的URL
func extractBgURLFromDecls(decls string) string {
        idx := strings.Index(decls, "url('")
        if idx < 0 {
                return ""
        }
        start := idx + 5 // len("url('")
        end := strings.Index(decls[start:], "')")
        if end < 0 {
                return ""
        }
        return decls[start : start+end]
}

// attachUserBackground 把课件级用户选择的背景URL挂载进生成上下文（tplInfo）
func (s *CoursewareGenService) attachUserBackground(ctx context.Context, cw *models.Courseware, tplInfo *cwTemplateInfo) {
	if cw == nil || tplInfo == nil {
		return
	}
	cover, content, err := repository.GetCoursewareBackgroundURLs(ctx, cw.ID)
	if err != nil {
		cwGenLog.Warn("读取课件背景选择失败，按未选处理", "error", err, "courseware_id", cw.ID)
		return
	}
	tplInfo.CoverBgURL = cover
	tplInfo.ContentBgURL = content
	if fontCode, fErr := repository.GetCoursewareFontScheme(ctx, cw.ID); fErr == nil {
		tplInfo.FontSchemeCode = fontCode
	} else {
		cwGenLog.Warn("读取课件字体方案失败，按未选处理", "error", fErr, "courseware_id", cw.ID)
	}
	// 页级背景覆盖：一次性加载全部页的页级背景设置（供生成/秒换时四级优先级解析）
	if pageBgMap, pbErr := repository.ListPageBgSettings(ctx, cw.ID); pbErr == nil {
		tplInfo.PageBgSettings = pageBgMap
	} else {
		cwGenLog.Warn("读取页级背景设置失败，按未设处理", "error", pbErr, "courseware_id", cw.ID)
	}
}

// ==================== 背景秒换（确定性字符串操作，零token） ====================

// cwBgInjectedStyleRe 匹配已注入的背景<style>块（TEDNA-TPL-BG 标记块，只认标记不认选择器）
var cwBgInjectedStyleRe = regexp.MustCompile(`(?s)<style>/\* TEDNA-TPL-BG[^<]*</style>`)

// buildBgStyleTag 把背景声明串构建为注入用<style>标签（每条声明加 !important 压过内联样式）
// 选择器 .cw-page:last-of-type —— 双cw-page封面(导航栏在前+内容在后)只命中内容div，
// 不污染导航栏；单cw-page页first=last命中唯一div，与原 .cw-page{} 效果一致，零回归。
func buildBgStyleTag(bgDecls string) string {
	if strings.TrimSpace(bgDecls) == "" {
		return ""
	}
	var parts []string
	for _, d := range strings.Split(bgDecls, ";") {
		d = strings.TrimSpace(d)
		if d != "" {
			parts = append(parts, d+" !important")
		}
	}
	return "<style>/* TEDNA-TPL-BG 模板官方背景兜底注入 */.cw-page:last-of-type{" + strings.Join(parts, ";") + "}</style>"
}

// swapInjectedBackground 在已生成页面HTML上替换/注入/移除背景块（核心秒换原语）
//   - 页面已有 TEDNA-TPL-BG 块：整块替换为新背景（newTag为空=移除块）
//   - 页面没有该块且新背景非空：
//     · 双cw-page页(含NAV_END)：跳过 normalizeRootCanvas(避免撑坏导航栏)，直接注入
//     · 单cw-page页：过画布闸门规范根容器后注入
//
// 返回 (新HTML, 是否发生变化)
func swapInjectedBackground(html string, bgDecls string) (string, bool) {
	newTag := buildBgStyleTag(bgDecls)

	// 路径A：已有背景块 → 整块替换（移除时 newTag 为空）；不动任何div尺寸
	if cwBgInjectedStyleRe.MatchString(html) {
		out := cwBgInjectedStyleRe.ReplaceAllLiteralString(html, newTag)
		return out, out != html
	}

	// 路径B：无背景块
	if newTag == "" {
		return html, false
	}

	// ★双cw-page封面(含NAV_END:导航栏div+内容div)——跳过 normalizeRootCanvas。
	// 因该函数只认"第一个div"当根容器，会把导航栏div的height强改成1080px，
	// 使导航栏(position:absolute+z-index:100+不透明背景)撑成满屏遮罩盖住内容。
	// 这类页直接把<style>注入到第一个div开标签之后即可(选择器:last-of-type精准命中内容div，
	// 注入位置无所谓)，不改任何div尺寸，导航栏保持原顶栏高度。
	if strings.Contains(html, "<!-- NAV_END -->") {
		t := strings.TrimSpace(html)
		if !strings.HasPrefix(strings.ToLower(t), "<div") {
			return html, false
		}
		gt := strings.Index(t, ">")
		if gt < 0 {
			return html, false
		}
		return t[:gt+1] + newTag + t[gt+1:], true
	}

	// 单cw-page页：维持原逻辑（过画布闸门规范根容器后在其开标签后注入）
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

// ==================== 背景图库业务服务 ====================

// CoursewareBackgroundService 课件背景图库服务（无状态）
type CoursewareBackgroundService struct {
	cfg *config.Config
}

// NewCoursewareBackgroundService 创建背景图库服务
func NewCoursewareBackgroundService(cfg *config.Config) *CoursewareBackgroundService {
	return &CoursewareBackgroundService{cfg: cfg}
}

// ListSets 查询当前用户可见的图集（系统图库 + 本人个人集）
func (s *CoursewareBackgroundService) ListSets(ctx context.Context, userID string) ([]*models.CoursewareBackgroundSet, error) {
	return repository.ListBackgroundSets(ctx, userID)
}

// GetSelection 查询课件当前的背景选择
func (s *CoursewareBackgroundService) GetSelection(ctx context.Context, coursewareID string) (*models.CoursewareBackgroundSelection, error) {
	cover, content, err := repository.GetCoursewareBackgroundURLs(ctx, coursewareID)
	if err != nil {
		return nil, err
	}
	return &models.CoursewareBackgroundSelection{CoverBgURL: cover, ContentBgURL: content}, nil
}

// SelectBackground 课件选用某背景图集：写URL快照 + 秒换全部已生成页
func (s *CoursewareBackgroundService) SelectBackground(ctx context.Context, coursewareID string, userID string, setID string) (*models.BackgroundSelectionResult, error) {
	cw, err := repository.GetCoursewareByID(ctx, coursewareID)
	if err != nil {
		return nil, fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != userID {
		return nil, fmt.Errorf("无权操作此课件")
	}
	if cw.Status == models.CoursewareStatusInPipeline {
		return nil, fmt.Errorf("已提交审核的课件不允许修改背景")
	}
	set, err := repository.GetBackgroundSetByID(ctx, setID)
	if err != nil {
		return nil, fmt.Errorf("背景图集不存在: %w", err)
	}
	if set.Status != models.CWBgStatusActive {
		return nil, fmt.Errorf("该背景图集已下架")
	}
	if set.Scope == models.CWBgScopePersonal && (set.UserID == nil || *set.UserID != userID) {
		return nil, fmt.Errorf("无权使用他人的个人背景图集")
	}
	if err := repository.UpdateCoursewareBackground(ctx, coursewareID, set.CoverPublicURL, set.ContentPublicURL); err != nil {
		return nil, err
	}
	// 读取全部页级背景设置，秒换时页级覆盖优先于课件级
	pageBgMap, _ := repository.ListPageBgSettings(ctx, coursewareID)
	swapped, swapErr := s.swapGeneratedPages(ctx, coursewareID, func(pageNum int) string {
		courseLevelDecls := buildContentBgDecls(set.ContentPublicURL)
		if pageNum == 1 {
			courseLevelDecls = buildCoverBgDecls(set.CoverPublicURL)
		}
		if pageBg, ok := pageBgMap[pageNum]; ok {
			if pageDecls := resolvePageBgDecls(pageBg, pageNum, courseLevelDecls); pageDecls != "" {
				return pageDecls
			}
		}
		return courseLevelDecls
	})
	if swapErr != nil {
		cwGenLog.Warn("背景秒换部分失败（选择已生效，未换成功的页将在重生时获得新背景）", "error", swapErr, "courseware_id", coursewareID)
	}
	cwGenLog.Info("课件背景已选用", "courseware_id", coursewareID, "set", set.Name, "swapped_pages", swapped)
	return &models.BackgroundSelectionResult{CoverBgURL: set.CoverPublicURL, ContentBgURL: set.ContentPublicURL, SwappedPages: swapped}, nil
}

// ClearBackground 清除课件背景选择：两列置NULL + 已生成页回退到模板自带背景（无则移除背景块）
func (s *CoursewareBackgroundService) ClearBackground(ctx context.Context, coursewareID string, userID string) (*models.BackgroundSelectionResult, error) {
	cw, err := repository.GetCoursewareByID(ctx, coursewareID)
	if err != nil {
		return nil, fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != userID {
		return nil, fmt.Errorf("无权操作此课件")
	}
	if cw.Status == models.CoursewareStatusInPipeline {
		return nil, fmt.Errorf("已提交审核的课件不允许修改背景")
	}
	if err := repository.UpdateCoursewareBackground(ctx, coursewareID, "", ""); err != nil {
		return nil, err
	}
	samples := s.loadTemplateSamplesForCW(ctx, cw)
	swapped, swapErr := s.swapGeneratedPages(ctx, coursewareID, func(pageNum int) string {
		return extractSampleBackgroundDecls(samples, pageNum)
	})
	if swapErr != nil {
		cwGenLog.Warn("背景清除秒换部分失败", "error", swapErr, "courseware_id", coursewareID)
	}
	cwGenLog.Info("课件背景已清除", "courseware_id", coursewareID, "swapped_pages", swapped)
	return &models.BackgroundSelectionResult{CoverBgURL: "", ContentBgURL: "", SwappedPages: swapped}, nil
}

// SetPageBackground 设置单页背景覆盖（上传图+蒙版模式+透明度）并秒换该页
func (s *CoursewareBackgroundService) SetPageBackground(ctx context.Context, coursewareID string, userID string, pageNum int, bgURL string, opacity *float64, mode string) (map[string]interface{}, error) {
        cw, err := repository.GetCoursewareByID(ctx, coursewareID)
        if err != nil {
                return nil, fmt.Errorf("课件不存在: %w", err)
        }
        if cw.UserID != userID {
                return nil, fmt.Errorf("无权操作此课件")
        }
        if cw.Status == models.CoursewareStatusInPipeline {
                return nil, fmt.Errorf("已提交审核的课件不允许修改背景")
        }
        // 校验 mode
        if mode == "" {
                mode = "default"
        }
        if mode != "default" && mode != "custom" && mode != "none" {
                return nil, fmt.Errorf("无效的蒙版模式，可选: default/custom/none")
        }
        // 校验 opacity
        if opacity != nil && (*opacity < 0 || *opacity > 1) {
                return nil, fmt.Errorf("蒙版透明度必须在 0.0~1.0 之间")
        }
        // 写入页级背景设置
        if err := repository.UpdatePageBgSetting(ctx, coursewareID, pageNum, bgURL, opacity, mode); err != nil {
                return nil, err
        }
        // 秒换该页
        swapped := 0
        page, pErr := repository.GetCoursewarePageByNumber(ctx, coursewareID, pageNum)
        if pErr == nil && strings.TrimSpace(page.HTMLContent) != "" {
                // 构建该页应生效的背景声明
                pageBgSetting := &repository.PageBgSetting{PageBgURL: bgURL, PageBgOpacity: opacity, PageBgMode: mode}
                // 先取课件级声明作为兜底
                courseLevelDecls := ""
                cover, content, _ := repository.GetCoursewareBackgroundURLs(ctx, coursewareID)
                if pageNum == 1 && cover != "" {
                        courseLevelDecls = buildCoverBgDecls(cover)
                } else if pageNum > 1 && content != "" {
                        courseLevelDecls = buildContentBgDecls(content)
                }
                finalDecls := resolvePageBgDecls(pageBgSetting, pageNum, courseLevelDecls)
                if finalDecls == "" {
                        finalDecls = courseLevelDecls // 页级无覆盖，用课件级
                }
                newHTML, changed := swapInjectedBackground(page.HTMLContent, finalDecls)
                if changed {
                        if uErr := repository.UpdateCWPageHTMLOnly(ctx, page.ID, newHTML); uErr == nil {
                                swapped = 1
                        }
                }
        }
        cwGenLog.Info("页级背景已设置", "courseware_id", coursewareID, "page", pageNum, "mode", mode, "swapped", swapped)
        return map[string]interface{}{
                "page_number":   pageNum,
                "page_bg_url":   bgURL,
                "page_bg_mode":  mode,
                "opacity":       opacity,
                "swapped":       swapped,
        }, nil
}

// ClearPageBackground 清除单页背景覆盖（回退到跟随课件级）并秒换该页
func (s *CoursewareBackgroundService) ClearPageBackground(ctx context.Context, coursewareID string, userID string, pageNum int) (map[string]interface{}, error) {
        cw, err := repository.GetCoursewareByID(ctx, coursewareID)
        if err != nil {
                return nil, fmt.Errorf("课件不存在: %w", err)
        }
        if cw.UserID != userID {
                return nil, fmt.Errorf("无权操作此课件")
        }
        if cw.Status == models.CoursewareStatusInPipeline {
                return nil, fmt.Errorf("已提交审核的课件不允许修改背景")
        }
        if err := repository.ClearPageBgSetting(ctx, coursewareID, pageNum); err != nil {
                return nil, err
        }
        // 秒换该页：回退到课件级背景
        swapped := 0
        page, pErr := repository.GetCoursewarePageByNumber(ctx, coursewareID, pageNum)
        if pErr == nil && strings.TrimSpace(page.HTMLContent) != "" {
                cover, content, _ := repository.GetCoursewareBackgroundURLs(ctx, coursewareID)
                courseLevelDecls := ""
                if pageNum == 1 && cover != "" {
                        courseLevelDecls = buildCoverBgDecls(cover)
                } else if pageNum > 1 && content != "" {
                        courseLevelDecls = buildContentBgDecls(content)
                }
                // 清除后无页级覆盖，用课件级；课件级也无则用模板级（清除背景块）
                newHTML, changed := swapInjectedBackground(page.HTMLContent, courseLevelDecls)
                if changed {
                        if uErr := repository.UpdateCWPageHTMLOnly(ctx, page.ID, newHTML); uErr == nil {
                                swapped = 1
                        }
                }
        }
        cwGenLog.Info("页级背景已清除", "courseware_id", coursewareID, "page", pageNum, "swapped", swapped)
        return map[string]interface{}{
                "page_number": pageNum,
                "cleared":     true,
                "swapped":     swapped,
        }, nil
}

// GetPageBackground 查询单页背景设置
func (s *CoursewareBackgroundService) GetPageBackground(ctx context.Context, coursewareID string, pageNum int) (*repository.PageBgSetting, error) {
        return repository.GetPageBgSetting(ctx, coursewareID, pageNum)
}

// swapGeneratedPages 对课件全部"已生成HTML"的页执行背景秒换；declsFor 按页码给出目标背景声明
func (s *CoursewareBackgroundService) swapGeneratedPages(ctx context.Context, coursewareID string, declsFor func(pageNum int) string) (int, error) {
	pages, err := repository.ListCoursewarePages(ctx, coursewareID)
	if err != nil {
		return 0, err
	}
	swapped := 0
	var firstErr error
	for _, p := range pages {
		if strings.TrimSpace(p.HTMLContent) == "" {
			continue
		}
		newHTML, changed := swapInjectedBackground(p.HTMLContent, declsFor(p.PageNumber))
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

// loadTemplateSamplesForCW 取课件所选模板的样例页数组（清除背景时的回退声明源）；任何失败返回nil
func (s *CoursewareBackgroundService) loadTemplateSamplesForCW(ctx context.Context, cw *models.Courseware) []string {
	if cw == nil || cw.StyleConfig == "" {
		return nil
	}
	var sc struct {
		TemplateID string `json:"template_id"`
	}
	if json.Unmarshal([]byte(cw.StyleConfig), &sc) != nil || sc.TemplateID == "" {
		return nil
	}
	tpl, err := repository.GetCWTemplateByID(ctx, sc.TemplateID)
	if err != nil || tpl.SamplePages == "" {
		return nil
	}
	var samples []string
	if json.Unmarshal([]byte(tpl.SamplePages), &samples) != nil {
		return nil
	}
	return samples
}
