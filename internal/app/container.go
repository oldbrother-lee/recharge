package app

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"recharge-go/configs"
	"recharge-go/internal/middleware"
	"recharge-go/internal/repository"
	notificationRepo "recharge-go/internal/repository/notification"
	"recharge-go/internal/service"
	notificationService "recharge-go/internal/service/notification"
	"recharge-go/internal/service/platform"
	"recharge-go/pkg/database"
	"recharge-go/pkg/lock"
	"recharge-go/pkg/log"
	"recharge-go/pkg/metrics"
	pkgMiddleware "recharge-go/pkg/middleware"
	"recharge-go/pkg/queue"
	redisx "recharge-go/pkg/redis"
	"time"

	"github.com/hibiken/asynq"
	redisc "github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Container 依赖注入容器
type Container struct {
	config             *configs.Config
	db                 *gorm.DB
	redis              *redisc.Client
	redisClient        *redisc.Client
	queue              *asynq.Client
	taskQueue          queue.Queue
	repositories       *Repositories
	services           *Services
	controllers        *Controllers
	logger             *zap.Logger
	metricsManager     *metrics.MetricsManager
	securityMiddleware *pkgMiddleware.SecurityMiddleware
	databaseManager    *database.DatabaseManager
	serviceName        string
}

// Repositories 仓储集合
type Repositories struct {
	User                   *repository.UserRepository
	Order                  repository.OrderRepository
	OrderStatistics        repository.OrderStatisticsRepository
	Platform               repository.PlatformRepository
	PlatformAPI            repository.PlatformAPIRepository
	PlatformAPIParam       repository.PlatformAPIParamRepository
	PlatformAccount        *repository.PlatformAccountRepository
	PlatformAccountVariant repository.PlatformAccountVariantRepository
	Product                repository.ProductRepository
	ProductType            *repository.ProductTypeRepository         // 添加ProductType repository
	ProductTypeCategory    *repository.ProductTypeCategoryRepository // 添加ProductTypeCategory repository
	ProductAPIRelation     repository.ProductAPIRelationRepository
	Retry                  repository.RetryRepository
	CallbackLog            repository.CallbackLogRepository
	BalanceLog             *repository.BalanceLogRepository
	BalanceQueryRecord     repository.BalanceQueryRecordRepository // 添加余额查询记录 repository
	Notification           notificationRepo.Repository
	TaskConfig             *repository.TaskConfigRepository
	TaskOrder              *repository.TaskOrderRepository
	DaichongOrder          *repository.DaichongOrderRepository
	PhoneLocation          *repository.PhoneLocationRepository
	Permission             *repository.PermissionRepository    // 添加Permission repository
	Role                   *repository.RoleRepository          // 添加Role repository
	UserLog                *repository.UserLogRepository       // 添加UserLog repository
	CreditLog              *repository.CreditLogRepository     // 添加CreditLog repository
	SystemConfig           *repository.SystemConfigRepository  // 添加SystemConfig repository
	ExternalAPIKey         repository.ExternalAPIKeyRepository // 添加ExternalAPIKey repository
	OrderException         repository.OrderExceptionRepository // 添加OrderException repository

}

// Services 服务集合
type Services struct {
	User                   *service.UserService
	UserGrade              *service.UserGradeService
	UserTag                *service.UserTagService
	Order                  service.OrderService
	Platform               *service.PlatformService
	PlatformService        *service.PlatformService // 添加这个字段
	Recharge               service.RechargeService
	Retry                  *service.RetryService // 添加Retry服务
	Notification           notificationService.NotificationService
	Statistics             service.StatisticsService
	StatisticsTask         *service.StatisticsTask // 添加StatisticsTask服务
	Balance                *service.BalanceService // 添加Balance服务
	PlatformAccountBalance *service.PlatformAccountBalanceService
	UnifiedRefund          *service.UnifiedRefundService // 添加统一退款服务
	UnifiedOrder           *service.UnifiedOrderService  // 添加统一订单服务
	Task                   *service.TaskService
	TaskConfigNotifier     *service.TaskConfigNotifier       // 添加任务配置通知器
	PhoneLocation          *service.PhoneLocationService     // 添加PhoneLocation服务
	Product                *service.ProductService           // 添加Product服务
	ProductType            *service.ProductTypeService       // 添加ProductType服务
	PlatformAPI            service.PlatformAPIService        // 添加PlatformAPI服务
	PlatformAPIParam       service.PlatformAPIParamService   // 添加PlatformAPIParam服务
	ProductAPIRelation     service.ProductAPIRelationService // 添加ProductAPIRelation服务
	UserLog                *service.UserLogService           // 添加UserLog服务
	Permission             *service.PermissionService        // 添加Permission服务
	Role                   *service.RoleService              // 添加Role服务
	Credit                 *service.CreditService            // 添加Credit服务
	PlatformPushStatus     *platform.PushStatusService       // 添加PlatformPushStatus服务
	PlatformSvc            *platform.Service                 // 添加platform.Service
	PlatformAccount        *service.PlatformAccountService
	PlatformAccountVariant *service.PlatformAccountVariantService
	SystemConfig           *service.SystemConfigService  // 添加SystemConfig服务
	PhoneQuery             service.PhoneQueryService     // 添加PhoneQuery服务
	OrderException         service.OrderExceptionService // 添加OrderException服务

}

