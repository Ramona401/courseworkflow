package services

import (
	"errors"
	"reflect"
	"testing"

	"tedna/internal/models"
)

func TestResolveComponentCreationDomainOrdinaryActorUsesTrustedDomain(
	t *testing.T,
) {
	actor := &AssistantActorContext{
		UserID:          "teacher-1",
		Role:            models.RoleOperator,
		EducationDomain: models.EducationDomainVocational,
	}

	got, err := ResolveComponentCreationDomain(
		actor,
		models.EducationDomainAdult,
	)
	if err != nil {
		t.Fatalf(
			"普通Actor解析创建域不应报错: %v",
			err,
		)
	}

	if got != models.EducationDomainVocational {
		t.Fatalf(
			"普通Actor必须使用可信Actor域，got=%q",
			got,
		)
	}
}

func TestResolveComponentCreationDomainAdminRequiresExplicitDomain(
	t *testing.T,
) {
	actor := &AssistantActorContext{
		UserID:          "admin-1",
		Role:            models.RoleAdmin,
		EducationDomain: models.EducationDomainMixed,
	}

	_, err := ResolveComponentCreationDomain(
		actor,
		"",
	)
	if !errors.Is(
		err,
		ErrComponentEducationDomainRequired,
	) {
		t.Fatalf(
			"mixed系统管理员未选域应返回required，got=%v",
			err,
		)
	}
}

func TestResolveComponentCreationDomainAdminCanCreateCommon(
	t *testing.T,
) {
	actor := &AssistantActorContext{
		UserID:          "admin-1",
		Role:            models.RoleAdmin,
		EducationDomain: models.EducationDomainMixed,
	}

	got, err := ResolveComponentCreationDomain(
		actor,
		models.EducationDomainCommon,
	)
	if err != nil {
		t.Fatalf(
			"系统管理员创建common不应报错: %v",
			err,
		)
	}

	if got != models.EducationDomainCommon {
		t.Fatalf(
			"系统管理员显式common应原样保留，got=%q",
			got,
		)
	}
}

func TestResolveComponentCreationDomainNonAdminMixedForbidden(
	t *testing.T,
) {
	actor := &AssistantActorContext{
		UserID:          "region-1",
		Role:            models.RoleRegionAdmin,
		EducationDomain: models.EducationDomainMixed,
	}

	_, err := ResolveComponentCreationDomain(
		actor,
		models.EducationDomainAdult,
	)
	if !errors.Is(
		err,
		ErrComponentEducationDomainForbidden,
	) {
		t.Fatalf(
			"非系统管理员mixed Actor创建组件应拒绝，got=%v",
			err,
		)
	}
}

func TestResolveComponentCreationDomainInvalidActorFailsClosed(
	t *testing.T,
) {
	actor := &AssistantActorContext{
		UserID:          "teacher-1",
		Role:            models.RoleOperator,
		EducationDomain: "",
	}

	_, err := ResolveComponentCreationDomain(
		actor,
		models.EducationDomainK12,
	)
	if !errors.Is(
		err,
		ErrComponentEducationDomainForbidden,
	) {
		t.Fatalf(
			"空Actor教育域必须fail-closed，got=%v",
			err,
		)
	}
}

func TestResolveComponentReadDomainMixedManagementAllowed(
	t *testing.T,
) {
	actor := &AssistantActorContext{
		UserID:          "inspector-1",
		Role:            models.RoleDistrictInspector,
		EducationDomain: models.EducationDomainMixed,
	}

	got, err := ResolveComponentReadDomain(actor)
	if err != nil {
		t.Fatalf(
			"mixed管理Actor读取域不应报错: %v",
			err,
		)
	}

	if got != models.EducationDomainMixed {
		t.Fatalf(
			"mixed管理Actor读取域错误，got=%q",
			got,
		)
	}
}

func TestNormalizeUniqueComponentIDs(
	t *testing.T,
) {
	input := []string{
		" component-a ",
		"",
		"component-b",
		"component-a",
		"   ",
		"component-c",
	}

	want := []string{
		"component-a",
		"component-b",
		"component-c",
	}

	got := NormalizeUniqueComponentIDs(input)

	if !reflect.DeepEqual(got, want) {
		t.Fatalf(
			"ID清洗结果不符，got=%v want=%v",
			got,
			want,
		)
	}
}
