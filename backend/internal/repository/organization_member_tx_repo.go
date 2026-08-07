package repository

// organization_member_tx_repo.go — 学校直接成员名单(school_members)的事务版写入
//
// 迭代一 Phase 3.2 新增：
//   organization_repo.go 已超 600 行红线，本文件独立承载 school_members 的事务版写入函数，
//   既避免给大文件继续添行，也为收尾阶段拆分 organization_repo 开个头。
//
// 设计原则：
//   - 本文件的 *Tx 函数接收调用方传入的 pgx.Tx，由调用方负责 Begin、Commit、Rollback；
//   - 函数自身只在该事务内执行数据库语句，不自行开启或提交事务；
//   - school_members.source 是稳定机器来源码，写库前必须完成规范化和长度校验；
//   - 禁止依赖数据库截断，避免超长来源值使整个用户创建事务回滚。
//
// 为什么需要事务版：
//   建用户 users 与入校 school_members 是两次数据库写入。调用方把两步放进同一事务，
//   任一步失败整体回滚，从根本上避免“用户创建成功但没有校籍”的半成品账号。
//
// 本次修复：
//   历史跨校批量来源值 admin_multi_school_batch_create 长度为 31，超过数据库
//   school_members.source varchar(30)，导致每行在入校步骤触发 SQLSTATE 22001。
//   本文件将历史值稳定转换为 admin_multi_school_batch，并在写库前拒绝其他超长值。

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

// ==================== school_members.source 稳定协议 ====================

const (
	// SchoolMemberSourceMaxLength 必须与数据库字段
	// school_members.source varchar(30) 的约束保持一致。
	SchoolMemberSourceMaxLength = 30

	// SchoolMemberSourceManual 是调用方未提供来源时的兼容默认值。
	SchoolMemberSourceManual = "manual"

	// SchoolMemberSourceAdminMultiSchoolBatch 是跨校批量导入的稳定来源标记。
	//
	// 当前值共 24 个字符，可以安全写入 varchar(30)。
	SchoolMemberSourceAdminMultiSchoolBatch = "admin_multi_school_batch"

	// schoolMemberSourceAdminMultiSchoolBatchLegacy 是历史超长来源标记。
	//
	// 仅用于兼容仍传递旧值的调用方，不允许原样写入数据库。
	schoolMemberSourceAdminMultiSchoolBatchLegacy = "admin_multi_school_batch_create"
)

// NormalizeSchoolMemberSource 归一化并校验学校成员来源标记。
//
// 规则：
//   1. 去除首尾空白；
//   2. 空值回退为 manual；
//   3. 历史跨校批量来源值转换为新的稳定短值；
//   4. 超过数据库 varchar(30) 限制时，在进入数据库前明确返回错误。
//
// PostgreSQL varchar(n)按字符数量限制，而不是按UTF-8字节数量限制，因此这里使用
// utf8.RuneCountInString，使Go层校验语义与数据库保持一致。
func NormalizeSchoolMemberSource(source string) (string, error) {
	source = strings.TrimSpace(source)

	if source == "" {
		source = SchoolMemberSourceManual
	}

	if source == schoolMemberSourceAdminMultiSchoolBatchLegacy {
		source = SchoolMemberSourceAdminMultiSchoolBatch
	}

	if utf8.RuneCountInString(source) > SchoolMemberSourceMaxLength {
		return "", fmt.Errorf(
			"学校成员来源标记不能超过%d个字符",
			SchoolMemberSourceMaxLength,
		)
	}

	return source, nil
}

// AddSchoolMemberTx 在指定事务内将用户加入学校直接成员名单。
//
// 参数：
//   - tx：调用方已开启的事务，本函数不负责提交或回滚；
//   - schoolID：学校组织ID；
//   - userID：用户ID；
//   - source：校籍来源稳定机器码，空值回退为 manual。
//
// 幂等规则：
//   使用 ON CONFLICT (school_id, user_id) DO NOTHING，同一用户重复加入同一学校时
//   不报错、不创建重复记录。
func AddSchoolMemberTx(
	ctx context.Context,
	tx pgx.Tx,
	schoolID string,
	userID string,
	source string,
) error {
	schoolID = strings.TrimSpace(schoolID)
	userID = strings.TrimSpace(userID)

	if schoolID == "" || userID == "" {
		return fmt.Errorf("schoolID 或 userID 为空")
	}

	normalizedSource, err := NormalizeSchoolMemberSource(source)
	if err != nil {
		return err
	}
	source = normalizedSource

	// 注意：err 已在上方声明，这里必须使用赋值符号“=”。
	// 使用“:=”会导致编译错误：no new variables on left side of :=。
	_, err = tx.Exec(ctx, `
		INSERT INTO school_members (
			school_id,
			user_id,
			joined_at,
			source
		)
		VALUES ($1, $2, now(), $3)
		ON CONFLICT (school_id, user_id) DO NOTHING
	`, schoolID, userID, source)
	if err != nil {
		return fmt.Errorf("加入学校成员(事务)失败: %w", err)
	}

	return nil
}
