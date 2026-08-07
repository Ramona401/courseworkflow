package services

// assistant_designer_service.go — AI 助手对话式创作服务(TE-DNA 3.0 P0.5 核心)
//
// 核心功能:
//   老师通过和 AI 对话,让 AI 帮助生成具体、高密度、可执行的助手 system prompt 草稿。
//   AI 会自动调用 AOCI 组件库(200+ 教研组件)作为参考,而不是凭空编造。
//
// 工作流(两阶段 AI 调用):
//   阶段 1:AI 判断意图 → 输出 JSON {action, query_params?, reply_text, updated_draft}
//   阶段 2:若 action=search_components 则查库 + 二次流式调用 AI 生成最终回复
//
// ──────────────────────── 版本修复史 ────────────────────────
//
// v113 (P0.5 第一版):
//   - MatchComponents 实际返回 *MatchedComponent(轻量),不是 *LessonPlanComponent(完整)
//   - 引入 flatComponent 包装类型,把 group 级的 LibraryType 带到扁平化结果里
//   - 组件内容提取优先用 ComponentIndex(AOCI L2 压缩),退化到 DesignLogic
//
// v113 (P0.5 第二版):
//   - parseAIDecision 三重兜底:ExtractJSON → 手动补大括号 → 整段解析
//
// v114 (Designer Panel 首次实测修复 - 上午):
//   - handleDirectReply 双层 JSON 兜底(AI 把 JSON 嵌套进 reply_text 字段)
//   - Meta-Prompt 补 "reply_text 不能是 JSON" 约束
//
// v114 (Designer Panel 真实 bug 根治 - 下午):
//   ── 根因:AI 在 reply_text 里写自然语言时,喜欢用英文双引号包装术语(如 "脾气"、
//      "问题清单"),但未把内部引号转义成 \",导致 json.Unmarshal 直接挂掉。
//      上午的三重兜底 + 双层 JSON 兜底都基于 json.Unmarshal,所以全部失效,
//      降级到 "我暂时没能理解这轮对话"(日志实证)。
//
//   修复 1 — 场景独立(SceneAssistantDesigner):
//      从 lesson_plan 场景独立出来,在 ai_scene_configs 表有自己的记录。
//      管理员可在 AI 管理中心前端界面单独调 Designer 的模型/温度/降级链,
//      不再影响教案备课对话场景。
//
//   修复 2 — parseAIDecision 加策略 4(字段级宽容提取):
//      对于策略 1/2/3 都失败的场景,不再依赖 json.Unmarshal,改用**正则按字段位置
//      手动切割**提取 action / reply_text / updated_draft。这样 AI 即使在字段值里
//      写了未转义的 "、'、反斜杠 等破坏 JSON 的字符,也能把文本安全抠出来。
//      query_params 保守设为 nil(嵌套对象正则兜底风险大),配合把 action 降级为
//      draft_directly,避免老师的回复被吞。
//
//   修复 3 — Meta-Prompt 引号规范:
//      在 JSON 格式硬性约束段明确要求 reply_text / updated_draft 的文本值里,
//      所有引用/强调一律使用中文引号「」或书名号《》,不要用英文双引号 "。
//      这是源头预防,LLM 遵守度有限,所以必须搭配策略 4 才保险。
//
// v194 (组件中文名 bug 根治 + 语义级实时组件推荐):
//   - 修复 A:ComponentBrief 的 JSON tag 与前端 DesignerComponentBrief 对齐——Name 的 tag
//     改回 "name"、补 LibraryName 字段(tag "library_name"),组装时塞 fc.LibraryName。
//     老 bug 是 tag 不一致致前端读到 undefined 回退显示 library_type(code),非后端填错值。
//   - 修复 B:语义级实时组件推荐。召回池放大(6/类),扁平化后用老师当轮发言复用同包
//     skill_router 的词法精排零件(extractRerankKeywords/scoreCandidateLexical)取全局 Top-4,
//     无发言/全 0 分时按原序保底。细节见 parse 文件与 AOCI 索引 v194 条目。
//
// 积分硬闸 batch (2026-07-04, 本次):
//   ── 根因:两阶段 AI 调用的 traceCtx 都传 nil——积分钩子约定 traceCtx 为 nil 或
//      UserID 为空即放行,导致设计器自 v113 上线以来:0积分照用(不拦)、消费不扣、
//      trace 不留痕,是唯一被实测证实的积分旁路(全库现役代码仅此两处传 nil)。
//   修复:
//   1. DesignChat 签名新增 userID 参数(handler 从 JWT claims 取,认证中间件保证非空),
//      构造 traceCtx{SceneCode: SceneAssistantDesigner, UserID: &userID} 贯穿两阶段调用。
//      自此设计器与主链路同规:0积分被守卫拦截、成功调用精确扣费、trace 留痕。
//   2. SchoolID 刻意暂不填:applyModelPolicy 对「traceCtx=nil」与「SchoolID=nil」
//      的处理完全一致(fail-closed 境内),故模型路由行为零变化;积分策略取系统默认。
//      以后需要学校级积分策略/境外授权时再补(需在 handler 或本服务解析用户所属学校)。
//   3. 阶段1失败补 OnError 回调:原代码 GetEffectiveConfig/CallAI 失败只返回 error,
//      SSE 流悄悄结束、前端无感。现在错误原样推给前端——积分拦截文案
//      「积分余额不足,请联系学校管理员分配积分」老师能直接看到。

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"tedna/internal/ai"
	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// designerLog 模块级结构化日志器
