package services

// workshop_stage_prompts.go — 阶段化备课工坊提示词构建
//
// 包含:六层系统提示词拼接 + 各上下文构建函数 + 对话规范 + 组件注入 + 辅助函数
//
// v76拆分:自然语言提取+废弃函数 移至 workshop_stage_extract.go
// v82变更:normalizeGradeToNumber 抽取为 utils.NormalizeGradeToNumber 统一工具函数
// v84变更:BuildStageChatPrompt 改造为分层记忆版本
//          新增 BuildStageChatPromptV2 支持 Working+Episodic 分层上下文
// v89-4变更:AutoMatchStageComponents 根据阶段类型自动设置 StageTiming 参数
// v110(TE-DNA 3.0 P0)变更:
//   - 新增 BuildStageSystemPromptV2,接收可选的 assistantPrompt 参数
//   - assistantPrompt 非空时替换第 4 层(阶段角色+变体段),其他层保持不变
//   - 原 BuildStageSystemPrompt 函数签名保持不变,内部转调 V2 传空 assistantPrompt
//   - 向后 100% 兼容,所有旧调用无须改动
// v190变更(对话模式UX修复·修复1·去掉3000字节物理截断):
//   - BuildStageChatPromptV2 内已有教案正文(content_markdown)的注入,
//     原先用 len(content) > 3000 按【字节】硬截断 + content[:3000] 切片,
//     存在三个问题:①按字节算长度,中文1字3字节,约1000汉字就触顶被截半;
//     ②content[:3000] 按字节切片,可能切碎半个汉字成乱码;
//     ③评审/修订阶段读到被截断的正文+"已截断"提示,必然误报"后半缺失/截断"。
//   - 改法:改用同文件已有的 safeUTF8Truncate(按 rune 安全截断,不切碎汉字),
//     上限提到 6000 rune——一份完整教案约 3500-4500 字符,6000 rune 能完整容纳
//     正常教案并留足余量,正常教案不再触发任何截断(根治"评审说截断"误判),
//     仅对极端超长正文做按字保护性截断防上下文爆量。
//   - 本次仅改此一处的截断逻辑,其余全文一字未动。
// v191变更(迭代3.5 Phase B·write/revise 阶段接入建议芯片·去复读竖切):
//   - 背景:对话模式 write 阶段原先要求老师手打"继续"来逐环节推进,AI 每轮套话,
//     是方案要消灭的"复读"根源。analyze/design 阶段已接入建议芯片(由 workshop_stages
//     表的 system_prompt 注入芯片规则),但 write/revise 两阶段提示词从未加芯片规则。
//   - 改动:仅在本文件 buildDialogueGuidelines 函数的 write / revise 两个 case 末尾,
//     各追加一段「建议动作输出规则」,让 AI 在这两个阶段也输出 ```suggested_actions 块。
//     老师由此可点【确认,继续写】【这段再改改】【去评审一遍】等芯片,不必手打"继续"。
//   - 为什么放在第5层(buildDialogueGuidelines)而非数据库列:第5层是整条系统提示词
//     拼接的物理末尾(第1~4层之后),且本就是写 write/revise 专属对话规范的地方;
//     而 system_prompt / prompt_variants 后面还会被拼上第5层,无法保证芯片块在最后。
//   - 位置无关强约束:write 阶段在 processChatStageAsync 中存在"态a已有完整正文"会
//     再追加一段指令(比第5层更靠后),故芯片规则文案写成"无论本轮在做什么、回复正文
//     结束后都把芯片块放整条回复最末尾",不依赖物理位置也能让 AI 正确放置芯片块。
//   - 协议安全:write/revise 阶段 AI 不输出 ```json 评审块(那是 review 专属),
//     追加 ```suggested_actions 围栏块与现有任何协议零冲突;review/analyze/design/
//     default 四个 case 一字未动;后端 ParseSuggestedActions 对所有阶段无差别解析广播。
//   - 本次仅改 buildDialogueGuidelines 的 write/revise 两个 case,其余全文一字未动。
// v192变更(第4层「助手叠加」改造·子轮A·骨架永远在+可循环叠加框架):
//   ============================ 这是什么 ============================
//   背景:第4层原逻辑是"助手 prompt 非空则整段替换阶段原生骨架"(v110)。事故表明:
//   阶段原生 system_prompt 是几千字的结构化多步流程骨架(analyze 四步、design 五维、
//   阶段边界红线、阶段交接、芯片规则),而个人/学校助手通常只有几百字的"教学经验+语气
//   偏好"补充。用补充内容整段替换流程骨架 = 用沙子换承重墙 → 流程/边界/交接全丢
//   (analyze 只走一步就被推进、design 的按维度推进整个消失)。
//
//   正确模型(用户定调,不可动摇):
//     "不管什么样,系统的流程提示词都要始终生效,尤其是对流程的。"
//     → 阶段原生骨架是地基(永远 100% 拼,助手碰不了);助手是叠加增量(只补充
//       教学经验/语气/校本规范这类"风格层",绝不能改写流程、抢最高优先级)。
//
//   ============================ 子轮A 改了什么 ============================
//   把第4层的 if(替换)/else(骨架) 改为:
//     (A) 阶段原生骨架永远拼:stage.SystemPrompt + 变体段(把原 else 提升为无条件);
//     (B) 叠加段:flag 开且有助手 prompt 时,经 buildAssistantOverlay 追加在骨架之后——
//         ── 前置冲突声明(锁死"凡与骨架流程/边界冲突,一律以阶段流程为准");
//         ── 逐个助手:StripSuggestedActionsBlock 剥离助手夹带的芯片块(芯片规则系统独占)
//            + safeUTF8Truncate 各段字数上限(防上下文爆量);
//     flag 关 → 逐字回退 v110 老行为(整段替换),作为出问题时的紧急回滚安全绳。
//
//   ============================ 为什么 buildAssistantOverlay 收列表 ============================
//   子轮A 当前只叠加一个助手(技能路由 RouteDefaultAssistant 现返回单个,个人优先),
//   故传 []string{assistantPrompt}(单元素)。但叠加框架写成"接受助手列表+可循环追加",
//   是为子轮B(学校+个人双层叠加)预留:届时路由返回个人+学校两个,只需把列表从单元素
//   扩成 [学校, 个人](顺序对应 A骨架→B学校→C个人,个人在后印象最深),
//   buildAssistantOverlay 一行不改即支持双层——分步骤实现,第4层零返工。
//
//   ============================ 边界与回滚 ============================
//   - 改动收敛在本文件第4层那一处 + 新增一个包级 flag + 一个叠加函数;
//   - 不碰其余五层、不碰阶段机、不碰芯片协议本身、不碰设计忠实规则、不碰技能路由;
//   - BuildStageSystemPromptV2 签名不动(仍收单个 assistantPrompt string),
//     内部包成单元素列表传给 buildAssistantOverlay,子轮B 再随路由改造升级签名;
//   - stageAssistantOverlayEnabled 默认 true(新范式即既定方向),关闭仅供紧急回滚,
//     回滚目标逐字等价 v110 线上行为(SRE 铁律:回滚锚点必须是已知稳定态)。

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// chatPromptContentMaxRunes 对话上下文中注入"已有教案正文"的最大字符数(按 rune 计)
//
// 取值说明(v190 修复1):
//   - 一份完整教案约 3500-4500 字符,6000 rune 可完整容纳并留安全余量,
//     使正常教案在 analyze/design/write/review/revise 任一阶段的对话上下文中
//     都不会被截断,从根上消除"评审说后半缺失/截断"的误判。
//   - 仅当正文极端超长(远超正常教案)时,才按 rune 安全截断防上下文爆量,
//     且 safeUTF8Truncate 按 rune 切不会切碎汉字成乱码。
const chatPromptContentMaxRunes = 6000

