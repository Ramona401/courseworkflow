package services

// lesson_plan_evidence_harness_retry.go
//
// 正式产物Judge只允许进行一次JSON格式受控重试：
//   - 首次结果能够解析时直接使用，不增加额外模型调用；
//   - 首次结果无法解析时，对相同证据和相同候选正文重新判定一次；
//   - 重试只要求返回判定协议JSON，不得重新生成或改写教案；
//   - 第二次仍无法解析时继续fail-closed，不展示、不保存正文。

import (
	"fmt"
	"strings"
)

const lessonPlanEvidenceJudgeJSONRetryInstruction = `

【JSON格式受控重试】
上一次Judge输出未能解析为协议要求的JSON。
请针对完全相同的证据和候选正文重新执行同一判定。

强制要求：
1. 只能输出一个完整JSON对象。
2. 不得输出Markdown代码围栏、解释、前言、结语或分析过程。
3. 必须包含pass、unsupported_model_additions、source_conflicts、missing_required_evidence、reasons、repair_instruction全部字段。
4. 不得生成或改写教案正文。`

// parseLessonPlanEvidenceVerdictWithRetry
// 先解析首次结果；失败时最多调用一次受控重试。
func parseLessonPlanEvidenceVerdictWithRetry(
	firstRaw string,
	retry func() (string, error),
) (
	*lessonPlanEvidenceVerdict,
	bool,
	error,
) {
	verdict, firstErr :=
		parseLessonPlanEvidenceVerdict(
			firstRaw,
		)

	if firstErr == nil {
		return verdict, false, nil
	}

	if retry == nil {
		return nil, false, fmt.Errorf(
			"Judge结果无法解析: %w",
			firstErr,
		)
	}

	retryRaw, retryErr :=
		retry()

	if retryErr != nil {
		return nil, true, fmt.Errorf(
			"Judge首次结果无法解析，受控重试调用失败: 首次=%v；重试=%w",
			firstErr,
			retryErr,
		)
	}

	retryRaw =
		strings.TrimSpace(
			retryRaw,
		)

	if retryRaw == "" {
		return nil, true, fmt.Errorf(
			"Judge首次结果无法解析，受控重试结果为空: 首次=%v",
			firstErr,
		)
	}

	verdict, retryParseErr :=
		parseLessonPlanEvidenceVerdict(
			retryRaw,
		)

	if retryParseErr != nil {
		return nil, true, fmt.Errorf(
			"Judge两次均未返回合法JSON: 首次=%v；重试=%v",
			firstErr,
			retryParseErr,
		)
	}

	return verdict, true, nil
}
