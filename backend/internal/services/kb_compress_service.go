package services

// kb_compress_service.go — 知识库课标压缩服务（业务编排：上传→抽取→逐item压缩+仲裁）
//
// 本迭代只做课标 pipeline（课本多图聚合压缩推迟到迭代二，届时复用底层引擎两零件）：
//   CreateJob          落 job + 暂存原始输入（文本/图片dataURI），此时不建 item
//   RunCompressAsync   异步两步：
//     第1步【抽取】     调 kb_extract(opus) 通读整份输入 → 识别出 N 个知识点
//                      → 据此创建 N 条 kb_compress_items（seq + source_excerpt 存该知识点原文）
//     第2步【逐item】   对每个 item 调引擎 CompressOneItem(多轮) + ArbitrateConsistency(仲裁)
//                      → 高置信 auto_passed、低置信 need_review，写回 item
//   全程通过 GlobalKBSSEHub 推进度。
//
// 抽取/压缩/仲裁默认都走 opus（场景码 kb_extract/kb_compress/kb_arbitrate，未实配回退全局默认）。
// 抽取提示词内联（抽取逻辑通用、无需入库版本管理，与 kbArbitrateSystemPrompt 风格一致）；
// 压缩提示词从 prompts 表取 prompt_curriculum_index（已入库，可后台管理）。

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"tedna/internal/ai"
	"tedna/internal/config"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// SceneKBExtract 知识点抽取场景码（默认 opus，未实配回退全局默认）
const SceneKBExtract = "kb_extract"

// KBMaxItemsPerJob 单个任务抽取知识点数上限（防 AI 异常返回海量条目导致 opus 调用失控烧钱）
// 超出则截断并告警；课标一份材料正常几十个知识点，50 是宽松上限。
const KBMaxItemsPerJob = 50

// ==================== 服务定义 ====================

// KBCompressService 知识库课标压缩服务
type KBCompressService struct {
	cfg    *config.Config
	engine *KBCompressEngine
}

// NewKBCompressService 创建压缩服务（内部据 cfg 构造压缩引擎）
func NewKBCompressService(cfg *config.Config) *KBCompressService {
	engine := NewKBCompressEngine(
		cfg.GetAESKey(), cfg.AIAPIBaseURL, cfg.AIAPIKey, cfg.AIDefaultModel,
	)
	return &KBCompressService{cfg: cfg, engine: engine}
}

// ==================== 抽取阶段的输入暂存（进程内，不落盘）====================

// kbJobInputStore 暂存 job 的原始输入（文本/图片dataURI），供异步抽取阶段读取。
// 课标图片是一次性原料（抽取出知识点后真相在 items，原图无保留价值），不落盘，
// 用进程内 map 暂存，抽取完成即删除。key=jobID。
// 注：单实例部署（AOCI 述当前单 ECS），进程内暂存可行；多实例需改 Redis，迭代二评估。
var kbJobInputStore = struct {
	mu   sync.Mutex
	data map[string]*KBJobInput
}{data: make(map[string]*KBJobInput)}

// KBJobInput 一个任务的原始输入
type KBJobInput struct {
	TextContent   string
	ImageDataURIs []string
	Rounds        int
	UserID        *string
	SchoolID      *string
}

func kbStoreInput(jobID string, in *KBJobInput) {
	kbJobInputStore.mu.Lock()
	defer kbJobInputStore.mu.Unlock()
	kbJobInputStore.data[jobID] = in
}

func kbTakeInput(jobID string) *KBJobInput {
	kbJobInputStore.mu.Lock()
	defer kbJobInputStore.mu.Unlock()
	in := kbJobInputStore.data[jobID]
	delete(kbJobInputStore.data, jobID)
	return in
}

// ==================== CreateJob 创建压缩任务 ====================