// NewContainer 创建新的容器实例
func NewContainer() (*Container, error) {
	return NewContainerWithConfig("configs/config.yaml")
}

// NewContainerWithConfig 使用指定配置文件创建容器实例
func NewContainerWithConfig(configPath string) (*Container, error) {
	return NewContainerWithConfigAndService(configPath, "")
}

// NewContainerWithConfigAndService 使用指定配置文件和服务名创建容器实例
func NewContainerWithConfigAndService(configPath, serviceName string) (*Container, error) {
	c := &Container{serviceName: serviceName}

	// 加载指定的配置文件
	viper.SetConfigFile(configPath)
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}
	// 为兼容已有服务构造，仍保留一次性 Unmarshal 快照
	var cfg configs.Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	c.config = &cfg

	// 初始化logger
	if err := c.initLogger(serviceName); err != nil {
		return nil, err
	}

	// 初始化数据库
	if err := c.initDB(); err != nil {
		return nil, err
	}

	// 初始化Redis
	if err := c.initRedis(); err != nil {
		return nil, err
	}

	// 初始化队列
	if err := c.initQueue(); err != nil {
		return nil, err
	}

	// 初始化仓储
	c.initRepositories()

	// 初始化优化组件
	if err := c.initOptimizedComponents(); err != nil {
		return nil, err
	}

	// 初始化服务
	c.initServices()

	// 初始化控制器
	c.initControllers()

	// 初始化中间件
	c.initMiddleware()

	return c, nil
}

