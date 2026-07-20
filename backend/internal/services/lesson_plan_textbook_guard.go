package services

// lesson_plan_textbook_guard.go — 教案创建、导入与运行时课本上下文统一硬闸
//
// 上下文15正式规则：
//   - 课本能力只允许K12具体教学域使用；
//   - 对话创建与已有教案导入必须在任何教案INSERT前校验课本ID；
//   - 运行时必须依据lesson_plans.education_domain创建快照重新校验；
//   - 所有页面必须存在、为active、无重复，并与教案学科和年级完全一致；
//   - 数据库错误必须向上返回，不能伪装成“没有课本”；
//   - 课本表不新增education_domain列，K12边界由教案快照域和可信调用参数共同保证。
//
// 本文件只承载创建、导入和运行时三条链路共用的确定性规则，
// 不负责HTTP状态码映射，也不直接修改教案关联列。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

const (
	// lessonPlanTextbookMaxPages 单份教案最多关联20张课本页面。
	//
	// OCR原文会进入每轮系统提示词，必须限制数量，防止上下文无界增长。
	lessonPlanTextbookMaxPages = 20
)

// validatedLessonPlanTextbookSelection 是课本选择校验后的内部结果。
//
// PageIDs保持老师提交时的原始顺序；
// PagesByID供运行时按照该顺序重新装配上下文。
type validatedLessonPlanTextbookSelection struct {
	PageIDs   []string
	PagesByID map[string]*models.TextbookPage
}

// validateLessonPlanTextbookSelection 完成教案课本选择的统一校验。
//
// 空数组表示没有关联课本：三个具体教学域都可以正常创建和运行。
// 只有请求确实携带课本ID时，才要求教育域严格为k12。
func validateLessonPlanTextbookSelection(
	ctx context.Context,
	educationDomain string,
	subject string,
	grade string,
	pageIDs []string,
) (*validatedLessonPlanTextbookSelection, error) {
	normalizedIDs, err := normalizeLessonPlanTextbookPageIDs(pageIDs)
	if err != nil {
		return nil, err
	}

	if len(normalizedIDs) == 0 {
		return &validatedLessonPlanTextbookSelection{
			PageIDs:   []string{},
			PagesByID: map[string]*models.TextbookPage{},
		}, nil
	}

	if len(normalizedIDs) > lessonPlanTextbookMaxPages {
		return nil, ErrLPTextbookSelectionInvalid
	}

	domain := strings.ToLower(
		strings.TrimSpace(educationDomain),
	)
	if domain != models.EducationDomainK12 {
		return nil, ErrLPTextbookEducationDomainDenied
	}

	normalizedSubject := strings.TrimSpace(subject)
	normalizedGrade := strings.TrimSpace(grade)
	if normalizedSubject == "" ||
		normalizedGrade == "" {
		return nil, ErrLPTextbookSelectionInvalid
	}

	pages, err :=
		repository.GetTextbookPagesByIDsForEducationDomain(
			ctx,
			normalizedIDs,
			domain,
		)
	if err != nil {
		if errors.Is(
			err,
			repository.ErrTextbookEducationDomainUnsupported,
		) {
			return nil, ErrLPTextbookEducationDomainDenied
		}

		// 数据库错误必须保留错误链并向上返回，
		// 不能被转换成空数组或伪装成“无课本”。
		return nil, fmt.Errorf(
			"查询课本页面失败: %w",
			err,
		)
	}

	// Repository只返回active页面。
	//
	// 返回数量与唯一ID数量不一致，表示至少有一个ID：
	//   - 不存在；
	//   - 已归档；
	//   - 是伪造值。
	//
	// 任何一种情况都整体拒绝，禁止部分挂载。
	if len(pages) != len(normalizedIDs) {
		return nil, ErrLPTextbookSelectionInvalid
	}

	pagesByID := make(
		map[string]*models.TextbookPage,
		len(pages),
	)

	for _, page := range pages {
		if page == nil {
			return nil, ErrLPTextbookSelectionInvalid
		}

		pageID := strings.TrimSpace(page.ID)
		if pageID == "" {
			return nil, ErrLPTextbookSelectionInvalid
		}

		if _, exists := pagesByID[pageID]; exists {
			return nil, ErrLPTextbookSelectionInvalid
		}

		if strings.TrimSpace(page.Status) != "active" {
			return nil, ErrLPTextbookSelectionInvalid
		}

		if strings.TrimSpace(page.Subject) !=
			normalizedSubject {
			return nil, ErrLPTextbookSelectionInvalid
		}

		if strings.TrimSpace(page.GradeRange) !=
			normalizedGrade {
			return nil, ErrLPTextbookSelectionInvalid
		}

		pagesByID[pageID] = page
	}

	// 再按原请求顺序逐项核对ID集合。
	//
	// 防止查询异常地返回了数量相同、但资源集合不同的数据。
	for _, pageID := range normalizedIDs {
		if pagesByID[pageID] == nil {
			return nil, ErrLPTextbookSelectionInvalid
		}
	}

	return &validatedLessonPlanTextbookSelection{
		PageIDs:   normalizedIDs,
		PagesByID: pagesByID,
	}, nil
}

