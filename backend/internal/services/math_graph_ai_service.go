package services

// math_graph_ai_service.go — 批次A·老师自定义数学图形 AI 生成/改编服务
// (2026-07-08首发;批次A+拍题出图;提示词v2常见错误自查,2026-07-09)
//
// 背景：课件工坊「数学动态图形」已有 48 个 JSXGraph 参数化模板,但老师最常见的需求是
//   "这个模板很好,但我想改成我的题目"。本服务提供多模式的构造代码 AI 生成能力:
//     - adapt(模板变种):输入 = 模板当前 buildConstruction 产出代码(含参数取值)+ 老师的
//       变种描述(如"把河改成折线"/"再加一个 SAS 对比"),输出 = 修改后的完整构造代码。
//       有底稿的改写成功率远高于从零生成,且天然继承模板的版式与教学设计,
//       48 个模板由此升维成"48 个题族的种子"。
//     - create(从零生成):老师纯自然语言描述图形,AI 从零产出构造代码,兜底覆盖
//       模板库没有的题型。
//     - 批次A+·拍题出图:create/adapt 均可携带题目照片(教辅拍照/试卷截图),走多模态
//       CallAIMultimodal 让 AI 直接读题(文字+题目配图一并理解,比传统 OCR 更强——
//       几何题的图形本身往往比文字更重要),产出对应的交互图形。图片仅首轮携带,
//       追改走纯文本 adapt(题意已转成代码)。
//     - 一键修复:前端预览执行报错时,把"当前代码+报错信息"以 adapt 模式回喂本接口,
//       AI 按【常见错误自查清单】定位修复,老师无需理解报错(提示词v2配套能力)。
//   各模式共用同一条系统提示词的变体(单接口多模式)。
//
// 提示词v2(2026-07-09):真实用户报错驱动的加固——gemini 产出过
//   board.create('text',[x,y],{...}) 两父元素写法(漏第三个内容参数)导致
//   "Can't create text with parent types 'number' and 'number'"。新增
//   【常见错误自查清单】枚举 text/angle/slider/glider 的父元素铁律,并要求输出前逐条自查。
//
// 生成产物约定(与前端 mathGraphUtils/MathGraphModal 的单一真相源架构严格对齐):
//   产出一段"操作已存在的 board 变量"的纯 JS 构造代码——
//     - 弹窗预览:new Function('board','JXG',code)(board, JXG) 直接执行;
//     - 课件融入:generateMathGraphEmbed 把同一段代码包进自包含 HTML。
//   代码执行前前端会统一过 applyMathPalette 调色板映射,故提示词强制 AI 只用
//   映射表左列的"原始高饱和色值",换装由前端完成,与 48 个内置模板视觉一体。
//
// 场景码：刻意复用 "courseware_media_prompt"(课件媒体提示词场景,已实配 gemini
//   多模态模型并走境内外分流)。组织索引明确警告:新建场景码会回落
//   ai_configs.default_model='豆包',而豆包在 OneAPI 无渠道会 503。故绝不新建场景码,
//   直接复用,计费与分流天然生效。gemini 天然支持 Vision,拍题出图与文本生成同一场景。
//   (与 lesson_plan_ref_service 复用 lesson_plan 场景同一理由、同一范本。)
//
// 路径: backend/internal/services/math_graph_ai_service.go

import (
	"context"
	"fmt"
	"strings"

	aiClient "tedna/internal/ai"
	"tedna/internal/logger"
	"tedna/internal/repository"
)

// 模块日志
var mathGraphLog = logger.WithModule("math_graph_ai")

// mathGraphSceneCode 场景码:复用课件媒体提示词场景(gemini 已实配),绝不新建裸场景码
const mathGraphSceneCode = "courseware_media_prompt"

// 输入长度上限(rune 计,防超长输入浪费 token)
const (
	// 老师描述上限:2000 字足够表达任何题目变化
	mathGraphDescMaxRunes = 2000
	// 底稿构造代码上限:现有 48 模板产出代码普遍 <5000 字,3 万上限极宽裕
	mathGraphBaseMaxRunes = 30000
	// 批次A+:题目图片 data URI 上限(字符计)。前端压缩后普遍 <1MB base64;
	// 12M 字符≈9MB 原图,极宽裕,超出说明前端压缩失效,直接拒绝防内存与带宽浪费
	mathGraphImageMaxChars = 12000000
)

