package services

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"tedna/internal/config"
	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// OSSService 阿里云OSS客户端（读取内部课程资产 + 上传课件资产到开放平台桶）
type OSSService struct {
	aesKey string
}

// NewOSSService 创建OSS服务实例
func NewOSSService(cfg *config.Config) *OSSService {
	return &OSSService{aesKey: cfg.AESKey}
}

// ossConfig OSS读取连接配置（内部课程资产，只读权限）
type ossConfig struct {
	Endpoint     string
	Bucket       string
	AccessKeyID  string
	AccessKeySec string
	IndexPrefix  string
	HTMLPrefix   string
}

// ossUploadConfig OSS上传专用配置（开放平台写入桶，权限与读取桶隔离）
type ossUploadConfig struct {
	Endpoint     string // 上传Endpoint
	Bucket       string // 上传专用Bucket（如 20260525zuo）
	AccessKeyID  string // 上传专用AccessKey ID
	AccessKeySec string // 上传专用AccessKey Secret（已解密）
	PublicHost   string // 公网访问域名（如 20260525zuo.oss-cn-beijing.aliyuncs.com）
}

var ossLog = logger.WithModule("oss_service")

// isPlaceholderOrEmpty 判断配置值是否为空或占位符
func isPlaceholderOrEmpty(v string) bool {
	return v == "" || v == "PLACEHOLDER_SET_IN_ADMIN"
}

// getOSSConfig 从数据库读取并解密OSS读取配置
func (s *OSSService) getOSSConfig() (*ossConfig, error) {
	configs, err := repository.GetAllEDConfigs()
	if err != nil {
		return nil, fmt.Errorf("读取外部数据配置失败: %w", err)
	}
	cfgMap := make(map[string]string)
	for _, c := range configs {
		cfgMap[c.ConfigKey] = c.ConfigValue
	}
	endpoint := cfgMap["oss_endpoint"]
	bucket := cfgMap["oss_bucket"]
	accessKeyID := cfgMap["oss_access_key_id"]
	accessKeyEnc := cfgMap["oss_access_key_enc"]
	indexPrefix := cfgMap["oss_index_prefix"]
	htmlPrefix := cfgMap["oss_html_prefix"]
	if isPlaceholderOrEmpty(endpoint) {
		return nil, fmt.Errorf("OSS Endpoint未配置")
	}
	if isPlaceholderOrEmpty(bucket) {
		return nil, fmt.Errorf("OSS Bucket未配置")
	}
	if isPlaceholderOrEmpty(accessKeyID) {
		return nil, fmt.Errorf("OSS AccessKey ID未配置")
	}
	if isPlaceholderOrEmpty(accessKeyEnc) {
		return nil, fmt.Errorf("OSS AccessKey Secret未配置")
	}
	accessKeySec, err := utils.DecryptAES(accessKeyEnc, s.aesKey)
	if err != nil {
		return nil, fmt.Errorf("解密OSS AccessKey Secret失败: %w", err)
	}
	if isPlaceholderOrEmpty(indexPrefix) {
		indexPrefix = "edupkuailab/"
	}
	if isPlaceholderOrEmpty(htmlPrefix) {
		htmlPrefix = "edupkuailab/lessons/"
	}
	return &ossConfig{
		Endpoint: endpoint, Bucket: bucket,
		AccessKeyID: accessKeyID, AccessKeySec: accessKeySec,
		IndexPrefix: indexPrefix, HTMLPrefix: htmlPrefix,
	}, nil
}

