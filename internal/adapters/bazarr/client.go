// Package bazarr provides the Bazarr API client for runtime configuration.
// Bazarr uses a hybrid approach: file-based config.yaml for initial setup
// and REST API for runtime configuration of language profiles and providers.
package bazarr

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client provides access to the Bazarr API
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a new Bazarr API client
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewClientWithHTTP creates a new client with a custom HTTP client
func NewClientWithHTTP(baseURL, apiKey string, httpClient *http.Client) *Client {
	return &Client{
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		apiKey:     apiKey,
		httpClient: httpClient,
	}
}

// SystemStatus represents the Bazarr system status response
type SystemStatus struct {
	Data struct {
		BazarrVersion string `json:"bazarr_version"`
		SonarrVersion string `json:"sonarr_version"`
		RadarrVersion string `json:"radarr_version"`
		OSVersion     string `json:"operating_system"`
		PythonVersion string `json:"python_version"`
		StartTime     string `json:"start_time"`
	} `json:"data"`
}

// HealthStatus represents the Bazarr health check response
type HealthStatus struct {
	Data struct {
		Sonarr *ConnectionStatus `json:"sonarr"`
		Radarr *ConnectionStatus `json:"radarr"`
		Bazarr *BazarrHealth     `json:"bazarr"`
	} `json:"data"`
}

// ConnectionStatus represents connection status to Sonarr/Radarr
type ConnectionStatus struct {
	Status  string `json:"status"`
	Version string `json:"version,omitempty"`
}

// BazarrHealth represents Bazarr internal health
type BazarrHealth struct {
	DiskSpace string `json:"disk_space"`
	Status    string `json:"status"`
}

// LanguageProfile represents a Bazarr language profile from the API
type LanguageProfile struct {
	ProfileID int                   `json:"profileId"`
	Name      string                `json:"name"`
	Cutoff    *int                  `json:"cutoff"`
	Items     []LanguageProfileItem `json:"items"`
}

// LanguageProfileItem represents a language within a profile
type LanguageProfileItem struct {
	ID           int    `json:"id"`
	Language     string `json:"language"`
	Forced       string `json:"forced"` // "True" or "False" (Bazarr quirk)
	HI           string `json:"hi"`     // "True" or "False"
	AudioExclude string `json:"audio_exclude"`
}

// Language represents an available language in Bazarr
type Language struct {
	Code2 string `json:"code2"` // ISO 639-1 code
	Code3 string `json:"code3"` // ISO 639-2 code
	Name  string `json:"name"`
}

// Provider represents a subtitle provider configuration
type Provider struct {
	Name     string                 `json:"name"`
	Enabled  bool                   `json:"enabled"`
	Settings map[string]interface{} `json:"settings,omitempty"`
}

// ProviderList represents the provider list response
type ProviderList struct {
	Data []Provider `json:"data"`
}

// SystemSettings represents Bazarr system settings
type SystemSettings struct {
	General    map[string]interface{} `json:"general"`
	Sonarr     map[string]interface{} `json:"sonarr"`
	Radarr     map[string]interface{} `json:"radarr"`
	Auth       map[string]interface{} `json:"auth"`
	Proxy      map[string]interface{} `json:"proxy"`
	Subsync    map[string]interface{} `json:"subsync"`
	Analytics  map[string]interface{} `json:"analytics"`
	Betaseries map[string]interface{} `json:"betaseries"`
}

// doRequest performs an HTTP request with authentication
func (c *Client) doRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	reqURL := c.baseURL + path

	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-API-KEY", c.apiKey)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.httpClient.Do(req)
}

