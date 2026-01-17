package radarr

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/poiley/nebularr-operator/internal/adapters/radarr/client"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

// getHostConfig fetches the current host configuration (includes authentication)
func (a *Adapter) getHostConfig(ctx context.Context, c *client.Client) (*client.HostConfigResource, error) {
	resp, err := c.GetApiV3ConfigHost(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get host config: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var config client.HostConfigResource
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode host config: %w", err)
	}

	return &config, nil
}

// updateHostConfig updates the host configuration
func (a *Adapter) updateHostConfig(ctx context.Context, c *client.Client, config client.HostConfigResource) error {
	if config.Id == nil {
		return fmt.Errorf("host config ID is required")
	}

	idStr := fmt.Sprintf("%d", *config.Id)
	resp, err := c.PutApiV3ConfigHostId(ctx, idStr, config)
	if err != nil {
		return fmt.Errorf("failed to update host config: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// applyAuthentication applies authentication configuration from IR
func (a *Adapter) applyAuthentication(ctx context.Context, c *client.Client, ir *irv1.AuthenticationIR) error {
	if ir == nil {
		return nil
	}

	// Get current config to preserve ID and unmanaged fields
	current, err := a.getHostConfig(ctx, c)
	if err != nil {
		return fmt.Errorf("failed to get current host config: %w", err)
	}

	// Update only authentication-related fields
	updated := *current

	// Map method string to AuthenticationType
	authMethod := mapAuthMethod(ir.Method)
	updated.AuthenticationMethod = &authMethod

	// Map authenticationRequired string to AuthenticationRequiredType
	authRequired := mapAuthRequired(ir.AuthenticationRequired)
	updated.AuthenticationRequired = &authRequired

	// Set username if using forms auth
	if ir.Method == "forms" && ir.Username != "" {
		updated.Username = stringPtr(ir.Username)
	}

	// Only set password if provided (for initial setup)
	// NOTE: Password changes require special handling - the API may require
	// both password and passwordConfirmation
	if ir.Password != "" {
		updated.Password = stringPtr(ir.Password)
		updated.PasswordConfirmation = stringPtr(ir.Password)
	}

	return a.updateHostConfig(ctx, c, updated)
}

// getAuthenticationIR converts the current config to IR format
func (a *Adapter) getAuthenticationIR(ctx context.Context, c *client.Client) (*irv1.AuthenticationIR, error) {
	config, err := a.getHostConfig(ctx, c)
	if err != nil {
		return nil, err
	}

	return a.hostConfigToAuthIR(config), nil
}

// hostConfigToAuthIR converts a client HostConfigResource to AuthenticationIR
func (a *Adapter) hostConfigToAuthIR(config *client.HostConfigResource) *irv1.AuthenticationIR {
	ir := &irv1.AuthenticationIR{
		Username: ptrToString(config.Username),
		// Password is not returned by the API for security reasons
	}

	// Map AuthenticationType back to our method string
	if config.AuthenticationMethod != nil {
		switch *config.AuthenticationMethod {
		case client.AuthenticationTypeNone:
			ir.Method = "none"
		case client.AuthenticationTypeForms:
			ir.Method = "forms"
		case client.AuthenticationTypeBasic:
			ir.Method = "basic"
		case client.AuthenticationTypeExternal:
			ir.Method = "external"
		default:
			ir.Method = string(*config.AuthenticationMethod)
		}
	}

	// Map AuthenticationRequiredType back to our string
	if config.AuthenticationRequired != nil {
		switch *config.AuthenticationRequired {
		case client.AuthenticationRequiredTypeEnabled:
			ir.AuthenticationRequired = "enabled"
		case client.AuthenticationRequiredTypeDisabledForLocalAddresses:
			ir.AuthenticationRequired = "disabledForLocalAddresses"
		default:
			ir.AuthenticationRequired = string(*config.AuthenticationRequired)
		}
	}

	return ir
}

// mapAuthMethod maps our method string to client.AuthenticationType
func mapAuthMethod(method string) client.AuthenticationType {
	switch method {
	case "none":
		return client.AuthenticationTypeNone
	case "forms":
		return client.AuthenticationTypeForms
	case "basic":
		return client.AuthenticationTypeBasic
	case "external":
		return client.AuthenticationTypeExternal
	default:
		return client.AuthenticationTypeNone
	}
}

// mapAuthRequired maps our authenticationRequired string to client.AuthenticationRequiredType
func mapAuthRequired(required string) client.AuthenticationRequiredType {
	switch required {
	case "enabled":
		return client.AuthenticationRequiredTypeEnabled
	case "disabledForLocalAddresses":
		return client.AuthenticationRequiredTypeDisabledForLocalAddresses
	default:
		return client.AuthenticationRequiredTypeEnabled
	}
}
