# 外部订单API系统

## 项目概述

本项目为充值系统提供了完整的外部API接口，允许第三方系统通过标准的RESTful API进行充值订单的创建、查询和状态回调。系统采用Go语言开发，基于Gin框架，提供高性能、安全可靠的API服务。

## 系统架构

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   第三方系统    │────│   外部API网关   │────│   充值核心系统   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                              │
                              ▼
                       ┌─────────────────┐
                       │   数据库存储    │
                       └─────────────────┘
```

### 核心组件

1. **认证中间件** (`internal/middleware/external_auth.go`)
   - API密钥验证
   - 签名校验
   - IP白名单检查
   - 请求频率限制

2. **外部订单控制器** (`internal/controller/external_order.go`)
   - 订单创建接口
   - 订单查询接口
   - 请求参数验证
   - 响应格式统一

3. **回调控制器** (`internal/controller/external_callback.go`)
   - 处理外部系统回调
   - 订单状态更新
   - 回调结果验证

4. **数据模型**
   - `ExternalAPIKey` (`internal/model/external_api_key.go`) - API密钥管理
   - `ExternalOrderLog` (`internal/model/external_order_log.go`) - 外部订单日志

5. **工具库**
   - 签名算法 (`internal/utils/signature.go`)
   - 数据仓库 (`internal/repository/external_api_key_repository.go`)

## 功能特性

### 🔐 安全特性
- **MD5签名算法**: 统一使用MD5签名算法，简化接入流程
- **时间戳验证**: 防止重放攻击
- **IP白名单**: 限制访问来源
- **请求频率限制**: 防止恶意请求
- **API密钥管理**: 支持密钥轮换和状态控制

### 📊 监控与日志
- **完整的请求日志**: 记录所有API调用
- **错误追踪**: 详细的错误信息和堆栈
- **性能监控**: 请求响应时间统计
- **状态变更日志**: 订单状态变更历史

### 🚀 高性能
- **异步处理**: 非阻塞的订单处理
- **连接池**: 数据库连接复用
- **缓存机制**: API密钥缓存
- **负载均衡**: 支持水平扩展

## 快速开始

### 环境要求

- Go 1.19+
- MySQL 8.0+
- Redis 6.0+ (可选，用于缓存)

### 安装部署

1. **克隆项目**
```bash
git clone <repository-url>
cd recharge-go
```

2. **安装依赖**
```bash
go mod download
```

3. **数据库初始化**
```bash
# 执行数据库迁移
mysql -u root -p < migrations/create_external_api_tables.sql
```

4. **配置文件**
```yaml
# config/config.yaml
database:
  host: localhost
  port: 3306
  username: root
  password: your_password
  database: recharge_db

server:
  port: 8080
  mode: release

external_api:
  signature_timeout: 300  # 签名有效期（秒）
  rate_limit: 100        # 每分钟请求限制
```

5. **启动服务**
```bash
go run main.go
```

### API密钥管理

1. **创建API密钥**
```sql
INSERT INTO external_api_keys (
    app_id, app_key, app_secret, app_name, 
    description, status, ip_whitelist, rate_limit
) VALUES (
    'your_app_001', 
    'key_123456789abcdef', 
    'secret_abcdefghijklmnopqrstuvwxyz123456',
    '测试应用',
    '用于测试的API密钥',
    1,
    '192.168.1.0/24,10.0.0.0/8',
    1000
);
```

2. **密钥状态管理**
```sql
-- 启用密钥
UPDATE external_api_keys SET status = 1 WHERE app_id = 'your_app_001';

-- 禁用密钥
UPDATE external_api_keys SET status = 0 WHERE app_id = 'your_app_001';

-- 删除密钥（软删除）
UPDATE external_api_keys SET deleted_at = NOW() WHERE app_id = 'your_app_001';
```

## API使用示例

### 创建订单

```bash
curl -X POST "https://your-domain.com/external/order" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: key_123456789abcdef" \
  -H "X-Signature: CALCULATED_SIGNATURE" \
  -d '{
    "app_id": "your_app_001",
    "mobile": "13800138000",
    "product_id": 1,
    "out_trade_num": "ORDER_20231201_001",
    "amount": 10.00,
    "timestamp": 1701398400,
    "nonce": "abc123",
    "sign": "CALCULATED_SIGNATURE"
  }'
```

### 查询订单

```bash
curl -X GET "https://your-domain.com/external/order/query?app_id=your_app_001&out_trade_num=ORDER_20231201_001&timestamp=1701398400&nonce=abc123&sign=CALCULATED_SIGNATURE" \
  -H "X-API-Key: key_123456789abcdef" \
  -H "X-Signature: CALCULATED_SIGNATURE"
