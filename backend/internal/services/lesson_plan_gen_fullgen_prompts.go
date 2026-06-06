package services

// lesson_plan_gen_fullgen_prompts.go — 多阶段「一键生成（全委托）」提示词常量集
//
// v172 整洁性拆分：原先这 4 个 fullGenerateXxxPrompt 长字符串常量 + resolveFullGeneratePrompt
//   都定义在 lesson_plan_gen_service.go 主文件里，使主文件超过 600 行红线。
//   本文件把它们独立出来（同 services 包，主文件内的引用与调用无需改动），
//   主文件随之回落到 600 行附近，符合「<600 行/文件」规范。
//
// 设计背景（v168/v169）：
//   「一键生成」让老师在某个阶段一次性产出完整内容，无需逐轮分段确认。
//   覆盖 4 个「写内容」的阶段：
//     - analyze（教学分析）落库走 extractGenericStageFromNatural（宽松，内容非空即存）
//     - design （教学设计）同上（宽松）
//     - write  （教案撰写）落库走 DetectLessonPlanContent（严格，需命中教案结构判定）
//     - revise （修订定稿）与 write 共用 handleWriteStageOutput，落库判定同样严格
//   review（AI 评审）阶段不参与一键生成（推进过去会自动触发评审）。
//
// 调用入口：processChatStageAsync（lesson_plan_gen_service.go）按当前阶段决定是否注入：
//   - write 阶段：在「正文为空 + fullGenerate」时注入 fullGenerateWritePrompt（其余两态见主文件）
//   - analyze/design/revise：fullGenerate=true 时调 resolveFullGeneratePrompt 取对应指令注入

// fullGenerateWritePrompt 教案撰写阶段·全委托一键生成指令（v168·功能B）
//
// 设计原则：本指令格式严格对齐 DetectLessonPlanContent（workshop_stage_extract.go）的判定条件，
// 确保 AI 一次性产出的教案能被识别并落库。DetectLessonPlanContent 要求：
//  1. ≥3 个教案标记词（教学目标/重点/难点/过程/准备/作业/板书...）
//  2. 有 # 开头的标题行，且标题含"教案/教学设计/教学目标/课题..."之一
//  3. 含"教学过程"或"教学环节"或"教学活动"
//  4. 含结尾标记（作业布置/板书设计/课后作业/课堂小结/教学反思...之一）
//  5. 去掉末尾客套话后正文 ≥800 字符
//
// 因此本指令强制：用 # 标题、含全部必备小节、教学过程分环节带时间、不分段、不寒暄。
const fullGenerateWritePrompt = `

== 全委托一键生成模式（系统级指令，最高优先级，覆盖上文所有分段确认要求）==
老师已明确选择"全委托 AI 一次性生成完整教案"，请你立即一次性输出一份**完整、可直接使用**的教案 Markdown，不要分段、不要等老师确认、不要说"接下来写下一部分"，本次回复就给出全部内容。

输出格式硬性要求（务必全部满足，否则系统无法识别保存）：
1. 以一级标题开头，格式为：# 《课题》教学设计（或 # 课题 教案）。
2. 必须包含以下小节，每个小节用 ## 二级标题：
   ## 教学目标（分知识与技能、过程与方法、情感态度价值观三维，或按核心素养列出，要具体可观察）
   ## 教学重难点（明确教学重点与教学难点）
   ## 教学准备（教具、学具、课件、场地等）
   ## 教学过程（这是核心，必须按环节展开，每个环节标注时间分配，如"一、导入（5分钟）"，环节要包含教师活动与学生活动）
   ## 作业布置（具体的课后作业或练习）
   ## 板书设计（本节课的板书结构）
3. 教学过程要详实，环节完整（导入→新授→巩固→小结等），内容充实，确保整份教案不少于 800 字。
4. 结尾直接以"板书设计"小节自然结束，**不要**添加"如需修改请告诉我""希望这份教案对您有帮助"等客套话。
5. 全程使用规范的 Markdown 标题层次（# ## ###），正文用自然中文，不要输出 JSON 或代码块包裹整篇教案。

请现在就开始输出完整教案。`

