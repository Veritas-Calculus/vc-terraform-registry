# Security

This document records the security posture of Veritas Registry: what the
automated scanners report, the findings of a manual audit, and the status of
each item. It is kept honest on purpose - items that are not yet fixed are
listed as such, with the reason and the remediation.

## Automated scanning

- CodeQL (Go and JavaScript, `security-and-quality`) runs on push, on pull
  requests, and weekly. At the time of writing it reports zero open alerts.
  Note that GitHub can silently disable scheduled scans on inactivity; check the
  Actions tab if results look stale.
- Dependabot watches Go modules, npm, Docker base images, and GitHub Actions.
  The one open advisory it reported, `react-router` GHSA-qwww-vcr4-c8h2
  (RSC-mode CSRF), is fixed by upgrading to `react-router` v8 (see below).
- `npm audit` and `govulncheck` are expected to be clean; run them in CI or
  locally before a release.

## Dependency fixes

| Item | Change |
|------|--------|
| `react-router` GHSA-qwww-vcr4-c8h2 (high) | Upgraded `react-router-dom` 7.x to `react-router` v8; `react-router-dom` is merged into `react-router` in v8, imports updated across all pages. `npm audit` clean. |
| `brace-expansion` (dev, high) | Resolved by `npm audit fix`. |

## Manual audit

A manual audit covered backend auth/authorization, backend file and network
handling, the frontend, container and Kubernetes posture, and the CI/CD supply
chain. Each finding below was verified against the source. The single critical
and all four high-severity findings are fixed and verified end to end; the most
impactful medium findings are fixed; the remainder are triaged with a status and
a remediation note.

### Fixed and verified

| Sev | Finding | Fix |
|-----|---------|-----|
| Critical | JWT signing key was effectively unconfigurable and defaulted to a public placeholder; `docker-compose.yml` also hardcoded a public key as a literal, and the documented `.env` override was silently ignored. Anyone could forge an admin token offline. | Root cause was a Viper misconfiguration: nested keys (`auth.secretkey`) were never bound to env vars (`AUTH_SECRETKEY`), so every environment override was dropped. Added `SetEnvKeyReplacer` plus explicit `BindEnv` for all keys. Added a fail-closed guard that refuses to start in release mode on any known placeholder key or a key shorter than 32 bytes. `docker-compose.yml` now uses `${AUTH_SECRETKEY:?}` so Compose aborts when it is unset. `k8s/secret.yaml` ships with no value and documents out-of-band creation. `.env.example` blanks the key. Regression tests cover the env-override path and the guard. |
| High | Default admin account `admin/admin123` was auto-created on first boot; no deployment path set `ADMIN_PASSWORD`. | If `ADMIN_PASSWORD` is unset, the server now generates a strong random password with `crypto/rand` and logs it once at first boot. The well-known fallback is gone. |
| High | A committed key in `k8s/secret.yaml` was applied by the documented deploy command. | Value removed; the manifest documents `kubectl create secret` from a generated value. |
| High | Unauthenticated mirror-protocol routes spawned one unbounded background upstream download per request (disk/bandwidth amplification DoS). | Background caching is now bounded by a semaphore (max 3 concurrent) and deduped by an in-flight set, launched via `tryStartBackgroundCache`. A burst degrades to "cached on a later request" instead of spawning unbounded goroutines. The routes stay unauthenticated so the Terraform network mirror keeps working. |
| Medium | The role claim was never enforced: `RequireRole` was applied to zero routes, so any authenticated principal was fully privileged. | Write/management routes are now behind an admin group (`AuthMiddleware` + `RequireRole("admin")`). `/auth/me` stays at authenticated-only. Verified: a valid non-admin token gets 403 on writes. |
| Medium | JWTs passed in the URL query string (SSE, export) were written verbatim to access logs by gin's default logger. | The engine now uses a custom logger that redacts sensitive query parameters (`token`) before logging. Verified: the token no longer appears in logs. |
| Medium | `GET /api/v1/settings` is public and returned `proxy_url` verbatim, disclosing outbound proxy credentials. | The public response now strips userinfo (`user:pass@`) from `proxy_url`; the admin path still returns the full value. Verified. |
| Medium | Wildcard CORS with `Access-Control-Allow-Credentials: true` (an invalid, risky combination). | CORS is now configurable via `CORS_ALLOWED_ORIGINS`. The API authenticates with a Bearer token, not cookies, so credentials are never emitted; a wildcard origin no longer carries credentials. |
| Medium | No rate limiting on `POST /api/v1/auth/login` - unlimited credential guessing and a cheap bcrypt CPU-exhaustion vector. | Added a per-IP fixed-window limiter (10/min). Verified: the 11th attempt returns 429. |
| Medium | SSRF: an attacker-chosen outbound proxy via the `proxy_url` query parameter. | The mirror stream handler no longer accepts a client-supplied proxy; it always uses the server-configured proxy. |
| Medium | Stored XSS: cached/uploaded provider files were served inline from the SPA origin with an attacker-influenced name. | The file-serving path now sets `Content-Type: application/octet-stream`, `X-Content-Type-Options: nosniff`, and `Content-Disposition: attachment`. |
| Medium | Both nginx configs sent no security headers. | Added `X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`, a Content-Security-Policy, and HSTS (on the TLS listener). |
| Medium | Backend was published in cleartext on `0.0.0.0:8080`, bypassing the TLS terminator. | Compose now binds the backend to `127.0.0.1:8080`; the frontend reaches it over the internal Docker network. |
| Medium | Containers ran as root with default capabilities and a writable root filesystem (Kubernetes). | The backend deployment now runs as non-root (uid 1000), read-only root filesystem, all capabilities dropped, `seccompProfile: RuntimeDefault`, and no auto-mounted service-account token. The frontend gets the subset compatible with the stock nginx image (see deferred). |
| Low | Gin ran in debug mode, dumping the route table and verbose logs. | The server sets the Gin mode from `SERVER_MODE` (release by default). |
| Low | JWT verification did not pin the signing algorithm and dereferenced `ExpiresAt` without a nil check. | `Verify` now pins HS256 via `WithValidMethods` and rejects a token with no `exp`. |
| Low | A negative `limit` query parameter disabled the SQL LIMIT and dumped whole tables. | `clampPagination` bounds page and limit. |
| Low | Kubernetes service-account token auto-mounted into the backend pod. | `automountServiceAccountToken: false` on both deployments. |
| Low | Dev compose pinned a Go toolchain older than `go.mod` requires, so `make dev-start` could not build. | Bumped the dev image to `golang:1.26-alpine`. |