// ==================== v192:第4层「助手叠加」改造·开关与参数 ====================

// stageAssistantOverlayEnabled 第4层「助手叠加」总开关(v192 子轮A)。
//
//   - true(默认,既定方向):第4层 = 阶段原生骨架【永远拼】+ 助手作为风格补充【叠加】在后。
//     这是修复事故的正确范式——流程骨架是承重墙,助手只是装修,谁都掀不翻骨架。
//   - false(紧急回滚安全绳):逐字回退 v110 老行为——助手 prompt 非空则【整段替换】第4层。
//     仅在叠加逻辑出现线上问题、需要立刻回到"确切已知的旧稳定态"排查时才关闭。
//
// 设计立场:本 flag 不是"两种范式长期二选一"的开关,而是"新范式上线的临时安全绳"。
// 灰度验证稳定后,此 flag 沉淀为永久 true,将来清理时连同 false 分支一并删除,
// 第4层只剩"骨架永远在 + 叠加"这一条路径。
var stageAssistantOverlayEnabled = true

// assistantOverlayMaxRunesPerPrompt 单个助手补充段注入第4层时的运行时上限（按 rune 计）。
//
// 设计调整：1500 rune 只能容纳简短语气偏好，无法完整承载学科教学方法、地方教研要求、
// 教师已有优势、成长目标和自检规则。现提高到 8000 rune：
//   - 推荐助手正文控制在 2000—5000 rune；
//   - 地方规范较多的复杂学科可到约 6000 rune；
//   - 8000 rune 是运行保护上限，不是鼓励每份助手都写满。
//
// 完整原稿仍可在 ai_assistants.full_prompt 中保存；这里只控制每轮备课实际注入量。
// 阶段骨架、配方、教材、学情和前序成果均不受此常量直接截断。
const assistantOverlayMaxRunesPerPrompt = 8000

// assistantOverlayConflictNotice 叠加段的前置冲突声明(§2.1)。
//
// 拼在所有助手补充段【最前面】,从 prompt 层面再锁一道"系统流程优先",
// 防止助手内容里夹带的流程类指令(如"先出完整框架")把 AI 带偏——
// 无论助手怎么写,凡与上方阶段原生流程/边界冲突,一律以阶段流程为准。
const assistantOverlayConflictNotice = `

== 教学风格与经验补充(优先级低于上方阶段流程)==
以下是为贴合本校教研共识与老师个人习惯而提供的教学风格、教学经验与偏好补充。请在严格遵守上方"阶段流程骨架"(分步流程、阶段边界、阶段交接、产出规范)的前提下,参考采纳以下补充来决定"具体怎么表达、补充哪些经验"。

重要:以下补充只影响表达风格与经验取舍,绝不改变上方规定的流程步骤与阶段边界。若以下任何内容与上方阶段流程或边界存在冲突,一律以上方阶段流程为准,忽略冲突的补充部分。
`

// ==================== 阶段完整系统提示词构建(六层拼接)====================

// BuildStageSystemPrompt 构建某阶段完整的系统提示词(向后兼容版,v110 前原签名不变)
//
// v110 改动:内部转调 V2,传空 assistantPrompt 保持与原行为完全一致
func BuildStageSystemPrompt(
	ctx context.Context,
	stage *models.WorkshopStage,
	recipe *models.TeachingRecipe,
	priorOutputs []*models.WorkshopStageOutput,
	subject string,
	grade string,
	promptMode string,
	lessonStructure string,
	selectedCompIDs []string,
) string {
	return BuildStageSystemPromptV2(
		ctx, stage, recipe, priorOutputs,
		subject, grade, promptMode, lessonStructure, selectedCompIDs,
		"", // v110 新参数:assistantPrompt 空 → 走原行为
		"", // 技能路由 Phase1 新参数:recentUserText 空 → 第3层走保底全量匹配(等价原行为)
	)
}

// BuildStageSystemPromptV2 v110 新增:支持 AI 助手 full_prompt 注入的版本
//
// assistantPrompt 语义(v192 子轮A 改造后):
//   - 空字符串 → 完全走原行为(第4层 = 阶段原生骨架 stage.SystemPrompt + 变体段)
//   - 非空字符串 →
//     · stageAssistantOverlayEnabled=true(默认):第4层 = 阶段原生骨架【永远拼】
//   - 助手作为风格补充【叠加】在骨架之后(经 buildAssistantOverlay,含前置冲突声明、
//     剥离助手芯片、字数上限);
//     · stageAssistantOverlayEnabled=false(紧急回滚):逐字回退 v110——整段替换第4层。
//
// 无论走哪条分支,第 1/2/3/3.5/5 层都不受影响,语义完全独立。
func BuildStageSystemPromptV2(
	ctx context.Context,
	stage *models.WorkshopStage,
	recipe *models.TeachingRecipe,
	priorOutputs []*models.WorkshopStageOutput,
	subject string,
	grade string,
	promptMode string,
	lessonStructure string,
	selectedCompIDs []string,
	assistantPrompt string,
	recentUserText string, // 技能路由 Phase1:老师当轮发言,供第3层知识型技能精排;空串→保底全量匹配
) string {
	var sb strings.Builder

	// 第1层:配方全局上下文
	if recipe != nil {
		globalCtx := BuildStageGlobalContext(recipe)
		if globalCtx != "" {
			sb.WriteString(globalCtx)
			sb.WriteString("\n")
		}
	}

	// 第2层:前序阶段产出物
	if len(priorOutputs) > 0 {
		priorCtx := BuildPriorOutputsContext(priorOutputs)
		if priorCtx != "" {
			sb.WriteString(priorCtx)
			sb.WriteString("\n")
		}
	}

	// 第2.5层:降级提示词(跳过的前置阶段补偿指令)
	if len(priorOutputs) > 0 {
		degradation := buildDegradationHint(priorOutputs, recipe)
		if degradation != "" {
			sb.WriteString(degradation)
			sb.WriteString("\n")
		}
	}

	// 第3层:本阶段专属组件内容
	// 优先级:用户手动选中 > 配方组件 > 自动匹配
	if stage.ComponentTypes != "" && stage.ComponentTypes != "[]" {
		componentCtx := ""
		if len(selectedCompIDs) > 0 {
			componentCtx = BuildSelectedComponentContext(ctx, selectedCompIDs)
		}
		if componentCtx == "" && recipe != nil {
			componentCtx = BuildStageComponentContext(ctx, recipe, stage.ComponentTypes)
		}
		if componentCtx == "" {
			// 技能路由 Phase1:第3层"自动匹配"产物改由 resolveStageComponentAutoMatch 决定——
			// 开关开启且有老师发言时走 RerankedStageComponents 精排,否则退回 AutoMatchStageComponents 保底。
			componentCtx = resolveStageComponentAutoMatch(ctx, stage.ComponentTypes, subject, grade, stage.StageCode, recentUserText)
		}
		if componentCtx != "" {
			sb.WriteString(componentCtx)
			sb.WriteString("\n")
		}
	}

	// 第3.5层:教案结构偏好
	if lessonStructure != "" && lessonStructure != "[]" {
		structureCtx := BuildLessonStructurePrompt(stage.StageCode, lessonStructure)
		if structureCtx != "" {
			sb.WriteString(structureCtx)
			sb.WriteString("\n")
		}
	}

	// ==================== 第4层:阶段角色提示词(v192「助手叠加」改造)====================
	// 核心原则(用户定调):阶段原生流程骨架永远 100% 生效,助手只做"风格补充"叠加,
	// 绝不允许助手整段替换骨架(那会丢掉分步流程/阶段边界/交接/芯片规则)。
	//
	// 分支:
	//   · assistantPrompt 为空 → 第4层 = 骨架(stage.SystemPrompt + 变体段),与历来行为一致。
	//   · assistantPrompt 非空 + 叠加开关开(默认)→ 骨架【永远拼】+ 助手叠加在后。
	//   · assistantPrompt 非空 + 叠加开关关(紧急回滚)→ 逐字回退 v110:整段替换第4层。
	buildStageRoleLayer(&sb, stage, promptMode, assistantPrompt)

	// 第5层:对话规范指引
	dialogueGuide := buildDialogueGuidelines(stage.StageCode, grade)
	if dialogueGuide != "" {
		sb.WriteString("\n")
		sb.WriteString(dialogueGuide)
	}

	return sb.String()
}

