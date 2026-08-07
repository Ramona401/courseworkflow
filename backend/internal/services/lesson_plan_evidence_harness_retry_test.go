package services

// lesson_plan_evidence_harness_retry_test.go
//
// 验证Judge JSON重试规则：
//   - 首次合法时不重试；
//   - 首次非法、第二次合法时成功；
//   - 两次非法时安全失败；
//   - 合法pass=false是业务判定，不触发格式重试。

import (
	"strings"
	"testing"
)

func TestParseLessonPlanEvidenceVerdictWithRetrySkipsValidResult(
	t *testing.T,
) {
	retryCalls := 0

	verdict, retried, err :=
		parseLessonPlanEvidenceVerdictWithRetry(
			`{
				"pass": true,
				"unsupported_model_additions": [],
				"source_conflicts": [],
				"missing_required_evidence": [],
				"reasons": [],
				"repair_instruction": ""
			}`,
			func() (string, error) {
				retryCalls++
				return "", nil
			},
		)

	if err != nil {
		t.Fatalf(
			"合法Judge结果不应失败: %v",
			err,
		)
	}

	if retried ||
		retryCalls != 0 {
		t.Fatalf(
			"合法结果不应重试: retried=%v calls=%d",
			retried,
			retryCalls,
		)
	}

	if verdict == nil ||
		!verdict.Pass {
		t.Fatalf(
			"通过判定异常: %#v",
			verdict,
		)
	}
}

func TestParseLessonPlanEvidenceVerdictWithRetryUsesSecondJSON(
	t *testing.T,
) {
	retryCalls := 0

	verdict, retried, err :=
		parseLessonPlanEvidenceVerdictWithRetry(
			"第一次不是JSON。",
			func() (string, error) {
				retryCalls++

				return `{
					"pass": true,
					"unsupported_model_additions": [],
					"source_conflicts": [],
					"missing_required_evidence": [],
					"reasons": [],
					"repair_instruction": ""
				}`, nil
			},
		)

	if err != nil {
		t.Fatalf(
			"第二次返回合法JSON时不应失败: %v",
			err,
		)
	}

	if !retried ||
		retryCalls != 1 {
		t.Fatalf(
			"重试状态异常: retried=%v calls=%d",
			retried,
			retryCalls,
		)
	}

	if verdict == nil ||
		!verdict.Pass {
		t.Fatalf(
			"重试后的判定异常: %#v",
			verdict,
		)
	}
}

func TestParseLessonPlanEvidenceVerdictWithRetryFailsAfterTwoInvalidResults(
	t *testing.T,
) {
	retryCalls := 0

	verdict, retried, err :=
		parseLessonPlanEvidenceVerdictWithRetry(
			"第一次不是JSON。",
			func() (string, error) {
				retryCalls++
				return "第二次仍然不是JSON。", nil
			},
		)

	if err == nil {
		t.Fatal(
			"两次非法输出后必须失败",
		)
	}

	if verdict != nil {
		t.Fatalf(
			"失败时不应返回判定: %#v",
			verdict,
		)
	}

	if !retried ||
		retryCalls != 1 {
		t.Fatalf(
			"重试状态异常: retried=%v calls=%d",
			retried,
			retryCalls,
		)
	}

	if !strings.Contains(
		err.Error(),
		"两次均未返回合法JSON",
	) {
		t.Fatalf(
			"错误信息异常: %v",
			err,
		)
	}
}

func TestParseLessonPlanEvidenceVerdictWithRetryKeepsValidRejection(
	t *testing.T,
) {
	retryCalls := 0

	verdict, retried, err :=
		parseLessonPlanEvidenceVerdictWithRetry(
			`{
				"pass": false,
				"unsupported_model_additions": ["加入了来源中不存在的数据"],
				"source_conflicts": [],
				"missing_required_evidence": [],
				"reasons": ["存在无依据新增"],
				"repair_instruction": "删除无依据数据"
			}`,
			func() (string, error) {
				retryCalls++
				return "", nil
			},
		)

	if err != nil {
		t.Fatalf(
			"合法不通过判定不应视为格式错误: %v",
			err,
		)
	}

	if retried ||
		retryCalls != 0 {
		t.Fatalf(
			"合法不通过判定不应重试: retried=%v calls=%d",
			retried,
			retryCalls,
		)
	}

	if verdict == nil ||
		verdict.Pass {
		t.Fatalf(
			"不通过判定异常: %#v",
			verdict,
		)
	}

	if len(
		verdict.UnsupportedModelAdditions,
	) != 1 {
		t.Fatalf(
			"违规列表异常: %#v",
			verdict,
		)
	}
}
