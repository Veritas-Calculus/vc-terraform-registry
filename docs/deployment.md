# Deployment

## Docker Compose (recommended)

Ensure Docker is running, then:

```bash
cp .env.example .env
# Set a strong AUTH_SECRETKEY (openssl rand -base64 32). Optionally set ADMIN_PASSWORD.

# Using make
make start

# Or directly
docker compose up -d
```

After startup:

- Frontend UI (HTTPS): `https://localhost:3443`
- Frontend UI (HTTP): `http://localhost:3000`
- Backend API: `http://127.0.0.1:8080`
- Health check: `http://127.0.0.1:8080/health`

The Terraform CLI must use HTTPS (port 3443). The backend is bound to localhost;
the frontend proxies to it over the internal Docker network.

## Development mode (hot reload)

```bash
make dev-start
# or
docker compose -f docker-compose.dev.yml up
```

Development ports:

- Frontend: `http://localhost:5173` (Vite dev server)
- Backend: `http://127.0.0.1:8080`

Development mode runs with authentication disabled for convenience. Do not use it
as a production configuration.

## Local development (without Docker)

Backend:

```bash
cd backend
go mod download
AUTH_SECRETKEY=$(openssl rand -base64 32) SERVER_MODE=debug go run cmd/server/main.go
```

Frontend:

```bash
cd frontend
npm install
npm run dev
```

## Kubernetes

Manifests are under `k8s/`. Create the secret out of band (never commit it):

```bash
kubectl create namespace terraform-registry
kubectl create secret generic terraform-registry-secret \
  --namespace terraform-registry \
  --from-literal=AUTH_SECRETKEY="$(openssl rand -base64 32)" \
  --from-literal=ADMIN_PASSWORD="$(openssl rand -base64 24)"

kubectl apply -k k8s/
```

The backend deployment runs as non-root with a read-only root filesystem, all
capabilities dropped, and no auto-mounted service-account token. See
[security](security.md) for the current hardening status and the remaining
follow-ups (for example, running the frontend on the unprivileged nginx image).

## Common commands

```bash
make logs        # or: docker compose logs -f
make stop        # or: docker compose down
make restart
make clean       # remove containers and volumes
```

## Backup and restore

Provider files and the SQLite database live under the storage volume. Back up the
`registry-data` volume (or the mounted `/data` directory) on a schedule. To
restore, stop the services, replace the volume contents, and start again.

## Updating

```bash
docker compose pull
docker compose up -d
```

## Troubleshooting

Service will not start:

```bash
lsof -i :8080          # check port conflicts
docker compose ps      # container status
docker compose logs    # error logs
```

Provider download fails: verify network connectivity to the upstream registry,
confirm the requested provider/version exists, check the storage backend, and
review the backend logs.