// buildStageRoleLayer 拼装第4层(阶段角色),封装 v192「骨架永远在 + 助手叠加」三分支逻辑。
//
// 抽成独立函数的原因:第4层逻辑随 v192 变复杂(原生骨架 / 叠加 / 回滚三分支),
// 内联在 BuildStageSystemPromptV2 会让主拼接流程难读;抽出后主流程一行 buildStageRoleLayer
// 即表达"拼第4层",分支细节内聚于此,也便于子轮B 扩双层时只动本函数与 buildAssistantOverlay。
//
// 参数:
//   - sb:六层拼接共用的 strings.Builder(直接写入,与其它层风格一致)。
//   - stage:当前阶段(取 stage.SystemPrompt 骨架 + stage.PromptVariants 变体段 + stage.StageCode 埋点)。
//   - promptMode:变体段选择模式(guided/peer/freeform)。
//   - assistantPrompt:已解析的助手 full_prompt(单个;空串=老师未挂任何助手)。
func buildStageRoleLayer(sb *strings.Builder, stage *models.WorkshopStage, promptMode string, assistantPrompt string) {
	wsPromptLog := logger.WithModule("workshop_stage_prompts")
	hasAssistant := strings.TrimSpace(assistantPrompt) != ""

	// ---------- 紧急回滚分支:flag 关 且 有助手 → 逐字回退 v110「整段替换」 ----------
	// 关闭叠加时,行为必须逐字等价 v110 线上(整段替换),作为已知稳定的回滚锚点。
	if hasAssistant && !stageAssistantOverlayEnabled {
		sb.WriteString("\n")
		sb.WriteString(assistantPrompt)
		sb.WriteString("\n")
		wsPromptLog.Info("第4层:叠加开关关闭,回退 v110 整段替换助手 full_prompt",
			"stage_code", stage.StageCode, "prompt_len", len(assistantPrompt))
		return
	}

	// ---------- (A) 阶段原生流程骨架:永远拼(承重墙,助手碰不了)----------
	// 无论老师有没有挂助手、挂了几个、助手写了什么,骨架都 100% 在。
	if stage.SystemPrompt != "" {
		sb.WriteString(stage.SystemPrompt)
		sb.WriteString("\n")
	}
	variantText := selectPromptVariant(stage.PromptVariants, promptMode)
	if variantText != "" {
		sb.WriteString("\n")
		sb.WriteString(variantText)
		sb.WriteString("\n")
	}

	// ---------- (B) 助手叠加段:flag 开 且 有助手 → 在骨架之后追加风格补充 ----------
	// 子轮A 只叠加一个助手,故传单元素列表;子轮B 扩双层时改传 [学校, 个人] 即可,本处零改。
	if hasAssistant && stageAssistantOverlayEnabled {
		overlay := buildAssistantOverlay([]string{assistantPrompt})
		if overlay != "" {
			sb.WriteString(overlay)
			wsPromptLog.Info("第4层:阶段骨架永远拼 + 助手叠加(风格补充)",
				"stage_code", stage.StageCode, "assistant_count", 1,
				"prompt_len", len(assistantPrompt))
		}
	}
}

// buildAssistantOverlay 构建第4层「助手叠加段」(v192 子轮A;为子轮B 双层叠加预留)。
//
// 入参 assistantPrompts:按拼接顺序排列的助手 full_prompt 列表。
//   - 子轮A:单元素 [当前那一个助手](技能路由按个人>本校>系统挑出的一个)。
//   - 子轮B:多元素 [学校助手, 个人助手](顺序即 A骨架→B学校→C个人,个人在后印象最深),
//     本函数无需任何改动即支持。
//
// 产出(拼在阶段原生骨架【之后】):
//
//	── 前置冲突声明(assistantOverlayConflictNotice):只拼一次,在所有补充段最前,
//	   锁死"凡与骨架流程/边界冲突,一律以阶段流程为准"。
//	── 逐个助手补充段:
//	     · StripSuggestedActionsBlock 剥离助手内容里夹带的 ```suggested_actions 芯片块
//	       (芯片规则系统独占,由阶段骨架/第5层统一下发,助手无权下发芯片);
//	     · safeUTF8Truncate 按 assistantOverlayMaxRunesPerPrompt 上限保护性截断(防爆量)。
//
// 全部助手 prompt 清洗后皆为空(或入参为空)→ 返回 ""(连冲突声明也不拼,第4层只剩骨架)。
//
// 注意:本函数只剥离助手的 suggested_actions 专用围栏(复用 StripSuggestedActionsBlock,
// 该函数绝不动 ```json 评审块等其它围栏),不对助手内容做任何流程/风格切分——
// 流程类夹带由前置冲突声明在 prompt 层兜底压制(PRD §4-4 方案一:纯 prompt 防御)。
func buildAssistantOverlay(assistantPrompts []string) string {
	// 先逐个清洗(剥芯片块 + 截断),过滤掉清洗后为空的助手,避免拼出空补充段。
	cleaned := make([]string, 0, len(assistantPrompts))
	for _, p := range assistantPrompts {
		if strings.TrimSpace(p) == "" {
			continue
		}
		// 剥离助手夹带的芯片块(系统独占芯片协议,助手不得下发芯片)。
		stripped := StripSuggestedActionsBlock(p)
		// 保护性按 rune 截断,防极端超长助手内容撑爆上下文(正常助手远不及上限)。
		stripped = strings.TrimSpace(stripped)
		originalRunes := len([]rune(stripped))
		stripped = safeUTF8Truncate(stripped, assistantOverlayMaxRunesPerPrompt)
		if originalRunes > assistantOverlayMaxRunesPerPrompt {
			logger.WithModule("workshop_stage_prompts").Warn(
				"助手完整原稿超过工坊运行预算，已按Unicode字符安全截断",
				"original_runes", originalRunes,
				"injected_runes", assistantOverlayMaxRunesPerPrompt,
			)
		}
		if strings.TrimSpace(stripped) == "" {
			continue
		}
		cleaned = append(cleaned, stripped)
	}

	if len(cleaned) == 0 {
		return ""
	}

	var sb strings.Builder
	// 前置冲突声明:只拼一次,在所有补充段之前,从 prompt 层锁死系统流程优先。
	sb.WriteString(assistantOverlayConflictNotice)
	// 逐个助手补充段:用分隔标题区隔,便于 AI 区分多条补充来源(子轮B 双层时尤其清晰)。
	for i, c := range cleaned {
		if len(cleaned) > 1 {
			// 多条补充时加序号小标题;单条(子轮A)不加,保持简洁。
			sb.WriteString(fmt.Sprintf("\n【教学补充 %d】\n", i+1))
		} else {
			sb.WriteString("\n")
		}
		sb.WriteString(c)
		sb.WriteString("\n")
	}
	return sb.String()
}

