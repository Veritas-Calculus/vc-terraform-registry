# Veritas Registry

[English](README.md) | 中文

面向内网与离线环境的私有化、自托管 Terraform Provider Registry 与镜像服务。
Veritas Registry 可代理并缓存上游 Registry 的 Provider，支持隔离网络下的手动上传，
并实现标准的 Terraform Registry 协议与 Provider Network Mirror 协议。

## 特性

- 私有化、自托管的 Terraform Provider 与 Module 仓库
- Provider 镜像：自动从上游（如 `registry.terraform.io`）代理并缓存
- 手动上传 Provider，适用于隔离网络环境
- 实现 Terraform Registry Protocol v1 与 Provider Network Mirror Protocol
- 上游 Provider 的定时同步
- 基于 Token 的认证与基于角色的访问控制
- 现代化 Web 界面（React、Tailwind）
- Docker Compose 一键部署；同时提供 Kubernetes 清单

## 快速开始

环境要求：Docker 20.10+、Docker Compose 2.0+、约 2 GB 内存、约 10 GB 磁盘。

```bash
git clone https://github.com/Veritas-Calculus/vc-terraform-registry.git
cd vc-terraform-registry

cp .env.example .env
# 编辑 .env，设置强随机 AUTH_SECRETKEY：openssl rand -base64 32
# release 模式下缺少该密钥服务将拒绝启动。

docker compose up -d
```

随后访问 `https://localhost:3443`（自签名证书，需要在浏览器中接受安全警告）。
Terraform CLI 要求使用 HTTPS，请使用 3443 端口。

首次启动时，若未设置 `ADMIN_PASSWORD`，后端会生成随机管理员密码并在日志中打印一次
（`docker compose logs backend`）。

端口：前端 HTTPS `3443`、前端 HTTP `3000`、后端 API `127.0.0.1:8080`。

## 文档

- [部署](docs/deployment.md) - Docker Compose、开发模式、Kubernetes、常用命令
- [配置](docs/configuration.md) - 环境变量与存储
- [API 与 Terraform CLI](docs/api.md) - REST API、Registry 协议、CLI 配置
- [Provider 镜像](docs/mirror.md) - 镜像、Network Mirror、常用 Provider
- [安全](docs/security.md) - 安全状况、审计状态与加固建议
- [Packer 接入（设计）](docs/design/packer-integration.md) - Packer 插件分发的调研与设计

## 技术栈

后端：Go 1.25、Gin、GORM、SQLite、JWT。
前端：React 19、Vite 8、Tailwind CSS 4、React Router 8。

## 贡献

欢迎提交 Issue 和 Pull Request。Fork 仓库、创建特性分支，并向 `main` 发起 Pull Request。

## 许可证

MIT，详见 [LICENSE](LICENSE)。
