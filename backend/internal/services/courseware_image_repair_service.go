package services

// courseware_image_repair_service.go — 课件配图自动修复、失败事实读取与定向补配
//
// 仅治理两类已经确认且可安全判定的失败：
//  1. IAOCI规划协议校验失败：允许一次受控协议修复后重新校验；
//  2. 图片供应商明确返回InputTextSensitiveContentDetected：允许一次教学语义保持的安全重述后重试。
//
// 自动修复预算固定为1；浏览器不提交可信失败范围、原始IAOCI或生成提示词；成功图片槽位按稳定
// image_key与已绑定资产复用，不重复调用图片供应商或重复媒体计费。教师端只看到安全错误码与文案。

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"tedna/internal/ai"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

const (
	CoursewareImageRepairCodePlanInvalid   = "iaoci_plan_invalid"
	CoursewareImageRepairCodeContentReview = "image_content_review"

	coursewareImageRepairLastErrorContentReview = "image_content_review_rejected"
)

var (
	ErrCoursewareImageRepairIncomplete      = errors.New("仍有配图未能自动修复")
	ErrCoursewareImageContentReviewRejected = errors.New("图片模型内容审核未通过")

	errCoursewareImageIAOCIPlanRepairCallFailed = errors.New("图片IAOCI协议修复调用失败")
	coursewareImageRepairRunning                sync.Map
)

// IsCoursewareImageRepairActive 只暴露当前进程是否正在执行失败配图定向补配。
// 服务重启后旧运行会被数据库收敛为interrupted，因此该进程内事实不需要持久化。
func IsCoursewareImageRepairActive(coursewareID string) bool {
	coursewareID = strings.TrimSpace(coursewareID)
	if coursewareID == "" {
		return false
	}
	_, active := coursewareImageRepairRunning.Load(coursewareID)
	return active
}

// coursewareImageIAOCIPlanRepairableError 表示“原规划+一次协议修复”仍未通过严格校验。
// 只有该类型才允许教师端出现“修复图片规划并补配”，网络、配置和鉴权错误不得冒充协议错误。
type coursewareImageIAOCIPlanRepairableError struct {
	cause error
}

func (e *coursewareImageIAOCIPlanRepairableError) Error() string {
	if e == nil || e.cause == nil {
		return "图片IAOCI规划协议自动修复后仍未通过"
	}
	return "图片IAOCI规划协议自动修复后仍未通过: " + e.cause.Error()
}

func (e *coursewareImageIAOCIPlanRepairableError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

func newCoursewareImageIAOCIPlanRepairableError(
	cause error,
) error {
	return &coursewareImageIAOCIPlanRepairableError{
		cause: cause,
	}
}

func isCoursewareImageIAOCIPlanRepairableError(err error) bool {
	var target *coursewareImageIAOCIPlanRepairableError
	return errors.As(err, &target) && target != nil
}

type coursewareImageRepairModeContextKey struct{}

// WithCoursewareImageRepairMode 标记本次版本化自动装配只执行失败配图修复。
// 该标记仅由已鉴权Handler写入后台context；浏览器布尔字段本身不携带可信范围。
func WithCoursewareImageRepairMode(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, coursewareImageRepairModeContextKey{}, true)
}

func coursewareImageRepairModeFromContext(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	enabled, _ := ctx.Value(coursewareImageRepairModeContextKey{}).(bool)
	return enabled
}

// CoursewareImageRepairItem 是教师可见的单个可修复配图失败事实。
type CoursewareImageRepairItem struct {
	PageID        string `json:"page_id"`
	PageNumber    int    `json:"page_number"`
	PageTitle     string `json:"page_title"`
	PlaceholderID string `json:"placeholder_id"`
	ImageKey      string `json:"image_key,omitempty"`
	ErrorCode     string `json:"error_code"`
	Message       string `json:"message"`
}

