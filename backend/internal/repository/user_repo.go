package repository

// user_repo.go — 用户数据访问层
//
// v64(迭代3)修改：
//   - 所有SELECT语句新增 teaching_profile 列读取
//   - 所有Scan新增 &user.TeachingProfileJSON 字段
//   - 新增 UpdateTeachingProfile：更新用户教学风格前测结果
//   - 新增 GetTeachingProfile：获取用户教学风格前测结果（解析后）
//
// 迭代一 Phase 3.2 新增（本次）：
//   - CreateUserTx     ：CreateUser 的事务版（接收调用方传入的 pgx.Tx，供"建用户+入校"原子事务编排）
//   - IsUniqueViolation：判断 error 是否为 PostgreSQL 唯一约束冲突(SQLSTATE 23505)
//                        用于方案A——并发下两请求同时插入同名用户时，靠 users_username_key
//                        唯一约束兜底，撞约束后翻译成"用户名已存在"友好错误。
//   原有全部函数(CreateUser/FindUserByID/...)一字未改，仅在文件末尾追加新函数。

import (
        "context"
        "encoding/json"
        "errors"
        "time"

        "github.com/jackc/pgx/v5"
        "github.com/jackc/pgx/v5/pgconn"
        "tedna/internal/database"
        "tedna/internal/models"
        "tedna/internal/utils"
)

// ==================== 错误常量 ====================

var (
        ErrUserNotFound      = errors.New("用户不存在")
        ErrUsernameExists    = errors.New("用户名已存在")
        ErrCannotDisableSelf = errors.New("不能禁用自己的账户")
        ErrWrongPassword     = errors.New("旧密码不正确")
)

// ==================== SELECT列清单 ====================

// userSelectColumns 用户表查询标准列
const userSelectColumns = `id, username, display_name, password_hash,
       role, status, last_login_at, login_count,
       created_at, updated_at, teaching_profile`

// scanUser 扫描单行用户
func scanUser(row pgx.Row) (*models.User, error) {
        user := &models.User{}
        err := row.Scan(
                &user.ID,
                &user.Username,
                &user.DisplayName,
                &user.PasswordHash,
                &user.Role,
                &user.Status,
                &user.LastLoginAt,
                &user.LoginCount,
                &user.CreatedAt,
                &user.UpdatedAt,
                &user.TeachingProfileJSON,
        )
        if err != nil {
                if errors.Is(err, pgx.ErrNoRows) {
                        return nil, ErrUserNotFound
                }
                return nil, err
        }
        return user, nil
}

// scanUsers 扫描多行用户
func scanUsers(rows pgx.Rows) ([]*models.User, error) {
        var users []*models.User
        for rows.Next() {
                user := &models.User{}
                err := rows.Scan(
                        &user.ID,
                        &user.Username,
                        &user.DisplayName,
                        &user.PasswordHash,
                        &user.Role,
                        &user.Status,
                        &user.LastLoginAt,
                        &user.LoginCount,
                        &user.CreatedAt,
                        &user.UpdatedAt,
                        &user.TeachingProfileJSON,
                )
                if err != nil {
                        return nil, err
                }
                users = append(users, user)
        }
        if err := rows.Err(); err != nil {
                return nil, err
        }
        return users, nil
}

// ==================== 认证相关查询 ====================

func FindUserByUsername(ctx context.Context, username string) (*models.User, error) {
        query := `SELECT ` + userSelectColumns + ` FROM users WHERE username = $1`
        return scanUser(database.DB.QueryRow(ctx, query, username))
}

func FindUserByID(ctx context.Context, id string) (*models.User, error) {
        query := `SELECT ` + userSelectColumns + ` FROM users WHERE id = $1`
        return scanUser(database.DB.QueryRow(ctx, query, id))
}

func UpdateLoginInfo(ctx context.Context, userID string) error {
        query := `
                UPDATE users
                SET last_login_at = $1, login_count = login_count + 1, updated_at = $1
                WHERE id = $2
        `
        _, err := database.DB.Exec(ctx, query, time.Now(), userID)
        return err
}

// ==================== 用户管理 CRUD ====================

