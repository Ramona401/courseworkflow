package services

import (
	"errors"
	"testing"

	"tedna/internal/models"
)

func newCoursewareAccessTestActor(
	role string,
	domain string,
) *CoursewareActorContext {
	return &CoursewareActorContext{
		UserID:          "user-1",
		Role:            role,
		SchoolID:        "school-1",
		EducationDomain: domain,
		MyGroupIDs: []string{
			"group-1",
		},
		MyLeadGroupIDs: []string{
			"group-1",
		},
		MyLeadOrBackboneGroupIDs: []string{
			"group-1",
		},
	}
}

func newCoursewareAccessTestResource(
	domain string,
) *models.Courseware {
	return &models.Courseware{
		ID:              "courseware-1",
		UserID:          "owner-1",
		EducationDomain: domain,
	}
}

func TestResolveCoursewareCreationEducationDomain(
	t *testing.T,
) {
	t.Parallel()

	tests := []struct {
		name       string
		actor      *CoursewareActorContext
		wantDomain string
		wantErr    error
	}{
		{
			name: "K12具体教学域允许创建",
			actor: newCoursewareAccessTestActor(
				models.RoleOperator,
				models.EducationDomainK12,
			),
			wantDomain: models.EducationDomainK12,
		},
		{
			name: "职教域自动规范化大小写和空格",
			actor: newCoursewareAccessTestActor(
				models.RoleOperator,
				"  VOCATIONAL  ",
			),
			wantDomain: models.EducationDomainVocational,
		},
		{
			name: "成教具体教学域允许创建",
			actor: newCoursewareAccessTestActor(
				models.RoleSeniorOperator,
				models.EducationDomainAdult,
			),
			wantDomain: models.EducationDomainAdult,
		},
		{
			name: "mixed管理上下文禁止直接创建教学课件",
			actor: newCoursewareAccessTestActor(
				models.RoleAdmin,
				models.EducationDomainMixed,
			),
			wantErr: ErrCoursewareCreationDomainRequired,
		},
		{
			name: "common不能作为当前用户教学域",
			actor: newCoursewareAccessTestActor(
				models.RoleOperator,
				models.EducationDomainCommon,
			),
			wantErr: ErrCoursewareCreationDomainRequired,
		},
		{
			name: "空教育域fail-closed",
			actor: newCoursewareAccessTestActor(
				models.RoleOperator,
				"",
			),
			wantErr: ErrCoursewareCreationDomainRequired,
		},
		{
			name: "非法教育域fail-closed",
			actor: newCoursewareAccessTestActor(
				models.RoleOperator,
				"invalid",
			),
			wantErr: ErrCoursewareCreationDomainRequired,
		},
		{
			name:    "nil Actor拒绝",
			actor:   nil,
			wantErr: ErrCoursewareActorRequired,
		},
		{
			name: "空用户ID拒绝",
			actor: &CoursewareActorContext{
				Role:            models.RoleOperator,
				EducationDomain: models.EducationDomainK12,
			},
			wantErr: ErrCoursewareActorRequired,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			gotDomain, err :=
				ResolveCoursewareCreationEducationDomain(
					test.actor,
				)

			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Fatalf(
						"期望错误%v，实际错误%v",
						test.wantErr,
						err,
					)
				}
				return
			}

			if err != nil {
				t.Fatalf("不期望错误，实际错误：%v", err)
			}
			if gotDomain != test.wantDomain {
				t.Fatalf(
					"期望教育域%s，实际%s",
					test.wantDomain,
					gotDomain,
				)
			}
		})
	}
}

