package repository

// courseware_background_repo.go — 课件背景图库数据访问（批次1新建）
//
// 覆盖：图集列表(系统+本人个人)/单集查询/课件背景两列读写/页面HTML单列更新。
// UpdateCWPageHTMLOnly 专供"背景秒换"——只更新 html_content，绝不触碰
// placeholder_map / matched_component_ids / status（区别于 UpdateCWPageHTML 会整体覆盖）。

import (
	"context"
	"fmt"
	"time"

	"tedna/internal/database"
	"tedna/internal/models"
)

// cwBgSelectColumns 背景图集统一查询列（14列）
const cwBgSelectColumns = `id, name, COALESCE(description,''), COALESCE(style_category,''),
scope, user_id,
COALESCE(cover_oss_url,''), cover_public_url,
COALESCE(content_oss_url,''), content_public_url,
status, sort_order, created_at, updated_at`

// scanCWBgSet 统一扫描背景图集行
func scanCWBgSet(scan func(dest ...interface{}) error) (*models.CoursewareBackgroundSet, error) {
	s := &models.CoursewareBackgroundSet{}
	err := scan(
		&s.ID, &s.Name, &s.Description, &s.StyleCategory,
		&s.Scope, &s.UserID,
		&s.CoverOssURL, &s.CoverPublicURL,
		&s.ContentOssURL, &s.ContentPublicURL,
		&s.Status, &s.SortOrder, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// ListBackgroundSets 查询可见的激活图集：系统图库 + 本人个人图集，系统优先、再按 sort_order
func ListBackgroundSets(ctx context.Context, userID string) ([]*models.CoursewareBackgroundSet, error) {
	sql := fmt.Sprintf(`SELECT %s FROM courseware_background_sets
WHERE status = 'active' AND (scope = 'system' OR (scope = 'personal' AND user_id = $1))
ORDER BY CASE scope WHEN 'system' THEN 0 ELSE 1 END, sort_order ASC, created_at ASC`, cwBgSelectColumns)
	rows, err := database.DB.Query(ctx, sql, userID)
	if err != nil {
		return nil, fmt.Errorf("查询背景图集列表失败: %w", err)
	}
	defer rows.Close()
	var sets []*models.CoursewareBackgroundSet
	for rows.Next() {
		s, sErr := scanCWBgSet(rows.Scan)
		if sErr != nil {
			return nil, fmt.Errorf("扫描背景图集行失败: %w", sErr)
		}
		sets = append(sets, s)
	}
	return sets, nil
}

// GetBackgroundSetByID 按ID查询单个背景图集
func GetBackgroundSetByID(ctx context.Context, id string) (*models.CoursewareBackgroundSet, error) {
	sql := fmt.Sprintf(`SELECT %s FROM courseware_background_sets WHERE id = $1`, cwBgSelectColumns)
	return scanCWBgSet(database.DB.QueryRow(ctx, sql, id).Scan)
}

// UpdateCoursewareBackground 写入课件背景两列（URL快照）；空串写NULL（=未选）
func UpdateCoursewareBackground(ctx context.Context, coursewareID string, coverURL string, contentURL string) error {
	sql := `UPDATE coursewares SET cover_bg_url = $1, content_bg_url = $2, updated_at = $3 WHERE id = $4`
	tag, err := database.DB.Exec(ctx, sql, nullIfEmpty(coverURL), nullIfEmpty(contentURL), time.Now(), coursewareID)
	if err != nil {
		return fmt.Errorf("写入课件背景失败: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("课件不存在: %s", coursewareID)
	}
	return nil
}

// GetCoursewareBackgroundURLs 读取课件背景两列（未选为两空串）
// 独立小查询而非扩展 GetCoursewareByID：零侵入既有23列扫描，每次生成只多查一次，开销可忽略
func GetCoursewareBackgroundURLs(ctx context.Context, coursewareID string) (string, string, error) {
	var cover, content string
	sql := `SELECT COALESCE(cover_bg_url,''), COALESCE(content_bg_url,'') FROM coursewares WHERE id = $1`
	if err := database.DB.QueryRow(ctx, sql, coursewareID).Scan(&cover, &content); err != nil {
		return "", "", fmt.Errorf("读取课件背景失败: %w", err)
	}
	return cover, content, nil
}

// UpdateCWPageHTMLOnly 仅更新页面 html_content（背景秒换专用，不触碰其它列）
func UpdateCWPageHTMLOnly(ctx context.Context, pageID string, htmlContent string) error {
	sql := `UPDATE courseware_pages SET html_content = $1, updated_at = $2 WHERE id = $3`
	_, err := database.DB.Exec(ctx, sql, htmlContent, time.Now(), pageID)
	return err
}
