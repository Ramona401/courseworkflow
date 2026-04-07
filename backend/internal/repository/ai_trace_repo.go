package repository

// ai_trace_repo.go — AI调用追踪数据访问层
//
// 职责：
//   1. 异步写入AI调用trace记录（channel缓冲，不阻塞主路径）
//   2. 仪表盘聚合查询（按场景/模型/日期/状态多维聚合）
//   3. 最近错误记录查询
//
// 被引用：
//   ai/client.go — EnqueueTrace写入
//   handlers/ai_trace_handler.go — 仪表盘查询

import (
	"context"
	"fmt"
	"strings"
	"time"

	"tedna/internal/database"
	"tedna/internal/logger"
	"tedna/internal/models"
)

// 模块日志
var traceLog = logger.WithModule("ai_trace_repo")

// ==================== 异步写入通道 ====================

// traceChan 异步写入通道，缓冲500条，防止AI调用高峰时阻塞
var traceChan chan models.TraceRecord

// InitTraceWriter 初始化异步写入协程
// 必须在服务启动时调用（routes.Setup中）
func InitTraceWriter() {
	traceChan = make(chan models.TraceRecord, 500)
	go traceConsumer()
	traceLog.Info("AI调用追踪异步写入器已启动", "buffer_size", 500)
}

// EnqueueTrace 将trace记录放入异步写入通道（非阻塞）
// 如果通道已满，丢弃记录并记录警告日志（不影响AI调用主路径）
func EnqueueTrace(rec models.TraceRecord) {
	select {
	case traceChan <- rec:
		// 成功入队
	default:
		// 通道满时丢弃，记录警告
		traceLog.Warn("AI追踪通道已满，丢弃记录",
			"scene", rec.SceneCode,
			"model", rec.ModelUsed,
			"status", rec.Status,
		)
	}
}

// traceConsumer 消费协程：从通道中读取trace记录并写入数据库
func traceConsumer() {
	for rec := range traceChan {
		if err := insertTrace(rec); err != nil {
			traceLog.Error("写入AI追踪记录失败",
				"error", err,
				"scene", rec.SceneCode,
				"model", rec.ModelUsed,
			)
		}
	}
}

