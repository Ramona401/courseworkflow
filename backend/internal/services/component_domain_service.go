package services

// component_domain_service.go — 教案组件教育域访问控制底座。
//
// 本文件集中实现组件域授权的纯规则与直接ID严格校验，供组件管理、
// 阶段推荐、配方注入、聊天选中组件和上下文回执共同复用。
//
// 统一规则：
//   1. 普通教学Actor创建资源时强制使用可信Actor教育域，忽略前端伪造域；
//   2. mixed系统管理员创建时必须显式选择资源域；
//   3. common只允许系统管理员创建；
//   4. region_admin和district_inspector虽可跨域查看，但不能创建组件资源；
//   5. 具体教案运行只认lesson_plans.education_domain快照；
//   6. 新提交的直接ID整组严格验证，任一非法则整组失败；
//   7. 历史JSON引用只加载同域或common，异域内容静默过滤。

import (
	"context"
	"errors"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

var (
	// ErrComponentEducationDomainRequired mixed系统管理员创建组件时未显式选域。
	ErrComponentEducationDomainRequired = errors.New("请明确选择组件所属教育域")

	// ErrComponentEducationDomainInvalid 请求或教案快照不是合法资源运行域。
	ErrComponentEducationDomainInvalid = errors.New("组件教育域无效")

	// ErrComponentEducationDomainForbidden Actor无权在目标教育域创建或读取资源。
	ErrComponentEducationDomainForbidden = errors.New("无权操作该教育域的组件")

	// ErrComponentSelectionInvalid 新提交的直接ID整组校验失败。
	//
	// 对外使用统一错误，不区分“不存在”和“异域”，避免泄漏异域资源存在性。
	ErrComponentSelectionInvalid = errors.New("选择的组件不存在、不可用或不适用于当前教案")
)

// ResolveComponentCreationDomain 决定手工创建组件时最终落库的资源域。
//
// 普通教学Actor：
//   - 必须拥有k12/vocational/adult具体教学域；
//   - 最终始终使用Actor教育域；
//   - 忽略前端伪造的其它域，包括common。
//
// mixed系统管理员：
//   - 必须显式传入k12/vocational/adult/common；
//   - 可以创建common公共组件。
//
// 其它mixed管理角色：
//   - 可以跨域查看和治理已有数据；
//   - 不能通过普通组件创建接口生产新资源。
func ResolveComponentCreationDomain(
	actor *AssistantActorContext,
	requestedDomain string,
) (string, error) {
	if actor == nil ||
		strings.TrimSpace(actor.UserID) == "" {
		return "",
			ErrComponentEducationDomainForbidden
	}

	actorDomain := strings.ToLower(
		strings.TrimSpace(
			actor.EducationDomain,
		),
	)

	requestedDomain = strings.ToLower(
		strings.TrimSpace(
			requestedDomain,
		),
	)

	if actorDomain == models.EducationDomainMixed {
		if actor.Role != models.RoleAdmin {
			return "",
				ErrComponentEducationDomainForbidden
		}

		if requestedDomain == "" {
			return "",
				ErrComponentEducationDomainRequired
		}

		if !models.IsResourceEducationDomain(
			requestedDomain,
		) {
			return "",
				ErrComponentEducationDomainInvalid
		}

		return requestedDomain, nil
	}

	if !models.IsTeachingEducationDomain(
		actorDomain,
	) {
		return "",
			ErrComponentEducationDomainForbidden
	}

	// 普通教学Actor不相信前端education_domain。
	return actorDomain, nil
}

// ResolveComponentReadDomain 返回管理接口可使用的可信当前域。
//
// 具体教学Actor只能查看同域或common；
// mixed管理Actor可以查看全部合法资源域；
// 空值、common当前域和非法值全部fail-closed。
func ResolveComponentReadDomain(
	actor *AssistantActorContext,
) (string, error) {
	if actor == nil {
		return "",
			ErrComponentEducationDomainForbidden
	}

	domain := strings.ToLower(
		strings.TrimSpace(
			actor.EducationDomain,
		),
	)

	if models.IsTeachingEducationDomain(domain) {
		return domain, nil
	}

	if domain == models.EducationDomainMixed &&
		isAssistantMixedManagementRole(actor.Role) {
		return domain, nil
	}

	return "",
		ErrComponentEducationDomainForbidden
}

// NormalizeUniqueComponentIDs 清洗直接ID列表并保持首次出现顺序。
//
// 空字符串被忽略，重复ID只保留第一次。整组严格校验以清洗后的唯一ID数
// 为基准，避免重复ID造成错误的数量比较和重复统计。
func NormalizeUniqueComponentIDs(
	componentIDs []string,
) []string {
	if len(componentIDs) == 0 {
		return []string{}
	}

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
		if id == "" || seen[id] {
			continue
		}

		seen[id] = true
		result = append(result, id)
	}

	return result
}

