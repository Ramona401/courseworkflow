package services

// courseware_comic_project_helpers.go — 漫画项目与可信知识上下文纯辅助
//
// 本文件负责：
//   - 教材单元和课标知识点重新读取；
//   - 教材单元kp_codes对子集选择约束；
//   - 教材和知识点稳定快照构建；
//   - 年级解析、默认页面配置和可选ID规范化；
//   - 教师编辑IAOCI关系边界校验。
//
// 覆盖层协议校验已拆分到courseware_comic_overlay_validation.go；
// 浏览器安全视图构建已拆分到courseware_comic_project_views.go。

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

var coursewareComicArabicGradePattern = regexp.MustCompile(`(1[0-2]|[1-9])`)

func loadCoursewareComicTextbookUnit(
	ctx context.Context,
	courseware *models.Courseware,
	request *models.CreateCoursewareComicProjectRequest,
	gradeNum int,
) (*models.TextbookUnit, error) {
	units, err :=
		repository.ListTextbookUnits(
			ctx,
			models.EducationDomainK12,
			strings.TrimSpace(
				courseware.Subject,
			),
			request.Publisher,
			gradeNum,
			request.Semester,
		)
	if err != nil {
		return nil, err
	}

	for _, unit := range units {
		if unit == nil ||
			strings.TrimSpace(unit.ID) !=
				request.TextbookUnitID {
			continue
		}

		if strings.TrimSpace(
			unit.Subject,
		) != strings.TrimSpace(
			courseware.Subject,
		) ||
			unit.GradeNum != gradeNum ||
			strings.TrimSpace(
				unit.Publisher,
			) != request.Publisher {
			return nil,
				ErrCoursewareComicProjectUnitNotFound
		}

		if request.Semester != "" &&
			strings.TrimSpace(
				unit.Semester,
			) != request.Semester {
			return nil,
				ErrCoursewareComicProjectUnitNotFound
		}

		return unit, nil
	}

	return nil,
		ErrCoursewareComicProjectUnitNotFound
}

// validateCoursewareComicKPCodesBelongToUnit 收紧教材单元边界。
//
// 教材单元已经声明kp_codes时，教师选择必须是该列表的子集。
// 历史或人工教材单元没有任何kp_codes时，仅执行学科和年级校验。
func validateCoursewareComicKPCodesBelongToUnit(
	unit *models.TextbookUnit,
	selectedCodes []string,
) error {
	if unit == nil {
		return ErrCoursewareComicProjectUnitNotFound
	}

	raw :=
		strings.TrimSpace(
			unit.KPCodesJSON,
		)

	if raw == "" ||
		raw == "[]" ||
		raw == "null" {
		return nil
	}

	var unitCodes []string

	if err := json.Unmarshal(
		[]byte(raw),
		&unitCodes,
	); err != nil {
		return fmt.Errorf(
			"教材单元知识点关联数据异常: %w",
			err,
		)
	}

	allowed := make(map[string]bool)

	for _, code := range unitCodes {
		code = strings.TrimSpace(code)
		if code != "" {
			allowed[code] = true
		}
	}

	if len(allowed) == 0 {
		return nil
	}

	for _, selectedCode := range selectedCodes {
		if !allowed[strings.TrimSpace(
			selectedCode,
		)] {
			return ErrCoursewareComicProjectKnowledgePointOutsideUnit
		}
	}

	return nil
}

func loadCoursewareComicKnowledgePoints(
	ctx context.Context,
	courseware *models.Courseware,
	gradeNum int,
	kpCodes []string,
) ([]*models.CurriculumKP, error) {
	if len(kpCodes) == 0 {
		return nil,
			ErrCoursewareComicProjectKnowledgePointInvalid
	}

	items, err :=
		repository.GetCurriculumKPsByCodes(
			ctx,
			models.EducationDomainK12,
			kpCodes,
		)
	if err != nil {
		return nil, err
	}

	byCode := make(
		map[string]*models.CurriculumKP,
		len(items),
	)

	for _, item := range items {
		if item == nil ||
			strings.TrimSpace(
				item.Subject,
			) != strings.TrimSpace(
				courseware.Subject,
			) ||
			(item.GradeNum != 0 &&
				item.GradeNum != gradeNum) {
			return nil,
				ErrCoursewareComicProjectKnowledgePointInvalid
		}

		byCode[item.KPCode] = item
	}

	result := make(
		[]*models.CurriculumKP,
		0,
		len(kpCodes),
	)

	for _, code := range kpCodes {
		item, exists := byCode[code]
		if !exists {
			return nil,
				ErrCoursewareComicProjectKnowledgePointInvalid
		}

		result = append(result, item)
	}

	return result, nil
}

