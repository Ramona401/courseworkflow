package services

// lesson_plan_ref_service.go — 备课参考资料附件(PDF/Word)压缩服务
//
// 背景：老师在对话/专家模式上传 PDF/Word 作为备课参考资料。前端在浏览器端提取文字
//   (docx 走 JSZip、文字版 PDF 走 pdf.js，均纯前端零落盘)。短文档直接原文注入；
//   长文档(≥3000字)先调本服务把原文 AI 压缩成"保留知识点/要求/重点、去冗余"的结构化要点，
//   再由前端每轮 chat 携带压缩结果注入(会话级、不落库、用完即走)。
//
// 与已有能力的一致性：这与平台"教案预处理规范化层""课程标准压缩"是同一打法——
//   预处理一次、缓存复用，避免每轮重复压缩浪费 token、拖慢响应。
//
// 场景码：刻意复用 "lesson_plan"(已配真实模型 + 走境内外分流)。
//   组织索引明确警告：新建场景码会回落 ai_configs.default_model='豆包'，而豆包在 OneAPI
//   无渠道会 503。故绝不新建场景码，直接复用备课文本场景，计费与分流天然生效。
//   (与 unit_plan_service.go 复用 lesson_plan 场景同一理由、同一范本。)

import (
        "context"
        "fmt"
        "strings"

        aiClient "tedna/internal/ai"
        "tedna/internal/logger"
        "tedna/internal/repository"
)

var lpRefLog = logger.WithModule("lp_ref_material")

// refCompressSceneCode 参考资料压缩场景码：复用备课文本场景，避免新场景回落豆包 503
const refCompressSceneCode = "lesson_plan"

// refCompressInputMaxRunes 压缩服务接受的原文上限(rune 计)。
//   超出部分直接截断——参考资料不是主体，几万字的书本没必要整本喂进去；
//   40000 rune(约 2~3 万汉字)足以覆盖绝大多数单元/章节级参考资料，超出者截断即可。
const refCompressInputMaxRunes = 40000

// LessonPlanRefService 参考资料压缩服务(持 cfg 供 AI 调用取密钥)
type LessonPlanRefService struct {
        cfg interface{ GetAESKey() string }
}

// NewLessonPlanRefService 创建参考资料压缩服务
func NewLessonPlanRefService(cfg interface{ GetAESKey() string }) *LessonPlanRefService {
        return &LessonPlanRefService{cfg: cfg}
}

// buildRefCompressSystemPrompt 构建压缩用系统提示词。
//   目标：把冗长参考资料压成"备课能直接用"的结构化要点——保留知识点、教学要求、重点
//   段落、关键结论/数据/例子，去掉排版垃圾、重复套话、与备课无关的旁支。
//   刻意用朴素 ASCII 与中文，不引入围栏/JSON，输出纯文本要点即可(注入用)。
func (s *LessonPlanRefService) buildRefCompressSystemPrompt(subject, grade string) string {
        var b strings.Builder
        b.WriteString("你是一名资深教研员，正在帮老师把一份较长的备课参考资料压缩成便于备课直接引用的结构化要点。\n\n")
        b.WriteString("【压缩目标】\n")
        b.WriteString("1. 保留：核心知识点、教学要求(学到什么程度)、重点与难点、关键结论/定义/数据/例子、可直接引用的原话要点。\n")
        b.WriteString("2. 去除：排版垃圾、页眉页脚、重复套话、与本资料主题无关的旁支、纯装饰性描述。\n")
        b.WriteString("3. 忠实：只压缩不虚构，绝不臆造原文没有的内容；有明确知识点/要求就分条列清。\n\n")
        b.WriteString("【输出格式】\n")
        b.WriteString("- 直接输出压缩后的结构化要点纯文本(可用分条/小标题组织)，不要加任何前言、说明、代码围栏或 JSON。\n")
        b.WriteString("- 篇幅控制在原文的三分之一以内，抓主干，宁精炼不啰嗦。\n")
        if strings.TrimSpace(subject) != "" || strings.TrimSpace(grade) != "" {
                b.WriteString("\n【聚焦范围】\n")
                b.WriteString(fmt.Sprintf("本资料服务于【%s】【%s】的备课，压缩时优先保留与该学科学段直接相关的内容。\n",
                        upDashRef(subject), upDashRef(grade)))
        }
        return b.String()
}

