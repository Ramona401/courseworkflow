package services

// generator_helpers.go — Generator 页面生成辅助函数
//
// 从 generator_service.go 拆分,包含:
//   - parsePageOps: 解析逐页修改指令
//   - detectOperation/opPriority/extractMergeSources: 操作类型检测
//   - buildModifyUserPrompt/buildCreateUserPrompt/buildMergeUserPrompt: Prompt构建
//   - extractTransFinalOutput/extractGeneratedHTML: 文本提取
//   - findReferencePageHTML/collectMergeSourceHTMLs: HTML获取

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"tedna/internal/models"
)

// ==================== 解析逐页修改指令 ====================
//
// 设计原则：
//   1. 只解析"逐页修改指令"区域（由extractPageOpsSection提取）
//   2. 每页的【原始页码】字段是取OSS内容的唯一依据
//   3. 完全不使用origCursor指针法，不自动补充任何页面
//   4. Translator必须输出所有页面的完整指令
//
// 虚拟页码规则：
//   新增页（create）page_number = 新课件位置 + 1000
//   例：在P09之后新增P10 → page_number=1010
//   排序时1010自然排在9之后11之前，前端按此顺序展示
func parsePageOps(transOutput string) []*models.PageOperation {
	var ops []*models.PageOperation
	opsMap := make(map[int]*models.PageOperation)
	createCounterMap := make(map[int]int)

	lines := strings.Split(transOutput, "\n")

	opKeywords := map[string]string{
		"保留": models.PageOpKeep, "keep": models.PageOpKeep, "无变化": models.PageOpKeep,
		"修改": models.PageOpModify, "modify": models.PageOpModify,
		"新增": models.PageOpCreate, "新建": models.PageOpCreate, "create": models.PageOpCreate, "插入": models.PageOpCreate,
		"合并": models.PageOpMerge, "merge": models.PageOpMerge,
		"删除": models.PageOpDelete, "delete": models.PageOpDelete, "移除": models.PageOpDelete,
	}

	pageHeaderRe := regexp.MustCompile(`(?m)^-*\s*P(\d{1,2})\s+(.+?)(?:\s*【|$)`)
	origPageFieldRe := regexp.MustCompile(`【原始页码】\s*(?:P|p)(\d{1,3})(?:\s*[+＋]\s*(?:P|p)(\d{1,3}))?`)
	origPageNoneRe := regexp.MustCompile(`【原始页码】\s*无`)

	var currentPage int
	var currentTitle string
	var currentDesc []string

	flushPage := func() {
		if currentPage <= 0 {
			return
		}
		desc := strings.Join(currentDesc, "\n")
		op := detectOperation(desc, opKeywords)

		var mergeSources []int
		if op == models.PageOpMerge {
			mergeSources = extractMergeSources(desc, currentPage)
		}

		// 从【原始页码】字段直接解析OriginalPageNumber
		// 这是取OSS内容的唯一依据，不使用任何指针法推算
		origPageNum := 0
		isCreateOp := false

		if origPageNoneRe.MatchString(desc) {
			// 【原始页码】无 → 纯新增页面
			isCreateOp = true
			origPageNum = 0
		} else if m := origPageFieldRe.FindStringSubmatch(desc); m != nil {
			// 【原始页码】P03 或 【原始页码】P03+P04
			if parsed, err := strconv.Atoi(m[1]); err == nil && parsed > 0 {
				origPageNum = parsed
			}
			if op == models.PageOpMerge && m[2] != "" {
				if parsed2, err := strconv.Atoi(m[2]); err == nil && parsed2 > 0 {
					mergeSources = []int{origPageNum, parsed2}
				}
			}
		} else {
			// 没有【原始页码】字段：兼容旧格式，用当前页码
			origPageNum = currentPage
		}

		if op == models.PageOpCreate || isCreateOp {
			// 新增页面使用虚拟页码
			createCounterMap[currentPage]++
			counter := createCounterMap[currentPage]
			virtualPageNum := currentPage + createPageOffset + (counter-1)*10

			newTitle := fmt.Sprintf("P%02d-new %s", currentPage, currentTitle)
			if len([]rune(newTitle)) > 60 {
				newTitle = string([]rune(newTitle)[:60])
			}

			opsMap[virtualPageNum] = &models.PageOperation{
				PageNumber:         virtualPageNum,
				OriginalPageNumber: 0,
				Operation:          models.PageOpCreate,
				Title:              newTitle,
				Description:        desc,
			}
		} else {
			if existing, ok := opsMap[currentPage]; ok {
				existing.Description += "\n" + desc
				if opPriority(op) > opPriority(existing.Operation) {
					existing.Operation = op
				}
				if len(mergeSources) > 0 {
					existing.MergeSources = mergeSources
				}
				if origPageNum > 0 {
					existing.OriginalPageNumber = origPageNum
				}
			} else {
				opsMap[currentPage] = &models.PageOperation{
					PageNumber:         currentPage,
					OriginalPageNumber: origPageNum,
					Operation:          op,
					Title:              currentTitle,
					Description:        desc,
					MergeSources:       mergeSources,
				}
			}
		}
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// 跳过表格行（含3个以上"|"）
		if strings.Count(trimmed, "|") >= 3 {
			continue
		}

		// 跳过纯分隔线
		if len(trimmed) >= 2 {
			allSep := true
			for _, ch := range trimmed {
				if ch != '-' && ch != '=' && ch != '＝' {
					allSep = false
					break
				}
			}
			if allSep {
				continue
			}
		}

		// 检测页面标题行
		if m := pageHeaderRe.FindStringSubmatch(trimmed); m != nil {
			flushPage()
			pn, _ := strconv.Atoi(m[1])
			currentPage = pn
			currentTitle = strings.TrimSpace(m[2])
			currentTitle = regexp.MustCompile(`\s*【.*】.*$`).ReplaceAllString(currentTitle, "")
			currentTitle = strings.TrimSpace(currentTitle)
			if len([]rune(currentTitle)) > 60 {
				currentTitle = string([]rune(currentTitle)[:60])
			}
			currentDesc = []string{trimmed}
			continue
		}

		// 支持纯页码格式 "P18"
		if m := regexp.MustCompile(`^P(\d{1,2})\b`).FindStringSubmatch(trimmed); m != nil {
			if currentPage > 0 {
				flushPage()
			}
			pn, _ := strconv.Atoi(m[1])
			currentPage = pn
			currentTitle = trimmed
			currentDesc = []string{trimmed}
			continue
		}

		if currentPage > 0 {
			currentDesc = append(currentDesc, trimmed)
		}
	}
	flushPage()

	// 按页码排序（虚拟页码1010排在10之后11之前，即最终课件顺序）
	for _, op := range opsMap {
		ops = append(ops, op)
	}
	sort.Slice(ops, func(i, j int) bool {
		return ops[i].PageNumber < ops[j].PageNumber
	})

	return ops
}

