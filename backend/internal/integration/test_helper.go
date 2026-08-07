package integration

// test_helper.go — 集成测试核心基础设施
//
// 本文件负责：
//   1. 构造严格指向tedna_test的测试配置；
//   2. 初始化和关闭测试数据库连接池；
//   3. 通过测试库专用SECURITY DEFINER函数清空测试数据；
//   4. 写入全局公共测试种子；
//   5. 使用生产完整路由包装启动httptest服务。
//
// HTTP请求与登录辅助位于test_http_helpers.go。
// 教学智能体数据库夹具位于assistant_runtime_fixture*.go。
// 完整路由所需环境绑定位于test_config_environment.go。
//
// 安全边界：
//   - DBName硬编码为tedna_test，不能被环境变量覆盖；
//   - 每次清理前再次读取current_database；
//   - 普通业务仓储继续使用tedna_user执行；
//   - 只有数据清理通过测试库专用重置函数提升权限；
//   - 重置函数自身也会再次校验数据库名；
//   - 测试环境禁用全部定时调度器；
//   - 路由入口与生产一致，使用SetupWithAssistantRuntime；
//   - 构造完整路由前，把测试配置绑定到当前测试进程环境，
//     兼容历史Handler构造器内部的config.Load调用；
//   - 测试环境变量由testing.T自动恢复，不污染其它测试进程。

import (
	"context"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"tedna/internal/config"
	"tedna/internal/database"
	"tedna/internal/routes"
	"tedna/internal/utils"
)

// 公共测试用户固定标识。
const (
	SeedAdminID       = "00000000-0000-0000-0000-000000000001"
	SeedAdminUsername = "admin"
	SeedAdminPassword = "admin123"
	SeedAdminRole     = "admin"

	SeedOperatorID       = "00000000-0000-0000-0000-000000000002"
	SeedOperatorUsername = "operator1"
	SeedOperatorPassword = "operator123"
	SeedOperatorRole     = "operator"

	SeedSeniorID       = "00000000-0000-0000-0000-000000000003"
	SeedSeniorUsername = "senior1"
	SeedSeniorPassword = "senior123"
	SeedSeniorRole     = "senior_operator"

	SeedViewerID       = "00000000-0000-0000-0000-000000000004"
	SeedViewerUsername = "viewer1"
	SeedViewerPassword = "viewer123"
	SeedViewerRole     = "viewer"

	SeedDisabledID       = "00000000-0000-0000-0000-000000000005"
	SeedDisabledUsername = "disabled1"
	SeedDisabledPassword = "disabled123"
	SeedDisabledRole     = "viewer"
)

// testConfig 构造严格隔离的测试配置。
func testConfig() *config.Config {
	return &config.Config{
		DBHost: envOrDefault(
			"TEST_DB_HOST",
			"127.0.0.1",
		),
		DBPort: envOrDefault(
			"TEST_DB_PORT",
			"5432",
		),
		DBUser: envOrDefault(
			"TEST_DB_USER",
			"tedna_user",
		),
		DBPassword: envOrDefault(
			"TEST_DB_PASSWORD",
			"9fIbnkYABWXt3VGPv8Pn",
		),

		// 不允许任何环境变量把集成测试切换到生产库。
		DBName: "tedna_test",

		Port:              "0",
		GinMode:           "test",
		DisableSchedulers: true,

		JWTSecret:
			"test-jwt-secret-for-integration-tests-only",
		AESKey:
			"c94985251907d9a973ee517d048d8430",

		// 提供不可访问的本地占位AI配置。
		// 当前定向测试不调用AI，配置只用于完整生产路由构造。
		AIAPIBaseURL:
			"http://127.0.0.1:1/v1",
		AIAPIKey:
			"integration-test-no-network-key",
		AIDefaultModel:
			"integration-test-model",

		AssistantRuntimePrivacySalt:
			"test-assistant-runtime-privacy-salt-not-for-production",
		AssistantRuntimeTokenTTLMinutes: 15,

		CoursewareGenConcurrency:         1,
		CoursewareAssemblyImgConcurrency: 1,
	}
}

// envOrDefault 读取允许覆盖的测试连接参数。
func envOrDefault(
	key string,
	defaultValue string,
) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return defaultValue
}