// CompressRefMaterial 把长参考资料原文压缩为结构化要点。
//
// 入参 content 为前端提取出的原文；subject/grade/fileName 供 AI 聚焦与日志(可空)。
// 返回压缩后的要点文本 + 原文/压缩后字数(rune 计)。
//
// callerID 供构造 TraceContext(计费归属 + 境内外分流)，与 unit_plan_service.callUnitAI 同款。
func (s *LessonPlanRefService) CompressRefMaterial(
        ctx context.Context,
        callerID string,
        content string,
        fileName string,
        subject string,
        grade string,
) (string, int, int, error) {
        content = strings.TrimSpace(content)
        if content == "" {
                return "", 0, 0, fmt.Errorf("参考资料内容为空")
        }

        // 原文过长先截断(rune 安全，避免中文截半)，参考资料不必整本喂入。
        origRunes := []rune(content)
        originalLen := len(origRunes)
        if originalLen > refCompressInputMaxRunes {
                content = string(origRunes[:refCompressInputMaxRunes])
                lpRefLog.Info("参考资料原文超上限，已截断后压缩",
                        "file", fileName, "orig_runes", originalLen, "truncated_to", refCompressInputMaxRunes)
        }

        aiCfg, err := aiClient.GetEffectiveConfig(s.cfg.GetAESKey(), refCompressSceneCode, "", "", "")
        if err != nil {
                return "", originalLen, 0, fmt.Errorf("AI配置加载失败: %w", err)
        }

        systemPrompt := s.buildRefCompressSystemPrompt(subject, grade)
        userPrompt := s.buildRefCompressUserPrompt(content, fileName)

        // 解析作者所属学校ID，供分流判定境外授权(查不到空串→fail-closed降级境内)。
        // schoolIDPtr 定义于 lesson_plan_gen_service.go(同包复用)。
        schoolID, _ := repository.GetSchoolIDByUserID(ctx, callerID)
        uid := callerID
        traceCtx := &aiClient.TraceContext{
                SceneCode: refCompressSceneCode,
                UserID:    &uid,
                SchoolID:  schoolIDPtr(schoolID),
        }

        result, err := aiClient.CallAI(aiCfg, systemPrompt, userPrompt, traceCtx)
        if err != nil {
                return "", originalLen, 0, fmt.Errorf("参考资料压缩失败: %w", err)
        }

        compressed := strings.TrimSpace(result.Content)
        if compressed == "" {
                return "", originalLen, 0, fmt.Errorf("压缩结果为空，请重试")
        }

        compressedLen := len([]rune(compressed))
        lpRefLog.Info("参考资料压缩完成",
                "file", fileName, "caller", callerID,
                "orig_len", originalLen, "compressed_len", compressedLen,
                "tokens", result.TokensUsed)

        return compressed, originalLen, compressedLen, nil
}

// buildRefCompressUserPrompt 构建压缩用户提示词：把待压缩原文包好交给 AI。
func (s *LessonPlanRefService) buildRefCompressUserPrompt(content, fileName string) string {
        var b strings.Builder
        if strings.TrimSpace(fileName) != "" {
                b.WriteString(fmt.Sprintf("【资料文件名】%s\n\n", fileName))
        }
        b.WriteString("【待压缩的参考资料原文】\n")
        b.WriteString(content)
        b.WriteString("\n\n请按系统指令把上面的资料压缩成便于备课引用的结构化要点纯文本。")
        return b.String()
}

// upDashRef 空值占位(供聚焦范围拼接)，与 unit_plan_service.upDash 同义但独立命名避免撞名。
func upDashRef(s string) string {
        if strings.TrimSpace(s) == "" {
                return "未指定"
        }
        return s
}
