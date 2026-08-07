package repository

// courseware_review_resubmission_rules_test.go
//
// 作者重新提交正式整改条件的纯规则测试。
//
// 本测试不连接数据库，只验证：
//
//   - 不同问题状态的提交分类；
//   - 页级修改完成内容指纹检查；
//   - 整课问题的修改完成证据；
//   - 页面变化和页面删除分类；
//   - 作者可读错误中的数量表达；
//   - errors.Is能够识别统一的整改未完成错误。

import (
	"errors"
	"strings"
	"testing"
	"time"

	"tedna/internal/models"
)

func TestClassifyCWReviewSubmissionItem(
	t *testing.T,
) {
	t.Parallel()

	appliedAt := time.Now()
	currentHTML := "<div>当前页面</div>"

	currentHash :=
		cwReviewItemContentHash(
			currentHTML,
		)

	tests := []struct {
		name              string
		item              *cwReviewSubmissionItemFact
		currentPageExists bool
		currentHTML       string
		expected          string
	}{
		{
			name: "页级修改完成且页面未变化",
			item: &cwReviewSubmissionItemFact{
				Status:          models.CWReviewItemStatusApplied,
				PageID:          "page-1",
				AppliedAt:       &appliedAt,
				AppliedPageHash: currentHash,
			},
			currentPageExists: true,
			currentHTML:       currentHTML,
			expected:          cwReviewSubmissionAssessmentReady,
		},
		{
			name: "整课问题修改完成证据完整",
			item: &cwReviewSubmissionItemFact{
				Status:          models.CWReviewItemStatusApplied,
				AppliedAt:       &appliedAt,
				AppliedPageHash: currentHash,
			},
			expected: cwReviewSubmissionAssessmentReady,
		},
		{
			name: "整改要求已确认但尚未修改",
			item: &cwReviewSubmissionItemFact{
				Status: models.CWReviewItemStatusConfirmed,
				PageID: "page-1",
			},
			expected: cwReviewSubmissionAssessmentUnfinished,
		},
		{
			name: "修改中不能提交",
			item: &cwReviewSubmissionItemFact{
				Status: models.CWReviewItemStatusApplying,
				PageID: "page-1",
			},
			expected: cwReviewSubmissionAssessmentUnfinished,
		},
		{
			name: "修改完成证据不完整",
			item: &cwReviewSubmissionItemFact{
				Status: models.CWReviewItemStatusApplied,
				PageID: "page-1",
			},
			currentPageExists: true,
			currentHTML:       currentHTML,
			expected:          cwReviewSubmissionAssessmentUnfinished,
		},
		{
			name: "已有页面变化状态",
			item: &cwReviewSubmissionItemFact{
				Status: models.CWReviewItemStatusStale,
				PageID: "page-1",
			},
			expected: cwReviewSubmissionAssessmentStale,
		},
		{
			name: "已有原页面删除状态",
			item: &cwReviewSubmissionItemFact{
				Status: models.CWReviewItemStatusOrphaned,
				PageID: "page-1",
			},
			expected: cwReviewSubmissionAssessmentOrphaned,
		},
		{
			name: "修改完成后页面内容变化",
			item: &cwReviewSubmissionItemFact{
				Status:          models.CWReviewItemStatusApplied,
				PageID:          "page-1",
				AppliedAt:       &appliedAt,
				AppliedPageHash: currentHash,
			},
			currentPageExists: true,
			currentHTML:       "<div>后来修改的页面</div>",
			expected:          cwReviewSubmissionAssessmentStale,
		},
		{
			name: "修改完成后原页面删除",
			item: &cwReviewSubmissionItemFact{
				Status:          models.CWReviewItemStatusApplied,
				PageID:          "page-1",
				AppliedAt:       &appliedAt,
				AppliedPageHash: currentHash,
			},
			currentPageExists: false,
			expected:          cwReviewSubmissionAssessmentOrphaned,
		},
		{
			name:     "空问题事实不能提交",
			item:     nil,
			expected: cwReviewSubmissionAssessmentUnfinished,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(
			test.name,
			func(t *testing.T) {
				t.Parallel()

				actual :=
					classifyCWReviewSubmissionItem(
						test.item,
						test.currentPageExists,
						test.currentHTML,
					)

				if actual != test.expected {
					t.Fatalf(
						"提交分类不正确: got=%s want=%s",
						actual,
						test.expected,
					)
				}
			},
		)
	}
}

func TestCWReviewSubmissionRemediationError(
	t *testing.T,
) {
	t.Parallel()

	err :=
		&CWReviewSubmissionRemediationError{
			UnfinishedCount:  16,
			ChangedPageCount: 8,
			MissingPageCount: 2,
		}

	if !errors.Is(
		err,
		ErrCWReviewSubmissionRemediationIncomplete,
	) {
		t.Fatal(
			"详细整改错误必须能够识别为统一业务错误",
		)
	}

	message := err.Error()

	expectedParts :=
		[]string{
			"16条尚未完成页面修改",
			"8条页面内容已变化，需要重新检查",
			"2条原问题页面已经删除",
			"请回到课件整改区处理并检查全部问题",
		}

	for _, expected := range expectedParts {
		if !strings.Contains(
			message,
			expected,
		) {
			t.Fatalf(
				"作者提示缺少内容 %q: %s",
				expected,
				message,
			)
		}
	}

	readiness :=
		&cwReviewSubmissionReadiness{
			ReadyCount:       2,
			UnfinishedCount:  16,
			ChangedPageCount: 8,
		}

	if !readiness.blocked() {
		t.Fatal(
			"存在未完成或页面变化问题时必须阻止提交",
		)
	}

	if !errors.Is(
		readiness.businessError(),
		ErrCWReviewSubmissionRemediationIncomplete,
	) {
		t.Fatal(
			"检查结果必须生成统一整改未完成错误",
		)
	}
}