### Deferred, with remediation

These are real but were not changed in this pass, either because a correct fix
needs its own design and tests, or because it is an operational/infra decision
rather than a code change. They are listed so nothing is lost.

| Sev | Finding | Why deferred and how to fix |
|-----|---------|-----------------------------|
| Medium | Downloaded provider binaries are cached with no verification against upstream `SHASUMS`; the registry publishes its own computed hash as authoritative. | This is a feature, not a one-line fix: verifying integrity means fetching and checking the upstream `SHA256SUMS` (and ideally its signature) before publishing. Track as a dedicated change with tests. |
| Medium | Concurrent downloads can race on a shared `.tmp` path, producing a corrupt archive with a hash the registry treats as verified. | Fix by writing to a unique temp file per download and renaming atomically. Needs a focused change in `internal/proxy` with a concurrency test. |
| Medium | `ImportProvider` accepts unbounded uploads and can amplify a zip bomb across duplicated manifest entries. | Add a decompression size/ratio cap and a per-request body limit. Needs a bounded reader and tests. |
| Medium | Arbitrary file read/delete via attacker-supplied `platforms[].file_path` in `CreateProvider` (mass assignment). | Now admin-gated, which drops the severity, but the real fix is a request DTO that never binds `file_path`, plus confining any path to the storage root. Needs a DTO refactor and tests. |
| Medium | The locally generated root CA private key is bind-mounted into the nginx container world-readable while users are told to trust that CA. | Operational: tighten file permissions on `certs/`, or provision certificates from a real CA / cert-manager. |
| Low | Frontend container runs as root; full non-root needs the unprivileged nginx image. | Switch the frontend image to `nginxinc/nginx-unprivileged` (listens on 8080) and update the port mappings, then enable `runAsNonRoot`, `readOnlyRootFilesystem`, and capability drop. |
| Low | GitHub Actions are not pinned to commit SHAs; `main` has no branch protection; the publish job runs on push with no gate. | Pin every `uses:` to a full commit SHA (Dependabot can then bump them), and enable branch protection / required checks on `main`. |
| Low | Published images disable provenance and SBOM. | These were disabled to work around an `unknown/unknown` architecture artifact; revisit once resolved and re-enable attestations. |
| Low | Deployments and manifests pull mutable `:latest` with `imagePullPolicy: Always`; base images are unpinned. | Pin images to a digest (`@sha256:...`) for reproducible, tamper-evident deploys. |
| Low | Mirror-protocol archive URLs are built from client-controlled `X-Forwarded-Host` / `X-Forwarded-Proto`. | Configure gin's trusted proxies and derive the external URL from configuration rather than untrusted headers. |
| Low | Login leaks whether a username exists via response timing (bcrypt runs only for existing users). | Run a dummy bcrypt comparison for unknown users to equalize timing. |
| Low | `GET /api/v1/sync/schedules` is public though the UI shows it only to signed-in users. | Move it into the authenticated group if the schedule inventory is considered sensitive. |

### Refuted

Nine reported items were investigated and refuted (for example, a claim that
the JWT key was still unconfigurable - already fixed here; and a claim that
`frontend/.gitignore` fails to ignore `.env` - it does). They are not tracked.

## Operational recommendations

1. Always set a strong, unique `AUTH_SECRETKEY` (`openssl rand -base64 32`).
2. Set `ADMIN_PASSWORD`, or capture the generated password from the first-boot
   logs and rotate it after first login.
3. Terminate TLS in front of the API; do not expose the backend port publicly.
4. Restrict `CORS_ALLOWED_ORIGINS` to known origins in production.
5. Keep dependencies current; let Dependabot and CodeQL run on schedule.
