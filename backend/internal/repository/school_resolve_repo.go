package repository

// school_resolve_repo.go — 用户所属学校 ID 解析（供模型分流读取 SchoolID）
//
// 背景：模型境内/境外分流（applyModelPolicy）依据 TraceContext.SchoolID 判断学校是否
//       被授权使用境外模型。但备课对话链路构造 traceCtx 时此前未填 SchoolID，
//       导致 PKU 等授权学校的老师也被 fail-closed 降级到境内模型。
//
// 本文件提供按 user_id 解析其所属学校 ID 的权威查询，查找链路与 GetUserPortalModules 一致：
//   1) school_members（v122 起的学校直接成员名单，权威来源）
//   2) 兜底：teaching_group_members → teaching_groups.school_id（历史用户只在教研组）
//
// 失败/未绑定任何学校 → 返回空串 + nil error（调用方据空串走 fail-closed 降级，安全）。

import (
	"context"

	"tedna/internal/database"
)

// GetSchoolIDByUserID 解析用户所属学校 ID。
// 返回 ("", nil) 表示用户未绑定任何学校（非错误）；只有 DB 异常才返回 error。
// 查找优先级：school_members 权威名单 → teaching_group_members 兜底。
func GetSchoolIDByUserID(ctx context.Context, userID string) (string, error) {
	if userID == "" {
		return "", nil
	}

	var schoolID string

	// 1) 权威来源：school_members 直接成员名单
	err := database.DB.QueryRow(ctx, `
		SELECT sm.school_id::text
		FROM school_members sm
		JOIN organizations o ON o.id = sm.school_id
		WHERE sm.user_id = $1 AND o.status = 'active'
		LIMIT 1
	`, userID).Scan(&schoolID)
	if err == nil && schoolID != "" {
		return schoolID, nil
	}

	// 2) 兜底：通过教研组反查学校
	err2 := database.DB.QueryRow(ctx, `
		SELECT tg.school_id::text
		FROM teaching_group_members tgm
		JOIN teaching_groups tg ON tg.id = tgm.group_id
		JOIN organizations o ON o.id = tg.school_id
		WHERE tgm.user_id = $1 AND o.status = 'active'
		LIMIT 1
	`, userID).Scan(&schoolID)
	if err2 == nil && schoolID != "" {
		return schoolID, nil
	}

	// 未绑定任何学校（两条链路都无记录）→ 空串 + nil（fail-closed 友好）
	return "", nil
}
