package services

// assistant_designer_parse.go — AI助手创作服务解析函数
//
// 从 assistant_designer_service.go 拆分,包含:
//   - parseAIDecision: AI决策JSON解析(四重兜底)
//   - extractFieldsLenient: 字段级宽容提取
//   - extractLongFieldValue: 长字段值提取

import (
	"encoding/json"
	"log"
	"fmt"
	"regexp"
	"strings"

	"tedna/internal/ai"
	"tedna/internal/models"
	"tedna/internal/utils"
)



// ============================================================================
//
// 策略 1:ai.ExtractJSON(标准处理,含 markdown 代码块)
// 策略 2:手动补开头缺失的大括号(应对 AI 偶发漏开头 { 的情况)
// 策略 3:整段文本直接反序列化(末位防线-依然依赖 JSON 转义合法)
// 策略 4:v114 新增 - 字段级宽容提取(不依赖 json.Unmarshal)
//        用正则按字段位置手动切割 action / reply_text / updated_draft,
//        专治 AI 在字段值里写了未转义 " 引号导致 json.Unmarshal 全挂的场景
//        (例如 "reply_text": "助手的"脾气"..." 这种经典 LLM 输出 bug)
//
// 背景:LLM 返回 JSON 时偶发问题:
//   - 包在代码块里(ExtractJSON 已处理)
//   - 漏第一个 {(例如返回 '\n  "action": "clarify"...')
//   - 末尾多余文字或前导解释
//   - **字段值里包含未转义的 " 单引号** ← 这是最常见最致命的

func parseAIDecision(raw string) (*AIDesignerDecision, error) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return nil, fmt.Errorf("空响应")
	}

	// 策略 1:标准提取
	if jsonStr, ok := ai.ExtractJSON(text); ok {
		var decision AIDesignerDecision
		if err := json.Unmarshal([]byte(jsonStr), &decision); err == nil {
			if decision.Action == "" {
				decision.Action = "draft_directly"
			}
			return &decision, nil
		}
	}

	// 策略 2:手动补大括号
	// 判据:文本含 "action" 字段但没以 { 开头
	openBrace := "{"
	closeBrace := "}"
	if !strings.HasPrefix(text, openBrace) && strings.Contains(text, "\"action\"") {
		fixed := text
		if !strings.HasSuffix(strings.TrimSpace(fixed), closeBrace) {
			fixed = fixed + "\n" + closeBrace
		}
		fixed = openBrace + "\n" + fixed
		var decision AIDesignerDecision
		if err := json.Unmarshal([]byte(fixed), &decision); err == nil {
			log.Printf("[designer] 手动补大括号后解析成功")
			if decision.Action == "" {
				decision.Action = "draft_directly"
			}
			return &decision, nil
		}
	}

	// 策略 3:直接整段解析(末位防线,依赖 JSON 合法)
	var decision AIDesignerDecision
	if err := json.Unmarshal([]byte(text), &decision); err == nil {
		if decision.Action == "" {
			decision.Action = "draft_directly"
		}
		return &decision, nil
	}

	// 策略 4 (v114):字段级宽容提取 - 绕过 json.Unmarshal
	// 只在看起来像 JSON 的文本上触发(以 { 开头,含有 action 或 reply_text)
	if strings.HasPrefix(text, "{") && (strings.Contains(text, "\"action\"") || strings.Contains(text, "\"reply_text\"")) {
		d := extractFieldsLenient(text)
		if d != nil && strings.TrimSpace(d.ReplyText) != "" {
			log.Printf("[designer] 策略 4 字段级宽容提取成功:action=%s reply_len=%d draft_len=%d",
				d.Action, len(d.ReplyText), len(d.UpdatedDraft))
			return d, nil
		}
	}

	return nil, fmt.Errorf("所有 JSON 提取策略均失败")
}