// CreateJob 创建一个课标压缩任务（落 job + 暂存原始输入，不建 item）
// 返回新任务 id。实际压缩由调用方随后调 RunCompressAsync 异步触发。
func (s *KBCompressService) CreateJob(ctx context.Context, req *models.KBCreateJobRequest, userID string, schoolID string) (string, error) {
	// 校验种类（本迭代只接受 curriculum）
	if req.Kind == "" {
		req.Kind = models.KBKindCurriculum
	}
	if req.Kind != models.KBKindCurriculum {
		return "", fmt.Errorf("本迭代仅支持课标(curriculum)压缩，教材(textbook)待迭代二")
	}
	if !models.IsValidKBKind(req.Kind) {
		return "", fmt.Errorf("无效的任务种类: %s", req.Kind)
	}
	if strings.TrimSpace(req.BatchTag) == "" {
		return "", fmt.Errorf("批次标识 batch_tag 不能为空")
	}
	// 输入校验：文本与图片至少有一个
	if strings.TrimSpace(req.TextContent) == "" && len(req.ImageDataURIs) == 0 {
		return "", fmt.Errorf("请提供课标文本或图片")
	}

	// 压缩模式：有图走 precise（多模态），纯文本走 fast
	mode := models.KBCompressModeFast
	if len(req.ImageDataURIs) > 0 {
		mode = models.KBCompressModePrecise
	}

	rounds := req.Rounds
	if rounds <= 0 {
		rounds = models.KBDefaultRounds
	}

	var createdBy *string
	if userID != "" {
		createdBy = &userID
	}

	job := &models.KBCompressJob{
		Kind:         req.Kind,
		BatchTag:     req.BatchTag,
		CompressMode: mode,
		Subject:      req.Subject,
		GradeNum:     req.GradeNum,
		Status:       models.KBJobStatusUploaded,
		TotalItems:   0,
		CreatedBy:    createdBy,
	}
	jobID, err := repository.CreateKBJob(ctx, job)
	if err != nil {
		return "", fmt.Errorf("创建压缩任务失败: %w", err)
	}

	// 暂存原始输入供异步抽取阶段读取（不落盘）
	var schoolPtr *string
	if schoolID != "" {
		schoolPtr = &schoolID
	}
	kbStoreInput(jobID, &KBJobInput{
		TextContent:   req.TextContent,
		ImageDataURIs: req.ImageDataURIs,
		Rounds:        rounds,
		UserID:        createdBy,
		SchoolID:      schoolPtr,
	})

	return jobID, nil
}

// ==================== RunCompressAsync 异步抽取+压缩 ====================

