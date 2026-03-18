package services

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"tedna/internal/config"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// ==================== OSSService ====================

// OSSService 阿里云OSS只读客户端
// 从external_data_configs表读取OSS配置，通过HTTP REST API访问OSS
type OSSService struct {
	aesKey string // AES密钥（用于解密OSS AccessKey Secret）
}

// NewOSSService 创建OSS服务实例
func NewOSSService(cfg *config.Config) *OSSService {
	return &OSSService{
		aesKey: cfg.AESKey,
	}
}

// ==================== OSS配置读取 ====================

// ossConfig OSS连接配置（从数据库读取）
type ossConfig struct {
	Endpoint    string // OSS端点（如 oss-cn-beijing.aliyuncs.com）
	Bucket      string // Bucket名称
	AccessKeyID string // AccessKey ID
	AccessKeySec string // AccessKey Secret（已解密）
	IndexPrefix string // 索引前缀（如 edupkuailab/）
	HTMLPrefix  string // HTML前缀（如 edupkuailab/lessons/）
}

// getOSSConfig 从数据库读取并解密OSS配置
func (s *OSSService) getOSSConfig() (*ossConfig, error) {
	// 读取所有外部数据配置
	configs, err := repository.GetAllEDConfigs()
	if err != nil {
		return nil, fmt.Errorf("读取外部数据配置失败: %w", err)
	}

	// 构建配置映射
	cfgMap := make(map[string]string)
	for _, c := range configs {
		cfgMap[c.ConfigKey] = c.ConfigValue
	}

	// 检查必要配置
	endpoint := cfgMap["oss_endpoint"]
	bucket := cfgMap["oss_bucket"]
	accessKeyID := cfgMap["oss_access_key_id"]
	accessKeyEnc := cfgMap["oss_access_key_enc"]
	indexPrefix := cfgMap["oss_index_prefix"]
	htmlPrefix := cfgMap["oss_html_prefix"]

	if endpoint == "" || endpoint == "PLACEHOLDER_SET_IN_ADMIN" {
		return nil, fmt.Errorf("OSS Endpoint未配置")
	}
	if bucket == "" || bucket == "PLACEHOLDER_SET_IN_ADMIN" {
		return nil, fmt.Errorf("OSS Bucket未配置")
	}
	if accessKeyID == "" || accessKeyID == "PLACEHOLDER_SET_IN_ADMIN" {
		return nil, fmt.Errorf("OSS AccessKey ID未配置")
	}
	if accessKeyEnc == "" || accessKeyEnc == "PLACEHOLDER_SET_IN_ADMIN" {
		return nil, fmt.Errorf("OSS AccessKey Secret未配置")
	}

	// 解密AccessKey Secret
	accessKeySec, err := utils.DecryptAES(accessKeyEnc, s.aesKey)
	if err != nil {
		return nil, fmt.Errorf("解密OSS AccessKey Secret失败: %w", err)
	}

	// 索引前缀默认值
	if indexPrefix == "" || indexPrefix == "PLACEHOLDER_SET_IN_ADMIN" {
		indexPrefix = "edupkuailab/"
	}
	if htmlPrefix == "" || htmlPrefix == "PLACEHOLDER_SET_IN_ADMIN" {
		htmlPrefix = "edupkuailab/lessons/"
	}

	return &ossConfig{
		Endpoint:    endpoint,
		Bucket:      bucket,
		AccessKeyID: accessKeyID,
		AccessKeySec: accessKeySec,
		IndexPrefix: indexPrefix,
		HTMLPrefix:  htmlPrefix,
	}, nil
}

// ==================== OSS HTTP签名 ====================

// signOSSRequest 生成阿里云OSS V1签名
// 签名格式: Authorization: OSS {AccessKeyId}:{Signature}
// Signature = Base64(HMAC-SHA1(AccessKeySecret, StringToSign))
// StringToSign = VERB + "\n" + Content-MD5 + "\n" + Content-Type + "\n" + Date + "\n" + CanonicalizedOSSHeaders + CanonicalizedResource
func (s *OSSService) signOSSRequest(cfg *ossConfig, method string, objectKey string) (*http.Request, error) {
	// 构建URL: https://{bucket}.{endpoint}/{objectKey}
	url := fmt.Sprintf("https://%s.%s/%s", cfg.Bucket, cfg.Endpoint, objectKey)

	// 构建请求
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建OSS请求失败: %w", err)
	}

	// 设置日期头
	date := time.Now().UTC().Format(http.TimeFormat)
	req.Header.Set("Date", date)

	// 构建StringToSign
	// GET\n\n\n{Date}\n/{bucket}/{objectKey}
	canonicalResource := fmt.Sprintf("/%s/%s", cfg.Bucket, objectKey)
	stringToSign := fmt.Sprintf("%s\n\n\n%s\n%s", method, date, canonicalResource)

	// 计算HMAC-SHA1签名
	mac := hmac.New(sha1.New, []byte(cfg.AccessKeySec))
	mac.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	// 设置Authorization头
	req.Header.Set("Authorization", fmt.Sprintf("OSS %s:%s", cfg.AccessKeyID, signature))

	return req, nil
}

// ossGet 执行OSS GET请求，返回响应体内容
func (s *OSSService) ossGet(cfg *ossConfig, objectKey string) ([]byte, error) {
	req, err := s.signOSSRequest(cfg, "GET", objectKey)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("OSS请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("OSS对象不存在: %s", objectKey)
	}
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OSS请求错误(HTTP %d): %s", resp.StatusCode, string(body)[:200])
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取OSS响应失败: %w", err)
	}
	return data, nil
}

