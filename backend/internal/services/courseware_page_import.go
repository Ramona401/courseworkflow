package services

// courseware_page_import.go — 【粘贴HTML建页·批次B新增】外部HTML导入服务
//
// 场景：老师看到别人做得好的课件页HTML代码，在 Step5「＋添加页面」弹窗选「📋 粘贴HTML」模式，
//   前端先 addPage 建一个空方案页，再把粘贴的完整HTML经本服务导入该页。
//
// 与 SaveManualEditedPage（就地文字编辑保存, courseware_page_version.go）的区别：
//   - 就地编辑回传的本来就是"本课件生成的规范页"，无需归一化，直接存旧版+写新版即可；
//   - 粘贴导入的是外来HTML，画布尺寸/根容器/导航栏页码/背景都不一定符合本课件契约，
//     故写库前做整备（详见 ImportPageHTML 流程注释）。
//
// 方法挂在 CoursewareGenService 上（与 RefinePage/SaveManualEditedPage 同结构体），
// 复用其全部既有零件：normalizeRootCanvas / injectPageNumIntoNav / applyTemplateBackground /
// SavePageVersionBeforeOverwrite。本文件不调 AI、不新增 repository 函数。
//
// 鉴权口径：仅作者本人（与就地编辑 SaveManualEditedPage 一致，从简；
//   集体备课参与者如需此能力，后续按 canRefineCourseware 口径再放宽）。

import (
        "context"
        "fmt"
        "strings"

        "tedna/internal/models"
        "tedna/internal/repository"
)

// 导航栏标记常量（与生成链路的 NAV_START/NAV_END 标准写法一致，含前后空格）。
// 仅用于识别"从平台其它课件复制来的页"——这类页自带标记，需把导航栏换成本课件的并重编页码；
// 外部来源的HTML通常没有标记，属预期情况，导航栏处理整体跳过、不强插 80px 导航条。
const (
        importNavStartMarker = "<!-- NAV_START -->"
        importNavEndMarker   = "<!-- NAV_END -->"
)

