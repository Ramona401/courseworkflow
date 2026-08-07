package repository

// courseware_image_relation_repo.go — 图片R关系仓储
//
// 负责：
//   - 使用稳定图片键替换关系；
//   - 校验同课件边界、自引用、重复和循环；
//   - 在同一事务中同步关系表、RC字段和aoci_text中的[R]行。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

const cwImageRelationSelectColumns = `
r.id,
r.courseware_id,
r.source_image_index_id,
r.target_image_index_id,
source_idx.image_key,
target_idx.image_key,
r.relation_code,
r.inherit_mask,
r.semantic_note,
r.created_at`

func scanCoursewareImageRelation(
	scanner interface {
		Scan(dest ...interface{}) error
	},
) (*models.CoursewareImageRelation, error) {
	item := &models.CoursewareImageRelation{}

	err := scanner.Scan(
		&item.ID,
		&item.CoursewareID,
		&item.SourceImageIndexID,
		&item.TargetImageIndexID,
		&item.SourceImageKey,
		&item.TargetImageKey,
		&item.RelationCode,
		&item.InheritMask,
		&item.SemanticNote,
		&item.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return item, nil
}

// ReplaceCoursewareImageRelationsByKeys 原子替换R关系。
func ReplaceCoursewareImageRelationsByKeys(
	ctx context.Context,
	coursewareID string,
	sourceImageKey string,
	specs []models.CoursewareImageRelationSpec,
) ([]*models.CoursewareImageRelation, error) {
	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"开启图片关系事务失败: %w",
			err,
		)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var sourceID string
	var sourceAOCIText string

	err = tx.QueryRow(
		ctx,
		`SELECT id, aoci_text
FROM courseware_image_indexes
WHERE courseware_id = $1
  AND image_key = $2
FOR UPDATE`,
		coursewareID,
		sourceImageKey,
	).Scan(
		&sourceID,
		&sourceAOCIText,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCoursewareImageIndexNotFound
	}
	if err != nil {
		return nil, fmt.Errorf(
			"锁定来源图片索引失败: %w",
			err,
		)
	}

	normalizedSpecs, err :=
		normalizeRepositoryImageRelations(specs)
	if err != nil {
		return nil, err
	}

	if _, err := tx.Exec(
		ctx,
		`DELETE FROM courseware_image_relations
WHERE courseware_id = $1
  AND source_image_index_id = $2`,
		coursewareID,
		sourceID,
	); err != nil {
		return nil, fmt.Errorf(
			"清理旧图片关系失败: %w",
			err,
		)
	}

	created := make(
		[]*models.CoursewareImageRelation,
		0,
		len(normalizedSpecs),
	)

	for _, spec := range normalizedSpecs {
		if spec.TargetImageKey == sourceImageKey {
			return nil, fmt.Errorf(
				"图片关系禁止自引用",
			)
		}

		var targetID string

		err := tx.QueryRow(
			ctx,
			`SELECT id
FROM courseware_image_indexes
WHERE courseware_id = $1
  AND image_key = $2`,
			coursewareID,
			spec.TargetImageKey,
		).Scan(&targetID)

		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf(
				"关系目标不存在: %s",
				spec.TargetImageKey,
			)
		}
		if err != nil {
			return nil, fmt.Errorf(
				"读取关系目标失败: %w",
				err,
			)
		}

		hasCycle, err :=
			coursewareImageRelationWouldCycle(
				ctx,
				tx,
				coursewareID,
				sourceID,
				targetID,
			)
		if err != nil {
			return nil, err
		}
		if hasCycle {
			return nil, fmt.Errorf(
				"%w：%s %s %s",
				ErrCoursewareImageRelationCycle,
				sourceImageKey,
				spec.RelationCode,
				spec.TargetImageKey,
			)
		}

		relation := &models.CoursewareImageRelation{
			CoursewareID:       coursewareID,
			SourceImageIndexID: sourceID,
			TargetImageIndexID: targetID,
			SourceImageKey:     sourceImageKey,
			TargetImageKey:     spec.TargetImageKey,
			RelationCode:       spec.RelationCode,
			InheritMask:        spec.InheritMask,
			SemanticNote:       spec.SemanticNote,
		}

		err = tx.QueryRow(
			ctx,
			`INSERT INTO courseware_image_relations (
courseware_id,
source_image_index_id,
target_image_index_id,
relation_code,
inherit_mask,
semantic_note
)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, created_at`,
			relation.CoursewareID,
			relation.SourceImageIndexID,
			relation.TargetImageIndexID,
			relation.RelationCode,
			relation.InheritMask,
			relation.SemanticNote,
		).Scan(
			&relation.ID,
			&relation.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"创建图片关系失败: %w",
				err,
			)
		}

		created = append(created, relation)
	}

	rebuiltAOCI, err :=
		rebuildRepositoryImageAOCIText(
			sourceAOCIText,
			normalizedSpecs,
		)
	if err != nil {
		return nil, fmt.Errorf(
			"同步IAOCI关系文本失败: %w",
			err,
		)
	}

	relationCount :=
		models.CWImageRelationCountCode(
			len(normalizedSpecs),
		)

	tag, err := tx.Exec(
		ctx,
		`UPDATE courseware_image_indexes
SET relation_count = $1,
	aoci_text = $2,
	version = version + 1,
	updated_at = now()
WHERE id = $3
  AND courseware_id = $4`,
		relationCount,
		rebuiltAOCI,
		sourceID,
		coursewareID,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"同步图片关系索引失败: %w",
			err,
		)
	}
	if tag.RowsAffected() != 1 {
		return nil, ErrCoursewareImageIndexNotFound
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf(
			"提交图片关系事务失败: %w",
			err,
		)
	}

	return created, nil
}

