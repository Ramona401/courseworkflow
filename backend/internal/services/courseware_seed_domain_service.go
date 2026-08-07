package services

// courseware_seed_domain_service.go — 课件组件教育域种子入口。
//
// 当前内置课件组件全部来自既有K12课件体系，因此本入口明确固定：
//   - 只检查K12组件是否已经存在；
//   - 只创建K12组件；
//   - force=true只删除并重建K12组件；
//   - vocational、adult和common组件永不受种子操作影响；
//   - 不创建未经产品定义的职教或成人教育正式内容；
//   - 模板仍沿用既有独立模板种子规则，不增加模板教育域。
//
// 旧seedComponents方法继续保留以降低改动面，但正式HTTP入口已切换到本文件。

import (
	"context"
	"fmt"

	"tedna/internal/models"
	"tedna/internal/repository"
)

const coursewareSeedEducationDomain =
	models.EducationDomainK12

// SeedAllForEducationDomain 执行域安全的课件工坊种子填充。
func (s *CoursewareSeedService) SeedAllForEducationDomain(
	ctx context.Context,
	force bool,
) (*SeedResult, error) {
	result := &SeedResult{}

	componentCount, err :=
		s.seedK12CoursewareComponents(
			ctx,
			force,
		)
	if err != nil {
		result.Errors = append(
			result.Errors,
			"组件种子失败: "+err.Error(),
		)
	}

	result.ComponentsCreated =
		componentCount

	// 模板不是本上下文的教育域资源，不增加education_domain。
	//
	// 继续保护生产库中已经通过SQL维护的正式模板；
	// 只有全新部署且模板表完全为空时才写入基础兜底模板。
	existingTemplates, listErr :=
		repository.ListCWTemplates(
			ctx,
			false,
		)

	switch {
	case listErr != nil:
		result.Errors = append(
			result.Errors,
			"查询模板失败: "+listErr.Error(),
		)

	case len(existingTemplates) > 0:
		seedLog.Info(
			"已有模板，跳过模板种子",
			"count",
			len(existingTemplates),
		)

		result.TemplatesSkipped =
			"模板已通过SQL手动管理，种子服务不再覆盖"

	default:
		templateCount, templateErr :=
			s.seedTemplates(
				ctx,
				false,
			)

		if templateErr != nil {
			result.Errors = append(
				result.Errors,
				"模板种子失败: "+
					templateErr.Error(),
			)
		}

		result.TemplatesCreated =
			templateCount
	}

	return result, nil
}

// seedK12CoursewareComponents 填充K12内置课件组件。
func (s *CoursewareSeedService) seedK12CoursewareComponents(
	ctx context.Context,
	force bool,
) (int, error) {
	_, total, err :=
		repository.
			ListCWComponentsForEducationDomain(
				ctx,
				models.EducationDomainMixed,
				coursewareSeedEducationDomain,
				"",
				"",
				"",
				nil,
				1,
				0,
			)
	if err != nil {
		return 0, fmt.Errorf(
			"查询K12组件失败: %w",
			err,
		)
	}

	if total > 0 && !force {
		seedLog.Info(
			"已有K12组件，跳过种子填充",
			"education_domain",
			coursewareSeedEducationDomain,
			"count",
			total,
		)

		return 0, nil
	}

	if force {
		if err := deleteAllK12CoursewareComponents(
			ctx,
		); err != nil {
			return 0, err
		}
	}

	components := buildSeedComponents()
	created := 0

	for _, component := range components {
		if component == nil {
			continue
		}

		if err := repository.
			CreateCWComponentWithEducationDomain(
				ctx,
				component,
				coursewareSeedEducationDomain,
			); err != nil {
			seedLog.Error(
				"创建K12组件失败",
				"name", component.Name,
				"education_domain",
				coursewareSeedEducationDomain,
				"error", err,
			)

			continue
		}

		created++
	}

	seedLog.Info(
		"K12课件组件创建完成",
		"education_domain",
		coursewareSeedEducationDomain,
		"created",
		created,
		"total",
		len(components),
	)

	return created, nil
}

// deleteAllK12CoursewareComponents 分页删除全部K12组件。
//
// 每轮固定从offset=0读取，因为删除后剩余记录会自然前移；
// 不会扫描或删除其它教育域资源。
func deleteAllK12CoursewareComponents(
	ctx context.Context,
) error {
	deleted := 0

	for {
		items, total, err :=
			repository.
				ListCWComponentsForEducationDomain(
					ctx,
					models.EducationDomainMixed,
					coursewareSeedEducationDomain,
					"",
					"",
					"",
					nil,
					200,
					0,
				)
		if err != nil {
			return fmt.Errorf(
				"读取待删除K12组件失败: %w",
				err,
			)
		}

		if total == 0 ||
			len(items) == 0 {
			break
		}

		for _, item := range items {
			if item == nil {
				continue
			}

			if err := repository.
				DeleteCWComponentForEducationDomain(
					ctx,
					item.ID,
					models.EducationDomainMixed,
				); err != nil {
				return fmt.Errorf(
					"删除K12组件%s失败: %w",
					item.ID,
					err,
				)
			}

			deleted++
		}
	}

	seedLog.Info(
		"force模式仅清空K12组件",
		"education_domain",
		coursewareSeedEducationDomain,
		"deleted",
		deleted,
	)

	return nil
}