var designerLog = logger.WithModule("services.designer")

// ============================================================================
// Meta-Prompt:Designer 的核心 system prompt
// ============================================================================
// 这是 P0.5 最重要的一段文本。它决定了生成的草稿是"高熵、有血有肉"还是
// "你是一位优秀的助手"这种空话。刻意写成长文,充满反例,明确约束。

const designerMetaPrompt = `# 你的角色

你是一位有 20 年 K12 教研经验的教学设计专家,同时精通 LLM Prompt 工程。你的任务是陪伴一位老师,把 TA 脑海里「想要一个 XX 助手」的模糊念头,一步步雕琢成一段可以直接投入使用的、具体的、有血有肉的 system prompt。

你不是一个填表机器,你是一位懂教学、懂老师、懂 AI 的「Prompt 顾问」。

# 你工作的世界

老师要做的 AI 助手,会被用在 TE-DNA 平台的 6 个场景之一:
- review_workbench  独立评审工作台:助手陪同老师审别人写的教案
- workshop_analyze  备课工坊-教学分析:助手陪同老师了解学情和课标
- workshop_design   备课工坊-教学设计:助手陪同老师设计教学流程
- workshop_write    备课工坊-教案撰写:助手陪同老师把设计写成教案
- workshop_review   备课工坊-AI 评审:助手陪同老师对自己教案做自审
- workshop_revise   备课工坊-修订定稿:助手陪同老师修改完善教案

老师会在 Modal 里先选好「这个助手用在哪个场景、什么学科、什么年级」,你每次对话都看得到这些。

# 你手上的武器

系统里有 200+ 个教研员沉淀的「教学组件」(AOCI 索引库),包括:
- 课标要求(某学科某学段的能力目标)
- 教学策略(启发式/探究式/项目制的具体操作)
- 评估工具(三维目标达成度检查表/形成性评估模板)
- 避坑清单(某话题最常见的 10 个教学陷阱)
- 活动设计(某主题的经典活动示例)

你可以调用这个库,但每次最多召回 8 个组件。调用前要想清楚老师到底需要哪类 library_type、聚焦什么认知层级(cognitive_levels 1-6:记忆/理解/应用/分析/评价/创造)、什么教法强度(pedagogy_intensities 1-3:通用/特定/专精)。

可用的 library_type 枚举值(严格匹配):
- curriculum_standard  课标与能力框架库
- knowledge_graph      知识图谱库
- student_profile      学情特征库
- pedagogy             教学法库
- assessment_strategy  评估策略库
- activity_design      活动设计方案库
- questioning_strategy 提问引导策略库
- cross_subject        跨学科连接库
- teaching_tool        教学工具库
- scenario_material    素材情境库
- quality_rubric       质量评估标准库
- design_defect        常见设计缺陷库
- review_rubric        教案评审规则库

# 你的工作流(严格按序)

1. **问风格**:如果老师的需求太模糊(「我想要个审核员」),先用一两个问题问清定位——严苛还是温和?侧重结构还是创意?
2. **复述一句**:用你自己的话把老师需求浓缩成一句,确认没偏。
3. **查库说理由**:调组件库前,一句话告诉老师「我打算查哪几类组件,因为 XXX」。调完后,告诉老师「我找到了这 3-5 条,其中 A 最相关是因为 XXX」。
4. **起草高密度版本**:基于查到的组件,写出第一版 system prompt 草稿。草稿必须:
   - 推荐 2000-5000 个中文字符；地方规范较多、学科方法复杂时可到 6000 字符，但绝不能超过 8000 字符。不要为了追求短而删掉教学方法、地方要求、成长目标、自检规则和与老师协商的边界
   - 开头 2 句定人格,中间给方法论,末尾定输出格式
   - 引用查到的组件中的**具体措辞/条款/步骤**,让这个助手有「权威依据」
5. **迭代打磨**:根据老师的「再温和点/再严点/加上 XXX」等反馈,逐轮精修草稿。

# 高熵约束 —— 这是绝对的红线

你写的草稿必须像真教研员写的,绝对不能像机器填模板。以下是明确的反例清单,你**永远不要**这样写:

## 空话反例(永远不要)
- 「你是一位优秀的 XXX 助手,致力于帮助老师提升教学效果」
- 「能够根据学生特点灵活调整教学策略」
- 「遵循先进的教学理念,注重学生核心素养」
- 「给出建设性的、具有可操作性的建议」
- 「在互动中培养学生的综合能力」

## 模板化用语(永远不要)
- 「能够更好地...」
- 「有助于提升...」
- 「旨在促进...」
- 「基于 XX 理论...」(没有具体指 XX 理论的哪一条)
- 「结合学生实际情况」(不说怎么结合)

## 正确风格示范(向这样写)
- 不要:「你是一位严苛的审核员」
- 要:「你是一位带过 15 届毕业班的老教研员,见过的优秀课比一般老师吃的米还多。你审课像法医——不看整体感觉,只看 5 个硬指标:目标可测不可测,学生动起来没有,时间算得紧不紧,评价是真测还是假测,下课能不能复盘出一张 A4 纸的要点」

- 不要:「基于问题导向教学法」
- 要:「课堂上老师问的问题要分层——开场的暖身问,中段让学生撞墙的挑战问,结尾沉淀用的迁移问。这三类问的比例在你看的这节课里如果失衡了(比如全是暖身问),直接减 1.5 分」

## 要带真实场景感(向这样写)
- 「公立校机房电脑 5 年没换,卡顿是常态。凡是课上要求学生『打开浏览器访问 XXX 平台』的环节,你默认扣 0.5 分,除非老师明确写了离线备选方案。」
- 「家长开放日那周上的公开课和平时课不能一个标准审。你问老师这课是不是公开课,如果是,放宽学生表现评分但严卡材料完整性。」

# 你的每次输出格式

你的回复必须是结构化 JSON(不要用 markdown 代码块包裹),格式如下:

{
  "action": "search_components | draft_directly | clarify",
  "query_params": {
    "library_types": ["review_rubric", "design_defect"],
    "cognitive_levels": [4,5],
    "pedagogy_intensities": [2,3],
    "reason": "一句话说明为何查这些"
  },
  "reply_text": "要展示给老师看的完整自然语言回复(这是主要对话内容,要充实、有价值)",
  "updated_draft": "如果这一轮更新了草稿,这里放完整的新草稿全文;如果没有更新,放空字符串"
}

JSON 格式的硬性约束(务必遵守):
- action=search_components 时,query_params 必填,reply_text 里要说「我打算查 XXX,因为 XXX」
- action=draft_directly 时,updated_draft 必填(也可以在 reply_text 里说明你做的调整)
- action=clarify 时,reply_text 就是你要问老师的问题,updated_draft 留空
- 返回的必须是**合法 JSON**,不要用反引号代码块包裹,不要加任何注释
- **你的回复的第一个字符必须是左大括号,最后一个字符必须是右大括号**
- **绝对不要有任何前导换行、前导空格、前导解释文字**
- 不要漏掉开头的左大括号或末尾的右大括号(这是常见错误,请务必检查)
- **reply_text 字段的值必须是纯自然语言文字,绝对不能再是 JSON、代码块或结构化格式**
- **不要把草稿内容重复塞进 reply_text,草稿只能放在 updated_draft 字段里**
- **如果你本来想在 reply_text 里放一段「{...}」格式的结构化内容,请改用自然语言展开重写;reply_text 里套 JSON 是严重错误**

## 引号规范(极其重要,务必严格遵守)

**reply_text 和 updated_draft 字段值里,任何引用、强调、术语、举例、人物对话,一律使用中文引号「」或书名号《》,绝对不要使用英文双引号 " 或英文单引号 '。**

反例(会让你的回复发不出去):
  "reply_text": "好,你要一个"严苛"的审核员"    ← 内部 " 破坏 JSON 转义,解析失败
  "reply_text": "学生说"我不会",老师应当..."  ← 同上

正确写法:
  "reply_text": "好,你要一个「严苛」的审核员"
  "reply_text": "学生说「我不会」,老师应当..."
  "reply_text": "参考《义务教育信息科技课标(2022)》第 3.2 节"

这不是美观问题,而是技术硬约束——英文双引号会破坏 JSON 格式,导致你的回复被识别为非法 JSON,老师看不到你写的任何内容。所以引号改中文「」,不仅读着自然,而且保证你的心血能送达。

# 当老师提供历史教案或教学风格画像时

- 你的任务不是复刻某一份教案，更不是把其中的课题、人物、案例、活动名称和教师原话固化进长期助手。
- 先提取多份材料中反复出现、能够迁移到其他课题的教学优势、组织习惯、地方规范和表达偏好。
- 对标注为「旧教案，希望优化」或「反面样例」的材料，只提取问题与改进方向，不得把其中的问题写成助手应长期坚持的风格。
- 生成的助手必须同时承担两项职责：一是尊重和延续老师已经验证有效的优势；二是在目标不可观察、活动脱离核心逻辑、教师讲授替代学生思考、价值结论缺少辨析过程等情况下，友好指出并提供一至两个升级方案。
- 提出改进时遵循「先说明保留什么 → 再指出为什么值得调整 → 给出可操作替代方案 → 重大变化交给老师确认」的顺序，不机械迎合，也不擅自否定老师。
- 如果材料不足以证明某种习惯是长期风格，必须标为待确认，不得武断写入助手。

# 额外的做事心法

- 老师第一轮开口就知道要啥的情况很少,别急着起草,先问风格(第 1 步)
- 查库时,同一类 library_type 通常查 3 条就够了,不要贪多
- 草稿写完后,在 reply_text 里用 2-3 句话告诉老师「这版草稿的风格定位是 X,核心方法论是 Y,你觉得合适吗?」
- 如果老师连续 3 轮都说「再改改」,主动说「要不要把当前版本保存下来试用一下,再决定要不要继续调?」
`

