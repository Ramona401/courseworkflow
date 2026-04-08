package services

// component_index_service.go — 组件AOCI索引AI压缩服务
//
// v83新增：P1-1 组件AOCI索引体系
//
// 包含：
//   - CompressComponentIndex：对单个组件调用AI生成压缩索引
//   - SaveComponentIndex：保存索引文本+更新冗余列
//   - BatchCompressAllComponents：批量为所有未索引组件生成索引
//   - 内部辅助：加载索引字典prompt、调用AI、解析并存储
//
// v89-2变更：CompressComponentIndex传入真实TraceContext替代nil

import (
	"context"
	"fmt"
	"strings"
	"time"

	aiClient "tedna/internal/ai"
	"tedna/internal/database"
	"tedna/internal/logger"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// ==================== 常量 ====================

// componentIndexSceneCode 组件索引压缩使用的AI场景
// 使用scanner场景（Haiku模型），低成本快速压缩
const componentIndexSceneCode = "scanner"

// componentIndexPromptKey 组件索引字典在prompts表中的key
const componentIndexPromptKey = "prompt_component_index"

// ==================== 日志 ====================

var ciLog = logger.WithModule("component_index")

// ==================== 服务结构体 ====================

// ComponentIndexService 组件索引压缩服务
type ComponentIndexService struct {
	cfg interface{ GetAESKey() string }
}

// NewComponentIndexService 创建组件索引服务
func NewComponentIndexService(cfg interface{ GetAESKey() string }) *ComponentIndexService {
	return &ComponentIndexService{cfg: cfg}
}

// ==================== 单个组件索引压缩 ====================

// CompressComponentIndex 对单个组件调用AI生成压缩索引
//
// 流程：
//   1. 从prompts表加载索引字典作为system prompt
//   2. 拼接组件全文作为user prompt
//   3. 调用AI（Haiku，低成本）生成压缩索引
//   4. 从索引文本解析冗余列值
//   5. 写入数据库
func (s *ComponentIndexService) CompressComponentIndex(ctx context.Context, componentID string) error {
	// 1. 读取组件完整信息
	comp, err := repository.GetComponentByID(ctx, componentID)
	if err != nil {
		return fmt.Errorf("读取组件失败: %w", err)
	}

	// 2. 加载索引字典prompt
	systemPrompt, err := s.loadIndexDictPrompt(ctx)
	if err != nil {
		return fmt.Errorf("加载索引字典失败: %w", err)
	}

	// 3. 构建组件全文
	userPrompt := utils.BuildComponentFullText(
		comp.LibraryType, comp.Subject, comp.GradeRange,
		comp.DisplayLabel, comp.DesignLogic, comp.FullGuide, comp.ExampleSnippet,
	)

	// 4. 调用AI压缩
	aiCfg, err := aiClient.GetEffectiveConfig(s.cfg.GetAESKey(), componentIndexSceneCode, "", "", "")
	if err != nil {
		return fmt.Errorf("AI配置加载失败: %w", err)
	}

	// v89-2：构建真实TraceContext（组件索引压缩，无plan/user，仅标记场景）
	traceCtx := &aiClient.TraceContext{
		SceneCode: componentIndexSceneCode,
	}

	result, err := aiClient.CallAI(aiCfg, systemPrompt, userPrompt, traceCtx)
	if err != nil {
		return fmt.Errorf("AI压缩调用失败: %w", err)
	}

	indexText := strings.TrimSpace(result.Content)
	if indexText == "" {
		return fmt.Errorf("AI返回空索引")
	}

	// 5. 清理AI可能添加的markdown标记
	indexText = cleanIndexOutput(indexText)

	// 6. 验证索引格式（至少包含LT:和[F]）
	if !strings.Contains(indexText, "LT:") || !strings.Contains(indexText, "[F]") {
		ciLog.Warn("AI生成的索引格式异常", "component_id", componentID, "index_preview", safePreview(indexText, 100))
		return fmt.Errorf("索引格式验证失败")
	}

	// 7. 保存索引
	if err := s.SaveComponentIndex(ctx, componentID, indexText); err != nil {
		return fmt.Errorf("保存索引失败: %w", err)
	}

	ciLog.Info("组件索引压缩完成",
		"component_id", componentID,
		"display_label", comp.DisplayLabel,
		"index_len", len(indexText),
		"tokens_used", result.TokensUsed,
	)
	return nil
}

// ==================== 保存索引 ====================

// SaveComponentIndex 保存索引文本到数据库，同时更新冗余索引列
func (s *ComponentIndexService) SaveComponentIndex(ctx context.Context, componentID string, indexText string) error {
	// 从索引文本解析冗余列值
	cogLevel := utils.ParseCognitiveLevel(indexText)
	timing := utils.ParseStageTiming(indexText)
	pedIntensity := utils.ParsePedagogyIntensity(indexText)

	query := `
		UPDATE lesson_plan_components
		SET component_index = $1,
		    idx_cognitive_level = $2,
		    idx_stage_timing = $3,
		    idx_pedagogy_intensity = $4,
		    updated_at = $5
		WHERE id = $6
	`
	_, err := database.DB.Exec(ctx, query,
		indexText, cogLevel, timing, pedIntensity, time.Now(), componentID,
	)
	if err != nil {
		return fmt.Errorf("更新组件索引失败: %w", err)
	}

	ciLog.Info("保存组件索引",
		"component_id", componentID,
		"cognitive_level", cogLevel,
		"stage_timing", timing,
		"pedagogy_intensity", pedIntensity,
	)
	return nil
}

// ==================== 批量压缩 ====================

// BatchCompressAllComponents 批量为所有未索引的active+approved组件生成索引
//
// 参数：
//   - batchSize: 每批处理数量（建议10-20，控制AI调用速率）
//   - delayMs: 每个组件之间的延迟毫秒数（防止API限流，建议500-1000）
//
// 返回：成功数、失败数、错误
func (s *ComponentIndexService) BatchCompressAllComponents(ctx context.Context, batchSize int, delayMs int) (int, int, error) {
	if batchSize <= 0 {
		batchSize = 20
	}
	if delayMs <= 0 {
		delayMs = 500
	}

	// 查询所有未索引的组件ID
	query := `
		SELECT id, display_label
		FROM lesson_plan_components
		WHERE status = 'active'
		  AND review_status = 'approved'
		  AND (component_index = '' OR component_index IS NULL)
		ORDER BY library_type, subject
		LIMIT $1
	`
	rows, err := database.DB.Query(ctx, query, batchSize)
	if err != nil {
		return 0, 0, fmt.Errorf("查询未索引组件失败: %w", err)
	}
	defer rows.Close()

	type compItem struct {
		ID           string
		DisplayLabel string
	}
	var items []compItem
	for rows.Next() {
		var item compItem
		if err := rows.Scan(&item.ID, &item.DisplayLabel); err != nil {
			return 0, 0, fmt.Errorf("扫描组件行失败: %w", err)
		}
		items = append(items, item)
	}

	if len(items) == 0 {
		ciLog.Info("没有未索引的组件，批量压缩跳过")
		return 0, 0, nil
	}

	ciLog.Info("开始批量压缩组件索引", "total", len(items), "batch_size", batchSize, "delay_ms", delayMs)

	successCount := 0
	failCount := 0
	for i, item := range items {
		ciLog.Info("压缩组件索引",
			"progress", fmt.Sprintf("%d/%d", i+1, len(items)),
			"component_id", item.ID,
			"display_label", item.DisplayLabel,
		)

		if err := s.CompressComponentIndex(ctx, item.ID); err != nil {
			ciLog.Warn("组件索引压缩失败",
				"component_id", item.ID,
				"display_label", item.DisplayLabel,
				"error", err,
			)
			failCount++
		} else {
			successCount++
		}

		// 延迟控制API调用速率
		if i < len(items)-1 && delayMs > 0 {
			time.Sleep(time.Duration(delayMs) * time.Millisecond)
		}
	}

	ciLog.Info("批量压缩完成",
		"total", len(items),
		"success", successCount,
		"failed", failCount,
	)
	return successCount, failCount, nil
}

// ==================== 内部辅助 ====================

// loadIndexDictPrompt 从prompts表加载组件索引字典
func (s *ComponentIndexService) loadIndexDictPrompt(ctx context.Context) (string, error) {
	prompt, err := repository.GetCurrentPromptByKey(componentIndexPromptKey)
	if err != nil {
		return "", fmt.Errorf("未找到组件索引字典(prompt_key=%s): %w", componentIndexPromptKey, err)
	}
	if strings.TrimSpace(prompt.Content) == "" {
		return "", fmt.Errorf("组件索引字典内容为空")
	}
	return prompt.Content, nil
}

// cleanIndexOutput 清理AI输出中可能包含的markdown标记
func cleanIndexOutput(text string) string {
	// 去掉```包裹
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimPrefix(text, "```text")
	text = strings.TrimPrefix(text, "```\n")
	text = strings.TrimSuffix(text, "\n```")
	return strings.TrimSpace(text)
}

// safePreview 安全预览字符串前N个字符
func safePreview(s string, maxChars int) string {
	runes := []rune(s)
	if len(runes) <= maxChars {
		return s
	}
	return string(runes[:maxChars]) + "..."
}
