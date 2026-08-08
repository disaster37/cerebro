# Cerebro

[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![CI](https://github.com/disaster37/cerebro/actions/workflows/ci.yml/badge.svg)](https://github.com/disaster37/cerebro/actions/workflows/ci.yml)

Cerebro is an open-source (MIT) web admin tool for **Elasticsearch 9** and
**OpenSearch 3**, built with [Go](https://go.dev) (Hertz) and
[Vue 3](https://vuejs.org) / [Nuxt 4](https://nuxt.com). It is a from-scratch rewrite of
the [original Scala/Play/AngularJS project](https://github.com/lmenezes/cerebro).

## Requirements

- **Build time:** Go 1.26 and Node.js 22.
- **Runtime:** nothing special — a single static binary that serves the bundled UI.

## Build

```sh
# Backend: produces the ./cerebro binary at the repo root
go build -o cerebro ./cmd/cerebro

# Frontend: installs deps and emits output/public (served by the binary)
npm ci --legacy-peer-deps && npm run build
```

## Run

From the repository root:

```sh
./cerebro
```

The server listens on port **9000** by default and reads its configuration from
`conf/application.yaml`. The binary serves the built UI from `output/public`, so it must
be launched from the repo root (or wherever `output/public` and `conf/` live).

## Configuration

All settings live in [`conf/application.yaml`](conf/application.yaml). The key ones:

| Setting        | Description                                                                 |
|----------------|-----------------------------------------------------------------------------|
| `secret`       | Secret used to sign session cookies. **Change it in production.**          |
| `basePath`     | Path prefix when serving behind a reverse proxy.                           |
| `data.path`    | Path to the local SQLite database (rest request history).                  |
| `hosts[]`      | Pre-registered clusters; each entry supports `host`, `name`, optional       |
|                | `type` (`elasticsearch` \| `opensearch`; auto-detected via `GET /` when     |
|                | omitted) and `headers-whitelist`.                                          |
| `auth.type`    | `""` (default, no auth), `basic`, or `ldap`; see `auth.settings.*` for the  |
|                | corresponding credentials / LDAP parameters.                                |

See the file itself for the full list and comments.

## Docker

```sh
docker build -f deploy/docker/Dockerfile -t cerebro:local .
```

A ready-to-use Compose stack is provided at
[`deploy/docker/docker-compose.yml`](deploy/docker/docker-compose.yml). It binds to
`127.0.0.1:9000` and **requires** a `CEREBRO_SECRET` environment variable:

```sh
export CEREBRO_SECRET=$(openssl rand -hex 32)
docker compose -f deploy/docker/docker-compose.yml up -d
```

> **Warning:** Authentication is disabled by default (`auth.type: ""`) and cerebro can
> proxy HTTP requests to arbitrary hosts (SSRF risk). Do not expose it beyond localhost
> until auth is enabled and a unique `CEREBRO_SECRET` is set. See `conf/application.yaml`.

## Kubernetes / Helm

- Raw manifests: [`deploy/kubernetes/`](deploy/kubernetes/) (deployment, service,
  configmap, secret, pvc, namespace, kustomization).
- Helm chart: [`deploy/helm/cerebro/`](deploy/helm/cerebro/) — lint with
  `helm lint deploy/helm/cerebro`.

## Development

See [`CONTRIBUTING.md`](CONTRIBUTING.md) for the development setup, build, test, and lint
instructions.

## Changelog & License

- [CHANGES.md](CHANGES.md)
- [LICENSE](LICENSE) (MIT)
