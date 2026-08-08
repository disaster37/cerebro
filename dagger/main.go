package main

import (
	"context"
	"fmt"

	"dagger/cerebro/internal/dagger"
)

// Image and path constants used across pipeline functions.
const (
	goImage   = "golang:1.26-bookworm" // disaster37/opensearch/v4 requires go 1.26
	nodeImage = "node:20-bookworm"
	esImage   = "docker.elastic.co/elasticsearch/elasticsearch:9.4.3"
	osImage   = "opensearchproject/opensearch:3.4.0" // same tag the v4 client's CI uses

	srcPath = "/src"
)

type Cerebro struct{}

// Lint runs go vet + golangci-lint on the Go backend and eslint on the frontend.
func (c *Cerebro) Lint(ctx context.Context, src *dagger.Directory) (string, error) {
	// golangci-lint is installed from the official release tarball with a pinned
	// SHA256 checksum verified after download, avoiding the supply-chain risk of
	// `go install` fetching arbitrary source at runtime. The tarball is extracted
	// to a non-root writable location (/tmp) and the binary is invoked from there.
	goLint := c.goBase(src).
		WithEnvVariable("HOME", "/tmp").
		WithExec([]string{"go", "vet", "./..."}).
		WithExec([]string{"bash", "-c", "if ! command -v golangci-lint >/dev/null 2>&1 && [ ! -x /tmp/golangci-lint/golangci-lint ]; then curl -fsSL -o /tmp/golangci-lint.tar.gz https://github.com/golangci/golangci-lint/releases/download/v2.12.2/golangci-lint-2.12.2-linux-amd64.tar.gz && echo '8df580d2670fed8fa984aac0507099af8df275e665215f5c7a2ae3943893a553  /tmp/golangci-lint.tar.gz' | sha256sum -c - && mkdir -p /tmp/golangci-lint && tar -xzf /tmp/golangci-lint.tar.gz -C /tmp/golangci-lint --strip-components=1 && rm /tmp/golangci-lint.tar.gz; fi"}).
		WithExec([]string{"bash", "-c", "golangci-lint run || /tmp/golangci-lint/golangci-lint run"})

	npmLint := c.nodeBase(src).
		WithExec([]string{"npm", "ci", "--legacy-peer-deps"}).
		WithExec([]string{"npm", "run", "lint"})

	goOut, err := goLint.Stdout(ctx)
	if err != nil {
		return goOut, fmt.Errorf("go lint: %w", err)
	}
	npmOut, err := npmLint.Stdout(ctx)
	if err != nil {
		return npmOut, fmt.Errorf("npm lint: %w", err)
	}
	return goOut + "\n" + npmOut, nil
}

// UnitTest runs `go test ./...` excluding acceptance tests (build tag `acceptance`).
func (c *Cerebro) UnitTest(ctx context.Context, src *dagger.Directory) (string, error) {
	return c.goBase(src).
		WithExec([]string{"go", "test", "-race", "-count=1", "./..."}).
		Stdout(ctx)
}

// AcceptanceTest runs the `acceptance`-tagged Go tests against Dagger-provisioned
// Elasticsearch 9 and OpenSearch 3 services. URLs are injected via
// ELASTICSEARCH_URL and OPENSEARCH_URL.
func (c *Cerebro) AcceptanceTest(ctx context.Context, src *dagger.Directory) (string, error) {
	result, err := c.goBase(src).
		WithServiceBinding("es", c.elasticsearchService()).
		WithServiceBinding("opensearch", c.opensearchService()).
		WithEnvVariable("ELASTICSEARCH_URL", "http://es:9200").
		WithEnvVariable("OPENSEARCH_URL", "http://opensearch:9200").
		WithExec([]string{"go", "test", "-race", "-count=1", "-tags=acceptance", "./..."}).
		Stdout(ctx)
	if err != nil {
		return result, fmt.Errorf("acceptance test: %w "+
			"(check that BOTH services are healthy — TestMain polls "+
			"ELASTICSEARCH_URL and OPENSEARCH_URL until they respond)", err)
	}
	return result, nil
}

// FrontendTest runs `npm run test` (vitest) on the frontend.
func (c *Cerebro) FrontendTest(ctx context.Context, src *dagger.Directory) (string, error) {
	return c.nodeBase(src).
		WithExec([]string{"npm", "ci", "--legacy-peer-deps"}).
		WithExec([]string{"npm", "run", "test"}).
		Stdout(ctx)
}

