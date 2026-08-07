package services

import (
	"strings"
	"testing"

	"tedna/internal/models"
)

func TestCanStreamLessonPlanWordRevision(t *testing.T) {
	base := &lessonPlanTurnContextPlan{
		FormalArtifact:          true,
		BlockingEvidenceHarness: true,
	}

	tests := []struct {
		name       string
		stage      string
		plan       *lessonPlanTurnContextPlan
		activeWord bool
		want       bool
	}{
		{
			name:       "active synchronized Word revise streams",
			stage:      "revise",
			plan:       base,
			activeWord: true,
			want:       true,
		},
		{
			name:       "write stage keeps existing execution",
			stage:      "write",
			plan:       base,
			activeWord: true,
			want:       false,
		},
		{
			name:       "non formal revise keeps existing execution",
			stage:      "revise",
			plan:       &lessonPlanTurnContextPlan{},
			activeWord: true,
			want:       false,
		},
		{
			name:       "ordinary lesson plan does not use Word stream policy",
			stage:      "revise",
			plan:       base,
			activeWord: false,
			want:       false,
		},
		{
			name:  "textbook retains blocking Harness",
			stage: "revise",
			plan: &lessonPlanTurnContextPlan{
				FormalArtifact: true,
				UseTextbook:    true,
			},
			activeWord: true,
			want:       false,
		},
		{
			name:  "attachment retains blocking Harness",
			stage: "revise",
			plan: &lessonPlanTurnContextPlan{
				FormalArtifact: true,
				UseRefMaterial: true,
			},
			activeWord: true,
			want:       false,
		},
		{
			name:  "unit plan retains blocking Harness",
			stage: "revise",
			plan: &lessonPlanTurnContextPlan{
				FormalArtifact: true,
				UseUnitPlan:    true,
			},
			activeWord: true,
			want:       false,
		},
		{
			name:  "raw outline retains blocking Harness",
			stage: "revise",
			plan: &lessonPlanTurnContextPlan{
				FormalArtifact:      true,
				UseRawCourseOutline: true,
			},
			activeWord: true,
			want:       false,
		},
		{
			name:  "class profile retains blocking Harness",
			stage: "revise",
			plan: &lessonPlanTurnContextPlan{
				FormalArtifact:  true,
				UseClassProfile: true,
			},
			activeWord: true,
			want:       false,
		},
		{
			name:  "confirmed knowledge context may remain in prompt",
			stage: "revise",
			plan: &lessonPlanTurnContextPlan{
				FormalArtifact:      true,
				UseKnowledgeLineage: true,
				UseContextCapsule:   true,
			},
			activeWord: true,
			want:       true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got :=
				canStreamLessonPlanWordRevision(
					test.stage,
					test.plan,
					test.activeWord,
				)
			if got != test.want {
				t.Fatalf(
					"canStreamLessonPlanWordRevision()=%v, want %v",
					got,
					test.want,
				)
			}
		})
	}
}

func TestRevisionMutationIntentBecomesFormalArtifact(t *testing.T) {
	lessonPlan := &models.LessonPlan{
		CurrentStage: "revise",
	}

	for _, message := range []string{
		"只改原有段落文字，把前面的问题调整下",
		"按照你的补充建议，补充到上面的教案正文里面，另外把课后反思也补充一下",
	} {
		request := &models.LessonPlanChatRequest{
			Message: message,
		}
		plan :=
			buildLessonPlanTurnContextPlan(
				lessonPlan,
				request,
			)
		if !plan.FormalArtifact {
			t.Fatalf(
				"明确正文修改请求未进入正式产物链: %s",
				message,
			)
		}
		if !plan.UsePriorOutputs {
			t.Fatalf(
				"正式修订请求应承接评审和前序正式产出: %s",
				message,
			)
		}
	}

	discussionPlan :=
		buildLessonPlanTurnContextPlan(
			lessonPlan,
			&models.LessonPlanChatRequest{
				Message: "这个修改建议是什么意思？",
			},
		)
	if discussionPlan.FormalArtifact {
		t.Fatal("仅询问修改建议含义不应自动生成整份正式修订稿")
	}
}

func TestExtractLessonPlanWordRevisionCandidate(t *testing.T) {
	baseline := `表格1 · 第1行
第1列：学 科
表格1 · 第16行
第1列：【小结】原有作业。
板 书 设 计
[图片：image9.wmf]
教 后 反 思
课前预习阶段：
课堂教学阶段：
课后提升阶段：`

	raw := `好的，修改清单如下：
1. 调整作业。
2. 补充反思。

表格1 · 第1行
第1列：学 科
表格1 · 第16行
第1列：【小结】原有作业。课后用橡皮泥制作模型。
板 书 设 计
[图片：image9.wmf]
教 后 反 思
课前预习阶段：
关注学生对电子层的已有认识。
课堂教学阶段：
记录互评和微观模拟的即时反馈。
课后提升阶段：
跟踪模型展评对三重表征的促进作用。

` + "```teacher_suggestion\n额外创新建议，不进入正文。\n```"

	candidate,
		suggestion :=
		extractLessonPlanWordRevisionCandidate(
			raw,
			baseline,
		)

	if !strings.HasPrefix(
		candidate,
		"表格1 · 第1行",
	) {
		t.Fatalf(
			"候选稿没有从原Word结构锚点开始: %q",
			candidate,
		)
	}
	if strings.Contains(
		candidate,
		"好的，修改清单",
	) {
		t.Fatal("候选稿不应包含模板前说明")
	}
	if strings.Contains(
		candidate,
		"额外创新建议",
	) {
		t.Fatal("teacher_suggestion不应进入正式候选")
	}
	if suggestion !=
		"额外创新建议，不进入正文。" {
		t.Fatalf(
			"建议块提取异常: %q",
			suggestion,
		)
	}
	for _, expected := range []string{
		"课前预习阶段：关注学生对电子层的已有认识。",
		"课堂教学阶段：记录互评和微观模拟的即时反馈。",
		"课后提升阶段：跟踪模型展评对三重表征的促进作用。",
	} {
		if !strings.Contains(
			candidate,
			expected,
		) {
			t.Fatalf(
				"空槽内容没有合并回原段落: %s",
				expected,
			)
		}
	}
}

func TestWordRevisionImageTokensRemainExact(t *testing.T) {
	current :=
		"[图片：image1.png]\n\n正文\n\n[图片：image9.wmf]"

	tests := []struct {
		name      string
		candidate string
		want      bool
	}{
		{
			name:      "same image tokens",
			candidate: "[图片：image1.png]\n\n修改正文\n\n[图片：image9.wmf]",
			want:      true,
		},
		{
			name:      "missing image token",
			candidate: "[图片：image1.png]\n\n修改正文",
			want:      false,
		},
		{
			name:      "reordered image tokens",
			candidate: "[图片：image9.wmf]\n\n修改正文\n\n[图片：image1.png]",
			want:      false,
		},
	}

	currentTokens :=
		lessonPlanWordImageMarkdownPattern.
			FindAllString(current, -1)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidateTokens :=
				lessonPlanWordImageMarkdownPattern.
					FindAllString(
						test.candidate,
						-1,
					)
			got :=
				equalLessonPlanWordStrings(
					currentTokens,
					candidateTokens,
				)
			if got != test.want {
				t.Fatalf(
					"图片标记保持结果=%v, want %v",
					got,
					test.want,
				)
			}
		})
	}
}
