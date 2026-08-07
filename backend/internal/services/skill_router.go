package services

// skill_router.go — 技能路由（Skill Router）·Phase 1 后端核心
//
// ============================ 这是什么 ============================
// 「备课工坊 Harness 化改造」PRD §6 的技能路由薄层。它的唯一职责是：
// 在不改动六层提示词拼接引擎、阶段机、芯片协议、设计忠实规则的前提下，
// 改变"喂给两个既有注入槽位的内容由谁决定"——
//
//   槽位一·知识型技能（叠加注入）：原由 AutoMatchStageComponents「按类型全量匹配」产出；
//     本路由提供 RerankedStageComponents 作为可替换的同形态产物——先按既有索引硬筛出
//     候选集，再用老师当轮发言做词法精排，取全局 Top-N（默认 4）注入。
//
//   槽位二·风格型技能（替换第4层）：原仅当老师手动传 assistant_id 时才挂助手；
//     本路由提供 RouteDefaultAssistant，在老师没传 assistant_id 时，按
//     「场景(阶段)+学科+学段+可见性」自动解析出一个默认助手 ID，交给既有的
//     LoadActiveAssistantForUse 去加载 full_prompt（激活校验+使用量埋点都复用既有路径）。
//
// ============================ Harness 七条对照 ============================
//  - 编排隐形：本文件只产出"注入内容/默认助手ID"，不产生任何面向老师的 UI。
//  - 按需加载：知识型技能按相关性当场精排，不提前全摆。
//  - 能力自描述：精排打分读组件索引里天然的 [F][T][P][D][C] 语义标签，数据驱动不写死映射。
//  - 强默认+逃生口：零配置即能跑；任意一步异常都安静降级，绝不报错给老师、绝不阻塞对话。
//  - 优雅降级：精排无命中/老师没发言→退回"学科+年级+阶段"全量匹配保底地板；
//             默认助手解析无命中→返回空串，调用方继续用阶段原生第4层（即不替换）。
//
// ============================ 纯增量与回滚 ============================
// 本文件为全新文件，不修改任何现有文件，单独编译通过、对运行时零影响（在被调用前）。
// 两项能力各由一个包级 feature flag 包裹，可独立开关、独立回滚：
//   skillRouterRerankEnabled           —— 知识型技能精排开关
//   skillRouterDefaultAssistantEnabled —— 风格型默认助手解析开关
// 两个开关默认 true（本期目标即启用），如需紧急回退改为 false 重新部署即可：
//   - 关 rerank   → 调用方应回退到 AutoMatchStageComponents（在后续接线时以此分支兜底）
//   - 关 default  → RouteDefaultAssistant 直接返回空串 → 调用方走"老师没选助手=不替换第4层"老行为

import (
	"context"
	"sort"
	"strings"

	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/utils"
)

// ==================== Feature Flags（可独立开关、独立回滚）====================

// skillRouterRerankEnabled 知识型技能精排开关。
// true：RerankedStageComponents 执行"硬筛候选集→词法精排→Top-N"；
// false：RerankedStageComponents 直接退回保底全量匹配（等价于不精排，行为最接近老路径）。
var skillRouterRerankEnabled = true

// skillRouterDefaultAssistantEnabled 风格型默认助手解析开关。
// true：RouteDefaultAssistant 正常解析默认助手；
// false：RouteDefaultAssistant 恒返回空串（调用方据此走"老师没选助手=不替换第4层"老行为）。
var skillRouterDefaultAssistantEnabled = true

// ==================== 可调参数（常量，集中便于调参）====================

const (
	// skillRouterTopN 知识型技能精排后全局注入的上限条数（PRD 建议 3-5，本期取 4）。
	// 注意：这是"跨 library_type 拍平后的全局 Top-N"，不是每类各取 N。
	skillRouterTopN = 4

	// skillRouterCandidateLimitPerType 硬筛阶段每个 library_type 取的候选条数。
	// MatchComponents 的 Limit 是"每 library_type 的 ROW_NUMBER Top-N"，
	// 为了给精排留足候选池，这里放大到 6（原 AutoMatchStageComponents 仅取 2）。
	skillRouterCandidateLimitPerType = 6

	// skillRouterRecentTextMaxRunes 老师当轮发言用于精排打分时的截断上限（按 rune）。
	// 发言通常很短，截断只为防御异常超长输入烧无谓算力。
	skillRouterRecentTextMaxRunes = 400

	// 词法精排打分权重：命中语义标签的分值高于命中标题。
	skillRouterScoreTagHit   = 3 // 候选索引的 [F][T][P][D][C] 标签行命中一个关键词 +3
	skillRouterScoreTitleHit = 1 // 候选标题命中一个关键词 +1

	// skillRouterKeywordMinRunes 参与匹配的关键词最小长度（按 rune），过滤"的/了/在"等噪音。
	skillRouterKeywordMinRunes = 2
)

