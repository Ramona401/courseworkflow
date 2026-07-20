package services

import (
	"errors"
	"testing"

	"tedna/internal/models"
)

func TestValidateCoursewareDomainForAuthorizedActor(
	t *testing.T,
) {
	t.Parallel()

	t.Run("作者跨域换校后仍可查看自己的历史快照", func(t *testing.T) {
		t.Parallel()

		actor := newCoursewareAccessTestActor(
			models.RoleOperator,
			models.EducationDomainK12,
		)
		courseware := newCoursewareAccessTestResource(
			models.EducationDomainVocational,
		)
		courseware.UserID = actor.UserID

		err := validateCoursewareDomainForAuthorizedActor(
			actor,
			courseware,
			true,
		)
		if err != nil {
			t.Fatalf("不期望错误，实际错误：%v", err)
		}
	})

	t.Run("非作者跨域访问仍然拒绝", func(t *testing.T) {
		t.Parallel()

		actor := newCoursewareAccessTestActor(
			models.RoleOperator,
			models.EducationDomainK12,
		)
		courseware := newCoursewareAccessTestResource(
			models.EducationDomainVocational,
		)

		err := validateCoursewareDomainForAuthorizedActor(
			actor,
			courseware,
			true,
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
	})

	t.Run("非法历史快照即使是作者也拒绝", func(t *testing.T) {
		t.Parallel()

		actor := newCoursewareAccessTestActor(
			models.RoleOperator,
			models.EducationDomainK12,
		)
		courseware := newCoursewareAccessTestResource(
			"invalid",
		)
		courseware.UserID = actor.UserID

		err := validateCoursewareDomainForAuthorizedActor(
			actor,
			courseware,
			true,
		)
		if !errors.Is(
			err,
			ErrCoursewareEducationDomainInvalid,
		) {
			t.Fatalf(
				"期望错误%v，实际错误%v",
				ErrCoursewareEducationDomainInvalid,
				err,
			)
		}
	})
}

func TestCoursewareViewPolicyAllows(
	t *testing.T,
) {
	t.Parallel()

	actor := newCoursewareAccessTestActor(
		models.RoleOperator,
		models.EducationDomainK12,
	)
	courseware := newCoursewareAccessTestResource(
		models.EducationDomainK12,
	)

	tests := []struct {
		name               string
		ownerID            string
		role               string
		publishState       string
		collabState        string
		isCollabMember     bool
		sharesOrganization bool
		want               bool
	}{
		{
			name:    "作者本人放行",
			ownerID: actor.UserID,
			role:    models.RoleOperator,
			want:    true,
		},
		{
			name:    "admin管理通道放行",
			ownerID: "owner-2",
			role:    models.RoleAdmin,
			want:    true,
		},
		{
			name:           "进行中的集体备课参与者放行",
			ownerID:        "owner-2",
			role:           models.RoleOperator,
			collabState:    models.CWCollabInSession,
			isCollabMember: true,
			want:           true,
		},
		{
			name:               "同组织共享课件放行",
			ownerID:            "owner-2",
			role:               models.RoleOperator,
			publishState:       models.CWPublishPublishedShared,
			sharesOrganization: true,
			want:               true,
		},
		{
			name:               "私有课件即使同组织也拒绝",
			ownerID:            "owner-2",
			role:               models.RoleOperator,
			publishState:       models.CWPublishPrivate,
			sharesOrganization: true,
			want:               false,
		},
		{
			name:           "非集体备课态成员记录不放行",
			ownerID:        "owner-2",
			role:           models.RoleOperator,
			collabState:    models.CWCollabIdle,
			isCollabMember: true,
			want:           false,
		},
		{
			name:    "region_admin不会因角色自动放行普通详情",
			ownerID: "owner-2",
			role:    models.RoleRegionAdmin,
			want:    false,
		},
		{
			name:    "district_inspector不会因角色自动放行普通详情",
			ownerID: "owner-2",
			role:    models.RoleDistrictInspector,
			want:    false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			localActor := *actor
			localActor.Role = test.role

			localCourseware := *courseware
			localCourseware.UserID = test.ownerID
			localCourseware.PublishState =
				test.publishState
			localCourseware.CollabState =
				test.collabState

			got := coursewareViewPolicyAllows(
				&localCourseware,
				&localActor,
				test.isCollabMember,
				test.sharesOrganization,
			)
			if got != test.want {
				t.Fatalf(
					"期望%v，实际%v",
					test.want,
					got,
				)
			}
		})
	}
}

