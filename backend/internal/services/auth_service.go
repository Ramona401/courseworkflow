package services

// 用户认证服务：登录验证 + JWT签发 + JWT验证
// Phase8日志升级：
//   - 查找用户失败（数据库错误）→ ERROR
//   - Token生成失败 → ERROR
//   - 更新登录信息失败 → WARN（不影响登录主流程，可接受）
//   - 用户登录成功 → INFO（记录username/role，便于审计）
//
// v172新增：登录/取当前用户时填充 PortalModules（门户板块可见性）
//   - 普通用户：按所属学校 settings.portal_modules 决定（缺省全开）
//   - admin：强制全开（保证管理员永远可进所有板块，便于配置和调试）
//
// 超管收口新增：JWTClaims 携带 IsSuper（超级管理员标记位）
//   - 签发 token 时从 user.IsSuper 写入 claims，使中间件 SuperAdminOnly 不查库
//     即可判定当前请求者是否超管（性能好、与 role 判定同源）。
//   - 存量 token（未带 is_super 字段）解析后 IsSuper 默认 false，即老 token 一律
//     按"非超管"处理——最坏结果是老 token 的超管需重新登录换新 token 才能进敏感入口，
//     fail-safe 收紧方向，不会误放行。

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"tedna/internal/config"
	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// 认证服务相关错误
var (
	ErrInvalidCredentials = errors.New("用户名或密码错误")
	ErrUserDisabled       = errors.New("账户已被禁用")
	ErrInvalidToken       = errors.New("无效的令牌")
	ErrTokenExpired       = errors.New("令牌已过期")
)

// JWTClaims 自定义 JWT 声明
type JWTClaims struct {
	UserID   string `json:"user_id"`  // 用户 UUID
	Username string `json:"username"` // 用户名
	Role     string `json:"role"`     // 用户角色
	// 超管收口：超级管理员标记位。仅在 Role=admin 时有意义——把"全能 admin"
	// 细分为超管(true)/二线(false)。中间件 SuperAdminOnly 直接读此字段判定，
	// 不查库。存量 token 无此字段时反序列化为 false（按非超管处理，收紧方向）。
	IsSuper              bool `json:"is_super"`
	jwt.RegisteredClaims      // 标准声明（过期时间等）
}

// TokenExpiry JWT 有效期：24小时
const TokenExpiry = 24 * time.Hour

// AuthService 认证服务
type AuthService struct {
	cfg *config.Config // 配置（含 JWTSecret）
}

// 模块日志：所有认证相关日志自动携带 module=auth 字段
var authLog = logger.WithModule("auth")

// NewAuthService 创建认证服务实例
func NewAuthService(cfg *config.Config) *AuthService {
	return &AuthService{cfg: cfg}
}

// fillUserInfoExtras 填充 UserInfo 的组织相关附加信息（Logo、组织名、门户板块可见性）
//
// v172：抽出公共逻辑，Login 与 GetCurrentUser 共用，避免重复。
//   - OrgLogoURL / OrgName：复用 GetUserOrgLogo
//   - PortalModules：admin 强制全开；其他角色按组织 settings 配置（缺省全开）
//
// 注意：IsSuper 不在此填充——它由 user.ToUserInfo() 从数据库真值直接透传，
//       与组织附加信息无关。
func fillUserInfoExtras(ctx context.Context, info *models.UserInfo) {
	// 组织 Logo 与名称
	orgLogo, orgName := repository.GetUserOrgLogo(ctx, info.ID)
	info.OrgLogoURL = orgLogo
	info.OrgName = orgName

	// 门户板块可见性
	if info.Role == models.RoleAdmin {
		// admin 永远全开，不受任何组织配置限制
		info.PortalModules = models.DefaultPortalModules()
	} else {
		info.PortalModules = repository.GetUserPortalModules(ctx, info.ID)
	}
}

