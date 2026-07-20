package repository

import (
	"testing"

	"tedna/internal/models"
)

func candidate(
	id string,
	name string,
	domain string,
	priority int,
) educationOrgCandidate {
	return educationOrgCandidate{
		ID:              id,
		Name:            name,
		EducationDomain: domain,
		SourcePriority:  priority,
		Settings:        `{"portal_modules":{"lesson_plan":true,"courseware":true,"workflow":false}}`,
	}
}

func TestResolveEducationContextVocationalOverridesMixed(t *testing.T) {
	candidates := []educationOrgCandidate{
		candidate(
			"mixed-school",
			"郯城教育局",
			models.EducationDomainMixed,
			2,
		),
		candidate(
			"vocational-school",
			"临沂市电子科技学校",
			models.EducationDomainVocational,
			2,
		),
	}

	result := resolveEducationContextFromCandidates(
		models.RoleOperator,
		candidates,
	)

	if result.EducationDomain != models.EducationDomainVocational {
		t.Fatalf(
			"教育域=%q，期望=%q",
			result.EducationDomain,
			models.EducationDomainVocational,
		)
	}

	if result.OrganizationID != "vocational-school" {
		t.Fatalf(
			"教学组织=%q，期望vocational-school",
			result.OrganizationID,
		)
	}

	if result.DomainConflict {
		t.Fatal("mixed与vocational并存不应被判定为具体教育域冲突")
	}
}

func TestResolveEducationContextK12OverridesMixed(t *testing.T) {
	candidates := []educationOrgCandidate{
		candidate(
			"mixed-school",
			"教育局",
			models.EducationDomainMixed,
			2,
		),
		candidate(
			"k12-school",
			"实验小学",
			models.EducationDomainK12,
			2,
		),
	}

	result := resolveEducationContextFromCandidates(
		models.RoleViewer,
		candidates,
	)

	if result.EducationDomain != models.EducationDomainK12 {
		t.Fatalf(
			"教育域=%q，期望=%q",
			result.EducationDomain,
			models.EducationDomainK12,
		)
	}

	if result.OrganizationID != "k12-school" {
		t.Fatalf(
			"教学组织=%q，期望k12-school",
			result.OrganizationID,
		)
	}
}

func TestResolveEducationContextDetectsConcreteDomainConflict(t *testing.T) {
	candidates := []educationOrgCandidate{
		candidate(
			"k12-school",
			"实验中学",
			models.EducationDomainK12,
			2,
		),
		candidate(
			"vocational-school",
			"职业学校",
			models.EducationDomainVocational,
			2,
		),
	}

	result := resolveEducationContextFromCandidates(
		models.RoleOperator,
		candidates,
	)

	if !result.DomainConflict {
		t.Fatal("同时属于k12和vocational时应标记DomainConflict")
	}

	if result.EducationDomain != models.EducationDomainK12 {
		t.Fatalf(
			"应按稳定候选顺序选择首个具体教育域，实际=%q",
			result.EducationDomain,
		)
	}
}

func TestPlatformManagementRolesAlwaysMixed(t *testing.T) {
	roles := []string{
		models.RoleAdmin,
		models.RoleDistrictInspector,
	}

	for _, role := range roles {
		result := resolveEducationContextFromCandidates(
			role,
			[]educationOrgCandidate{
				candidate(
					"k12-school",
					"实验学校",
					models.EducationDomainK12,
					2,
				),
			},
		)

		if result.EducationDomain != models.EducationDomainMixed {
			t.Fatalf(
				"角色=%s，教育域=%q，期望mixed",
				role,
				result.EducationDomain,
			)
		}

		if result.PortalModules[models.PortalModuleWorkflow] != true {
			t.Fatalf(
				"跨域管理身份应强制全部门户板块开启，角色=%s",
				role,
			)
		}
	}
}

func TestSeniorOperatorUsesFirstDeterministicConcreteCandidate(t *testing.T) {
	candidates := []educationOrgCandidate{
		candidate(
			"appointed-school",
			"任命学校",
			models.EducationDomainVocational,
			1,
		),
		candidate(
			"membership-school",
			"普通校籍学校",
			models.EducationDomainK12,
			2,
		),
	}

	result := resolveEducationContextFromCandidates(
		models.RoleSeniorOperator,
		candidates,
	)

	if result.OrganizationID != "appointed-school" {
		t.Fatalf(
			"学校管理员应优先正式任命学校，实际=%q",
			result.OrganizationID,
		)
	}

	if result.EducationDomain != models.EducationDomainVocational {
		t.Fatalf(
			"学校管理员教育域=%q，期望vocational",
			result.EducationDomain,
		)
	}

	if !result.DomainConflict {
		t.Fatal("正式任命学校与其它具体教育域校籍并存时应记录冲突")
	}
}

func TestNoCandidateUsesRoleCompatibleDefault(t *testing.T) {
	teacher := resolveEducationContextFromCandidates(
		models.RoleViewer,
		nil,
	)
	if teacher.EducationDomain != models.EducationDomainK12 {
		t.Fatalf(
			"无组织普通教师默认域=%q，期望k12",
			teacher.EducationDomain,
		)
	}

	manager := resolveEducationContextFromCandidates(
		models.RoleAdmin,
		nil,
	)
	if manager.EducationDomain != models.EducationDomainMixed {
		t.Fatalf(
			"无组织系统管理员默认域=%q，期望mixed",
			manager.EducationDomain,
		)
	}

	regionAdmin := resolveEducationContextFromCandidates(
		models.RoleRegionAdmin,
		nil,
	)
	if regionAdmin.EducationDomain != "" {
		t.Fatalf(
			"无任命区域管理员不能默认K12或mixed，实际=%q",
			regionAdmin.EducationDomain,
		)
	}
}