// mathGraphDefaultBoundingBox 从零生成模式的默认画板范围(前端未传时兜底)
const mathGraphDefaultBoundingBox = "[-10, 8, 10, -6]"

// MathGraphAIService 数学图形 AI 生成/改编服务(持 cfg 供 AI 调用取密钥)
type MathGraphAIService struct {
	cfg interface{ GetAESKey() string }
}

// NewMathGraphAIService 创建数学图形 AI 服务
func NewMathGraphAIService(cfg interface{ GetAESKey() string }) *MathGraphAIService {
	return &MathGraphAIService{cfg: cfg}
}

// MathGraphGenInput 生成/改编入参(handler 侧解析后传入)
type MathGraphGenInput struct {
	// Mode 模式:"adapt" 模板变种(带底稿) / "create" 从零生成
	Mode string
	// Description 老师的自然语言描述(变种描述或新图形描述;带图时可为补充说明,允许较短)
	Description string
	// BaseCode 底稿构造代码(adapt 模式必填 = 模板 buildConstruction 当前产出)
	BaseCode string
	// TemplateName 底稿模板名称(adapt 模式可选,帮助 AI 理解教学场景)
	TemplateName string
	// BoundingBox 画板坐标范围字符串,如 "[-10, 8, 10, -6]"(可选,缺省用默认值)
	BoundingBox string
	// Image 批次A+:题目图片 data URI(data:image/jpeg;base64,xxx,可选)。
	// 非空时走多模态调用,AI 直接读题出图;空时与批次A行为完全一致(纯文本)。
	Image string
}

