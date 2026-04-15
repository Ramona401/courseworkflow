package services

import (
	"errors"
	"fmt"
	"strings"

	"tedna/internal/config"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

var (
	ErrCourseCodeRequired   = errors.New("课程编号不能为空")
	ErrModuleIDRequired     = errors.New("外部模块ID不能为空")
	ErrCourseCodeExists     = errors.New("课程编号已存在")
	ErrCourseNotFound       = errors.New("课程不存在")
	ErrModuleIDAlreadyBound = errors.New("该模块ID已绑定其他课程")
	ErrIndexNotAvailable    = errors.New("OSS上无该模块索引文件")
	ErrIndexContentEmpty    = errors.New("索引内容为空")
)

// CourseService 课程管理业务逻辑层
type CourseService struct {
	ossService *OSSService
}

// NewCourseService 创建课程服务实例
func NewCourseService(cfg *config.Config) *CourseService {
	return &CourseService{ossService: NewOSSService(cfg)}
}

// ListCourses 获取课程列表（按角色过滤）
func (s *CourseService) ListCourses(userID string, role string) (*models.CourseListResponse, error) {
	items, err := repository.ListCoursesForUser(userID, role)
	if err != nil {
		return nil, fmt.Errorf("获取课程列表失败: %w", err)
	}
	if items == nil {
		items = []*models.CourseListItem{}
	}
	return &models.CourseListResponse{Courses: items, Total: len(items)}, nil
}

// CreateCourse 注册新课程
func (s *CourseService) CreateCourse(req *models.CreateCourseRequest) (*models.Course, error) {
	req.CourseCode = strings.TrimSpace(req.CourseCode)
	if req.CourseCode == "" {
		return nil, ErrCourseCodeRequired
	}
	if req.ExternalModuleID <= 0 {
		return nil, ErrModuleIDRequired
	}
	exists, err := repository.CheckCourseCodeExists(req.CourseCode)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrCourseCodeExists
	}
	_, err = repository.GetCourseByModuleID(req.ExternalModuleID)
	if err == nil {
		return nil, ErrModuleIDAlreadyBound
	}
	courseName := strings.TrimSpace(req.CourseName)
	if courseName == "" {
		catalog, catErr := s.ossService.FetchCatalog()
		if catErr == nil {
			for _, m := range catalog.Modules {
				if m.ID == req.ExternalModuleID {
					courseName = strings.TrimSpace(m.Name)
					break
				}
			}
		}
	}
	if courseName == "" {
		courseName = req.CourseCode
	}
	course := &models.Course{
		CourseCode: req.CourseCode, CourseName: courseName,
		ExternalModuleID: &req.ExternalModuleID, GradeNum: req.GradeNum,
		Stage: req.Stage, Semester: req.Semester, Status: models.CourseStatusActive,
	}
	if err := repository.CreateCourse(course); err != nil {
		return nil, err
	}
	return course, nil
}

// RegisterAndFetch 注册课程并自动拉取索引（一步完成）
func (s *CourseService) RegisterAndFetch(req *models.CreateCourseRequest) (map[string]interface{}, error) {
	// 先注册
	course, err := s.CreateCourse(req)
	if err != nil {
		return nil, err
	}
	// 再拉取索引
	idx, fetchErr := s.FetchIndex(course.CourseCode)
	result := map[string]interface{}{
		"course":      course,
		"index_fetched": fetchErr == nil,
	}
	if fetchErr == nil {
		result["page_count"] = idx.PageCount
		result["total_length"] = idx.TotalLength
		result["index_hash"] = idx.IndexHash
	} else {
		result["index_error"] = fetchErr.Error()
	}
	return result, nil
}

