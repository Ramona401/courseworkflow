package services

// lesson_index_service.go — 教案AOCI索引AI压缩服务
//
// v86新增：P2-1 教案AOCI索引扩展
//
// 职责：
//   1. CompressLessonIndex：调用AI将教案全文压缩为结构化索引
//   2. SaveLessonIndex：保存索引到数据库（含冗余列提取）
//   3. BatchIndexAllLessonPlans：批量为现有教案生成索引
//
// 使用scanner场景（Haiku模型）低成本压缩
//
// v89-2变更：CompressLessonIndex增加planID参数，传入真实TraceContext

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"tedna/internal/ai"
	"tedna/internal/config"
	"tedna/internal/database"
	"tedna/internal/logger"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// 模块日志
var liLog = logger.WithModule("lesson_index")

// LessonIndexService 教案索引服务
type LessonIndexService struct {
	cfg *config.Config
}

// NewLessonIndexService 创建教案索引服务实例
func NewLessonIndexService(cfg *config.Config) *LessonIndexService {
	return &LessonIndexService{cfg: cfg}
}

// CompressLessonIndex 调用AI将教案全文压缩为结构化索引
//
// 使用scanner场景（Haiku模型）低成本压缩
// 从prompts表加载索引字典（prompt_lesson_index）作为system prompt
//
// v89-2变更：增加planID参数，构建真实TraceContext用于成本追踪
//
// 参数：
//   - fullText: 教案全文（由BuildLessonFullText构建）
//   - planID: 教案ID（用于TraceContext关联，可为空字符串）
//
// 返回：
//   - indexText: AI生成的索引文本
//   - error: 错误信息
func (s *LessonIndexService) CompressLessonIndex(fullText string, planID string) (string, error) {
	if strings.TrimSpace(fullText) == "" {
		return "", fmt.Errorf("教案内容为空，无法生成索引")
	}

	// 1. 从prompts表加载索引字典
	dictPrompt, err := repository.GetCurrentPromptByKey("prompt_lesson_index")
	if err != nil {
		return "", fmt.Errorf("加载教案索引字典失败: %w", err)
	}

	// 2. 获取AI配置（使用scanner场景，Haiku模型低成本）
	aiCfg, err := ai.GetEffectiveConfig(
		s.cfg.AESKey,
		"scanner", // 使用scanner场景（Haiku）
		s.cfg.AIAPIBaseURL,
		s.cfg.AIAPIKey,
		s.cfg.AIDefaultModel,
	)
	if err != nil {
		return "", fmt.Errorf("获取AI配置失败: %w", err)
	}

	// 3. v89-2：构建真实TraceContext
	var traceCtx *ai.TraceContext
	if planID != "" {
		pid := planID
		traceCtx = &ai.TraceContext{
			SceneCode:    "scanner",
			LessonPlanID: &pid,
		}
	} else {
		// 批量索引等无特定教案ID的场景，仅标记场景
		traceCtx = &ai.TraceContext{
			SceneCode: "scanner",
		}
	}

	// 4. 调用AI压缩（v89-2：传入traceCtx替代nil）
	result, err := ai.CallAI(aiCfg, dictPrompt.Content, fullText, traceCtx)
	if err != nil {
		return "", fmt.Errorf("AI压缩教案索引失败: %w", err)
	}

	indexText := strings.TrimSpace(result.Content)

	// 4.5 清理AI输出中可能包含的额外内容（标题、代码块、说明等）
	indexText = cleanLessonIndexOutput(indexText)

	// 5. 验证索引格式
	if !utils.ValidateLessonIndex(indexText) {
		return "", fmt.Errorf("AI生成的索引格式无效（缺少SJ:或[O]标签）: %s", truncateStr(indexText, 200))
	}

	liLog.Info("教案索引压缩成功",
		"model", result.ModelUsed,
		"tokens", result.TokensUsed,
		"index_len", len(indexText),
		"plan_id", planID,
	)

	return indexText, nil
}

// SaveLessonIndex 保存教案索引到数据库
//
// 步骤：
//   1. 从indexText提取冗余列值（CG/PQ/ST）
//   2. 计算质量等级（QL）
//   3. 替换索引中的QL:0为实际计算值
//   4. 写入数据库
func (s *LessonIndexService) SaveLessonIndex(ctx context.Context, planID string, indexText string, aiScore *float64, status string) error {
	// 提取冗余列
	cogLevel := utils.ParseLessonCognitiveLevel(indexText)
	pedIntensity := utils.ParseLessonPedagogyIntensity(indexText)
	structType := utils.ParseStructureType(indexText)

	// 计算质量等级
	qualLevel := utils.CalculateQualityLevel(aiScore, status)

	// 替换索引中的QL:0为实际值
	indexText = strings.Replace(indexText, "QL:0", fmt.Sprintf("QL:%d", qualLevel), 1)

	// 写入数据库
	err := repository.UpdateLessonPlanIndex(ctx, planID, indexText, cogLevel, pedIntensity, structType, qualLevel)
	if err != nil {
		return fmt.Errorf("保存教案索引失败: %w", err)
	}

	liLog.Info("教案索引保存成功",
		"plan_id", planID,
		"cg", cogLevel, "pq", pedIntensity, "st", structType, "ql", qualLevel,
	)

	return nil
}

