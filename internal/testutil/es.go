//go:build acceptance

// Package testutil provides helpers for acceptance tests that run against real
// Elasticsearch and OpenSearch clusters provisioned by Dagger.
package testutil

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"cerebro/internal/elastic"
)

// ESURL returns the Elasticsearch endpoint (env ELASTICSEARCH_URL, default
// "http://localhost:9200").
func ESURL() string {
	if u := os.Getenv("ELASTICSEARCH_URL"); u != "" {
		return u
	}
	return "http://localhost:9200"
}

// OpenSearchURL returns the OpenSearch endpoint (env OPENSEARCH_URL, default
// "http://localhost:9201").
func OpenSearchURL() string {
	if u := os.Getenv("OPENSEARCH_URL"); u != "" {
		return u
	}
	return "http://localhost:9201"
}

// WaitForEndpoint polls GET <url>/ every 2s until it returns 200 or timeout;
// returns an error containing the last failure detail on timeout.
func WaitForEndpoint(ctx context.Context, url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 5 * time.Second}
	var lastErr string
	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url+"/", nil)
		if err != nil {
			return err
		}
		req.Header.Set("Accept", "application/json")
		res, err := client.Do(req)
		if err == nil {
			body, _ := io.ReadAll(res.Body)
			res.Body.Close()
			if res.StatusCode == 200 {
				return nil
			}
			lastErr = fmt.Sprintf("status %d: %s", res.StatusCode, string(body))
		} else {
			lastErr = err.Error()
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("endpoint %s not ready after %s: %s", url, timeout, lastErr)
		}
		time.Sleep(2 * time.Second)
	}
}

// NewAcceptanceClient builds the production dual-client router (gzip off).
func NewAcceptanceClient() (elastic.ElasticClient, error) {
	return elastic.NewMultiClient(false)
}