func buildCoursewareComicUnitSnapshot(
	unit *models.TextbookUnit,
) (*models.CoursewareComicTextbookUnitSnapshot, error) {
	if unit == nil {
		return nil,
			ErrCoursewareComicProjectUnitNotFound
	}

	kpCodes := []string{}

	raw :=
		strings.TrimSpace(
			unit.KPCodesJSON,
		)

	if raw != "" &&
		raw != "null" {
		if err := json.Unmarshal(
			[]byte(raw),
			&kpCodes,
		); err != nil {
			return nil,
				fmt.Errorf(
					"教材单元知识点关联数据异常: %w",
					err,
				)
		}
	}

	return &models.CoursewareComicTextbookUnitSnapshot{
		ID:             unit.ID,
		Publisher:      unit.Publisher,
		GradeNum:       unit.GradeNum,
		Semester:       unit.Semester,
		UnitNumber:     unit.UnitNumber,
		UnitTitle:      unit.UnitTitle,
		LessonTitle:    unit.LessonTitle,
		ContentSummary: unit.ContentSummary,
		KPCodes:        kpCodes,
	}, nil
}

func buildCoursewareComicKPSnapshots(
	items []*models.CurriculumKP,
) []models.CoursewareComicKnowledgePointSnapshot {
	result := make(
		[]models.CoursewareComicKnowledgePointSnapshot,
		0,
		len(items),
	)

	for _, item := range items {
		result = append(
			result,
			models.CoursewareComicKnowledgePointSnapshot{
				KPCode:              item.KPCode,
				KPName:              item.KPName,
				ContentRequirement:  item.ContentRequirement,
				AcademicRequirement: item.AcademicRequirement,
				TeachingHint:        item.TeachingHint,
				DepthLevel:          item.DepthLevel,
				SourceRef:           item.SourceRef,
			},
		)
	}

	return result
}

func buildCoursewareComicKnowledgeContent(
	unit *models.TextbookUnit,
	items []*models.CurriculumKP,
) string {
	var builder strings.Builder

	builder.WriteString("教材单元：")
	builder.WriteString(unit.UnitTitle)

	if strings.TrimSpace(
		unit.LessonTitle,
	) != "" {
		builder.WriteString("；课题：")
		builder.WriteString(unit.LessonTitle)
	}

	if strings.TrimSpace(
		unit.ContentSummary,
	) != "" {
		builder.WriteString("\n单元概述：")
		builder.WriteString(unit.ContentSummary)
	}

	for index, item := range items {
		builder.WriteString(
			fmt.Sprintf(
				"\n知识点%d：%s（%s）",
				index+1,
				item.KPName,
				item.KPCode,
			),
		)

		if item.ContentRequirement != "" {
			builder.WriteString("\n内容要求：")
			builder.WriteString(
				item.ContentRequirement,
			)
		}

		if item.AcademicRequirement != "" {
			builder.WriteString("\n学业要求：")
			builder.WriteString(
				item.AcademicRequirement,
			)
		}

		if item.TeachingHint != "" {
			builder.WriteString("\n教学提示：")
			builder.WriteString(
				item.TeachingHint,
			)
		}
	}

	return strings.TrimSpace(
		builder.String(),
	)
}

func buildCoursewareComicDefaultConfigs(
	panelCount int,
	layoutMode string,
) (string, string, error) {
	if panelCount < 4 ||
		panelCount > 8 ||
		!models.IsValidCWComicLayoutMode(
			layoutMode,
		) {
		return "", "",
			ErrCoursewareComicProjectInvalidRequest
	}

	pageLayout, err :=
		json.Marshal(
			map[string]interface{}{
				"version":     1,
				"panel_count": panelCount,
				"layout_mode": layoutMode,
				"auto_fit":    true,
			},
		)
	if err != nil {
		return "", "", err
	}

	interaction, err :=
		json.Marshal(
			map[string]interface{}{
				"version":               1,
				"editable_text":         true,
				"answer_mode":           models.CWComicAnswerModeClickReveal,
				"preserve_teacher_text": true,
			},
		)
	if err != nil {
		return "", "", err
	}

	return string(pageLayout),
		string(interaction),
		nil
}

