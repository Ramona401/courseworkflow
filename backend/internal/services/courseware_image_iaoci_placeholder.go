package services

// courseware_image_iaoci_placeholder.go — 图片IAOCI HTML占位操作
//
// 本文件负责：
//   1. 识别<div class="img-placeholder">真实图片槽位；
//   2. 保留已有稳定placeholder_id；
//   3. 为缺失或重复ID的槽位生成稳定ID；
//   4. 读取data-desc语义；
//   5. 精确定位指定占位的配对div；
//   6. 精确填入对应图片；
//   7. 隐藏失败或不需要的单个槽位。

import (
	"fmt"
	stdhtml "html"
	"regexp"
	"strings"
)

// cwImagePlaceholderSlot 表示页面中的一个真实图片槽位。
type cwImagePlaceholderSlot struct {
	PlaceholderID string
	Description   string
	ImageKey      string
	Order         int
}

// CoursewareImageAOCIPlanItem 是自动装配使用的一张图片计划。
type CoursewareImageAOCIPlanItem struct {
	IndexID       string
	PlaceholderID string
	ImageKey      string
	AOCIText      string
	Caption       string
	Prompt        string
	Size          string
	Order         int
}

var (
	cwImagePlaceholderDivOpenRe = regexp.MustCompile(`(?is)<div\b[^>]*>`)

	cwImagePlaceholderClassRe = regexp.MustCompile(
		`(?i)\bclass\s*=\s*["'][^"']*img-placeholder[^"']*["']`,
	)

	cwImagePlaceholderIDAttrRe = regexp.MustCompile(
		`(?i)\bdata-placeholder-id\s*=\s*["']([^"']*)["']`,
	)

	cwImagePlaceholderDescAttrRe = regexp.MustCompile(
		`(?i)\bdata-desc\s*=\s*["']([^"']*)["']`,
	)
)

// cwEnsureImagePlaceholderIDs 为真实图片占位补稳定ID。
func cwEnsureImagePlaceholderIDs(
	pageHTML string,
) (
	string,
	[]cwImagePlaceholderSlot,
	bool,
) {
	pageHTML, nestedChanged := cwNormalizeNestedImagePlaceholderContainers(pageHTML)
	usedIDs := make(map[string]bool)
	slots := make(
		[]cwImagePlaceholderSlot,
		0,
	)

	changed := nestedChanged
	order := 0

	normalizedHTML :=
		cwImagePlaceholderDivOpenRe.
			ReplaceAllStringFunc(
				pageHTML,
				func(openTag string) string {
					if !cwImagePlaceholderClassRe.
						MatchString(openTag) {
						return openTag
					}

					order++

					placeholderID := ""

					idMatch :=
						cwImagePlaceholderIDAttrRe.
							FindStringSubmatchIndex(
								openTag,
							)

					if len(idMatch) >= 4 {
						// 切片必须是[start:end]，右边不能带逗号。
						placeholderID = strings.TrimSpace(
							openTag[idMatch[2]:idMatch[3]],
						)
					}

					if placeholderID == "" ||
						usedIDs[placeholderID] {
						placeholderID =
							cwNextImagePlaceholderID(
								order,
								usedIDs,
							)

						if len(idMatch) >= 4 {
							openTag =
								openTag[:idMatch[2]] +
									placeholderID +
									openTag[idMatch[3]:]
						} else {
							insertAt :=
								strings.LastIndex(
									openTag,
									">",
								)

							if insertAt >= 0 {
								openTag =
									openTag[:insertAt] +
										` data-placeholder-id="` +
										placeholderID +
										`"` +
										openTag[insertAt:]
							}
						}

						changed = true
					}

					usedIDs[placeholderID] = true

					description := ""

					descMatch :=
						cwImagePlaceholderDescAttrRe.
							FindStringSubmatch(
								openTag,
							)

					if len(descMatch) >= 2 {
						description =
							strings.TrimSpace(
								stdhtml.UnescapeString(
									descMatch[1],
								),
							)
					}

					if description == "" {
						description = fmt.Sprintf(
							"页面第%d个内容配图槽位",
							order,
						)
					}

					slots = append(
						slots,
						cwImagePlaceholderSlot{
							PlaceholderID: placeholderID,
							Description:   description,
							Order:         order,
						},
					)

					return openTag
				},
			)

	return normalizedHTML, slots, changed
}

