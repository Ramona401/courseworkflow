package services

// courseware_preset_style_thumbnail.go — 系统预设画风真实缩略图生成服务
//
// 设计目标：
//   - 系统操作员一次性生成全部预设画风真实缩略图；
//   - 教师打开画风选择器时直接查看，不等待、不重复调用图片模型；
//   - 每种画风使用相同主体和构图，仅改变艺术语言，便于横向比较；
//   - 图片永久保存到现有课件资产公开目录；
//   - manifest.json记录稳定键、名称、真实文件名、MIME和URL；
//   - 默认复用已存在文件，支持断点恢复和避免重复消耗；
//   - --force模式才重新生成已有缩略图。
//
// 保存位置：
//
//	/www/wwwroot/tedna/uploads/courseware-assets/style-presets/
//
// 浏览器URL：
//
//	/uploads/courseware-assets/style-presets/manifest.json
//
// 本服务只生成系统展示缩略图，不创建courseware_assets记录，
// 不绑定具体课程、页面、用户或学校，也不写入课程锚点。

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"tedna/internal/ai"
)

const (
	cwPresetThumbnailDirName =
		"style-presets"

	cwPresetThumbnailManifestName =
		"manifest.json"

	// 2560×1440正好满足当前图片网关的最小总像素要求，
	// 同时保持适合卡片缩略图的16:9横向比例。
	cwPresetThumbnailImageSize =
		"2560x1440"

	cwPresetThumbnailMaxBytes int64 =
		24 * 1024 * 1024
)

// CoursewarePresetStyleThumbnailItem 是manifest中的单个缩略图。
type CoursewarePresetStyleThumbnailItem struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	URL      string `json:"url"`
	FileName string `json:"file_name"`
	MimeType string `json:"mime_type"`
	FileSize int64  `json:"file_size"`
}

// CoursewarePresetStyleThumbnailManifest 是前端读取的静态缩略图清单。
type CoursewarePresetStyleThumbnailManifest struct {
	Version     int                                      `json:"version"`
	GeneratedAt string                                   `json:"generated_at"`
	ImageSize   string                                   `json:"image_size"`
	Styles      []CoursewarePresetStyleThumbnailItem     `json:"styles"`
}

// GenerateCoursewarePresetStyleThumbnails 生成或恢复全部预设缩略图。
//
// force=false：
//   - 若某个稳定键已经存在合法jpg/png/webp文件，则直接复用；
//   - 只为缺失键调用图片API。
//
// force=true：
//   - 每个键重新调用图片API；
//   - 新图片成功落盘后才替换旧文件；
//   - 单张失败不会提前删除旧文件。
func GenerateCoursewarePresetStyleThumbnails(
	ctx context.Context,
	imageConfig *ai.ImageConfig,
	force bool,
) (
	*CoursewarePresetStyleThumbnailManifest,
	error,
) {
	if imageConfig == nil {
		return nil, fmt.Errorf(
			"系统预设缩略图生成配置为空",
		)
	}

	styles :=
		ListCoursewarePresetStyleDescriptors()

	if len(styles) == 0 {
		return nil, fmt.Errorf(
			"没有可生成的系统预设画风",
		)
	}

	outputDir :=
		filepath.Join(
			CWAssetUploadDir,
			cwPresetThumbnailDirName,
		)

	if err :=
		os.MkdirAll(
			outputDir,
			0755,
		); err != nil {
		return nil, fmt.Errorf(
			"创建系统预设缩略图目录失败: %w",
			err,
		)
	}

	items := make(
		[]CoursewarePresetStyleThumbnailItem,
		0,
		len(styles),
	)

	for index, style := range styles {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf(
				"系统预设缩略图生成已取消: %w",
				ctx.Err(),
			)
		default:
		}

		if !force {
			if existing, ok :=
				findExistingPresetStyleThumbnail(
					outputDir,
					style,
				); ok {
				items = append(
					items,
					existing,
				)

				cwAssetLog.Info(
					"复用已有系统预设画风缩略图",
					"preset_style_key", style.Key,
					"preset_style_label", style.Label,
					"progress", fmt.Sprintf(
						"%d/%d",
						index+1,
						len(styles),
					),
					"url", existing.URL,
				)

				continue
			}
		}

		cwAssetLog.Info(
			"开始生成系统预设画风缩略图",
			"preset_style_key", style.Key,
			"preset_style_label", style.Label,
			"progress", fmt.Sprintf(
				"%d/%d",
				index+1,
				len(styles),
			),
			"force", force,
		)

		item, err :=
			generateOnePresetStyleThumbnail(
				ctx,
				imageConfig,
				outputDir,
				style,
			)
		if err != nil {
			return nil, fmt.Errorf(
				"生成“%s”缩略图失败: %w",
				style.Label,
				err,
			)
		}

		items = append(
			items,
			*item,
		)
	}

	manifest :=
		&CoursewarePresetStyleThumbnailManifest{
			Version: 1,
			GeneratedAt: time.Now().
				Format(
					time.RFC3339,
				),
			ImageSize:
				cwPresetThumbnailImageSize,
			Styles: items,
		}

	if err :=
		writePresetStyleThumbnailManifest(
			outputDir,
			manifest,
		); err != nil {
		return nil, err
	}

	return manifest, nil
}

