package services

// lesson_plan_context_capsule_runtime.go — 胶囊运行时轻量读取
//
// 运行原则：
//   - 只做一次active胶囊数据库读取；
//   - 不调用AI；
//   - 不等待旁路更新；
//   - 不读取课本或课程大纲全文；
//   - 不增加流式首字延迟中的模型调用；
//   - 当前阶段焦点在读取时动态追加，不依赖胶囊中可能滞后的stage_focus。
//
// active胶囊不存在或已经stale时返回空字符串。
// 调用方继续使用现有active课程大纲知识脉络作为零等待兜底。

import (
	"context"
	"fmt"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// BuildLessonPlanContextCapsuleRuntime 返回可立即注入的稳定核心记忆。
func BuildLessonPlanContextCapsuleRuntime(
	ctx context.Context,
	lessonPlan *models.LessonPlan,
) (
	string,
	*models.LessonPlanContextCapsule,
	error,
) {
	if lessonPlan == nil ||
		strings.TrimSpace(lessonPlan.ID) == "" {
		return "", nil, nil
	}

	capsule, err :=
		repository.GetActiveLessonPlanContextCapsule(
			ctx,
			lessonPlan.ID,
		)
	if err != nil {
		return "", nil, fmt.Errorf(
			"读取备课核心共识胶囊失败: %w",
			err,
		)
	}

	if capsule == nil ||
		!capsule.IsActiveUsable() ||
		strings.TrimSpace(capsule.ContextText) == "" {
		return "", nil, nil
	}

	stageName := stageCodeToName(
		strings.TrimSpace(
			lessonPlan.CurrentStage,
		),
	)

	var builder strings.Builder
	builder.WriteString(
		"\n\n" +
			strings.TrimSpace(capsule.ContextText) +
			"\n",
	)

	builder.WriteString(
		"\n【当前自然工作焦点】\n",
	)

	builder.WriteString(
		"- 当前后台工作重点是“" +
			stageName +
			"”，只调整本轮注意力，不代表重新开始一段对话。\n",
	)

	builder.WriteString(
		"- 继续带着前面已经确认的课程核心、教学决定、边界和订正结果推进；忽略胶囊结构中可能滞后的旧阶段任务。\n",
	)

	builder.WriteString(
		"- 回复中不得向教师宣告阶段变化，不得重新介绍自己，也不得再次确认已经确认过的内容。\n",
	)

	return builder.String(), capsule, nil
}
