package services

// auth_service.go — 用户认证、JWT与登录上下文装配
//
// 教育域隔离：
//   Login与GetCurrentUser统一调用repository.ResolveUserEducationContext；
//   一次装配组织Logo、组织名称、门户板块、education_domain、education_org_id、
//   education_domain_ready、education_domain_error和education_profile。
//
// 区域管理员：
//   - 固定域来自organization_admins.education_domain；
//   - 多个同域区域任命允许；
//   - 无任命、空值、非法值、多域冲突和数据库查询失败全部fail-closed；
//   - 身份认证仍成功，但education_domain_ready=false；
//   - 绝不回退K12，也不取第一条任命作为授权判断。
//
// JWT继续只保存用户身份和超管标记，不保存教育域。
// 因此重新登录和GET /auth/me都会读取数据库中的最新任命状态。

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

var (
	ErrInvalidCredentials = errors.New("用户名或密码错误")
	ErrUserDisabled       = errors.New("账户已被禁用")
	ErrInvalidToken       = errors.New("无效的令牌")
	ErrTokenExpired       = errors.New("令牌已过期")
)

// JWTClaims 自定义JWT声明。
type JWTClaims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	IsSuper  bool   `json:"is_super"`
	jwt.RegisteredClaims
}

const TokenExpiry = 24 * time.Hour

type AuthService struct {
	cfg *config.Config
}

var authLog = logger.WithModule("auth")

func NewAuthService(cfg *config.Config) *AuthService {
	return &AuthService{cfg: cfg}
}

// markEducationDomainUnavailable 将用户教育域设置为统一异常状态。
//
// 该方法不暴露具体数据库错误，只下发固定提示。
// EducationProfile暂用mixed画像保持结构完整；下一上下文由前端统一守卫阻断教学入口。
func markEducationDomainUnavailable(info *models.UserInfo) {
	info.EducationDomain = ""
	info.EducationOrgID = ""
	info.EducationDomainReady = false
	info.EducationDomainError =
		models.EducationDomainNotReadyMessage
	info.EducationProfile = models.EducationProfileForDomain(
		models.EducationDomainMixed,
	)
}

// applyResolvedUserEducationContext 把Repository解析结果装配到UserInfo。
//
// 区域管理员额外执行防御性复核：即使未来Repository被错误修改并返回mixed、
// common或空值，本层仍会标记未就绪，不让非法域进入登录响应授权上下文。
func applyResolvedUserEducationContext(
	info *models.UserInfo,
	educationContext *models.UserEducationContext,
) {
	if educationContext == nil {
		markEducationDomainUnavailable(info)
		return
	}

	info.OrgLogoURL = educationContext.OrganizationLogo
	info.OrgName = educationContext.OrganizationName
	info.EducationOrgID = educationContext.OrganizationID

	if info.Role == models.RoleRegionAdmin &&
		!models.IsTeachingEducationDomain(
			educationContext.EducationDomain,
		) {
		markEducationDomainUnavailable(info)
		info.PortalModules = models.DefaultPortalModules()
		return
	}

	info.EducationDomain = educationContext.EducationDomain
	info.EducationDomainReady = true
	info.EducationDomainError = ""
	info.EducationProfile = models.EducationProfileForDomain(
		educationContext.EducationDomain,
	)

	if info.Role == models.RoleAdmin {
		info.PortalModules = models.DefaultPortalModules()
	} else {
		info.PortalModules = educationContext.PortalModules
	}
}