// Build compiles the Go binary and the Nuxt static output, returns the dist Directory.
func (c *Cerebro) Build(ctx context.Context, src *dagger.Directory) (*dagger.Directory, error) {
	goBin := c.goBase(src).
		WithExec([]string{"go", "build", "-o", "build/cerebro", "./cmd/cerebro"}).
		Directory(srcPath + "/build")

	nuxt := c.nodeBase(src).
		WithExec([]string{"npm", "ci", "--legacy-peer-deps"}).
		WithExec([]string{"npm", "run", "build"}).
		Directory(srcPath + "/output/public")

	return dag.Directory().
		WithDirectory("bin", goBin).
		WithDirectory("public", nuxt), nil
}

// All runs Lint → UnitTest → FrontendTest → AcceptanceTest → Build in order; fails fast.
func (c *Cerebro) All(ctx context.Context, src *dagger.Directory) (*dagger.Directory, error) {
	type step struct {
		name string
		fn   func(context.Context, *dagger.Directory) (string, error)
	}
	for _, s := range []step{
		{"lint", c.Lint},
		{"unit test", c.UnitTest},
		{"frontend test", c.FrontendTest},
		{"acceptance test", c.AcceptanceTest},
	} {
		if _, err := s.fn(ctx, src); err != nil {
			return nil, fmt.Errorf("%s: %w", s.name, err)
		}
	}
	return c.Build(ctx, src)
}

// goBase returns a Go container with source mounted and module/build caches.
func (c *Cerebro) goBase(src *dagger.Directory) *dagger.Container {
	return c.baseContainer(goImage, src).
		WithEnvVariable("GOMODCACHE", "/cache/go/pkg/mod").
		WithEnvVariable("GOCACHE", "/cache/go-build").
		WithMountedCache("/cache/go/pkg/mod", dag.CacheVolume("go-mod"), dagger.ContainerWithMountedCacheOpts{Owner: "nobody"}).
		WithMountedCache("/cache/go-build", dag.CacheVolume("go-build"), dagger.ContainerWithMountedCacheOpts{Owner: "nobody"})
}

// nodeBase returns a Node container with source mounted and npm cache cached.
func (c *Cerebro) nodeBase(src *dagger.Directory) *dagger.Container {
	return c.baseContainer(nodeImage, src).
		WithEnvVariable("npm_config_cache", "/cache/npm").
		WithMountedCache("/cache/npm", dag.CacheVolume("npm"), dagger.ContainerWithMountedCacheOpts{Owner: "nobody"})
}

// elasticsearchService returns an ES 9 single-node service for acceptance tests.
func (c *Cerebro) elasticsearchService() *dagger.Service {
	// Run as the non-root `elasticsearch` user. The ES image ships this user and
	// its entrypoint handles privilege setup; if a future ES image requires root,
	// remove the WithUser line below (this is an internal test-only service).
	return dag.Container().
		From(esImage).
		WithUser("elasticsearch").
		WithEnvVariable("discovery.type", "single-node").
		WithEnvVariable("xpack.security.enabled", "false").
		WithEnvVariable("ES_JAVA_OPTS", "-Xms512m -Xmx512m").
		WithExposedPort(9200).
		AsService(dagger.ContainerAsServiceOpts{UseEntrypoint: true})
}

// opensearchService returns an OpenSearch 3.x single-node service for
// acceptance tests. The security plugin is disabled so plain HTTP on 9200
// works without credentials or TLS-skip plumbing (the shared targetTransport
// does strict cert verification; cerebro's test scope does not include the
// OpenSearch security APIs). The image's default user is already "opensearch"
// (UID 1000) and the entrypoint refuses root, so no WithUser is needed —
// same pattern as the v4 client's own CI (ci/dagger/main.go), which also
// proves discovery.type=single-node avoids vm.max_map_count bootstrap issues
// in Dagger.
func (c *Cerebro) opensearchService() *dagger.Service {
	return dag.Container().
		From(osImage).
		WithEnvVariable("discovery.type", "single-node").
		WithEnvVariable("DISABLE_SECURITY_PLUGIN", "true").
		WithEnvVariable("DISABLE_PERFORMANCE_ANALYZER_AGENT_CLI", "true").
		WithEnvVariable("OPENSEARCH_JAVA_OPTS", "-Xms512M -Xmx512M").
		WithExposedPort(9200).
		AsService(dagger.ContainerAsServiceOpts{UseEntrypoint: true})
}

// baseContainer returns a base container with the source directory mounted at srcPath
// and workdir set to srcPath. Callers add their own caches.
func (c *Cerebro) baseContainer(image string, src *dagger.Directory) *dagger.Container {
	return dag.Container().
		From(image).
		WithUser("nobody").
		WithMountedDirectory(srcPath, src, dagger.ContainerWithMountedDirectoryOpts{Owner: "nobody"}).
		WithWorkdir(srcPath)
}
