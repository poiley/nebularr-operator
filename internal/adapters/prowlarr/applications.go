package prowlarr

import (
	"context"
	"fmt"

	"github.com/poiley/nebularr-operator/internal/adapters"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

// Package-level cache for application IDs
var applicationIDCache = make(map[string]int) // "baseURL:name" -> ID

// getManagedApplications retrieves applications tagged with ownership tag
func (a *Adapter) getManagedApplications(ctx context.Context, c *httpClient, tagID int) ([]irv1.ProwlarrApplicationIR, error) {
	var apps []ApplicationResource
	if err := c.get(ctx, "/api/v1/applications", &apps); err != nil {
		return nil, fmt.Errorf("failed to get applications: %w", err)
	}

	managed := make([]irv1.ProwlarrApplicationIR, 0, len(apps))
	for _, app := range apps {
		if !hasTag(app.Tags, tagID) {
			continue
		}

		// Cache the ID
		cacheKey := fmt.Sprintf("%s:%s", c.baseURL, app.Name)
		applicationIDCache[cacheKey] = app.ID

		// Convert to IR
		ir := irv1.ProwlarrApplicationIR{
			Name:      app.Name,
			Type:      implToAppType(app.Implementation),
			SyncLevel: app.SyncLevel,
		}

		// Extract settings from fields
		for _, field := range app.Fields {
			switch field.Name {
			case "prowlarrUrl":
				if v, ok := field.Value.(string); ok {
					ir.ProwlarrURL = v
				}
			case "baseUrl":
				if v, ok := field.Value.(string); ok {
					ir.URL = v
				}
			case "apiKey":
				if v, ok := field.Value.(string); ok {
					ir.APIKey = v
				}
			case "syncCategories":
				if cats, ok := field.Value.([]interface{}); ok {
					for _, cat := range cats {
						if c, ok := cat.(float64); ok {
							ir.SyncCategories = append(ir.SyncCategories, int(c))
						}
					}
				}
			}
		}

		managed = append(managed, ir)
	}

	return managed, nil
}

// implToAppType converts implementation name to IR app type
func implToAppType(impl string) string {
	switch impl {
	case AppImplRadarr:
		return irv1.AppTypeRadarr
	case AppImplSonarr:
		return irv1.AppTypeSonarr
	case AppImplLidarr:
		return irv1.AppTypeLidarr
	case AppImplReadarr:
		return irv1.AppTypeReadarr
	default:
		return impl
	}
}

// appTypeToImpl converts IR app type to implementation name
func appTypeToImpl(appType string) string {
	switch appType {
	case irv1.AppTypeRadarr:
		return AppImplRadarr
	case irv1.AppTypeSonarr:
		return AppImplSonarr
	case irv1.AppTypeLidarr:
		return AppImplLidarr
	case irv1.AppTypeReadarr:
		return AppImplReadarr
	default:
		return appType
	}
}

// diffApplications computes changes needed for applications
func (a *Adapter) diffApplications(current, desired *irv1.ProwlarrIR, changes *adapters.ChangeSet) error {
	currentByName := make(map[string]irv1.ProwlarrApplicationIR)
	for _, app := range current.Applications {
		currentByName[app.Name] = app
	}

	desiredByName := make(map[string]irv1.ProwlarrApplicationIR)
	for _, app := range desired.Applications {
		desiredByName[app.Name] = app
	}

	// Find creates and updates
	for name, desiredApp := range desiredByName {
		currentApp, exists := currentByName[name]
		if !exists {
			// Create
			changes.Creates = append(changes.Creates, adapters.Change{
				ResourceType: adapters.ResourceApplication,
				Name:         name,
				Payload:      desiredApp,
			})
		} else if !applicationsEqual(currentApp, desiredApp) {
			// Update
			changes.Updates = append(changes.Updates, adapters.Change{
				ResourceType: adapters.ResourceApplication,
				Name:         name,
				Payload:      desiredApp,
			})
		}
	}

	// Find deletes
	for name := range currentByName {
		if _, exists := desiredByName[name]; !exists {
			changes.Deletes = append(changes.Deletes, adapters.Change{
				ResourceType: adapters.ResourceApplication,
				Name:         name,
			})
		}
	}

	return nil
}

// applicationsEqual compares two applications for equality
func applicationsEqual(a, b irv1.ProwlarrApplicationIR) bool {
	if a.Type != b.Type ||
		a.URL != b.URL ||
		a.ProwlarrURL != b.ProwlarrURL ||
		a.SyncLevel != b.SyncLevel {
		return false
	}

	// Compare sync categories
	if len(a.SyncCategories) != len(b.SyncCategories) {
		return false
	}
	catMap := make(map[int]bool)
	for _, c := range a.SyncCategories {
		catMap[c] = true
	}
	for _, c := range b.SyncCategories {
		if !catMap[c] {
			return false
		}
	}

	return true
	// Note: APIKey is not compared (secret)
}

// createApplication creates an application in Prowlarr
func (a *Adapter) createApplication(ctx context.Context, c *httpClient, app irv1.ProwlarrApplicationIR, tagID int) error {
	impl := appTypeToImpl(app.Type)
	resource := ApplicationResource{
		Name:           app.Name,
		Implementation: impl,
		ConfigContract: impl + "Settings", // Required by Prowlarr API
		SyncLevel:      app.SyncLevel,
		Tags:           []int{tagID},
	}

	// Build fields
	resource.Fields = buildApplicationFields(app)

	var created ApplicationResource
	if err := c.post(ctx, "/api/v1/applications", resource, &created); err != nil {
		return fmt.Errorf("failed to create application %s: %w", app.Name, err)
	}

	// Cache the ID
	cacheKey := fmt.Sprintf("%s:%s", c.baseURL, app.Name)
	applicationIDCache[cacheKey] = created.ID

	return nil
}

// updateApplication updates an existing application
func (a *Adapter) updateApplication(ctx context.Context, c *httpClient, app irv1.ProwlarrApplicationIR, tagID int) error {
	cacheKey := fmt.Sprintf("%s:%s", c.baseURL, app.Name)
	id, ok := applicationIDCache[cacheKey]
	if !ok {
		var apps []ApplicationResource
		if err := c.get(ctx, "/api/v1/applications", &apps); err != nil {
			return fmt.Errorf("failed to get applications: %w", err)
		}
		for _, existing := range apps {
			if existing.Name == app.Name {
				id = existing.ID
				applicationIDCache[cacheKey] = id
				break
			}
		}
		if id == 0 {
			return fmt.Errorf("application %s not found", app.Name)
		}
	}

	impl := appTypeToImpl(app.Type)
	resource := ApplicationResource{
		ID:             id,
		Name:           app.Name,
		Implementation: impl,
		ConfigContract: impl + "Settings", // Required by Prowlarr API
		SyncLevel:      app.SyncLevel,
		Tags:           []int{tagID},
	}

	resource.Fields = buildApplicationFields(app)

	path := fmt.Sprintf("/api/v1/applications/%d", id)
	if err := c.put(ctx, path, resource, nil); err != nil {
		return fmt.Errorf("failed to update application %s: %w", app.Name, err)
	}

	return nil
}

// deleteApplication deletes an application
func (a *Adapter) deleteApplication(ctx context.Context, c *httpClient, name string) error {
	cacheKey := fmt.Sprintf("%s:%s", c.baseURL, name)
	id, ok := applicationIDCache[cacheKey]
	if !ok {
		var apps []ApplicationResource
		if err := c.get(ctx, "/api/v1/applications", &apps); err != nil {
			return fmt.Errorf("failed to get applications: %w", err)
		}
		for _, existing := range apps {
			if existing.Name == name {
				id = existing.ID
				break
			}
		}
		if id == 0 {
			return nil // Already deleted
		}
	}

	path := fmt.Sprintf("/api/v1/applications/%d", id)
	if err := c.delete(ctx, path); err != nil {
		return fmt.Errorf("failed to delete application %s: %w", name, err)
	}

	delete(applicationIDCache, cacheKey)
	return nil
}

// buildApplicationFields builds fields for an application
func buildApplicationFields(app irv1.ProwlarrApplicationIR) []ApplicationField {
	var fields []ApplicationField

	if app.ProwlarrURL != "" {
		fields = append(fields, ApplicationField{
			Name:  "prowlarrUrl",
			Value: app.ProwlarrURL,
		})
	}

	if app.URL != "" {
		fields = append(fields, ApplicationField{
			Name:  "baseUrl",
			Value: app.URL,
		})
	}

	if app.APIKey != "" {
		fields = append(fields, ApplicationField{
			Name:  "apiKey",
			Value: app.APIKey,
		})
	}

	// Use default categories if none specified
	categories := app.SyncCategories
	if len(categories) == 0 {
		if defaults, ok := irv1.DefaultSyncCategories[app.Type]; ok {
			categories = defaults
		}
	}

	if len(categories) > 0 {
		fields = append(fields, ApplicationField{
			Name:  "syncCategories",
			Value: categories,
		})
	}

	return fields
}