// RunCompressAsync 异步执行：抽取知识点 → 创建 items → 逐 item 压缩+仲裁。
// 由 handler 以 go s.RunCompressAsync(jobID) 触发；用独立 context.Background()，
// 不随请求 ctx 取消而中断（后台任务需跑完），全程 defer recover 防 panic 影响进程。
func (s *KBCompressService) RunCompressAsync(jobID string) {
	ctx := context.Background()
	defer func() {
		if r := recover(); r != nil {
			kbSseLog.Warn("KB压缩后台任务 panic 已恢复", "job_id", jobID, "recover", r)
			_ = repository.UpdateKBJobStatus(ctx, jobID, models.KBJobStatusFailed)
			GlobalKBSSEHub.Broadcast(jobID, KBSSEEvent{
				EventType: KBSSEError,
				Data:      map[string]interface{}{"message": fmt.Sprintf("压缩任务异常: %v", r)},
			})
		}
	}()

	input := kbTakeInput(jobID)
	if input == nil {
		_ = repository.UpdateKBJobStatus(ctx, jobID, models.KBJobStatusFailed)
		GlobalKBSSEHub.Broadcast(jobID, KBSSEEvent{
			EventType: KBSSEError,
			Data:      map[string]interface{}{"message": "任务输入已丢失，请重新上传"},
		})
		return
	}

	// ---- 第1步：抽取知识点 ----
	_ = repository.UpdateKBJobStatus(ctx, jobID, models.KBJobStatusParsing)
	GlobalKBSSEHub.Broadcast(jobID, KBSSEEvent{
		EventType: KBSSEExtractStart,
		Data:      map[string]interface{}{"message": "正在通读材料，识别知识点..."},
	})

	extracted, err := s.extractKnowledgePoints(ctx, input)
	if err != nil {
		_ = repository.UpdateKBJobStatus(ctx, jobID, models.KBJobStatusFailed)
		GlobalKBSSEHub.Broadcast(jobID, KBSSEEvent{
			EventType: KBSSEError,
			Data:      map[string]interface{}{"message": "抽取知识点失败: " + err.Error()},
		})
		return
	}
	if len(extracted) == 0 {
		_ = repository.UpdateKBJobStatus(ctx, jobID, models.KBJobStatusFailed)
		GlobalKBSSEHub.Broadcast(jobID, KBSSEEvent{
			EventType: KBSSEError,
			Data:      map[string]interface{}{"message": "未能从材料中识别出任何知识点"},
		})
		return
	}

	// 据抽取结果创建 items
	for i, kp := range extracted {
		item := &models.KBCompressItem{
			JobID:         jobID,
			Kind:          models.KBKindCurriculum,
			Seq:           i + 1,
			SourceExcerpt: kp,
			ReviewStatus:  models.KBReviewStatusPending,
		}
		if _, cerr := repository.CreateKBItem(ctx, item); cerr != nil {
			kbSseLog.Warn("创建压缩单元失败", "job_id", jobID, "seq", i+1, "err", cerr)
		}
	}
	_ = repository.UpdateKBJobProgress(ctx, jobID, 0, models.KBJobStatusCompressing)
	GlobalKBSSEHub.Broadcast(jobID, KBSSEEvent{
		EventType: KBSSEExtractDone,
		Data: map[string]interface{}{
			"total_items": len(extracted),
			"message":     fmt.Sprintf("识别出 %d 个知识点，开始逐个压缩...", len(extracted)),
		},
	})

	// ---- 第2步：加载压缩提示词 ----
	compressPromptObj, err := repository.GetCurrentPromptByKey("prompt_curriculum_index")
	if err != nil || compressPromptObj == nil {
		_ = repository.UpdateKBJobStatus(ctx, jobID, models.KBJobStatusFailed)
		GlobalKBSSEHub.Broadcast(jobID, KBSSEEvent{
			EventType: KBSSEError,
			Data:      map[string]interface{}{"message": "加载压缩提示词失败"},
		})
		return
	}
	compressPrompt := compressPromptObj.Content

	// ---- 第3步：逐 item 压缩+仲裁 ----
	items, err := repository.ListKBItemsByJob(ctx, jobID)
	if err != nil {
		_ = repository.UpdateKBJobStatus(ctx, jobID, models.KBJobStatusFailed)
		GlobalKBSSEHub.Broadcast(jobID, KBSSEEvent{
			EventType: KBSSEError,
			Data:      map[string]interface{}{"message": "读取待压缩单元失败: " + err.Error()},
		})
		return
	}

	done := 0
	for _, it := range items {
		GlobalKBSSEHub.Broadcast(jobID, KBSSEEvent{
			EventType: KBSSEItemStart,
			Data:      map[string]interface{}{"seq": it.Seq, "message": fmt.Sprintf("正在压缩第 %d 个知识点...", it.Seq)},
		})

		// 多轮压缩
		out, cerr := s.engine.CompressOneItem(KBCompressInput{
			Kind:           models.KBKindCurriculum,
			SourceText:     it.SourceExcerpt,
			CompressPrompt: compressPrompt,
			Rounds:         input.Rounds,
			UserID:         input.UserID,
			SchoolID:       input.SchoolID,
		})
		if cerr != nil {
			// 引擎级错误（如配置取不到）：标记该 item 失败，继续下一个
			_ = repository.UpdateKBItemCompressResult(ctx, it.ID,
				"[]", models.KBConfidenceLow, "", "", models.KBReviewStatusNeedReview,
				0, cerr.Error(), 0)
			done++
			_ = repository.UpdateKBJobProgress(ctx, jobID, done, models.KBJobStatusCompressing)
			continue
		}

		// 仲裁
		confidence, arb := s.engine.ArbitrateConsistency(out, input.UserID, input.SchoolID)

		// 决定 final_line 与审核状态
		finalLine := ""
		reviewStatus := models.KBReviewStatusNeedReview
		if confidence == models.KBConfidenceHigh && arb != nil {
			// 高置信：取仲裁选中的轮次作 final_line，自动通过候选
			finalLine = pickRoundLine(out.Rounds, arb.ChosenRound)
			reviewStatus = models.KBReviewStatusAutoPassed
		}

		draftJSON := models.DraftRoundsToJSON(out.Rounds)
		arbJSON := models.ArbitrationToJSON(arb)
		_ = repository.UpdateKBItemCompressResult(ctx, it.ID,
			draftJSON, confidence, arbJSON, finalLine, reviewStatus,
			out.DoneCount+out.FailCount, out.LastError, out.TokensTotal)

		done++
		_ = repository.UpdateKBJobProgress(ctx, jobID, done, models.KBJobStatusCompressing)
		GlobalKBSSEHub.Broadcast(jobID, KBSSEEvent{
			EventType: KBSSEItemDone,
			Data: map[string]interface{}{
				"seq":        it.Seq,
				"confidence": confidence,
				"done":       done,
				"total":      len(items),
			},
		})
	}

	// ---- 完成：转入待审核 ----
	_ = repository.UpdateKBJobProgress(ctx, jobID, done, models.KBJobStatusReviewing)
	GlobalKBSSEHub.Broadcast(jobID, KBSSEEvent{
		EventType: KBSSEJobDone,
		Data: map[string]interface{}{
			"total_items": len(items),
			"message":     fmt.Sprintf("压缩完成，共 %d 个知识点，请进入审核", len(items)),
		},
	})
}

