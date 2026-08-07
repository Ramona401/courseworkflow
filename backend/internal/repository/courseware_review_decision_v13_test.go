package repository

// courseware_review_decision_v13_test.go
//
// V1.3正式复审问题选择规则检查。
//
// 本文件不连接数据库，只验证：
//
//   - 通过审核必须解决本轮全部旧问题；
//   - 继续退回可以只解决一部分；
//   - 任何决定都不能提交不属于本轮的旧问题ID；
//   - 空列表和重复ID按稳定规则处理。

import (
	"errors"
	"testing"

	"tedna/internal/models"
)

func TestValidateCWReviewCarryoverResolution(
	t *testing.T,
) {
	tests := []struct {
		name         string
		decision     string
		carryoverIDs []string
		resolvedIDs  []string
		wantError    bool
	}{
		{
			name:         "首次审核没有旧问题可以通过",
			decision:     models.ReviewDecisionApproved,
			carryoverIDs: []string{},
			resolvedIDs:  []string{},
			wantError:    false,
		},
		{
			name:     "通过审核必须解决全部旧问题",
			decision: models.ReviewDecisionApproved,
			carryoverIDs: []string{
				"item-1",
				"item-2",
			},
			resolvedIDs: []string{
				"item-1",
				"item-2",
			},
			wantError: false,
		},
		{
			name:     "通过审核遗漏一条旧问题",
			decision: models.ReviewDecisionApproved,
			carryoverIDs: []string{
				"item-1",
				"item-2",
			},
			resolvedIDs: []string{
				"item-1",
			},
			wantError: true,
		},
		{
			name:     "继续退回允许确认部分问题解决",
			decision: models.ReviewDecisionRevision,
			carryoverIDs: []string{
				"item-1",
				"item-2",
			},
			resolvedIDs: []string{
				"item-1",
			},
			wantError: false,
		},
		{
			name:     "继续退回允许暂不确认旧问题解决",
			decision: models.ReviewDecisionRevision,
			carryoverIDs: []string{
				"item-1",
			},
			resolvedIDs: []string{},
			wantError:   false,
		},
		{
			name:     "不能确认不属于本轮的问题",
			decision: models.ReviewDecisionRevision,
			carryoverIDs: []string{
				"item-1",
			},
			resolvedIDs: []string{
				"item-2",
			},
			wantError: true,
		},
		{
			name:     "重复选择按一条处理",
			decision: models.ReviewDecisionApproved,
			carryoverIDs: []string{
				"item-1",
			},
			resolvedIDs: []string{
				"item-1",
				"item-1",
			},
			wantError: false,
		},
	}

	for _, test := range tests {
		t.Run(
			test.name,
			func(t *testing.T) {
				err :=
					validateCWReviewCarryoverResolution(
						test.decision,
						test.carryoverIDs,
						test.resolvedIDs,
					)

				hasExpectedError :=
					errors.Is(
						err,
						ErrCWReviewDecisionCarryoverInvalid,
					)

				if test.wantError &&
					!hasExpectedError {
					t.Fatalf(
						"期望复审问题选择失败，实际错误：%v",
						err,
					)
				}

				if !test.wantError &&
					err != nil {
					t.Fatalf(
						"期望复审问题选择通过，实际错误：%v",
						err,
					)
				}
			},
		)
	}
}
