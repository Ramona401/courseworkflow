package services

// lesson_plan_creation_service.go — 普通教案创建教育域硬闸
//
// 本文件独立承载普通教案创建逻辑，避免继续扩大lesson_plan_service.go。
//
// 创建顺序固定为：
//   1. 校验并规范化表单；
//   2. 从users表实时读取作者角色；
//   3. 调用上下文9解析器取得唯一具体教学域；
//   4. 解析失败时在任何INSERT之前返回；
//   5. 调用普通创建专用Repository方法；
//   6. Repository显式写入education_domain；
//   7. 数据库最终快照必须与Service解析结果一致。
//
// 对话备课、导入和Fork仍继续使用既有repository.CreateLessonPlan，
// 分别留给上下文11、12、13处理，本文件不提前改变它们的语义。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

var (
	// ErrLPCreationEducationDomainRequired 表示无法得到唯一具体教学域。
	ErrLPCreationEducationDomainRequired = errors.New(
		"无确定教学教育域不能创建教案，请联系管理员检查学校归属或教育域配置",
	)

	// ErrLPCreationEducationDomainConflict 表示用户同时属于多个具体教育域。
	ErrLPCreationEducationDomainConflict = errors.New(
		"当前账号同时关联多个教学教育域，暂不能创建教案，请联系管理员处理归属冲突",
	)

	// ErrLPCreationEducationDomainResolveFailed 表示数据库、基础设施
	// 或最终数据库快照一致性校验失败。
	ErrLPCreationEducationDomainResolveFailed = errors.New(
		"教案创建教育域解析失败，请稍后重试",
	)
)

// 以下函数变量默认指向正式Repository。
// 测试可以临时替换，验证失败路径不会产生教案记录。
var (
	lessonPlanCreationFindUser = repository.FindUserByID

	lessonPlanCreationResolveDomain = repository.ResolveLessonPlanCreationEducationDomain

	lessonPlanCreationInsert = repository.CreateLessonPlanWithEducationDomain
)

// createLessonPlanWithEducationDomainGate
// 执行普通教案创建的完整教育域硬闸。
func (s *LessonPlanService) createLessonPlanWithEducationDomainGate(
	ctx context.Context,
	req *models.CreateLessonPlanRequest,
	authorID string,
) (*models.LessonPlan, error) {
	if req == nil {
		return nil, ErrLPTitleRequired
	}

	req.Title = strings.TrimSpace(req.Title)
	req.Subject = strings.TrimSpace(req.Subject)
	req.Grade = strings.TrimSpace(req.Grade)
	req.Topic = strings.TrimSpace(req.Topic)

	if req.Title == "" {
		return nil, ErrLPTitleRequired
	}
	if req.Subject == "" {
		return nil, ErrLPSubjectRequired
	}
	if req.Grade == "" {
		return nil, ErrLPGradeRequired
	}
	if req.Topic == "" {
		return nil, ErrLPTopicRequired
	}

	authorID = strings.TrimSpace(authorID)
	if authorID == "" {
		return nil, ErrLPCreationEducationDomainRequired
	}

	// 角色必须从数据库实时读取，不能只信任JWT中的历史角色，
	// 也不能让前端通过请求体自行指定教育域。
	author, err := lessonPlanCreationFindUser(
		ctx,
		authorID,
	)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrLPCreationEducationDomainRequired
		}

		lpLog.Error(
			"读取教案创建者失败",
			"author", authorID,
			"error", err,
		)
		return nil, fmt.Errorf(
			"%w: %v",
			ErrLPCreationEducationDomainResolveFailed,
			err,
		)
	}

	creationDomain, err :=
		lessonPlanCreationResolveDomain(
			ctx,
			authorID,
			author.Role,
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
				"解析教案创建教育域失败",
				"author", authorID,
				"role", author.Role,
				"error", err,
			)
			return nil, fmt.Errorf(
				"%w: %v",
				ErrLPCreationEducationDomainResolveFailed,
				err,
			)
		}
	}

	creationDomain = strings.ToLower(
		strings.TrimSpace(creationDomain),
	)
	if !models.IsTeachingEducationDomain(
		creationDomain,
	) {
		return nil, ErrLPCreationEducationDomainRequired
	}

	duration := req.DurationMinutes
	if duration <= 0 {
		duration = 45
	}

	lp := &models.LessonPlan{
		Title:           req.Title,
		Subject:         req.Subject,
		Grade:           req.Grade,
		Topic:           req.Topic,
		DurationMinutes: duration,
		Status:          models.LPStatusDraft,
		Visibility:      models.LPVisibilityPersonal,
		AuthorID:        authorID,
		EducationDomain: creationDomain,
	}

	// 普通创建专用Repository会在同一事务中：
	//   - 显式INSERT education_domain；
	//   - 读取数据库最终快照；
	//   - 快照不一致时回滚。
	if err := lessonPlanCreationInsert(
		ctx,
		lp,
		creationDomain,
	); err != nil {
		lpLog.Error(
			"创建教案失败",
			"title", req.Title,
			"author", authorID,
			"education_domain", creationDomain,
			"error", err,
		)

		if errors.Is(
			err,
			repository.ErrLessonPlanExplicitEducationDomainRequired,
		) ||
			errors.Is(
				err,
				repository.
					ErrLessonPlanExplicitEducationDomainSnapshotMismatch,
			) {
			return nil, fmt.Errorf(
				"%w: %v",
				ErrLPCreationEducationDomainResolveFailed,
				err,
			)
		}

		return nil, err
	}

	storedDomain := strings.ToLower(
		strings.TrimSpace(lp.EducationDomain),
	)
	if storedDomain != creationDomain ||
		!models.IsTeachingEducationDomain(storedDomain) {
		lpLog.Error(
			"普通教案创建后教育域快照不一致",
			"plan_id", lp.ID,
			"author", authorID,
			"resolved_education_domain", creationDomain,
			"stored_education_domain", storedDomain,
		)
		return nil, fmt.Errorf(
			"%w: service=%s database=%s",
			ErrLPCreationEducationDomainResolveFailed,
			creationDomain,
			storedDomain,
		)
	}

	lpLog.Info(
		"创建教案成功",
		"plan_id", lp.ID,
		"title", lp.Title,
		"author", authorID,
		"resolved_education_domain", creationDomain,
		"stored_education_domain", storedDomain,
	)

	return lp, nil
}
