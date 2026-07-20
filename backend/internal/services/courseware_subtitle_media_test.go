package services

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCoursewareSubtitleSafeFileToken(
	t *testing.T,
) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "空值",
			input: "",
			want:  "item",
		},
		{
			name:  "短ID",
			input: "a1",
			want:  "a1",
		},
		{
			name:  "标准标识",
			input: "abc-123_def",
			want:  "abc-123_def",
		},
		{
			name:  "危险字符替换",
			input: "../a b:c",
			want:  "a_b_c",
		},
	}

	for _, testCase := range tests {
		t.Run(
			testCase.name,
			func(t *testing.T) {
				got :=
					coursewareSubtitleSafeFileToken(
						testCase.input,
					)

				if got != testCase.want {
					t.Fatalf(
						"安全文件标识不一致: got=%s want=%s",
						got,
						testCase.want,
					)
				}
			},
		)
	}
}

func TestCleanupCoursewareSubtitleFiles(
	t *testing.T,
) {
	path := filepath.Join(
		t.TempDir(),
		"temporary.mp3",
	)

	if err := os.WriteFile(
		path,
		[]byte("test"),
		0600,
	); err != nil {
		t.Fatalf(
			"建立测试文件失败: %v",
			err,
		)
	}

	cleanupCoursewareSubtitleFiles(
		[]string{
			path,
			path,
			"",
		},
	)

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf(
			"测试文件应已删除，实际错误=%v",
			err,
		)
	}
}

func TestCoursewareSubtitleStringPointerEqual(
	t *testing.T,
) {
	left := "page-1"
	same := "page-1"
	different := "page-2"

	if !coursewareSubtitleStringPointerEqual(
		nil,
		nil,
	) {
		t.Fatal("两个nil应相等")
	}

	if !coursewareSubtitleStringPointerEqual(
		&left,
		&same,
	) {
		t.Fatal("相同字符串值应相等")
	}

	if coursewareSubtitleStringPointerEqual(
		&left,
		&different,
	) {
		t.Fatal("不同字符串值不应相等")
	}

	if coursewareSubtitleStringPointerEqual(
		&left,
		nil,
	) {
		t.Fatal("非nil与nil不应相等")
	}
}
