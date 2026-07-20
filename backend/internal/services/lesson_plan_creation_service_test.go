package services

// lesson_plan_creation_service_test.go
//
// 本文件覆盖上下文9和上下文10的普通创建链路，不连接数据库：
//   - 统一解析器只返回三个具体教学域；
//   - 无归属、非法域和跨域冲突全部fail-closed；
//   - Service把解析结果作为独立参数传入显式创建Repository；
//   - 数据库异常不会产生教案记录；
//   - 返回快照与解析结果不一致时明确失败。

import (
	"context"
	"errors"
	"testing"

	"tedna/internal/models"
	"tedna/internal/repository"
)

func TestResolveLessonPlanCreationEducationDomainFromCandidates(
	t *testing.T,
) {
	tests := []struct {
		name       string
		candidates []repository.LessonPlanCreationEducationDomainCandidate
		wantDomain string
		wantErr    error
	}{
		{
			name: "K12教学组织",
			candidates: []repository.LessonPlanCreationEducationDomainCandidate{
				{
					OrganizationID:  "school-k12",
					EducationDomain: models.EducationDomainK12,
				},
			},
			wantDomain: models.EducationDomainK12,
		},
		{
			name: "职业教育域自动规范化",
			candidates: []repository.LessonPlanCreationEducationDomainCandidate{
				{
					OrganizationID:  "school-vocational",
					EducationDomain: " VOCATIONAL ",
				},
			},
			wantDomain: models.EducationDomainVocational,
		},
		{
			name: "成人教育教学组织",
			candidates: []repository.LessonPlanCreationEducationDomainCandidate{
				{
					OrganizationID:  "school-adult",
					EducationDomain: models.EducationDomainAdult,
				},
			},
			wantDomain: models.EducationDomainAdult,
		},
		{
			name: "多个同域组织允许",
			candidates: []repository.LessonPlanCreationEducationDomainCandidate{
				{
					OrganizationID:  "school-a",
					EducationDomain: models.EducationDomainVocational,
				},
				{
					OrganizationID:  "school-b",
					EducationDomain: " vocational ",
				},
			},
			wantDomain: models.EducationDomainVocational,
		},
		{
			name:       "无有效教学组织拒绝",
			candidates: nil,
			wantErr: repository.
				ErrLessonPlanCreationEducationDomainUnavailable,
		},
		{
			name: "空教育域拒绝",
			candidates: []repository.LessonPlanCreationEducationDomainCandidate{
				{
					OrganizationID:  "school-empty",
					EducationDomain: "",
				},
			},
			wantErr: repository.
				ErrLessonPlanCreationEducationDomainUnavailable,
		},
		{
			name: "mixed管理域拒绝",
			candidates: []repository.LessonPlanCreationEducationDomainCandidate{
				{
					OrganizationID:  "school-mixed",
					EducationDomain: models.EducationDomainMixed,
				},
			},
			wantErr: repository.
				ErrLessonPlanCreationEducationDomainUnavailable,
		},
		{
			name: "common资源域拒绝",
			candidates: []repository.LessonPlanCreationEducationDomainCandidate{
				{
					OrganizationID:  "school-common",
					EducationDomain: models.EducationDomainCommon,
				},
			},
			wantErr: repository.
				ErrLessonPlanCreationEducationDomainUnavailable,
		},
		{
			name: "非法字符串拒绝",
			candidates: []repository.LessonPlanCreationEducationDomainCandidate{
				{
					OrganizationID:  "school-invalid",
					EducationDomain: "general",
				},
			},
			wantErr: repository.
				ErrLessonPlanCreationEducationDomainUnavailable,
		},
		{
			name: "跨具体教育域冲突",
			candidates: []repository.LessonPlanCreationEducationDomainCandidate{
				{
					OrganizationID:  "school-k12",
					EducationDomain: models.EducationDomainK12,
				},
				{
					OrganizationID:  "school-adult",
					EducationDomain: models.EducationDomainAdult,
				},
			},
			wantErr: repository.
				ErrLessonPlanCreationEducationDomainConflict,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			domain, err :=
				repository.
					ResolveLessonPlanCreationEducationDomainFromCandidates(
						testCase.candidates,
					)

			if testCase.wantErr != nil {
				if !errors.Is(err, testCase.wantErr) {
					t.Fatalf(
						"期望错误%v，实际错误%v",
						testCase.wantErr,
						err,
					)
				}
				if domain != "" {
					t.Fatalf(
						"失败时不应返回教育域，实际=%q",
						domain,
					)
				}
				return
			}

			if err != nil {
				t.Fatalf("不期望错误，实际=%v", err)
			}
			if domain != testCase.wantDomain {
				t.Fatalf(
					"教育域=%q，期望=%q",
					domain,
					testCase.wantDomain,
				)
			}
		})
	}
}

