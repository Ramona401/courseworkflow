package services

// lesson_plan_gen_actions.go — 迭代3.5 Phase B：对话式备课「建议芯片(suggested_actions)」协议
//
// ============================================================================
// 这是什么
// ----------------------------------------------------------------------------
// 唤起式对话备课中，AI 每轮回复后可附带 2-4 个「建议芯片」(suggested_actions)，
// 老师点芯片即可推进（无需打字）。本文件负责后端侧三件事：
//   1. 从 AI 回复文本中解析出 suggested_actions 结构化块；
//   2. 把该块从展示文本中剥离（老师不该看到一坨 JSON）；
//   3. （广播由调用方 processChatStageAsync 完成，事件类型见 models.LPSSESuggestedActions）
//
// ============================================================================
// 为什么不复用 review 的 extractLastJSONCodeBlock —— 关键架构决策（务必理解）
// ----------------------------------------------------------------------------
// 线上「review 阶段」已有一套 ```json 块契约：extractReviewStageFromNatural →
// parseReviewJSONBlock → extractLastJSONCodeBlock 抓「最后一个 ```json 块」当评审结果
// (total_score/dimensions/...)。这是 4.3 输出契约红线，不能破坏。
//
// 若 suggested_actions 也用裸 ```json 块并放在回复末尾，则在 review 阶段 AI 会同时输出
// 两个 json 块（评审块 + 芯片块），extractLastJSONCodeBlock 取「最后一个」会抓到芯片块，
// total_score 缺失 → 评审解析失败降级 → 评审契约被打破。
//
// 因此本协议用「专用围栏」而非「裸 json + 末尾位置」来区分：
//   - 芯片块必须用 ```suggested_actions 围栏（自定义 info-string）；
//   - 且块内 JSON 必须含顶层 "suggested_actions" 数组键。
// 这样两套协议互不侵犯：
//   * review 的正则是 ```(?:json)?，info-string 为 suggested_actions 时不匹配（验证见下），
//     故评审解析「看不见」芯片块；
//   * 本解析器只认 ```suggested_actions 围栏 + suggested_actions 键，故「看不见」评审块。
//
// 围栏不被 review 正则匹配的原因（已逐字核对线上正则 "(?s)```(?:json)?\\s*\\n(.*?)```"）：
//   对 ```suggested_actions\n...，三反引号后 (?:json)? 匹配空，\s* 匹配零空白，
//   随后要求 \n，但实际下一个字符是 's'(suggested_actions 的首字母) → 整体不匹配。
//
// ============================================================================
// 安全底线（协议增强、缺了不阻塞）
// ----------------------------------------------------------------------------
//   - 解析失败 / 无块 / 类型非法 → 返回 nil，调用方不广播芯片事件，正文照常展示；
//   - action_type 只允许协议正式五种枚举，未知类型整条丢弃（不阻塞其余合法芯片）；
//   - 前端有固定剧本常量芯片兜底，本块只是「动态覆盖」，永不成为必需路径。
// ============================================================================

import (
	"encoding/json"
	"regexp"
	"strings"

	"tedna/internal/models"
)

// ==================== 协议常量 ====================

// saActionTypeWhitelist 芯片协议正式 action_type 五种枚举（与前端 conversationScript.ts 的
// ChipActionType 前五种逐字对齐）。AI 动态芯片只允许这五种；advance_stage/publish/focus_input
// 是前端内部扩展，仅剧本常量可用，AI 不得动态下发，故不在白名单内。
var saActionTypeWhitelist = map[string]bool{
	"send_text":         true,
	"full_generate":     true,
	"switch_stage":      true,
	"open_tool":         true,
	"confirm_structure": true,
}

// saMaxChips 单轮最多采纳的芯片数（设计 2.3：2-4 条；上限取 4，多余截断防 AI 失控刷屏）
const saMaxChips = 4

// saFenceRegex 匹配「专用围栏」```suggested_actions\n ... \n```
// 非贪婪、跨行；info-string 固定为 suggested_actions，与 review 的 ```(?:json)? 互斥。
var saFenceRegex = regexp.MustCompile("(?s)```suggested_actions\\s*\\n(.*?)```")

// ==================== 数据结构 ====================

// suggestedActionsEnvelope AI 输出块的外层结构 {"suggested_actions":[...]}
type suggestedActionsEnvelope struct {
	SuggestedActions []models.SuggestedAction `json:"suggested_actions"`
}

// ==================== 解析 ====================

// ParseSuggestedActions 从 AI 回复文本中解析建议芯片。
//
// 返回：
//   - actions：解析并清洗后的合法芯片（可能为空切片）；
//   - ok：是否成功解析到「至少一条合法芯片」。ok=false 时调用方不应广播芯片事件。
//
// 解析步骤：
//  1. 用专用围栏正则定位 ```suggested_actions 块（取最后一个，容忍正文里偶现同名词）；
//  2. json.Unmarshal 到 envelope，要求含 suggested_actions 数组；
//  3. 逐条清洗：label 必填、action_type 必须在白名单、id 缺失则按序补 sa_N；
//  4. 截断到 saMaxChips 条。
//
// 任一步失败均返回 (nil,false)，绝不 panic、绝不阻塞。
func ParseSuggestedActions(content string) ([]models.SuggestedAction, bool) {
	block := extractSuggestedActionsBlock(content)
	if block == "" {
		return nil, false
	}

	var env suggestedActionsEnvelope
	if err := json.Unmarshal([]byte(block), &env); err != nil {
		return nil, false
	}
	if len(env.SuggestedActions) == 0 {
		return nil, false
	}

	cleaned := make([]models.SuggestedAction, 0, len(env.SuggestedActions))
	for i, a := range env.SuggestedActions {
		label := strings.TrimSpace(a.Label)
		actionType := strings.TrimSpace(a.ActionType)
		if label == "" {
			continue // label 是芯片唯一可见文案，缺失直接丢弃该条
		}
		if !saActionTypeWhitelist[actionType] {
			continue // 未知 action_type 丢弃该条，不阻塞其余合法芯片
		}
		id := strings.TrimSpace(a.ID)
		if id == "" {
			id = saGenChipID(i)
		}
		cleaned = append(cleaned, models.SuggestedAction{
			ID:         id,
			Emoji:      strings.TrimSpace(a.Emoji),
			Label:      label,
			ActionType: actionType,
			Payload:    a.Payload,
		})
		if len(cleaned) >= saMaxChips {
			break
		}
	}

	if len(cleaned) == 0 {
		return nil, false
	}
	return cleaned, true
}

