package repository

// lesson_plan_joined_select_columns_test.go
//
// 该回归测试只检查JOIN查询使用的列清单是否全部带主表别名。
// 它不连接数据库，也不运行历史集成测试；目标是防止未来再次把未限定的
// id、created_at、updated_at等列放回多表查询，导致运行期SQLSTATE 42702。

import (
	"strings"
	"testing"
)

func TestLessonPlanJoinedSelectColumnsAreQualified(t *testing.T) {
	tests := []struct {
		name       string
		columns    string
		tableAlias string
		expected   int
	}{
		{
			name:       "context capsule",
			columns:    lessonPlanContextCapsuleSelectColumnsFromCapsule,
			tableAlias: "capsule.",
			expected:   17,
		},
		{
			name:       "knowledge lineage",
			columns:    lessonPlanKnowledgeLineageSelectColumnsFromLineage,
			tableAlias: "lineage.",
			expected:   17,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			rawColumns := strings.Split(
				strings.TrimSpace(testCase.columns),
				",",
			)

			if len(rawColumns) != testCase.expected {
				t.Fatalf(
					"JOIN列数量异常：got=%d want=%d",
					len(rawColumns),
					testCase.expected,
				)
			}

			for _, rawColumn := range rawColumns {
				column := strings.TrimSpace(rawColumn)
				if column == "" {
					t.Fatal("JOIN列清单存在空列")
				}
				if !strings.HasPrefix(
					column,
					testCase.tableAlias,
				) {
					t.Fatalf(
						"JOIN列未限定主表别名：%s",
						column,
					)
				}
			}
		})
	}
}
