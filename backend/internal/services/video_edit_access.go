package services

// video_edit_access.go — FFmpeg写入口的可信Actor、并发与正式资源复验
//
// 本文件只承担安全边界，不拼接具体FFmpeg参数：
//   - 所有高影响Service显式接收CoursewareActorContext；
//   - 作者控制授权统一使用LoadCoursewareForOwnerControlMutation；
//   - 同一课件同一时刻只允许一个视频编辑长任务；
//   - 正式资产通过courseware_id + asset_id + asset_type重新绑定；
//   - FFmpeg执行前后比较资产记录和本地文件版本；
//   - 派生资产继承的page_id必须仍属于当前课件。

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"tedna/internal/models"
	"tedna/internal/repository"
)

const (
	// VideoEditMaxBodyBytes 限制公开FFmpeg入口的JSON正文。
	VideoEditMaxBodyBytes = 512 << 10
)

var (
	// ErrVideoEditInputInvalid 表示公开输入形状或数值无效。
	ErrVideoEditInputInvalid = errors.New(
		"视频编辑参数无效",
	)
	// ErrVideoEditAssetNotFound 对外统一表示正式资产不存在。
	ErrVideoEditAssetNotFound = errors.New(
		"视频编辑资产不存在",
	)
	// ErrVideoEditBusy 表示同课件已有视频编辑长任务运行。
	ErrVideoEditBusy = errors.New(
		"当前课件正在执行其它视频编辑任务",
	)
	// ErrVideoEditSourceChanged 表示长任务期间正式输入发生变化。
	ErrVideoEditSourceChanged = errors.New(
		"视频编辑源数据已发生变化，请刷新后重试",
	)
	// ErrVideoEditOutputInvalid 表示FFmpeg没有生成可提交的文件。
	ErrVideoEditOutputInvalid = errors.New(
		"视频编辑输出文件无效",
	)
)

// videoEditTaskRegistry 是进程内课件级长任务互斥表。
//
// 服务当前为单生产进程；本互斥可以阻止同课件多个FFmpeg任务同时
// 覆盖磁盘和竞争资产写入。任务结束后无论成功失败都会释放。
var videoEditTaskRegistry = struct {
	sync.Mutex
	active map[string]string
}{
	active: make(map[string]string),
}

// videoEditInputSnapshot 保存正式资产及本地文件版本。
type videoEditInputSnapshot struct {
	Asset    *models.CoursewareAsset
	Path     string
	Size     int64
	ModTime  time.Time
	FileMode os.FileMode
}

// PreflightOwnerMutation 在Handler解析正文前执行作者控制预检。
//
// Service正式方法仍会调用beginVideoEditOperation重新加载正式课件；
// 本预检不能替代FFmpeg前后和资产写库前的重新授权。
func (s *VideoEditService) PreflightOwnerMutation(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
) (
	*CoursewareActorContext,
	error,
) {
	_, scopedActor, err :=
		(&CoursewareService{}).
			LoadCoursewareForOwnerControlMutation(
				ctx,
				strings.TrimSpace(coursewareID),
				actor,
			)
	if err != nil {
		return nil, err
	}

	return scopedActor, nil
}

// beginVideoEditOperation 重新授权并登记课件级长任务。
func (s *VideoEditService) beginVideoEditOperation(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	action string,
) (
	*models.Courseware,
	*CoursewareActorContext,
	func(),
	error,
) {
	coursewareID = strings.TrimSpace(coursewareID)
	if coursewareID == "" {
		return nil,
			nil,
			nil,
			fmt.Errorf(
				"%w: courseware_id不能为空",
				ErrVideoEditInputInvalid,
			)
	}

	courseware,
		scopedActor,
		err :=
		(&CoursewareService{}).
			LoadCoursewareForOwnerControlMutation(
				ctx,
				coursewareID,
				actor,
			)
	if err != nil {
		return nil, nil, nil, err
	}

	release, err :=
		acquireVideoEditTask(
			coursewareID,
			action,
		)
	if err != nil {
		return nil, nil, nil, err
	}

	return courseware,
		scopedActor,
		release,
		nil
}

// reauthorizeVideoEditOwner 重新加载正式课件并再次执行作者控制授权。
func reauthorizeVideoEditOwner(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
) (
	*models.Courseware,
	*CoursewareActorContext,
	error,
) {
	return (&CoursewareService{}).
		LoadCoursewareForOwnerControlMutation(
			ctx,
			coursewareID,
			actor,
		)
}

// acquireVideoEditTask 获取课件级FFmpeg互斥。
func acquireVideoEditTask(
	coursewareID string,
	action string,
) (
	func(),
	error,
) {
	videoEditTaskRegistry.Lock()

	if runningAction, exists :=
		videoEditTaskRegistry.active[coursewareID]; exists {
		videoEditTaskRegistry.Unlock()

		return nil,
			fmt.Errorf(
				"%w: 正在执行%s",
				ErrVideoEditBusy,
				runningAction,
			)
	}

	videoEditTaskRegistry.active[coursewareID] =
		strings.TrimSpace(action)

	videoEditTaskRegistry.Unlock()

	var once sync.Once

	return func() {
		once.Do(func() {
			videoEditTaskRegistry.Lock()
			delete(
				videoEditTaskRegistry.active,
				coursewareID,
			)
			videoEditTaskRegistry.Unlock()
		})
	}, nil
}

