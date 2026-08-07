package repository

// courseware_suggestion_repo.go — 课件页"AI物料建议"持久化数据层
//
// 把 AI 生成的【生图建议】【视频分镜】落库到 courseware_pages 表的两个 jsonb 列:
//   - image_suggestions  jsonb  生图建议数组 [{caption,prompt}, ...]
//   - video_storyboards  jsonb  视频分镜数组 [{scene,image_prompt,video_prompt,narration}, ...]
// 目的: 进页"先读库、没有才调AI", 省去每次进页/刷新都重算的 token 浪费
// (生成出来的图/视频本就已存 courseware_assets, 这里补上"AI建议文本"这一层的持久化)。
//
// 读: 用 COALESCE(列::text,'') 把 jsonb 取成 Go string(NULL→空串), 调用方拿到空串即判定"库里没有"。
// 写: 复用同包 nullIfEmptyJSON(见 courseware_asset_repo.go)——空串写 SQL NULL, 非空原样交 Postgres 解析;
//     入参须为合法 JSON 字符串(调用方用 encoding/json 序列化或已校验)。
// 定位键: courseware_id + page_number(该表 UNIQUE 约束, 唯一定位一页)。

import (
	"context"

	"tedna/internal/database"
)

// GetPageImageSuggestions 读取指定页已存的生图建议(JSON文本); 无记录或列为 NULL 时返回空串。
func GetPageImageSuggestions(ctx context.Context, coursewareID string, pageNumber int) (string, error) {
	var s string
	sql := `SELECT COALESCE(image_suggestions::text,'') FROM courseware_pages
        WHERE courseware_id = $1 AND page_number = $2`
	if err := database.DB.QueryRow(ctx, sql, coursewareID, pageNumber).Scan(&s); err != nil {
		return "", err
	}
	return s, nil
}

// GetPageVideoStoryboards 读取指定页已存的视频分镜(JSON文本); 无则返回空串。
func GetPageVideoStoryboards(ctx context.Context, coursewareID string, pageNumber int) (string, error) {
	var s string
	sql := `SELECT COALESCE(video_storyboards::text,'') FROM courseware_pages
        WHERE courseware_id = $1 AND page_number = $2`
	if err := database.DB.QueryRow(ctx, sql, coursewareID, pageNumber).Scan(&s); err != nil {
		return "", err
	}
	return s, nil
}

// UpdatePageImageSuggestions 覆盖写指定页的生图建议。
//
// 自动装配context存在时，写入必须仍属于当前running运行；普通媒体工作台
// 不携带装配身份，继续使用原有页码定位逻辑。
func UpdatePageImageSuggestions(
	ctx context.Context,
	coursewareID string,
	pageNumber int,
	suggestionsJSON string,
) error {
	if assembly, ok :=
		coursewareAssemblyWriteContextFrom(
			ctx,
		); ok {
		if coursewareID !=
			assembly.CoursewareID {
			return ErrCoursewareAssemblyVersionConflict
		}

		tag, err := database.DB.Exec(
			ctx,
			`UPDATE courseware_pages AS page
                        SET
                                image_suggestions = $1,
                                updated_at = NOW()
                        FROM coursewares AS courseware
                        WHERE page.courseware_id = $2
                          AND page.page_number = $3
                          AND courseware.id = page.courseware_id
                          AND courseware.assembly_version = $4
                          AND courseware.assembly_status = 'running'
                          AND courseware.active_assembly_run_id = $5`,
			nullIfEmptyJSON(
				suggestionsJSON,
			),
			coursewareID,
			pageNumber,
			assembly.Version,
			assembly.RunID,
		)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return ErrCoursewareAssemblyVersionConflict
		}

		return nil
	}

	sql :=
		`UPDATE courseware_pages
                SET image_suggestions = $1, updated_at = NOW()
                WHERE courseware_id = $2 AND page_number = $3`

	_, err := database.DB.Exec(
		ctx,
		sql,
		nullIfEmptyJSON(
			suggestionsJSON,
		),
		coursewareID,
		pageNumber,
	)

	return err
}

// UpdatePageVideoStoryboards 覆盖写指定页的视频分镜。
//
// 装配取消后，迟到的视频分镜结果不得继续覆盖页面媒体建议。
func UpdatePageVideoStoryboards(
	ctx context.Context,
	coursewareID string,
	pageNumber int,
	storyboardsJSON string,
) error {
	if assembly, ok :=
		coursewareAssemblyWriteContextFrom(
			ctx,
		); ok {
		if coursewareID !=
			assembly.CoursewareID {
			return ErrCoursewareAssemblyVersionConflict
		}

		tag, err := database.DB.Exec(
			ctx,
			`UPDATE courseware_pages AS page
                        SET
                                video_storyboards = $1,
                                updated_at = NOW()
                        FROM coursewares AS courseware
                        WHERE page.courseware_id = $2
                          AND page.page_number = $3
                          AND courseware.id = page.courseware_id
                          AND courseware.assembly_version = $4
                          AND courseware.assembly_status = 'running'
                          AND courseware.active_assembly_run_id = $5`,
			nullIfEmptyJSON(
				storyboardsJSON,
			),
			coursewareID,
			pageNumber,
			assembly.Version,
			assembly.RunID,
		)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return ErrCoursewareAssemblyVersionConflict
		}

		return nil
	}

	sql :=
		`UPDATE courseware_pages
                SET video_storyboards = $1, updated_at = NOW()
                WHERE courseware_id = $2 AND page_number = $3`

	_, err := database.DB.Exec(
		ctx,
		sql,
		nullIfEmptyJSON(
			storyboardsJSON,
		),
		coursewareID,
		pageNumber,
	)

	return err
}
