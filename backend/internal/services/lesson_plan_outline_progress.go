package services

// lesson_plan_outline_progress.go — 课程大纲隐藏校验期间的SSE进度心跳
//
// 设计目标：
//   1. 课程大纲闸门在最终正文通过前不允许泄漏候选首稿、自动修正稿或隐藏推理；
//   2. 隐藏链路可能持续数分钟，不能让前端因90秒没有chunk而误判“AI连接不上”；
//   3. 复用既有retry_notice事件，不新增HTTP/SSE协议，不修改数据库；
//   4. 心跳有最长持续时间，真正异常时仍会让前端看门狗最终接管。
//
// 本文件只负责“运行中提示”，不参与课程大纲判定，也不改变正式生成结果。

import (
	"context"
	"strings"
	"sync"
	"time"

	"tedna/internal/models"
)

const (
	// 每20秒发送一次轻量活动事件，明显短于前端90秒无活动看门狗。
	courseOutlineProgressInterval = 20 * time.Second

	// 最多持续4分钟。超过后停止心跳，若调用仍无结果，
	// 前端将在最后一次活动后的看门狗阈值到达时给出恢复入口。
	courseOutlineProgressMaxDuration = 4 * time.Minute
)

var courseOutlineProgressMessages = []string{
	"已读取课程大纲，正在生成本轮候选内容…",
	"正在核对课程大纲边界，暂时不会展示未校验草稿…",
	"正在检查是否有遗漏或超纲内容，请稍候…",
	"如需自动修正，系统会先完成修正再展示最终版本…",
}

// startCourseOutlineGuardProgress 启动课程大纲闸门期间的SSE活动心跳。
//
// 返回的停止函数可重复调用且并发安全；调用方必须在隐藏校验结束后立即调用。
// planID为空时返回空操作函数，保持fail-closed且不制造无归属广播。
func startCourseOutlineGuardProgress(
	ctx context.Context,
	planID string,
	turnID string,
) func() {
	planID = strings.TrimSpace(
		planID,
	)
	if planID == "" {
		return func() {}
	}

	done := make(chan struct{})
	var stopOnce sync.Once

	broadcast := func(
		content string,
	) {
		GlobalLPSSEHub.Broadcast(
			planID,
			models.LPSSEEvent{
				EventType:    models.LPSSERetryNotice,
				PlanID:       planID,
				ClientTurnID: turnID,
				Content:      content,
			},
		)
	}

	// 立即发出第一条活动提示，避免老师在首个8秒软提示前看到空白等待。
	broadcast(
		courseOutlineProgressMessages[0],
	)

	go func() {
		ticker :=
			time.NewTicker(
				courseOutlineProgressInterval,
			)
		maxTimer :=
			time.NewTimer(
				courseOutlineProgressMaxDuration,
			)

		defer ticker.Stop()
		defer maxTimer.Stop()

		messageIndex := 1

		for {
			select {
			case <-ctx.Done():
				return

			case <-done:
				return

			case <-maxTimer.C:
				broadcast(
					"课程大纲校验耗时较长，系统仍在等待模型返回；若持续无结果，页面会自动提供重试入口。",
				)
				return

			case <-ticker.C:
				// 单独计算数组索引，避免把右方括号放到函数调用换行之后。
				// Go会在以右括号结束的行尾自动插入分号；原写法因此无法解析。
				currentMessageIndex :=
					messageIndex %
						len(
							courseOutlineProgressMessages,
						)

				broadcast(
					courseOutlineProgressMessages[currentMessageIndex],
				)

				messageIndex++
			}
		}
	}()

	return func() {
		stopOnce.Do(
			func() {
				close(done)
			},
		)
	}
}
