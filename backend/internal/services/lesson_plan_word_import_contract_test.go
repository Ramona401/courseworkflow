package services

import (
	"context"
	"testing"

	"tedna/internal/models"
)

// TestResolveTrustedLessonPlanWordImportSessionKeepsOrdinaryImport 验证
// 普通粘贴、普通Word和PDF导入不携带会话ID时，不访问Word会话仓储，
// 也不改变原有来源类型和正文。
func TestResolveTrustedLessonPlanWordImportSessionKeepsOrdinaryImport(
	t *testing.T,
) {
	request := &models.ImportExistingPlanRequest{
		SourceType:      "paste",
		ContentMarkdown: "普通导入正文",
	}

	session, err :=
		resolveTrustedLessonPlanWordImportSession(
			context.Background(),
			request,
			"author-contract-001",
		)
	if err != nil {
		t.Fatalf(
			"普通导入不应返回Word会话错误: %v",
			err,
		)
	}

	if session != nil {
		t.Fatal(
			"普通导入不应返回Word导入会话",
		)
	}

	if request.SourceType != "paste" {
		t.Fatalf(
			"普通导入来源被意外修改为%q",
			request.SourceType,
		)
	}

	if request.ContentMarkdown !=
		"普通导入正文" {
		t.Fatalf(
			"普通导入正文被意外修改为%q",
			request.ContentMarkdown,
		)
	}
}

// TestNormalizeLessonPlanImportSourceTypeAcceptsDocxFidelity 验证新增来源
// 与普通导入来源共存，且继续执行统一的小写和空白规范化。
func TestNormalizeLessonPlanImportSourceTypeAcceptsDocxFidelity(
	t *testing.T,
) {
	sourceType, err :=
		normalizeLessonPlanImportSourceType(
			"  DOCX_FIDELITY  ",
		)
	if err != nil {
		t.Fatalf(
			"docx_fidelity来源应被接受: %v",
			err,
		)
	}

	if sourceType != "docx_fidelity" {
		t.Fatalf(
			"来源规范化结果错误，实际值=%q",
			sourceType,
		)
	}
}