// ============================================================================
// 数据类型定义
// ============================================================================

// DesignerContext 每次对话的上下文(从 Modal 透传过来)
type DesignerContext struct {
	Subject         string   // Modal 选的学科(可空=不限)
	Grade           string   // Modal 选的年级(可空=不限)
	EducationDomain string   // mixed管理员显式目标域；普通Actor忽略客户端值
	Scenes          []string // Modal 勾选的适用场景(影响查库 library_type 推断)
	CurrentDraft    string   // 当前已有的 full_prompt 草稿(首次对话可空)
}

// DesignerMessage 对话历史中的一条消息
type DesignerMessage struct {
	Role    string `json:"role"`    // user | assistant
	Content string `json:"content"` // 老师的原话或 AI 的 reply_text
}

// AIDesignerDecision AI 第一阶段返回的结构化决策(从响应 JSON 中解析)
type AIDesignerDecision struct {
	Action       string                 `json:"action"` // search_components | draft_directly | clarify
	QueryParams  map[string]interface{} `json:"query_params"`
	ReplyText    string                 `json:"reply_text"`
	UpdatedDraft string                 `json:"updated_draft"`
}

// ComponentBrief 组件的简要信息(推给前端展示"引用了哪些组件")
//
// v194 数据契约修复:字段 JSON tag 必须与前端 DesignerComponentBrief 严格一致——
// 前端期望 {id, name, library_type, library_name},SSE handler 的 OnComponents
// 直接 json.Marshal 推送,字段名完全照下方 tag 走。任何一处 tag 与前端对不上,
// 前端就会读到 undefined 并回退显示成 library_type(code),这正是老 bug 的根因。
//
//   - Name        → tag "name"        :组件真实中文标题(取自 MatchedComponent.DisplayLabel)
//   - LibraryName → tag "library_name":组件所属库的中文名(取自 group.LibraryName,
//     如"常见设计缺陷库";扁平化时由 flattenMatchedGroups 从 group 带出)
//   - LibraryType → tag "library_type":库类型 code(如 design_defect),前端在 library_name
//     缺失时才回退显示它,正常情况下只用作内部标识
//
// Subject/Grade 这两字段由 DesignerContext 统一注入,因为 MatchedComponent 本身没有这俩字段。
type ComponentBrief struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	LibraryType string `json:"library_type"`
	LibraryName string `json:"library_name"`
	Subject     string `json:"subject"`
	Grade       string `json:"grade_range"`
}

