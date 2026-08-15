package handlers

import "testing"

func TestCWReviewHistoryDetailPathExtractor(t *testing.T) {
	reviewID :=
		"123e4567-e89b-12d3-a456-426614174000"

	validPath :=
		"/api/v1/courseware-reviews/records/" +
			reviewID +
			"/detail"

	if got :=
		extractCWReviewedRecordDetailID(
			validPath,
		); got != reviewID {
		t.Fatalf(
			"unexpected extracted review id: %q",
			got,
		)
	}

	for _, path := range []string{
		"/api/v1/courseware-reviews/" +
			reviewID +
			"/detail",

		"/api/v1/courseware-reviews/records/" +
			reviewID,

		"/api/v1/courseware-reviews/records/" +
			reviewID +
			"/other/detail",

		"/api/v1/courseware-reviews/records//detail",
	} {
		if got :=
			extractCWReviewedRecordDetailID(
				path,
			); got != "" {
			t.Fatalf(
				"invalid path accepted: %s -> %q",
				path,
				got,
			)
		}
	}
}
