package services

// courseware_gen_regenerate_page.go
//
// 根据课件页面方案从零重生单页。
//
// AI生成前固化稳定页面基线，AI返回后重新授权并复核导航栏。
// 最终使用与页面微调相同的原子版本快照和CAS写回仓储，
// 防止长时间AI调用覆盖期间产生的其它页面修改。

import (
	"context"
	"fmt"
	"strings"

	"tedna/internal/ai"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// RegenerateSinglePage 根据页面方案从零重生指定页面。
func (s *CoursewareGenService) RegenerateSinglePage(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	pageNumber int,
) (string, error) {
	courseware, scopedActor, err :=
		(&CoursewareService{}).LoadCoursewareForRefine(
			ctx,
			coursewareID,
			actor,
		)
	if err != nil {
		return "", err
	}

	userID := scopedActor.UserID

	if strings.TrimSpace(courseware.NavTemplateHTML) == "" {
		return "", fmt.Errorf("请先确认导航栏样式后再重新生成单页")
	}

	pages, err := repository.ListCoursewarePages(ctx, coursewareID)
	if err != nil || len(pages) == 0 {
		return "", fmt.Errorf("课件没有页面方案")
	}

	totalPages := len(pages)
	var page *models.CoursewarePage

	for _, candidate := range pages {
		if candidate.PageNumber == pageNumber {
			page = candidate
			break
		}
	}
	if page == nil {
		return "", fmt.Errorf("第%d页不存在", pageNumber)
	}

	mutationGuard, err := resolveCWPageMutationGuard(page, nil)
	if err != nil {
		return "", err
	}
	baselinePage := page

	styleConfig := s.parseStyleConfig(courseware.StyleConfig)
	templateInfo, templateErr := s.loadTemplateInfo(
		ctx,
		styleConfig.TemplateID,
	)
	if templateErr != nil {
		templateInfo = s.defaultTemplateInfo()
	}

	logoURL, orgName := s.resolveLogoAndOrg(
		ctx,
		courseware,
		styleConfig,
	)

	s.attachUserBackground(ctx, courseware, templateInfo)

	lessonContext := loadLessonPlanContextForGen(ctx, courseware)

	generationPrompt, err := repository.GetCurrentPromptByKey(
		"prompt_courseware_generate",
	)
	if err != nil {
		return "", fmt.Errorf("加载生成提示词失败: %w", err)
	}

	aiConfig, err := ai.GetEffectiveConfig(
		s.cfg.GetAESKey(),
		"courseware_generate",
		s.cfg.AIAPIBaseURL,
		s.cfg.AIAPIKey,
		s.cfg.AIDefaultModel,
	)
	if err != nil {
		return "", fmt.Errorf("获取AI配置失败: %w", err)
	}

	matchedComponents := s.matchComponentsForPage(
		ctx,
		baselinePage,
		courseware.Subject,
		courseware.Grade,
	)

	userPrompt := s.buildBatchUserPrompt(
		baselinePage,
		pageNumber,
		totalPages,
		templateInfo,
		logoURL,
		orgName,
		matchedComponents,
		courseware,
		lessonContext,
	)

	if pageNumber == 1 {
		userPrompt =
			"⚠️ 这是封面页（第1页），请生成大标题居中的封面设计，突出课件标题、学科年级和机构品牌。\n\n" +
				userPrompt
	}

	schoolID, _ := repository.GetSchoolIDByUserID(ctx, userID)
	traceContext := &ai.TraceContext{
		SceneCode: "courseware_generate",
		UserID:    &userID,
		SchoolID:  schoolIDPtr(schoolID),
	}

	result, aiErr := ai.CallAI(
		aiConfig,
		generationPrompt.Content,
		userPrompt,
		traceContext,
	)
	if aiErr != nil {
		return "", fmt.Errorf("AI重新生成失败: %w", aiErr)
	}

	contentHTML := s.extractHTMLFromAIOutput(result.Content)
	if contentHTML == "" {
		return "", fmt.Errorf("AI输出未包含有效HTML")
	}

	fullPageHTML := s.assembleFullPage(
		contentHTML,
		courseware.NavTemplateHTML,
		pageNumber,
		totalPages,
		templateInfo,
	)

	validation := validateRefinedPageHTML(
		baselinePage.HTMLContent,
		fullPageHTML,
		"",
		true,
	)
	if !validation.OK {
		cwGenLog.Warn(
			"单页重生输出未通过完整性校验，已保留原版未写库",
			"courseware_id", coursewareID,
			"page_num", pageNumber,
			"reason", validation.Reason,
			"detail", validation.Detail,
		)

		return "", fmt.Errorf("%s", validation.Reason)
	}

	if validation.FixedHTML != "" {
		cwGenLog.Info(
			"单页重生输出经轻微漏闭合自动补全后写库",
			"courseware_id", coursewareID,
			"page_num", pageNumber,
			"detail", validation.Detail,
		)
		fullPageHTML = validation.FixedHTML
	}

	latestCourseware, _, finalAuthErr :=
		(&CoursewareService{}).LoadCoursewareForRefine(
			ctx,
			coursewareID,
			scopedActor,
		)
	if finalAuthErr != nil {
		return "", finalAuthErr
	}

	latestPage, pageErr := repository.GetCoursewarePageByNumber(
		ctx,
		coursewareID,
		pageNumber,
	)
	if pageErr != nil {
		return "", fmt.Errorf(
			"%w: %v",
			ErrCoursewarePageNotFound,
			pageErr,
		)
	}

	if err := ensureCWPageStillMatchesMutationBaseline(
		latestPage,
		baselinePage,
		mutationGuard,
	); err != nil {
		return "", err
	}

	if latestCourseware.NavTemplateHTML != courseware.NavTemplateHTML {
		return "", ErrCoursewarePageMutationConflict
	}

	matchedIDs := s.buildMatchedComponentIDs(matchedComponents)

	casResult, databaseErr :=
		repository.UpdateCWPageHTMLWithVersionCAS(
			ctx,
			&repository.CoursewarePageCASWriteInput{
				PageID:                      mutationGuard.PageID,
				CoursewareID:                coursewareID,
				PageNumber:                  mutationGuard.PageNumber,
				ExpectedHTMLHash:            mutationGuard.HTMLHash,
				ExpectedPlaceholderMap:      baselinePage.PlaceholderMap,
				ExpectedMatchedComponentIDs: baselinePage.MatchedComponentIDs,
				ExpectedPageStatus:          baselinePage.Status,
				NewHTMLContent:              fullPageHTML,
				NewPlaceholderMap:           "",
				NewMatchedComponentIDs:      matchedIDs,
				NewPageStatus:               models.CWPageStatusGenerated,
				VersionSource:               models.CWPageVersionSourceRegenerate,
				VersionNote:                 "",
			},
		)
	if databaseErr != nil {
		return "", fmt.Errorf(
			"保存重生结果失败: %w",
			mapCWPageCASWriteError(databaseErr),
		)
	}

	cwGenLog.Info(
		"单页重新生成完成",
		"courseware_id", coursewareID,
		"page_num", pageNumber,
		"page_id", mutationGuard.PageID,
		"saved_version_no", casResult.VersionNo,
		"model", result.ModelUsed,
		"tokens", result.TokensUsed,
	)

	return fullPageHTML, nil
}
