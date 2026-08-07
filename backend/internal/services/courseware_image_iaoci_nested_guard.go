package services

// courseware_image_iaoci_nested_guard.go — 嵌套图片占位结构保护。
//
// 当一个img-placeholder内部还包含其它img-placeholder时，外层只是布局容器。
// 若把外层当作真实槽位，填图会清空内部全部子槽位，表现为
// “一张图片占了几张图片的位置”。
//
// 本模块在规划前确定性规范化：
//   - 最内层img-placeholder继续一槽一图；
//   - 含子槽位的外层容器移除img-placeholder类；
//   - 同时移除外层旧data-placeholder-id；
//   - 不修改尺寸、布局、其它class、data-desc和正文内容。

import (
        "regexp"
        "sort"
        "strings"
)

type cwNestedImagePlaceholderCandidate struct {
        openStart  int
        openEnd    int
        closeStart int
        openTag    string
        isOuter    bool
}

var (
        cwNestedImagePlaceholderClassAttrRe = regexp.MustCompile(
                `(?is)\bclass\s*=\s*(?:"([^"]*)"|'([^']*)')`,
        )

        cwNestedImagePlaceholderIDAttrRe = regexp.MustCompile(
                `(?is)\s+data-placeholder-id\s*=\s*(?:"[^"]*"|'[^']*')`,
        )
)

// cwNormalizeNestedImagePlaceholderContainers 清理结构性外层占位类。
func cwNormalizeNestedImagePlaceholderContainers(
        pageHTML string,
) (
        string,
        bool,
) {
        if strings.TrimSpace(pageHTML) == "" {
                return pageHTML, false
        }

        lowerHTML := strings.ToLower(pageHTML)

        locations :=
                cwImagePlaceholderDivOpenRe.
                        FindAllStringIndex(
                                pageHTML,
                                -1,
                        )

        candidates := make(
                []cwNestedImagePlaceholderCandidate,
                0,
                len(locations),
        )

        for _, location := range locations {
                openTag :=
                        pageHTML[location[0]:location[1]]

                if !cwImagePlaceholderClassRe.
                        MatchString(
                                openTag,
                        ) {
                        continue
                }

                closeStart, ok :=
                        cwFindMatchingDivClose(
                                pageHTML,
                                lowerHTML,
                                location[1],
                        )

                if !ok {
                        continue
                }

                candidates = append(
                        candidates,
                        cwNestedImagePlaceholderCandidate{
                                openStart:  location[0],
                                openEnd:    location[1],
                                closeStart: closeStart,
                                openTag:    openTag,
                        },
                )
        }

        for parentIndex := range candidates {
                for childIndex := range candidates {
                        if parentIndex == childIndex {
                                continue
                        }

                        parent :=
                                candidates[parentIndex]

                        child :=
                                candidates[childIndex]

                        if child.openStart >= parent.openEnd &&
                                child.openStart < parent.closeStart {
                                candidates[parentIndex].isOuter = true
                                break
                        }
                }
        }

        outers := make(
                []cwNestedImagePlaceholderCandidate,
                0,
        )

        for _, candidate := range candidates {
                if candidate.isOuter {
                        outers = append(
                                outers,
                                candidate,
                        )
                }
        }

        if len(outers) == 0 {
                return pageHTML, false
        }

        // 从后向前替换，避免前面的字符串长度变化影响后面下标。
        sort.Slice(
                outers,
                func(left int, right int) bool {
                        return outers[left].openStart >
                                outers[right].openStart
                },
        )

        result := pageHTML
        changed := false

        for _, candidate := range outers {
                replacement :=
                        cwRemoveImagePlaceholderClass(
                                candidate.openTag,
                        )

                replacement =
                        cwNestedImagePlaceholderIDAttrRe.
                                ReplaceAllString(
                                        replacement,
                                        "",
                                )

                if replacement == candidate.openTag {
                        continue
                }

                result =
                        result[:candidate.openStart] +
                                replacement +
                                result[candidate.openEnd:]

                changed = true
        }

        return result, changed
}

// cwRemoveImagePlaceholderClass 从开标签中移除精确的img-placeholder类。
func cwRemoveImagePlaceholderClass(
        openTag string,
) string {
        match :=
                cwNestedImagePlaceholderClassAttrRe.
                        FindStringSubmatchIndex(
                                openTag,
                        )

        if len(match) < 6 {
                return openTag
        }

        valueStart := -1
        valueEnd := -1
        quote := `"`

        if match[2] >= 0 &&
                match[3] >= 0 {
                valueStart = match[2]
                valueEnd = match[3]
        } else if match[4] >= 0 &&
                match[5] >= 0 {
                valueStart = match[4]
                valueEnd = match[5]
                quote = `'`
        } else {
                return openTag
        }

        kept := make(
                []string,
                0,
        )

        for _, className :=
                range strings.Fields(
                        openTag[valueStart:valueEnd],
                ) {
                if !strings.EqualFold(
                        className,
                        "img-placeholder",
                ) {
                        kept = append(
                                kept,
                                className,
                        )
                }
        }

        replacement :=
                "class=" +
                        quote +
                        strings.Join(
                                kept,
                                " ",
                        ) +
                        quote

        return openTag[:match[0]] +
                replacement +
                openTag[match[1]:]
}

// cwImagePlaceholderContainsNested 判断目标槽位内部是否还有子图片槽位。
//
// 这是填图前的最后一道保险：即使历史HTML尚未规范化，
// 也不允许清空含子槽位的外层容器。
func cwImagePlaceholderContainsNested(
        pageHTML string,
        innerStart int,
        closeStart int,
) bool {
        if innerStart < 0 ||
                closeStart < innerStart ||
                closeStart > len(pageHTML) {
                return true
        }

        innerHTML :=
                pageHTML[innerStart:closeStart]

        locations :=
                cwImagePlaceholderDivOpenRe.
                        FindAllStringIndex(
                                innerHTML,
                                -1,
                        )

        for _, location := range locations {
                openTag :=
                        innerHTML[location[0]:location[1]]

                if cwImagePlaceholderClassRe.
                        MatchString(
                                openTag,
                        ) {
                        return true
                }
        }

        return false
}