// loadVideoEditInput 使用复合边界加载正式资产与本地文件快照。
func loadVideoEditInput(
	ctx context.Context,
	coursewareID string,
	assetID string,
	expectedType string,
) (
	*videoEditInputSnapshot,
	error,
) {
	assetID = strings.TrimSpace(assetID)
	if assetID == "" {
		return nil,
			fmt.Errorf(
				"%w: asset_id不能为空",
				ErrVideoEditInputInvalid,
			)
	}

	asset, err :=
		repository.GetCWAssetByID(
			ctx,
			assetID,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"%w: %v",
				ErrVideoEditAssetNotFound,
				err,
			)
	}

	if asset.CoursewareID != coursewareID ||
		asset.AssetType != expectedType {
		return nil,
			ErrVideoEditAssetNotFound
	}

	path := resolveAssetPath(asset)
	if path == "" {
		return nil,
			fmt.Errorf(
				"%w: 本地文件不存在",
				ErrVideoEditAssetNotFound,
			)
	}

	info, err := os.Stat(path)
	if err != nil ||
		!info.Mode().IsRegular() {
		return nil,
			fmt.Errorf(
				"%w: 本地文件不存在",
				ErrVideoEditAssetNotFound,
			)
	}

	return &videoEditInputSnapshot{
		Asset:    asset,
		Path:     path,
		Size:     info.Size(),
		ModTime:  info.ModTime(),
		FileMode: info.Mode(),
	}, nil
}

// reloadVideoEditInputUnchanged 重新加载并确认正式资产和磁盘文件未变化。
func reloadVideoEditInputUnchanged(
	ctx context.Context,
	coursewareID string,
	expected *videoEditInputSnapshot,
	expectedType string,
) error {
	if expected == nil ||
		expected.Asset == nil {
		return ErrVideoEditSourceChanged
	}

	latest, err :=
		loadVideoEditInput(
			ctx,
			coursewareID,
			expected.Asset.ID,
			expectedType,
		)
	if err != nil {
		return ErrVideoEditSourceChanged
	}

	if !videoEditAssetRevisionEqual(
		expected.Asset,
		latest.Asset,
	) ||
		expected.Path != latest.Path ||
		expected.Size != latest.Size ||
		!expected.ModTime.Equal(
			latest.ModTime,
		) ||
		expected.FileMode != latest.FileMode {
		return ErrVideoEditSourceChanged
	}

	return nil
}

// reloadVideoEditInputsUnchanged 批量复验正式输入。
func reloadVideoEditInputsUnchanged(
	ctx context.Context,
	coursewareID string,
	inputs []*videoEditInputSnapshot,
	expectedType string,
) error {
	for _, input := range inputs {
		if err :=
			reloadVideoEditInputUnchanged(
				ctx,
				coursewareID,
				input,
				expectedType,
			); err != nil {
			return err
		}
	}

	return nil
}

// videoEditAssetRevisionEqual 比较会影响派生结果的正式资产字段。
func videoEditAssetRevisionEqual(
	before *models.CoursewareAsset,
	after *models.CoursewareAsset,
) bool {
	if before == nil ||
		after == nil {
		return false
	}

	return before.ID == after.ID &&
		before.CoursewareID ==
			after.CoursewareID &&
		before.AssetType ==
			after.AssetType &&
		before.OssURL ==
			after.OssURL &&
		before.PublicOSSURL ==
			after.PublicOSSURL &&
		before.Status ==
			after.Status &&
		before.MimeType ==
			after.MimeType &&
		before.Metadata ==
			after.Metadata &&
		videoEditStringPointerEqual(
			before.PageID,
			after.PageID,
		)
}

// videoEditStringPointerEqual 比较可空字符串值。
func videoEditStringPointerEqual(
	left *string,
	right *string,
) bool {
	switch {
	case left == nil && right == nil:
		return true

	case left == nil || right == nil:
		return false

	default:
		return *left == *right
	}
}

// cloneVideoEditPageID 深复制可空page_id。
func cloneVideoEditPageID(
	pageID *string,
) *string {
	if pageID == nil {
		return nil
	}

	value := strings.TrimSpace(*pageID)
	if value == "" {
		return nil
	}

	return &value
}

// validateVideoEditInheritedPage 确认派生资产继承的页面仍属于当前课件。
func validateVideoEditInheritedPage(
	ctx context.Context,
	coursewareID string,
	pageID *string,
) (
	*string,
	error,
) {
	normalized := cloneVideoEditPageID(
		pageID,
	)
	if normalized == nil {
		return nil, nil
	}

	pages, err :=
		repository.ListCoursewarePages(
			ctx,
			coursewareID,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"读取课件页面失败: %w",
				err,
			)
	}

	for _, page := range pages {
		if page == nil {
			continue
		}

		if page.ID == *normalized &&
			page.CoursewareID ==
				coursewareID {
			return normalized, nil
		}
	}

	return nil, ErrVideoEditSourceChanged
}
