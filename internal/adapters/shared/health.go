// Package shared provides common functionality used across multiple *arr adapters.
package shared

import (
	"context"
	"fmt"

	"github.com/poiley/nebularr-operator/internal/adapters/httpclient"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

// GetHealth fetches the current health status from an *arr service.
// apiVersion should be "v1" or "v3" depending on the service.
func GetHealth(ctx context.Context, c *httpclient.Client, apiVersion string) (*irv1.HealthStatus, error) {
	var healthChecks []HealthResource
	endpoint := fmt.Sprintf("/api/%s/health", apiVersion)
	if err := c.Get(ctx, endpoint, &healthChecks); err != nil {
		return nil, fmt.Errorf("failed to get health: %w", err)
	}

	status := &irv1.HealthStatus{
		Healthy: true,
		Issues:  make([]irv1.HealthIssue, 0, len(healthChecks)),
	}

	for _, check := range healthChecks {
		issueType := irv1.HealthIssueTypeNotice
		switch check.Type {
		case "error":
			issueType = irv1.HealthIssueTypeError
			status.Healthy = false
		case "warning":
			issueType = irv1.HealthIssueTypeWarning
		}

		status.Issues = append(status.Issues, irv1.HealthIssue{
			Source:  check.Source,
			Type:    issueType,
			Message: check.Message,
			WikiURL: check.WikiURL,
		})
	}

	return status, nil
}
