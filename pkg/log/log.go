package log

import (
    "context"
    "fmt"
    "os"
    "path/filepath"
    "runtime"
    "strings"
    "time"

    "github.com/gin-gonic/gin"
    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
    "gopkg.in/natefinch/lumberjack.v2"
)

type Config struct {
    Level      string
    Format     string
    Output     string
    MaxSize    int
    MaxBackups int
    MaxAge     int
    Compress   bool
    Caller     bool
    Stacktrace bool
    SamplingInitial   int
    SamplingThereafter int
}

var global *zap.Logger
var Log *zap.Logger
var atomicLevel zap.AtomicLevel

func Init(cfg Config) error {
    lvl, err := zapcore.ParseLevel(cfg.Level)
    if err != nil {
        return err
    }
    atomicLevel = zap.NewAtomicLevelAt(lvl)
    encCfg := zap.NewProductionEncoderConfig()
    encCfg.TimeKey = "timestamp"
    encCfg.LevelKey = "level"
    encCfg.NameKey = "logger"
    encCfg.CallerKey = "caller"
    encCfg.MessageKey = "message"
    encCfg.StacktraceKey = "stacktrace"
    encCfg.EncodeTime = zapcore.ISO8601TimeEncoder
    encCfg.EncodeLevel = zapcore.LowercaseLevelEncoder
    encCfg.EncodeDuration = zapcore.SecondsDurationEncoder
    encCfg.EncodeCaller = zapcore.ShortCallerEncoder
    var enc zapcore.Encoder
    switch cfg.Format {
    case "console":
        enc = zapcore.NewConsoleEncoder(encCfg)
    default:
        enc = zapcore.NewJSONEncoder(encCfg)
    }
    var ws zapcore.WriteSyncer
    if cfg.Output == "stdout" || cfg.Output == "" {
        ws = zapcore.AddSync(os.Stdout)
    } else {
        if err := os.MkdirAll(filepath.Dir(cfg.Output), 0755); err != nil {
            ws = zapcore.AddSync(os.Stdout)
        } else {
            lj := &lumberjack.Logger{Filename: cfg.Output, MaxSize: cfg.MaxSize, MaxBackups: cfg.MaxBackups, MaxAge: cfg.MaxAge, Compress: cfg.Compress}
            ws = zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), zapcore.AddSync(lj))
        }
    }
    core := zapcore.NewCore(enc, ws, atomicLevel)
    if cfg.SamplingInitial > 0 && cfg.SamplingThereafter > 0 {
        core = zapcore.NewSamplerWithOptions(core, time.Second, cfg.SamplingInitial, cfg.SamplingThereafter)
    }
    opts := []zap.Option{}
    if cfg.Caller {
        opts = append(opts, zap.AddCaller(), zap.AddCallerSkip(1))
    }
    if cfg.Stacktrace {
        opts = append(opts, zap.AddStacktrace(zapcore.ErrorLevel))
    } else {
        opts = append(opts, zap.AddStacktrace(zapcore.FatalLevel))
    }
    global = zap.New(core, opts...)
    Log = global
    zap.ReplaceGlobals(global)
    return nil
}

func L() *zap.Logger {
    if global == nil {
        logger, _ := zap.NewProduction(zap.AddCallerSkip(1))
        global = logger
        Log = global
    }
    return global
}

func WithContext(ctx context.Context) *zap.Logger {
    l := L()
    fields := []zap.Field{}
    if v := ctx.Value("request_id"); v != nil {
        if s, ok := v.(string); ok {
            fields = append(fields, zap.String("request_id", s))
        } else {
            fields = append(fields, zap.String("request_id", fmt.Sprint(v)))
        }
    }
    if v := ctx.Value("user_id"); v != nil {
        switch u := v.(type) {
        case int64:
            fields = append(fields, zap.Int64("user_id", u))
        case int:
            fields = append(fields, zap.Int("user_id", u))
        case string:
            fields = append(fields, zap.String("user_id", u))
        default:
            fields = append(fields, zap.String("user_id", fmt.Sprint(u)))
        }
    }
    if v := ctx.Value("trace_id"); v != nil {
        if s, ok := v.(string); ok {
            fields = append(fields, zap.String("trace_id", s))
        } else {
            fields = append(fields, zap.String("trace_id", fmt.Sprint(v)))
        }
    }
    if v := ctx.Value("order_number"); v != nil {
        if s, ok := v.(string); ok {
            fields = append(fields, zap.String("order_number", s))
        } else {
            fields = append(fields, zap.String("order_number", fmt.Sprint(v)))
        }
    }
    return l.With(fields...)
}

func SetLevel(level string) error {
    lvl, err := zapcore.ParseLevel(level)
    if err != nil {
        return err
    }
    atomicLevel.SetLevel(lvl)
    return nil
}

func GetLevel() string { return atomicLevel.Level().String() }

func Named(name string) *zap.Logger { return L().Named(name) }

func With(fields ...zap.Field) *zap.Logger { return L().With(fields...) }

