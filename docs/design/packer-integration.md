# Packer Integration and Intranet Mirror Acceleration

Status: Draft for review
Author: Platform / Registry
Date: 2026-08-01
Target codebase: `vc-terraform-registry` (branch `fix/exhaustive-deps`, worktree `registry-hardening`)
Packer version of record: **v1.16.0**, released 2026-07-24 (verified via `https://api.github.com/repos/hashicorp/packer/releases/latest` returning `"tag_name": "v1.16.0"`, `"published_at": "2026-07-24T09:26:22Z"`)

---

## 1. Problem statement

### 1.1 What was asked

"Add Packer support to the registry, for intranet mirror acceleration."

### 1.2 The request is ambiguous, and the two readings are unrelated systems

"Packer intranet mirror acceleration" resolves to two entirely different problems. They share no protocol, no artifact type, and no upstream host. Conflating them is the single most common way this project goes wrong, so we name both up front.

| | Reading A: **plugin distribution** | Reading B: **machine image distribution** |
|---|---|---|
| Artifact | `packer-plugin-*` binaries (zipped, per OS/arch) | AMIs, vSphere templates, qcow2, Docker images, Vagrant boxes |
| Typical size | 10-60 MB | 500 MB - 20 GB |
| Consumed by | `packer init`, `packer build` at plugin-load time | Terraform, vSphere, ECS, VM provisioning |
| Upstream | `releases.hashicorp.com` and `github.com` | None; you produce them |
| Failure mode today | `packer init` fails or is slow behind a restrictive egress policy; GitHub API rate limits | No internal catalog; teams pass AMI IDs around in wiki pages |
| Analogue in this repo | Terraform provider mirror (already built) | Nothing; closest analogue is HCP Packer, which stores metadata only |

