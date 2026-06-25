package services

// unit_plan_match.go — 大单元备课·单元方案注入上下文构建（独立模块）
//
// 职责：把一份「已挂载、且 status=active」的单元方案（unit_plans 一行）
// 拼成喂给备课 AI 的「硬指令上下文块」，由 workshop_stage_service.go 的
// LoadStagePromptContextV2 在五个阶段（analyze/design/write/review/revise）注入。
//
// 与课程大纲（course_outline_match.go 的 BuildCourseOutlinesContext）的区别：
//   - 课程大纲：按学科+学段「自动匹配」一批，是「整册参考」，只在 analyze/design 注入。
//   - 单元方案：老师「显式挂载」一份，是「这堂课所属大单元的纲」，五阶段全程注入，
//     且在 analyze/design 两阶段「顶替」课程大纲（有大单元就不要大纲——Yuhan 决策）。
//
// 硬指令风格沿用 BuildCourseOutlinesContext 踩坑后的经验：明确要求 AI
// 「认领这份资料、不许说看不到、不许用旧记忆硬猜」，并强调「本堂课不得背离大单元整体设计」
// （对应 Yuhan 的要求：评测和修改都不能背离大单元）。
//
// 本文件不做任何匹配/打分（显式挂载无需匹配）——是否注入由调用方先用
// repository.GetUnitPlanByID 取出、校验 Status==active 后决定，本函数只负责「拼块」。

import (
	"fmt"
	"strings"

	"tedna/internal/models"
)

// unitPlanContextMaxRunes 单元方案注入上下文的总字符（rune）上限。
//
// 单元方案的 content（方案文档）+ atlas（图谱表格）可能较长，全量注入会撑爆
// system prompt 的 token。这里按 rune 截断（中文按字计，不会截断半个汉字），
// 与课本 OCR 注入、组件 L2 压缩注入的「有上限」一致。
// 取 6000 rune（约等于一份单元方案的核心内容），超出部分截断并加省略提示。
const unitPlanContextMaxRunes = 6000

// truncateRunesForUnitPlan 按 rune 安全截断（避免截断半个汉字），超长则加省略标记。
func truncateRunesForUnitPlan(s string, maxRunes int) string {
	r := []rune(s)
	if len(r) <= maxRunes {
		return s
	}
	return string(r[:maxRunes]) + "\n……（本单元方案内容较长，此处已截断，核心设计意图见上文）"
}

// BuildUnitPlanContext 把一份 active 单元方案拼成可注入的硬指令上下文块。
//
// 入参 up 必须是调用方已校验过 Status==active 的单元方案；本函数不再校验状态。
// 返回值为完整的上下文块（含首尾标记与硬指令）；若 up 为 nil 或方案/图谱都为空，返回空串
// （调用方据此决定不注入，不会拼一个空壳块进 prompt）。
func BuildUnitPlanContext(up *models.UnitPlan) string {
	if up == nil {
		return ""
	}

	content := strings.TrimSpace(up.Content)
	atlas := strings.TrimSpace(up.Atlas)
	// 方案文档与图谱都为空，说明这份单元方案没有实质内容，不注入。
	if content == "" && atlas == "" {
		return ""
	}

	// 标识信息：单元名 + 单元主题 + 标题，供 AI 认领这份资料。
	unit := strings.TrimSpace(up.Unit)
	theme := strings.TrimSpace(up.UnitTheme)
	title := strings.TrimSpace(up.Title)

	var b strings.Builder
	b.WriteString("\n\n【本课所属大单元整体设计（老师已显式挂载，必须遵循）】\n")
	b.WriteString("以下是本节课所属「大单元」的整体教学设计方案。这是老师为这一整个单元亲自定下的顶层设计，\n")
	b.WriteString("本节课是该大单元的一个课时，必须服从大单元的整体目标、主线与设计意图。\n")
	b.WriteString("重要要求：\n")
	b.WriteString("  1. 本节课的教学分析、教学设计、教案撰写、评审与修订，都不得背离下面的大单元整体设计；\n")
	b.WriteString("  2. 评审时若发现本课偏离了大单元的主线或目标，应明确指出；修订时应向大单元设计靠拢；\n")
	b.WriteString("  3. 这是老师亲自挂载的真实资料，请直接阅读并采用，不要说\"看不到资料\"，也不要凭旧记忆另行编造单元设计。\n")

	// 标识行
	if unit != "" {
		b.WriteString(fmt.Sprintf("\n---- 单元：%s ----\n", unit))
	} else {
		b.WriteString("\n---- 单元整体设计 ----\n")
	}
	if theme != "" {
		b.WriteString(fmt.Sprintf("单元主题：%s\n", theme))
	}
	if title != "" {
		b.WriteString(fmt.Sprintf("方案标题：%s\n", title))
	}

	// 主体：方案文档 + 图谱表格，合并后统一按 rune 上限截断。
	// 合并而非分别截断，保证总长度可控；方案文档优先（图谱是辅助呈现）。
	var body strings.Builder
	if content != "" {
		body.WriteString("\n【大单元教学设计方案】\n")
		body.WriteString(content)
	}
	if atlas != "" {
		body.WriteString("\n\n【大单元整体设计图谱】\n")
		body.WriteString(atlas)
	}
	b.WriteString(truncateRunesForUnitPlan(body.String(), unitPlanContextMaxRunes))

	b.WriteString("\n---- 大单元整体设计结束 ----\n")
	return b.String()
}
