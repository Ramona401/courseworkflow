package services

// courseware_assistant_plan_parse_test.go
//
// 本测试只验证纯内存JSON解析、结构校验、答案保护、教学方式和引用关系。
// 不连接数据库、不读取AI配置，也不会实际调用模型。

import (
	"errors"
	"strings"
	"testing"

	"tedna/internal/models"
)

const testCoursewareAssistantTeachingMode = models.CoursewareAssistantTeachingModeGuidedReasoning

// validCoursewareAssistantPlanJSON 返回一份完整合法方案。
func validCoursewareAssistantPlanJSON() string {
	return `{
  "teaching_mode": "guided_reasoning",
  "name": "三角形面积探究伙伴",
  "welcome_message": "先观察拼接过程，再告诉我你发现了什么关系。",
  "teaching_role": "通过观察、比较和逐层追问帮助学生自主发现三角形面积公式，不直接公布公式。",
  "learning_objective": "学生能够根据两个相同三角形的拼接关系解释三角形面积公式的来源。",
  "guiding_principles": [
    "先描述观察到的图形变化，再讨论数量关系",
    "每次只推进一个认知台阶"
  ],
  "question_chain": [
    {
      "id": "Q1",
      "prompt": "两个相同的三角形拼在一起后形成了什么图形？",
      "teaching_intent": "确认学生识别拼接后的整体图形。",
      "expected_signals": [
        "学生指出形成平行四边形"
      ],
      "hint_ladder": [
        "观察两组对边的位置关系",
        "想一想哪种四边形的两组对边分别平行"
      ],
      "misconception_branch_ids": [
        "M1"
      ],
      "next_step_id": "Q2",
      "completion_signal": "学生能够正确说出平行四边形"
    },
    {
      "id": "Q2",
      "prompt": "一个三角形的面积与这个平行四边形的面积有什么关系？",
      "teaching_intent": "建立一半关系并为推导公式做准备。",
      "expected_signals": [
        "学生指出一个三角形占平行四边形的一半"
      ],
      "hint_ladder": [
        "拼图中一共用了几个完全相同的三角形"
      ],
      "misconception_branch_ids": [],
      "next_step_id": "",
      "completion_signal": "学生能够解释二分之一关系"
    }
  ],
  "misconception_branches": [
    {
      "id": "M1",
      "match_signals": [
        "学生把拼成的图形判断为长方形"
      ],
      "response_strategy": "引导学生比较相邻边是否垂直，不直接给出图形名称。",
      "follow_up_question": "这个图形的四个角都一定是直角吗？",
      "return_to_step_id": "Q1"
    }
  ],
  "forbidden_behaviors": [
    "不得直接公布三角形面积公式",
    "不得声称已经看到学生完成真实拖拽"
  ],
  "completion_criteria": [
    "学生能够用拼接关系解释公式中的除以二",
    "学生能够说清底和高来自拼成图形的哪些量"
  ],
  "answer_leak_policy": {
    "direct_answer_allowed": false,
    "require_student_try": true,
    "maximum_hint_level": 3,
    "prohibited_behaviors": [
      "不得跳过学生观察直接给公式",
      "不得在第一轮提示中公布结论"
    ],
    "safe_closure_guidance": "达到提示上限后总结学生已经确认的事实，并邀请学生根据这些事实再作一次推断。"
  },
  "context_scope": {
    "version": "v1",
    "include_visible_text": true,
    "include_page_plan": true,
    "include_interaction_evidence": true,
    "include_lesson_plan_excerpt": true,
    "include_previous_page_summary": true,
    "include_next_page_summary": true,
    "max_lesson_plan_excerpt_chars": 4000
  }
}`
}

// TestCoursewareAssistantPlanParseValid 验证完整方案可以解析和规范化。
func TestCoursewareAssistantPlanParseValid(
	t *testing.T,
) {
	result, err :=
		parseCoursewareAssistantPlanAIResult(
			validCoursewareAssistantPlanJSON(),
			testCoursewareAssistantTeachingMode,
		)
	if err != nil {
		t.Fatalf(
			"合法方案解析失败: %v",
			err,
		)
	}

	if result.Title !=
		"三角形面积探究伙伴" {
		t.Fatalf(
			"名称解析错误: %s",
			result.Title,
		)
	}

	if result.GuidancePlan.Version !=
		models.CoursewareAssistantGuidancePlanCurrentVersion {
		t.Fatalf(
			"方案版本未规范化: %s",
			result.GuidancePlan.Version,
		)
	}

	if result.GuidancePlan.TeachingMode !=
		testCoursewareAssistantTeachingMode {
		t.Fatalf(
			"教学方式未按请求保留: %s",
			result.GuidancePlan.TeachingMode,
		)
	}

	if len(
		result.GuidancePlan.QuestionChain,
	) != 2 {
		t.Fatalf(
			"问题链数量错误: %d",
			len(result.GuidancePlan.QuestionChain),
		)
	}

	if result.GuidancePlan.
		AnswerLeakPolicy.
		DirectAnswerAllowed {
		t.Fatal(
			"合法方案不应允许直接答案",
		)
	}

	if !result.ContextConfig.
		IncludeVisibleText {
		t.Fatal(
			"上下文范围没有保留可见文字开关",
		)
	}
}