// generateOnePresetStyleThumbnail 为一种画风生成真实缩略图。
//
// 每种画风最多尝试两次，应对偶发网关或临时下载失败。
func generateOnePresetStyleThumbnail(
	ctx context.Context,
	imageConfig *ai.ImageConfig,
	outputDir string,
	style CoursewarePresetStyleDescriptor,
) (
	*CoursewarePresetStyleThumbnailItem,
	error,
) {
	prompt :=
		buildPresetStyleThumbnailPrompt(
			style,
		)

	var lastErr error

	for attempt := 1; attempt <= 2; attempt++ {
		result, generationErr :=
			ai.GenerateImage(
				ctx,
				imageConfig,
				prompt,
				cwPresetThumbnailImageSize,
				1,
				"",
				nil,
			)

		if generationErr != nil {
			lastErr =
				generationErr
		} else if result == nil ||
			len(result.URLs) == 0 ||
			strings.TrimSpace(
				result.URLs[0],
			) == "" {
			lastErr =
				fmt.Errorf(
					"图片网关未返回有效地址",
				)
		} else {
			item, saveErr :=
				downloadPresetStyleThumbnail(
					ctx,
					outputDir,
					style,
					result.URLs[0],
				)

			if saveErr == nil {
				return item, nil
			}

			lastErr =
				saveErr
		}

		if attempt < 2 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(
				2 * time.Second,
			):
			}
		}
	}

	if lastErr == nil {
		lastErr =
			fmt.Errorf(
				"未知缩略图生成错误",
			)
	}

	return nil, lastErr
}

// buildPresetStyleThumbnailPrompt 构造统一对比场景。
//
// 所有画风必须使用相同主体、视角和构图，
// 否则老师看到的是主体差异，而不是画风差异。
func buildPresetStyleThumbnailPrompt(
	style CoursewarePresetStyleDescriptor,
) string {
	var builder strings.Builder

	builder.WriteString(
		"请生成一张用于教师选择课件插图画风的16:9横向样板缩略图。",
	)
	builder.WriteString(
		"十种画风必须使用完全一致的场景和构图：",
	)
	builder.WriteString(
		"正视中景，一张整洁的浅色学习桌位于画面中央；",
	)
	builder.WriteString(
		"桌上从左到右摆放一个小型地球仪、一本打开但没有任何可读文字的书、",
	)
	builder.WriteString(
		"三个基础几何积木和一株小型绿植；",
	)
	builder.WriteString(
		"背景是一面简洁的知识展示墙，只使用抽象形状，不能出现文字。",
	)
	builder.WriteString(
		"主体数量、位置、视角、景别和留白保持稳定，重点只展示画风差异。",
	)
	builder.WriteString(
		"\n\n本张缩略图采用的艺术语言：",
	)
	builder.WriteString(
		style.ArtText,
	)
	builder.WriteString(
		"。\n\n画面要求：构图清楚、主体完整、适合缩小后辨认；",
	)
	builder.WriteString(
		"禁止人物、可读文字、字母、数字、Logo、品牌标识、签名和水印；",
	)
	builder.WriteString(
		"禁止边框、拼贴、分屏和多方案对比；只输出一张完整画面。",
	)

	return builder.String()
}

