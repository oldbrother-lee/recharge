package database

import (
	"context"
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	"gorm.io/plugin/dbresolver"

	loggerV2 "recharge-go/pkg/logger"
)

// DatabaseManager 数据库管理器
type DatabaseManager struct {
	db     *gorm.DB
	config *DatabaseConfig
	logger *loggerV2.LoggerV2
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	User            string        `yaml:"user"`
	Password        string        `yaml:"password"`
	Name            string        `yaml:"name"`
	Charset         string        `yaml:"charset"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
	SSLMode         string        `yaml:"ssl_mode"`
	SlowThreshold   time.Duration `yaml:"slow_threshold"`
	LogLevel        string        `yaml:"log_level"`

	// 读写分离配置
	ReadReplicas []ReplicaConfig `yaml:"read_replicas"`
}

// ReplicaConfig 从库配置
type ReplicaConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Name     string `yaml:"name"`
}

// ConnectionStats 连接统计
type ConnectionStats struct {
	MaxOpenConnections int           `json:"max_open_connections"`
	OpenConnections    int           `json:"open_connections"`
	InUse              int           `json:"in_use"`
	Idle               int           `json:"idle"`
	WaitCount          int64         `json:"wait_count"`
	WaitDuration       time.Duration `json:"wait_duration"`
	MaxIdleClosed      int64         `json:"max_idle_closed"`
	MaxIdleTimeClosed  int64         `json:"max_idle_time_closed"`
	MaxLifetimeClosed  int64         `json:"max_lifetime_closed"`
}

// NewDatabaseManager 创建数据库管理器
func NewDatabaseManager(config *DatabaseConfig, logger *loggerV2.LoggerV2) (*DatabaseManager, error) {
	manager := &DatabaseManager{
		config: config,
		logger: logger,
	}

	if err := manager.connect(); err != nil {
		return nil, err
	}

	return manager, nil
}

// connect 连接数据库
func (dm *DatabaseManager) connect() error {
	// 构建DSN
	dsn := dm.buildDSN(dm.config.Host, dm.config.Port, dm.config.User, dm.config.Password, dm.config.Name)

	// 配置GORM日志
	gormLogger := dm.createGormLogger()

	// 连接主数据库
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormLogger,
		// 禁用外键约束检查
		DisableForeignKeyConstraintWhenMigrating: true,
		// 命名策略
		NamingStrategy: &CustomNamingStrategy{},
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// 配置连接池
	if err := dm.configureConnectionPool(db); err != nil {
		return fmt.Errorf("failed to configure connection pool: %w", err)
	}

	// 配置读写分离
	if len(dm.config.ReadReplicas) > 0 {
		if err := dm.configureReadReplicas(db); err != nil {
			dm.logger.Warn("Failed to configure read replicas", loggerV2.ErrorV2(err))
		}
	}

	dm.db = db
	return nil
}

// buildDSN 构建数据源名称
func (dm *DatabaseManager) buildDSN(host string, port int, user, password, dbname string) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		user, password, host, port, dbname, dm.config.Charset)
}

// createGormLogger 创建GORM日志器
func (dm *DatabaseManager) createGormLogger() logger.Interface {
	// 解析日志级别
	var logLevel logger.LogLevel
	switch dm.config.LogLevel {
	case "silent":
		logLevel = logger.Silent
	case "error":
		logLevel = logger.Error
	case "warn":
		logLevel = logger.Warn
	case "info":
		logLevel = logger.Info
	default:
		logLevel = logger.Info
	}

	return &CustomGormLogger{
		logger:        dm.logger,
		logLevel:      logLevel,
		slowThreshold: dm.config.SlowThreshold,
	}
}

// configureConnectionPool 配置连接池
func (dm *DatabaseManager) configureConnectionPool(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	// 设置最大空闲连接数
	sqlDB.SetMaxIdleConns(dm.config.MaxIdleConns)

	// 设置最大打开连接数
	sqlDB.SetMaxOpenConns(dm.config.MaxOpenConns)

	// 设置连接最大生命周期
	sqlDB.SetConnMaxLifetime(dm.config.ConnMaxLifetime)

	return nil
}

// configureReadReplicas 配置读写分离
func (dm *DatabaseManager) configureReadReplicas(db *gorm.DB) error {
	replicas := make([]gorm.Dialector, 0, len(dm.config.ReadReplicas))

	for _, replica := range dm.config.ReadReplicas {
		dsn := dm.buildDSN(replica.Host, replica.Port, replica.User, replica.Password, replica.Name)
		replicas = append(replicas, mysql.Open(dsn))
	}

	return db.Use(dbresolver.Register(dbresolver.Config{
		Replicas: replicas,
		Policy:   dbresolver.RandomPolicy{},
	}))
}

// GetDB 获取数据库连接
func (dm *DatabaseManager) GetDB() *gorm.DB {
	return dm.db
}

// GetStats 获取连接统计
func (dm *DatabaseManager) GetStats() (*ConnectionStats, error) {
	sqlDB, err := dm.db.DB()
	if err != nil {
		return nil, err
	}

	stats := sqlDB.Stats()
	return &ConnectionStats{
		MaxOpenConnections: stats.MaxOpenConnections,
		OpenConnections:    stats.OpenConnections,
		InUse:              stats.InUse,
		Idle:               stats.Idle,
		WaitCount:          stats.WaitCount,
		WaitDuration:       stats.WaitDuration,
		MaxIdleClosed:      stats.MaxIdleClosed,
		MaxIdleTimeClosed:  stats.MaxIdleTimeClosed,
		MaxLifetimeClosed:  stats.MaxLifetimeClosed,
	}, nil
}

// Ping 检查数据库连接
func (dm *DatabaseManager) Ping(ctx context.Context) error {
	sqlDB, err := dm.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

// Close 关闭数据库连接
func (dm *DatabaseManager) Close() error {
	sqlDB, err := dm.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Transaction 执行事务
func (dm *DatabaseManager) Transaction(ctx context.Context, fn func(*gorm.DB) error) error {
	return dm.db.WithContext(ctx).Transaction(fn)
}

// WithContext 添加上下文
func (dm *DatabaseManager) WithContext(ctx context.Context) *gorm.DB {
	return dm.db.WithContext(ctx)
}

// CustomNamingStrategy 自定义命名策略
type CustomNamingStrategy struct {
	schema.NamingStrategy
}

// CustomGormLogger 自定义GORM日志器
type CustomGormLogger struct {
	logger        *loggerV2.LoggerV2
	logLevel      logger.LogLevel
	slowThreshold time.Duration
}

// LogMode 设置日志模式
func (l *CustomGormLogger) LogMode(level logger.LogLevel) logger.Interface {
	newLogger := *l
	newLogger.logLevel = level
	return &newLogger
}

// Info 记录信息日志
func (l *CustomGormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= logger.Info {
		l.logger.WithContext(ctx).Info(fmt.Sprintf(msg, data...))
	}
}

// Warn 记录警告日志
func (l *CustomGormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= logger.Warn {
		l.logger.WithContext(ctx).Warn(fmt.Sprintf(msg, data...))
	}
}

// Error 记录错误日志
func (l *CustomGormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= logger.Error {
		l.logger.WithContext(ctx).Error(fmt.Sprintf(msg, data...))
	}
}

// Trace 记录SQL跟踪日志
func (l *CustomGormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.logLevel <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	fields := []loggerV2.Field{
		loggerV2.String("sql", sql),
		loggerV2.Duration("elapsed", elapsed),
		loggerV2.Int64("rows", rows),
	}

	if err != nil {
		fields = append(fields, loggerV2.ErrorV2(err))
		l.logger.WithContext(ctx).Error("SQL execution failed", fields...)
	} else if elapsed > l.slowThreshold && l.slowThreshold != 0 {
		l.logger.WithContext(ctx).Warn("Slow SQL query detected", fields...)
	} else if l.logLevel >= logger.Info {
		l.logger.WithContext(ctx).Debug("SQL executed", fields...)
	}
}
