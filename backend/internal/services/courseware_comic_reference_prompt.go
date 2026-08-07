package services

// courseware_comic_reference_prompt.go — 参考资源AI上下文装配
//
// 本文件只供知识点漫画AI规划服务内部调用。
//
// 容量规则：
//   - 最多读取8项参考资源；
//   - 每项最多6000个Unicode字符；
//   - 所有参考资源合计最多24000个Unicode字符；
//   - 有压缩摘要时优先使用摘要；
//   - 没有摘要时才使用正文；
//   - 图片在文字规划阶段只提供标题元数据，不向文本模型伪造图片理解结果。
//
// 浏览器响应绝不使用本内部结构。

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"tedna/internal/models"
	"tedna/internal/repository"
)

const (
	coursewareComicReferencePromptMaxItems =
		8

	coursewareComicReferencePromptItemMaxRunes =
		6000

	coursewareComicReferencePromptTotalMaxRunes =
		24000
)

var ErrCoursewareComicReferenceReadFailed =
	errors.New(
		"读取知识点漫画参考资料失败",
	)

// coursewareComicPromptReference
// 是文本模型接收的单条可信参考资源。
type coursewareComicPromptReference struct {
	ResourceType string `json:"resource_type"`
	Title        string `json:"title"`
	Content      string `json:"content"`
}

// loadCoursewareComicReferencePromptContext
// 按项目三重边界读取并裁剪参考资料上下文。
func loadCoursewareComicReferencePromptContext(
	ctx context.Context,
	coursewareID string,
	projectID string,
	userID string,
) ([]coursewareComicPromptReference, error) {
	items, err :=
		repository.
			ListCoursewareComicReferenceResourcesByProjectForUser(
				ctx,
				strings.TrimSpace(
					coursewareID,
				),
				strings.TrimSpace(
					projectID,
				),
				strings.TrimSpace(
					userID,
				),
			)
	if err != nil {
		return nil,
			fmt.Errorf(
				"%w: %v",
				ErrCoursewareComicReferenceReadFailed,
				err,
			)
	}

	result :=
		make(
			[]coursewareComicPromptReference,
			0,
			len(items),
		)

	totalRunes :=
		0

	for _, item :=
		range items {
		if item == nil ||
			len(result) >=
				coursewareComicReferencePromptMaxItems ||
			totalRunes >=
				coursewareComicReferencePromptTotalMaxRunes {
			break
		}

		content :=
			strings.TrimSpace(
				item.SummaryText,
			)

		if content == "" {
			content =
				strings.TrimSpace(
					item.ContentText,
				)
		}

		if item.ResourceType ==
			models.CWComicReferenceUploadedImage {
			content =
				"教师提供了一张视觉参考图片。文字规划阶段只使用标题元数据；图片生成阶段必须重新读取正式图片资产。"
		}

		remaining :=
			coursewareComicReferencePromptTotalMaxRunes -
				totalRunes

		itemLimit :=
			coursewareComicReferencePromptItemMaxRunes

		if remaining <
			itemLimit {
			itemLimit =
				remaining
		}

		content =
			coursewareComicReferenceTruncateRunes(
				content,
				itemLimit,
			)

		if content == "" {
			continue
		}

		result =
			append(
				result,
				coursewareComicPromptReference{
					ResourceType:
						item.ResourceType,
					Title:
						coursewareComicReferenceTruncateRunes(
							item.Title,
							500,
						),
					Content:
						content,
				},
			)

		totalRunes +=
			utf8.RuneCountInString(
				content,
			)
	}

	return result, nil
}
