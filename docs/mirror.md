# Provider mirror

Veritas Registry can obtain providers two ways.

## Automatic mirroring (recommended)

When Terraform requests a provider, the registry proxies the download from the
upstream registry (default `https://registry.terraform.io`) and caches it locally
for subsequent requests. Background caching from the unauthenticated mirror-protocol
routes is bounded (limited concurrency, deduplicated) so it cannot be abused to
exhaust disk or bandwidth.

## Manual upload

Upload a provider binary through the web UI or the API, for isolated networks,
custom providers, or pre-downloading for offline use.

```bash
curl -X POST http://localhost:8080/api/v1/providers/upload \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -F "namespace=myorg" \
  -F "name=myprovider" \
  -F "version=1.0.0" \
  -F "os=linux" \
  -F "arch=amd64" \
  -F "file=@terraform-provider-myprovider_1.0.0_linux_amd64.zip"
```

## Configure Terraform to use the mirror

Network mirror (recommended), in `~/.terraformrc`:

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

Or pin a provider source directly:

```hcl
terraform {
  required_providers {
    proxmox = {
      source  = "YOUR_REGISTRY_HOST:3443/telmate/proxmox"
      version = "~> 2.9"
    }
  }
}
```

## Mirror API

```bash
# Query available upstream versions (admin)
curl -H "Authorization: Bearer YOUR_TOKEN" \
  http://localhost:8080/api/v1/mirror/upstream/telmate/proxmox

# Mirror a specific version (admin)
curl -X POST -H "Authorization: Bearer YOUR_TOKEN" \
  "http://localhost:8080/api/v1/mirror/telmate/proxmox?version=2.9.14&os=linux&arch=amd64"

# List mirrored providers (public)
curl http://localhost:8080/api/v1/mirror/providers
```

The outbound proxy used for upstream fetches is configured server-side (in
settings); a per-request proxy cannot be supplied by clients.

## Popular providers

| Provider | Namespace | Name |
|----------|-----------|------|
| Proxmox VE | telmate | proxmox |
| AWS | hashicorp | aws |
| Azure | hashicorp | azurerm |
| Google Cloud | hashicorp | google |
| Kubernetes | hashicorp | kubernetes |
| Cloudflare | cloudflare | cloudflare |