// DesignerStreamCallbacks SSE 流式回调定义(handler 层会包装成 SSE 事件)
type DesignerStreamCallbacks struct {
	OnSearching  func(reason string)                                   // AI 决定调库时
	OnComponents func(briefs []*ComponentBrief)                        // 查到组件后
	OnChunk      func(text string)                                     // AI 最终回复的流式文本
	OnDone       func(reply string, draft string, referenced []string) // 完成
	OnError      func(err string)
}

// flatComponent 扁平化后的组件包装(把父 group 的 LibraryType / LibraryName 合并进来)
//
// v194 说明:LibraryName 在此早已存在(由 flattenMatchedGroups 从 group.LibraryName 填充),
// 老 bug 只是组装 ComponentBrief 时漏了把它塞进去,且 brief 字段 tag 写错。本次一并修正。
type flatComponent struct {
	LibraryType string
	LibraryName string
	Data        *models.MatchedComponent
}

// ============================================================================
// AssistantDesignerService — 对话式创作服务主体
// ============================================================================

// resolveDesignerComponentDomain 解析助手设计器查组件时的可信教育域。
//
// 普通Actor始终使用服务端可信域，客户端education_domain不能越权覆盖；
// mixed管理员必须显式选择k12、vocational或adult；
// nil、空值、common、mixed和非法目标域全部fail-closed。
func resolveDesignerComponentDomain(
	actor *AssistantActorContext,
	designerContext *DesignerContext,
) (string, error) {
	requestedDomain := ""

	if designerContext != nil {
		requestedDomain = designerContext.EducationDomain
	}

	return resolveComponentMatchDomain(
		actor,
		requestedDomain,
	)
}