func TestCreateLessonPlanEducationDomainGate(
	t *testing.T,
) {
	originalFindUser := lessonPlanCreationFindUser
	originalResolveDomain := lessonPlanCreationResolveDomain
	originalInsert := lessonPlanCreationInsert

	defer func() {
		lessonPlanCreationFindUser = originalFindUser
		lessonPlanCreationResolveDomain = originalResolveDomain
		lessonPlanCreationInsert = originalInsert
	}()

	validRequest := func() *models.CreateLessonPlanRequest {
		return &models.CreateLessonPlanRequest{
			Title:           "测试教案",
			Subject:         "测试课程",
			Grade:           "测试层级",
			Topic:           "测试主题",
			DurationMinutes: 45,
		}
	}

	t.Run("三个具体教学域均显式传入创建Repository", func(t *testing.T) {
		domains := []string{
			models.EducationDomainK12,
			models.EducationDomainVocational,
			models.EducationDomainAdult,
		}

		for _, domain := range domains {
			domain := domain

			lessonPlanCreationFindUser = func(
				ctx context.Context,
				id string,
			) (*models.User, error) {
				return &models.User{
					ID:   id,
					Role: models.RoleOperator,
				}, nil
			}

			lessonPlanCreationResolveDomain = func(
				ctx context.Context,
				userID string,
				role string,
			) (string, error) {
				return domain, nil
			}

			insertCalls := 0
			lessonPlanCreationInsert = func(
				ctx context.Context,
				lp *models.LessonPlan,
				explicitDomain string,
			) error {
				insertCalls++

				if explicitDomain != domain {
					t.Fatalf(
						"Repository显式参数=%q，期望=%q",
						explicitDomain,
						domain,
					)
				}
				if lp.EducationDomain != domain {
					t.Fatalf(
						"写入前实体教育域=%q，期望=%q",
						lp.EducationDomain,
						domain,
					)
				}

				lp.ID = "created-" + domain
				lp.EducationDomain = explicitDomain
				return nil
			}

			service := &LessonPlanService{}
			result, err :=
				service.CreateLessonPlan(
					context.Background(),
					validRequest(),
					"user-1",
				)
			if err != nil {
				t.Fatalf(
					"教育域=%s不应返回错误，实际=%v",
					domain,
					err,
				)
			}
			if insertCalls != 1 {
				t.Fatalf(
					"教育域=%s创建Repository调用次数=%d，期望1",
					domain,
					insertCalls,
				)
			}
			if result == nil ||
				result.EducationDomain != domain {
				t.Fatalf(
					"教育域=%s返回结果异常：%#v",
					domain,
					result,
				)
			}
		}
	})

	t.Run("跨域冲突不会产生教案记录", func(t *testing.T) {
		lessonPlanCreationFindUser = func(
			ctx context.Context,
			id string,
		) (*models.User, error) {
			return &models.User{
				ID:   id,
				Role: models.RoleOperator,
			}, nil
		}

		lessonPlanCreationResolveDomain = func(
			ctx context.Context,
			userID string,
			role string,
		) (string, error) {
			return "",
				repository.
					ErrLessonPlanCreationEducationDomainConflict
		}

		insertCalls := 0
		lessonPlanCreationInsert = func(
			ctx context.Context,
			lp *models.LessonPlan,
			explicitDomain string,
		) error {
			insertCalls++
			return nil
		}

		service := &LessonPlanService{}
		result, err :=
			service.CreateLessonPlan(
				context.Background(),
				validRequest(),
				"user-1",
			)

		if result != nil {
			t.Fatalf("冲突时不应返回教案，实际=%#v", result)
		}
		if !errors.Is(
			err,
			ErrLPCreationEducationDomainConflict,
		) {
			t.Fatalf(
				"期望错误%v，实际=%v",
				ErrLPCreationEducationDomainConflict,
				err,
			)
		}
		if insertCalls != 0 {
			t.Fatalf(
				"冲突时不应调用创建Repository，实际=%d",
				insertCalls,
			)
		}
	})

	t.Run("无有效教学组织不会产生教案记录", func(t *testing.T) {
		lessonPlanCreationFindUser = func(
			ctx context.Context,
			id string,
		) (*models.User, error) {
			return &models.User{
				ID:   id,
				Role: models.RoleViewer,
			}, nil
		}

		lessonPlanCreationResolveDomain = func(
			ctx context.Context,
			userID string,
			role string,
		) (string, error) {
			return "",
				repository.
					ErrLessonPlanCreationEducationDomainUnavailable
		}

		insertCalls := 0
		lessonPlanCreationInsert = func(
			ctx context.Context,
			lp *models.LessonPlan,
			explicitDomain string,
		) error {
			insertCalls++
			return nil
		}

		service := &LessonPlanService{}
		_, err :=
			service.CreateLessonPlan(
				context.Background(),
				validRequest(),
				"user-1",
			)

		if !errors.Is(
			err,
			ErrLPCreationEducationDomainRequired,
		) {
			t.Fatalf(
				"期望错误%v，实际=%v",
				ErrLPCreationEducationDomainRequired,
				err,
			)
		}
		if insertCalls != 0 {
			t.Fatalf(
				"无组织时不应调用创建Repository，实际=%d",
				insertCalls,
			)
		}
	})

	t.Run("数据库解析异常明确失败且不产生记录", func(t *testing.T) {
		lessonPlanCreationFindUser = func(
			ctx context.Context,
			id string,
		) (*models.User, error) {
			return &models.User{
				ID:   id,
				Role: models.RoleOperator,
			}, nil
		}

		lessonPlanCreationResolveDomain = func(
			ctx context.Context,
			userID string,
			role string,
		) (string, error) {
			return "", errors.New("模拟数据库连接失败")
		}

		insertCalls := 0
		lessonPlanCreationInsert = func(
			ctx context.Context,
			lp *models.LessonPlan,
			explicitDomain string,
		) error {
			insertCalls++
			return nil
		}

		service := &LessonPlanService{}
		_, err :=
			service.CreateLessonPlan(
				context.Background(),
				validRequest(),
				"user-1",
			)

		if !errors.Is(
			err,
			ErrLPCreationEducationDomainResolveFailed,
		) {
			t.Fatalf(
				"期望错误%v，实际=%v",
				ErrLPCreationEducationDomainResolveFailed,
				err,
			)
		}
		if insertCalls != 0 {
			t.Fatalf(
				"数据库异常时不应调用创建Repository，实际=%d",
				insertCalls,
			)
		}
	})

	t.Run("数据库返回不同快照时明确失败", func(t *testing.T) {
		lessonPlanCreationFindUser = func(
			ctx context.Context,
			id string,
		) (*models.User, error) {
			return &models.User{
				ID:   id,
				Role: models.RoleOperator,
			}, nil
		}

		lessonPlanCreationResolveDomain = func(
			ctx context.Context,
			userID string,
			role string,
		) (string, error) {
			return models.EducationDomainVocational, nil
		}

		lessonPlanCreationInsert = func(
			ctx context.Context,
			lp *models.LessonPlan,
			explicitDomain string,
		) error {
			lp.ID = "unexpected-plan"
			lp.EducationDomain = models.EducationDomainK12
			return nil
		}

		service := &LessonPlanService{}
		result, err :=
			service.CreateLessonPlan(
				context.Background(),
				validRequest(),
				"user-1",
			)

		if result != nil {
			t.Fatalf(
				"快照不一致时不应返回教案，实际=%#v",
				result,
			)
		}
		if !errors.Is(
			err,
			ErrLPCreationEducationDomainResolveFailed,
		) {
			t.Fatalf(
				"期望错误%v，实际=%v",
				ErrLPCreationEducationDomainResolveFailed,
				err,
			)
		}
	})
}
