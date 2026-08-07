package repository

// lesson_plan_context_capsule_turn_order_test.go
//
// 不连接数据库，只验证轮次ID解析和乱序判断。

import "testing"

func TestLessonPlanContextCapsuleTurnSequence(
	t *testing.T,
) {
	testCases := []struct {
		Name   string
		TurnID string
		Want   int64
		OK     bool
	}{
		{
			Name:   "普通聊天轮次",
			TurnID: "t15_1785582763587",
			Want:   1785582763587,
			OK:     true,
		},
		{
			Name:   "结构化阶段动作",
			TurnID: "stage_write_confirm_1785603000000",
			Want:   1785603000000,
			OK:     true,
		},
		{
			Name:   "空轮次",
			TurnID: "",
			Want:   0,
			OK:     false,
		},
		{
			Name:   "历史无数字后缀",
			TurnID: "legacy-turn",
			Want:   0,
			OK:     false,
		},
		{
			Name:   "非法数字后缀",
			TurnID: "turn_not-a-number",
			Want:   0,
			OK:     false,
		},
	}

	for _, testCase := range testCases {
		t.Run(
			testCase.Name,
			func(t *testing.T) {
				got, ok :=
					lessonPlanContextCapsuleTurnSequence(
						testCase.TurnID,
					)

				if got != testCase.Want ||
					ok != testCase.OK {
					t.Fatalf(
						"轮次序号解析异常: turn=%q got=%d ok=%v want=%d wantOK=%v",
						testCase.TurnID,
						got,
						ok,
						testCase.Want,
						testCase.OK,
					)
				}
			},
		)
	}
}

func TestLessonPlanContextCapsuleIncomingTurnIsOlder(
	t *testing.T,
) {
	testCases := []struct {
		Name     string
		Current  string
		Incoming string
		Want     bool
	}{
		{
			Name:     "旧t14晚到必须拦截",
			Current:  "t15_1785582763587",
			Incoming: "t14_1785582737626",
			Want:     true,
		},
		{
			Name:     "新t15允许写入",
			Current:  "t14_1785582737626",
			Incoming: "t15_1785582763587",
			Want:     false,
		},
		{
			Name:     "相同轮次允许确定性补充",
			Current:  "t15_1785582763587",
			Incoming: "retry_1785582763587",
			Want:     false,
		},
		{
			Name:     "结构化阶段动作较新",
			Current:  "t15_1785582763587",
			Incoming: "stage_write_confirm_1785603000000",
			Want:     false,
		},
		{
			Name:     "当前历史ID不可比较",
			Current:  "legacy-turn",
			Incoming: "t15_1785582763587",
			Want:     false,
		},
		{
			Name:     "待写入历史ID不可比较",
			Current:  "t15_1785582763587",
			Incoming: "legacy-turn",
			Want:     false,
		},
	}

	for _, testCase := range testCases {
		t.Run(
			testCase.Name,
			func(t *testing.T) {
				got :=
					lessonPlanContextCapsuleIncomingTurnIsOlder(
						testCase.Current,
						testCase.Incoming,
					)

				if got != testCase.Want {
					t.Fatalf(
						"旧轮次判断异常: current=%q incoming=%q got=%v want=%v",
						testCase.Current,
						testCase.Incoming,
						got,
						testCase.Want,
					)
				}
			},
		)
	}
}