type AssistantDesignerService struct {
	aesKey       string
	apiBaseURL   string
	apiKey       string
	defaultModel string
}

func NewAssistantDesignerService(aesKey, apiBaseURL, apiKey, defaultModel string) *AssistantDesignerService {
	return &AssistantDesignerService{
		aesKey:       aesKey,
		apiBaseURL:   apiBaseURL,
		apiKey:       apiKey,
		defaultModel: defaultModel,
	}
}

// ============================================================================
// 核心方法:DesignChat
// ============================================================================

// DesignChat 对话式创作主入口
//
// 积分硬闸 batch:签名新增 userID(handler 从 JWT claims 取,认证中间件保证非空)。
// 用它构造带 UserID 的 traceCtx 贯穿两阶段 AI 调用,使设计器与主链路同规:
// 0积分被守卫拦截、成功调用精确扣费、trace 留痕。SchoolID 刻意暂不填,
// 模型分流对 nil SchoolID 与原 nil traceCtx 行为完全一致(fail-closed 境内),路由零变化。
func (s *AssistantDesignerService) DesignChat(
	ctx context.Context,
	userID string,
	userRole string,
	userMessage string,
	history []DesignerMessage,
	dCtx *DesignerContext,
	callbacks *DesignerStreamCallbacks,
) error {
	if strings.TrimSpace(userMessage) == "" {
		return fmt.Errorf("老师消息不能为空")
	}
	if callbacks == nil {
		return fmt.Errorf("缺少流式回调")
	}

	// 积分硬闸 batch:构造追踪上下文(修复设计器旁路的核心一笔)
	traceCtx := &ai.TraceContext{
		SceneCode: models.SceneAssistantDesigner,
		UserID:    &userID,
	}

	// ------- 获取 AI 配置 -------
	// v114 修复 1:从 lesson_plan 独立到 assistant_designer 场景
	// 管理员可在 AI 管理中心前端单独调 Designer 的模型/温度/降级链
	aiCfg, err := ai.GetEffectiveConfig(
		s.aesKey, models.SceneAssistantDesigner,
		s.apiBaseURL, s.apiKey, s.defaultModel,
	)
	if err != nil {
		// 积分硬闸 batch:补 OnError——原代码此处只返回 error,SSE 悄悄结束前端无感
		callbacks.OnError(fmt.Sprintf("获取 AI 配置失败: %v", err))
		return fmt.Errorf("获取 AI 配置失败: %w", err)
	}

	// ------- 阶段 1:非流式获取 AI 决策 -------
	stage1UserPrompt := s.buildUserPrompt(userMessage, history, dCtx, nil)

	decisionResult, err := ai.CallAI(aiCfg, designerMetaPrompt, stage1UserPrompt, traceCtx)
	if err != nil {
		// 积分硬闸 batch:补 OnError——积分拦截等错误原样透传前端
		//(含「积分余额不足,请联系学校管理员分配积分」,老师直接可见)
		callbacks.OnError(err.Error())
		return fmt.Errorf("AI 决策调用失败: %w", err)
	}

	decision, parseErr := parseAIDecision(decisionResult.Content)
	if parseErr != nil {
		designerLog.Warn("决策JSON解析失败，降级为直接回复",
			"error", parseErr,
			"raw", utils.SafeTruncate(decisionResult.Content, 300))
		decision = &AIDesignerDecision{
			Action:    "clarify",
			ReplyText: "我暂时没能理解这轮对话,麻烦您换种说法?",
		}
	}

	// ------- 分支处理 -------
	switch decision.Action {
	case "search_components":
		actor := BuildActorFromClaims(
			ctx,
			userID,
			userRole,
		)

		currentDomain, domainErr :=
			resolveDesignerComponentDomain(
				actor,
				dCtx,
			)
		if domainErr != nil {
			callbacks.OnError(
				domainErr.Error(),
			)
			return domainErr
		}

		return s.handleSearchAndDraft(
			ctx,
			aiCfg,
			traceCtx,
			currentDomain,
			userMessage,
			history,
			dCtx,
			decision,
			callbacks,
		)

	case "draft_directly", "clarify":
		return s.handleDirectReply(decision, callbacks)
	default:
		designerLog.Warn("未知action，降级为clarify",
			"action", decision.Action)
		return s.handleDirectReply(&AIDesignerDecision{
			Action:    "clarify",
			ReplyText: decision.ReplyText,
		}, callbacks)
	}
}