// ==================== 降级提示词注入 ====================

func buildDegradationHint(priorOutputs []*models.WorkshopStageOutput, recipe *models.TeachingRecipe) string {
	var skippedStages []string
	for _, out := range priorOutputs {
		if out.Status == models.StageOutputSkipped {
			skippedStages = append(skippedStages, stageCodeToName(out.StageCode))
		}
	}
	if len(skippedStages) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("=== 降级补偿提示 ===\n")
	sb.WriteString(fmt.Sprintf("注意:老师跳过了以下阶段:%s。\n", strings.Join(skippedStages, "、")))
	sb.WriteString("这些阶段没有产生分析报告或设计方案。请你根据以下信息自行快速补充:\n")
	if recipe != nil {
		if strings.TrimSpace(recipe.StudentProfile) != "" {
			sb.WriteString("- 学情信息已在配方中提供,请参考配方全局信息中的学情档案\n")
		} else {
			sb.WriteString("- 学情信息缺失,请根据学科和年级特点做合理假设\n")
		}
		if strings.TrimSpace(recipe.TeachingStyle) != "" {
			sb.WriteString("- 教学风格偏好已在配方中提供,请参考\n")
		}
	} else {
		sb.WriteString("- 没有配方信息,请根据学科、年级和课题特点做合理的教学分析和设计假设\n")
	}
	sb.WriteString("请在本阶段工作中自然融入这些补充分析,不需要单独列出。\n")
	return sb.String()
}

// ==================== 教案结构注入 ====================

func BuildLessonStructurePrompt(stageCode string, lessonStructureJSON string) string {
	if stageCode == "analyze" {
		return ""
	}
	var blocks []models.LessonStructureBlock
	if err := json.Unmarshal([]byte(lessonStructureJSON), &blocks); err != nil || len(blocks) == 0 {
		return ""
	}
	var sb strings.Builder
	if stageCode == "design" {
		sb.WriteString("=== 老师定义的教案结构(教学设计须遵循)===\n")
		sb.WriteString("老师已定义了明确的教案结构,请在教学设计阶段遵循以下板块和环节安排来规划课堂:\n\n")
		designTotalDuration := 0
		for _, b := range blocks {
			required := ""
			if b.Required {
				required = "(必含)"
			}
			sb.WriteString(fmt.Sprintf("【%s】%s\n", b.Name, required))
			if b.Requirement != "" {
				sb.WriteString(fmt.Sprintf("  要求:%s\n", b.Requirement))
			}
			if len(b.SubSections) > 0 {
				sb.WriteString("  教学过程环节安排:\n")
				for _, sub := range b.SubSections {
					sb.WriteString(fmt.Sprintf("    ▸ %s(%d分钟)", sub.Name, sub.Duration))
					if sub.Goal != "" {
						sb.WriteString(fmt.Sprintf(" — 目标:%s", sub.Goal))
					}
					if sub.OutputRequirement != "" {
						sb.WriteString(fmt.Sprintf(" — 输出要求:%s", sub.OutputRequirement))
					}
					sb.WriteString("\n")
					designTotalDuration += sub.Duration
				}
			}
			sb.WriteString("\n")
		}
		if designTotalDuration > 0 {
			sb.WriteString(fmt.Sprintf("⏱ 教学过程各环节合计 %d 分钟,设计时请确保总时长与课时一致。\n", designTotalDuration))
		}
		sb.WriteString("\n请按以上环节结构来设计每个环节的教学活动,不要自行增删环节或调整时间分配。\n")
		return sb.String()
	}
	sb.WriteString("=== 老师定义的教案结构 ===\n")
	sb.WriteString("请严格按照以下结构输出教案,每个板块的要求务必遵循:\n\n")
	totalDuration := 0
	for _, b := range blocks {
		required := "选填"
		if b.Required {
			required = "必填"
		}
		sb.WriteString(fmt.Sprintf("【%s】(%s)\n", b.Name, required))
		if b.Requirement != "" {
			sb.WriteString(fmt.Sprintf("  要求:%s\n", b.Requirement))
		}
		if len(b.SubSections) > 0 {
			sb.WriteString("  教学过程环节安排:\n")
			for _, sub := range b.SubSections {
				sb.WriteString(fmt.Sprintf("    ▸ %s(%d分钟)", sub.Name, sub.Duration))
				if sub.Goal != "" {
					sb.WriteString(fmt.Sprintf(" — 目标:%s", sub.Goal))
				}
				if sub.OutputRequirement != "" {
					sb.WriteString(fmt.Sprintf(" — 输出要求:%s", sub.OutputRequirement))
				}
				sb.WriteString("\n")
				totalDuration += sub.Duration
			}
		}
		sb.WriteString("\n")
	}
	if totalDuration > 0 {
		sb.WriteString(fmt.Sprintf("⏱ 教学过程各环节合计 %d 分钟,请确保总时长与课时一致。\n", totalDuration))
	}
	return sb.String()
}

// ==================== 变体段选择 ====================

func selectPromptVariant(promptVariantsJSON string, promptMode string) string {
	if promptVariantsJSON == "" || promptVariantsJSON == "{}" {
		return ""
	}
	var variants map[string]string
	if err := json.Unmarshal([]byte(promptVariantsJSON), &variants); err != nil {
		return ""
	}
	mode := promptMode
	if mode == "" || mode == models.PromptModePerStage {
		mode = models.PromptModeGuided
	}
	if text, ok := variants[mode]; ok && strings.TrimSpace(text) != "" {
		return text
	}
	if text, ok := variants[models.PromptModeGuided]; ok {
		return text
	}
	return ""
}

// ==================== 对话规范指引 ====================

