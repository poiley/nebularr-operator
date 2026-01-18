package lidarr

import (
	"context"
	"fmt"

	"github.com/poiley/nebularr-operator/internal/adapters/httpclient"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

// HostConfigResource represents Lidarr host configuration (includes authentication)
type HostConfigResource struct {
	ID                        int    `json:"id,omitempty"`
	BindAddress               string `json:"bindAddress"`
	Port                      int    `json:"port"`
	SslPort                   int    `json:"sslPort"`
	EnableSsl                 bool   `json:"enableSsl"`
	LaunchBrowser             bool   `json:"launchBrowser"`
	AuthenticationMethod      string `json:"authenticationMethod"`   // none, forms, basic, external
	AuthenticationRequired    string `json:"authenticationRequired"` // enabled, disabledForLocalAddresses
	Username                  string `json:"username,omitempty"`
	Password                  string `json:"password,omitempty"`
	PasswordConfirmation      string `json:"passwordConfirmation,omitempty"`
	LogLevel                  string `json:"logLevel"`
	ConsoleLogLevel           string `json:"consoleLogLevel"`
	Branch                    string `json:"branch"`
	ApiKey                    string `json:"apiKey"`
	SslCertPath               string `json:"sslCertPath"`
	SslCertPassword           string `json:"sslCertPassword"`
	UrlBase                   string `json:"urlBase"`
	InstanceName              string `json:"instanceName"`
	UpdateAutomatically       bool   `json:"updateAutomatically"`
	UpdateMechanism           string `json:"updateMechanism"`
	UpdateScriptPath          string `json:"updateScriptPath"`
	ProxyEnabled              bool   `json:"proxyEnabled"`
	ProxyType                 string `json:"proxyType"`
	ProxyHostname             string `json:"proxyHostname"`
	ProxyPort                 int    `json:"proxyPort"`
	ProxyUsername             string `json:"proxyUsername"`
	ProxyPassword             string `json:"proxyPassword"`
	ProxyBypassFilter         string `json:"proxyBypassFilter"`
	ProxyBypassLocalAddresses bool   `json:"proxyBypassLocalAddresses"`
	CertificateValidation     string `json:"certificateValidation"`
	BackupFolder              string `json:"backupFolder"`
	BackupInterval            int    `json:"backupInterval"`
	BackupRetention           int    `json:"backupRetention"`
}

// getHostConfig fetches the current host configuration (includes authentication)
func (a *Adapter) getHostConfig(ctx context.Context, c *httpclient.Client) (*HostConfigResource, error) {
	var config HostConfigResource
	if err := c.Get(ctx, "/api/v1/config/host", &config); err != nil {
		return nil, fmt.Errorf("failed to get host config: %w", err)
	}
	return &config, nil
}

// updateHostConfig updates the host configuration
func (a *Adapter) updateHostConfig(ctx context.Context, c *httpclient.Client, config HostConfigResource) error {
	path := fmt.Sprintf("/api/v1/config/host/%d", config.ID)
	var result HostConfigResource
	return c.Put(ctx, path, config, &result)
}

// applyAuthentication applies authentication configuration from IR
func (a *Adapter) applyAuthentication(ctx context.Context, c *httpclient.Client, ir *irv1.AuthenticationIR) error {
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

	// Map method string to Lidarr API values
	updated.AuthenticationMethod = mapAuthMethod(ir.Method)
	updated.AuthenticationRequired = mapAuthRequired(ir.AuthenticationRequired)

	// Set username if using forms auth
	if ir.Method == "forms" && ir.Username != "" {
		updated.Username = ir.Username
	}

	// Only set password if provided (for initial setup)
	if ir.Password != "" {
		updated.Password = ir.Password
		updated.PasswordConfirmation = ir.Password
	}

	return a.updateHostConfig(ctx, c, updated)
}

// getAuthenticationIR converts the current config to IR format
func (a *Adapter) getAuthenticationIR(ctx context.Context, c *httpclient.Client) (*irv1.AuthenticationIR, error) {
	config, err := a.getHostConfig(ctx, c)
	if err != nil {
		return nil, err
	}

	return a.hostConfigToAuthIR(config), nil
}

// hostConfigToAuthIR converts a HostConfigResource to AuthenticationIR
func (a *Adapter) hostConfigToAuthIR(config *HostConfigResource) *irv1.AuthenticationIR {
	ir := &irv1.AuthenticationIR{
		Username: config.Username,
		// Password is not returned by the API for security reasons
	}

	// Map authentication method back to our string
	switch config.AuthenticationMethod {
	case "None":
		ir.Method = "none"
	case "Forms":
		ir.Method = "forms"
	case "Basic":
		ir.Method = "basic"
	case "External":
		ir.Method = "external"
	default:
		ir.Method = config.AuthenticationMethod
	}

	// Map authentication required back to our string
	switch config.AuthenticationRequired {
	case "Enabled":
		ir.AuthenticationRequired = "enabled"
	case "DisabledForLocalAddresses":
		ir.AuthenticationRequired = "disabledForLocalAddresses"
	default:
		ir.AuthenticationRequired = config.AuthenticationRequired
	}

	return ir
}

// mapAuthMethod maps our method string to Lidarr API values
func mapAuthMethod(method string) string {
	switch method {
	case "none":
		return "None"
	case "forms":
		return "Forms"
	case "basic":
		return "Basic"
	case "external":
		return "External"
	default:
		return "None"
	}
}

// mapAuthRequired maps our authenticationRequired string to Lidarr API values
func mapAuthRequired(required string) string {
	switch required {
	case "enabled":
		return "Enabled"
	case "disabledForLocalAddresses":
		return "DisabledForLocalAddresses"
	default:
		return "Enabled"
	}
}
