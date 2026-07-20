package services

import (
	"errors"
	"testing"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// TestNormalizeRequiredRegionAdminEducationDomain 锁定请求值标准化和错误分层。
func TestNormalizeRequiredRegionAdminEducationDomain(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      string
		wantError error
	}{
		{
			name:  "K12允许并标准化",
			input: " K12 ",
			want:  models.EducationDomainK12,
		},
		{
			name:  "职业教育允许并标准化",
			input: "VOCATIONAL",
			want:  models.EducationDomainVocational,
		},
		{
			name:  "成人教育允许",
			input: models.EducationDomainAdult,
			want:  models.EducationDomainAdult,
		},
		{
			name:      "空值返回必填错误",
			input:     "   ",
			wantError: ErrOrgAdminEducationDomainRequired,
		},
		{
			name:      "mixed返回非法错误",
			input:     models.EducationDomainMixed,
			wantError: ErrOrgAdminEducationDomainInvalid,
		},
		{
			name:      "common返回非法错误",
			input:     models.EducationDomainCommon,
			wantError: ErrOrgAdminEducationDomainInvalid,
		},
		{
			name:      "未知值返回非法错误",
			input:     "general",
			wantError: ErrOrgAdminEducationDomainInvalid,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			actual, err :=
				normalizeRequiredRegionAdminEducationDomain(
					testCase.input,
				)

			if testCase.wantError != nil {
				if !errors.Is(err, testCase.wantError) {
					t.Fatalf(
						"错误=%v，期望=%v",
						err,
						testCase.wantError,
					)
				}
				return
			}

			if err != nil {
				t.Fatalf("不期望错误，实际错误=%v", err)
			}
			if actual != testCase.want {
				t.Fatalf(
					"标准化结果=%q，期望=%q",
					actual,
					testCase.want,
				)
			}
		})
	}
}

// TestValidateRegionAdminEducationDomainAssignment 锁定区域管理员任命核心规则。
//
// 覆盖：
//   - 合法K12、职业教育、成人教育任命；
//   - 同域多区域允许；
//   - 跨域任命拒绝；
//   - 存在历史空值或非法任命时拒绝；
//   - 区域没有所选类型学校时拒绝；
//   - 非具体教学域拒绝；
//   - 状态查询异常性空结果按fail-closed拒绝。
func TestValidateRegionAdminEducationDomainAssignment(
	t *testing.T,
) {
	emptyState := func() *repository.UserRegionAdminEducationDomainState {
		return &repository.UserRegionAdminEducationDomainState{
			Domains: []string{},
		}
	}

	tests := []struct {
		name             string
		educationDomain  string
		availableDomains []string
		state            *repository.UserRegionAdminEducationDomainState
		wantError        error
	}{
		{
			name:             "合法K12任命",
			educationDomain:  models.EducationDomainK12,
			availableDomains: []string{models.EducationDomainK12},
			state:            emptyState(),
		},
		{
			name:             "合法职业教育任命",
			educationDomain:  models.EducationDomainVocational,
			availableDomains: []string{models.EducationDomainVocational},
			state:            emptyState(),
		},
		{
			name:             "合法成人教育任命",
			educationDomain:  models.EducationDomainAdult,
			availableDomains: []string{models.EducationDomainAdult},
			state:            emptyState(),
		},
		{
			name:             "同域多区域允许",
			educationDomain:  models.EducationDomainK12,
			availableDomains: []string{models.EducationDomainK12},
			state: &repository.UserRegionAdminEducationDomainState{
				Domains: []string{models.EducationDomainK12},
			},
		},
		{
			name:             "跨域任命拒绝",
			educationDomain:  models.EducationDomainK12,
			availableDomains: []string{models.EducationDomainK12},
			state: &repository.UserRegionAdminEducationDomainState{
				Domains: []string{
					models.EducationDomainVocational,
				},
			},
			wantError: ErrOrgAdminEducationDomainConflict,
		},
		{
			name:             "存在历史空教育域任命时拒绝",
			educationDomain:  models.EducationDomainK12,
			availableDomains: []string{models.EducationDomainK12},
			state: &repository.UserRegionAdminEducationDomainState{
				Domains:         []string{},
				HasUnconfigured: true,
			},
			wantError: ErrOrgAdminEducationDomainUnconfigured,
		},
		{
			name:             "已有任命状态含非法域时拒绝",
			educationDomain:  models.EducationDomainK12,
			availableDomains: []string{models.EducationDomainK12},
			state: &repository.UserRegionAdminEducationDomainState{
				Domains: []string{"invalid-domain"},
			},
			wantError: ErrOrgAdminEducationDomainUnconfigured,
		},
		{
			name:             "区域不存在该学校类型时拒绝",
			educationDomain:  models.EducationDomainVocational,
			availableDomains: []string{models.EducationDomainK12},
			state:            emptyState(),
			wantError: ErrOrgAdminEducationDomainUnavailable,
		},
		{
			name:             "非法目标教育域拒绝",
			educationDomain:  models.EducationDomainMixed,
			availableDomains: []string{models.EducationDomainMixed},
			state:            emptyState(),
			wantError:        ErrOrgAdminEducationDomainInvalid,
		},
		{
			name:             "任命状态为空指针时fail-closed",
			educationDomain:  models.EducationDomainAdult,
			availableDomains: []string{models.EducationDomainAdult},
			state:            nil,
			wantError: ErrOrgAdminEducationDomainUnconfigured,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			err := validateRegionAdminEducationDomainAssignment(
				testCase.educationDomain,
				testCase.availableDomains,
				testCase.state,
			)

			if testCase.wantError == nil {
				if err != nil {
					t.Fatalf("不期望错误，实际错误=%v", err)
				}
				return
			}

			if !errors.Is(err, testCase.wantError) {
				t.Fatalf(
					"错误=%v，期望=%v",
					err,
					testCase.wantError,
				)
			}
		})
	}
}

// TestContainsRegionAdminEducationDomain 确保匹配只做大小写和空白规范化，
// 不会把非法值通过默认K12逻辑放行。
func TestContainsRegionAdminEducationDomain(t *testing.T) {
	domains := []string{
		" K12 ",
		"VOCATIONAL",
		models.EducationDomainAdult,
	}

	if !containsRegionAdminEducationDomain(
		domains,
		models.EducationDomainK12,
	) {
		t.Fatal("应匹配K12")
	}

	if !containsRegionAdminEducationDomain(
		domains,
		models.EducationDomainVocational,
	) {
		t.Fatal("应匹配职业教育")
	}

	if !containsRegionAdminEducationDomain(
		domains,
		models.EducationDomainAdult,
	) {
		t.Fatal("应匹配成人教育")
	}

	if containsRegionAdminEducationDomain(
		domains,
		models.EducationDomainMixed,
	) {
		t.Fatal("mixed不应匹配任何具体教学教育域")
	}

	if containsRegionAdminEducationDomain(
		domains,
		"invalid-domain",
	) {
		t.Fatal("非法教育域不应匹配")
	}
}