func buildDialogueGuidelines(stageCode string, grade string) string {
	base := `== 对话规范 ==
1. 请用自然语言与老师对话,所有内容直接输出,不要使用任何XML标签(如<stage_output>等)。
2. 不要输出JSON格式的结构化数据,老师看不懂这些格式。
3. 你的所有回复内容都会直接展示给老师,请确保内容对老师友好且有价值。
4. 阶段是否完成由老师手动点击"完成本阶段"按钮决定,你不需要判断阶段是否结束。
`
	switch stageCode {
	case "analyze":
		return base + "\n本阶段特殊规范:\n- 通过对话了解学情和教学需求,直接输出你的分析和建议\n- 可以引用课标、学生特征等进行分析讨论\n- 当分析充分时告诉老师可以进入下一阶段,但不要强制\n" + pedagogyLogicCore(grade)
	case "design":
		return base + "\n本阶段特殊规范:\n- 基于前序分析成果,与老师讨论教学设计方案\n- 可以提供多个方案选项供老师选择,用自然语言描述每个方案的优劣\n- 讨论教学目标、重难点、教学策略、活动设计等\n- 当设计方案确定后告诉老师可以进入下一阶段\n" + pedagogyLogicCore(grade)
	case "write":
		// v191:write 阶段在原"分段确认机制"规范末尾追加「建议动作输出规则」。
		// 让老师可点芯片推进(不必手打"继续"),AI 据当前回合状态选对应一组芯片。
		return base + "\n本阶段特殊规范(分段确认机制):\n- 先展示教案框架(各环节标题+时间),等老师确认\n- 每次只输出1-2个环节的详细内容,然后停下来等老师确认\n- 老师说确认/继续/可以后,再输出下一批环节\n- 所有环节输出完毕后,最后一次性输出包含全部内容的完整Markdown教案(用于系统提取保存)\n- 完整教案必须包含:教学目标、教学重难点、教学准备、教学过程(含各环节时间分配)、作业布置、板书设计\n- 如果老师中途提修改意见,先调整该环节再继续\n" + writeReviseFidelityRule + writeStageChipRule + pedagogyLogicCore(grade)
	case "review":
		return base + "\n本阶段特殊规范:\n- 直接输出评审报告,包含:总评分(满分10分)、各维度评分和点评、优点、改进建议\n- 评审维度包括:教学目标(T1)、教学内容(T2)、教学方法(T3)、教学评价(T4)\n- 每个维度给出具体分数和简短评语\n- 改进建议要具体可操作,指出具体位置和修改方向\n- 评审完成后等待老师确认,不要主动修改教案\n"
	case "revise":
		// v191:revise 阶段在原修订规范末尾追加「建议动作输出规则」。
		// 让老师可点芯片确认/继续调整/再评审/定稿,AI 据当前回合状态选对应一组芯片。
		return base + "\n本阶段特殊规范:\n- 先列出修改清单(哪些地方需要改、为什么改、怎么改)\n- 与老师确认修改方案后,输出修订后的完整Markdown教案\n- 修订后的教案同样使用 # ## ### 等Markdown层次结构\n- 修订说明可以在教案前面简要列出,然后紧接完整教案\n" + writeReviseFidelityRule + reviseStageChipRule
	default:
		return base
	}
}

// ==================== v191:write/revise 阶段建议芯片规则文案 ====================
//
// 设计要点(对齐 lesson_plan_gen_actions.go 的 suggested_actions 协议):
//   - 用专用围栏 ```suggested_actions(不是 ```json),块内含顶层 "suggested_actions" 数组键;
//   - action_type 只用协议五枚举中本阶段会用到的几种(send_text / full_generate / switch_stage);
//   - 位置无关强约束:明确告诉 AI"无论本轮在做什么,回复正文结束后把芯片块放整条回复最末尾",
//     以兼容 write 态a(processChatStageAsync 中已有完整正文时会再追加一段更靠后的指令);
//   - 情境感知:让 AI 按"当前回合处于哪种状态"选对应一组芯片,而非每轮给同一套(去模板化)。
//
// 用字符串拼接嵌入围栏("```suggested_actions" 等)以避免 Go 原始字符串反引号与 Markdown 围栏冲突。

// writeReviseFidelityRule write/revise 阶段追加的「设计忠实 + 创新建议剥离」规则（v193）
//
// 目的：让 AI 出完整教案时忠实于「教学设计」阶段已与老师敲定的方案，不擅自加戏；
// 确有创新想法时，写进专用的 teacher_suggestion 块（后端 splitSuggestionBlock 会把
// 该块从教案正文切走、转而拼进对话气泡 narrative 给老师定夺），保证落库教案正文只含共识内容。
const writeReviseFidelityRule = `

== 设计忠实与创新建议剥离（重要）==
1. 撰写/修订完整教案时，凡是前面"教学设计"阶段已经和老师明确商定的教学环节、活动形式、课堂收尾、作业安排，你必须忠实还原老师确认过的方案，不得擅自更换形式、升级玩法或替换成你认为更好的设计。
2. 教学设计阶段没有具体讨论、但一份完整教案结构上必须有的部分（如作业布置、板书设计、教学准备），你可以根据课堂内容合理补全。
3. 但是——如果你在撰写中产生了某个"教学设计阶段从未与老师讨论过、且会实质影响课堂组织方式"的新想法或新活动设计（例如一种具体的作业活动形式、一个新的互动环节、一种新的评价玩法），绝对不要把它直接写进教案正文当作既定方案。正确做法是：把这类创新建议单独写进一个 ` + "```teacher_suggestion" + ` 代码块里（块内用自然中文说明你的建议及理由），由老师自己决定是否采纳。教案正文本身只包含与老师达成共识的内容。该建议块写在完整教案 Markdown 之外（另起一段），不要塞进教案的某个小节里。建议最多写 1-2 条，点到为止。

示例（教案正文之外，另起一个块）：
` + "```teacher_suggestion" + `
我注意到本节课的作业还可以更有意思：除了书面片段，还可以让学生录一段配音发到班级群、评选"最美好声音"。这是我的额外建议，是否采用由您决定，我没有把它写进上面的教案正文。
` + "```" + `
`

// writeStageChipRule write 阶段追加的建议芯片规则
const writeStageChipRule = `
== 建议动作输出规则(系统级,老师无需关心,供界面生成可点选芯片)==
在你本轮回复的正文【全部输出完毕之后】,无论本轮你是在展示框架、写某几个环节、还是已输出完整教案,都另起一行追加一个「建议动作」代码块,用 ` + "```suggested_actions" + ` 围栏包裹(围栏标识就是 suggested_actions,不是 json),块内是严格 JSON。该代码块必须是整条回复的【最后部分】,块之外不得再出现任何 suggested_actions 块。

请根据【本轮所处的状态】选择对应的一组芯片(2-4 条,每条 label 不超过 8 字、口语化):

情形一·正在展示框架或正在逐环节撰写(完整教案尚未输出):
` + "```suggested_actions" + `
{
  "suggested_actions": [
    {"id": "go_on", "emoji": "✅", "label": "确认，继续写", "action_type": "send_text", "payload": {"text": "确认，请继续写下一部分"}},
    {"id": "tweak", "emoji": "✏️", "label": "这段再改改", "action_type": "send_text", "payload": {"text": "刚写的这部分我想调整一下："}},
    {"id": "one_shot", "emoji": "⚡", "label": "直接出完整教案", "action_type": "full_generate", "payload": {"stage": "write"}}
  ]
}
` + "```" + `

情形二·完整教案已输出完毕(本轮刚一次性给出了完整 Markdown 教案):
` + "```suggested_actions" + `
{
  "suggested_actions": [
    {"id": "to_review", "emoji": "🔎", "label": "去评审一遍", "action_type": "switch_stage", "payload": {"stage": "review"}},
    {"id": "edit_more", "emoji": "✏️", "label": "我再改改", "action_type": "send_text", "payload": {"text": "我想再修改教案的某个部分："}}
  ]
}
` + "```" + `

要求:
1. action_type 只能用 send_text / full_generate / switch_stage 三种(本阶段适用)。
2. 芯片是"建议"不是"必选",请给当前最自然的下一步,文案随本轮内容自然措辞,不要每轮一模一样地硬凑。
3. payload.text 是点击后替老师发出的话,要顺着当前语境写。
`