// downloadPresetStyleThumbnail 下载、验签并原子保存缩略图。
func downloadPresetStyleThumbnail(
	ctx context.Context,
	outputDir string,
	style CoursewarePresetStyleDescriptor,
	remoteURL string,
) (
	*CoursewarePresetStyleThumbnailItem,
	error,
) {
	remoteURL =
		strings.TrimSpace(
			remoteURL,
		)

	if remoteURL == "" {
		return nil, fmt.Errorf(
			"缩略图远程地址为空",
		)
	}

	request, err :=
		http.NewRequestWithContext(
			ctx,
			http.MethodGet,
			remoteURL,
			nil,
		)
	if err != nil {
		return nil, fmt.Errorf(
			"创建缩略图下载请求失败: %w",
			err,
		)
	}

	client :=
		&http.Client{
			Timeout: 60 * time.Second,
		}

	response, err :=
		client.Do(
			request,
		)
	if err != nil {
		return nil, fmt.Errorf(
			"下载缩略图失败: %w",
			err,
		)
	}
	defer response.Body.Close()

	if response.StatusCode !=
		http.StatusOK {
		return nil, fmt.Errorf(
			"下载缩略图HTTP错误: %d",
			response.StatusCode,
		)
	}

	temporaryFile, err :=
		os.CreateTemp(
			outputDir,
			"."+
				style.Key+
				"-*.part",
		)
	if err != nil {
		return nil, fmt.Errorf(
			"创建缩略图临时文件失败: %w",
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

	written, copyErr :=
		io.Copy(
			temporaryFile,
			io.LimitReader(
				response.Body,
				cwPresetThumbnailMaxBytes+1,
			),
		)

	closeErr :=
		temporaryFile.Close()

	if copyErr != nil {
		return nil, fmt.Errorf(
			"写入缩略图失败: %w",
			copyErr,
		)
	}

	if closeErr != nil {
		return nil, fmt.Errorf(
			"关闭缩略图临时文件失败: %w",
			closeErr,
		)
	}

	if written <= 0 {
		return nil, fmt.Errorf(
			"图片网关返回空缩略图",
		)
	}

	if written >
		cwPresetThumbnailMaxBytes {
		return nil, fmt.Errorf(
			"缩略图超过24MB限制",
		)
	}

	mimeType, err :=
		detectGeneratedImageMIMEType(
			temporaryPath,
			response.Header.Get(
				"Content-Type",
			),
		)
	if err != nil {
		return nil, err
	}

	extension, err :=
		generatedImageExtension(
			mimeType,
		)
	if err != nil {
		return nil, err
	}

	fileName :=
		style.Key +
			extension

	finalPath :=
		filepath.Join(
			outputDir,
			fileName,
		)

	if err :=
		os.Rename(
			temporaryPath,
			finalPath,
		); err != nil {
		return nil, fmt.Errorf(
			"确认缩略图文件失败: %w",
			err,
		)
	}

	removeTemporary =
		false

	// 新文件已经安全落盘后，删除同一键的其它旧扩展名版本。
	removeOtherPresetThumbnailVariants(
		outputDir,
		style.Key,
		fileName,
	)

	fileInfo, err :=
		os.Stat(
			finalPath,
		)
	if err != nil {
		return nil, fmt.Errorf(
			"读取缩略图文件信息失败: %w",
			err,
		)
	}

	if !fileInfo.Mode().
		IsRegular() ||
		fileInfo.Size() <= 0 {
		return nil, fmt.Errorf(
			"缩略图文件状态异常",
		)
	}

	url :=
		CWAssetURLPrefix +
			cwPresetThumbnailDirName +
			"/" +
			fileName

	return &CoursewarePresetStyleThumbnailItem{
		Key:      style.Key,
		Label:    style.Label,
		URL:      url,
		FileName: fileName,
		MimeType: mimeType,
		FileSize: fileInfo.Size(),
	}, nil
}

// findExistingPresetStyleThumbnail 查找可复用的稳定文件。
func findExistingPresetStyleThumbnail(
	outputDir string,
	style CoursewarePresetStyleDescriptor,
) (
	CoursewarePresetStyleThumbnailItem,
	bool,
) {
	candidates :=
		[]struct {
			Extension string
			MimeType  string
		}{
			{
				Extension: ".jpg",
				MimeType:  "image/jpeg",
			},
			{
				Extension: ".png",
				MimeType:  "image/png",
			},
			{
				Extension: ".webp",
				MimeType:  "image/webp",
			},
		}

	for _, candidate :=
		range candidates {
		fileName :=
			style.Key +
				candidate.Extension

		fullPath :=
			filepath.Join(
				outputDir,
				fileName,
			)

		fileInfo, err :=
			os.Stat(
				fullPath,
			)

		if err != nil ||
			!fileInfo.Mode().
				IsRegular() ||
			fileInfo.Size() <= 0 {
			continue
		}

		return CoursewarePresetStyleThumbnailItem{
			Key:      style.Key,
			Label:    style.Label,
			URL: CWAssetURLPrefix +
				cwPresetThumbnailDirName +
				"/" +
				fileName,
			FileName: fileName,
			MimeType: candidate.MimeType,
			FileSize: fileInfo.Size(),
		}, true
	}

	return CoursewarePresetStyleThumbnailItem{}, false
}

// removeOtherPresetThumbnailVariants 删除同键的旧格式文件。
func removeOtherPresetThumbnailVariants(
	outputDir string,
	styleKey string,
	keepFileName string,
) {
	for _, extension :=
		range []string{
			".jpg",
			".png",
			".webp",
		} {
		fileName :=
			styleKey +
				extension

		if fileName ==
			keepFileName {
			continue
		}

		_ =
			os.Remove(
				filepath.Join(
					outputDir,
					fileName,
				),
			)
	}
}

// writePresetStyleThumbnailManifest 原子写入manifest.json。
func writePresetStyleThumbnailManifest(
	outputDir string,
	manifest *CoursewarePresetStyleThumbnailManifest,
) error {
	if manifest == nil {
		return fmt.Errorf(
			"系统预设缩略图清单为空",
		)
	}

	content, err :=
		json.MarshalIndent(
			manifest,
			"",
			"  ",
		)
	if err != nil {
		return fmt.Errorf(
			"序列化系统预设缩略图清单失败: %w",
			err,
		)
	}

	temporaryFile, err :=
		os.CreateTemp(
			outputDir,
			".manifest-*.json.part",
		)
	if err != nil {
		return fmt.Errorf(
			"创建缩略图清单临时文件失败: %w",
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

	if _, err :=
		temporaryFile.Write(
			content,
		); err != nil {
		_ =
			temporaryFile.Close()

		return fmt.Errorf(
			"写入缩略图清单失败: %w",
			err,
		)
	}

	if err :=
		temporaryFile.Close(); err != nil {
		return fmt.Errorf(
			"关闭缩略图清单临时文件失败: %w",
			err,
		)
	}

	finalPath :=
		filepath.Join(
			outputDir,
			cwPresetThumbnailManifestName,
		)

	if err :=
		os.Rename(
			temporaryPath,
			finalPath,
		); err != nil {
		return fmt.Errorf(
			"确认缩略图清单文件失败: %w",
			err,
		)
	}

	removeTemporary =
		false

	return nil
}
