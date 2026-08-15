package services

// courseware_auto_assembly_runtime.go — 自动装配运行控制、断点重试范围与共享辅助
//
// 本文件从主编排中拆出三个稳定职责：
//   - 进程内运行互斥与协作式取消信号；
//   - 根据上一轮数据库完整性事实计算“只补生成未成功页”的可信服务端范围；
//   - 主编排和取消入口共用的页面集合、SSE错误与完成文案辅助。
//
// 重试范围只信任数据库上一轮固化的CoursewareGenerationIntegrity，不接受浏览器提交page_id。
// missing页必须强制重生，因为其页面方案指纹已经变化；failed/cancelled页若当前数据库已经恢复为
// 有效生成态，则新运行只做重新对账，不覆盖教师后来已经修好的页面。

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"tedna/internal/models"
	"tedna/internal/repository"
)

var cwAssemblyRunning sync.Map

// cwAssemblyCancelSignal 是并发安全的一次性取消信号。
type cwAssemblyCancelSignal struct {
	channel chan struct{}
	once    sync.Once
}

func newCWAssemblyCancelSignal() *cwAssemblyCancelSignal {
	return &cwAssemblyCancelSignal{
		channel: make(chan struct{}),
	}
}

func (signal *cwAssemblyCancelSignal) Cancel() {
	if signal == nil {
		return
	}

	signal.once.Do(func() {
		close(signal.channel)
	})
}

// cwAssemblyCancelMap 保存当前进程中的装配停止信号。
var cwAssemblyCancelMap sync.Map

func isCWAssemblyCancelled(cancelChannel <-chan struct{}) bool {
	if cancelChannel == nil {
		return false
	}

	select {
	case <-cancelChannel:
		return true
	default:
		return false
	}
}

// coursewareAutoAssemblyRetryScope 是一次自动装配运行的服务端可信补生成范围。
//
// Enabled=false表示普通装配/普通断点续装；Enabled=true表示上一轮完整性未通过，
// 本轮只能处理PageIDs中的页面。ForcePageIDs是方案指纹变化的missing页，即使当前HTML非空也必须重生。
type coursewareAutoAssemblyRetryScope struct {
	Enabled      bool
	PageIDs      map[string]struct{}
	ForcePageIDs map[string]struct{}
}

type coursewareAutoAssemblyRetryScopeContextKey struct{}

func withCoursewareAutoAssemblyRetryScope(
	ctx context.Context,
	scope coursewareAutoAssemblyRetryScope,
) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	if !scope.Enabled {
		return ctx
	}

	pageIDs := make(map[string]struct{}, len(scope.PageIDs))
	for pageID := range scope.PageIDs {
		pageIDs[pageID] = struct{}{}
	}

	forcePageIDs := make(map[string]struct{}, len(scope.ForcePageIDs))
	for pageID := range scope.ForcePageIDs {
		forcePageIDs[pageID] = struct{}{}
	}

	return context.WithValue(
		ctx,
		coursewareAutoAssemblyRetryScopeContextKey{},
		coursewareAutoAssemblyRetryScope{
			Enabled:      true,
			PageIDs:      pageIDs,
			ForcePageIDs: forcePageIDs,
		},
	)
}

func coursewareAutoAssemblyRetryScopeFrom(
	ctx context.Context,
) (coursewareAutoAssemblyRetryScope, bool) {
	if ctx == nil {
		return coursewareAutoAssemblyRetryScope{}, false
	}

	scope, ok := ctx.Value(
		coursewareAutoAssemblyRetryScopeContextKey{},
	).(coursewareAutoAssemblyRetryScope)
	if !ok || !scope.Enabled {
		return coursewareAutoAssemblyRetryScope{}, false
	}

	if scope.PageIDs == nil {
		scope.PageIDs = map[string]struct{}{}
	}
	if scope.ForcePageIDs == nil {
		scope.ForcePageIDs = map[string]struct{}{}
	}

	return scope, true
}

func (scope coursewareAutoAssemblyRetryScope) contains(pageID string) bool {
	_, exists := scope.PageIDs[strings.TrimSpace(pageID)]
	return exists
}

func (scope coursewareAutoAssemblyRetryScope) force(pageID string) bool {
	_, exists := scope.ForcePageIDs[strings.TrimSpace(pageID)]
	return exists
}

