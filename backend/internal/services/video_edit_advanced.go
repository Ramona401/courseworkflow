package services

// video_edit_advanced.go — 作者专属高级视频拼接
//
// 功能：
//   - 每段独立裁剪；
//   - 无转场时使用concat快速拼接；
//   - 有转场时使用xfade与acrossfade；
//   - 转场失败时在重新授权和输入复验后降级为无转场拼接。
//
// 安全边界：
//   - 请求显式携带可信Actor，不接受裸userID；
//   - 每次片段裁剪和最终FFmpeg前重新授权；
//   - 最终FFmpeg后、资产写库前再次授权；
//   - 全部源资产及磁盘文件版本必须保持不变；
//   - 临时片段、清单和失败输出全部清理；
//   - 新资产继承第一个正式源视频的真实页面归属。

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// validTransitions 与前端转场常量保持一致。
var validTransitions = map[string]bool{
	"none":        true,
	"fade":        true,
	"fadeblack":   true,
	"fadewhite":   true,
	"dissolve":    true,
	"wipeleft":    true,
	"wiperight":   true,
	"wipeup":      true,
	"wipedown":    true,
	"slideleft":   true,
	"slideright":  true,
	"slideup":     true,
	"slidedown":   true,
	"circleopen":  true,
	"circleclose": true,
}

// normalizeTransition 规范化转场名称。
func normalizeTransition(
	transition string,
) string {
	normalized := strings.ToLower(
		strings.TrimSpace(
			transition,
		),
	)

	if normalized == "" {
		return "none"
	}

	if validTransitions[normalized] {
		return normalized
	}

	videoEditLog.Warn(
		"不支持的转场效果，降级为fade",
		"transition", transition,
	)

	return "fade"
}

// VideoClip 单个视频片段配置。
type VideoClip struct {
	AssetID    string  `json:"asset_id"`
	StartSec   float64 `json:"start_sec"`
	EndSec     float64 `json:"end_sec"`
	Transition string  `json:"transition,omitempty"`
	TransDur   float64 `json:"trans_dur,omitempty"`
}

// AdvancedConcatRequest 高级拼接请求。
type AdvancedConcatRequest struct {
	CoursewareID string
	Clips        []VideoClip
	Actor        *CoursewareActorContext
}

// AdvancedConcatResponse 高级拼接响应。
type AdvancedConcatResponse struct {
	AssetID  string `json:"asset_id"`
	URL      string `json:"url"`
	Duration string `json:"duration"`
	Message  string `json:"message"`
}

// preparedVideoClip 是已经绑定正式资产和媒体时长的内部片段。
type preparedVideoClip struct {
	input      *videoEditInputSnapshot
	startSec   float64
	endSec     float64
	duration   float64
	needsTrim  bool
	transition string
	transDur   float64
}

