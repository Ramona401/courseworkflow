package repository

import (
	"errors"
	"testing"

	"tedna/internal/models"
)

func regionCandidate(
	id string,
	name string,
	domain string,
) regionAdminEducationDomainCandidate {
	return regionAdminEducationDomainCandidate{
		OrganizationID:   id,
		OrganizationName: name,
		EducationDomain:  domain,
	}
}

func TestResolveRegionAdminEducationContextSupportsThreeDomains(
	t *testing.T,
) {
	domains := []string{
		models.EducationDomainK12,
		models.EducationDomainVocational,
		models.EducationDomainAdult,
	}

	for _, domain := range domains {
		result, err :=
			resolveRegionAdminEducationContextFromCandidates(
				[]regionAdminEducationDomainCandidate{
					regionCandidate(
						"region-"+domain,
						"测试区域",
						domain,
					),
				},
			)
		if err != nil {
			t.Fatalf(
				"教育域=%s不应返回错误，实际=%v",
				domain,
				err,
			)
		}
		if result.EducationDomain != domain {
			t.Fatalf(
				"教育域=%q，期望=%q",
				result.EducationDomain,
				domain,
			)
		}
	}
}

func TestResolveRegionAdminEducationContextAllowsSameDomainRegions(
	t *testing.T,
) {
	result, err :=
		resolveRegionAdminEducationContextFromCandidates(
			[]regionAdminEducationDomainCandidate{
				regionCandidate(
					"region-a",
					"第一区域",
					models.EducationDomainVocational,
				),
				regionCandidate(
					"region-b",
					"第二区域",
					" VOCATIONAL ",
				),
			},
		)
	if err != nil {
		t.Fatalf("多个同域区域任命应允许，实际错误=%v", err)
	}

	if result.EducationDomain != models.EducationDomainVocational {
		t.Fatalf(
			"教育域=%q，期望vocational",
			result.EducationDomain,
		)
	}

	if result.OrganizationID != "region-a" {
		t.Fatalf(
			"品牌组织应使用稳定第一条，实际=%q",
			result.OrganizationID,
		)
	}
}

func TestResolveRegionAdminEducationContextFailsClosed(t *testing.T) {
	tests := []struct {
		name       string
		candidates []regionAdminEducationDomainCandidate
	}{
		{
			name:       "没有任命",
			candidates: nil,
		},
		{
			name: "教育域为空",
			candidates: []regionAdminEducationDomainCandidate{
				regionCandidate("region-a", "区域A", ""),
			},
		},
		{
			name: "mixed非法",
			candidates: []regionAdminEducationDomainCandidate{
				regionCandidate(
					"region-a",
					"区域A",
					models.EducationDomainMixed,
				),
			},
		},
		{
			name: "common非法",
			candidates: []regionAdminEducationDomainCandidate{
				regionCandidate(
					"region-a",
					"区域A",
					models.EducationDomainCommon,
				),
			},
		},
		{
			name: "未知教育域非法",
			candidates: []regionAdminEducationDomainCandidate{
				regionCandidate(
					"region-a",
					"区域A",
					"general",
				),
			},
		},
		{
			name: "多个不同教育域冲突",
			candidates: []regionAdminEducationDomainCandidate{
				regionCandidate(
					"region-a",
					"区域A",
					models.EducationDomainK12,
				),
				regionCandidate(
					"region-b",
					"区域B",
					models.EducationDomainAdult,
				),
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			result, err :=
				resolveRegionAdminEducationContextFromCandidates(
					testCase.candidates,
				)

			if result != nil {
				t.Fatalf(
					"异常状态不应返回可用上下文，实际=%#v",
					result,
				)
			}

			if !errors.Is(
				err,
				ErrRegionAdminEducationDomainNotReady,
			) {
				t.Fatalf(
					"错误=%v，期望ErrRegionAdminEducationDomainNotReady",
					err,
				)
			}
		})
	}
}

func TestRegionAdminUserInfoDefaultsToNotReady(t *testing.T) {
	info := (&models.User{
		Role: models.RoleRegionAdmin,
	}).ToUserInfo()

	if info.EducationDomainReady {
		t.Fatal("区域管理员在任命解析前不能标记为ready")
	}

	if info.EducationDomain != "" {
		t.Fatalf(
			"区域管理员初始教育域必须为空，实际=%q",
			info.EducationDomain,
		)
	}

	if info.EducationDomainError !=
		models.EducationDomainNotReadyMessage {
		t.Fatalf(
			"异常提示=%q，期望=%q",
			info.EducationDomainError,
			models.EducationDomainNotReadyMessage,
		)
	}
}
