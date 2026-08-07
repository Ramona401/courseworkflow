package services

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"tedna/internal/models"
)

func TestParseHistoricalCWComponentIDsJSON(
	t *testing.T,
) {
	tests := []struct {
		name        string
		rawJSON     string
		want        []string
		wantErrorIs error
	}{
		{
			name:    "空值",
			rawJSON: "",
			want:    []string{},
		},
		{
			name:    "JSON null",
			rawJSON: "null",
			want:    []string{},
		},
		{
			name:    "空数组",
			rawJSON: "[]",
			want:    []string{},
		},
		{
			name: "清理空ID和重复ID",
			rawJSON: `[
				" component-a ",
				"",
				"component-b",
				"component-a"
			]`,
			want: []string{
				"component-a",
				"component-b",
			},
		},
		{
			name:        "对象格式拒绝",
			rawJSON:     `{"id":"component-a"}`,
			wantErrorIs: ErrCWComponentReferenceJSONInvalid,
		},
		{
			name:        "数字数组拒绝",
			rawJSON:     `[1,2,3]`,
			wantErrorIs: ErrCWComponentReferenceJSONInvalid,
		},
		{
			name:        "损坏JSON拒绝",
			rawJSON:     `["component-a"`,
			wantErrorIs: ErrCWComponentReferenceJSONInvalid,
		},
	}

	for _, test := range tests {
		t.Run(
			test.name,
			func(t *testing.T) {
				actual, err :=
					parseHistoricalCWComponentIDsJSON(
						test.rawJSON,
					)

				if test.wantErrorIs != nil {
					if !errors.Is(
						err,
						test.wantErrorIs,
					) {
						t.Fatalf(
							"期望错误%v，实际%v",
							test.wantErrorIs,
							err,
						)
					}

					return
				}

				if err != nil {
					t.Fatalf(
						"不期望错误，实际%v",
						err,
					)
				}

				if !reflect.DeepEqual(
					actual,
					test.want,
				) {
					t.Fatalf(
						"期望%v，实际%v",
						test.want,
						actual,
					)
				}
			},
		)
	}
}

func TestEncodeHistoricalCWComponentIDsJSON(
	t *testing.T,
) {
	tests := []struct {
		name string
		ids  []string
		want string
	}{
		{
			name: "nil写为空",
			ids:  nil,
			want: "",
		},
		{
			name: "空数组写为空",
			ids:  []string{},
			want: "",
		},
		{
			name: "规范去重后编码",
			ids: []string{
				" component-a ",
				"component-b",
				"component-a",
				"",
			},
			want: `["component-a","component-b"]`,
		},
	}

	for _, test := range tests {
		t.Run(
			test.name,
			func(t *testing.T) {
				actual :=
					encodeHistoricalCWComponentIDsJSON(
						test.ids,
					)

				if actual != test.want {
					t.Fatalf(
						"期望%q，实际%q",
						test.want,
						actual,
					)
				}
			},
		)
	}
}

func TestSanitizeCoursewarePageMatchedComponentIDsForWrite(
	t *testing.T,
) {
	ctx := context.Background()

	t.Run(
		"合法K12课件空引用无需查询数据库",
		func(t *testing.T) {
			courseware := &models.Courseware{
				ID:              "courseware-k12",
				EducationDomain: models.EducationDomainK12,
			}

			actual, err :=
				sanitizeCoursewarePageMatchedComponentIDsForWrite(
					ctx,
					courseware,
					"",
				)

			if err != nil {
				t.Fatalf(
					"不期望错误，实际%v",
					err,
				)
			}

			if actual != "" {
				t.Fatalf(
					"空引用期望空字符串，实际%q",
					actual,
				)
			}
		},
	)

	t.Run(
		"nil课件fail closed",
		func(t *testing.T) {
			_, err :=
				sanitizeCoursewarePageMatchedComponentIDsForWrite(
					ctx,
					nil,
					"",
				)

			if !errors.Is(
				err,
				ErrCWComponentReferenceCoursewareRequired,
			) {
				t.Fatalf(
					"期望课件快照错误，实际%v",
					err,
				)
			}
		},
	)

	t.Run(
		"空课件ID fail closed",
		func(t *testing.T) {
			courseware := &models.Courseware{
				EducationDomain:
					models.EducationDomainK12,
			}

			_, err :=
				sanitizeCoursewarePageMatchedComponentIDsForWrite(
					ctx,
					courseware,
					"",
				)

			if !errors.Is(
				err,
				ErrCWComponentReferenceCoursewareRequired,
			) {
				t.Fatalf(
					"期望课件快照错误，实际%v",
					err,
				)
			}
		},
	)

	t.Run(
		"非法课件教育域fail closed",
		func(t *testing.T) {
			courseware := &models.Courseware{
				ID:              "courseware-invalid",
				EducationDomain: models.EducationDomainMixed,
			}

			_, err :=
				sanitizeCoursewarePageMatchedComponentIDsForWrite(
					ctx,
					courseware,
					"",
				)

			if !errors.Is(
				err,
				ErrCWComponentEducationDomainInvalid,
			) {
				t.Fatalf(
					"期望教育域错误，实际%v",
					err,
				)
			}
		},
	)

	t.Run(
		"损坏JSON在数据库查询前拒绝",
		func(t *testing.T) {
			courseware := &models.Courseware{
				ID:              "courseware-k12",
				EducationDomain: models.EducationDomainK12,
			}

			_, err :=
				sanitizeCoursewarePageMatchedComponentIDsForWrite(
					ctx,
					courseware,
					`["component-a"`,
				)

			if !errors.Is(
				err,
				ErrCWComponentReferenceJSONInvalid,
			) {
				t.Fatalf(
					"期望JSON错误，实际%v",
					err,
				)
			}
		},
	)
}