func TestCoursewareEditPolicyAllows(
	t *testing.T,
) {
	t.Parallel()

	actor := newCoursewareAccessTestActor(
		models.RoleOperator,
		models.EducationDomainK12,
	)
	courseware := newCoursewareAccessTestResource(
		models.EducationDomainK12,
	)
	courseware.Status = models.CoursewareStatusPreview
	courseware.PublishState = models.CWPublishPrivate

	t.Run("作者可编辑", func(t *testing.T) {
		t.Parallel()

		local := *courseware
		local.UserID = actor.UserID

		if !coursewareEditPolicyAllows(
			&local,
			actor,
			false,
			true,
		) {
			t.Fatal("作者应可编辑")
		}
	})

	t.Run("admin仅在完整编辑管理通道放行", func(t *testing.T) {
		t.Parallel()

		admin := *actor
		admin.Role = models.RoleAdmin

		if !coursewareEditPolicyAllows(
			courseware,
			&admin,
			false,
			true,
		) {
			t.Fatal("完整编辑通道应放行admin")
		}

		if coursewareEditPolicyAllows(
			courseware,
			&admin,
			false,
			false,
		) {
			t.Fatal("教研微调通道不应自动放行admin")
		}
	})

	t.Run("集体备课参与者可编辑", func(t *testing.T) {
		t.Parallel()

		local := *courseware
		local.CollabState =
			models.CWCollabInSession

		if !coursewareEditPolicyAllows(
			&local,
			actor,
			true,
			false,
		) {
			t.Fatal("集体备课参与者应可微调")
		}
	})

	t.Run("审核锁定态任何人都不能编辑", func(t *testing.T) {
		t.Parallel()

		local := *courseware
		local.UserID = actor.UserID
		local.Status =
			models.CoursewareStatusInPipeline

		if coursewareEditPolicyAllows(
			&local,
			actor,
			false,
			true,
		) {
			t.Fatal("审核锁定态不应允许编辑")
		}
	})

	t.Run("发布审核submitted任何人都不能编辑", func(t *testing.T) {
		t.Parallel()

		local := *courseware
		local.UserID = actor.UserID
		local.PublishState =
			models.CWPublishSubmitted

		if coursewareEditPolicyAllows(
			&local,
			actor,
			false,
			true,
		) {
			t.Fatal("submitted状态不应允许编辑")
		}
	})
}

func TestScopeAuthorizedCoursewareActor(
	t *testing.T,
) {
	t.Parallel()

	actor := newCoursewareAccessTestActor(
		models.RoleOperator,
		models.EducationDomainK12,
	)
	courseware := newCoursewareAccessTestResource(
		models.EducationDomainVocational,
	)

	scoped := scopeAuthorizedCoursewareActor(
		actor,
		courseware,
	)

	if scoped.EducationDomain !=
		models.EducationDomainVocational {
		t.Fatalf(
			"期望收敛到%s，实际%s",
			models.EducationDomainVocational,
			scoped.EducationDomain,
		)
	}

	if actor.EducationDomain !=
		models.EducationDomainK12 {
		t.Fatal("原Actor不应被修改")
	}

	scoped.MyGroupIDs[0] = "changed"
	if actor.MyGroupIDs[0] != "group-1" {
		t.Fatal("收敛Actor的切片修改污染了原Actor")
	}
}
