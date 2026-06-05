package models

import (
	"time"
)

// ==================== 外部数据源配置模型 ====================

// ExternalDataConfig 对应数据库 external_data_configs 表（键值对存储）
type ExternalDataConfig struct {
	ID          string     `json:"id"`           // UUID 主键
	ConfigKey   string     `json:"config_key"`   // 配置键名
	ConfigValue string     `json:"config_value"` // 配置值（敏感字段为密文）
	Description string     `json:"description"`  // 配置描述
	UpdatedBy   *string    `json:"updated_by"`   // 最后更新者ID
	UpdatedAt   *time.Time `json:"updated_at"`   // 最后更新时间
}

// ==================== 配置键名常量 ====================

const (
	// OSS 读取相关配置（内部课程资产，只读权限）
	EDKeyOSSEndpoint     = "oss_endpoint"       // OSS Endpoint
	EDKeyOSSBucket       = "oss_bucket"         // OSS Bucket名称
	EDKeyOSSAccessKeyID  = "oss_access_key_id"  // OSS 读取AccessKey ID
	EDKeyOSSAccessKeyEnc = "oss_access_key_enc" // OSS 读取AccessKey Secret（AES加密存储）
	EDKeyOSSIndexPrefix  = "oss_index_prefix"   // OSS 索引文件路径前缀
	EDKeyOSSHTMLPrefix   = "oss_html_prefix"    // OSS HTML文件路径前缀

	// OSS 上传相关配置（课件资产上传，开放平台写权限，与读取桶权限隔离）
	EDKeyOSSUploadBucket       = "oss_upload_bucket"         // 课件上传专用Bucket
	EDKeyOSSUploadEndpoint     = "oss_upload_endpoint"       // 课件上传专用Endpoint
	EDKeyOSSUploadAccessKeyID  = "oss_upload_access_key_id"  // 课件上传专用AccessKey ID
	EDKeyOSSUploadAccessKeyEnc = "oss_upload_access_key_enc" // 课件上传专用AccessKey Secret（AES加密存储）

	// 推送API 相关配置
	EDKeyPushAPIURL   = "push_api_url"   // 推送回原始服务器的API地址
	EDKeyPushAPIToken = "push_api_token" // 推送API认证Token（AES加密存储）
)

// SensitiveEDKeys 需要AES加密存储的敏感配置键
// 两对AccessKey Secret + 推送Token 为敏感字段
var SensitiveEDKeys = map[string]bool{
	EDKeyOSSAccessKeyEnc:       true,
	EDKeyOSSUploadAccessKeyEnc: true,
	EDKeyPushAPIToken:          true,
}

// IsSensitiveEDKey 判断是否为敏感配置键
func IsSensitiveEDKey(key string) bool {
	return SensitiveEDKeys[key]
}

// EDKeyDescriptions 配置键名→中文描述映射
var EDKeyDescriptions = map[string]string{
	EDKeyOSSEndpoint:           "OSS Endpoint（如 oss-cn-beijing.aliyuncs.com）",
	EDKeyOSSBucket:             "OSS Bucket名称",
	EDKeyOSSAccessKeyID:        "OSS 读取AccessKey ID",
	EDKeyOSSAccessKeyEnc:       "OSS 读取AccessKey Secret（加密存储）",
	EDKeyOSSIndexPrefix:        "OSS 索引文件路径前缀",
	EDKeyOSSHTMLPrefix:         "OSS HTML文件路径前缀",
	EDKeyOSSUploadBucket:       "课件上传专用OSS Bucket",
	EDKeyOSSUploadEndpoint:     "课件上传专用OSS Endpoint",
	EDKeyOSSUploadAccessKeyID:  "课件上传专用OSS AccessKey ID",
	EDKeyOSSUploadAccessKeyEnc: "课件上传专用OSS AccessKey Secret（加密存储）",
	EDKeyPushAPIURL:            "推送回原始服务器的API地址",
	EDKeyPushAPIToken:          "推送API认证Token（加密存储）",
}

// EDKeyGroupOSS OSS读取配置分组（前端展示用）
var EDKeyGroupOSS = []string{
	EDKeyOSSEndpoint,
	EDKeyOSSBucket,
	EDKeyOSSAccessKeyID,
	EDKeyOSSAccessKeyEnc,
	EDKeyOSSIndexPrefix,
	EDKeyOSSHTMLPrefix,
}

// EDKeyGroupOSSUpload OSS上传配置分组（前端展示用）
var EDKeyGroupOSSUpload = []string{
	EDKeyOSSUploadBucket,
	EDKeyOSSUploadEndpoint,
	EDKeyOSSUploadAccessKeyID,
	EDKeyOSSUploadAccessKeyEnc,
}

var EDKeyGroupPush = []string{
	EDKeyPushAPIURL,
	EDKeyPushAPIToken,
}

// ==================== 请求/响应结构体 ====================

// ExternalDataConfigItem 返回给前端的单条配置（敏感字段脱敏）
type ExternalDataConfigItem struct {
	ConfigKey   string     `json:"config_key"`   // 配置键名
	ConfigValue string     `json:"config_value"` // 配置值（敏感字段已脱敏）
	Description string     `json:"description"`  // 配置描述
	IsSensitive bool       `json:"is_sensitive"` // 是否为敏感字段
	IsSet       bool       `json:"is_set"`       // 是否已配置（非占位符）
	UpdatedAt   *time.Time `json:"updated_at"`   // 最后更新时间
}

// ExternalDataConfigListResponse 配置列表响应
type ExternalDataConfigListResponse struct {
	Configs []*ExternalDataConfigItem `json:"configs"` // 配置列表
	Total   int                       `json:"total"`   // 总数
}

// UpdateExternalDataConfigsRequest 批量更新配置请求体
type UpdateExternalDataConfigsRequest struct {
	Configs map[string]string `json:"configs"` // 键值对（key→value，空字符串表示不修改敏感字段）
}
