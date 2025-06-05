package utils

import (
	"fmt"
	"sync"
	"time"
)

// UptimeManager 系统运行时间管理器
type UptimeManager struct {
	startTime time.Time
	mu        sync.RWMutex
}

var (
	uptimeManager *UptimeManager
	once          sync.Once
)

// GetUptimeManager 获取运行时间管理器实例（单例模式）
func GetUptimeManager() *UptimeManager {
	once.Do(func() {
		uptimeManager = &UptimeManager{
			startTime: time.Now(),
		}
	})
	return uptimeManager
}

// SetStartTime 设置启动时间（用于应用启动时调用）
func (um *UptimeManager) SetStartTime(t time.Time) {
	um.mu.Lock()
	defer um.mu.Unlock()
	um.startTime = t
}

// GetStartTime 获取启动时间
func (um *UptimeManager) GetStartTime() time.Time {
	um.mu.RLock()
	defer um.mu.RUnlock()
	return um.startTime
}

// GetUptime 获取运行时间（格式化字符串）
func (um *UptimeManager) GetUptime() string {
	um.mu.RLock()
	defer um.mu.RUnlock()

	duration := time.Since(um.startTime)
	days := int(duration.Hours()) / 24
	hours := int(duration.Hours()) % 24
	minutes := int(duration.Minutes()) % 60

	return fmt.Sprintf("%d天%d小时%d分钟", days, hours, minutes)
}

// GetUptimeDuration 获取运行时间（Duration类型）
func (um *UptimeManager) GetUptimeDuration() time.Duration {
	um.mu.RLock()
	defer um.mu.RUnlock()
	return time.Since(um.startTime)
}

// GetSystemInfo 获取系统信息
func (um *UptimeManager) GetSystemInfo() map[string]interface{} {
	um.mu.RLock()
	defer um.mu.RUnlock()

	return map[string]interface{}{
		"uptime":         um.GetUptime(),
		"start_time":     um.startTime.Format("2006-01-02 15:04:05"),
		"current_time":   time.Now().Format("2006-01-02 15:04:05"),
		"uptime_seconds": int64(time.Since(um.startTime).Seconds()),
	}
}