// get performs a GET request and decodes the response
func (c *Client) get(ctx context.Context, path string, result interface{}) error {
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

// post performs a POST request
func (c *Client) post(ctx context.Context, path string, body interface{}, result interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = strings.NewReader(string(data))
	}

	resp, err := c.doRequest(ctx, http.MethodPost, path, bodyReader)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

// patch performs a PATCH request
func (c *Client) patch(ctx context.Context, path string, body interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = strings.NewReader(string(data))
	}

	resp, err := c.doRequest(ctx, http.MethodPatch, path, bodyReader)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// delete performs a DELETE request
func (c *Client) delete(ctx context.Context, path string) error {
	resp, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// GetSystemStatus retrieves the Bazarr system status
func (c *Client) GetSystemStatus(ctx context.Context) (*SystemStatus, error) {
	var status SystemStatus
	if err := c.get(ctx, "/api/system/status", &status); err != nil {
		return nil, fmt.Errorf("failed to get system status: %w", err)
	}
	return &status, nil
}

// GetHealth retrieves the Bazarr health status
func (c *Client) GetHealth(ctx context.Context) (*HealthStatus, error) {
	var health HealthStatus
	if err := c.get(ctx, "/api/system/health", &health); err != nil {
		return nil, fmt.Errorf("failed to get health status: %w", err)
	}
	return &health, nil
}

// GetLanguages retrieves available languages
func (c *Client) GetLanguages(ctx context.Context) ([]Language, error) {
	var response struct {
		Data []Language `json:"data"`
	}
	if err := c.get(ctx, "/api/system/languages", &response); err != nil {
		return nil, fmt.Errorf("failed to get languages: %w", err)
	}
	return response.Data, nil
}

// GetLanguageProfiles retrieves all language profiles
func (c *Client) GetLanguageProfiles(ctx context.Context) ([]LanguageProfile, error) {
	var response struct {
		Data []LanguageProfile `json:"data"`
	}
	if err := c.get(ctx, "/api/system/languages/profiles", &response); err != nil {
		return nil, fmt.Errorf("failed to get language profiles: %w", err)
	}
	return response.Data, nil
}

// CreateLanguageProfile creates a new language profile
func (c *Client) CreateLanguageProfile(ctx context.Context, profile LanguageProfile) (*LanguageProfile, error) {
	var response struct {
		Data LanguageProfile `json:"data"`
	}
	if err := c.post(ctx, "/api/system/languages/profiles", profile, &response); err != nil {
		return nil, fmt.Errorf("failed to create language profile: %w", err)
	}
	return &response.Data, nil
}

// UpdateLanguageProfile updates an existing language profile
func (c *Client) UpdateLanguageProfile(ctx context.Context, profile LanguageProfile) error {
	path := fmt.Sprintf("/api/system/languages/profiles/%d", profile.ProfileID)
	if err := c.patch(ctx, path, profile); err != nil {
		return fmt.Errorf("failed to update language profile: %w", err)
	}
	return nil
}

// DeleteLanguageProfile deletes a language profile
func (c *Client) DeleteLanguageProfile(ctx context.Context, profileID int) error {
	path := fmt.Sprintf("/api/system/languages/profiles/%d", profileID)
	if err := c.delete(ctx, path); err != nil {
		return fmt.Errorf("failed to delete language profile: %w", err)
	}
	return nil
}

// GetProviders retrieves all subtitle providers
func (c *Client) GetProviders(ctx context.Context) ([]Provider, error) {
	var response ProviderList
	if err := c.get(ctx, "/api/providers", &response); err != nil {
		return nil, fmt.Errorf("failed to get providers: %w", err)
	}
	return response.Data, nil
}

// UpdateProviders updates provider settings
func (c *Client) UpdateProviders(ctx context.Context, providers []Provider) error {
	if err := c.post(ctx, "/api/providers", providers, nil); err != nil {
		return fmt.Errorf("failed to update providers: %w", err)
	}
	return nil
}

// GetSettings retrieves all system settings
func (c *Client) GetSettings(ctx context.Context) (*SystemSettings, error) {
	var response struct {
		Data SystemSettings `json:"data"`
	}
	if err := c.get(ctx, "/api/system/settings", &response); err != nil {
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}
	return &response.Data, nil
}

// UpdateSettings updates system settings
func (c *Client) UpdateSettings(ctx context.Context, settings map[string]interface{}) error {
	if err := c.post(ctx, "/api/system/settings", settings, nil); err != nil {
		return fmt.Errorf("failed to update settings: %w", err)
	}
	return nil
}

// TestConnection tests the connection to Bazarr
func (c *Client) TestConnection(ctx context.Context) error {
	_, err := c.GetSystemStatus(ctx)
	return err
}

// GetVersion retrieves just the Bazarr version
func (c *Client) GetVersion(ctx context.Context) (string, error) {
	status, err := c.GetSystemStatus(ctx)
	if err != nil {
		return "", err
	}
	return status.Data.BazarrVersion, nil
}

// SearchSubtitles triggers a subtitle search for a specific item
// itemType is "episode" or "movie"
func (c *Client) SearchSubtitles(ctx context.Context, itemType string, itemID int, language string) error {
	params := url.Values{}
	params.Set("action", "search")
	params.Set(itemType+"id", fmt.Sprintf("%d", itemID))
	params.Set("language", language)

	path := fmt.Sprintf("/api/subtitles?%s", params.Encode())
	if err := c.post(ctx, path, nil, nil); err != nil {
		return fmt.Errorf("failed to search subtitles: %w", err)
	}
	return nil
}