// detectOperation 从描述文本中检测操作类型
func detectOperation(desc string, opKeywords map[string]string) string {
	opLineRe := regexp.MustCompile(`【操作】\s*(.+)`)
	if m := opLineRe.FindStringSubmatch(desc); m != nil {
		opText := strings.ToLower(strings.TrimSpace(m[1]))
		for kw, op := range opKeywords {
			if strings.Contains(opText, kw) {
				return op
			}
		}
	}
	headerOpRe := regexp.MustCompile(`【(修改|保留|新增|新建|删除|合并)】`)
	if m := headerOpRe.FindStringSubmatch(desc); m != nil {
		if op, ok := opKeywords[m[1]]; ok {
			return op
		}
	}
	if strings.Contains(desc, "✅无变化") || strings.Contains(desc, "✅ 无变化") {
		return models.PageOpKeep
	}
	return models.PageOpKeep
}

// opPriority 操作优先级
func opPriority(op string) int {
	switch op {
	case models.PageOpDelete:
		return 5
	case models.PageOpMerge:
		return 4
	case models.PageOpCreate:
		return 3
	case models.PageOpModify:
		return 2
	case models.PageOpKeep:
		return 1
	default:
		return 0
	}
}

// extractMergeSources 从描述中提取merge源页码
func extractMergeSources(desc string, currentPage int) []int {
	re := regexp.MustCompile(`(?i)(?:合并|merge)\s*(?:P|页面?)(\d+)\s*[+和与&,，]\s*(?:P|页面?)(\d+)`)
	if m := re.FindStringSubmatch(desc); m != nil {
		s1, _ := strconv.Atoi(m[1])
		s2, _ := strconv.Atoi(m[2])
		return []int{s1, s2}
	}
	return nil
}

