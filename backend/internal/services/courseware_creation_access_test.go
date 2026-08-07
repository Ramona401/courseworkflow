package services

// courseware_creation_access_test.go — 从教案创建课件教育域策略测试
//
// 本测试只覆盖不访问数据库的纯策略函数，重点防止以下回归：
//   - 普通教学作者不能被mixed管理规则误伤；
//   - 普通admin不能借mixed身份创建个人课件；
//   - superadmin例外仍必须满足本人教案和具体教育域；
//   - region_admin、district_inspector不能误用superadmin例外；
//   - 他人教案和非法快照域始终fail-closed。

import (
	"errors"
	"testing"

	"tedna/internal/models"
)

func TestResolveCoursewareEducationDomainFromLessonPlanPolicy(
	t *testing.T,
) {
	t.Parallel()

	tests := []struct {
		name            string
		actor           *CoursewareActorContext
		lessonPlan      *models.LessonPlan
		allowSuperAdmin bool
		wantDomain      string
		wantErr         error
	}{
		{
			name: "普通K12作者继承本人教案快照域",
			actor: &CoursewareActorContext{
				UserID:          "user-k12",
				Role:            "operator",
				EducationDomain: models.EducationDomainK12,
			},
			lessonPlan: &models.LessonPlan{
				AuthorID:        "user-k12",
				EducationDomain: models.EducationDomainK12,
			},
			wantDomain: models.EducationDomainK12,
		},
		{
			name: "普通admin即使是本人教案也保持拒绝",
			actor: &CoursewareActorContext{
				UserID:          "admin-normal",
				Role:            models.RoleAdmin,
				EducationDomain: models.EducationDomainMixed,
			},
			lessonPlan: &models.LessonPlan{
				AuthorID:        "admin-normal",
				EducationDomain: models.EducationDomainVocational,
			},
			wantErr: ErrCoursewareCreationDomainRequired,
		},
		{
			name: "已确认superadmin可继承本人职教教案快照域",
			actor: &CoursewareActorContext{
				UserID:          "admin-super",
				Role:            models.RoleAdmin,
				EducationDomain: models.EducationDomainMixed,
			},
			lessonPlan: &models.LessonPlan{
				AuthorID:        "admin-super",
				EducationDomain: models.EducationDomainVocational,
			},
			allowSuperAdmin: true,
			wantDomain:      models.EducationDomainVocational,
		},
		{
			name: "superadmin例外不能读取他人教案",
			actor: &CoursewareActorContext{
				UserID:          "admin-super",
				Role:            models.RoleAdmin,
				EducationDomain: models.EducationDomainMixed,
			},
			lessonPlan: &models.LessonPlan{
				AuthorID:        "another-user",
				EducationDomain: models.EducationDomainAdult,
			},
			allowSuperAdmin: true,
			wantErr:         ErrCoursewareLessonPlanNotOwned,
		},
		{
			name: "具体域region_admin可继承本人同域教案",
			actor: &CoursewareActorContext{
				UserID:          "region-admin",
				Role:            models.RoleRegionAdmin,
				EducationDomain: models.EducationDomainK12,
			},
			lessonPlan: &models.LessonPlan{
				AuthorID:        "region-admin",
				EducationDomain: models.EducationDomainK12,
			},
			allowSuperAdmin: true,
			wantDomain:      models.EducationDomainK12,
		},
		{
			name: "具体域region_admin不能跨域创建课件",
			actor: &CoursewareActorContext{
				UserID:          "region-admin",
				Role:            models.RoleRegionAdmin,
				EducationDomain: models.EducationDomainK12,
			},
			lessonPlan: &models.LessonPlan{
				AuthorID:        "region-admin",
				EducationDomain: models.EducationDomainVocational,
			},
			allowSuperAdmin: true,
			wantErr:         ErrCoursewareCreationDomainRequired,
		},
		{
			name: "mixed region_admin不能创建个人课件",
			actor: &CoursewareActorContext{
				UserID:          "region-admin",
				Role:            models.RoleRegionAdmin,
				EducationDomain: models.EducationDomainMixed,
			},
			lessonPlan: &models.LessonPlan{
				AuthorID:        "region-admin",
				EducationDomain: models.EducationDomainK12,
			},
			allowSuperAdmin: true,
			wantErr:         ErrCoursewareCreationDomainRequired,
		},
		{
			name: "district_inspector不能复用superadmin例外",
			actor: &CoursewareActorContext{
				UserID:          "inspector",
				Role:            models.RoleDistrictInspector,
				EducationDomain: models.EducationDomainMixed,
			},
			lessonPlan: &models.LessonPlan{
				AuthorID:        "inspector",
				EducationDomain: models.EducationDomainK12,
			},
			allowSuperAdmin: true,
			wantErr:         ErrCoursewareCreationDomainRequired,
		},
		{
			name: "superadmin本人教案快照域非法仍拒绝",
			actor: &CoursewareActorContext{
				UserID:          "admin-super",
				Role:            models.RoleAdmin,
				EducationDomain: models.EducationDomainMixed,
			},
			lessonPlan: &models.LessonPlan{
				AuthorID:        "admin-super",
				EducationDomain: models.EducationDomainMixed,
			},
			allowSuperAdmin: true,
			wantErr:         ErrCoursewareLessonPlanDomainInvalid,
		},
		{
			name: "缺少Actor拒绝",
			lessonPlan: &models.LessonPlan{
				AuthorID:        "user-k12",
				EducationDomain: models.EducationDomainK12,
			},
			wantErr: ErrCoursewareActorRequired,
		},
		{
			name: "缺少教案拒绝",
			actor: &CoursewareActorContext{
				UserID:          "user-k12",
				Role:            "operator",
				EducationDomain: models.EducationDomainK12,
			},
			wantErr: ErrCoursewareLessonPlanDomainInvalid,
		},
	}

	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			gotDomain, err :=
				resolveCoursewareEducationDomainFromLessonPlanPolicy(
					testCase.actor,
					testCase.lessonPlan,
					testCase.allowSuperAdmin,
				)

			if testCase.wantErr != nil {
				if !errors.Is(err, testCase.wantErr) {
					t.Fatalf(
						"期望错误%v，实际错误%v",
						testCase.wantErr,
						err,
					)
				}
				if gotDomain != "" {
					t.Fatalf(
						"错误场景不应返回教育域，实际为%q",
						gotDomain,
					)
				}
				return
			}

			if err != nil {
				t.Fatalf(
					"不期望错误，实际错误%v",
					err,
				)
			}
			if gotDomain != testCase.wantDomain {
				t.Fatalf(
					"期望教育域%q，实际为%q",
					testCase.wantDomain,
					gotDomain,
				)
			}
		})
	}
}
