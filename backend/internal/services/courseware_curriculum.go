package services

// courseware_curriculum.go — 课程知识库约束构建
//
// 把选中的知识点与课标深度拼成难度适配约束，
// 供主题课件方案生成提示词使用。
//
// 安全规则：
//   - 必须显式接收课件教育域快照；
//   - 只有K12域可以读取当前课程知识库；
//   - vocational、adult、mixed、common、空值和非法值返回空约束；
//   - K12数据库错误向上返回，不能伪装成无知识库约束。

import (
	"context"
	"fmt"
	"log"
	"strings"

	"tedna/internal/repository"
)

// cwDepthLevelLabel 是课程知识点深度档中文标签。
var cwDepthLevelLabel = map[int]string{
	1: "体验感知（初步接触、感性认识，不求深究）",
	2: "理解应用（理解原理并能在具体情境中运用）",
	3: "分析迁移（深入分析、综合运用、迁移到新情境）",
}

// BuildCurriculumConstraint 根据教育域快照和知识点编码构建课标约束。
//
// 返回值：
//   - 有有效K12知识点时返回约束文本；
//   - 非K12域或没有有效知识点时返回空文本和nil；
//   - K12数据库错误返回非nil错误。
func BuildCurriculumConstraint(
	ctx context.Context,
	educationDomain string,
	knowledgePointCodes []string,
) (string, error) {
	if len(knowledgePointCodes) == 0 {
		return "", nil
	}

	knowledgePoints, err := repository.GetCurriculumKPsByCodes(
		ctx,
		educationDomain,
		knowledgePointCodes,
	)
	if err != nil {
		log.Printf(
			"[courseware_curriculum] 查询课标知识点失败: domain=%s codes=%v err=%v",
			educationDomain,
			knowledgePointCodes,
			err,
		)
		return "", err
	}

	if len(knowledgePoints) == 0 {
		return "", nil
	}

	maxDepth := 1
	for _, knowledgePoint := range knowledgePoints {
		if knowledgePoint.DepthLevel > maxDepth {
			maxDepth = knowledgePoint.DepthLevel
		}
	}

	var builder strings.Builder

	builder.WriteString(
		"\n## 课标知识点与难度要求（本课件必须严格遵循以下课程标准约束）\n",
	)
	builder.WriteString(
		"以下是本课件需覆盖的知识点，每个知识点都标注了课程标准规定的【学习深度】与【学到什么程度】。",
	)
	builder.WriteString(
		"请你在规划课件时，使每一页的内容难度、例题深度、练习难度都严格贴合对应知识点的深度要求，",
	)
	builder.WriteString(
		"既不能拔高（超纲增加学生负担），也不能降低（达不到课标要求）。\n\n",
	)

	for index, knowledgePoint := range knowledgePoints {
		depthLabel := cwDepthLevelLabel[knowledgePoint.DepthLevel]
		if depthLabel == "" {
			depthLabel = "理解应用"
		}

		builder.WriteString(
			fmt.Sprintf(
				"### 知识点%d：%s\n",
				index+1,
				knowledgePoint.KPName,
			),
		)

		if knowledgePoint.Domain != "" {
			builder.WriteString(
				fmt.Sprintf(
					"- 所属领域：%s\n",
					knowledgePoint.Domain,
				),
			)
		}

		builder.WriteString(
			fmt.Sprintf(
				"- 学习深度档：第%d档（%s）\n",
				knowledgePoint.DepthLevel,
				depthLabel,
			),
		)

		if knowledgePoint.ContentRequirement != "" {
			builder.WriteString(
				fmt.Sprintf(
					"- 内容要求（学什么）：%s\n",
					knowledgePoint.ContentRequirement,
				),
			)
		}

		if knowledgePoint.AcademicRequirement != "" {
			builder.WriteString(
				fmt.Sprintf(
					"- 学业要求（学到什么程度）：%s\n",
					knowledgePoint.AcademicRequirement,
				),
			)
		}

		if knowledgePoint.TeachingHint != "" {
			builder.WriteString(
				fmt.Sprintf(
					"- 教学提示：%s\n",
					knowledgePoint.TeachingHint,
				),
			)
		}

		if knowledgePoint.CoreCompetency != "" {
			builder.WriteString(
				fmt.Sprintf(
					"- 培养的核心素养：%s\n",
					knowledgePoint.CoreCompetency,
				),
			)
		}

		builder.WriteString("\n")
	}

	overallLabel := cwDepthLevelLabel[maxDepth]
	if overallLabel == "" {
		overallLabel = "理解应用"
	}

	builder.WriteString(
		fmt.Sprintf(
			"## 整体难度基调\n本课件整体难度以第%d档（%s）为上限基准，",
			maxDepth,
			overallLabel,
		),
	)
	builder.WriteString(
		"知识讲授循序渐进，练习难度与课标学业要求对齐，确保符合该年级学生的认知水平。\n",
	)

	return builder.String(), nil
}
