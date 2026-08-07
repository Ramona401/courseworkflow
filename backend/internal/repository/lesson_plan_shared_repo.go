package repository

// lesson_plan_shared_repo.go — 共享教案列表安全查询
//
// 本文件从lesson_plan_repo.go拆出，避免核心Repository继续超过600行。
// HTTP列表统一通过ListLessonPlansWithSharedAccess执行：
//
//   - 管辖/本人分支继续服务“我的教案”和正式数据范围列表；
//   - 共享分支必须同时满足共享状态、非personal可见范围、组织作者白名单、
//     教育域=当前具体域或common；
//   - COUNT和分页SELECT共用同一WHERE，统计不能先算全量再在前端隐藏。
//
// 旧ListLessonPlans保留给存量内部调用，但上下文17之后HTTP入口不得再使用它。

import (
	"context"
	"fmt"
	"strings"

	"tedna/internal/database"
	"tedna/internal/models"
)

// LessonPlanListSharedAccess 是列表查询使用的共享教案访问快照。
//
// SharedAuthorIDs和EducationDomain必须由services层通过实时用户、组织关系和
// 唯一具体教学域解析得到，Repository不接受前端直接传入的域或作者白名单。
// SharedOnly=true表示当前请求是共享市场查询，必须只返回满足共享候选条件的资源。
type LessonPlanListSharedAccess struct {
	SharedAuthorIDs []string
	EducationDomain string
	SharedOnly      bool
}

