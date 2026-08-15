package models

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCWReviewHistoryIssueJSONDoesNotExposeMutableState(t *testing.T) {
	version :=
		&CoursewareReviewHistoryDeliveredInstruction{
			VersionID:  "version-1",
			VersionNo:  1,
			Content:    "当时正式交付的修改要求",
			SourceType: "manual",
		}

	payload, err :=
		json.Marshal(
			CoursewareReviewHistoryIssue{
				ID:                            "item-1",
				Severity:                      "high",
				Dimension:                     CWAIReviewDimensionTeachingLogic,
				DeliveredInstructionAvailable: true,
				DeliveredInstruction:          version,
				PreviousModificationRecords:   []CoursewareReviewHistoryModificationRecord{},
			},
		)
	if err != nil {
		t.Fatalf("marshal history issue: %v", err)
	}

	raw := string(payload)

	for _, forbidden := range []string{
		`"status"`,
		`"current_instruction_version_id"`,
		`"applied_instruction_version_id"`,
		`"content_hash"`,
		`"page_snapshot_hash"`,
		`"created_by"`,
		`"confirmed_by"`,
		`"applied_page_hash"`,
		`"resolved_by"`,
	} {
		if strings.Contains(raw, forbidden) {
			t.Fatalf(
				"history issue leaked forbidden field %s: %s",
				forbidden,
				raw,
			)
		}
	}
}

func TestCWReviewHistoryDetailKeepsHistoricalAndCurrentPagesSeparate(
	t *testing.T,
) {
	detail :=
		CoursewareReviewHistoryDetail{
			HistoricalPagesAvailable: true,
			HistoricalPages: []CoursewareReviewHistoryPage{
				{
					PageID:        "page-1",
					PageNumber:    1,
					HTMLContent:   "historical-html",
					CurrentExists: false,
				},
			},
			CurrentPages: []CoursewareReviewHistoryCurrentPage{
				{
					PageID:      "page-2",
					PageNumber:  1,
					HTMLContent: "current-html",
				},
			},
		}

	payload, err := json.Marshal(detail)
	if err != nil {
		t.Fatalf("marshal history detail: %v", err)
	}

	raw := string(payload)

	if !strings.Contains(
		raw,
		`"historical_pages"`,
	) ||
		!strings.Contains(
			raw,
			`"current_pages"`,
		) {
		t.Fatalf(
			"history/current page collections not separated: %s",
			raw,
		)
	}
}