**This document solves Reading A (plugin distribution).** Reading B is covered at the level of options and an architecture sketch in [section 7](#7-machine-image-distribution-reading-b).

### 1.3 Why Reading A is the right scope

Four reasons, in order of weight:

1. **It is the direct analogue of what this repository already is.** The registry is a Terraform provider mirror. Packer plugins are the same artifact shape: `{namespace}/{name}/{version}/{os}/{arch}` zips with SHA256 checksums. The existing `proxy`, `scheduler`, export/import and SSE progress machinery transfer almost directly. Reading B would require blob storage sized for multi-gigabyte images and cloud-provider-specific publishing logic, which is a different product.

2. **It is the reading where an intranet mirror actually accelerates anything.** Machine images are already stored in the platform that consumes them (AMIs live in AWS, templates live in vCenter). There is no intermediate fetch to accelerate. Plugins, by contrast, are fetched from the public internet on every cold `packer init`.

3. **The pain is real and unaddressed by anyone.** There is no Packer package type in JFrog Artifactory (`https://docs.jfrog.com/artifactory/docs/supported-package-types`), Sonatype Nexus (`https://help.sonatype.com/en/formats.html`), or Cloudsmith (`https://docs.cloudsmith.com/formats`) — all three support Terraform, and two support Vagrant. No open-source Packer plugin mirror was found in a systematic GitHub search (see [section 9](#9-open-questions)). The canonical upstream feature request, `https://github.com/hashicorp/packer/issues/11164`, has been open since 2021-07-22 with 51 thumbs-up reactions.

4. **Reading B has a credible non-registry answer** (per-platform native distribution plus a thin metadata catalog), which we describe in section 7 rather than building.

### 1.4 What "success" means

A Packer user on the corporate network can build a template that requires third-party plugins:

- without their machine reaching `github.com` or `releases.hashicorp.com`;
- with byte-for-byte integrity guarantees at least as strong as upstream Packer's;
- with a version-pinned, reproducible result;
- with a single command in CI, and no per-developer manual steps.

Explicit non-goal: making `packer init` transparently resolve plugins from our registry over the network. As section 2 establishes, **Packer cannot do this and no amount of registry-side work changes that.**

---

## 2. What Packer supports today

### 2.1 Lead with the constraint: Packer has no mirror protocol

There is no Packer equivalent of Terraform's Provider Network Mirror Protocol. There is no `provider_installation` block, no `network_mirror`, no `filesystem_mirror`, and no environment variable or config-file key that redirects plugin installation to an alternate host.

The official documentation states it directly:

> "Packer only supports using `github.com` as the hostname when you install plugins using the `packer init` command. You can point to non-GitHub addresses, such as an internal proxy or plugin binary store, but Packer only pins the required version if you install the plugin using the `packer plugins install --path` command."
> — `https://developer.hashicorp.com/packer/docs/templates/hcl_templates/blocks/packer`

And on the install command reference:

> "To remotely download from a source, the plugin must be hosted on GitHub."
> — `https://developer.hashicorp.com/packer/docs/commands/plugins/install`

In source, `packer init` builds a hardcoded slice of exactly two getters, in a fixed order, with a comment stating the ordering is deliberate:

- `release.Getter{Name: "releases.hashicorp.com"}` — base URL is the compile-time constant `officialReleaseURL = "https://releases.hashicorp.com/"` (`https://github.com/hashicorp/packer/blob/v1.16.0/packer/plugin-getter/release/getter.go`)
- `github.Getter{Name: "github.com"}` — `defaultHostname = "github.com"` (`https://github.com/hashicorp/packer/blob/v1.16.0/packer/plugin-getter/github/getter.go`)

Both are in `https://github.com/hashicorp/packer/blob/v1.16.0/command/init.go`. Neither host is configurable. `PACKER_GITHUB_API_TOKEN` supplies credentials only, never a host.

Critically, the `releases.hashicorp.com` getter calls `github.NewGithubPlugin()` to parse the source address, so **both** getters inherit the assertion that the source address's first component is literally `github.com`. A source like `registry.example.internal/myorg/docker` is rejected before any network I/O with:

```
"registry.example.internal/myorg/docker" doesn't appear to be a valid "github.com"
source address; check source and try again.
```

**Consequence for this design: there is no protocol to implement.** Any "Packer registry" is, by construction, a distribution tool that populates the client's plugin directory — not a service `packer init` talks to.

### 2.2 What *is* permitted: arbitrary hostnames for locally installed plugins

This is the seam the design exploits, and it is documented and supported, not a hack.

The HCL source-address parser (`hcl2template/addrs/plugin.go`, `ParsePluginSourceString`) does **not** validate the hostname. It only requires: no leading or trailing `/`, at least two `/` characters, no URL scheme, no query string, no fragment, a valid final component, and at most 16 components. The `github.com` restriction lives exclusively in the getter layer.

This was made explicit upstream in Packer 1.11.0 (`https://github.com/hashicorp/packer/blob/v1.16.0/CHANGELOG.md`):

> "core: Relax Packer source address URIs within the `required_plugins` block to support the installation of local plugin binary using a custom or internal source address (e.g. mycompany.com/plugins/happyorg/happycloud). Remote installation using `packer init` or `packer plugins install` does not support non-GitHub source URIs. Users using alternative hosts must install plugins manually using `packer plugins install --path`. [GH-12911], [GH-12962]"

So: **`source = "registry.example.internal/myorg/proxmox"` is legal, loads correctly, and pins its version — provided the binary was placed there by `packer plugins install --path`.**

Two caveats, both load-bearing:

- `packer init` on a pre-seeded private-host source exits 0 and makes **zero** network calls, because `init` short-circuits on already-satisfied requirements. (Mechanism visible in `command/init.go`: for each requirement it calls `ListInstallations` and `continue`s when installations exist and neither `-force` nor `-upgrade` was passed.)
- `packer init -upgrade` and `packer init -force` **fail** on a private-host source, because they bypass the short-circuit and reach the getter's hostname assertion. Any CI pipeline that runs `packer init -upgrade` is incompatible with private source addresses. This is a hard constraint we must document to users.

Marked **UNVERIFIED**: I did not personally execute `packer init` / `packer init -upgrade` against a pre-seeded private-host source. The short-circuit behaviour is read off `command/init.go` control flow and corroborated by prior empirical testing recorded in the research input (against Packer v1.15.4, not v1.16.0). This must be re-confirmed on v1.16.0 before implementation — see [section 9](#9-open-questions).

### 2.3 Plugin directory layout and filename conventions

This is the exact contract our seeder must satisfy. Source: `https://developer.hashicorp.com/packer/docs/plugins/creation/plugin-load-spec` and `https://developer.hashicorp.com/packer/docs/plugins/install`.

**Plugin root directory**

| Platform | Default root | Override |
|---|---|---|
| Unix/Linux/macOS | `$HOME/.config/packer/plugins` | `PACKER_PLUGIN_PATH` (single path only) |
| Windows | `%APPDATA%\packer.d\plugins` | `PACKER_PLUGIN_PATH` |

The default is derived by the SDK: `PACKER_CONFIG_DIR` → `$PACKER_CONFIG_DIR/.packer.d`; else a pre-existing `$HOME/.packer.d` wins for legacy reasons; else `$XDG_CONFIG_HOME/packer`; else `$HOME/.config/packer`. `PACKER_PLUGIN_PATH` overrides the whole root and **rejects multiple paths** ("Multiple paths are no longer supported for PACKER_PLUGIN_PATH", `packer/plugin_folders.go`).

**Directory hierarchy** — mirrors the source address, component by component:

```
<plugin-root-dir>/
└── <hostname>/
    └── <namespace>/
        └── <name>/
            ├── packer-plugin-<name>_v<version>_x<api_major>.<api_minor>_<os>_<arch>[.exe]
            └── packer-plugin-<name>_v<version>_x<api_major>.<api_minor>_<os>_<arch>[.exe]_SHA256SUM
```

Concrete, from the official docs:

```
/Users/dev/.packer.d/plugins
└── github.com
    └── hashicorp
        └── amazon
            ├── packer-plugin-amazon_v1.1.0_x5.0_darwin_arm64
            └── packer-plugin-amazon_v1.1.0_x5.0_darwin_arm64_SHA256SUM
```

**Filename grammar:** `packer-plugin-<name>_<version>_<api_version>_<os>_<arch>[.exe]` where `<version>` is `v<major>.<minor>.<patch>[-<prerelease>]` and `<api_version>` is `x<api_major>.<api_minor>` (currently `x5.0`).

**Checksum sidecar:** filename is the binary filename plus the literal suffix `_SHA256SUM` (singular `SUM`). Its contents are, verbatim from the docs, "the raw hexdigest of the sha256 sum from the contents of the plugin binary" — no filename column, no trailing newline. Note this hashes the **extracted binary**, not the zip.

The sidecar is **mandatory**. Without it Packer refuses to consider the binary; `ListInstallations` logs `No checksum found for %q ignoring possibly unsafe binary` and skips it, and `packer validate` then reports the plugin as missing.

### 2.4 The two upstream hosts use different zip filename conventions

This is a concrete implementation trap. I verified both live on 2026-08-01:

```
$ curl -s https://releases.hashicorp.com/packer-plugin-docker/1.1.4/packer-plugin-docker_1.1.4_SHA256SUMS | head -2
63f0fd69dd6b12fa5bdb13a46c76866e8d5045cf12f3362b98d11604d0da613b  packer-plugin-docker_1.1.4_darwin_amd64.zip
232798d923528285f367a8a13488cdb02caf55f7f6f987b3e5bf8da587c7112e  packer-plugin-docker_1.1.4_darwin_arm64.zip

$ curl -s https://api.github.com/repos/hashicorp/packer-plugin-proxmox/releases/latest | grep '"name": "packer-plugin'
  "name": "packer-plugin-proxmox_v1.2.4_SHA256SUMS",
  "name": "packer-plugin-proxmox_v1.2.4_SHA256SUMS.sig",
  "name": "packer-plugin-proxmox_v1.2.4_x5.0_darwin_amd64.zip",
  "name": "packer-plugin-proxmox_v1.2.4_x5.0_linux_amd64.zip",
```

| | `releases.hashicorp.com` | `github.com` releases |
|---|---|---|
| Version in zip name | bare (`1.1.4`) | `v`-prefixed (`v1.2.4`) |
| API version in zip name | **absent** | present (`x5.0`) |
| Zip name | `packer-plugin-docker_1.1.4_darwin_amd64.zip` | `packer-plugin-proxmox_v1.2.4_x5.0_darwin_amd64.zip` |
| Checksums file | `<plugin>_<ver>_SHA256SUMS` | `<plugin>_v<ver>_SHA256SUMS` |
| API version discovered via | `_manifest.json` → `metadata.protocol_version` | encoded in the zip filename |
| GPG signature published | yes (`.sig`, `.72D7468F.sig`) | yes (`.sig`, per-author key) |
| GPG signature **verified by Packer** | **no** | **no** |
| Binary name inside the zip | `packer-plugin-docker_v1.1.4_x5.0_darwin_arm64` | same convention |

The `manifest.json` shape, verified live:

```
$ curl -s https://releases.hashicorp.com/packer-plugin-docker/1.1.4/packer-plugin-docker_1.1.4_manifest.json
{
  "version": "1",
  "metadata": {
    "protocol_version": "5.0"
  }
}
```

**Implication:** our fetcher needs two code paths, and for `releases.hashicorp.com` plugins it must fetch `_manifest.json` to learn the API version, because the zip filename does not carry it — but the extracted binary's filename does.

### 2.5 Packer performs no signature verification

Trust in the upstream chain reduces entirely to TLS plus the integrity of the `SHA256SUMS` file served over that same channel. Packer never downloads or checks the GPG `.sig` files that both hosts publish. Packer v1.16.0 explicitly dropped `x/crypto/openpgp` from the dependency tree ("security: drop x/crypto/openpgp by upgrading go-github v33 → v75", `https://github.com/hashicorp/packer/blob/v1.16.0/CHANGELOG.md`).

This is an opportunity, not just a caveat: **a registry that verifies the detached GPG signature at ingest offers a strictly stronger guarantee than upstream Packer does.** See section 5.7.

Relatedly, Packer v1.16.0 fixed a path-traversal vulnerability in the GitHub getter, where the filename read from the remote `SHA256SUMS` asset was passed unsanitised into `filepath.Join`:

> "plugin/getter: prevent path traversal vulnerability in GitHub plugin getter filename handling. [GH-13680]"
> — `https://github.com/hashicorp/packer/pull/13680`

Our ingest path reads filenames from exactly the same attacker-influenced documents. This repository already has the right primitive (`proxy.SanitizeFilename`, which applies `filepath.Base` and rejects null bytes); the design mandates its use on every remote-supplied filename.

### 2.6 Environment variables that matter

From `https://developer.hashicorp.com/packer/docs/configure`, cross-checked against the v1.16.0 source:

| Variable | Effect | Relevant to us |
|---|---|---|
| `PACKER_PLUGIN_PATH` | Sets the plugin root directly. Single path only. | Yes — the seeder targets it |
| `PACKER_CONFIG_DIR` | Base for the default plugin root | Yes — alternative targeting |
| `PACKER_GITHUB_API_TOKEN` | Auth for `api.github.com` only; scoped by `req.Host` so it is not leaked to the S3 redirect | Yes — registry-side ingest, to avoid rate limits |
| `HTTP_PROXY` / `HTTPS_PROXY` / `NO_PROXY` | Honoured by both getters (Go `http.DefaultTransport`) since **v1.14.1** | Yes — Option C |
| `PACKER_LOG` / `PACKER_LOG_PATH` | Reveals getter request URLs at `[DEBUG]`/`[TRACE]` | Diagnostics only |

There is **no** `PACKER_PLUGIN_MIRROR`, `PACKER_REGISTRY_URL`, or equivalent. Any such variable named elsewhere is invented.

Two stale-documentation warnings: `https://developer.hashicorp.com/packer/docs/configure` lists `PACKER_PLUGIN_MIN_PORT` / `PACKER_PLUGIN_MAX_PORT` as user-settable, but v1.16.0's `main.go` hardcodes 10000/25000 and only *exports* those names into the plugin subprocess environment. And `PACKER_GETTER_READ_TIMEOUT`, despite the name, governs ISO downloads inside plugins (it lives in `packer-plugin-sdk`'s `multistep/commonsteps/step_download.go`), not the plugin getter.

### 2.7 Proxy support history

Packer 1.14.0 (2025-07-30) moved eight HashiCorp-maintained plugins to `releases.hashicorp.com` — amazon, ansible, azure, docker, googlecompute, qemu, vagrant, virtualbox (`https://developer.hashicorp.com/packer/docs/plugins/install`). That release shipped the release getter with a custom transport carrying `TLSClientConfig: &tls.Config{InsecureSkipVerify: true}`, which both **disabled TLS certificate verification for plugin downloads** and broke proxy support (a custom `Transport` with no `Proxy` field set does not consult the environment).

Packer 1.14.1 (2025-08-05) replaced it with a bare `&http.Client{}` (`https://github.com/hashicorp/packer/pull/13439`), restoring both certificate verification and `HTTP_PROXY` support:

> "The releases binary from official site releases.hashicorp.com was not getting installed from a proxy server, which was addressed in this PR. [GH-13439]"

**Consequence:** proxy-based approaches require Packer >= 1.14.1, and Packer 1.14.0 exactly must be treated as a version to block.

---

## 3. Options

Four options, evaluated. Option A is the do-nothing-in-the-registry baseline.

### Option A (baseline): Publish a seeding script; change nothing in the registry

**Mechanism.** Ship a documented shell/Go script in this repo. The script reads a template's `required_plugins`, downloads zips from upstream (or from a manually curated internal file share), extracts the binary, and calls `packer plugins install --path`.

**Packer client configuration.** Template keeps canonical addresses:

```hcl
packer {
  required_plugins {
    proxmox = {
      version = ">= 1.2.0"
      source  = "github.com/hashicorp/proxmox"
    }
  }
}
```

Then, before `packer build`:

```bash
export PACKER_PLUGIN_PATH="$HOME/.config/packer/plugins"
./scripts/packer-seed.sh   # downloads + packer plugins install --path
packer init .              # exits 0, no network
```

**Registry implements.** Nothing. Possibly hosts the script and a tarball of pre-downloaded zips as a static file.

**Limitations.**
- The download step still needs internet access from *somewhere*. If that somewhere is each developer's laptop, nothing is accelerated and the egress problem is not solved.
- No caching, no dedup, no shared state. Every CI runner re-downloads from upstream, hitting GitHub's unauthenticated 60-requests/hour rate limit. This is precisely the complaint in `https://github.com/hashicorp/packer/issues/12657`.
- No air-gap story: a script cannot cross an air gap; a curated artifact bundle can.
- No audit trail of which plugin versions are approved or in use.
- Version-constraint resolution still has to be implemented somewhere, so most of the hard logic exists either way.

**Verdict.** Correct as a *fallback* and as the client-side half of any real solution, but it does not solve the stated problem (acceleration, egress control) on its own.

### Option B (recommended): Registry as plugin artifact store, plus a seeder client

**Mechanism.** The registry becomes the internal source of truth for plugin binaries. It ingests from `releases.hashicorp.com` and `github.com` (on the registry host, which is the only machine needing egress), verifies checksums, stores the zips, and serves them over a small documented HTTP API of our own design. A thin client (`vcreg-packer`) resolves version constraints against that API, downloads, and shells out to `packer plugins install --path`.

**Packer client configuration.** Two supported shapes:

*Shape 1 — private source address (recommended; strongest pinning, zero egress):*
```hcl
packer {
  required_plugins {
    proxmox = {
      version = ">= 1.2.0"
      source  = "registry.example.internal/hashicorp/proxmox"
    }
  }
}
```

*Shape 2 — canonical GitHub address (for templates shared with the outside world):*
```hcl
source = "github.com/hashicorp/proxmox"
```
Both work; shape 2 keeps the template portable but means a developer with internet access may silently resolve from upstream instead of the registry.

Client-side:
```bash
export VCREG_URL="https://registry.example.internal"
vcreg-packer seed .        # resolves constraints, downloads, installs
packer init .              # exits 0, no network
```

**Registry implements.**
- Two new tables (`packer_plugins`, `packer_plugin_binaries`).
- An upstream fetcher for both hosts, reusing the existing proxy/SOCKS5 HTTP client.
- Five public read routes and six authenticated management routes.
- A new storage subtree, `{storagePath}/packer.plugins/...`.
- Scheduler integration for periodic refresh.
- Export/import for air-gap transfer, cloning the existing provider bundle format.
- Frontend pages for browsing, mirroring and setup instructions.
- A seeder binary (small, static, cross-compiled).

**Limitations.**
- `packer init -upgrade` / `-force` fail on shape-1 source addresses. Documented; CI must use plain `packer init`.
- We must reimplement Packer's version-constraint semantics (`=`, `!=`, `>`, `>=`, `<`, `<=`, `~>`) in the seeder. `~> 0.9` means `>= 0.9, < 1.0`; prerelease versions match only exact constraints (`https://developer.hashicorp.com/packer/docs/templates/hcl_templates/blocks/packer`). This is the largest correctness risk in the project.
- Requires distributing one extra binary to clients. Mitigated by making it a single static Go binary and by providing a pure-shell fallback for the simple pinned-version case.
- Does not make `packer init` fetch over the network from us. Nothing can.

### Option C: Transparent caching proxy emulating both upstream hosts

**Mechanism.** The registry serves the exact URL shapes of `releases.hashicorp.com` and the GitHub release-download endpoints. Clients set `HTTPS_PROXY` to a TLS-intercepting proxy in front of the registry (or the hostnames are overridden via corporate DNS), with the corporate CA installed in the system trust store. `packer init` runs unmodified against canonical `github.com/...` addresses.

**Packer client configuration.**
```bash
export HTTPS_PROXY="http://packer-proxy.example.internal:3128"
export HTTP_PROXY="http://packer-proxy.example.internal:3128"
export NO_PROXY="localhost,127.0.0.1,.example.internal"
# plus: corporate root CA in the system trust store
packer init .
```

**Registry implements.** The four `releases.hashicorp.com` endpoint shapes:

| Purpose | Path |
|---|---|
| version index | `/{plugin-type}/index.json` |
| checksums | `/{plugin-type}/{version}/{plugin-type}_{version}_SHA256SUMS` |
| API version | `/{plugin-type}/{version}/{plugin-type}_{version}_manifest.json` |
| archive | `/{plugin-type}/{version}/{zip filename from SHA256SUMS}` |

Plus GitHub emulation: `api.github.com/repos/{ns}/packer-plugin-{name}/git/matching-refs/tags` and `github.com/{ns}/packer-plugin-{name}/releases/download/{vVersion}/...`.

Usefully, Packer decodes `index.json` into a minimal struct — only the `versions` map keys are consumed; the `builds` array and its `url` fields are parsed away — so the emulated index can be small.

**Limitations.** These are severe for a project whose branch is named `registry-hardening`:
- Requires **organisation-wide TLS interception** of `github.com` and `api.github.com`, and a corporate root CA on every build machine. This is a large ops programme and a meaningful security-posture change: it puts the registry in a position to MITM all GitHub traffic, not just plugin downloads.
- `NO_PROXY` granularity is per-host, not per-path. Proxying `github.com` for Packer proxies it for `git`, `go mod`, `npm`, and everything else on that machine.
- Requires Packer >= 1.14.1 (see 2.7). Packer 1.14.0 will not proxy at all.
- Emulating the GitHub API surface is an open-ended commitment; it drifts.
- Marked **UNVERIFIED**: end-to-end success through a real MITM proxy including the `github.com` → `objects.githubusercontent.com` redirect chain was not tested. The proxy env vars are demonstrably honoured (a bogus proxy produces `proxyconnect tcp: ... connection refused`), but that only proves the client dials the proxy, not that the full download completes. GH-13439's history suggests this path has been fragile.

**Verdict.** Genuinely attractive because templates stay canonical and `packer init` is untouched. Rejected as the primary approach because the ops and security cost is borne by the whole organisation, and because it does not solve air-gap (a MITM proxy still needs upstream egress). Retained as a **Phase 3 optional add-on** — the storage and metadata built for Option B can back these URL shapes cheaply, for sites that already run an intercepting proxy.

### Option D: Contribute a pluggable getter upstream

**Mechanism.** Implement a third `plugingetter.Getter` in `hashicorp/packer` that reads a host from config or an environment variable, and get it merged.

**Precedent.** Maintainer `azr` explicitly invited this in 2021 on `https://github.com/hashicorp/packer/issues/11164`: "Packer is meant to support other plugin getters than GitHub! We just did not have the time to do it due to other priorities... we would be more than happy to review PRs on this!" The `Getter` interface's own doc comment anticipates external sources.

**Counter-evidence.** A community PR attempting the adjacent parsing work, `https://github.com/hashicorp/packer/pull/12863` ("Chore: improve plugin source parsing", proposing `source = "artifactory.mycompany.com/my-remote-repo/hashicorp/amazon"`), was **closed without merging** in June 2024; its author noted that a custom-domain getter "would require a huge PR." No maintainer has responded on #11164 since 2024-10-09. The 2021 "definitely on our roadmap" statement is now five years stale.

**Limitations.** Unbounded timeline, no control over acceptance, and even on success it only helps users who upgrade to the release that contains it. Cannot be the plan.

**Verdict.** Worth doing **in parallel** as a low-cost, high-leverage side bet (see section 8), but it cannot be on the critical path.

### Comparison

| Criterion | A: script only | **B: artifact store + seeder** | C: MITM proxy | D: upstream PR |
|---|---|---|---|---|
| Solves egress control | No | Yes | Partly (proxy still needs egress) | Yes, eventually |
| Solves air gap | No | Yes (export/import bundle) | No | No |
| Accelerates repeat fetches | No | Yes | Yes | Yes |
| `packer init` unchanged | Yes | Yes (plain init; not `-upgrade`) | Yes (incl. `-upgrade`) | Yes |
| Template stays canonical | Yes | Optional (shape 2) | Yes | Yes |
| Client-side install needed | Script | One static binary | Root CA + proxy env | None |
| Org-wide TLS interception | No | No | **Yes** | No |
| Reuses this codebase | No | Heavily | Partly | No |
| Registry effort (eng-days) | ~2 | ~20 | ~12 on top of B | Unbounded |
| Delivery certainty | High | High | Medium | Very low |

---

## 4. Recommendation

**Adopt Option B.** Build the registry into a Packer plugin artifact store, and ship a small seeder client that drives `packer plugins install --path`.

Justification against each alternative:

**Against A.** Option A does not solve the problem as stated. "Intranet mirror acceleration" requires something on the intranet holding the bytes; a script holds nothing. A also leaves each CI runner talking to `api.github.com`, which is the actual reported failure mode (rate limiting). That said, A is not discarded — the seeder in B *is* A's script, pointed at our registry instead of the internet, and B must retain a documented "point the seeder at upstream directly" fallback for developers working from home.

**Against C.** C is the only option that preserves `packer init -upgrade`, which is a real advantage. It is rejected because the cost is organisation-wide TLS interception of GitHub, imposed on every build machine, to solve a problem confined to Packer plugins. For a repository whose current work stream is security hardening, standing up an infrastructure-wide MITM is the wrong trade. C also fails the air-gap requirement entirely, and depends on a proxy code path with a documented history of breakage (GH-13439). Crucially, C is not foreclosed: the metadata and blobs Option B stores are exactly what C's endpoint emulation would serve, so C remains a cheap Phase 3 for sites that already operate an intercepting proxy.

**Against D.** D is the only option that removes the constraint rather than working around it, and it should be attempted — but a closed PR in the adjacent area and 22 months of maintainer silence on the tracking issue make it un-plannable. Treat as a side bet.

**The decisive argument for B** is fit with what already exists. The registry already models `{namespace}/{name}/{version}` artifacts with per-`{os}/{arch}` binaries and SHA256 sums (`backend/internal/models/provider.go`). It already has a path-traversal-hardened storage-path builder (`proxy.BuildSafeProviderPath`), a proxy-aware HTTP client with SOCKS5 support (`proxy.ProxyService.updateHTTPClient`), an SSE mirror-with-progress flow, a cron scheduler, and a zip-based export/import bundle format for air-gapped transfer (`MirrorHandler.ExportProvider` / `ImportProvider`). Packer plugins are close enough to Terraform providers that roughly 60 percent of the work is already written; they differ in exactly three ways (an extra hostname path segment, an extra API-version dimension, and two divergent upstream filename conventions), all of which are additive.

**Accepted costs, stated plainly:**
1. `packer init -upgrade` will not work with private source addresses. Users must run plain `packer init`.
2. We own version-constraint resolution. This is real work and the top correctness risk.
3. Clients need one extra binary, or the pure-shell fallback for pinned versions.

---

## 5. Design of the recommended option

All paths below are relative to `/Users/chris/git/vc/terraform-registry/.claude/worktrees/registry-hardening`.

### 5.0 Corrections to prior assumptions about this codebase

Two defects noted in earlier research have already been fixed on this branch, and the design assumes the fixed behaviour:

- **`STORAGE_PATH` / `DATABASE_URL` env vars now work.** `backend/pkg/config/config.go` calls `viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))` and explicitly `BindEnv`s every key. Earlier notes claiming these were silently ignored are stale. A new `packer.*` config key must be added to the same explicit bind list or it will not be readable from the environment.
- **CORS is no longer wildcard-with-credentials.** `api.SetupRouter` now takes `corsAllowedOrigins []string` and `corsMiddleware` never emits `Access-Control-Allow-Credentials`. New routes inherit this correct behaviour automatically.
- **Role-based access control is now enforced.** The single `authorized` group has been split into two: `authenticated` (any valid token; currently only `GET /auth/me`) and `admin` (`auth.AuthMiddleware` plus `auth.RequireRole("admin")`), which holds every state-changing operation. Previously `auth.RequireRole` existed but was never mounted, so any authenticated principal had full write access. **All new Packer management routes must go in the `admin` group.**
- **Access logs redact query strings.** The engine is now built with `gin.New()` plus a custom `redactingLogger()` (`backend/internal/api/middleware.go`), so the JWTs that SSE and export endpoints pass via `?token=` no longer land in the access log.
- **Login is rate limited** (10 requests per minute per client IP). Not directly relevant to Packer, but it establishes the `rateLimitMiddleware` pattern should the plugin download endpoints later need throttling.

Three defects remain and are **prerequisites** for the scheduler portion of this work (section 5.6):

- `SyncHandler.RunScheduleNow` (`backend/internal/api/sync_handler.go`) does not run anything. It stamps `last_status = "running"` and returns; `SyncHandler` holds no reference to the scheduler, and `Scheduler.TriggerSync` has zero callers. The status is left stuck at `running` forever.
- `Scheduler.runSync` constructs its client with `proxy.NewProxyService(s.storagePath, "")` and never calls `SetProxy`, so **scheduled syncs bypass the configured HTTP/SOCKS5 proxy** even though every API handler applies it.
- `Scheduler.downloadAndSavePlatform` only creates platform rows, never updates them, and never sets `FileSize`.

### 5.1 Data model

New file: `backend/internal/models/packer.go`. Style matches `backend/internal/models/provider.go` exactly (GORM tags, soft delete, composite unique index, JSON tags).

```go
// Package models defines data structures for the application.
package models

import (
	"time"

	"gorm.io/gorm"
)

// PackerPluginHost is the upstream host a Packer plugin was ingested from.
// Packer itself only ever fetches from these two hosts; see
// https://github.com/hashicorp/packer/blob/v1.16.0/command/init.go
type PackerPluginHost string

const (
	// PackerHostReleases is the HashiCorp official release site, used for
	// HashiCorp-maintained plugins since Packer 1.14.0.
	PackerHostReleases PackerPluginHost = "releases.hashicorp.com"
	// PackerHostGitHub is github.com, used for all third-party plugins.
	PackerHostGitHub PackerPluginHost = "github.com"
	// PackerHostLocal marks a plugin uploaded or imported directly, with no
	// upstream provenance.
	PackerHostLocal PackerPluginHost = "local"
)

// PackerPlugin represents one version of a Packer plugin.
//
// Coordinates mirror Packer's source-address grammar
// <HOSTNAME>/[SUBFOLDER/]<NAMESPACE>/<TYPE>, documented at
// https://developer.hashicorp.com/packer/docs/templates/hcl_templates/blocks/packer
// Namespace+Name+Version is unique; the upstream host is provenance metadata,
// not part of the identity, so the same plugin cannot be ingested twice under
// two hosts.
type PackerPlugin struct {
	ID          uint             `gorm:"primarykey" json:"id"`
	Namespace   string           `gorm:"index:idx_packer_plugin,unique;not null" json:"namespace"`
	Name        string           `gorm:"index:idx_packer_plugin,unique;not null" json:"name"`
	Version     string           `gorm:"index:idx_packer_plugin,unique;not null" json:"version"`
	Description string           `json:"description"`
	SourceType  SourceType       `gorm:"type:varchar(20);default:'upload'" json:"source_type"`
	UpstreamHost PackerPluginHost `gorm:"type:varchar(32);default:'local'" json:"upstream_host"`
	// SourceAddress is the canonical upstream address, e.g.
	// "github.com/hashicorp/proxmox". Recorded so the UI can show provenance
	// and the seeder can offer the canonical form as an install target.
	SourceAddress string `json:"source_address"`
	// APIVersion is the Packer plugin API version, e.g. "x5.0". It appears in
	// the installed binary filename and is discovered from the zip filename
	// (github.com) or from _manifest.json metadata.protocol_version
	// (releases.hashicorp.com).
	APIVersion string `gorm:"not null;default:'x5.0'" json:"api_version"`
	// SignatureVerified records whether the upstream detached GPG signature
	// over SHA256SUMS was checked at ingest. Packer itself never checks it.
	SignatureVerified bool           `gorm:"default:false" json:"signature_verified"`
	SigningKeyID      string         `json:"signing_key_id"`
	Published         time.Time      `json:"published"`
	Downloads         int64          `json:"downloads"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
	Binaries          []PackerPluginBinary `gorm:"foreignKey:PluginID" json:"binaries,omitempty"`
}

// PackerPluginBinary represents one OS/arch archive of a Packer plugin.
//
// Two distinct checksums are stored, because Packer uses them differently:
//   - ZipSHA256 verifies the downloaded archive against the upstream
//     SHA256SUMS file.
//   - BinarySHA256 is the hexdigest of the EXTRACTED binary, which is what
//     goes into the mandatory <binary>_SHA256SUM sidecar on the client.
//     See https://developer.hashicorp.com/packer/docs/plugins/creation/plugin-load-spec
type PackerPluginBinary struct {
	ID       uint   `gorm:"primarykey" json:"id"`
	PluginID uint   `gorm:"not null;uniqueIndex:idx_packer_binary" json:"plugin_id"`
	OS       string `gorm:"not null;uniqueIndex:idx_packer_binary" json:"os"`
	Arch     string `gorm:"not null;uniqueIndex:idx_packer_binary" json:"arch"`
	// Filename is the archive name as served by upstream. The two hosts differ:
	// releases.hashicorp.com omits the v-prefix and API version, github.com
	// includes both. Always passed through proxy.SanitizeFilename before use.
	Filename string `gorm:"not null" json:"filename"`
	// BinaryFilename is the name of the executable inside the archive, and the
	// exact name the client must write to disk, e.g.
	// packer-plugin-proxmox_v1.2.4_x5.0_linux_amd64
	BinaryFilename string         `gorm:"not null" json:"binary_filename"`
	FilePath       string         `gorm:"not null" json:"file_path"`
	ZipSHA256      string         `gorm:"not null" json:"zip_sha256"`
	BinarySHA256   string         `gorm:"not null" json:"binary_sha256"`
	FileSize       int64          `json:"file_size"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}
```

**Why new tables rather than reusing `Provider` with a discriminator column:**

| Consideration | Reuse `Provider` | New tables |
|---|---|---|
| `idx_provider` unique index | Must be widened to include the discriminator; touches every existing query | Untouched |
| API version dimension | No column; needs one added to a shared table | Native |
| Two checksums (zip + binary) | `ProviderPlatform` has one `SHA256Sum` | Native |
| Existing queries (`ListProviders`, `ListMirroredProviders`, mirror protocol handlers) | All must learn to filter by kind, or silently start returning Packer plugins to Terraform clients | Unaffected |
| Duplicated code | Less | `savePlatformEntry`, delete-cleanup, export/import, aggregation query all cloned |
| Risk of leaking plugins into the Terraform mirror protocol | Real | None |

The deciding factor is the last row. `ProviderMirrorHandler.ListAvailableVersions` and `GetVersionArchives` serve the Terraform CLI directly; a discriminator column that any single query forgets to filter on would advertise Packer plugin zips to Terraform as provider archives. Separate tables make that failure impossible. The duplicated code is real but bounded (roughly 300 lines) and is offset by the shared helper extraction in 5.5.

`PackerPluginBinary` also fixes a defect inherited from `ProviderPlatform`: it declares `uniqueIndex:idx_packer_binary` on `(plugin_id, os, arch)`. `ProviderPlatform` has only a plain index on `provider_id`, so concurrent mirrors can insert duplicate os/arch rows; uniqueness there is enforced only by read-then-write in application code. We do not reproduce that bug.

**Migration.** Add to the `AutoMigrate` call in `backend/cmd/server/main.go`:

```go
if err := db.AutoMigrate(
    &models.Provider{},
    &models.Module{},
    &models.User{},
    &models.ProviderPlatform{},
    &models.MirrorConfig{},
    &models.Settings{},
    &models.SyncSchedule{},
    &models.PackerPlugin{},        // new
    &models.PackerPluginBinary{},  // new
); err != nil {
```

**`SyncSchedule` gains one column** so a single scheduler serves both artifact kinds:

```go
// ArtifactKind selects what this schedule syncs: "provider" (default, for
// backward compatibility with existing rows) or "packer_plugin".
ArtifactKind string `gorm:"not null;default:'provider';index" json:"artifact_kind"`
```

`MirrorConfig` is currently dead — migrated but never read or written by any handler, scheduler or proxy code. It should either be repurposed for per-source upstream configuration or deleted; either way it is out of scope here and should not be extended.

### 5.2 On-disk storage layout

Today, `proxy.buildSafeProviderPath` writes providers directly under the storage root:

```
{storagePath}/{namespace}/{name}/{version}/{os}/{arch}/{filename}.zip
```

**Collision hazard.** Adding Packer plugins under the same root would let a Packer plugin `hashicorp/docker` v1.1.4 linux/amd64 collide with a Terraform provider at identical coordinates. A first-segment namespace is required.

**Chosen prefix: `packer.plugins`.** This is provably collision-free with Terraform provider namespaces, because provider namespaces are validated by `validIdentifier` in `backend/internal/api/mirror_protocol_handler.go`:

```go
var validIdentifier = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)
```

which does not permit a dot, while `proxy.sanitizePathComponent` (which builds the path) does permit dots. So `packer.plugins` is a legal directory component that no valid provider namespace can ever equal.

**Caveat and required guard.** `validateProviderParams` is called by the mirror-protocol handlers and `SyncHandler.CreateSchedule`, but **not** by `MirrorHandler.UploadProvider`, `DownloadProvider`, `GetProviderVersions`, `GetProviderDownloadInfo`, or `Handler.GetProvider`. Those rely on the looser `sanitizePathComponent`. An uploaded provider could therefore claim namespace `packer.plugins`. The design mandates an explicit reserved-prefix rejection inside `buildSafeProviderPath`:

```go
// reservedRootSegments are first-path-segment names owned by other artifact
// kinds. A Terraform provider namespace can never legitimately equal one of
// these (validIdentifier forbids dots), but not every entry point runs that
// validation, so reject them here where every write path converges.
var reservedRootSegments = map[string]struct{}{
	"packer.plugins": {},
}
```

**Resulting layout:**

```
{storagePath}/
├── hashicorp/                                   # Terraform providers (unchanged)
│   └── aws/5.0.0/linux/amd64/terraform-provider-aws_5.0.0_linux_amd64.zip
└── packer.plugins/
    └── github.com/                              # upstream host
        └── hashicorp/                           # namespace
            └── proxmox/                         # plugin type
                └── 1.2.4/                       # version, bare (no v prefix)
                    └── linux/amd64/
                        └── packer-plugin-proxmox_v1.2.4_x5.0_linux_amd64.zip
```

We store the **archive**, not the extracted binary, because it is what upstream signs and checksums, and because the client's `packer plugins install --path` needs a binary it can `describe`, which the seeder extracts locally into a temp dir. Storing the archive keeps the registry's copy byte-identical to upstream, which is what makes the `SHA256SUMS` comparison meaningful.

New function in `backend/internal/proxy/proxy.go`, reusing `sanitizePathComponent` unchanged:

```go
// packerPluginRoot is the reserved first path segment for Packer plugin
// archives. It contains a dot, which validIdentifier forbids in Terraform
// provider namespaces, so it cannot collide with a provider tree.
const packerPluginRoot = "packer.plugins"

// BuildSafePackerPluginPath validates all components and builds a safe
// directory path for a Packer plugin archive. The extra host segment mirrors
// Packer's own on-disk hierarchy, which is derived from the source address:
// https://developer.hashicorp.com/packer/docs/plugins/creation/plugin-load-spec
func BuildSafePackerPluginPath(storagePath, host, namespace, name, version, osType, arch string) (string, error) {
	// ... sanitizePathComponent on each of host/namespace/name/version/os/arch,
	// filepath.Join with packerPluginRoot prepended, then the same
	// strings.HasPrefix(cleanStorage+separator) escape check used by
	// buildSafeProviderPath.
}
```

### 5.3 Route table

Added to `backend/internal/api/router.go`. Handler constructed alongside the others:

```go
packerHandler := NewPackerPluginHandler(db, storagePath)
```

**Public read routes** (seeder and UI read path). Registered directly on `router`, alongside the other public routes and before the `authenticated` and `admin` groups:

```go
// Packer plugin distribution.
// Packer has no network mirror protocol -- see
// https://developer.hashicorp.com/packer/docs/templates/hcl_templates/blocks/packer
// These endpoints are consumed by the vcreg-packer seeder, which drives
// `packer plugins install --path`, not by `packer init` itself.
router.GET("/v1/packer/plugins", packerHandler.ListPlugins)
router.GET("/v1/packer/plugins/:namespace/:name/versions", packerHandler.ListPluginVersions)
router.GET("/v1/packer/plugins/:namespace/:name/:version/manifest", packerHandler.GetPluginManifest)
router.GET("/v1/packer/plugins/:namespace/:name/:version/download/:os/:arch", packerHandler.GetBinaryDownloadInfo)
router.GET("/v1/packer/plugins/:namespace/:name/:version/download/:os/:arch/binary", packerHandler.DownloadBinary)
```

**Admin management routes**, inside the existing `admin := router.Group("/api/v1")` block (which applies `auth.AuthMiddleware(jwtManager)` **and** `auth.RequireRole("admin")`). Every one of these mutates state or triggers outbound network traffic, so none of them belongs in the weaker `authenticated` group:

```go
// Packer plugin management (admin only)
admin.GET("/packer/plugins/upstream/:namespace/:name", packerHandler.ListUpstreamVersions)
admin.POST("/packer/plugins/mirror/:namespace/:name", packerHandler.MirrorPlugin)
admin.GET("/packer/plugins/mirror/:namespace/:name/stream", packerHandler.MirrorPluginWithProgress)
admin.POST("/packer/plugins/upload", packerHandler.UploadPlugin)
admin.DELETE("/packer/plugins/:id", packerHandler.DeletePlugin)
admin.GET("/packer/plugins/export/:id", packerHandler.ExportPlugin)
admin.POST("/packer/plugins/import", packerHandler.ImportPlugin)
```

Note that `ListUpstreamVersions` is admin-gated even though it is a read: it causes the server to make an outbound request to an attacker-chosen `namespace`/`name`, so exposing it publicly would turn the registry into a request-forwarding primitive.

**Public UI read routes**, matching the existing provider pattern:

```go
router.GET("/api/v1/packer/plugins", packerHandler.ListPluginsUI)
router.GET("/api/v1/packer/plugins/:namespace/:name", packerHandler.GetPluginVersionsDetail)
```

**Routing note.** Gin's radix tree can panic at registration time on conflicting siblings. The public tree deliberately uses the static segment `versions` (mirroring the existing `/v1/providers/:namespace/:name/versions`) rather than `index.json`, so `:version` is the only parameter at that depth and no static/parameter sibling conflict arises. The existing tree already proves the harder case works — `/registry.terraform.io/:namespace/:name/index.json` coexists with `/registry.terraform.io/:namespace/:name/:version` under Gin 1.12.0 — but a route-registration test asserting the full table builds without panic is a required deliverable.

**Auth posture.** The five `/v1/packer/...` routes are public, matching the existing Terraform protocol surface, which is unauthenticated regardless of `auth.enabled` (`cfg.Auth.Enabled` is used only to populate the JSON body of `GET /api/v1/auth/status`). This is a deliberate inheritance of existing behaviour, not an endorsement.

The hardening work on this branch has already tightened the management surface (RBAC via `auth.RequireRole("admin")`) without touching the protocol surface, so read access to artifacts remains open by design. Two things follow, and both should be settled at review time rather than after implementation:

- If the protocol surface is later gated, the Packer routes must be gated **identically and simultaneously**, and `vcreg-packer` must learn to send a bearer token (`VCREG_TOKEN`). Design the seeder's HTTP client to attach an optional token from the outset so this is a config change, not a rewrite.
- Unlike Terraform provider archives, plugin binaries are directly executable on the client. An anonymous reader who can enumerate `/v1/packer/plugins` learns exactly which plugin versions the organisation runs. That is a mild disclosure, not a vulnerability, but it argues for gating the Packer surface first if the two are ever split.

**Response shapes.**

`GET /v1/packer/plugins/:namespace/:name/versions`:
```json
{
  "namespace": "hashicorp",
  "name": "proxmox",
  "source_address": "github.com/hashicorp/proxmox",
  "versions": [
    {
      "version": "1.2.4",
      "api_version": "x5.0",
      "signature_verified": true,
      "platforms": [{ "os": "linux", "arch": "amd64" }]
    }
  ]
}
```

`GET /v1/packer/plugins/:namespace/:name/:version/download/:os/:arch`:
```json
{
  "os": "linux",
  "arch": "amd64",
  "api_version": "x5.0",
  "filename": "packer-plugin-proxmox_v1.2.4_x5.0_linux_amd64.zip",
  "binary_filename": "packer-plugin-proxmox_v1.2.4_x5.0_linux_amd64",
  "download_url": "https://registry.example.internal/v1/packer/plugins/hashicorp/proxmox/1.2.4/download/linux/amd64/binary",
  "zip_sha256": "2f0a...c91d",
  "binary_sha256": "8b3e...47af",
  "file_size": 12345678
}
```

`binary_sha256` is what the seeder writes into the `_SHA256SUM` sidecar if it ever needs to bypass `packer plugins install --path`; in the normal flow Packer generates the sidecar itself.

### 5.4 Upstream fetcher

New package: `backend/internal/packerhub`. `proxy.ProxyService` cannot be reused directly — it is hardwired to the Terraform registry v1 JSON shapes (`GetProviderVersions` builds `%s/v1/providers/%s/%s/versions`, `DownloadInfo` decodes `shasum`/`shasums_url`) — but its HTTP client construction should be.

**Required refactor (small, mechanical):** extract the transport builder from `ProxyService.updateHTTPClient` into a package-level constructor so the Packer fetcher inherits HTTP and SOCKS5 proxy support, timeouts and connection pooling for free:

```go
// NewHTTPClient builds an http.Client honouring the registry's configured
// outbound proxy. Extracted from ProxyService.updateHTTPClient so that other
// upstream fetchers (e.g. internal/packerhub) share one proxy implementation.
func NewHTTPClient(proxyEnabled bool, proxyURL, proxyType string) *http.Client
```

While there, fix the latent data race: `ProxyService.httpClient` is guarded by `p.mu` in `SetProxy` but read without the mutex in `GetProviderVersions`, `GetProviderDownloadInfo`, `DownloadAndCacheProvider` and `DownloadAndStoreProvider`.

**Fetcher interface:**

```go
// Fetcher retrieves Packer plugin metadata and archives from an upstream host.
// Two implementations exist because Packer itself has exactly two getters and
// they use incompatible filename conventions; see
// https://github.com/hashicorp/packer/blob/v1.16.0/command/init.go
type Fetcher interface {
	Host() models.PackerPluginHost
	ListVersions(ctx context.Context, namespace, name string) ([]string, error)
	ListPlatforms(ctx context.Context, namespace, name, version string) ([]Platform, error)
	// Download fetches the archive, verifies it against the upstream
	// SHA256SUMS entry, and returns the archive bytes plus the extracted
	// binary's name and hexdigest.
	Download(ctx context.Context, namespace, name, version, os, arch string) (*Archive, error)
}
```

**`releasesFetcher`** (`releases.hashicorp.com`) — four endpoint shapes, all verified live on 2026-08-01:

| Purpose | URL |
|---|---|
| versions | `https://releases.hashicorp.com/packer-plugin-{name}/index.json` |
| checksums | `https://releases.hashicorp.com/packer-plugin-{name}/{ver}/packer-plugin-{name}_{ver}_SHA256SUMS` |
| API version | `https://releases.hashicorp.com/packer-plugin-{name}/{ver}/packer-plugin-{name}_{ver}_manifest.json` |
| archive | `https://releases.hashicorp.com/packer-plugin-{name}/{ver}/{filename from SHA256SUMS}` |

Note `{ver}` is bare (no `v` prefix), and platforms are derived by parsing the `SHA256SUMS` filenames, since the archive names do not carry the API version.

**Important:** Packer's own release getter builds these URLs from the plugin **type only**, discarding the namespace, and is tried first for every source address. Reading the source, `source = "github.com/someone-else/docker"` would first resolve against `https://releases.hashicorp.com/packer-plugin-docker/index.json` and, because that path exists, install HashiCorp's docker plugin under the third party's namespace. This is a namespace-confusion hazard we must **not** reproduce: our fetcher selects `releasesFetcher` only for an explicit allowlist of the eight HashiCorp-maintained plugins (amazon, ansible, azure, docker, googlecompute, qemu, vagrant, virtualbox — `https://developer.hashicorp.com/packer/docs/plugins/install`) under the `hashicorp` namespace, and routes everything else to GitHub. Marked **UNVERIFIED**: the upstream confusion behaviour is an inference from URL construction and getter ordering in the v1.16.0 source, not an executed test.

**`githubFetcher`** (`github.com`) — repository must be named `packer-plugin-{name}` and tagged `v{major}.{minor}.{patch}`:

| Purpose | URL |
|---|---|
| versions | `https://api.github.com/repos/{ns}/packer-plugin-{name}/git/matching-refs/tags` |
| checksums | `https://github.com/{ns}/packer-plugin-{name}/releases/download/v{ver}/packer-plugin-{name}_v{ver}_SHA256SUMS` |
| archive | `https://github.com/{ns}/packer-plugin-{name}/releases/download/v{ver}/{filename from SHA256SUMS}` |

Sends `PACKER_GITHUB_API_TOKEN` (or a new `packer.githubtoken` config key) as a bearer token **only** to `api.github.com`, never to `github.com` or the `objects.githubusercontent.com` redirect target — matching Packer's own `HostSpecificTokenAuthTransport` behaviour, which exists because S3 rejects requests carrying a GitHub token.

**Ingest pipeline** (per binary):

1. Fetch `SHA256SUMS`; parse the two-column `<hex>  <filename>` format.
2. **Sanitise every filename with `proxy.SanitizeFilename`** before it touches any path operation. This is the exact class of bug fixed upstream in GH-13680, where a crafted filename in the remote checksums file escaped the plugin directory via `filepath.Join`.
3. Optionally fetch the detached `SHA256SUMS.sig` and verify against a configured keyring; set `SignatureVerified` and `SigningKeyID`. See 5.7.
4. Download the archive to `<final>.tmp`, streaming through `io.MultiWriter(file, sha256.New())`.
5. **Compare against the expected digest; on mismatch, `os.Remove` the temp file and fail.** This mirrors `DownloadAndCacheProvider` and explicitly does *not* mirror `DownloadAndStoreProvider`, which computes a digest and never compares it.
6. Open the archive, locate the single executable entry, hash it to obtain `BinarySHA256`, and record `BinaryFilename`. Enforce a size cap using the existing `maxFileSize` pattern (`io.LimitReader`, 500 MB) from `MirrorHandler.extractZipFile`, and reject archive entries whose names are not plain basenames (zip-slip).
7. `os.Rename` into place; write the DB rows.

Step 5 deserves emphasis: `proxy.DownloadAndStoreProvider` currently computes a SHA256 and never verifies it, even though its only caller (`ProviderMirrorHandler.asyncCacheProvider`) has already fetched the authoritative checksum one line earlier. **The Packer fetcher must not copy that pattern**, and fixing the provider path is recommended as a companion change.

### 5.5 Reuse map

| Existing asset | Reuse for Packer | Change needed |
|---|---|---|
| `proxy.sanitizePathComponent`, `SanitizeFilename` | Direct | None |
| `proxy.buildSafeProviderPath` | Sibling function | Add `BuildSafePackerPluginPath` + reserved-prefix guard |
| `ProxyService.updateHTTPClient` (proxy/SOCKS5, timeouts) | Shared | Extract `proxy.NewHTTPClient`; fix the unguarded read race |
| `MirrorProgress` struct + SSE pattern | Direct | Reuse the struct verbatim |
| `MirrorHandler.getOrCreateProvider` / `savePlatformEntry` | Pattern | Clone as `getOrCreatePlugin` / `saveBinaryEntry`, with a real unique index |
| `ExportProvider` / `ImportProvider` zip bundle | Pattern | Clone manifest as `PackerPluginExportManifest` |
| `scheduler.Scheduler` (robfig/cron v3, 5-field, 30s DB reload) | Extend | Branch on `SyncSchedule.ArtifactKind` |
| `auth.AuthMiddleware` (incl. `?token=` for SSE) | Direct | None |
| `internal/storage` (`Storage` interface, `LocalStorage`) | **Not reused** | Currently dead code — zero importers. Either wire both artifact kinds through it as a precursor to object storage, or delete it. Do not half-adopt it. |

### 5.6 Scheduler integration

`Scheduler.runSync` gains a branch:

```go
switch schedule.ArtifactKind {
case "packer_plugin":
    err = s.syncPackerPlugin(schedule)
default: // "provider"
    err = s.mirrorProvider(...)
}
```

Three existing defects must be fixed first, because Packer syncs inherit them:

1. **Apply proxy settings.** `runSync` must read `models.Settings` and call `SetProxy` (or use the extracted `proxy.NewHTTPClient`) before any upstream call. Today scheduled syncs silently bypass the configured proxy, which in a restricted-egress environment means they simply never work.
2. **Wire `RunScheduleNow`.** Give `SyncHandler` a `*scheduler.Scheduler` and have `RunScheduleNow` call `TriggerSync` (currently zero callers) in a goroutine. Today the button stamps `last_status = "running"` and nothing ever runs or clears it.
3. **Update, do not only insert.** `downloadAndSavePlatform` must update existing rows (refreshing `file_path`, checksums and `file_size`) as `MirrorHandler.savePlatformEntry` does.

Also note that scheduled syncs currently pull **only the single latest upstream version** (`mirrorProvider` is called with `version == ""`, which resolves to `versions.Versions[0]`), and that "latest" is computed by **lexicographic string comparison** throughout the codebase — `MAX(version)`, `ORDER BY version DESC` on a TEXT column — so `1.10.0` sorts below `1.9.0`. There is no semver library in `go.mod`.

Since the seeder must implement Packer's constraint operators anyway, introduce a single semver dependency (`github.com/hashicorp/go-version`, which is what Packer itself uses) and use it consistently for Packer plugins. Retrofitting it to providers is a desirable but separable change.

### 5.7 Signature verification (differentiator)

Packer verifies no signatures at all. Both upstream hosts publish detached GPG signatures over `SHA256SUMS`:

- `releases.hashicorp.com/packer-plugin-docker/index.json` advertises `"shasums_signature": "packer-plugin-docker_1.1.4_SHA256SUMS.sig"` and `"shasums_signatures": ["...SHA256SUMS.72D7468F.sig", "...SHA256SUMS.sig"]`, where `72D7468F` is HashiCorp's release key ID.
- Third-party plugin releases carry an author-signed `.sig` produced by the scaffolding `.goreleaser.yml` (`gpg --batch --local-user $GPG_FINGERPRINT --detach-sign`). Confirmed present on the live proxmox release listing in section 2.4.

Verifying at ingest and surfacing `signature_verified` in the UI and API gives the registry a genuine security advantage over `packer init`, at modest cost. Since Packer dropped `x/crypto/openpgp`, we would add a maintained OpenPGP library (e.g. `github.com/ProtonMail/go-crypto/openpgp`) plus an admin-managed keyring. Recommended for **Phase 2**, not Phase 1, to keep the critical path short. `SignatureVerified` defaults to `false` and must be rendered honestly as "not verified" rather than omitted.

### 5.8 Frontend

New pages under `frontend/src/pages/`, registered in `frontend/src/App.jsx` alongside the existing routes:

| File | Route | Content |
|---|---|---|
| `PackerPlugins.jsx` | `/packer` | List/search mirrored plugins; per-row version count, upstream host badge, signature-verified state |
| `PackerPluginDetail.jsx` | `/packer/:namespace/:name/:version` | Per-OS/arch table, checksums, provenance, copy-paste `required_plugins` block |
| `PackerMirror.jsx` | `/packer/mirror` | Mirror-from-upstream form with SSE progress; upload; export/import. Clone of `Mirror.jsx` |

`frontend/src/services/api.js` gains matching functions. The SSE helper already handles the `?token=` query-parameter auth pattern (`AuthMiddleware` accepts the JWT from either the `Authorization` header or `?token=`) and is reused as-is.

`frontend/src/pages/Docs.jsx` gains a "Packer Plugins" section. Its content must state plainly that `packer init` cannot fetch from this registry over the network, that the seeder is required, and that `packer init -upgrade` is unsupported with private source addresses. The existing Terraform section hardcodes a `network_mirror` snippet keyed off `REGISTRY_HOST`; the Packer section has no such equivalent and must not imply one exists.

**Security note carried forward:** the Packer SSE and export endpoints reuse the `?token=` query-parameter auth path, because `EventSource` cannot send an `Authorization` header. Access-log exposure is already mitigated by `redactingLogger()`, but the token still reaches `Referer` headers, browser history and any intermediate proxy log. Resolving this repo-wide (short-lived single-use stream tokens, or replacing `EventSource` with `fetch` + `ReadableStream` so a real header can be sent) is worthwhile but out of scope here. The Packer work should not widen the exposure: reuse the existing helper rather than inventing a second query-token path.

---

## 6. Client-side usage

### 6.1 Recommended: private source address plus seeder

`packer/proxmox-ubuntu/main.pkr.hcl`:

```hcl
packer {
  required_version = ">= 1.14.1"

  required_plugins {
    proxmox = {
      version = "~> 1.2"
      source  = "registry.example.internal/hashicorp/proxmox"
    }
  }
}

source "proxmox-iso" "ubuntu" {
  # ...
}

build {
  sources = ["source.proxmox-iso.ubuntu"]
}
```

`~> 1.2` means `>= 1.2, < 2.0`, per `https://developer.hashicorp.com/packer/docs/templates/hcl_templates/blocks/packer`.

Shell:

```bash
export VCREG_URL="https://registry.example.internal"
export PACKER_PLUGIN_PATH="$HOME/.config/packer/plugins"

# Resolve constraints against the registry, download, and install.
vcreg-packer seed ./packer/proxmox-ubuntu

# Now offline: exits 0 with no network calls.
packer init ./packer/proxmox-ubuntu
packer validate ./packer/proxmox-ubuntu
packer build ./packer/proxmox-ubuntu
```

Do **not** run `packer init -upgrade` or `packer init -force` — both bypass the already-installed short-circuit and reach the getter's `github.com` hostname assertion, failing with:

```
"registry.example.internal/hashicorp/proxmox" doesn't appear to be a valid
"github.com" source address; check source and try again.
```

### 6.2 CI (GitLab runner, restricted egress)

```yaml
build-image:
  image: hashicorp/packer:1.16.0
  variables:
    VCREG_URL: "https://registry.example.internal"
    PACKER_PLUGIN_PATH: "$CI_PROJECT_DIR/.packer-plugins"
  cache:
    key: "packer-plugins-$CI_COMMIT_REF_SLUG"
    paths:
      - .packer-plugins/
  script:
    - vcreg-packer seed ./packer
    - packer init ./packer          # NOT -upgrade
    - packer validate ./packer
    - packer build ./packer
```

Caching `PACKER_PLUGIN_PATH` between runs makes the seeder a no-op on warm runners.

### 6.3 Pure-shell fallback (no seeder binary, pinned version)

For a single pinned version, the whole flow is four commands and needs nothing but `curl`, `unzip` and `packer`:

```bash
#!/usr/bin/env bash
set -euo pipefail

VCREG_URL="https://registry.example.internal"
NS="hashicorp"; NAME="proxmox"; VER="1.2.4"
OS="$(go env GOOS 2>/dev/null || uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(go env GOARCH 2>/dev/null || uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')"

BASE="${VCREG_URL}/v1/packer/plugins/${NS}/${NAME}/${VER}/download/${OS}/${ARCH}"
TMP="$(mktemp -d)"; trap 'rm -rf "$TMP"' EXIT

# Fetch metadata (for the expected checksum) and the archive.
EXPECTED="$(curl -fsSL "${BASE}" | sed -n 's/.*"zip_sha256"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')"
curl -fsSL -o "${TMP}/plugin.zip" "${BASE}/binary"

ACTUAL="$(sha256sum "${TMP}/plugin.zip" | cut -d' ' -f1)"
[ "$ACTUAL" = "$EXPECTED" ] || { echo "checksum mismatch: want $EXPECTED got $ACTUAL" >&2; exit 1; }

unzip -q -o "${TMP}/plugin.zip" -d "${TMP}/x"
BIN="$(find "${TMP}/x" -type f -name 'packer-plugin-*' ! -name '*.txt' | head -1)"
chmod +x "$BIN"

# Packer computes and writes the mandatory _SHA256SUM sidecar itself.
packer plugins install --path "$BIN" "registry.example.internal/${NS}/${NAME}"
```

The `packer plugins install --path` form is documented at `https://developer.hashicorp.com/packer/docs/plugins/install` as:

```
packer plugins install --path <path-to-downloaded-extracted-binary> <hostname>/<namespace>/<plugin-name>
```

### 6.4 Verifying the result

```bash
$ packer plugins installed
/home/dev/.config/packer/plugins/registry.example.internal/hashicorp/proxmox/packer-plugin-proxmox_v1.2.4_x5.0_linux_amd64

$ ls /home/dev/.config/packer/plugins/registry.example.internal/hashicorp/proxmox/
packer-plugin-proxmox_v1.2.4_x5.0_linux_amd64
packer-plugin-proxmox_v1.2.4_x5.0_linux_amd64_SHA256SUM

$ PACKER_LOG=1 packer init ./packer 2>&1 | grep -ciE 'github|releases\.hashicorp'
0
```

The last check is the acceptance test for "no egress": zero getter requests. Note `packer plugins installed` takes no flags and lists binaries matching the current OS/arch.

### 6.5 Air-gapped transfer

The high side exports a bundle; the low side imports it. This clones the existing provider export/import format (a zip containing `manifest.json` plus `{os}/{arch}/{filename}` entries).

```bash
# Internet-connected registry
curl -fsSL -H "Authorization: Bearer $TOKEN" \
  "https://registry.dmz.example.com/api/v1/packer/plugins/export/42" \
  -o packer-plugin-proxmox_1.2.4.zip

# ... physical transfer ...

# Air-gapped registry
curl -fsSL -H "Authorization: Bearer $TOKEN" \
  -F "file=@packer-plugin-proxmox_1.2.4.zip" \
  "https://registry.airgap.example.internal/api/v1/packer/plugins/import"
```

One inherited defect to fix in the clone: `MirrorHandler.saveImportedPlatform` trusts the SHA256 recorded in the bundle manifest rather than recomputing it from the extracted bytes. The Packer importer must recompute and compare, otherwise the bundle format has no integrity property at all.

---

## 7. Machine image distribution (Reading B)

Not solved by this document. Recorded so the reader can act on it separately.

### 7.1 What Packer offers today

Packer core v1.16.0 ships exactly six post-processors: `artifice`, `checksum`, `compress`, `manifest`, `provenance`, `shell-local`. Everything that publishes to a real target lives in plugins.

| Target | Component | Notes |
|---|---|---|
| Internal Docker registry | `docker-push` post-processor | Set `login_server` to the internal registry plus `login`, `login_username`, `login_password`. Also supports `ecr_login`. The most turnkey internal target. `https://developer.hashicorp.com/packer/integrations/hashicorp/docker/latest/components/post-processor/docker-push` |
| vSphere | `vsphere` post-processor | Uploads a VMX, OVA or OVF artifact to a vSphere endpoint. `https://github.com/hashicorp/packer-plugin-vsphere/blob/main/docs/post-processors/vsphere.mdx` |
| vSphere template | `vsphere-template` post-processor | Separate component that converts the result into a template |
| GCS | `googlecompute-export` post-processor | Exports the built GCE image as a gzipped tarball to GCS `paths`. Deletes the source image unless `keep_input_artifact = true`. Only accepts googlecompute-builder or artifice artifacts |
| S3 | `amazon-import` post-processor | Requires `s3_bucket_name`; the uploaded object is removed after import unless `skip_clean = true` |
| Proxmox | **none** | `packer-plugin-proxmox` has builders only (`clone`, `iso`) and no post-processor directory. Templates are produced by the builder directly |
| qemu / qcow2 | **none** | Ship via core `artifice` → `compress` → `shell-local` |

Marked **UNVERIFIED**: the individual field names of the `vsphere` post-processor (host, username, password, datacenter, datastore, cluster, resource_pool, insecure) come from `@include` partials that are not expanded in the raw `.mdx`, and were not confirmed against source.

### 7.2 HCP Packer is a metadata registry, not a blob store

> "HCP Packer stores the metadata associated with the artifacts you build but not the artifact."
> — `https://developer.hashicorp.com/hcp/docs/packer`

Its object model is registry → bucket → version → build → artifact, with channels as named pointers to versions. The API is a public Swagger 2.0 spec at `https://developer.hashicorp.com/hcp/api-docs/packer`, host `api.cloud.hashicorp.com`, version `2023-01-01`. Packer's client is compiled into core (`internal/hcp/`), triggered by an `hcp_packer_registry` block inside a `build` block — **not** by a post-processor. It calls roughly 13 of the 52 published operations.

There is **no documented self-hosted HCP Packer**. Marked **UNVERIFIED** that any private offering exists; the public docs describe only the managed service. Mechanically, the underlying `hcp-sdk-go` honours `HCP_API_ADDRESS`, `HCP_AUTH_URL` and `HCP_API_TLS`, so a compatible third-party server is addressable — but a compatible server would also have to implement an OAuth2 client-credentials token endpoint and the cloud-resource-manager organization/project endpoints, and nothing sanctions third-party endpoints.

Note the terminology trap: Packer templates use hyphens (`hcp-packer-version`, `hcp-packer-artifact`), Terraform uses underscores (`hcp_packer_version`, `hcp_packer_artifact`). Also note the Terraform `hashicorp/hcp` provider hard-**removed** `hcp_packer_image` and `hcp_packer_iteration` in v0.84.0 (2024-03-22); they are not merely deprecated.

Finally, `toowoxx/packer` on the Terraform Registry is an unrelated community provider that wraps the Packer CLI. It is not HCP Packer and has nothing to do with plugin distribution. Do not conflate them.

### 7.3 Sketch for an internal image catalog

If Reading B is later prioritised, the shape that fits this registry is **metadata-only**, following HCP Packer's model — store pointers, never the multi-gigabyte images.

```
packer build
   |
   +-- builder (proxmox-iso / vsphere-iso / amazon-ebs)
   |        produces the image in the target platform
   |
   +-- post-processor "manifest"        -> manifest.json (custom_data carries
   |                                       team, git sha, CVE scan id, ...)
   +-- post-processor "provenance"      -> SLSA Build L1/L2 attestation (new in
   |                                       1.16.0); fulcio_url and rekor_url are
   |                                       CONFIGURABLE, so this component can be
   |                                       pointed at internal Sigstore infra
   +-- post-processor "shell-local"     -> curl POST manifest + attestation to
                                           the registry catalog API
                                                |
                                                v
                    registry: image_catalog table
                    (namespace, name, version, platform, region,
                     external_identifier, channel, attestation_ref)
                                                |
                                                v
                    Terraform reads it:
                    data "http" / a small internal provider ->
                    aws_instance.ami / proxmox_vm_qemu.clone
```

The `provenance` post-processor is the most interesting piece for a hardening-focused project: it produces SLSA Provenance v1 predicates and signs via a local PEM key, cloud KMS (`awskms://`, `gcpkms://`, `azurekms://`, `hashivault://`) or keyless Sigstore, and it is one of the very few Packer components whose upstream endpoints (`fulcio_url`, `rekor_url`) are configurable — unlike the plugin getters (`https://github.com/hashicorp/packer/blob/v1.16.0/CHANGELOG.md`, GH-13667).

Rough cost if pursued: 8-12 engineer-days for the catalog table, ingest endpoint, channel-promotion semantics and a UI, plus per-platform integration work. It shares almost nothing with Reading A beyond the auth middleware and the frontend shell.

---

## 8. Effort estimate

Engineer-days, one engineer familiar with this codebase. Phase 1 is the minimum shippable increment.

### Phase 1: core plugin distribution (target 20 days)

| Component | Days | Notes |
|---|---|---|
| Data model + AutoMigrate + `SyncSchedule.ArtifactKind` | 0.5 | Mechanical; mirrors `provider.go` |
| Storage path builder + reserved-prefix guard + tests | 1.0 | Reuses `sanitizePathComponent`; must include traversal tests |
| `proxy.NewHTTPClient` extraction + mutex race fix | 0.5 | Small refactor, unblocks the fetcher |
| `internal/packerhub` releases.hashicorp.com fetcher | 1.5 | Four endpoints; `manifest.json` for API version |
| `internal/packerhub` github.com fetcher | 2.0 | Tags API, token scoping, redirect handling, rate-limit backoff |
| Archive ingest: verify, extract, hash binary, zip-slip guards | 2.0 | Highest security-sensitivity code in the project |
| Public read handlers (5 routes) | 1.5 | Includes the route-registration panic test |
| Management handlers (mirror, SSE, upload, delete) | 2.0 | Clones `MirrorHandler` patterns |
| Export/import bundle + recompute-checksum fix | 1.5 | Clone of provider bundle |
| Seeder client `vcreg-packer` | 3.0 | Includes constraint resolution; see risk below |
| Frontend: 3 pages + api.js + Docs section | 3.0 | Clones `Mirror.jsx` / `Providers.jsx` |
| Tests (unit + handler + an end-to-end seed-then-init) | 1.5 | On top of per-component tests above |
| **Phase 1 total** | **20.0** | |

### Phase 2 and 3

| Component | Days | Notes |
|---|---|---|
| Scheduler integration for Packer plugins | 1.0 | After the prerequisite fixes below |
| Prerequisite scheduler fixes (proxy, `RunScheduleNow`, update-vs-insert) | 2.0 | Independently valuable; currently broken for providers too |
| Semver adoption (`hashicorp/go-version`) for Packer paths | 1.0 | Lexicographic sorting is wrong today |
| GPG signature verification at ingest + keyring admin UI | 3.0 | The real differentiator over upstream Packer |
| Option C endpoint emulation (`releases.hashicorp.com` + GitHub shapes) | 3.0 | Cheap once Phase 1 storage exists; excludes proxy/CA ops work |
| Upstream getter PR (Option D side bet) | 5.0+ | Unbounded; do not schedule on the critical path |

### Riskiest unknowns

Ranked by expected damage.

1. **Version-constraint resolution (high).** We must reimplement Packer's semantics: `=`, `!=`, `>`, `>=`, `<`, `<=`, `~>`, with `~> 0.9` meaning `>= 0.9, < 1.0`, and prereleases matching only exact constraints. Using `hashicorp/go-version` (Packer's own dependency) reduces but does not eliminate divergence risk, because Packer's *selection* policy on top of the constraint is also in play. Mitigation: differential-test our resolver against real `packer init` output for a matrix of constraints, and make the seeder print the resolved version so mismatches are visible rather than silent.

2. **`packer init` short-circuit behaviour on v1.16.0 (high).** The entire design rests on plain `packer init` exiting 0 with no network calls when a private-host requirement is already satisfied. This is read off `command/init.go` control flow and corroborated by testing against v1.15.4, **not** verified on v1.16.0. Mitigation: verify on day 1, before any code is written. If it regressed, the whole approach needs rework.

3. **GitHub rate limiting during bulk ingest (medium).** Unauthenticated `api.github.com` allows 60 requests/hour. Mirroring many plugins will hit it. Mitigation: require a token in registry config, cache tag listings, exponential backoff on `RateLimitError`.

4. **Gin route-tree conflicts (low-medium, but fails loudly at startup).** The static/parameter sibling pattern is already proven in this tree, but the new subtree is deeper. Mitigation: a test that calls `SetupRouter` and asserts no panic, added before the handlers.

5. **Cross-platform seeder correctness (medium).** OS/arch detection, Windows `.exe` handling, and the plugin-root resolution chain (`PACKER_PLUGIN_PATH` → `PACKER_CONFIG_DIR` → legacy `$HOME/.packer.d` → `XDG_CONFIG_HOME` → `$HOME/.config/packer`) each have edge cases. Mitigation: delegate placement to `packer plugins install --path` wherever possible, so Packer owns the layout; only implement the fallback path when Packer is absent.

6. **Behavioural drift in Packer (ongoing, low per-release).** Packer shipped a path-traversal fix in the getter as recently as v1.16.0. Mitigation: pin a tested Packer version range in docs, and re-run the end-to-end acceptance test against each new minor release.

---

## 9. Open questions

1. **Does plain `packer init` still exit 0 with zero network calls on v1.16.0 for a pre-seeded private-host source?** The behaviour is inferred from `command/init.go` and was empirically confirmed only on v1.15.4. Blocking; verify first.
2. **Does `packer plugins install --path` genuinely preserve version pinning?** The packer block reference says version is pinned only when installed this way, but the mechanism was not confirmed end-to-end, and there is a known defect where `plugins_install.go` guards on `MetaArgs.Path` rather than `PluginPath`, meaning the documented "cannot be specified with a version constraint" error never fires and the version argument is silently ignored.
3. **Should the public `/v1/packer/...` routes be authenticated?** They currently follow the existing Terraform protocol surface, which is public regardless of `auth.enabled`. If the hardening work gates that surface, the Packer routes and the seeder must change together.
4. **Repurpose or delete `MirrorConfig` and `internal/storage`?** Both are dead. `internal/storage` is the natural seam for an S3/blob backend and has zero importers; adopting it for Packer only would leave the codebase with two storage idioms.
5. **Is `packer.plugins` the right reserved prefix,** or should the provider tree be moved under `terraform.providers/` for symmetry? The latter is cleaner but requires a data migration of existing deployments.
6. **Does an HTTPS caching proxy work end-to-end for plugin installs (Option C viability)?** Proxy env vars are demonstrably honoured, but the full TLS-interception path including the `github.com` → `objects.githubusercontent.com` redirect chain was not tested.
7. **Is the release-getter namespace-confusion behaviour a known upstream design decision or an unreported bug?** No issue or advisory discussing it was found. Worth reporting upstream regardless; we avoid it via the allowlist in 5.4.
8. **Is there a post-IBM-acquisition roadmap for plugin mirroring?** Issue #11164 carries a `sync to jira` label implying internal tracking, but no maintainer has commented publicly since 2024-10-09.
9. **Does `releases.hashicorp.com` expose an index of all `packer-plugin-*` names?** Only the per-plugin `/{type}/index.json` endpoint was confirmed. If not, the registry must be seeded from a known plugin list.

## References

Packer documentation (primary):
- Plugin installation: `https://developer.hashicorp.com/packer/docs/plugins/install`
- Packer block / `required_plugins`: `https://developer.hashicorp.com/packer/docs/templates/hcl_templates/blocks/packer`
- `packer plugins install` command: `https://developer.hashicorp.com/packer/docs/commands/plugins/install`
- Plugin load specification: `https://developer.hashicorp.com/packer/docs/plugins/creation/plugin-load-spec`
- Plugins overview: `https://developer.hashicorp.com/packer/docs/plugins`
- Configuring Packer / environment variables: `https://developer.hashicorp.com/packer/docs/configure`

Packer source (primary):
- Getter construction and ordering: `https://github.com/hashicorp/packer/blob/v1.16.0/command/init.go`
- `packer plugins install` implementation: `https://github.com/hashicorp/packer/blob/v1.16.0/command/plugins_install.go`
- releases.hashicorp.com getter: `https://github.com/hashicorp/packer/blob/v1.16.0/packer/plugin-getter/release/getter.go`
- github.com getter and hostname assertion: `https://github.com/hashicorp/packer/blob/v1.16.0/packer/plugin-getter/github/getter.go`
- Install, checksum and discovery logic: `https://github.com/hashicorp/packer/blob/v1.16.0/packer/plugin-getter/plugins.go`
- Checksummer and `_SHA256SUM` naming: `https://github.com/hashicorp/packer/blob/v1.16.0/packer/plugin-getter/checksum.go`
- Source address parser: `https://github.com/hashicorp/packer/blob/v1.16.0/hcl2template/addrs/plugin.go`
- Plugin folder resolution: `https://github.com/hashicorp/packer/blob/v1.16.0/packer/plugin_folders.go`
- CHANGELOG (1.11.0 local-source relaxation, 1.14.0 releases.hashicorp.com move, 1.16.0 provenance and GH-13680): `https://github.com/hashicorp/packer/blob/v1.16.0/CHANGELOG.md`
- Path-traversal fix in the GitHub getter: `https://github.com/hashicorp/packer/pull/13680`
- Proxy support fix and `InsecureSkipVerify` removal: `https://github.com/hashicorp/packer/pull/13439`
- Open feature request, non-GitHub plugin sources: `https://github.com/hashicorp/packer/issues/11164`
- Closed-as-stale private registry request: `https://github.com/hashicorp/packer/issues/12427`
- Closed community PR on source parsing: `https://github.com/hashicorp/packer/pull/12863`

Packer plugins (primary):
- vSphere post-processors: `https://github.com/hashicorp/packer-plugin-vsphere/blob/main/docs/post-processors/vsphere.mdx`
- docker-push post-processor: `https://developer.hashicorp.com/packer/integrations/hashicorp/docker/latest/components/post-processor/docker-push`
- googlecompute-export post-processor: `https://github.com/hashicorp/packer-plugin-googlecompute/blob/main/docs/post-processors/googlecompute-export.mdx`
- amazon-import post-processor: `https://developer.hashicorp.com/packer/integrations/hashicorp/amazon/latest/components/post-processor/import`

HCP Packer (primary):
- Overview: `https://developer.hashicorp.com/hcp/docs/packer`
- Object model: `https://developer.hashicorp.com/hcp/docs/packer/store`
- API reference: `https://developer.hashicorp.com/hcp/api-docs/packer`
- Terraform provider data sources: `https://github.com/hashicorp/terraform-provider-hcp/tree/main/docs/data-sources`
- Provider CHANGELOG (v0.84.0 removals): `https://github.com/hashicorp/terraform-provider-hcp/blob/main/CHANGELOG.md`

Third-party repository managers (vendor documentation; secondary relative to the Packer-centric trust hierarchy):
- JFrog Artifactory supported package types: `https://docs.jfrog.com/artifactory/docs/supported-package-types`
- Sonatype Nexus formats: `https://help.sonatype.com/en/formats.html`
- Cloudsmith formats: `https://docs.cloudsmith.com/formats`

Live endpoints verified 2026-08-01:
- `https://releases.hashicorp.com/packer-plugin-docker/index.json`
- `https://releases.hashicorp.com/packer-plugin-docker/1.1.4/packer-plugin-docker_1.1.4_SHA256SUMS`
- `https://releases.hashicorp.com/packer-plugin-docker/1.1.4/packer-plugin-docker_1.1.4_manifest.json`
- `https://api.github.com/repos/hashicorp/packer/releases/latest`
- `https://api.github.com/repos/hashicorp/packer-plugin-proxmox/releases/latest`

This repository (read directly, worktree `registry-hardening`):
- `backend/internal/models/provider.go`, `backend/internal/api/router.go`, `backend/internal/proxy/proxy.go`, `backend/internal/api/mirror_handler.go`, `backend/internal/api/mirror_protocol_handler.go`, `backend/internal/api/sync_handler.go`, `backend/internal/scheduler/scheduler.go`, `backend/internal/storage/storage.go`, `backend/cmd/server/main.go`, `backend/pkg/config/config.go`, `frontend/src/App.jsx`, `frontend/src/services/api.js`
