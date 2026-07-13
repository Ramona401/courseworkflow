package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"tedna/internal/models"
	"tedna/internal/repository"
)

var (
	ErrUnitPlanMaterialNoPermission = errors.New("您没有权限操作该单元方案的参考资料")
	ErrUnitPlanMaterialTypeInvalid  = errors.New("参考资料类型非法")
	ErrUnitPlanMaterialNameRequired = errors.New("参考资料文件名不能为空")
	ErrUnitPlanMaterialContentEmpty = errors.New("参考资料内容为空")
	ErrUnitPlanMaterialTooLong      = errors.New("参考资料内容过长，请拆分后上传")
)

const (
	unitPlanMaterialContentMaxRunes = 120000
	unitPlanMaterialSummaryMaxRunes = 30000
	unitPlanMaterialNameMaxRunes    = 255

	unitPlanContextMaterialLimit  = 8
	unitPlanContextSingleMaxRunes = 6000
	unitPlanContextTotalMaxRunes  = 24000
)

// UnitPlanMaterialService 大单元参考资料服务。
type UnitPlanMaterialService struct{}

func NewUnitPlanMaterialService() *UnitPlanMaterialService {
	return &UnitPlanMaterialService{}
}

// List 返回当前用户可见的资料列表及是否可管理。
func (s *UnitPlanMaterialService) List(
	ctx context.Context,
	role string,
	userID string,
	unitPlanID string,
) ([]*models.UnitPlanMaterialListItem, bool, error) {
	plan, canRead, err := s.resolvePlanAccess(ctx, role, userID, unitPlanID)
	if err != nil {
		return nil, false, err
	}
	if !canRead {
		return nil, false, ErrUnitPlanMaterialNoPermission
	}

	items, err := repository.ListUnitPlanMaterials(ctx, unitPlanID)
	if err != nil {
		return nil, false, err
	}

	return items, plan.CreatedBy == userID, nil
}

// Create 新增资料。只有单元方案创建者可以操作。
func (s *UnitPlanMaterialService) Create(
	ctx context.Context,
	role string,
	userID string,
	unitPlanID string,
	req *models.CreateUnitPlanMaterialRequest,
) (*models.UnitPlanMaterial, error) {
	plan, _, err := s.resolvePlanAccess(ctx, role, userID, unitPlanID)
	if err != nil {
		return nil, err
	}
	if plan.CreatedBy != userID {
		return nil, ErrUnitPlanMaterialNoPermission
	}

	materialType := strings.TrimSpace(req.MaterialType)
	if !models.IsValidUnitPlanMaterialType(materialType) {
		return nil, ErrUnitPlanMaterialTypeInvalid
	}

	fileName := strings.TrimSpace(req.FileName)
	if fileName == "" {
		return nil, ErrUnitPlanMaterialNameRequired
	}
	if utf8.RuneCountInString(fileName) > unitPlanMaterialNameMaxRunes {
		return nil, ErrUnitPlanMaterialNameRequired
	}

	content := strings.TrimSpace(req.ContentText)
	summary := strings.TrimSpace(req.SummaryText)

	if content == "" && summary == "" {
		return nil, ErrUnitPlanMaterialContentEmpty
	}

	contentLength := utf8.RuneCountInString(content)
	summaryLength := utf8.RuneCountInString(summary)

	if contentLength > unitPlanMaterialContentMaxRunes ||
		summaryLength > unitPlanMaterialSummaryMaxRunes {
		return nil, ErrUnitPlanMaterialTooLong
	}

	material := &models.UnitPlanMaterial{
		UnitPlanID:     unitPlanID,
		MaterialType:   materialType,
		FileName:       fileName,
		ContentText:    content,
		SummaryText:    summary,
		OriginalLength: contentLength,
		SummaryLength:  summaryLength,
		UploadedBy:     userID,
	}

	if err := repository.CreateUnitPlanMaterial(ctx, material); err != nil {
		return nil, err
	}

	return material, nil
}

// Delete 软删除资料。只有单元方案创建者可以操作。
func (s *UnitPlanMaterialService) Delete(
	ctx context.Context,
	role string,
	userID string,
	unitPlanID string,
	materialID string,
) error {
	plan, _, err := s.resolvePlanAccess(ctx, role, userID, unitPlanID)
	if err != nil {
		return err
	}
	if plan.CreatedBy != userID {
		return ErrUnitPlanMaterialNoPermission
	}

	material, err := repository.GetUnitPlanMaterialByID(ctx, materialID)
	if err != nil {
		return err
	}
	if material.UnitPlanID != unitPlanID {
		return repository.ErrUnitPlanMaterialNotFound
	}

	return repository.ArchiveUnitPlanMaterial(ctx, materialID)
}

