# VC Terraform Registry

[English](#english) | [中文](#中文)

---

## English

A private Terraform Provider Registry offering enterprise-grade module and provider management solutions.

### Features

- **Private Deployment** - Complete private Terraform Provider Registry to protect internal infrastructure code
- **Apple-style UI** - Modern, clean and elegant Web user interface
- **Offline Deployment** - Support fully offline environment deployment, suitable for internal networks
- **Provider Mirror** - Automatically proxy/cache providers from upstream registries (e.g., registry.terraform.io)
- **Manual Upload** - Support manual provider binary upload for air-gapped environments
- **Complete Documentation** - Built-in documentation system with Markdown format support
- **One-click Deployment** - Quick start with Docker Compose, simplified deployment process
- **Version Management** - Complete version control and history tracking
- **Search Functionality** - Quick search and discovery of Providers and modules
- **Access Control** - Support for authentication and authorization mechanisms

### Prerequisites

- Docker 20.10+
- Docker Compose 2.0+
- At least 2GB available memory
- At least 10GB available disk space

### Quick Start

#### Using Docker Compose

1. Clone repository

```bash
git clone https://github.com/Veritas-Calculus/vc-terraform-registry.git
cd vc-terraform-registry
```

2. Configure environment variables

```bash
cp .env.example .env
# Edit .env file to configure necessary parameters
```

3. Start services

```bash
docker-compose up -d
```

4. Access Web UI

Open browser and navigate to: `http://localhost:80`

#### Development Mode

**Backend:**
```bash
cd backend
go mod download
go run cmd/server/main.go
```

**Frontend:**
```bash
cd frontend
npm install
npm run dev
```

### Tech Stack

**Backend:**
- Go 1.24
- Gin (Web Framework)
- GORM (ORM)
- SQLite (Database)
- JWT (Authentication)

**Frontend:**
- React 19
- Vite 7
- TailwindCSS 4
- React Router 7

---

## 中文

一个私有化的 Terraform Provider Registry，提供企业级的模块和 Provider 管理解决方案。

### 特性

- **私有化部署** - 完全的私有化 Terraform Provider Registry，保护企业内部基础设施代码
- **Apple 风格 UI** - 现代化、简洁优雅的 Web 用户界面
- **离线部署** - 支持完全离线环境部署，适合内网环境
- � **Provider 镜像** - 自动代理/缓存上游 Registry（如 registry.terraform.io）的 Provider
- **手动上传** - 支持手动上传 Provider 二进制文件，适用于隔离网络环境
- �**完整文档** - 内置文档系统，支持 Markdown 格式的模块和 Provider 文档
- **一键部署** - 通过 Docker Compose 快速启动，简化部署流程
- **版本管理** - 完整的版本控制和历史记录
- **搜索功能** - 快速搜索和发现 Providers 和模块
- **访问控制** - 支持认证和授权机制

### 前置要求

- Docker 20.10+
- Docker Compose 2.0+
- 至少 2GB 可用内存
- 至少 10GB 可用磁盘空间

### 快速开始

#### 使用 Docker Compose 部署

**前提：确保 Docker Desktop 已启动**

```bash
# 克隆仓库
git clone https://github.com/Veritas-Calculus/vc-terraform-registry.git
cd vc-terraform-registry

# 方式1：使用启动脚本
./start.sh

# 方式2：使用 make 命令
make start

# 方式3：使用 docker-compose
docker-compose up -d
```

服务启动后访问：
- 前端 UI: http://localhost:3000
- 后端 API: http://localhost:8080
- 健康检查: http://localhost:8080/health

#### 开发模式

使用开发模式可以实现热重载：

```bash
# 启动开发环境
make dev-start

# 或使用 docker-compose
docker-compose -f docker-compose.dev.yml up
```

开发模式端口：
- 前端: http://localhost:5173 (Vite 开发服务器)
- 后端: http://localhost:8080

#### 本地开发（不使用 Docker）

**后端：**
```bash
cd backend
go mod download
go run cmd/server/main.go
```

**前端：**
```bash
cd frontend
npm install
npm run dev
```

#### 常用命令

```bash
# 查看日志
make logs
# 或
docker-compose logs -f

# 停止服务
make stop
# 或
./stop.sh
# 或
docker-compose down

# 重启服务
make restart

# 清理所有数据
make clean
```

### 技术栈

**后端：**
- Go 1.24
- Gin (Web框架)
- GORM (ORM)
- SQLite (数据库)
- JWT (认证)

**前端：**
- React 19
- Vite 7
- TailwindCSS 4
- React Router 7

### 项目结构

```
.
├── docker-compose.yml       # Docker Compose 配置
├── .env.example            # 环境变量示例
├── backend/                # 后端服务 (Go)
│   ├── cmd/server/        # 主程序入口
│   ├── internal/
│   │   ├── api/           # API 接口
│   │   ├── auth/          # 认证授权
│   │   ├── models/        # 数据模型
│   │   └── storage/       # 存储层
│   └── pkg/config/        # 配置管理
├── frontend/              # 前端 UI (React)
│   ├── src/
│   │   ├── components/    # React 组件
│   │   ├── pages/         # 页面组件
│   │   └── services/      # API 服务
│   └── public/
└── scripts/               # 部署和维护脚本
```

### 配置 Terraform CLI

在你的 Terraform 配置中添加私有 Registry：

```hcl
terraform {
  required_providers {
    custom = {
      source  = "registry.example.com/namespace/custom"
      version = "~> 1.0"
    }
  }
}
```

配置 Terraform CLI 认证：

```bash
# 在 ~/.terraformrc 或 %APPDATA%/terraform.rc 中添加
credentials "registry.example.com" {
  token = "your-api-token"
}
```

## 项目结构

```
.
├── docker-compose.yml       # Docker Compose 配置
├── .env.example            # 环境变量示例
├── backend/                # 后端服务 (Go)
│   ├── cmd/server/        # 主程序入口
│   ├── internal/
│   │   ├── api/           # API 接口
│   │   ├── auth/          # 认证授权
│   │   ├── models/        # 数据模型
│   │   └── storage/       # 存储层
│   └── pkg/config/        # 配置管理
├── frontend/              # 前端 UI (React)
│   ├── src/
│   │   ├── components/    # React 组件
│   │   ├── pages/         # 页面组件
│   │   └── services/      # API 服务
│   └── public/
└── scripts/               # 部署和维护脚本
```

## 配置说明

### 环境变量

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| `SERVER_PORT` | 服务端口 | `8080` |
| `SERVER_HOST` | 服务主机地址 | `0.0.0.0` |
| `STORAGE_PATH` | Provider 存储路径 | `/data/registry` |
| `DATABASE_URL` | 数据库连接字符串 | `sqlite:///data/registry.db` |
| `AUTH_ENABLED` | 是否启用认证 | `true` |
| `AUTH_SECRETKEY` | JWT 密钥 | `change-me-in-production` |
| `LOG_LEVEL` | 日志级别 | `info` |

### 存储配置

支持多种存储后端：

- **本地文件系统** - 适合单机部署
- **S3 兼容存储** - 适合分布式部署
- **阿里云 OSS** - 国内用户推荐

## API 使用指南

### 健康检查

```bash
curl http://localhost:8080/health
```

### 获取 Provider 列表

```bash
curl http://localhost:8080/api/v1/providers
```

### 上传 Provider

```bash
curl -X POST http://localhost:8080/api/v1/providers \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "namespace": "myorg",
    "name": "myprovider",
    "version": "1.0.0",
    "description": "My custom provider"
  }'
```

### 搜索 Provider

```bash
curl "http://localhost:8080/api/v1/providers/search?q=aws"
```

### 获取 Module 列表

```bash
curl http://localhost:8080/api/v1/modules
```

## Provider 镜像功能 / Provider Mirror Feature

### 使用方式 / Usage

VC Terraform Registry 支持两种方式获取 Provider：

**1. 自动镜像（推荐）**

从上游 registry.terraform.io 自动代理下载并缓存 Provider。当 Terraform 请求 Provider 时，Registry 会自动从上游下载并缓存到本地。

**2. 手动上传**

通过 Web UI 或 API 手动上传 Provider 二进制文件，适用于：
- 内网隔离环境
- 自定义 Provider
- 预下载离线使用

### 配置 Terraform 使用私有 Registry / Configure Terraform

**方式一：使用 Network Mirror（推荐）**

在 `~/.terraformrc` 或 `terraform.rc` 中配置：

```hcl
provider_installation {
  network_mirror {
    url = "http://YOUR_REGISTRY_HOST:8080/"
  }
  direct {
    exclude = ["registry.terraform.io/*/*"]
  }
}
```

**方式二：指定 Provider Source**

在 Terraform 配置中直接指定：

```hcl
terraform {
  required_providers {
    proxmox = {
      source  = "YOUR_REGISTRY_HOST:8080/telmate/proxmox"
      version = "~> 2.9"
    }
  }
}
```

### 镜像 API / Mirror API

#### 查询上游可用版本

```bash
# 查询 Proxmox Provider 可用版本
curl http://localhost:8080/api/v1/mirror/upstream/telmate/proxmox
```

#### 手动触发镜像

```bash
# 镜像指定版本
curl -X POST "http://localhost:8080/api/v1/mirror/telmate/proxmox?version=2.9.14&os=linux&arch=amd64"

# 镜像最新版本
curl -X POST "http://localhost:8080/api/v1/mirror/telmate/proxmox"
```

#### 上传 Provider

```bash
curl -X POST http://localhost:8080/api/v1/providers/upload \
  -F "namespace=myorg" \
  -F "name=myprovider" \
  -F "version=1.0.0" \
  -F "os=linux" \
  -F "arch=amd64" \
  -F "file=@terraform-provider-myprovider_1.0.0_linux_amd64.zip"
```

#### Terraform Registry Protocol

遵循标准 Terraform Registry Protocol v1：

```bash
# 服务发现
curl http://localhost:8080/.well-known/terraform.json

# 获取 Provider 版本列表
curl http://localhost:8080/v1/providers/telmate/proxmox/versions

# 获取下载信息
curl http://localhost:8080/v1/providers/telmate/proxmox/2.9.14/download/linux/amd64
```

### 常用 Provider 列表 / Popular Providers

| Provider | Namespace | Name | Description |
|----------|-----------|------|-------------|
| Proxmox VE | telmate | proxmox | Proxmox Virtual Environment |
| AWS | hashicorp | aws | Amazon Web Services |
| Azure | hashicorp | azurerm | Microsoft Azure |
| Google Cloud | hashicorp | google | Google Cloud Platform |
| Kubernetes | hashicorp | kubernetes | Kubernetes |
| Cloudflare | cloudflare | cloudflare | Cloudflare |

## 维护

### 备份

```bash
# 备份数据
./scripts/backup.sh

# 恢复数据
./scripts/restore.sh backup-2026-01-11.tar.gz
```

### 日志查看

```bash
# 查看实时日志
docker-compose logs -f

# 查看特定服务日志
docker-compose logs -f registry
```

### 更新

```bash
# 拉取最新镜像
docker-compose pull

# 重启服务
docker-compose up -d
```

## 安全建议

1. **启用 HTTPS** - 生产环境务必使用 HTTPS
2. **配置防火墙** - 限制访问来源
3. **定期备份** - 设置自动备份策略
4. **更新密钥** - 定期轮换 API Token
5. **审计日志** - 启用并定期检查审计日志

## 故障排查

### 无法启动服务

```bash
# 检查端口占用
lsof -i :8080

# 检查容器状态
docker-compose ps

# 查看错误日志
docker-compose logs
```

### Provider 下载失败

1. 检查网络连接
2. 验证 Provider 版本是否存在
3. 检查存储后端是否正常
4. 查看 Registry 日志

## 贡献

欢迎提交 Issue 和 Pull Request！

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

## 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情

## 联系方式

- 项目主页: [https://github.com/Veritas-Calculus/vc-terraform-registry](https://github.com/Veritas-Calculus/vc-terraform-registry)
- 问题反馈: [https://github.com/Veritas-Calculus/vc-terraform-registry/issues](https://github.com/Veritas-Calculus/vc-terraform-registry/issues)
- 文档: [https://docs.example.com](https://docs.example.com)

## 致谢

感谢所有为本项目做出贡献的开发者！

---

如果这个项目对你有帮助，请给我们一个 Star！