// BatchRegisterAndFetch 批量注册所有有索引的未注册模块并拉取索引
func (s *CourseService) BatchRegisterAndFetch() (map[string]interface{}, error) {
	// 获取OSS目录
	catalog, err := s.ossService.GetCatalogWithStatus()
	if err != nil {
		return nil, err
	}
	var registered, failed, skipped int
	var errors_list []string
	for _, mod := range catalog.Modules {
		// 跳过：已注册、无索引、名称不含课程编号
		if mod.IsRegistered {
			skipped++
			continue
		}
		if !mod.HasIndex {
			skipped++
			continue
		}
		// 从模块名称提取课程编号
		name := strings.TrimSpace(mod.Name)
		courseCode := extractCourseCodeFromName(name, mod.ID)
		req := &models.CreateCourseRequest{
			ExternalModuleID: mod.ID,
			CourseCode:       courseCode,
			CourseName:       name,
		}
		_, err := s.RegisterAndFetch(req)
		if err != nil {
			failed++
			errors_list = append(errors_list, fmt.Sprintf("%s(%d): %s", courseCode, mod.ID, err.Error()))
		} else {
			registered++
		}
	}
	return map[string]interface{}{
		"registered": registered, "failed": failed,
		"skipped": skipped, "total": len(catalog.Modules),
		"errors": errors_list,
	}, nil
}

// BatchFetchIndexes 对所有已注册课程重新拉取索引
func (s *CourseService) BatchFetchIndexes() (map[string]interface{}, error) {
	items, err := repository.ListCourses()
	if err != nil {
		return nil, err
	}
	var success, failed int
	var errors_list []string
	for _, item := range items {
		if item.ExternalModuleID == nil || *item.ExternalModuleID <= 0 {
			continue
		}
		_, fetchErr := s.FetchIndex(item.CourseCode)
		if fetchErr != nil {
			failed++
			errors_list = append(errors_list, fmt.Sprintf("%s: %s", item.CourseCode, fetchErr.Error()))
		} else {
			success++
		}
	}
	return map[string]interface{}{
		"success": success, "failed": failed,
		"total": len(items), "errors": errors_list,
	}, nil
}

// FetchIndex 从OSS拉取指定课程的索引
func (s *CourseService) FetchIndex(courseCode string) (*models.CourseIndex, error) {
	course, err := repository.GetCourseByCode(courseCode)
	if err != nil {
		return nil, ErrCourseNotFound
	}
	if course.ExternalModuleID == nil || *course.ExternalModuleID <= 0 {
		return nil, fmt.Errorf("课程 %s 未绑定外部模块ID", courseCode)
	}
	moduleID := *course.ExternalModuleID
	indexFile, err := s.ossService.FetchModuleIndex(moduleID)
	if err != nil {
		return nil, fmt.Errorf("从OSS拉取索引失败: %w", err)
	}
	indexContent := s.ossService.BuildIndexContent(indexFile)
	if indexContent == "" {
		return nil, ErrIndexContentEmpty
	}
	indexHash := utils.SHA256Hash(indexContent)
	// v99修复Bug3：过滤索引文件中的模块级摘要条目（Name不以P数字开头）
	// 摘要条目的Content以PG:开头（如"PG:20|KD:15%..."），不是实际页面
	pageCount := countPageEntries(indexFile.Indexes)
	idx := &models.CourseIndex{
		CourseID: course.ID, IndexContent: indexContent,
		IndexHash: indexHash, PageCount: pageCount, TotalLength: len(indexContent),
	}
	if err := repository.UpsertCourseIndex(idx); err != nil {
		return nil, err
	}
	savedIdx, err := repository.GetCourseIndex(course.ID)
	if err != nil {
		return idx, nil
	}
	return savedIdx, nil
}

// GetIndexFull 获取完整索引（仅admin）
func (s *CourseService) GetIndexFull(courseCode string) (*models.CourseIndexFullResponse, error) {
	course, err := repository.GetCourseByCode(courseCode)
	if err != nil {
		return nil, ErrCourseNotFound
	}
	idx, err := repository.GetCourseIndex(course.ID)
	if err != nil {
		return nil, fmt.Errorf("课程 %s 尚未拉取索引", courseCode)
	}
	return &models.CourseIndexFullResponse{
		CourseCode: course.CourseCode, CourseName: course.CourseName,
		ModuleID: course.ExternalModuleID, IndexContent: idx.IndexContent,
		IndexHash: idx.IndexHash, PageCount: idx.PageCount,
		TotalLength: idx.TotalLength, FetchedAt: idx.FetchedAt,
	}, nil
}