func ListUsers(ctx context.Context) ([]*models.User, error) {
        query := `SELECT ` + userSelectColumns + ` FROM users ORDER BY created_at ASC`
        rows, err := database.DB.Query(ctx, query)
        if err != nil {
                return nil, err
        }
        defer rows.Close()
        return scanUsers(rows)
}

func CheckUsernameExists(ctx context.Context, username string) (bool, error) {
        var count int
        err := database.DB.QueryRow(ctx,
                `SELECT COUNT(*) FROM users WHERE username = $1`, username,
        ).Scan(&count)
        if err != nil {
                return false, err
        }
        return count > 0, nil
}

func CreateUser(ctx context.Context, user *models.User) error {
        query := `
                INSERT INTO users (id, username, display_name, password_hash, role, status, created_at, updated_at)
                VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
        `
        now := time.Now()
        _, err := database.DB.Exec(ctx, query,
                user.ID, user.Username, user.DisplayName, user.PasswordHash,
                user.Role, user.Status, now, now,
        )
        return err
}

func UpdateUser(ctx context.Context, id string, displayName string, role string) error {
        query := `
                UPDATE users
                SET display_name = $1, role = $2, updated_at = $3
                WHERE id = $4
        `
        result, err := database.DB.Exec(ctx, query, displayName, role, time.Now(), id)
        if err != nil {
                return err
        }
        if result.RowsAffected() == 0 {
                return ErrUserNotFound
        }
        return nil
}

func UpdatePassword(ctx context.Context, id string, passwordHash string) error {
        result, err := database.DB.Exec(ctx,
                `UPDATE users SET password_hash = $1, updated_at = $2 WHERE id = $3`,
                passwordHash, time.Now(), id,
        )
        if err != nil {
                return err
        }
        if result.RowsAffected() == 0 {
                return ErrUserNotFound
        }
        return nil
}

func UpdateStatus(ctx context.Context, id string, status string) error {
        result, err := database.DB.Exec(ctx,
                `UPDATE users SET status = $1, updated_at = $2 WHERE id = $3`,
                status, time.Now(), id,
        )
        if err != nil {
                return err
        }
        if result.RowsAffected() == 0 {
                return ErrUserNotFound
        }
        return nil
}

// ==================== 用户中心自助操作 ====================

func UpdateUserDisplayName(ctx context.Context, userID string, displayName string) error {
        result, err := database.DB.Exec(ctx,
                `UPDATE users SET display_name = $1, updated_at = $2 WHERE id = $3`,
                displayName, time.Now(), userID,
        )
        if err != nil {
                return err
        }
        if result.RowsAffected() == 0 {
                return ErrUserNotFound
        }
        return nil
}

func ChangeUserPassword(ctx context.Context, userID string, oldPassword string, newPassword string) error {
        var currentHash string
        err := database.DB.QueryRow(ctx,
                `SELECT password_hash FROM users WHERE id = $1`, userID,
        ).Scan(&currentHash)
        if err != nil {
                if errors.Is(err, pgx.ErrNoRows) {
                        return ErrUserNotFound
                }
                return err
        }
        if !utils.CheckPassword(oldPassword, currentHash) {
                return ErrWrongPassword
        }
        newHash, err := utils.HashPassword(newPassword)
        if err != nil {
                return err
        }
        result, err := database.DB.Exec(ctx,
                `UPDATE users SET password_hash = $1, updated_at = $2 WHERE id = $3`,
                newHash, time.Now(), userID,
        )
        if err != nil {
                return err
        }
        if result.RowsAffected() == 0 {
                return ErrUserNotFound
        }
        return nil
}

// ==================== 课程分配 ====================

func GetUserAssignments(ctx context.Context, userID string) ([]*models.CourseAssignment, error) {
        query := `
                SELECT id, user_id, course_code, assigned_by, assigned_at
                FROM user_course_assignments
                WHERE user_id = $1
                ORDER BY assigned_at ASC
        `
        rows, err := database.DB.Query(ctx, query, userID)
        if err != nil {
                return nil, err
        }
        defer rows.Close()

        var assignments []*models.CourseAssignment
        for rows.Next() {
                a := &models.CourseAssignment{}
                if err := rows.Scan(&a.ID, &a.UserID, &a.CourseCode, &a.AssignedBy, &a.AssignedAt); err != nil {
                        return nil, err
                }
                assignments = append(assignments, a)
        }
        if err := rows.Err(); err != nil {
                return nil, err
        }
        return assignments, nil
}

