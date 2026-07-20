package services

// video_draft_service.go — 视频编辑器草稿可信Actor治理
//
// 权限语义：
//   - 列表：课件可信查看权，只返回当前用户自己的草稿；
//   - 保存：课件教研微调权；
//   - 删除：当前用户仍具有课件查看权，并且草稿属于本人；
//   - admin不会因为平台角色自动进入普通教研草稿写通道；
//   - 集体备课参与者可在进行中的合法会话内保存自己的草稿；
//   - 草稿内的每个视频资产ID必须重新验证属于当前课件。
//
// 输入格式校验被拆分到video_draft_validation.go，避免本文件超过
// 600行，同时让纯校验逻辑可以独立执行单元测试。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"tedna/internal/logger"
	"tedna/internal/repository"
)

const videoDraftMaxKeep = 20

var (
	ErrVideoDraftInputInvalid = errors.New(
		"视频草稿参数无效",
	)
	ErrVideoDraftNotFound = errors.New(
		"视频草稿不存在",
	)
)

var videoDraftLog = logger.WithModule(
	"video_draft",
)

// VideoDraftSaveInput 保存视频草稿请求。
type VideoDraftSaveInput struct {
	Name      string          `json:"name"`
	ClipsData json.RawMessage `json:"clips_data"`
	ClipCount int             `json:"clip_count"`
}

// VideoDraftService 视频编辑器草稿服务。
type VideoDraftService struct {
	coursewareService *CoursewareService
}

// NewVideoDraftService 创建视频草稿服务。
func NewVideoDraftService() *VideoDraftService {
	return &VideoDraftService{
		coursewareService: NewCoursewareService(),
	}
}

// PreflightSaveDraft 在Handler解析大正文前完成保存权限预检。
//
// 本方法只用于尽早拒绝无权请求。SaveDraft仍会重新加载正式课件，
// 并在正式写入前再次授权，不能用本预检替代Service终校验。
func (s *VideoDraftService) PreflightSaveDraft(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
) (
	*CoursewareActorContext,
	error,
) {
	_, scopedActor, err :=
		s.coursewareService.LoadCoursewareForRefine(
			ctx,
			coursewareID,
			actor,
		)
	if err != nil {
		return nil, err
	}

	return scopedActor, nil
}

// ListDrafts 按可信课件查看权读取当前用户自己的草稿。
func (s *VideoDraftService) ListDrafts(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
) (
	[]*repository.VideoDraftItem,
	error,
) {
	if _, err := s.coursewareService.LoadCoursewareForView(
		ctx,
		coursewareID,
		actor,
	); err != nil {
		return nil, err
	}

	drafts, err := repository.ListVideoDrafts(
		ctx,
		coursewareID,
		actor.UserID,
	)
	if err != nil {
		videoDraftLog.Error(
			"查询视频草稿失败",
			"courseware_id", coursewareID,
			"user_id", actor.UserID,
			"error", err,
		)

		return nil, fmt.Errorf(
			"查询视频草稿失败",
		)
	}

	return drafts, nil
}

// SaveDraft 在教研微调权限下保存当前用户自己的草稿。
func (s *VideoDraftService) SaveDraft(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	input *VideoDraftSaveInput,
) (
	*repository.VideoDraftItem,
	error,
) {
	_, scopedActor, err :=
		s.coursewareService.LoadCoursewareForRefine(
			ctx,
			coursewareID,
			actor,
		)
	if err != nil {
		return nil, err
	}

	name, clipsJSON, clipCount, err :=
		normalizeVideoDraftInput(input)
	if err != nil {
		return nil, err
	}

	// 正式写入前再次加载课件并重新执行微调授权。
	_, latestActor, err :=
		s.coursewareService.LoadCoursewareForRefine(
			ctx,
			coursewareID,
			scopedActor,
		)
	if err != nil {
		return nil, err
	}

	// 草稿中的片段ID会在导出时作为正式asset_id使用，因此不能只做
	// JSON形状校验，必须重新确认每个ID都属于当前课件的视频资产。
	if err := validateVideoDraftAssets(
		ctx,
		coursewareID,
		clipsJSON,
	); err != nil {
		return nil, err
	}

	draft, err := repository.CreateVideoDraftCapped(
		ctx,
		coursewareID,
		latestActor.UserID,
		name,
		clipsJSON,
		clipCount,
		videoDraftMaxKeep,
	)
	if err != nil {
		videoDraftLog.Error(
			"保存视频草稿失败",
			"courseware_id", coursewareID,
			"user_id", latestActor.UserID,
			"clip_count", clipCount,
			"error", err,
		)

		return nil, fmt.Errorf(
			"保存视频草稿失败",
		)
	}

	videoDraftLog.Info(
		"视频草稿保存",
		"courseware_id", coursewareID,
		"draft_id", draft.ID,
		"user_id", latestActor.UserID,
		"clip_count", clipCount,
	)

	return draft, nil
}

