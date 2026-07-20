package repository

import (
	"testing"

	"tedna/internal/models"
)

// TestNormalizeOrgAdminEducationDomainForWrite 锁定数据库写入参数规则：
//   - 三个具体教学教育域可以写入，并统一标准化为小写；
//   - 学校管理员无论请求中是否夹带教育域，都必须写为NULL；
//   - mixed、common、空值和未知角色不能作为区域管理员任命写入。
func TestNormalizeOrgAdminEducationDomainForWrite(t *testing.T) {
	tests := []struct {
		name            string
		roleType        string
		educationDomain string
		wantValue       string
		wantNil         bool
		wantErr         bool
	}{
		{
			name:            "合法K12区域任命",
			roleType:        models.OrgAdminRoleRegion,
			educationDomain: " K12 ",
			wantValue:       models.EducationDomainK12,
		},
		{
			name:            "合法职业教育区域任命",
			roleType:        models.OrgAdminRoleRegion,
			educationDomain: "VOCATIONAL",
			wantValue:       models.EducationDomainVocational,
		},
		{
			name:            "合法成人教育区域任命",
			roleType:        models.OrgAdminRoleRegion,
			educationDomain: "adult",
			wantValue:       models.EducationDomainAdult,
		},
		{
			name:            "学校管理员教育域必须写NULL",
			roleType:        models.OrgAdminRoleSchool,
			educationDomain: "vocational",
			wantNil:         true,
		},
		{
			name:            "区域管理员空教育域拒绝",
			roleType:        models.OrgAdminRoleRegion,
			educationDomain: "",
			wantErr:         true,
		},
		{
			name:            "mixed不能作为区域固定教育域",
			roleType:        models.OrgAdminRoleRegion,
			educationDomain: models.EducationDomainMixed,
			wantErr:         true,
		},
		{
			name:            "common不能作为区域固定教育域",
			roleType:        models.OrgAdminRoleRegion,
			educationDomain: models.EducationDomainCommon,
			wantErr:         true,
		},
		{
			name:            "未知管理员类型拒绝",
			roleType:        "unknown_admin",
			educationDomain: models.EducationDomainK12,
			wantErr:         true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			value, err :=
				normalizeOrgAdminEducationDomainForWrite(
					testCase.roleType,
					testCase.educationDomain,
				)

			if testCase.wantErr {
				if err == nil {
					t.Fatal("期望返回错误，实际没有错误")
				}
				return
			}

			if err != nil {
				t.Fatalf("不期望错误，实际错误=%v", err)
			}

			if testCase.wantNil {
				if value != nil {
					t.Fatalf(
						"学校管理员教育域参数必须为nil，实际=%#v",
						value,
					)
				}
				return
			}

			actual, ok := value.(string)
			if !ok {
				t.Fatalf(
					"区域管理员教育域参数类型错误，实际=%T",
					value,
				)
			}
			if actual != testCase.wantValue {
				t.Fatalf(
					"标准化结果=%q，期望=%q",
					actual,
					testCase.wantValue,
				)
			}
		})
	}
}
