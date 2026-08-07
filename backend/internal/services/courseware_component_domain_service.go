package services

// courseware_component_domain_service.go — 课件组件管理面教育域服务。
//
// 本文件只负责组件管理API的数据库编排：
//   - 创建；
//   - 列表；
//   - 直接ID详情；
//   - 更新；
//   - 删除。
//
// 纯教育域策略位于courseware_component_domain_policy.go。
// 运行时匹配和页面组件ID复核位于
// courseware_component_domain_runtime_service.go。

import (
	"context"
	"errors"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// CreateCWComponentForActor 使用可信Actor创建课件组件。
func CreateCWComponentForActor(
	ctx context.Context,
	actor *AssistantActorContext,
	request *models.CreateCWComponentDomainRequest,
) (*models.CWComponentResource, error) {
	if request == nil {
		return nil,
			ErrCWComponentRequestRequired
	}

	if err := validateCWComponentWriteFields(
		request.Name,
		request.ComponentType,
		request.CodeContent,
		"",
	); err != nil {
		return nil, err
	}

	educationDomain, err :=
		resolveCWComponentCreationDomain(
			actor,
			request.EducationDomain,
		)
	if err != nil {
		return nil, err
	}

	component := &models.CoursewareComponent{
		Name: strings.TrimSpace(
			request.Name,
		),
		Description: strings.TrimSpace(
			request.Description,
		),
		ComponentType: strings.TrimSpace(
			request.ComponentType,
		),
		CodeContent: request.CodeContent,
		PreviewImageURL: strings.TrimSpace(
			request.PreviewImageURL,
		),
		PreviewHTML: request.PreviewHTML,
		SubjectScope: normalizeCWComponentScope(
			request.SubjectScope,
		),
		GradeScope: normalizeCWComponentScope(
			request.GradeScope,
		),
		TechDependencies: request.TechDependencies,
		Tags:             request.Tags,
		IsActive:         true,
		ReviewStatus:     models.CWCompReviewDraft,
	}

	if err := repository.
		CreateCWComponentWithEducationDomain(
			ctx,
			component,
			educationDomain,
		); err != nil {
		return nil, err
	}

	return &models.CWComponentResource{
		CoursewareComponent: component,
		EducationDomain:     educationDomain,
	}, nil
}

// ListCWComponentsForActor 按可信Actor域查询组件列表。
func ListCWComponentsForActor(
	ctx context.Context,
	actor *AssistantActorContext,
	requestedDomain string,
	componentType string,
	subjectScope string,
	gradeScope string,
	isActive *bool,
	limit int,
	offset int,
) (*models.CWComponentDomainListResponse, error) {
	readDomain, err :=
		resolveCWComponentReadDomain(actor)
	if err != nil {
		return nil, err
	}

	targetDomain, err :=
		resolveCWComponentListTarget(
			readDomain,
			requestedDomain,
		)
	if err != nil {
		return nil, err
	}

	items, total, err :=
		repository.
			ListCWComponentsForEducationDomain(
				ctx,
				readDomain,
				targetDomain,
				strings.TrimSpace(
					componentType,
				),
				strings.TrimSpace(
					subjectScope,
				),
				strings.TrimSpace(
					gradeScope,
				),
				isActive,
				limit,
				offset,
			)
	if err != nil {
		return nil, err
	}

	if items == nil {
		items =
			[]*models.CWComponentDomainListItem{}
	}

	return &models.CWComponentDomainListResponse{
		Components: items,
		Total:      total,
	}, nil
}

// GetCWComponentForActor 按可信Actor域读取直接ID详情。
//
// 异域资源和真实不存在统一返回ErrCWComponentNotFound。
func GetCWComponentForActor(
	ctx context.Context,
	actor *AssistantActorContext,
	id string,
) (*models.CWComponentResource, error) {
	readDomain, err :=
		resolveCWComponentReadDomain(actor)
	if err != nil {
		return nil, err
	}

	resource, err :=
		repository.
			GetCWComponentByIDForEducationDomain(
				ctx,
				strings.TrimSpace(id),
				readDomain,
			)

	if errors.Is(
		err,
		repository.ErrCWComponentDomainNotFound,
	) {
		return nil,
			ErrCWComponentNotFound
	}

	return resource, err
}

// loadManageableCWComponent 加载admin可治理的目标组件。
//
// common对具体教学域只读，但mixed管理上下文可以受控治理。
// 不存在、异域和不可修改资源统一按不存在处理。
func loadManageableCWComponent(
	ctx context.Context,
	actor *AssistantActorContext,
	id string,
) (*models.CWComponentResource, string, error) {
	if actor == nil ||
		actor.Role != models.RoleAdmin {
		return nil, "",
			ErrCWComponentEducationDomainForbidden
	}

	readDomain, err :=
		resolveCWComponentReadDomain(actor)
	if err != nil {
		return nil, "", err
	}

	resource, err :=
		repository.
			GetCWComponentByIDForEducationDomain(
				ctx,
				strings.TrimSpace(id),
				readDomain,
			)

	if errors.Is(
		err,
		repository.ErrCWComponentDomainNotFound,
	) {
		return nil, "",
			ErrCWComponentNotFound
	}

	if err != nil {
		return nil, "", err
	}

	if !cwComponentCanMutate(
		readDomain,
		resource.EducationDomain,
	) {
		return nil, "",
			ErrCWComponentNotFound
	}

	return resource, readDomain, nil
}

// UpdateCWComponentForActor 更新组件内容，不允许修改教育域。
func UpdateCWComponentForActor(
	ctx context.Context,
	actor *AssistantActorContext,
	id string,
	request *models.UpdateCWComponentRequest,
) error {
	if request == nil {
		return ErrCWComponentRequestRequired
	}

	if err := validateCWComponentWriteFields(
		request.Name,
		request.ComponentType,
		request.CodeContent,
		request.ReviewStatus,
	); err != nil {
		return err
	}

	resource, readDomain, err :=
		loadManageableCWComponent(
			ctx,
			actor,
			id,
		)
	if err != nil {
		return err
	}

	request.Name = strings.TrimSpace(
		request.Name,
	)
	request.Description = strings.TrimSpace(
		request.Description,
	)
	request.ComponentType = strings.TrimSpace(
		request.ComponentType,
	)
	request.PreviewImageURL = strings.TrimSpace(
		request.PreviewImageURL,
	)
	request.SubjectScope =
		normalizeCWComponentScope(
			request.SubjectScope,
		)
	request.GradeScope =
		normalizeCWComponentScope(
			request.GradeScope,
		)

	err = repository.
		UpdateCWComponentForEducationDomain(
			ctx,
			resource.ID,
			readDomain,
			request,
		)

	if errors.Is(
		err,
		repository.ErrCWComponentDomainNotFound,
	) {
		return ErrCWComponentNotFound
	}

	return err
}

// DeleteCWComponentForActor 删除admin有权治理的组件。
func DeleteCWComponentForActor(
	ctx context.Context,
	actor *AssistantActorContext,
	id string,
) error {
	resource, readDomain, err :=
		loadManageableCWComponent(
			ctx,
			actor,
			id,
		)
	if err != nil {
		return err
	}

	err = repository.
		DeleteCWComponentForEducationDomain(
			ctx,
			resource.ID,
			readDomain,
		)

	if errors.Is(
		err,
		repository.ErrCWComponentDomainNotFound,
	) {
		return ErrCWComponentNotFound
	}

	return err
}
