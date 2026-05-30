package services

// courseware_asset_video.go — 课件视频生成服务（从courseware_asset_service.go拆分）
//
// v0.42.1 新增：AI视频生成（异步提交+状态查询+下载保存）
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
CoursewareID string // 课件ID
PageNumber   int    // 页码
Prompt       string // 视频描述提示词
RefImageURL  string // 参考图URL（图生视频模式，可选，须公网可访问）
UserID       string // 操作者ID
}

// GenerateVideoServiceResponse AI视频生成任务提交响应
type GenerateVideoServiceResponse struct {
AssetID   string `json:"asset_id"`   // 资产记录ID（status=generating）
TaskID    string `json:"task_id"`    // 豆包视频任务ID（用于后续轮询）
ModelUsed string `json:"model_used"` // 使用的模型
Message   string `json:"message"`    // 提示信息
}

// GenerateVideo 提交视频生成任务（异步，返回task_id供前端轮询）
func (s *CoursewareAssetService) GenerateVideo(
ctx context.Context,
req *GenerateVideoServiceRequest,
) (*GenerateVideoServiceResponse, error) {
// 1. 校验课件和权限
cw, err := repository.GetCoursewareByID(ctx, req.CoursewareID)
if err != nil {
return nil, fmt.Errorf("课件不存在: %w", err)
}
if cw.UserID != req.UserID {
return nil, fmt.Errorf("无权操作此课件")
}

// 2. 校验页面
page, err := repository.GetCoursewarePageByNumber(ctx, req.CoursewareID, req.PageNumber)
if err != nil {
return nil, fmt.Errorf("页面不存在: 课件=%s 页码=%d", req.CoursewareID, req.PageNumber)
}

// 3. 获取视频生成API配置
videoCfg, err := ai.GetVideoConfig(s.cfg.GetAESKey())
if err != nil {
return nil, fmt.Errorf("视频生成API未配置: %w", err)
}

// 4. 构建参考图完整URL（本地路径转公网URL供豆包API下载）
refURL := ""
if req.RefImageURL != "" {
if strings.HasPrefix(req.RefImageURL, "/uploads/") {
refURL = "https://workflow.pkuailab.com" + req.RefImageURL
} else {
refURL = req.RefImageURL
}
}

// 5. 提交视频生成任务
traceCtx := &ai.TraceContext{
SceneCode: "courseware_video_gen",
UserID:    &req.UserID,
}
result, err := ai.SubmitVideoTask(ctx, videoCfg, req.Prompt, refURL, traceCtx)
if err != nil {
return nil, fmt.Errorf("视频任务提交失败: %w", err)
}

// 6. 写入数据库（status=generating，等待后续查询更新）
asset := &models.CoursewareAsset{
CoursewareID:     req.CoursewareID,
PageID:           &page.ID,
PlaceholderID:    result.TaskID, // 复用placeholder_id字段存储豆包task_id
AssetType:        models.CWAssetTypeVideo,
GenerationPrompt: req.Prompt,
OssURL:           "",
FileSize:         0,
MimeType:         "video/mp4",
Status:           models.CWAssetStatusGenerating,
}
if err := repository.CreateCWAsset(ctx, asset); err != nil {
return nil, fmt.Errorf("记录视频资产失败: %w", err)
}

cwAssetLog.Info("视频生成任务已提交",
"courseware_id", req.CoursewareID,
"page_number", req.PageNumber,
"asset_id", asset.ID,
"task_id", result.TaskID,
"model", result.ModelUsed,
"prompt_len", len(req.Prompt),
"has_ref_image", refURL != "",
)

return &GenerateVideoServiceResponse{
AssetID:   asset.ID,
TaskID:    result.TaskID,
ModelUsed: result.ModelUsed,
Message:   "视频生成任务已提交，通常需要30-120秒完成",
}, nil
}

// ==================== v0.42.1 视频任务状态查询 ====================

// QueryVideoStatusResponse 视频任务状态查询响应
type QueryVideoStatusResponse struct {
AssetID    string `json:"asset_id"`    // 资产记录ID
TaskID     string `json:"task_id"`     // 豆包任务ID
Status     string `json:"status"`      // 状态：generating/uploaded/failed
VideoURL   string `json:"video_url"`   // 视频本地URL（成功时有值）
Duration   int    `json:"duration"`    // 视频时长（秒）
Resolution string `json:"resolution"`  // 分辨率
Ratio      string `json:"ratio"`       // 画面比例
ErrorMsg   string `json:"error_msg"`   // 错误信息（失败时有值）
Message    string `json:"message"`     // 提示信息
}

