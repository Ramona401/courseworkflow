package services

// lesson_plan_word_media_helpers.go — Word图片资产辅助能力
//
// 本文件承载：
//   - 图片文件和lesson_plan_assets记录创建；
//   - 按Word运行顺序重建Markdown图片；
//   - 图片真实格式识别和alt文本安全化；
//   - 导入失败后的图片物理文件清理。
//
// 主导入编排保留在lesson_plan_word_media_service.go，
// 避免单文件持续膨胀并保持职责清晰。

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"tedna/internal/models"
	"tedna/internal/repository"
)

func storeLessonPlanWordImportedImage(
	ctx context.Context,
	lessonPlanID string,
	ownerID string,
	image *lessonPlanWordImportedImage,
) error {
	if image == nil ||
		len(image.Data) == 0 {
		return ErrAssetFileInvalid
	}

	planDirectory := filepath.Clean(
		filepath.Join(
			AssetUploadDir,
			lessonPlanID,
		),
	)

	rootDirectory := filepath.Clean(
		AssetUploadDir,
	)

	if planDirectory == rootDirectory ||
		!strings.HasPrefix(
			planDirectory,
			rootDirectory+
				string(os.PathSeparator),
		) {
		return errors.New(
			"Word图片资产目录越界",
		)
	}

	if err := os.MkdirAll(
		planDirectory,
		0o755,
	); err != nil {
		return fmt.Errorf(
			"创建Word图片资产目录失败: %w",
			err,
		)
	}

	dataHash := sha256.Sum256(
		image.Data,
	)

	storedName := fmt.Sprintf(
		"word_%s_%s%s",
		hex.EncodeToString(
			dataHash[:8],
		),
		uuid.NewString(),
		image.Extension,
	)

	fullPath := filepath.Join(
		planDirectory,
		storedName,
	)

	file, err := os.OpenFile(
		fullPath,
		os.O_WRONLY|
			os.O_CREATE|
			os.O_EXCL,
		0o644,
	)
	if err != nil {
		return fmt.Errorf(
			"创建Word图片资产文件失败: %w",
			err,
		)
	}

	writeErr := error(nil)

	if _, err := file.Write(
		image.Data,
	); err != nil {
		writeErr = err
	} else if err := file.Sync(); err != nil {
		writeErr = err
	}

	closeErr := file.Close()

	if writeErr != nil {
		_ = os.Remove(fullPath)

		return fmt.Errorf(
			"保存Word图片资产失败: %w",
			writeErr,
		)
	}

	if closeErr != nil {
		_ = os.Remove(fullPath)

		return fmt.Errorf(
			"关闭Word图片资产失败: %w",
			closeErr,
		)
	}

	relativePath := filepath.ToSlash(
		filepath.Join(
			lessonPlanID,
			storedName,
		),
	)

	asset := &models.LessonPlanAsset{
		LessonPlanID: lessonPlanID,
		UploaderID:   ownerID,
		AssetType:    models.AssetTypeImage,
		FileName:     path.Base(image.Target),
		FilePath:     relativePath,
		FileSize:     int64(len(image.Data)),
		MimeType:     image.MimeType,
		AltText:      image.Label,
	}

	if err :=
		repository.CreateLessonPlanAsset(
			ctx,
			asset,
		); err != nil {
		_ = os.Remove(fullPath)
		return err
	}

	image.AssetID =
		asset.ID
	image.FullPath =
		fullPath
	image.RelativePath =
		relativePath
	image.URL =
		AssetURLPrefix +
			relativePath

	return nil
}

