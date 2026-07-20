package services

// lesson_plan_gen_start_test.go
//
// 本测试只验证上下文11新增的“会话创建教育域硬闸”纯编排：
//   - k12、vocational、adult三个具体域均显式传入Repository；
//   - mixed、空域、冲突和解析异常均在Create函数调用前失败；
//   - Service会再次核对数据库返回的教育域快照。
//
// 课本ID不属于本测试关注范围。
// 测试请求保持课本数组为空，避免教育域编排测试误入真实课本Repository。
// 课本能力应由lesson_plan_textbook_guard的独立测试覆盖。
//
// 测试通过依赖注入脱离数据库，不修改生产数据，也不依赖生产账号。

import (
	"context"
	"errors"
	"testing"

	"tedna/internal/models"
	"tedna/internal/repository"
)

func TestPrepareConversationLessonPlanWritesExplicitTeachingDomain(
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
			createCalled := false

			deps := lessonPlanConversationCreationDeps{
				findUser: func(
					ctx context.Context,
					userID string,
				) (*models.User, error) {
					if userID != "user-1" {
						t.Fatalf(
							"读取了错误用户: %s",
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
					if userID != "user-1" {
						t.Fatalf(
							"解析了错误用户: %s",
							userID,
						)
					}
					if role != models.RoleOperator {
						t.Fatalf(
							"未使用数据库实时角色: %s",
							role,
						)
					}

					return domain, nil
				},
				createWithEducationDomain: func(
					ctx context.Context,
					lp *models.LessonPlan,
					educationDomain string,
				) error {
					createCalled = true

					if educationDomain != domain {
						t.Fatalf(
							"显式写域错误: got=%s want=%s",
							educationDomain,
							domain,
						)
					}
					if lp.AuthorID != "user-1" {
						t.Fatalf(
							"作者错误: %s",
							lp.AuthorID,
						)
					}

					lp.ID = "plan-1"
					lp.EducationDomain = domain

					return nil
				},
			}

			req := &models.StartConversationRequest{
				Subject:  "数学",
				Grade:    "七年级",
				Topic:    "一次函数",
				RecipeID: "recipe-1",
				GroupID:  "group-1",
			}

			lp, err := prepareConversationLessonPlan(
				context.Background(),
				req,
				"user-1",
				45,
				deps,
			)
			if err != nil {
				t.Fatalf(
					"创建失败: %v",
					err,
				)
			}
			if !createCalled {
				t.Fatal(
					"未调用显式写域Repository",
				)
			}
			if lp == nil {
				t.Fatal(
					"返回教案为空",
				)
			}
			if lp.EducationDomain != domain {
				t.Fatalf(
					"数据库快照错误: got=%s want=%s",
					lp.EducationDomain,
					domain,
				)
			}
			if lp.RecipeID == nil ||
				*lp.RecipeID != "recipe-1" {
				t.Fatal(
					"专家模式配方ID未正确保留",
				)
			}
			if lp.GroupID == nil ||
				*lp.GroupID != "group-1" {
				t.Fatal(
					"教研组ID未正确保留",
				)
			}
		})
	}
}

func TestPrepareConversationLessonPlanRejectsDomainBeforeInsert(
	t *testing.T,
) {
	tests := []struct {
		name        string
		resolved    string
		resolveErr  error
		expectedErr error
	}{
		{
			name: "无有效教学组织",
			resolveErr: repository.
				ErrLessonPlanCreationEducationDomainUnavailable,
			expectedErr:
				ErrLPCreationEducationDomainRequired,
		},
		{
			name: "跨具体教育域冲突",
			resolveErr: repository.
				ErrLessonPlanCreationEducationDomainConflict,
			expectedErr:
				ErrLPCreationEducationDomainConflict,
		},
		{
			name: "解析器数据库异常",
			resolveErr: errors.New(
				"database unavailable",
			),
			expectedErr:
				ErrLPCreationEducationDomainResolveFailed,
		},
		{
			name:     "mixed不是教学创建域",
			resolved: models.EducationDomainMixed,
			expectedErr:
				ErrLPCreationEducationDomainResolveFailed,
		},
		{
			name:     "空教育域",
			resolved: "",
			expectedErr:
				ErrLPCreationEducationDomainResolveFailed,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			createCalled := false

			deps := lessonPlanConversationCreationDeps{
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
					return test.resolved,
						test.resolveErr
				},
				createWithEducationDomain: func(
					ctx context.Context,
					lp *models.LessonPlan,
					educationDomain string,
				) error {
					createCalled = true
					return nil
				},
			}

			lp, err := prepareConversationLessonPlan(
				context.Background(),
				&models.StartConversationRequest{
					Subject: "数学",
					Grade:   "七年级",
					Topic:   "一次函数",
				},
				"user-1",
				45,
				deps,
			)
			if err == nil {
				t.Fatal(
					"预期失败，实际成功",
				)
			}
			if !errors.Is(
				err,
				test.expectedErr,
			) {
				t.Fatalf(
					"错误类型不符: got=%v want=%v",
					err,
					test.expectedErr,
				)
			}
			if createCalled {
				t.Fatal(
					"教育域失败后仍调用了INSERT",
				)
			}
			if lp != nil {
				t.Fatal(
					"INSERT前失败不应返回教案对象",
				)
			}
		})
	}
}

func TestPrepareConversationLessonPlanDetectsStoredDomainMismatch(
	t *testing.T,
) {
	deps := lessonPlanConversationCreationDeps{
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
			return models.EducationDomainK12,
				nil
		},
		createWithEducationDomain: func(
			ctx context.Context,
			lp *models.LessonPlan,
			educationDomain string,
		) error {
			lp.ID = "plan-mismatch"
			lp.EducationDomain =
				models.EducationDomainAdult
			return nil
		},
	}

	lp, err := prepareConversationLessonPlan(
		context.Background(),
		&models.StartConversationRequest{
			Subject: "数学",
			Grade:   "七年级",
			Topic:   "一次函数",
		},
		"user-1",
		45,
		deps,
	)
	if err == nil {
		t.Fatal(
			"快照不一致时预期失败",
		)
	}
	if !errors.Is(
		err,
		ErrLPCreationEducationDomainResolveFailed,
	) {
		t.Fatalf(
			"错误类型不符: %v",
			err,
		)
	}
	if lp == nil ||
		lp.ID != "plan-mismatch" {
		t.Fatal(
			"应返回已创建教案供上层执行补偿清理",
		)
	}
}