// ============================================================================
// extractFieldsLenient - 策略 4 的核心实现
// ============================================================================
//
// 思路:不再尝试反序列化整个 JSON,而是**按字段名+冒号+引号**定位每个字段值的起点,
//       然后向后扫描直到找到该字段的"结束标志"(下一个顶层字段名出现前的最后一个 "),
//       这样字段值内部的任意 " 都不会破坏提取。
//
// 局限:
//   - query_params 是嵌套对象,不尝试解析(保持 nil),让调用方降级为 draft_directly
//   - 只支持字段顺序 action → query_params → reply_text → updated_draft 这种常见排列
//     (AI 几乎总是按 Meta-Prompt 里的示例顺序输出,否则我们的 Meta-Prompt 写错了)

var (
	// action 字段值相对简单:限定枚举值,不含引号
	reAction = regexp.MustCompile(`"action"\s*:\s*"([a-z_]+)"`)
)

func extractFieldsLenient(text string) *AIDesignerDecision {
	d := &AIDesignerDecision{
		Action:       "draft_directly", // 默认兜底
		QueryParams:  nil,               // 策略 4 不解析嵌套对象
		ReplyText:    "",
		UpdatedDraft: "",
	}

	// action: 简单正则
	if m := reAction.FindStringSubmatch(text); len(m) >= 2 {
		d.Action = m[1]
	}

	// reply_text: 宽容提取到下一个顶层字段之前
	d.ReplyText = extractLongFieldValue(text, "reply_text", []string{"updated_draft"})

	// updated_draft: 宽容提取到文本末尾的 } 之前
	d.UpdatedDraft = extractLongFieldValue(text, "updated_draft", nil)

	// 如果 action=search_components 但没有 query_params,降级为 draft_directly
	// 避免 handler 层试图查库时用 nil 的 query_params 引发空指针
	if d.Action == "search_components" && d.QueryParams == nil {
		d.Action = "draft_directly"
	}

	return d
}

