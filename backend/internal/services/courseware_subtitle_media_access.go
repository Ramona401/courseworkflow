package services

// courseware_subtitle_media_access.go — TTS与硬字幕烧录公共安全底座
//
// 两条媒体能力均会产生外部成本或新文件，只允许课件作者执行。
// 本文件负责Actor授权、字幕复合读取、字幕版本复验、视频资产归属和失败文件清理。

import (
	"context"
	"fmt"
	"os"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// coursewareSubtitleSafeFileToken 生成适合文件名的安全短标识。
//
// 不直接截取固定长度，因此空ID和短ID均不会panic。
func coursewareSubtitleSafeFileToken(
	value string,
) string {
	var builder strings.Builder

	for _, char := range strings.TrimSpace(value) {
		allowed := (char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '-' ||
			char == '_'

		if allowed {
			builder.WriteRune(char)
		} else {
			builder.WriteByte('_')
		}

		if builder.Len() >= 24 {
			break
		}
	}

	token := strings.Trim(
		builder.String(),
		"_-",
	)
	if token == "" {
		return "item"
	}

	return token
}

// cleanupCoursewareSubtitleFiles 删除本轮生成但尚未提交成功的文件。
func cleanupCoursewareSubtitleFiles(
	paths []string,
) {
	seen := make(
		map[string]struct{},
		len(paths),
	)

	for _, rawPath := range paths {
		path := strings.TrimSpace(rawPath)
		if path == "" {
			continue
		}

		if _, exists := seen[path]; exists {
			continue
		}

		seen[path] = struct{}{}
		_ = os.Remove(path)
	}
}

// coursewareSubtitleStringPointerEqual 比较两个可空字符串值，而不是比较指针地址。
func coursewareSubtitleStringPointerEqual(
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

// loadOwnedCoursewareSubtitleMediaInputs 执行作者控制授权并复合加载字幕。
func loadOwnedCoursewareSubtitleMediaInputs(
	ctx context.Context,
	coursewareID string,
	subtitleID string,
	actor *CoursewareActorContext,
) (
	*CoursewareActorContext,
	*models.CoursewareSubtitle,
	error,
) {
	_, scopedActor, err :=
		(&CoursewareService{}).
			LoadCoursewareForOwnerControlMutation(
				ctx,
				coursewareID,
				actor,
			)
	if err != nil {
		return nil, nil, err
	}

	subtitle, err :=
		repository.GetCoursewareSubtitleForCourseware(
			ctx,
			coursewareID,
			subtitleID,
		)
	if err != nil {
		return nil,
			nil,
			fmt.Errorf(
				"%w: %v",
				ErrCoursewareSubtitleNotFound,
				err,
			)
	}

	return scopedActor,
		subtitle,
		nil
}

// reloadOwnedCoursewareSubtitleMediaInputs 重新授权并检查字幕数据库版本未变化。
func reloadOwnedCoursewareSubtitleMediaInputs(
	ctx context.Context,
	coursewareID string,
	subtitleID string,
	actor *CoursewareActorContext,
	expected *models.CoursewareSubtitle,
) (
	*CoursewareActorContext,
	*models.CoursewareSubtitle,
	error,
) {
	scopedActor,
		latestSubtitle,
		err :=
		loadOwnedCoursewareSubtitleMediaInputs(
			ctx,
			coursewareID,
			subtitleID,
			actor,
		)
	if err != nil {
		return nil, nil, err
	}

	if !coursewareSubtitleRevisionUnchanged(
		expected,
		latestSubtitle,
	) {
		return nil,
			nil,
			ErrCoursewareSubtitleMutationConflict
	}

	return scopedActor,
		latestSubtitle,
		nil
}

// loadCoursewareSubtitleVideoAsset 复合校验烧录源视频。
func loadCoursewareSubtitleVideoAsset(
	ctx context.Context,
	coursewareID string,
	videoAssetID string,
) (
	*models.CoursewareAsset,
	string,
	error,
) {
	asset, err :=
		repository.GetCWAssetByID(
			ctx,
			videoAssetID,
		)
	if err != nil {
		return nil,
			"",
			fmt.Errorf(
				"视频资产不存在: %w",
				err,
			)
	}

	if asset.CoursewareID != coursewareID ||
		asset.AssetType != models.CWAssetTypeVideo {
		return nil,
			"",
			ErrCoursewareSubtitleScopeTargetMismatch
	}

	sourcePath := resolveAssetPath(asset)
	if strings.TrimSpace(sourcePath) == "" {
		return nil,
			"",
			fmt.Errorf("视频文件不存在")
	}

	return asset,
		sourcePath,
		nil
}