func TestResolveCoursewareEducationDomainFromLessonPlan(
	t *testing.T,
) {
	t.Parallel()

	newPlan := func(
		authorID string,
		domain string,
	) *models.LessonPlan {
		return &models.LessonPlan{
			ID:              "lesson-plan-1",
			AuthorID:        authorID,
			EducationDomain: domain,
		}
	}

	tests := []struct {
		name       string
		actor      *CoursewareActorContext
		lessonPlan *models.LessonPlan
		wantDomain string
		wantErr    error
	}{
		{
			name: "作者从K12教案创建时继承K12快照",
			actor: newCoursewareAccessTestActor(
				models.RoleOperator,
				models.EducationDomainK12,
			),
			lessonPlan: newPlan(
				"user-1",
				models.EducationDomainK12,
			),
			wantDomain: models.EducationDomainK12,
		},
		{
			name: "作者跨域换校后仍继承历史职教教案快照",
			actor: newCoursewareAccessTestActor(
				models.RoleOperator,
				models.EducationDomainK12,
			),
			lessonPlan: newPlan(
				"user-1",
				models.EducationDomainVocational,
			),
			wantDomain: models.EducationDomainVocational,
		},
		{
			name: "普通作者暂时无确定当前域仍可使用自己的合法历史快照",
			actor: newCoursewareAccessTestActor(
				models.RoleOperator,
				"",
			),
			lessonPlan: newPlan(
				"user-1",
				models.EducationDomainAdult,
			),
			wantDomain: models.EducationDomainAdult,
		},
		{
			name: "非作者不能直接从他人教案创建课件",
			actor: newCoursewareAccessTestActor(
				models.RoleOperator,
				models.EducationDomainK12,
			),
			lessonPlan: newPlan(
				"another-user",
				models.EducationDomainK12,
			),
			wantErr: ErrCoursewareLessonPlanNotOwned,
		},
		{
			name: "mixed管理员即使是存量教案作者也不能创建个人课件",
			actor: newCoursewareAccessTestActor(
				models.RoleAdmin,
				models.EducationDomainMixed,
			),
			lessonPlan: newPlan(
				"user-1",
				models.EducationDomainK12,
			),
			wantErr: ErrCoursewareCreationDomainRequired,
		},
		{
			name: "区域管理员不能从教案创建个人课件",
			actor: newCoursewareAccessTestActor(
				models.RoleRegionAdmin,
				models.EducationDomainMixed,
			),
			lessonPlan: newPlan(
				"user-1",
				models.EducationDomainVocational,
			),
			wantErr: ErrCoursewareCreationDomainRequired,
		},
		{
			name: "common教案不能派生具体课件",
			actor: newCoursewareAccessTestActor(
				models.RoleOperator,
				models.EducationDomainK12,
			),
			lessonPlan: newPlan(
				"user-1",
				models.EducationDomainCommon,
			),
			wantErr: ErrCoursewareLessonPlanDomainInvalid,
		},
		{
			name: "mixed教案快照非法",
			actor: newCoursewareAccessTestActor(
				models.RoleOperator,
				models.EducationDomainK12,
			),
			lessonPlan: newPlan(
				"user-1",
				models.EducationDomainMixed,
			),
			wantErr: ErrCoursewareLessonPlanDomainInvalid,
		},
		{
			name: "空教案快照fail-closed",
			actor: newCoursewareAccessTestActor(
				models.RoleOperator,
				models.EducationDomainK12,
			),
			lessonPlan: newPlan(
				"user-1",
				"",
			),
			wantErr: ErrCoursewareLessonPlanDomainInvalid,
		},
		{
			name: "非法教案快照fail-closed",
			actor: newCoursewareAccessTestActor(
				models.RoleOperator,
				models.EducationDomainK12,
			),
			lessonPlan: newPlan(
				"user-1",
				"invalid",
			),
			wantErr: ErrCoursewareLessonPlanDomainInvalid,
		},
		{
			name:  "nil Actor拒绝",
			actor: nil,
			lessonPlan: newPlan(
				"user-1",
				models.EducationDomainK12,
			),
			wantErr: ErrCoursewareActorRequired,
		},
		{
			name: "nil教案拒绝",
			actor: newCoursewareAccessTestActor(
				models.RoleOperator,
				models.EducationDomainK12,
			),
			lessonPlan: nil,
			wantErr:    ErrCoursewareLessonPlanDomainInvalid,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			gotDomain, err :=
				ResolveCoursewareEducationDomainFromLessonPlan(
					test.actor,
					test.lessonPlan,
				)

			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Fatalf(
						"期望错误%v，实际错误%v",
						test.wantErr,
						err,
					)
				}
				return
			}

			if err != nil {
				t.Fatalf("不期望错误，实际错误：%v", err)
			}
			if gotDomain != test.wantDomain {
				t.Fatalf(
					"期望教育域%s，实际%s",
					test.wantDomain,
					gotDomain,
				)
			}
		})
	}
}