// buildMathGraphSystemPrompt 构建系统提示词——共享核心规范 + 模式差异段。
// 核心规范逐条对齐前端三个文件的硬约束(mathGraphUtils 模板编写规范 /
// mathGraphTemplateShared 滑杆写法与版式 / applyMathPalette 调色约定)。
// hasImage=true 时追加拍题出图任务段(读题→设计交互→出图)。
func (s *MathGraphAIService) buildMathGraphSystemPrompt(mode string, boundingBox string, hasImage bool) string {
	var b strings.Builder
	b.WriteString("你是一名精通 JSXGraph 1.12 的 K12 数学交互课件工程师。你的任务是产出一段可直接执行的 JSXGraph 构造代码,用于课堂交互演示。\n\n")

	b.WriteString("【运行环境(必须严格遵守)】\n")
	b.WriteString("1. 代码运行在一个已初始化完成的画板上:变量 board(JXG.Board 实例)与全局对象 JXG 已存在,直接使用。\n")
	b.WriteString("2. 绝对禁止调用 JXG.JSXGraph.initBoard 或创建新画板;绝对禁止操作 DOM(document/window);绝对禁止输出 HTML 或 <script> 标签。\n")
	b.WriteString("3. 画板 boundingBox 固定为 " + boundingBox + "(格式[左,上,右,下]),所有元素坐标必须落在该范围内并留出边距;画板已锁定纵横比,几何图形不会变形。\n")
	b.WriteString("4. 坐标轴与网格由外层配置控制,你不需要也不允许创建坐标轴。\n\n")

	b.WriteString("【代码硬规范(违反任何一条都会导致渲染失败)】\n")
	b.WriteString("1. 只输出纯 JavaScript 构造代码本身:不要任何解释文字、不要 Markdown 代码围栏(```)、不要前言后语。\n")
	b.WriteString("2. 字符串一律用单引号,禁止双引号字符串与反引号模板串。\n")
	b.WriteString("3. 代码中绝对禁止出现 \"</script\" 这一字符序列(会破坏外层 HTML 嵌入)。\n")
	b.WriteString("4. 数学撇号必须用 ′(U+2032),如 A′、B′,禁止用单引号字符冒充(会破坏字符串)。\n")
	b.WriteString("5. 用 var 声明变量(代码可能在非严格模式的拼接环境中运行,禁止 let/const 以外泄块级语义,统一 var 最稳妥)。\n")
	b.WriteString("6. 控制在 120 行以内,注释精简(单行 // 中文注释,不要块注释)。\n\n")

	// 提示词v2:真实报错驱动的父元素铁律清单(text 两父元素错误已在生产出现过)
	b.WriteString("【常见错误自查清单(每条都是真实渲染失败案例,输出前必须逐条自查)】\n")
	b.WriteString("1. text 必须三个父元素 [x, y, 内容],内容是字符串或函数,绝不能只给 [x, y] 把文字放属性里:\n")
	b.WriteString("   ✅ board.create('text', [1, 2, '标注文字'], {fontSize:13});\n")
	b.WriteString("   ✅ board.create('text', [1, 2, function(){ return 'k=' + k.Value().toFixed(1); }], {fontSize:14});\n")
	b.WriteString("   ❌ board.create('text', [1, 2], {name:'标注文字'});  // 报错:Can't create text with parent types 'number' and 'number'\n")
	b.WriteString("2. slider 父元素必须是三个平级数组 [[x1,y],[x2,y],[min,初值,max]],绝不能嵌套成 [[[x1,y],[x2,y]],[min,初值,max]]。\n")
	b.WriteString("3. angle 父元素是三个点 [边上点1, 顶点, 边上点2](顶点居中),不能传数字。\n")
	b.WriteString("4. glider 父元素是 [x初值, y初值, 依附的线/圆对象],第三个必须是已创建的元素变量。\n")
	b.WriteString("5. 引用其他元素做动态计算时,确保该元素已在前文创建(先创建后引用,禁止前向引用)。\n")
	b.WriteString("6. 函数式父元素(function(){...})内绝对禁止用 this:this 不指向任何元素,this.Value()/this.X() 会报 this.Value is not a function。一律直接用元素变量名,如 sliderAD.Value()、P.X():\n")
	b.WriteString("   ✅ board.create('text', [x, y, function(){ return 'AD = ' + sliderAD.Value().toFixed(1); }], {fontSize:14});\n")
	b.WriteString("   ❌ board.create('text', [x, y, function(){ return 'AD = ' + this.Value().toFixed(1); }], {fontSize:14});  // 报错:this.Value is not a function\n")
	b.WriteString("7. 函数式父元素里取滑杆值一律 用变量名.Value(),取点坐标一律 用变量名.X()/.Y(),都是方法调用要带括号。\n")
	b.WriteString("8. 输出前通读一遍代码,确认每个 board.create 的父元素个数与类型都符合上述铁律,并全文搜索确认没有任何 this. 出现。\n\n")

	b.WriteString("【交互与版式规范(与平台 48 个内置模板保持一致的观感)】\n")
	b.WriteString("1. 可调量一律做成 JSXGraph 滑杆,注意父元素必须是三个平级数组:\n")
	b.WriteString("   var k = board.create('slider', [[x1, y], [x2, y], [min, 初值, max]], {name:'k', snapWidth:0.1, strokeColor:'#7C3AED', fillColor:'#7C3AED', highline:{strokeColor:'#7C3AED', strokeWidth:3}, baseline:{strokeColor:'#E5E7EB', strokeWidth:2}, label:{fontSize:14}});\n")
	b.WriteString("   滑杆放在画板顶部靠左,一行一个,自上而下排列;滑杆读数用 k.Value() 获取。\n")
	b.WriteString("2. 动态读数文字放顶部靠右,用函数式 text 实时刷新:\n")
	b.WriteString("   board.create('text', [x, y, function(){ return 'k = ' + k.Value().toFixed(1); }], {fontSize:14, strokeColor:'#1F2937', fixed:true});\n")
	b.WriteString("3. 版式分区:顶部=控制区(滑杆+读数),中部=教学图形主体(独占最大区域),底部=一行灰色操作提示(fontSize:12, strokeColor:'#6B7280', fixed:true),如'拖动滑杆观察图形变化'。\n")
	b.WriteString("4. 建议在顶部控制区下方垫一块半透明白背板把控件与网格分层(有滑杆时才需要):\n")
	b.WriteString("   board.create('polygon', [[x1,y1],[x2,y1],[x2,y2],[x1,y2]], {fillColor:'#FFFFFF', fillOpacity:0.72, borders:{strokeColor:'#E9EDF4', strokeWidth:1, layer:2}, vertices:{visible:false}, fixed:true, highlight:false, layer:2});\n")
	b.WriteString("5. 可拖拽点(glider/自由点)是课堂交互主体,合理设置 name 标签;辅助线用 dash:2 虚线。\n")
	b.WriteString("6. 需要动态计算时优先用函数式父元素(function(){...})保证联动刷新;角度弧用 board.create('angle',...),注意弧度制。\n\n")

	b.WriteString("【配色规范(必须只用下列原始色值,平台会统一做主题化映射,私自用其他颜色会破坏视觉一体性)】\n")
	b.WriteString("- 主体图形/函数曲线/斜边:#2563EB(蓝)\n")
	b.WriteString("- 强调点/对边/根:#DC2626(红)\n")
	b.WriteString("- 镜像/第二对象/邻边:#059669(绿)\n")
	b.WriteString("- 对称轴/角弧/强调元素:#F59E0B(橙)\n")
	b.WriteString("- 滑杆/位似/多边形:#7C3AED(紫)\n")
	b.WriteString("- 顶点标签/结论文字:#1F2937(深灰);提示文字:#6B7280(中灰)\n")
	b.WriteString("- 水面/河流:#0EA5E9(水蓝);浅色填充可用 #DBEAFE/#FECACA/#D1FAE5/#FDE68A\n")
	b.WriteString("- 点的标准写法:fillColor 与 strokeColor 同色成对,如 {fillColor:'#DC2626', strokeColor:'#DC2626', size:3}(平台会自动升级为白环点样式)。\n\n")

	// 批次A+:拍题出图任务段(有图时优先级最高,放在模式段之前)
	if hasImage {
		b.WriteString("【拍题出图(用户随消息附了一张题目图片,这是本次任务的核心输入)】\n")
		b.WriteString("1. 仔细读图:完整理解题目文字(题干、已知条件、设问)与题目配图(几何图形、函数图象、标注)——配图往往比文字更关键,务必按配图还原图形结构与点线关系。\n")
		b.WriteString("2. 设计交互:判断题中哪些量适合做成滑杆(如动点参数、边长、角度),哪些点适合做成可拖拽点,让静态题目变成课堂可演示的动态图形。\n")
		b.WriteString("3. 忠实还原:点的名称、线段关系、图形结构严格按题目来,不要自行更换字母或增删条件;题干可提炼一句要点作为底部标注文字。\n")
		b.WriteString("4. 如用户同时给了文字补充说明(如'只画第2小问'/'把动点P做成滑杆'),按说明聚焦,说明优先于你自己的判断。\n")
		b.WriteString("5. 图片模糊或题意不完整时,基于可辨认部分给出最合理的图形,宁可少画不要臆造。\n\n")
	}

	if mode == "adapt" {
		b.WriteString("【本次任务:模板变种(有底稿的改写)】\n")
		b.WriteString("用户会给你一段现成的构造代码(底稿)和一个变化描述。你要:\n")
		b.WriteString("1. 以底稿为基础做最小必要修改来实现用户描述的变化——能保留的滑杆、背板、读数、提示、配色、版式全部原样保留;\n")
		b.WriteString("2. 底稿中与变化无关的教学设计(标注、辅助线、结论文字)不要动;\n")
		b.WriteString("3. 变化引入的新元素严格遵守上述全部规范,与底稿风格无缝一体;\n")
		b.WriteString("4. 若用户描述里包含执行报错信息(如 Can't create ...),优先对照【常见错误自查清单】定位底稿中的病灶行并修复,其余部分保持不动;\n")
		if hasImage {
			b.WriteString("5. 若附了题目图片,变化目标 = 把底稿改造成该题目对应的图形(底稿提供版式与交互骨架,题目提供内容)。\n")
		}
		b.WriteString("最后输出修改后的【完整】构造代码(不是差异片段),可直接整段替换执行。\n")
	} else {
		b.WriteString("【本次任务:从零生成】\n")
		if hasImage {
			b.WriteString("用户给了题目图片(可能附文字补充)。你要按上面【拍题出图】的要求,从零产出该题目对应的完整交互构造代码。\n")
		} else {
			b.WriteString("用户会用自然语言描述一个数学图形。你要:\n")
			b.WriteString("1. 理解其教学意图(演示什么概念、学生要观察什么),据此决定哪些量做成滑杆、哪些点可拖拽;\n")
			b.WriteString("2. 严格遵守上述全部规范,从零产出完整构造代码;\n")
			b.WriteString("3. 图形要有教学表达力:关键元素有 name 标签,有动态读数或结论文字,底部有操作提示。\n")
		}
	}
	return b.String()
}