// ListCoursewareImageRelationsBySource 返回来源图片关系。
func ListCoursewareImageRelationsBySource(
	ctx context.Context,
	coursewareID string,
	sourceImageKey string,
) ([]*models.CoursewareImageRelation, error) {
	sql := `SELECT ` +
		cwImageRelationSelectColumns +
		` FROM courseware_image_relations r
JOIN courseware_image_indexes source_idx
	ON source_idx.id = r.source_image_index_id
	AND source_idx.courseware_id = r.courseware_id
JOIN courseware_image_indexes target_idx
	ON target_idx.id = r.target_image_index_id
	AND target_idx.courseware_id = r.courseware_id
WHERE r.courseware_id = $1
  AND source_idx.image_key = $2
ORDER BY r.created_at ASC`

	rows, err := database.DB.Query(
		ctx,
		sql,
		coursewareID,
		sourceImageKey,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"查询图片关系失败: %w",
			err,
		)
	}
	defer rows.Close()

	items := make(
		[]*models.CoursewareImageRelation,
		0,
	)

	for rows.Next() {
		item, scanErr :=
			scanCoursewareImageRelation(rows)
		if scanErr != nil {
			return nil, fmt.Errorf(
				"扫描图片关系失败: %w",
				scanErr,
			)
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"遍历图片关系失败: %w",
			err,
		)
	}

	return items, nil
}

func normalizeRepositoryImageRelations(
	specs []models.CoursewareImageRelationSpec,
) ([]models.CoursewareImageRelationSpec, error) {
	normalized := make(
		[]models.CoursewareImageRelationSpec,
		0,
		len(specs),
	)

	seen := make(map[string]bool, len(specs))

	for _, spec := range specs {
		spec.TargetImageKey =
			strings.TrimSpace(spec.TargetImageKey)
		spec.RelationCode =
			strings.TrimSpace(spec.RelationCode)
		spec.SemanticNote =
			strings.TrimSpace(spec.SemanticNote)

		if !models.IsValidCWImageRelationCode(
			spec.RelationCode,
		) {
			return nil, fmt.Errorf(
				"图片关系类型不合法: %s",
				spec.RelationCode,
			)
		}

		mask, ok :=
			models.NormalizeCWImageInheritMask(
				spec.InheritMask,
			)
		if !ok {
			return nil, fmt.Errorf(
				"继承掩码不合法: %s",
				spec.InheritMask,
			)
		}

		spec.InheritMask = mask

		if !isRepositoryImageKey(
			spec.TargetImageKey,
		) ||
			spec.TargetImageKey == "@ANCHOR" {
			return nil, fmt.Errorf(
				"关系目标键不合法: %s",
				spec.TargetImageKey,
			)
		}

		if strings.ContainsAny(
			spec.SemanticNote,
			"{}",
		) {
			return nil, fmt.Errorf(
				"关系说明不能包含大括号",
			)
		}

		identity :=
			spec.TargetImageKey +
				"\x00" +
				spec.RelationCode

		if seen[identity] {
			return nil, fmt.Errorf(
				"图片关系重复：%s %s",
				spec.RelationCode,
				spec.TargetImageKey,
			)
		}

		seen[identity] = true
		normalized = append(normalized, spec)
	}

	return normalized, nil
}

