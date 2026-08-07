package services

// lesson_plan_fork_service.go — 教案Fork教育域与共享可见性硬闸
//
// HTTP正式入口先执行共享市场可见性检查，使异域、私有、未共享和组织范围外
// 来源统一返回404。通过后继续复用上下文13的显式教育域与原子事务链：
//   - 实时解析调用者唯一具体教学域；
//   - 具体域来源只能Fork到同域，common来源落入调用者具体域；
//   - 副本INSERT与来源fork_count递增位于同一事务；
//   - 任一步失败时零新增、零计数变化。
//
// 内部forkLessonPlanWithEducationDomainGate保留细粒度错误，供单元测试锁定规则。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// lessonPlanForkDeps 是Fork硬闸的最小可注入依赖集合。
type lessonPlanForkDeps struct {
	getSource func(
		ctx context.Context,
		sourceID string,
	) (*models.LessonPlan, error)

	findUser func(
		ctx context.Context,
		userID string,
	) (*models.User, error)

	resolveEducationDomain func(
		ctx context.Context,
		userID string,
		role string,
	) (string, error)

	forkAtomic func(
		ctx context.Context,
		sourceID string,
		newAuthorID string,
		sourceEducationDomain string,
		targetEducationDomain string,
	) (*models.LessonPlan, error)
}

// defaultLessonPlanForkDeps 返回生产环境真实依赖。
func defaultLessonPlanForkDeps() lessonPlanForkDeps {
	return lessonPlanForkDeps{
		getSource: repository.GetLessonPlanByID,
		findUser:  repository.FindUserByID,
		resolveEducationDomain: repository.
			ResolveLessonPlanCreationEducationDomain,
		forkAtomic: repository.
			ForkLessonPlanWithEducationDomains,
	}
}

// ForkLessonPlan 是HTTP正式入口。
// 前置共享可见性失败统一返回ErrLPNotFound。
func (
	s *LessonPlanService,
) ForkLessonPlan(
	ctx context.Context,
	sourceID string,
	callerID string,
) (*models.LessonPlan, error) {
	if _, err := s.loadSharedLessonPlanForRead(
		ctx,
		sourceID,
		callerID,
		nil,
	); err != nil {
		return nil, err
	}

	forked, err := s.forkLessonPlanWithEducationDomainGate(
		ctx,
		sourceID,
		callerID,
		defaultLessonPlanForkDeps(),
	)
	if err != nil {
		// 前置检查与事务锁之间若发生撤回共享或教育域变化，
		// 仍按资源不可见处理，避免竞态错误差异泄露状态。
		if errors.Is(
			err,
			ErrLPForkNotAllowed,
		) ||
			errors.Is(
				err,
				ErrLPForkEducationDomainMismatch,
			) {
			return nil, ErrLPNotFound
		}
		return nil, err
	}
	return forked, nil
}