// buildMathGraphUserPrompt 构建用户提示词:把底稿(如有)与描述包好交给 AI
func (s *MathGraphAIService) buildMathGraphUserPrompt(in *MathGraphGenInput) string {
	var b strings.Builder
	hasImage := strings.TrimSpace(in.Image) != ""
	if in.Mode == "adapt" {
		if strings.TrimSpace(in.TemplateName) != "" {
			b.WriteString("【底稿模板名称】" + strings.TrimSpace(in.TemplateName) + "\n\n")
		}
		b.WriteString("【底稿构造代码(基于它做最小必要修改)】\n")
		b.WriteString(in.BaseCode)
		if hasImage {
			b.WriteString("\n\n【用户补充说明(题目见附图)】\n")
		} else {
			b.WriteString("\n\n【用户要求的变化】\n")
		}
		b.WriteString(in.Description)
		b.WriteString("\n\n请输出修改后的完整构造代码(纯 JS,无围栏无解释)。")
	} else {
		if hasImage {
			b.WriteString("【用户补充说明(题目见附图,按图出交互图形)】\n")
		} else {
			b.WriteString("【用户描述的新图形】\n")
		}
		b.WriteString(in.Description)
		b.WriteString("\n\n请输出完整构造代码(纯 JS,无围栏无解释)。")
	}
	return b.String()
}

