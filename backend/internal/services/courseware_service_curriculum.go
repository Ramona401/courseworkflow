package services

// courseware_service_curriculum.go
//
// 本文件承载：
//   - 课件页面基础操作；
//   - 课件步骤回退；
//   - 从主题和3D场景创建课件；
//   - K12课程知识点编码的可信教育域校验。
//
// 课程知识点直接编码查询必须显式携带可信Actor教育域。
// vocational、adult、mixed、空值和非法值均返回空候选；
// K12数据库错误向上返回，不能伪装成“没有知识点”。

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== 页面操作 ====================

// GetPages 获取课件的所有页面
func (s *CoursewareService) GetPages(ctx context.Context, coursewareID string) ([]*models.CoursewarePage, error) {
	return repository.ListCoursewarePages(ctx, coursewareID)
}

// GetPagesForView 安全获取课件全部页面。
func (s *CoursewareService) GetPagesForView(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
) ([]*models.CoursewarePage, error) {
	if _, err := s.LoadCoursewareForView(
		ctx,
		coursewareID,
		actor,
	); err != nil {
		return nil, err
	}

	return repository.ListCoursewarePages(
		ctx,
		coursewareID,
	)
}

// UpdatePageIndex 更新单页索引说明
func (s *CoursewareService) UpdatePageIndex(ctx context.Context, coursewareID string, pageNumber int, userID string, req *models.UpdateCWPageIndexRequest) error {
	cw, err := repository.GetCoursewareByID(ctx, coursewareID)
	if err != nil {
		return fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != userID {
		return fmt.Errorf("无权操作此课件")
	}
	return repository.UpdateCWPageIndex(ctx, coursewareID, pageNumber, req)
}

// AddPage 手动添加课件页面
func (s *CoursewareService) AddPage(ctx context.Context, coursewareID string, userID string, req *models.AddCWPageRequest) (*models.CoursewarePage, error) {
	cw, err := repository.GetCoursewareByID(ctx, coursewareID)
	if err != nil {
		return nil, fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != userID {
		return nil, fmt.Errorf("无权操作此课件")
	}

	count, _ := repository.CountCoursewarePages(ctx, coursewareID)
	page := &models.CoursewarePage{
		CoursewareID:        coursewareID,
		PageNumber:          count + 1,
		Title:               req.Title,
		Purpose:             req.Purpose,
		ContentSummary:      req.ContentSummary,
		InteractionType:     req.InteractionType,
		VisualFormat:        req.VisualFormat,
		MediaRequirements:   req.MediaRequirements,
		EstimatedComplexity: req.EstimatedComplexity,
		Status:              models.CWPageStatusPending,
	}
	if page.EstimatedComplexity <= 0 {
		page.EstimatedComplexity = 1
	}
	if err := repository.CreateCoursewarePage(ctx, page); err != nil {
		return nil, fmt.Errorf("添加页面失败: %w", err)
	}
	_ = repository.UpdateCoursewarePageCount(ctx, coursewareID, count+1)
	_ = s.ResyncCWPageNumbers(ctx, coursewareID)
	return page, nil
}

// DeletePage 删除课件页面
func (s *CoursewareService) DeletePage(ctx context.Context, coursewareID string, pageNumber int, userID string) error {
	cw, err := repository.GetCoursewareByID(ctx, coursewareID)
	if err != nil {
		return fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != userID {
		return fmt.Errorf("无权操作此课件")
	}
	if err := repository.DeleteCoursewarePage(ctx, coursewareID, pageNumber); err != nil {
		return err
	}
	count, _ := repository.CountCoursewarePages(ctx, coursewareID)
	_ = repository.UpdateCoursewarePageCount(ctx, coursewareID, count)
	_ = s.ResyncCWPageNumbers(ctx, coursewareID)
	return nil
}

// ReorderPages 重新排序课件页面
func (s *CoursewareService) ReorderPages(ctx context.Context, coursewareID string, userID string, pageIDs []string) error {
	cw, err := repository.GetCoursewareByID(ctx, coursewareID)
	if err != nil {
		return fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != userID {
		return fmt.Errorf("无权操作此课件")
	}
	if err := repository.ReorderCoursewarePages(ctx, coursewareID, pageIDs); err != nil {
		return err
	}
	_ = s.ResyncCWPageNumbers(ctx, coursewareID)
	return nil
}

// ==================== 步骤回退 ====================

// RollbackStatus 回退课件状态到指定目标步骤。
func (s *CoursewareService) RollbackStatus(ctx context.Context, id string, userID string, targetStatus string) error {
	cw, err := repository.GetCoursewareByID(ctx, id)
	if err != nil {
		return fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != userID {
		return fmt.Errorf("无权操作此课件")
	}
	if cw.Status == models.CoursewareStatusInPipeline {
		return fmt.Errorf("已提交审核的课件不可回退")
	}

	targetOrder, targetOK := models.CoursewareStatusOrder[targetStatus]
	currentOrder, currentOK := models.CoursewareStatusOrder[cw.Status]
	if !targetOK || !currentOK {
		return fmt.Errorf("无效的状态: current=%s target=%s", cw.Status, targetStatus)
	}
	if targetOrder >= currentOrder {
		return fmt.Errorf("只能回退到更早的步骤（当前=%s, 目标=%s）", cw.Status, targetStatus)
	}

	if targetStatus == models.CoursewareStatusDraft ||
		targetStatus == models.CoursewareStatusIndexing {
		_ = repository.UpdateCoursewareNavTemplate(ctx, id, "")
		_ = repository.UpdateCoursewareStyle(ctx, id, "")
	}

	cwServiceLog.Info(
		"课件状态回退",
		"courseware_id", id,
		"from", cw.Status,
		"to", targetStatus,
		"user_id", userID,
	)
	return repository.UpdateCoursewareStatus(ctx, id, targetStatus)
}

// ==================== 从主题直接创建课件 ====================

// CreateCoursewareFromTopic 从主题直接创建课件。
func (s *CoursewareService) CreateCoursewareFromTopic(
	ctx context.Context,
	actor *CoursewareActorContext,
	req *models.CreateCoursewareFromTopicRequest,
) (*models.Courseware, error) {
	domain, err := ResolveCoursewareCreationEducationDomain(actor)
	if err != nil {
		return nil, err
	}
	if req == nil {
		return nil, fmt.Errorf("课件创建请求不能为空")
	}

	req.Subject = strings.TrimSpace(req.Subject)
	req.Grade = strings.TrimSpace(req.Grade)
	req.Topic = strings.TrimSpace(req.Topic)

	if req.Subject == "" {
		return nil, fmt.Errorf("学科不能为空")
	}
	if req.Grade == "" {
		return nil, fmt.Errorf("年级不能为空")
	}
	if req.Topic == "" {
		return nil, fmt.Errorf("主题名称不能为空")
	}

	// 在INSERT之前完成知识点直接编码查询。
	//
	// K12数据库错误会阻断创建，防止先留下课件再把查询错误
	// 伪装成“没有知识库约束”。非K12域安全返回空编码。
	validCodes := []string{}
	if len(req.KPCodes) > 0 {
		validCodes, err = s.filterValidKPCodes(
			ctx,
			actor,
			req.KPCodes,
			req.Subject,
			req.Grade,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"校验知识点编码失败: %w",
				err,
			)
		}
	}

	courseware := &models.Courseware{
		LessonPlanID:    nil,
		UserID:          actor.UserID,
		Title:           req.Topic,
		Subject:         req.Subject,
		EducationDomain: domain,
		Grade:           req.Grade,
		Status:          models.CoursewareStatusDraft,
		SourceType:      models.CWSourceTopicDirect,
		PageCount:       0,
	}

	if err := repository.CreateCourseware(ctx, courseware); err != nil {
		return nil, fmt.Errorf("创建课件失败: %w", err)
	}

	if len(validCodes) < len(req.KPCodes) {
		cwServiceLog.Warn(
			"部分知识点编码无效或因教育域限制被过滤",
			"courseware_id", courseware.ID,
			"education_domain", domain,
			"submitted", len(req.KPCodes),
			"valid", len(validCodes),
		)
	}

	if len(validCodes) > 0 {
		knowledgePointJSON, marshalErr := json.Marshal(validCodes)
		if marshalErr != nil {
			cwServiceLog.Warn(
				"序列化课件知识点编码失败",
				"error", marshalErr,
				"courseware_id", courseware.ID,
			)
		} else if updateErr := repository.UpdateCoursewareKPCodes(
			ctx,
			courseware.ID,
			string(knowledgePointJSON),
		); updateErr != nil {
			cwServiceLog.Warn(
				"存储课件知识点编码失败",
				"error", updateErr,
				"courseware_id", courseware.ID,
			)
		} else {
			courseware.KPCodes = string(knowledgePointJSON)
		}
	}

	cwServiceLog.Info(
		"从主题创建课件",
		"courseware_id", courseware.ID,
		"subject", req.Subject,
		"grade", req.Grade,
		"topic", req.Topic,
		"user_id", actor.UserID,
		"education_domain", domain,
	)

	return courseware, nil
}

// ==================== 创建3D互动单页课件 ====================

// CreateCoursewareFrom3D 创建3D互动单页课件。
func (s *CoursewareService) CreateCoursewareFrom3D(
	ctx context.Context,
	actor *CoursewareActorContext,
	subject string,
	grade string,
	topic string,
	description string,
) (*models.Courseware, error) {
	domain, err := ResolveCoursewareCreationEducationDomain(actor)
	if err != nil {
		return nil, err
	}
	userID := actor.UserID

	if subject == "" {
		return nil, fmt.Errorf("学科不能为空")
	}
	if grade == "" {
		return nil, fmt.Errorf("年级不能为空")
	}
	if topic == "" {
		return nil, fmt.Errorf("主题名称不能为空")
	}
	if len([]rune(description)) < 20 {
		return nil, fmt.Errorf("详细描述至少需要20个字")
	}

	cw := &models.Courseware{
		LessonPlanID:    nil,
		UserID:          userID,
		Title:           topic,
		Subject:         subject,
		EducationDomain: domain,
		Grade:           grade,
		Status:          models.CoursewareStatusGenerating,
		SourceType:      models.CWSource3DSingle,
		PageCount:       1,
	}

	if err := repository.CreateCourseware(ctx, cw); err != nil {
		return nil, fmt.Errorf("创建课件失败: %w", err)
	}

	page := &models.CoursewarePage{
		CoursewareID:        cw.ID,
		PageNumber:          1,
		Title:               topic,
		Purpose:             "3D互动演示：" + topic,
		ContentSummary:      description,
		InteractionType:     "3d",
		VisualFormat:        "fullscreen_media",
		MediaRequirements:   description,
		EstimatedComplexity: 5,
		Status:              models.CWPageStatusPending,
	}
	if err := repository.CreateCoursewarePage(ctx, page); err != nil {
		cwServiceLog.Warn(
			"创建3D页面记录失败",
			"error", err,
			"courseware_id", cw.ID,
		)
	}

	cwServiceLog.Info(
		"创建3D互动单页课件",
		"courseware_id", cw.ID,
		"subject", subject,
		"grade", grade,
		"topic", topic,
		"desc_len", len([]rune(description)),
		"user_id", userID,
	)
	return cw, nil
}

// ==================== 课程知识库编码校验 ====================

// filterValidKPCodes 只保留真实存在且学科、年级匹配的知识点编码。
//
// 非K12域返回空切片；K12数据库错误向上返回并阻断创建。
func (s *CoursewareService) filterValidKPCodes(
	ctx context.Context,
	actor *CoursewareActorContext,
	codes []string,
	subject string,
	gradeText string,
) ([]string, error) {
	if len(codes) == 0 {
		return []string{}, nil
	}

	educationDomain := ""
	if actor != nil {
		educationDomain = strings.ToLower(
			strings.TrimSpace(actor.EducationDomain),
		)
	}

	knowledgePoints, err := repository.GetCurriculumKPsByCodes(
		ctx,
		educationDomain,
		codes,
	)
	if err != nil {
		return nil, err
	}
	if len(knowledgePoints) == 0 {
		return []string{}, nil
	}

	gradeNum := parseGradeNumForKP(gradeText)

	validSet := make(map[string]bool, len(knowledgePoints))
	for _, knowledgePoint := range knowledgePoints {
		if knowledgePoint.Subject != subject {
			continue
		}
		if gradeNum > 0 &&
			knowledgePoint.GradeNum > 0 &&
			knowledgePoint.GradeNum != gradeNum {
			continue
		}
		validSet[knowledgePoint.KPCode] = true
	}

	validCodes := make([]string, 0, len(codes))
	for _, code := range codes {
		if validSet[code] {
			validCodes = append(validCodes, code)
		}
	}

	return validCodes, nil
}

// parseGradeNumForKP 年级文字转数字，无法识别时返回0。
func parseGradeNumForKP(gradeText string) int {
	value := strings.TrimSpace(gradeText)
	if value == "" {
		return 0
	}

	if strings.Contains(value, "高一") ||
		strings.Contains(value, "高中一") {
		return 10
	}
	if strings.Contains(value, "高二") ||
		strings.Contains(value, "高中二") {
		return 11
	}
	if strings.Contains(value, "高三") ||
		strings.Contains(value, "高中三") {
		return 12
	}

	if strings.Contains(value, "初一") ||
		strings.Contains(value, "初中一") {
		return 7
	}
	if strings.Contains(value, "初二") ||
		strings.Contains(value, "初中二") {
		return 8
	}
	if strings.Contains(value, "初三") ||
		strings.Contains(value, "初中三") {
		return 9
	}

	chineseNumbers := map[string]int{
		"一":  1,
		"二":  2,
		"三":  3,
		"四":  4,
		"五":  5,
		"六":  6,
		"七":  7,
		"八":  8,
		"九":  9,
		"十":  10,
		"十一": 11,
		"十二": 12,
	}
	for chineseNumber, number := range chineseNumbers {
		if strings.Contains(
			value,
			chineseNumber+"年级",
		) {
			return number
		}
	}

	for number := 12; number >= 1; number-- {
		text := fmt.Sprintf("%d", number)
		if strings.Contains(value, text+"年级") ||
			value == text {
			return number
		}
	}

	return 0
}