var skillRouterLog = logger.WithModule("skill_router")

// ==================== 槽位二：风格型默认助手解析 ====================

// stageCodeToAssistantScene 阶段代码 → AI 助手场景常量映射。
// 备课五阶段 analyze/design/write/review/revise 一一对应到助手 scenes：
//
//	analyze → workshop_analyze
//	design  → workshop_design
//	write   → workshop_write
//	review  → workshop_review
//	revise  → workshop_revise
//
// 不可识别的 stageCode 返回空串（调用方据此跳过默认助手解析）。
func stageCodeToAssistantScene(stageCode string) string {
	switch stageCode {
	case "analyze":
		return models.SceneWorkshopAnalyze
	case "design":
		return models.SceneWorkshopDesign
	case "write":
		return models.SceneWorkshopWrite
	case "review":
		return models.SceneWorkshopReview
	case "revise":
		return models.SceneWorkshopRevise
	default:
		return ""
	}
}

// defaultAssistantLister 风格型默认助手解析所需的最小依赖接口。
// 之所以用接口而非直接依赖 *AIAssistantService，是为了：
//
//	1）解耦——本路由只需要"按可见性列出助手"这一个能力；
//	2）可测——接线/测试时可注入替身。
//
// *AIAssistantService 的 ListAssistants 方法签名与之天然吻合，接线时直接传入即可。
type defaultAssistantLister interface {
	ListAssistants(
		ctx context.Context,
		actor *AssistantActorContext,
		scene, subject, gradeRange string,
		onlyActive bool,
	) (*models.AIAssistantListResponse, error)
}

// assistantSourceRank 把助手来源映射为 PRD 要求的优先级序号（数字越小越优先）。
//
// PRD §6.4 优先级：个人置顶 > 本教研组 > 本校 > 系统通用兜底。
// 注意：底层 ai_assistant_repo.go 的 ListAIAssistants 的 ORDER BY 是
// system→0/group→1/personal→2（系统优先），与本要求【正好相反】，
// 因此本路由必须对返回列表【重新排序】，不能沿用仓储自带顺序。
//
// 当前数据模型 source 仅有 system/group/personal 三档（group 即"本校"维度）；
// "教研组"独立档要等 Phase 2 scope 统一后才有，届时在此补一档即可，不影响现逻辑。
func assistantSourceRank(source string) int {
	switch source {
	case models.AssistantSourcePersonal:
		return 0 // 个人置顶
	case models.AssistantSourceGroup:
		return 1 // 本校（Phase 2 后细分教研组时在此之前插入"教研组"档）
	case models.AssistantSourceSystem:
		return 2 // 系统通用兜底
	default:
		return 3 // 未知来源排最后
	}
}