// getUploadConfig 获取OSS上传专用配置
// AccessKey优先级：独立上传Key(oss_upload_access_key_id/enc) > 回退读取Key(oss_access_key_id/enc)
// 这样既支持读写权限隔离（推荐：填独立上传Key），又向后兼容（未填则复用读取Key）
func (s *OSSService) getUploadConfig() (*ossUploadConfig, error) {
	configs, err := repository.GetAllEDConfigs()
	if err != nil {
		return nil, fmt.Errorf("读取外部数据配置失败: %w", err)
	}
	cfgMap := make(map[string]string)
	for _, c := range configs {
		cfgMap[c.ConfigKey] = c.ConfigValue
	}

	// 上传专用Bucket和Endpoint（必填）
	uploadBucket := cfgMap["oss_upload_bucket"]
	uploadEndpoint := cfgMap["oss_upload_endpoint"]
	if isPlaceholderOrEmpty(uploadBucket) {
		return nil, fmt.Errorf("OSS上传Bucket未配置(oss_upload_bucket)")
	}
	if isPlaceholderOrEmpty(uploadEndpoint) {
		return nil, fmt.Errorf("OSS上传Endpoint未配置(oss_upload_endpoint)")
	}

	// AccessKey选取：优先独立上传Key，未配置则回退读取Key
	uploadKeyID := cfgMap["oss_upload_access_key_id"]
	uploadKeyEnc := cfgMap["oss_upload_access_key_enc"]
	var accessKeyID, accessKeyEnc string
	if !isPlaceholderOrEmpty(uploadKeyID) && !isPlaceholderOrEmpty(uploadKeyEnc) {
		// 使用独立上传Key（推荐，权限隔离）
		accessKeyID = uploadKeyID
		accessKeyEnc = uploadKeyEnc
		ossLog.Info("OSS上传使用独立上传AccessKey")
	} else {
		// 回退到读取Key（向后兼容）
		accessKeyID = cfgMap["oss_access_key_id"]
		accessKeyEnc = cfgMap["oss_access_key_enc"]
		ossLog.Info("OSS上传回退使用读取AccessKey（未配置独立上传Key）")
	}
	if isPlaceholderOrEmpty(accessKeyID) {
		return nil, fmt.Errorf("OSS上传AccessKey ID未配置")
	}
	if isPlaceholderOrEmpty(accessKeyEnc) {
		return nil, fmt.Errorf("OSS上传AccessKey Secret未配置")
	}
	accessKeySec, err := utils.DecryptAES(accessKeyEnc, s.aesKey)
	if err != nil {
		return nil, fmt.Errorf("解密OSS上传AccessKey Secret失败: %w", err)
	}

	// 公网访问域名：将内网endpoint替换为外网
	// oss-cn-beijing-internal.aliyuncs.com → oss-cn-beijing.aliyuncs.com
	publicEndpoint := strings.Replace(uploadEndpoint, "-internal", "", 1)
	publicHost := uploadBucket + "." + publicEndpoint
	return &ossUploadConfig{
		Endpoint:     uploadEndpoint,
		Bucket:       uploadBucket,
		AccessKeyID:  accessKeyID,
		AccessKeySec: accessKeySec,
		PublicHost:   publicHost,
	}, nil
}

// ==================== OSS上传（课件资产上传到云盘） ====================

