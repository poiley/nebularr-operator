package lidarr

import (
	"context"
	"fmt"

	"github.com/poiley/nebularr-operator/internal/adapters/httpclient"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

// ImportListResource represents a Lidarr import list
type ImportListResource struct {
	ID                 int     `json:"id,omitempty"`
	Name               string  `json:"name"`
	Implementation     string  `json:"implementation"`
	ConfigContract     string  `json:"configContract"`
	EnableAutomaticAdd bool    `json:"enableAutomaticAdd"`
	SearchForNewAlbum  bool    `json:"searchForNewAlbum"`
	QualityProfileID   int     `json:"qualityProfileId"`
	MetadataProfileID  int     `json:"metadataProfileId"`
	RootFolderPath     string  `json:"rootFolderPath"`
	ShouldMonitor      string  `json:"shouldMonitor"`   // none, specificAlbum, entireArtist
	MonitorNewItems    string  `json:"monitorNewItems"` // none, new, all
	ListType           string  `json:"listType"`
	ListOrder          int     `json:"listOrder"`
	Tags               []int   `json:"tags"`
	Fields             []Field `json:"fields"`
}

// getImportLists fetches all import lists from Lidarr
func (a *Adapter) getImportLists(ctx context.Context, c *httpclient.Client) ([]ImportListResource, error) {
	var lists []ImportListResource
	if err := c.Get(ctx, "/api/v1/importlist", &lists); err != nil {
		return nil, fmt.Errorf("failed to get import lists: %w", err)
	}
	return lists, nil
}

// getImportListSchemas fetches available import list schemas
func (a *Adapter) getImportListSchemas(ctx context.Context, c *httpclient.Client) ([]ImportListResource, error) {
	var schemas []ImportListResource
	if err := c.Get(ctx, "/api/v1/importlist/schema", &schemas); err != nil {
		return nil, fmt.Errorf("failed to get import list schemas: %w", err)
	}
	return schemas, nil
}

// findSchemaByType finds a schema by implementation type
func findSchemaByType(schemas []ImportListResource, listType string) *ImportListResource {
	for i := range schemas {
		if schemas[i].Implementation == listType {
			return &schemas[i]
		}
	}
	return nil
}

// buildImportListFields builds the fields array from settings
func buildImportListFields(settings map[string]string, schema *ImportListResource) []Field {
	fields := make([]Field, 0)

	// Create a map of schema fields for validation
	schemaFields := make(map[string]bool)
	for _, f := range schema.Fields {
		schemaFields[f.Name] = true
	}

	for name, value := range settings {
		if schemaFields[name] {
			fields = append(fields, Field{Name: name, Value: value})
		}
	}

	return fields
}

// ImportListApplyStats tracks the results of applying import lists
type ImportListApplyStats struct {
	Created int
	Updated int
	Deleted int
	Skipped int
	Errors  []error
}

// applyImportLists applies import list changes directly to Lidarr
func (a *Adapter) applyImportLists(
	ctx context.Context,
	c *httpclient.Client,
	ir *irv1.IR,
	tagID int,
) (*ImportListApplyStats, error) {
	stats := &ImportListApplyStats{}

	if len(ir.ImportLists) == 0 {
		return stats, nil
	}

	// Get existing import lists
	existing, err := a.getImportLists(ctx, c)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing import lists: %w", err)
	}

	// Get schemas for validation
	schemas, err := a.getImportListSchemas(ctx, c)
	if err != nil {
		return nil, fmt.Errorf("failed to get import list schemas: %w", err)
	}

	// Index existing by name
	existingByName := make(map[string]*ImportListResource)
	for i := range existing {
		existingByName[existing[i].Name] = &existing[i]
	}

	// Track desired names for orphan detection
	desiredNames := make(map[string]bool)

	for _, list := range ir.ImportLists {
		desiredNames[list.Name] = true

		// Find schema for this type
		schema := findSchemaByType(schemas, list.Type)
		if schema == nil {
			stats.Skipped++
			stats.Errors = append(stats.Errors, fmt.Errorf("unknown import list type %s for %s", list.Type, list.Name))
			continue
		}

		// Build fields from settings
		fields := buildImportListFields(list.Settings, schema)

		// Build the payload
		payload := a.irToImportList(&list, schema, fields, tagID)

		existingList := existingByName[list.Name]

		if existingList == nil {
			// Create new import list
			if err := a.createImportList(ctx, c, payload); err != nil {
				stats.Errors = append(stats.Errors, fmt.Errorf("failed to create import list %s: %w", list.Name, err))
			} else {
				stats.Created++
			}
		} else {
			// Update existing import list
			payload.ID = existingList.ID
			if err := a.updateImportList(ctx, c, payload); err != nil {
				stats.Errors = append(stats.Errors, fmt.Errorf("failed to update import list %s: %w", list.Name, err))
			} else {
				stats.Updated++
			}
		}
	}

	// Delete orphaned import lists (managed by us but not in desired state)
	for name, existingList := range existingByName {
		if !desiredNames[name] && hasTag(existingList.Tags, tagID) {
			if err := a.deleteImportList(ctx, c, existingList.ID); err != nil {
				stats.Errors = append(stats.Errors, fmt.Errorf("failed to delete import list %s: %w", name, err))
			} else {
				stats.Deleted++
			}
		}
	}

	return stats, nil
}

