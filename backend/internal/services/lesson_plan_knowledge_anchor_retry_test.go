package services

// lesson_plan_knowledge_anchor_retry_test.go — 课程锚点受控重试的确定性边界测试
//
// 这些测试不调用真实AI、不读取数据库，只验证：
//   - 只有显式标记的AI协议错误会重试一次；
//   - 第二次提示只携带失败类别，不泄露首轮错误中的原文片段；
//   - 第二次仍失败时继续fail-closed，并保留既有errors.Is语义；
//   - Token与模型统计覆盖首轮失败和二次调用；
//   - 非重试型确定性错误只执行一次。

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestLessonPlanKnowledgeAnchorControlledRetrySucceedsOnSecondAttempt(t *testing.T) {
	attemptCount := 0
	secondPrompt := ""

	parsed, modelUsed, tokensUsed, err := runLessonPlanKnowledgeAnchorControlledRetry(
		"plan-test",
		func(systemPrompt string) (*lessonPlanKnowledgeAnchorAIResult, string, int, error) {
			attemptCount++
			if attemptCount == 1 {
				return nil, "model-a", 10, newLessonPlanKnowledgeAnchorRetryableError(
					lessonPlanKnowledgeAnchorRetryEvidenceMismatch,
					fmt.Errorf(
						"%w: 来源证据无法在教学分析对话中找到：敏感原文片段",
						ErrLessonPlanKnowledgeAnchorsIncomplete,
					),
				)
			}

			secondPrompt = systemPrompt
			return &lessonPlanKnowledgeAnchorAIResult{}, "model-b", 20, nil
		},
	)
	if err != nil {
		t.Fatalf("受控重试第二次成功时不应返回错误: %v", err)
	}
	if parsed == nil {
		t.Fatal("受控重试第二次成功时应返回解析结果")
	}
	if attemptCount != 2 {
		t.Fatalf("期望执行2次，实际执行%d次", attemptCount)
	}
	if modelUsed != "model-a;model-b" {
		t.Fatalf("模型统计不正确: %q", modelUsed)
	}
	if tokensUsed != 30 {
		t.Fatalf("Token累计不正确: %d", tokensUsed)
	}
	if !strings.Contains(secondPrompt, "上一轮证据字段没有全部使用教学分析对话中的真实原文短句") {
		t.Fatalf("第二次提示未携带受控失败类别: %q", secondPrompt)
	}
	if strings.Contains(secondPrompt, "敏感原文片段") {
		t.Fatal("第二次提示不得回喂首轮错误中的原文片段")
	}
}

func TestLessonPlanKnowledgeAnchorControlledRetryFailsClosedAfterSecondFailure(t *testing.T) {
	attemptCount := 0

	_, _, _, err := runLessonPlanKnowledgeAnchorControlledRetry(
		"plan-test",
		func(_ string) (*lessonPlanKnowledgeAnchorAIResult, string, int, error) {
			attemptCount++
			if attemptCount == 1 {
				return nil, "model-a", 11, newLessonPlanKnowledgeAnchorRetryableError(
					lessonPlanKnowledgeAnchorRetryIncompleteCore,
					fmt.Errorf(
						"%w: teaching_scope",
						ErrLessonPlanKnowledgeAnchorsIncomplete,
					),
				)
			}

			return nil, "model-a", 12, newLessonPlanKnowledgeAnchorRetryableError(
				lessonPlanKnowledgeAnchorRetryEvidenceMismatch,
				fmt.Errorf(
					"%w: 来源证据无法在教学分析对话中找到：第二轮错误证据",
					ErrLessonPlanKnowledgeAnchorsIncomplete,
				),
			)
		},
	)
	if err == nil {
		t.Fatal("第二次仍失败时必须fail-closed")
	}
	if attemptCount != 2 {
		t.Fatalf("期望最多执行2次，实际执行%d次", attemptCount)
	}
	if !errors.Is(err, ErrLessonPlanKnowledgeAnchorsIncomplete) {
		t.Fatalf("应保留既有课程锚点不完整错误语义: %v", err)
	}
}

