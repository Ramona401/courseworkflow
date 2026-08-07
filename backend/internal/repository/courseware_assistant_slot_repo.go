package repository

// courseware_assistant_slot_repo.go
//
// 本文件只负责课件教学智能体插槽写入：
//   1. 创建；
//   2. 更新可编辑字段；
//   3. 硬删除。
//
// 边界规则：
//   - 创建必须通过courseware_id和稳定page_id解析真实页面；
//   - 更新不得改变courseware_id、page_id和created_by；
//   - 更新和删除必须同时匹配courseware_id、page_id和slot_id；
//   - 已进入回收站的课件不能继续写入插槽；
//   - 删除固定采用硬删除；数据库会将既有部署的slot_id置空，
//     不会删除已发布部署或不可变版本；
//   - 本仓储不判断当前登录用户是否为课件作者。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"tedna/internal/database"
	"tedna/internal/models"
)

// CreateCoursewareAssistantSlot 为一个真实课件页面创建唯一插槽。
func CreateCoursewareAssistantSlot(
	ctx context.Context,
	coursewareID string,
	pageID string,
	createdBy string,
	request *models.CreateCoursewareAssistantSlotRequest,
) (
	*models.CoursewareAssistantSlotView,
	error,
) {
	if request == nil {
		return nil,
			ErrCoursewareAssistantSlotInvalidInput
	}

	guidanceJSON,
		contextJSON,
		err := marshalCoursewareAssistantSlotPayload(
		request.GuidancePlan,
		request.ContextConfig,
	)
	if err != nil {
		return nil, err
	}

	var assistantID interface{}
	if request.AssistantID != nil &&
		strings.TrimSpace(*request.AssistantID) != "" {
		assistantID =
			strings.TrimSpace(*request.AssistantID)
	}

	query := `
		WITH inserted AS (
			INSERT INTO courseware_assistant_slots (
				courseware_id,
				page_id,
				assistant_id,
				created_by,
				display_mode,
				display_position,
				title,
				welcome_message,
				teaching_role,
				learning_objective,
				guidance_plan_json,
				context_config_json,
				status
			)
			SELECT
				page.courseware_id,
				page.id,
				$3,
				$4,
				$5,
				$6,
				$7,
				$8,
				$9,
				$10,
				$11::jsonb,
				$12::jsonb,
				$13
			FROM courseware_pages AS page
			JOIN coursewares AS courseware
				ON courseware.id = page.courseware_id
				AND courseware.deleted_at IS NULL
			WHERE page.courseware_id = $1
				AND page.id = $2
			RETURNING *
		)
		SELECT` + coursewareAssistantSlotSelectColumns + `
		FROM inserted AS slot` +
		coursewareAssistantSlotReadJoins

	item, err := scanCoursewareAssistantSlotView(
		database.DB.QueryRow(
			ctx,
			query,
			strings.TrimSpace(coursewareID),
			strings.TrimSpace(pageID),
			assistantID,
			strings.TrimSpace(createdBy),
			models.CoursewareAssistantDisplayModeFloating,
			models.CoursewareAssistantPositionBottomRight,
			request.Title,
			request.WelcomeMessage,
			request.TeachingRole,
			request.LearningObjective,
			guidanceJSON,
			contextJSON,
			models.CoursewareAssistantSlotStatusActive,
		),
	)
	if err != nil {
		mappedErr :=
			mapCoursewareAssistantSlotWriteError(err)

		// INSERT ... SELECT没有返回行且不存在约束错误时，
		// 说明课件或稳定页面不存在。
		if errors.Is(
			mappedErr,
			ErrCoursewareAssistantSlotNotFound,
		) {
			return nil,
				ErrCoursewareAssistantSlotPageNotFound
		}

		return nil, fmt.Errorf(
			"创建课件教学智能体插槽失败: %w",
			mappedErr,
		)
	}

	return item, nil
}