func InjectOrderNumber(ctx context.Context, orderNumber string) context.Context {
    if ctx == nil {
        ctx = context.Background()
    }
    return context.WithValue(ctx, "order_number", orderNumber)
}

func Debug(ctx context.Context, msg string, fields ...zap.Field) { WithContext(ctx).Debug(msg, fields...) }
func Info(ctx context.Context, msg string, fields ...zap.Field)  { WithContext(ctx).Info(msg, fields...) }
func Warn(ctx context.Context, msg string, fields ...zap.Field)  { WithContext(ctx).Warn(msg, fields...) }
func Error(ctx context.Context, msg string, fields ...zap.Field) { WithContext(ctx).Error(msg, fields...) }

func String(k, v string) zap.Field   { return zap.String(k, v) }
func Int(k string, v int) zap.Field  { return zap.Int(k, v) }
func Int64(k string, v int64) zap.Field { return zap.Int64(k, v) }
func Bool(k string, v bool) zap.Field { return zap.Bool(k, v) }
func Time(k string, v time.Time) zap.Field { return zap.Time(k, v) }
func Duration(k string, v time.Duration) zap.Field { return zap.Duration(k, v) }
func Float64(k string, v float64) zap.Field { return zap.Float64(k, v) }
func Any(k string, v interface{}) zap.Field { return zap.Any(k, v) }
func Err(e error) zap.Field { return zap.Error(e) }

func GinLogger() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        path := c.Request.URL.Path
        query := c.Request.URL.RawQuery
        c.Next()
        dur := time.Since(start)
        st := c.Writer.Status()
        l := WithContext(c.Request.Context())
        lf := []zap.Field{
            String("method", c.Request.Method),
            String("path", path),
            Int("status_code", st),
            Duration("duration", dur),
            String("query", query),
            String("ip", c.ClientIP()),
            String("user_agent", c.Request.UserAgent()),
            String("content_type", c.ContentType()),
            String("errors", c.Errors.ByType(gin.ErrorTypePrivate).String()),
        }
        if st >= 500 {
            l.Error("http_request_failed", lf...)
        } else if st >= 400 {
            l.Warn("http_request_error", lf...)
        } else {
            l.Info("http_request", lf...)
        }
    }
}

func GinRecovery() gin.HandlerFunc {
    return func(c *gin.Context) {
        defer func() {
            if err := recover(); err != nil {
                stack := make([]byte, 4096)
                n := runtime.Stack(stack, false)
                WithContext(c.Request.Context()).Error("panic_recovered",
                    Any("panic", err),
                    String("stack", shortenStack(string(stack[:n]))),
                )
                c.AbortWithStatus(500)
            }
        }()
        c.Next()
    }
}

func SQL(ctx context.Context, query string, dur time.Duration, err error, fields ...zap.Field) {
    l := WithContext(ctx)
    fs := append(fields, String("query", query), Duration("duration", dur))
    if err != nil {
        fs = append(fs, Err(err))
        l.Error("sql_failed", fs...)
    } else if dur > time.Second {
        l.Warn("sql_slow", fs...)
    } else {
        l.Debug("sql", fs...)
    }
}

func shortenStack(s string) string {
    lines := strings.Split(s, "\n")
    for i := range lines {
        if idx := strings.Index(lines[i], "recharge-go/"); idx >= 0 {
            p := lines[i][idx:]
            lines[i] = p
        }
    }
    return strings.Join(lines, "\n")
}

// ===== 迁移期兼容辅助 =====
// 提供与旧版命名相同的别名，统一到新封装，便于逐步替换

func StringV2(k, v string) zap.Field     { return zap.String(k, v) }
func IntV2(k string, v int) zap.Field    { return zap.Int(k, v) }
func Int64V2(k string, v int64) zap.Field { return zap.Int64(k, v) }
func BoolV2(k string, v bool) zap.Field  { return zap.Bool(k, v) }
func TimeV2(k string, v time.Time) zap.Field { return zap.Time(k, v) }
func DurationV2(k string, v time.Duration) zap.Field { return zap.Duration(k, v) }
func Float64V2(k string, v float64) zap.Field { return zap.Float64(k, v) }
func AnyV2(k string, v interface{}) zap.Field { return zap.Any(k, v) }
func ErrorV2(e error) zap.Field         { return zap.Error(e) }

func WithContextV2(ctx context.Context) *zap.Logger { return WithContext(ctx) }

func WithContextCategory(ctx context.Context, name string) *zap.Logger {
    return WithContext(ctx).With(String("category", name))
}

func GetCategoryLogger(name string) *zap.Logger {
    return L().With(String("category", name))
}

func InfoV2(msg string, fields ...zap.Field)  { L().Info(msg, fields...) }
func WarnV2(msg string, fields ...zap.Field)  { L().Warn(msg, fields...) }
func ErrorLogV2(msg string, fields ...zap.Field) { L().Error(msg, fields...) }
func DebugV2(msg string, fields ...zap.Field) { L().Debug(msg, fields...) }
