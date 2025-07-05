# 订单数量监控功能

## 功能概述

订单数量监控功能是一个自动化的订单流量控制机制，用于防止系统因处理中订单过多而导致性能问题。当处理中的订单数量达到预设阈值时，系统会自动暂停新订单的拉取；当订单数量降到安全水平时，系统会自动恢复订单拉取。

## 核心特性

- **自动监控**：实时监控处理中订单的数量
- **智能暂停**：当订单数量超过阈值时自动暂停拉单
- **自动恢复**：当订单数量降到安全水平时自动恢复拉单
- **线程安全**：使用读写锁保证并发安全
- **可配置**：通过配置文件灵活设置阈值

## 配置说明

在 `configs/config.yaml` 文件中的 `task` 部分添加以下配置：

```yaml
task:
  # ... 其他配置 ...
  suspend_threshold: 1000  # 暂停阈值：处理中订单数量达到此值时暂停拉单
  resume_threshold: 800    # 恢复阈值：处理中订单数量降到此值时恢复拉单
```

### 配置参数说明

- `suspend_threshold`：暂停阈值，当处理中订单数量达到或超过此值时，系统会暂停新订单的拉取
- `resume_threshold`：恢复阈值，当处理中订单数量降到此值以下时，系统会恢复订单拉取

**注意**：`suspend_threshold` 必须大于 `resume_threshold`，以避免频繁的暂停/恢复切换。

## 工作原理

1. **监控循环**：在每次订单查询前，系统会检查当前处理中订单的数量
2. **阈值判断**：
   - 如果订单数量 >= `suspend_threshold`，则暂停拉单
   - 如果订单数量 < `resume_threshold`，则恢复拉单
   - 如果订单数量在两个阈值之间，则保持当前状态不变
3. **状态控制**：通过 `isPullingAllowed()` 方法控制是否允许拉取新订单

## 代码实现

### 新增的仓库方法

在 `internal/repository/order.go` 中新增了以下方法：

```go
// CountByStatuses 统计指定状态的订单数量
CountByStatuses(ctx context.Context, statuses []int) (int64, error)

// CountProcessingOrders 统计处理中的订单数量（状态为1的订单）
CountProcessingOrders(ctx context.Context) (int64, error)
```

### 新增的服务方法

在 `internal/service/task.go` 中新增了以下方法：

```go
// checkOrderThresholds 检查订单数量阈值
checkOrderThresholds(ctx context.Context) error

// suspendPulling 暂停拉单
suspendPulling()

// resumePulling 恢复拉单
resumePulling()

// isPullingAllowed 检查是否允许拉单
isPullingAllowed() bool
```

## 使用示例

### 配置示例

```yaml
task:
  interval: 30
  order_details_interval: 60
  max_retries: 3
  retry_delay: 5
  max_concurrent: 10
  batch_size: 100
  suspend_threshold: 1000  # 当处理中订单达到1000个时暂停
  resume_threshold: 800    # 当处理中订单降到800个以下时恢复
```

### 日志输出

系统会在以下情况输出日志：

- 暂停拉单时：`"订单数量超过阈值，暂停拉单: count=1050, threshold=1000"`
- 恢复拉单时：`"订单数量降到安全水平，恢复拉单: count=750, threshold=800"`
- 跳过查询时：`"拉单已暂停，跳过查询: TaskID=123"`

## 监控和调试

### 测试

运行测试以验证功能：

```bash
go test ./test -v -run TestOrderThreshold
```

### 性能影响

- 每次订单查询前会执行一次数据库查询来统计订单数量
- 使用了高效的 SQL COUNT 查询，性能影响最小
- 读写锁的使用保证了并发安全，但不会显著影响性能

## 注意事项

1. **阈值设置**：根据系统处理能力合理设置阈值，避免设置过低导致频繁暂停
2. **数据库性能**：确保订单表在状态字段上有索引，以提高统计查询性能
3. **监控告警**：建议配置监控告警，当触发暂停/恢复时及时通知运维人员
4. **日志级别**：调试时可以将日志级别设置为 DEBUG 以查看详细的状态变化

## 故障排除

### 常见问题

1. **配置不生效**：检查配置文件格式是否正确，重启服务
2. **频繁暂停恢复**：调整阈值间距，确保 `suspend_threshold - resume_threshold` 有足够的缓冲
3. **性能问题**：检查订单表索引，优化统计查询性能

### 调试方法

1. 查看日志输出，确认监控逻辑是否正常执行
2. 检查数据库中处理中订单的实际数量
3. 验证配置文件中的阈值设置是否合理