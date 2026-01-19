// Package shared provides common functionality used across multiple *arr adapters.
package shared

import (
	"context"
	"fmt"

	"github.com/poiley/nebularr-operator/internal/adapters/httpclient"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

// HostConfigResource represents host configuration (includes authentication)
// This struct is identical across Sonarr, Lidarr, and Radarr.
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

// GetAuthenticationIR fetches and converts host config to AuthenticationIR.
func GetAuthenticationIR(ctx context.Context, c *httpclient.Client, apiPath string) (*irv1.AuthenticationIR, error) {
	config, err := FetchConfig[HostConfigResource](ctx, c, apiPath)
	if err != nil {
		return nil, err
	}
	return HostConfigToAuthIR(config), nil
}

// ApplyAuthentication applies authentication configuration from IR.
func ApplyAuthentication(ctx context.Context, c *httpclient.Client, apiPath string, ir *irv1.AuthenticationIR) error {
	if ir == nil {
		return nil
	}

	// Get current config to preserve ID and unmanaged fields
	current, err := FetchConfig[HostConfigResource](ctx, c, apiPath)
	if err != nil {
		return fmt.Errorf("failed to get current host config: %w", err)
	}

	// Update only authentication-related fields
	current.AuthenticationMethod = MapAuthMethod(ir.Method)
	current.AuthenticationRequired = MapAuthRequired(ir.AuthenticationRequired)

	// Set username if using forms auth
	if ir.Method == "forms" && ir.Username != "" {
		current.Username = ir.Username
	}

	// Only set password if provided (for initial setup)
	if ir.Password != "" {
		current.Password = ir.Password
		current.PasswordConfirmation = ir.Password
	}

	return UpdateConfig(ctx, c, apiPath, current.ID, *current)
}

// HostConfigToAuthIR converts a HostConfigResource to AuthenticationIR.
func HostConfigToAuthIR(config *HostConfigResource) *irv1.AuthenticationIR {
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

// MapAuthMethod maps IR method string to API values.
func MapAuthMethod(method string) string {
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

// MapAuthRequired maps IR authenticationRequired string to API values.
func MapAuthRequired(required string) string {
	switch required {
	case "enabled":
		return "Enabled"
	case "disabledForLocalAddresses":
		return "DisabledForLocalAddresses"
	default:
		return "Enabled"
	}
}
