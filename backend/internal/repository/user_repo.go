package repository

// user_repo.go — 用户数据访问层
//
// v64(迭代3)修改：
//   - 所有SELECT语句新增 teaching_profile 列读取
//   - 所有Scan新增 &user.TeachingProfileJSON 字段
//   - 新增 UpdateTeachingProfile：更新用户教学风格前测结果
//   - 新增 GetTeachingProfile：获取用户教学风格前测结果（解析后）
//
// v110修改：
//   - users 新增 school_id 字段支持（方案A）
//   - 新增学校管理员查询辅助函数：GetUserSchoolID / ListUsersBySchool / IsUserInSchool
//   - CreateUser / UpdateUser 支持 school_id 读写

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
	// ErrWrongPassword 旧密码验证失败（用于用户自改密码）
	ErrWrongPassword = errors.New("旧密码不正确")
)

// ==================== 内部常量：统一的SELECT列清单 ====================

// userSelectColumns 用户表查询的标准列清单
// v64：teaching_profile
// v110：school_id
const userSelectColumns = `id, username, display_name, password_hash,
       role, status, last_login_at, login_count,
       created_at, updated_at, teaching_profile, school_id`

// scanUser 统一的用户行扫描函数
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
		&user.SchoolID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

// scanUsers 统一的用户列表行扫描函数
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
			&user.SchoolID,
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

// FindUserByUsername 根据用户名查找用户（用于登录验证）
func FindUserByUsername(ctx context.Context, username string) (*models.User, error) {
	query := `SELECT ` + userSelectColumns + ` FROM users WHERE username = $1`
	return scanUser(database.DB.QueryRow(ctx, query, username))
}

// FindUserByID 根据 UUID 查找用户
func FindUserByID(ctx context.Context, id string) (*models.User, error) {
	query := `SELECT ` + userSelectColumns + ` FROM users WHERE id = $1`
	return scanUser(database.DB.QueryRow(ctx, query, id))
}

// GetUserSchoolID 获取用户所属学校ID（可为空）
func GetUserSchoolID(ctx context.Context, userID string) (*string, error) {
	var schoolID *string
	query := `SELECT school_id FROM users WHERE id = $1`
	err := database.DB.QueryRow(ctx, query, userID).Scan(&schoolID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return schoolID, nil
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

// ==================== 用户管理 CRUD（admin操作） ====================

// ListUsers 获取所有用户列表（仅admin调用）
func ListUsers(ctx context.Context) ([]*models.User, error) {
	query := `SELECT ` + userSelectColumns + ` FROM users ORDER BY created_at ASC`
	rows, err := database.DB.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanUsers(rows)
}

// ListUsersBySchool 按学校ID获取用户列表（学校管理员使用）
func ListUsersBySchool(ctx context.Context, schoolID string) ([]*models.User, error) {
	query := `SELECT ` + userSelectColumns + ` FROM users WHERE school_id = $1 ORDER BY created_at ASC`
	rows, err := database.DB.Query(ctx, query, schoolID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanUsers(rows)
}

// IsUserInSchool 校验指定用户是否属于某学校
func IsUserInSchool(ctx context.Context, userID string, schoolID string) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM users WHERE id = $1 AND school_id = $2`
	err := database.DB.QueryRow(ctx, query, userID, schoolID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// CheckUsernameExists 检查用户名是否已存在（全局唯一）
func CheckUsernameExists(ctx context.Context, username string) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM users WHERE username = $1`
	err := database.DB.QueryRow(ctx, query, username).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// CreateUser 创建新用户
func CreateUser(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (id, username, display_name, password_hash, role, status, created_at, updated_at, school_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
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
		user.SchoolID,
	)
	return err
}

// UpdateUser 更新用户基本信息（显示名+角色+school_id）
func UpdateUser(ctx context.Context, id string, displayName string, role string, schoolID *string) error {
	query := `
		UPDATE users
		SET display_name = $1, role = $2, school_id = $3, updated_at = $4
		WHERE id = $5
	`
	now := time.Now()
	result, err := database.DB.Exec(ctx, query, displayName, role, schoolID, now, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

// UpdatePassword admin重置用户密码（直接覆盖，不验证旧密码）
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

// ==================== 用户中心自助操作（AccountHandler调用） ====================

// UpdateUserDisplayName 用户自己更新显示名称
func UpdateUserDisplayName(ctx context.Context, userID string, displayName string) error {
	query := `
		UPDATE users
		SET display_name = $1, updated_at = $2
		WHERE id = $3
	`
	now := time.Now()
	result, err := database.DB.Exec(ctx, query, displayName, now, userID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

// ChangeUserPassword 用户自己修改密码（需要验证旧密码）
func ChangeUserPassword(ctx context.Context, userID string, oldPassword string, newPassword string) error {
	// 第1步：查询当前密码哈希
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

	// 第2步：验证旧密码
	if !utils.CheckPassword(oldPassword, currentHash) {
		return ErrWrongPassword
	}

	// 第3步：哈希新密码
	newHash, err := utils.HashPassword(newPassword)
	if err != nil {
		return err
	}

	// 第4步：更新密码
	now := time.Now()
	result, err := database.DB.Exec(ctx,
		`UPDATE users SET password_hash = $1, updated_at = $2 WHERE id = $3`,
		newHash, now, userID,
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
func ReplaceUserAssignments(ctx context.Context, userID string, courseCodes []string, assignedBy string) error {
	// 开启事务
	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

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

// ==================== v64(迭代3)新增：教学风格前测 ====================

// UpdateTeachingProfile 更新用户教学风格前测结果
func UpdateTeachingProfile(ctx context.Context, userID string, profileJSON string) error {
	query := `
		UPDATE users
		SET teaching_profile = $1::jsonb, updated_at = $2
		WHERE id = $3
	`
	now := time.Now()
	result, err := database.DB.Exec(ctx, query, profileJSON, now, userID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

// GetTeachingProfile 获取用户教学风格前测结果（解析后的结构体）
func GetTeachingProfile(ctx context.Context, userID string) (*models.TeachingProfile, error) {
	var profileJSON *string
	query := `SELECT teaching_profile FROM users WHERE id = $1`
	err := database.DB.QueryRow(ctx, query, userID).Scan(&profileJSON)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	if profileJSON == nil {
		return nil, nil // 未完成前测
	}
	var profile models.TeachingProfile
	if err := json.Unmarshal([]byte(*profileJSON), &profile); err != nil {
		return nil, err
	}
	return &profile, nil
}