// BatchIndexAllLessonPlans 批量为所有未索引的教案生成AOCI索引
//
// 异步执行，每批20个，间隔1秒防限流
// 只处理有内容的教案（content_markdown非空）
func (s *LessonIndexService) BatchIndexAllLessonPlans() {
	liLog.Info("开始批量教案索引生成")

	ctx := context.Background()

	// 查询所有未索引且有内容的教案
	rows, err := database.DB.Query(ctx, `
		SELECT id, subject, grade, topic, title, duration_minutes,
		       content_markdown, content_structured, ai_review_result,
		       matched_components, ai_review_score, status
		FROM lesson_plans
		WHERE lesson_index = '' AND content_markdown != ''
		ORDER BY updated_at DESC
	`)
	if err != nil {
		liLog.Error("查询未索引教案失败", "error", err)
		return
	}
	defer rows.Close()

	type planInfo struct {
		id, subject, grade, topic, title string
		duration                         int
		contentMd, contentStruct         string
		aiReviewResult, matchedComp      string
		aiScore                          *float64
		status                           string
	}

	var plans []planInfo
	for rows.Next() {
		var p planInfo
		if err := rows.Scan(
			&p.id, &p.subject, &p.grade, &p.topic, &p.title, &p.duration,
			&p.contentMd, &p.contentStruct, &p.aiReviewResult,
			&p.matchedComp, &p.aiScore, &p.status,
		); err != nil {
			liLog.Error("扫描教案行失败", "error", err)
			continue
		}
		plans = append(plans, p)
	}

	total := len(plans)
	liLog.Info("找到未索引教案", "total", total)

	successCount := 0
	failCount := 0

	for i, p := range plans {
		// 构建全文
		fullText := utils.BuildLessonFullText(
			p.subject, p.grade, p.topic, p.title, p.duration,
			p.contentMd, p.contentStruct, p.aiReviewResult, p.matchedComp,
		)

		// AI压缩（v89-2：传入planID）
		indexText, err := s.CompressLessonIndex(fullText, p.id)
		if err != nil {
			liLog.Error("教案索引压缩失败",
				"plan_id", p.id, "title", p.title, "error", err,
			)
			failCount++
			continue
		}

		// 保存
		if err := s.SaveLessonIndex(ctx, p.id, indexText, p.aiScore, p.status); err != nil {
			liLog.Error("教案索引保存失败",
				"plan_id", p.id, "error", err,
			)
			failCount++
			continue
		}

		successCount++
		log.Printf("[教案索引] 进度 %d/%d 成功（ID: %s, 标题: %s）",
			i+1, total, p.id, truncateStr(p.title, 30))

		// 每批20个后暂停1秒防限流
		if (i+1)%20 == 0 && i+1 < total {
			log.Printf("[教案索引] 已处理%d个，暂停1秒...", i+1)
			time.Sleep(1 * time.Second)
		}
	}

	liLog.Info("批量教案索引完成",
		"total", total, "success", successCount, "fail", failCount,
	)
}

// truncateStr 截断字符串（辅助日志）
func truncateStr(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

// cleanLessonIndexOutput 清理AI输出，提取纯净的教案索引内容
//
// AI经常在索引前后添加额外内容：
//   - Markdown标题（# AI教案索引压缩结果）
//   - 代码块标记（```...```）
//   - 压缩说明表格
//   - 亮点分析等
//
// 策略：找到编码行（含SJ:和|）作为起点，找到最后一个语义标签行（[X]开头）作为终点
// 只保留起点到终点之间的内容
func cleanLessonIndexOutput(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}

	lines := strings.Split(text, "\n")
	startIdx := -1
	endIdx := -1

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// 找到编码行（包含SJ:和|分隔符，且不是Markdown表格行）
		if startIdx == -1 && strings.Contains(trimmed, "SJ:") && strings.Contains(trimmed, "|") && !strings.HasPrefix(trimmed, "|") {
			startIdx = i
		}

		// 找到语义标签行 [O] [S] [M] [H] [R]（方括号开头，第3个字符是]）
		if startIdx >= 0 && len(trimmed) >= 3 && trimmed[0] == '[' && trimmed[2] == ']' {
			endIdx = i
		}
	}

	// 如果找到了有效范围，提取该范围内容
	if startIdx >= 0 && endIdx >= startIdx {
		var result []string
		for i := startIdx; i <= endIdx; i++ {
			result = append(result, lines[i])
		}
		return strings.TrimSpace(strings.Join(result, "\n"))
	}

	// 没找到有效范围，做简单的代码块清理兜底
	// 去掉开头的 ``` 行
	cleaned := text
	for strings.HasPrefix(cleaned, "```") {
		idx := strings.Index(cleaned, "\n")
		if idx >= 0 {
			cleaned = cleaned[idx+1:]
		} else {
			break
		}
	}
	// 去掉结尾的 ```
	cleaned = strings.TrimSpace(cleaned)
	if strings.HasSuffix(cleaned, "```") {
		cleaned = cleaned[:len(cleaned)-3]
	}
	return strings.TrimSpace(cleaned)
}
