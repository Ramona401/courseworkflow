package services

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"tedna/internal/config"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// OSSService 阿里云OSS只读客户端
type OSSService struct {
	aesKey string
}

// NewOSSService 创建OSS服务实例
func NewOSSService(cfg *config.Config) *OSSService {
	return &OSSService{aesKey: cfg.AESKey}
}

// ossConfig OSS连接配置
type ossConfig struct {
	Endpoint     string
	Bucket       string
	AccessKeyID  string
	AccessKeySec string
	IndexPrefix  string
	HTMLPrefix   string
}

// getOSSConfig 从数据库读取并解密OSS配置
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
	accessKeySec, err := utils.DecryptAES(accessKeyEnc, s.aesKey)
	if err != nil {
		return nil, fmt.Errorf("解密OSS AccessKey Secret失败: %w", err)
	}
	if indexPrefix == "" || indexPrefix == "PLACEHOLDER_SET_IN_ADMIN" {
		indexPrefix = "edupkuailab/"
	}
	if htmlPrefix == "" || htmlPrefix == "PLACEHOLDER_SET_IN_ADMIN" {
		htmlPrefix = "edupkuailab/lessons/"
	}
	return &ossConfig{
		Endpoint: endpoint, Bucket: bucket,
		AccessKeyID: accessKeyID, AccessKeySec: accessKeySec,
		IndexPrefix: indexPrefix, HTMLPrefix: htmlPrefix,
	}, nil
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
	// v99修复Bug3+Bug8：过滤模块级摘要条目，只保留实际页面条目
	// 改用Content字段判断（Name格式因模块而异，Content格式统一）
	// 页面条目Content以"P数字:"开头，摘要条目Content以"PG:"开头
	var pageEntries []*models.OSSIndexEntry
	for _, e := range entries {
		c := strings.TrimSpace(e.Content)
		if len(c) >= 4 && c[0] == 'P' && c[1] >= '0' && c[1] <= '9' {
			// 排除模块摘要"PG:"开头
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
// 优化：通过列目录一次性获取所有索引文件名，避免逐个HEAD
func (s *OSSService) GetCatalogWithStatus() (*models.OSSCatalogResponse, error) {
	catalog, err := s.FetchCatalog()
	if err != nil {
		return nil, err
	}
	registeredMap, err := repository.GetAllRegisteredModuleIDs()
	if err != nil {
		return nil, fmt.Errorf("查询已注册模块失败: %w", err)
	}
	// 一次性列出indexes/目录获取所有索引文件
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
// 对应 OSS路径：edupkuailab/modules/{module_id}.json
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
// 对应 OSS路径：edupkuailab/lessons/{lesson_id}.html
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
// 策略：从lesson.Title解析页码（如"P10-清晰度挑战-V1.0"→页码10）
// 不依赖order顺序，避免modules JSON与indexes JSON不同步导致页码错位
// 返回：map[页码]lesson_id
func (s *OSSService) BuildPageLessonMap(moduleID int) (map[int]int, error) {
	detail, err := s.FetchModuleDetail(moduleID)
	if err != nil {
		return nil, err
	}
	if len(detail.Lessons) == 0 {
		return nil, fmt.Errorf("模块%d无课时数据", moduleID)
	}

	// 从title解析页码：匹配"P数字-"开头格式
	// 例：P08-任务发布-v2.0 → 页码8
	// 例：P10-清晰度挑战-V1.0 → 页码10
	pageNumRe := regexp.MustCompile(`^P(\d{1,3})-`)
	pageMap := make(map[int]int)
	fallbackIdx := 1 // 解析失败时的兜底页码

	for _, lesson := range detail.Lessons {
		// v90-3修复Bug3：过滤禁用页面，避免被纳入索引压缩和AI评估
		// Status != 1 表示课时被禁用；StudentDisabled == 1 表示对学生不可见
		// 这类页面不应参与Pipeline评估和优化流程
		if lesson.Status != 1 || lesson.StudentDisabled == 1 {
			continue
		}
		m := pageNumRe.FindStringSubmatch(lesson.Title)
		if m != nil {
			pageNum := 0
			_, _ = fmt.Sscanf(m[1], "%d", &pageNum)
			if pageNum > 0 {
				// title解析成功，直接用页码
				pageMap[pageNum] = lesson.ID
				continue
			}
		}
		// title不符合P数字格式，兜底按order顺序（按order排序后填入空缺位置）
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
