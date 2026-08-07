package services

import "testing"

func TestLessonPlanPublishIntentAcceptsExplicitConfirmations(
	t *testing.T,
) {
	cases := []string{
		"教案我满意了，就这样定稿",
		" 教案我满意了，就这样定稿。 ",
		"完成并发布",
		"不用改了，发布",
		"确认发布教案",
		"就按这个版本发布",
		"确定这个版本",
	}

	for _, value := range cases {
		if !isLessonPlanPublishIntent(
			value,
		) {
			t.Fatalf(
				"明确发布意图未命中: %q",
				value,
			)
		}
	}
}

func TestLessonPlanPublishIntentRejectsOrdinaryDiscussion(
	t *testing.T,
) {
	cases := []string{
		"请修改发布前的检查说明",
		"这份教案定稿前还要改哪里",
		"发布以后还能继续编辑吗",
		"帮我写一版完整教案",
		"请把教后反思补充一下",
		"我想讨论定稿流程",
		"",
	}

	for _, value := range cases {
		if isLessonPlanPublishIntent(
			value,
		) {
			t.Fatalf(
				"普通讨论被误判为发布意图: %q",
				value,
			)
		}
	}
}