// initTestDB 初始化全局测试数据库连接池。
func initTestDB(
	t *testing.T,
	cfg *config.Config,
) {
	t.Helper()

	if cfg == nil ||
		cfg.DBName != "tedna_test" {
		t.Fatal(
			"安全检查失败：测试数据库必须为tedna_test",
		)
	}

	dsnURL := &url.URL{
		Scheme: "postgres",
		User: url.UserPassword(
			cfg.DBUser,
			cfg.DBPassword,
		),
		Host: cfg.DBHost + ":" + cfg.DBPort,
		Path: "/" + cfg.DBName,
	}

	query := dsnURL.Query()
	query.Set(
		"sslmode",
		"disable",
	)
	dsnURL.RawQuery = query.Encode()

	poolConfig, err := pgxpool.ParseConfig(
		dsnURL.String(),
	)
	if err != nil {
		t.Fatalf(
			"解析测试数据库DSN失败: %v",
			err,
		)
	}

	poolConfig.MaxConns = 10
	poolConfig.MinConns = 2
	poolConfig.MaxConnLifetime = 30 * time.Minute
	poolConfig.MaxConnIdleTime = 10 * time.Minute

	pool, err := pgxpool.NewWithConfig(
		context.Background(),
		poolConfig,
	)
	if err != nil {
		t.Fatalf(
			"连接测试数据库失败: %v",
			err,
		)
	}

	if err := pool.Ping(
		context.Background(),
	); err != nil {
		pool.Close()

		t.Fatalf(
			"Ping测试数据库失败: %v",
			err,
		)
	}

	var currentDatabase string

	if err := pool.QueryRow(
		context.Background(),
		"SELECT current_database()",
	).Scan(
		&currentDatabase,
	); err != nil {
		pool.Close()

		t.Fatalf(
			"读取当前测试数据库失败: %v",
			err,
		)
	}

	if currentDatabase != "tedna_test" {
		pool.Close()

		t.Fatalf(
			"安全检查失败：实际连接数据库为%s",
			currentDatabase,
		)
	}

	database.DB = pool

	t.Cleanup(func() {
		pool.Close()

		if database.DB == pool {
			database.DB = nil
		}
	})
}

// CleanAndSeed 清空测试数据并写入公共种子。
func CleanAndSeed(
	t *testing.T,
) {
	t.Helper()

	if database.DB == nil {
		t.Fatal(
			"测试数据库连接池未初始化",
		)
	}

	ctx := context.Background()

	var currentDatabase string

	if err := database.DB.QueryRow(
		ctx,
		"SELECT current_database()",
	).Scan(
		&currentDatabase,
	); err != nil {
		t.Fatalf(
			"确认测试数据库身份失败: %v",
			err,
		)
	}

	if currentDatabase != "tedna_test" {
		t.Fatalf(
			"拒绝清理非测试数据库：%s",
			currentDatabase,
		)
	}

	// 不直接向tedna_user开放不可变表的TRUNCATE权限。
	//
	// 测试库专用函数由postgres拥有并使用SECURITY DEFINER执行，
	// 函数内部再次确认数据库名必须为tedna_test。
	_, err := database.DB.Exec(
		ctx,
		`
		SELECT public.tedna_test_reset_public_tables()
		`,
	)
	if err != nil {
		t.Fatalf(
			"调用测试数据库受控重置函数失败: %v",
			err,
		)
	}

	seedUsers(
		t,
		ctx,
	)

	seedWorkshopStages(
		t,
		ctx,
	)
}