func cwNextImagePlaceholderID(
	start int,
	usedIDs map[string]bool,
) string {
	for candidateNumber := start; ; candidateNumber++ {
		candidate := fmt.Sprintf(
			"IMG_SLOT_%02d",
			candidateNumber,
		)

		if !usedIDs[candidate] {
			return candidate
		}
	}
}

// cwFindImagePlaceholderRange 定位指定占位的完整div区间。
func cwFindImagePlaceholderRange(
	pageHTML string,
	placeholderID string,
) (
	int,
	int,
	int,
	bool,
) {
	lowerHTML := strings.ToLower(pageHTML)

	locations :=
		cwImagePlaceholderDivOpenRe.
			FindAllStringIndex(
				pageHTML,
				-1,
			)

	for _, location := range locations {
		openTag :=
			pageHTML[location[0]:location[1]]

		if !cwImagePlaceholderClassRe.
			MatchString(openTag) {
			continue
		}

		idMatch :=
			cwImagePlaceholderIDAttrRe.
				FindStringSubmatch(openTag)

		if len(idMatch) < 2 ||
			strings.TrimSpace(idMatch[1]) !=
				placeholderID {
			continue
		}

		closeStart, ok :=
			cwFindMatchingDivClose(
				pageHTML,
				lowerHTML,
				location[1],
			)
		if !ok {
			return 0, 0, 0, false
		}

		if cwImagePlaceholderContainsNested(
                                pageHTML,
                                location[1],
                                closeStart,
                        ) {
                                return 0, 0, 0, false
                        }

                        return location[0],
			location[1],
			closeStart,
			true
	}

	return 0, 0, 0, false
}

// cwFillImagePlaceholder 精确填充一个图片槽位。
func cwFillImagePlaceholder(
	pageHTML string,
	placeholderID string,
	imageURL string,
	altText string,
) (string, bool) {
	start, openEnd, closeStart, ok :=
		cwFindImagePlaceholderRange(
			pageHTML,
			placeholderID,
		)
	if !ok {
		return pageHTML, false
	}

	openTag := pageHTML[start:openEnd]

	afterClose :=
		pageHTML[closeStart+len("</div>"):]

	imageTag := fmt.Sprintf(
		`<img src="%s" alt="%s" data-iaoci-slot="%s" style="width:100%%;height:100%%;object-fit:cover;display:block;border-radius:inherit;" />`,
		stdhtml.EscapeString(
			strings.TrimSpace(imageURL),
		),
		stdhtml.EscapeString(
			strings.TrimSpace(altText),
		),
		stdhtml.EscapeString(
			placeholderID,
		),
	)

	return pageHTML[:start] +
			openTag +
			imageTag +
			"</div>" +
			afterClose,
		true
}

// cwHideImagePlaceholder 隐藏失败或不需要的槽位。
func cwHideImagePlaceholder(
	pageHTML string,
	placeholderID string,
	state string,
) (string, bool) {
	start, _, closeStart, ok :=
		cwFindImagePlaceholderRange(
			pageHTML,
			placeholderID,
		)
	if !ok {
		return pageHTML, false
	}

	afterClose :=
		pageHTML[closeStart+len("</div>"):]

	hidden := fmt.Sprintf(
		`<div class="img-placeholder iaoci-image-hidden" data-placeholder-id="%s" data-image-state="%s" style="display:none"></div>`,
		stdhtml.EscapeString(
			placeholderID,
		),
		stdhtml.EscapeString(
			state,
		),
	)

	return pageHTML[:start] +
			hidden +
			afterClose,
		true
}
