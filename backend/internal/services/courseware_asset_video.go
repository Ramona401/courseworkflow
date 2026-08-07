package services

// courseware_asset_video.go — 课件视频生成服务（从courseware_asset_service.go拆分）
//
// v0.42.1 新增：AI视频生成（异步提交+状态查询+下载保存）
// 视频锚点轮新增：
//   - GenerateVideoServiceRequest 新增 SourceFrameAssetID（首帧图资产ID，两步流"先出首帧图再生视频"传入）
//   - GenerateVideo 创建视频资产后，若 SourceFrameAssetID 非空，写 metadata 溯源
//     {"source_frame_asset_id":"<首帧图asset_id>"}，建立"视频←首帧图"血缘（失败仅WARN不阻断主流程）
//
// 功能：
//   - GenerateVideo: 提交豆包Seedance视频生成任务（返回task_id）
//   - QueryVideoStatus: 查询任务状态（成功时下载保存到本地）
//   - downloadAndSaveVideo: 下载远程视频并保存到磁盘
//
// 与 courseware_asset_service.go 共享常量和CoursewareAssetService接收器

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"tedna/internal/ai"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== v0.42.1 AI视频生成（异步模式） ====================

// GenerateVideoServiceRequest AI视频生成请求参数
type GenerateVideoServiceRequest struct {
        CoursewareID       string
        PageNumber         int
        Prompt             string
        RefImageURL        string
        Actor              *CoursewareActorContext
        SourceFrameAssetID string
        OperationID        string
}

// GenerateVideoServiceResponse AI视频生成任务提交响应
type GenerateVideoServiceResponse struct {
	AssetID   string `json:"asset_id"`   // 资产记录ID（status=generating）
	TaskID    string `json:"task_id"`    // 豆包视频任务ID（用于后续轮询）
	ModelUsed string `json:"model_used"` // 使用的模型
	Message   string `json:"message"`    // 提示信息
}

// validateVideoSourceFrameAsset 校验视频首帧资产的资源边界。
func validateVideoSourceFrameAsset(
	coursewareID string,
	asset *models.CoursewareAsset,
) error {
	if asset == nil {
		return fmt.Errorf("首帧资产不存在")
	}
	if asset.CoursewareID != coursewareID {
		return fmt.Errorf("首帧资产不属于此课件")
	}
	if asset.AssetType != models.CWAssetTypeImage {
		return fmt.Errorf("首帧资产必须是图片")
	}

	return nil
}

// resolveVideoReferenceURL 解析视频任务使用的可信参考图。
//
// 存在SourceFrameAssetID时，忽略客户端同时提交的RefImageURL，
// 只使用数据库内已经完成同课件图片校验的资产地址。
func (s *CoursewareAssetService) resolveVideoReferenceURL(
	ctx context.Context,
	req *GenerateVideoServiceRequest,
) (string, error) {
	sourceFrameAssetID := strings.TrimSpace(
		req.SourceFrameAssetID,
	)

	if sourceFrameAssetID != "" {
		sourceAsset, err := repository.GetCWAssetByID(
			ctx,
			sourceFrameAssetID,
		)
		if err != nil {
			return "", fmt.Errorf(
				"首帧资产不存在: %w",
				err,
			)
		}

		if err := validateVideoSourceFrameAsset(
			req.CoursewareID,
			sourceAsset,
		); err != nil {
			return "", err
		}

		publicURL := resolveAssetPublicURL(
			sourceAsset,
		)
		if publicURL == "" {
			return "", fmt.Errorf(
				"首帧图片没有可用的公网地址",
			)
		}

		return publicURL, nil
	}

	refImageURL := strings.TrimSpace(
		req.RefImageURL,
	)
	if refImageURL == "" {
		return "", nil
	}

	if strings.HasPrefix(
		refImageURL,
		"/uploads/",
	) {
		return cwAssetPublicHost + refImageURL, nil
	}

	return refImageURL, nil
}

