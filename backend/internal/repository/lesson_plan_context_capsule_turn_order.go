package repository

// lesson_plan_context_capsule_turn_order.go — 胶囊旁路轮次顺序保护
//
// 背景：
//   - 主回复完成后，胶囊在旁路异步更新；
//   - 相邻两轮可能并发执行；
//   - 较新的教师轮次可能先完成，较旧轮次反而后完成；
//   - 若只按数据库到达顺序写入，旧轮次会生成更高版本并覆盖新共识。
//
// 当前轮次ID约定：
//   - 普通对话：t15_1785582763587；
//   - 结构化阶段动作：stage_write_confirm_1785603000000；
//   - 最后一个下划线后的正整数作为可比较时间序号。
//
// 兼容规则：
//   - 新旧双方都有合法序号时才进行拦截；
//   - 历史无序号ID继续允许写入，避免破坏存量兼容；
//   - 相同序号允许写入，兼容同一轮内确定性补充。

import (
	"strconv"
	"strings"
)

// lessonPlanContextCapsuleTurnSequence 解析轮次ID末尾的时间序号。
func lessonPlanContextCapsuleTurnSequence(
	turnID string,
) (
	int64,
	bool,
) {
	turnID =
		strings.TrimSpace(
			turnID,
		)

	if turnID == "" {
		return 0, false
	}

	separator :=
		strings.LastIndex(
			turnID,
			"_",
		)

	if separator < 0 ||
		separator+1 >= len(turnID) {
		return 0, false
	}

	sequence, err :=
		strconv.ParseInt(
			turnID[separator+1:],
			10,
			64,
		)

	if err != nil ||
		sequence <= 0 {
		return 0, false
	}

	return sequence, true
}

// lessonPlanContextCapsuleIncomingTurnIsOlder 判断待写入轮次是否更旧。
func lessonPlanContextCapsuleIncomingTurnIsOlder(
	currentTurnID string,
	incomingTurnID string,
) bool {
	currentSequence, currentOK :=
		lessonPlanContextCapsuleTurnSequence(
			currentTurnID,
		)

	incomingSequence, incomingOK :=
		lessonPlanContextCapsuleTurnSequence(
			incomingTurnID,
		)

	if !currentOK ||
		!incomingOK {
		return false
	}

	return incomingSequence <
		currentSequence
}
