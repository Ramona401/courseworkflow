package services

// recipe_resolver.go — 对话模式配方自动解析器
//
// 【背景与确诊】
//   专家模式 StartForm 显式传 recipe_id，故 lp.RecipeID 非空，配方的「教案结构(lesson_structure)+
//   流程(stages_config)+学情/风格/学校要求」经 LoadStagePromptContextV2 全量注入，AI 拿到完整骨架。
//   对话模式 handleStart 调 startConversation 时【根本不传 recipe_id】，lp.RecipeID 恒为 nil，
//   于是上述三类配方上下文【全部不注入】，AI 只剩裸五阶段空骨架——这正是 yingjun 截图里
//   「九大板块不全、五环节塌成四环节、知识加油站过长、创想家工坊缺失」的真相。
//
// 【设计原则（Yuhan 四条原则的落地）】
//   原则1（老师个性化 > 默认 harness）：靠「把配方挂上去」自动实现——配方一旦挂载，其
//       教案结构/流程经 MergeStages + BuildLessonStructurePrompt 直接重写阶段骨架（第3.5层硬指令），
//       天然高于助手 overlay（第4层只能补风格）。无需任何新的 override 机制。
//   原则2+3+4（学校管理员/组长配方自然关联、可配默认、按学科）：本解析器三级 fail-open 解析。
//
// 【解析顺序】当对话模式 StartConversation 没拿到显式 recipe_id 时，按下列顺序解析一个配方ID：
//   第①级：学校管理员在 organizations.settings 配置的默认配方
//           - 优先 default_recipe_by_subject[subject]（按学科精确匹配）
//           - 否则 default_recipe_id（全学科兜底默认）
//   第②级：老师所属 group/school 范围下、按学科匹配的一份 active 配方
//           （取 scope_ref_id ∈ {老师的教研组ID集合 ∪ 学校ID} 且 subject 匹配的最新一条）
//   第③级：空串 —— 退回当前纯骨架行为（零风险，与改造前完全一致）
//
// 【fail-open 铁律】任何一步出错（DB 异常 / 配方已删 / settings 解析失败 / 学校归属查不到）
//   一律返回空串，绝不报错、绝不阻塞建会话。最坏情况 == 改造前的现状（纯骨架），不会更差。
//
// 【与专家模式的关系】本解析器仅在 req.RecipeID 为空时被调用；显式 recipe_id 永远最高优先，
//   专家模式一字不受影响。解析出的 recipeID 回填 req.RecipeID 后，复用专家模式【完全相同】的
//   下游（lp.RecipeID 赋值 + recipeStagesConfig + InitStagesForPlan + RecordRecipeUsage），
//   保证对话模式老师拿到与专家模式测试一致的好结果。
//
// 【类型注意·v203.1 修复】teaching_recipes.scope_ref_id 列在 PostgreSQL 里是 uuid 类型，
//   而本解析器收集的 scopeRefIDs 来自 GetSchoolIDByUserID（内部 ::text 转出的 string）与
//   GetUserTeachingGroups（.ID 为 string），是 text 切片。直接 `scope_ref_id = ANY($text[])`
//   会触发 `operator does not exist: uuid = text` 错误，被 fail-open 吞掉致第②级永远落空。
//   故 SQL 用 `scope_ref_id::text = ANY($3)` 列侧转 text 比较（与入参 text 切片吻合，
//   且比参数侧 ::uuid[] 更健壮——脏 ID 只是匹配不上而非整条报错）。

