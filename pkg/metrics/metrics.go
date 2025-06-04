package metrics

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	loggerV2 "recharge-go/pkg/logger"
)

// MetricsManager 指标管理器
type MetricsManager struct {
	registry *prometheus.Registry
	logger   *loggerV2.LoggerV2

	// HTTP指标
	httpRequestsTotal   *prometheus.CounterVec
	httpRequestDuration *prometheus.HistogramVec
	httpRequestSize     *prometheus.HistogramVec
	httpResponseSize    *prometheus.HistogramVec

	// 数据库指标
	dbConnectionsActive prometheus.Gauge
	dbConnectionsIdle   prometheus.Gauge
	dbConnectionsTotal  prometheus.Gauge
	dbQueryDuration     *prometheus.HistogramVec
	dbQueriesTotal      *prometheus.CounterVec

	// 业务指标
	businessOperationsTotal   *prometheus.CounterVec
	businessOperationDuration *prometheus.HistogramVec

	// 系统指标
	goroutinesCount prometheus.Gauge
	memoryUsage     prometheus.Gauge
	cpuUsage        prometheus.Gauge
}

// NewMetricsManager 创建指标管理器
func NewMetricsManager(logger *loggerV2.LoggerV2) *MetricsManager {
	registry := prometheus.NewRegistry()

	mm := &MetricsManager{
		registry: registry,
		logger:   logger,
	}

	mm.initMetrics()
	mm.registerMetrics()

	return mm
}

// initMetrics 初始化指标
func (mm *MetricsManager) initMetrics() {
	// HTTP指标
	mm.httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status_code"},
	)

	mm.httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	mm.httpRequestSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_size_bytes",
			Help:    "HTTP request size in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"method", "endpoint"},
	)

	mm.httpResponseSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_response_size_bytes",
			Help:    "HTTP response size in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"method", "endpoint"},
	)

	// 数据库指标
	mm.dbConnectionsActive = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_connections_active",
			Help: "Number of active database connections",
		},
	)

	mm.dbConnectionsIdle = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_connections_idle",
			Help: "Number of idle database connections",
		},
	)

	mm.dbConnectionsTotal = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_connections_total",
			Help: "Total number of database connections",
		},
	)

	mm.dbQueryDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_query_duration_seconds",
			Help:    "Database query duration in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5},
		},
		[]string{"operation", "table"},
	)

	mm.dbQueriesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_queries_total",
			Help: "Total number of database queries",
		},
		[]string{"operation", "table", "status"},
	)

	// 业务指标
	mm.businessOperationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "business_operations_total",
			Help: "Total number of business operations",
		},
		[]string{"operation", "status"},
	)

	mm.businessOperationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "business_operation_duration_seconds",
			Help:    "Business operation duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"},
	)

	// 系统指标
	mm.goroutinesCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "goroutines_count",
			Help: "Number of goroutines",
		},
	)

	mm.memoryUsage = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "memory_usage_bytes",
			Help: "Memory usage in bytes",
		},
	)

	mm.cpuUsage = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "cpu_usage_percent",
			Help: "CPU usage percentage",
		},
	)
}

// registerMetrics 注册指标
func (mm *MetricsManager) registerMetrics() {
	// HTTP指标
	mm.registry.MustRegister(mm.httpRequestsTotal)
	mm.registry.MustRegister(mm.httpRequestDuration)
	mm.registry.MustRegister(mm.httpRequestSize)
	mm.registry.MustRegister(mm.httpResponseSize)

	// 数据库指标
	mm.registry.MustRegister(mm.dbConnectionsActive)
	mm.registry.MustRegister(mm.dbConnectionsIdle)
	mm.registry.MustRegister(mm.dbConnectionsTotal)
	mm.registry.MustRegister(mm.dbQueryDuration)
	mm.registry.MustRegister(mm.dbQueriesTotal)

	// 业务指标
	mm.registry.MustRegister(mm.businessOperationsTotal)
	mm.registry.MustRegister(mm.businessOperationDuration)

	// 系统指标
	mm.registry.MustRegister(mm.goroutinesCount)
	mm.registry.MustRegister(mm.memoryUsage)
	mm.registry.MustRegister(mm.cpuUsage)
}