// insertTrace 执行单条trace记录的数据库INSERT
func insertTrace(rec models.TraceRecord) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 根据模型和token数估算成本
	cost := models.EstimateCost(rec.ModelUsed, rec.PromptTokens, rec.CompletionTokens)

	_, err := database.DB.Exec(ctx, `
		INSERT INTO ai_call_traces
			(scene_code, model_used, prompt_tokens, completion_tokens, total_tokens,
			 latency_ms, status, error_message, pipeline_id, lesson_plan_id, user_id,
			 estimated_cost_usd, output_length, is_stream)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		rec.SceneCode, rec.ModelUsed,
		rec.PromptTokens, rec.CompletionTokens, rec.TotalTokens,
		rec.LatencyMs, rec.Status, rec.ErrorMessage,
		rec.PipelineID, rec.LessonPlanID, rec.UserID,
		cost, rec.OutputLength, rec.IsStream,
	)
	return err
}

// ==================== 仪表盘聚合查询 ====================

// GetTraceDashboard 获取AI调用仪表盘全部数据（一次请求返回）
// 包含：概览数字 + 按场景聚合 + 按模型聚合 + 每日趋势 + 最近错误
func GetTraceDashboard(ctx context.Context, params models.TraceQueryParams) (*models.TraceDashboard, error) {
	dash := &models.TraceDashboard{}

	// 构建公共WHERE子句（时间范围+场景+模型筛选）
	whereClause, args := buildTraceWhere(params)

	// ---- 1. 概览数字 ----
	overviewSQL := fmt.Sprintf(`
		SELECT
			COUNT(*),
			COALESCE(SUM(total_tokens), 0),
			COALESCE(SUM(estimated_cost_usd), 0),
			COALESCE(AVG(latency_ms)::int, 0),
			CASE WHEN COUNT(*) > 0
				THEN ROUND(COUNT(*) FILTER (WHERE status != 'success')::numeric / COUNT(*)::numeric * 100, 2)
				ELSE 0
			END
		FROM ai_call_traces
		%s`, whereClause)

	err := database.DB.QueryRow(ctx, overviewSQL, args...).Scan(
		&dash.TotalCalls, &dash.TotalTokens, &dash.TotalCostUSD,
		&dash.AvgLatencyMs, &dash.ErrorRate,
	)
	if err != nil {
		return nil, fmt.Errorf("概览查询失败: %w", err)
	}

	// ---- 2. 按场景聚合 ----
	sceneSQL := fmt.Sprintf(`
		SELECT
			scene_code,
			COUNT(*) AS call_count,
			COUNT(*) FILTER (WHERE status = 'success') AS success_count,
			COUNT(*) FILTER (WHERE status != 'success') AS error_count,
			COALESCE(AVG(latency_ms)::int, 0) AS avg_latency,
			COALESCE(SUM(total_tokens), 0) AS total_tokens,
			COALESCE(SUM(prompt_tokens), 0) AS total_prompt,
			COALESCE(SUM(completion_tokens), 0) AS total_completion,
			COALESCE(SUM(estimated_cost_usd), 0) AS cost
		FROM ai_call_traces
		%s
		GROUP BY scene_code
		ORDER BY cost DESC`, whereClause)

	sceneRows, err := database.DB.Query(ctx, sceneSQL, args...)
	if err != nil {
		return nil, fmt.Errorf("场景聚合查询失败: %w", err)
	}
	defer sceneRows.Close()

	for sceneRows.Next() {
		var s models.TraceSceneStats
		if err := sceneRows.Scan(
			&s.SceneCode, &s.CallCount, &s.SuccessCount, &s.ErrorCount,
			&s.AvgLatencyMs, &s.TotalTokens, &s.TotalPromptTokens,
			&s.TotalCompletionTokens, &s.EstimatedCostUSD,
		); err != nil {
			return nil, fmt.Errorf("扫描场景聚合行失败: %w", err)
		}
		// 填充场景中文名（从models.SceneNameMap获取）
		if name, ok := models.SceneNameMap[s.SceneCode]; ok {
			s.SceneName = name
		} else {
			s.SceneName = s.SceneCode
		}
		dash.ByScene = append(dash.ByScene, s)
	}

	// ---- 3. 按模型聚合 ----
	modelSQL := fmt.Sprintf(`
		SELECT
			model_used,
			COUNT(*) AS call_count,
			COUNT(*) FILTER (WHERE status = 'success') AS success_count,
			COUNT(*) FILTER (WHERE status != 'success') AS error_count,
			COALESCE(AVG(latency_ms)::int, 0),
			COALESCE(SUM(total_tokens), 0),
			COALESCE(SUM(estimated_cost_usd), 0)
		FROM ai_call_traces
		%s
		GROUP BY model_used
		ORDER BY call_count DESC`, whereClause)

	modelRows, err := database.DB.Query(ctx, modelSQL, args...)
	if err != nil {
		return nil, fmt.Errorf("模型聚合查询失败: %w", err)
	}
	defer modelRows.Close()

	for modelRows.Next() {
		var m models.TraceModelStats
		if err := modelRows.Scan(
			&m.ModelUsed, &m.CallCount, &m.SuccessCount, &m.ErrorCount,
			&m.AvgLatencyMs, &m.TotalTokens, &m.EstimatedCostUSD,
		); err != nil {
			return nil, fmt.Errorf("扫描模型聚合行失败: %w", err)
		}
		dash.ByModel = append(dash.ByModel, m)
	}

	// ---- 4. 每日趋势（最近30天）----
	trendSQL := fmt.Sprintf(`
		SELECT
			TO_CHAR(created_at, 'YYYY-MM-DD') AS date,
			COUNT(*) AS call_count,
			COUNT(*) FILTER (WHERE status != 'success') AS error_count,
			COALESCE(SUM(total_tokens), 0) AS total_tokens,
			COALESCE(SUM(estimated_cost_usd), 0) AS cost,
			COALESCE(AVG(latency_ms)::int, 0) AS avg_latency
		FROM ai_call_traces
		%s
		GROUP BY TO_CHAR(created_at, 'YYYY-MM-DD')
		ORDER BY date DESC
		LIMIT 30`, whereClause)

	trendRows, err := database.DB.Query(ctx, trendSQL, args...)
	if err != nil {
		return nil, fmt.Errorf("趋势查询失败: %w", err)
	}
	defer trendRows.Close()

	for trendRows.Next() {
		var t models.TraceDailyTrend
		if err := trendRows.Scan(
			&t.Date, &t.CallCount, &t.ErrorCount,
			&t.TotalTokens, &t.EstimatedCostUSD, &t.AvgLatencyMs,
		); err != nil {
			return nil, fmt.Errorf("扫描趋势行失败: %w", err)
		}
		dash.DailyTrend = append(dash.DailyTrend, t)
	}

	// ---- 5. 最近错误记录（最多20条）----
	errorSQL := `
		SELECT id, scene_code, model_used, prompt_tokens, completion_tokens,
			   total_tokens, latency_ms, status, error_message,
			   pipeline_id, lesson_plan_id, user_id,
			   estimated_cost_usd, output_length, is_stream, created_at
		FROM ai_call_traces
		WHERE status != 'success'
		ORDER BY created_at DESC
		LIMIT 20`

	errorRows, err := database.DB.Query(ctx, errorSQL)
	if err != nil {
		return nil, fmt.Errorf("错误记录查询失败: %w", err)
	}
	defer errorRows.Close()

	for errorRows.Next() {
		var t models.AICallTrace
		if err := errorRows.Scan(
			&t.ID, &t.SceneCode, &t.ModelUsed,
			&t.PromptTokens, &t.CompletionTokens, &t.TotalTokens,
			&t.LatencyMs, &t.Status, &t.ErrorMessage,
			&t.PipelineID, &t.LessonPlanID, &t.UserID,
			&t.EstimatedCostUSD, &t.OutputLength, &t.IsStream, &t.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("扫描错误记录行失败: %w", err)
		}
		dash.RecentErrors = append(dash.RecentErrors, t)
	}

	// 确保切片不为nil（前端友好）
	if dash.ByScene == nil {
		dash.ByScene = []models.TraceSceneStats{}
	}
	if dash.ByModel == nil {
		dash.ByModel = []models.TraceModelStats{}
	}
	if dash.DailyTrend == nil {
		dash.DailyTrend = []models.TraceDailyTrend{}
	}
	if dash.RecentErrors == nil {
		dash.RecentErrors = []models.AICallTrace{}
	}

	return dash, nil
}

// ==================== WHERE子句构建 ====================

// buildTraceWhere 根据查询参数构建WHERE子句和参数列表
// 支持：时间范围 + 场景 + 模型 + 状态筛选
func buildTraceWhere(params models.TraceQueryParams) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	// 时间范围
	if params.DateFrom != "" {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d::date", argIdx))
		args = append(args, params.DateFrom)
		argIdx++
	}
	if params.DateTo != "" {
		// 加1天确保包含当天全部记录
		conditions = append(conditions, fmt.Sprintf("created_at < ($%d::date + interval '1 day')", argIdx))
		args = append(args, params.DateTo)
		argIdx++
	}

	// 场景筛选
	if params.SceneCode != "" {
		conditions = append(conditions, fmt.Sprintf("scene_code = $%d", argIdx))
		args = append(args, params.SceneCode)
		argIdx++
	}

	// 模型筛选
	if params.ModelUsed != "" {
		conditions = append(conditions, fmt.Sprintf("model_used = $%d", argIdx))
		args = append(args, params.ModelUsed)
		argIdx++
	}

	// 状态筛选
	if params.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, params.Status)
		argIdx++
	}

	if len(conditions) == 0 {
		return "", nil
	}

	return "WHERE " + strings.Join(conditions, " AND "), args
}