// AdvancedConcat 执行独立裁剪和转场拼接。
func (s *VideoEditService) AdvancedConcat(
	ctx context.Context,
	req *AdvancedConcatRequest,
) (
	*AdvancedConcatResponse,
	error,
) {
	if req == nil ||
		len(req.Clips) < 1 ||
		len(req.Clips) > 10 {
		return nil,
			fmt.Errorf(
				"%w: 单次高级拼接需要1到10个片段",
				ErrVideoEditInputInvalid,
			)
	}

	_,
		scopedActor,
		release,
		err :=
		s.beginVideoEditOperation(
			ctx,
			req.CoursewareID,
			req.Actor,
			"高级视频拼接",
		)
	if err != nil {
		return nil, err
	}
	defer release()

	prepared := make(
		[]preparedVideoClip,
		0,
		len(req.Clips),
	)
	inputs := make(
		[]*videoEditInputSnapshot,
		0,
		len(req.Clips),
	)

	for index, clip := range req.Clips {
		input, err :=
			loadVideoEditInput(
				ctx,
				req.CoursewareID,
				clip.AssetID,
				models.CWAssetTypeVideo,
			)
		if err != nil {
			return nil,
				fmt.Errorf(
					"片段%d: %w",
					index+1,
					err,
				)
		}

		sourceDuration, err :=
			probeMediaDurationSeconds(
				ctx,
				input.Path,
			)
		if err != nil {
			return nil,
				fmt.Errorf(
					"片段%d: 读取时长失败: %w",
					index+1,
					err,
				)
		}

		startSec := clip.StartSec
		endSec := clip.EndSec

		if endSec == 0 {
			endSec = sourceDuration
		}

		if err :=
			validateVideoEditTimeRange(
				startSec,
				endSec,
				sourceDuration,
				0.5,
			); err != nil {
			return nil,
				fmt.Errorf(
					"片段%d: %w",
					index+1,
					err,
				)
		}

		transDur := clip.TransDur
		if transDur == 0 {
			transDur = 0.5
		}
		if !isFiniteVideoEditNumber(
			transDur,
		) ||
			transDur < 0 ||
			transDur > 10 {
			return nil,
				fmt.Errorf(
					"%w: 片段%d转场时长无效",
					ErrVideoEditInputInvalid,
					index+1,
				)
		}

		prepared = append(
			prepared,
			preparedVideoClip{
				input:     input,
				startSec:  startSec,
				endSec:    endSec,
				duration:  endSec - startSec,
				needsTrim: startSec > 0 || clip.EndSec > 0,
				transition: normalizeTransition(
					clip.Transition,
				),
				transDur: transDur,
			},
		)
		inputs = append(
			inputs,
			input,
		)
	}

	for index := 0; index < len(prepared)-1; index++ {
		current := prepared[index]
		next := prepared[index+1]

		if current.transition == "none" {
			continue
		}

		maxTransition :=
			current.duration
		if next.duration < maxTransition {
			maxTransition = next.duration
		}

		if current.transDur >=
			maxTransition-0.05 {
			return nil,
				fmt.Errorf(
					"%w: 第%d段转场时长必须小于相邻片段时长",
					ErrVideoEditInputInvalid,
					index+1,
				)
		}
	}

	pageID, err :=
		validateVideoEditInheritedPage(
			ctx,
			req.CoursewareID,
			inputs[0].Asset.PageID,
		)
	if err != nil {
		return nil, err
	}

	segmentPaths := make(
		[]string,
		0,
		len(prepared),
	)
	tempFiles := make(
		[]string,
		0,
		len(prepared),
	)

	defer func() {
		for _, path := range tempFiles {
			cleanupVideoEditFile(path)
		}
	}()

	for index, clip := range prepared {
		if !clip.needsTrim {
			segmentPaths = append(
				segmentPaths,
				clip.input.Path,
			)
			continue
		}

		_,
			scopedActor,
			err =
			reauthorizeVideoEditOwner(
				ctx,
				req.CoursewareID,
				scopedActor,
			)
		if err != nil {
			return nil, err
		}

		if err :=
			reloadVideoEditInputUnchanged(
				ctx,
				req.CoursewareID,
				clip.input,
				models.CWAssetTypeVideo,
			); err != nil {
			return nil, err
		}

		tempPath,
			_,
			err :=
			createVideoEditOutputPath(
				os.TempDir(),
				"tedna_adv_segment_*.mp4",
			)
		if err != nil {
			return nil, err
		}
		tempFiles = append(
			tempFiles,
			tempPath,
		)

		command := exec.CommandContext(
			ctx,
			"ffmpeg",
			"-y",
			"-ss", fmt.Sprintf(
				"%.3f",
				clip.startSec,
			),
			"-i", clip.input.Path,
			"-t", fmt.Sprintf(
				"%.3f",
				clip.duration,
			),
			"-c", "copy",
			tempPath,
		)

		output, err :=
			command.CombinedOutput()
		if err != nil {
			videoEditLog.Error(
				"高级拼接片段裁剪失败",
				"clip", index+1,
				"error", err,
				"output", string(output),
			)

			return nil,
				fmt.Errorf(
					"片段%d裁剪失败: %w",
					index+1,
					err,
				)
		}

		if _, err :=
			validateVideoEditOutput(
				tempPath,
			); err != nil {
			return nil,
				fmt.Errorf(
					"片段%d输出无效: %w",
					index+1,
					err,
				)
		}

		segmentPaths = append(
			segmentPaths,
			tempPath,
		)
	}

	outputDir := filepath.Join(
		CWAssetUploadDir,
		req.CoursewareID,
		"videos",
	)
	outputPath,
		outputName,
		err :=
		createVideoEditOutputPath(
			outputDir,
			"advanced_concat_*.mp4",
		)
	if err != nil {
		return nil, err
	}

	keepOutput := false

	defer func() {
		if !keepOutput {
			cleanupVideoEditFile(
				outputPath,
			)
		}
	}()

	_,
		scopedActor,
		err =
		reauthorizeVideoEditOwner(
			ctx,
			req.CoursewareID,
			scopedActor,
		)
	if err != nil {
		return nil, err
	}

	if err :=
		reloadVideoEditInputsUnchanged(
			ctx,
			req.CoursewareID,
			inputs,
			models.CWAssetTypeVideo,
		); err != nil {
		return nil, err
	}

	if err :=
		s.runAdvancedConcat(
			ctx,
			req.CoursewareID,
			scopedActor,
			segmentPaths,
			prepared,
			inputs,
			outputPath,
		); err != nil {
		return nil, err
	}

	info, err :=
		validateVideoEditOutput(
			outputPath,
		)
	if err != nil {
		return nil, err
	}

	_,
		scopedActor,
		err =
		reauthorizeVideoEditOwner(
			ctx,
			req.CoursewareID,
			scopedActor,
		)
	if err != nil {
		return nil, err
	}

	if err :=
		reloadVideoEditInputsUnchanged(
			ctx,
			req.CoursewareID,
			inputs,
			models.CWAssetTypeVideo,
		); err != nil {
		return nil, err
	}

	pageID, err =
		validateVideoEditInheritedPage(
			ctx,
			req.CoursewareID,
			pageID,
		)
	if err != nil {
		return nil, err
	}

	duration :=
		getMediaDurationLabel(
			ctx,
			outputPath,
		)
	localURL := videoEditLocalURL(
		req.CoursewareID,
		"videos",
		outputName,
	)

	newAsset := &models.CoursewareAsset{
		CoursewareID:  req.CoursewareID,
		PageID:        pageID,
		PlaceholderID: "",
		AssetType:     models.CWAssetTypeVideo,
		GenerationPrompt: fmt.Sprintf(
			"高级拼接%d个片段",
			len(prepared),
		),
		OssURL:   localURL,
		FileSize: info.Size(),
		MimeType: "video/mp4",
		Status:   models.CWAssetStatusUploaded,
	}

	if err :=
		repository.CreateCWAsset(
			ctx,
			newAsset,
		); err != nil {
		return nil,
			fmt.Errorf(
				"记录高级拼接视频失败: %w",
				err,
			)
	}

	keepOutput = true

	return &AdvancedConcatResponse{
		AssetID:  newAsset.ID,
		URL:      localURL,
		Duration: duration,
		Message: fmt.Sprintf(
			"成功处理%d个片段，总时长%s",
			len(prepared),
			duration,
		),
	}, nil
}
