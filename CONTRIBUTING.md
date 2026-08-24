# Contributing to Cerebro

You can contribute to Cerebro through code, documentation, or bug reports. This guide
covers the development setup, build/test/lint commands, and the pull request process.

## Bug reports

When filing a bug report, please include:

- The **Cerebro version** you are running.
- The **backend flavor and version** (Elasticsearch or OpenSearch, and the exact version).
- A relevant excerpt of `conf/application.yaml` with **secrets redacted**.
- Clear **reproduction steps**, plus the **expected** and **actual** behavior.

## Repository layout

| Path                     | Contents                                                       |
|--------------------------|----------------------------------------------------------------|
| `cmd/cerebro/`           | Go entrypoint that wires config, server, and UI serving.       |
| `internal/`              | Go packages (server, config, cluster client, handlers, ...).  |
| `app/`                   | Nuxt 4 / Vue 3 frontend (default `srcDir`, `ssr: false`).      |
| `test/`                  | Vitest specs for the frontend.                                 |
| `conf/`                  | Runtime config (`application.yaml`).                           |
| `deploy/`                | Docker, Kubernetes, and Helm deployment artifacts.            |
| `dagger/`                | Dagger CI module (lint, test, build, acceptance tests).        |
| `.github/workflows/`     | CI pipeline definitions.                                       |

## Development setup

Requirements: **Go 1.26** and **Node.js 22**.

Install frontend dependencies:

```sh
npm ci --legacy-peer-deps
```

Backend dev loop (serves on `:9000`):

```sh
go run ./cmd/cerebro
```

Frontend dev server:

```sh
npm run dev
```

> Note: the app calls the backend via relative URLs and `nuxt.config.ts` does not
> configure a dev proxy, so the verified full dev loop is to build the UI and run the
> Go binary from the repo root:

```sh
npm run build && ./cerebro
```

## Build

```sh
# Backend binary at the repo root
go build -o cerebro ./cmd/cerebro

# Frontend (emits output/public)
npm run build
```

## Testing

```sh
# Go unit tests
go test ./...

# Frontend unit tests (vitest)
npm test
```

Acceptance tests are tagged with `//go:build acceptance` and run via the Dagger module
against real Elasticsearch 9.4.3 and OpenSearch 3.4.0 containers.

## Lint / style

```sh
# Frontend: ESLint v9 flat config (eslint.config.js)
npm run lint

# Go: gofmt + golangci-lint
gofmt -l .
golangci-lint run
```

`golangci-lint` v2.12.2 is pinned in the Dagger module for reproducible CI runs.

## Pull requests

- **Open an issue first** for new features or larger changes, so the approach can be
  agreed on before you invest time.
- **Add tests** that validate your changes.
- **Do not commit generated artifacts.** The `cerebro` binary, `dist/`, `output/`, and
  `server.pid` are gitignored — keep them out of your commits.
- **Squash** development commits (fixups, typos) so only meaningful commits remain.
- **Rebase** your branch against `main` before submitting.