// CoursewareImageRepairState 是当前课件可定向补配的安全事实。
type CoursewareImageRepairState struct {
	RetryableCount    int                         `json:"retryable_count"`
	AffectedPageCount int                         `json:"affected_page_count"`
	Items             []CoursewareImageRepairItem `json:"items"`
}

// ReadCoursewareImageRepairState 从当前页面HTML与图片索引读取可修复失败，不信任前端提交范围。
func ReadCoursewareImageRepairState(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
) (*CoursewareImageRepairState, error) {
	_, scopedActor, err := (&CoursewareService{}).LoadCoursewareForOwnerRuntime(ctx, coursewareID, actor)
	if err != nil {
		return nil, err
	}
	if scopedActor == nil {
		return nil, ErrCoursewareActorRequired
	}

	pages, err := repository.ListCoursewarePages(ctx, coursewareID)
	if err != nil {
		return nil, fmt.Errorf("读取课件页面配图失败事实失败: %w", err)
	}
	indexes, err := repository.ListCoursewareImageIndexesByCourseware(ctx, coursewareID)
	if err != nil {
		return nil, fmt.Errorf("读取课件图片索引失败事实失败: %w", err)
	}

	pageByID := make(map[string]*models.CoursewarePage, len(pages))
	for _, page := range pages {
		if page != nil {
			pageByID[page.ID] = page
		}
	}

	result := &CoursewareImageRepairState{Items: make([]CoursewareImageRepairItem, 0)}
	seen := make(map[string]bool)
	affectedPages := make(map[string]bool)

	for _, page := range pages {
		if page == nil {
			continue
		}
		openTags := cwImagePlaceholderDivOpenRe.FindAllString(page.HTMLContent, -1)
		for _, openTag := range openTags {
			if !cwImagePlaceholderClassRe.MatchString(openTag) {
				continue
			}
			stateMatch := cwImagePlaceholderStateAttrRe.FindStringSubmatch(openTag)
			if len(stateMatch) < 2 || strings.TrimSpace(stateMatch[1]) != "plan_failed" {
				continue
			}
			idMatch := cwImagePlaceholderIDAttrRe.FindStringSubmatch(openTag)
			if len(idMatch) < 2 {
				continue
			}

			placeholderID := strings.TrimSpace(idMatch[1])
			if placeholderID == "" {
				continue
			}
			key := page.ID + "\x00" + placeholderID
			if seen[key] {
				continue
			}
			seen[key] = true

			imageKey, keyErr := utils.BuildImageAOCIKey(page.ID, placeholderID)
			if keyErr != nil {
				imageKey = ""
			}
			result.Items = append(result.Items, CoursewareImageRepairItem{
				PageID:        page.ID,
				PageNumber:    page.PageNumber,
				PageTitle:     page.Title,
				PlaceholderID: placeholderID,
				ImageKey:      imageKey,
				ErrorCode:     CoursewareImageRepairCodePlanInvalid,
				Message:       "图片规划格式未通过，可智能修复后补配",
			})
			affectedPages[page.ID] = true
		}
	}

	for _, index := range indexes {
		if index == nil || index.PageID == nil || strings.TrimSpace(*index.PageID) == "" ||
			index.Status != models.CWImageIndexStatusFailed ||
			!isCoursewareImageContentReviewFailureText(index.LastError) {
			continue
		}
		page := pageByID[*index.PageID]
		if page == nil {
			continue
		}

		placeholderID := strings.TrimSpace(index.PlaceholderID)
		if placeholderID == "" {
			continue
		}
		key := page.ID + "\x00" + placeholderID
		if seen[key] {
			continue
		}
		seen[key] = true

		result.Items = append(result.Items, CoursewareImageRepairItem{
			PageID:        page.ID,
			PageNumber:    page.PageNumber,
			PageTitle:     page.Title,
			PlaceholderID: placeholderID,
			ImageKey:      index.ImageKey,
			ErrorCode:     CoursewareImageRepairCodeContentReview,
			Message:       "图片模型内容审核未通过，可安全调整描述后补配",
		})
		affectedPages[page.ID] = true
	}

	sort.Slice(result.Items, func(left int, right int) bool {
		if result.Items[left].PageNumber != result.Items[right].PageNumber {
			return result.Items[left].PageNumber < result.Items[right].PageNumber
		}
		return result.Items[left].PlaceholderID < result.Items[right].PlaceholderID
	})
	result.RetryableCount = len(result.Items)
	result.AffectedPageCount = len(affectedPages)
	return result, nil
}