// ValidateLessonPlanTextbookSelection 为创建和导入链提供公开校验入口。
//
// 返回值是清理、去重校验后的ID数组。
// 调用方必须使用该返回值落库，不能继续使用未经校验的请求原值。
func ValidateLessonPlanTextbookSelection(
	ctx context.Context,
	educationDomain string,
	subject string,
	grade string,
	pageIDs []string,
) ([]string, error) {
	selection, err :=
		validateLessonPlanTextbookSelection(
			ctx,
			educationDomain,
			subject,
			grade,
			pageIDs,
		)
	if err != nil {
		return nil, err
	}

	return selection.PageIDs, nil
}

// ValidateStartConversationTextbooks 校验并规范化开始备课请求中的课本ID。
func ValidateStartConversationTextbooks(
	ctx context.Context,
	educationDomain string,
	req *models.StartConversationRequest,
) error {
	if req == nil {
		return errors.New(
			"开始备课请求不能为空",
		)
	}

	pageIDs, err :=
		ValidateLessonPlanTextbookSelection(
			ctx,
			educationDomain,
			req.Subject,
			req.Grade,
			req.TextbookPageIDs,
		)
	if err != nil {
		return err
	}

	req.TextbookPageIDs = pageIDs
	return nil
}

// ValidateImportedLessonPlanTextbooks 校验并规范化已有教案导入请求中的课本ID。
func ValidateImportedLessonPlanTextbooks(
	ctx context.Context,
	educationDomain string,
	req *models.ImportExistingPlanRequest,
) error {
	if req == nil {
		return errors.New(
			"导入教案请求不能为空",
		)
	}

	pageIDs, err :=
		ValidateLessonPlanTextbookSelection(
			ctx,
			educationDomain,
			req.Subject,
			req.Grade,
			req.TextbookPageIDs,
		)
	if err != nil {
		return err
	}

	req.TextbookPageIDs = pageIDs
	return nil
}

// BuildLessonPlanTextbookContext 按教案正式数据库快照构建运行时课本上下文。
//
// 本函数是后续运行时注入链的唯一入口：
//   - 从lesson_plans.textbook_page_ids解析页面ID；
//   - 用lesson_plans.education_domain快照执行K12硬闸；
//   - 重新验证页面active、学科、年级和完整ID集合；
//   - 数据库错误直接返回给上层，终止本轮AI请求；
//   - 最终按照老师保存的ID顺序装配OCR原文。
func BuildLessonPlanTextbookContext(
	ctx context.Context,
	lessonPlan *models.LessonPlan,
) (string, error) {
	if lessonPlan == nil {
		return "", errors.New(
			"构建课本上下文失败：教案为空",
		)
	}

	rawPageIDs := strings.TrimSpace(
		lessonPlan.TextbookPageIDs,
	)
	if rawPageIDs == "" ||
		rawPageIDs == "[]" {
		return "", nil
	}

	var pageIDs []string
	if err := json.Unmarshal(
		[]byte(rawPageIDs),
		&pageIDs,
	); err != nil {
		return "", fmt.Errorf(
			"解析教案课本关联失败: %w",
			err,
		)
	}

	selection, err :=
		validateLessonPlanTextbookSelection(
			ctx,
			lessonPlan.EducationDomain,
			lessonPlan.Subject,
			lessonPlan.Grade,
			pageIDs,
		)
	if err != nil {
		return "", err
	}

	if len(selection.PageIDs) == 0 {
		return "", nil
	}

	var builder strings.Builder

	builder.WriteString(
		"\n== 课本原文参考 ==\n",
	)
	builder.WriteString(
		"以下是老师关联的课本真实内容，请严格参考课本原文进行教学设计：\n\n",
	)

	for index, pageID := range selection.PageIDs {
		page := selection.PagesByID[pageID]
		if page == nil {
			return "",
				ErrLPTextbookSelectionInvalid
		}

		builder.WriteString(
			fmt.Sprintf(
				"--- 课本第%d页（%s · %s）---\n",
				index+1,
				page.TextbookName,
				page.Chapter,
			),
		)

		if strings.TrimSpace(page.OCRText) != "" {
			builder.WriteString(page.OCRText)
			builder.WriteString("\n")
		} else {
			builder.WriteString(
				"[此页图片尚未识别文字，请提醒老师先进行AI识别]\n",
			)
		}

		builder.WriteString("\n")

		// 使用次数属于旁路统计。
		//
		// 统计失败不影响已经完成严格校验的本轮上下文。
		go func(id string) {
			_ = repository.IncrementTextbookUsage(
				context.Background(),
				id,
			)
		}(page.ID)
	}

	return builder.String(), nil
}
