package services

// courseware_background_service.go — 课件背景图库服务（批次1新建）
//
// 职责：
//   1. 图集列表（系统+个人）
//   2. 课件选择/清除背景：写URL快照两列 + 【秒换】全部已生成页的注入背景
//      —— 秒换 = 纯字符串替换带 TEDNA-TPL-BG 标记的<style>块，零AI调用零token秒级完成，
//         根治"确认导航栏页选背景时封面早已生成"的时序问题（无需重画封面）。
//   3. 用户背景声明构建（统一蒙版方案，PRD Q2/Q3 既定参数）：
//      封面=左浓右透方向性渐变蒙版（文字习惯放左，与4套新模板同款）；
//      内页=奶白半透明平铺蒙版（把图压成隐约肌理）。
//   4. 三级优先级解析 resolveUserBgDecls + 生成上下文挂载 attachUserBackground，
//      供 applyTemplateBackground / appendSamplePageReference 实现：
//      老师图库选择(课件级) > 模板自带背景(样例提取) > 无。

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
	return "background:linear-gradient(105deg, rgba(255,253,248,0.94) 0%, rgba(255,253,248,0.80) 32%, rgba(255,253,248,0.34) 62%, rgba(255,253,248,0.08) 100%), url('" + url + "') center/cover no-repeat"
}

// buildContentBgDecls 构建内页背景声明：奶白半透明平铺蒙版 + 用户内页图（压成隐约肌理）
func buildContentBgDecls(url string) string {
	return "background:linear-gradient(rgba(255,253,250,0.86), rgba(255,253,250,0.86)), url('" + url + "') center/cover no-repeat"
}

// resolveUserBgDecls 三级优先级的第一级：课件级用户选择的背景（无则返回空串交给下一级）
// 封面(pageNum==1)用头图+方向蒙版；其余页用内页图+平铺蒙版
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

// attachUserBackground 把课件级用户选择的背景URL挂载进生成上下文（tplInfo）
// 三条生成路径（封面预览/批量/单页重生）在 loadTemplateInfo 后各调一次；查询失败静默跳过不阻断生成
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
}

// ==================== 背景秒换（确定性字符串操作，零token） ====================

// cwBgInjectedStyleRe 匹配已注入的背景<style>块（applyTemplateBackground 写入的 TEDNA-TPL-BG 标记块）
var cwBgInjectedStyleRe = regexp.MustCompile(`(?s)<style>/\* TEDNA-TPL-BG[^<]*</style>`)

// buildBgStyleTag 把背景声明串构建为注入用<style>标签（每条声明加 !important 压过内联样式）
// 声明为空返回空串（语义=移除背景块）；标记文本与 applyTemplateBackground 完全一致保证幂等互认
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
	return "<style>/* TEDNA-TPL-BG 模板官方背景兜底注入 */.cw-page{" + strings.Join(parts, ";") + "}</style>"
}

// swapInjectedBackground 在已生成页面HTML上替换/注入/移除背景块（核心秒换原语）
//   - 页面已有 TEDNA-TPL-BG 块：整块替换为新背景（newTag为空=移除块）
//   - 页面没有该块且新背景非空：走画布闸门后注入到根容器开标签之后
//
// 返回 (新HTML, 是否发生变化)
func swapInjectedBackground(html string, bgDecls string) (string, bool) {
	newTag := buildBgStyleTag(bgDecls)
	if cwBgInjectedStyleRe.MatchString(html) {
		out := cwBgInjectedStyleRe.ReplaceAllLiteralString(html, newTag)
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

// ==================== 背景图库业务服务 ====================

// CoursewareBackgroundService 课件背景图库服务（无状态）
type CoursewareBackgroundService struct {
	cfg *config.Config // 批次3：AI生成/上传背景需要AES密钥(豆包配置解密)与OSS上传配置
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
	// 1. 课件归属与状态校验
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
	// 2. 图集校验（存在+激活+个人集须本人）
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
	// 3. URL快照写入课件两列
	if err := repository.UpdateCoursewareBackground(ctx, coursewareID, set.CoverPublicURL, set.ContentPublicURL); err != nil {
		return nil, err
	}
	// 4. 秒换全部已生成页（封面用头图蒙版声明，内页用内页图蒙版声明）
	swapped, swapErr := s.swapGeneratedPages(ctx, coursewareID, func(pageNum int) string {
		if pageNum == 1 {
			return buildCoverBgDecls(set.CoverPublicURL)
		}
		return buildContentBgDecls(set.ContentPublicURL)
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
	// 回退声明源：模板自带背景（三级优先级的第二级）；模板也没有则为空=移除背景块
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
			continue // 尚未生成的页：生成时经 applyTemplateBackground 按新优先级注入
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