// reviseStageChipRule revise 阶段追加的建议芯片规则
const reviseStageChipRule = `
== 建议动作输出规则(系统级,老师无需关心,供界面生成可点选芯片)==
在你本轮回复的正文【全部输出完毕之后】,无论本轮你是在列修改清单、还是已输出修订后的完整教案,都另起一行追加一个「建议动作」代码块,用 ` + "```suggested_actions" + ` 围栏包裹(围栏标识就是 suggested_actions,不是 json),块内是严格 JSON。该代码块必须是整条回复的【最后部分】,块之外不得再出现任何 suggested_actions 块。

请根据【本轮所处的状态】选择对应的一组芯片(2-4 条,每条 label 不超过 8 字、口语化):

情形一·正在讨论修改方案或刚列出修改清单(修订尚未定稿):
` + "```suggested_actions" + `
{
  "suggested_actions": [
    {"id": "apply", "emoji": "✅", "label": "这样改可以", "action_type": "send_text", "payload": {"text": "这样改可以，请按这个方案修订"}},
    {"id": "more_tweak", "emoji": "✏️", "label": "还要再调", "action_type": "send_text", "payload": {"text": "我还想再调整一下："}}
  ]
}
` + "```" + `

情形二·修订后的完整教案已输出完毕:
` + "```suggested_actions" + `
{
  "suggested_actions": [
    {"id": "re_review", "emoji": "🔎", "label": "再评审一次", "action_type": "switch_stage", "payload": {"stage": "review"}},
    {"id": "finalize", "emoji": "👍", "label": "就定稿了", "action_type": "send_text", "payload": {"text": "教案我满意了，就这样定稿"}}
  ]
}
` + "```" + `

要求:
1. action_type 只能用 send_text / switch_stage 两种(本阶段适用)。
2. 芯片是"建议"不是"必选",请给当前最自然的下一步,文案随本轮内容自然措辞,不要每轮一模一样地硬凑。
3. payload.text 是点击后替老师发出的话,要顺着当前语境写。
`

// ==================== 配方全局上下文 ====================

func BuildStageGlobalContext(recipe *models.TeachingRecipe) string {
	var sb strings.Builder
	hasContent := false
	sb.WriteString("=== 配方全局信息 ===\n")
	if strings.TrimSpace(recipe.StudentProfile) != "" {
		sb.WriteString(fmt.Sprintf("\n【学情档案】\n%s\n", recipe.StudentProfile))
		hasContent = true
	}
	if strings.TrimSpace(recipe.TeachingStyle) != "" {
		sb.WriteString(fmt.Sprintf("\n【教学风格偏好】\n%s\n", recipe.TeachingStyle))
		hasContent = true
	}
	if strings.TrimSpace(recipe.SchoolRequirements) != "" {
		sb.WriteString(fmt.Sprintf("\n【学校要求】\n%s\n", recipe.SchoolRequirements))
		hasContent = true
	}
	if strings.TrimSpace(recipe.CustomNotes) != "" {
		sb.WriteString(fmt.Sprintf("\n【备课心得】\n%s\n", recipe.CustomNotes))
		hasContent = true
	}
	if strings.TrimSpace(recipe.CustomPrompt) != "" {
		sb.WriteString(fmt.Sprintf("\n【自定义指令】\n%s\n", recipe.CustomPrompt))
		hasContent = true
	}
	if !hasContent {
		return ""
	}
	return sb.String()
}

// ==================== 前序阶段产出物上下文 ====================

func BuildPriorOutputsContext(outputs []*models.WorkshopStageOutput) string {
	if len(outputs) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("=== 前序阶段产出 ===\n")
	for _, out := range outputs {
		stageName := stageCodeToName(out.StageCode)
		if out.Status == models.StageOutputSkipped {
			sb.WriteString(fmt.Sprintf("\n【阶段%d — %s】(已跳过)\n", out.StageOrder, stageName))
			continue
		}
		if out.Status != models.StageOutputCompleted {
			continue
		}
		sb.WriteString(fmt.Sprintf("\n【阶段%d — %s】\n", out.StageOrder, stageName))
		if out.StructuredOutput != "" && out.StructuredOutput != "{}" {
			sb.WriteString(out.StructuredOutput)
			sb.WriteString("\n")
		}
		if strings.TrimSpace(out.NarrativeOutput) != "" {
			narrative := safeUTF8Truncate(out.NarrativeOutput, 500)
			sb.WriteString(fmt.Sprintf("总结:%s\n", narrative))
		}
	}
	return sb.String()
}

// ==================== 阶段专属组件上下文 ====================

func BuildStageComponentContext(ctx context.Context, recipe *models.TeachingRecipe, componentTypesJSON string) string {
	var stageTypes []string
	if err := json.Unmarshal([]byte(componentTypesJSON), &stageTypes); err != nil || len(stageTypes) == 0 {
		return ""
	}
	var allComponentIDs []string
	if err := json.Unmarshal([]byte(recipe.ComponentIDs), &allComponentIDs); err != nil || len(allComponentIDs) == 0 {
		return ""
	}
	groups, err := repository.GetRecipeComponentContents(ctx, allComponentIDs)
	if err != nil || len(groups) == 0 {
		return ""
	}
	typeSet := make(map[string]bool)
	for _, t := range stageTypes {
		typeSet[t] = true
	}
	var sb strings.Builder
	hasContent := false
	for _, g := range groups {
		if !typeSet[g.LibraryType] {
			continue
		}
		if !hasContent {
			sb.WriteString("=== 本阶段参考资料 ===\n")
			hasContent = true
		}
		sb.WriteString(fmt.Sprintf("\n【%s】\n", g.LibraryName))
		for _, c := range g.Components {
			sb.WriteString(fmt.Sprintf("▸ %s\n", c.DisplayLabel))
			if c.DesignLogic != "" {
				sb.WriteString(fmt.Sprintf("  设计逻辑:%s\n", c.DesignLogic))
			}
			if c.FullGuide != "" {
				guide := c.FullGuide
				if len(guide) > 1000 {
					guide = guide[:1000] + "...(已截断)"
				}
				sb.WriteString(fmt.Sprintf("  完整指引:%s\n", guide))
			}
		}
	}
	return sb.String()
}

