package models

// lesson_plan_asset.go — 教案附属资产模型
//
// v123 新增:为教案系统提供图片/文件附属资产存储能力
//
// 设计说明:
//   - 教案的 content_markdown 字段以 Markdown 文本形式存储正文,本身天然支持
//     ![alt](url) 图片语法,前端 renderMarkdown 已经在用。
//   - 本模型仅记录"图片资产元信息+物理文件路径",不直接修改 content_markdown,
//     图片插入由前端编辑器拼接 Markdown 完成,二者解耦。
//   - 为美术老师等强图片需求场景提供基础能力,后续课件工坊也会复用相同设计。

import "time"

// ==================== 资产类型常量 ====================

const (
        // AssetTypeImage 图片类型(当前唯一支持的类型)
        AssetTypeImage = "image"
        // AssetTypeFile 通用文件(预留,后续可扩展为附件/PDF/音频等)
        AssetTypeFile = "file"
)

// ==================== 实体 ====================

// LessonPlanAsset 教案附属资产(对应 lesson_plan_assets 表)
type LessonPlanAsset struct {
        ID           string     `json:"id"`              // UUID 主键
        LessonPlanID string     `json:"lesson_plan_id"`  // 所属教案 ID(外键,CASCADE 删除)
        UploaderID   string     `json:"uploader_id"`     // 上传者用户 ID
        AssetType    string     `json:"asset_type"`      // image / file
        FileName     string     `json:"file_name"`       // 原始文件名(仅展示用)
        FilePath     string     `json:"file_path"`       // 存储相对路径(相对于 /uploads/lesson-plans/)
        FileSize     int64      `json:"file_size"`       // 文件字节数
        MimeType     string     `json:"mime_type"`       // image/jpeg / image/png / image/webp / image/gif
        AltText      string     `json:"alt_text"`        // 图片描述(无障碍/Markdown alt 文本)
        Width        int        `json:"width"`           // 图片宽度(像素,可选)
        Height       int        `json:"height"`          // 图片高度(像素,可选)
        CreatedAt    *time.Time `json:"created_at"`
        UpdatedAt    *time.Time `json:"updated_at"`
}

// ==================== 列表项响应(给前端用,带可访问 URL) ====================

// LessonPlanAssetListItem 列表项响应(在实体基础上增加 URL 和上传者名)
type LessonPlanAssetListItem struct {
        LessonPlanAsset
        URL          string `json:"url"`           // 完整可访问 URL: /uploads/lesson-plans/{file_path}
        UploaderName string `json:"uploader_name"` // 上传者显示名(LEFT JOIN users 填充)
}

// ==================== 请求结构体 ====================

// UpdateAssetRequest 更新资产元数据请求(目前只支持改 alt_text)
type UpdateAssetRequest struct {
        AltText string `json:"alt_text"`
}

// ==================== 列表响应 ====================

// AssetListResponse 教案资产列表响应
type AssetListResponse struct {
        Assets []*LessonPlanAssetListItem `json:"assets"`
        Total  int                        `json:"total"`
}

// ==================== 上传响应 ====================

// AssetUploadResponse 上传成功响应(给前端编辑器立即拼 Markdown 用)
type AssetUploadResponse struct {
        ID       string `json:"id"`        // 资产 ID(便于后续删除等操作)
        FileName string `json:"file_name"` // 原始文件名(供 Markdown alt 展示)
        FileSize int64  `json:"file_size"`
        URL      string `json:"url"`       // 可直接拼到 Markdown ![alt](url) 中的 URL
        Markdown string `json:"markdown"`  // 已拼好的完整 Markdown 片段(可一键插入)
}
