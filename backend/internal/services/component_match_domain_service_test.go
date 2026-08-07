package services

import (
	"errors"
	"testing"

	"tedna/internal/models"
)

func TestResolveComponentMatchDomainOrdinaryUsesTrustedDomain(
	t *testing.T,
) {
	actor := &AssistantActorContext{
		UserID:          "teacher-1",
		Role:            models.RoleOperator,
		EducationDomain: models.EducationDomainVocational,
	}

	got, err := resolveComponentMatchDomain(
		actor,
		models.EducationDomainAdult,
	)

	if err != nil {
		t.Fatalf(
			"普通Actor匹配域不应报错: %v",
			err,
		)
	}

	if got != models.EducationDomainVocational {
		t.Fatalf(
			"普通Actor必须使用可信域，got=%q",
			got,
		)
	}
}

func TestResolveComponentMatchDomainMixedRequiresConcreteDomain(
	t *testing.T,
) {
	actor := &AssistantActorContext{
		UserID:          "admin-1",
		Role:            models.RoleAdmin,
		EducationDomain: models.EducationDomainMixed,
	}

	_, err := resolveComponentMatchDomain(
		actor,
		"",
	)

	if !errors.Is(
		err,
		ErrComponentEducationDomainRequired,
	) {
		t.Fatalf(
			"mixed未选目标域应返回required，got=%v",
			err,
		)
	}

	_, err = resolveComponentMatchDomain(
		actor,
		models.EducationDomainCommon,
	)

	if !errors.Is(
		err,
		ErrComponentEducationDomainInvalid,
	) {
		t.Fatalf(
			"common不能作为匹配当前域，got=%v",
			err,
		)
	}
}

func TestResolveComponentMatchDomainMixedAcceptsTeachingDomain(
	t *testing.T,
) {
	actor := &AssistantActorContext{
		UserID:          "admin-1",
		Role:            models.RoleAdmin,
		EducationDomain: models.EducationDomainMixed,
	}

	got, err := resolveComponentMatchDomain(
		actor,
		models.EducationDomainAdult,
	)

	if err != nil {
		t.Fatalf(
			"mixed显式adult不应报错: %v",
			err,
		)
	}

	if got != models.EducationDomainAdult {
		t.Fatalf(
			"mixed目标域错误，got=%q",
			got,
		)
	}
}

func TestResolveComponentMatchDomainInvalidActorFailsClosed(
	t *testing.T,
) {
	actor := &AssistantActorContext{
		UserID:          "teacher-1",
		Role:            models.RoleOperator,
		EducationDomain: "",
	}

	_, err := resolveComponentMatchDomain(
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
