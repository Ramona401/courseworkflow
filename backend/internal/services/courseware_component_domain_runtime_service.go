package services

// courseware_component_domain_runtime_service.go — 课件组件运行时域服务。
//
// 本文件负责：
//   - 公开匹配接口的可信目标域解析；
//   - 具体课件生成时的同域或common匹配；
//   - 新提交组件ID的整组严格校验；
//   - 历史matched_component_ids的兼容过滤。
//
// 具体课件运行时必须传入coursewares.education_domain快照，
// 不能使用管理员mixed上下文或客户端请求字段替代。

import (
	"context"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// resolveCWComponentMatchDomain 解析公开匹配接口使用的具体教学域。
func resolveCWComponentMatchDomain(
	actor *AssistantActorContext,
	requestedDomain string,
) (string, error) {
	readDomain, err :=
		resolveCWComponentReadDomain(actor)
	if err != nil {
		return "", err
	}

	if models.IsTeachingEducationDomain(
		readDomain,
	) {
		// 普通教学Actor不信任客户端education_domain。
		return readDomain, nil
	}

	requestedDomain =
		normalizeCWComponentDomain(
			requestedDomain,
		)

	if requestedDomain == "" {
		return "",
			ErrCWComponentEducationDomainRequired
	}

	if !models.IsTeachingEducationDomain(
		requestedDomain,
	) {
		return "",
			ErrCWComponentEducationDomainInvalid
	}

	return requestedDomain, nil
}

// MatchCWComponentsForActor 按可信Actor或mixed显式目标域匹配组件。
func MatchCWComponentsForActor(
	ctx context.Context,
	actor *AssistantActorContext,
	request *models.MatchCWComponentsDomainRequest,
) ([]*models.MatchedCWComponentResource, error) {
	if request == nil {
		return nil,
			ErrCWComponentRequestRequired
	}

	if request.ComponentType != "" &&
		!models.IsValidCWComponentType(
			strings.TrimSpace(
				request.ComponentType,
			),
		) {
		return nil,
			ErrCWComponentTypeInvalid
	}

	currentDomain, err :=
		resolveCWComponentMatchDomain(
			actor,
			request.EducationDomain,
		)
	if err != nil {
		return nil, err
	}

	matchRequest :=
		request.MatchCWComponentsRequest

	matchRequest.ComponentType =
		strings.TrimSpace(
			matchRequest.ComponentType,
		)
	matchRequest.SubjectScope =
		strings.TrimSpace(
			matchRequest.SubjectScope,
		)
	matchRequest.GradeScope =
		strings.TrimSpace(
			matchRequest.GradeScope,
		)
	matchRequest.VisualFormat =
		strings.TrimSpace(
			matchRequest.VisualFormat,
		)
	matchRequest.TechTag =
		strings.TrimSpace(
			matchRequest.TechTag,
		)

	items, err :=
		repository.
			MatchCWComponentsForEducationDomain(
				ctx,
				&matchRequest,
				currentDomain,
			)
	if err != nil {
		return nil, err
	}

	if items == nil {
		items =
			[]*models.MatchedCWComponentResource{}
	}

	return items, nil
}

// normalizeUniqueCWComponentIDs 清洗ID并保持首次出现顺序。
func normalizeUniqueCWComponentIDs(
	componentIDs []string,
) []string {
	seen := make(
		map[string]bool,
		len(componentIDs),
	)
	result := make(
		[]string,
		0,
		len(componentIDs),
	)

	for _, rawID := range componentIDs {
		id := strings.TrimSpace(rawID)

		if id == "" ||
			seen[id] {
			continue
		}

		seen[id] = true
		result = append(result, id)
	}

	return result
}

// ValidateCWComponentIDsForEducationDomain 严格验证一组组件ID。
//
// 任一ID不存在、停用、未审核、教育域非法或跨域时整组失败。
// 该方法用于新的直接ID提交以及即将继续写回的页面引用。
func ValidateCWComponentIDsForEducationDomain(
	ctx context.Context,
	componentIDs []string,
	coursewareDomain string,
) ([]string, error) {
	coursewareDomain =
		normalizeCWComponentDomain(
			coursewareDomain,
		)

	if !models.IsTeachingEducationDomain(
		coursewareDomain,
	) {
		return nil,
			ErrCWComponentEducationDomainInvalid
	}

	normalizedIDs :=
		normalizeUniqueCWComponentIDs(
			componentIDs,
		)

	if len(normalizedIDs) == 0 {
		return []string{}, nil
	}

	records, err :=
		repository.
			GetCWComponentAccessRecordsByIDs(
				ctx,
				normalizedIDs,
			)
	if err != nil {
		return nil, err
	}

	recordMap := make(
		map[string]*repository.CWComponentAccessRecord,
		len(records),
	)

	for _, record := range records {
		if record != nil {
			recordMap[record.ID] = record
		}
	}

	for _, componentID := range normalizedIDs {
		record, exists :=
			recordMap[componentID]

		if !exists ||
			!record.IsActive ||
			record.ReviewStatus !=
				models.CWCompReviewApproved ||
			!models.ResourceEducationDomainMatches(
				record.EducationDomain,
				coursewareDomain,
			) {
			return nil,
				ErrCWComponentSelectionInvalid
		}
	}

	return normalizedIDs, nil
}

// FilterHistoricalCWComponentIDsForEducationDomain 安全过滤历史页面引用。
//
// 历史JSON兼容规则：不存在、停用、未审核、非法域或异域ID静默过滤；
// 课件快照域非法时明确报错，禁止异常课件静默回退为K12。
func FilterHistoricalCWComponentIDsForEducationDomain(
	ctx context.Context,
	componentIDs []string,
	coursewareDomain string,
) ([]string, error) {
	coursewareDomain =
		normalizeCWComponentDomain(
			coursewareDomain,
		)

	if !models.IsTeachingEducationDomain(
		coursewareDomain,
	) {
		return nil,
			ErrCWComponentEducationDomainInvalid
	}

	normalizedIDs :=
		normalizeUniqueCWComponentIDs(
			componentIDs,
		)

	if len(normalizedIDs) == 0 {
		return []string{}, nil
	}

	records, err :=
		repository.
			GetCWComponentAccessRecordsByIDs(
				ctx,
				normalizedIDs,
			)
	if err != nil {
		return nil, err
	}

	allowed := make(
		map[string]bool,
		len(records),
	)

	for _, record := range records {
		if record == nil ||
			!record.IsActive ||
			record.ReviewStatus !=
				models.CWCompReviewApproved {
			continue
		}

		if models.ResourceEducationDomainMatches(
			record.EducationDomain,
			coursewareDomain,
		) {
			allowed[record.ID] = true
		}
	}

	result := make(
		[]string,
		0,
		len(normalizedIDs),
	)

	for _, componentID := range normalizedIDs {
		if allowed[componentID] {
			result = append(
				result,
				componentID,
			)
		}
	}

	return result, nil
}