// TestCoursewareAssistantPlanParseMarkdownJSON 验证合法代码块可以提取。
func TestCoursewareAssistantPlanParseMarkdownJSON(
	t *testing.T,
) {
	raw :=
		"```json\n" +
			validCoursewareAssistantPlanJSON() +
			"\n```"

	result, err :=
		parseCoursewareAssistantPlanAIResult(
			raw,
			testCoursewareAssistantTeachingMode,
		)
	if err != nil {
		t.Fatalf(
			"代码块JSON解析失败: %v",
			err,
		)
	}

	if result == nil ||
		result.Title == "" {
		t.Fatal(
			"代码块JSON没有生成完整方案",
		)
	}
}

// TestCoursewareAssistantPlanRejectsDirectAnswer 验证答案泄露协议无法绕过。
func TestCoursewareAssistantPlanRejectsDirectAnswer(
	t *testing.T,
) {
	raw :=
		strings.Replace(
			validCoursewareAssistantPlanJSON(),
			`"direct_answer_allowed": false`,
			`"direct_answer_allowed": true`,
			1,
		)

	_, err :=
		parseCoursewareAssistantPlanAIResult(
			raw,
			testCoursewareAssistantTeachingMode,
		)
	if err == nil {
		t.Fatal(
			"允许直接答案的方案应当被拒绝",
		)
	}

	if !errors.Is(
		err,
		ErrCoursewareAssistantPlanInvalidOutput,
	) {
		t.Fatalf(
			"错误类型不正确: %v",
			err,
		)
	}
}

// TestCoursewareAssistantPlanRejectsBrokenReference 验证不存在的步骤引用被拒绝。
func TestCoursewareAssistantPlanRejectsBrokenReference(
	t *testing.T,
) {
	raw :=
		strings.Replace(
			validCoursewareAssistantPlanJSON(),
			`"return_to_step_id": "Q1"`,
			`"return_to_step_id": "Q99"`,
			1,
		)

	_, err :=
		parseCoursewareAssistantPlanAIResult(
			raw,
			testCoursewareAssistantTeachingMode,
		)
	if err == nil {
		t.Fatal(
			"引用不存在步骤的方案应当被拒绝",
		)
	}

	if !errors.Is(
		err,
		ErrCoursewareAssistantPlanInvalidOutput,
	) {
		t.Fatalf(
			"错误类型不正确: %v",
			err,
		)
	}
}

// TestCoursewareAssistantPlanRejectsUnknownField 验证未知扩展字段被拒绝。
func TestCoursewareAssistantPlanRejectsUnknownField(
	t *testing.T,
) {
	raw :=
		strings.Replace(
			validCoursewareAssistantPlanJSON(),
			`"name": "三角形面积探究伙伴",`,
			`"name": "三角形面积探究伙伴", "arbitrary_tool_url": "https://example.invalid",`,
			1,
		)

	_, err :=
		parseCoursewareAssistantPlanAIResult(
			raw,
			testCoursewareAssistantTeachingMode,
		)
	if err == nil {
		t.Fatal(
			"包含未知工具字段的方案应当被拒绝",
		)
	}

	if !errors.Is(
		err,
		ErrCoursewareAssistantPlanInvalidOutput,
	) {
		t.Fatalf(
			"错误类型不正确: %v",
			err,
		)
	}
}

// TestCoursewareAssistantPlanRejectsTeachingModeMismatch
// 验证模型不能把教师选择的教学方式替换为其它方式。
func TestCoursewareAssistantPlanRejectsTeachingModeMismatch(
	t *testing.T,
) {
	_, err :=
		parseCoursewareAssistantPlanAIResult(
			validCoursewareAssistantPlanJSON(),
			models.CoursewareAssistantTeachingModeExplainBack,
		)
	if err == nil {
		t.Fatal(
			"AI返回的教学方式与教师选择不一致时应当被拒绝",
		)
	}

	if !errors.Is(
		err,
		ErrCoursewareAssistantPlanInvalidOutput,
	) {
		t.Fatalf(
			"错误类型不正确: %v",
			err,
		)
	}
}
