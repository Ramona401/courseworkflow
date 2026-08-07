package services

// courseware_component_domain_policy.go — 课件组件教育域纯策略。
//
// 本文件只包含不访问数据库的确定性规则：
//   - 可信Actor读取域解析；
//   - mixed管理员创建目标域解析；
//   - mixed管理列表筛选；
//   - 组件修改权限；
//   - 写入字段与范围规范化。
//
// common治理规则：
//   - k12、vocational、adult具体教学上下文只能读取和使用common，不能修改；
//   - mixed管理上下文可以受控创建、审核、更新和删除common；
//   - 普通用户和客户端参数不能把自己提升为mixed管理上下文；
//   - mixed仍然只表示管理Actor，绝不能写入资源education_domain。

import (
	"errors"
	"strings"

	"tedna/internal/models"
)

var (
	ErrCWComponentRequestRequired = errors.New(
		"课件组件请求不能为空",
	)

	ErrCWComponentNameRequired = errors.New(
		"课件组件名称不能为空",
	)

	ErrCWComponentTypeRequired = errors.New(
		"课件组件类型不能为空",
	)

	ErrCWComponentTypeInvalid = errors.New(
		"课件组件类型无效",
	)

	ErrCWComponentCodeRequired = errors.New(
		"课件组件代码内容不能为空",
	)

	ErrCWComponentReviewInvalid = errors.New(
		"课件组件审核状态无效",
	)

	ErrCWComponentEducationDomainRequired = errors.New(
		"请明确选择课件组件所属教育域",
	)

	ErrCWComponentEducationDomainInvalid = errors.New(
		"课件组件教育域无效",
	)

	ErrCWComponentEducationDomainForbidden = errors.New(
		"无权操作该教育域的课件组件",
	)

	ErrCWComponentNotFound = errors.New(
		"课件组件不存在",
	)

	ErrCWComponentSelectionInvalid = errors.New(
		"页面引用的课件组件不存在、不可用或不适用于当前课件",
	)
)

// normalizeCWComponentDomain 只规范大小写和首尾空白。
//
// 本函数绝不把空值或非法值回退成K12。
func normalizeCWComponentDomain(
	domain string,
) string {
	return strings.ToLower(
		strings.TrimSpace(domain),
	)
}

// resolveCWComponentReadDomain 解析组件管理接口使用的可信当前域。
//
// 允许：
//   - k12、vocational、adult具体教学域；
//   - admin、region_admin、district_inspector的mixed管理上下文。
//
// common、空值、非法值及普通角色伪造的mixed均拒绝。
func resolveCWComponentReadDomain(
	actor *AssistantActorContext,
) (string, error) {
	if actor == nil ||
		strings.TrimSpace(actor.UserID) == "" {
		return "",
			ErrCWComponentEducationDomainForbidden
	}

	domain := normalizeCWComponentDomain(
		actor.EducationDomain,
	)

	if models.IsTeachingEducationDomain(
		domain,
	) {
		return domain, nil
	}

	if domain == models.EducationDomainMixed &&
		isCoursewareMixedReadManagementRole(
			actor.Role,
		) {
		return domain, nil
	}

	return "",
		ErrCWComponentEducationDomainForbidden
}

// resolveCWComponentCreationDomain 决定组件创建时最终写入的资源域。
//
// 当前组件写接口保持admin专属：
//   - mixed admin必须显式选择k12/vocational/adult/common；
//   - 具体教学域admin始终使用其可信域，忽略客户端伪造值；
//   - common只允许mixed admin显式创建；
//   - mixed和非法值绝不能写入资源。
func resolveCWComponentCreationDomain(
	actor *AssistantActorContext,
	requestedDomain string,
) (string, error) {
	if actor == nil ||
		actor.Role != models.RoleAdmin {
		return "",
			ErrCWComponentEducationDomainForbidden
	}

	currentDomain, err :=
		resolveCWComponentReadDomain(actor)
	if err != nil {
		return "", err
	}

	if models.IsTeachingEducationDomain(
		currentDomain,
	) {
		return currentDomain, nil
	}

	requestedDomain =
		normalizeCWComponentDomain(
			requestedDomain,
		)

	if requestedDomain == "" {
		return "",
			ErrCWComponentEducationDomainRequired
	}

	if !models.IsResourceEducationDomain(
		requestedDomain,
	) {
		return "",
			ErrCWComponentEducationDomainInvalid
	}

	return requestedDomain, nil
}

// resolveCWComponentListTarget 解析mixed管理列表的精确筛选域。
//
// 普通教学Actor的客户端筛选值一律忽略。
// mixed管理Actor可不筛选，也可筛选四种合法资源域。
func resolveCWComponentListTarget(
	readDomain string,
	requestedDomain string,
) (string, error) {
	if readDomain != models.EducationDomainMixed {
		return "", nil
	}

	requestedDomain =
		normalizeCWComponentDomain(
			requestedDomain,
		)

	if requestedDomain == "" {
		return "", nil
	}

	if !models.IsResourceEducationDomain(
		requestedDomain,
	) {
		return "",
			ErrCWComponentEducationDomainInvalid
	}

	return requestedDomain, nil
}

// cwComponentCanMutate 判断可信管理域能否修改目标资源。
//
// 规则：
//   - mixed管理上下文可治理k12/vocational/adult/common合法资源；
//   - 具体教学域只能治理完全同域资源；
//   - 具体教学域不能修改common；
//   - mixed资源、空值和其它非法资源域永远不可修改。
func cwComponentCanMutate(
	currentDomain string,
	resourceDomain string,
) bool {
	currentDomain =
		normalizeCWComponentDomain(
			currentDomain,
		)

	resourceDomain =
		normalizeCWComponentDomain(
			resourceDomain,
		)

	if !models.IsResourceEducationDomain(
		resourceDomain,
	) {
		return false
	}

	if currentDomain == models.EducationDomainMixed {
		return true
	}

	if !models.IsTeachingEducationDomain(
		currentDomain,
	) {
		return false
	}

	return resourceDomain == currentDomain
}

// validateCWComponentWriteFields 校验组件内容字段。
func validateCWComponentWriteFields(
	name string,
	componentType string,
	codeContent string,
	reviewStatus string,
) error {
	if strings.TrimSpace(name) == "" {
		return ErrCWComponentNameRequired
	}

	componentType = strings.TrimSpace(
		componentType,
	)

	if componentType == "" {
		return ErrCWComponentTypeRequired
	}

	if !models.IsValidCWComponentType(
		componentType,
	) {
		return ErrCWComponentTypeInvalid
	}

	if strings.TrimSpace(codeContent) == "" {
		return ErrCWComponentCodeRequired
	}

	if reviewStatus != "" &&
		reviewStatus != models.CWCompReviewDraft &&
		reviewStatus != models.CWCompReviewApproved &&
		reviewStatus != models.CWCompReviewArchived {
		return ErrCWComponentReviewInvalid
	}

	return nil
}

// normalizeCWComponentScope 规范化学科和学习层级范围。
//
// 空值表示不限范围，统一写为既有数据库语义ALL。
func normalizeCWComponentScope(
	value string,
) string {
	value = strings.TrimSpace(value)

	if value == "" {
		return "ALL"
	}

	return value
}