// GetIndexSummary 获取索引摘要（非admin可见）
func (s *CourseService) GetIndexSummary(courseCode string) (*models.CourseIndexSummaryResponse, error) {
	course, err := repository.GetCourseByCode(courseCode)
	if err != nil {
		return nil, ErrCourseNotFound
	}
	idx, err := repository.GetCourseIndex(course.ID)
	if err != nil {
		return &models.CourseIndexSummaryResponse{
			CourseCode: course.CourseCode, CourseName: course.CourseName, HasIndex: false,
		}, nil
	}
	pageTitles := extractPageTitlesFromContent(idx.IndexContent)
	return &models.CourseIndexSummaryResponse{
		CourseCode: course.CourseCode, CourseName: course.CourseName,
		PageCount: idx.PageCount, TotalLength: idx.TotalLength,
		PageTitles: pageTitles, HasIndex: true,
	}, nil
}

// GetOSSCatalog 获取OSS目录
func (s *CourseService) GetOSSCatalog() (*models.OSSCatalogResponse, error) {
	return s.ossService.GetCatalogWithStatus()
}

// extractCourseCodeFromName 从模块名称提取课程编号
// 如 "G1-01-AI动物识别实验室" → "G1-01"
// 如 "星云企业AI训练营" → "M228"（无编号时用模块ID）
func extractCourseCodeFromName(name string, moduleID int) string {
	name = strings.TrimSpace(name)
	// 匹配 G{数字}-{数字} 模式
	if len(name) >= 4 && name[0] == 'G' {
		for i := 1; i < len(name); i++ {
			if name[i] == '-' {
				// 找到第一个-，继续找第二个-或中文字符
				for j := i + 1; j < len(name); j++ {
					if name[j] == '-' {
						return name[:j]
					}
					// 遇到非ASCII字符（中文）停止
					if name[j] > 127 {
						return name[:j]
					}
				}
				return name // 没找到第二个分隔符，返回全名
			}
		}
	}
	return fmt.Sprintf("M%d", moduleID)
}

func extractPageTitlesFromContent(content string) []string {
	if content == "" {
		return nil
	}
	lines := strings.Split(content, "\n")
	var titles []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if len(line) >= 4 && line[0] == 'P' && line[3] == ':' {
			pageID := line[:3]
			sIdx := strings.Index(line, "[S]")
			if sIdx >= 0 {
				summary := line[sIdx+3:]
				nextBracket := strings.Index(summary, " [")
				if nextBracket > 0 && nextBracket < 60 {
					summary = summary[:nextBracket]
				}
				if len(summary) > 40 {
					summary = summary[:40] + "..."
				}
				titles = append(titles, pageID+": "+summary)
			} else {
				titles = append(titles, pageID)
			}
		}
	}
	return titles
}

// countPageEntries 计算索引条目中实际页面条目的数量
// v99新增：索引文件最后一条通常是模块级摘要（Name为课程名如"G1-03-表情翻译机"，Content以"PG:"开头）
// 这类条目不是实际页面，不应计入页面数
func countPageEntries(entries []*models.OSSIndexEntry) int {
	count := 0
	for _, e := range entries {
		if isPageEntry(e) {
			count++
		}
	}
	return count
}

// isPageEntry 判断索引条目是否为实际页面（而非模块级摘要）
// 页面条目的Name以"P数字"开头（如"P01-课程封面-V1.0"）
// 模块级摘要的Name为课程名（如"G1-03-表情翻译机"），Content以"PG:"开头
// isPageEntry 判断索引条目是否为实际页面（而非模块级摘要）
// v99修复：改用Content字段判断，因为Name格式因模块而异（有的用小写p，有的用课程名前缀）
// 页面条目的Content统一以"P数字:"开头（如"P01: PT:ST|..."），格式稳定可靠
// 模块级摘要的Content以"PG:"开头（如"PG:20|KD:15%..."）
func isPageEntry(e *models.OSSIndexEntry) bool {
	content := strings.TrimSpace(e.Content)
	if len(content) < 4 {
		return false
	}
	// 页面条目Content以P后跟数字再跟冒号开头（如"P01:"、"P10:"）
	if content[0] == 'P' && content[1] >= '0' && content[1] <= '9' {
		// 排除模块摘要"PG:"开头
		if len(content) >= 3 && content[1] == 'G' {
			return false
		}
		return true
	}
	return false
}

