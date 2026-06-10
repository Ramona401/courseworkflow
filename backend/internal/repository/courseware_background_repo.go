package repository

// courseware_background_repo.go — 课件背景图库数据访问（批次1新建，批次3扩展生产入口）
//
// 覆盖：图集列表(系统+本人个人)/单集查询/课件背景两列读写/页面HTML单列更新。
// UpdateCWPageHTMLOnly 专供"背景秒换"——只更新 html_content，绝不触碰
// placeholder_map / matched_component_ids / status（区别于 UpdateCWPageHTML 会整体覆盖）。
//
// 批次3新增：CreateBackgroundSet(新建图集) / CountActivePersonalSets(个人集配额) /
// ArchiveBackgroundSet(归档删除，不删OSS对象) / PromoteSetToSystem(admin升级为系统图库)。

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

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

// CreateBackgroundSet 批次3：新建背景图集（AI生成/手动上传的个人集，或将来admin直建系统集）
// ID为空时由Go侧生成UUID（不依赖列默认值）；created_at/updated_at 回写到入参结构体
func CreateBackgroundSet(ctx context.Context, s *models.CoursewareBackgroundSet) error {
	if s == nil {
		return fmt.Errorf("背景图集对象为空")
	}
	if s.ID == "" {
		s.ID = uuid.NewString()
	}
	now := time.Now()
	sql := `INSERT INTO courseware_background_sets
(id, name, description, style_category, scope, user_id,
 cover_oss_url, cover_public_url, content_oss_url, content_public_url,
 status, sort_order, created_at, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`
	_, err := database.DB.Exec(ctx, sql,
		s.ID, s.Name, nullIfEmpty(s.Description), nullIfEmpty(s.StyleCategory),
		s.Scope, s.UserID,
		nullIfEmpty(s.CoverOssURL), s.CoverPublicURL,
		nullIfEmpty(s.ContentOssURL), s.ContentPublicURL,
		s.Status, s.SortOrder, now, now)
	if err != nil {
		return fmt.Errorf("写入背景图集失败: %w", err)
	}
	s.CreatedAt = &now
	s.UpdatedAt = &now
	return nil
}

// CountActivePersonalSets 批次3：统计某用户激活态个人图集数量（配额上限校验用）
func CountActivePersonalSets(ctx context.Context, userID string) (int, error) {
	var n int
	sql := `SELECT COUNT(*) FROM courseware_background_sets
WHERE scope = 'personal' AND user_id = $1 AND status = 'active'`
	if err := database.DB.QueryRow(ctx, sql, userID).Scan(&n); err != nil {
		return 0, fmt.Errorf("统计个人背景图集失败: %w", err)
	}
	return n, nil
}

// ArchiveBackgroundSet 批次3：归档图集（删除语义）。只改status不删行不删OSS对象——
// 已选该集的课件存的是URL快照，归档后这些课件背景照常显示，引用安全。
func ArchiveBackgroundSet(ctx context.Context, setID string) error {
	sql := `UPDATE courseware_background_sets SET status = 'archived', updated_at = $1 WHERE id = $2`
	tag, err := database.DB.Exec(ctx, sql, time.Now(), setID)
	if err != nil {
		return fmt.Errorf("归档背景图集失败: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("背景图集不存在: %s", setID)
	}
	return nil
}

// PromoteSetToSystem 批次3：把个人图集升级为系统图库（admin专属）。
// user_id 置NULL——该列对users有CASCADE外键，置空后原作者账号被删也不会连带删掉系统图集。
func PromoteSetToSystem(ctx context.Context, setID string) error {
	sql := `UPDATE courseware_background_sets SET scope = 'system', user_id = NULL, updated_at = $1 WHERE id = $2`
	tag, err := database.DB.Exec(ctx, sql, time.Now(), setID)
	if err != nil {
		return fmt.Errorf("升级系统图库失败: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("背景图集不存在: %s", setID)
	}
	return nil
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
