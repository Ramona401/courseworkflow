package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"tedna/internal/database"
	"tedna/internal/models"
)

// ==================== 错误常量 ====================

var (
	ErrUserNotFound      = errors.New("用户不存在")
	ErrUsernameExists    = errors.New("用户名已存在")
	ErrCannotDisableSelf = errors.New("不能禁用自己的账户")
)

// ==================== 认证相关查询（P1-1已有） ====================

// FindUserByUsername 根据用户名查找用户（用于登录验证）
func FindUserByUsername(ctx context.Context, username string) (*models.User, error) {
	user := &models.User{}

	query := `
		SELECT id, username, display_name, password_hash,
		       role, status, last_login_at, login_count,
		       created_at, updated_at
		FROM users
		WHERE username = $1
	`

	err := database.DB.QueryRow(ctx, query, username).Scan(
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
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return user, nil
}

// FindUserByID 根据 UUID 查找用户（用于 JWT 验证后获取用户信息）
func FindUserByID(ctx context.Context, id string) (*models.User, error) {
	user := &models.User{}

	query := `
		SELECT id, username, display_name, password_hash,
		       role, status, last_login_at, login_count,
		       created_at, updated_at
		FROM users
		WHERE id = $1
	`

	err := database.DB.QueryRow(ctx, query, id).Scan(
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
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return user, nil
}

// UpdateLoginInfo 更新用户登录时间和登录次数
func UpdateLoginInfo(ctx context.Context, userID string) error {
	query := `
		UPDATE users
		SET last_login_at = $1,
		    login_count = login_count + 1,
		    updated_at = $1
		WHERE id = $2
	`

	now := time.Now()
	_, err := database.DB.Exec(ctx, query, now, userID)
	return err
}

// ==================== 用户管理 CRUD（P1-4新增） ====================

// ListUsers 获取所有用户列表（仅admin调用）
func ListUsers(ctx context.Context) ([]*models.User, error) {
	query := `
		SELECT id, username, display_name, password_hash,
		       role, status, last_login_at, login_count,
		       created_at, updated_at
		FROM users
		ORDER BY created_at ASC
	`

	rows, err := database.DB.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

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

// CheckUsernameExists 检查用户名是否已存在（用于创建/编辑时校验）
func CheckUsernameExists(ctx context.Context, username string) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM users WHERE username = $1`
	err := database.DB.QueryRow(ctx, query, username).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// CreateUser 创建新用户（仅admin调用）
func CreateUser(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (id, username, display_name, password_hash, role, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	now := time.Now()
	_, err := database.DB.Exec(ctx, query,
		user.ID,
		user.Username,
		user.DisplayName,
		user.PasswordHash,
		user.Role,
		user.Status,
		now,
		now,
	)

	return err
}

// UpdateUser 更新用户基本信息（显示名+角色）
func UpdateUser(ctx context.Context, id string, displayName string, role string) error {
	query := `
		UPDATE users
		SET display_name = $1, role = $2, updated_at = $3
		WHERE id = $4
	`

	now := time.Now()
	result, err := database.DB.Exec(ctx, query, displayName, role, now, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	return nil
}

// UpdatePassword 重置用户密码
func UpdatePassword(ctx context.Context, id string, passwordHash string) error {
	query := `
		UPDATE users
		SET password_hash = $1, updated_at = $2
		WHERE id = $3
	`

	now := time.Now()
	result, err := database.DB.Exec(ctx, query, passwordHash, now, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	return nil
}

// UpdateStatus 更新用户状态（启用/禁用）
func UpdateStatus(ctx context.Context, id string, status string) error {
	query := `
		UPDATE users
		SET status = $1, updated_at = $2
		WHERE id = $3
	`

	now := time.Now()
	result, err := database.DB.Exec(ctx, query, status, now, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	return nil
}

// ==================== 课程分配（P1-4新增） ====================

// GetUserAssignments 获取用户的课程分配列表
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
		err := rows.Scan(
			&a.ID,
			&a.UserID,
			&a.CourseCode,
			&a.AssignedBy,
			&a.AssignedAt,
		)
		if err != nil {
			return nil, err
		}
		assignments = append(assignments, a)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return assignments, nil
}

// ReplaceUserAssignments 全量替换用户的课程分配（事务操作）
// 先删除该用户所有旧分配，再批量插入新分配
func ReplaceUserAssignments(ctx context.Context, userID string, courseCodes []string, assignedBy string) error {
	// 开启事务
	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// 1. 删除该用户所有旧的课程分配
	_, err = tx.Exec(ctx, `DELETE FROM user_course_assignments WHERE user_id = $1`, userID)
	if err != nil {
		return err
	}

	// 2. 批量插入新的课程分配
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

	// 3. 提交事务
	return tx.Commit(ctx)
}
