package services

// courseware_page_version.go — 课件页面级版本与回退服务（页面级版本与回退·新建）
//
// 方法挂在 CoursewareGenService 上（与 RefinePage/RegenerateSinglePage 同结构体，
// 存版调用点就近，无需跨 service）。本文件不调 AI、不撑大 courseware_gen_service.go。
//
// 提供 5 个方法：
//   - SavePageVersionBeforeOverwrite  统一快照入口：在任何"即将覆盖 html_content"处先存旧版
//                                     （内部判空——oldHTML 为空则跳过，首次生成不算覆盖）
//   - ListCWPageVersions              鉴权后返回某页版本列表（轻量，不含 html）
//   - GetCWPageVersionHTML            鉴权后返回某个历史版本的完整 HTML（版本对比UI用，只读不改任何状态）
//   - RollbackCWPage                  回退到指定历史版本（先把当前版存为 rollback 版，保证可逆）
//   - SaveManualEditedPage            【就地文字编辑·新增】保存老师在预览iframe里就地改过的整页HTML
//                                     （只改文字/字号/颜色的确定性编辑，覆盖前存 manual 版本快照）
//
// 本期存版挂载点：RefinePage(refine) / RegenerateSinglePage(regenerate)，均在 courseware_gen_refine.go；
//   SaveManualEditedPage(manual)，供前端「✏️ 就地改文字」编辑器保存。
// 背景/字体秒换本期不挂（可随时再换回、回退价值低、逐页批量会耗配额），留枚举位备用。