// Login 登录：验证用户名密码，返回 JWT token + 用户信息
func (s *AuthService) Login(ctx context.Context, req *models.LoginRequest) (*models.LoginResponse, error) {
	// 1. 根据用户名查找用户
	user, err := repository.FindUserByUsername(ctx, req.Username)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		// ERROR：数据库查询失败，系统级错误
		authLog.Error("查找用户失败",
			"username", req.Username,
			"error", err,
		)
		return nil, err
	}

	// 2. 检查用户状态是否为 active
	if user.Status != models.StatusActive {
		authLog.Warn("禁用账户尝试登录",
			"username", user.Username,
			"user_id", user.ID,
			"status", user.Status,
		)
		return nil, ErrUserDisabled
	}

	// 3. 验证密码（bcrypt 比对）
	if !utils.CheckPassword(req.Password, user.PasswordHash) {
		authLog.Warn("密码验证失败",
			"username", req.Username,
		)
		return nil, ErrInvalidCredentials
	}

	// 4. 生成 JWT token
	token, err := s.GenerateToken(user)
	if err != nil {
		// ERROR：Token生成失败，系统级错误
		authLog.Error("生成token失败",
			"username", user.Username,
			"user_id", user.ID,
			"error", err,
		)
		return nil, err
	}

	// 5. 更新登录时间和次数
	if err := repository.UpdateLoginInfo(ctx, user.ID); err != nil {
		// WARN：更新失败不影响登录主流程，记录警告继续执行
		authLog.Warn("更新登录信息失败",
			"username", user.Username,
			"user_id", user.ID,
			"error", err,
		)
	}

	// 6. INFO：登录成功，记录关键字段便于审计
	authLog.Info("用户登录成功",
		"username", user.Username,
		"user_id", user.ID,
		"role", user.Role,
	)

	// 7. 返回 token 和用户信息（含组织 Logo/名称 + 门户板块可见性 + 超管标记）
	//    IsSuper 已由 user.ToUserInfo() 从数据库真值透传，无需在此重复赋值。
	info := user.ToUserInfo()
	fillUserInfoExtras(ctx, info)

	return &models.LoginResponse{
		Token: token,
		User:  info,
	}, nil
}

// GenerateToken 根据用户信息生成 JWT token
func (s *AuthService) GenerateToken(user *models.User) (string, error) {
	now := time.Now()

	// 构造 JWT 声明（超管收口：写入 IsSuper，使中间件不查库即可判定）
	claims := &JWTClaims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		IsSuper:  user.IsSuper,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(TokenExpiry)), // 24小时后过期
			IssuedAt:  jwt.NewNumericDate(now),                  // 签发时间
			NotBefore: jwt.NewNumericDate(now),                  // 生效时间
			Issuer:    "tedna",                                  // 签发者
		},
	}

	// 使用 HS256 签名算法创建 token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.JWTSecret))
}

// ValidateToken 验证 JWT token 并返回声明
func (s *AuthService) ValidateToken(tokenString string) (*JWTClaims, error) {
	// 解析 token
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// 确保签名方法是 HMAC
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(s.cfg.JWTSecret), nil
	})

	if err != nil {
		// 区分过期和其他错误
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}

	// 提取并验证声明
	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// GetCurrentUser 根据 JWT 声明获取当前用户完整信息
func (s *AuthService) GetCurrentUser(ctx context.Context, claims *JWTClaims) (*models.UserInfo, error) {
	user, err := repository.FindUserByID(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}

	// 再次检查用户状态（防止 token 有效但用户已被禁用）
	if user.Status != models.StatusActive {
		return nil, ErrUserDisabled
	}

	// info 由 ToUserInfo() 从数据库真值构造，IsSuper 反映当前库中真实标记
	// （即使 token 里是老值，/auth/me 拿到的也是最新库值，前端据此收口入口）
	info := user.ToUserInfo()

	// 填充组织 Logo/名称 + 门户板块可见性（与 Login 一致）
	fillUserInfoExtras(ctx, info)

	return info, nil
}
