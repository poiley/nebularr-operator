// Package prowlarr provides services for Prowlarr integration.
// This includes auto-registration of *arr apps with Prowlarr.
package prowlarr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

// RegistrationService handles auto-registration of *arr apps with Prowlarr
type RegistrationService struct {
	httpClient *http.Client
}

// NewRegistrationService creates a new RegistrationService
func NewRegistrationService() *RegistrationService {
	return &RegistrationService{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// AppRegistration contains the info needed to register an app with Prowlarr
type AppRegistration struct {
	// Name is the display name for the app in Prowlarr
	Name string

	// Type is the app type: radarr, sonarr, lidarr
	Type string

	// URL is the app's URL (how Prowlarr reaches the app)
	URL string

	// APIKey is the app's API key
	APIKey string

	// ProwlarrURL is the URL Prowlarr uses for itself (how the app reaches Prowlarr)
	// If empty, defaults to Prowlarr's own URL
	ProwlarrURL string

	// SyncLevel: disabled, addOnly, fullSync
	SyncLevel string

	// SyncCategories to sync (Newznab category IDs)
	// If empty, uses defaults based on app type
	SyncCategories []int

	// Tags to filter which indexers sync to this app
	Tags []string
}

// ProwlarrConnection contains connection info for Prowlarr
type ProwlarrConnection struct {
	URL    string
	APIKey string
}

// Register registers an *arr app with Prowlarr
// If the app already exists (by name), it updates it
func (s *RegistrationService) Register(ctx context.Context, prowlarr ProwlarrConnection, app AppRegistration) error {
	// Check if app already exists
	existingID, err := s.findApplicationByName(ctx, prowlarr, app.Name)
	if err != nil {
		return fmt.Errorf("failed to check for existing application: %w", err)
	}

	// Build the application resource
	resource := s.buildApplicationResource(app, prowlarr.URL)

	if existingID > 0 {
		// Update existing
		resource.ID = existingID
		return s.updateApplication(ctx, prowlarr, resource)
	}

	// Create new
	return s.createApplication(ctx, prowlarr, resource)
}

// Unregister removes an *arr app from Prowlarr
func (s *RegistrationService) Unregister(ctx context.Context, prowlarr ProwlarrConnection, appName string) error {
	id, err := s.findApplicationByName(ctx, prowlarr, appName)
	if err != nil {
		return fmt.Errorf("failed to find application: %w", err)
	}

	if id == 0 {
		// Already doesn't exist
		return nil
	}

	return s.deleteApplication(ctx, prowlarr, id)
}

// ApplicationResource represents a Prowlarr application
type ApplicationResource struct {
	ID             int                `json:"id,omitempty"`
	Name           string             `json:"name"`
	Implementation string             `json:"implementation"`
	ConfigContract string             `json:"configContract,omitempty"`
	SyncLevel      string             `json:"syncLevel"`
	Tags           []int              `json:"tags,omitempty"`
	Fields         []ApplicationField `json:"fields,omitempty"`
}

// ApplicationField represents a configuration field
type ApplicationField struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
}

// buildApplicationResource creates the Prowlarr API resource
func (s *RegistrationService) buildApplicationResource(app AppRegistration, prowlarrURL string) ApplicationResource {
	// Determine implementation and config contract based on app type
	impl, contract := s.getImplementationForType(app.Type)

	syncLevel := app.SyncLevel
	if syncLevel == "" {
		syncLevel = irv1.SyncLevelFullSync
	}

	// Build fields
	fields := []ApplicationField{
		{Name: "baseUrl", Value: app.URL},
		{Name: "apiKey", Value: app.APIKey},
	}

	// Add Prowlarr URL (how the app reaches Prowlarr)
	prowlarrSelfURL := app.ProwlarrURL
	if prowlarrSelfURL == "" {
		prowlarrSelfURL = prowlarrURL
	}
	fields = append(fields, ApplicationField{Name: "prowlarrUrl", Value: prowlarrSelfURL})

	// Add sync categories
	categories := app.SyncCategories
	if len(categories) == 0 {
		if defaults, ok := irv1.DefaultSyncCategories[app.Type]; ok {
			categories = defaults
		}
	}
	if len(categories) > 0 {
		fields = append(fields, ApplicationField{Name: "syncCategories", Value: categories})
	}

	return ApplicationResource{
		Name:           app.Name,
		Implementation: impl,
		ConfigContract: contract,
		SyncLevel:      syncLevel,
		Fields:         fields,
	}
}

// getImplementationForType returns the Prowlarr implementation name and config contract
func (s *RegistrationService) getImplementationForType(appType string) (string, string) {
	switch appType {
	case irv1.AppTypeRadarr:
		return "Radarr", "RadarrSettings"
	case irv1.AppTypeSonarr:
		return "Sonarr", "SonarrSettings"
	case irv1.AppTypeLidarr:
		return "Lidarr", "LidarrSettings"
	default:
		return appType, ""
	}
}

// findApplicationByName looks up an application by name
func (s *RegistrationService) findApplicationByName(ctx context.Context, prowlarr ProwlarrConnection, name string) (int, error) {
	var apps []ApplicationResource
	if err := s.get(ctx, prowlarr, "/api/v1/applications", &apps); err != nil {
		return 0, err
	}

	for _, app := range apps {
		if app.Name == name {
			return app.ID, nil
		}
	}

	return 0, nil
}

// createApplication creates an application in Prowlarr
func (s *RegistrationService) createApplication(ctx context.Context, prowlarr ProwlarrConnection, app ApplicationResource) error {
	var created ApplicationResource
	return s.post(ctx, prowlarr, "/api/v1/applications", app, &created)
}

// updateApplication updates an application in Prowlarr
func (s *RegistrationService) updateApplication(ctx context.Context, prowlarr ProwlarrConnection, app ApplicationResource) error {
	path := fmt.Sprintf("/api/v1/applications/%d", app.ID)
	return s.put(ctx, prowlarr, path, app, nil)
}

// deleteApplication deletes an application from Prowlarr
func (s *RegistrationService) deleteApplication(ctx context.Context, prowlarr ProwlarrConnection, id int) error {
	path := fmt.Sprintf("/api/v1/applications/%d", id)
	return s.delete(ctx, prowlarr, path)
}

// HTTP helpers

func (s *RegistrationService) get(ctx context.Context, prowlarr ProwlarrConnection, path string, result interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, prowlarr.URL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Api-Key", prowlarr.APIKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

func (s *RegistrationService) post(ctx context.Context, prowlarr ProwlarrConnection, path string, body, result interface{}) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, prowlarr.URL+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("X-Api-Key", prowlarr.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

func (s *RegistrationService) put(ctx context.Context, prowlarr ProwlarrConnection, path string, body, result interface{}) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, prowlarr.URL+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("X-Api-Key", prowlarr.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

func (s *RegistrationService) delete(ctx context.Context, prowlarr ProwlarrConnection, path string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, prowlarr.URL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Api-Key", prowlarr.APIKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
