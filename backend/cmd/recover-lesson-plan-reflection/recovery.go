package main

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var lessonPlanRecoveryImagePattern = regexp.MustCompile(
	`!\[[^\]]*\]\([^\)]*\)|\[图片：[^\]]*\]`,
)

const (
	reflectionPreClass = "大部分学生对“原子结构示意图”的预习情况良好，能够自主画出前18号元素的部分示意图。但部分学生对最外层电子数与元素化学性质的关系理解停留在机械记忆层面，未能建立“结构决定性质”的逻辑关联，需要在课堂中通过对比分析予以重点强化。"

	reflectionClassroom = "课堂中引入“能量跑道”的物理科学解释，有效避免了将电子层类比为“排座位”或“行星轨道”所带来的物理概念偏差，学生对电子分层排布的微观图景理解更加准确。在离子符号书写的教学中，新增的即时互评环节针对性强，及时纠正了学生将电荷数与正负号顺序写反（如写成 $Mg^{+2}$）的习惯性笔误，课堂生成效果显著。"

	reflectionAfterClass = "课后作业的设计应偏重于原子与离子结构示意图的对比辨析，特别是针对“氦原子与镁原子最外层同为2个电子但性质截然不同”这一典型易错点，需要设计微专题练习，引导学生深入理解“相对稳定结构”的本质，进一步巩固“微观结构决定宏观性质”的核心素养。"

	reflectionSuggestionMarker = "微观粒子卡片连连看"
)

const oldReflectionSuffix = `**教  后  反  思**
**课前预习阶段：**
** **
**课堂教学阶段：**
**  **
**课后提升阶段：**`

// buildReflectionRecoveryMarkdown 只替换当前v19尾部的三个既有反思槽位。
//
// 原Word只有六个对应段落：标题、课前标签、空白正文、课堂标签、
// 空白正文、课后标签。为保持段落数量完全不变，课后正文与最后一个
// 既有标签写在同一段落中。
func buildReflectionRecoveryMarkdown(
	current string,
) (string, error) {
	current = strings.TrimSpace(current)
	if current == "" {
		return "", errors.New("当前正式正文为空")
	}

	if strings.Count(
		current,
		oldReflectionSuffix,
	) != 1 ||
		!strings.HasSuffix(
			current,
			oldReflectionSuffix,
		) {
		return "", errors.New(
			"当前正文不再是已确认的v19教后反思空白基线",
		)
	}

	for _, body := range []string{
		reflectionPreClass,
		reflectionClassroom,
		reflectionAfterClass,
	} {
		if strings.Contains(current, body) {
			return "", errors.New(
				"教后反思正文已经存在，拒绝重复恢复",
			)
		}
	}

	newSuffix := strings.Join(
		[]string{
			"**教  后  反  思**",
			"**课前预习阶段：**",
			reflectionPreClass,
			"**课堂教学阶段：**",
			reflectionClassroom,
			"**课后提升阶段：**" +
				reflectionAfterClass,
		},
		"\n",
	)

	next := strings.TrimSuffix(
		current,
		oldReflectionSuffix,
	) + newSuffix

	if strings.Count(current, "\n") !=
		strings.Count(next, "\n") {
		return "", errors.New(
			"恢复内容改变了Word段落换行数量",
		)
	}

	currentImages :=
		lessonPlanRecoveryImagePattern.
			FindAllString(current, -1)
	nextImages :=
		lessonPlanRecoveryImagePattern.
			FindAllString(next, -1)

	if len(currentImages) != len(nextImages) {
		return "", errors.New(
			"恢复内容改变了图片标记数量",
		)
	}
	for index := range currentImages {
		if currentImages[index] != nextImages[index] {
			return "", fmt.Errorf(
				"恢复内容改变了第%d个图片标记",
				index+1,
			)
		}
	}

	if strings.Contains(
		next,
		reflectionSuggestionMarker,
	) {
		return "", errors.New(
			"仅供参考的补充建议不得写入正文",
		)
	}

	return next, nil
}

func validateRecoveredReflection(
	content string,
) error {
	for _, required := range []string{
		reflectionPreClass,
		reflectionClassroom,
		reflectionAfterClass,
	} {
		if !strings.Contains(
			content,
			required,
		) {
			return errors.New(
				"恢复后的正式正文缺少确认内容",
			)
		}
	}

	if strings.Contains(
		content,
		reflectionSuggestionMarker,
	) {
		return errors.New(
			"恢复后的正式正文错误包含补充建议",
		)
	}

	return nil
}
