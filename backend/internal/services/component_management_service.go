package services

// component_management_service.go — 组件管理CRUD教育域业务层。
//
// 旧ComponentService方法暂时保留给尚未完成教育域收口的内部调用链。
// 本文件中的ForActor方法将在Handler最终接线时统一使用。

import (
	"context"
	"errors"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

var (
	ErrComponentInvalidInjectionMode = errors.New("无效的注入模式，可选值：silent/recommend/on_demand")

	ErrComponentInvalidScope = errors.New("无效的可见范围，可选值：global/region/school/group/personal")

	ErrComponentNotReviewable = errors.New("该组件不在待审核状态")
)

// resolveComponentManagementListTarget 解析mixed管理列表精确筛选域。
//
// 普通教学Actor忽略客户端education_domain，防止客户端扩大读取范围。
// mixed管理Actor可不筛选，也可筛选四种合法资源域。
func resolveComponentManagementListTarget(
	readDomain string,
	requestedDomain string,
) (string, error) {
	requestedDomain = strings.ToLower(
		strings.TrimSpace(requestedDomain),
	)

	if readDomain != models.EducationDomainMixed {
		return "", nil
	}

	if requestedDomain == "" {
		return "", nil
	}

	if !models.IsResourceEducationDomain(
		requestedDomain,
	) {
		return "",
			ErrComponentEducationDomainInvalid
	}

	return requestedDomain, nil
}

// componentDomainManageAllowed 判断可信管理域能否修改目标组件。
//
// 普通Actor只能修改完全同域资源，不能修改common。
// mixed管理Actor可管理全部合法资源域，但不能管理非法或mixed资源。
func componentDomainManageAllowed(
	currentDomain string,
	componentDomain string,
) bool {
	currentDomain = strings.ToLower(
		strings.TrimSpace(currentDomain),
	)

	componentDomain = strings.ToLower(
		strings.TrimSpace(componentDomain),
	)

	if currentDomain == models.EducationDomainMixed {
		return models.IsResourceEducationDomain(
			componentDomain,
		)
	}

	if !models.IsTeachingEducationDomain(
		currentDomain,
	) {
		return false
	}

	return componentDomain == currentDomain
}

func validateComponentWriteRequest(
	libraryType string,
	displayLabel string,
	injectionMode string,
	scope string,
	requireLibraryType bool,
) error {
	libraryType = strings.TrimSpace(
		libraryType,
	)

	if requireLibraryType &&
		libraryType == "" {
		return ErrComponentLibTypeRequired
	}

	if requireLibraryType &&
		!models.IsValidLibraryType(
			libraryType,
		) {
		return ErrComponentLibTypeInvalid
	}

	if strings.TrimSpace(
		displayLabel,
	) == "" {
		return ErrComponentLabelRequired
	}

	if injectionMode != "" &&
		!models.IsValidInjectionMode(
			injectionMode,
		) {
		return ErrComponentInvalidInjectionMode
	}

	if scope != "" &&
		!models.IsValidScope(scope) {
		return ErrComponentInvalidScope
	}

	return nil
}

// CreateComponentForActor 使用可信Actor教育域创建组件。
func (s *ComponentService) CreateComponentForActor(
	ctx context.Context,
	actor *AssistantActorContext,
	req *models.CreateComponentRequest,
) (*models.LessonPlanComponent, error) {
	if req == nil {
		return nil,
			ErrComponentLabelRequired
	}

	if err := validateComponentWriteRequest(
		req.LibraryType,
		req.DisplayLabel,
		req.InjectionMode,
		req.Scope,
		true,
	); err != nil {
		return nil, err
	}

	educationDomain, err :=
		ResolveComponentCreationDomain(
			actor,
			req.EducationDomain,
		)
	if err != nil {
		return nil, err
	}

	createdBy := actor.UserID

	component := &models.LessonPlanComponent{
		EducationDomain: educationDomain,
		LibraryType: strings.TrimSpace(
			req.LibraryType,
		),
		Subject: strings.TrimSpace(
			req.Subject,
		),
		GradeRange: strings.TrimSpace(
			req.GradeRange,
		),
		Tags:           req.Tags,
		InjectionMode:  req.InjectionMode,
		DisplayLabel:   strings.TrimSpace(req.DisplayLabel),
		DesignLogic:    req.DesignLogic,
		ExampleSnippet: req.ExampleSnippet,
		FullGuide:      req.FullGuide,
		Content:        req.Content,
		Source:         "manual",
		Scope:          req.Scope,
		ScopeRefID:     req.ScopeRefID,
		CreatedBy:      &createdBy,
		ReviewStatus:   models.ComponentReviewApproved,
		Status:         "active",
	}

	if err := repository.
		CreateComponentWithEducationDomain(
			ctx,
			component,
		); err != nil {
		return nil, err
	}

	return component, nil
}

// ListComponentsForActor 按可信Actor教育域获取组件列表。
func (s *ComponentService) ListComponentsForActor(
	ctx context.Context,
	actor *AssistantActorContext,
	requestedDomain string,
	libraryType string,
	subject string,
	reviewStatus string,
	scope string,
	limit int,
	offset int,
) (*models.ComponentListResponse, error) {
	readDomain, err :=
		ResolveComponentReadDomain(actor)
	if err != nil {
		return nil, err
	}

	targetDomain, err :=
		resolveComponentManagementListTarget(
			readDomain,
			requestedDomain,
		)
	if err != nil {
		return nil, err
	}

	items, total, err :=
		repository.ListComponentsForEducationDomain(
			ctx,
			readDomain,
			targetDomain,
			strings.TrimSpace(libraryType),
			strings.TrimSpace(subject),
			strings.TrimSpace(reviewStatus),
			strings.TrimSpace(scope),
			limit,
			offset,
		)
	if err != nil {
		return nil, err
	}

	if items == nil {
		items = []*models.ComponentListItem{}
	}

	return &models.ComponentListResponse{
		Components: items,
		Total:      total,
	}, nil
}

// GetComponentForActor 按可信Actor教育域读取组件详情。
//
// 异域组件和不存在组件统一返回ErrComponentNotFound。
func (s *ComponentService) GetComponentForActor(
	ctx context.Context,
	actor *AssistantActorContext,
	id string,
) (*models.LessonPlanComponent, error) {
	readDomain, err :=
		ResolveComponentReadDomain(actor)
	if err != nil {
		return nil, err
	}

	component, err :=
		repository.GetComponentByIDForEducationDomain(
			ctx,
			strings.TrimSpace(id),
			readDomain,
		)

	if errors.Is(
		err,
		repository.ErrComponentNotFound,
	) {
		return nil,
			ErrComponentNotFound
	}

	return component, err
}

// loadManageableComponent 加载Actor可以修改的目标组件。
func loadManageableComponent(
	ctx context.Context,
	actor *AssistantActorContext,
	id string,
) (*models.LessonPlanComponent, string, error) {
	readDomain, err :=
		ResolveComponentReadDomain(actor)
	if err != nil {
		return nil, "", err
	}

	component, err :=
		repository.GetComponentByIDForEducationDomain(
			ctx,
			strings.TrimSpace(id),
			readDomain,
		)

	if errors.Is(
		err,
		repository.ErrComponentNotFound,
	) {
		return nil, "",
			ErrComponentNotFound
	}

	if err != nil {
		return nil, "", err
	}

	if !componentDomainManageAllowed(
		readDomain,
		component.EducationDomain,
	) {
		return nil, "",
			ErrComponentNotFound
	}

	return component, readDomain, nil
}

// UpdateComponentForActor 按可信Actor域更新组件。
func (s *ComponentService) UpdateComponentForActor(
	ctx context.Context,
	actor *AssistantActorContext,
	id string,
	req *models.UpdateComponentRequest,
) error {
	if req == nil {
		return ErrComponentLabelRequired
	}

	if err := validateComponentWriteRequest(
		"",
		req.DisplayLabel,
		req.InjectionMode,
		req.Scope,
		false,
	); err != nil {
		return err
	}

	component, readDomain, err :=
		loadManageableComponent(
			ctx,
			actor,
			id,
		)
	if err != nil {
		return err
	}

	err = repository.UpdateComponentForEducationDomain(
		ctx,
		component.ID,
		readDomain,
		req,
	)

	if errors.Is(
		err,
		repository.ErrComponentNotFound,
	) {
		return ErrComponentNotFound
	}

	return err
}

// DeleteComponentForActor 按可信Actor域软删除组件。
func (s *ComponentService) DeleteComponentForActor(
	ctx context.Context,
	actor *AssistantActorContext,
	id string,
) error {
	component, readDomain, err :=
		loadManageableComponent(
			ctx,
			actor,
			id,
		)
	if err != nil {
		return err
	}

	err = repository.DeleteComponentForEducationDomain(
		ctx,
		component.ID,
		readDomain,
	)

	if errors.Is(
		err,
		repository.ErrComponentNotFound,
	) {
		return ErrComponentNotFound
	}

	return err
}

// ReviewComponentForActor 按可信Actor域审核组件。
func (s *ComponentService) ReviewComponentForActor(
	ctx context.Context,
	actor *AssistantActorContext,
	id string,
	req *models.ReviewComponentRequest,
) error {
	if req == nil {
		return ErrComponentReviewInvalid
	}

	validDecision :=
		req.Decision ==
			models.ComponentReviewApproved ||
			req.Decision ==
				models.ComponentReviewRejected

	if !validDecision {
		return ErrComponentReviewInvalid
	}

	component, readDomain, err :=
		loadManageableComponent(
			ctx,
			actor,
			id,
		)
	if err != nil {
		return err
	}

	reviewable :=
		component.ReviewStatus ==
			models.ComponentReviewCaptured ||
			component.ReviewStatus ==
				models.ComponentReviewPending

	if !reviewable {
		return ErrComponentNotReviewable
	}

	err = repository.ReviewComponentForEducationDomain(
		ctx,
		component.ID,
		readDomain,
		actor.UserID,
		req.Decision,
	)

	if errors.Is(
		err,
		repository.ErrComponentNotFound,
	) {
		return ErrComponentNotFound
	}

	return err
}
