package app

import (
	"database/sql"
	"fmt"
	"recharge-go/internal/config"
	"recharge-go/internal/middleware"
	"recharge-go/internal/pkg/db"
	"recharge-go/internal/repository"
	notificationRepo "recharge-go/internal/repository/notification"
	"recharge-go/internal/service"
	notificationService "recharge-go/internal/service/notification"
	"recharge-go/internal/service/platform"
	"recharge-go/pkg/database"
	loggerV2 "recharge-go/pkg/logger"
	"recharge-go/pkg/metrics"
	pkgMiddleware "recharge-go/pkg/middleware"
	"recharge-go/pkg/queue"
	"recharge-go/pkg/redis"
	"time"

	redisV8 "github.com/go-redis/redis/v8"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Container 依赖注入容器
type Container struct {
	config             *config.Config
	db                 *gorm.DB
	redis              *redisV8.Client
	redisClient        *redisV8.Client
	queue              *asynq.Client
	repositories       *Repositories
	services           *Services
	controllers        *Controllers
	logger             *zap.Logger
	loggerV2           *loggerV2.LoggerV2
	metricsManager     *metrics.MetricsManager
	securityMiddleware *pkgMiddleware.SecurityMiddleware
	databaseManager    *database.DatabaseManager
}

// Repositories 仓储集合
type Repositories struct {
	User                *repository.UserRepository
	Order               repository.OrderRepository
	OrderStatistics     repository.OrderStatisticsRepository
	Platform            repository.PlatformRepository
	PlatformAPI         repository.PlatformAPIRepository
	PlatformAPIParam    repository.PlatformAPIParamRepository
	PlatformAccount     *repository.PlatformAccountRepository
	Product             repository.ProductRepository
	ProductType         *repository.ProductTypeRepository         // 添加ProductType repository
	ProductTypeCategory *repository.ProductTypeCategoryRepository // 添加ProductTypeCategory repository
	ProductAPIRelation  repository.ProductAPIRelationRepository
	Retry               repository.RetryRepository
	CallbackLog         repository.CallbackLogRepository
	BalanceLog          *repository.BalanceLogRepository
	Notification        notificationRepo.Repository
	TaskConfig          *repository.TaskConfigRepository
	TaskOrder           *repository.TaskOrderRepository
	DaichongOrder       *repository.DaichongOrderRepository
	PhoneLocation       *repository.PhoneLocationRepository
	Permission          *repository.PermissionRepository // 添加Permission repository
	Role                *repository.RoleRepository       // 添加Role repository
	UserLog             *repository.UserLogRepository    // 添加UserLog repository
	CreditLog           *repository.CreditLogRepository  // 添加CreditLog repository
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
	Task                   *service.TaskService
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
}

// NewContainer 创建新的容器实例
func NewContainer() (*Container, error) {
	c := &Container{}

	// 加载配置
	var err error
	c.config, err = config.LoadConfig("configs/config.yaml")
	if err != nil {
		return nil, err
	}

	// 初始化logger
	if err := c.initLogger(); err != nil {
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
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		c.config.Database.User,
		c.config.Database.Password,
		c.config.Database.Host,
		c.config.Database.Port,
		c.config.Database.DBName,
	)

	dbInstance, err := db.NewDB(dsn)
	if err != nil {
		return err
	}
	c.db = dbInstance.DB
	return nil
}

// 初始化Redis
func (c *Container) initRedis() error {
	err := redis.InitRedis(c.config.Redis.Host, c.config.Redis.Port, c.config.Redis.Password, c.config.Redis.DB)
	if err != nil {
		return err
	}
	c.redisClient = redis.GetClient()
	return nil
}

// 初始化队列
func (c *Container) initQueue() error {
	redisAddr := fmt.Sprintf("%s:%d", c.config.Redis.Host, c.config.Redis.Port)
	client := asynq.NewClient(asynq.RedisClientOpt{
		Addr:     redisAddr,
		Password: c.config.Redis.Password,
		DB:       c.config.Redis.DB,
	})
	c.queue = client
	return nil
}

// 初始化中间件
func (c *Container) initMiddleware() {
	// 初始化MF178认证中间件
	middleware.InitMF178Auth(c.db)
}

// 初始化仓储
func (c *Container) initRepositories() {
	c.repositories = &Repositories{
		User:                repository.NewUserRepository(c.db),
		Order:               repository.NewOrderRepository(c.db),
		OrderStatistics:     repository.NewOrderStatisticsRepository(c.db),
		Platform:            repository.NewPlatformRepository(c.db),
		PlatformAPI:         repository.NewPlatformAPIRepository(c.db),
		PlatformAPIParam:    repository.NewPlatformAPIParamRepository(c.db),
		PlatformAccount:     repository.NewPlatformAccountRepository(c.db),
		Product:             repository.NewProductRepository(c.db),
		ProductType:         repository.NewProductTypeRepository(c.db),
		ProductTypeCategory: repository.NewProductTypeCategoryRepository(c.db),
		ProductAPIRelation:  repository.NewProductAPIRelationRepository(c.db),
		Retry:               repository.NewRetryRepository(c.db),
		CallbackLog:         repository.NewCallbackLogRepository(c.db),
		BalanceLog:          repository.NewBalanceLogRepository(c.db),
		Notification:        notificationRepo.NewRepository(c.db),
		TaskConfig:          repository.NewTaskConfigRepository(c.db),
		TaskOrder:           repository.NewTaskOrderRepository(c.db),
		DaichongOrder:       repository.NewDaichongOrderRepository(c.db),
		PhoneLocation:       repository.NewPhoneLocationRepository(c.db),
		Permission:          repository.NewPermissionRepository(c.db),
		Role:                repository.NewRoleRepository(c.db),
		UserLog:             repository.NewUserLogRepository(c.db),
		CreditLog:           repository.NewCreditLogRepository(c.db),
	}
}

// 初始化服务
func (c *Container) initServices() error {
	// 创建队列实例
	queueInstance := queue.NewRedisQueue()

	// 创建平台账号余额服务
	c.services = &Services{}
	c.services.PlatformAccountBalance = service.NewPlatformAccountBalanceService(
		c.db,
		c.repositories.PlatformAccount,
		c.repositories.User,
		c.repositories.BalanceLog,
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

	c.services.Platform = service.NewPlatformService(c.repositories.Platform, c.repositories.Order)
	c.services.PlatformService = c.services.Platform
	c.services.Statistics = service.NewStatisticsService(c.repositories.OrderStatistics, c.repositories.Order)
	c.services.Notification = notificationService.NewNotificationService(c.repositories.Notification, queueInstance)

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
		c.repositories.Notification,
		queueInstance,
	)

	// 创建订单服务
	c.services.Order = service.NewOrderService(
		c.repositories.Order,
		c.services.Recharge,
		c.repositories.Notification,
		queueInstance,
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

	// 初始化余额服务
	c.services.Balance = service.NewBalanceService(
		c.repositories.BalanceLog,
		c.repositories.User,
	)

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
	taskConfig := &service.TaskConfig{
		Interval:      time.Second * 1, // 每秒检查一次，匹配订单
		MaxRetries:    3,
		RetryDelay:    time.Second * 30,
		MaxConcurrent: 5,
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

	return nil
}

// initLogger 初始化日志
func (c *Container) initLogger() error {
	logger, err := zap.NewProduction()
	if err != nil {
		return fmt.Errorf("初始化logger失败: %w", err)
	}
	c.logger = logger
	return nil
}

// GetConfig 获取配置
func (c *Container) GetConfig() *config.Config {
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
func (c *Container) GetRedis() *redisV8.Client {
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
func (c *Container) GetLoggerV2() *loggerV2.LoggerV2 {
	return c.loggerV2
}

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
	// 初始化优化后的日志器
	loggerConfig := &loggerV2.LoggerConfigV2{
		Level:      "info",
		Format:     "json",
		Output:     "logs/app.log",
		MaxSize:    100,
		MaxBackups: 5,
		MaxAge:     30,
		Compress:   true,
		Caller:     true,
		Stacktrace: true,
	}

	var err error
	c.loggerV2, err = loggerV2.NewLoggerV2(loggerConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize logger v2: %w", err)
	}

	// 初始化指标管理器
	c.metricsManager = metrics.NewMetricsManager(c.loggerV2)

	// 初始化数据库管理器
	dbConfig := &database.DatabaseConfig{
		Host:            c.config.Database.Host,
		Port:            c.config.Database.Port,
		User:            c.config.Database.User,
		Password:        c.config.Database.Password,
		Name:            c.config.Database.DBName,
		Charset:         "utf8mb4",
		MaxIdleConns:    10,
		MaxOpenConns:    100,
		ConnMaxLifetime: time.Hour,
		SlowThreshold:   time.Second,
		LogLevel:        "info",
	}

	c.databaseManager, err = database.NewDatabaseManager(dbConfig, c.loggerV2)
	if err != nil {
		return fmt.Errorf("failed to initialize database manager: %w", err)
	}

	// 初始化安全中间件
	securityConfig := &pkgMiddleware.SecurityConfig{
		JWT: pkgMiddleware.JWTConfig{
			Secret:     c.config.JWT.Secret,
			Expiration: time.Duration(c.config.JWT.Expire) * time.Hour,
			Issuer:     "recharge-system",
			SkipPaths:  []string{"/api/v1/auth/login", "/api/v1/health"},
		},
		RateLimit: pkgMiddleware.RateLimitConfig{
			Enabled:   true,
			RPS:       100,
			Burst:     200,
			Window:    time.Minute,
			SkipPaths: []string{"/api/v1/health"},
		},
		CORS: pkgMiddleware.CORSConfig{
			AllowOrigins:     []string{"*"},
			AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"*"},
			AllowCredentials: true,
			MaxAge:           86400,
		},
	}

	c.securityMiddleware = pkgMiddleware.NewSecurityMiddleware(securityConfig, c.loggerV2)

	return nil
}

// Close 关闭容器，释放资源
func (c *Container) Close() error {
	if c.queue != nil {
		c.queue.Close()
	}
	if err := redis.Close(); err != nil {
		return err
	}
	return nil
}
