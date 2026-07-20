package services

// video_edit_transition_service.go — 高级拼接最终FFmpeg执行
//
// 本文件负责单片段复制、无转场concat和xfade转场链。
// xfade失败时，必须重新授权并重新复验全部正式源资产后，才允许
// 降级为无转场拼接，避免长任务过程中失权或源文件变化仍继续产出。

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"tedna/internal/models"
)

// runAdvancedConcat 按片段配置选择最终拼接方式。
func (s *VideoEditService) runAdvancedConcat(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	segmentPaths []string,
	clips []preparedVideoClip,
	inputs []*videoEditInputSnapshot,
	outputPath string,
) error {
	if len(segmentPaths) == 0 ||
		len(segmentPaths) != len(clips) {
		return fmt.Errorf(
			"%w: 高级拼接片段为空",
			ErrVideoEditInputInvalid,
		)
	}

	if len(segmentPaths) == 1 {
		command := exec.CommandContext(
			ctx,
			"ffmpeg",
			"-y",
			"-i", segmentPaths[0],
			"-c", "copy",
			outputPath,
		)

		output, err :=
			command.CombinedOutput()
		if err != nil {
			videoEditLog.Error(
				"单片段复制失败",
				"error", err,
				"output", string(output),
			)

			return fmt.Errorf(
				"单片段处理失败: %w",
				err,
			)
		}

		return nil
	}

	hasTransition := false

	for index := 0; index < len(clips)-1; index++ {
		if clips[index].transition != "none" {
			hasTransition = true
			break
		}
	}

	if !hasTransition {
		return runVideoEditConcatCopy(
			ctx,
			segmentPaths,
			outputPath,
		)
	}

	err := runVideoEditTransitionConcat(
		ctx,
		segmentPaths,
		clips,
		outputPath,
	)
	if err == nil {
		return nil
	}

	videoEditLog.Warn(
		"转场拼接失败，准备重新授权后降级",
		"error", err,
	)

	if _, _, authErr :=
		reauthorizeVideoEditOwner(
			ctx,
			coursewareID,
			actor,
		); authErr != nil {
		return authErr
	}

	if reloadErr :=
		reloadVideoEditInputsUnchanged(
			ctx,
			coursewareID,
			inputs,
			models.CWAssetTypeVideo,
		); reloadErr != nil {
		return reloadErr
	}

	if fallbackErr :=
		runVideoEditConcatCopy(
			ctx,
			segmentPaths,
			outputPath,
		); fallbackErr != nil {
		return fmt.Errorf(
			"转场和降级拼接均失败: %w",
			fallbackErr,
		)
	}

	return nil
}

// runVideoEditConcatCopy 使用concat协议快速拼接。
func runVideoEditConcatCopy(
	ctx context.Context,
	segmentPaths []string,
	outputPath string,
) error {
	listFile, err :=
		writeVideoEditConcatList(
			segmentPaths,
		)
	if err != nil {
		return err
	}
	defer cleanupVideoEditFile(listFile)

	command := exec.CommandContext(
		ctx,
		"ffmpeg",
		"-y",
		"-f", "concat",
		"-safe", "0",
		"-i", listFile,
		"-c", "copy",
		outputPath,
	)

	output, err :=
		command.CombinedOutput()
	if err != nil {
		videoEditLog.Error(
			"无转场拼接失败",
			"error", err,
			"output", string(output),
		)

		return fmt.Errorf(
			"拼接失败: %w",
			err,
		)
	}

	return nil
}

// runVideoEditTransitionConcat 使用xfade和acrossfade拼接。
func runVideoEditTransitionConcat(
	ctx context.Context,
	segmentPaths []string,
	clips []preparedVideoClip,
	outputPath string,
) error {
	args := []string{
		"-y",
	}

	for _, path := range segmentPaths {
		args = append(
			args,
			"-i", path,
		)
	}

	filterParts := make(
		[]string,
		0,
		(len(segmentPaths)-1)*2,
	)

	offset := 0.0
	previousVideoLabel := "[0:v]"
	previousAudioLabel := "[0:a]"

	for index := 1; index < len(segmentPaths); index++ {
		previousClip := clips[index-1]

		transition :=
			previousClip.transition
		transitionDuration :=
			previousClip.transDur

		if transition == "none" {
			transition = "fade"
			transitionDuration = 0.3
		}

		offset += previousClip.duration
		xfadeOffset :=
			offset -
				transitionDuration
		if xfadeOffset < 0 {
			xfadeOffset = 0
		}

		outputVideoLabel :=
			fmt.Sprintf(
				"[v%d]",
				index,
			)
		outputAudioLabel :=
			fmt.Sprintf(
				"[a%d]",
				index,
			)

		if index ==
			len(segmentPaths)-1 {
			outputVideoLabel = "[vout]"
			outputAudioLabel = "[aout]"
		}

		filterParts = append(
			filterParts,
			fmt.Sprintf(
				"%s[%d:v]xfade=transition=%s:duration=%.3f:offset=%.3f%s",
				previousVideoLabel,
				index,
				transition,
				transitionDuration,
				xfadeOffset,
				outputVideoLabel,
			),
			fmt.Sprintf(
				"%s[%d:a]acrossfade=d=%.3f%s",
				previousAudioLabel,
				index,
				transitionDuration,
				outputAudioLabel,
			),
		)

		previousVideoLabel =
			outputVideoLabel
		previousAudioLabel =
			outputAudioLabel
		offset -= transitionDuration
	}

	filterComplex :=
		strings.Join(
			filterParts,
			";",
		)

	args = append(
		args,
		"-filter_complex", filterComplex,
		"-map", "[vout]",
		"-map", "[aout]",
		"-c:v", "libx264",
		"-preset", "fast",
		"-crf", "23",
		"-c:a", "aac",
		"-b:a", "128k",
		"-movflags", "+faststart",
		outputPath,
	)

	command := exec.CommandContext(
		ctx,
		"ffmpeg",
		args...,
	)

	output, err :=
		command.CombinedOutput()
	if err != nil {
		videoEditLog.Error(
			"转场拼接失败",
			"error", err,
			"output", string(output),
		)

		return fmt.Errorf(
			"转场拼接失败: %w",
			err,
		)
	}

	return nil
}