// HTTPMetricsMiddleware HTTP指标中间件
func (mm *MetricsManager) HTTPMetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.FullPath()
		method := c.Request.Method

		// 记录请求大小
		if c.Request.ContentLength > 0 {
			mm.httpRequestSize.WithLabelValues(method, path).Observe(float64(c.Request.ContentLength))
		}

		c.Next()

		// 记录响应指标
		duration := time.Since(start)
		statusCode := strconv.Itoa(c.Writer.Status())

		mm.httpRequestsTotal.WithLabelValues(method, path, statusCode).Inc()
		mm.httpRequestDuration.WithLabelValues(method, path).Observe(duration.Seconds())
		mm.httpResponseSize.WithLabelValues(method, path).Observe(float64(c.Writer.Size()))
	}
}

// RecordDBMetrics 记录数据库指标
func (mm *MetricsManager) RecordDBMetrics(operation, table string, duration time.Duration, err error) {
	status := "success"
	if err != nil {
		status = "error"
	}

	mm.dbQueriesTotal.WithLabelValues(operation, table, status).Inc()
	mm.dbQueryDuration.WithLabelValues(operation, table).Observe(duration.Seconds())
}

// UpdateDBConnectionMetrics 更新数据库连接指标
func (mm *MetricsManager) UpdateDBConnectionMetrics(active, idle, total int) {
	mm.dbConnectionsActive.Set(float64(active))
	mm.dbConnectionsIdle.Set(float64(idle))
	mm.dbConnectionsTotal.Set(float64(total))
}

// RecordBusinessOperation 记录业务操作
func (mm *MetricsManager) RecordBusinessOperation(operation string, duration time.Duration, err error) {
	status := "success"
	if err != nil {
		status = "error"
	}

	mm.businessOperationsTotal.WithLabelValues(operation, status).Inc()
	mm.businessOperationDuration.WithLabelValues(operation).Observe(duration.Seconds())
}

// UpdateSystemMetrics 更新系统指标
func (mm *MetricsManager) UpdateSystemMetrics(goroutines int, memoryBytes uint64, cpuPercent float64) {
	mm.goroutinesCount.Set(float64(goroutines))
	mm.memoryUsage.Set(float64(memoryBytes))
	mm.cpuUsage.Set(cpuPercent)
}

// GetHandler 获取Prometheus处理器
func (mm *MetricsManager) GetHandler() http.Handler {
	return promhttp.HandlerFor(mm.registry, promhttp.HandlerOpts{})
}

// StartSystemMetricsCollector 启动系统指标收集器
func (mm *MetricsManager) StartSystemMetricsCollector(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			mm.collectSystemMetrics()
		}
	}
}

// collectSystemMetrics 收集系统指标
func (mm *MetricsManager) collectSystemMetrics() {
	// 这里可以集成系统监控库来收集实际的系统指标
	// 例如：gopsutil, runtime等

	// 示例：收集goroutine数量
	// import "runtime"
	// mm.goroutinesCount.Set(float64(runtime.NumGoroutine()))

	// 示例：收集内存使用情况
	// var m runtime.MemStats
	// runtime.ReadMemStats(&m)
	// mm.memoryUsage.Set(float64(m.Alloc))
}

// BusinessOperationTimer 业务操作计时器
type BusinessOperationTimer struct {
	mm        *MetricsManager
	operation string
	startTime time.Time
}

// NewBusinessOperationTimer 创建业务操作计时器
func (mm *MetricsManager) NewBusinessOperationTimer(operation string) *BusinessOperationTimer {
	return &BusinessOperationTimer{
		mm:        mm,
		operation: operation,
		startTime: time.Now(),
	}
}

// Finish 完成计时
func (bot *BusinessOperationTimer) Finish(err error) {
	duration := time.Since(bot.startTime)
	bot.mm.RecordBusinessOperation(bot.operation, duration, err)
}

// DBOperationTimer 数据库操作计时器
type DBOperationTimer struct {
	mm        *MetricsManager
	operation string
	table     string
	startTime time.Time
}

// NewDBOperationTimer 创建数据库操作计时器
func (mm *MetricsManager) NewDBOperationTimer(operation, table string) *DBOperationTimer {
	return &DBOperationTimer{
		mm:        mm,
		operation: operation,
		table:     table,
		startTime: time.Now(),
	}
}

// Finish 完成计时
func (dot *DBOperationTimer) Finish(err error) {
	duration := time.Since(dot.startTime)
	dot.mm.RecordDBMetrics(dot.operation, dot.table, duration, err)
}