// ==================== 抽取知识点 ====================

// extractKnowledgePoints 调 kb_extract(opus) 通读输入，识别出 N 个知识点的原文片段。
// 文本走 CallAI，图片走 CallAIMultimodal（逐图抽取后汇总）。
// 返回每个知识点的原文片段切片（供创建 items 的 source_excerpt）。
func (s *KBCompressService) extractKnowledgePoints(ctx context.Context, input *KBJobInput) ([]string, error) {
	aiCfg, err := ai.GetEffectiveConfig(
		s.cfg.GetAESKey(), SceneKBExtract,
		s.cfg.AIAPIBaseURL, s.cfg.AIAPIKey, s.cfg.AIDefaultModel,
	)
	if err != nil {
		return nil, fmt.Errorf("获取抽取AI配置失败: %w", err)
	}

	var allPoints []string

	// 文本输入：整段喂抽取 AI
	if strings.TrimSpace(input.TextContent) != "" {
		traceCtx := &ai.TraceContext{SceneCode: SceneKBExtract, UserID: input.UserID, SchoolID: input.SchoolID}
		result, callErr := ai.CallAI(aiCfg, kbExtractSystemPrompt, buildKBExtractUserPrompt(input.TextContent), traceCtx)
		if callErr != nil {
			return nil, fmt.Errorf("文本抽取调用失败: %w", callErr)
		}
		allPoints = append(allPoints, parseExtractedPoints(result.Content)...)
	}

	// 图片输入：逐图多模态抽取
	for idx, dataURI := range input.ImageDataURIs {
		if strings.TrimSpace(dataURI) == "" {
			continue
		}
		traceCtx := &ai.TraceContext{SceneCode: SceneKBExtract, UserID: input.UserID, SchoolID: input.SchoolID}
		result, callErr := ai.CallAIMultimodal(aiCfg, kbExtractSystemPrompt,
			buildKBExtractUserPrompt(""), dataURI, traceCtx)
		if callErr != nil {
			kbSseLog.Warn("图片抽取失败", "image_idx", idx, "err", callErr)
			continue
		}
		allPoints = append(allPoints, parseExtractedPoints(result.Content)...)
	}

	// 上限保护：防 AI 异常返回海量条目导致逐 item opus 压缩失控烧钱
	if len(allPoints) > KBMaxItemsPerJob {
		kbSseLog.Warn("抽取知识点数超上限，已截断",
			"extracted", len(allPoints), "limit", KBMaxItemsPerJob)
		allPoints = allPoints[:KBMaxItemsPerJob]
	}

	return allPoints, nil
}