// ImportPageHTML 把老师粘贴的外部完整HTML导入指定页，返回整备落库后的最终HTML。
//
// 流程（整备 + 快照 + 写库）：
//  1. 鉴权：课件归属校验（仅作者本人）+ in_pipeline 拦截（审核中不允许改）。
//  2. 定位目标页（拿到 page.ID 与旧 HTML——通常是 AddPageModal 刚建的空页，旧值为空）。
//  3. 画布契约归一：normalizeRootCanvas 压住 1920×1080 根容器、剥除误带的 transform、补 cw-page 类，
//     保证外来代码进入本课件后不变形、可被播放器统一等比缩放。
//  4. 导航栏替换重编号（仅当 粘贴内容自带 NAV_START/NAV_END 标记 且 本课件已确认导航栏模板 时）：
//     把标记区间整体替换为【本课件】的导航栏 + 本页页码/总页数，使从平台其它课件复制来的页
//     与本课件导航风格统一、页码正确。外部HTML无标记则整体跳过——不强插导航条，
//     避免破坏外来页面自身的版面（属预期设计，非缺陷）。
//     注：仅整备本页导航；其它已有页导航里"总页数"随加页产生的滞后，与「AI生成」加页路径
//     行为一致，统一归导航栏一致性机制处理，不在本导入流程内逐页刷写。
//  5. 背景/字体幂等补注：applyTemplateBackground 按既有优先级补注本课件当前背景（含页级覆盖），
//     使粘贴页与整套课件视觉统一（幂等——已有注入块则不重复注入）。
//  6. 覆盖前版本快照：SavePageVersionBeforeOverwrite(manual, "粘贴HTML导入前")——
//     新建空页旧值为空时统一入口内部自动跳过；若老师对已有内容的页重复导入，则旧版可回退。
//  7. UpdateCWPageHTML 写库并把页状态置为 generated（与微调写库同口径），
//     使导入页立即计入"已生成页面"，进入预览/放映/导出等全部下游流程。
//
// 参数：
//
//	pastedHTML —— 老师粘贴的完整页面HTML（handler 层已做非空与 5MB 上限校验）
func (s *CoursewareGenService) ImportPageHTML(ctx context.Context, coursewareID string, userID string, pageNum int, pastedHTML string) (string, error) {
        // 1. 课件归属校验（仅作者本人，与就地编辑同口径）
        cw, err := repository.GetCoursewareByID(ctx, coursewareID)
        if err != nil {
                return "", fmt.Errorf("课件不存在: %w", err)
        }
        if cw.UserID != userID {
                return "", fmt.Errorf("无权操作此课件")
        }
        // 已提交审核的课件不允许导入（与回退/秒换/就地编辑同口径，防改动审核中内容）
        if cw.Status == models.CoursewareStatusInPipeline {
                return "", fmt.Errorf("已提交审核的课件不允许导入页面")
        }

        // 2. 定位目标页（拿到 page.ID 与旧 HTML）
        page, err := repository.GetCoursewarePageByNumber(ctx, coursewareID, pageNum)
        if err != nil {
                return "", fmt.Errorf("页面不存在: %w", err)
        }

        // 3. 基本校验 + 画布契约归一
        if strings.TrimSpace(pastedHTML) == "" {
                return "", fmt.Errorf("粘贴的内容为空，未导入")
        }
        normalized := normalizeRootCanvas(pastedHTML)

        // 4. 导航栏替换重编号（仅平台内复制来的、自带 NAV 标记的页；外部HTML无标记则跳过）
        if strings.TrimSpace(cw.NavTemplateHTML) != "" {
                si := strings.Index(normalized, importNavStartMarker)
                ei := strings.Index(normalized, importNavEndMarker)
                if si >= 0 && ei > si {
                        // 总页数取当前课件全部页（含本次新建页），用于导航栏"第N/共M页"显示
                        totalPages := 0
                        if pages, pErr := repository.ListCoursewarePages(ctx, coursewareID); pErr == nil {
                                totalPages = len(pages)
                        }
                        // 用本课件导航栏模板注入本页页码；模板/注入结果若自带标记先剥掉，防止双重包裹
                        newNav := injectPageNumIntoNav(cw.NavTemplateHTML, pageNum, totalPages)
                        newNav = strings.ReplaceAll(newNav, importNavStartMarker, "")
                        newNav = strings.ReplaceAll(newNav, importNavEndMarker, "")
                        normalized = normalized[:si] + importNavStartMarker + newNav + importNavEndMarker + normalized[ei+len(importNavEndMarker):]
                        cwGenLog.Info("粘贴导入：已将来源页导航栏替换为本课件导航栏",
                                "courseware_id", coursewareID, "page_num", pageNum, "total_pages", totalPages)
                }
        }

        // 5. 背景/字体幂等补注（与 RefinePage 的背景保持逻辑同口径：已有注入块不重复）
        styleCfg := s.parseStyleConfig(cw.StyleConfig)
        if tplInfo, tErr := s.loadTemplateInfo(ctx, styleCfg.TemplateID); tErr == nil {
                s.attachUserBackground(ctx, cw, tplInfo)
                normalized = s.applyTemplateBackground(normalized, tplInfo, pageNum)
        }

        // 6. 覆盖前版本快照（manual 来源；新建空页旧值为空时入口内部自动跳过，存版失败不阻断导入）
        s.SavePageVersionBeforeOverwrite(ctx, page.ID, coursewareID, page.HTMLContent,
                models.CWPageVersionSourceManual, "粘贴HTML导入前")

        // 7. 写库并置 generated 状态（与微调写库同口径），导入页立即计入已生成页面
        if dbErr := repository.UpdateCWPageHTML(ctx, page.ID, normalized, "", page.MatchedComponentIDs, models.CWPageStatusGenerated); dbErr != nil {
                return "", fmt.Errorf("保存导入内容失败: %w", dbErr)
        }

        cwGenLog.Info("粘贴HTML导入完成",
                "courseware_id", coursewareID, "page_num", pageNum, "page_id", page.ID,
                "html_len", len(normalized))
        return normalized, nil
}