// QueryVideoStatus 查询视频生成任务状态
// 如果任务已完成，自动下载视频保存到本地并更新数据库
func (s *CoursewareAssetService) QueryVideoStatus(
ctx context.Context,
assetID string,
userID string,
) (*QueryVideoStatusResponse, error) {
// 1. 查询资产记录
asset, err := repository.GetCWAssetByID(ctx, assetID)
if err != nil {
return nil, fmt.Errorf("视频资产不存在: %w", err)
}

// 2. 权限校验
cw, err := repository.GetCoursewareByID(ctx, asset.CoursewareID)
if err != nil {
return nil, fmt.Errorf("课件不存在: %w", err)
}
if cw.UserID != userID {
return nil, fmt.Errorf("无权操作此课件")
}

// 3. 如果已完成或失败，直接返回数据库中的状态
if asset.Status == models.CWAssetStatusUploaded || asset.Status == models.CWAssetStatusConfirmed {
return &QueryVideoStatusResponse{
AssetID:  asset.ID,
TaskID:   asset.PlaceholderID,
Status:   "uploaded",
VideoURL: asset.OssURL,
Message:  "视频已生成完成",
}, nil
}

// 4. 如果状态不是generating，说明异常
if asset.Status != models.CWAssetStatusGenerating {
return &QueryVideoStatusResponse{
AssetID:  asset.ID,
TaskID:   asset.PlaceholderID,
Status:   "failed",
ErrorMsg: "视频资产状态异常: " + asset.Status,
Message:  "视频生成出现问题",
}, nil
}

// 5. 从placeholder_id读取豆包task_id，查询任务状态
taskID := asset.PlaceholderID
if taskID == "" {
return nil, fmt.Errorf("视频任务ID为空")
}

videoCfg, err := ai.GetVideoConfig(s.cfg.GetAESKey())
if err != nil {
return nil, fmt.Errorf("视频生成API配置加载失败: %w", err)
}

queryResult, err := ai.QueryVideoTask(ctx, videoCfg, taskID)
if err != nil {
return nil, fmt.Errorf("查询视频任务状态失败: %w", err)
}

// 6. 根据豆包返回的状态处理
switch queryResult.Status {
case "running":
return &QueryVideoStatusResponse{
AssetID: asset.ID,
TaskID:  taskID,
Status:  "generating",
Message: "视频正在生成中，请稍后重试查询",
}, nil

case "succeeded":
if queryResult.VideoURL == "" {
return nil, fmt.Errorf("视频生成成功但未返回视频URL")
}

// 下载视频保存到本地，同时获取文件大小
localURL, fileSize, err := s.downloadAndSaveVideo(ctx, asset.CoursewareID, taskID, queryResult.VideoURL)
if err != nil {
cwAssetLog.Error("下载视频失败", "asset_id", asset.ID, "task_id", taskID, "error", err)
return nil, fmt.Errorf("下载视频文件失败: %w", err)
}

// 更新数据库（含文件大小）
if err := repository.UpdateCWAssetOSSURL(ctx, asset.ID, localURL, fileSize, "video/mp4"); err != nil {
cwAssetLog.Warn("更新视频URL失败", "asset_id", asset.ID, "error", err)
}

cwAssetLog.Info("视频生成完成并保存到本地",
"asset_id", asset.ID,
"task_id", taskID,
"duration", queryResult.Duration,
"resolution", queryResult.Resolution,
"file_size", fileSize,
"local_url", localURL,
)

return &QueryVideoStatusResponse{
AssetID:    asset.ID,
TaskID:     taskID,
Status:     "uploaded",
VideoURL:   localURL,
Duration:   queryResult.Duration,
Resolution: queryResult.Resolution,
Ratio:      queryResult.Ratio,
Message:    fmt.Sprintf("视频生成完成！时长%d秒，分辨率%s", queryResult.Duration, queryResult.Resolution),
}, nil

case "failed":
_ = repository.UpdateCWAssetStatus(ctx, asset.ID, models.CWAssetStatusPending)
cwAssetLog.Warn("视频生成失败", "asset_id", asset.ID, "task_id", taskID, "error", queryResult.ErrorMsg)
return &QueryVideoStatusResponse{
AssetID:  asset.ID,
TaskID:   taskID,
Status:   "failed",
ErrorMsg: queryResult.ErrorMsg,
Message:  "视频生成失败: " + queryResult.ErrorMsg,
}, nil

default:
return &QueryVideoStatusResponse{
AssetID: asset.ID,
TaskID:  taskID,
Status:  "generating",
Message: "视频状态: " + queryResult.Status,
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
