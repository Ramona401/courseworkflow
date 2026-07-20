package services

// course_outline_domain_test.go
//
// 本文件验证上下文16课程大纲教育域纯编排规则，不连接数据库：
//   - admin保留K12管理兼容域，但保持mixed管理标记；
//   - admin出版社列表固定为空，不触发大纲Repository查询；
//   - vocational/adult不能提交具名出版社；
//   - 普通教学角色使用数据库实时角色和严格教育域解析器；
//   - 跨域冲突转换为稳定业务错误；
//   - 资源归属域通过独立依赖解析并严格校验。

import (
	"context"
	"errors"
	"testing"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// preserveCourseOutlineDomainDependencies
// 保存正式依赖并在测试结束后恢复，避免污染其它测试。
func preserveCourseOutlineDomainDependencies(
	t *testing.T,
) {
	originalFindUser :=
		courseOutlineFindUser
	originalResolveActorDomain :=
		courseOutlineResolveActorDomain
	originalResolveScopeDomain :=
		courseOutlineResolveScopeDomain

	t.Cleanup(func() {
		courseOutlineFindUser =
			originalFindUser
		courseOutlineResolveActorDomain =
			originalResolveActorDomain
		courseOutlineResolveScopeDomain =
			originalResolveScopeDomain
	})
}

func TestResolveCourseOutlineActorKeepsAdminK12ManagementCompatibility(
	t *testing.T,
) {
	preserveCourseOutlineDomainDependencies(t)

	resolveCalled := false

	courseOutlineFindUser = func(
		ctx context.Context,
		userID string,
	) (*models.User, error) {
		if userID != "admin-1" {
			t.Fatalf(
				"读取了错误用户: %s",
				userID,
			)
		}

		return &models.User{
			ID:   userID,
			Role: models.RoleAdmin,
		}, nil
	}

	courseOutlineResolveActorDomain = func(
		ctx context.Context,
		userID string,
		role string,
	) (string, error) {
		resolveCalled = true
		return "", errors.New(
			"admin不应进入普通教学域解析",
		)
	}

	actor, err := resolveCourseOutlineActor(
		context.Background(),
		"admin-1",
	)
	if err != nil {
		t.Fatalf(
			"管理员兼容域解析失败: %v",
			err,
		)
	}
	if actor == nil {
		t.Fatal(
			"管理员Actor为空",
		)
	}
	if resolveCalled {
		t.Fatal(
			"管理员错误进入普通教学域解析器",
		)
	}
	if actor.Role != models.RoleAdmin {
		t.Fatalf(
			"管理员实时角色错误: %s",
			actor.Role,
		)
	}
	if actor.EducationDomain !=
		models.EducationDomainK12 {
		t.Fatalf(
			"管理员管理兼容域错误: %s",
			actor.EducationDomain,
		)
	}
	if !actor.MixedManagement {
		t.Fatal(
			"管理员必须保留mixed管理标记",
		)
	}
}

func TestAdminPublisherListReturnsSafeEmpty(
	t *testing.T,
) {
	preserveCourseOutlineDomainDependencies(t)

	courseOutlineFindUser = func(
		ctx context.Context,
		userID string,
	) (*models.User, error) {
		return &models.User{
			ID:   userID,
			Role: models.RoleAdmin,
		}, nil
	}

	courseOutlineResolveActorDomain = func(
		ctx context.Context,
		userID string,
		role string,
	) (string, error) {
		t.Fatal(
			"管理员出版社列表不应进入普通教学域解析器",
		)
		return "", nil
	}

	service := NewCourseOutlineService()

	publishers, err :=
		service.ListAvailablePublishers(
			context.Background(),
			"admin-1",
			"数学",
			"七年级",
		)
	if err != nil {
		t.Fatalf(
			"管理员出版社安全空列表失败: %v",
			err,
		)
	}
	if publishers == nil {
		t.Fatal(
			"安全空列表不能返回nil",
		)
	}
	if len(publishers) != 0 {
		t.Fatalf(
			"管理员出版社列表应为空，实际=%v",
			publishers,
		)
	}
}

func TestResolveCourseOutlineActorUsesLiveRoleAndStrictDomain(
	t *testing.T,
) {
	preserveCourseOutlineDomainDependencies(t)

	courseOutlineFindUser = func(
		ctx context.Context,
		userID string,
	) (*models.User, error) {
		return &models.User{
			ID:   userID,
			Role: models.RoleOperator,
		}, nil
	}

	courseOutlineResolveActorDomain = func(
		ctx context.Context,
		userID string,
		role string,
	) (string, error) {
		if userID != "teacher-1" {
			t.Fatalf(
				"严格域解析用户错误: %s",
				userID,
			)
		}
		if role != models.RoleOperator {
			t.Fatalf(
				"未使用数据库实时角色: %s",
				role,
			)
		}

		return models.EducationDomainVocational,
			nil
	}

	actor, err := resolveCourseOutlineActor(
		context.Background(),
		"teacher-1",
	)
	if err != nil {
		t.Fatalf(
			"普通教学Actor解析失败: %v",
			err,
		)
	}
	if actor.EducationDomain !=
		models.EducationDomainVocational {
		t.Fatalf(
			"教育域错误: %s",
			actor.EducationDomain,
		)
	}
	if actor.MixedManagement {
		t.Fatal(
			"普通教学角色不能带mixed管理标记",
		)
	}
}

func TestResolveCourseOutlineActorMapsDomainConflict(
	t *testing.T,
) {
	preserveCourseOutlineDomainDependencies(t)

	courseOutlineFindUser = func(
		ctx context.Context,
		userID string,
	) (*models.User, error) {
		return &models.User{
			ID:   userID,
			Role: models.RoleOperator,
		}, nil
	}

	courseOutlineResolveActorDomain = func(
		ctx context.Context,
		userID string,
		role string,
	) (string, error) {
		return "",
			repository.
				ErrLessonPlanCreationEducationDomainConflict
	}

	actor, err := resolveCourseOutlineActor(
		context.Background(),
		"teacher-1",
	)
	if actor != nil {
		t.Fatal(
			"教育域冲突时Actor必须为空",
		)
	}
	if !errors.Is(
		err,
		ErrOutlineEducationDomainConflict,
	) {
		t.Fatalf(
			"冲突错误映射不正确: %v",
			err,
		)
	}
}

func TestNormalizeCourseOutlinePublisherForDomain(
	t *testing.T,
) {
	tests := []struct {
		name        string
		domain      string
		publisher   string
		expected    string
		expectedErr error
	}{
		{
			name:      "K12允许具名出版社并去空白",
			domain:    models.EducationDomainK12,
			publisher: " 人教版 ",
			expected:  "人教版",
		},
		{
			name:      "K12允许空出版社",
			domain:    models.EducationDomainK12,
			publisher: "",
			expected:  "",
		},
		{
			name:      "职教允许普通大纲空出版社",
			domain:    models.EducationDomainVocational,
			publisher: " ",
			expected:  "",
		},
		{
			name:        "职教拒绝具名出版社",
			domain:      models.EducationDomainVocational,
			publisher:   "人教版",
			expectedErr: ErrOutlinePublisherNotAllowed,
		},
		{
			name:        "成教拒绝具名出版社",
			domain:      models.EducationDomainAdult,
			publisher:   "统编版",
			expectedErr: ErrOutlinePublisherNotAllowed,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			result, err :=
				normalizeCourseOutlinePublisherForDomain(
					test.domain,
					test.publisher,
				)

			if test.expectedErr != nil {
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
				return
			}

			if err != nil {
				t.Fatalf(
					"规范化失败: %v",
					err,
				)
			}
			if result != test.expected {
				t.Fatalf(
					"规范化结果错误: got=%q want=%q",
					result,
					test.expected,
				)
			}
		})
	}
}

func TestResolveCourseOutlineResourceDomainUsesTrustedResolver(
	t *testing.T,
) {
	preserveCourseOutlineDomainDependencies(t)

	courseOutlineResolveScopeDomain = func(
		ctx context.Context,
		scope string,
		targetID string,
	) (string, error) {
		if scope != models.CourseOutlineScopeGroup {
			t.Fatalf(
				"资源scope错误: %s",
				scope,
			)
		}
		if targetID != "group-1" {
			t.Fatalf(
				"资源归属ID错误: %s",
				targetID,
			)
		}

		return " vocational ",
			nil
	}

	domain, err :=
		resolveCourseOutlineResourceDomain(
			context.Background(),
			models.CourseOutlineScopeGroup,
			"group-1",
		)
	if err != nil {
		t.Fatalf(
			"资源域解析失败: %v",
			err,
		)
	}
	if domain !=
		models.EducationDomainVocational {
		t.Fatalf(
			"资源域规范化错误: %s",
			domain,
		)
	}
}
