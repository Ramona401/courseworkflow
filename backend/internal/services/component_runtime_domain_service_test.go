package services

import (
	"context"
	"reflect"
	"testing"

	"tedna/internal/models"
)

func TestLessonComponentDomainContext(
	t *testing.T,
) {
	ctx := withLessonComponentDomain(
		context.Background(),
		" Vocational ",
	)

	got, ok :=
		lessonComponentDomainFromContext(
			ctx,
		)

	if !ok {
		t.Fatal(
			"合法教学域应能从Context读取",
		)
	}

	if got != models.EducationDomainVocational {
		t.Fatalf(
			"got=%q want=%q",
			got,
			models.EducationDomainVocational,
		)
	}
}

func TestLessonComponentDomainContextFailsClosed(
	t *testing.T,
) {
	values := []string{
		"",
		models.EducationDomainCommon,
		models.EducationDomainMixed,
		"unknown",
	}

	for _, value := range values {
		ctx := withLessonComponentDomain(
			context.Background(),
			value,
		)

		if got, ok :=
			lessonComponentDomainFromContext(
				ctx,
			); ok {
			t.Fatalf(
				"非法运行域必须fail-closed，input=%q got=%q",
				value,
				got,
			)
		}
	}
}

func TestParseRuntimeStageTypes(
	t *testing.T,
) {
	got, ok := parseRuntimeStageTypes(
		`[
			"pedagogy",
			"activity_design",
			"pedagogy"
		]`,
	)

	if !ok {
		t.Fatal(
			"合法阶段类型不应失败",
		)
	}

	want := []string{
		models.LibPedagogy,
		models.LibActivityDesign,
	}

	if !reflect.DeepEqual(
		got,
		want,
	) {
		t.Fatalf(
			"got=%v want=%v",
			got,
			want,
		)
	}

	if _, ok := parseRuntimeStageTypes(
		`["unknown_library"]`,
	); ok {
		t.Fatal(
			"未知library_type必须fail-closed",
		)
	}
}

func TestFilterComponentGroupsByLibraryTypes(
	t *testing.T,
) {
	groups := []*models.MatchedComponentGroup{
		{
			LibraryType: models.LibPedagogy,
			Components: []*models.MatchedComponent{
				{
					ID: "component-a",
				},
			},
		},
		{
			LibraryType: models.LibTeachingTool,
			Components: []*models.MatchedComponent{
				{
					ID: "component-b",
				},
			},
		},
	}

	filtered :=
		filterComponentGroupsByLibraryTypes(
			groups,
			[]string{
				models.LibPedagogy,
			},
		)

	if len(filtered) != 1 {
		t.Fatalf(
			"应只保留1组，got=%d",
			len(filtered),
		)
	}

	if filtered[0].LibraryType !=
		models.LibPedagogy {
		t.Fatalf(
			"保留类型错误：%s",
			filtered[0].LibraryType,
		)
	}

	if countRuntimeComponents(
		filtered,
	) != 1 {
		t.Fatal(
			"实际加载数量应为1",
		)
	}
}
