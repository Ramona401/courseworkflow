package repository

// teacher_assistant_pref_repo.go — 老师×学科 AI 助手选择偏好数据访问层
//
// 对话式备课·助手轻量选择入口 PRD §5.1 / §6.2
//
// 对应数据库表:teacher_assistant_prefs(PK user_id+subject)
//
// 三态语义(全系统唯一真相,resolveAssistantPrompt 读取时据此分流):
//   1) GetPref 返回 found=true  且 assistantID 非空 → 老师显式选了某助手,用它。
//   2) GetPref 返回 found=true  且 assistantID = "" → 老师显式选了「系统默认(纯骨架)」,不挂助手。
//   3) GetPref 返回 found=false                     → 老师从没选过,调用方走教研员学科推荐 /
//                                                     RouteDefaultAssistant 兜底。
//
// 关键设计:pgx.ErrNoRows 不当错误处理,转成 (found=false, err=nil) —— 让上层
// 平滑走「从没选过」分支(兜底推荐),而不是把"查无记录"当成需要报错/阻塞对话的异常。
// 这是 PRD「任一步失败→纯骨架,绝不报错给老师」哲学在数据层的落地。

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"tedna/internal/database"
)

// GetPref 读取某老师在某学科下的 AI 助手选择偏好。
//
// 返回值三态约定(见文件头注释):
//   - found=false               : 无记录(从没选过)。assistantID 此时无意义(返回 "")。
//   - found=true, assistantID="" : 显式选择「系统默认」(纯骨架)。
//   - found=true, assistantID!="": 选定了具体助手。
//
// 查无记录(pgx.ErrNoRows)是正常业务态,转为 (found=false, err=nil),绝不向上抛错。
// 只有真正的数据库错误才返回非 nil err。
func GetPref(ctx context.Context, userID, subject string) (assistantID string, found bool, err error) {
	query := `
		SELECT assistant_id
		FROM teacher_assistant_prefs
		WHERE user_id = $1 AND subject = $2
	`
	err = database.DB.QueryRow(ctx, query, userID, subject).Scan(&assistantID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// 从没选过:正常态,交由调用方走学科推荐兜底。
			return "", false, nil
		}
		return "", false, fmt.Errorf("查询老师助手偏好失败: %w", err)
	}
	return assistantID, true, nil
}

// UpsertPref 写入/更新某老师在某学科下的 AI 助手选择偏好(幂等)。
//
// assistantID 语义:
//   - 非空字符串 : 老师选定了该助手 ID。
//   - 空字符串 "" : 老师显式选择「系统默认(纯骨架)」—— 这是合法且有意义的写入,
//                  与"删除记录(回到从没选过)"语义不同,必须能被持久化。
//
// 按 (user_id, subject) 主键冲突时覆盖 assistant_id 并刷新 updated_at,
// 与 ai_config_repo.UpsertConfigValue / kb_authorized_repo.AddKBAuthorized 同款幂等写法。
func UpsertPref(ctx context.Context, userID, subject, assistantID string) error {
	query := `
		INSERT INTO teacher_assistant_prefs (user_id, subject, assistant_id, updated_at)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (user_id, subject)
		DO UPDATE SET assistant_id = EXCLUDED.assistant_id, updated_at = now()
	`
	_, err := database.DB.Exec(ctx, query, userID, subject, assistantID)
	if err != nil {
		return fmt.Errorf("写入老师助手偏好失败: %w", err)
	}
	return nil
}

// DeletePref 删除某老师在某学科下的偏好记录,使其回到「从没选过」状态(下次走学科推荐兜底)。
//
// 注意与 UpsertPref(assistantID="") 的区别:
//   - DeletePref          → 无记录 → 走学科推荐/RouteDefaultAssistant 兜底。
//   - UpsertPref(.., "")  → 有记录但空 → 显式纯骨架,不挂任何助手。
//
// 本期前端未必用到,但保留此能力(零成本),供将来「恢复默认」类操作调用。
// 删除不存在的记录不报错(RowsAffected==0 视为幂等成功)。
func DeletePref(ctx context.Context, userID, subject string) error {
	query := `DELETE FROM teacher_assistant_prefs WHERE user_id = $1 AND subject = $2`
	_, err := database.DB.Exec(ctx, query, userID, subject)
	if err != nil {
		return fmt.Errorf("删除老师助手偏好失败: %w", err)
	}
	return nil
}
