package services

// courseware_page_version.go — 课件页面版本、回退和手工HTML覆盖服务
//
// 本文件负责作者私有页面源码的版本保护和覆盖安全：
//   - 覆盖旧HTML前保存完整版本快照；
//   - 页面、课件和版本必须形成真实一致的归属关系；
//   - 生成中、自动装配中和审核提交后的课件禁止覆盖；
//   - 作者换校后仍以课件历史education_domain快照运行；
//   - admin和集体备课参与者不能进入作者私有版本链。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// CoursewarePageHTMLMaxBytes 单页HTML允许的最大字节数。
//
// Handler和Service统一使用本常量。Service必须再次校验，避免内部调用或未来
// 新Handler绕过HTTP层限制后写入异常超大页面。
const CoursewarePageHTMLMaxBytes = 5 * 1024 * 1024

var (
	// ErrCoursewarePageNotFound 表示路径指定页面不存在或不属于路径课件。
	ErrCoursewarePageNotFound = errors.New(
		"课件页面不存在",
	)

	// ErrCoursewarePageVersionNotFound 同时用于版本不存在和版本路径归属不一致。
	//
	// 归属不一致统一返回不存在，避免通过随机version_id探测其它课件版本。
	ErrCoursewarePageVersionNotFound = errors.New(
		"课件页面版本不存在",
	)

	// ErrCoursewarePageMutationConflict 表示课件当前处于生成或审核锁定状态。
	ErrCoursewarePageMutationConflict = errors.New(
		"课件页面当前不可修改",
	)

	// ErrCoursewarePageHTMLInvalid 表示客户端提交的整页HTML不合法。
	ErrCoursewarePageHTMLInvalid = errors.New(
		"课件页面HTML无效",
	)

	// ErrCoursewarePageVersionSnapshotFailed 表示破坏性覆盖前无法保存旧版。
	//
	// 回退、手工保存和外部HTML导入必须保证可逆，因此该错误会阻止后续覆盖。
	ErrCoursewarePageVersionSnapshotFailed = errors.New(
		"页面版本快照保存失败",
	)
)

// validateCoursewarePageMutationState 校验页面是否处于可覆盖状态。
//
// 同时检查静态状态和运行锁：
//   - status=generating：课件仍处于生成阶段；
//   - cwGenRunning：批量页面生成正在实际运行；
//   - cwAssemblyRunning：全自动装配正在实际运行；
//   - status=in_pipeline：旧生产状态机审核锁；
//   - publish_state=submitted：正式发布审核锁。
func validateCoursewarePageMutationState(
	courseware *models.Courseware,
) error {
	if courseware == nil {
		return fmt.Errorf(
			"%w: 课件数据为空",
			ErrCoursewarePageMutationConflict,
		)
	}

	if courseware.Status ==
		models.CoursewareStatusGenerating {
		return fmt.Errorf(
			"%w: 课件处于生成阶段，暂不能覆盖页面",
			ErrCoursewarePageMutationConflict,
		)
	}

	if _, running := cwGenRunning.Load(
		courseware.ID,
	); running {
		return fmt.Errorf(
			"%w: 课件正在批量生成，暂不能覆盖页面",
			ErrCoursewarePageMutationConflict,
		)
	}

	if _, running := cwAssemblyRunning.Load(
		courseware.ID,
	); running {
		return fmt.Errorf(
			"%w: 课件正在全自动装配，暂不能覆盖页面",
			ErrCoursewarePageMutationConflict,
		)
	}

	if courseware.Status ==
		models.CoursewareStatusInPipeline {
		return fmt.Errorf(
			"%w: 课件已进入审核流程，暂不能覆盖页面",
			ErrCoursewarePageMutationConflict,
		)
	}

	if courseware.PublishState ==
		models.CWPublishSubmitted {
		return fmt.Errorf(
			"%w: 课件已提交审核，暂不能覆盖页面",
			ErrCoursewarePageMutationConflict,
		)
	}

	return nil
}