// stripMathGraphFences 剥离 AI 输出可能带的 Markdown 代码围栏与语言标记。
// 兼容 ```javascript / ```js / ``` 三种围栏开头;无围栏则原样返回。
func stripMathGraphFences(raw string) string {
	out := strings.TrimSpace(raw)
	if !strings.HasPrefix(out, "```") {
		return out
	}
	// 去掉首行围栏(含语言标记)
	if idx := strings.Index(out, "\n"); idx >= 0 {
		out = out[idx+1:]
	} else {
		return ""
	}
	// 去掉末尾围栏
	out = strings.TrimSpace(out)
	out = strings.TrimSuffix(out, "```")
	return strings.TrimSpace(out)
}

// Generate 生成/改编构造代码主入口。
// callerID 供构造 TraceContext(计费归属 + 境内外分流),与 lesson_plan_ref 同款。
// 批次A+:in.Image 非空时走 CallAIMultimodal(拍题出图),否则走 CallAI(纯文本)。
// 返回可直接执行的纯 JS 构造代码字符串。
func (s *MathGraphAIService) Generate(ctx context.Context, callerID string, in *MathGraphGenInput) (string, error) {
	// ---- 入参校验与规范化 ----
	if in == nil {
		return "", fmt.Errorf("请求参数为空")
	}
	in.Mode = strings.TrimSpace(in.Mode)
	if in.Mode != "adapt" && in.Mode != "create" {
		return "", fmt.Errorf("mode 必须是 adapt(模板变种) 或 create(从零生成)")
	}
	in.Description = strings.TrimSpace(in.Description)
	in.Image = strings.TrimSpace(in.Image)
	hasImage := in.Image != ""
	// 带图时描述可为空(题目全在图上);纯文本时描述必填
	if in.Description == "" && !hasImage {
		return "", fmt.Errorf("图形描述为空")
	}
	if in.Description == "" {
		in.Description = "请根据附图的题目生成对应的交互图形。"
	}
	if descRunes := []rune(in.Description); len(descRunes) > mathGraphDescMaxRunes {
		in.Description = string(descRunes[:mathGraphDescMaxRunes])
	}
	// 批次A+:图片校验(格式 + 体量)
	if hasImage {
		if !strings.HasPrefix(in.Image, "data:image/") {
			return "", fmt.Errorf("题目图片格式不正确(需为 data:image/... 的 base64 数据)")
		}
		if len(in.Image) > mathGraphImageMaxChars {
			return "", fmt.Errorf("题目图片过大,请压缩后重试")
		}
	}
	in.BaseCode = strings.TrimSpace(in.BaseCode)
	if in.Mode == "adapt" {
		if in.BaseCode == "" {
			return "", fmt.Errorf("模板变种模式必须提供底稿代码 base_code")
		}
		if baseRunes := []rune(in.BaseCode); len(baseRunes) > mathGraphBaseMaxRunes {
			return "", fmt.Errorf("底稿代码过长(超过 %d 字符),请检查前端传参", mathGraphBaseMaxRunes)
		}
	}
	if strings.TrimSpace(in.BoundingBox) == "" {
		in.BoundingBox = mathGraphDefaultBoundingBox
	}

	// ---- 加载 AI 配置(复用已配场景,分流与计费天然生效) ----
	aiCfg, err := aiClient.GetEffectiveConfig(s.cfg.GetAESKey(), mathGraphSceneCode, "", "", "")
	if err != nil {
		return "", fmt.Errorf("AI配置加载失败: %w", err)
	}

	systemPrompt := s.buildMathGraphSystemPrompt(in.Mode, in.BoundingBox, hasImage)
	userPrompt := s.buildMathGraphUserPrompt(in)

	// 解析作者所属学校ID,供分流判定境外授权(查不到空串→fail-closed降级境内)。
	// schoolIDPtr 定义于 lesson_plan_gen_service.go(同包复用)。
	schoolID, _ := repository.GetSchoolIDByUserID(ctx, callerID)
	uid := callerID
	traceCtx := &aiClient.TraceContext{
		SceneCode: mathGraphSceneCode,
		UserID:    &uid,
		SchoolID:  schoolIDPtr(schoolID),
	}

	// 批次A+:带图走多模态,不带图走纯文本(两入口的重试/降级/分流/积分行为一致)
	var result *aiClient.CallResult
	if hasImage {
		result, err = aiClient.CallAIMultimodal(aiCfg, systemPrompt, userPrompt, in.Image, traceCtx)
	} else {
		result, err = aiClient.CallAI(aiCfg, systemPrompt, userPrompt, traceCtx)
	}
	if err != nil {
		return "", fmt.Errorf("图形代码生成失败: %w", err)
	}

	// ---- 产物清洗与安全校验 ----
	code := stripMathGraphFences(result.Content)
	if code == "" {
		return "", fmt.Errorf("生成结果为空,请换个说法重试")
	}
	// 禁 </script:该序列会破坏外层 HTML 嵌入(与模板编写规范同一条硬约束)
	if strings.Contains(strings.ToLower(code), "</script") {
		return "", fmt.Errorf("生成的代码包含非法片段(</script),已拦截,请重试")
	}
	// 禁自建画板:构造代码只允许操作外部传入的 board 变量
	if strings.Contains(code, "initBoard") {
		return "", fmt.Errorf("生成的代码试图自建画板(initBoard),已拦截,请重试")
	}

	mathGraphLog.Info("数学图形构造代码生成完成",
		"mode", in.Mode, "caller", callerID, "template", in.TemplateName,
		"with_image", hasImage,
		"desc_len", len([]rune(in.Description)), "code_len", len([]rune(code)),
		"tokens", result.TokensUsed)

	return code, nil
}