// RouteDefaultAssistant 解析"老师没手动选助手时"应默认挂载的助手 ID。
//
// 入参：
//   - lister：助手列表能力（接线时传 *AIAssistantService）。
//   - actor ：操作者上下文（由调用方 BuildActorFromClaims 构造好后传入，复用同一份，不重复反查）。
//   - stageCode：当前阶段（analyze/design/write/review/revise）。
//   - subject / grade：教案学科与年级（grade 原样传入，仓储内部会做学段归一化）。
//
// 返回：
//   - assistantID：解析到的默认助手 ID；解析不到任何可见助手时返回空串。
//
// 设计要点：
//
//	1）只返回 ID，不返回 full_prompt——把"加载内容+激活校验+使用量埋点"统一交给既有的
//	   LoadActiveAssistantForUse，避免两套加载路径、保证埋点口径一致。
//	2）优先级重排：仓储按系统优先返回，本函数按 PRD 重排为 个人>本校>系统，取最优一条。
//	3）静默降级：flag 关闭 / 场景不可识别 / 列表查询失败 / 无可见助手 → 一律返回空串，
//	   绝不 panic、绝不把错误抛给对话流。调用方拿到空串就继续用阶段原生第4层。
func RouteDefaultAssistant(
	ctx context.Context,
	lister defaultAssistantLister,
	actor *AssistantActorContext,
	stageCode string,
	subject string,
	grade string,
) string {
	// 开关关闭：直接返回空串，调用方走"未选助手=不替换第4层"老行为。
	if !skillRouterDefaultAssistantEnabled {
		return ""
	}
	if lister == nil || actor == nil {
		return ""
	}

	scene := stageCodeToAssistantScene(stageCode)
	if scene == "" {
		// 阶段无对应助手场景（理论上不会发生），不解析默认助手。
		return ""
	}

	// 按 场景+学科+学段+可见性 列出"激活的"可见助手。
	// onlyActive=true：默认助手必须是启用状态，停用的不参与自动挂载。
	resp, err := lister.ListAssistants(ctx, actor, scene, subject, grade, true)
	if err != nil {
		skillRouterLog.Warn("默认助手解析-列表查询失败，降级为不挂默认助手",
			"stage", stageCode, "scene", scene, "subject", subject, "error", err)
		return ""
	}
	if resp == nil || len(resp.Assistants) == 0 {
		// 该场景下无任何可见助手——这是正常情况（例如老师所在学校还没建任何技能），
		// 调用方据空串继续用阶段原生第4层，不报错。
		return ""
	}

	// 二次严格复核：即使仓储、测试替身或未来其它列表实现返回宽松候选，
	// 默认路由也只接受学科、具体年级和当前场景全部明确一致的助手。
	strictItems := make(
		[]*models.AIAssistantListItem,
		0,
		len(resp.Assistants),
	)
	for _, item := range resp.Assistants {
		if strictAssistantMatchesListItem(
			item,
			subject,
			grade,
			scene,
		) {
			strictItems = append(strictItems, item)
		}
	}
	if len(strictItems) == 0 {
		skillRouterLog.Info(
			"默认助手解析-没有具体年级严格匹配候选，使用系统阶段骨架",
			"stage", stageCode,
			"scene", scene,
			"subject", subject,
			"grade", grade,
		)
		return ""
	}

	// 关键：按 PRD 优先级重排（个人>本校>系统），同档内保持仓储原有的次级排序稳定性。
	// 用 SliceStable 保证同 source 内不打乱仓储已排好的 sort_order/created_at 次序。
	items := strictItems
	sort.SliceStable(items, func(i, j int) bool {
		return assistantSourceRank(items[i].Source) < assistantSourceRank(items[j].Source)
	})

	chosen := items[0]
	skillRouterLog.Info("默认助手已解析",
		"stage", stageCode, "scene", scene, "subject", subject, "grade", grade,
		"assistant_id", chosen.ID, "assistant_name", chosen.Name, "source", chosen.Source,
		"candidate_count", len(items))
	return chosen.ID
}

// ==================== 槽位一：知识型技能精排 ====================

// rerankCandidate 精排过程中的候选项内部表示（拍平后的单条组件）。
type rerankCandidate struct {
	libraryType  string
	libraryName  string
	component    *models.MatchedComponent
	score        int // 词法精排得分；保底场景（无发言）下全部为 0，按 quality 原序
	originalRank int // 在硬筛结果里的原始顺序，用作打分相同时的稳定次级键
}

// 域感知运行时精排入口位于RerankedStageComponentsForRuntime。

// formatRerankedComponents 把精排后选中的候选格式化为可注入第3层的中文文本。
//
// 输出格式与 AutoMatchStageComponents 保持一致（同样的抬头风格、同样的【库名】分组、
// 同样用 utils.FormatIndexForPrompt 渲染索引），使其作为后者的"同形态可替换产物"，
// 接线时一行 swap 即可，AI 端读到的注入结构无任何变化。
//
// 与老路径的唯一可见差别：抬头从"系统自动匹配"改为"智能匹配"，以如实反映这是精排结果；
// 这只是抬头措辞，不影响 AI 对组件内容的理解。
func formatRerankedComponents(chosen []*rerankCandidate) string {
	if len(chosen) == 0 {
		return ""
	}

	// 维持选中顺序的前提下按 library_type 归组（同类相邻展示，更易读）。
	var groupOrder []string
	grouped := make(map[string][]*rerankCandidate)
	groupName := make(map[string]string)
	for _, cand := range chosen {
		if _, ok := grouped[cand.libraryType]; !ok {
			groupOrder = append(groupOrder, cand.libraryType)
			groupName[cand.libraryType] = cand.libraryName
		}
		grouped[cand.libraryType] = append(grouped[cand.libraryType], cand)
	}

	var sb strings.Builder
	sb.WriteString("=== 本阶段参考资料(智能匹配)===\n")
	sb.WriteString("以下是根据本节课内容与老师当前讨论自动匹配的教学参考组件,请在本阶段工作中适当参考:\n")
	for _, lt := range groupOrder {
		sb.WriteString("\n【" + groupName[lt] + "】\n")
		for _, cand := range grouped[lt] {
			c := cand.component
			if c.ComponentIndex != "" {
				sb.WriteString(utils.FormatIndexForPrompt(c.ComponentIndex, c.DisplayLabel))
				sb.WriteString("\n")
			} else {
				sb.WriteString("▸ " + c.DisplayLabel + "\n")
				if c.DesignLogic != "" {
					sb.WriteString("  设计逻辑:" + c.DesignLogic + "\n")
				}
			}
		}
	}
	return sb.String()
}

