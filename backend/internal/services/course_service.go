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

// ==================== 错误常量 ====================

var (
	ErrCourseCodeRequired    = errors.New("课程编号不能为空")
	ErrModuleIDRequired      = errors.New("外部模块ID不能为空")
	ErrCourseCodeExists      = errors.New("课程编号已存在")
	ErrCourseNotFound        = errors.New("课程不存在")
	ErrModuleIDAlreadyBound  = errors.New("该模块ID已绑定其他课程")
	ErrIndexNotAvailable     = errors.New("OSS上无该模块索引文件")
	ErrIndexContentEmpty     = errors.New("索引内容为空")
)

// ==================== CourseService ====================

// CourseService 课程管理业务逻辑层
type CourseService struct {
	ossService *OSSService // OSS客户端
}

// NewCourseService 创建课程服务实例
func NewCourseService(cfg *config.Config) *CourseService {
	return &CourseService{
		ossService: NewOSSService(cfg),
	}
}

// ==================== 课程列表 ====================

// ListCourses 获取课程列表（按角色过滤）
// admin看所有课程，非admin只看分配给自己的
func (s *CourseService) ListCourses(userID string, role string) (*models.CourseListResponse, error) {
	items, err := repository.ListCoursesForUser(userID, role)
	if err != nil {
		return nil, fmt.Errorf("获取课程列表失败: %w", err)
	}

	if items == nil {
		items = []*models.CourseListItem{}
	}

	return &models.CourseListResponse{
		Courses: items,
		Total:   len(items),
	}, nil
}

// ==================== 课程注册 ====================

// CreateCourse 注册新课程（从OSS模块注册到本系统）
// 1. 校验课程编号唯一性
// 2. 校验模块ID未被绑定
// 3. 如果未提供课程名称，从OSS模块元数据读取
// 4. 写入courses表
func (s *CourseService) CreateCourse(req *models.CreateCourseRequest) (*models.Course, error) {
	// 参数校验
	req.CourseCode = strings.TrimSpace(req.CourseCode)
	if req.CourseCode == "" {
		return nil, ErrCourseCodeRequired
	}
	if req.ExternalModuleID <= 0 {
		return nil, ErrModuleIDRequired
	}

	// 检查课程编号唯一性
	exists, err := repository.CheckCourseCodeExists(req.CourseCode)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrCourseCodeExists
	}

	// 检查模块ID是否已被绑定
	_, err = repository.GetCourseByModuleID(req.ExternalModuleID)
	if err == nil {
		// 找到了，说明已绑定
		return nil, ErrModuleIDAlreadyBound
	}

	// 如果未提供课程名称，尝试从OSS读取模块名称
	courseName := strings.TrimSpace(req.CourseName)
	if courseName == "" {
		catalog, catErr := s.ossService.FetchCatalog()
		if catErr == nil {
			for _, m := range catalog.Modules {
				if m.ID == req.ExternalModuleID {
					courseName = m.Name
					break
				}
			}
		}
	}
	if courseName == "" {
		courseName = req.CourseCode // 最终回退：用编号作名称
	}

	// 构建课程对象
	course := &models.Course{
		CourseCode:       req.CourseCode,
		CourseName:       courseName,
		ExternalModuleID: &req.ExternalModuleID,
		GradeNum:         req.GradeNum,
		Stage:            req.Stage,
		Semester:         req.Semester,
		Status:           models.CourseStatusActive,
	}

	// 写入数据库
	if err := repository.CreateCourse(course); err != nil {
		return nil, err
	}

	return course, nil
}

// ==================== 索引拉取 ====================

