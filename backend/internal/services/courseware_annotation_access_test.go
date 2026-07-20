package services

import (
	"testing"
	"time"

	"tedna/internal/models"
)

func TestCoursewareAnnotationManagePolicyAllows(
	t *testing.T,
) {
	courseware := &models.Courseware{
		ID:     "courseware-1",
		UserID: "owner-1",
	}

	annotation := &models.CoursewareAnnotation{
		ID:           "annotation-1",
		CoursewareID: "courseware-1",
		ReviewerID:   "reviewer-1",
	}

	tests := []struct {
		name  string
		actor *CoursewareActorContext
		want  bool
	}{
		{
			name: "课件作者",
			actor: &CoursewareActorContext{
				UserID: "owner-1",
			},
			want: true,
		},
		{
			name: "批注创建者",
			actor: &CoursewareActorContext{
				UserID: "reviewer-1",
			},
			want: true,
		},
		{
			name: "admin不自动放行",
			actor: &CoursewareActorContext{
				UserID: "admin-1",
				Role:   models.RoleAdmin,
			},
			want: false,
		},
		{
			name: "其它参与者不管理他人批注",
			actor: &CoursewareActorContext{
				UserID: "member-1",
			},
			want: false,
		},
	}

	for _, testCase := range tests {
		t.Run(
			testCase.name,
			func(t *testing.T) {
				got :=
					coursewareAnnotationManagePolicyAllows(
						courseware,
						annotation,
						testCase.actor,
					)

				if got != testCase.want {
					t.Fatalf(
						"管理策略结果不一致: got=%v want=%v",
						got,
						testCase.want,
					)
				}
			},
		)
	}
}

func TestCoursewareAnnotationRevisionUnchanged(
	t *testing.T,
) {
	now := time.Now()
	same := now
	later := now.Add(time.Second)

	base := &models.CoursewareAnnotation{
		ID:           "annotation-1",
		CoursewareID: "courseware-1",
		PageNumber:   2,
		ReviewerID:   "reviewer-1",
		UpdatedAt:    now,
	}

	if !coursewareAnnotationRevisionUnchanged(
		base,
		&models.CoursewareAnnotation{
			ID:           "annotation-1",
			CoursewareID: "courseware-1",
			PageNumber:   2,
			ReviewerID:   "reviewer-1",
			UpdatedAt:    same,
		},
	) {
		t.Fatal(
			"相同数据库版本应判定为未变化",
		)
	}

	if coursewareAnnotationRevisionUnchanged(
		base,
		&models.CoursewareAnnotation{
			ID:           "annotation-1",
			CoursewareID: "courseware-1",
			PageNumber:   2,
			ReviewerID:   "reviewer-1",
			UpdatedAt:    later,
		},
	) {
		t.Fatal(
			"更新时间变化应判定为冲突",
		)
	}

	if coursewareAnnotationRevisionUnchanged(
		base,
		&models.CoursewareAnnotation{
			ID:           "annotation-1",
			CoursewareID: "courseware-2",
			PageNumber:   2,
			ReviewerID:   "reviewer-1",
			UpdatedAt:    same,
		},
	) {
		t.Fatal(
			"课件归属变化应判定为冲突",
		)
	}
}
