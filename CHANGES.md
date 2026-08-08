Cerebro Releases
------------

### Pre-rewrite history (v0.1.0 - v0.9.4)

Cerebro was originally written in Scala, Play Framework and AngularJS. The release notes
for those versions (July 2016 - April 2021) were removed during the legacy cleanup; they
remain available in the git history of this file.

### Dual backend (Elasticsearch 9 + OpenSearch 3)

#### Features
- Cerebro now supports both Elasticsearch and OpenSearch target clusters.
- Per-host `type` config field (`elasticsearch` | `opensearch`, optional;
  auto-detected via `GET /` sniffing `version.distribution` when omitted).
- `POST /navbar` response gains `backendType` and `backendVersion` fields.
- Navbar shows a backend badge (blue `Elasticsearch <ver>` / indigo
  `OpenSearch <ver>`) identifying the connected cluster flavor.

#### Backend
- Elasticsearch client bumped to `github.com/elastic/go-elasticsearch/v9 v9.5.0`.
- OpenSearch engine via `github.com/disaster37/opensearch/v4` (resty-based raw
  requests; lossless `{status,body}` passthrough including non-2xx).
- `MultiClient` router dispatches per request on the resolved flavor; flavor
  cache (10 min TTL) keyed by host URL; configured `type` overrides detection.
- Shared `targetTransport` (host rewrite, Basic Auth, whitelisted headers,
  gzip/identity) used by both engines and by detection.
- Go toolchain bumped to `go 1.26`; SQLite driver `modernc.org/sqlite` added
  (fixes the pre-existing missing-driver bug for `data.path`).

#### Tests / CI
- Go unit tests (stdlib + httptest) for transport, response, flavor, dispatch,
  OS engine request building, config, and navbar handler.
- Acceptance tests (`//go:build acceptance`) run the same suite against real
  Elasticsearch 9.4.3 and OpenSearch 3.4.0 containers provisioned by Dagger.
- Dagger module: `golang:1.26-bookworm`, golangci-lint v2.12.2 (pinned SHA256),
  ES 9.4.3 + OpenSearch 3.4.0 services, dual `ELASTICSEARCH_URL` /
  `OPENSEARCH_URL` bindings.
- Frontend vitest specs for `BackendBadge` and `useCluster`.
