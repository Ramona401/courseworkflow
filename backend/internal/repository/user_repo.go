package repository

// user_repo.go — 用户数据访问层
//
// v64(迭代3)修改：
//   - 所有SELECT语句新增 teaching_profile 列读取
//   - 所有Scan新增 &user.TeachingProfileJSON 字段
//   - 新增 UpdateTeachingProfile：更新用户教学风格前测结果
//   - 新增 GetTeachingProfile：获取用户教学风格前测结果（解析后）

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
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