// 初始化数据库
func (c *Container) initDB() error {
	dbConfig := &database.DatabaseConfig{
		Host:            viper.GetString("database.host"),
		Port:            viper.GetInt("database.port"),
		User:            viper.GetString("database.user"),
		Password:        viper.GetString("database.password"),
		Name:            viper.GetString("database.dbname"),
		Charset:         "utf8mb4",
		MaxIdleConns:    viper.GetInt("database.max_idle_conns"),
		MaxOpenConns:    viper.GetInt("database.max_open_conns"),
		ConnMaxLifetime: time.Duration(viper.GetInt("database.conn_max_lifetime")) * time.Second,
		SlowThreshold:   time.Second,
		LogLevel:        viper.GetString("log.level"),
	}

	dm, err := database.NewDatabaseManager(dbConfig)
	if err != nil {
		return err
	}
	c.databaseManager = dm
	c.db = dm.GetDB()

	if err := database.AutoMigrateDB(c.db); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.databaseManager.Ping(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	return nil
}

// 初始化Redis
func (c *Container) initRedis() error {
	err := redisx.InitRedis(
		viper.GetString("redis.host"),
		viper.GetInt("redis.port"),
		viper.GetString("redis.password"),
		viper.GetInt("redis.db"),
	)
	if err != nil {
		return err
	}
	c.redisClient = redisx.GetClient()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, pingErr := c.redisClient.Ping(ctx).Result(); pingErr != nil {
		return fmt.Errorf("redis ping failed: %w", pingErr)
	}
	return nil
}

// 初始化队列
func (c *Container) initQueue() error {
	redisAddr := fmt.Sprintf("%s:%d", viper.GetString("redis.host"), viper.GetInt("redis.port"))
	client := asynq.NewClient(asynq.RedisClientOpt{
		Addr:     redisAddr,
		Password: viper.GetString("redis.password"),
		DB:       viper.GetInt("redis.db"),
	})
	c.queue = client
	return nil
}

// 初始化中间件
func (c *Container) initMiddleware() {
	// 初始化MF178认证中间件
	middleware.InitMF178Auth(c.db)
	// 初始化全局认证中间件
	middleware.InitGlobalAuthMiddleware(c.db)
}

// 初始化仓储
func (c *Container) initRepositories() {
	c.repositories = &Repositories{
		User:                   repository.NewUserRepository(c.db),
		Order:                  repository.NewOrderRepository(c.db),
		OrderStatistics:        repository.NewOrderStatisticsRepository(c.db),
		Platform:               repository.NewPlatformRepository(c.db),
		PlatformAPI:            repository.NewPlatformAPIRepository(c.db),
		PlatformAPIParam:       repository.NewPlatformAPIParamRepository(c.db),
		PlatformAccount:        repository.NewPlatformAccountRepository(c.db),
		PlatformAccountVariant: repository.NewPlatformAccountVariantRepository(c.db),
		Product:                repository.NewProductRepository(c.db),
		ProductType:            repository.NewProductTypeRepository(c.db),
		ProductTypeCategory:    repository.NewProductTypeCategoryRepository(c.db),
		ProductAPIRelation:     repository.NewProductAPIRelationRepository(c.db),
		Retry:                  repository.NewRetryRepository(c.db),
		CallbackLog:            repository.NewCallbackLogRepository(c.db),
		BalanceLog:             repository.NewBalanceLogRepository(c.db),
		BalanceQueryRecord:     repository.NewBalanceQueryRecordRepository(c.db),
		Notification:           notificationRepo.NewRepository(c.db),
		TaskConfig:             repository.NewTaskConfigRepository(c.db),
		TaskOrder:              repository.NewTaskOrderRepository(c.db),
		DaichongOrder:          repository.NewDaichongOrderRepository(c.db),
		PhoneLocation:          repository.NewPhoneLocationRepository(c.db),
		Permission:             repository.NewPermissionRepository(c.db),
		Role:                   repository.NewRoleRepository(c.db),
		UserLog:                repository.NewUserLogRepository(c.db),
		CreditLog:              repository.NewCreditLogRepository(c.db),
		SystemConfig:           repository.NewSystemConfigRepository(c.db),
		ExternalAPIKey:         repository.NewExternalAPIKeyRepository(c.db),
		OrderException:         repository.NewOrderExceptionRepository(c.db),
	}
}

// 初始化服务
func (c *Container) initServices() error {
	// 创建队列实例
	queueInstance := queue.NewRedisQueue()
	c.taskQueue = queueInstance

	// 创建平台账号余额服务
	c.services = &Services{}
	c.services.PlatformAccountBalance = service.NewPlatformAccountBalanceService(
		c.db,
		c.repositories.PlatformAccount,
		c.repositories.User,
		c.repositories.BalanceLog,
	)

	// 初始化余额服务（需要在充值服务之前创建）
	c.services.Balance = service.NewBalanceService(
		c.repositories.BalanceLog,
		c.repositories.User,
	)

	// 创建分布式锁管理器
	distributedLock := lock.NewRedisDistributedLock(c.redisClient)
	refundLockManager := lock.NewRefundLockManager(distributedLock)

	// 初始化统一退款服务
	c.services.UnifiedRefund = service.NewUnifiedRefundService(
		c.db,
		c.repositories.User,
		c.repositories.Order,
		c.repositories.BalanceLog,
		refundLockManager,
		c.services.Balance,
		c.services.PlatformAccountBalance,
	)

	// 创建其他服务
	c.services.User = service.NewUserService(
		c.repositories.User,
		repository.NewUserGradeRepository(c.db),
		repository.NewUserTagRepository(c.db),
		repository.NewUserTagRelationRepository(c.db),
		repository.NewUserGradeRelationRepository(c.db),
		repository.NewUserLogRepository(c.db),
	)

	// 创建UserGrade和UserTag服务
	c.services.UserGrade = service.NewUserGradeService(
		repository.NewUserGradeRepository(c.db),
		repository.NewUserGradeRelationRepository(c.db),
	)
	c.services.UserTag = service.NewUserTagService(
		repository.NewUserTagRepository(c.db),
		repository.NewUserTagRelationRepository(c.db),
	)

	c.services.Platform = service.NewPlatformService(c.repositories.Platform, c.repositories.Order, c.repositories.ExternalAPIKey)
	c.services.PlatformService = c.services.Platform
	c.services.Statistics = service.NewStatisticsService(c.repositories.OrderStatistics, c.repositories.Order)
	c.services.Notification = notificationService.NewNotificationService(c.repositories.Notification, queueInstance)

	// 初始化PhoneQuery服务（需要在充值服务之前创建）
	c.services.PhoneQuery = service.NewPhoneQueryService()

	// 初始化SystemConfig服务（需要在其他服务之前创建）
	c.services.SystemConfig = service.NewSystemConfigService(c.repositories.SystemConfig)

	// 初始化OrderException服务
	c.services.OrderException = service.NewOrderExceptionService(
		c.repositories.OrderException,
		c.repositories.Order,
		c.logger,
	)

	// 创建统一订单服务（暂时不传入retryService，避免循环依赖）
	c.services.UnifiedOrder = service.NewUnifiedOrderService(
		c.repositories.Order,
		c.repositories.BalanceQueryRecord,
		c.services.PhoneQuery,
		c.services.Balance,
		c.repositories.Notification,
		queueInstance,
		c.db,
		c.logger,
		c.services.SystemConfig,
		c.repositories.Product,
		nil, // retryService 稍后设置
		c.services.OrderException,
	)

	// 创建充值服务
	c.services.Recharge = service.NewRechargeService(
		c.db,
		c.repositories.Order,
		c.repositories.Platform,
		c.repositories.PlatformAPI,
		c.repositories.Retry,
		c.repositories.CallbackLog,
		c.repositories.ProductAPIRelation,
		c.repositories.Product,
		c.repositories.PlatformAPIParam,
		c.services.PlatformAccountBalance,
		c.services.Balance,
		c.services.PhoneQuery,             // 添加手机查询服务依赖
		c.repositories.BalanceQueryRecord, // 添加余额查询记录仓库依赖
		c.services.UnifiedOrder,           // 添加统一订单服务
		c.services.SystemConfig,           // 添加系统配置服务
		c.repositories.Notification,
		queueInstance,
	)

	// 创建订单服务
	c.services.Order = service.NewOrderService(
		c.repositories.Order,
		c.repositories.BalanceLog,
		c.repositories.User,
		c.services.Recharge,
		c.services.UnifiedRefund,
		refundLockManager,
		c.repositories.Notification,
		queueInstance,
		c.db,
		c.repositories.Product,
		c.services.Credit,
		c.repositories.BalanceQueryRecord,
	)

	// 设置相互依赖
	c.services.Recharge.SetOrderService(c.services.Order)

	// 初始化重试服务
	c.services.Retry = service.NewRetryService(
		c.repositories.Retry,
		c.repositories.Order,
		c.repositories.Platform,
		c.repositories.Product,
		c.repositories.ProductAPIRelation,
		c.services.Recharge,
		c.services.Order,
	)

	// 设置统一订单服务的重试服务依赖（解决循环依赖）
	c.services.UnifiedOrder.SetRetryService(c.services.Retry)

	// 初始化统计任务服务
	c.services.StatisticsTask = service.NewStatisticsTask(
		c.services.Statistics,
		c.logger,
	)

	// 初始化platform.Service
	c.services.PlatformSvc = platform.NewService(
		repository.NewPlatformTokenRepository(c.db),
		c.repositories.Platform,
	)

	// 初始化TaskService
	taskConfig := &configs.TaskConfig{
		Interval:             viper.GetInt("task.interval"),
		OrderDetailsInterval: viper.GetInt("task.order_details_interval"),
		MaxRetries:           viper.GetInt("task.max_retries"),
		RetryDelay:           viper.GetInt("task.retry_delay"),
		MaxConcurrent:        viper.GetInt("task.max_concurrent"),
		BatchSize:            viper.GetInt("task.batch_size"),
		SuspendThreshold:     viper.GetInt("task.suspend_threshold"),
		ResumeThreshold:      viper.GetInt("task.resume_threshold"),
	}
	c.services.Task = service.NewTaskService(
		c.repositories.TaskConfig,
		c.repositories.TaskOrder,
		c.repositories.Order,
		c.repositories.DaichongOrder,
		c.services.PlatformSvc,
		c.services.Order,
		taskConfig,
		c.repositories.PlatformAccount,
	)

	// 初始化TaskConfigNotifier
	c.services.TaskConfigNotifier = service.NewTaskConfigNotifier(c.redisClient)

	// 初始化PhoneLocationService
	c.services.PhoneLocation = service.NewPhoneLocationService(c.repositories.PhoneLocation)

	// 初始化Product服务
	c.services.Product = service.NewProductService(c.repositories.Product)

	// 初始化ProductType服务
	c.services.ProductType = service.NewProductTypeService(c.repositories.ProductType, c.repositories.ProductTypeCategory)

	// 初始化PlatformAPI服务
	c.services.PlatformAPI = service.NewPlatformAPIService(c.repositories.PlatformAPI)

	// 初始化Permission服务
	c.services.Permission = service.NewPermissionService(c.repositories.Permission)

	// 初始化Role服务
	c.services.Role = service.NewRoleService(c.repositories.Role)

	// 初始化PlatformAPIParam服务
	c.services.PlatformAPIParam = service.NewPlatformAPIParamService(c.repositories.PlatformAPIParam)

	// 初始化ProductAPIRelation服务
	c.services.ProductAPIRelation = service.NewProductAPIRelationService(c.repositories.ProductAPIRelation)

	// 初始化UserLog服务
	c.services.UserLog = service.NewUserLogService(c.repositories.UserLog)

	// 初始化Credit服务
	c.services.Credit = service.NewCreditService(c.repositories.User, c.repositories.CreditLog)

	// 初始化PlatformPushStatus服务
	c.services.PlatformPushStatus = platform.NewPushStatusService(c.repositories.PlatformAccount)
	c.services.PlatformAccount = service.NewPlatformAccountService(c.repositories.PlatformAccount)
	c.services.PlatformAccountVariant = service.NewPlatformAccountVariantService(c.repositories.PlatformAccountVariant, c.repositories.PlatformAccount)

	// 初始化系统配置数据
	if err := c.services.SystemConfig.InitSystemConfigs(context.Background()); err != nil {
		c.logger.Error("初始化系统配置失败", zap.Error(err))
		// 不返回错误，允许系统继续启动
	}

	return nil
}

// initLogger 初始化日志
func (c *Container) initLogger(serviceName string) error {
	outputFile := "stdout"
	if serviceName != "" {
		outputFile = filepath.Join("logs", serviceName+".log")
	}
	if err := log.Init(log.Config{
		Level:      "info",
		Format:     "json",
		Output:     outputFile,
		MaxSize:    100,
		MaxBackups: 5,
		MaxAge:     30,
		Compress:   true,
		Caller:     true,
		Stacktrace: true,
	}); err != nil {
		return fmt.Errorf("初始化日志失败: %w", err)
	}
	c.logger = log.L()
	return nil
}

// GetConfig 获取配置
func (c *Container) GetConfig() *configs.Config {
	return c.config
}

// GetDB 获取数据库连接
func (c *Container) GetDB() *gorm.DB {
	return c.db
}

// GetSQLDB 获取SQL数据库连接（用于迁移）
func (c *Container) GetSQLDB() (*sql.DB, error) {
	return c.db.DB()
}

// GetRedis 获取Redis客户端
func (c *Container) GetRedis() *redisc.Client {
	return c.redisClient
}

// GetQueue 获取队列客户端
func (c *Container) GetQueue() *asynq.Client {
	return c.queue
}

// GetRepositories 获取仓储集合
func (c *Container) GetRepositories() *Repositories {
	return c.repositories
}

// GetServices 获取服务集合
func (c *Container) GetServices() *Services {
	return c.services
}

// GetLoggerV2 获取优化后的日志器
// 移除旧版 v2 logger 依赖，统一使用 pkg/log

// GetMetricsManager 获取指标管理器
func (c *Container) GetMetricsManager() *metrics.MetricsManager {
	return c.metricsManager
}

// GetSecurityMiddleware 获取安全中间件
func (c *Container) GetSecurityMiddleware() *pkgMiddleware.SecurityMiddleware {
	return c.securityMiddleware
}

// GetDatabaseManager 获取数据库管理器
func (c *Container) GetDatabaseManager() *database.DatabaseManager {
	return c.databaseManager
}

// initOptimizedComponents 初始化优化组件
func (c *Container) initOptimizedComponents() error {
	// 统一在 initLogger 中完成日志初始化，避免重复初始化导致配置不一致

	// 初始化指标管理器
	c.metricsManager = metrics.NewMetricsManager()

	if c.databaseManager == nil {
		dbConfig := &database.DatabaseConfig{
			Host:            viper.GetString("database.host"),
			Port:            viper.GetInt("database.port"),
			User:            viper.GetString("database.user"),
			Password:        viper.GetString("database.password"),
			Name:            viper.GetString("database.dbname"),
			Charset:         "utf8mb4",
			MaxIdleConns:    viper.GetInt("database.max_idle_conns"),
			MaxOpenConns:    viper.GetInt("database.max_open_conns"),
			ConnMaxLifetime: time.Hour,
			SlowThreshold:   time.Second,
			LogLevel:        viper.GetString("log.level"),
		}

		dm, err := database.NewDatabaseManager(dbConfig)
		if err != nil {
			return fmt.Errorf("failed to initialize database manager: %w", err)
		}
		c.databaseManager = dm
	}

	// 初始化安全中间件：优先读取 configs/config.yaml 的 security.* 配置，缺省时采用合理默认
	jwtSkip := viper.GetStringSlice("security.jwt.skip_paths")
	rlEnabled := viper.GetBool("security.rate_limit.enabled")
	rlRPS := viper.GetInt("security.rate_limit.rps")
	rlBurst := viper.GetInt("security.rate_limit.burst")
	rlWindow := viper.GetDuration("security.rate_limit.window")
	rlSkip := viper.GetStringSlice("security.rate_limit.skip_paths")
	rlInclude := viper.GetStringSlice("security.rate_limit.include_paths")
	corsAllowOrigins := viper.GetStringSlice("security.cors.allow_origins")
	corsAllowMethods := viper.GetStringSlice("security.cors.allow_methods")
	corsAllowHeaders := viper.GetStringSlice("security.cors.allow_headers")
	corsExposeHeaders := viper.GetStringSlice("security.cors.expose_headers")
	corsAllowCredentials := viper.GetBool("security.cors.allow_credentials")
	corsMaxAge := viper.GetInt("security.cors.max_age")
	corsSkip := viper.GetStringSlice("security.cors.skip_paths")

	if rlWindow == 0 {
		rlWindow = time.Minute
	}
	if rlRPS == 0 {
		rlRPS = 100
	}
	if rlBurst == 0 {
		rlBurst = 200
	}
	if len(corsAllowOrigins) == 0 {
		corsAllowOrigins = []string{"*"}
	}
	if len(corsAllowMethods) == 0 {
		corsAllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	}
	if len(corsAllowHeaders) == 0 {
		corsAllowHeaders = []string{"Authorization", "Content-Type", "X-Request-ID"}
	}

	securityConfig := &pkgMiddleware.SecurityConfig{
		JWT: pkgMiddleware.JWTConfig{
			Secret:     viper.GetString("jwt.secret"),
			Expiration: time.Duration(viper.GetInt("jwt.expire")) * time.Hour,
			Issuer:     "recharge-system",
			SkipPaths:  jwtSkip,
		},
		RateLimit: pkgMiddleware.RateLimitConfig{
			Enabled:      rlEnabled,
			RPS:          rlRPS,
			Burst:        rlBurst,
			Window:       rlWindow,
			SkipPaths:    rlSkip,
			IncludePaths: rlInclude,
		},
		CORS: pkgMiddleware.CORSConfig{
			AllowOrigins:     corsAllowOrigins,
			AllowMethods:     corsAllowMethods,
			AllowHeaders:     corsAllowHeaders,
			ExposeHeaders:    corsExposeHeaders,
			AllowCredentials: corsAllowCredentials,
			MaxAge:           corsMaxAge,
			SkipPaths:        corsSkip,
		},
	}

	c.securityMiddleware = pkgMiddleware.NewSecurityMiddleware(securityConfig)

	log.L().Info("Security policies",
		log.Any("jwt_skip_paths", securityConfig.JWT.SkipPaths),
		log.Any("ratelimit_enabled", securityConfig.RateLimit.Enabled),
		log.Any("ratelimit_skip_paths", securityConfig.RateLimit.SkipPaths),
		log.Any("cors_skip_paths", securityConfig.CORS.SkipPaths),
	)

	return nil
}

// Close 关闭容器，释放资源
func (c *Container) Close() error {
	if c.queue != nil {
		c.queue.Close()
	}
	// 停止安全中间件的后台协程
	if c.securityMiddleware != nil {
		c.securityMiddleware.Stop()
	}
	if c.redisClient != nil {
		if err := c.redisClient.Close(); err != nil {
			return err
		}
	}
	if c.databaseManager != nil {
		if err := c.databaseManager.Close(); err != nil {
			return err
		}
	}
	return nil
}

// GetTaskQueue 获取业务队列（Redis 列表实现）
func (c *Container) GetTaskQueue() queue.Queue {
	return c.taskQueue
}
