package main

// public_files.go — 系统预设画风公开文件和轻量卡片图处理。
//
// 高清原图继续作为正式风格锚点来源。
// 教师选择弹窗使用640×360的轻量JPEG卡片图：
//
//	{preset_key}_card.jpg
//
// 优化策略：
//   - 从已生成高清图本地裁切缩放，不调用图片API；
//   - 中心裁切为16:9，避免拉伸变形；
//   - JPEG质量78，适合178px宽的界面卡片；
//   - 卡片图比高清原图新时直接复用；
//   - 所有公开文件权限统一为0644；
//   - PNG和JPEG原图均支持；
//   - 若未来出现WebP等当前标准库无法解码的格式，命令返回错误，
//     防止界面误以为优化已经完成。

import (
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"

	"tedna/internal/services"
)

const (
	presetThumbnailPublicDirectory =
		"/www/wwwroot/tedna/uploads/courseware-assets/style-presets"

	presetThumbnailManifestFileName =
		"manifest.json"

	presetCardWidth =
		640

	presetCardHeight =
		360

	presetCardJPEGQuality =
		78
)

// ensurePresetThumbnailPublicFiles 校验公开原图并生成轻量卡片图。
func ensurePresetThumbnailPublicFiles(
	manifest *services.CoursewarePresetStyleThumbnailManifest,
) error {
	if manifest == nil {
		return fmt.Errorf(
			"缩略图清单为空",
		)
	}

	if len(manifest.Styles) == 0 {
		return fmt.Errorf(
			"缩略图清单没有任何画风",
		)
	}

	if err :=
		os.MkdirAll(
			presetThumbnailPublicDirectory,
			0755,
		); err != nil {
		return fmt.Errorf(
			"创建缩略图公开目录失败: %w",
			err,
		)
	}

	if err :=
		os.Chmod(
			presetThumbnailPublicDirectory,
			0755,
		); err != nil {
		return fmt.Errorf(
			"设置缩略图公开目录权限失败: %w",
			err,
		)
	}

	for _, item :=
		range manifest.Styles {
		fileName :=
			strings.TrimSpace(
				item.FileName,
			)

		styleKey :=
			strings.TrimSpace(
				item.Key,
			)

		if fileName == "" ||
			styleKey == "" {
			return fmt.Errorf(
				"缩略图清单存在空文件名或空画风键",
			)
		}

		if filepath.Base(
			fileName,
		) != fileName {
			return fmt.Errorf(
				"画风%s的缩略图文件名不安全",
				styleKey,
			)
		}

		sourcePath :=
			filepath.Join(
				presetThumbnailPublicDirectory,
				fileName,
			)

		if err :=
			validatePresetThumbnailFile(
				sourcePath,
				styleKey,
			); err != nil {
			return err
		}

		if err :=
			os.Chmod(
				sourcePath,
				0644,
			); err != nil {
			return fmt.Errorf(
				"设置画风%s高清图读取权限失败: %w",
				styleKey,
				err,
			)
		}

		if err :=
			ensurePresetCardPreview(
				sourcePath,
				styleKey,
			); err != nil {
			return fmt.Errorf(
				"生成画风%s轻量卡片图失败: %w",
				styleKey,
				err,
			)
		}
	}

	manifestPath :=
		filepath.Join(
			presetThumbnailPublicDirectory,
			presetThumbnailManifestFileName,
		)

	if err :=
		validatePresetThumbnailFile(
			manifestPath,
			"manifest",
		); err != nil {
		return err
	}

	if err :=
		os.Chmod(
			manifestPath,
			0644,
		); err != nil {
		return fmt.Errorf(
			"设置缩略图清单读取权限失败: %w",
			err,
		)
	}

	return nil
}