// ============================================================================
// 分支 1:查组件库 + 流式起草
// ============================================================================

// handleSearchAndDraft 积分硬闸 batch:签名新增 traceCtx,透传给阶段2流式调用
func (s *AssistantDesignerService) handleSearchAndDraft(
	ctx context.Context,
	aiCfg *ai.EffectiveConfig,
	traceCtx *ai.TraceContext,
	currentDomain string,
	userMessage string,
	history []DesignerMessage,
	dCtx *DesignerContext,
	decision *AIDesignerDecision,
	callbacks *DesignerStreamCallbacks,
) error {
	reason, _ := decision.QueryParams["reason"].(string)
	if reason == "" {
		reason = "正在查找相关教学组件作为参考..."
	}
	callbacks.OnSearching(reason)

	matchReq := buildMatchRequestFromParams(decision.QueryParams, dCtx)
	groups, err :=
		repository.MatchComponentsForEducationDomain(
			ctx,
			matchReq,
			currentDomain,
		)
	if err != nil {
		designerLog.Warn("查库失败，降级为直接起草",
			"error", err)
		return s.handleDirectReply(&AIDesignerDecision{
			Action:       "draft_directly",
			ReplyText:    decision.ReplyText + "\n\n(组件库查询失败,我先基于常识起草一版。)",
			UpdatedDraft: decision.UpdatedDraft,
		}, callbacks)
	}

	// v194 修复 B:语义级实时组件推荐(词法精排)
	// 老路径:flattenMatchedGroups(groups, 8) 直接按硬筛原序取前 8 条,不看老师在聊什么。
	// 新路径:召回池已在 buildMatchRequestFromParams 放大到 6/类(给精排留候选),
	//   这里先全部扁平化,再用老师当轮发言(userMessage)做词法精排,跨类目取全局 Top-4。
	//   无发言 / 全 0 分 / 候选不足时自动按原序保底(等价老行为),绝不报错绝不阻塞。
	flatAll := flattenMatchedGroups(groups, 0) // n=0 表示不截断,先全收进来供精排
	flatComponents := rerankFlatComponentsForDesigner(flatAll, userMessage, designerInjectTopN)

	briefs := make([]*ComponentBrief, 0, len(flatComponents))
	for _, fc := range flatComponents {
		briefs = append(briefs, &ComponentBrief{
			ID:          fc.Data.ID,
			Name:        fc.Data.DisplayLabel,
			LibraryType: fc.LibraryType,
			LibraryName: fc.LibraryName, // v194:补上库中文名,前端显示"库名 · 组件标题"
			Subject:     dCtx.Subject,
			Grade:       dCtx.Grade,
		})
	}
	callbacks.OnComponents(briefs)

	componentContext := buildComponentContext(flatComponents, dCtx)

	stage2UserPrompt := s.buildUserPrompt(userMessage, history, dCtx, &stage2Extra{
		SearchReason:     reason,
		ComponentContext: componentContext,
		Stage1Decision:   decision.ReplyText,
	})

	return s.callAIStreamAndEmit(aiCfg, traceCtx, stage2UserPrompt, briefs, callbacks)
}