func isCoursewareImageContentReviewFailureText(value string) bool {
	normalized := strings.TrimSpace(value)
	return normalized == coursewareImageRepairLastErrorContentReview ||
		strings.Contains(normalized, "InputTextSensitiveContentDetected")
}

// repairImageIAOCIPlanOnce 对一次IAOCI规划协议失败做窄范围修复并重新执行全部严格校验。
func (s *CoursewareAssetService) repairImageIAOCIPlanOnce(
	ctx context.Context,
	aiConfig *ai.EffectiveConfig,
	systemPrompt string,
	userInput string,
	rawOutput string,
	validationErr error,
	traceContext *ai.TraceContext,
	slots []cwImagePlaceholderSlot,
	expectedOrder map[string]int,
	expectedSlot map[string]cwImagePlaceholderSlot,
	historyKeySet map[string]bool,
) ([]*models.ImageAOCI, error) {
	if aiConfig == nil || validationErr == nil {
		return nil, fmt.Errorf("图片IAOCI自动修复参数不完整")
	}

	repairSystemPrompt := strings.TrimSpace(systemPrompt) +
		"\n\n【协议自动修复模式】你只修复上一版IAOCI违反机器协议的部分。" +
		"必须保持槽位数量、稳定image_key、教学事实、主体、场景和构图意图不变。" +
		"R关系若目标、继承掩码或关系格式不合法，可删除该条非必要R关系，但不得编造不存在的image_key。" +
		"缺失F/L/A/C/S/E/N或R声明时补齐协议结构。只输出完整IAOCI块，不解释。"

	repairInput := strings.TrimSpace(userInput) +
		"\n\n## 上一次IAOCI输出\n```text\n" + cwLimitImageRepairRunes(rawOutput, 12000) +
		"\n```\n\n## 严格校验错误\n" + cwLimitImageRepairRunes(validationErr.Error(), 1200) +
		"\n\n请只修复上述协议错误并返回全部槽位的完整IAOCI。"

	repaired, callErr := ai.CallAI(aiConfig, repairSystemPrompt, repairInput, traceContext)
	if callErr != nil {
		return nil, fmt.Errorf(
			"%w: %v",
			errCoursewareImageIAOCIPlanRepairCallFailed,
			callErr,
		)
	}
	if repaired == nil || strings.TrimSpace(repaired.Content) == "" {
		return nil, fmt.Errorf("图片IAOCI自动修复未返回内容")
	}

	parsed, parseErr := cwParseImageAOCIBlocks(repaired.Content)
	if parseErr != nil {
		return nil, fmt.Errorf("图片IAOCI自动修复后仍无法解析: %w", parseErr)
	}
	if validateErr := cwValidatePlannedImageAOCIs(
		parsed, slots, expectedOrder, expectedSlot, historyKeySet,
	); validateErr != nil {
		return nil, fmt.Errorf("图片IAOCI自动修复后仍未通过校验: %w", validateErr)
	}

	cwAssetLog.Info(
		"图片IAOCI协议自动修复成功",
		"model", repaired.ModelUsed,
		"tokens", repaired.TokensUsed,
		"slot_count", len(slots),
	)
	return parsed, nil
}

