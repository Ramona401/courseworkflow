package services

// courseware_curriculum.go — 课程知识库注入：把选中的知识点 + 课标深度
// 拼成"难度自动适配约束段落"，供 buildTopicDirectPrompt 注入 AI 提示词。
//
// 设计：本文件是"从主题创建课件→难度自动适配"的唯一注入逻辑入口，独立成文件，
// 不让 courseware_index_service.go(已1180行)继续膨胀。
//
// 核心机制：
//   前端勾选若干知识点(kp_codes) → 查 curriculum_standards 取每个知识点的
//   学业要求(学到什么程度) + 深度档 + 核心素养 → 拼成约束段落 →
//   AI 据此规划课件，难度自动贴合课标该年级该知识点的要求。
//
// 容错：kp_codes 为空 / 查询失败 / 查不到 → 返回空串，调用方退回原有"纯主题规划"逻辑，
//       绝不因知识库问题阻断课件生成。

import (
	"context"
	"fmt"
	"log"
	"strings"

	"tedna/internal/repository"
)

// 深度档中文标签（对齐 curriculum_standards.depth_level：1体验感知/2理解应用/3分析迁移）
var cwDepthLevelLabel = map[int]string{
	1: "体验感知（初步接触、感性认识，不求深究）",
	2: "理解应用（理解原理并能在具体情境中运用）",
	3: "分析迁移（深入分析、综合运用、迁移到新情境）",
}

// BuildCurriculumConstraint 根据选中的知识点编码，构建"课标难度适配约束段落"
//
// 返回值：可直接拼进 AI 用户提示词的一段中文文本；若无有效知识点则返回空串。
// 该段落明确告诉 AI：本课件要覆盖哪些知识点、每个知识点学到什么程度、整体难度档，
// 使 AI 规划方案时自动贴合课标对该年级该知识点的深度要求。
func BuildCurriculumConstraint(ctx context.Context, kpCodes []string) string {
	if len(kpCodes) == 0 {
		return ""
	}

	kps, err := repository.GetCurriculumKPsByCodes(ctx, kpCodes)
	if err != nil {
		// 知识库查询失败不阻断生成，仅记日志，退回无约束逻辑
		log.Printf("[courseware_curriculum] 查询课标知识点失败，退回无约束规划: codes=%v err=%v", kpCodes, err)
		return ""
	}
	if len(kps) == 0 {
		log.Printf("[courseware_curriculum] 未匹配到任何课标知识点: codes=%v", kpCodes)
		return ""
	}

	// 统计整体难度档（取选中知识点深度档的最大值作为课件整体难度上限参考）
	maxDepth := 1
	for _, kp := range kps {
		if kp.DepthLevel > maxDepth {
			maxDepth = kp.DepthLevel
		}
	}

	var sb strings.Builder
	sb.WriteString("\n## 课标知识点与难度要求（本课件必须严格遵循以下课程标准约束）\n")
	sb.WriteString("以下是本课件需覆盖的知识点，每个知识点都标注了课程标准规定的【学习深度】与【学到什么程度】。")
	sb.WriteString("请你在规划课件时，使每一页的内容难度、例题深度、练习难度都严格贴合对应知识点的深度要求，")
	sb.WriteString("既不能拔高（超纲增加学生负担），也不能降低（达不到课标要求）。\n\n")

	for i, kp := range kps {
		depthLabel := cwDepthLevelLabel[kp.DepthLevel]
		if depthLabel == "" {
			depthLabel = "理解应用"
		}
		sb.WriteString(fmt.Sprintf("### 知识点%d：%s\n", i+1, kp.KPName))
		if kp.Domain != "" {
			sb.WriteString(fmt.Sprintf("- 所属领域：%s\n", kp.Domain))
		}
		sb.WriteString(fmt.Sprintf("- 学习深度档：第%d档（%s）\n", kp.DepthLevel, depthLabel))
		if kp.ContentRequirement != "" {
			sb.WriteString(fmt.Sprintf("- 内容要求（学什么）：%s\n", kp.ContentRequirement))
		}
		if kp.AcademicRequirement != "" {
			sb.WriteString(fmt.Sprintf("- 学业要求（学到什么程度）：%s\n", kp.AcademicRequirement))
		}
		if kp.TeachingHint != "" {
			sb.WriteString(fmt.Sprintf("- 教学提示：%s\n", kp.TeachingHint))
		}
		if kp.CoreCompetency != "" {
			sb.WriteString(fmt.Sprintf("- 培养的核心素养：%s\n", kp.CoreCompetency))
		}
		sb.WriteString("\n")
	}

	// 整体难度提示（注意：此处不使用装饰性引号/中点等歧义字符，避免Go源码解析问题）
	overallLabel := cwDepthLevelLabel[maxDepth]
	if overallLabel == "" {
		overallLabel = "理解应用"
	}
	sb.WriteString(fmt.Sprintf("## 整体难度基调\n本课件整体难度以第%d档（%s）为上限基准，", maxDepth, overallLabel))
	sb.WriteString("知识讲授循序渐进，练习难度与课标学业要求对齐，确保符合该年级学生的认知水平。\n")

	return sb.String()
}
