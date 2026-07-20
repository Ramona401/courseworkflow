package services

// recipe_resolver.go — 对话模式配方自动解析器
//
// 【背景与确诊】
// 专家模式StartForm显式传recipe_id，配方的教案结构、阶段流程、
// 学情、教学风格和学校要求会通过既有阶段装配链完整注入。
//
// 对话模式没有显式recipe_id时，需要按学校默认和老师所属范围
// 自动解析一份配方，否则会退回纯系统阶段骨架。
//
// 【自动解析顺序】
// 当StartConversation的recipe_mode为auto时，按以下顺序解析：
//
//  1. 学校管理员在organizations.settings中配置的默认配方：
//     - 优先default_recipe_by_subject[subject]；
//     - 其次default_recipe_id全学科默认。
//  2. 老师所属教研组或学校范围内，同学科、同具体年级的最新active配方。
//  3. 没有合法配方时返回空串，调用方退回纯系统骨架。
//
// 【教育域规则】
// 自动解析发生在lesson_plan创建之前，此时尚无lesson_plan.education_domain快照。
// 因此本文件通过BuildActorFromClaims解析作者当前确定性教学Actor：
//
//   - k12教学上下文只能自动使用k12或common配方；
//   - vocational教学上下文只能自动使用vocational或common配方；
//   - adult教学上下文只能自动使用adult或common配方；
//   - mixed只用于跨域管理，不能直接作为具体教学运行域；
//   - common、空值和非法当前域同样不能作为教学运行域。
//
// 教案创建完成后，数据库会写入lesson_plan.education_domain快照。
// 后续已有教案运行由strict_resource_match.go使用该快照再次复核，
// 因而形成“创建前按作者当前教学域过滤、创建后按教案快照域运行”的双层防线。
//
// 【fail-open业务退化与fail-closed授权】
// 数据库异常、组织设置异常、配方已删除或没有匹配时，返回空串退回纯骨架，
// 不阻断老师创建会话；但教育域不确定时绝不猜测或回退k12，
// 而是严格不自动挂载任何配方。
//
// 【类型注意】
// teaching_recipes.scope_ref_id是uuid，scopeRefIDs是字符串切片。
// SQL必须使用scope_ref_id::text = ANY($3)，避免uuid=text操作符错误。