// applyLessonPlanWordImportedImageURLs 根据结构化运行重新构建含图片段落。
//
// 直接按运行顺序重建，避免同名图片占位符使用ReplaceAll时串图。
func applyLessonPlanWordImportedImageURLs(
	document *LessonPlanWordPreviewDocument,
	imageURLByRelationshipID map[string]string,
) int {
	if document == nil ||
		len(imageURLByRelationshipID) == 0 {
		return 0
	}

	replacementCount := 0

	for blockIndex := range document.Blocks {
		block :=
			&document.Blocks[blockIndex]

		hasImageRun := false

		for _, run := range block.Runs {
			if run.Kind == "image" {
				hasImageRun = true
				break
			}
		}

		if !hasImageRun {
			continue
		}

		markdownParts := make(
			[]string,
			0,
			len(block.Runs),
		)

		for _, run := range block.Runs {
			switch run.Kind {
			case "text":
				rendered :=
					renderLessonPlanWordRunMarkdown(
						run,
					)

				if rendered != "" {
					markdownParts = append(
						markdownParts,
						rendered,
					)
				}

			case "image":
				label := "图片"

				if strings.TrimSpace(
					run.MediaTarget,
				) != "" {
					label =
						sanitizeLessonPlanWordImageLabel(
							path.Base(
								run.MediaTarget,
							),
						)
				}

				imageURL := strings.TrimSpace(
					imageURLByRelationshipID[run.RelationshipID],
				)

				if imageURL == "" {
					markdownParts = append(
						markdownParts,
						"[图片："+label+"]",
					)
					continue
				}

				markdownParts = append(
					markdownParts,
					"!["+
						label+
						"]("+
						imageURL+
						")",
				)

				replacementCount++

			case "formula":
				markdownParts = append(
					markdownParts,
					"{{"+
						strings.ToUpper(
							run.FormulaID,
						)+
						":"+
						run.Text+
						"}}",
				)
			}
		}

		block.Markdown = strings.TrimSpace(
			strings.Join(
				markdownParts,
				"",
			),
		)
	}

	return replacementCount
}

func detectLessonPlanWordImportedImage(
	data []byte,
) (
	mimeType string,
	extension string,
	supported bool,
) {
	if len(data) == 0 {
		return "", "", false
	}

	detected := strings.ToLower(
		strings.TrimSpace(
			strings.Split(
				http.DetectContentType(
					data,
				),
				";",
			)[0],
		),
	)

	switch detected {
	case "image/png":
		return detected, ".png", true
	case "image/jpeg":
		return detected, ".jpg", true
	case "image/gif":
		return detected, ".gif", true
	case "image/webp":
		return detected, ".webp", true
	default:
		return "", "", false
	}
}

func sanitizeLessonPlanWordImageLabel(
	value string,
) string {
	value = strings.TrimSpace(
		value,
	)

	value = strings.NewReplacer(
		"[", "_",
		"]", "_",
		"\r", " ",
		"\n", " ",
	).Replace(value)

	value = strings.TrimSpace(
		value,
	)

	if value == "" {
		value = "Word图片"
	}

	runes := []rune(value)

	if len(runes) > 100 {
		value = string(
			runes[:100],
		)
	}

	return value
}

// removeLessonPlanWordImportedAssetFiles 在教案数据库补偿删除成功后，
// 删除本次Word导入创建的教案图片物理文件。
func removeLessonPlanWordImportedAssetFiles(
	lessonPlanID string,
	fullPaths []string,
) error {
	if len(fullPaths) == 0 {
		return nil
	}

	allowedDirectory := filepath.Clean(
		filepath.Join(
			AssetUploadDir,
			lessonPlanID,
		),
	)

	var combinedErr error

	for _, fullPath := range fullPaths {
		cleanPath := filepath.Clean(
			strings.TrimSpace(
				fullPath,
			),
		)

		if cleanPath == "" {
			continue
		}

		if cleanPath == allowedDirectory ||
			!strings.HasPrefix(
				cleanPath,
				allowedDirectory+
					string(os.PathSeparator),
			) {
			combinedErr =
				combineLessonPlanWordCleanupError(
					combinedErr,
					"拒绝删除教案目录之外的Word图片资产",
					errors.New(cleanPath),
				)
			continue
		}

		if err := os.Remove(
			cleanPath,
		); err != nil &&
			!os.IsNotExist(err) {
			combinedErr =
				combineLessonPlanWordCleanupError(
					combinedErr,
					"删除Word图片资产失败",
					err,
				)
		}
	}

	if combinedErr == nil {
		_ = os.Remove(
			allowedDirectory,
		)
	}

	return combinedErr
}
