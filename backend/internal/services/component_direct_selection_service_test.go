package services

import (
	"errors"
	"reflect"
	"testing"

	"tedna/internal/models"
)

func TestParseStageAllowedComponentTypes(
	t *testing.T,
) {
	got, err := parseStageAllowedComponentTypes(
		`[
			"pedagogy",
			"activity_design",
			"pedagogy"
		]`,
	)

	if err != nil {
		t.Fatalf(
			"合法阶段组件类型不应报错: %v",
			err,
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
}

func TestParseStageAllowedComponentTypesEmpty(
	t *testing.T,
) {
	got, err :=
		parseStageAllowedComponentTypes(
			"[]",
		)

	if err != nil {
		t.Fatalf(
			"空数组不应报错: %v",
			err,
		)
	}

	if len(got) != 0 {
		t.Fatalf(
			"空数组应返回空类型，got=%v",
			got,
		)
	}
}

func TestParseStageAllowedComponentTypesInvalidJSON(
	t *testing.T,
) {
	_, err :=
		parseStageAllowedComponentTypes(
			`["pedagogy"`,
		)

	if !errors.Is(
		err,
		ErrComponentSelectionInvalid,
	) {
		t.Fatalf(
			"非法JSON必须fail-closed，got=%v",
			err,
		)
	}
}

func TestParseStageAllowedComponentTypesUnknownType(
	t *testing.T,
) {
	_, err :=
		parseStageAllowedComponentTypes(
			`["unknown_library"]`,
		)

	if !errors.Is(
		err,
		ErrComponentSelectionInvalid,
	) {
		t.Fatalf(
			"未知组件类型必须fail-closed，got=%v",
			err,
		)
	}
}