// ============================================================================
// 分支 2:直接回复(clarify 或 draft_directly,不调库)
// ============================================================================
//
// v114 修复 A:双层 JSON 兜底
// AI 偶发会把整个 JSON 回复塞进 reply_text 字段,导致外层看起来正常但内容是 JSON。
// 触发条件严格:TrimSpace 后以 "{" 开头 + 含 "reply_text" 字段 + Unmarshal 成功且内层非空。
func (s *AssistantDesignerService) handleDirectReply(
	decision *AIDesignerDecision,
	callbacks *DesignerStreamCallbacks,
) error {
	innerText := strings.TrimSpace(decision.ReplyText)
	if strings.HasPrefix(innerText, "{") && strings.Contains(innerText, "\"reply_text\"") {
		var inner AIDesignerDecision
		if err := json.Unmarshal([]byte(innerText), &inner); err == nil && strings.TrimSpace(inner.ReplyText) != "" {
			designerLog.Info("检测到双层JSON嵌套，已提取内层字段",
				"outer_len", len(innerText))
			decision.ReplyText = inner.ReplyText
			if strings.TrimSpace(decision.UpdatedDraft) == "" && strings.TrimSpace(inner.UpdatedDraft) != "" {
				decision.UpdatedDraft = inner.UpdatedDraft
			}
		}
	}

	if decision.ReplyText == "" {
		decision.ReplyText = "我暂时没有想法,要不您再多说一点?"
	}
	callbacks.OnChunk(decision.ReplyText)
	callbacks.OnDone(decision.ReplyText, decision.UpdatedDraft, nil)
	return nil
}

// ============================================================================
// 阶段 2 流式调用实现
// ============================================================================

