package services

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateCoursewareGeneratedImageFile(
	t *testing.T,
) {
	t.Parallel()

	valid :=
		&coursewareGeneratedImageFile{
			URL:      " /uploads/courseware-assets/test.jpg ",
			FileSize: 1024,
			MimeType: "image/jpg",
		}

	if err :=
		validateCoursewareGeneratedImageFile(
			valid,
		); err != nil {
		t.Fatalf(
			"valid metadata rejected: %v",
			err,
		)
	}

	if valid.URL !=
		"/uploads/courseware-assets/test.jpg" {
		t.Fatalf(
			"URL not normalized: %q",
			valid.URL,
		)
	}

	if valid.MimeType !=
		"image/jpeg" {
		t.Fatalf(
			"MIME not normalized: %q",
			valid.MimeType,
		)
	}

	tests := []struct {
		name string
		file *coursewareGeneratedImageFile
	}{
		{
			name: "nil result",
			file: nil,
		},
		{
			name: "empty URL",
			file: &coursewareGeneratedImageFile{
				FileSize: 1,
				MimeType: "image/png",
			},
		},
		{
			name: "invalid size",
			file: &coursewareGeneratedImageFile{
				URL:      "/image.png",
				FileSize: 0,
				MimeType: "image/png",
			},
		},
		{
			name: "invalid MIME",
			file: &coursewareGeneratedImageFile{
				URL:      "/image.bin",
				FileSize: 1,
				MimeType: "application/octet-stream",
			},
		},
	}

	for _, testCase := range tests {
		testCase := testCase

		t.Run(
			testCase.name,
			func(t *testing.T) {
				t.Parallel()

				if err :=
					validateCoursewareGeneratedImageFile(
						testCase.file,
					); err == nil {
					t.Fatal(
						"expected metadata validation error",
					)
				}
			},
		)
	}
}

func TestNormalizeGeneratedImageMIMEType(
	t *testing.T,
) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "jpeg with parameters",
			input:    "image/jpeg; charset=binary",
			expected: "image/jpeg",
		},
		{
			name:     "jpg alias",
			input:    "image/jpg",
			expected: "image/jpeg",
		},
		{
			name:     "png",
			input:    "image/png",
			expected: "image/png",
		},
		{
			name:     "webp",
			input:    "image/webp",
			expected: "image/webp",
		},
		{
			name:     "unsupported",
			input:    "text/html",
			expected: "",
		},
	}

	for _, testCase := range tests {
		testCase := testCase

		t.Run(
			testCase.name,
			func(t *testing.T) {
				t.Parallel()

				actual :=
					normalizeGeneratedImageMIMEType(
						testCase.input,
					)

				if actual !=
					testCase.expected {
					t.Fatalf(
						"unexpected MIME: got %q want %q",
						actual,
						testCase.expected,
					)
				}
			},
		)
	}
}

func TestDetectGeneratedImageMIMETypePrefersFileSignature(
	t *testing.T,
) {
	t.Parallel()

	filePath :=
		filepath.Join(
			t.TempDir(),
			"actual-image.part",
		)

	jpegContent := []byte{
		0xff,
		0xd8,
		0xff,
		0xe0,
		0x00,
		0x10,
		'J',
		'F',
		'I',
		'F',
		0x00,
		0x01,
		0x01,
		0x00,
		0x00,
		0x01,
		0x00,
		0x01,
		0x00,
		0x00,
	}

	if err :=
		os.WriteFile(
			filePath,
			jpegContent,
			0600,
		); err != nil {
		t.Fatalf(
			"write fixture failed: %v",
			err,
		)
	}

	actual, err :=
		detectGeneratedImageMIMEType(
			filePath,
			"image/png",
		)

	if err != nil {
		t.Fatalf(
			"detect MIME failed: %v",
			err,
		)
	}

	if actual !=
		"image/jpeg" {
		t.Fatalf(
			"file signature must win: got %q",
			actual,
		)
	}
}

func TestDetectGeneratedImageMIMETypeRejectsSpoofedHeader(
	t *testing.T,
) {
	t.Parallel()

	filePath :=
		filepath.Join(
			t.TempDir(),
			"not-image.part",
		)

	if err :=
		os.WriteFile(
			filePath,
			[]byte(
				"<html>not an image</html>",
			),
			0600,
		); err != nil {
		t.Fatalf(
			"write fixture failed: %v",
			err,
		)
	}

	_, err :=
		detectGeneratedImageMIMEType(
			filePath,
			"image/png",
		)

	if err == nil {
		t.Fatal(
			"HTML with image header must be rejected",
		)
	}
}

func TestGeneratedImageExtension(
	t *testing.T,
) {
	t.Parallel()

	tests := []struct {
		mimeType  string
		extension string
	}{
		{
			mimeType:  "image/jpeg",
			extension: ".jpg",
		},
		{
			mimeType:  "image/png",
			extension: ".png",
		},
		{
			mimeType:  "image/webp",
			extension: ".webp",
		},
	}

	for _, testCase := range tests {
		extension, err :=
			generatedImageExtension(
				testCase.mimeType,
			)

		if err != nil {
			t.Fatalf(
				"extension failed for %s: %v",
				testCase.mimeType,
				err,
			)
		}

		if extension !=
			testCase.extension {
			t.Fatalf(
				"unexpected extension for %s: got %s want %s",
				testCase.mimeType,
				extension,
				testCase.extension,
			)
		}
	}
}
