package config

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// AppVersion 应用版本号。
//
// 修复R-03：从硬编码迁移到配置，healthHandler和其他需要版本号的地方
// 统一读取此常量。发版时只需修改此处一个位置，避免多处硬编码漏改。
//
// 审查修复C-01：版本号必须与实际发布版本保持一致。
// v0.41.0：生成质量对齐edu平台（提示词v5、预览降级、响应式、场景预留）。
// v0.43.0：迭代二Phase1-P1，课件批量生成从串行改为受控并发。
const AppVersion = "0.43.0"

// Config 全局配置结构体。
type Config struct {
	// 数据库配置。
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	// 服务器配置。
	Port    string
	GinMode string

	// CoursewareAssistantEnabled 控制教师端课件教学智能体功能。
	//
	// true：
	//   - 教师端插槽管理、方案生成、部署管理和内部预览入口开放。
	//
	// false：
	//   - 所有教师端教学智能体保留路径统一返回404；
	//   - 不构造本模块的教师端服务依赖；
	//   - 不影响课件工坊其他既有功能。
	//
	// 为兼容已经上线的教师端能力，默认值为true。
	CoursewareAssistantEnabled bool

	// CoursewareAssistantPublicRuntimeEnabled 控制公开iframe和新外部会话入口。
	//
	// true：
	//   - 公开embed页面开放；
	//   - external会话创建入口开放。
	//
	// false：
	//   - 公开embed页面返回503；
	//   - external会话创建入口返回503；
	//   - 会话读取和聊天路径仍保留，供teacher_preview内部预览复用。
	//
	// 默认值为false，保证新环境和缺失配置的环境不会意外开放公网运行时。
	// 已签发的external短时令牌可能在有效期内继续调用，最长不超过15分钟；
	// 需要立即硬停止时还应暂停或撤销对应部署。
	CoursewareAssistantPublicRuntimeEnabled bool

	// 教学智能体公开运行配置。
	//
	// PrivacySalt只用于匿名客户端和IP的用途隔离HMAC，
	// 不得与JWT、AES密钥、数据库密码或AI密钥共用。
	//
	// TokenTTLMinutes控制短时运行令牌有效期，合法范围为5—15分钟，
	// 默认15分钟。该范围必须与AssistantRuntimeTokenService保持一致，
	// 避免配置层接受的TTL在运行令牌层被判定为未配置。
	AssistantRuntimePrivacySalt     string
	AssistantRuntimeTokenTTLMinutes int

	// DisableSchedulers 禁用定时任务调度器。
	//
	// 测试环境可设为true，避免测试进程残留调度goroutine。
	DisableSchedulers bool

	// JWT配置。
	JWTSecret string

	// AES加密密钥，用于加密存储AI API Key等敏感数据。
	AESKey string

	// AI API配置。
	AIAPIBaseURL   string
	AIAPIKey       string
	AIDefaultModel string

	// CoursewareGenConcurrency 控制课件批量页面生成并发数。
	//
	// 默认4；设为1即退化为原串行行为。
	// 出现AI网关并发配额错误时可调小。
	CoursewareGenConcurrency int

	// CoursewareAssemblyImgConcurrency 控制全自动装配配图流水线并发数。
	//
	// 图片生成和HTML生成使用不同AI通道与独立信号量。
	CoursewareAssemblyImgConcurrency int
}