// kbExtractSystemPrompt 知识点抽取系统提示词（内联，抽取逻辑通用无需入库）
const kbExtractSystemPrompt = `你是TE-DNA课标知识点抽取专家。任务：通读给定的课程标准材料（文本或图片），识别出其中包含的每一个独立知识点，并把每个知识点对应的原文范围完整摘出。

课标特点：知识点常密集分布，一页/一段往往论述多个知识点，也可能一个知识点跨多段论述。你要做的是按"独立内容要求/学业要求条目"为单位，切分出每个知识点的完整原文。

输出要求：严格输出一个JSON数组，每个元素是一个知识点的完整原文片段字符串，不要任何解释或Markdown围栏：
["知识点1的完整原文（含其内容要求与学业要求）", "知识点2的完整原文", ...]

注意：
- 每个片段必须是原文的忠实摘录，包含该知识点的内容要求与学业要求，供后续压缩成索引。
- 不要遗漏材料中的任何知识点，也不要把无关的页眉页脚/说明性文字当作知识点。
- 若材料中确无可识别的独立知识点，返回空数组 []。`

// buildKBExtractUserPrompt 构建抽取用户提示词（文本输入时附正文，图片输入时正文为空）
func buildKBExtractUserPrompt(textContent string) string {
	if strings.TrimSpace(textContent) == "" {
		return "请通读这张课标图片，识别并摘出其中包含的每一个知识点的完整原文，按系统提示词要求输出JSON数组。"
	}
	return strings.Join([]string{
		"【课标材料原文】",
		strings.TrimSpace(textContent),
		"",
		"请识别并摘出其中包含的每一个知识点的完整原文，按系统提示词要求输出JSON数组。",
	}, "\n")
}

// parseExtractedPoints 解析抽取 AI 输出的 JSON 数组（多级兜底，失败返回空切片）
func parseExtractedPoints(aiOutput string) []string {
	s := strings.TrimSpace(aiOutput)
	// 剥围栏
	if strings.HasPrefix(s, "```") {
		if idx := strings.Index(s, "\n"); idx >= 0 {
			s = s[idx+1:]
		}
		s = strings.TrimSuffix(strings.TrimSpace(s), "```")
		s = strings.TrimSpace(s)
	}
	// 第一级：直接解析 JSON 数组
	var points []string
	if err := json.Unmarshal([]byte(s), &points); err == nil {
		return cleanPointList(points)
	}
	// 第二级：截取首个 [ 到末个 ] 再解析
	start := strings.Index(s, "[")
	end := strings.LastIndex(s, "]")
	if start >= 0 && end > start {
		if err := json.Unmarshal([]byte(s[start:end+1]), &points); err == nil {
			return cleanPointList(points)
		}
	}
	return []string{}
}

// cleanPointList 清洗抽取结果：去空白项、去过短噪音项
func cleanPointList(raw []string) []string {
	var out []string
	for _, p := range raw {
		t := strings.TrimSpace(p)
		if len([]rune(t)) >= 5 { // 过短的片段视为噪音
			out = append(out, t)
		}
	}
	return out
}

// pickRoundLine 从多轮草稿中取指定轮次的 line；找不到取第一个成功轮
func pickRoundLine(rounds []models.KBDraftRound, chosenRound int) string {
	for _, r := range rounds {
		if r.Round == chosenRound && r.Error == "" && strings.TrimSpace(r.Line) != "" {
			return r.Line
		}
	}
	for _, r := range rounds {
		if r.Error == "" && strings.TrimSpace(r.Line) != "" {
			return r.Line
		}
	}
	return ""
}

// 让 time 包被使用（CreateJob/RunCompressAsync 当前未直接用 time，但保留以备扩展时间戳逻辑）
var _ = time.Now