// ossObjectExists 检查OSS对象是否存在（HEAD请求）
func (s *OSSService) ossObjectExists(cfg *ossConfig, objectKey string) bool {
	req, err := s.signOSSRequest(cfg, "HEAD", objectKey)
	if err != nil {
		return false
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200
}

// ==================== 业务方法 ====================

// FetchCatalog 从OSS拉取全局目录（catalog.json）
func (s *OSSService) FetchCatalog() (*models.OSSCatalog, error) {
	cfg, err := s.getOSSConfig()
	if err != nil {
		return nil, err
	}

	// 读取 {prefix}catalog.json
	objectKey := cfg.IndexPrefix + "catalog.json"
	data, err := s.ossGet(cfg, objectKey)
	if err != nil {
		return nil, fmt.Errorf("拉取catalog.json失败: %w", err)
	}

	var catalog models.OSSCatalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return nil, fmt.Errorf("解析catalog.json失败: %w", err)
	}

	return &catalog, nil
}

// FetchModuleIndex 从OSS拉取指定模块的索引文件
// 路径: {prefix}indexes/{moduleID}.json
func (s *OSSService) FetchModuleIndex(moduleID int) (*models.OSSIndexFile, error) {
	cfg, err := s.getOSSConfig()
	if err != nil {
		return nil, err
	}

	// 读取 {prefix}indexes/{moduleID}.json
	objectKey := fmt.Sprintf("%sindexes/%d.json", cfg.IndexPrefix, moduleID)
	data, err := s.ossGet(cfg, objectKey)
	if err != nil {
		return nil, fmt.Errorf("拉取索引文件失败(module=%d): %w", moduleID, err)
	}

	var indexFile models.OSSIndexFile
	if err := json.Unmarshal(data, &indexFile); err != nil {
		return nil, fmt.Errorf("解析索引文件失败(module=%d): %w", moduleID, err)
	}

	return &indexFile, nil
}

// CheckModuleIndexExists 检查OSS上是否存在指定模块的索引文件
func (s *OSSService) CheckModuleIndexExists(moduleID int) (bool, error) {
	cfg, err := s.getOSSConfig()
	if err != nil {
		return false, err
	}

	objectKey := fmt.Sprintf("%sindexes/%d.json", cfg.IndexPrefix, moduleID)
	return s.ossObjectExists(cfg, objectKey), nil
}

// BuildIndexContent 将OSS索引文件转换为TE-DNA索引原文
// 将indexes数组中每条记录的content按sort_order排序后拼接
func (s *OSSService) BuildIndexContent(indexFile *models.OSSIndexFile) string {
	if indexFile == nil || len(indexFile.Indexes) == 0 {
		return ""
	}

	// 按sort_order排序
	entries := make([]*models.OSSIndexEntry, len(indexFile.Indexes))
	copy(entries, indexFile.Indexes)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].SortOrder < entries[j].SortOrder
	})

	// 拼接所有索引内容
	var parts []string
	for _, entry := range entries {
		content := strings.TrimSpace(entry.Content)
		if content != "" {
			parts = append(parts, content)
		}
	}

	return strings.Join(parts, "\n")
}

// ExtractPageTitles 从OSS索引文件中提取页面标题列表（用于非admin摘要）
func (s *OSSService) ExtractPageTitles(indexFile *models.OSSIndexFile) []string {
	if indexFile == nil || len(indexFile.Indexes) == 0 {
		return nil
	}

	// 按sort_order排序
	entries := make([]*models.OSSIndexEntry, len(indexFile.Indexes))
	copy(entries, indexFile.Indexes)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].SortOrder < entries[j].SortOrder
	})

	var titles []string
	for _, entry := range entries {
		if entry.Name != "" {
			titles = append(titles, entry.Name)
		}
	}
	return titles
}

// GetCatalogWithStatus 获取OSS目录并标记各模块的注册状态和索引状态
func (s *OSSService) GetCatalogWithStatus() (*models.OSSCatalogResponse, error) {
	// 1. 从OSS拉取目录
	catalog, err := s.FetchCatalog()
	if err != nil {
		return nil, err
	}

	// 2. 查询已注册的模块ID映射
	registeredMap, err := repository.GetAllRegisteredModuleIDs()
	if err != nil {
		return nil, fmt.Errorf("查询已注册模块失败: %w", err)
	}

	// 3. 获取OSS配置（用于检查索引文件是否存在）
	cfg, err := s.getOSSConfig()
	if err != nil {
		return nil, err
	}

	// 4. 组装响应
	var modules []*models.OSSModuleListItem
	for _, m := range catalog.Modules {
		item := &models.OSSModuleListItem{
			ID:          m.ID,
			Name:        m.Name,
			LessonCount: m.LessonCount,
			Status:      m.Status,
		}

		// 检查是否已注册
		if courseCode, ok := registeredMap[m.ID]; ok {
			item.IsRegistered = true
			item.CourseCode = courseCode
		}

		// 检查OSS上是否有索引文件
		objectKey := fmt.Sprintf("%sindexes/%d.json", cfg.IndexPrefix, m.ID)
		item.HasIndex = s.ossObjectExists(cfg, objectKey)

		modules = append(modules, item)
	}

	return &models.OSSCatalogResponse{
		Version:      catalog.Version,
		TotalModules: catalog.TotalModules,
		TotalLessons: catalog.TotalLessons,
		Modules:      modules,
		GeneratedAt:  catalog.GeneratedAt,
	}, nil
}