import (
	"context"
	"encoding/json"
	"strings"

	"tedna/internal/database"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// recipeAutoMountActiveStatus 配方有效状态值。
//
// models/recipe.go目前没有导出的active状态常量，
// 因而在本自动解析模块内保留单一常量，避免散落魔法字符串。
const recipeAutoMountActiveStatus = "active"

// organizationSettings 只解析自动配方功能关心的两个组织设置键。
//
// organizations.settings中可能还包含门户模块等其它配置，
// JSON反序列化会自动忽略本结构未声明的字段。
type organizationSettings struct {
	// DefaultRecipeID 是学校级全学科默认配方ID。
	DefaultRecipeID string `json:"default_recipe_id"`

	// DefaultRecipeBySubject 是“学科名称→默认配方ID”的映射。
	DefaultRecipeBySubject map[string]string `json:"default_recipe_by_subject"`
}

// ResolveDefaultRecipe 为对话模式解析一份应自动挂载的配方ID。
//
// 公开签名保持不变，避免扩大调用链修改范围。
//
// 自动解析发生在教案创建前，因此本函数先读取作者角色，再通过
// BuildActorFromClaims解析作者当前的学校、教研组和具体教学教育域。
//
// 任一条件不确定时返回空串，让调用方退回纯骨架：
//   - 作者不存在或角色读取失败；
//   - Actor无法构造；
//   - 当前教育域不是k12、vocational或adult；
//   - 没有合法学校默认或共享配方。
func (s *LessonPlanGenService) ResolveDefaultRecipe(
	ctx context.Context,
	authorID string,
	subject string,
	grade string,
) string {
	authorID = strings.TrimSpace(authorID)
	subject = strings.TrimSpace(subject)
	grade = strings.TrimSpace(grade)

	if authorID == "" ||
		subject == "" ||
		grade == "" {
		return ""
	}

	author, err := repository.FindUserByID(
		ctx,
		authorID,
	)
	if err != nil || author == nil {
		if err != nil {
			lpGenLog.Warn(
				"配方自动解析:读取作者身份失败,退回纯骨架",
				"author", authorID,
				"error", err,
			)
		}
		return ""
	}

	actor := BuildActorFromClaims(
		ctx,
		authorID,
		author.Role,
	)
	if actor == nil {
		lpGenLog.Warn(
			"配方自动解析:无法构造作者教学Actor,退回纯骨架",
			"author", authorID,
		)
		return ""
	}

	currentEducationDomain := strings.ToLower(
		strings.TrimSpace(actor.EducationDomain),
	)

	// mixed只用于跨域管理，不能作为具体教学运行域。
	// common只允许作为资源域，也不能作为当前教学域。
	// 空值和非法值不得通过NormalizeEducationDomain回退成k12。
	if !models.IsTeachingEducationDomain(
		currentEducationDomain,
	) {
		lpGenLog.Info(
			"配方自动解析:作者没有确定的具体教学域,不自动挂载配方",
			"author", authorID,
			"education_domain", currentEducationDomain,
		)
		return ""
	}

	schoolID := strings.TrimSpace(actor.SchoolID)

	if schoolID != "" {
		if recipeID := s.resolveSchoolDefaultRecipe(
			ctx,
			schoolID,
			subject,
			grade,
			currentEducationDomain,
		); recipeID != "" {
			lpGenLog.Info(
				"配方自动解析:命中学校管理员默认配方",
				"author", authorID,
				"subject", subject,
				"grade", grade,
				"education_domain", currentEducationDomain,
				"school_id", schoolID,
				"recipe_id", recipeID,
			)
			return recipeID
		}
	}

	scopeRefIDs := collectRecipeAutoMountScopeRefIDs(
		actor.MyGroupIDs,
		schoolID,
	)
	if len(scopeRefIDs) > 0 {
		if recipeID := resolveSharedRecipeBySubject(
			ctx,
			scopeRefIDs,
			subject,
			grade,
			currentEducationDomain,
		); recipeID != "" {
			lpGenLog.Info(
				"配方自动解析:命中group或school共享配方",
				"author", authorID,
				"subject", subject,
				"grade", grade,
				"education_domain", currentEducationDomain,
				"recipe_id", recipeID,
			)
			return recipeID
		}
	}

	lpGenLog.Info(
		"配方自动解析:没有同域同学科同具体年级配方,退回纯骨架",
		"author", authorID,
		"subject", subject,
		"grade", grade,
		"education_domain", currentEducationDomain,
	)

	return ""
}

// resolveSchoolDefaultRecipe 读取指定学校settings中的默认配方并校验。
//
// 优先读取default_recipe_by_subject[subject]，没有配置时再读取
// default_recipe_id全学科默认值。
//
// 候选配方必须同时满足：
//   - 配方存在；
//   - status为active；
//   - 学科和具体年级严格匹配；
//   - 配方教育域等于当前教学域或为common。
func (s *LessonPlanGenService) resolveSchoolDefaultRecipe(
	ctx context.Context,
	schoolID string,
	subject string,
	grade string,
	currentEducationDomain string,
) string {
	org, err := repository.GetOrganizationByID(
		ctx,
		schoolID,
	)
	if err != nil || org == nil {
		if err != nil {
			lpGenLog.Warn(
				"配方自动解析:读取学校组织失败",
				"school_id", schoolID,
				"error", err,
			)
		}
		return ""
	}

	if strings.TrimSpace(org.Settings) == "" ||
		strings.TrimSpace(org.Settings) == "{}" {
		return ""
	}

	var settings organizationSettings
	if err := json.Unmarshal(
		[]byte(org.Settings),
		&settings,
	); err != nil {
		lpGenLog.Warn(
			"配方自动解析:学校settings解析失败,跳过默认配方",
			"school_id", schoolID,
			"error", err,
		)
		return ""
	}

	candidate := ""

	if settings.DefaultRecipeBySubject != nil {
		if value, ok :=
			settings.DefaultRecipeBySubject[subject]; ok {
			candidate = strings.TrimSpace(value)
		}
	}

	if candidate == "" {
		candidate = strings.TrimSpace(
			settings.DefaultRecipeID,
		)
	}

	if candidate == "" {
		return ""
	}

	return validateRecipeForAutoMount(
		ctx,
		candidate,
		subject,
		grade,
		currentEducationDomain,
	)
}

// collectRecipeAutoMountScopeRefIDs 收集自动共享配方查询所需的归属ID。
//
// 来源包括：
//   - BuildActorFromClaims解析出的全部所属教研组ID；
//   - Actor当前确定性教学学校ID。
//
// 本函数只做去空、清洗和去重，不重新访问数据库，
// 避免Actor解析后再次形成另一套学校或教研组口径。
func collectRecipeAutoMountScopeRefIDs(
	groupIDs []string,
	schoolID string,
) []string {
	seen := make(map[string]struct{})
	ids := make([]string, 0, len(groupIDs)+1)

	addID := func(id string) {
		id = strings.TrimSpace(id)
		if id == "" {
			return
		}

		if _, exists := seen[id]; exists {
			return
		}

		seen[id] = struct{}{}
		ids = append(ids, id)
	}

	for _, groupID := range groupIDs {
		addID(groupID)
	}

	addID(schoolID)

	return ids
}

// resolveSharedRecipeBySubject 在老师可见的group和school范围内，
// 查询一份同学科、同具体年级、同教育域或common的最新active配方。
//
// SQL执行第一层过滤：
//   - status=active；
//   - scope为group或school；
//   - scope_ref_id属于Actor可见归属；
//   - subject严格一致；
//   - education_domain为当前具体教学域或common。
//
// Go扫描后再次调用ResourceEducationDomainMatches复核，
// 防止未来SQL调整形成跨教育域旁路。
//
// 任意数据库错误或没有命中时返回空串，调用方退回纯骨架。
func resolveSharedRecipeBySubject(
	ctx context.Context,
	scopeRefIDs []string,
	subject string,
	grade string,
	currentEducationDomain string,
) string {
	subject = strings.TrimSpace(subject)
	grade = strings.TrimSpace(grade)

	normalizedDomain := strings.ToLower(
		strings.TrimSpace(currentEducationDomain),
	)

	if len(scopeRefIDs) == 0 ||
		subject == "" ||
		grade == "" ||
		!models.IsTeachingEducationDomain(
			normalizedDomain,
		) {
		return ""
	}

	scopeValues := []string{
		models.RecipeScopeGroup,
		models.RecipeScopeSchool,
	}

	const query = `
		SELECT id,
		       grade_range,
		       education_domain
		FROM teaching_recipes
		WHERE status = $1
		  AND scope = ANY($2)
		  AND scope_ref_id::text = ANY($3)
		  AND subject = $4
		  AND (
		        education_domain = $5
		        OR education_domain = $6
		  )
		ORDER BY updated_at DESC
	`

	rows, err := database.DB.Query(
		ctx,
		query,
		recipeAutoMountActiveStatus,
		scopeValues,
		scopeRefIDs,
		subject,
		normalizedDomain,
		models.EducationDomainCommon,
	)
	if err != nil {
		lpGenLog.Warn(
			"配方自动解析:查询共享配方失败,退回纯骨架",
			"subject", subject,
			"grade", grade,
			"education_domain", normalizedDomain,
			"error", err,
		)
		return ""
	}
	defer rows.Close()

	for rows.Next() {
		var recipeID string
		var recipeGrade string
		var recipeEducationDomain string

		if err := rows.Scan(
			&recipeID,
			&recipeGrade,
			&recipeEducationDomain,
		); err != nil {
			lpGenLog.Warn(
				"配方自动解析:扫描共享配方失败,退回纯骨架",
				"error", err,
			)
			return ""
		}

		// SQL过滤后的第二层确定性复核。
		if !models.ResourceEducationDomainMatches(
			recipeEducationDomain,
			normalizedDomain,
		) {
			continue
		}

		if utils.IsStrictGradeMatch(
			recipeGrade,
			grade,
		) {
			return strings.TrimSpace(recipeID)
		}
	}

	if err := rows.Err(); err != nil {
		lpGenLog.Warn(
			"配方自动解析:遍历共享配方失败,退回纯骨架",
			"error", err,
		)
	}

	return ""
}

// validateRecipeForAutoMount 校验学校设置指定的候选配方。
//
// 学校默认配方按ID读取，必须执行以下第二层校验：
//   - 配方存在且active；
//   - 配方学科和具体年级与本次教案严格一致；
//   - 配方教育域与当前具体教学域兼容。
//
// 任一条件不满足均返回空串，不自动挂载。
func validateRecipeForAutoMount(
	ctx context.Context,
	recipeID string,
	subject string,
	grade string,
	currentEducationDomain string,
) string {
	recipeID = strings.TrimSpace(recipeID)
	subject = strings.TrimSpace(subject)
	grade = strings.TrimSpace(grade)

	normalizedDomain := strings.ToLower(
		strings.TrimSpace(currentEducationDomain),
	)

	if recipeID == "" ||
		subject == "" ||
		grade == "" ||
		!models.IsTeachingEducationDomain(
			normalizedDomain,
		) {
		return ""
	}

	recipe, err := repository.GetRecipeByID(
		ctx,
		recipeID,
	)
	if err != nil {
		lpGenLog.Warn(
			"配方自动解析:学校默认配方无效或不存在",
			"recipe_id", recipeID,
			"error", err,
		)
		return ""
	}

	if recipe == nil ||
		recipe.Status != recipeAutoMountActiveStatus {
		return ""
	}

	if !models.ResourceEducationDomainMatches(
		recipe.EducationDomain,
		normalizedDomain,
	) {
		lpGenLog.Info(
			"配方自动解析:学校默认配方教育域不匹配,跳过",
			"recipe_id", recipeID,
			"recipe_education_domain", recipe.EducationDomain,
			"current_education_domain", normalizedDomain,
		)
		return ""
	}

	if !strictRecipeMatchesEntity(
		recipe,
		subject,
		grade,
	) {
		lpGenLog.Info(
			"配方自动解析:学校默认配方与当前学科或具体年级不匹配,跳过",
			"recipe_id", recipeID,
			"recipe_subject", recipe.Subject,
			"recipe_grade", recipe.GradeRange,
			"want_subject", subject,
			"want_grade", grade,
		)
		return ""
	}

	return recipeID
}