// GenerateVideo 提交视频生成任务（异步，返回task_id供前端轮询）
func (s *CoursewareAssetService) GenerateVideo(
        ctx context.Context,
        req *GenerateVideoServiceRequest,
) (*GenerateVideoServiceResponse, error) {
        if req == nil {
                return nil,
                        ErrCoursewareActorRequired
        }

        req.CoursewareID =
                strings.TrimSpace(
                        req.CoursewareID,
                )

        req.Prompt =
                strings.TrimSpace(
                        req.Prompt,
                )

        req.RefImageURL =
                strings.TrimSpace(
                        req.RefImageURL,
                )

        req.SourceFrameAssetID =
                strings.TrimSpace(
                        req.SourceFrameAssetID,
                )

        req.OperationID =
                strings.TrimSpace(
                        req.OperationID,
                )

        if req.CoursewareID == "" ||
                req.PageNumber <= 0 ||
                req.Prompt == "" {
                return nil,
                        ErrMediaBillingInvalidRequest
        }

        courseware, scopedActor, err :=
                (&CoursewareService{}).
                        LoadCoursewareForOwnerRuntime(
                                ctx,
                                req.CoursewareID,
                                req.Actor,
                        )

        if err != nil {
                return nil, err
        }

        page, err :=
                repository.GetCoursewarePageByNumber(
                        ctx,
                        courseware.ID,
                        req.PageNumber,
                )

        if err != nil {
                return nil,
                        fmt.Errorf(
                                "页面不存在: 课件=%s 页码=%d",
                                courseware.ID,
                                req.PageNumber,
                        )
        }

        refURL, err :=
                s.resolveVideoReferenceURL(
                        ctx,
                        req,
                )

        if err != nil {
                return nil, err
        }

        videoConfig, err :=
                ai.GetVideoConfig(
                        s.cfg.GetAESKey(),
                )

        if err != nil {
                return nil,
                        fmt.Errorf(
                                "视频生成API未配置: %w",
                                err,
                        )
        }

        schoolID, _ :=
                repository.GetSchoolIDByUserID(
                        ctx,
                        scopedActor.UserID,
                )

        identity, err :=
                newCoursewareVideoBillingIdentity(
                        scopedActor.UserID,
                        schoolID,
                        courseware.ID,
                        page.ID,
                        page.PageNumber,
                        videoConfig.Model,
                        req.OperationID,
                        req.Prompt,
                        refURL,
                        req.SourceFrameAssetID,
                )

        if err != nil {
                return nil, err
        }

        billing, err :=
                reserveCoursewareVideoBilling(
                        ctx,
                        identity,
                )

        if err != nil {
                return nil, err
        }

        if billing == nil {
                return nil,
                        ErrMediaBillingInvalidRequest
        }

        if !billing.ReservationCreated {
                return s.recoverCoursewareVideoSubmission(
                        billing,
                        identity,
                        req,
                        page,
                )
        }

        userID :=
                scopedActor.UserID

        traceContext :=
                &ai.TraceContext{
                        SceneCode:
                                "courseware_video_gen",
                        UserID:
                                &userID,
                        SchoolID:
                                identity.SchoolID,
                }

        result, err :=
                ai.SubmitVideoTask(
                        ctx,
                        videoConfig,
                        req.Prompt,
                        refURL,
                        traceContext,
                )

        if err != nil {
                if ai.IsVideoSubmitUncertain(
                        err,
                ) {
                        cwAssetLog.Error(
                                "视频提交结果不确定，保持积分预留并禁止自动重复提交",
                                "idempotency_key",
                                identity.IdempotencyKey,
                                "courseware_id",
                                identity.CoursewareID,
                                "page_number",
                                identity.PageNumber,
                                "error",
                                err,
                        )

                        return nil,
                                fmt.Errorf(
                                        "%w: 供应商提交结果尚未确认，请稍后刷新本页查看",
                                        ErrCoursewareVideoBillingInProgress,
                                )
                }

                releaseErr :=
                        releaseCoursewareVideoBilling(
                                billing,
                                "",
                                "video_provider_submit_failed",
                                map[string]interface{}{
                                        "submit_error":
                                                truncateMediaBillingReason(
                                                        err.Error(),
                                                ),
                                },
                        )

                if releaseErr != nil {
                        cwAssetLog.Error(
                                "视频提交失败后的积分释放失败",
                                "idempotency_key",
                                identity.IdempotencyKey,
                                "error",
                                releaseErr,
                        )
                }

                return nil,
                        fmt.Errorf(
                                "视频任务提交失败: %w",
                                err,
                        )
        }

        taskID :=
                strings.TrimSpace(
                        result.TaskID,
                )

        if taskID == "" {
                return nil,
                        fmt.Errorf(
                                "视频供应商未返回任务ID",
                        )
        }

        billingService :=
                NewMediaBillingService()

        taskBindErr :=
                bindCoursewareVideoExternalTask(
                        billingService,
                        identity.IdempotencyKey,
                        taskID,
                )

        if taskBindErr != nil {
                cwAssetLog.Error(
                        "视频供应商任务已创建但计费任务绑定失败",
                        "idempotency_key",
                        identity.IdempotencyKey,
                        "task_id",
                        taskID,
                        "error",
                        taskBindErr,
                )
        }

        asset, err :=
                s.createGeneratingVideoAsset(
                        req,
                        page,
                        identity,
                        taskID,
                )

        if err != nil {
                cwAssetLog.Error(
                        "视频供应商任务已创建但资产记录失败",
                        "idempotency_key",
                        identity.IdempotencyKey,
                        "task_id",
                        taskID,
                        "task_bound",
                        taskBindErr == nil,
                        "error",
                        err,
                )

                return nil,
                        fmt.Errorf(
                                "视频任务已提交，但资产记录失败: %w",
                                err,
                        )
        }

        assetBindErr :=
                bindCoursewareVideoAsset(
                        billingService,
                        identity.IdempotencyKey,
                        asset.ID,
                )

        if assetBindErr != nil {
                cwAssetLog.Error(
                        "视频资产已创建但计费资产绑定失败",
                        "idempotency_key",
                        identity.IdempotencyKey,
                        "asset_id",
                        asset.ID,
                        "error",
                        assetBindErr,
                )
        }

        if taskBindErr != nil {
                retryErr :=
                        bindCoursewareVideoExternalTask(
                                billingService,
                                identity.IdempotencyKey,
                                taskID,
                        )

                if retryErr == nil {
                        taskBindErr = nil
                } else {
                        cwAssetLog.Error(
                                "视频计费任务绑定重试仍失败",
                                "idempotency_key",
                                identity.IdempotencyKey,
                                "task_id",
                                taskID,
                                "error",
                                retryErr,
                        )
                }
        }

        cwAssetLog.Info(
                "视频任务已提交并进入积分预留状态",
                "courseware_id",
                identity.CoursewareID,
                "page_number",
                identity.PageNumber,
                "asset_id",
                asset.ID,
                "task_id",
                taskID,
                "prompt_len",
                len(req.Prompt),
                "has_ref_image",
                identity.HasReferenceImage,
                "has_source_frame",
                identity.SourceFrameAssetID != "",
                "estimated_provider_tokens",
                coursewareVideoEstimatedProviderTokens,
                "asset_bound",
                assetBindErr == nil,
                "task_bound",
                taskBindErr == nil,
        )

        return &GenerateVideoServiceResponse{
                AssetID:
                        asset.ID,
                TaskID:
                        taskID,
                ModelUsed:
                        result.ModelUsed,
                Message:
                        "视频生成任务已提交，通常需要30-120秒完成",
        }, nil
}

