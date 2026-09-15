# Knowbase Server - 知识方舟平台

知识方舟平台后端服务，负责围绕 DevOps 研发流程相关知识的管理和分发。

## 快速开始

### 环境要求

- Go 1.24+
- MySQL 8.0+
- 百度智能云 BOS 存储

### 环境变量配置

启动服务前需要设置以下环境变量：

```bash
# 必需的环境变量
export CONFIG_PATH=/path/to/resource/config/dev.yaml  # 配置文件路径
export PORT=8080                                       # 服务端口
export DB_CONNECTION="user:password@tcp(host:port)/dbname?charset=utf8mb4&parseTime=True&loc=Local"  # 数据库连接字符串
export BOS_AK=your_bos_access_key                     # BOS AccessKey
export BOS_SK=your_bos_secret_key                     # BOS SecretKey

# 可选的环境变量
export JWT_SECRET=your-jwt-secret-key                 # JWT密钥（默认值见代码）
export AUTH_HEADER_KEY=X-Auth-Token                   # Header认证的Key（默认：x-auth-client-token）
```

### 安装依赖

```bash
go mod download
```

### 数据库初始化

执行SQL脚本初始化数据库：

```bash
mysql -u username -p database_name < resource/sql/001_initial_schema.sql
```

### 编译和运行

```bash
# 编译
make build

# 运行
./knowbase-server
```

或者直接运行：

```bash
go run main.go
```

## 项目结构

详细的项目结构和架构设计请参考：
- [设计文档](docs/design.md)
- [CLAUDE.md](CLAUDE.md)

## API接口

## 如何贡献

贡献patch流程、质量要求请参考 [百度 Go 编码规范](https://ku.baidu-int.com/d/AD4lVjdO8PfAC6)

## 链接
- [编码规范](https://ku.baidu-int.com/d/AD4lVjdO8PfAC6)
- [编译环境使用指南](https://ku.baidu-int.com/d/SzGt0sD37hWmmp)