// extractSuggestedActionsBlock 提取「最后一个」```suggested_actions 块的内容。
// 取最后一个：与评审 JSON 约定在末尾一致，且容忍 AI 在正文中偶然提及该词。
func extractSuggestedActionsBlock(text string) string {
	matches := saFenceRegex.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return ""
	}
	last := matches[len(matches)-1]
	if len(last) < 2 {
		return ""
	}
	candidate := strings.TrimSpace(last[1])
	if !strings.HasPrefix(candidate, "{") {
		return ""
	}
	return candidate
}

// ==================== 剥离 ====================

// StripSuggestedActionsBlock 从展示文本中移除所有 ```suggested_actions 块，
// 让老师在对话气泡里看不到这坨 JSON。剥离后做尾部空白清理。
//
// 注意：只剥离专用围栏块，绝不动 ```json（评审块）等其它围栏，互不干扰。
func StripSuggestedActionsBlock(content string) string {
	if content == "" {
		return content
	}
	stripped := saFenceRegex.ReplaceAllString(content, "")
	// 清理因剥离遗留的尾部多余空行与分隔线（AI 常在芯片块前写 "---"）
	stripped = strings.TrimRight(stripped, " \t\n")
	stripped = strings.TrimSuffix(strings.TrimRight(stripped, " \t\n"), "---")
	return strings.TrimRight(stripped, " \t\n")
}

// saGenChipID 为缺失 id 的芯片按序生成稳定 id（sa_1/sa_2...）
func saGenChipID(idx int) string {
	switch idx {
	case 0:
		return "sa_1"
	case 1:
		return "sa_2"
	case 2:
		return "sa_3"
	case 3:
		return "sa_4"
	default:
		// 理论上不会到这里（已被 saMaxChips 截断），兜底
		return "sa_x"
	}
}

// ==================== 提示词片段常量（B-3 才注入五阶段，B-1 仅定义留用）====================

// SuggestedActionsPromptFragment 是追加到五阶段 system_prompt 末尾的「建议动作输出规则」。
//
// B-1 仅定义此常量、暂不注入任何阶段（线上行为零变化）。B-3 阶段会经
// workshop_stages 表的版本管理（GetPromptVersions 可回滚）逐阶段把对应内容并入提示词，
// 并非由代码强行拼接——此常量作为「权威文案源」供 B-3 抄写，确保前后端与培训流程卡一致。
//
// 关键约束（写进文案，约束 AI）：
//   - 专用围栏 ```suggested_actions，块内含 "suggested_actions" 数组键；
//   - 块必须是回复最后部分，块外不得再出现该块；
//   - action_type 仅五种正式枚举；label ≤ 8 字、口语化；2-4 条；
//   - review 阶段特别说明：评审 JSON 块在前、本芯片块在后，两块各自独立围栏。
const SuggestedActionsPromptFragment = `

== 建议动作输出规则（系统级，老师无需关心，供界面生成可点选芯片）==
在你本轮回复的正文全部输出完毕后，另起一行追加一个「建议动作」代码块，用 ` + "```suggested_actions" + ` 围栏包裹（注意围栏标识就是 suggested_actions，不是 json），块内是严格 JSON，格式如下：

` + "```suggested_actions" + `
{
  "suggested_actions": [
    {"id": "continue", "emoji": "✅", "label": "确认，继续", "action_type": "send_text", "payload": {"text": "确认无误，请继续下一步"}},
    {"id": "revise", "emoji": "✏️", "label": "我要调整", "action_type": "send_text", "payload": {"text": "我想调整一下，"}}
  ]
}
` + "```" + `

硬性要求：
1. 该代码块必须是整条回复的最后部分，块之外不得再出现任何 suggested_actions 块。
2. 只给 2-4 条，每条 label 不超过 8 个字、口语化（像老师会说的话）。
3. action_type 只能是以下五种之一：
   - send_text：把 payload.text 当作老师的话发送（最常用，用于"确认继续/我要改"等）
   - full_generate：一键直接出完整教案（payload 可含 {"stage":"write"}）
   - switch_stage：跳到某阶段（payload 含 {"stage":"..."}）
   - open_tool：唤起某能力（payload 含 {"tool":"components"} 等）
   - confirm_structure：确认结构卡（暂按 send_text 处理）
4. 芯片是"建议"不是"必选"，请给当前最自然的下一步动作，不要硬凑。
`

// 说明：上面常量中嵌入围栏用字符串拼接（"```suggested_actions" 等）以避免 Go 原始字符串
// 反引号与 Markdown 围栏冲突，编译期即为最终文案。