// fillUserInfoExtras 填充登录用户的组织、门户与教育域上下文。
func fillUserInfoExtras(ctx context.Context, info *models.UserInfo) {
	educationContext, err := repository.ResolveUserEducationContext(
		ctx,
		info.ID,
		info.Role,
	)
	if err != nil {
		// 区域管理员教育域异常不阻断身份认证，但必须fail-closed：
		// 清空教育域并显式下发未就绪状态，绝不默认K12。
		if info.Role == models.RoleRegionAdmin {
			orgLogo, orgName := repository.GetUserOrgLogo(
				ctx,
				info.ID,
			)
			info.OrgLogoURL = orgLogo
			info.OrgName = orgName
			info.PortalModules = models.DefaultPortalModules()
			markEducationDomainUnavailable(info)

			authLog.Warn(
				"区域管理员教育域解析失败，已按未就绪状态下发",
				"user_id", info.ID,
				"role", info.Role,
				"error", err,
			)
			return
		}

		// 非区域管理员继续保持当前兼容行为。
		// 本上下文只收口区域管理员登录，不提前扩大到其它角色。
		authLog.Warn(
			"解析用户教育域失败，使用兼容默认",
			"user_id", info.ID,
			"role", info.Role,
			"error", err,
		)

		orgLogo, orgName := repository.GetUserOrgLogo(ctx, info.ID)
		info.OrgLogoURL = orgLogo
		info.OrgName = orgName

		if info.Role == models.RoleAdmin ||
			info.Role == models.RoleDistrictInspector {
			info.PortalModules = models.DefaultPortalModules()
			info.EducationDomain = models.EducationDomainMixed
		} else {
			info.PortalModules = repository.GetUserPortalModules(
				ctx,
				info.ID,
			)
			info.EducationDomain = models.EducationDomainK12
		}

		info.EducationDomainReady = true
		info.EducationDomainError = ""
		info.EducationProfile = models.EducationProfileForDomain(
			info.EducationDomain,
		)
		return
	}

	applyResolvedUserEducationContext(
		info,
		educationContext,
	)

	if educationContext.DomainConflict {
		authLog.Warn(
			"用户同时属于多个具体教育域，已按确定性首个教学组织解析",
			"user_id", info.ID,
			"role", info.Role,
			"education_domain", educationContext.EducationDomain,
			"education_org_id", educationContext.OrganizationID,
		)
	}
}

// Login 登录。
func (s *AuthService) Login(
	ctx context.Context,
	req *models.LoginRequest,
) (*models.LoginResponse, error) {
	user, err := repository.FindUserByUsername(ctx, req.Username)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		authLog.Error(
			"查找用户失败",
			"username", req.Username,
			"error", err,
		)
		return nil, err
	}

	if user.Status != models.StatusActive {
		authLog.Warn(
			"禁用账户尝试登录",
			"username", user.Username,
			"user_id", user.ID,
			"status", user.Status,
		)
		return nil, ErrUserDisabled
	}

	if !utils.CheckPassword(req.Password, user.PasswordHash) {
		authLog.Warn(
			"密码验证失败",
			"username", req.Username,
		)
		return nil, ErrInvalidCredentials
	}

	token, err := s.GenerateToken(user)
	if err != nil {
		authLog.Error(
			"生成token失败",
			"username", user.Username,
			"user_id", user.ID,
			"error", err,
		)
		return nil, err
	}

	if err := repository.UpdateLoginInfo(ctx, user.ID); err != nil {
		authLog.Warn(
			"更新登录信息失败",
			"username", user.Username,
			"user_id", user.ID,
			"error", err,
		)
	}

	info := user.ToUserInfo()
	fillUserInfoExtras(ctx, info)

	authLog.Info(
		"用户登录成功",
		"username", user.Username,
		"user_id", user.ID,
		"role", user.Role,
		"education_domain", info.EducationDomain,
		"education_domain_ready", info.EducationDomainReady,
		"education_org_id", info.EducationOrgID,
	)

	return &models.LoginResponse{
		Token: token,
		User:  info,
	}, nil
}

// GenerateToken 根据用户信息生成JWT。
func (s *AuthService) GenerateToken(user *models.User) (string, error) {
	now := time.Now()

	claims := &JWTClaims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		IsSuper:  user.IsSuper,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(TokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "tedna",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.JWTSecret))
}

// ValidateToken 验证JWT。
func (s *AuthService) ValidateToken(
	tokenString string,
) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&JWTClaims{},
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, ErrInvalidToken
			}
			return []byte(s.cfg.JWTSecret), nil
		},
	)

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// GetCurrentUser 根据JWT声明返回当前用户完整信息。
func (s *AuthService) GetCurrentUser(
	ctx context.Context,
	claims *JWTClaims,
) (*models.UserInfo, error) {
	user, err := repository.FindUserByID(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}

	if user.Status != models.StatusActive {
		return nil, ErrUserDisabled
	}

	info := user.ToUserInfo()
	fillUserInfoExtras(ctx, info)

	return info, nil
}