// resolveCoursewareAutoAssemblyRetryScope 在创建新run之前读取上一轮终态完整性，
// 计算本轮真正需要补生成的稳定page_id集合。
func resolveCoursewareAutoAssemblyRetryScope(
	ctx context.Context,
	coursewareID string,
) (coursewareAutoAssemblyRetryScope, error) {
	state, err := repository.GetCoursewareAssemblyState(ctx, coursewareID)
	if err != nil {
		if errors.Is(err, repository.ErrCoursewareAssemblyNotFound) {
			return coursewareAutoAssemblyRetryScope{}, nil
		}
		return coursewareAutoAssemblyRetryScope{}, fmt.Errorf(
			"读取上一轮自动装配状态失败: %w",
			err,
		)
	}

	if state == nil ||
		state.Version <= 0 ||
		state.RunKind != models.CoursewareGenerationRunKindAssembly ||
		!models.IsCoursewareAssemblyFinalStatus(state.Status) {
		return coursewareAutoAssemblyRetryScope{}, nil
	}

	runKind, integrity, err := repository.ReadCoursewareGenerationIntegrity(
		ctx,
		coursewareID,
		state.Version,
	)
	if err != nil {
		return coursewareAutoAssemblyRetryScope{}, fmt.Errorf(
			"读取上一轮自动装配完整性失败: %w",
			err,
		)
	}
	if runKind != models.CoursewareGenerationRunKindAssembly ||
		integrity == nil ||
		integrity.Complete {
		return coursewareAutoAssemblyRetryScope{}, nil
	}

	scope := coursewareAutoAssemblyRetryScope{
		Enabled:      true,
		PageIDs:      map[string]struct{}{},
		ForcePageIDs: map[string]struct{}{},
	}

	addRetryRef := func(
		ref models.CoursewareGenerationPageRef,
		force bool,
	) {
		pageID := strings.TrimSpace(ref.PageID)
		if pageID == "" {
			return
		}

		scope.PageIDs[pageID] = struct{}{}
		if force {
			scope.ForcePageIDs[pageID] = struct{}{}
		}
	}

	for _, ref := range integrity.FailedPages {
		addRetryRef(ref, false)
	}
	for _, ref := range integrity.CancelledPages {
		addRetryRef(ref, false)
	}
	for _, ref := range integrity.MissingPages {
		addRetryRef(ref, true)
	}

	currentPages, err := repository.ListCoursewarePages(ctx, coursewareID)
	if err != nil {
		return coursewareAutoAssemblyRetryScope{}, fmt.Errorf(
			"读取当前页面用于补生成范围确认失败: %w",
			err,
		)
	}

	currentByID := make(map[string]*models.CoursewarePage, len(currentPages))
	for _, page := range currentPages {
		if page == nil || strings.TrimSpace(page.ID) == "" {
			continue
		}
		currentByID[page.ID] = page
	}

	for pageID := range scope.PageIDs {
		page, exists := currentByID[pageID]
		if !exists {
			return coursewareAutoAssemblyRetryScope{}, fmt.Errorf(
				"上一轮待补页面已不在当前方案中(page_id=%s)，请先重新确认页面方案后再装配",
				pageID,
			)
		}

		// missing表示方案指纹发生过变化，必须基于新方案重生；其他失败页若教师已经手工修复为
		// 有效生成态，则本轮只需重新冻结并对账，不得再次覆盖。
		if !scope.force(pageID) &&
			isCoursewareAutoAssemblyGenerationSuccessful(page) {
			delete(scope.PageIDs, pageID)
		}
	}

	return scope, nil
}

// isCoursewareAutoAssemblyGenerationSuccessful 与R-04完整性仓储的成功口径保持一致。
// 如仓储成功状态枚举变化，本函数必须同步更新，避免补生成范围与终态对账产生分叉。
func isCoursewareAutoAssemblyGenerationSuccessful(
	page *models.CoursewarePage,
) bool {
	if page == nil || strings.TrimSpace(page.HTMLContent) == "" {
		return false
	}

	switch strings.TrimSpace(page.Status) {
	case models.CWPageStatusGenerated,
		models.CWPageStatusMediaFilling,
		models.CWPageStatusConfirmed:
		return true
	default:
		return false
	}
}

