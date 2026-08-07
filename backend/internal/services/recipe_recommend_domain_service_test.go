package services

import (
	"errors"
	"testing"

	"tedna/internal/models"
)

func TestBuildRecipeRecommendationMatchRequestUsesTrustedDomain(
	t *testing.T,
) {
	actor := &AssistantActorContext{
		UserID:          "teacher-vocational",
		Role:            models.RoleOperator,
		EducationDomain: models.EducationDomainVocational,
	}

	request, currentDomain, err :=
		buildRecipeRecommendationMatchRequest(
			actor,
			&models.RecipeRecommendRequest{
				EducationDomain: models.EducationDomainAdult,
				Subject:         " 信息技术 ",
				GradeRange:      "5",
			},
			3,
		)
	if err != nil {
		t.Fatalf(
			"普通Actor推荐请求不应失败：%v",
			err,
		)
	}

	if currentDomain !=
		models.EducationDomainVocational {
		t.Fatalf(
			"必须使用可信Actor域，got=%q",
			currentDomain,
		)
	}

	if request.EducationDomain !=
		models.EducationDomainVocational {
		t.Fatalf(
			"请求域未覆盖为可信域，got=%q",
			request.EducationDomain,
		)
	}

	if request.Subject != "信息技术" {
		t.Fatalf(
			"学科未清洗，got=%q",
			request.Subject,
		)
	}

	if request.GradeRange != "5" {
		t.Fatalf(
			"数字年级不应改变，got=%q",
			request.GradeRange,
		)
	}

	if request.Limit != 3 {
		t.Fatalf(
			"普通推荐条数错误，got=%d",
			request.Limit,
		)
	}
}

func TestBuildRecipeRecommendationMatchRequestMixedAdmin(
	t *testing.T,
) {
	actor := &AssistantActorContext{
		UserID:          "admin-mixed",
		Role:            models.RoleAdmin,
		EducationDomain: models.EducationDomainMixed,
	}

	t.Run(
		"缺少显式目标域拒绝",
		func(t *testing.T) {
			_, _, err :=
				buildRecipeRecommendationMatchRequest(
					actor,
					&models.RecipeRecommendRequest{
						Subject: "数学",
					},
					5,
				)

			if !errors.Is(
				err,
				ErrComponentEducationDomainRequired,
			) {
				t.Fatalf(
					"got=%v",
					err,
				)
			}
		},
	)

	t.Run(
		"显式成人教育域通过",
		func(t *testing.T) {
			request,
				currentDomain,
				err :=
				buildRecipeRecommendationMatchRequest(
					actor,
					&models.RecipeRecommendRequest{
						EducationDomain: models.EducationDomainAdult,
						Subject:         "数学",
					},
					5,
				)

			if err != nil {
				t.Fatalf(
					"显式合法域不应失败：%v",
					err,
				)
			}

			if currentDomain !=
				models.EducationDomainAdult {
				t.Fatalf(
					"got=%q",
					currentDomain,
				)
			}

			if request.Limit != 5 {
				t.Fatalf(
					"画像推荐条数错误，got=%d",
					request.Limit,
				)
			}
		},
	)

	t.Run(
		"common不能作为当前推荐域",
		func(t *testing.T) {
			_, _, err :=
				buildRecipeRecommendationMatchRequest(
					actor,
					&models.RecipeRecommendRequest{
						EducationDomain: models.EducationDomainCommon,
						Subject:         "数学",
					},
					5,
				)

			if !errors.Is(
				err,
				ErrComponentEducationDomainInvalid,
			) {
				t.Fatalf(
					"got=%v",
					err,
				)
			}
		},
	)
}

func TestBuildRecipeRecommendationMatchRequestRequiresSubject(
	t *testing.T,
) {
	actor := &AssistantActorContext{
		UserID:          "teacher-k12",
		Role:            models.RoleOperator,
		EducationDomain: models.EducationDomainK12,
	}

	_, _, err :=
		buildRecipeRecommendationMatchRequest(
			actor,
			&models.RecipeRecommendRequest{},
			3,
		)

	if !errors.Is(
		err,
		ErrRecipeSubjectRequired,
	) {
		t.Fatalf(
			"got=%v",
			err,
		)
	}
}