// UploadFileToOSS 将本地文件上传到OSS，返回公网可访问的URL
func (s *OSSService) UploadFileToOSS(localPath string, ossKey string, contentType string) (string, error) {
	cfg, err := s.getUploadConfig()
	if err != nil {
		return "", fmt.Errorf("获取OSS上传配置失败: %w", err)
	}

	file, err := os.Open(localPath)
	if err != nil {
		return "", fmt.Errorf("打开本地文件失败: %w", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("获取文件信息失败: %w", err)
	}

	putURL := fmt.Sprintf("https://%s.%s/%s", cfg.Bucket, cfg.Endpoint, ossKey)
	req, err := http.NewRequest("PUT", putURL, file)
	if err != nil {
		return "", fmt.Errorf("创建上传请求失败: %w", err)
	}

	date := time.Now().UTC().Format(http.TimeFormat)
	req.Header.Set("Date", date)
	req.Header.Set("Content-Type", contentType)
	req.ContentLength = fileInfo.Size()

	resource := fmt.Sprintf("/%s/%s", cfg.Bucket, ossKey)
	signStr := "PUT" + "\n" + "\n" + contentType + "\n" + date + "\n" + resource
	mac := hmac.New(sha1.New, []byte(cfg.AccessKeySec))
	mac.Write([]byte(signStr))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	req.Header.Set("Authorization", fmt.Sprintf("OSS %s:%s", cfg.AccessKeyID, signature))

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("OSS上传请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)
		if len(bodyStr) > 300 {
			bodyStr = bodyStr[:300]
		}
		return "", fmt.Errorf("OSS上传失败(HTTP %d): %s", resp.StatusCode, bodyStr)
	}

	publicURL := fmt.Sprintf("https://%s/%s", cfg.PublicHost, ossKey)

	ossLog.Info("文件上传OSS成功",
		"local_path", localPath,
		"oss_key", ossKey,
		"size", fileInfo.Size(),
		"public_url", publicURL,
	)

	return publicURL, nil
}

// UploadAssetToOSS 将课件资产（图片/视频/音频）上传到OSS
func (s *OSSService) UploadAssetToOSS(localURL string) (string, error) {
	if !strings.HasPrefix(localURL, "/uploads/") {
		return "", fmt.Errorf("不支持的资源路径格式: %s", localURL)
	}
	localPath := "/www/wwwroot/tedna" + localURL

	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		return "", fmt.Errorf("本地文件不存在: %s", localPath)
	}

	ossKey := strings.TrimPrefix(localURL, "/uploads/")

	ext := strings.ToLower(filepath.Ext(localPath))
	contentType := "application/octet-stream"
	switch ext {
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".png":
		contentType = "image/png"
	case ".webp":
		contentType = "image/webp"
	case ".gif":
		contentType = "image/gif"
	case ".svg":
		contentType = "image/svg+xml"
	case ".mp4":
		contentType = "video/mp4"
	case ".webm":
		contentType = "video/webm"
	case ".mov":
		contentType = "video/quicktime"
	case ".avi":
		contentType = "video/x-msvideo"
	case ".mp3":
		contentType = "audio/mpeg"
	case ".wav":
		contentType = "audio/wav"
	}

	return s.UploadFileToOSS(localPath, ossKey, contentType)
}

// signAndGet 签名并执行OSS GET对象请求
func (s *OSSService) signAndGet(cfg *ossConfig, objectKey string) ([]byte, error) {
	url := fmt.Sprintf("https://%s.%s/%s", cfg.Bucket, cfg.Endpoint, objectKey)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	date := time.Now().UTC().Format(http.TimeFormat)
	req.Header.Set("Date", date)
	resource := fmt.Sprintf("/%s/%s", cfg.Bucket, objectKey)
	sig := s.sign(cfg.AccessKeySec, "GET", date, resource)
	req.Header.Set("Authorization", fmt.Sprintf("OSS %s:%s", cfg.AccessKeyID, sig))

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
		return nil, fmt.Errorf("OSS错误(HTTP %d): %s", resp.StatusCode, string(body)[:200])
	}
	return io.ReadAll(resp.Body)
}

