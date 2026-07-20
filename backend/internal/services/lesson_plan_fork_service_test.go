package services

// lesson_plan_fork_service_test.go
//
// 本测试验证上下文13新增的Fork教育域硬闸：
//   - k12、vocational、adult同域Fork均成功；
//   - 来源域作为独立参数传给原子Repository；
//   - 跨域在Repository调用前失败；
//   - mixed、common、空来源域全部fail-closed；
//   - 调用者无域、冲突和基础设施错误正确映射；
//   - Repository返回错误域快照时拒绝；
//   - 非共享或未通过来源不能Fork。
//
// 测试使用依赖注入脱离数据库，不修改全局函数变量。

import (
	"context"
	"errors"
	"testing"

	"tedna/internal/models"
	"tedna/internal/repository"
)

func TestForkLessonPlanWithEducationDomainGateTeachingDomains(
	t *testing.T,
) {
	domains := []string{
		models.EducationDomainK12,
		models.EducationDomainVocational,
		models.EducationDomainAdult,
	}

	for _, domain := range domains {
		domain := domain

		t.Run(domain, func(t *testing.T) {
			forkCalled := false

			deps := lessonPlanForkDeps{
				getSource: func(
					ctx context.Context,
					sourceID string,
				) (*models.LessonPlan, error) {
					if sourceID != "source-1" {
						t.Fatalf(
							"来源ID错误: %s",
							sourceID,
						)
					}

					return &models.LessonPlan{
						ID:              sourceID,
						Status:          models.LPStatusApproved,
						EducationDomain: domain,
					}, nil
				},

				findUser: func(
					ctx context.Context,
					userID string,
				) (*models.User, error) {
					if userID != "caller-1" {
						t.Fatalf(
							"调用者ID错误: %s",
							userID,
						)
					}

					return &models.User{
						Role: models.RoleOperator,
					}, nil
				},

				resolveEducationDomain: func(
					ctx context.Context,
					userID string,
					role string,
				) (string, error) {
					if role != models.RoleOperator {
						t.Fatalf(
							"未使用数据库实时角色: %s",
							role,
						)
					}
					return domain, nil
				},

				forkAtomic: func(
					ctx context.Context,
					sourceID string,
					newAuthorID string,
					educationDomain string,
				) (*models.LessonPlan, error) {
					forkCalled = true

					if educationDomain != domain {
						t.Fatalf(
							"传给Repository的域错误: got=%s want=%s",
							educationDomain,
							domain,
						)
					}
					if newAuthorID != "caller-1" {
						t.Fatalf(
							"副本作者错误: %s",
							newAuthorID,
						)
					}

					return &models.LessonPlan{
						ID:              "fork-1",
						AuthorID:        newAuthorID,
						EducationDomain: domain,
						ForkedFrom:      stringPointerForForkTest(sourceID),
					}, nil
				},
			}

			service := &LessonPlanService{}
			result, err :=
				service.forkLessonPlanWithEducationDomainGate(
					context.Background(),
					"source-1",
					"caller-1",
					deps,
				)
			if err != nil {
				t.Fatalf(
					"同域Fork失败: %v",
					err,
				)
			}
			if !forkCalled {
				t.Fatal(
					"未调用原子Fork Repository",
				)
			}
			if result == nil ||
				result.EducationDomain != domain {
				t.Fatalf(
					"副本教育域错误: %+v",
					result,
				)
			}
		})
	}
}

func TestForkLessonPlanRejectsCrossDomainBeforeInsert(
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
				Status:          models.LPStatusPublishedShared,
				EducationDomain: models.EducationDomainK12,
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
			return models.EducationDomainAdult, nil
		},

		forkAtomic: func(
			ctx context.Context,
			sourceID string,
			newAuthorID string,
			educationDomain string,
		) (*models.LessonPlan, error) {
			forkCalled = true
			return nil, nil
		},
	}

	service := &LessonPlanService{}
	_, err :=
		service.forkLessonPlanWithEducationDomainGate(
			context.Background(),
			"source-1",
			"caller-1",
			deps,
		)
	if err == nil ||
		!errors.Is(
			err,
			ErrLPForkEducationDomainMismatch,
		) {
		t.Fatalf(
			"跨域错误类型不符: %v",
			err,
		)
	}
	if forkCalled {
		t.Fatal(
			"跨域失败后仍调用了INSERT",
		)
	}
}

