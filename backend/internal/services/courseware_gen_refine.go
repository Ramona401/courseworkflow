package services

// courseware_gen_refine.go
//
// 课件导航栏AI微调服务。
//
// 单页AI微调、全页重构和单页重生已经按页面写入职责拆分到：
//
//     courseware_gen_refine_page.go
//
// 导航栏微调仍使用当前封面页作为真实底稿，并在AI返回后重新授权、
// 重新绑定稳定页面ID。导航栏最终确认仍由SaveNavTemplate负责，
// 本入口只更新第1页预览HTML，不提前写入nav_template_html。

import (
	"context"
	"fmt"
	"strings"

	"tedna/internal/ai"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// RefineNav 根据老师修改意见微调第1页导航栏。
func (s *CoursewareGenService) RefineNav(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	instruction string,
) (string, error) {
	// 1. 加载正式课件并执行教研微调授权。
	courseware, scopedActor, err :=
		(&CoursewareService{}).
			LoadCoursewareForRefine(
				ctx,
				coursewareID,
				actor,
			)
	if err != nil {
		return "", err
	}

	userID := scopedActor.UserID

	// 2. 导航栏确认页的真实底稿必须是当前第1页。
	pages, pageListErr :=
		repository.ListCoursewarePages(
			ctx,
			coursewareID,
		)
	if pageListErr != nil ||
		len(pages) == 0 {
		return "", fmt.Errorf(
			"没有可用的封面页面",
		)
	}

	var coverPage *models.CoursewarePage

	for _, page := range pages {
		if page.PageNumber == 1 {
			coverPage = page
			break
		}
	}

	if coverPage == nil ||
		strings.TrimSpace(
			coverPage.HTMLContent,
		) == "" {
		return "", fmt.Errorf(
			"第1页封面尚未生成，无法微调导航栏",
		)
	}

	currentNav :=
		ExtractNavByMarkers(
			coverPage.HTMLContent,
		)

	if strings.TrimSpace(
		currentNav,
	) == "" {
		// 极端存量页面兜底：
		// 只有第1页无法识别导航栏时才参考已保存模板。
		currentNav =
			courseware.NavTemplateHTML
	}

	if strings.TrimSpace(
		currentNav,
	) == "" {
		return "", fmt.Errorf(
			"无法从封面页提取导航栏",
		)
	}

	// 3. 导航栏提示词严格限制修改范围。
	systemPrompt := `你是课件导航栏样式微调助手。你会收到一段导航栏HTML代码和老师的修改意见。

【绝对约束】
1. 只修改老师明确要求修改的部分
2. 不得修改老师未提到的任何样式、颜色、字号、布局和文字
3. 除非老师明确要求，否则不得添加或删除元素；老师明确要求删除Logo、机构名、年级、页码或其它指定元素时，只允许删除其明确指定的元素
4. 不得重构整体结构，不得把模板导航栏改成另一种导航栏
5. 必须保留模板导航栏原有高度、排版方式、背景、边框、间距和对齐规则；不得强制改成平台默认80px格式
6. 输出完整的修改后导航栏HTML，用<!-- NAV_START -->和<!-- NAV_END -->包裹
7. 不得输出封面正文，不得修改导航栏之外的任何页面内容

如果老师的要求模糊，选择最小改动方案。
直接输出修改后的HTML代码，不要输出任何解释文字。`

	userPrompt :=
		"## 当前导航栏HTML\n```html\n" +
			currentNav +
			"\n```\n\n## 老师的修改意见\n" +
			instruction +
			"\n\n请只修改导航栏中老师明确指出的部分，其余内容逐字逐结构保留。用<!-- NAV_START -->和<!-- NAV_END -->包裹输出。"

	// 4. 调用导航栏独立AI场景。
	aiConfig, err :=
		ai.GetEffectiveConfig(
			s.cfg.GetAESKey(),
			models.SceneCWNavRefine,
			s.cfg.AIAPIBaseURL,
			s.cfg.AIAPIKey,
			s.cfg.AIDefaultModel,
		)
	if err != nil {
		return "", fmt.Errorf(
			"获取AI配置失败: %w",
			err,
		)
	}

	schoolID, _ :=
		repository.GetSchoolIDByUserID(
			ctx,
			userID,
		)

	traceContext := &ai.TraceContext{
		SceneCode: models.SceneCWNavRefine,
		UserID:    &userID,
		SchoolID:  schoolIDPtr(schoolID),
	}

	result, aiErr :=
		ai.CallAI(
			aiConfig,
			systemPrompt,
			userPrompt,
			traceContext,
		)
	if aiErr != nil {
		return "", fmt.Errorf(
			"AI微调失败: %w",
			aiErr,
		)
	}

	// 5. 提取AI返回导航栏，并由后端确定性管理页码。
	refinedNav :=
		ExtractNavByMarkers(
			result.Content,
		)

	if strings.TrimSpace(
		refinedNav,
	) == "" {
		refinedNav =
			s.extractHTMLFromAIOutput(
				result.Content,
			)
	}

	if strings.TrimSpace(
		refinedNav,
	) == "" {
		return "", fmt.Errorf(
			"AI输出未包含有效的导航栏HTML",
		)
	}

	refinedNav =
		StripNavPageNumbers(
			refinedNav,
		)

	previewNav :=
		injectPageNumIntoNav(
			refinedNav,
			1,
			len(pages),
		)

	// 6. AI返回后重新授权并按课件ID和页码重新绑定封面。
	latestCourseware, _, finalAuthErr :=
		(&CoursewareService{}).
			LoadCoursewareForRefine(
				ctx,
				coursewareID,
				scopedActor,
			)
	if finalAuthErr != nil {
		return "", finalAuthErr
	}

	latestCoverPage, pageErr :=
		repository.GetCoursewarePageByNumber(
			ctx,
			coursewareID,
			1,
		)
	if pageErr != nil {
		return "", fmt.Errorf(
			"%w: %v",
			ErrCoursewarePageNotFound,
			pageErr,
		)
	}

	if latestCoverPage.ID !=
		coverPage.ID {
		return "",
			ErrCoursewarePageMutationConflict
	}

	courseware = latestCourseware
	coverPage = latestCoverPage

	// 7. 只替换第1页导航栏，封面正文逐字保留。
	updatedPageHTML, replaced :=
		replaceRefinedNavInPageHTML(
			coverPage.HTMLContent,
			previewNav,
		)
	if !replaced {
		return "", fmt.Errorf(
			"无法可靠定位封面导航栏，已保留原页面",
		)
	}

	// 导航栏微调重新进入待确认状态。
	if strings.TrimSpace(
		courseware.NavTemplateHTML,
	) != "" {
		clearErr :=
			repository.UpdateCoursewareNavTemplate(
				ctx,
				coursewareID,
				"",
			)
		if clearErr != nil {
			return "", fmt.Errorf(
				"清除旧导航栏确认状态失败: %w",
				clearErr,
			)
		}

		courseware.NavTemplateHTML = ""
	}

	// 导航栏微调暂不属于R-01整改页面执行链，
	// 继续保留原有版本保护和页面写入方式。
	s.SavePageVersionBeforeOverwrite(
		ctx,
		coverPage.ID,
		coursewareID,
		coverPage.HTMLContent,
		models.CWPageVersionSourceNavResync,
		instruction,
	)

	databaseErr :=
		repository.UpdateCWPageHTMLOnly(
			ctx,
			coverPage.ID,
			updatedPageHTML,
		)
	if databaseErr != nil {
		return "", fmt.Errorf(
			"保存微调后的封面导航栏失败: %w",
			databaseErr,
		)
	}

	cwGenLog.Info(
		"导航栏微调完成并已替换回封面",
		"courseware_id",
		coursewareID,
		"page_id",
		coverPage.ID,
		"instruction",
		instruction,
		"model",
		result.ModelUsed,
		"tokens",
		result.TokensUsed,
	)

	// 此处刻意不写coursewares.nav_template_html。
	return previewNav, nil
}