// ==================== v0.42.1 视频任务状态查询 ====================

// QueryVideoStatusResponse 视频任务状态查询响应
type QueryVideoStatusResponse struct {
	AssetID    string `json:"asset_id"`   // 资产记录ID
	TaskID     string `json:"task_id"`    // 豆包任务ID
	Status     string `json:"status"`     // 状态：generating/uploaded/failed
	VideoURL   string `json:"video_url"`  // 视频本地URL（成功时有值）
	Duration   int    `json:"duration"`   // 视频时长（秒）
	Resolution string `json:"resolution"` // 分辨率
	Ratio      string `json:"ratio"`      // 画面比例
	ErrorMsg   string `json:"error_msg"`  // 错误信息（失败时有值）
	Message    string `json:"message"`    // 提示信息
}

// QueryVideoStatus 查询视频生成任务状态
// 如果任务已完成，自动下载视频保存到本地并更新数据库
func (s *CoursewareAssetService) QueryVideoStatus(
        ctx context.Context,
        coursewareID string,
        assetID string,
        actor *CoursewareActorContext,
) (*QueryVideoStatusResponse, error) {
        if _, _, err :=
                (&CoursewareService{}).
                        LoadCoursewareForOwnerRuntime(
                                ctx,
                                coursewareID,
                                actor,
                        ); err != nil {
                return nil, err
        }

        asset, err :=
                repository.GetCWAssetByID(
                        ctx,
                        assetID,
                )

        if err != nil {
                return nil,
                        fmt.Errorf(
                                "视频资产不存在: %w",
                                err,
                        )
        }

        if asset.CoursewareID !=
                coursewareID {
                return nil,
                        fmt.Errorf(
                                "视频资产不属于路径指定课件",
                        )
        }

        if asset.AssetType !=
                models.CWAssetTypeVideo {
                return nil,
                        fmt.Errorf(
                                "资产不是视频资产",
                        )
        }

        billing, err :=
                loadCoursewareVideoBilling(
                        ctx,
                        asset,
                )

        if err != nil {
                return nil, err
        }

        taskID :=
                resolveCoursewareVideoTaskID(
                        billing,
                        asset,
                )

        if asset.Status ==
                models.CWAssetStatusUploaded ||
                asset.Status ==
                        models.CWAssetStatusConfirmed {
                if billing != nil &&
                        billing.Status ==
                                models.MediaBillingStatusReserved {
                        settleErr :=
                                settleCoursewareVideoBillingFromAsset(
                                        billing,
                                        asset,
                                )

                        if settleErr != nil &&
                                taskID != "" {
                                videoConfig, configErr :=
                                        ai.GetVideoConfig(
                                                s.cfg.GetAESKey(),
                                        )

                                if configErr == nil {
                                        queryResult, queryErr :=
                                                ai.QueryVideoTask(
                                                        ctx,
                                                        videoConfig,
                                                        taskID,
                                                )

                                        if queryErr == nil &&
                                                queryResult.Status ==
                                                        "succeeded" {
                                                _ =
                                                        updateCoursewareVideoAssetResultMetadata(
                                                                asset,
                                                                queryResult,
                                                        )

                                                retryErr :=
                                                        settleCoursewareVideoBilling(
                                                                billing,
                                                                asset,
                                                                queryResult,
                                                        )

                                                if retryErr != nil {
                                                        cwAssetLog.Error(
                                                                "已完成视频补结算失败",
                                                                "asset_id",
                                                                asset.ID,
                                                                "task_id",
                                                                taskID,
                                                                "error",
                                                                retryErr,
                                                        )
                                                }
                                        }
                                }
                        }
                }

                metadata :=
                        coursewareVideoMetadataMap(
                                asset.Metadata,
                        )

                return &QueryVideoStatusResponse{
                        AssetID:
                                asset.ID,
                        TaskID:
                                taskID,
                        Status:
                                "uploaded",
                        VideoURL:
                                asset.OssURL,
                        Duration:
                                coursewareVideoMetadataInt(
                                        metadata,
                                        "video_duration",
                                ),
                        Resolution:
                                coursewareVideoMetadataText(
                                        metadata,
                                        "video_resolution",
                                ),
                        Ratio:
                                coursewareVideoMetadataText(
                                        metadata,
                                        "video_ratio",
                                ),
                        Message:
                                "视频已生成完成",
                }, nil
        }

        if asset.Status !=
                models.CWAssetStatusGenerating {
                return &QueryVideoStatusResponse{
                        AssetID:
                                asset.ID,
                        TaskID:
                                taskID,
                        Status:
                                "failed",
                        ErrorMsg:
                                "视频资产状态异常: " +
                                        asset.Status,
                        Message:
                                "视频生成出现问题",
                }, nil
        }

        if billing != nil &&
                (billing.Status ==
                        models.MediaBillingStatusFailed ||
                        billing.Status ==
                                models.MediaBillingStatusCancelled) {
                return &QueryVideoStatusResponse{
                        AssetID:
                                asset.ID,
                        TaskID:
                                taskID,
                        Status:
                                "failed",
                        ErrorMsg:
                                billing.FailureReason,
                        Message:
                                "视频生成任务已经失败或取消",
                }, nil
        }

        if taskID == "" {
                return nil,
                        fmt.Errorf(
                                "视频任务ID为空",
                        )
        }

        videoConfig, err :=
                ai.GetVideoConfig(
                        s.cfg.GetAESKey(),
                )

        if err != nil {
                return nil,
                        fmt.Errorf(
                                "视频生成API配置加载失败: %w",
                                err,
                        )
        }

        queryResult, err :=
                ai.QueryVideoTask(
                        ctx,
                        videoConfig,
                        taskID,
                )

        if err != nil {
                return nil,
                        fmt.Errorf(
                                "查询视频任务状态失败: %w",
                                err,
                        )
        }

        switch queryResult.Status {
        case "running":
                return &QueryVideoStatusResponse{
                        AssetID:
                                asset.ID,
                        TaskID:
                                taskID,
                        Status:
                                "generating",
                        Message:
                                "视频正在生成中，请稍后重试查询",
                }, nil

        case "succeeded":
                if strings.TrimSpace(
                        queryResult.VideoURL,
                ) == "" {
                        return nil,
                                fmt.Errorf(
                                        "视频生成成功但未返回视频URL",
                                )
                }

                if queryResult.TotalTokens <= 0 {
                        return nil,
                                fmt.Errorf(
                                        "视频生成成功但未返回有效token用量",
                                )
                }

                if _, _, authErr :=
                        (&CoursewareService{}).
                                LoadCoursewareForOwnerRuntime(
                                        ctx,
                                        coursewareID,
                                        actor,
                                ); authErr != nil {
                        return nil, authErr
                }

                metadataErr :=
                        updateCoursewareVideoAssetResultMetadata(
                                asset,
                                queryResult,
                        )

                if metadataErr != nil {
                        cwAssetLog.Warn(
                                "保存视频供应商结果元数据失败",
                                "asset_id",
                                asset.ID,
                                "task_id",
                                taskID,
                                "error",
                                metadataErr,
                        )
                }

                if billing != nil {
                        settleErr :=
                                settleCoursewareVideoBilling(
                                        billing,
                                        asset,
                                        queryResult,
                                )

                        if settleErr != nil {
                                cwAssetLog.Error(
                                        "视频供应商成功后的积分结算失败，预留保持待补偿",
                                        "asset_id",
                                        asset.ID,
                                        "task_id",
                                        taskID,
                                        "total_tokens",
                                        queryResult.TotalTokens,
                                        "error",
                                        settleErr,
                                )
                        }
                }

                localURL, fileSize, downloadErr :=
                        s.downloadAndSaveVideo(
                                ctx,
                                coursewareID,
                                taskID,
                                queryResult.VideoURL,
                        )

                if downloadErr != nil {
                        cwAssetLog.Error(
                                "视频已经成功并结算，但下载保存失败",
                                "asset_id",
                                asset.ID,
                                "task_id",
                                taskID,
                                "error",
                                downloadErr,
                        )

                        return nil,
                                fmt.Errorf(
                                        "下载视频文件失败: %w",
                                        downloadErr,
                                )
                }

                updateCtx, cancel :=
                        context.WithTimeout(
                                context.Background(),
                                coursewareVideoBillingDBTimeout,
                        )

                updateErr :=
                        repository.UpdateCWAssetOSSURL(
                                updateCtx,
                                asset.ID,
                                localURL,
                                fileSize,
                                "video/mp4",
                        )

                cancel()

                if updateErr != nil {
                        return nil,
                                fmt.Errorf(
                                        "视频已经生成，但资产状态保存失败: %w",
                                        updateErr,
                                )
                }

                cwAssetLog.Info(
                        "视频生成完成、积分结算并保存到本地",
                        "asset_id",
                        asset.ID,
                        "task_id",
                        taskID,
                        "duration",
                        queryResult.Duration,
                        "resolution",
                        queryResult.Resolution,
                        "total_tokens",
                        queryResult.TotalTokens,
                        "file_size",
                        fileSize,
                        "local_url",
                        localURL,
                )

                return &QueryVideoStatusResponse{
                        AssetID:
                                asset.ID,
                        TaskID:
                                taskID,
                        Status:
                                "uploaded",
                        VideoURL:
                                localURL,
                        Duration:
                                queryResult.Duration,
                        Resolution:
                                queryResult.Resolution,
                        Ratio:
                                queryResult.Ratio,
                        Message:
                                fmt.Sprintf(
                                        "视频生成完成！时长%d秒，分辨率%s",
                                        queryResult.Duration,
                                        queryResult.Resolution,
                                ),
                }, nil

        case "failed":
                if billing != nil {
                        releaseErr :=
                                releaseCoursewareVideoBilling(
                                        billing,
                                        taskID,
                                        "video_provider_task_failed",
                                        map[string]interface{}{
                                                "provider_error":
                                                        truncateMediaBillingReason(
                                                                queryResult.ErrorMsg,
                                                        ),
                                        },
                                )

                        if releaseErr != nil {
                                return nil,
                                        fmt.Errorf(
                                                "视频任务失败，但积分预留释放尚未完成: %w",
                                                releaseErr,
                                        )
                        }
                }

                statusCtx, cancel :=
                        context.WithTimeout(
                                context.Background(),
                                coursewareVideoBillingDBTimeout,
                        )

                statusErr :=
                        repository.UpdateCWAssetStatus(
                                statusCtx,
                                asset.ID,
                                models.CWAssetStatusPending,
                        )

                cancel()

                if statusErr != nil {
                        cwAssetLog.Warn(
                                "视频失败状态写入资产失败",
                                "asset_id",
                                asset.ID,
                                "error",
                                statusErr,
                        )
                }

                return &QueryVideoStatusResponse{
                        AssetID:
                                asset.ID,
                        TaskID:
                                taskID,
                        Status:
                                "failed",
                        ErrorMsg:
                                queryResult.ErrorMsg,
                        Message:
                                "视频生成失败: " +
                                        queryResult.ErrorMsg,
                }, nil

        default:
                return &QueryVideoStatusResponse{
                        AssetID:
                                asset.ID,
                        TaskID:
                                taskID,
                        Status:
                                "generating",
                        Message:
                                "视频状态: " +
                                        queryResult.Status,
                }, nil
        }
}

