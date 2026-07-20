package handlers

// courseware_alignment_tracked.go — 手动重算课件对齐报告的受控入口
//
// 安全顺序：
//
//	JWT claims
//	→ 作者私有控制面预检
//	→ Service再次加载正式课件
//	→ 后台任务登记
//	→ 独立Actor快照
//	→ 后台runAlignment最终授权
//	→ 写报告与调用AI
//
// 对齐报告会消耗AI并覆盖派生报告，因此只允许课件作者本人触发。
// admin和集体备课参与者不能借课件ID进入作者私有对齐任务。

import (
	"net/http"

	"tedna/internal/middleware"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// RecheckAlignmentTracked 手动触发受Tracker管理的课件对齐重算。
func (h *CoursewareIndexHandler) RecheckAlignmentTracked(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			"仅支持POST请求",
		)
		return
	}

	claims, ok := middleware.GetClaims(
		r.Context(),
	)
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	coursewareID := extractCoursewareMiddleID(
		r.URL.Path,
		"/recheck-alignment",
	)
	if coursewareID == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	if h.alignmentService == nil {
		utils.InternalError(
			w,
			"对齐校验服务未初始化",
		)
		return
	}

	// 必须先完成作者域和审核锁预检，再进入Service任务登记。
	// 无权请求不能占用courseware_alignment任务键。
	scopedActor, err :=
		authorizeCoursewareSchemeMutationForHandler(
			r.Context(),
			coursewareID,
			claims.UserID,
			claims.Role,
		)
	if err != nil {
		writeCoursewareControlError(w, err)
		return
	}

	result, err :=
		h.alignmentService.TriggerAlignmentTracked(
			r.Context(),
			coursewareID,
			scopedActor,
		)
	if err != nil {
		writeCoursewareControlError(w, err)
		return
	}

	switch result {
	case services.BackgroundStarted:
		utils.Success(w, map[string]string{
			"courseware_id": coursewareID,
			"message":       "对齐校验已重新启动，请稍后刷新查看",
		})
		return

	case services.BackgroundAlreadyRunning:
		utils.Success(w, map[string]string{
			"courseware_id": coursewareID,
			"message":       "该课件的对齐校验正在进行中，无需重复启动",
		})
		return

	case services.BackgroundRejectedDraining:
		utils.Fail(
			w,
			http.StatusServiceUnavailable,
			"系统正在升级，本次对齐校验尚未开始，请稍后重试",
		)
		return

	case services.BackgroundInvalid:
		utils.BadRequest(
			w,
			"对齐校验任务参数无效",
		)
		return

	default:
		utils.InternalError(
			w,
			"对齐校验任务启动失败",
		)
		return
	}
}
