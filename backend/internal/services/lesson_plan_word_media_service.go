package services

// lesson_plan_word_media_service.go — Word内嵌图片提取与教案资产同步
//
// 正式确认Word教案时：
//   1. 从短时私有DOCX中读取结构JSON记录的图片关系；
//   2. 只提取被正文实际引用的安全栅格图片；
//   3. 创建lesson_plan_assets记录和教案专属图片文件；
//   4. 将图片运行转换成真正的Markdown图片；
//   5. 同步短时Word会话和lesson_plans.content_markdown；
//   6. 后续确认流程继续创建Word当前文档和不可变版本1。
//
// SVG、WMF、EMF、BMP、外部图片、缺失媒体和超大图片不会公开提取。
// 原始DOCX始终完整保留，因此这些对象不会从原Word母版中丢失。

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// Word自动提取的图片沿用现有教案资产单图5MB上限。
const maxLessonPlanWordImportedImageBytes = int64(MaxAssetFileSize)

type lessonPlanWordImportedMediaResult struct {
	SemanticMarkdown     string
	SemanticMarkdownHash string
	StructureJSON        string
	MetricsJSON          string
	WarningsJSON         string
	AssetFullPaths       []string
	ImportedImageCount   int
}

type lessonPlanWordImportedImage struct {
	RelationshipID string
	Target         string
	Label          string
	MimeType       string
	Extension      string
	Data           []byte
	AssetID        string
	URL            string
	FullPath       string
	RelativePath   string
}