// extractLongFieldValue 从 text 里按"字段名":"值" 定位字段,把值原文抽出来
// 终止条件:遇到 nextFields 中任一字段名的出现;或文本末尾的 } 之前
// 返回的字符串是字段值的原文(已做基本反转义:\n → 真换行,\" → ",\\ → \)
func extractLongFieldValue(text, field string, nextFields []string) string {
	// 找字段起点:"field_name" : "
	keyPattern := `"` + field + `"`
	keyIdx := strings.Index(text, keyPattern)
	if keyIdx < 0 {
		return ""
	}

	// 从 keyIdx 后找第一个 " (字段值的开引号)
	afterKey := keyIdx + len(keyPattern)
	quoteIdx := strings.Index(text[afterKey:], "\"")
	if quoteIdx < 0 {
		return ""
	}
	valueStart := afterKey + quoteIdx + 1 // 跳过开引号

	// 找字段值的终点:
	// - 如果 nextFields 非空,找第一个 "nextField" 出现的位置,然后回退到最近的 "
	// - 如果 nextFields 为空,找最后一个 " (在文本末尾 } 之前)
	valueEnd := -1

	if len(nextFields) > 0 {
		// 找最近的下一字段起点
		minNext := -1
		for _, nf := range nextFields {
			pat := `"` + nf + `"`
			idx := strings.Index(text[valueStart:], pat)
			if idx >= 0 {
				absIdx := valueStart + idx
				if minNext < 0 || absIdx < minNext {
					minNext = absIdx
				}
			}
		}
		if minNext > 0 {
			// 从 minNext 往前扫描,找最近的 "(这个 " 就是当前字段值的闭引号)
			// 跳过紧邻的 "," 或 "\n," 等结构
			sub := text[valueStart:minNext]
			lastQuote := strings.LastIndex(sub, "\"")
			if lastQuote >= 0 {
				valueEnd = valueStart + lastQuote
			}
		}
	}

	if valueEnd < 0 {
		// nextFields 为空或没找到,退到文本末尾找最后一个 "
		// 先把末尾的 } 和空白去掉
		trimmed := strings.TrimRight(text, " \t\n\r}")
		lastQuote := strings.LastIndex(trimmed, "\"")
		if lastQuote > valueStart {
			valueEnd = lastQuote
		}
	}

	if valueEnd <= valueStart {
		return ""
	}

	raw := text[valueStart:valueEnd]
	// 做基本反转义(AI 有时是转义了的,有时没有,都兼容一下)
	// 注意顺序:\\ 必须先转,否则会破坏 \" 和 \n 的处理
	raw = strings.ReplaceAll(raw, `\\`, "\x00") // 临时占位
	raw = strings.ReplaceAll(raw, `\"`, `"`)
	raw = strings.ReplaceAll(raw, `\n`, "\n")
	raw = strings.ReplaceAll(raw, `\t`, "\t")
	raw = strings.ReplaceAll(raw, `\r`, "\r")
	raw = strings.ReplaceAll(raw, "\x00", `\`) // 还原 \
	return raw
}

// ============================================================================
// 辅助:从 AI 的 query_params 构造 MatchComponentsRequest
// ============================================================================

func buildMatchRequestFromParams(
	params map[string]interface{},
	dCtx *DesignerContext,
) *models.MatchComponentsRequest {
	req := &models.MatchComponentsRequest{
		Subject:    dCtx.Subject,
		GradeRange: dCtx.Grade,
	}

	if v, ok := params["library_types"].([]interface{}); ok {
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				req.LibraryTypes = append(req.LibraryTypes, s)
			}
		}
	}

	req.CognitiveLevel = extractIntArray(params, "cognitive_levels")
	req.StageTiming = extractIntArray(params, "stage_timing")
	req.PedagogyIntensity = extractIntArray(params, "pedagogy_intensities")

	req.Limit = 3

	return req
}

func extractIntArray(m map[string]interface{}, key string) []int {
	v, ok := m[key].([]interface{})
	if !ok {
		return nil
	}
	result := make([]int, 0, len(v))
	for _, item := range v {
		if f, ok := item.(float64); ok {
			result = append(result, int(f))
		}
	}
	return result
}

// ============================================================================
// 辅助:扁平化 + 组件上下文拼接
// ============================================================================

func flattenMatchedGroups(
	groups []*models.MatchedComponentGroup,
	n int,
) []*flatComponent {
	flat := make([]*flatComponent, 0, n)
	for _, g := range groups {
		for _, c := range g.Components {
			if len(flat) >= n {
				return flat
			}
			flat = append(flat, &flatComponent{
				LibraryType: g.LibraryType,
				LibraryName: g.LibraryName,
				Data:        c,
			})
		}
	}
	return flat
}

func buildComponentContext(flatComps []*flatComponent, dCtx *DesignerContext) string {
	if len(flatComps) == 0 {
		return "(未查到相关组件)"
	}
	var b strings.Builder
	for i, fc := range flatComps {
		c := fc.Data
		b.WriteString(fmt.Sprintf("## [%d] %s\n", i+1, c.DisplayLabel))
		b.WriteString(fmt.Sprintf("- 类型:%s (%s)", fc.LibraryType, fc.LibraryName))
		if dCtx.Subject != "" {
			b.WriteString(fmt.Sprintf(" 学科:%s", dCtx.Subject))
		}
		if dCtx.Grade != "" {
			b.WriteString(fmt.Sprintf(" 学段:%s", dCtx.Grade))
		}
		b.WriteString("\n")

		if strings.TrimSpace(c.ComponentIndex) != "" {
			b.WriteString("- AOCI 索引:\n")
			b.WriteString(c.ComponentIndex)
			b.WriteString("\n")
		} else if strings.TrimSpace(c.DesignLogic) != "" {
			b.WriteString("- 设计逻辑:")
			b.WriteString(utils.SafeTruncate(c.DesignLogic, 400))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	return b.String()
}

func defaultStr(s, fallback string) string {
	if strings.TrimSpace(s) == "" {
		return fallback
	}
	return s
}