// listObjects 列出OSS指定前缀下的所有对象Key
func (s *OSSService) listObjects(cfg *ossConfig, prefix string) ([]string, error) {
	var allKeys []string
	marker := ""
	for {
		query := "list-type=2&max-keys=1000&prefix=" + prefix
		if marker != "" {
			query += "&continuation-token=" + marker
		}
		url := fmt.Sprintf("https://%s.%s/?%s", cfg.Bucket, cfg.Endpoint, query)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
		date := time.Now().UTC().Format(http.TimeFormat)
		req.Header.Set("Date", date)
		resource := fmt.Sprintf("/%s/", cfg.Bucket)
		sig := s.sign(cfg.AccessKeySec, "GET", date, resource)
		req.Header.Set("Authorization", fmt.Sprintf("OSS %s:%s", cfg.AccessKeyID, sig))

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("列OSS目录失败: %w", err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("列OSS目录错误(HTTP %d)", resp.StatusCode)
		}

		bodyStr := string(body)
		for {
			start := strings.Index(bodyStr, "<Key>")
			if start < 0 {
				break
			}
			end := strings.Index(bodyStr[start:], "</Key>")
			if end < 0 {
				break
			}
			key := bodyStr[start+5 : start+end]
			bodyStr = bodyStr[start+end+6:]
			allKeys = append(allKeys, key)
		}

		if !strings.Contains(string(body), "<IsTruncated>true</IsTruncated>") {
			break
		}
		tStart := strings.Index(string(body), "<NextContinuationToken>")
		tEnd := strings.Index(string(body)[tStart:], "</NextContinuationToken>")
		if tStart < 0 || tEnd < 0 {
			break
		}
		marker = string(body)[tStart+23 : tStart+tEnd]
	}
	return allKeys, nil
}

// sign 计算OSS V1签名
func (s *OSSService) sign(secret string, method string, date string, resource string) string {
	str := method + "\n\n\n" + date + "\n" + resource
	mac := hmac.New(sha1.New, []byte(secret))
	mac.Write([]byte(str))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// FetchCatalog 从OSS拉取全局目录
func (s *OSSService) FetchCatalog() (*models.OSSCatalog, error) {
	cfg, err := s.getOSSConfig()
	if err != nil {
		return nil, err
	}
	data, err := s.signAndGet(cfg, cfg.IndexPrefix+"catalog.json")
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
func (s *OSSService) FetchModuleIndex(moduleID int) (*models.OSSIndexFile, error) {
	cfg, err := s.getOSSConfig()
	if err != nil {
		return nil, err
	}
	objectKey := fmt.Sprintf("%sindexes/%d.json", cfg.IndexPrefix, moduleID)
	data, err := s.signAndGet(cfg, objectKey)
	if err != nil {
		return nil, fmt.Errorf("拉取索引失败(module=%d): %w", moduleID, err)
	}
	var indexFile models.OSSIndexFile
	if err := json.Unmarshal(data, &indexFile); err != nil {
		return nil, fmt.Errorf("解析索引失败(module=%d): %w", moduleID, err)
	}
	return &indexFile, nil
}

// BuildIndexContent 将OSS索引文件转换为TE-DNA索引原文
func (s *OSSService) BuildIndexContent(indexFile *models.OSSIndexFile) string {
	if indexFile == nil || len(indexFile.Indexes) == 0 {
		return ""
	}
	entries := make([]*models.OSSIndexEntry, len(indexFile.Indexes))
	copy(entries, indexFile.Indexes)
	var pageEntries []*models.OSSIndexEntry
	for _, e := range entries {
		c := strings.TrimSpace(e.Content)
		if len(c) >= 4 && c[0] == 'P' && c[1] >= '0' && c[1] <= '9' {
			if len(c) >= 3 && c[1] == 'G' {
				continue
			}
			pageEntries = append(pageEntries, e)
		}
	}
	entries = pageEntries
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].SortOrder < entries[j].SortOrder
	})
	var parts []string
	for _, entry := range entries {
		c := strings.TrimSpace(entry.Content)
		if c != "" {
			parts = append(parts, c)
		}
	}
	return strings.Join(parts, "\n")
}

// ExtractPageTitles 提取页面标题列表
func (s *OSSService) ExtractPageTitles(indexFile *models.OSSIndexFile) []string {
	if indexFile == nil || len(indexFile.Indexes) == 0 {
		return nil
	}
	entries := make([]*models.OSSIndexEntry, len(indexFile.Indexes))
	copy(entries, indexFile.Indexes)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].SortOrder < entries[j].SortOrder
	})
	var titles []string
	for _, e := range entries {
		if e.Name != "" {
			titles = append(titles, e.Name)
		}
	}
	return titles
}