// selectCoursewareAutoAssemblyWorkPages 选择HTML流水线和已有HTML媒体流水线的页面。
//
// 普通模式沿用历史行为：HTML为空的页生成HTML，其余已有HTML页进入媒体恢复。
// 完整性补生成模式严格只处理scope.PageIDs：目标页统一重新生成HTML，非目标页不执行呈现保护、
// IAOCI规划、生图或媒体计费，从根上避免“只补一页却整课重新配图”。
func selectCoursewareAutoAssemblyWorkPages(
	pages []*models.CoursewarePage,
	scope coursewareAutoAssemblyRetryScope,
	retryMode bool,
) (
	[]*models.CoursewarePage,
	[]*models.CoursewarePage,
	error,
) {
	if retryMode {
		remainingPages := make(
			[]*models.CoursewarePage,
			0,
			len(scope.PageIDs),
		)
		found := make(map[string]struct{}, len(scope.PageIDs))

		for _, page := range pages {
			if page == nil || !scope.contains(page.ID) {
				continue
			}

			found[page.ID] = struct{}{}
			remainingPages = append(remainingPages, page)
		}

		for pageID := range scope.PageIDs {
			if _, exists := found[pageID]; !exists {
				return nil, nil, fmt.Errorf(
					"补生成目标页在本轮页面快照中不存在(page_id=%s)",
					pageID,
				)
			}
		}

		return remainingPages, []*models.CoursewarePage{}, nil
	}

	remainingPages := make(
		[]*models.CoursewarePage,
		0,
		len(pages),
	)
	for _, page := range pages {
		if page == nil || strings.TrimSpace(page.HTMLContent) != "" {
			continue
		}
		remainingPages = append(remainingPages, page)
	}

	return remainingPages,
		collectAlreadyHtmlPagesForImage(
			pages,
			remainingPages,
		),
		nil
}

// collectAlreadyHtmlPagesForImage 收集普通断点续装中已有HTML、可继续媒体处理的页面。
func collectAlreadyHtmlPagesForImage(
	allPages []*models.CoursewarePage,
	remainingPages []*models.CoursewarePage,
) []*models.CoursewarePage {
	remainingSet := make(map[string]bool, len(remainingPages))
	for _, page := range remainingPages {
		if page != nil {
			remainingSet[page.ID] = true
		}
	}

	result := make([]*models.CoursewarePage, 0)
	for _, page := range allPages {
		if page == nil {
			continue
		}
		if strings.TrimSpace(page.HTMLContent) != "" &&
			!remainingSet[page.ID] {
			result = append(result, page)
		}
	}

	return result
}

// CancelAutoAssembly 停止继续派发全自动装配的新页面和新图片任务。
//
// 已经成功落库的数据不回滚；新进程再次执行AutoAssemble时从数据库断点续装。
func (s *CoursewareAutoAssemblyService) CancelAutoAssembly(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
) error {
	if _, _, err := (&CoursewareService{}).LoadCoursewareForOwnerRuntime(
		ctx,
		coursewareID,
		actor,
	); err != nil {
		return err
	}

	value, exists := cwAssemblyCancelMap.Load(coursewareID)
	if !exists {
		cwAssemblyLog.Info(
			"当前没有运行中的全自动装配任务",
			"courseware_id",
			coursewareID,
		)
		return nil
	}

	signal, ok := value.(*cwAssemblyCancelSignal)
	if !ok || signal == nil {
		return nil
	}

	signal.Cancel()

	cwAssemblyLog.Info(
		"已通知全自动装配停止继续派发",
		"courseware_id",
		coursewareID,
	)

	return nil
}

// pushError 推送装配错误。
func (s *CoursewareAutoAssemblyService) pushError(
	coursewareID string,
	message string,
) {
	GlobalCWSSEHub.Broadcast(
		coursewareID,
		CWSSEEvent{
			EventType: CWSSEError,
			Data: map[string]interface{}{
				"message": message,
			},
		},
	)
}

// buildPageDoneMessage 构造单页完成消息。
func (s *CoursewareAutoAssemblyService) buildPageDoneMessage(
	pageNumber int,
	result cwAssemblyPageResult,
) string {
	parts := []string{
		fmt.Sprintf("第 %d 页装配完成", pageNumber),
	}

	switch {
	case result.imageSkipped:
		parts = append(parts, "无需配图")
	case result.imageOK:
		parts = append(parts, "全部图片槽位完成✓")
	default:
		parts = append(parts, "部分或全部图片槽位失败")
	}

	if result.videoOK {
		parts = append(parts, "视频占位✓")
	}

	return strings.Join(parts, "，")
}