func TestForkLessonPlanRejectsInvalidSourceDomainBeforeInsert(
	t *testing.T,
) {
	invalidDomains := []string{
		"",
		models.EducationDomainMixed,
		models.EducationDomainCommon,
		"invalid-domain",
	}

	for _, domain := range invalidDomains {
		domain := domain

		t.Run(domain, func(t *testing.T) {
			findUserCalled := false
			forkCalled := false

			deps := lessonPlanForkDeps{
				getSource: func(
					ctx context.Context,
					sourceID string,
				) (*models.LessonPlan, error) {
					return &models.LessonPlan{
						ID:              sourceID,
						Status:          models.LPStatusApproved,
						EducationDomain: domain,
					}, nil
				},

				findUser: func(
					ctx context.Context,
					userID string,
				) (*models.User, error) {
					findUserCalled = true
					return nil, nil
				},

				forkAtomic: func(
					ctx context.Context,
					sourceID string,
					newAuthorID string,
					educationDomain string,
				) (*models.LessonPlan, error) {
					forkCalled = true
					return nil, nil
				},
			}

			service := &LessonPlanService{}
			_, err :=
				service.forkLessonPlanWithEducationDomainGate(
					context.Background(),
					"source-1",
					"caller-1",
					deps,
				)
			if err == nil ||
				!errors.Is(
					err,
					ErrLPCreationEducationDomainResolveFailed,
				) {
				t.Fatalf(
					"非法来源域错误类型不符: %v",
					err,
				)
			}
			if findUserCalled || forkCalled {
				t.Fatal(
					"非法来源域后仍继续执行创建链",
				)
			}
		})
	}
}

func TestForkLessonPlanMapsCallerDomainErrors(
	t *testing.T,
) {
	tests := []struct {
		name        string
		resolveErr  error
		expectedErr error
	}{
		{
			name: "调用者没有有效教学域",
			resolveErr: repository.
				ErrLessonPlanCreationEducationDomainUnavailable,
			expectedErr: ErrLPCreationEducationDomainRequired,
		},
		{
			name: "调用者教育域冲突",
			resolveErr: repository.
				ErrLessonPlanCreationEducationDomainConflict,
			expectedErr: ErrLPCreationEducationDomainConflict,
		},
		{
			name: "解析基础设施异常",
			resolveErr: errors.New(
				"database unavailable",
			),
			expectedErr: ErrLPCreationEducationDomainResolveFailed,
		},
	}

	for _, testCase := range tests {
		testCase := testCase

		t.Run(testCase.name, func(t *testing.T) {
			forkCalled := false

			deps := lessonPlanForkDeps{
				getSource: func(
					ctx context.Context,
					sourceID string,
				) (*models.LessonPlan, error) {
					return &models.LessonPlan{
						ID:              sourceID,
						Status:          models.LPStatusApproved,
						EducationDomain: models.EducationDomainK12,
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
					return "",
						testCase.resolveErr
				},

				forkAtomic: func(
					ctx context.Context,
					sourceID string,
					newAuthorID string,
					educationDomain string,
				) (*models.LessonPlan, error) {
					forkCalled = true
					return nil, nil
				},
			}

			service := &LessonPlanService{}
			_, err :=
				service.forkLessonPlanWithEducationDomainGate(
					context.Background(),
					"source-1",
					"caller-1",
					deps,
				)
			if err == nil ||
				!errors.Is(
					err,
					testCase.expectedErr,
				) {
				t.Fatalf(
					"错误映射不符: got=%v want=%v",
					err,
					testCase.expectedErr,
				)
			}
			if forkCalled {
				t.Fatal(
					"教育域解析失败后仍调用了INSERT",
				)
			}
		})
	}
}

func TestForkLessonPlanDetectsStoredDomainMismatch(
	t *testing.T,
) {
	deps := lessonPlanForkDeps{
		getSource: func(
			ctx context.Context,
			sourceID string,
		) (*models.LessonPlan, error) {
			return &models.LessonPlan{
				ID:              sourceID,
				Status:          models.LPStatusApproved,
				EducationDomain: models.EducationDomainK12,
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
			return models.EducationDomainK12, nil
		},

		forkAtomic: func(
			ctx context.Context,
			sourceID string,
			newAuthorID string,
			educationDomain string,
		) (*models.LessonPlan, error) {
			return &models.LessonPlan{
				ID:              "fork-1",
				AuthorID:        newAuthorID,
				EducationDomain: models.EducationDomainAdult,
				ForkedFrom:      stringPointerForForkTest(sourceID),
			}, nil
		},
	}

	service := &LessonPlanService{}
	_, err :=
		service.forkLessonPlanWithEducationDomainGate(
			context.Background(),
			"source-1",
			"caller-1",
			deps,
		)
	if err == nil ||
		!errors.Is(
			err,
			ErrLPCreationEducationDomainResolveFailed,
		) {
		t.Fatalf(
			"错误快照未被拒绝: %v",
			err,
		)
	}
}

func TestForkLessonPlanRequiresForkableSource(
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
				Status:          models.LPStatusDraft,
				EducationDomain: models.EducationDomainK12,
			}, nil
		},

		forkAtomic: func(
			ctx context.Context,
			sourceID string,
			newAuthorID string,
			educationDomain string,
		) (*models.LessonPlan, error) {
			forkCalled = true
			return nil, nil
		},
	}

	service := &LessonPlanService{}
	_, err :=
		service.forkLessonPlanWithEducationDomainGate(
			context.Background(),
			"source-1",
			"caller-1",
			deps,
		)
	if err == nil ||
		!errors.Is(
			err,
			ErrLPForkNotAllowed,
		) {
		t.Fatalf(
			"非公开来源错误类型不符: %v",
			err,
		)
	}
	if forkCalled {
		t.Fatal(
			"非公开来源仍调用了INSERT",
		)
	}
}

func stringPointerForForkTest(
	value string,
) *string {
	return &value
}