// GetCatalogWithStatus 获取OSS目录并标记注册状态和索引状态
func (s *OSSService) GetCatalogWithStatus() (*models.OSSCatalogResponse, error) {
	catalog, err := s.FetchCatalog()
	if err != nil {
		return nil, err
	}
	registeredMap, err := repository.GetAllRegisteredModuleIDs()
	if err != nil {
		return nil, fmt.Errorf("查询已注册模块失败: %w", err)
	}
	cfg, err := s.getOSSConfig()
	indexSet := make(map[int]bool)
	if err == nil {
		prefix := cfg.IndexPrefix + "indexes/"
		keys, listErr := s.listObjects(cfg, prefix)
		if listErr == nil {
			for _, key := range keys {
				if strings.HasSuffix(key, ".json") {
					name := key[strings.LastIndex(key, "/")+1:]
					name = strings.TrimSuffix(name, ".json")
					var id int
					if _, err := fmt.Sscanf(name, "%d", &id); err == nil && id > 0 {
						indexSet[id] = true
					}
				}
			}
		}
	}

	var modules []*models.OSSModuleListItem
	for _, m := range catalog.Modules {
		item := &models.OSSModuleListItem{
			ID: m.ID, Name: m.Name,
			LessonCount: m.LessonCount, Status: m.Status,
		}
		if courseCode, ok := registeredMap[m.ID]; ok {
			item.IsRegistered = true
			item.CourseCode = courseCode
		}
		item.HasIndex = indexSet[m.ID]
		modules = append(modules, item)
	}
	return &models.OSSCatalogResponse{
		Version: catalog.Version, TotalModules: catalog.TotalModules,
		TotalLessons: catalog.TotalLessons, Modules: modules,
		GeneratedAt: catalog.GeneratedAt,
	}, nil
}

// ==================== Generator所需的OSS读取（P4-6新增）====================

// FetchModuleDetail 从OSS拉取模块详情（含lessons列表，用于建立页码→lesson_id映射）
func (s *OSSService) FetchModuleDetail(moduleID int) (*models.OSSModuleDetail, error) {
	cfg, err := s.getOSSConfig()
	if err != nil {
		return nil, err
	}
	objectKey := fmt.Sprintf("%smodules/%d.json", cfg.IndexPrefix, moduleID)
	data, err := s.signAndGet(cfg, objectKey)
	if err != nil {
		return nil, fmt.Errorf("拉取模块详情失败(module=%d): %w", moduleID, err)
	}
	var detail models.OSSModuleDetail
	if err := json.Unmarshal(data, &detail); err != nil {
		return nil, fmt.Errorf("解析模块详情失败(module=%d): %w", moduleID, err)
	}
	return &detail, nil
}

// FetchLessonHTML 从OSS读取单个课时的HTML内容
func (s *OSSService) FetchLessonHTML(lessonID int) (string, error) {
	cfg, err := s.getOSSConfig()
	if err != nil {
		return "", err
	}
	objectKey := fmt.Sprintf("%s%d.html", cfg.HTMLPrefix, lessonID)
	data, err := s.signAndGet(cfg, objectKey)
	if err != nil {
		return "", fmt.Errorf("读取课件HTML失败(lesson=%d): %w", lessonID, err)
	}
	return string(data), nil
}