// ==================== 用户手动选择组件上下文(迭代12)====================

func BuildSelectedComponentContext(ctx context.Context, componentIDs []string) string {
	if len(componentIDs) == 0 {
		return ""
	}
	groups, err := repository.GetRecipeComponentContents(ctx, componentIDs)
	if err != nil || len(groups) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("=== 本阶段参考资料(老师手动选择)===\n")
	sb.WriteString("以下是老师为本阶段特别选择的教学参考组件,请重点参考:\n")
	for _, g := range groups {
		sb.WriteString(fmt.Sprintf("\n【%s】\n", g.LibraryName))
		for _, c := range g.Components {
			sb.WriteString(fmt.Sprintf("▸ %s\n", c.DisplayLabel))
			if c.DesignLogic != "" {
				sb.WriteString(fmt.Sprintf("  设计逻辑:%s\n", c.DesignLogic))
			}
			if c.FullGuide != "" {
				guide := c.FullGuide
				if len(guide) > 1000 {
					guide = guide[:1000] + "...(已截断)"
				}
				sb.WriteString(fmt.Sprintf("  完整指引:%s\n", guide))
			}
		}
	}
	wsPromptLog := logger.WithModule("workshop_stage_prompts")
	wsPromptLog.Info("用户手动选择的组件已注入提示词", "component_count", len(componentIDs), "matched_groups", len(groups))
	return sb.String()
}

// ==================== 自动匹配阶段组件 ====================

// stageTimingMap 阶段代码→课堂时机筛选映射表(v89-4新增)
var stageTimingMap = map[string][]int{
	"analyze": {1, 4},
	"design":  {2, 4},
	"write":   {2},
	"review":  {4},
}

// resolveStageComponentAutoMatch 决定第3层"自动匹配"槽位的注入内容(技能路由 Phase1)。
//
// 这是 BuildStageSystemPromptV2 第3层在"用户未手动选 + 配方无组件"时的兜底产物来源。
// 选择逻辑:
//   - 当技能路由精排开关开启(skillRouterRerankEnabled,定义于 skill_router.go)
//     且老师本轮有发言(recentUserText 非空)时 → 走 RerankedStageComponents 词法精排;
//   - 否则(开关关 / 无发言,如开场白路径)→ 退回 AutoMatchStageComponents 全量匹配保底。
//
// 两者入参契约、输出格式完全一致,故此处可无缝二选一,对第3层之外零影响。
// recentUserText 为空时行为与改造前逐字等价(开场白路径恒走保底,符合"还没有老师发言"的语义)。
func resolveStageComponentAutoMatch(ctx context.Context, componentTypesJSON string, subject string, grade string, stageCode string, recentUserText string) string {
	if skillRouterRerankEnabled && strings.TrimSpace(recentUserText) != "" {
		if out := RerankedStageComponents(ctx, componentTypesJSON, subject, grade, stageCode, recentUserText); out != "" {
			return out
		}
		// 精排返回空(候选为空等)→ 落回全量匹配,保证第3层不因精排空手而丢内容。
	}
	return AutoMatchStageComponents(ctx, componentTypesJSON, subject, grade, stageCode)
}

// AutoMatchStageComponents 根据阶段组件类型+学科+年级自动匹配组件
func AutoMatchStageComponents(ctx context.Context, componentTypesJSON string, subject string, grade string, stageCode string) string {
	var stageTypes []string
	if err := json.Unmarshal([]byte(componentTypesJSON), &stageTypes); err != nil || len(stageTypes) == 0 {
		return ""
	}

	normalizedGrade := utils.NormalizeGradeToNumber(grade)

	matchReq := &models.MatchComponentsRequest{
		Subject:      subject,
		GradeRange:   normalizedGrade,
		LibraryTypes: stageTypes,
		Limit:        2,
	}

	if timings, ok := stageTimingMap[stageCode]; ok {
		matchReq.StageTiming = timings
	}

	groups, err := repository.MatchComponents(ctx, matchReq)
	if err != nil || len(groups) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("=== 本阶段参考资料(系统自动匹配)===\n")
	sb.WriteString("以下是根据学科和年级自动匹配的教学参考组件,请在本阶段工作中适当参考:\n")
	for _, g := range groups {
		sb.WriteString(fmt.Sprintf("\n【%s】\n", g.LibraryName))
		for _, c := range g.Components {
			if c.ComponentIndex != "" {
				sb.WriteString(utils.FormatIndexForPrompt(c.ComponentIndex, c.DisplayLabel))
				sb.WriteString("\n")
			} else {
				sb.WriteString(fmt.Sprintf("▸ %s\n", c.DisplayLabel))
				if c.DesignLogic != "" {
					sb.WriteString(fmt.Sprintf("  设计逻辑:%s\n", c.DesignLogic))
				}
			}
		}
	}
	wsPromptLog := logger.WithModule("workshop_stage_prompts")
	wsPromptLog.Info("自动匹配阶段组件", "subject", subject, "grade", grade, "stage_code", stageCode, "stage_types", stageTypes, "matched_groups", len(groups))
	return sb.String()
}

// ==================== 阶段内对话提示词(v84分层记忆改造)====================

// BuildStageChatPrompt 构建阶段内对话的用户提示词(向下兼容版本)
func BuildStageChatPrompt(lp *models.LessonPlan, stageHistory []*models.ConversationMessage, userMsg *models.ConversationMessage) string {
	return BuildStageChatPromptV2(lp, stageHistory, "", userMsg)
}

// BuildStageChatPromptV2 构建阶段内对话的用户提示词(v84分层记忆版)
//
// v190 修复1:已有教案正文注入改用 safeUTF8Truncate 按 rune 安全截断,
// 上限 chatPromptContentMaxRunes(6000),正常教案不再被截断,
// 根治评审/修订阶段"读到被截断正文 → 误报后半缺失/截断"的问题。
func BuildStageChatPromptV2(
	lp *models.LessonPlan,
	currentStageMessages []*models.ConversationMessage,
	episodicSummary string,
	userMsg *models.ConversationMessage,
) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("【当前备课信息】\n学科:%s\n年级:%s\n课题:%s\n课时:%d分钟\n\n",
		lp.Subject, lp.Grade, lp.Topic, lp.DurationMinutes))

	if lp.ContentMarkdown != "" {
		// v190 修复1:按 rune 安全截断(safeUTF8Truncate),不按字节切,
		// 不切碎汉字成乱码;6000 rune 足以完整容纳正常教案(约3500-4500字符),
		// 正常教案不触发截断;仅极端超长正文做保护性截断防上下文爆量。
		content := safeUTF8Truncate(lp.ContentMarkdown, chatPromptContentMaxRunes)
		sb.WriteString("【已有教案内容】\n")
		sb.WriteString(content)
		sb.WriteString("\n\n")
	}

	if strings.TrimSpace(episodicSummary) != "" {
		sb.WriteString(episodicSummary)
		sb.WriteString("\n")
	}

	recentHistory := currentStageMessages
	if len(recentHistory) > 20 {
		recentHistory = recentHistory[len(recentHistory)-20:]
	}
	if len(recentHistory) > 0 {
		sb.WriteString("【本阶段对话记录】\n")
		for _, h := range recentHistory {
			role := "教师"
			if h.Role == models.ConvRoleAssistant {
				role = "AI助手"
			}
			sb.WriteString(fmt.Sprintf("%s:%s\n", role, h.Content))
		}
		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf("教师:%s\n\nAI助手:", userMsg.Content))
	return sb.String()
}

