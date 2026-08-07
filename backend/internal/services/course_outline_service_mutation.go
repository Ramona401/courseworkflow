package services

// course_outline_service_mutation.go — 课程大纲创建、更新和删除
//
// 所有写操作均实时解析操作者教育域，并校验资源正式归属与写权限。
// K12显式保存standard或five_four；非K12固定保存standard技术值。

import (
	"context"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// CreateOutline 创建课程大纲。
func (s *CourseOutlineService) CreateOutline(
	ctx context.Context,
	userID string,
	req *models.CreateCourseOutlineRequest,
) (
	*models.CourseOutline,
	string,
	error,
) {
	if req == nil {
		return nil, "",
			ErrOutlineFieldRequired
	}

	actor, err := resolveCourseOutlineActor(
		ctx,
		userID,
	)
	if err != nil {
		return nil, "", err
	}

	req.Scope = strings.TrimSpace(req.Scope)
	req.ScopeTargetID =
		strings.TrimSpace(req.ScopeTargetID)

	if !models.IsValidCourseOutlineScope(
		req.Scope,
	) {
		return nil, "",
			ErrOutlineScopeInvalid
	}
	if req.Scope ==
		models.CourseOutlineScopeSystem {
		req.ScopeTargetID =
			models.CourseOutlineSystemTargetID
	}

	req.Subject = strings.TrimSpace(req.Subject)
	req.Grade = strings.TrimSpace(req.Grade)
	req.Volume = strings.TrimSpace(req.Volume)
	req.Title = strings.TrimSpace(req.Title)

	if req.Subject == "" ||
		req.Grade == "" ||
		req.Volume == "" ||
		req.Title == "" ||
		strings.TrimSpace(req.Content) == "" ||
		req.ScopeTargetID == "" {
		return nil, "",
			ErrOutlineFieldRequired
	}

	resourceDomain, err :=
		resolveCourseOutlineResourceDomain(
			ctx,
			req.Scope,
			req.ScopeTargetID,
		)
	if err != nil {
		return nil, "", err
	}
	if resourceDomain !=
		actor.EducationDomain {
		return nil, "",
			ErrOutlineEducationDomainMismatch
	}
	if !s.canManageScope(
		ctx,
		actor,
		req.Scope,
		req.ScopeTargetID,
	) {
		return nil, "",
			ErrOutlineNoPermission
	}

	publisher, err :=
		normalizeCourseOutlinePublisherForDomain(
			actor.EducationDomain,
			req.Publisher,
		)
	if err != nil {
		return nil, "", err
	}

	schoolSystem, err :=
		normalizeCourseOutlineSchoolSystemForDomain(
			actor.EducationDomain,
			req.SchoolSystem,
			"",
		)
	if err != nil {
		return nil, "", err
	}

	outline := &models.CourseOutline{
		Scope:         req.Scope,
		ScopeTargetID: req.ScopeTargetID,
		Subject:       req.Subject,
		Grade:         req.Grade,
		Volume:        req.Volume,
		Publisher:     publisher,
		SchoolSystem:  schoolSystem,
		Title:         req.Title,
		Content:       req.Content,
		SourceType: models.
			CourseOutlineSourcePaste,
		CreatedBy: actor.UserID,
	}

	if err := repository.
		CreateCourseOutlineWithSchoolSystem(
			ctx,
			outline,
		); err != nil {
		return nil, "", err
	}

	return outline,
		actor.EducationDomain,
		nil
}

// UpdateOutline 更新课程大纲。
func (s *CourseOutlineService) UpdateOutline(
	ctx context.Context,
	userID string,
	id string,
	req *models.UpdateCourseOutlineRequest,
) error {
	if req == nil {
		return ErrOutlineFieldRequired
	}

	actor, err := resolveCourseOutlineActor(
		ctx,
		userID,
	)
	if err != nil {
		return err
	}

	existing, err :=
		repository.GetCourseOutlineByIDWithSchoolSystem(
			ctx,
			strings.TrimSpace(id),
		)
	if err != nil {
		return err
	}

	resourceDomain, err :=
		resolveCourseOutlineResourceDomain(
			ctx,
			existing.Scope,
			existing.ScopeTargetID,
		)
	if err != nil {
		return err
	}
	if resourceDomain !=
		actor.EducationDomain {
		return ErrOutlineEducationDomainMismatch
	}
	if !s.canManageScope(
		ctx,
		actor,
		existing.Scope,
		existing.ScopeTargetID,
	) {
		return ErrOutlineNoPermission
	}

	req.Subject = strings.TrimSpace(req.Subject)
	req.Grade = strings.TrimSpace(req.Grade)
	req.Volume = strings.TrimSpace(req.Volume)
	req.Title = strings.TrimSpace(req.Title)

	if req.Subject == "" ||
		req.Grade == "" ||
		req.Volume == "" ||
		req.Title == "" ||
		strings.TrimSpace(req.Content) == "" {
		return ErrOutlineFieldRequired
	}

	publisher, err :=
		normalizeCourseOutlinePublisherForDomain(
			actor.EducationDomain,
			req.Publisher,
		)
	if err != nil {
		return err
	}
	req.Publisher = publisher

	schoolSystem, err :=
		normalizeCourseOutlineSchoolSystemForDomain(
			actor.EducationDomain,
			req.SchoolSystem,
			existing.SchoolSystem,
		)
	if err != nil {
		return err
	}
	req.SchoolSystem = schoolSystem

	return repository.
		UpdateCourseOutlineWithSchoolSystem(
			ctx,
			existing.ID,
			req,
		)
}

// DeleteOutline 软删除课程大纲。
func (s *CourseOutlineService) DeleteOutline(
	ctx context.Context,
	userID string,
	id string,
) error {
	actor, err := resolveCourseOutlineActor(
		ctx,
		userID,
	)
	if err != nil {
		return err
	}

	existing, err :=
		repository.GetCourseOutlineByIDWithSchoolSystem(
			ctx,
			strings.TrimSpace(id),
		)
	if err != nil {
		return err
	}

	resourceDomain, err :=
		resolveCourseOutlineResourceDomain(
			ctx,
			existing.Scope,
			existing.ScopeTargetID,
		)
	if err != nil {
		return err
	}
	if resourceDomain !=
		actor.EducationDomain {
		return ErrOutlineEducationDomainMismatch
	}
	if !s.canManageScope(
		ctx,
		actor,
		existing.Scope,
		existing.ScopeTargetID,
	) {
		return ErrOutlineNoPermission
	}

	return repository.DeleteCourseOutline(
		ctx,
		existing.ID,
	)
}
