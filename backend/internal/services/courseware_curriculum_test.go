package services

// courseware_curriculum_test.go
//
// 本测试不连接数据库，验证课程知识库直接编码查询的Service安全短路：
//   - 非K12教育域不访问数据库并返回空约束；
//   - 空编码不访问数据库；
//   - 主题创建编码校验在全部非K12域返回空数组；
//   - 年级文本解析保持既有口径。

import (
	"context"
	"testing"

	"tedna/internal/models"
)

func TestBuildCurriculumConstraintReturnsEmptyOutsideK12(
	t *testing.T,
) {
	tests := []struct {
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
			name:   "跨域管理",
			domain: models.EducationDomainMixed,
		},
		{
			name:   "通用资源域",
			domain: models.EducationDomainCommon,
		},
		{
			name: "空域",
		},
		{
			name:   "非法域",
			domain: "invalid-domain",
		},
	}

	for _, testCase := range tests {
		testCase := testCase

		t.Run(testCase.name, func(t *testing.T) {
			constraint, err :=
				BuildCurriculumConstraint(
					context.Background(),
					testCase.domain,
					[]string{"MATH-3-1"},
				)
			if err != nil {
				t.Fatalf(
					"非K12域不应返回错误: %v",
					err,
				)
			}
			if constraint != "" {
				t.Fatalf(
					"非K12域不应得到课标约束: %q",
					constraint,
				)
			}
		})
	}
}

func TestBuildCurriculumConstraintEmptyCodes(
	t *testing.T,
) {
	constraint, err := BuildCurriculumConstraint(
		context.Background(),
		models.EducationDomainK12,
		nil,
	)
	if err != nil {
		t.Fatalf(
			"空编码不应返回错误: %v",
			err,
		)
	}
	if constraint != "" {
		t.Fatalf(
			"空编码不应产生约束: %q",
			constraint,
		)
	}
}

func TestFilterValidKPCodesReturnsEmptyOutsideK12(
	t *testing.T,
) {
	tests := []struct {
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
			name:   "跨域管理",
			domain: models.EducationDomainMixed,
		},
		{
			name:   "通用资源域",
			domain: models.EducationDomainCommon,
		},
		{
			name: "空域",
		},
		{
			name:   "非法域",
			domain: "invalid-domain",
		},
	}

	service := &CoursewareService{}

	for _, testCase := range tests {
		testCase := testCase

		t.Run(testCase.name, func(t *testing.T) {
			actor := &CoursewareActorContext{
				UserID:          "user-1",
				Role:            models.RoleOperator,
				EducationDomain: testCase.domain,
			}

			validCodes, err :=
				service.filterValidKPCodes(
					context.Background(),
					actor,
					[]string{"MATH-3-1"},
					"数学",
					"三年级",
				)
			if err != nil {
				t.Fatalf(
					"非K12域编码校验不应返回错误: %v",
					err,
				)
			}
			if len(validCodes) != 0 {
				t.Fatalf(
					"非K12域不应保留K12编码: %v",
					validCodes,
				)
			}
		})
	}
}

func TestParseGradeNumForKP(
	t *testing.T,
) {
	tests := []struct {
		input    string
		expected int
	}{
		{
			input:    "三年级",
			expected: 3,
		},
		{
			input:    "初二",
			expected: 8,
		},
		{
			input:    "高三",
			expected: 12,
		},
		{
			input:    "10年级",
			expected: 10,
		},
		{
			input:    "无法识别",
			expected: 0,
		},
	}

	for _, testCase := range tests {
		testCase := testCase

		t.Run(testCase.input, func(t *testing.T) {
			result := parseGradeNumForKP(
				testCase.input,
			)
			if result != testCase.expected {
				t.Fatalf(
					"年级解析错误: input=%s got=%d want=%d",
					testCase.input,
					result,
					testCase.expected,
				)
			}
		})
	}
}
