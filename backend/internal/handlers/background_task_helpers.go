package handlers

// background_task_helpers.go — Handler层后台任务统一登记与执行辅助
//
// 适用场景：
//   HTTP接口完成参数和权限校验后，需要立即返回，并在goroutine中继续执行AI长任务。
//
// 核心要求：
//   1. 必须先向GlobalBackgroundTasks登记，再启动goroutine；
//   2. 服务进入draining后拒绝新的外部任务；
//   3. 唯一任务键已存在时拒绝重复启动；
//   4. 后台任务统一通过BackgroundTask.Run执行，自动Done并恢复panic；
//   5. 已经登记成功的任务，即使随后进入draining，也属于已接受任务，应继续完成。
//
// 错误口径：
//   - 503：服务正在升级，新任务尚未启动；
//   - 409：同一资源已有同类任务运行；
//   - 500：任务登记参数或系统状态异常。

import (
	"net/http"
	"time"

	"tedna/internal/logger"
	"tedna/internal/services"
	"tedna/internal/utils"
)

var trackedHandlerLog = logger.WithModule("tracked_handlers")

// startTrackedBackgroundTask 登记一个用户发起的后台任务。
//
// 返回值：
//   - task：登记成功后的任务句柄；
//   - true：调用方可以启动goroutine；
//   - false：已向客户端写出错误响应，调用方应立即return。
func startTrackedBackgroundTask(
	w http.ResponseWriter,
	taskType string,
	resourceID string,
	class services.BackgroundTaskClass,
	onDrain func(),
	alreadyRunningMessage string,
) (*services.BackgroundTask, bool) {
	task, result := services.GlobalBackgroundTasks.TryStartExternal(
		taskType,
		resourceID,
		class,
		onDrain,
	)

	switch result {
	case services.BackgroundStarted:
		return task, true

	case services.BackgroundRejectedDraining:
		utils.Fail(
			w,
			http.StatusServiceUnavailable,
			"系统正在升级，本次任务尚未开始，请稍后重试",
		)
		return nil, false

	case services.BackgroundAlreadyRunning:
		message := alreadyRunningMessage
		if message == "" {
			message = "相同任务正在执行，请勿重复提交"
		}
		utils.Fail(w, http.StatusConflict, message)
		return nil, false

	default:
		utils.Fail(
			w,
			http.StatusInternalServerError,
			"后台任务登记失败，请稍后重试",
		)
		return nil, false
	}
}

// runTrackedBackgroundTask 启动已登记任务。
//
// delay用于给前端建立SSE连接；delay结束后任务继续执行。
// 已经登记成功的任务属于服务已接受工作，即使期间进入draining，也应完成或由onDrain钩子收缩。
func runTrackedBackgroundTask(
	task *services.BackgroundTask,
	taskType string,
	resourceID string,
	delay time.Duration,
	work func() error,
) {
	go func() {
		err := task.Run(func() error {
			if delay > 0 {
				time.Sleep(delay)
			}
			return work()
		})

		if err == nil {
			return
		}

		trackedHandlerLog.Warn(
			"后台任务执行失败",
			"task_type", taskType,
			"resource_id", resourceID,
			"error", err,
		)
	}()
}