// ==================== Prompt构建函数 ====================

func buildModifyUserPrompt(courseCode string, op *models.PageOperation, origHTML string) string {
	var parts []string
	parts = append(parts,
		"【⚠️⚠️⚠️ 最重要 — 原始页面HTML，你必须在此基础上修改，禁止重写】",
		"以下是当前线上运行的完整HTML。你的输出必须以此为基础，只修改下方指令要求的部分，其余代码原封不动保留。",
		"",
		origHTML,
		"",
		"══════════════════════════════════════════════",
		"▲ 以上是原始HTML（必须作为修改基础） ▼ 以下是修改指令",
		"══════════════════════════════════════════════",
		"",
		fmt.Sprintf("【课程编号】%s", courseCode),
		fmt.Sprintf("【页面】P%02d — %s", op.PageNumber, op.Title),
		"【操作类型】modify（在原始HTML基础上修改）",
		"",
		"【Translator修改指令（必须严格执行）】",
		op.Description,
		"",
		"【⚠️ 最终提醒】你的输出必须与上方原始HTML有90%以上代码重合。只改指令要求的部分。导航栏、视频、图片不允许任何改动。输出完整HTML。",
	)
	return strings.Join(parts, "\n")
}

func buildCreateUserPrompt(courseCode string, op *models.PageOperation, refHTML string) string {
	var parts []string
	if refHTML != "" {
		parts = append(parts,
			"【⚠️⚠️⚠️ 格式参考页面 — 新建页面必须完全模仿此格式】",
			"以下是相邻页面的完整HTML。你必须：",
			"1. 完全复制其HTML结构、CSS样式、导航栏、配色方案、布局框架",
			"2. 只替换内容区域的文字和互动元素",
			"3. 导航栏100%原样复制",
			"",
			refHTML,
			"",
			"══════════════════════════════════════════════",
			"▲ 以上是格式参考页面 ▼ 以下是新页面内容要求",
			"══════════════════════════════════════════════",
			"",
		)
	} else {
		parts = append(parts,
			"【⚠️ 无可用参考页面，请生成完整独立HTML页面】",
			"要求：白色背景，100vh一屏，文字最小22px，自包含HTML。",
			"",
		)
	}
	parts = append(parts,
		fmt.Sprintf("【课程编号】%s", courseCode),
		fmt.Sprintf("【页面】%s（新增页面）", op.Title),
		"【操作类型】create（新建页面）",
		"",
		"【Translator创建指令（必须严格执行）】",
		op.Description,
		"",
		"【⚠️ 最终提醒】新建页面必须与参考页面的格式、布局、导航栏、CSS完全一致。只有内容区域不同。输出完整自包含HTML。",
	)
	return strings.Join(parts, "\n")
}

type sourcePageHTML struct {
	pageNum  int
	html     string
	lessonID *int
}

func buildMergeUserPrompt(courseCode string, op *models.PageOperation, sources []sourcePageHTML) string {
	var parts []string
	parts = append(parts,
		fmt.Sprintf("【⚠️⚠️⚠️ 合并任务 — 需要将以下%d个页面合并为1个页面】", len(sources)),
		"",
	)
	for i, src := range sources {
		parts = append(parts,
			fmt.Sprintf("═══ 源页面 %d/%d: P%02d ═══", i+1, len(sources), src.pageNum),
			src.html,
			"",
		)
	}
	parts = append(parts,
		"══════════════════════════════════════════════",
		"▲ 以上是需要合并的所有源页面 ▼ 以下是合并要求",
		"══════════════════════════════════════════════",
		"",
		"【合并规则】",
		fmt.Sprintf("1. 以第一个源页面(P%02d)的HTML结构、导航栏、CSS为基础", sources[0].pageNum),
		"2. 将其他源页面的核心内容整合进来",
		"3. 导航栏只保留一份",
		"4. 所有源页面的视频、图片等资产原位保留",
		"5. 合并后仍为一屏(100vh)，内容放不下用标签页切换或折叠面板",
		"",
		fmt.Sprintf("【课程编号】%s", courseCode),
		fmt.Sprintf("【目标页面】P%02d — %s", op.PageNumber, op.Title),
		"【操作类型】merge",
		"",
		"【Translator合并指令】",
		op.Description,
		"",
		"【⚠️ 最终提醒】合并后页面必须保留所有源页面的视频和图片资产。导航栏只保留一份。一屏展示。输出完整HTML。",
	)
	return strings.Join(parts, "\n")
}