// ListLessonPlansWithSharedAccess 获取教案列表。
func ListLessonPlansWithSharedAccess(
	ctx context.Context,
	callerID string,
	authorID string,
	groupID string,
	status string,
	subject string,
	grade string,
	limit int,
	offset int,
	qualityLevel int,
	structureType int,
	cognitiveLevel int,
	pedagogyIntensity int,
	scopeUserIDs []string,
	scopeIsAdmin bool,
	sharedAccess *LessonPlanListSharedAccess,
) ([]*models.LessonPlanListItem, int, error) {
	where := " WHERE lp.deleted_at IS NULL"
	args := []interface{}{}
	argIdx := 1

	if authorID != "" {
		where += fmt.Sprintf(
			" AND lp.author_id = $%d",
			argIdx,
		)
		args = append(args, authorID)
		argIdx++
	}
	if groupID != "" {
		where += fmt.Sprintf(
			" AND lp.group_id = $%d",
			argIdx,
		)
		args = append(args, groupID)
		argIdx++
	}
	if status != "" {
		where += fmt.Sprintf(
			" AND lp.status = $%d",
			argIdx,
		)
		args = append(args, status)
		argIdx++
	}
	if subject != "" {
		where += fmt.Sprintf(
			" AND lp.subject = $%d",
			argIdx,
		)
		args = append(args, subject)
		argIdx++
	}
	if grade != "" {
		where += fmt.Sprintf(
			" AND lp.grade = $%d",
			argIdx,
		)
		args = append(args, grade)
		argIdx++
	}

	// AOCI索引维度筛选。
	if qualityLevel > 0 {
		where += fmt.Sprintf(
			" AND lp.idx_quality_level >= $%d",
			argIdx,
		)
		args = append(args, qualityLevel)
		argIdx++
	}
	if structureType > 0 {
		where += fmt.Sprintf(
			" AND lp.idx_structure_type = $%d",
			argIdx,
		)
		args = append(args, structureType)
		argIdx++
	}
	if cognitiveLevel > 0 {
		where += fmt.Sprintf(
			" AND lp.idx_cognitive_level >= $%d",
			argIdx,
		)
		args = append(args, cognitiveLevel)
		argIdx++
	}
	if pedagogyIntensity > 0 {
		where += fmt.Sprintf(
			" AND lp.idx_pedagogy_intensity = $%d",
			argIdx,
		)
		args = append(args, pedagogyIntensity)
		argIdx++
	}

	sharedEducationDomain := ""
	sharedAuthorIDs := []string{}
	sharedAccessReady := false
	if sharedAccess != nil {
		sharedEducationDomain = strings.ToLower(
			strings.TrimSpace(
				sharedAccess.EducationDomain,
			),
		)
		sharedAuthorIDs =
			sharedAccess.SharedAuthorIDs
		sharedAccessReady =
			models.IsTeachingEducationDomain(
				sharedEducationDomain,
			) &&
				len(sharedAuthorIDs) > 0
	}

	// 只在真正消费共享分支时追加参数，
	// 防止admin普通列表产生“参数多于占位符”的查询错误。
	appendSharedPredicate := func() string {
		if !sharedAccessReady {
			return "1=0"
		}

		authorPlaceholder := argIdx
		domainPlaceholder := argIdx + 1
		args = append(
			args,
			sharedAuthorIDs,
			sharedEducationDomain,
		)
		argIdx += 2

		return fmt.Sprintf(
			"(lp.status IN ('published_shared','approved') "+
				"AND lp.visibility IN ('group','school','region','public') "+
				"AND lp.author_id::text = ANY($%d) "+
				"AND lp.education_domain IN ($%d,'common'))",
			authorPlaceholder,
			domainPlaceholder,
		)
	}

	// 共享市场查询无论调用者是否为admin，都必须具有唯一具体教学域。
	if sharedAccess != nil &&
		sharedAccess.SharedOnly {
		where += " AND " +
			appendSharedPredicate()
	} else if !scopeIsAdmin {
		// 非admin普通列表保留“管辖作者/本人”能力，
		// 但共享资源分支必须经过统一组织与教育域过滤。
		isMyPlansView :=
			authorID != "" &&
				callerID != "" &&
				authorID == callerID

		scopePlaceholder := argIdx
		args = append(args, scopeUserIDs)
		argIdx++

		sharedPredicate :=
			appendSharedPredicate()

		if isMyPlansView {
			callerPlaceholder := argIdx
			args = append(args, callerID)
			argIdx++

			where += fmt.Sprintf(
				" AND ("+
					"lp.author_id::text = ANY($%d) "+
					"OR %s "+
					"OR lp.author_id = $%d"+
					")",
				scopePlaceholder,
				sharedPredicate,
				callerPlaceholder,
			)
		} else {
			where += fmt.Sprintf(
				" AND ("+
					"lp.author_id::text = ANY($%d) "+
					"OR %s"+
					")",
				scopePlaceholder,
				sharedPredicate,
			)
		}
	}

	var total int
	countQuery :=
		"SELECT COUNT(*) FROM lesson_plans lp" +
			where
	if err := database.DB.QueryRow(
		ctx,
		countQuery,
		args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf(
			"查询教案总数失败: %w",
			err,
		)
	}

	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	listQuery := fmt.Sprintf(`
                SELECT lp.id, lp.title, lp.subject, lp.grade, lp.topic, lp.duration_minutes,
                       lp.status, lp.visibility, lp.author_id,
                       COALESCE(u.display_name, '') AS author_name,
                       lp.ai_review_score, lp.fork_count, lp.view_count,
                       lp.forked_from, lp.recipe_id,
                       COALESCE(tr.name, '') AS recipe_name,
                       COALESCE(lp.lesson_index, '') AS lesson_index,
                       lp.idx_quality_level,
                       lp.review_level,
                       lp.created_at, lp.updated_at
                FROM lesson_plans lp
                LEFT JOIN users u ON u.id = lp.author_id
                LEFT JOIN teaching_recipes tr ON tr.id = lp.recipe_id
                %s
                ORDER BY lp.updated_at DESC
                LIMIT $%d OFFSET $%d
        `, where, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := database.DB.Query(
		ctx,
		listQuery,
		args...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf(
			"查询教案列表失败: %w",
			err,
		)
	}
	defer rows.Close()

	items := make(
		[]*models.LessonPlanListItem,
		0,
	)
	for rows.Next() {
		item := &models.LessonPlanListItem{}
		err := rows.Scan(
			&item.ID,
			&item.Title,
			&item.Subject,
			&item.Grade,
			&item.Topic,
			&item.DurationMinutes,
			&item.Status,
			&item.Visibility,
			&item.AuthorID,
			&item.AuthorName,
			&item.AIReviewScore,
			&item.ForkCount,
			&item.ViewCount,
			&item.ForkedFrom,
			&item.RecipeID,
			&item.RecipeName,
			&item.LessonIndex,
			&item.IdxQualityLevel,
			&item.ReviewLevel,
			&item.CreatedAt,
			&item.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf(
				"扫描教案行失败: %w",
				err,
			)
		}

		item.StatusName =
			models.LPStatusNameMap[item.Status]
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf(
			"遍历教案列表失败: %w",
			err,
		)
	}

	return items, total, nil
}