// RepairImagePromptAfterContentReview 对明确的输入文本审核拒绝做一次教学语义保持的安全重述。
func (s *CoursewareAssetService) RepairImagePromptAfterContentReview(
	ctx context.Context,
	coursewareID string,
	imageKey string,
	actor *CoursewareActorContext,
) error {
	_, scopedActor, err := (&CoursewareService{}).LoadCoursewareForOwnerRuntime(ctx, coursewareID, actor)
	if err != nil {
		return err
	}
	index, err := repository.GetCoursewareImageIndexByKey(ctx, coursewareID, imageKey)
	if err != nil {
		return err
	}

	originalPrompt := strings.TrimSpace(index.GenerationPrompt)
	if originalPrompt == "" {
		return fmt.Errorf("图片内容审核修复缺少原生成提示词")
	}
	aiConfig, err := ai.GetEffectiveConfig(
		s.cfg.GetAESKey(),
		sceneCWMediaPrompt,
		s.cfg.AIAPIBaseURL,
		s.cfg.AIAPIKey,
		s.cfg.AIDefaultModel,
	)
	if err != nil {
		return fmt.Errorf("获取图片安全重述模型失败: %w", err)
	}

	userID := scopedActor.UserID
	schoolID, _ := repository.GetSchoolIDByUserID(ctx, userID)
	traceContext := &ai.TraceContext{
		SceneCode: sceneCWMediaPrompt,
		UserID:    &userID,
		SchoolID:  schoolIDPtr(schoolID),
	}

	const safetySystemPrompt = "你是K12教学配图提示词安全重述器。" +
		"任务不是规避平台审核，而是在不改变教学事实的前提下，把容易产生歧义的非必要描述改写为明确、" +
		"中性、课堂化的视觉表达。必须保留知识点、必要主体、数量关系、关键动作、教学场景用途、艺术风格与画幅要求；" +
		"不得添加成人、色情、暴力、危险、仇恨、政治或其他敏感内容。若涉及儿童、身体部位、健康或动作，" +
		"使用正常着装、非伤害、教学示意的中性表述。只输出一份可直接用于文生图的完整提示词，不解释、不用代码块。"

	userInput := "## 本图教学焦点\n" + strings.TrimSpace(index.FocusText) +
		"\n\n## 原生成提示词\n" + cwLimitImageRepairRunes(originalPrompt, 6000)

	rewritten, callErr := ai.CallAI(aiConfig, safetySystemPrompt, userInput, traceContext)
	if callErr != nil {
		return fmt.Errorf("图片提示词安全重述失败: %w", callErr)
	}
	if rewritten == nil {
		return fmt.Errorf("图片提示词安全重述未返回内容")
	}

	repairedPrompt := cleanCoursewareImageSafetyRewrite(rewritten.Content)
	if repairedPrompt == "" {
		return fmt.Errorf("图片提示词安全重述结果为空")
	}
	if len([]rune(repairedPrompt)) > 7000 {
		return fmt.Errorf("图片提示词安全重述结果过长")
	}

	newVersion, updateErr := repository.UpdateCoursewareImageIndexGenerationPromptForRetry(
		ctx, index.ID, index.Version, repairedPrompt,
	)
	if updateErr != nil {
		return updateErr
	}

	cwAssetLog.Info(
		"图片内容审核失败已完成一次安全重述",
		"courseware_id", coursewareID,
		"image_key", imageKey,
		"image_index_id", index.ID,
		"old_version", index.Version,
		"new_version", newVersion,
		"model", rewritten.ModelUsed,
		"tokens", rewritten.TokensUsed,
	)
	return nil
}