// ensurePresetCardPreview 创建或复用640×360轻量JPEG卡片图。
func ensurePresetCardPreview(
	sourcePath string,
	styleKey string,
) error {
	sourceInfo, err :=
		os.Stat(
			sourcePath,
		)

	if err != nil {
		return fmt.Errorf(
			"读取高清图信息失败: %w",
			err,
		)
	}

	cardFileName :=
		styleKey +
			"_card.jpg"

	cardPath :=
		filepath.Join(
			presetThumbnailPublicDirectory,
			cardFileName,
		)

	if cardInfo, statErr :=
		os.Stat(
			cardPath,
		); statErr == nil &&
		cardInfo.Mode().
			IsRegular() &&
		cardInfo.Size() > 0 &&
		!cardInfo.ModTime().
			Before(
				sourceInfo.ModTime(),
			) {
		return os.Chmod(
			cardPath,
			0644,
		)
	}

	sourceFile, err :=
		os.Open(
			sourcePath,
		)

	if err != nil {
		return fmt.Errorf(
			"打开高清图失败: %w",
			err,
		)
	}

	sourceImage, _, decodeErr :=
		image.Decode(
			sourceFile,
		)

	closeErr :=
		sourceFile.Close()

	if decodeErr != nil {
		return fmt.Errorf(
			"解码高清图失败: %w",
			decodeErr,
		)
	}

	if closeErr != nil {
		return fmt.Errorf(
			"关闭高清图失败: %w",
			closeErr,
		)
	}

	cardImage :=
		resizePresetThumbnailCenterCrop(
			sourceImage,
			presetCardWidth,
			presetCardHeight,
		)

	temporaryFile, err :=
		os.CreateTemp(
			presetThumbnailPublicDirectory,
			"."+
				styleKey+
				"_card_*.jpg.part",
		)

	if err != nil {
		return fmt.Errorf(
			"创建卡片图临时文件失败: %w",
			err,
		)
	}

	temporaryPath :=
		temporaryFile.Name()

	removeTemporary :=
		true

	defer func() {
		if removeTemporary {
			_ =
				os.Remove(
					temporaryPath,
				)
		}
	}()

	if err :=
		jpeg.Encode(
			temporaryFile,
			cardImage,
			&jpeg.Options{
				Quality:
					presetCardJPEGQuality,
			},
		); err != nil {
		_ =
			temporaryFile.Close()

		return fmt.Errorf(
			"编码卡片图失败: %w",
			err,
		)
	}

	if err :=
		temporaryFile.Close(); err != nil {
		return fmt.Errorf(
			"关闭卡片图临时文件失败: %w",
			err,
		)
	}

	if err :=
		os.Rename(
			temporaryPath,
			cardPath,
		); err != nil {
		return fmt.Errorf(
			"确认卡片图文件失败: %w",
			err,
		)
	}

	removeTemporary = false

	if err :=
		os.Chmod(
			cardPath,
			0644,
		); err != nil {
		return fmt.Errorf(
			"设置卡片图读取权限失败: %w",
			err,
		)
	}

	return validatePresetThumbnailFile(
		cardPath,
		styleKey+"_card",
	)
}

// resizePresetThumbnailCenterCrop 以中心裁切方式生成指定16:9图片。
//
// 这里采用确定性的最近邻采样。
// 高清图缩小到640×360时速度快、依赖为零，并足以满足小卡片预览。
func resizePresetThumbnailCenterCrop(
	source image.Image,
	targetWidth int,
	targetHeight int,
) *image.RGBA {
	sourceBounds :=
		source.Bounds()

	sourceWidth :=
		sourceBounds.Dx()

	sourceHeight :=
		sourceBounds.Dy()

	cropWidth :=
		sourceWidth

	cropHeight :=
		sourceHeight

	targetRatio :=
		float64(targetWidth) /
			float64(targetHeight)

	sourceRatio :=
		float64(sourceWidth) /
			float64(sourceHeight)

	if sourceRatio > targetRatio {
		cropWidth =
			int(
				float64(
					sourceHeight,
				) *
					targetRatio,
			)
	} else if sourceRatio <
		targetRatio {
		cropHeight =
			int(
				float64(
					sourceWidth,
				) /
					targetRatio,
			)
	}

	cropLeft :=
		(sourceWidth -
			cropWidth) /
			2

	cropTop :=
		(sourceHeight -
			cropHeight) /
			2

	target :=
		image.NewRGBA(
			image.Rect(
				0,
				0,
				targetWidth,
				targetHeight,
			),
		)

	for y := 0; y < targetHeight; y++ {
		sourceY :=
			sourceBounds.Min.Y +
				cropTop +
				y*cropHeight/
					targetHeight

		for x := 0; x < targetWidth; x++ {
			sourceX :=
				sourceBounds.Min.X +
					cropLeft +
					x*cropWidth/
						targetWidth

			target.Set(
				x,
				y,
				source.At(
					sourceX,
					sourceY,
				),
			)
		}
	}

	return target
}

// validatePresetThumbnailFile 校验文件存在、非空且为普通文件。
func validatePresetThumbnailFile(
	fullPath string,
	label string,
) error {
	fileInfo, err :=
		os.Stat(
			fullPath,
		)

	if err != nil {
		return fmt.Errorf(
			"读取%s文件失败: %w",
			label,
			err,
		)
	}

	if !fileInfo.Mode().
		IsRegular() {
		return fmt.Errorf(
			"%s不是普通文件",
			label,
		)
	}

	if fileInfo.Size() <= 0 {
		return fmt.Errorf(
			"%s文件为空",
			label,
		)
	}

	return nil
}
