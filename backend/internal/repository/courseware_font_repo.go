package repository

// courseware_font_repo.go — 课件字体方案列读写（字体F1新建）
//
// 只涉及 coursewares.font_scheme 单列（VARCHAR(40) NOT NULL DEFAULT ''，空串=未选）。
// 页面HTML秒换复用 courseware_background_repo.go 的 UpdateCWPageHTMLOnly，本文件不重复。

import (
	"context"
	"fmt"
	"time"

	"tedna/internal/database"
)

// GetCoursewareFontScheme 读取课件字体方案code（未选为空串）
// 独立小查询而非扩展 GetCoursewareByID 的列扫描：零侵入既有代码，开销可忽略
func GetCoursewareFontScheme(ctx context.Context, coursewareID string) (string, error) {
	var code string
	sqlStr := "SELECT COALESCE(font_scheme,'') FROM coursewares WHERE id = $1"
	if err := database.DB.QueryRow(ctx, sqlStr, coursewareID).Scan(&code); err != nil {
		return "", fmt.Errorf("读取课件字体方案失败: %w", err)
	}
	return code, nil
}

// UpdateCoursewareFontScheme 写入课件字体方案code（空串=清除选择）
func UpdateCoursewareFontScheme(ctx context.Context, coursewareID string, schemeCode string) error {
	sqlStr := "UPDATE coursewares SET font_scheme = $1, updated_at = $2 WHERE id = $3"
	tag, err := database.DB.Exec(ctx, sqlStr, schemeCode, time.Now(), coursewareID)
	if err != nil {
		return fmt.Errorf("写入课件字体方案失败: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("课件不存在: %s", coursewareID)
	}
	return nil
}