// generateImageFromIAOCIWithAutoRepair 执行生图，并把自动修复预算严格限制为一次。
//
// 普通装配：原始生图若明确触发内容审核，安全重述一次后再重试一次。
// 人工智能补配：若数据库已经记录上次内容审核失败，先按该错误安全重述一次，再只生图一次；
// 本次若再次被审核拒绝，不再叠加第二次自动重述，避免一次点击形成隐式循环。
func (s *CoursewareAutoAssemblyService) generateImageFromIAOCIWithAutoRepair(
	ctx context.Context,
	pageContext *cwAssemblyPageContext,
	page *models.CoursewarePage,
	plan CoursewareImageAOCIPlanItem,
	relationReference string,
) (*GenerateImageServiceResponse, error) {
	request := &GenerateImageIAOCIRequest{
		CoursewareID:        pageContext.coursewareID,
		PageNumber:          page.PageNumber,
		PlaceholderID:       plan.PlaceholderID,
		ImageKey:            plan.ImageKey,
		Prompt:              plan.Prompt,
		Size:                plan.Size,
		RelationRefImageURL: relationReference,
		Actor:               pageContext.actor,
	}

	manualRepairApplied := false
	if coursewareImageRepairModeFromContext(ctx) {
		index, indexErr := repository.GetCoursewareImageIndexByKey(
			ctx,
			pageContext.coursewareID,
			plan.ImageKey,
		)
		if indexErr == nil &&
			index != nil &&
			index.Status == models.CWImageIndexStatusFailed &&
			isCoursewareImageContentReviewFailureText(index.LastError) {
			GlobalCWSSEHub.Broadcast(pageContext.coursewareID, CWSSEEvent{
				EventType: "assembly_page_image",
				Data: map[string]interface{}{
					"page_number":    page.PageNumber,
					"stage":          "image_slot_repair",
					"placeholder_id": plan.PlaceholderID,
					"image_key":      plan.ImageKey,
					"message": fmt.Sprintf(
						"第 %d 页：根据上次内容审核失败原因安全调整图片描述并补配一次…",
						page.PageNumber,
					),
				},
			})

			repairErr := s.assetService.RepairImagePromptAfterContentReview(
				ctx,
				pageContext.coursewareID,
				plan.ImageKey,
				pageContext.actor,
			)
			if repairErr != nil {
				s.markImageAOCIIndexFailed(
					ctx,
					pageContext.coursewareID,
					plan.ImageKey,
					coursewareImageRepairLastErrorContentReview,
				)
				return nil, fmt.Errorf(
					"%w: 人工补配安全重述失败",
					ErrCoursewareImageContentReviewRejected,
				)
			}
			manualRepairApplied = true
		}
	}

	response, err := s.assetService.GenerateImageFromIAOCI(ctx, request)
	if err == nil && response != nil {
		return response, nil
	}
	if !ai.IsImageInputTextSensitiveError(err) {
		return response, err
	}

	// 人工补配已经先执行过一次安全重述，本次再次被拒绝即正式失败，不允许同一点击继续循环。
	if manualRepairApplied {
		s.markImageAOCIIndexFailed(
			ctx,
			pageContext.coursewareID,
			plan.ImageKey,
			coursewareImageRepairLastErrorContentReview,
		)
		return nil, ErrCoursewareImageContentReviewRejected
	}

	GlobalCWSSEHub.Broadcast(pageContext.coursewareID, CWSSEEvent{
		EventType: "assembly_page_image",
		Data: map[string]interface{}{
			"page_number":    page.PageNumber,
			"stage":          "image_slot_repair",
			"placeholder_id": plan.PlaceholderID,
			"image_key":      plan.ImageKey,
			"message": fmt.Sprintf(
				"第 %d 页：图片描述触发内容审核，正在自动安全调整并重试一次…",
				page.PageNumber,
			),
		},
	})

	repairErr := s.assetService.RepairImagePromptAfterContentReview(
		ctx,
		pageContext.coursewareID,
		plan.ImageKey,
		pageContext.actor,
	)
	if repairErr != nil {
		s.markImageAOCIIndexFailed(
			ctx,
			pageContext.coursewareID,
			plan.ImageKey,
			coursewareImageRepairLastErrorContentReview,
		)
		return nil, fmt.Errorf(
			"%w: 自动安全重述失败",
			ErrCoursewareImageContentReviewRejected,
		)
	}

	response, retryErr := s.assetService.GenerateImageFromIAOCI(ctx, request)
	if retryErr == nil && response != nil {
		cwAssemblyLog.Info(
			"图片内容审核自动修复后重试成功",
			"courseware_id", pageContext.coursewareID,
			"page", page.PageNumber,
			"placeholder_id", plan.PlaceholderID,
			"image_key", plan.ImageKey,
		)
		return response, nil
	}
	if ai.IsImageInputTextSensitiveError(retryErr) {
		s.markImageAOCIIndexFailed(
			ctx,
			pageContext.coursewareID,
			plan.ImageKey,
			coursewareImageRepairLastErrorContentReview,
		)
		return nil, ErrCoursewareImageContentReviewRejected
	}
	return response, retryErr
}

