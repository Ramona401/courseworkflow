package services

// courseware_gen_refine_page.go
//
// 课件页面AI写回的公共模式、页面守卫与兼容入口。
//
// 业务实现已继续按职责拆分：
//   - 保留结构微调和全页重构：courseware_gen_refine_page_apply.go；
//   - 单页从零重生：courseware_gen_regenerate_page.go；
//   - 原子版本快照与CAS写回：repository/courseware_page_cas_repo.go。
//
// 兼容边界：
//   - RefinePage保留原签名，默认执行保留结构微调；
//   - RefinePageWithMode保留原签名，供全页重构讨论等既有调用使用；
//   - RefinePageWithModeGuarded接受后端可信页面守卫，供整改项执行链使用。

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

const (
	cwRefineModePreserve = "preserve"
	cwRefineModeRebuild  = "rebuild"
)

// CoursewarePageMutationGuard 是页面AI执行所绑定的可信基线。
//
// PageID、PageNumber和HTMLHash必须来自后端正式页面或数据库事务，
// 不能直接使用浏览器声明的数据构造。
type CoursewarePageMutationGuard struct {
	PageID     string
	PageNumber int
	HTMLHash   string
}

func normalizeCWRefineMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case cwRefineModeRebuild:
		return cwRefineModeRebuild
	default:
		return cwRefineModePreserve
	}
}

// hashCWPageHTMLForMutation 使用与审核快照一致的SHA-256十六进制格式。
func hashCWPageHTMLForMutation(htmlContent string) string {
	sum := sha256.Sum256([]byte(htmlContent))
	return fmt.Sprintf("%x", sum[:])
}

// resolveCWPageMutationGuard 将当前正式页面与可选上游守卫绑定。
//
// 未传守卫时，以本次服务读取到的页面形成普通AI修改基线。
// 已传守卫时，当前页面必须仍与上游事务确认的稳定页面完全一致。
func resolveCWPageMutationGuard(
	page *models.CoursewarePage,
	requested *CoursewarePageMutationGuard,
) (*CoursewarePageMutationGuard, error) {
	if page == nil {
		return nil, ErrCoursewarePageMutationConflict
	}

	current := &CoursewarePageMutationGuard{
		PageID:     page.ID,
		PageNumber: page.PageNumber,
		HTMLHash:   hashCWPageHTMLForMutation(page.HTMLContent),
	}

	if requested == nil {
		return current, nil
	}

	requestedPageID := strings.TrimSpace(requested.PageID)
	requestedHash := strings.ToLower(strings.TrimSpace(requested.HTMLHash))

	if requestedPageID == "" ||
		requested.PageNumber <= 0 ||
		len(requestedHash) != 64 ||
		requestedPageID != current.PageID ||
		requested.PageNumber != current.PageNumber ||
		requestedHash != current.HTMLHash {
		return nil, ErrCoursewarePageMutationConflict
	}

	return &CoursewarePageMutationGuard{
		PageID:     requestedPageID,
		PageNumber: requested.PageNumber,
		HTMLHash:   requestedHash,
	}, nil
}

// ensureCWPageStillMatchesMutationBaseline 是AI返回后的快速冲突检测。
//
// 最终一致性仍由仓储事务内的FOR UPDATE与CAS比较保证。
// 本检查用于在进入组件复核和事务写入前尽早返回清晰冲突。
func ensureCWPageStillMatchesMutationBaseline(
	page *models.CoursewarePage,
	baselinePage *models.CoursewarePage,
	guard *CoursewarePageMutationGuard,
) error {
	if page == nil || baselinePage == nil || guard == nil {
		return ErrCoursewarePageMutationConflict
	}

	if page.ID != guard.PageID ||
		page.PageNumber != guard.PageNumber ||
		hashCWPageHTMLForMutation(page.HTMLContent) != guard.HTMLHash ||
		page.PlaceholderMap != baselinePage.PlaceholderMap ||
		page.MatchedComponentIDs != baselinePage.MatchedComponentIDs ||
		page.Status != baselinePage.Status {
		return ErrCoursewarePageMutationConflict
	}

	return nil
}

// mapCWPageCASWriteError 将仓储CAS冲突统一映射到页面修改冲突协议。
func mapCWPageCASWriteError(err error) error {
	if errors.Is(err, repository.ErrCoursewarePageCASConflict) {
		return ErrCoursewarePageMutationConflict
	}
	return err
}

// RefinePage 保留原签名，旧调用默认走保留结构微调。
func (s *CoursewareGenService) RefinePage(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	pageNumber int,
	instruction string,
	imageDataURI string,
) (string, error) {
	return s.RefinePageWithMode(
		ctx,
		coursewareID,
		actor,
		pageNumber,
		instruction,
		imageDataURI,
		cwRefineModePreserve,
	)
}

// RefinePageWithMode 保留原调用入口。
func (s *CoursewareGenService) RefinePageWithMode(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	pageNumber int,
	instruction string,
	imageDataURI string,
	mode string,
) (string, error) {
	return s.RefinePageWithModeGuarded(
		ctx,
		coursewareID,
		actor,
		pageNumber,
		instruction,
		imageDataURI,
		mode,
		nil,
	)
}
