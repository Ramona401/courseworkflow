package services

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCoursewareSourceUploadErrorClassification(t *testing.T) {
	cause := errors.New("zip: not a valid zip file")
	err := newCoursewareSourceUploadError(
		CoursewareSourceUploadInvalidFormat,
		"无法读取这个Word文件",
		cause,
	)

	uploadErr, ok := AsCoursewareSourceUploadError(err)
	if !ok {
		t.Fatal("expected upload error classification")
	}
	if uploadErr.Kind != CoursewareSourceUploadInvalidFormat {
		t.Fatalf("unexpected kind: %s", uploadErr.Kind)
	}
	if uploadErr.Error() != "无法读取这个Word文件" {
		t.Fatalf("unexpected public message: %q", uploadErr.Error())
	}
	if !errors.Is(err, cause) {
		t.Fatal("expected wrapped cause to remain available to backend")
	}
	if strings.Contains(uploadErr.Error(), "zip:") {
		t.Fatal("public error must not expose zip internals")
	}
}

func TestExtractDocContentRejectsInvalidZip(t *testing.T) {
	path := filepath.Join(
		t.TempDir(),
		"fake.docx",
	)

	if err := os.WriteFile(
		path,
		[]byte("this is not a zip archive"),
		0600,
	); err != nil {
		t.Fatal(err)
	}

	service := &CoursewarePPTService{}
	result, err := service.ExtractDocContent(path)
	if err == nil {
		t.Fatal("expected invalid DOCX to fail")
	}
	if result != nil {
		t.Fatalf("expected nil result, got %#v", result)
	}
}