// ==================== 辅助工具函数 ====================

func extractTransFinalOutput(stepData string) string {
	if stepData == "" || stepData == "null" {
		return ""
	}
	var data map[string]json.RawMessage
	if err := json.Unmarshal([]byte(stepData), &data); err != nil {
		return ""
	}
	raw, ok := data["final_trans_output"]
	if !ok || string(raw) == "null" {
		return ""
	}
	var output string
	if err := json.Unmarshal(raw, &output); err != nil {
		return strings.Trim(string(raw), `"`)
	}
	return output
}

func extractGeneratedHTML(aiOutput string) string {
	codeBlockRe := regexp.MustCompile("(?s)```html\\s*\n?(.*?)```")
	if m := codeBlockRe.FindStringSubmatch(aiOutput); m != nil {
		return strings.TrimSpace(m[1])
	}
	doctypeRe := regexp.MustCompile(`(?si)(<(!DOCTYPE|html)[\s\S]*)`)
	if m := doctypeRe.FindStringSubmatch(aiOutput); m != nil {
		return strings.TrimSpace(m[1])
	}
	return strings.TrimSpace(aiOutput)
}

func findReferencePageHTML(ossService *OSSService, pageLessonMap map[int]int, pageNum int) string {
	for n := pageNum - 1; n >= 1; n-- {
		if lid, ok := pageLessonMap[n]; ok {
			html, err := ossService.FetchLessonHTML(lid)
			if err == nil && len(html) > 200 {
				return html
			}
		}
	}
	for n := pageNum + 1; n <= pageNum+5; n++ {
		if lid, ok := pageLessonMap[n]; ok {
			html, err := ossService.FetchLessonHTML(lid)
			if err == nil && len(html) > 200 {
				return html
			}
		}
	}
	for _, lid := range pageLessonMap {
		html, err := ossService.FetchLessonHTML(lid)
		if err == nil && len(html) > 200 {
			return html
		}
	}
	return ""
}

func collectMergeSourceHTMLs(ossService *OSSService, pageLessonMap map[int]int, op *models.PageOperation) []sourcePageHTML {
	var sources []sourcePageHTML
	if len(op.MergeSources) >= 2 {
		for _, pn := range op.MergeSources {
			if lid, ok := pageLessonMap[pn]; ok {
				html, err := ossService.FetchLessonHTML(lid)
				if err == nil && len(html) > 100 {
					lidPtr := new(int)
					*lidPtr = lid
					sources = append(sources, sourcePageHTML{pageNum: pn, html: html, lessonID: lidPtr})
				}
			}
		}
		if len(sources) >= 2 {
			return sources
		}
	}
	sources = nil
	origPn := op.GetOrigPageNum()
	if lid, ok := pageLessonMap[origPn]; ok {
		html, err := ossService.FetchLessonHTML(lid)
		if err == nil && len(html) > 100 {
			lidPtr := new(int)
			*lidPtr = lid
			sources = append(sources, sourcePageHTML{pageNum: origPn, html: html, lessonID: lidPtr})
		}
	}
	nextPn := origPn + 1
	if lid, ok := pageLessonMap[nextPn]; ok {
		html, err := ossService.FetchLessonHTML(lid)
		if err == nil && len(html) > 100 {
			lidPtr := new(int)
			*lidPtr = lid
			sources = append(sources, sourcePageHTML{pageNum: nextPn, html: html, lessonID: lidPtr})
		}
	}
	return sources
}