// fullGenerateAnalyzePrompt 教学分析阶段·全委托一键生成指令（v169）
//
// 落库路径：analyze 走 extractGenericStageFromNatural（判定宽松，内容非空即存为 narrative），
// 故格式要求不必像 write 那样严格，重点是引导 AI 一次性输出完整的、结构清晰的学情/教材分析。
const fullGenerateAnalyzePrompt = `

== 全委托一键生成模式（系统级指令，最高优先级，覆盖上文所有分段确认/逐步追问要求）==
老师已明确选择"全委托 AI 一次性完成本阶段（教学分析）"，请你立即一次性输出一份**完整的教学分析**，不要分段、不要反过来追问老师、不要说"接下来分析下一部分"，本次回复就给出全部内容。

请用 Markdown 输出，至少包含以下方面（用 ## 二级标题分节）：
## 教材分析（本课内容在教材中的地位、知识结构、与前后内容的联系）
## 课程标准对接（本课对应的课程标准/核心素养要求）
## 学情分析（该年级学生的认知特点、已有知识基础、可能的学习难点与误区）
## 核心概念与重难点预判（本课的核心概念，以及预计的教学重点和难点）

要求：内容具体、贴合学科与年级，避免空话套话。结尾不要加"如需调整请告诉我"之类的客套话。

请现在就开始输出完整的教学分析。`

// fullGenerateDesignPrompt 教学设计阶段·全委托一键生成指令（v169）
//
// 落库路径：design 走 extractGenericStageFromNatural（宽松）。
const fullGenerateDesignPrompt = `

== 全委托一键生成模式（系统级指令，最高优先级，覆盖上文所有分段确认/逐步追问要求）==
老师已明确选择"全委托 AI 一次性完成本阶段（教学设计）"，请你立即一次性输出一份**完整的教学设计方案**，不要分段、不要反过来追问老师，本次回复就给出全部内容。

请用 Markdown 输出，至少包含以下方面（用 ## 二级标题分节）：
## 教学目标（三维目标或核心素养目标，要具体可观察、可评估）
## 教学重难点（明确重点与难点，并说明突破难点的策略）
## 教学策略（采用的教学方法、学习方式，如探究式/任务驱动/小组合作等及理由）
## 教学活动设计（按环节展开，每个环节给出名称、预计时长、教师活动、学生活动、设计意图）
## 评价设计（如何检验目标达成，形成性评价与总结性评价的安排）

要求：活动设计要可操作、环节衔接合理、贴合学科与年级。结尾不要加客套话。

请现在就开始输出完整的教学设计方案。`

// fullGenerateRevisePrompt 修订定稿阶段·全委托一键生成指令（v169）
//
// 落库路径：revise 与 write 共用 handleWriteStageOutput，需命中 DetectLessonPlanContent（严格）。
// 故格式要求与 write 完全一致（# 标题 + 全部必备小节 + ≥800字），
// 区别在于：要求 AI 基于"前面阶段已完成的教案正文 + AI 评审建议"做修订后输出完整教案。
// 已完成的正文与评审结论已由 LoadStagePromptContextV2 + Episodic 摘要注入上下文，AI 可直接参考。
const fullGenerateRevisePrompt = `

== 全委托一键生成模式（系统级指令，最高优先级，覆盖上文所有分段确认要求）==
老师已明确选择"全委托 AI 一次性完成修订定稿"。请你参考前面阶段已经完成的教案正文，以及 AI 评审阶段提出的改进建议，**对教案进行修订并一次性输出修订后的完整教案 Markdown**。不要只输出修改点、不要分段、不要等老师确认，本次回复直接给出修订后的整份教案。

输出格式硬性要求（务必全部满足，否则系统无法识别保存）：
1. 以一级标题开头，格式为：# 《课题》教学设计（或 # 课题 教案）。
2. 必须包含以下小节，每个小节用 ## 二级标题：教学目标、教学重难点、教学准备、教学过程（按环节展开并标注时间分配，含教师活动与学生活动）、作业布置、板书设计。
3. 在原教案基础上落实评审建议的改进点（如时间分配、评价工具、活动设计等），但仍输出**完整**教案而非仅改动部分。
4. 整份教案不少于 800 字，结尾以"板书设计"小节自然结束，不要加客套话。
5. 使用规范 Markdown 标题层次，不要用 JSON 或代码块包裹整篇教案。

请现在就开始输出修订后的完整教案。`

// resolveFullGeneratePrompt 按阶段返回对应的全委托一键生成指令（v169）
//
// 返回空字符串表示该阶段不支持一键生成（如 review）。
// write 阶段的"已有正文→防重复"判定不在此处，仍由 processChatStageAsync 单独处理。
func resolveFullGeneratePrompt(stageCode string) string {
	switch stageCode {
	case "analyze":
		return fullGenerateAnalyzePrompt
	case "design":
		return fullGenerateDesignPrompt
	case "write":
		return fullGenerateWritePrompt
	case "revise":
		return fullGenerateRevisePrompt
	default:
		return ""
	}
}