// validateCoursewarePageHTMLPayload 校验外部提交的整页HTML。
func validateCoursewarePageHTMLPayload(
	html string,
) error {
	if strings.TrimSpace(html) == "" {
		return fmt.Errorf(
			"%w: 页面内容为空",
			ErrCoursewarePageHTMLInvalid,
		)
	}

	if len(html) > CoursewarePageHTMLMaxBytes {
		return fmt.Errorf(
			"%w: 页面内容超过5MB上限",
			ErrCoursewarePageHTMLInvalid,
		)
	}

	return nil
}

// validateCoursewarePageVersionPath 校验课件、页面和版本三层真实归属。
//
// 即使仓储已经使用三字段精确SQL，本层仍保留二次防线，避免未来仓储接口
// 调整后意外放大版本读取或回退范围。
func validateCoursewarePageVersionPath(
	coursewareID string,
	page *models.CoursewarePage,
	version *models.CoursewarePageVersion,
) error {
	if strings.TrimSpace(coursewareID) == "" ||
		page == nil ||
		version == nil {
		return ErrCoursewarePageVersionNotFound
	}

	if page.CoursewareID != coursewareID ||
		version.PageID != page.ID ||
		version.CoursewareID != coursewareID {
		return ErrCoursewarePageVersionNotFound
	}

	return nil
}

// isValidCoursewarePageStatus 校验版本快照中的页面状态。
func isValidCoursewarePageStatus(
	status string,
) bool {
	switch status {
	case models.CWPageStatusPending,
		models.CWPageStatusGenerated,
		models.CWPageStatusMediaFilling,
		models.CWPageStatusConfirmed:
		return true
	default:
		return false
	}
}

// resolveCoursewarePageVersionRestoreMetadata 决定回退时应恢复的页面元数据。
//
// 新完整版本恢复自身元数据；迁移前旧版本只恢复HTML，保留当前页面元数据，
// 防止使用当前值伪装成历史快照。
func resolveCoursewarePageVersionRestoreMetadata(
	page *models.CoursewarePage,
	version *models.CoursewarePageVersion,
) (
	placeholderMap string,
	matchedComponentIDs string,
	pageStatus string,
	metadataRestored bool,
	err error,
) {
	if page == nil || version == nil {
		return "", "", "", false,
			ErrCoursewarePageVersionNotFound
	}

	if !version.MetadataSnapshotComplete {
		return page.PlaceholderMap,
			page.MatchedComponentIDs,
			page.Status,
			false,
			nil
	}

	if !isValidCoursewarePageStatus(
		version.PageStatus,
	) {
		return "", "", "", false,
			fmt.Errorf(
				"%w: 完整版本的页面状态无效",
				ErrCoursewarePageVersionSnapshotFailed,
			)
	}

	return version.PlaceholderMap,
		version.MatchedComponentIDs,
		version.PageStatus,
		true,
		nil
}

// loadOwnedCoursewarePage 加载作者私有课件及路径指定页面。
func (s *CoursewareGenService) loadOwnedCoursewarePage(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	pageNum int,
) (
	*models.Courseware,
	*models.CoursewarePage,
	error,
) {
	courseware, _, err :=
		(&CoursewareService{}).
			LoadCoursewareForOwnerRuntime(
				ctx,
				coursewareID,
				actor,
			)
	if err != nil {
		return nil, nil, err
	}

	page, err :=
		repository.GetCoursewarePageByNumber(
			ctx,
			coursewareID,
			pageNum,
		)
	if err != nil ||
		page == nil ||
		page.CoursewareID != coursewareID {
		return nil, nil, fmt.Errorf(
			"%w: 课件=%s 页码=%d",
			ErrCoursewarePageNotFound,
			coursewareID,
			pageNum,
		)
	}

	return courseware, page, nil
}

// loadOwnedCoursewarePageForMutation 加载并校验可覆盖状态。
func (s *CoursewareGenService) loadOwnedCoursewarePageForMutation(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	pageNum int,
) (
	*models.Courseware,
	*models.CoursewarePage,
	error,
) {
	courseware, page, err :=
		s.loadOwnedCoursewarePage(
			ctx,
			coursewareID,
			actor,
			pageNum,
		)
	if err != nil {
		return nil, nil, err
	}

	if err := validateCoursewarePageMutationState(
		courseware,
	); err != nil {
		return nil, nil, err
	}

	return courseware, page, nil
}

