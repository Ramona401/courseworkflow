package repository

// courseware_collab_repo.go — 阶段4 集体备课·参与者表仓储
//
// 对应 courseware_collab_members 表（线下集体备课的参与者名单）。
// 设计原则：最小增量。本仓储只管"谁参与了某课件的集体备课"，
//   议课走 courseware_annotations（批注），留痕走 courseware_page_versions（版本快照），均不在此。
//
// 提供函数：
//   - AddCollabMember     加参与者（幂等：UNIQUE(courseware_id,user_id) 冲突时 DO NOTHING）
//   - RemoveCollabMember  移除单个参与者
//   - ListCollabMembers   按课件查参与者列表（LEFT JOIN users 带用户名，返回 CollabMemberView）
//   - IsCollabMember      判断某用户是否本课件参与者（权限判定核心，走唯一索引，快）
//   - ClearCollabMembers  清空某课件全部参与者（结束集体备课时调用）
//   - CountCollabMembers  统计某课件参与者数

import (
        "context"
        "fmt"

        "tedna/internal/database"
        "tedna/internal/models"
)

// AddCollabMember 给某课件的集体备课加一名参与者（幂等）。
//
// 幂等保证：UNIQUE(courseware_id,user_id) 冲突时 ON CONFLICT DO NOTHING——
// 重复拉同一个人不会报错、不会产生重复行，方便上层无脑调用。
//
// 参数：
//
//      coursewareID — 归属课件
//      userID       — 被拉入的参与者用户ID
//      initiatorID  — 本场集体备课发起者用户ID（冗余存，便于审计"谁拉的人"）
func AddCollabMember(ctx context.Context, coursewareID string, userID string, initiatorID string) error {
        sql := `INSERT INTO courseware_collab_members (id, courseware_id, user_id, initiator_id)
VALUES (gen_random_uuid(), $1, $2, $3)
ON CONFLICT (courseware_id, user_id) DO NOTHING`
        _, err := database.DB.Exec(ctx, sql, coursewareID, userID, initiatorID)
        if err != nil {
                return fmt.Errorf("加集体备课参与者失败: %w", err)
        }
        return nil
}

// RemoveCollabMember 从某课件的集体备课移除一名参与者。
// 移除后该用户立即失去对本课件的共享微调权（权限实时按本表判定）。
func RemoveCollabMember(ctx context.Context, coursewareID string, userID string) error {
        sql := `DELETE FROM courseware_collab_members WHERE courseware_id = $1 AND user_id = $2`
        _, err := database.DB.Exec(ctx, sql, coursewareID, userID)
        if err != nil {
                return fmt.Errorf("移除集体备课参与者失败: %w", err)
        }
        return nil
}

// ListCollabMembers 查询某课件的全部集体备课参与者（带用户名，按加入时间正序）。
// LEFT JOIN users 取 display_name（优先）或 username，前端无需另查用户名。
func ListCollabMembers(ctx context.Context, coursewareID string) ([]*models.CollabMemberView, error) {
        sql := `SELECT m.id, m.courseware_id, m.user_id,
COALESCE(NULLIF(u.display_name, ''), u.username, ''),
m.initiator_id, m.added_at
FROM courseware_collab_members m
LEFT JOIN users u ON u.id = m.user_id
WHERE m.courseware_id = $1
ORDER BY m.added_at ASC`
        rows, err := database.DB.Query(ctx, sql, coursewareID)
        if err != nil {
                return nil, fmt.Errorf("查询集体备课参与者列表失败: %w", err)
        }
        defer rows.Close()

        var members []*models.CollabMemberView
        for rows.Next() {
                mv := &models.CollabMemberView{}
                if err := rows.Scan(
                        &mv.ID, &mv.CoursewareID, &mv.UserID,
                        &mv.UserName,
                        &mv.InitiatorID, &mv.AddedAt,
                ); err != nil {
                        return nil, fmt.Errorf("扫描集体备课参与者行失败: %w", err)
                }
                members = append(members, mv)
        }
        return members, nil
}

// IsCollabMember 判断某用户是否某课件的集体备课参与者（权限判定核心）。
// 走 UNIQUE(courseware_id,user_id) 唯一索引，EXISTS 判定，开销极小。
// 注意：本函数只判"是否在参与者名单里"，不判 collab_state——
//
//      是否真正生效（in_session 才授权）由 service 层叠加 collab_state 判定。
func IsCollabMember(ctx context.Context, coursewareID string, userID string) (bool, error) {
        sql := `SELECT EXISTS(
SELECT 1 FROM courseware_collab_members WHERE courseware_id = $1 AND user_id = $2
)`
        var exists bool
        if err := database.DB.QueryRow(ctx, sql, coursewareID, userID).Scan(&exists); err != nil {
                return false, fmt.Errorf("判定集体备课参与者失败: %w", err)
        }
        return exists, nil
}

