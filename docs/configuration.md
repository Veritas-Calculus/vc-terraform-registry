# Configuration

Configuration is read from environment variables (and an optional `config.yaml`).
Environment variables take precedence.

## Environment variables

| Variable | Description | Default |
|----------|-------------|---------|
| `SERVER_PORT` | API listen port | `8080` |
| `SERVER_HOST` | API bind address | `0.0.0.0` |
| `SERVER_MODE` | `release` or `debug`. Release hides debug routes and enforces the key check below | `release` |
| `STORAGE_PATH` | Provider storage directory | `/data/registry` |
| `STORAGE_TYPE` | Storage backend | `local` |
| `DATABASE_URL` | Database connection string | `sqlite:///data/registry.db` |
| `AUTH_ENABLED` | Enable authentication | `true` |
| `AUTH_SECRETKEY` | JWT signing key. Required in release mode; must be a strong random value of at least 32 bytes | none |
| `ADMIN_PASSWORD` | Initial admin password. Required on first boot when `AUTH_ENABLED=true`; the server will not seed a well-known password or log a generated one | none |
| `CORS_ALLOWED_ORIGINS` | Comma-separated allowed browser origins, or `*` for any | `*` |
| `LOG_LEVEL` | Log level | `info` |

### AUTH_SECRETKEY

In release mode the server refuses to start when `AUTH_SECRETKEY` is missing, is a
known placeholder, or is shorter than 32 bytes. This prevents deployments that
sign tokens with a public or weak key. Generate one with:

```bash
openssl rand -base64 32
```

### CORS

The API authenticates with a Bearer token in the `Authorization` header, not with
cookies, so it never sends `Access-Control-Allow-Credentials`. When the frontend
and API are served from the same origin (the default Docker setup, where nginx
proxies the API), no cross-origin access is required. Set `CORS_ALLOWED_ORIGINS`
to an explicit list when a different origin must call the API from a browser.

## Storage

The default backend is the local filesystem, suitable for single-node
deployments. Provider archives are stored under `STORAGE_PATH`; the SQLite
database holds metadata. Back up both together (see [deployment](deployment.md)).
