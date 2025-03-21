# OSS 文件管理系统 (OSS Manager Backend)

对象存储服务(OSS)文件管理系统的后端服务，支持阿里云OSS、AWS S3、CloudFlare R2等多种对象存储服务。

## 功能特点

- 多种对象存储服务支持（阿里云OSS、AWS S3、CloudFlare R2）
- 完善的用户权限管理（RBAC）
- 支持大文件分片上传与断点续传
- 文件MD5异步计算
- 详细的审计日志
- RESTful API 接口

## 技术栈

- Go 1.24
- Gin Web 框架
- GORM (PostgreSQL)
- JWT 认证
- Zap 日志
- Viper 配置管理

## 目录结构

```
ossmanager-backend/
├── cmd/
│   └── api/
│       └── main.go        # 应用入口
├── pkg/
│   ├── api/               # API 请求处理
│   │   ├── handlers/      # 请求处理程序
│   │   ├── middlewares/   # 中间件
│   │   └── routes/        # 路由定义
│   ├── auth/              # 认证相关
│   │   ├── jwt.go
│   │   └── rbac.go
│   ├── config/            # 配置管理
│   │   └── config.go
│   ├── db/                # 数据库相关
│   │   ├── migrations/    # 数据库迁移
│   │   └── models/        # 数据模型
│   ├── logger/            # 日志模块
│   │   └── logger.go
│   ├── oss/               # 对象存储相关
│   │   ├── aliyun.go
│   │   ├── aws.go
│   │   ├── cloudflare.go
│   │   └── interface.go
│   └── utils/             # 工具函数
│       ├── pagination.go
│       ├── response.go
│       └── validator.go
└── configs/               # 配置文件
    ├── app.yaml
    └── oss.yaml
```

## 安装与运行

### 前提条件

- Go 1.21+
- PostgreSQL 13+
- 对象存储服务（阿里云OSS、AWS S3 或 CloudFlare R2）

### 配置

1. 复制示例配置文件并进行修改

```bash
cp configs/app.example.yaml configs/app.yaml
cp configs/oss.example.yaml configs/oss.yaml
```

2. 修改 `configs/app.yaml` 和 `configs/oss.yaml` 中的配置项

### 构建与运行

```bash
# 编译
go build -o bin/ossmanager-api ./cmd/api

# 运行
./bin/ossmanager-api
```

### 使用 Docker 运行

```bash
# 构建镜像
docker build -t ossmanager-backend .

# 运行容器
docker run -p 8080:8080 --env-file .env ossmanager-backend
```

## API 文档

API 文档请参考 [API.md](./API.md) 文件。

## 数据库迁移

项目使用的数据库迁移文件位于 `pkg/db/migrations` 目录下，可以使用以下命令进行迁移：

```bash
# 创建数据库（如果不存在）
createdb -U postgres ossmanager

# 执行迁移
psql -U postgres -d ossmanager -f pkg/db/migrations/001_init_schema.sql
```

## 开发

### 目录说明

- `cmd/api`: 应用程序入口
- `pkg/api`: API 请求处理逻辑
- `pkg/auth`: 认证与授权
- `pkg/config`: 配置管理
- `pkg/db`: 数据库相关
- `pkg/logger`: 日志管理
- `pkg/oss`: 对象存储服务
- `pkg/utils`: 工具函数

### 编码规范

项目遵循 Go 语言最佳实践和编码规范。请参考 [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments) 进行开发。

## 许可证

[MIT License](LICENSE)
