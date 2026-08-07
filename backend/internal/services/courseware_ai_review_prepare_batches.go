package services

// courseware_ai_review_prepare_batches.go
//
// 课件AI审核准备阶段的批次规划与初始连续性账本。
//
// 批次规则：
//   - 页面按教学顺序执行；
//   - 目标每批5页，最少3页，最多6页；
//   - 优先在新的教学环节前切分；
//   - 相邻批次保留1页重叠，用于连续性判断；
//   - 第一批使用会话初始账本；
//   - 后续批次实际执行时使用上一批完整账本。
//
// 本文件只规划范围和初始清单，不调用AI，不读取浏览器页面正文。

import (
	"encoding/json"
	"fmt"
	"strings"

	"tedna/internal/models"
)

// buildCWAIReviewBatches 根据页面顺序建立待执行批次。
func buildCWAIReviewBatches(
	pages []models.CWAIReviewPageDigest,
	initialLedgerJSON string,
) ([]*models.CoursewareAIReviewBatch, error) {
	if len(pages) == 0 {
		return []*models.CoursewareAIReviewBatch{}, nil
	}

	batches := make(
		[]*models.CoursewareAIReviewBatch,
		0,
		(len(pages)+cwAIReviewBatchTargetPages-1)/
			cwAIReviewBatchTargetPages,
	)

	start := 0
	batchNo := 1

	for start < len(pages) {
		end, reason := chooseCWAIReviewBatchEnd(
			pages,
			start,
		)
		if end <= start {
			end = start + 1
		}
		if end > len(pages) {
			end = len(pages)
		}

		current := pages[start:end]
		pageNumbers := make(
			[]int,
			0,
			len(current),
		)
		pageIDs := make(
			[]string,
			0,
			len(current),
		)
		pageHashes := make(map[string]string)

		for _, page := range current {
			pageNumbers = append(
				pageNumbers,
				page.PageNumber,
			)
			pageIDs = append(
				pageIDs,
				page.PageID,
			)
			pageHashes[fmt.Sprintf(
				"%d",
				page.PageNumber,
			)] = page.HTMLHash
		}

		overlap := batchNo > 1

		scope := map[string]interface{}{
			"batch_no":              batchNo,
			"start_page":            current[0].PageNumber,
			"end_page":              current[len(current)-1].PageNumber,
			"page_numbers":          pageNumbers,
			"page_ids":              pageIDs,
			"overlap_from_previous": overlap,
			"boundary_reason":       reason,
		}

		manifest := map[string]interface{}{
			"page_numbers": pageNumbers,
			"page_hashes":  pageHashes,
			"content_mode": map[string]interface{}{
				"visible_text":               true,
				"page_scheme":                true,
				"interaction_events":         true,
				"reachable_script_functions": true,
				"dom_targets":                true,
				"css_visibility_rules":       true,
				"answer_exposure_signals":    true,
			},
			"continuity_source": func() string {
				if batchNo == 1 {
					return "session_initial_ledger"
				}
				return "previous_batch_continuity_after"
			}(),
		}

		scopeJSON, err := json.Marshal(scope)
		if err != nil {
			return nil, fmt.Errorf(
				"序列化第%d批页面范围失败: %w",
				batchNo,
				err,
			)
		}

		manifestJSON, err := json.Marshal(manifest)
		if err != nil {
			return nil, fmt.Errorf(
				"序列化第%d批输入清单失败: %w",
				batchNo,
				err,
			)
		}

		continuityBefore := "{}"
		if batchNo == 1 {
			continuityBefore = initialLedgerJSON
		}

		batch := &models.CoursewareAIReviewBatch{
			BatchNo:       batchNo,
			PageScopeJSON: string(scopeJSON),
			Status:        models.CWAIReviewBatchPending,
			InputHash: cwAIReviewHash(
				string(manifestJSON) +
					string(scopeJSON),
			),
			ContinuityBeforeJSON: continuityBefore,
			InputManifestJSON:    string(manifestJSON),
			ResultJSON:           "{}",
			ContinuityAfterJSON:  "{}",
			RiskPagesJSON:        "[]",
		}

		batches = append(batches, batch)

		if end >= len(pages) {
			break
		}

		nextStart := end - 1
		if nextStart <= start {
			nextStart = end
		}

		start = nextStart
		batchNo++
	}

	return batches, nil
}

// chooseCWAIReviewBatchEnd 选择当前批次结束位置。
func chooseCWAIReviewBatchEnd(
	pages []models.CWAIReviewPageDigest,
	start int,
) (int, string) {
	maxEnd := start + cwAIReviewBatchMaxPages
	if maxEnd > len(pages) {
		maxEnd = len(pages)
	}

	targetEnd := start + cwAIReviewBatchTargetPages
	if targetEnd > maxEnd {
		targetEnd = maxEnd
	}

	minEnd := start + cwAIReviewBatchMinPages
	if minEnd > maxEnd {
		return maxEnd,
			"剩余页面不足最小批次，全部纳入"
	}

	for end := minEnd; end <= maxEnd; end++ {
		if end >= len(pages) {
			return len(pages), "课件结束"
		}

		nextTitle := strings.TrimSpace(
			pages[end].Title,
		)
		if isCWAIReviewSectionStart(nextTitle) {
			return end,
				"下一页进入新的教学环节"
		}
	}

	return targetEnd, "达到目标批次页数"
}

// isCWAIReviewSectionStart 判断标题是否表示新教学环节。
func isCWAIReviewSectionStart(
	title string,
) bool {
	title = strings.ToLower(
		strings.TrimSpace(title),
	)
	if title == "" {
		return false
	}

	markers := []string{
		"导入",
		"情境",
		"学习目标",
		"教学目标",
		"新知",
		"概念",
		"探究",
		"实验",
		"例题",
		"案例",
		"练习",
		"巩固",
		"测验",
		"评价",
		"总结",
		"小结",
		"作业",
		"拓展",
		"reflection",
		"summary",
		"practice",
		"quiz",
	}

	for _, marker := range markers {
		if strings.Contains(title, marker) {
			return true
		}
	}

	return false
}

// cwAIReviewInitialContinuityLedger 创建第一批使用的空连续性账本。
func cwAIReviewInitialContinuityLedger() map[string]interface{} {
	return map[string]interface{}{
		"version": 1,
		"teaching_thread": map[string]interface{}{
			"current_stage":           "",
			"established_conclusions": []interface{}{},
			"open_questions":          []interface{}{},
			"next_expected_step":      "",
		},
		"cases":             []interface{}{},
		"terms":             []interface{}{},
		"formulas":          []interface{}{},
		"symbols":           []interface{}{},
		"interaction_state": []interface{}{},
		"continuity_risks":  []interface{}{},
		"reviewed_pages":    []interface{}{},
	}
}
