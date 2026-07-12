package repository

// courseware_page_bg_repo.go — 课件页级背景覆盖数据访问
//
// 职责：读写 courseware_pages 表的三个页级背景列
//   page_bg_url     — 该页专属背景图公网URL（空=跟随课件级）
//   page_bg_opacity — 该页蒙版透明度 0.0~1.0（NULL=跟随默认值）
//   page_bg_mode    — 蒙版模式 default/custom/none
//
// 设计：
//   - 三列与既有19列查询完全解耦（不改 cwPageSelectColumns / scanCWPage），
//     通过独立小查询读写，零侵入已有页面CRUD链路。
//   - 秒换逻辑复用 courseware_background_service.go 的 swapGeneratedPages 范式。

import (
	"context"
	"fmt"
	"time"

	"tedna/internal/database"
)

// PageBgSetting 单页背景设置（三列的读出结构）
type PageBgSetting struct {
	PageBgURL     string   `json:"page_bg_url"`
	PageBgOpacity *float64 `json:"page_bg_opacity"` // nil=跟随默认
	PageBgMode    string   `json:"page_bg_mode"`    // default/custom/none
}

// GetPageBgSetting 读取单页的页级背景设置
func GetPageBgSetting(ctx context.Context, coursewareID string, pageNumber int) (*PageBgSetting, error) {
	s := &PageBgSetting{}
	sql := `SELECT COALESCE(page_bg_url,''), page_bg_opacity, COALESCE(page_bg_mode,'default')
FROM courseware_pages WHERE courseware_id = $1 AND page_number = $2`
	err := database.DB.QueryRow(ctx, sql, coursewareID, pageNumber).Scan(
		&s.PageBgURL, &s.PageBgOpacity, &s.PageBgMode,
	)
	if err != nil {
		return nil, fmt.Errorf("读取页级背景设置失败: %w", err)
	}
	return s, nil
}

// UpdatePageBgSetting 写入单页的页级背景设置
// url 空串=清除页级背景（跟随课件级）；opacity nil=跟随默认；mode 空串归一为 default
func UpdatePageBgSetting(ctx context.Context, coursewareID string, pageNumber int, url string, opacity *float64, mode string) error {
	if mode == "" {
		mode = "default"
	}
	var opacityArg interface{}
	if opacity != nil {
		opacityArg = *opacity
	}
	sql := `UPDATE courseware_pages SET page_bg_url = $1, page_bg_opacity = $2, page_bg_mode = $3, updated_at = $4
WHERE courseware_id = $5 AND page_number = $6`
	tag, err := database.DB.Exec(ctx, sql, nullIfEmpty(url), opacityArg, mode, time.Now(), coursewareID, pageNumber)
	if err != nil {
		return fmt.Errorf("写入页级背景设置失败: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("课件页面不存在: courseware=%s page=%d", coursewareID, pageNumber)
	}
	return nil
}

// ClearPageBgSetting 清除单页的页级背景设置（三列回归默认）
func ClearPageBgSetting(ctx context.Context, coursewareID string, pageNumber int) error {
	return UpdatePageBgSetting(ctx, coursewareID, pageNumber, "", nil, "default")
}

// ListPageBgSettings 批量读取课件全部页的页级背景设置（秒换时一次查全部，避免逐页查询）
func ListPageBgSettings(ctx context.Context, coursewareID string) (map[int]*PageBgSetting, error) {
	sql := `SELECT page_number, COALESCE(page_bg_url,''), page_bg_opacity, COALESCE(page_bg_mode,'default')
FROM courseware_pages WHERE courseware_id = $1 ORDER BY page_number`
	rows, err := database.DB.Query(ctx, sql, coursewareID)
	if err != nil {
		return nil, fmt.Errorf("批量读取页级背景设置失败: %w", err)
	}
	defer rows.Close()
	result := make(map[int]*PageBgSetting)
	for rows.Next() {
		var pn int
		s := &PageBgSetting{}
		if err := rows.Scan(&pn, &s.PageBgURL, &s.PageBgOpacity, &s.PageBgMode); err != nil {
			return nil, fmt.Errorf("扫描页级背景行失败: %w", err)
		}
		result[pn] = s
	}
	return result, nil
}