import (
	"context"
	"fmt"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// SavePageVersionBeforeOverwrite 统一快照入口：把"即将被覆盖的旧 HTML"存为一个版本。
//
// 在任何即将调用 UpdateCWPageHTML / UpdateCWPageHTMLOnly 覆盖 html_content 的地方，
// 先调用本方法把旧值存档。内部对空值做判断：
//   - oldHTML 为空白：跳过（首次生成不算"覆盖"，不产生空版本快照）。
//   - 存版失败：仅记日志、不返回错误——存版是"附加保护"，绝不能因存版失败阻断用户的微调/重生主流程。
//
// 参数：
//
//	pageID/coursewareID —— 归属页与归属课件
//	oldHTML            —— 覆盖前的旧 HTML（调用方传 page.HTMLContent 旧值）
//	source            —— 来源枚举（models.CWPageVersionSourceRefine / Regenerate / ...）
//	note              —— 可选备注（微调指令 / 重生说明等）
func (s *CoursewareGenService) SavePageVersionBeforeOverwrite(ctx context.Context, pageID string, coursewareID string, oldHTML string, source string, note string) {
	// 判空：旧 HTML 为空 = 首次生成，不算覆盖，不存版
	if strings.TrimSpace(oldHTML) == "" {
		return
	}
	v, err := repository.CreatePageVersion(ctx, pageID, coursewareID, oldHTML, source, note)
	if err != nil {
		// 存版失败不阻断主流程（用户的微调/重生照常进行），仅记日志
		cwGenLog.Warn("页面版本快照保存失败（不影响本次修改）",
			"error", err, "page_id", pageID, "courseware_id", coursewareID, "source", source)
		return
	}
	cwGenLog.Info("页面版本快照已保存",
		"courseware_id", coursewareID, "page_id", pageID,
		"version_no", v.VersionNo, "source", source)
}

// ListCWPageVersions 鉴权后返回某页的版本列表（按 version_no 倒序，最新在前，不含 html_content）。
//
// 鉴权：校验课件存在且属于当前用户（与 RefinePage/RegenerateSinglePage 同口径）。
// 返回的每条版本附带来源中文标签（source_label）供前端直接显示。
func (s *CoursewareGenService) ListCWPageVersions(ctx context.Context, coursewareID string, userID string, pageNum int) ([]*models.CoursewarePageVersionListItem, error) {
	// 1. 课件归属校验
	cw, err := repository.GetCoursewareByID(ctx, coursewareID)
	if err != nil {
		return nil, fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != userID {
		return nil, fmt.Errorf("无权查看此课件")
	}
	// 2. 定位目标页（拿到 page.ID 作为版本查询键）
	page, err := repository.GetCoursewarePageByNumber(ctx, coursewareID, pageNum)
	if err != nil {
		return nil, fmt.Errorf("页面不存在: %w", err)
	}
	// 3. 查版本列表
	items, err := repository.ListPageVersions(ctx, page.ID)
	if err != nil {
		return nil, err
	}
	return items, nil
}

// GetCWPageVersionHTML 鉴权后返回某个历史版本的完整 HTML（版本对比UI用，只读，绝不改任何状态）。
//
// 用途：前端"版本对比"弹窗——需同时拿到【某历史版】与【当前版】两份完整 HTML 做左右并排渲染/源码对比。
// 当前版直接用页面 html_content（前端已有），历史版就靠本方法按 versionID 单独取。
//
// 流程（与 RollbackCWPage 的取版分支同口径，但只读不写）：
//  1. 校验课件归属 + 定位目标页（拿到 page.ID 用于防越权）。
//  2. 取目标版本（GetPageVersion，含完整 html_content）。
//  3. 防越权/防错页：目标版本必须属于本页（否则可用别页的 versionID 窥探他人内容）。
//  4. 返回该版本的完整 HTML + 版本号 + 来源标签（供弹窗标题展示"正在对比 v3 · 微调前"）。
//
// 参数：
//
//	versionID —— 要查看的历史版本 id（来自 ListCWPageVersions 返回的某条）
//
// 返回：目标版本的完整 HTML、版本号、来源枚举原始值（前端可再映射中文，但一般直接用列表里已有的 source_label）。
func (s *CoursewareGenService) GetCWPageVersionHTML(ctx context.Context, coursewareID string, userID string, pageNum int, versionID string) (html string, versionNo int, source string, err error) {
	// 1. 课件归属校验
	cw, e := repository.GetCoursewareByID(ctx, coursewareID)
	if e != nil {
		return "", 0, "", fmt.Errorf("课件不存在: %w", e)
	}
	if cw.UserID != userID {
		return "", 0, "", fmt.Errorf("无权查看此课件")
	}
	// 2. 定位目标页（用于防越权校验：目标版本必须属于本页）
	page, e := repository.GetCoursewarePageByNumber(ctx, coursewareID, pageNum)
	if e != nil {
		return "", 0, "", fmt.Errorf("页面不存在: %w", e)
	}
	// 3. 取目标版本完整内容
	target, e := repository.GetPageVersion(ctx, versionID)
	if e != nil {
		return "", 0, "", fmt.Errorf("目标版本不存在: %w", e)
	}
	// 4. 防越权/防错页：目标版本必须属于本页（拒绝用别页/别课件的 versionID 窥探）
	if target.PageID != page.ID {
		return "", 0, "", fmt.Errorf("目标版本不属于本页")
	}
	return target.HTMLContent, target.VersionNo, target.Source, nil
}

// RollbackCWPage 回退指定页到某历史版本，返回回退后的完整 HTML。
//
// 流程（保证回退可逆）：
//  1. 校验课件归属 + 定位目标页。
//  2. 取目标版本（GetPageVersion，含完整 html_content）；校验该版本确属本页（防越权传别页的 versionID）。
//  3. 先把【当前】html_content 存为一个新版本（source=rollback），这样回退本身也能再退回。
//  4. 把目标版本的 html 写回 courseware_pages.html_content（用 UpdateCWPageHTMLOnly，只动 html 不碰 status）。
//  5. 返回回退后的 html，供前端刷新大预览。
//
// 参数：
//
//	versionID —— 要回退到的目标版本 id（来自 ListCWPageVersions 返回的某条）
func (s *CoursewareGenService) RollbackCWPage(ctx context.Context, coursewareID string, userID string, pageNum int, versionID string) (string, error) {
	// 1. 课件归属校验
	cw, err := repository.GetCoursewareByID(ctx, coursewareID)
	if err != nil {
		return "", fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != userID {
		return "", fmt.Errorf("无权操作此课件")
	}
	// 已提交审核的课件不允许回退（与背景/字体秒换同口径，防改动审核中内容）
	if cw.Status == models.CoursewareStatusInPipeline {
		return "", fmt.Errorf("已提交审核的课件不允许回退页面版本")
	}
	// 2. 定位目标页
	page, err := repository.GetCoursewarePageByNumber(ctx, coursewareID, pageNum)
	if err != nil {
		return "", fmt.Errorf("页面不存在: %w", err)
	}
	// 3. 取目标版本完整内容
	target, err := repository.GetPageVersion(ctx, versionID)
	if err != nil {
		return "", fmt.Errorf("目标版本不存在: %w", err)
	}
	// 防越权/防错页：目标版本必须属于本页
	if target.PageID != page.ID {
		return "", fmt.Errorf("目标版本不属于本页")
	}
	if strings.TrimSpace(target.HTMLContent) == "" {
		return "", fmt.Errorf("目标版本内容为空，无法回退")
	}
	// 4. 先把【当前】HTML 存为 rollback 版（保证回退可逆）——用统一快照入口，内部判空（当前为空则不存）
	s.SavePageVersionBeforeOverwrite(ctx, page.ID, coursewareID, page.HTMLContent,
		models.CWPageVersionSourceRollback,
		fmt.Sprintf("回退到第%d版前的当前内容", target.VersionNo))
	// 5. 把目标版本 HTML 写回（只动 html_content + updated_at，不碰 placeholder_map/status）
	if err := repository.UpdateCWPageHTMLOnly(ctx, page.ID, target.HTMLContent); err != nil {
		return "", fmt.Errorf("写回回退内容失败: %w", err)
	}
	cwGenLog.Info("页面已回退到历史版本",
		"courseware_id", coursewareID, "page_num", pageNum,
		"rolled_back_to_version", target.VersionNo, "version_id", versionID)
	return target.HTMLContent, nil
}

// SaveManualEditedPage 【就地文字编辑·新增】保存老师在预览 iframe 里就地改过的整页 HTML。
//
// 场景：前端「✏️ 就地改文字」编辑器——老师在预览里点选某段文字，改文字内容/字号/颜色后保存。
//
//	前端已在 iframe 内用纯 DOM 操作完成修改（只改文字节点值与内联 style，不新增节点、不产生脏 DOM），
//	并把清理掉编辑器痕迹后的整页纯净 HTML 回传本方法。本方法只负责"存旧版 + 写新版"两步落库，
//	与背景/字体秒换、回退同属"确定性 HTML 覆盖"，不调 AI。
//
// 流程（与 RollbackCWPage 同口径的鉴权 + 覆盖前存版）：
//  1. 校验课件归属（仅作者本人）+ in_pipeline 拦截（审核中不允许改）。
//  2. 定位目标页，拿到 page.ID 与旧 HTML。
//  3. 新 HTML 基本校验：非空、且必须与旧 HTML 不同（相同则无需落库，直接返回旧值省一次写与一条空版本）。
//  4. 覆盖前把【旧】HTML 存为 manual 版本快照（可回退、可对比）。
//  5. UpdateCWPageHTMLOnly 写回新 HTML（只动 html_content + updated_at，不碰 placeholder_map/status）。
//  6. 返回写回后的 HTML（即入参 newHTML），供前端刷新大预览。
//
// 参数：
//
//	newHTML —— 前端回传的、已清理编辑器痕迹的整页纯净 HTML
func (s *CoursewareGenService) SaveManualEditedPage(ctx context.Context, coursewareID string, userID string, pageNum int, newHTML string) (string, error) {
	// 1. 课件归属校验
	cw, err := repository.GetCoursewareByID(ctx, coursewareID)
	if err != nil {
		return "", fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != userID {
		return "", fmt.Errorf("无权操作此课件")
	}
	// 已提交审核的课件不允许就地编辑（与回退/秒换同口径，防改动审核中内容）
	if cw.Status == models.CoursewareStatusInPipeline {
		return "", fmt.Errorf("已提交审核的课件不允许就地编辑页面")
	}
	// 2. 定位目标页（拿到 page.ID 与旧 HTML）
	page, err := repository.GetCoursewarePageByNumber(ctx, coursewareID, pageNum)
	if err != nil {
		return "", fmt.Errorf("页面不存在: %w", err)
	}
	// 3. 新 HTML 基本校验
	trimmed := strings.TrimSpace(newHTML)
	if trimmed == "" {
		return "", fmt.Errorf("编辑后的内容为空，未保存")
	}
	// 内容未变（例如老师点了保存但实际没改）：无需落库，直接返回旧值，避免产生一条无意义的版本快照
	if newHTML == page.HTMLContent {
		return page.HTMLContent, nil
	}
	// 4. 覆盖前把【旧】HTML 存为 manual 版本快照（可回退、可对比；内部判空——旧值为空则不存）
	s.SavePageVersionBeforeOverwrite(ctx, page.ID, coursewareID, page.HTMLContent,
		models.CWPageVersionSourceManual,
		"就地文字编辑前")
	// 5. 把新 HTML 写回（只动 html_content + updated_at，不碰 placeholder_map/status）
	if err := repository.UpdateCWPageHTMLOnly(ctx, page.ID, newHTML); err != nil {
		return "", fmt.Errorf("保存编辑内容失败: %w", err)
	}
	cwGenLog.Info("页面就地文字编辑已保存",
		"courseware_id", coursewareID, "page_num", pageNum, "page_id", page.ID)
	return newHTML, nil
}