// seedUsers 写入公共测试用户。
func seedUsers(
	t *testing.T,
	ctx context.Context,
) {
	t.Helper()

	users := []struct {
		id          string
		username    string
		displayName string
		password    string
		role        string
		status      string
		isSuper     bool
	}{
		{
			id:          SeedAdminID,
			username:    SeedAdminUsername,
			displayName: "管理员",
			password:    SeedAdminPassword,
			role:        SeedAdminRole,
			status:      "active",
			isSuper:     true,
		},
		{
			id:          SeedOperatorID,
			username:    SeedOperatorUsername,
			displayName: "操作员1",
			password:    SeedOperatorPassword,
			role:        SeedOperatorRole,
			status:      "active",
		},
		{
			id:          SeedSeniorID,
			username:    SeedSeniorUsername,
			displayName: "高级操作员1",
			password:    SeedSeniorPassword,
			role:        SeedSeniorRole,
			status:      "active",
		},
		{
			id:          SeedViewerID,
			username:    SeedViewerUsername,
			displayName: "查看者1",
			password:    SeedViewerPassword,
			role:        SeedViewerRole,
			status:      "active",
		},
		{
			id:          SeedDisabledID,
			username:    SeedDisabledUsername,
			displayName: "禁用用户1",
			password:    SeedDisabledPassword,
			role:        SeedDisabledRole,
			status:      "disabled",
		},
	}

	for _, user := range users {
		passwordHash, err := utils.HashPassword(
			user.password,
		)
		if err != nil {
			t.Fatalf(
				"哈希测试用户密码失败(user=%s): %v",
				user.username,
				err,
			)
		}

		_, err = database.DB.Exec(
			ctx,
			`
			INSERT INTO users (
				id,
				username,
				display_name,
				password_hash,
				role,
				status,
				is_super,
				created_at,
				updated_at
			)
			VALUES (
				$1,
				$2,
				$3,
				$4,
				$5,
				$6,
				$7,
				NOW(),
				NOW()
			)
			`,
			user.id,
			user.username,
			user.displayName,
			passwordHash,
			user.role,
			user.status,
			user.isSuper,
		)
		if err != nil {
			t.Fatalf(
				"插入测试用户失败(user=%s): %v",
				user.username,
				err,
			)
		}
	}
}

// seedWorkshopStages 写入系统备课阶段种子。
func seedWorkshopStages(
	t *testing.T,
	ctx context.Context,
) {
	t.Helper()

	stages := []struct {
		code      string
		name      string
		order     int
		aiRole    string
		skippable bool
	}{
		{
			code:      "analyze",
			name:      "教学分析",
			order:     1,
			aiRole:    "教学分析师",
			skippable: true,
		},
		{
			code:      "design",
			name:      "教学设计",
			order:     2,
			aiRole:    "教学设计师",
			skippable: true,
		},
		{
			code:   "write",
			name:   "教案撰写",
			order:  3,
			aiRole: "教案撰写专家",
		},
		{
			code:      "review",
			name:      "AI评审",
			order:     4,
			aiRole:    "教案评审专家",
			skippable: true,
		},
		{
			code:   "revise",
			name:   "修订定稿",
			order:  5,
			aiRole: "教案修订专家",
		},
	}

	for _, stage := range stages {
		_, err := database.DB.Exec(
			ctx,
			`
			INSERT INTO workshop_stages (
				id,
				stage_code,
				stage_name,
				stage_order,
				source,
				ai_role,
				system_prompt,
				skippable,
				status,
				created_at,
				updated_at
			)
			VALUES (
				gen_random_uuid(),
				$1,
				$2,
				$3,
				'system',
				$4,
				'',
				$5,
				'active',
				NOW(),
				NOW()
			)
			`,
			stage.code,
			stage.name,
			stage.order,
			stage.aiRole,
			stage.skippable,
		)
		if err != nil {
			t.Fatalf(
				"插入测试阶段失败(stage=%s): %v",
				stage.code,
				err,
			)
		}
	}
}

// SetupTestServer 使用生产最终路由包装启动HTTP测试服务。
func SetupTestServer(
	t *testing.T,
) (
	*httptest.Server,
	*config.Config,
) {
	t.Helper()

	cfg := testConfig()

	// 完整生产路由中仍有少量历史Handler构造器会再次调用config.Load。
	// 必须先把严格隔离的测试配置绑定到当前测试进程环境，
	// 再初始化数据库和构造路由，防止它们读取生产.env或缺失必填配置。
	bindTestConfigEnvironment(
		t,
		cfg,
	)

	initTestDB(
		t,
		cfg,
	)

	handler := routes.SetupWithAssistantRuntime(
		cfg,
	)

	server := httptest.NewServer(
		handler,
	)

	t.Cleanup(func() {
		server.Close()
	})

	return server,
		cfg
}