import (
	"context"
	"encoding/json"

	"tedna/internal/database"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// recipeAutoMountActiveStatus 配方有效状态值。
// models/recipe.go 全程用字面量 "active"/"archived"（无导出常量），此处本地定义一个常量，
// 避免在判断处散落魔法字符串，且与 recipe_repo.go 的软删过滤口径一致。
const recipeAutoMountActiveStatus = "active"

// organizationSettings 仅解析本解析器关心的两个键，其余键忽略。
// 对应 organizations.settings JSONB（同一份 settings 里还有 portal_modules 等其他键，互不影响）。
type organizationSettings struct {
	// DefaultRecipeID 学校级全学科兜底默认配方ID（学校管理员配置）。
	DefaultRecipeID string `json:"default_recipe_id"`
	// DefaultRecipeBySubject 学校级按学科默认配方映射，键为学科名，值为配方ID。
	DefaultRecipeBySubject map[string]string `json:"default_recipe_by_subject"`
}

// ResolveDefaultRecipe 为对话模式解析一个应自动挂载的配方ID。
//
// 入参：
//   - authorID：发起备课的老师用户ID（解析其所属学校/教研组用）
//   - subject ：本次备课学科（按学科匹配默认配方与共享配方）
//
// 返回：解析到的配方ID；任何失败/无匹配均返回空串（fail-open，调用方据空串退回纯骨架）。
//
// 本方法挂在 LessonPlanGenService 上，便于复用其已注入的依赖与日志器 lpGenLog。
func (s *LessonPlanGenService) ResolveDefaultRecipe(ctx context.Context, authorID string, subject string) string {
	if authorID == "" || subject == "" {
		return ""
	}

	// ============ 第①级：学校管理员配置的默认配方 ============
	// 先解析老师所属学校ID，再读该学校 organizations.settings 里的默认配方配置。
	schoolID, sErr := repository.GetSchoolIDByUserID(ctx, authorID)
	if sErr != nil {
		lpGenLog.Warn("配方自动解析:查询老师所属学校失败,跳过学校默认配方级", "author", authorID, "error", sErr)
	}
	if schoolID != "" {
		if recipeID := s.resolveSchoolDefaultRecipe(ctx, schoolID, subject); recipeID != "" {
			lpGenLog.Info("配方自动解析:命中学校管理员默认配方", "author", authorID, "subject", subject, "school_id", schoolID, "recipe_id", recipeID)
			return recipeID
		}
	}

	// ============ 第②级：group/school 范围下按学科匹配的 active 配方 ============
	// 收集老师可见的 scope_ref_id 集合：所属各教研组ID + 所属学校ID。
	scopeRefIDs := s.collectScopeRefIDs(ctx, authorID, schoolID)
	if len(scopeRefIDs) > 0 {
		if recipeID := resolveSharedRecipeBySubject(ctx, scopeRefIDs, subject); recipeID != "" {
			lpGenLog.Info("配方自动解析:命中group/school共享配方", "author", authorID, "subject", subject, "recipe_id", recipeID)
			return recipeID
		}
	}

	// ============ 第③级：空 —— 退回纯骨架（零风险） ============
	lpGenLog.Info("配方自动解析:无可挂载配方,退回纯骨架", "author", authorID, "subject", subject)
	return ""
}

// resolveSchoolDefaultRecipe 读取指定学校 settings 中的默认配方配置并校验有效性。
// 优先按学科精确匹配（default_recipe_by_subject[subject]），否则取全学科兜底（default_recipe_id）。
// 解析到的配方ID会校验：配方存在 + status=active + subject 与本次一致（防跨学科误挂）。
// 任何一步失败均返回空串。
func (s *LessonPlanGenService) resolveSchoolDefaultRecipe(ctx context.Context, schoolID string, subject string) string {
	org, err := repository.GetOrganizationByID(ctx, schoolID)
	if err != nil || org == nil {
		if err != nil {
			lpGenLog.Warn("配方自动解析:读取学校组织失败", "school_id", schoolID, "error", err)
		}
		return ""
	}
	if org.Settings == "" || org.Settings == "{}" {
		return ""
	}

	var st organizationSettings
	if err := json.Unmarshal([]byte(org.Settings), &st); err != nil {
		// settings 不是合法 JSON 或结构不符——fail-open 跳过，不影响 portal_modules 等其他用途。
		lpGenLog.Warn("配方自动解析:学校settings解析失败,跳过默认配方", "school_id", schoolID, "error", err)
		return ""
	}

	// 先按学科精确匹配
	candidate := ""
	if st.DefaultRecipeBySubject != nil {
		if v, ok := st.DefaultRecipeBySubject[subject]; ok {
			candidate = v
		}
	}
	// 学科无配置则用全学科兜底默认
	if candidate == "" {
		candidate = st.DefaultRecipeID
	}
	if candidate == "" {
		return ""
	}

	// 校验候选配方：存在 + active + 学科匹配（防止管理员把别科配方误配过来）。
	return validateRecipeForAutoMount(ctx, candidate, subject)
}

// collectScopeRefIDs 收集老师可见的共享配方归属ID集合（去重）：
//   - 老师所属全部教研组的 group ID
//   - 老师所属学校的 school ID
//
// 任一查询失败仅 Warn 不中断，尽力收集；最终可能返回空切片（调用方据空跳过第②级）。
func (s *LessonPlanGenService) collectScopeRefIDs(ctx context.Context, authorID string, schoolID string) []string {
	seen := make(map[string]struct{})
	var ids []string

	addID := func(id string) {
		if id == "" {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}

	// 教研组ID集合
	groups, gErr := repository.GetUserTeachingGroups(ctx, authorID)
	if gErr != nil {
		lpGenLog.Warn("配方自动解析:查询老师教研组失败,跳过教研组共享配方", "author", authorID, "error", gErr)
	} else {
		for _, g := range groups {
			if g != nil {
				addID(g.ID)
			}
		}
	}

	// 学校ID
	addID(schoolID)

	return ids
}

// resolveSharedRecipeBySubject 在给定 scope_ref_id 集合内，查一份 group/school 范围、
// 指定学科、status=active 的配方ID（按更新时间倒序取最新一条）。
//
// 这是一个独立的轻量数据访问（不复用 recipe_repo_market.go 的 ListMarketRecipes——
// 那是"全网市场排行榜按综合分排序"语义，与"给定老师可见范围内取一份确定配方"语义不同）。
// 直接内聚在本文件，避免改动 recipe_repo.go 大文件，降低爆炸半径。
//
// scope 取值用 models.RecipeScopeGroup/School 常量构造为数组参数（scope = ANY($2)），
// 既避免 SQL 字面量拼接、又用上 models 常量保证与全局口径一致。
//
// ⚠ v203.1 类型修复：scope_ref_id 列是 uuid 类型，入参 scopeRefIDs 是 text 切片，
//
//	必须用 `scope_ref_id::text = ANY($3)` 列侧转 text 比较，否则 pgx 抛
//	`operator does not exist: uuid = text`，被下方 err 兜底吞掉致第②级恒落空。
//
// 任何 DB 错误（含 pgx.ErrNoRows 无匹配）返回空串（fail-open）。
func resolveSharedRecipeBySubject(ctx context.Context, scopeRefIDs []string, subject string) string {
	if len(scopeRefIDs) == 0 || subject == "" {
		return ""
	}

	// 仅取 group/school 共享配方（personal 配方不参与自动挂载——那是创建者私有的）。
	// scope = ANY($2)：scope 列是 varchar，与 text 数组天然可比，无需转换。
	// scope_ref_id::text = ANY($3)：scope_ref_id 列是 uuid，入参为 text 切片，列侧转 text 比较。
	// status / subject 均用参数。全程参数化，无任何 SQL 字符串拼接。
	scopeValues := []string{models.RecipeScopeGroup, models.RecipeScopeSchool}
	const query = `
                SELECT id
                FROM teaching_recipes
                WHERE status = $1
                  AND scope = ANY($2)
                  AND scope_ref_id::text = ANY($3)
                  AND subject = $4
                ORDER BY updated_at DESC
                LIMIT 1
        `
	var recipeID string
	err := database.DB.QueryRow(ctx, query,
		recipeAutoMountActiveStatus, // $1 status
		scopeValues,                 // $2 scope = ANY (varchar 列, text 数组)
		scopeRefIDs,                 // $3 scope_ref_id::text = ANY (uuid 列转 text 比 text 数组)
		subject,                     // $4 subject
	).Scan(&recipeID)
	if err != nil {
		// pgx.ErrNoRows（无匹配）也走这里——属正常情况，返回空串退回下一级，不必区分。
		return ""
	}
	return recipeID
}

// validateRecipeForAutoMount 校验一个候选配方ID是否可用于自动挂载：
//   - 配方存在（GetRecipeByID 不报错）
//   - status == active（未被软删）
//   - subject == 本次备课学科（防跨学科误挂，例如管理员把数学配方配到语文默认上）
//
// 全部通过返回该ID，否则返回空串（fail-open）。
func validateRecipeForAutoMount(ctx context.Context, recipeID string, subject string) string {
	recipe, err := repository.GetRecipeByID(ctx, recipeID)
	if err != nil {
		// 配方不存在或已删——管理员配的默认配方失效，静默退回下一级。
		lpGenLog.Warn("配方自动解析:学校默认配方无效(已删/不存在)", "recipe_id", recipeID, "error", err)
		return ""
	}
	if recipe == nil {
		return ""
	}
	if recipe.Status != recipeAutoMountActiveStatus {
		return ""
	}
	// 学科不匹配则不挂（避免把别科配方注入当前学科的备课，反而带偏）。
	if recipe.Subject != "" && subject != "" && recipe.Subject != subject {
		lpGenLog.Info("配方自动解析:学校默认配方学科不匹配,跳过", "recipe_id", recipeID, "recipe_subject", recipe.Subject, "want_subject", subject)
		return ""
	}
	return recipeID
}
