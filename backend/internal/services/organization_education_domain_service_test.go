package services

import (
	"errors"
	"testing"

	"tedna/internal/models"
)

// TestValidateOrganizationEducationDomainUpdatePolicy
// 锁定上下文8的永久不可变规则：
//   - 学校即使请求值与当前值相同，也不能再使用旧换域接口；
//   - 区域固定为mixed，也不能使用旧换域接口；
//   - 未知组织类型按异常数据fail-closed，不作为合法组织放行。
func TestValidateOrganizationEducationDomainUpdatePolicy(
	t *testing.T,
) {
	tests := []struct {
		name             string
		organizationType string
		wantError        error
	}{
		{
			name:             "学校教育域创建后不可修改",
			organizationType: models.OrgTypeSchool,
			wantError:        ErrOrganizationDomainImmutable,
		},
		{
			name:             "区域教育域固定为mixed不可修改",
			organizationType: models.OrgTypeRegion,
			wantError:        ErrOrganizationDomainImmutable,
		},
		{
			name:             "未知组织类型fail-closed",
			organizationType: "unknown",
			wantError:        ErrOrganizationDomainInvalidType,
		},
		{
			name:             "空组织类型fail-closed",
			organizationType: "",
			wantError:        ErrOrganizationDomainInvalidType,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			err :=
				validateOrganizationEducationDomainUpdatePolicy(
					testCase.organizationType,
				)

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
