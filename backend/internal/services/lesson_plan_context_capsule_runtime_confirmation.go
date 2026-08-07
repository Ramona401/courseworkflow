package services

// lesson_plan_context_capsule_runtime_confirmation.go
//
// 教案环节确认属于对话行为约束，而不是普通教学方向。
//
// 本模块只把可靠的教师确认状态转换成运行时防重复确认约束：
//   - 使用现有确定性解析器，不重新实现环节识别规则；
//   - 只接受active状态的teacher_explicit或teacher_source_confirmed条目；
//   - AI推断、候选、延期或已替代条目不会进入运行时强事实；
//   - 兼容历史lesson_plan_part、lesson_plan_section等旧条目；
//   - 多个历史条目统一归并为升序、去重的累积确认范围；
//   - 不直接注入内部条目原文，避免它混入普通教学方向。

import (
	"strings"

	"tedna/internal/models"
)

// writeLessonPlanCapsuleRuntimeConfirmedSections
// 写入教师已经确认的教案环节及防重复确认要求。
func writeLessonPlanCapsuleRuntimeConfirmedSections(
	builder *strings.Builder,
	document *models.LessonPlanContextCapsuleDocument,
) {
	if builder == nil ||
		document == nil {
		return
	}

	// false表示只读取结构化确认条目，不从摘要或当前进度反推。
	// 摘要和当前进度属于派生文本，不能单独升级为教师确认事实。
	confirmedSections :=
		lessonPlanCapsuleConfirmedSectionsFromDocument(
			document,
			false,
		)

	if len(confirmedSections) == 0 {
		return
	}

	formattedSections :=
		strings.TrimSpace(
			formatLessonPlanCapsuleSections(
				confirmedSections,
			),
		)

	if formattedSections == "" {
		return
	}

	builder.WriteString(
		"\n【已确认教案进度·不得重复确认】\n",
	)

	// 使用确定性派生表述，不直接复制内部进度条目原文。
	builder.WriteString(
		"- 教案" +
			formattedSections +
			"已经由教师确认。\n",
	)

	builder.WriteString(
		"以上确认状态必须直接沿用，不得再次询问教师是否确认；只有教师明确修改、撤销或恢复时才可更新。\n",
	)
}