func validateCoursewareComicEditedRelations(
	currentPanel *models.CoursewareComicPanel,
	allPanels []*models.CoursewareComicPanel,
	relations []models.CoursewareImageRelationSpec,
) error {
	if currentPanel == nil {
		return ErrCoursewareComicPromptInvalid
	}

	panelNoByKey := make(map[string]int)

	for _, panel := range allPanels {
		if panel != nil {
			panelNoByKey[panel.ImageKey] =
				panel.PanelNo
		}
	}

	if currentPanel.PanelNo == 1 {
		if len(relations) != 0 {
			return ErrCoursewareComicPromptInvalid
		}

		return nil
	}

	hasPreviousCharacterContinuity :=
		false

	for _, relation := range relations {
		targetPanelNo, exists :=
			panelNoByKey[relation.TargetImageKey]

		if !exists ||
			targetPanelNo >=
				currentPanel.PanelNo {
			return ErrCoursewareComicPromptInvalid
		}

		if targetPanelNo ==
			currentPanel.PanelNo-1 &&
			relation.RelationCode ==
				models.CWImageRelationContinue &&
			strings.Contains(
				relation.InheritMask,
				models.CWImageInheritCharacter,
			) {
			hasPreviousCharacterContinuity =
				true
		}
	}

	if !hasPreviousCharacterContinuity {
		return ErrCoursewareComicPromptInvalid
	}

	return nil
}

func normalizeCoursewareComicKPCodes(
	values []string,
) ([]string, error) {
	if len(values) == 0 ||
		len(values) >
			coursewareComicMaxKnowledgePoints {
		return nil,
			ErrCoursewareComicProjectKnowledgePointInvalid
	}

	result := make(
		[]string,
		0,
		len(values),
	)
	seen := make(map[string]bool)

	for _, value := range values {
		value = strings.TrimSpace(value)

		if value == "" ||
			len(value) > 128 {
			return nil,
				ErrCoursewareComicProjectKnowledgePointInvalid
		}

		if seen[value] {
			continue
		}

		seen[value] = true
		result = append(result, value)
	}

	if len(result) == 0 {
		return nil,
			ErrCoursewareComicProjectKnowledgePointInvalid
	}

	return result, nil
}

func parseCoursewareComicGradeNum(
	value string,
) int {
	value = strings.TrimSpace(value)

	switch {
	case strings.Contains(value, "高一"):
		return 10
	case strings.Contains(value, "高二"):
		return 11
	case strings.Contains(value, "高三"):
		return 12
	case strings.Contains(value, "初一"):
		return 7
	case strings.Contains(value, "初二"):
		return 8
	case strings.Contains(value, "初三"):
		return 9
	}

	chineseGrades := []struct {
		Text  string
		Value int
	}{
		{"十二年级", 12},
		{"十一年级", 11},
		{"十年级", 10},
		{"九年级", 9},
		{"八年级", 8},
		{"七年级", 7},
		{"六年级", 6},
		{"五年级", 5},
		{"四年级", 4},
		{"三年级", 3},
		{"二年级", 2},
		{"一年级", 1},
	}

	for _, grade := range chineseGrades {
		if strings.Contains(
			value,
			grade.Text,
		) {
			return grade.Value
		}
	}

	match :=
		coursewareComicArabicGradePattern.
			FindString(value)

	if match == "" {
		return 0
	}

	number, _ :=
		strconv.Atoi(match)

	if number < 1 ||
		number > 12 {
		return 0
	}

	return number
}

func normalizeCoursewareComicOptionalID(
	value *string,
) *string {
	if value == nil {
		return nil
	}

	normalized :=
		strings.TrimSpace(*value)

	if normalized == "" {
		return nil
	}

	return &normalized
}

func appendCoursewareComicHardConstraint(
	value string,
	constraint string,
) string {
	value = strings.TrimSpace(value)
	constraint =
		strings.TrimSpace(
			constraint,
		)

	if strings.Contains(
		value,
		constraint,
	) {
		return value
	}

	if value == "" {
		return constraint
	}

	return value + "；" + constraint
}