// FetchIndex 从OSS拉取指定课程的索引并存入数据库
// 1. 查找课程记录
// 2. 从OSS读取 indexes/{module_id}.json
// 3. 将索引内容拼接为TE-DNA编码原文
// 4. 计算SHA-256和页面数
// 5. 存入course_indexes表（upsert）
func (s *CourseService) FetchIndex(courseCode string) (*models.CourseIndex, error) {
	// 查找课程
	course, err := repository.GetCourseByCode(courseCode)
	if err != nil {
		return nil, ErrCourseNotFound
	}

	// 检查是否有外部模块ID
	if course.ExternalModuleID == nil || *course.ExternalModuleID <= 0 {
		return nil, fmt.Errorf("课程 %s 未绑定外部模块ID", courseCode)
	}
	moduleID := *course.ExternalModuleID

	// 从OSS拉取索引文件
	indexFile, err := s.ossService.FetchModuleIndex(moduleID)
	if err != nil {
		return nil, fmt.Errorf("从OSS拉取索引失败: %w", err)
	}

	// 构建TE-DNA索引原文（拼接所有content）
	indexContent := s.ossService.BuildIndexContent(indexFile)
	if indexContent == "" {
		return nil, ErrIndexContentEmpty
	}

	// 计算SHA-256校验码
	indexHash := utils.SHA256Hash(indexContent)

	// 计算页面数（索引条目数）
	pageCount := len(indexFile.Indexes)

	// 存入数据库（upsert）
	idx := &models.CourseIndex{
		CourseID:     course.ID,
		IndexContent: indexContent,
		IndexHash:    indexHash,
		PageCount:    pageCount,
		TotalLength:  len(indexContent),
	}

	if err := repository.UpsertCourseIndex(idx); err != nil {
		return nil, err
	}

	// 回查完整记录（含fetched_at）
	savedIdx, err := repository.GetCourseIndex(course.ID)
	if err != nil {
		return idx, nil // 回查失败但保存成功，返回构建的对象
	}

	return savedIdx, nil
}

// ==================== 索引查看 ====================

// GetIndexFull 获取完整索引（仅admin）
func (s *CourseService) GetIndexFull(courseCode string) (*models.CourseIndexFullResponse, error) {
	// 查找课程
	course, err := repository.GetCourseByCode(courseCode)
	if err != nil {
		return nil, ErrCourseNotFound
	}

	// 查找索引
	idx, err := repository.GetCourseIndex(course.ID)
	if err != nil {
		return nil, fmt.Errorf("课程 %s 尚未拉取索引", courseCode)
	}

	return &models.CourseIndexFullResponse{
		CourseCode:   course.CourseCode,
		CourseName:   course.CourseName,
		ModuleID:     course.ExternalModuleID,
		IndexContent: idx.IndexContent,
		IndexHash:    idx.IndexHash,
		PageCount:    idx.PageCount,
		TotalLength:  idx.TotalLength,
		FetchedAt:    idx.FetchedAt,
	}, nil
}

// GetIndexSummary 获取索引摘要（非admin可见）
// 只返回页面数、总长度、页面标题列表，不返回索引原文
func (s *CourseService) GetIndexSummary(courseCode string) (*models.CourseIndexSummaryResponse, error) {
	// 查找课程
	course, err := repository.GetCourseByCode(courseCode)
	if err != nil {
		return nil, ErrCourseNotFound
	}

	// 查找索引
	idx, err := repository.GetCourseIndex(course.ID)
	if err != nil {
		// 无索引
		return &models.CourseIndexSummaryResponse{
			CourseCode: course.CourseCode,
			CourseName: course.CourseName,
			HasIndex:   false,
		}, nil
	}

	// 从索引内容中提取页面标题
	// 格式: P{nn}: PT:{type}|... [S]...
	pageTitles := extractPageTitlesFromContent(idx.IndexContent)

	return &models.CourseIndexSummaryResponse{
		CourseCode:  course.CourseCode,
		CourseName:  course.CourseName,
		PageCount:   idx.PageCount,
		TotalLength: idx.TotalLength,
		PageTitles:  pageTitles,
		HasIndex:    true,
	}, nil
}

// ==================== OSS目录 ====================

// GetOSSCatalog 获取OSS目录（含注册状态，给前端展示可注册课程列表）
func (s *CourseService) GetOSSCatalog() (*models.OSSCatalogResponse, error) {
	return s.ossService.GetCatalogWithStatus()
}

// ==================== 内部工具 ====================

// extractPageTitlesFromContent 从TE-DNA索引原文中提取页面标题
// 匹配每行开头的 P{nn}: 模式，提取简短标识
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

		// 匹配 P{nn}: PT:{type}|... [S]{摘要}
		// 提取 P{nn} 和 [S] 后面的简短内容
		if len(line) >= 4 && line[0] == 'P' && line[3] == ':' {
			pageID := line[:3] // P01, P02, ...

			// 提取[S]标记后的内容作为标题（截取前40字符）
			sIdx := strings.Index(line, "[S]")
			if sIdx >= 0 {
				summary := line[sIdx+3:]
				// 截取到下一个标记或40字符
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
