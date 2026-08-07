package services

// courseware_review_v13_rules_test.go
//
// V1.3整改复审闭环中的纯规则检查。
//
// 本文件只检查哪些级别属于正式复审范围，不连接数据库。
// 防止0级、未知级别或越界级别被误认为正式复审。

import (
	"testing"

	"tedna/internal/models"
)

func TestIsCWPendingReviewLevel(
	t *testing.T,
) {
	tests := []struct {
		name        string
		reviewLevel int
		expected    bool
	}{
		{
			name:        "未进入正式审核",
			reviewLevel: 0,
			expected:    false,
		},
		{
			name:        "L1教研组审核",
			reviewLevel: models.ReviewLevelL1,
			expected:    true,
		},
		{
			name:        "L2学校审核",
			reviewLevel: models.ReviewLevelL2,
			expected:    true,
		},
		{
			name:        "超出正式审核级别",
			reviewLevel: 3,
			expected:    false,
		},
	}

	for _, test := range tests {
		t.Run(
			test.name,
			func(t *testing.T) {
				actual :=
					isCWPendingReviewLevel(
						test.reviewLevel,
					)

				if actual !=
					test.expected {
					t.Fatalf(
						"复审级别判断错误：level=%d，实际=%v，期望=%v",
						test.reviewLevel,
						actual,
						test.expected,
					)
				}
			},
		)
	}
}