// forkLessonPlanWithEducationDomainGate 执行完整教育域硬闸。
func (
	s *LessonPlanService,
) forkLessonPlanWithEducationDomainGate(
	ctx context.Context,
	sourceID string,
	callerID string,
	deps lessonPlanForkDeps,
) (*models.LessonPlan, error) {
	sourceID = strings.TrimSpace(sourceID)
	callerID = strings.TrimSpace(callerID)

	if sourceID == "" {
		return nil, ErrLPNotFound
	}
	if callerID == "" {
		return nil, ErrLPCreationEducationDomainRequired
	}

	source, err := deps.getSource(
		ctx,
		sourceID,
	)
	if err != nil {
		if errors.Is(
			err,
			repository.ErrLessonPlanNotFound,
		) {
			return nil, ErrLPNotFound
		}
		return nil, err
	}
	if source == nil {
		return nil, ErrLPNotFound
	}

	if source.Status != models.LPStatusPublishedShared &&
		source.Status != models.LPStatusApproved {
		return nil, ErrLPForkNotAllowed
	}

	sourceDomain := strings.ToLower(
		strings.TrimSpace(source.EducationDomain),
	)
	if !models.IsResourceEducationDomain(sourceDomain) {
		lpLog.Error(
			"Fork来源教案资源域快照非法",
			"source_id", sourceID,
			"source_domain", sourceDomain,
		)
		return nil, fmt.Errorf(
			"%w: 来源教案资源域快照非法",
			ErrLPCreationEducationDomainResolveFailed,
		)
	}

	caller, err := deps.findUser(
		ctx,
		callerID,
	)
	if err != nil {
		if errors.Is(
			err,
			repository.ErrUserNotFound,
		) {
			return nil, ErrLPCreationEducationDomainRequired
		}
		lpLog.Error(
			"Fork读取调用者实时角色失败",
			"caller", callerID,
			"error", err,
		)
		return nil, fmt.Errorf(
			"%w: 读取调用者实时角色失败",
			ErrLPCreationEducationDomainResolveFailed,
		)
	}
	if caller == nil ||
		strings.TrimSpace(caller.Role) == "" {
		return nil, ErrLPCreationEducationDomainRequired
	}

	callerDomain, err := deps.resolveEducationDomain(
		ctx,
		callerID,
		caller.Role,
	)
	if err != nil {
		switch {
		case errors.Is(
			err,
			repository.ErrLessonPlanCreationEducationDomainConflict,
		):
			return nil, ErrLPCreationEducationDomainConflict

		case errors.Is(
			err,
			repository.ErrLessonPlanCreationEducationDomainUnavailable,
		),
			errors.Is(
				err,
				repository.ErrRegionAdminEducationDomainNotReady,
			):
			return nil, ErrLPCreationEducationDomainRequired

		default:
			lpLog.Error(
				"Fork解析调用者教育域失败",
				"caller", callerID,
				"role", caller.Role,
				"error", err,
			)
			return nil, fmt.Errorf(
				"%w: %v",
				ErrLPCreationEducationDomainResolveFailed,
				err,
			)
		}
	}

	callerDomain = strings.ToLower(
		strings.TrimSpace(callerDomain),
	)
	if !models.IsTeachingEducationDomain(callerDomain) {
		return nil, ErrLPCreationEducationDomainRequired
	}
	if sourceDomain != models.EducationDomainCommon &&
		callerDomain != sourceDomain {
		lpLog.Info(
			"跨教育域Fork被拦截",
			"source_id", sourceID,
			"source_domain", sourceDomain,
			"caller", callerID,
			"caller_domain", callerDomain,
		)
		return nil, ErrLPForkEducationDomainMismatch
	}

	// common只作为公共来源快照，副本必须写入调用者具体教学域。
	targetDomain := callerDomain

	newLessonPlan, err := deps.forkAtomic(
		ctx,
		sourceID,
		callerID,
		sourceDomain,
		targetDomain,
	)
	if err != nil {
		switch {
		case errors.Is(
			err,
			repository.ErrLessonPlanNotFound,
		):
			return nil, ErrLPNotFound

		case errors.Is(
			err,
			repository.ErrLessonPlanForkSourceNotForkable,
		):
			return nil, ErrLPForkNotAllowed

		case errors.Is(
			err,
			repository.ErrLessonPlanForkEducationDomainMismatch,
		):
			return nil, ErrLPForkEducationDomainMismatch

		case errors.Is(
			err,
			repository.ErrLessonPlanForkEducationDomainRequired,
		),
			errors.Is(
				err,
				repository.ErrLessonPlanForkEducationDomainSnapshotMismatch,
			):
			return nil, fmt.Errorf(
				"%w: %v",
				ErrLPCreationEducationDomainResolveFailed,
				err,
			)

		default:
			lpLog.Error(
				"Fork教案原子事务失败",
				"source_id", sourceID,
				"caller", callerID,
				"source_domain", sourceDomain,
				"target_domain", targetDomain,
				"error", err,
			)
			return nil, err
		}
	}

	if newLessonPlan == nil {
		return nil, fmt.Errorf(
			"%w: Fork结果为空",
			ErrLPCreationEducationDomainResolveFailed,
		)
	}

	storedDomain := strings.ToLower(
		strings.TrimSpace(newLessonPlan.EducationDomain),
	)
	if storedDomain != targetDomain ||
		!models.IsTeachingEducationDomain(storedDomain) {
		return nil, fmt.Errorf(
			"%w: source=%s target=%s database=%s",
			ErrLPCreationEducationDomainResolveFailed,
			sourceDomain,
			targetDomain,
			storedDomain,
		)
	}

	if newLessonPlan.ForkedFrom == nil ||
		*newLessonPlan.ForkedFrom != sourceID ||
		newLessonPlan.AuthorID != callerID {
		return nil, fmt.Errorf(
			"%w: Fork关系快照不一致",
			ErrLPCreationEducationDomainResolveFailed,
		)
	}

	lpLog.Info(
		"Fork教案成功",
		"source_id", sourceID,
		"new_id", newLessonPlan.ID,
		"author", callerID,
		"source_domain", sourceDomain,
		"education_domain", storedDomain,
	)

	return newLessonPlan, nil
}