// BuildPageLessonMap 建立页码→lesson_id的映射
func (s *OSSService) BuildPageLessonMap(moduleID int) (map[int]int, error) {
	detail, err := s.FetchModuleDetail(moduleID)
	if err != nil {
		return nil, err
	}
	if len(detail.Lessons) == 0 {
		return nil, fmt.Errorf("模块%d无课时数据", moduleID)
	}

	pageNumRe := regexp.MustCompile(`^P(\d{1,3})-`)
	pageMap := make(map[int]int)
	fallbackIdx := 1

	for _, lesson := range detail.Lessons {
		if lesson.Status != 1 || lesson.StudentDisabled == 1 {
			continue
		}
		m := pageNumRe.FindStringSubmatch(lesson.Title)
		if m != nil {
			pageNum := 0
			_, _ = fmt.Sscanf(m[1], "%d", &pageNum)
			if pageNum > 0 {
				pageMap[pageNum] = lesson.ID
				continue
			}
		}
		for pageMap[fallbackIdx] != 0 {
			fallbackIdx++
		}
		pageMap[fallbackIdx] = lesson.ID
		fallbackIdx++
	}

	if len(pageMap) == 0 {
		return nil, fmt.Errorf("模块%d页码映射为空", moduleID)
	}
	return pageMap, nil
}

// ==================== OSS删除（删图时连带删云盘副本） ====================

// DeleteObjectFromOSS 从OSS删除指定公网URL对应的对象
// 入参为上传时返回的公网URL（如 https://20260525zuo.oss-cn-beijing.aliyuncs.com/courseware-assets/xxx.jpg）
// 内部从URL中解析出OSS Key，用V1签名发起DELETE请求
// 删除幂等：对象不存在(404)视为成功；其余错误返回error由调用方决定是否阻断
func (s *OSSService) DeleteObjectFromOSS(publicURL string) error {
	cfg, err := s.getUploadConfig()
	if err != nil {
		return fmt.Errorf("获取OSS上传配置失败: %w", err)
	}

	// 从公网URL解析OSS Key：去掉 "https://{publicHost}/" 前缀
	prefix := "https://" + cfg.PublicHost + "/"
	if !strings.HasPrefix(publicURL, prefix) {
		// URL不属于当前上传桶(可能是历史旧桶遗留的URL)。
		// 这种情况无法用当前桶的Key删除,记INFO并视为"跳过"(返回nil),
		// 不当作错误——避免换桶后删本地资产时被旧URL阻断或刷WARN。
		ossLog.Info("OSS删除跳过:URL不属于当前上传桶(可能为旧桶遗留)",
			"public_url", publicURL, "current_host", cfg.PublicHost)
		return nil
	}
	ossKey := strings.TrimPrefix(publicURL, prefix)
	if ossKey == "" {
		return fmt.Errorf("解析出的OSS Key为空: %s", publicURL)
	}

	// 构建DELETE请求（走上传Endpoint）
	delURL := fmt.Sprintf("https://%s.%s/%s", cfg.Bucket, cfg.Endpoint, ossKey)
	req, err := http.NewRequest("DELETE", delURL, nil)
	if err != nil {
		return fmt.Errorf("创建删除请求失败: %w", err)
	}

	// V1签名（DELETE方法,无Content-Type参与签名,与GET一致用sign辅助）
	date := time.Now().UTC().Format(http.TimeFormat)
	req.Header.Set("Date", date)
	resource := fmt.Sprintf("/%s/%s", cfg.Bucket, ossKey)
	sig := s.sign(cfg.AccessKeySec, "DELETE", date, resource)
	req.Header.Set("Authorization", fmt.Sprintf("OSS %s:%s", cfg.AccessKeyID, sig))

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("OSS删除请求失败: %w", err)
	}
	defer resp.Body.Close()

	// OSS删除成功返回204;对象不存在返回404也视为成功(幂等)
	if resp.StatusCode == 204 || resp.StatusCode == 200 || resp.StatusCode == 404 {
		ossLog.Info("OSS对象删除成功(或本不存在)", "oss_key", ossKey, "status", resp.StatusCode)
		return nil
	}

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	if len(bodyStr) > 300 {
		bodyStr = bodyStr[:300]
	}
	return fmt.Errorf("OSS删除失败(HTTP %d): %s", resp.StatusCode, bodyStr)
}
