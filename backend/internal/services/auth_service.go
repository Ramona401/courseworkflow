package services

// 用户认证服务：登录验证 + JWT签发 + JWT验证
// Phase8日志升级：
//   - 查找用户失败（数据库错误）→ ERROR
//   - Token生成失败 → ERROR
//   - 更新登录信息失败 → WARN（不影响登录主流程，可接受）
//   - 用户登录成功 → INFO（记录username/role，便于审计）

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
	jwt.RegisteredClaims              // 标准声明（过期时间等）
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

	// 7. 返回 token 和用户信息
	return &models.LoginResponse{
		Token: token,
		User:  user.ToUserInfo(),
	}, nil
}

// GenerateToken 根据用户信息生成 JWT token
func (s *AuthService) GenerateToken(user *models.User) (string, error) {
	now := time.Now()

	// 构造 JWT 声明
	claims := &JWTClaims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
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

	return user.ToUserInfo(), nil
}
