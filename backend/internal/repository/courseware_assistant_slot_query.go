package repository

// courseware_assistant_slot_query.go
//
// 本文件只负责课件教学智能体插槽读取：
//   1. 按课件列出全部插槽；
//   2. 按稳定页面ID读取当前页插槽；
//   3. 按courseware_id、page_id和slot_id复合读取单项。
//
// 查询不会读取助手完整提示词，也不会推断当前用户是否有权限。
// 作者权限、课件锁定状态和教育域判断由下一开发单元的Service负责。

import (
	"context"
	"fmt"
	"strings"

	"tedna/internal/database"
	"tedna/internal/models"
)

// ListCoursewareAssistantSlotsByCoursewareID 按页面当前顺序列出课件全部插槽。
func ListCoursewareAssistantSlotsByCoursewareID(
	ctx context.Context,
	coursewareID string,
) (
	[]*models.CoursewareAssistantSlotView,
	error,
) {
	query := `
		SELECT` + coursewareAssistantSlotSelectColumns + `
		FROM courseware_assistant_slots AS slot` +
		coursewareAssistantSlotReadJoins + `
		WHERE slot.courseware_id = $1
		ORDER BY
			page.page_number ASC,
			slot.created_at ASC`

	rows, err := database.DB.Query(
		ctx,
		query,
		strings.TrimSpace(coursewareID),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"查询课件教学智能体插槽列表失败: %w",
			err,
		)
	}
	defer rows.Close()

	items := make(
		[]*models.CoursewareAssistantSlotView,
		0,
	)

	for rows.Next() {
		item, err :=
			scanCoursewareAssistantSlotView(rows)
		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"遍历课件教学智能体插槽列表失败: %w",
			err,
		)
	}

	return items, nil
}

// GetCoursewareAssistantSlotByPage 按课件和稳定页面ID读取当前页唯一插槽。
func GetCoursewareAssistantSlotByPage(
	ctx context.Context,
	coursewareID string,
	pageID string,
) (
	*models.CoursewareAssistantSlotView,
	error,
) {
	query := `
		SELECT` + coursewareAssistantSlotSelectColumns + `
		FROM courseware_assistant_slots AS slot` +
		coursewareAssistantSlotReadJoins + `
		WHERE slot.courseware_id = $1
			AND slot.page_id = $2`

	item, err := scanCoursewareAssistantSlotView(
		database.DB.QueryRow(
			ctx,
			query,
			strings.TrimSpace(coursewareID),
			strings.TrimSpace(pageID),
		),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"读取课件页面教学智能体插槽失败: %w",
			err,
		)
	}

	return item, nil
}

// GetCoursewareAssistantSlotByID 使用课件、页面和插槽三重边界读取单项。
//
// 即使调用方获得了其它课件的slot_id，也不能通过本函数跨资源读取。
func GetCoursewareAssistantSlotByID(
	ctx context.Context,
	coursewareID string,
	pageID string,
	slotID string,
) (
	*models.CoursewareAssistantSlotView,
	error,
) {
	query := `
		SELECT` + coursewareAssistantSlotSelectColumns + `
		FROM courseware_assistant_slots AS slot` +
		coursewareAssistantSlotReadJoins + `
		WHERE slot.courseware_id = $1
			AND slot.page_id = $2
			AND slot.id = $3`

	item, err := scanCoursewareAssistantSlotView(
		database.DB.QueryRow(
			ctx,
			query,
			strings.TrimSpace(coursewareID),
			strings.TrimSpace(pageID),
			strings.TrimSpace(slotID),
		),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"按复合边界读取课件教学智能体插槽失败: %w",
			err,
		)
	}

	return item, nil
}
