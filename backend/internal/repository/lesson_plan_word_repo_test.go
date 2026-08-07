package repository

import (
	"strings"
	"testing"
)

// TestLessonPlanWordDocumentQualifiedSelectColumns 确保JOIN查询使用的每个列
// 都显式绑定word_document别名，避免lesson_plans等关联表存在同名字段时
// 再次触发PostgreSQL SQLSTATE 42702歧义错误。
func TestLessonPlanWordDocumentQualifiedSelectColumns(
	t *testing.T,
) {
	lines := strings.Split(
		lessonPlanWordDocumentQualifiedSelectColumns,
		"\n",
	)

	columnCount := 0

	for _, line := range lines {
		column := strings.TrimSpace(line)
		if column == "" {
			continue
		}

		columnCount++

		if !strings.HasPrefix(
			column,
			"word_document.",
		) {
			t.Fatalf(
				"JOIN查询列未绑定word_document别名: %q",
				column,
			)
		}
	}

	if columnCount == 0 {
		t.Fatal("JOIN查询限定列清单不能为空")
	}
}