// RepairFailedCoursewareImages 运行一次受数据库版本保护的“只补失败配图”主体。
//
// 与普通AutoAssemble共用进程内互斥和取消信号，但不会进入HTML生成、视频生成或成功页重跑。
func (s *CoursewareAutoAssemblyService) RepairFailedCoursewareImages(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	skipVideo bool,
) error {
	if s == nil || s.assetService == nil || s.ossService == nil {
		return fmt.Errorf("配图智能修复服务未完整初始化")
	}

	courseware, scopedActor, err :=
		(&CoursewareService{}).LoadCoursewareForOwnerRuntime(
			ctx,
			coursewareID,
			actor,
		)
	if err != nil {
		return err
	}
	if courseware == nil || scopedActor == nil {
		return ErrCoursewareEducationDomainInvalid
	}

	if _, busy := cwAssemblyRunning.LoadOrStore(
		coursewareID,
		struct{}{},
	); busy {
		return fmt.Errorf("课件正在装配中: %s", coursewareID)
	}
	defer cwAssemblyRunning.Delete(coursewareID)

	coursewareImageRepairRunning.Store(coursewareID, struct{}{})
	defer coursewareImageRepairRunning.Delete(coursewareID)

	cancelSignal := newCWAssemblyCancelSignal()
	cwAssemblyCancelMap.Store(coursewareID, cancelSignal)
	defer cwAssemblyCancelMap.Delete(coursewareID)

	repairCtx := WithCoursewareImageRepairMode(ctx)
	return s.repairFailedCoursewareImages(
		repairCtx,
		courseware,
		scopedActor,
		skipVideo,
		cancelSignal.channel,
	)
}