func ReplaceUserAssignments(ctx context.Context, userID string, courseCodes []string, assignedBy string) error {
        tx, err := database.DB.Begin(ctx)
        if err != nil {
                return err
        }
        defer func() { _ = tx.Rollback(ctx) }()

        _, err = tx.Exec(ctx, `DELETE FROM user_course_assignments WHERE user_id = $1`, userID)
        if err != nil {
                return err
        }

        if len(courseCodes) > 0 {
                now := time.Now()
                for _, code := range courseCodes {
                        _, err = tx.Exec(ctx,
                                `INSERT INTO user_course_assignments (id, user_id, course_code, assigned_by, assigned_at)
                                 VALUES (gen_random_uuid(), $1, $2, $3, $4)`,
                                userID, code, assignedBy, now,
                        )
                        if err != nil {
                                return err
                        }
                }
        }
        return tx.Commit(ctx)
}

// ==================== 教学风格前测 ====================

func UpdateTeachingProfile(ctx context.Context, userID string, profileJSON string) error {
        result, err := database.DB.Exec(ctx,
                `UPDATE users SET teaching_profile = $1::jsonb, updated_at = $2 WHERE id = $3`,
                profileJSON, time.Now(), userID,
        )
        if err != nil {
                return err
        }
        if result.RowsAffected() == 0 {
                return ErrUserNotFound
        }
        return nil
}

func GetTeachingProfile(ctx context.Context, userID string) (*models.TeachingProfile, error) {
        var profileJSON *string
        err := database.DB.QueryRow(ctx,
                `SELECT teaching_profile FROM users WHERE id = $1`, userID,
        ).Scan(&profileJSON)
        if err != nil {
                if errors.Is(err, pgx.ErrNoRows) {
                        return nil, ErrUserNotFound
                }
                return nil, err
        }
        if profileJSON == nil {
                return nil, nil
        }
        var profile models.TeachingProfile
        if err := json.Unmarshal([]byte(*profileJSON), &profile); err != nil {
                return nil, err
        }
        return &profile, nil
}

// ==================== 迭代一 Phase 3.2 新增：事务版建用户 + 唯一约束判定 ====================

// CreateUserTx 在指定事务内插入一条用户记录（CreateUser 的事务版）
//
// 与非事务版 CreateUser 的差异：
//   - 执行器：使用调用方传入的 tx(pgx.Tx)，而非全局 database.DB；
//   - 其余(INSERT 列、now() 时间戳处理)完全一致。
//
// 用途：
//   供 service 层把"建用户(users) + 入校(school_members)"包进同一事务原子提交。
//   本函数不 Begin/Commit/Rollback，事务生命周期由调用方掌控。
//
// 并发安全（方案A）：
//   若并发下另一请求已插入同名用户，本 INSERT 会撞 users_username_key 唯一约束，
//   返回 PostgreSQL 错误码 23505。调用方应在 Commit 前用 IsUniqueViolation 判定该错误，
//   翻译成业务层"用户名已存在"。本函数原样返回底层 error 不做翻译（保持职责单一）。
func CreateUserTx(ctx context.Context, tx pgx.Tx, user *models.User) error {
        query := `
                INSERT INTO users (id, username, display_name, password_hash, role, status, created_at, updated_at)
                VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
        `
        now := time.Now()
        _, err := tx.Exec(ctx, query,
                user.ID, user.Username, user.DisplayName, user.PasswordHash,
                user.Role, user.Status, now, now,
        )
        return err
}

// IsUniqueViolation 判断 error 是否为 PostgreSQL 唯一约束冲突(SQLSTATE 23505)
//
// 用于方案A的并发兜底：事务内 INSERT 撞 users_username_key 时，
// 上层据此把底层数据库错误翻译成友好的"用户名已存在"。
//
// 实现：解包到 *pgconn.PgError，比对 Code == "23505"。
// 不依赖具体约束名(constraint_name)，仅认 SQLSTATE，pgx 的标准做法。
func IsUniqueViolation(err error) bool {
        if err == nil {
                return false
        }
        var pgErr *pgconn.PgError
        if errors.As(err, &pgErr) {
                return pgErr.Code == "23505"
        }
        return false
}
