# Veritas Registry

English | [中文](README.zh-CN.md)

A private, self-hosted Terraform provider registry and mirror for internal and
air-gapped networks. Veritas Registry proxies and caches providers from upstream
registries, accepts manual uploads for disconnected environments, and speaks the
standard Terraform Registry and Provider Network Mirror protocols.

## Features

- Private, self-hosted registry for Terraform providers and modules
- Provider mirroring with automatic proxy/cache from upstream (e.g. `registry.terraform.io`)
- Manual provider upload for air-gapped environments
- Terraform Registry Protocol v1 and Provider Network Mirror Protocol
- Scheduled synchronization of upstream providers
- Token-based authentication and role-based access control
- Modern web UI (React, Tailwind)
- One-command deployment with Docker Compose; Kubernetes manifests included

## Quick start

Requirements: Docker 20.10+, Docker Compose 2.0+, ~2 GB RAM, ~10 GB disk.

```bash
git clone https://github.com/Veritas-Calculus/vc-terraform-registry.git
cd vc-terraform-registry

cp .env.example .env
# Edit .env and set a strong AUTH_SECRETKEY: openssl rand -base64 32
# The server refuses to start in release mode without one.

docker compose up -d
```

Then open `https://localhost:3443` (self-signed certificate; accept the warning).
The Terraform CLI requires HTTPS, so use the 3443 port.

On first boot, if you did not set `ADMIN_PASSWORD`, the backend generates a random
admin password and prints it once in the logs (`docker compose logs backend`).

Ports: frontend HTTPS `3443`, frontend HTTP `3000`, backend API `127.0.0.1:8080`.

## Documentation

- [Deployment](docs/deployment.md) - Docker Compose, development mode, Kubernetes, common commands
- [Configuration](docs/configuration.md) - environment variables and storage
- [API and Terraform CLI](docs/api.md) - REST API, registry protocol, CLI configuration
- [Provider mirror](docs/mirror.md) - mirroring, network mirror, popular providers
- [Security](docs/security.md) - security posture, audit status, hardening guidance
- [Packer integration (design)](docs/design/packer-integration.md) - research and design for Packer plugin distribution

## Tech stack

Backend: Go 1.25, Gin, GORM, SQLite, JWT.
Frontend: React 19, Vite 8, Tailwind CSS 4, React Router 8.

## Contributing

Issues and pull requests are welcome. Fork the repository, create a feature
branch, and open a pull request against `main`.

## License

MIT. See [LICENSE](LICENSE).