// Load 从环境变量加载配置。
//
// 优先读取.env文件；文件不存在时继续使用系统环境变量和systemd注入变量。
func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("未找到 .env 文件，使用系统环境变量")
	}

	cfg := &Config{
		DBHost:     getEnv("DB_HOST", "127.0.0.1"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "tedna_user"),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", "tedna"),
		Port:       getEnv("PORT", "8080"),
		GinMode:    getEnv("GIN_MODE", "release"),

		// 教师端功能保持兼容开启；公开运行时采用安全默认关闭。
		CoursewareAssistantEnabled: GetBoolEnv(
			"COURSEWARE_ASSISTANT_ENABLED",
			true,
		),
		CoursewareAssistantPublicRuntimeEnabled: GetBoolEnv(
			"COURSEWARE_ASSISTANT_PUBLIC_RUNTIME_ENABLED",
			false,
		),

		AssistantRuntimePrivacySalt: getEnv(
			"ASSISTANT_RUNTIME_PRIVACY_SALT",
			"",
		),
		AssistantRuntimeTokenTTLMinutes: GetIntEnv(
			"ASSISTANT_RUNTIME_TOKEN_TTL_MINUTES",
			15,
		),

		JWTSecret:      getEnv("JWT_SECRET", ""),
		AESKey:         getEnv("AES_KEY", ""),
		AIAPIBaseURL:   getEnv("AI_API_BASE_URL", ""),
		AIAPIKey:       getEnv("AI_API_KEY", ""),
		AIDefaultModel: getEnv("AI_DEFAULT_MODEL", "anthropic/claude-sonnet-4-5"),

		// 迭代二Phase1-P1：课件批量生成并发数，默认4。
		CoursewareGenConcurrency: GetIntEnv(
			"COURSEWARE_GEN_CONCURRENCY",
			4,
		),
		CoursewareAssemblyImgConcurrency: GetIntEnv(
			"COURSEWARE_ASSEMBLY_IMG_CONCURRENCY",
			2,
		),
	}

	// 教师端总开关关闭时，公开运行时不得单独保持开启。
	//
	// 这一收敛规则防止运维误把公开入口置为true，
	// 但同时关闭教师端控制面，导致无法从界面暂停或撤销部署。
	if !cfg.CoursewareAssistantEnabled &&
		cfg.CoursewareAssistantPublicRuntimeEnabled {
		log.Println(
			"COURSEWARE_ASSISTANT_ENABLED=false，已强制关闭COURSEWARE_ASSISTANT_PUBLIC_RUNTIME_ENABLED",
		)
		cfg.CoursewareAssistantPublicRuntimeEnabled = false
	}

	// 非法并发值统一回退为1，防止容量为0的信号量造成永久阻塞。
	if cfg.CoursewareGenConcurrency < 1 {
		log.Printf(
			"COURSEWARE_GEN_CONCURRENCY 非法值(%d)，已回退为1（串行）",
			cfg.CoursewareGenConcurrency,
		)
		cfg.CoursewareGenConcurrency = 1
	}

	if cfg.CoursewareAssemblyImgConcurrency < 1 {
		log.Printf(
			"COURSEWARE_ASSEMBLY_IMG_CONCURRENCY 非法值(%d)，已回退为1",
			cfg.CoursewareAssemblyImgConcurrency,
		)
		cfg.CoursewareAssemblyImgConcurrency = 1
	}

	// 公开运行令牌必须保持短时。
	//
	// AssistantRuntimeTokenService只接受5—15分钟；
	// 配置层必须使用完全相同的范围。
	if cfg.AssistantRuntimeTokenTTLMinutes < 5 ||
		cfg.AssistantRuntimeTokenTTLMinutes > 15 {
		log.Printf(
			"ASSISTANT_RUNTIME_TOKEN_TTL_MINUTES 非法值(%d)，已回退为15分钟",
			cfg.AssistantRuntimeTokenTTLMinutes,
		)
		cfg.AssistantRuntimeTokenTTLMinutes = 15
	}

	// 隐私盐缺失时服务仍可启动，但公开运行会话和教师预览会话创建
	// 必须由运行会话服务fail-closed。
	if cfg.AssistantRuntimePrivacySalt == "" {
		log.Println(
			"ASSISTANT_RUNTIME_PRIVACY_SALT 未配置，教学智能体运行会话创建将保持不可用",
		)
	}

	// 验证启动所需关键配置。
	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET 未配置")
	}

	if cfg.DBPassword == "" {
		log.Fatal("DB_PASSWORD 未配置")
	}

	if cfg.AESKey == "" {
		log.Fatal("AES_KEY 未配置，该密钥用于加密存储AI API密钥等敏感数据")
	}

	return cfg
}

// getEnv 获取字符串环境变量；不存在或为空时返回默认值。
func getEnv(
	key string,
	defaultValue string,
) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return defaultValue
}

// GetIntEnv 获取整型环境变量。
//
// 变量不存在或解析失败时返回默认值。
func GetIntEnv(
	key string,
	defaultValue int,
) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}

	return parsed
}

// GetBoolEnv 获取布尔环境变量。
//
// 支持的真值：true、1、yes、on。
// 支持的假值：false、0、no、off。
// 变量不存在时返回默认值；非法值记录安全日志并回退默认值。
func GetBoolEnv(
	key string,
	defaultValue bool,
) bool {
	rawValue := strings.TrimSpace(
		strings.ToLower(
			os.Getenv(key),
		),
	)

	switch rawValue {
	case "":
		return defaultValue

	case "true", "1", "yes", "on":
		return true

	case "false", "0", "no", "off":
		return false

	default:
		log.Printf(
			"%s 存在非法布尔值(%q)，已回退为%t",
			key,
			rawValue,
			defaultValue,
		)
		return defaultValue
	}
}

// GetAESKey 返回AES加密密钥。
func (c *Config) GetAESKey() string {
	return c.AESKey
}

// GetAssistantRuntimeTokenTTL 返回公开运行短时令牌有效期。
//
// 手工构造Config的测试和内部调用也必须遵守5—15分钟限制；
// 非法值统一回退15分钟，与Load及令牌服务保持完全一致。
func (c *Config) GetAssistantRuntimeTokenTTL() time.Duration {
	if c == nil ||
		c.AssistantRuntimeTokenTTLMinutes < 5 ||
		c.AssistantRuntimeTokenTTLMinutes > 15 {
		return 15 * time.Minute
	}

	return time.Duration(
		c.AssistantRuntimeTokenTTLMinutes,
	) * time.Minute
}