// SavePageVersionBeforeOverwriteStrict 强制保存覆盖前的旧HTML。
//
// oldHTML为空表示首次生成，不需要保存版本；其它情况下保存失败必须返回错误，
// 调用方不得继续覆盖页面。
func (s *CoursewareGenService) SavePageVersionBeforeOverwriteStrict(
	ctx context.Context,
	pageID string,
	coursewareID string,
	oldHTML string,
	source string,
	note string,
) error {
	if strings.TrimSpace(oldHTML) == "" {
		return nil
	}

	version, err := repository.CreatePageVersion(
		ctx,
		pageID,
		coursewareID,
		oldHTML,
		source,
		note,
	)
	if err != nil {
		return fmt.Errorf(
			"%w: %v",
			ErrCoursewarePageVersionSnapshotFailed,
			err,
		)
	}

	cwGenLog.Info(
		"页面版本快照已保存",
		"courseware_id",
		coursewareID,
		"page_id",
		pageID,
		"version_no",
		version.VersionNo,
		"source",
		source,
	)

	return nil
}

// SavePageVersionBeforeOverwrite 保留原best-effort接口。
//
// AI微调和重生等既有调用继续使用本接口，不因附加版本能力失败而中断AI主流程。
// 回退、手工保存和HTML导入等破坏性确定性覆盖使用Strict接口。
func (s *CoursewareGenService) SavePageVersionBeforeOverwrite(
	ctx context.Context,
	pageID string,
	coursewareID string,
	oldHTML string,
	source string,
	note string,
) {
	if err := s.SavePageVersionBeforeOverwriteStrict(
		ctx,
		pageID,
		coursewareID,
		oldHTML,
		source,
		note,
	); err != nil {
		cwGenLog.Warn(
			"页面版本快照保存失败（不影响本次AI修改）",
			"error",
			err,
			"page_id",
			pageID,
			"courseware_id",
			coursewareID,
			"source",
			source,
		)
	}
}

// ListCWPageVersions 返回作者指定页面的历史版本列表。
func (s *CoursewareGenService) ListCWPageVersions(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	pageNum int,
) (
	[]*models.CoursewarePageVersionListItem,
	error,
) {
	_, page, err := s.loadOwnedCoursewarePage(
		ctx,
		coursewareID,
		actor,
		pageNum,
	)
	if err != nil {
		return nil, err
	}

	items, err := repository.ListPageVersions(
		ctx,
		page.ID,
		coursewareID,
	)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items =
			[]*models.CoursewarePageVersionListItem{}
	}

	return items, nil
}

// GetCWPageVersionHTML 返回作者指定页面的一个完整历史版本。
func (s *CoursewareGenService) GetCWPageVersionHTML(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	pageNum int,
	versionID string,
) (
	html string,
	versionNo int,
	source string,
	err error,
) {
	_, page, err := s.loadOwnedCoursewarePage(
		ctx,
		coursewareID,
		actor,
		pageNum,
	)
	if err != nil {
		return "", 0, "", err
	}

	versionID = strings.TrimSpace(versionID)
	if versionID == "" {
		return "", 0, "",
			ErrCoursewarePageVersionNotFound
	}

	target, err := repository.GetPageVersion(
		ctx,
		versionID,
		page.ID,
		coursewareID,
	)
	if err != nil {
		if errors.Is(
			err,
			repository.ErrCoursewarePageVersionNotFound,
		) {
			return "", 0, "",
				ErrCoursewarePageVersionNotFound
		}

		return "", 0, "", fmt.Errorf(
			"查询页面版本失败: %w",
			err,
		)
	}

	if err := validateCoursewarePageVersionPath(
		coursewareID,
		page,
		target,
	); err != nil {
		return "", 0, "", err
	}

	return target.HTMLContent,
		target.VersionNo,
		target.Source,
		nil
}

