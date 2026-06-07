package repository

// organization_member_tx_repo.go — 学校直接成员名单(school_members)的事务版写入
//
// 迭代一 Phase 3.2 新增：
//   organization_repo.go 已超 600 行红线，本文件独立承载 school_members 的"事务版"写入函数，
//   既避免给大文件继续添行，也为收尾阶段拆分 organization_repo 开个头。
//
// 设计原则：
//   - 本文件的 *Tx 函数接收调用方传入的 pgx.Tx，由调用方负责 Begin / Commit / Rollback；
//     函数自身只在该事务内执行一条语句，不自行开启或提交事务。
//   - 与 organization_repo.go 里的非事务版 AddSchoolMember 行为【逐字对齐】：
//     相同的 INSERT 列、相同的 ON CONFLICT (school_id, user_id) DO NOTHING 幂等语义、
//     相同的空值保护与 source 默认值。唯一区别是执行器从 database.DB 换成传入的 tx。
//
// 为什么需要事务版：
//   建用户(users 表) 与 入校(school_members 表) 是两次独立写入。历史实现是
//   "先建 user 成功，再补 AddSchoolMember，失败仅打 WARN"，并发或异常下会留下
//   "建了用户却不在本校名单"的孤儿账号。Phase 3.2 把两步包进同一事务原子提交，
//   任一步失败整体回滚，从根上杜绝孤儿。本文件提供入校那一步的事务版。

import (
        "context"
        "fmt"

        "github.com/jackc/pgx/v5"
)

// AddSchoolMemberTx 在指定事务内，将用户加入学校的直接成员名单(school_members)
//
// 与非事务版 AddSchoolMember(organization_repo.go) 的差异：
//   - 执行器：使用调用方传入的 tx(pgx.Tx)，而非全局 database.DB；
//   - 其余(列、ON CONFLICT 幂等、空值保护、source 默认 manual)完全一致。
//
// 参数：
//   tx       — 调用方已开启的事务；本函数不 Commit/Rollback，由调用方掌控
//   schoolID — 学校组织ID(organizations.id, type=school)
//   userID   — 用户ID
//   source   — 来源标记('school_admin_create'/'admin_create'/'group_member'/'migration'/'manual')；
//              传空则默认 'manual'(与非事务版一致)
//
// 幂等：ON CONFLICT (school_id, user_id) DO NOTHING —— 同一(school,user)重复写不报错、不重复插。
func AddSchoolMemberTx(ctx context.Context, tx pgx.Tx, schoolID string, userID string, source string) error {
        if schoolID == "" || userID == "" {
                return fmt.Errorf("schoolID 或 userID 为空")
        }
        if source == "" {
                source = "manual"
        }
        _, err := tx.Exec(ctx, `
                INSERT INTO school_members (school_id, user_id, joined_at, source)
                VALUES ($1, $2, now(), $3)
                ON CONFLICT (school_id, user_id) DO NOTHING
        `, schoolID, userID, source)
        if err != nil {
                return fmt.Errorf("加入学校成员(事务)失败: %w", err)
        }
        return nil
}