// ClearCollabMembers 清空某课件的全部集体备课参与者（结束集体备课时调用）。
// 结束集体备课 = collab_state 回 idle + 清空参与者名单，两步都做才彻底收权。
func ClearCollabMembers(ctx context.Context, coursewareID string) error {
        sql := `DELETE FROM courseware_collab_members WHERE courseware_id = $1`
        _, err := database.DB.Exec(ctx, sql, coursewareID)
        if err != nil {
                return fmt.Errorf("清空集体备课参与者失败: %w", err)
        }
        return nil
}

// CountCollabMembers 统计某课件的集体备课参与者数。
func CountCollabMembers(ctx context.Context, coursewareID string) (int, error) {
        var count int
        sql := `SELECT COUNT(*) FROM courseware_collab_members WHERE courseware_id = $1`
        if err := database.DB.QueryRow(ctx, sql, coursewareID).Scan(&count); err != nil {
                return 0, fmt.Errorf("统计集体备课参与者数失败: %w", err)
        }
        return count, nil
}

// ListUsersBasicByIDs 按用户ID列表批量查基本信息（供集体备课候选成员列表）。
//
// 过滤规则（候选成员应是"能参与教研的真实老师"）：
//   - 只取 status='active' 的用户；
//   - 仅排除 admin（平台管理员，非教研参与者）；
//     普通教师（viewer）可被邀请参与集体备课——参与者的共享微调权
//     由 canRefineCourseware 按"in_session + 名单内"实时判定，与账户身份无关，
//     放开 viewer 不产生越权。
//   - 入参 ids 为空直接返回空列表（不查库）。
//
// 本函数同时被通知旁路复用（resolveActorName / resolveUserName 解析操作人姓名），
// 放开 viewer 后普通教师操作产生的通知也能正确带上姓名；admin 仍排除，
// 故 admin 操作时通知退化不带名（原有行为保留）。
//
// 返回按 display_name 排序，便于前端下拉稳定展示。
func ListUsersBasicByIDs(ctx context.Context, ids []string) ([]*models.CollabCandidate, error) {
        if len(ids) == 0 {
                return []*models.CollabCandidate{}, nil
        }
        sql := `SELECT id, username, COALESCE(NULLIF(display_name, ''), username), role
FROM users
WHERE id = ANY($1)
  AND status = 'active'
  AND role <> 'admin'
ORDER BY display_name ASC`
        rows, err := database.DB.Query(ctx, sql, ids)
        if err != nil {
                return nil, fmt.Errorf("批量查候选成员失败: %w", err)
        }
        defer rows.Close()

        var out []*models.CollabCandidate
        for rows.Next() {
                c := &models.CollabCandidate{}
                if err := rows.Scan(&c.UserID, &c.Username, &c.DisplayName, &c.Role); err != nil {
                        return nil, fmt.Errorf("扫描候选成员行失败: %w", err)
                }
                out = append(out, c)
        }
        return out, nil
}

// ListJoinedCollabCoursewares 列出某用户作为【参与者】被拉入、且课件仍处于 in_session 的课件。
//
// 用途：参与者（非作者）的"我参与的集体备课"入口——让被邀请的老师找到并进入这些课件。
// JOIN courseware_collab_members（我是参与者）→ coursewares（取课件信息 + 只要 in_session）
//   → users（取作者名"这是谁的课件"）。按课件更新时间倒序。
//
// 注意：只列 collab_state='in_session'。集体备课结束后参与者自动失权，这些课件即从列表消失。
func ListJoinedCollabCoursewares(ctx context.Context, userID string) ([]*models.JoinedCollabItem, error) {
        sql := `SELECT c.id, c.title, c.subject, c.grade, c.status, c.page_count,
c.user_id, COALESCE(NULLIF(u.display_name, ''), u.username, ''),
COALESCE(c.collab_state, 'idle'), m.added_at, c.updated_at
FROM courseware_collab_members m
JOIN coursewares c ON c.id = m.courseware_id
LEFT JOIN users u ON u.id = c.user_id
WHERE m.user_id = $1
  AND COALESCE(c.collab_state, 'idle') = 'in_session'
ORDER BY c.updated_at DESC`
        rows, err := database.DB.Query(ctx, sql, userID)
        if err != nil {
                return nil, fmt.Errorf("查询我参与的集体备课失败: %w", err)
        }
        defer rows.Close()

        var out []*models.JoinedCollabItem
        for rows.Next() {
                it := &models.JoinedCollabItem{}
                if err := rows.Scan(
                        &it.ID, &it.Title, &it.Subject, &it.Grade, &it.Status, &it.PageCount,
                        &it.OwnerID, &it.OwnerName,
                        &it.CollabState, &it.JoinedAt, &it.UpdatedAt,
                ); err != nil {
                        return nil, fmt.Errorf("扫描我参与的集体备课行失败: %w", err)
                }
                it.StatusName = models.CoursewareStatusNameMap[it.Status]
                out = append(out, it)
        }
        return out, nil
}