```

## 开发指南

### 项目结构

```
recharge-go/
├── cmd/                    # 应用入口
├── internal/
│   ├── controller/         # 控制器层
│   │   ├── external_order.go
│   │   └── external_callback.go
│   ├── middleware/         # 中间件
│   │   └── external_auth.go
│   ├── model/             # 数据模型
│   │   ├── external_api_key.go
│   │   └── external_order_log.go
│   ├── repository/        # 数据访问层
│   │   └── external_api_key_repository.go
│   ├── service/           # 业务逻辑层
│   ├── router/            # 路由配置
│   │   └── external_order.go
│   └── utils/             # 工具库
│       └── signature.go
├── pkg/                   # 公共包
├── docs/                  # 文档
│   └── external_api.md
├── migrations/            # 数据库迁移
│   └── create_external_api_tables.sql
└── README_EXTERNAL_API.md
```

### 添加新的API接口

1. **定义路由**
```go
// internal/router/external_order.go
func SetupExternalOrderRoutes(r *gin.Engine, db *gorm.DB, queue queue.Queue) {
    // ... 现有代码
    
    // 添加新路由
    externalAPI.POST("/new-endpoint", controller.NewEndpoint)
}
```

2. **实现控制器**
```go
// internal/controller/external_order.go
func (c *ExternalOrderController) NewEndpoint(ctx *gin.Context) {
    // 获取API密钥信息
    apiKey := ctx.MustGet("api_key").(*model.ExternalAPIKey)
    clientIP := ctx.MustGet("client_ip").(string)
    
    // 业务逻辑实现
    // ...
}
```

3. **更新文档**
```markdown
<!-- docs/external_api.md -->
### 新接口

**接口地址**: `POST /external/new-endpoint`

**请求参数**:
...
```

### 自定义签名算法

```go
// internal/utils/signature.go
func (v *SignatureValidator) GenerateCustomSignature(params map[string]interface{}) (string, error) {
    // 实现自定义签名算法
    // ...
    return signature, nil
}
```

### 扩展认证中间件

```go
// internal/middleware/external_auth.go
func (m *ExternalAuthMiddleware) CustomValidation() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 自定义验证逻辑
        // ...
        c.Next()
    }
}
```

## 监控与运维

### 日志配置

```go
// 配置日志级别和输出格式
logrus.SetLevel(logrus.InfoLevel)
logrus.SetFormatter(&logrus.JSONFormatter{})
```

### 性能监控

```go
// 添加性能监控中间件
func PerformanceMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        c.Next()
        duration := time.Since(start)
        
        logrus.WithFields(logrus.Fields{
            "method":   c.Request.Method,
            "path":     c.Request.URL.Path,
            "duration": duration.Milliseconds(),
            "status":   c.Writer.Status(),
        }).Info("API Request")
    }
}
```

### 健康检查

```go
// 添加健康检查接口
func HealthCheck(c *gin.Context) {
    c.JSON(200, gin.H{
        "status": "ok",
        "timestamp": time.Now().Unix(),
        "version": "1.0.0",
    })
}
```

## 测试

### 单元测试

```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./internal/utils

# 生成测试覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### 集成测试

```go
// internal/controller/external_order_test.go
func TestCreateOrder(t *testing.T) {
    // 设置测试环境
    router := setupTestRouter()
    
    // 构造测试请求
    req := httptest.NewRequest("POST", "/external/order", strings.NewReader(testData))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-API-Key", "test_key")
    
    // 执行请求
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)
    
    // 验证结果
    assert.Equal(t, 200, w.Code)
}
```

## 部署

### Docker部署

```dockerfile
# Dockerfile
FROM golang:1.19-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o main .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .
CMD ["./main"]
```

```yaml
# docker-compose.yml
version: '3.8'
services:
  api:
    build: .
    ports:
      - "8080:8080"
    environment:
      - DB_HOST=mysql
      - DB_PORT=3306
      - DB_USER=root
      - DB_PASSWORD=password
      - DB_NAME=recharge_db
    depends_on:
      - mysql
      
  mysql:
    image: mysql:8.0
    environment:
      - MYSQL_ROOT_PASSWORD=password
      - MYSQL_DATABASE=recharge_db
    volumes:
      - mysql_data:/var/lib/mysql
      - ./migrations:/docker-entrypoint-initdb.d
      
volumes:
  mysql_data:
```

### Kubernetes部署

```yaml
# k8s/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: external-api
spec:
  replicas: 3
  selector:
    matchLabels:
      app: external-api
  template:
    metadata:
      labels:
        app: external-api
    spec:
      containers:
      - name: api
        image: your-registry/external-api:latest
        ports:
        - containerPort: 8080
        env:
        - name: DB_HOST
          value: "mysql-service"
        - name: DB_PORT
          value: "3306"
---
apiVersion: v1
kind: Service
metadata:
  name: external-api-service
spec:
  selector:
    app: external-api
  ports:
  - port: 80
    targetPort: 8080
  type: LoadBalancer
```

## 故障排查

### 常见问题

1. **签名验证失败**
   - 检查时间戳是否在有效范围内
   - 确认参数排序和拼接格式正确
   - 验证app_secret是否正确

2. **IP白名单限制**
   - 检查客户端真实IP
   - 确认白名单配置格式正确
   - 考虑代理和负载均衡的影响

3. **请求频率超限**
   - 检查rate_limit配置
   - 确认请求分布是否均匀
   - 考虑使用连接池

### 日志分析

```bash
# 查看API调用日志
grep "external_api" /var/log/app.log | jq .

# 统计错误率
grep "ERROR" /var/log/app.log | wc -l

# 分析响应时间
grep "duration" /var/log/app.log | awk '{print $5}' | sort -n
```

## 贡献指南

1. Fork 项目
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 打开 Pull Request

## 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 联系我们

- 项目维护者: [Your Name](mailto:your.email@example.com)
- 问题反馈: [GitHub Issues](https://github.com/your-org/recharge-go/issues)
- 技术支持: api-support@your-domain.com