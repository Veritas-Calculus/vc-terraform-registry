# API and Terraform CLI

Base URL examples below use `http://localhost:8080` for direct API access. In a
deployed setup, call the API through the frontend origin over HTTPS.

## Authentication

Obtain a token, then send it as a Bearer token on write requests.

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"YOUR_PASSWORD"}'
```

Write and management endpoints require the `admin` role. Read endpoints are
public. The login endpoint is rate limited per client IP.

## Health check

```bash
curl http://localhost:8080/health
```

## Providers

```bash
# List providers
curl http://localhost:8080/api/v1/providers

# Get a provider
curl http://localhost:8080/api/v1/providers/{namespace}/{name}/{version}

# Search
curl "http://localhost:8080/api/v1/providers/search?q=aws"

# Create (admin)
curl -X POST http://localhost:8080/api/v1/providers \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"namespace":"myorg","name":"myprovider","version":"1.0.0","description":"My provider"}'
```

## Modules

```bash
curl http://localhost:8080/api/v1/modules
curl http://localhost:8080/api/v1/modules/{namespace}/{name}/{provider}/{version}
```

## Terraform Registry Protocol v1

```bash
# Service discovery
curl http://localhost:8080/.well-known/terraform.json

# Provider versions
curl http://localhost:8080/v1/providers/{namespace}/{name}/versions

# Download info
curl http://localhost:8080/v1/providers/{namespace}/{name}/{version}/download/{os}/{arch}
```

## Configure the Terraform CLI

Add the registry as a network mirror in `~/.terraformrc` (or `%APPDATA%/terraform.rc`):

```hcl
provider_installation {
  network_mirror {
    url = "https://YOUR_REGISTRY_HOST:3443/"
  }
  direct {
    exclude = ["registry.terraform.io/*/*"]
  }
}
```

If the registry requires a token, add credentials:

```hcl
credentials "YOUR_REGISTRY_HOST" {
  token = "your-api-token"
}
```

See [provider mirror](mirror.md) for the mirror API and popular providers.