// ==================== 阶段开场白提示词 ====================

func BuildStageOpeningPrompt(lp *models.LessonPlan, stage *models.WorkshopStage, stageOrder int, totalStages int) string {
	return fmt.Sprintf(`教师正在进行阶段化备课,现在进入第%d/%d个阶段。

备课信息:
学科:%s
年级:%s
课题:%s
课时:%d分钟

当前阶段:%s(%s)
你的角色:%s

请用友好的对话方式开场,简要说明本阶段的目标和你能帮助老师做什么。
不要超过150字,用自然的口吻。如果有前序阶段的成果,简要提及将如何在本阶段利用。`,
		stageOrder, totalStages, lp.Subject, lp.Grade, lp.Topic, lp.DurationMinutes,
		stage.StageName, stage.StageCode, stage.AIRole)
}

// ==================== 辅助函数 ====================

// stageCodeToName 阶段代码转中文名
func stageCodeToName(code string) string {
	nameMap := map[string]string{
		"analyze": "教学分析", "design": "教学设计", "write": "教案撰写",
		"review": "AI评审", "revise": "修订定稿",
	}
	if name, ok := nameMap[code]; ok {
		return name
	}
	return code
}

// safeUTF8Truncate 安全截断UTF-8字符串
//
// 按 rune(字符)截断而非字节,绝不切碎半个汉字成乱码;
// 入参 maxChars 为最大字符数,超出部分用 "..." 替代。
func safeUTF8Truncate(s string, maxChars int) string {
	runes := []rune(s)
	if len(runes) <= maxChars {
		return s
	}
	return string(runes[:maxChars]) + "..."
}

// ==================== v198:教学逻辑内核(pedagogyLogicCore)====================
//
// 背景与动机:
//
//	AI 做教案时倾向于按"语义相似度"拼出一份"看起来像好教案"的东西,而非从学科逻辑的
//	严谨性出发。典型病症有四:① 为了通俗好讲而编造违背学科本质的伪机制(如把 AI 图像识别
//	讲成"先识别颜色再识别形状再组合"这类人类预设串行流水线,违背 AI 从样本中自学特征的
//	本质);② 导入与情境脱离学生真实生活(2025 新课标明确要求与学生实际生活相关联);
//	③ 每个环节换一个新例子,徒增学生(尤其低年级)认知负荷、削弱逻辑主线;④ 通篇堆砌
//	互不关联的前沿术语与华丽活动,却没有一条清晰的核心逻辑主线贯穿。
//
// 本函数产出一段「教学逻辑内核」规范,注入 analyze/design/write 三个阶段的第5层(对话规范),
// 让 AI 从一开始就以"学科逻辑严谨性"而非"语义像不像好教案"为标准来设计与撰写。
//
// 关键校准(用户定调,不可教条化):
//
//	学科本质约束不能限定太死。允许、且【鼓励】用学生这个年龄能听懂的生活化比喻把抽象
//	原理讲明白(合理具象化);要扣的只是"歪曲学科真实机制、会让学生形成错误认知"的那种
//	伪简化。给 AI 的是"正例+反例对照",指明安全区在哪,而非一味禁止具象化把 AI 吓得不敢讲。
//
// 分学段差异(第3条例子连贯性):按 grade 经 NormalizeGradeToSegment 归一为 小学/初中/高中,
//
//	动态拼入对应一档措辞——小学最严(单例贯穿、禁频繁换例),初中(主线+至多一对照例),
//	高中(主线下多角度延展但共享同一核心逻辑)。无法识别学段时用稳妥的通用口径。
func pedagogyLogicCore(grade string) string {
	// 第3条"例子连贯性"按学段差异化:小学最严,初高中递进放宽
	segment := utils.NormalizeGradeToSegment(grade)
	var coherenceRule string
	switch segment {
	case "小学":
		coherenceRule = "本课面向小学生,例子连贯性要求最严:请用【同一个核心例子/情境贯穿全课】,在不同环节上对这一个例子持续深入、层层引申升华(例如导入提出它、新授拆解它、巩固迁移它),【不要每个环节都换一个新例子】——频繁换例会显著增加小学生的认知负荷、打断逻辑主线。除非确有必要,否则全课尽量只围绕这一个例子展开。"
	case "初中":
		coherenceRule = "本课面向初中生:请以【一个核心例子/情境为主线贯穿全课】并层层深入,至多再引入一个起对照或递进作用的例子,且该对照例必须与主例共享同一条核心逻辑;避免每个环节各换新例造成认知负荷与主线断裂。"
	case "高中":
		coherenceRule = "本课面向高中生:允许围绕核心逻辑做多角度延展、引入多个例子,但所有例子都必须服务并回扣【同一条核心学科逻辑】,形成由浅入深的逻辑链,而不是彼此孤立的例子拼盘。"
	default:
		coherenceRule = "例子连贯性:尽量用一个核心例子/情境贯穿全课、层层深入引申,避免每个环节都换新例子增加学生认知负荷、打断逻辑主线;确需第二个例子时,它应与主例共享同一条核心逻辑、起对照或递进作用。"
	}

	return `
== 教学逻辑内核(本阶段的根本要求,优先级高于"看起来像不像好教案")==
1. 学科逻辑锚定:动笔/讨论前先想清楚"本课要让学生真正理解的那一条学科核心逻辑/原理"是什么,本阶段产出的所有内容(分析、目标、环节、例子、活动)都应围绕讲清这一条来组织,而不是各自看着热闹却互不咬合。允许并【鼓励】用学生这个年龄能听懂的生活化比喻把抽象原理讲明白(合理具象化);但比喻只能简化,不能建立"违背学科真实机制、会让学生形成错误认知"的心智模型。
   反例(不要这样):讲"AI图像识别"时说成"AI先识别颜色、再识别形状、再组合起来判断"——这把 AI 讲成了人类预先设定步骤的串行流水线,违背了 AI 从大量样本中自己学到特征的本质,会让学生形成错误理解。
   正例(可以这样):讲"AI图像识别"时说成"AI 看过几百万张猫的照片,自己总结出了'猫大概长什么样'的感觉,而不是我们一条一条教它规则"——既具象、学生能懂,又没歪曲本质。
2. 真实生活化情景引入:导入与核心情境要取自学生【真实的生活经验或当下真实的社会/科技问题】(2025 新课标要求课程设计与学生实际生活相关联),不要用脱离学生经验的抽象设例硬凑。
3. ` + coherenceRule + `
4. 逻辑严谨优先:判断一个环节/活动/例子该不该用,标准是"它是否真的在为本课核心逻辑服务、机制是否站得住脚、是否贴近学生真实生活",而不是"它读起来像不像一个漂亮的教学设计"。不要为了显得丰富就堆砌专业术语、罗列多个互不关联的前沿名词,或堆砌华丽但偏离主线的活动。可以自检:能否用一句话说清本课的学科核心逻辑?每个环节是否都在为这一条服务?
`
}
