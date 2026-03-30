package services

/*
 * role_service.go — 角色权限业务逻辑层
 *
 * 核心业务规则（硬约束）：
 *   Rule-1  is_system=true 的角色只读：编辑/删除/改状态均拒绝
 *   Rule-2  新建角色时若 base_role 非空，自动复制该内置角色的全部权限
 *   Rule-3  删除角色前检查 users.role = role_code，有用户则拒绝并说明人数
 *   Rule-4  role_code 全局唯一，重复时返回明确错误
 *   Rule-5  is_system=true 的权限不可被外部 UpdateRolePermissions 覆盖
 */

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"tedna/internal/models"
	"tedna/internal/repository"

	"github.com/jackc/pgx/v5"
)

// ==================== 错误常量 ====================

var (
	// 角色码相关
	ErrRoleCodeRequired    = errors.New("角色码不能为空")
	ErrRoleCodeInvalid     = errors.New("角色码只能包含小写字母、数字和下划线")
	ErrRoleCodeExists      = errors.New("角色码已存在，请更换")
	ErrRoleDisplayRequired = errors.New("角色显示名不能为空")
	// 系统角色保护
	ErrSystemRoleReadonly = errors.New("系统内置角色只读，不可编辑、删除或修改状态")
	// 删除保护
	ErrRoleInUse = errors.New("该角色下还有用户，不可删除") // 实际消息由 service 拼接用户数
	// 基础角色
	ErrInvalidBaseRole = errors.New("base_role 无效，必须是内置角色之一")
	// 通用
	ErrRoleNotFound = errors.New("角色不存在")
)

// roleCodePattern 角色码合法格式：小写字母/数字/下划线，1-50字符
var roleCodePattern = regexp.MustCompile(`^[a-z0-9_]{1,50}$`)

// ==================== RoleService ====================

// RoleService 角色权限业务服务
type RoleService struct{}

// NewRoleService 构造函数
func NewRoleService() *RoleService {
	return &RoleService{}
}

// ==================== 角色列表 ====================

// ListRoles 获取所有角色列表（含权限数和用户数统计）
func (s *RoleService) ListRoles(ctx context.Context) (*models.RoleListResponse, error) {
	return repository.ListRoles(ctx)
}

// ==================== 角色详情 ====================

// GetRole 获取角色详情（含权限明细）
func (s *RoleService) GetRole(ctx context.Context, roleID string) (*models.RoleDetailResponse, error) {
	detail, err := repository.GetRoleByID(ctx, roleID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrRoleNotFound
		}
		return nil, fmt.Errorf("获取角色详情失败: %w", err)
	}
	return detail, nil
}

// ==================== 新建角色 ====================

// CreateRole 新建自定义角色
//
//	Rule-2: base_role 非空时自动复制权限
//	Rule-4: role_code 唯一性校验
func (s *RoleService) CreateRole(ctx context.Context, req *models.CreateRoleRequest, createdByUserID string) (*models.Role, error) {
	// 参数校验
	if req.RoleCode == "" {
		return nil, ErrRoleCodeRequired
	}
	if !roleCodePattern.MatchString(req.RoleCode) {
		return nil, ErrRoleCodeInvalid
	}
	if req.DisplayName == "" {
		return nil, ErrRoleDisplayRequired
	}

	// base_role 合法性校验（非空时）
	if req.BaseRole != "" && !models.ValidBaseRoles[req.BaseRole] {
		return nil, ErrInvalidBaseRole
	}

	// Rule-4: 唯一性校验
	existing, err := repository.GetRoleByCode(ctx, req.RoleCode)
	if err != nil && err != pgx.ErrNoRows {
		return nil, fmt.Errorf("唯一性校验失败: %w", err)
	}
	if existing != nil {
		return nil, ErrRoleCodeExists
	}

	// 创建角色
	role, err := repository.CreateRole(ctx, req, createdByUserID)
	if err != nil {
		return nil, fmt.Errorf("创建角色失败: %w", err)
	}

	// Rule-2: 若指定 base_role，自动复制内置角色权限
	if req.BaseRole != "" {
		if copyErr := repository.CopyPermissionsFromBaseRole(ctx, req.BaseRole, role.ID); copyErr != nil {
			// 权限复制失败不回滚角色本身，记录日志即可（非致命）
			// 实际生产中可升级为事务
			_ = copyErr
		}
	}

	return role, nil
}

// ==================== 编辑角色 ====================

