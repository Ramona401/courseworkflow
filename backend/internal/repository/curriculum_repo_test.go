package repository

// curriculum_repo_test.go
//
// 验证课程知识库Repository最终防线：
//   - 教育域匹配严格区分K12与其它域；
//   - 非K12查询在访问database.DB之前直接返回类型正确的空切片；
//   - K12空编码同样直接返回空切片。
//
// 测试刻意不初始化数据库连接；一旦非K12短路失效，测试会因访问空数据库
// 连接而失败，从而阻止跨域基础数据查询回归。

import (
	"context"
	"testing"

	"tedna/internal/models"
)

func TestIsK12CurriculumEducationDomain(
	t *testing.T,
) {
	tests := []struct {
		name     string
		domain   string
		expected bool
	}{
		{
			name:     "标准K12",
			domain:   models.EducationDomainK12,
			expected: true,
		},
		{
			name:     "大小写和空格规范化",
			domain:   "  K12  ",
			expected: true,
		},
		{
			name:   "职业教育",
			domain: models.EducationDomainVocational,
		},
		{
			name:   "成人教育",
			domain: models.EducationDomainAdult,
		},
		{
			name:   "mixed",
			domain: models.EducationDomainMixed,
		},
		{
			name:   "common",
			domain: models.EducationDomainCommon,
		},
		{
			name: "空值",
		},
		{
			name:   "非法值",
			domain: "invalid-domain",
		},
	}

	for _, testCase := range tests {
		testCase := testCase

		t.Run(testCase.name, func(t *testing.T) {
			result := isK12CurriculumEducationDomain(
				testCase.domain,
			)
			if result != testCase.expected {
				t.Fatalf(
					"K12教育域判断错误: domain=%q got=%v want=%v",
					testCase.domain,
					result,
					testCase.expected,
				)
			}
		})
	}
}

func TestCurriculumReadFunctionsReturnEmptyOutsideK12(
	t *testing.T,
) {
	domains := []struct {
		name   string
		domain string
	}{
		{
			name:   "职业教育",
			domain: models.EducationDomainVocational,
		},
		{
			name:   "成人教育",
			domain: models.EducationDomainAdult,
		},
		{
			name:   "mixed",
			domain: models.EducationDomainMixed,
		},
		{
			name:   "common",
			domain: models.EducationDomainCommon,
		},
		{
			name: "空值",
		},
		{
			name:   "非法值",
			domain: "invalid-domain",
		},
	}

	for _, testCase := range domains {
		testCase := testCase

		t.Run(testCase.name, func(t *testing.T) {
			ctx := context.Background()

			knowledgePoints, err :=
				ListCurriculumKPsBySubjectGrade(
					ctx,
					testCase.domain,
					"数学",
					3,
				)
			if err != nil {
				t.Fatalf(
					"非K12知识点查询不应返回错误: %v",
					err,
				)
			}
			if knowledgePoints == nil ||
				len(knowledgePoints) != 0 {
				t.Fatalf(
					"非K12知识点结果必须是非nil空切片: %#v",
					knowledgePoints,
				)
			}

			byCodes, err :=
				GetCurriculumKPsByCodes(
					ctx,
					testCase.domain,
					[]string{"MATH-3-1"},
				)
			if err != nil {
				t.Fatalf(
					"非K12编码查询不应返回错误: %v",
					err,
				)
			}
			if byCodes == nil ||
				len(byCodes) != 0 {
				t.Fatalf(
					"非K12编码结果必须是非nil空切片: %#v",
					byCodes,
				)
			}

			units, err := ListTextbookUnits(
				ctx,
				testCase.domain,
				"数学",
				"人教版",
				3,
				"上册",
			)
			if err != nil {
				t.Fatalf(
					"非K12教材查询不应返回错误: %v",
					err,
				)
			}
			if units == nil ||
				len(units) != 0 {
				t.Fatalf(
					"非K12教材结果必须是非nil空切片: %#v",
					units,
				)
			}

			publishers, err :=
				ListTextbookPublishers(
					ctx,
					testCase.domain,
					"数学",
					3,
				)
			if err != nil {
				t.Fatalf(
					"非K12出版社查询不应返回错误: %v",
					err,
				)
			}
			if publishers == nil ||
				len(publishers) != 0 {
				t.Fatalf(
					"非K12出版社结果必须是非nil空切片: %#v",
					publishers,
				)
			}
		})
	}
}

func TestGetCurriculumKPsByCodesEmptyInput(
	t *testing.T,
) {
	knowledgePoints, err :=
		GetCurriculumKPsByCodes(
			context.Background(),
			models.EducationDomainK12,
			nil,
		)
	if err != nil {
		t.Fatalf(
			"K12空编码查询不应返回错误: %v",
			err,
		)
	}
	if knowledgePoints == nil ||
		len(knowledgePoints) != 0 {
		t.Fatalf(
			"K12空编码必须返回非nil空切片: %#v",
			knowledgePoints,
		)
	}
}