// BuildContext 为AI装配当前单元方案的有效资料。
//
// 每份资料优先使用summary_text，没有摘要时使用content_text。
// 同时限制资料数量、单份长度和总长度，防止无差别注入造成认知负担。
func (s *UnitPlanMaterialService) BuildContext(
	ctx context.Context,
	unitPlanID string,
) (string, int, error) {
	materials, err := repository.ListActiveUnitPlanMaterialsForContext(ctx, unitPlanID)
	if err != nil {
		return "", 0, err
	}

	var builder strings.Builder
	usedCount := 0
	totalRunes := 0

	for _, material := range materials {
		if usedCount >= unitPlanContextMaterialLimit ||
			totalRunes >= unitPlanContextTotalMaxRunes {
			break
		}

		text := strings.TrimSpace(material.EffectiveText())
		if text == "" {
			continue
		}

		text = truncateMaterialRunes(text, unitPlanContextSingleMaxRunes)

		remaining := unitPlanContextTotalMaxRunes - totalRunes
		text = truncateMaterialRunes(text, remaining)
		if text == "" {
			break
		}

		builder.WriteString("\n\n【大单元参考资料：")
		builder.WriteString(materialTypeLabel(material.MaterialType))
		builder.WriteString("｜")
		builder.WriteString(material.FileName)
		builder.WriteString("】\n")
		builder.WriteString(text)

		usedCount++
		totalRunes += utf8.RuneCountInString(text)
	}

	if usedCount == 0 {
		return "", 0, nil
	}

	builder.WriteString(
		"\n\n以上资料仅作为本单元设计依据；引用时应忠实于资料内容，资料未提供的信息不要臆造。",
	)

	return builder.String(), usedCount, nil
}

func (s *UnitPlanMaterialService) resolvePlanAccess(
	ctx context.Context,
	role string,
	userID string,
	unitPlanID string,
) (*models.UnitPlan, bool, error) {
	plan, err := repository.GetUnitPlanByID(ctx, unitPlanID)
	if err != nil {
		return nil, false, err
	}

	if plan.Status == models.UnitPlanStatusArchived {
		return plan, false, nil
	}

	if role == models.RoleAdmin || plan.CreatedBy == userID {
		return plan, true, nil
	}

	if plan.Status != models.UnitPlanStatusActive {
		return plan, false, nil
	}

	groups, _ := repository.GetUserTeachingGroups(ctx, userID)
	groupIDs := make([]string, 0, len(groups))

	for _, group := range groups {
		groupIDs = append(groupIDs, group.ID)
	}

	schoolIDs := make([]string, 0)

	if role == models.RoleSeniorOperator {
		school, schoolErr := repository.GetSchoolByAdminUserID(ctx, userID)
		if schoolErr == nil && school != nil && school.ID != "" {
			schoolIDs = append(schoolIDs, school.ID)
		}
	}

	visiblePlans, err := repository.ListUnitPlans(
		ctx,
		false,
		groupIDs,
		schoolIDs,
		userID,
	)
	if err != nil {
		return nil, false, err
	}

	for _, item := range visiblePlans {
		if item.ID == unitPlanID {
			return plan, true, nil
		}
	}

	return plan, false, nil
}

func materialTypeLabel(materialType string) string {
	switch materialType {
	case models.UnitPlanMaterialTypeTextbook:
		return "教材或课本"
	case models.UnitPlanMaterialTypeTeacherGuide:
		return "教师用书"
	case models.UnitPlanMaterialTypePreviousUnitPlan:
		return "既有大单元方案"
	case models.UnitPlanMaterialTypeTeachingRequirement:
		return "学校或区域教研要求"
	case models.UnitPlanMaterialTypeExcellentCase:
		return "优秀课例"
	default:
		return "其他资料"
	}
}

func truncateMaterialRunes(text string, maxRunes int) string {
	text = strings.TrimSpace(text)
	if text == "" || maxRunes <= 0 {
		return ""
	}

	runes := []rune(text)
	if len(runes) <= maxRunes {
		return text
	}

	if maxRunes <= 20 {
		return string(runes[:maxRunes])
	}

	return fmt.Sprintf(
		"%s\n……（资料内容已按上下文容量截断）",
		string(runes[:maxRunes-18]),
	)
}
