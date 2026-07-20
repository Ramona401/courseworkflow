package services

// organization_education_domain_service.go — 组织教育域只读业务逻辑
//
// 业务规则：
//   - region的教育域固定为mixed；
//   - school的教育域在创建时确定为k12/vocational/adult；
//   - 创建成功后，任何普通业务都不能修改组织教育域；
//   - 管理接口继续提供组织教育域只读列表；
//   - 旧PUT接口保留兼容路由，但对真实组织统一返回409 Conflict；
//   - Service不再调用任何教育域更新或课程目录写入Repository。

import (
	"context"
	"errors"

	"tedna/internal/models"
	"tedna/internal/repository"
)

var (
	// ErrOrganizationDomainImmutable 表示组织教育域已经永久锁定。
	ErrOrganizationDomainImmutable = errors.New(
		"组织教育域创建后不可修改",
	)

	// ErrOrganizationDomainInvalidType 表示数据库存在未知组织类型。
	// 正常生产数据只应存在region或school。
	ErrOrganizationDomainInvalidType = errors.New(
		"无效的组织类型",
	)
)

type OrganizationEducationDomainService struct{}

func NewOrganizationEducationDomainService() *OrganizationEducationDomainService {
	return &OrganizationEducationDomainService{}
}

// ListOrganizations 列出全部组织教育域。
func (s *OrganizationEducationDomainService) ListOrganizations(
	ctx context.Context,
) (*models.OrganizationEducationDomainListResponse, error) {
	items, err := repository.ListOrganizationEducationDomains(ctx)
	if err != nil {
		return nil, err
	}

	if items == nil {
		items = []*models.OrganizationEducationDomainItem{}
	}

	return &models.OrganizationEducationDomainListResponse{
		Organizations: items,
		Total:         len(items),
	}, nil
}

// RejectUpdate 拒绝旧教育域修改接口。
//
// 保留组织存在性检查的原因：
//   - 随机或已经删除的组织ID继续返回404；
//   - 真实区域或学校统一返回409；
//   - 接口不解析、不接受也不执行任何目标教育域值；
//   - 不写审计日志，因为没有发生状态变更。
func (s *OrganizationEducationDomainService) RejectUpdate(
	ctx context.Context,
	organizationID string,
) error {
	existing, err :=
		repository.GetOrganizationEducationDomainByID(
			ctx,
			organizationID,
		)
	if err != nil {
		return err
	}

	return validateOrganizationEducationDomainUpdatePolicy(
		existing.Type,
	)
}

// validateOrganizationEducationDomainUpdatePolicy 是不可变策略的纯函数。
//
// 单独保留纯函数便于无数据库单元测试锁定以下规则：
//   - 学校不能修改；
//   - 区域不能修改；
//   - 未知组织类型不能被当作正常组织放行。
func validateOrganizationEducationDomainUpdatePolicy(
	organizationType string,
) error {
	switch organizationType {
	case models.OrgTypeSchool, models.OrgTypeRegion:
		return ErrOrganizationDomainImmutable
	default:
		return ErrOrganizationDomainInvalidType
	}
}
