package services

// courseware_auto_assembly_html.go —— 全自动装配单页HTML生成链
//
// 本文件只负责从页面方案生成完整HTML：组件匹配、提示词构建、AI调用、
// HTML结构完整性校验、互动契约验收、导航组装和最终落库。
//
// 真实HTML结构残缺只允许自动从原页面方案全新生成一次；
// 不会对未闭合script/style等高风险截断做机械补全。

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"tedna/internal/ai"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== 链① 单页 HTML 生成 ====================

// generateOnePageHTML 生成单页完整 HTML 并落库，返回组装后的完整 HTML。
//
// 完全复刻 GenerateRemainingPages 的单页生成链路，保证与批量生成产出一致：
//
//	匹配组件 → 构建 batch user prompt → 调 AI → 抽取内容区HTML → 后端拼接导航模板组装完整页 → 落库。
//
// 出错返回 error，由主编排记为该页 HTML 失败（该页不再进入配图流水线）。
func (s *CoursewareAutoAssemblyService) generateOnePageHTML(
	ctx context.Context, pc *cwAssemblyPageContext, page *models.CoursewarePage,
) (generatedHTML string, returnErr error) {
	// R-04：单页生成或必要校验明确失败时，立即把稳定page_id失败事实写入本次运行账本。
	// 取消/换版后记录会因版本冲突被拒绝，此时只保留取消语义，不把迟到失败污染新运行。
	defer func() {
		if returnErr == nil || page == nil || strings.TrimSpace(page.ID) == "" {
			return
		}

		recordErr := repository.RecordCoursewareGenerationPageFailure(
			ctx,
			page.ID,
		)
		if recordErr != nil &&
			!errors.Is(
				recordErr,
				repository.ErrCoursewareAssemblyVersionConflict,
			) {
			cwAssemblyLog.Warn(
				"记录自动装配单页HTML失败事实失败",
				"page_id", page.ID,
				"page", page.PageNumber,
				"error", recordErr,
			)
		}
	}()

	// 1. 匹配组件（真实签名：matchComponentsForPage(ctx, page, subject, grade)）
	matched := s.genService.matchComponentsForPage(ctx, page, pc.cw.Subject, pc.cw.Grade)

	// 2. 构建批量模式 user prompt（真实签名 9 参：
	//    page, pageNum, totalPages, tplInfo, logoURL, orgName, matchedComps, cw, lessonContext）
	userPrompt := s.genService.buildBatchUserPrompt(
		page, page.PageNumber, pc.totalPages,
		pc.tplInfo, pc.logoURL, pc.orgName,
		matched, pc.cw, pc.lessonContext,
	)
	// 封面页(第1页)补封面提示（与 RegenerateSinglePage 对齐；批量提示词默认不含封面提示）
	if page.PageNumber == 1 {
		userPrompt = "⚠️ 这是封面页（第1页），请生成大标题居中的封面设计，突出课件标题、学科年级与机构品牌。\n\n" + userPrompt
	}

	// 3. 调AI生成内容区。
	// API错误、空HTML、HTML结构残缺或互动契约未落实时，均在当前页自动纠偏重试。
	traceCtx := &ai.TraceContext{
		SceneCode: "courseware_generate",
		UserID:    &pc.userID,
		SchoolID:  schoolIDPtr(pc.schoolID),
	}

	var lastErr error
	fullHTML := ""
	maxAttempts := cwGenMaxAttempts
	htmlStructureRetryUsed := false
	attemptPrompt := userPrompt

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		callResult, callErr := ai.CallAI(
			pc.aiCfg,
			pc.genPrompt.Content,
			attemptPrompt,
			traceCtx,
		)
		lastErr = callErr

		if lastErr == nil &&
			(callResult == nil ||
				strings.TrimSpace(callResult.Content) == "") {
			lastErr = fmt.Errorf("AI返回空内容")
		}

		if lastErr == nil {
			contentHTML := s.genService.extractHTMLFromAIOutput(
				callResult.Content,
			)
			if strings.TrimSpace(contentHTML) == "" {
				lastErr = fmt.Errorf("抽取HTML为空")
			} else {
				structureCheck := validateRefinedPageHTML(
					"",
					contentHTML,
					"",
					true,
				)
				if !structureCheck.OK {
					lastErr = fmt.Errorf(
						"HTML结构不完整: %s",
						structureCheck.Reason,
					)

					if htmlStructureRetryUsed {
						cwAssemblyLog.Warn(
							"全自动装配HTML结构重生后仍不完整，停止自动重试",
							"page", page.PageNumber,
							"attempt", attempt,
							"detail", structureCheck.Detail,
						)
						break
					}

					htmlStructureRetryUsed = true
					attemptPrompt = userPrompt
					if attempt >= maxAttempts {
						maxAttempts = attempt + 1
					}

					cwAssemblyLog.Warn(
						"全自动装配HTML结构不完整，准备从原页面方案全新生成一次",
						"page", page.PageNumber,
						"attempt", attempt,
						"detail", structureCheck.Detail,
					)

					time.Sleep(cwGenRetryBaseDelay)
					continue
				}

				if structureCheck.FixedHTML != "" {
					contentHTML = structureCheck.FixedHTML
				}

				// 只检查AI生成的内容区，避免系统导航栏事件干扰互动类型验收。
				interactionCheck :=
					validateGeneratedPageInteraction(
						page.InteractionType,
						contentHTML,
					)

				if interactionCheck.OK {
					fullHTML = s.genService.assembleFullPage(
						contentHTML,
						pc.navHTML,
						page.PageNumber,
						pc.totalPages,
						pc.tplInfo,
					)

					if attempt > 1 {
						cwAssemblyLog.Info(
							"全自动装配互动契约纠偏成功",
							"page", page.PageNumber,
							"interaction_type", page.InteractionType,
							"attempt", attempt,
						)
					}
					break
				}

				lastErr = fmt.Errorf(
					"互动方式未落实: %s",
					interactionCheck.Reason,
				)
				attemptPrompt = buildCWInteractionRepairPrompt(
					userPrompt,
					page,
					interactionCheck,
				)

				cwAssemblyLog.Warn(
					"全自动装配互动契约验收失败，准备纠偏重试",
					"page", page.PageNumber,
					"interaction_type", page.InteractionType,
					"reason", interactionCheck.Reason,
					"detail", interactionCheck.Detail,
					"attempt", attempt,
					"max_attempts", maxAttempts,
				)
			}
		}

		if lastErr != nil && attempt < maxAttempts {
			time.Sleep(
				time.Duration(attempt) *
					cwGenRetryBaseDelay,
			)
		}
	}

	if strings.TrimSpace(fullHTML) == "" {
		if lastErr == nil {
			lastErr = fmt.Errorf(
				"未形成通过互动契约验收的有效HTML",
			)
		}
		return "", fmt.Errorf(
			"单页生成失败(最多尝试%d次): %v",
			maxAttempts,
			lastErr,
		)
	}

	// 6. 落库（真实签名 6 参：UpdateCWPageHTML(ctx, pageID, html, placeholderMap, matchedIDs, status)）
	matchedIDs := s.genService.buildMatchedComponentIDs(matched)
	if err := repository.UpdateCWPageHTML(ctx, page.ID, fullHTML, "", matchedIDs, models.CWPageStatusGenerated); err != nil {
		return "", fmt.Errorf("HTML落库失败: %w", err)
	}

	return fullHTML, nil
}
