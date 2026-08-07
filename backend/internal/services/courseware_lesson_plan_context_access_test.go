package services

import (
	"context"
	"errors"
	"testing"

	"tedna/internal/models"
)

func TestResolveCoursewareLessonPlanContextAccessError(t *testing.T) {
	normalErr := errors.New("普通查看链内部错误")
	reviewErr := errors.New("审核查看链错误")

	tests := []struct {
		name      string
		normalErr error
		reviewErr error
		expected  error
	}{
		{
			name:      "两条路径都只是未授权时不制造内部错误",
			normalErr: nil,
			reviewErr: nil,
			expected:  nil,
		},
		{
			name:      "审核路径错误优先返回",
			normalErr: normalErr,
			reviewErr: reviewErr,
			expected:  reviewErr,
		},
		{
			name:      "仅普通路径错误时保留真实错误",
			normalErr: normalErr,
			reviewErr: nil,
			expected:  normalErr,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := resolveCoursewareLessonPlanContextAccessError(
				test.normalErr,
				test.reviewErr,
			)
			if !errors.Is(actual, test.expected) {
				t.Fatalf("expected %v, got %v", test.expected, actual)
			}
		})
	}
}

func TestCanReviewLoadedCoursewareAllowsAdminReviewMaterialRead(t *testing.T) {
	service := NewCoursewareReviewService()
	courseware := &models.Courseware{
		ID:              "courseware-review-material",
		UserID:          "owner-user",
		EducationDomain: models.EducationDomainK12,
	}
	actor := &CoursewareActorContext{
		UserID:          "admin-user",
		Role:            models.RoleAdmin,
		EducationDomain: models.EducationDomainMixed,
	}

	allowed, err := service.CanReviewLoadedCourseware(
		context.Background(),
		courseware,
		actor,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Fatal("expected admin review-detail access to be allowed")
	}
}

func TestCoursewareLessonPlanContextPolicyKeepsNormalViewAndReviewViewIndependent(
	t *testing.T,
) {
	courseware := &models.Courseware{
		ID:           "courseware-private",
		UserID:       "owner-user",
		PublishState: models.CWPublishPrivate,
	}
	actor := &CoursewareActorContext{
		UserID: "other-user",
		Role:   models.RoleOperator,
	}

	if coursewareViewPolicyAllows(courseware, actor, false, false) {
		t.Fatal("private courseware must not be granted by normal view policy")
	}

	if err := resolveCoursewareLessonPlanContextAccessError(nil, nil); err != nil {
		t.Fatalf("independent authorization paths should resolve clean denial, got %v", err)
	}
}
