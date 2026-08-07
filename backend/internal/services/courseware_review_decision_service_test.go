package services

import (
	"reflect"
	"testing"

	"tedna/internal/models"
)

func TestIsValidCWReviewDecision(
	t *testing.T,
) {
	t.Parallel()

	testCases := []struct {
		name string
		req  *models.CWReviewDecisionRequest
		want bool
	}{
		{
			name: "空请求无效",
			req:  nil,
			want: false,
		},
		{
			name: "通过有效",
			req: &models.CWReviewDecisionRequest{
				Decision:
					models.ReviewDecisionApproved,
			},
			want: true,
		},
		{
			name: "退回有效",
			req: &models.CWReviewDecisionRequest{
				Decision:
					models.ReviewDecisionRevision,
			},
			want: true,
		},
		{
			name: "未知值无效",
			req: &models.CWReviewDecisionRequest{
				Decision: "unknown",
			},
			want: false,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(
			testCase.name,
			func(t *testing.T) {
				t.Parallel()

				got := isValidCWReviewDecision(
					testCase.req,
				)
				if got != testCase.want {
					t.Fatalf(
						"isValidCWReviewDecision()=%v，期望%v",
						got,
						testCase.want,
					)
				}
			},
		)
	}
}

func TestNormalizeCWFormalReviewItemIDs(
	t *testing.T,
) {
	t.Parallel()

	input := []string{
		" item-a ",
		"",
		"item-b",
		"item-a",
		"  ",
		"item-c",
	}

	want := []string{
		"item-a",
		"item-b",
		"item-c",
	}

	got := normalizeCWFormalReviewItemIDs(
		input,
	)

	if !reflect.DeepEqual(got, want) {
		t.Fatalf(
			"normalizeCWFormalReviewItemIDs()=%v，期望%v",
			got,
			want,
		)
	}
}
