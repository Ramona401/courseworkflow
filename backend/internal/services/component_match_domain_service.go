package services

// component_match_domain_service.go — 组件匹配教育域业务层。
//
// 普通教学Actor：始终使用可信Actor教育域，忽略客户端伪造域。
// mixed管理Actor：必须显式指定k12、vocational或adult目标域。
// common只能作为资源域，不能作为本次匹配的当前教学域。

import (
	"context"
	"errors"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

var (
	ErrComponentMatchRequestRequired = errors.New("匹配请求不能为空")

	ErrComponentMatchSubjectRequired = errors.New("学科不能为空")
)

// resolveComponentMatchDomain 解析组件匹配最终使用的具体教学域。
func resolveComponentMatchDomain(
	actor *AssistantActorContext,
	requestedDomain string,
) (string, error) {
	readDomain, err :=
		ResolveComponentReadDomain(actor)
	if err != nil {
		return "", err
	}

	if models.IsTeachingEducationDomain(
		readDomain,
	) {
		return readDomain, nil
	}

	requestedDomain = strings.ToLower(
		strings.TrimSpace(requestedDomain),
	)

	if requestedDomain == "" {
		return "",
			ErrComponentEducationDomainRequired
	}

	if !models.IsTeachingEducationDomain(
		requestedDomain,
	) {
		return "",
			ErrComponentEducationDomainInvalid
	}

	return requestedDomain, nil
}

// MatchComponentsForActor 按可信Actor或mixed显式目标域匹配组件。
func (s *ComponentService) MatchComponentsForActor(
	ctx context.Context,
	actor *AssistantActorContext,
	req *models.MatchComponentsRequest,
) (*models.MatchComponentsResponse, error) {
	if req == nil {
		return nil,
			ErrComponentMatchRequestRequired
	}

	if strings.TrimSpace(
		req.Subject,
	) == "" {
		return nil,
			ErrComponentMatchSubjectRequired
	}

	currentDomain, err :=
		resolveComponentMatchDomain(
			actor,
			req.EducationDomain,
		)
	if err != nil {
		return nil, err
	}

	request := *req
	request.EducationDomain = currentDomain
	request.Subject = strings.TrimSpace(
		request.Subject,
	)
	request.GradeRange = strings.TrimSpace(
		request.GradeRange,
	)

	groups, err :=
		repository.MatchComponentsForEducationDomain(
			ctx,
			&request,
			currentDomain,
		)
	if err != nil {
		return nil, err
	}

	if groups == nil {
		groups = []*models.MatchedComponentGroup{}
	}

	return &models.MatchComponentsResponse{
		Groups: groups,
	}, nil
}

// SmartRecommendComponentsForActor 按可信Actor教育域进行画像加权推荐。
func (s *ComponentService) SmartRecommendComponentsForActor(
	ctx context.Context,
	actor *AssistantActorContext,
	requestedDomain string,
	subject string,
	gradeRange string,
	profile *models.TeachingProfile,
) ([]*models.MatchedComponentGroup, error) {
	subject = strings.TrimSpace(subject)

	if subject == "" {
		return nil,
			ErrComponentMatchSubjectRequired
	}

	currentDomain, err :=
		resolveComponentMatchDomain(
			actor,
			requestedDomain,
		)
	if err != nil {
		return nil, err
	}

	request := &models.MatchComponentsRequest{
		EducationDomain: currentDomain,
		Subject:         subject,
		GradeRange: utils.NormalizeGradeToNumber(
			strings.TrimSpace(gradeRange),
		),
		Limit: 5,
	}

	groups, err :=
		repository.SmartMatchComponentsForEducationDomain(
			ctx,
			request,
			currentDomain,
			buildProfileTags(profile),
		)
	if err != nil {
		return nil, err
	}

	if groups == nil {
		groups = []*models.MatchedComponentGroup{}
	}

	return groups, nil
}