func TestLessonPlanKnowledgeAnchorControlledRetryDoesNotRetryDeterministicError(t *testing.T) {
	attemptCount := 0
	deterministicErr := fmt.Errorf(
		"%w: 教师尚未在教学分析阶段提供实际确认信息",
		ErrLessonPlanKnowledgeAnchorsIncomplete,
	)

	_, _, _, err := runLessonPlanKnowledgeAnchorControlledRetry(
		"plan-test",
		func(_ string) (*lessonPlanKnowledgeAnchorAIResult, string, int, error) {
			attemptCount++
			return nil, "", 0, deterministicErr
		},
	)
	if !errors.Is(err, ErrLessonPlanKnowledgeAnchorsIncomplete) {
		t.Fatalf("确定性错误应原样返回: %v", err)
	}
	if attemptCount != 1 {
		t.Fatalf("确定性错误不得触发二次调用，实际执行%d次", attemptCount)
	}
}

func TestLessonPlanKnowledgeAnchorRetryPromptUsesOnlyControlledReason(t *testing.T) {
	prompt := buildLessonPlanKnowledgeAnchorRetrySystemPrompt(
		lessonPlanKnowledgeAnchorRetryIncompleteCore,
	)

	if !strings.Contains(prompt, "上一轮没有同时满足ready与全部课程锚点核心字段要求") {
		t.Fatalf("重试提示缺少受控类别说明: %q", prompt)
	}
	if !strings.Contains(prompt, "不得为了通过校验而补猜缺失事实") {
		t.Fatalf("重试提示必须继续保持fail-closed约束: %q", prompt)
	}
	if !strings.Contains(prompt, "仍必须返回ready=false") {
		t.Fatalf("真实缺失时必须允许继续返回ready=false: %q", prompt)
	}
}

func TestLessonPlanKnowledgeAnchorParseInvalidJSONIsRetryable(t *testing.T) {
	_, err := parseAndValidateLessonPlanKnowledgeAnchorAIContent(
		"这不是JSON",
		"教师：请按已经确认的教学分析执行。",
	)
	if err == nil {
		t.Fatal("非法JSON必须返回错误")
	}

	reason, retryable := lessonPlanKnowledgeAnchorRetryReasonOf(err)
	if !retryable {
		t.Fatalf("非法JSON应属于可受控重试的AI协议错误: %v", err)
	}
	if reason != lessonPlanKnowledgeAnchorRetryInvalidJSON {
		t.Fatalf("非法JSON失败类别不正确: %q", reason)
	}
	if !errors.Is(err, ErrLessonPlanKnowledgeAnchorExtractionUnavailable) {
		t.Fatalf("非法JSON仍应保留原有提取不可用错误语义: %v", err)
	}
}

func TestLessonPlanKnowledgeAnchorParseIncompleteCoreIsRetryable(t *testing.T) {
	raw := `{
  "ready": false,
  "missing_fields": ["teaching_scope"],
  "ambiguity_notes": [],
  "anchors": {
    "lesson_object": "集合",
    "teaching_scope": "",
    "source_evidence": [],
    "teaching_objectives": [],
    "knowledge_points": [],
    "learning_depth": "",
    "excluded_content": [],
    "teacher_confirmed": false
  }
}`

	_, err := parseAndValidateLessonPlanKnowledgeAnchorAIContent(
		raw,
		"教师：本节课学习集合。",
	)
	if err == nil {
		t.Fatal("核心字段不完整必须返回错误")
	}

	reason, retryable := lessonPlanKnowledgeAnchorRetryReasonOf(err)
	if !retryable {
		t.Fatalf("核心字段不完整应允许一次受控重试: %v", err)
	}
	if reason != lessonPlanKnowledgeAnchorRetryIncompleteCore {
		t.Fatalf("核心字段失败类别不正确: %q", reason)
	}
	if !errors.Is(err, ErrLessonPlanKnowledgeAnchorsIncomplete) {
		t.Fatalf("核心字段不完整仍应保留原有错误语义: %v", err)
	}
}