// UpdateCoursewareAssistantSlot 更新插槽可编辑字段。
//
// 固定归属字段和展示位置均不在SET列表中，不能通过本函数迁移插槽。
func UpdateCoursewareAssistantSlot(
	ctx context.Context,
	coursewareID string,
	pageID string,
	slotID string,
	request *models.UpdateCoursewareAssistantSlotRequest,
) (
	*models.CoursewareAssistantSlotView,
	error,
) {
	if request == nil {
		return nil,
			ErrCoursewareAssistantSlotInvalidInput
	}

	guidanceJSON,
		contextJSON,
		err := marshalCoursewareAssistantSlotPayload(
		request.GuidancePlan,
		request.ContextConfig,
	)
	if err != nil {
		return nil, err
	}

	var assistantID interface{}
	if request.AssistantID != nil &&
		strings.TrimSpace(*request.AssistantID) != "" {
		assistantID =
			strings.TrimSpace(*request.AssistantID)
	}

	status := strings.TrimSpace(request.Status)
	if status == "" {
		status =
			models.CoursewareAssistantSlotStatusActive
	}

	query := `
		WITH updated AS (
			UPDATE courseware_assistant_slots AS slot
			SET
				assistant_id = $4,
				title = $5,
				welcome_message = $6,
				teaching_role = $7,
				learning_objective = $8,
				guidance_plan_json = $9::jsonb,
				context_config_json = $10::jsonb,
				status = $11,
				updated_at = NOW()
			FROM
				coursewares AS courseware,
				courseware_pages AS page
			WHERE slot.id = $1
				AND slot.courseware_id = $2
				AND slot.page_id = $3
				AND courseware.id =
					slot.courseware_id
				AND courseware.deleted_at IS NULL
				AND page.id = slot.page_id
				AND page.courseware_id =
					slot.courseware_id
			RETURNING slot.*
		)
		SELECT` + coursewareAssistantSlotSelectColumns + `
		FROM updated AS slot` +
		coursewareAssistantSlotReadJoins

	item, err := scanCoursewareAssistantSlotView(
		database.DB.QueryRow(
			ctx,
			query,
			strings.TrimSpace(slotID),
			strings.TrimSpace(coursewareID),
			strings.TrimSpace(pageID),
			assistantID,
			request.Title,
			request.WelcomeMessage,
			request.TeachingRole,
			request.LearningObjective,
			guidanceJSON,
			contextJSON,
			status,
		),
	)
	if err != nil {
		mappedErr :=
			mapCoursewareAssistantSlotWriteError(err)

		return nil, fmt.Errorf(
			"更新课件教学智能体插槽失败: %w",
			mappedErr,
		)
	}

	return item, nil
}

// DeleteCoursewareAssistantSlot 使用三重资源边界硬删除插槽。
//
// 外键会把既有assistant_deployments.slot_id置空，
// 已发布部署和不可变版本不会随插槽删除而丢失。
func DeleteCoursewareAssistantSlot(
	ctx context.Context,
	coursewareID string,
	pageID string,
	slotID string,
) error {
	result, err := database.DB.Exec(
		ctx,
		`
		DELETE FROM courseware_assistant_slots AS slot
		USING
			coursewares AS courseware,
			courseware_pages AS page
		WHERE slot.id = $1
			AND slot.courseware_id = $2
			AND slot.page_id = $3
			AND courseware.id =
				slot.courseware_id
			AND courseware.deleted_at IS NULL
			AND page.id = slot.page_id
			AND page.courseware_id =
				slot.courseware_id`,
		strings.TrimSpace(slotID),
		strings.TrimSpace(coursewareID),
		strings.TrimSpace(pageID),
	)
	if err != nil {
		return fmt.Errorf(
			"删除课件教学智能体插槽失败: %w",
			err,
		)
	}

	if result.RowsAffected() == 0 {
		return ErrCoursewareAssistantSlotNotFound
	}

	return nil
}