func TestValidateCoursewareEducationDomainForActor(
	t *testing.T,
) {
	t.Parallel()

	tests := []struct {
		name       string
		actor      *CoursewareActorContext
		courseware *models.Courseware
		wantErr    error
	}{
		{
			name: "K12用户可以进入K12课件",
			actor: newCoursewareAccessTestActor(
				models.RoleOperator,
				models.EducationDomainK12,
			),
			courseware: newCoursewareAccessTestResource(
				models.EducationDomainK12,
			),
		},
		{
			name: "具体教学域可以读取common资源",
			actor: newCoursewareAccessTestActor(
				models.RoleOperator,
				models.EducationDomainVocational,
			),
			courseware: newCoursewareAccessTestResource(
				models.EducationDomainCommon,
			),
		},
		{
			name: "K12用户不能进入职教课件",
			actor: newCoursewareAccessTestActor(
				models.RoleOperator,
				models.EducationDomainK12,
			),
			courseware: newCoursewareAccessTestResource(
				models.EducationDomainVocational,
			),
			wantErr: ErrCoursewareEducationDomainMismatch,
		},
		{
			name: "mixed admin可以跨域管理合法课件",
			actor: newCoursewareAccessTestActor(
				models.RoleAdmin,
				models.EducationDomainMixed,
			),
			courseware: newCoursewareAccessTestResource(
				models.EducationDomainAdult,
			),
		},
		{
			name: "mixed区域管理员可以跨域管理合法课件",
			actor: newCoursewareAccessTestActor(
				models.RoleRegionAdmin,
				models.EducationDomainMixed,
			),
			courseware: newCoursewareAccessTestResource(
				models.EducationDomainVocational,
			),
		},
		{
			name: "普通角色伪造mixed必须拒绝",
			actor: newCoursewareAccessTestActor(
				models.RoleOperator,
				models.EducationDomainMixed,
			),
			courseware: newCoursewareAccessTestResource(
				models.EducationDomainK12,
			),
			wantErr: ErrCoursewareEducationDomainMismatch,
		},
		{
			name: "current common必须拒绝",
			actor: newCoursewareAccessTestActor(
				models.RoleOperator,
				models.EducationDomainCommon,
			),
			courseware: newCoursewareAccessTestResource(
				models.EducationDomainCommon,
			),
			wantErr: ErrCoursewareEducationDomainMismatch,
		},
		{
			name: "非法资源域必须拒绝",
			actor: newCoursewareAccessTestActor(
				models.RoleOperator,
				models.EducationDomainK12,
			),
			courseware: newCoursewareAccessTestResource(
				"invalid",
			),
			wantErr: ErrCoursewareEducationDomainInvalid,
		},
		{
			name: "资源域为空必须拒绝",
			actor: newCoursewareAccessTestActor(
				models.RoleOperator,
				models.EducationDomainK12,
			),
			courseware: newCoursewareAccessTestResource(
				"",
			),
			wantErr: ErrCoursewareEducationDomainInvalid,
		},
		{
			name: "nil课件必须拒绝",
			actor: newCoursewareAccessTestActor(
				models.RoleOperator,
				models.EducationDomainK12,
			),
			courseware: nil,
			wantErr:    ErrCoursewareEducationDomainInvalid,
		},
		{
			name:  "nil Actor必须拒绝",
			actor: nil,
			courseware: newCoursewareAccessTestResource(
				models.EducationDomainK12,
			),
			wantErr: ErrCoursewareActorRequired,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateCoursewareEducationDomainForActor(
				test.actor,
				test.courseware,
			)

			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Fatalf(
						"期望错误%v，实际错误%v",
						test.wantErr,
						err,
					)
				}
				return
			}

			if err != nil {
				t.Fatalf("不期望错误，实际错误：%v", err)
			}
		})
	}
}

