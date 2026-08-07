package repository

import (
	"strings"
	"testing"

	"tedna/internal/models"
)

func TestResolveCoursewareComicAggregateStatus(
	t *testing.T,
) {
	cases :=
		[]struct {
			name           string
			totalCount     int
			generatedCount int
			failedCount    int
			expected       string
		}{
			{
				name:           "all generated",
				totalCount:     4,
				generatedCount: 4,
				failedCount:    0,
				expected:       models.CWComicProjectStatusReady,
			},
			{
				name:           "generation continues",
				totalCount:     4,
				generatedCount: 2,
				failedCount:    0,
				expected:       models.CWComicProjectStatusGenerating,
			},
			{
				name:           "one failed",
				totalCount:     4,
				generatedCount: 1,
				failedCount:    1,
				expected:       models.CWComicProjectStatusFailed,
			},
			{
				name:           "empty project remains generating",
				totalCount:     0,
				generatedCount: 0,
				failedCount:    0,
				expected:       models.CWComicProjectStatusGenerating,
			},
		}

	for _, testCase := range cases {
		testCase :=
			testCase

		t.Run(
			testCase.name,
			func(t *testing.T) {
				actual :=
					resolveCoursewareComicAggregateStatus(
						testCase.totalCount,
						testCase.generatedCount,
						testCase.failedCount,
					)

				if actual !=
					testCase.expected {
					t.Fatalf(
						"聚合状态错误：得到%q，期望%q",
						actual,
						testCase.expected,
					)
				}
			},
		)
	}
}

func TestCoursewareComicAggregateUpdateSQLUsesIsolatedBooleanParameter(
	t *testing.T,
) {
	sqlText :=
		coursewareComicAggregateProjectUpdateSQL

	if strings.Count(
		sqlText,
		"$1",
	) != 1 {
		t.Fatalf(
			"项目状态参数$1必须只出现一次，当前SQL：%s",
			sqlText,
		)
	}

	if !strings.Contains(
		sqlText,
		"$2::boolean",
	) {
		t.Fatalf(
			"聚合SQL缺少独立boolean参数：%s",
			sqlText,
		)
	}

	if strings.Contains(
		sqlText,
		"WHEN $1 =",
	) {
		t.Fatalf(
			"聚合SQL不得再次用$1执行字符串比较：%s",
			sqlText,
		)
	}
}
