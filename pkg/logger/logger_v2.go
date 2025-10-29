package logger

import (
    "context"
    "fmt"
    "os"
    "path/filepath"
    "sync"
    "runtime"
    "time"

    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// LoggerV2 优化后的日志器
type LoggerV2 struct {
	*zap.Logger
	config *LoggerConfigV2
}

// LoggerConfigV2 日志配置
type LoggerConfigV2 struct {
	Level      string `yaml:"level"`
	Format     string `yaml:"format"`
	Output     string `yaml:"output"`
	MaxSize    int    `yaml:"max_size"`
	MaxBackups int    `yaml:"max_backups"`
	MaxAge     int    `yaml:"max_age"`
	Compress   bool   `yaml:"compress"`
	Caller     bool   `yaml:"caller"`
	Stacktrace bool   `yaml:"stacktrace"`
}

// FieldV2 类型别名，方便使用
type FieldV2 = zap.Field
type Field = zap.Field // 保持兼容性

// 常用字段构造函数 - 使用V2后缀避免冲突
var (
	StringV2   = zap.String
	IntV2      = zap.Int
	Int64V2    = zap.Int64
	Float64V2  = zap.Float64
	BoolV2     = zap.Bool
	TimeV2     = zap.Time
	DurationV2 = zap.Duration
	ErrorV2    = zap.Error
	AnyV2      = zap.Any

	// 保持原有名称以兼容现有代码（除了Error，避免冲突）
	String   = zap.String
	Int      = zap.Int
	Int64    = zap.Int64
	Float64  = zap.Float64
	Bool     = zap.Bool
	Time     = zap.Time
	Duration = zap.Duration
	Any      = zap.Any
)

// 全局日志器实例
var globalLogger *LoggerV2

// NewLoggerV2 创建新的日志器
func NewLoggerV2(config *LoggerConfigV2) (*LoggerV2, error) {
	// 解析日志级别
	level, err := zapcore.ParseLevel(config.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level: %w", err)
	}

	// 创建编码器配置
	encoderConfig := getEncoderConfig(config.Format)

	// 创建编码器
	var encoder zapcore.Encoder
	switch config.Format {
	case "json":
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	case "console":
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	default:
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	// 创建写入器
	writeSyncer := getWriteSyncer(config)

	// 创建核心
	core := zapcore.NewCore(encoder, writeSyncer, level)

	// 创建日志器选项
	options := []zap.Option{
		zap.AddStacktrace(zapcore.ErrorLevel),
	}

	if config.Caller {
		options = append(options, zap.AddCaller(), zap.AddCallerSkip(1))
	}

	if config.Stacktrace {
		options = append(options, zap.AddStacktrace(zapcore.WarnLevel))
	}

	// 创建zap日志器
	zapLogger := zap.New(core, options...)

	return &LoggerV2{
		Logger: zapLogger,
		config: config,
	}, nil
}

// getEncoderConfig 获取编码器配置
func getEncoderConfig(format string) zapcore.EncoderConfig {
	config := zap.NewProductionEncoderConfig()
	config.TimeKey = "timestamp"
	config.LevelKey = "level"
	config.NameKey = "logger"
	config.CallerKey = "caller"
	config.MessageKey = "message"
	config.StacktraceKey = "stacktrace"
	config.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncodeLevel = zapcore.LowercaseLevelEncoder
	config.EncodeDuration = zapcore.SecondsDurationEncoder
	config.EncodeCaller = zapcore.ShortCallerEncoder

	if format == "console" {
		config.EncodeLevel = zapcore.CapitalColorLevelEncoder
		config.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05")
	}

	return config
}

// getWriteSyncer 获取写入器
func getWriteSyncer(config *LoggerConfigV2) zapcore.WriteSyncer {
    if config.Output == "stdout" || config.Output == "" {
        return zapcore.AddSync(os.Stdout)
    }

    // 确保日志目录存在
    logDir := filepath.Dir(config.Output)
    if err := os.MkdirAll(logDir, 0755); err != nil {
        // 如果创建目录失败，回退到stdout
        return zapcore.AddSync(os.Stdout)
    }

    // 使用lumberjack进行日志轮转
    lumberjackLogger := &lumberjack.Logger{
        Filename:   config.Output,
        MaxSize:    config.MaxSize,
        MaxBackups: config.MaxBackups,
        MaxAge:     config.MaxAge,
        Compress:   config.Compress,
    }

    // 同时写入文件与终端，便于在终端查看详细日志
    return zapcore.NewMultiWriteSyncer(
        zapcore.AddSync(os.Stdout),
        zapcore.AddSync(lumberjackLogger),
    )
}

// InitGlobalLoggerV2 初始化全局日志器
func InitGlobalLoggerV2(config LoggerConfigV2) error {
	logger, err := NewLoggerV2(&config)
	if err != nil {
		return err
	}
	globalLogger = logger
	return nil
}

// GetGlobalLoggerV2 获取全局日志器
func GetGlobalLoggerV2() *LoggerV2 {
	return globalLogger
}

// InitGlobalLogger 初始化全局日志器
func InitGlobalLogger(config *LoggerConfigV2) error {
	logger, err := NewLoggerV2(config)
	if err != nil {
		return err
	}
	globalLogger = logger
	return nil
}

// GetGlobalLogger 获取全局日志器
func GetGlobalLogger() *LoggerV2 {
    if globalLogger == nil {
        // 如果全局日志器未初始化，创建一个默认的
        config := &LoggerConfigV2{
            Level:  "info",
            Format: "console",
            Output: "stdout",
            Caller: true,
        }
        logger, _ := NewLoggerV2(config)
        globalLogger = logger
    }
    return globalLogger
}

// WithContext 添加上下文信息
func (l *LoggerV2) WithContext(ctx context.Context) *LoggerV2 {
	// 从上下文中提取常用字段（安全类型处理，避免panic）
	fields := []Field{}

	// 请求ID
	if v := ctx.Value("request_id"); v != nil {
		if s, ok := v.(string); ok {
			fields = append(fields, StringV2("request_id", s))
		} else {
			fields = append(fields, StringV2("request_id", fmt.Sprint(v)))
		}
	}

	// 用户ID（可能为int64）
	if v := ctx.Value("user_id"); v != nil {
		switch u := v.(type) {
		case int64:
			fields = append(fields, Int64V2("user_id", u))
		case int:
			fields = append(fields, IntV2("user_id", u))
		case string:
			fields = append(fields, StringV2("user_id", u))
		default:
			fields = append(fields, StringV2("user_id", fmt.Sprint(u)))
		}
	}

	// 跟踪ID
	if v := ctx.Value("trace_id"); v != nil {
		if s, ok := v.(string); ok {
			fields = append(fields, StringV2("trace_id", s))
		} else {
			fields = append(fields, StringV2("trace_id", fmt.Sprint(v)))
		}
	}

	// 订单号
	if v := ctx.Value("order_number"); v != nil {
		if s, ok := v.(string); ok {
			fields = append(fields, StringV2("order_number", s))
		} else {
			fields = append(fields, StringV2("order_number", fmt.Sprint(v)))
		}
	}

	return &LoggerV2{
		Logger: l.Logger.With(fields...),
		config: l.config,
	}
}

// InjectOrderNumber 将订单号注入到上下文，便于全链路日志携带
func InjectOrderNumber(ctx context.Context, orderNumber string) context.Context {
    if ctx == nil {
        ctx = context.Background()
    }
    return context.WithValue(ctx, "order_number", orderNumber)
}

// ========= 分类日志支持 =========
// 允许按功能将日志写入不同文件，例如 "recharge"、"notification" 等

type loggerRegistry struct {
    mu        sync.RWMutex
    loggers   map[string]*LoggerV2
    baseConf  *LoggerConfigV2
}

var categoryRegistry *loggerRegistry

// InitLoggerRegistry 初始化分类日志注册表（可选），传入基础配置用于克隆
func InitLoggerRegistry(config LoggerConfigV2) {
    categoryRegistry = &loggerRegistry{
        loggers:  make(map[string]*LoggerV2),
        baseConf: &config,
    }
}

// getBaseConfig 获取用于创建分类日志器的基础配置
func getBaseConfig() *LoggerConfigV2 {
    if categoryRegistry != nil && categoryRegistry.baseConf != nil {
        return categoryRegistry.baseConf
    }
    if globalLogger != nil && globalLogger.config != nil {
        return globalLogger.config
    }
    // 默认配置（写到 stdout，避免因未初始化而崩溃）
    return &LoggerConfigV2{
        Level:  "info",
        Format: "json",
        Output: "stdout",
        MaxSize:    100,
        MaxBackups: 5,
        MaxAge:     30,
        Compress:   true,
        Caller:     true,
        Stacktrace: true,
    }
}

// GetCategoryLogger 获取或创建指定类别的日志器，输出到 logs/<name>.log
func GetCategoryLogger(name string) *LoggerV2 {
    if name == "" {
        return GetGlobalLogger()
    }
    if categoryRegistry == nil {
        // 懒初始化注册表
        InitLoggerRegistry(*getBaseConfig())
    }
    categoryRegistry.mu.RLock()
    l := categoryRegistry.loggers[name]
    categoryRegistry.mu.RUnlock()
    if l != nil {
        return l
    }
    // 创建新日志器（克隆基础配置并设置独立输出文件）
    base := getBaseConfig()
    cfg := *base
    // 始终写到 logs/<name>.log，确保分类隔离
    cfg.Output = filepath.Join("logs", name+".log")
    newLogger, err := NewLoggerV2(&cfg)
    if err != nil {
        // 退回到全局日志器避免中断
        return GetGlobalLogger()
    }
    categoryRegistry.mu.Lock()
    categoryRegistry.loggers[name] = newLogger
    categoryRegistry.mu.Unlock()
    return newLogger
}

// WithContextCategory 创建带上下文的分类日志器
func WithContextCategory(ctx context.Context, name string) *LoggerV2 {
    l := GetCategoryLogger(name)
    if l == nil {
        return nil
    }
    return l.WithContext(ctx)
}

// WithFields 添加字段
func (l *LoggerV2) WithFields(fields ...Field) *LoggerV2 {
	return &LoggerV2{
		Logger: l.Logger.With(fields...),
		config: l.config,
	}
}

// WithError 添加错误字段
func (l *LoggerV2) WithError(err error) *LoggerV2 {
	return l.WithFields(ErrorV2(err))
}

// LogRequest 记录HTTP请求
func (l *LoggerV2) LogRequest(method, path string, statusCode int, duration time.Duration, fields ...Field) {
	allFields := append(fields,
		String("method", method),
		String("path", path),
		Int("status_code", statusCode),
		Duration("duration", duration),
	)

	if statusCode >= 500 {
		l.Error("HTTP request failed", allFields...)
	} else if statusCode >= 400 {
		l.Warn("HTTP request error", allFields...)
	} else {
		l.Info("HTTP request", allFields...)
	}
}

// LogSQL 记录SQL查询
func (l *LoggerV2) LogSQL(query string, duration time.Duration, err error, fields ...Field) {
	allFields := append(fields,
		String("query", query),
		Duration("duration", duration),
	)

	if err != nil {
		allFields = append(allFields, zap.Error(err))
		l.Error("SQL query failed", allFields...)
	} else if duration > time.Second {
		l.Warn("Slow SQL query", allFields...)
	} else {
		l.Debug("SQL query", allFields...)
	}
}

// LogPanic 记录panic
func (l *LoggerV2) LogPanic(recovered interface{}) {
	stack := make([]byte, 4096)
	length := runtime.Stack(stack, false)

	l.Error("Panic recovered",
		Any("panic", recovered),
		String("stack", string(stack[:length])),
	)
}

// Sync 同步日志
func (l *LoggerV2) Sync() error {
	return l.Logger.Sync()
}

// Close 关闭日志器
func (l *LoggerV2) Close() error {
	return l.Sync()
}

// 便捷的全局日志记录函数 - 使用V2后缀避免冲突
func DebugV2(msg string, fields ...Field) {
	if globalLogger != nil {
		globalLogger.Debug(msg, fields...)
	}
}

func InfoV2(msg string, fields ...Field) {
	if globalLogger != nil {
		globalLogger.Info(msg, fields...)
	}
}

func WarnV2(msg string, fields ...Field) {
	if globalLogger != nil {
		globalLogger.Warn(msg, fields...)
	}
}

func ErrorLogV2(msg string, fields ...Field) {
	if globalLogger != nil {
		globalLogger.Error(msg, fields...)
	}
}

// WithContextV2 创建带上下文的日志器
func WithContextV2(ctx context.Context) *LoggerV2 {
	if globalLogger != nil {
		return globalLogger.WithContext(ctx)
	}
	return nil
}

// 注意：全局便捷函数已移除，请使用V2后缀的版本或直接使用LoggerV2实例