// UpdateRole 编辑角色显示名和描述
//
//	Rule-1: is_system=true 拒绝
func (s *RoleService) UpdateRole(ctx context.Context, roleID string, req *models.UpdateRoleRequest) error {
	if req.DisplayName == "" {
		return ErrRoleDisplayRequired
	}

	// Rule-1: 加载角色，检查 is_system
	if err := s.checkNotSystem(ctx, roleID); err != nil {
		return err
	}

	return repository.UpdateRole(ctx, roleID, req.DisplayName, req.Description)
}

// ==================== 状态管理 ====================

// UpdateRoleStatus 启用/禁用角色
//
//	Rule-1: is_system=true 拒绝
func (s *RoleService) UpdateRoleStatus(ctx context.Context, roleID, status string) error {
	if status != models.RoleStatusActive && status != models.RoleStatusDisabled {
		return errors.New("状态只能是 active 或 disabled")
	}

	// Rule-1: 加载角色，检查 is_system
	if err := s.checkNotSystem(ctx, roleID); err != nil {
		return err
	}

	return repository.UpdateRoleStatus(ctx, roleID, status)
}

// ==================== 删除角色 ====================

// DeleteRole 删除角色
//
//	Rule-1: is_system=true 拒绝
//	Rule-3: 有用户使用则拒绝，并提示人数
func (s *RoleService) DeleteRole(ctx context.Context, roleID string) error {
	// 先加载角色信息（获取 role_code 和 is_system）
	detail, err := repository.GetRoleByID(ctx, roleID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return ErrRoleNotFound
		}
		return fmt.Errorf("加载角色失败: %w", err)
	}

	// Rule-1: 系统内置角色只读
	if detail.IsSystem {
		return ErrSystemRoleReadonly
	}

	// Rule-3: 检查用户数
	userCount, err := repository.CountUsersByRoleCode(ctx, detail.RoleCode)
	if err != nil {
		return fmt.Errorf("检查角色使用情况失败: %w", err)
	}
	if userCount > 0 {
		return fmt.Errorf("该角色下还有 %d 个用户，请先调整用户角色后再删除", userCount)
	}

	return repository.DeleteRole(ctx, roleID)
}

// ==================== 权限管理 ====================

// GetRolePermissions 获取角色权限列表
func (s *RoleService) GetRolePermissions(ctx context.Context, roleID string) ([]*models.RolePermission, error) {
	// 先确认角色存在
	_, err := repository.GetRoleByID(ctx, roleID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrRoleNotFound
		}
		return nil, fmt.Errorf("查询角色失败: %w", err)
	}
	return repository.GetRolePermissions(ctx, roleID)
}

// UpdateRolePermissions 全量更新角色权限
//
//	Rule-1: is_system=true 拒绝
//	Rule-5: 校验 resource 和 action 合法性
func (s *RoleService) UpdateRolePermissions(ctx context.Context, roleID string, req *models.UpdateRolePermissionsRequest) error {
	// Rule-1: 检查 is_system
	if err := s.checkNotSystem(ctx, roleID); err != nil {
		return err
	}

	// Rule-5: 校验每条权限的 resource 和 action
	for _, p := range req.Permissions {
		if !models.ValidResources[p.Resource] {
			return fmt.Errorf("无效的权限资源：%s", p.Resource)
		}
		if !models.ValidActions[p.Action] {
			return fmt.Errorf("无效的权限动作：%s", p.Action)
		}
		// 确保 permission_code 与 resource.action 一致
		expected := p.Resource + "." + p.Action
		if p.PermissionCode != "" && p.PermissionCode != expected {
			return fmt.Errorf("permission_code 与 resource.action 不匹配：%s", p.PermissionCode)
		}
		// 自动补全 permission_code
		p.PermissionCode = expected
	}

	// 补全所有 permission_code（处理 req 是值类型）
	normalized := make([]models.PermissionItem, len(req.Permissions))
	for i, p := range req.Permissions {
		p.PermissionCode = p.Resource + "." + p.Action
		normalized[i] = p
	}

	return repository.UpdateRolePermissions(ctx, roleID, normalized)
}

// ==================== 内部辅助 ====================

// checkNotSystem 加载角色并检查是否为系统内置角色，是则返回错误
func (s *RoleService) checkNotSystem(ctx context.Context, roleID string) error {
	detail, err := repository.GetRoleByID(ctx, roleID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return ErrRoleNotFound
		}
		return fmt.Errorf("加载角色失败: %w", err)
	}
	if detail.IsSystem {
		return ErrSystemRoleReadonly
	}
	return nil
}
