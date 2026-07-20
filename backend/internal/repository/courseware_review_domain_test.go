package repository

import (
	"strings"
	"testing"

	"tedna/internal/models"
)

func TestBuildCWReviewEducationDomainFilter(
	t *testing.T,
) {
	t.Parallel()

	tests := []struct {
		name        string
		alias       string
		domain      string
		startIdx    int
		wantSQLPart string
		wantArgs    int
		wantNextIdx int
	}{
		{
			name:        "K12精确过滤",
			alias:       "c",
			domain:      models.EducationDomainK12,
			startIdx:    3,
			wantSQLPart: "c.education_domain = $3",
			wantArgs:    1,
			wantNextIdx: 4,
		},
		{
			name:        "mixed只允许具体教学域",
			alias:       "c",
			domain:      models.EducationDomainMixed,
			startIdx:    5,
			wantSQLPart: "c.education_domain IN ('k12', 'vocational', 'adult')",
			wantArgs:    0,
			wantNextIdx: 5,
		},
		{
			name:        "common当前域fail-closed",
			alias:       "c",
			domain:      models.EducationDomainCommon,
			startIdx:    1,
			wantSQLPart: "1 = 0",
			wantArgs:    0,
			wantNextIdx: 1,
		},
		{
			name:        "非法当前域fail-closed",
			alias:       "c",
			domain:      "invalid",
			startIdx:    4,
			wantSQLPart: "1 = 0",
			wantArgs:    0,
			wantNextIdx: 4,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			sqlPart, args, nextIdx :=
				buildCWReviewEducationDomainFilter(
					test.alias,
					test.domain,
					test.startIdx,
				)

			if !strings.Contains(
				sqlPart,
				test.wantSQLPart,
			) {
				t.Fatalf(
					"SQL期望包含%q，实际%q",
					test.wantSQLPart,
					sqlPart,
				)
			}
			if len(args) != test.wantArgs {
				t.Fatalf(
					"参数数量期望%d，实际%d",
					test.wantArgs,
					len(args),
				)
			}
			if nextIdx != test.wantNextIdx {
				t.Fatalf(
					"nextIdx期望%d，实际%d",
					test.wantNextIdx,
					nextIdx,
				)
			}
		})
	}
}