// irToImportList converts an IR import list to a Lidarr ImportListResource
func (a *Adapter) irToImportList(ir *irv1.ImportListIR, schema *ImportListResource, fields []Field, tagID int) ImportListResource {
	// Default monitor if not set (Lidarr uses different values)
	shouldMonitor := "entireArtist"
	monitorNewItems := "all"

	return ImportListResource{
		Name:               ir.Name,
		Implementation:     ir.Type,
		ConfigContract:     schema.ConfigContract,
		EnableAutomaticAdd: ir.EnableAuto,
		SearchForNewAlbum:  ir.SearchOnAdd,
		QualityProfileID:   ir.QualityProfileID,
		MetadataProfileID:  1, // Default metadata profile
		RootFolderPath:     ir.RootFolderPath,
		ShouldMonitor:      shouldMonitor,
		MonitorNewItems:    monitorNewItems,
		ListType:           "program",
		ListOrder:          0,
		Tags:               []int{tagID},
		Fields:             fields,
	}
}

// createImportList creates a new import list
func (a *Adapter) createImportList(ctx context.Context, c *httpclient.Client, payload ImportListResource) error {
	var result ImportListResource
	return c.Post(ctx, "/api/v1/importlist", payload, &result)
}

// updateImportList updates an existing import list
func (a *Adapter) updateImportList(ctx context.Context, c *httpclient.Client, payload ImportListResource) error {
	path := fmt.Sprintf("/api/v1/importlist/%d", payload.ID)
	var result ImportListResource
	return c.Put(ctx, path, payload, &result)
}

// deleteImportList deletes an import list
func (a *Adapter) deleteImportList(ctx context.Context, c *httpclient.Client, id int) error {
	path := fmt.Sprintf("/api/v1/importlist/%d", id)
	return c.Delete(ctx, path)
}

// getManagedImportLists retrieves import lists managed by Nebularr (tagged)
func (a *Adapter) getManagedImportLists(ctx context.Context, c *httpclient.Client, tagID int) ([]irv1.ImportListIR, error) {
	lists, err := a.getImportLists(ctx, c)
	if err != nil {
		return nil, err
	}

	var managed []irv1.ImportListIR
	for _, list := range lists {
		if hasTag(list.Tags, tagID) {
			managed = append(managed, a.importListToIR(&list))
		}
	}

	return managed, nil
}

// importListToIR converts a Lidarr ImportListResource to an IR ImportListIR
func (a *Adapter) importListToIR(list *ImportListResource) irv1.ImportListIR {
	ir := irv1.ImportListIR{
		Name:             list.Name,
		Type:             list.Implementation,
		Enabled:          true, // Lidarr import lists are always enabled
		EnableAuto:       list.EnableAutomaticAdd,
		SearchOnAdd:      list.SearchForNewAlbum,
		QualityProfileID: list.QualityProfileID,
		RootFolderPath:   list.RootFolderPath,
		Settings:         make(map[string]string),
	}

	// Convert fields to settings
	for _, f := range list.Fields {
		if f.Value != nil {
			switch v := f.Value.(type) {
			case string:
				ir.Settings[f.Name] = v
			case float64:
				ir.Settings[f.Name] = fmt.Sprintf("%v", v)
			default:
				ir.Settings[f.Name] = fmt.Sprintf("%v", v)
			}
		}
	}

	return ir
}
