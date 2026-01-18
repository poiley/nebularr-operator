// Package shared provides common functionality used across multiple *arr adapters.
package shared

import (
	"context"
	"fmt"

	"github.com/poiley/nebularr-operator/internal/adapters/httpclient"
)

// FetchConfig is a generic helper to fetch any config type from an API endpoint.
func FetchConfig[T any](ctx context.Context, c *httpclient.Client, apiPath string) (*T, error) {
	var config T
	if err := c.Get(ctx, apiPath, &config); err != nil {
		return nil, fmt.Errorf("failed to get config from %s: %w", apiPath, err)
	}
	return &config, nil
}

// UpdateConfig is a generic helper to update any config type via PUT.
func UpdateConfig[T any](ctx context.Context, c *httpclient.Client, apiPath string, id int, config T) error {
	path := fmt.Sprintf("%s/%d", apiPath, id)
	var result T
	return c.Put(ctx, path, config, &result)
}