// downloadAndSaveVideo 下载远程视频并保存到本地磁盘
// 返回 (本地URL, 文件大小bytes, error)
func (s *CoursewareAssetService) downloadAndSaveVideo(ctx context.Context, coursewareID string, taskID string, videoURL string) (string, int64, error) {
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Get(videoURL)
	if err != nil {
		return "", 0, fmt.Errorf("下载视频失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("下载视频HTTP错误: %d", resp.StatusCode)
	}

	storedName := fmt.Sprintf("%d_video_%s.mp4", time.Now().UnixMilli(), taskID)

	assetDir := filepath.Join(CWAssetUploadDir, coursewareID, "videos")
	if err := os.MkdirAll(assetDir, 0755); err != nil {
		return "", 0, fmt.Errorf("创建视频目录失败: %w", err)
	}

	fullPath := filepath.Join(assetDir, storedName)
	dst, err := os.Create(fullPath)
	if err != nil {
		return "", 0, fmt.Errorf("创建视频文件失败: %w", err)
	}
	defer dst.Close()

	written, err := io.Copy(dst, resp.Body)
	if err != nil {
		_ = os.Remove(fullPath)
		return "", 0, fmt.Errorf("写入视频文件失败: %w", err)
	}

	relativePath := filepath.Join(coursewareID, "videos", storedName)
	localURL := CWAssetURLPrefix + relativePath

	return localURL, written, nil
}
