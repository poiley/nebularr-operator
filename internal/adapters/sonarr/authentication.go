package sonarr

import (
	"context"

	"github.com/poiley/nebularr-operator/internal/adapters/httpclient"
	"github.com/poiley/nebularr-operator/internal/adapters/shared"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

const hostConfigAPIPath = "/api/v3/config/host"

// applyAuthentication applies authentication configuration from IR
func (a *Adapter) applyAuthentication(ctx context.Context, c *httpclient.Client, ir *irv1.AuthenticationIR) error {
	return shared.ApplyAuthentication(ctx, c, hostConfigAPIPath, ir)
}

// getAuthenticationIR converts the current config to IR format
func (a *Adapter) getAuthenticationIR(ctx context.Context, c *httpclient.Client) (*irv1.AuthenticationIR, error) {
	return shared.GetAuthenticationIR(ctx, c, hostConfigAPIPath)
}