// validateVideoDraftAssets 重新验证草稿片段资产属于当前课件。
func validateVideoDraftAssets(
	ctx context.Context,
	coursewareID string,
	clipsJSON string,
) error {
	var clips []videoDraftClipInput

	if err := json.Unmarshal(
		[]byte(clipsJSON),
		&clips,
	); err != nil {
		return fmt.Errorf(
			"%w: clips_data格式错误",
			ErrVideoDraftInputInvalid,
		)
	}

	assetIDs := make(
		[]string,
		0,
		len(clips),
	)

	for _, clip := range clips {
		assetID := strings.TrimSpace(
			clip.ID,
		)
		if assetID == "" {
			return fmt.Errorf(
				"%w: 草稿片段缺少视频资产ID",
				ErrVideoDraftInputInvalid,
			)
		}

		assetIDs = append(
			assetIDs,
			assetID,
		)
	}

	matched, err :=
		repository.VideoDraftAssetsBelongToCourseware(
			ctx,
			coursewareID,
			assetIDs,
		)
	if err != nil {
		videoDraftLog.Error(
			"校验视频草稿资产失败",
			"courseware_id", coursewareID,
			"asset_count", len(assetIDs),
			"error", err,
		)

		return fmt.Errorf(
			"校验视频草稿片段失败",
		)
	}

	if !matched {
		return fmt.Errorf(
			"%w: 草稿片段必须全部来自当前课件的视频资产",
			ErrVideoDraftInputInvalid,
		)
	}

	return nil
}

// DeleteDraft 在合法课件查看权下删除当前路径课件中的本人草稿。
//
// 删除不要求当前仍具有微调权，避免集体备课结束后本人草稿无法清理；
// 但操作者仍必须具有当前课件合法查看权，并且Repository继续绑定
// courseware_id、draft_id和user_id三层边界。
func (s *VideoDraftService) DeleteDraft(
	ctx context.Context,
	coursewareID string,
	draftID string,
	actor *CoursewareActorContext,
) error {
	if _, err :=
		s.coursewareService.LoadCoursewareForView(
			ctx,
			coursewareID,
			actor,
		); err != nil {
		return err
	}

	// 第一次复合读取，确认草稿属于当前路径课件和当前用户。
	if _, err := repository.GetVideoDraftForCoursewareUser(
		ctx,
		coursewareID,
		draftID,
		actor.UserID,
	); err != nil {
		if errors.Is(
			err,
			repository.ErrVideoDraftNotFound,
		) {
			return ErrVideoDraftNotFound
		}

		videoDraftLog.Error(
			"读取待删除视频草稿失败",
			"courseware_id", coursewareID,
			"draft_id", draftID,
			"user_id", actor.UserID,
			"error", err,
		)

		return fmt.Errorf(
			"读取视频草稿失败",
		)
	}

	// 正式删除前再次加载正式课件并重新验证查看权。
	if _, err :=
		s.coursewareService.LoadCoursewareForView(
			ctx,
			coursewareID,
			actor,
		); err != nil {
		return err
	}

	if err := repository.DeleteVideoDraftForCoursewareUser(
		ctx,
		coursewareID,
		draftID,
		actor.UserID,
	); err != nil {
		if errors.Is(
			err,
			repository.ErrVideoDraftNotFound,
		) {
			return ErrVideoDraftNotFound
		}

		videoDraftLog.Error(
			"删除视频草稿失败",
			"courseware_id", coursewareID,
			"draft_id", draftID,
			"user_id", actor.UserID,
			"error", err,
		)

		return fmt.Errorf(
			"删除视频草稿失败",
		)
	}

	videoDraftLog.Info(
		"视频草稿删除",
		"courseware_id", coursewareID,
		"draft_id", draftID,
		"user_id", actor.UserID,
	)

	return nil
}