// ValidateLessonComponentIDsForUse 严格验证一批新提交的直接组件ID。
//
// 必须同时满足：
//  1. lessonDomain是k12/vocational/adult具体教案快照域；
//  2. 所有ID都真实存在；
//  3. 所有组件都为active；
//  4. 所有组件都已经approved；
//  5. 组件教育域与教案域相同，或组件为common；
//  6. 传入allowedLibraryTypes时，组件类型必须属于当前阶段。
//
// 任一ID不满足即整组失败。调用方必须在写用户消息、阶段输出或current_stage
// 之前调用，保证失败请求没有部分业务副作用。
func ValidateLessonComponentIDsForUse(
	ctx context.Context,
	componentIDs []string,
	lessonDomain string,
	allowedLibraryTypes []string,
) ([]string, error) {
	lessonDomain = strings.ToLower(
		strings.TrimSpace(
			lessonDomain,
		),
	)

	if !models.IsTeachingEducationDomain(
		lessonDomain,
	) {
		return nil,
			ErrComponentEducationDomainInvalid
	}

	normalizedIDs := NormalizeUniqueComponentIDs(
		componentIDs,
	)
	if len(normalizedIDs) == 0 {
		return []string{}, nil
	}

	records, err :=
		repository.GetComponentAccessRecordsByIDs(
			ctx,
			normalizedIDs,
		)
	if err != nil {
		return nil, err
	}

	recordMap := make(
		map[string]*repository.ComponentAccessRecord,
		len(records),
	)

	for _, record := range records {
		if record == nil {
			continue
		}

		recordMap[record.ID] = record
	}

	allowedTypes := make(
		map[string]bool,
		len(allowedLibraryTypes),
	)
	for _, rawType := range allowedLibraryTypes {
		libraryType := strings.TrimSpace(rawType)
		if libraryType != "" {
			allowedTypes[libraryType] = true
		}
	}

	for _, componentID := range normalizedIDs {
		record, exists := recordMap[componentID]
		if !exists {
			return nil,
				ErrComponentSelectionInvalid
		}

		if record.Status != "active" ||
			record.ReviewStatus !=
				models.ComponentReviewApproved {
			return nil,
				ErrComponentSelectionInvalid
		}

		if !models.ResourceEducationDomainMatches(
			record.EducationDomain,
			lessonDomain,
		) {
			return nil,
				ErrComponentSelectionInvalid
		}

		if len(allowedTypes) > 0 &&
			!allowedTypes[record.LibraryType] {
			return nil,
				ErrComponentSelectionInvalid
		}
	}

	return normalizedIDs, nil
}

// LoadHistoricalLessonComponentGroups 安全加载历史组件JSON引用。
//
// 与新提交不同，历史配方和历史阶段输出必须兼容旧数据：
// 异域、失效或不存在的ID直接过滤，不修改历史JSON，也不泄漏被过滤资源。
// lessonDomain非法时明确报错，防止异常教案快照回退为K12。
func LoadHistoricalLessonComponentGroups(
	ctx context.Context,
	componentIDs []string,
	lessonDomain string,
) ([]*models.MatchedComponentGroup, error) {
	lessonDomain = strings.ToLower(
		strings.TrimSpace(
			lessonDomain,
		),
	)

	if !models.IsTeachingEducationDomain(
		lessonDomain,
	) {
		return nil,
			ErrComponentEducationDomainInvalid
	}

	normalizedIDs := NormalizeUniqueComponentIDs(
		componentIDs,
	)
	if len(normalizedIDs) == 0 {
		return []*models.MatchedComponentGroup{}, nil
	}

	return repository.GetComponentContentsForEducationDomain(
		ctx,
		normalizedIDs,
		lessonDomain,
	)
}
