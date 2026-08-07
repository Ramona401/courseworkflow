package services

import (
	"context"
	"errors"
	"testing"

	"tedna/internal/models"
)

func TestResolveDesignerComponentDomainConcreteActorIgnoresRequestedDomain(
	t *testing.T,
) {
	actor := &AssistantActorContext{
		UserID:          "user-k12",
		Role:            "operator",
		EducationDomain: models.EducationDomainK12,
	}

	got, err := resolveDesignerComponentDomain(
		actor,
		&DesignerContext{
			EducationDomain: models.EducationDomainAdult,
		},
	)
	if err != nil {
		t.Fatalf(
			"具体教学域Actor不应失败：%v",
			err,
		)
	}

	if got != models.EducationDomainK12 {
		t.Fatalf(
			"普通Actor必须使用可信域，got=%q",
			got,
		)
	}
}

func TestResolveDesignerComponentDomainMixedAdminRequiresTarget(
	t *testing.T,
) {
	actor := &AssistantActorContext{
		UserID:          "admin-mixed",
		Role:            models.RoleAdmin,
		EducationDomain: models.EducationDomainMixed,
	}

	_, err := resolveDesignerComponentDomain(
		actor,
		&DesignerContext{},
	)

	if !errors.Is(
		err,
		ErrComponentEducationDomainRequired,
	) {
		t.Fatalf(
			"mixed管理员缺少目标域应返回required，got=%v",
			err,
		)
	}
}

func TestResolveDesignerComponentDomainMixedAdminUsesTarget(
	t *testing.T,
) {
	actor := &AssistantActorContext{
		UserID:          "admin-mixed",
		Role:            models.RoleAdmin,
		EducationDomain: models.EducationDomainMixed,
	}

	got, err := resolveDesignerComponentDomain(
		actor,
		&DesignerContext{
			EducationDomain: models.EducationDomainVocational,
		},
	)
	if err != nil {
		t.Fatalf(
			"mixed管理员显式合法目标域不应失败：%v",
			err,
		)
	}

	if got != models.EducationDomainVocational {
		t.Fatalf(
			"got=%q want=%q",
			got,
			models.EducationDomainVocational,
		)
	}
}

func TestParseAIReplyFailsClosedWithoutTeachingDomain(
	t *testing.T,
) {
	service := &LessonPlanGenService{}

	message := service.parseAIReply(
		context.Background(),
		"【推荐组件】请参考以下教学方案",
		&models.LessonPlan{
			Subject:         "数学",
			Grade:           "三年级",
			EducationDomain: models.EducationDomainMixed,
		},
	)

	if message.Type !=
		models.ConvMsgTypeComponents {
		t.Fatalf(
			"消息类型应保持components，got=%s",
			message.Type,
		)
	}

	if len(message.Components) != 0 {
		t.Fatalf(
			"非法教案快照域不能返回组件，got=%d",
			len(message.Components),
		)
	}
}