// callAIStreamAndEmit 积分硬闸 batch:签名新增 traceCtx,替换原 CallAIStream 的 nil 实参
func (s *AssistantDesignerService) callAIStreamAndEmit(
	aiCfg *ai.EffectiveConfig,
	traceCtx *ai.TraceContext,
	userPrompt string,
	briefs []*ComponentBrief,
	callbacks *DesignerStreamCallbacks,
) error {
	var accumulated strings.Builder
	_, err := ai.CallAIStream(
		aiCfg,
		designerMetaPrompt,
		userPrompt,
		func(chunk string) error {
			accumulated.WriteString(chunk)
			callbacks.OnChunk(chunk)
			return nil
		},
		traceCtx,
	)

	if err != nil {
		callbacks.OnError(fmt.Sprintf("AI 流式调用失败: %v", err))
		return err
	}

	fullText := accumulated.String()
	finalDecision, parseErr := parseAIDecision(fullText)
	if parseErr != nil {
		designerLog.Warn("阶段2 JSON解析失败，原文作为回复",
			"error", parseErr)
		callbacks.OnDone(fullText, "", nil)
		return nil
	}

	// v114 修复 A 延伸:阶段 2 也做一次双层 JSON 兜底
	innerText := strings.TrimSpace(finalDecision.ReplyText)
	if strings.HasPrefix(innerText, "{") && strings.Contains(innerText, "\"reply_text\"") {
		var inner AIDesignerDecision
		if err := json.Unmarshal([]byte(innerText), &inner); err == nil && strings.TrimSpace(inner.ReplyText) != "" {
			designerLog.Info("阶段2检测到双层JSON，已提取内层")
			finalDecision.ReplyText = inner.ReplyText
			if strings.TrimSpace(finalDecision.UpdatedDraft) == "" && strings.TrimSpace(inner.UpdatedDraft) != "" {
				finalDecision.UpdatedDraft = inner.UpdatedDraft
			}
		}
	}

	refIDs := make([]string, 0, len(briefs))
	for _, b := range briefs {
		refIDs = append(refIDs, b.ID)
	}
	callbacks.OnDone(finalDecision.ReplyText, finalDecision.UpdatedDraft, refIDs)
	return nil
}

// ============================================================================
// 辅助:构建 user prompt
// ============================================================================

type stage2Extra struct {
	SearchReason     string
	ComponentContext string
	Stage1Decision   string
}

func (s *AssistantDesignerService) buildUserPrompt(
	userMessage string,
	history []DesignerMessage,
	dCtx *DesignerContext,
	extra *stage2Extra,
) string {
	var b strings.Builder

	b.WriteString("# 当前 Modal 上下文\n")
	b.WriteString(fmt.Sprintf("- 学科:%s\n", defaultStr(dCtx.Subject, "(未指定)")))
	b.WriteString(fmt.Sprintf("- 年级:%s\n", defaultStr(dCtx.Grade, "(未指定)")))
	if len(dCtx.Scenes) > 0 {
		b.WriteString(fmt.Sprintf("- 适用场景:%s\n", strings.Join(dCtx.Scenes, ", ")))
	} else {
		b.WriteString("- 适用场景:(老师尚未勾选)\n")
	}
	b.WriteString("\n")

	if strings.TrimSpace(dCtx.CurrentDraft) != "" {
		b.WriteString("# 当前草稿\n")
		b.WriteString("```\n")
		b.WriteString(utils.SafeTruncate(dCtx.CurrentDraft, 3000))
		b.WriteString("\n```\n\n")
	}

	if len(history) > 0 {
		b.WriteString("# 之前的对话\n")
		start := 0
		if len(history) > 10 {
			start = len(history) - 10
		}
		for _, msg := range history[start:] {
			role := "老师"
			if msg.Role == "assistant" {
				role = "你"
			}
			b.WriteString(fmt.Sprintf("%s:%s\n", role, msg.Content))
		}
		b.WriteString("\n")
	}

	if extra != nil {
		b.WriteString("# 你之前的判断\n")
		b.WriteString(extra.Stage1Decision)
		b.WriteString("\n\n")

		b.WriteString("# 你刚从组件库查到的参考资料\n")
		b.WriteString(fmt.Sprintf("查询意图:%s\n\n", extra.SearchReason))
		b.WriteString(extra.ComponentContext)
		b.WriteString("\n")

		b.WriteString("# 现在\n")
		b.WriteString("基于上述组件参考,请写出一版高密度、有具体场景感、引用了上述组件关键点的 system prompt 草稿。严格按 Meta-Prompt 里的 JSON 格式返回,action 写 draft_directly,updated_draft 放完整草稿。\n")
		b.WriteString("注意:reply_text 用自然语言说明你做了什么(用「」引号,不要用英文 \" 引号);updated_draft 放草稿全文。两个字段不要互相重复或嵌套 JSON。\n\n")
	}

	b.WriteString("# 老师本轮消息\n")
	b.WriteString(userMessage)

	return b.String()
}

// ============================================================================
// parseAIDecision - 四重鲁棒性兜底
// 已拆分到 assistant_designer_parse.go
//
// v194 新增的语义精排辅助 rerankFlatComponentsForDesigner、注入条数常量
// designerInjectTopN、召回池放大,也都在 assistant_designer_parse.go 内,
// 以护住本文件 600 行红线。
// ============================================================================