func rebuildRepositoryImageAOCIText(
	indexText string,
	specs []models.CoursewareImageRelationSpec,
) (string, error) {
	indexText = strings.ReplaceAll(
		indexText,
		"\r\n",
		"\n",
	)
	indexText = strings.ReplaceAll(
		indexText,
		"\r",
		"\n",
	)

	rawLines := strings.Split(indexText, "\n")
	lines := make([]string, 0, len(rawLines))

	for _, rawLine := range rawLines {
		line := strings.TrimSpace(rawLine)
		if line != "" {
			lines = append(lines, line)
		}
	}

	if len(lines) == 0 ||
		!strings.HasPrefix(lines[0], "IK:") {
		return "", fmt.Errorf(
			"来源aoci_text不是规范IAOCI",
		)
	}

	headerParts := strings.Split(lines[0], "|")
	foundRC := false
	relationCount :=
		models.CWImageRelationCountCode(len(specs))

	for index, part := range headerParts {
		part = strings.TrimSpace(part)

		if strings.HasPrefix(part, "RC:") {
			headerParts[index] = "RC:" + relationCount
			foundRC = true
		} else {
			headerParts[index] = part
		}
	}

	if !foundRC {
		return "", fmt.Errorf(
			"来源IAOCI缺少RC字段",
		)
	}

	result := []string{
		strings.Join(headerParts, "|"),
	}

	foundNegative := false

	for _, line := range lines[1:] {
		if strings.HasPrefix(line, "[R]") {
			continue
		}

		if strings.HasPrefix(line, "[N]") {
			foundNegative = true

			if len(specs) == 0 {
				result = append(result, "[R]0")
			} else {
				for _, spec := range specs {
					relationLine, err :=
						formatRepositoryImageRelation(spec)
					if err != nil {
						return "", err
					}

					result = append(
						result,
						"[R]"+relationLine,
					)
				}
			}
		}

		result = append(result, line)
	}

	if !foundNegative {
		return "", fmt.Errorf(
			"来源IAOCI缺少[N]标签",
		)
	}

	return strings.Join(result, "\n"), nil
}

func formatRepositoryImageRelation(
	spec models.CoursewareImageRelationSpec,
) (string, error) {
	if !models.IsValidCWImageRelationCode(
		spec.RelationCode,
	) {
		return "", fmt.Errorf("关系类型不合法")
	}

	mask, ok :=
		models.NormalizeCWImageInheritMask(
			spec.InheritMask,
		)
	if !ok {
		return "", fmt.Errorf("关系掩码不合法")
	}

	if !isRepositoryImageKey(
		spec.TargetImageKey,
	) ||
		spec.TargetImageKey == "@ANCHOR" {
		return "", fmt.Errorf("关系目标键不合法")
	}

	note := strings.TrimSpace(spec.SemanticNote)

	if strings.ContainsAny(note, "{}") {
		return "", fmt.Errorf(
			"关系说明不能包含大括号",
		)
	}

	line :=
		spec.RelationCode +
			spec.TargetImageKey +
			"[" + mask + "]"

	if note != "" {
		line += "{" + note + "}"
	}

	return line, nil
}

func coursewareImageRelationWouldCycle(
	ctx context.Context,
	tx pgx.Tx,
	coursewareID string,
	sourceID string,
	targetID string,
) (bool, error) {
	var hasCycle bool

	err := tx.QueryRow(
		ctx,
		`WITH RECURSIVE relation_walk(image_index_id) AS (
	SELECT target_image_index_id
	FROM courseware_image_relations
	WHERE courseware_id = $1
	  AND source_image_index_id = $2

	UNION

	SELECT relation.target_image_index_id
	FROM courseware_image_relations relation
	JOIN relation_walk walk
	  ON relation.source_image_index_id =
	     walk.image_index_id
	WHERE relation.courseware_id = $1
)
SELECT EXISTS (
	SELECT 1
	FROM relation_walk
	WHERE image_index_id = $3
)`,
		coursewareID,
		targetID,
		sourceID,
	).Scan(&hasCycle)
	if err != nil {
		return false, fmt.Errorf(
			"检查图片关系环失败: %w",
			err,
		)
	}

	return hasCycle, nil
}
