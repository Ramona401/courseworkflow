package services

// lesson_plan_common_shared_test.go — common共享教案回归测试
//
// 覆盖上下文17两个关键规则：
//   - 当前具体教学域可以访问组织范围内的common共享教案；
//   - Fork common来源时，来源快照保持common，副本落入调用者具体教学域。

import (
	"context"
	"testing"

	"tedna/internal/models"
)

func TestSharedLessonPlanAccessAllowsCommonForConcreteDomain(
	t *testing.T,
) {
	access := &lessonPlanSharedAccessContext{
		CurrentEducationDomain: models.EducationDomainK12,
		visibleAuthorSet: map[string]struct{}{
			"author-1": {},
		},
	}

	commonPlan := &models.LessonPlan{
		AuthorID:        "author-1",
		Status:          models.LPStatusApproved,
		Visibility:      models.LPVisibilityGroup,
		EducationDomain: models.EducationDomainCommon,
	}
	if !access.canAccessSharedLessonPlan(commonPlan) {
		t.Fatal(
			"k12具体域未能访问common共享教案",
		)
	}

	crossDomainPlan := &models.LessonPlan{
		AuthorID:        "author-1",
		Status:          models.LPStatusApproved,
		Visibility:      models.LPVisibilityGroup,
		EducationDomain: models.EducationDomainAdult,
	}
	if access.canAccessSharedLessonPlan(crossDomainPlan) {
		t.Fatal(
			"k12具体域错误访问adult共享教案",
		)
	}
}

func TestForkLessonPlanCommonSourceUsesCallerDomain(
	t *testing.T,
) {
	forkCalled := false

	deps := lessonPlanForkDeps{
		getSource: func(
			ctx context.Context,
			sourceID string,
		) (*models.LessonPlan, error) {
			return &models.LessonPlan{
				ID:              sourceID,
				Status:          models.LPStatusApproved,
				EducationDomain: models.EducationDomainCommon,
			}, nil
		},

		findUser: func(
			ctx context.Context,
			userID string,
		) (*models.User, error) {
			return &models.User{
				Role: models.RoleOperator,
			}, nil
		},

		resolveEducationDomain: func(
			ctx context.Context,
			userID string,
			role string,
		) (string, error) {
			return models.EducationDomainVocational, nil
		},

		forkAtomic: func(
			ctx context.Context,
			sourceID string,
			newAuthorID string,
			sourceEducationDomain string,
			targetEducationDomain string,
		) (*models.LessonPlan, error) {
			forkCalled = true

			if sourceEducationDomain !=
				models.EducationDomainCommon {
				t.Fatalf(
					"common来源域传递错误: %s",
					sourceEducationDomain,
				)
			}
			if targetEducationDomain !=
				models.EducationDomainVocational {
				t.Fatalf(
					"common副本目标域错误: %s",
					targetEducationDomain,
				)
			}

			return &models.LessonPlan{
				ID:              "fork-common-1",
				AuthorID:        newAuthorID,
				EducationDomain: targetEducationDomain,
				ForkedFrom: stringPointerForForkTest(
					sourceID,
				),
			}, nil
		},
	}

	service := &LessonPlanService{}
	result, err :=
		service.forkLessonPlanWithEducationDomainGate(
			context.Background(),
			"source-common-1",
			"caller-1",
			deps,
		)
	if err != nil {
		t.Fatalf(
			"common来源Fork失败: %v",
			err,
		)
	}
	if !forkCalled {
		t.Fatal(
			"common来源未调用原子Fork Repository",
		)
	}
	if result == nil ||
		result.EducationDomain !=
			models.EducationDomainVocational {
		t.Fatalf(
			"common副本未落入调用者具体域: %+v",
			result,
		)
	}
}
