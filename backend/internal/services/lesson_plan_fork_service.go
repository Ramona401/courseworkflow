package services

// lesson_plan_fork_service.go — 教案Fork教育域硬闸
//
// Fork规则：
//   1. 来源必须是共享发布或评审通过状态；
//   2. 来源education_domain必须是具体教学域；
//   3. 实时读取调用者users.role；
//   4. 调用统一创建教育域解析器；
//   5. 调用者域必须与来源资源域完全一致；
//   6. Repository显式写入来源域；
//   7. 副本INSERT和来源fork_count递增位于同一事务。
//
// 本文件不接受前端传入教育域，不使用JWT中的历史角色，
// 也不依赖数据库触发器为副本推导教育域。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// lessonPlanForkDeps 是Fork教育域硬闸的最小依赖集合。
//
// 正式环境使用真实Repository函数；测试通过注入脱离数据库，
// 不修改包级全局函数，避免并发测试污染。
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
		educationDomain string,
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
			ForkLessonPlanWithEducationDomain,
	}
}

// ForkLessonPlan Fork教案。
func (s *LessonPlanService) ForkLessonPlan(
	ctx context.Context,
	sourceID string,
	callerID string,
) (*models.LessonPlan, error) {
	return s.forkLessonPlanWithEducationDomainGate(
		ctx,
		sourceID,
		callerID,
		defaultLessonPlanForkDeps(),
	)
}

// forkLessonPlanWithEducationDomainGate
// 执行Fork的完整教育域硬闸。
func (s *LessonPlanService) forkLessonPlanWithEducationDomainGate(
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
		strings.TrimSpace(
			source.EducationDomain,
		),
	)
	if !models.IsTeachingEducationDomain(
		sourceDomain,
	) {
		lpLog.Error(
			"Fork来源教案教育域快照非法",
			"source_id", sourceID,
			"source_domain", sourceDomain,
		)
		return nil, fmt.Errorf(
			"%w: 来源教案教育域快照非法",
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

	callerDomain, err :=
		deps.resolveEducationDomain(
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
	if !models.IsTeachingEducationDomain(
		callerDomain,
	) {
		return nil, ErrLPCreationEducationDomainRequired
	}

	if callerDomain != sourceDomain {
		lpLog.Info(
			"跨教育域Fork被拦截",
			"source_id", sourceID,
			"source_domain", sourceDomain,
			"caller", callerID,
			"caller_domain", callerDomain,
		)
		return nil, ErrLPForkEducationDomainMismatch
	}

	newLessonPlan, err := deps.forkAtomic(
		ctx,
		sourceID,
		callerID,
		sourceDomain,
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
				"education_domain", sourceDomain,
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
		strings.TrimSpace(
			newLessonPlan.EducationDomain,
		),
	)
	if storedDomain != sourceDomain ||
		!models.IsTeachingEducationDomain(
			storedDomain,
		) {
		return nil, fmt.Errorf(
			"%w: source=%s database=%s",
			ErrLPCreationEducationDomainResolveFailed,
			sourceDomain,
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
		"education_domain", storedDomain,
	)

	return newLessonPlan, nil
}