func TestScopeCoursewareActorToSnapshot(
	t *testing.T,
) {
	t.Parallel()

	t.Run(
		"mixed管理员进入职教课件后收敛到职教快照",
		func(t *testing.T) {
			t.Parallel()

			actor := newCoursewareAccessTestActor(
				models.RoleAdmin,
				models.EducationDomainMixed,
			)
			courseware := newCoursewareAccessTestResource(
				models.EducationDomainVocational,
			)

			scoped, err := ScopeCoursewareActorToSnapshot(
				actor,
				courseware,
			)
			if err != nil {
				t.Fatalf("不期望错误，实际错误：%v", err)
			}
			if scoped.EducationDomain !=
				models.EducationDomainVocational {
				t.Fatalf(
					"期望收敛到%s，实际%s",
					models.EducationDomainVocational,
					scoped.EducationDomain,
				)
			}

			// 原Actor仍保持mixed，证明函数没有原地修改调用方对象。
			if actor.EducationDomain !=
				models.EducationDomainMixed {
				t.Fatalf(
					"原Actor不应被修改，实际教育域：%s",
					actor.EducationDomain,
				)
			}

			// 验证组织切片已经深拷贝。
			scoped.MyGroupIDs[0] = "changed-group"
			if actor.MyGroupIDs[0] != "group-1" {
				t.Fatal("scoped切片修改污染了原Actor")
			}
		},
	)

	t.Run(
		"普通K12用户进入K12课件后保持K12",
		func(t *testing.T) {
			t.Parallel()

			actor := newCoursewareAccessTestActor(
				models.RoleOperator,
				models.EducationDomainK12,
			)
			courseware := newCoursewareAccessTestResource(
				models.EducationDomainK12,
			)

			scoped, err := ScopeCoursewareActorToSnapshot(
				actor,
				courseware,
			)
			if err != nil {
				t.Fatalf("不期望错误，实际错误：%v", err)
			}
			if scoped.EducationDomain !=
				models.EducationDomainK12 {
				t.Fatalf(
					"期望K12，实际%s",
					scoped.EducationDomain,
				)
			}
		},
	)

	t.Run(
		"common只能作为候选资源不能作为具体课件运行域",
		func(t *testing.T) {
			t.Parallel()

			actor := newCoursewareAccessTestActor(
				models.RoleOperator,
				models.EducationDomainAdult,
			)
			courseware := newCoursewareAccessTestResource(
				models.EducationDomainCommon,
			)

			_, err := ScopeCoursewareActorToSnapshot(
				actor,
				courseware,
			)
			if !errors.Is(
				err,
				ErrCoursewareRuntimeDomainRequired,
			) {
				t.Fatalf(
					"期望错误%v，实际错误%v",
					ErrCoursewareRuntimeDomainRequired,
					err,
				)
			}
		},
	)

	t.Run(
		"跨域课件在收敛前即被拒绝",
		func(t *testing.T) {
			t.Parallel()

			actor := newCoursewareAccessTestActor(
				models.RoleOperator,
				models.EducationDomainK12,
			)
			courseware := newCoursewareAccessTestResource(
				models.EducationDomainAdult,
			)

			_, err := ScopeCoursewareActorToSnapshot(
				actor,
				courseware,
			)
			if !errors.Is(
				err,
				ErrCoursewareEducationDomainMismatch,
			) {
				t.Fatalf(
					"期望错误%v，实际错误%v",
					ErrCoursewareEducationDomainMismatch,
					err,
				)
			}
		},
	)
}