// ==================== 词法精排辅助 ====================

// extractRerankKeywords 从老师当轮发言中提取用于词法匹配的关键词。
//
// 朴素切词策略（v1 确定性、零依赖）：
//
//	1）先按"非中日韩文字 & 非字母数字"的字符切分（空格、标点、符号都作分隔）；
//	2）连续的 CJK 文字作为一个整块保留（中文不依赖空格分词，整块参与子串包含匹配）；
//	3）过滤掉长度 < skillRouterKeywordMinRunes 的碎片（滤掉"的/了/吗"等单字噪音）。
//
// 返回的关键词用于在候选索引文本里做"包含(Contains)"判断——
// 这是词法重合，不是语义理解；语义增强是 PRD v2 的事，不在本期。
func extractRerankKeywords(text string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	// 截断超长输入。
	runes := []rune(text)
	if len(runes) > skillRouterRecentTextMaxRunes {
		runes = runes[:skillRouterRecentTextMaxRunes]
		text = string(runes)
	}

	var tokens []string
	var buf []rune
	flush := func() {
		if len(buf) == 0 {
			return
		}
		seg := string(buf)
		buf = buf[:0]
		if len([]rune(seg)) >= skillRouterKeywordMinRunes {
			tokens = append(tokens, seg)
		}
	}
	for _, r := range text {
		if isCJKRune(r) || isAlphaNumRune(r) {
			buf = append(buf, r)
		} else {
			flush()
		}
	}
	flush()

	// 去重（保持出现顺序）。
	seen := make(map[string]bool, len(tokens))
	var uniq []string
	for _, t := range tokens {
		key := strings.ToLower(t)
		if seen[key] {
			continue
		}
		seen[key] = true
		uniq = append(uniq, t)
	}
	return uniq
}

// scoreCandidateLexical 对单个候选组件做词法打分。
//
// 打分对象分两部分，权重不同：
//   - 索引里的语义标签行（[F]/[T]/[P]/[D]/[C] 开头的行）：命中一个关键词 +skillRouterScoreTagHit；
//   - 组件标题 DisplayLabel：命中一个关键词 +skillRouterScoreTitleHit。
//
// 命中以"关键词作为子串出现"为准（大小写不敏感）。同一关键词在标签与标题各计一次。
//
// 设计取舍：标签是组件作者写的"何时适用/方法特征"，相关性信号最强，故权重最高；
// 标题次之。不引入 TF-IDF/向量等重器——那是 v2 语义增强的范畴。
func scoreCandidateLexical(c *models.MatchedComponent, keywords []string) int {
	if c == nil || len(keywords) == 0 {
		return 0
	}
	tagText := strings.ToLower(extractSemanticTagLines(c.ComponentIndex))
	titleText := strings.ToLower(c.DisplayLabel)

	score := 0
	for _, kw := range keywords {
		k := strings.ToLower(kw)
		if k == "" {
			continue
		}
		if tagText != "" && strings.Contains(tagText, k) {
			score += skillRouterScoreTagHit
		}
		if titleText != "" && strings.Contains(titleText, k) {
			score += skillRouterScoreTitleHit
		}
	}
	return score
}

// extractSemanticTagLines 从组件索引文本里抽出所有语义标签行（[X]... 行），拼成一段供打分。
//
// 组件索引文本结构（见 utils/aoci_component.go）：
//
//	第一行：编码行 "LT:..|SJ:..|GR:..|CG:..|TM:..|PQ:.."
//	其后若干行：语义标签行，形如 "[F]用于... [T]... [P]..."（首字符 '[' 且第三字符 ']'）
//
// 本函数只取语义标签行（跳过编码行），因为编码是结构维度、不含自然语言关键词，
// 对词法匹配无意义。判定口径与 utils.FormatIndexForPrompt 内收集 tagLines 的口径一致。
func extractSemanticTagLines(indexText string) string {
	if indexText == "" {
		return ""
	}
	var sb strings.Builder
	for _, line := range strings.Split(indexText, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		// 跳过编码行（含 "LT:" 且含 "|"）。
		if strings.Contains(trimmed, "LT:") && strings.Contains(trimmed, "|") {
			continue
		}
		// 收集语义标签行：首字符 '[' 且第三字符 ']'（与 FormatIndexForPrompt 同口径）。
		r := []rune(trimmed)
		if len(r) >= 3 && r[0] == '[' && r[2] == ']' {
			sb.WriteString(trimmed)
			sb.WriteString(" ")
		}
	}
	return sb.String()
}

// ==================== 字符判定小工具 ====================

// isAlphaNumRune 判断是否为 ASCII 字母或数字（用于保留英文词与数字 token）。
func isAlphaNumRune(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}
