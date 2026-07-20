package handlers

// kb_compress_tracked.go — 知识库压缩任务的受控启动入口
//
// 任务登记必须发生在CreateJob之前：
//   - draining期间直接返回503，不创建无法启动的数据库任务；
//   - 登记成功后才创建job并保存进程内原始输入；
//   - CreateJob失败立即Done；
//   - 创建成功后启动goroutine，Tracker覆盖完整RunCompressAsync生命周期。
//
// 唯一任务键使用：kb_compress:<userID>:<batchTag>。
// 同一用户同一批次不会被重复提交。

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// CreateJobTracked 创建并启动受Tracker管理的知识库压缩任务。
func (h *KBCompressHandler) CreateJobTracked(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	var req models.KBCreateJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	resourceID := claims.UserID + ":" + strings.TrimSpace(req.BatchTag)

	task, started := startTrackedBackgroundTask(
		w,
		"kb_compress",
		resourceID,
		services.BackgroundTaskCritical,
		nil,
		"该批次的知识库压缩任务正在执行，请勿重复提交",
	)
	if !started {
		return
	}

	jobID, err := h.compressService.CreateJob(
		r.Context(),
		&req,
		claims.UserID,
		"",
	)
	if err != nil {
		task.Done()
		utils.BadRequest(w, err.Error())
		return
	}

	runTrackedBackgroundTask(
		task,
		"kb_compress",
		jobID,
		800*time.Millisecond,
		func() error {
			h.compressService.RunCompressAsync(jobID)
			return nil
		},
	)

	utils.Success(w, map[string]interface{}{
		"job_id":  jobID,
		"message": "压缩任务已创建，请通过SSE监听进度",
	})
}
