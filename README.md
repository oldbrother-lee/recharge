# Recharge Go

A Go-based  recharge system with user management and authentication.

## Features

- User registration and authentication
- JWT-based authentication
- User profile management
- Role-based access control
- RESTful API endpoints
- MySQL database integration
- Structured logging with Zap
- Configuration management with Viper

## Project Structure

```
├── api/                    # API 接口定义
│   ├── v1/                # API 版本1
│   │   ├── user.go        # 用户接口定义
│   │   ├── order.go       # 订单接口定义
│   │   └── permission.go  # 权限接口定义
│   └── swagger/           # Swagger 文档
├── cmd/                    # 应用程序入口
│   └── main.go            # 主程序入口
├── configs/               # 配置文件
│   ├── config.yaml        # 主配置文件
│   ├── config.dev.yaml    # 开发环境配置
│   └── config.prod.yaml   # 生产环境配置
├── docs/                  # 文档
│   ├── api/              # API文档
│   ├── swagger/          # Swagger文档
│   └── README.md         # 项目说明文档
├── internal/              # 内部包
│   ├── config/           # 配置相关
│   ├── constant/         # 常量定义
│   ├── controller/       # 控制器层
│   ├── middleware/       # 中间件
│   ├── model/           # 数据模型
│   ├── repository/      # 数据访问层
│   ├── service/         # 业务逻辑层
│   ├── router/          # 路由定义
│   └── utils/           # 工具函数
├── pkg/                 # 公共包
│   ├── database/        # 数据库相关
│   ├── logger/          # 日志相关
│   └── validator/       # 验证器
└── scripts/            # 脚本文件
```

## Getting Started

### Prerequisites

- Go 1.16 or later
- MySQL 5.7 or later

### Installation

1. Clone the repository:
```bash
git clone https://github.com/yourusername/recharge-go.git
cd recharge-go
```

2. Install dependencies:
```bash
go mod download
```

3. Configure the database:
- Create a MySQL database
- Update the database configuration in `configs/config.yaml`

4. Run the application:
```bash
go run cmd/main.go
```

## API Endpoints

### User Management

- `POST /api/v1/user/register` - Register a new user
- `POST /api/v1/user/login` - User login
- `GET /api/v1/user/profile` - Get user profile
- `PUT /api/v1/user/profile` - Update user profile
- `PUT /api/v1/user/password` - Change password
- `GET /api/v1/user/list` - List users

## Configuration

The application uses Viper for configuration management. The main configuration file is located at `configs/config.yaml`. You can create environment-specific configurations by creating `config.dev.yaml` and `config.prod.yaml`.

## License

This project is licensed under the MIT License - see the LICENSE file for details. 
