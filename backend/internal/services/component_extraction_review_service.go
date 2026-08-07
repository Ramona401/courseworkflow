package services

// component_extraction_review_service.go — 组件萃取队列与审核。
//
// 教育域规则：
//   - 普通Actor只能查看和确认完全同域萃取；
//   - mixed管理Actor可以处理三个具体教学域；
//   - common不能作为自动萃取来源教案域；
//   - 来源教案缺失、组件缺失、异域组件和脏关联不进入队列；
//   - 无权访问与记录不存在统一映射为ErrExtractionNotFound。
//
// 所有HTTP调用必须使用带Actor的方法。
// 本文件不再保留无Actor兼容入口。

import (
	"context"
	"errors"
	"strings"
	"time"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// extractionActorCanAccessDomain 判断Actor是否能处理目标教案域。
func extractionActorCanAccessDomain(
	actorDomain string,
	lessonDomain string,
) bool {
	actorDomain = strings.ToLower(
		strings.TrimSpace(actorDomain),
	)

	lessonDomain = strings.ToLower(
		strings.TrimSpace(lessonDomain),
	)

	if !models.IsTeachingEducationDomain(
		lessonDomain,
	) {
		return false
	}

	if actorDomain == models.EducationDomainMixed {
		return true
	}

	return models.IsTeachingEducationDomain(actorDomain) &&
		actorDomain == lessonDomain
}

// ListPendingExtractionItemsForActor 获取Actor域可见的待审萃取。
func (s *ComponentService) ListPendingExtractionItemsForActor(
	ctx context.Context,
	actor *AssistantActorContext,
	limit int,
) (*models.ExtractionListResponse, error) {
	currentDomain, err :=
		ResolveComponentReadDomain(actor)
	if err != nil {
		return nil, err
	}

	records, err :=
		repository.ListPendingExtractionsForEducationDomain(
			ctx,
			currentDomain,
			limit,
		)
	if err != nil {
		return nil, err
	}

	items := make(
		[]*models.ExtractionListItem,
		0,
		len(records),
	)

	for _, record := range records {
		if record == nil {
			continue
		}

		if !extractionActorCanAccessDomain(
			currentDomain,
			record.EducationDomain,
		) {
			continue
		}

		libraryName :=
			models.LibraryTypeNameMap[record.ExtractionType]

		item := &models.ExtractionListItem{
			ID:             record.ID,
			SourceType:     record.SourceType,
			SourceContent:  record.SourceContent,
			ExtractionType: record.ExtractionType,
			LibraryName:    libraryName,
			Status:         record.Status,
			PlanTitle:      record.PlanTitle,
			CreatedByName:  record.CreatedByName,
			CreatedAt:      record.CreatedAt.Format(time.RFC3339),
		}

		items = append(
			items,
			item,
		)
	}

	return &models.ExtractionListResponse{
		Extractions: items,
		Total:       len(items),
	}, nil
}

// ConfirmExtractionByIDForActor 确认或拒绝Actor可管理的萃取。
//
// Repository会在同一事务中更新：
//   - component_extractions.status；
//   - lesson_plan_components.review_status。
func (s *ComponentService) ConfirmExtractionByIDForActor(
	ctx context.Context,
	actor *AssistantActorContext,
	extractionID string,
	decision string,
) error {
	if decision != "confirmed" &&
		decision != "rejected" {
		return ErrExtractionDecisionInvalid
	}

	currentDomain, err :=
		ResolveComponentReadDomain(actor)
	if err != nil {
		return err
	}

	err = repository.ConfirmExtractionForEducationDomain(
		ctx,
		strings.TrimSpace(extractionID),
		currentDomain,
		actor.UserID,
		decision,
	)

	if errors.Is(
		err,
		repository.ErrExtractionNotFound,
	) {
		return ErrExtractionNotFound
	}

	return err
}
