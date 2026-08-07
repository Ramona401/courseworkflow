package services

import (
	"errors"
	"testing"

	"tedna/internal/models"
)

func TestSelectLessonPlanWordRestoreVersionPrefersExactVersion(
	t *testing.T,
) {
	candidates := []*models.LessonPlanWordDocumentVersion{
		{
			Version:              8,
			FileSHA256:           "file-new",
			StructureHash:        "structure-new",
			SemanticMarkdownHash: "semantic",
		},
		{
			Version:              3,
			FileSHA256:           "file-exact",
			StructureHash:        "structure-exact",
			SemanticMarkdownHash: "semantic",
		},
	}

	selected, err :=
		selectLessonPlanWordRestoreVersion(
			candidates,
			3,
		)
	if err != nil {
		t.Fatalf("选择同号Word版本失败: %v", err)
	}
	if selected == nil ||
		selected.Version != 3 ||
		selected.FileSHA256 != "file-exact" {
		t.Fatalf("没有优先选择同号Word版本: %+v", selected)
	}
}

func TestSelectLessonPlanWordRestoreVersionAllowsEquivalentCopies(
	t *testing.T,
) {
	candidates := []*models.LessonPlanWordDocumentVersion{
		{
			Version:              9,
			FileSHA256:           "same-file",
			StructureHash:        "same-structure",
			SemanticMarkdownHash: "same-semantic",
		},
		{
			Version:              6,
			FileSHA256:           "same-file",
			StructureHash:        "same-structure",
			SemanticMarkdownHash: "same-semantic",
		},
	}

	selected, err :=
		selectLessonPlanWordRestoreVersion(
			candidates,
			2,
		)
	if err != nil {
		t.Fatalf("等价Word快照应允许恢复: %v", err)
	}
	if selected == nil || selected.Version != 9 {
		t.Fatalf("应选择列表中的最新等价快照: %+v", selected)
	}
}

func TestSelectLessonPlanWordRestoreVersionRejectsAmbiguousLayouts(
	t *testing.T,
) {
	candidates := []*models.LessonPlanWordDocumentVersion{
		{
			Version:              9,
			FileSHA256:           "file-a",
			StructureHash:        "structure-a",
			SemanticMarkdownHash: "same-semantic",
		},
		{
			Version:              6,
			FileSHA256:           "file-b",
			StructureHash:        "structure-b",
			SemanticMarkdownHash: "same-semantic",
		},
	}

	selected, err :=
		selectLessonPlanWordRestoreVersion(
			candidates,
			2,
		)
	if selected != nil {
		t.Fatalf("歧义快照不应返回候选: %+v", selected)
	}
	if !errors.Is(
		err,
		ErrLessonPlanWordRestoreSnapshotAmbiguous,
	) {
		t.Fatalf("歧义快照错误类型不正确: %v", err)
	}
}

func TestSelectLessonPlanWordRestoreVersionRejectsMissingSnapshot(
	t *testing.T,
) {
	selected, err :=
		selectLessonPlanWordRestoreVersion(
			nil,
			2,
		)
	if selected != nil {
		t.Fatalf("空候选不应返回版本: %+v", selected)
	}
	if !errors.Is(
		err,
		ErrLessonPlanWordRestoreSnapshotNotFound,
	) {
		t.Fatalf("缺失快照错误类型不正确: %v", err)
	}
}

func TestValidateLessonPlanWordRestoreStaleReasonAllowsSemanticDrift(
	t *testing.T,
) {
	document := &models.LessonPlanWordDocument{
		Status:       models.LessonPlanWordDocumentStatusStale,
		ErrorMessage: "平台语义正文已由其它链路修改，需要重新同步原格式Word文档",
	}

	if err := validateLessonPlanWordRestoreStaleReason(
		document,
	); err != nil {
		t.Fatalf("正文失步应允许从历史Word恢复: %v", err)
	}
}

func TestValidateLessonPlanWordRestoreStaleReasonRejectsMetadataDrift(
	t *testing.T,
) {
	document := &models.LessonPlanWordDocument{
		Status:       models.LessonPlanWordDocumentStatusStale,
		ErrorMessage: "教案标题、课程定位或课时时长已由其它链路修改，需要重新同步原格式Word文档",
	}

	err := validateLessonPlanWordRestoreStaleReason(document)
	if !errors.Is(
		err,
		ErrLessonPlanWordRestoreMetadataStale,
	) {
		t.Fatalf("课程元信息失步应被拒绝: %v", err)
	}
}

func TestValidateLessonPlanWordRestoreStaleReasonRejectsUnknownDrift(
	t *testing.T,
) {
	document := &models.LessonPlanWordDocument{
		Status:       models.LessonPlanWordDocumentStatusStale,
		ErrorMessage: "",
	}

	err := validateLessonPlanWordRestoreStaleReason(document)
	if !errors.Is(
		err,
		ErrLessonPlanWordRestoreMetadataStale,
	) {
		t.Fatalf("未知stale原因必须fail-closed: %v", err)
	}
}