// repairFailedCoursewareImages 只修复当前数据库中仍存在的可修复配图页。
// 同页可能有多个失败槽位，但只执行一次页面IAOCI规划；已generated且绑定资产的槽位由规划器复用。
func (s *CoursewareAutoAssemblyService) repairFailedCoursewareImages(
	ctx context.Context,
	courseware *models.Courseware,
	actor *CoursewareActorContext,
	skipVideo bool,
	cancelChannel <-chan struct{},
) error {
	if courseware == nil {
		return ErrCoursewareEducationDomainInvalid
	}
	repairState, err := ReadCoursewareImageRepairState(ctx, courseware.ID, actor)
	if err != nil {
		return err
	}
	pages, err := repository.ListCoursewarePages(ctx, courseware.ID)
	if err != nil {
		return fmt.Errorf("读取待补配页面失败: %w", err)
	}

	affected := make(map[string]bool)
	for _, item := range repairState.Items {
		affected[item.PageID] = true
	}

	GlobalCWSSEHub.Broadcast(courseware.ID, CWSSEEvent{
		EventType: "assembly_start",
		Data: map[string]interface{}{
			"courseware_id":        courseware.ID,
			"total_pages":          len(pages),
			"skip_video":           skipVideo,
			"repair_failed_images": true,
			"message":              "正在智能修复失败配图；已成功页面和图片不会重新生成。",
		},
	})

	if len(affected) == 0 {
		GlobalCWSSEHub.Broadcast(courseware.ID, CWSSEEvent{
			EventType: "assembly_done",
			Data: map[string]interface{}{
				"courseware_id": courseware.ID,
				"skip_video":    skipVideo,
				"html_success":  0,
				"html_fail":     0,
				"image_success": 0,
				"image_fail":    0,
				"image_skip":    0,
				"video_success": 0,
				"video_skip":    len(pages),
				"total_pages":   len(pages),
				"errors":        []string{},
				"message":       "当前没有需要智能补配的失败图片。",
			},
		})
		return nil
	}

	pageContext := &cwAssemblyPageContext{
		coursewareID: courseware.ID,
		actor:        actor,
		cw:           courseware,
		totalPages:   len(pages),
		skipVideo:    true,
	}
	successPages := 0
	failedPages := 0
	errorsSafe := make([]string, 0)

	for _, page := range pages {
		if page == nil || !affected[page.ID] {
			continue
		}
		if isCWAssemblyCancelled(cancelChannel) {
			break
		}

		result := cwAssemblyPageResult{
			pageNum: page.PageNumber,
			pageID:  page.ID,
			title:   page.Title,
			htmlOK:  true,
		}
		s.assemblePageImagesIAOCI(ctx, pageContext, page, &result)

		if result.imageOK || result.imageSkipped {
			successPages++
		} else {
			failedPages++
			if strings.TrimSpace(result.errMsg) != "" {
				errorsSafe = append(errorsSafe, fmt.Sprintf("第%d页仍有配图失败", page.PageNumber))
			}
		}

		GlobalCWSSEHub.Broadcast(courseware.ID, CWSSEEvent{
			EventType: "assembly_page_done",
			Data: map[string]interface{}{
				"page_number":   page.PageNumber,
				"page_id":       page.ID,
				"title":         page.Title,
				"html_ok":       true,
				"image_ok":      result.imageOK,
				"image_skipped": result.imageSkipped,
				"video_ok":      false,
				"video_skipped": true,
				"error":         result.errMsg,
			},
		})
	}

	remaining, stateErr := ReadCoursewareImageRepairState(ctx, courseware.ID, actor)
	if stateErr != nil {
		return stateErr
	}
	remainingCount := remaining.RetryableCount
	doneMessage := fmt.Sprintf(
		"智能补配完成：成功处理%d页，仍有%d处可修复配图失败。",
		successPages,
		remainingCount,
	)

	GlobalCWSSEHub.Broadcast(courseware.ID, CWSSEEvent{
		EventType: "assembly_done",
		Data: map[string]interface{}{
			"courseware_id":        courseware.ID,
			"skip_video":           skipVideo,
			"html_success":         0,
			"html_fail":            0,
			"image_success":        successPages,
			"image_fail":           failedPages,
			"image_skip":           0,
			"video_success":        0,
			"video_skip":           len(pages),
			"total_pages":          len(pages),
			"errors":               errorsSafe,
			"message":              doneMessage,
			"repair_failed_images": true,
		},
	})
	if remainingCount > 0 {
		return fmt.Errorf(
			"%w: 仍有%d处可再次人工补配的失败",
			ErrCoursewareImageRepairIncomplete,
			remainingCount,
		)
	}
	if failedPages > 0 {
		return fmt.Errorf(
			"智能补配仍有%d页失败，当前错误不属于可再次自动修复类型",
			failedPages,
		)
	}
	return nil
}

func cleanCoursewareImageSafetyRewrite(value string) string {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "```") {
		return value
	}

	lines := strings.Split(value, "\n")
	if len(lines) >= 2 {
		lines = lines[1:]
	}
	if len(lines) > 0 && strings.HasPrefix(strings.TrimSpace(lines[len(lines)-1]), "```") {
		lines = lines[:len(lines)-1]
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func cwLimitImageRepairRunes(value string, limit int) string {
	value = strings.TrimSpace(value)
	if limit <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}