// RollbackCWPage 将指定页面回退到一个历史版本。
//
// 回退前必须成功保存当前HTML，保证回退操作自身可逆。
func (s *CoursewareGenService) RollbackCWPage(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	pageNum int,
	versionID string,
) (string, error) {
	_, page, err :=
		s.loadOwnedCoursewarePageForMutation(
			ctx,
			coursewareID,
			actor,
			pageNum,
		)
	if err != nil {
		return "", err
	}

	versionID = strings.TrimSpace(versionID)
	if versionID == "" {
		return "",
			ErrCoursewarePageVersionNotFound
	}

	target, err := repository.GetPageVersion(
		ctx,
		versionID,
		page.ID,
		coursewareID,
	)
	if err != nil {
		if errors.Is(
			err,
			repository.ErrCoursewarePageVersionNotFound,
		) {
			return "",
				ErrCoursewarePageVersionNotFound
		}

		return "", fmt.Errorf(
			"查询目标版本失败: %w",
			err,
		)
	}

	if err := validateCoursewarePageVersionPath(
		coursewareID,
		page,
		target,
	); err != nil {
		return "", err
	}

	if strings.TrimSpace(
		target.HTMLContent,
	) == "" {
		return "",
			ErrCoursewarePageVersionNotFound
	}

	if err := s.SavePageVersionBeforeOverwriteStrict(
		ctx,
		page.ID,
		coursewareID,
		page.HTMLContent,
		models.CWPageVersionSourceRollback,
		fmt.Sprintf(
			"回退到第%d版前的当前内容",
			target.VersionNo,
		),
	); err != nil {
		return "", err
	}

	placeholderMap,
		matchedComponentIDs,
		pageStatus,
		metadataRestored,
		metadataErr :=
		resolveCoursewarePageVersionRestoreMetadata(
			page,
			target,
		)
	if metadataErr != nil {
		return "", metadataErr
	}

	if err := repository.UpdateCWPageHTML(
		ctx,
		page.ID,
		target.HTMLContent,
		placeholderMap,
		matchedComponentIDs,
		pageStatus,
	); err != nil {
		return "", fmt.Errorf(
			"写回回退内容失败: %w",
			err,
		)
	}

	cwGenLog.Info(
		"页面已回退到历史版本",
		"courseware_id",
		coursewareID,
		"page_num",
		pageNum,
		"rolled_back_to_version",
		target.VersionNo,
		"version_id",
		versionID,
		"metadata_restored",
		metadataRestored,
	)

	return target.HTMLContent, nil
}

// SaveManualEditedPage 保存老师就地编辑后的整页HTML。
func (s *CoursewareGenService) SaveManualEditedPage(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	pageNum int,
	newHTML string,
) (string, error) {
	_, page, err :=
		s.loadOwnedCoursewarePageForMutation(
			ctx,
			coursewareID,
			actor,
			pageNum,
		)
	if err != nil {
		return "", err
	}

	if strings.TrimSpace(
		page.HTMLContent,
	) == "" {
		return "", fmt.Errorf(
			"%w: 页面尚未生成，无法就地编辑",
			ErrCoursewarePageHTMLInvalid,
		)
	}

	if err := validateCoursewarePageHTMLPayload(
		newHTML,
	); err != nil {
		return "", err
	}

	if newHTML == page.HTMLContent {
		return page.HTMLContent, nil
	}

	validation := validateRefinedPageHTML(
		page.HTMLContent,
		newHTML,
		"就地文字编辑",
		false,
	)
	if !validation.OK {
		return "", fmt.Errorf(
			"%w: %s",
			ErrCoursewarePageHTMLInvalid,
			validation.Reason,
		)
	}
	if validation.FixedHTML != "" {
		newHTML = validation.FixedHTML
	}

	if err := validateCoursewarePageHTMLPayload(
		newHTML,
	); err != nil {
		return "", err
	}

	if err := s.SavePageVersionBeforeOverwriteStrict(
		ctx,
		page.ID,
		coursewareID,
		page.HTMLContent,
		models.CWPageVersionSourceManual,
		"就地文字编辑前",
	); err != nil {
		return "", err
	}

	if err := repository.UpdateCWPageHTML(
		ctx,
		page.ID,
		newHTML,
		page.PlaceholderMap,
		page.MatchedComponentIDs,
		models.CWPageStatusGenerated,
	); err != nil {
		return "", fmt.Errorf(
			"保存编辑内容失败: %w",
			err,
		)
	}

	cwGenLog.Info(
		"页面就地文字编辑已保存",
		"courseware_id",
		coursewareID,
		"page_num",
		pageNum,
		"page_id",
		page.ID,
	)

	return newHTML, nil
}