// hydrateLessonPlanWordImportMediaAssets 提取图片并同步可信语义正文。
func hydrateLessonPlanWordImportMediaAssets(
	ctx context.Context,
	session *models.LessonPlanWordImportSession,
	lessonPlanID string,
	ownerID string,
	educationDomain string,
) (
	result *lessonPlanWordImportedMediaResult,
	resultErr error,
) {
	if session == nil {
		return nil, errors.New(
			"Word导入会话为空",
		)
	}

	lessonPlanID = strings.TrimSpace(
		lessonPlanID,
	)
	ownerID = strings.TrimSpace(
		ownerID,
	)
	educationDomain = strings.ToLower(
		strings.TrimSpace(
			educationDomain,
		),
	)

	if _, err := uuid.Parse(
		lessonPlanID,
	); err != nil ||
		ownerID == "" ||
		!models.IsTeachingEducationDomain(
			educationDomain,
		) {
		return nil,
			repository.ErrLessonPlanWordInputInvalid
	}

	sourcePath, err :=
		resolveLessonPlanWordPrivateStoragePath(
			session.StorageKey,
		)
	if err != nil {
		return nil, err
	}

	var document LessonPlanWordPreviewDocument

	if err := json.Unmarshal(
		[]byte(session.StructureJSON),
		&document,
	); err != nil {
		return nil, fmt.Errorf(
			"读取Word图片结构失败: %w",
			err,
		)
	}

	archiveReader, err := zip.OpenReader(
		sourcePath,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"打开Word图片压缩包失败: %w",
			err,
		)
	}
	defer archiveReader.Close()

	entries, err :=
		validateLessonPlanWordArchive(
			archiveReader.File,
		)
	if err != nil {
		return nil, err
	}

	mediaByRelationshipID := make(
		map[string]LessonPlanWordPreviewMedia,
		len(document.Media),
	)

	for _, media := range document.Media {
		mediaByRelationshipID[media.RelationshipID] =
			media
	}

	referencedRelationships :=
		make(map[string]bool)

	for _, block := range document.Blocks {
		for _, run := range block.Runs {
			relationshipID := strings.TrimSpace(
				run.RelationshipID,
			)

			if run.Kind == "image" &&
				relationshipID != "" {
				referencedRelationships[relationshipID] =
					true
			}
		}
	}

	relationshipIDs := make(
		[]string,
		0,
		len(referencedRelationships),
	)

	for relationshipID := range referencedRelationships {
		relationshipIDs = append(
			relationshipIDs,
			relationshipID,
		)
	}

	sort.Strings(
		relationshipIDs,
	)

	createdPaths := make(
		[]string,
		0,
		len(relationshipIDs),
	)

	createdAssetIDs := make(
		[]string,
		0,
		len(relationshipIDs),
	)

	completed := false

	// 本函数自身负责清理内部失败产生的资产记录和物理文件。
	// 外层导入流程只负责本函数成功返回后发生的后续失败。
	defer func() {
		if completed {
			return
		}

		cleanupContext, cancel :=
			context.WithTimeout(
				context.Background(),
				5*time.Second,
			)
		defer cancel()

		for _, assetID := range createdAssetIDs {
			if assetID == "" {
				continue
			}

			if err :=
				repository.DeleteLessonPlanAsset(
					cleanupContext,
					assetID,
				); err != nil &&
				!errors.Is(
					err,
					repository.
						ErrLessonPlanAssetNotFound,
				) {
				assetLog.Warn(
					"清理失败Word图片资产记录失败",
					"asset_id",
					assetID,
					"error",
					err,
				)
			}
		}

		for _, createdPath := range createdPaths {
			if err := os.Remove(
				createdPath,
			); err != nil &&
				!os.IsNotExist(err) {
				assetLog.Warn(
					"清理失败Word图片物理文件失败",
					"path",
					createdPath,
					"error",
					err,
				)
			}
		}
	}()

	imageURLByRelationshipID :=
		make(map[string]string)

	skippedMissing := 0
	skippedUnsupported := 0
	skippedOversized := 0

	for _, relationshipID := range relationshipIDs {
		media, exists :=
			mediaByRelationshipID[relationshipID]

		if !exists ||
			media.Missing ||
			strings.EqualFold(
				media.TargetMode,
				"External",
			) ||
			strings.TrimSpace(
				media.Target,
			) == "" {
			skippedMissing++
			continue
		}

		target := path.Clean(
			strings.TrimSpace(
				media.Target,
			),
		)

		entry, exists :=
			entries[target]

		if !exists {
			skippedMissing++
			continue
		}

		if entry.UncompressedSize64 >
			uint64(
				maxLessonPlanWordImportedImageBytes,
			) {
			skippedOversized++
			continue
		}

		data, readErr :=
			readLessonPlanWordZipEntry(
				entries,
				target,
				maxLessonPlanWordImportedImageBytes,
			)
		if readErr != nil {
			return nil, readErr
		}

		mimeType,
			extension,
			supported :=
			detectLessonPlanWordImportedImage(
				data,
			)

		if !supported {
			skippedUnsupported++
			continue
		}

		importedImage :=
			lessonPlanWordImportedImage{
				RelationshipID: relationshipID,
				Target:         target,
				Label: sanitizeLessonPlanWordImageLabel(
					path.Base(target),
				),
				MimeType:  mimeType,
				Extension: extension,
				Data:      data,
			}

		if err :=
			storeLessonPlanWordImportedImage(
				ctx,
				lessonPlanID,
				ownerID,
				&importedImage,
			); err != nil {
			return nil, err
		}

		createdPaths = append(
			createdPaths,
			importedImage.FullPath,
		)

		createdAssetIDs = append(
			createdAssetIDs,
			importedImage.AssetID,
		)

		imageURLByRelationshipID[relationshipID] = importedImage.URL
	}

	replacementCount :=
		applyLessonPlanWordImportedImageURLs(
			&document,
			imageURLByRelationshipID,
		)

	if len(imageURLByRelationshipID) > 0 &&
		replacementCount == 0 {
		return nil, errors.New(
			"Word图片已经提取，但未能写入语义正文",
		)
	}

	warnings := make(
		[]LessonPlanWordPreviewWarning,
		0,
	)

	if strings.TrimSpace(
		session.WarningsJSON,
	) != "" {
		if err := json.Unmarshal(
			[]byte(
				session.WarningsJSON,
			),
			&warnings,
		); err != nil {
			return nil, fmt.Errorf(
				"读取Word图片告警失败: %w",
				err,
			)
		}
	}

	if skippedMissing > 0 {
		warnings = append(
			warnings,
			LessonPlanWordPreviewWarning{
				Code:    "word_image_missing",
				Message: "部分Word图片关系缺失或为外部图片，网页继续显示占位符",
				Count:   skippedMissing,
			},
		)
	}

	if skippedUnsupported > 0 {
		warnings = append(
			warnings,
			LessonPlanWordPreviewWarning{
				Code:    "word_image_unsupported",
				Message: "部分图片格式不适合网页安全展示，原对象仍保留在DOCX中",
				Count:   skippedUnsupported,
			},
		)
	}

	if skippedOversized > 0 {
		warnings = append(
			warnings,
			LessonPlanWordPreviewWarning{
				Code:    "word_image_oversized",
				Message: "部分单张图片超过5MB，网页继续显示占位符",
				Count:   skippedOversized,
			},
		)
	}

	semanticMarkdown := strings.TrimSpace(
		buildLessonPlanWordSemanticMarkdown(
			document,
		),
	)

	if semanticMarkdown == "" {
		return nil, errors.New(
			"提取Word图片后语义正文为空",
		)
	}

	structureBytes, err :=
		json.Marshal(document)
	if err != nil {
		return nil, fmt.Errorf(
			"序列化Word图片结构失败: %w",
			err,
		)
	}

	warningBytes, err :=
		json.Marshal(warnings)
	if err != nil {
		return nil, fmt.Errorf(
			"序列化Word图片告警失败: %w",
			err,
		)
	}

	semanticHash := sha256.Sum256(
		[]byte(
			semanticMarkdown,
		),
	)

	result =
		&lessonPlanWordImportedMediaResult{
			SemanticMarkdown: semanticMarkdown,
			SemanticMarkdownHash: hex.EncodeToString(
				semanticHash[:],
			),
			StructureJSON: string(structureBytes),
			MetricsJSON:   session.MetricsJSON,
			WarningsJSON:  string(warningBytes),
			AssetFullPaths: append(
				[]string(nil),
				createdPaths...,
			),
			ImportedImageCount: replacementCount,
		}

	if replacementCount > 0 {
		if err :=
			repository.
				SyncLessonPlanWordImportMedia(
					ctx,
					repository.
						SyncLessonPlanWordImportMediaInput{
						ImportSessionID:          session.ID,
						LessonPlanID:             lessonPlanID,
						OwnerID:                  ownerID,
						EducationDomain:          educationDomain,
						ExpectedFileSHA256:       session.FileSHA256,
						ExpectedSemanticMarkdown: session.SemanticMarkdown,
						StructureJSON:            result.StructureJSON,
						SemanticMarkdown:         result.SemanticMarkdown,
						SemanticMarkdownHash:     result.SemanticMarkdownHash,
						MetricsJSON:              result.MetricsJSON,
						WarningsJSON:             result.WarningsJSON,
					},
				); err != nil {
			return nil, err
		}

		session.StructureJSON =
			result.StructureJSON
		session.SemanticMarkdown =
			result.SemanticMarkdown
		session.SemanticMarkdownHash =
			result.SemanticMarkdownHash
		session.WarningsJSON =
			result.WarningsJSON
	}

	completed = true

	return result, nil
}
