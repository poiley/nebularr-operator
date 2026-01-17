package radarr

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/poiley/nebularr-operator/internal/adapters/radarr/client"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

// getImportLists fetches all import lists from Radarr
func (a *Adapter) getImportLists(ctx context.Context, c *client.Client) ([]client.ImportListResource, error) {
	resp, err := c.GetApiV3Importlist(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get import lists: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var lists []client.ImportListResource
	if err := json.NewDecoder(resp.Body).Decode(&lists); err != nil {
		return nil, fmt.Errorf("failed to decode import lists: %w", err)
	}

	return lists, nil
}

// getImportListSchemas fetches available import list schemas
func (a *Adapter) getImportListSchemas(ctx context.Context, c *client.Client) ([]client.ImportListResource, error) {
	resp, err := c.GetApiV3ImportlistSchema(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get import list schemas: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var schemas []client.ImportListResource
	if err := json.NewDecoder(resp.Body).Decode(&schemas); err != nil {
		return nil, fmt.Errorf("failed to decode import list schemas: %w", err)
	}

	return schemas, nil
}

// findSchemaByType finds a schema by implementation type
func findSchemaByType(schemas []client.ImportListResource, listType string) *client.ImportListResource {
	for i := range schemas {
		if ptrToString(schemas[i].Implementation) == listType {
			return &schemas[i]
		}
	}
	return nil
}

// buildImportListFields builds the fields array from settings
func buildImportListFields(settings map[string]string, schema *client.ImportListResource) []client.Field {
	fields := make([]client.Field, 0)

	if schema.Fields == nil {
		return fields
	}

	// Create a map of schema fields for validation
	schemaFields := make(map[string]bool)
	for _, f := range *schema.Fields {
		if f.Name != nil {
			schemaFields[*f.Name] = true
		}
	}

	for name, value := range settings {
		// Try the name as-is first
		if schemaFields[name] {
			nameCopy := name
			var valueCopy interface{} = value
			fields = append(fields, client.Field{Name: &nameCopy, Value: &valueCopy})
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

// applyImportLists applies import list changes directly to Radarr
func (a *Adapter) applyImportLists(
	ctx context.Context,
	c *client.Client,
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
	existingByName := make(map[string]*client.ImportListResource)
	for i := range existing {
		if existing[i].Name != nil {
			existingByName[*existing[i].Name] = &existing[i]
		}
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

		// Build the payload using client types
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
			payload.Id = existingList.Id
			if err := a.updateImportList(ctx, c, payload); err != nil {
				stats.Errors = append(stats.Errors, fmt.Errorf("failed to update import list %s: %w", list.Name, err))
			} else {
				stats.Updated++
			}
		}
	}

	// Delete orphaned import lists (managed by us but not in desired state)
	for name, existingList := range existingByName {
		if !desiredNames[name] && a.hasTag(existingList.Tags, tagID) {
			if err := a.deleteImportList(ctx, c, ptrToInt(existingList.Id)); err != nil {
				stats.Errors = append(stats.Errors, fmt.Errorf("failed to delete import list %s: %w", name, err))
			} else {
				stats.Deleted++
			}
		}
	}

	return stats, nil
}

// irToImportList converts an IR import list to a client ImportListResource
func (a *Adapter) irToImportList(ir *irv1.ImportListIR, schema *client.ImportListResource, fields []client.Field, tagID int) client.ImportListResource {
	// Convert monitor type
	var monitor *client.MonitorTypes
	if ir.Monitor != "" {
		m := client.MonitorTypes(ir.Monitor)
		monitor = &m
	}

	// Convert minimum availability
	var minAvail *client.MovieStatusType
	if ir.MinimumAvailability != "" {
		ma := client.MovieStatusType(ir.MinimumAvailability)
		minAvail = &ma
	}

	// Convert list type
	listType := client.ImportListType("program")

	tags := []int32{int32(tagID)}
	listOrder := int32(0)
	qualityProfileID := int32(ir.QualityProfileID)

	return client.ImportListResource{
		Name:                stringPtr(ir.Name),
		Enabled:             boolPtr(ir.Enabled),
		EnableAuto:          boolPtr(ir.EnableAuto),
		SearchOnAdd:         boolPtr(ir.SearchOnAdd),
		QualityProfileId:    &qualityProfileID,
		RootFolderPath:      stringPtr(ir.RootFolderPath),
		Monitor:             monitor,
		MinimumAvailability: minAvail,
		ListType:            &listType,
		ListOrder:           &listOrder,
		Implementation:      stringPtr(ir.Type),
		ConfigContract:      schema.ConfigContract,
		Fields:              &fields,
		Tags:                &tags,
	}
}

// hasTag checks if the resource has the specified tag
func (a *Adapter) hasTag(tags *[]int32, tagID int) bool {
	if tags == nil {
		return false
	}
	for _, t := range *tags {
		if int(t) == tagID {
			return true
		}
	}
	return false
}

// createImportList creates a new import list
func (a *Adapter) createImportList(ctx context.Context, c *client.Client, payload client.ImportListResource) error {
	resp, err := c.PostApiV3Importlist(ctx, nil, payload)
	if err != nil {
		return fmt.Errorf("failed to create import list: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// updateImportList updates an existing import list
func (a *Adapter) updateImportList(ctx context.Context, c *client.Client, payload client.ImportListResource) error {
	resp, err := c.PutApiV3ImportlistId(ctx, *payload.Id, nil, payload)
	if err != nil {
		return fmt.Errorf("failed to update import list: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// deleteImportList deletes an import list
func (a *Adapter) deleteImportList(ctx context.Context, c *client.Client, id int) error {
	resp, err := c.DeleteApiV3ImportlistId(ctx, int32(id))
	if err != nil {
		return fmt.Errorf("failed to delete import list: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// getManagedImportLists retrieves import lists managed by Nebularr (tagged)
func (a *Adapter) getManagedImportLists(ctx context.Context, c *client.Client, tagID int) ([]irv1.ImportListIR, error) {
	lists, err := a.getImportLists(ctx, c)
	if err != nil {
		return nil, err
	}

	var managed []irv1.ImportListIR
	for _, list := range lists {
		if a.hasTag(list.Tags, tagID) {
			managed = append(managed, a.importListToIR(&list))
		}
	}

	return managed, nil
}

// importListToIR converts a client ImportListResource to an IR ImportListIR
func (a *Adapter) importListToIR(list *client.ImportListResource) irv1.ImportListIR {
	ir := irv1.ImportListIR{
		Name:           ptrToString(list.Name),
		Type:           ptrToString(list.Implementation),
		Enabled:        ptrToBool(list.Enabled),
		EnableAuto:     ptrToBool(list.EnableAuto),
		SearchOnAdd:    ptrToBool(list.SearchOnAdd),
		RootFolderPath: ptrToString(list.RootFolderPath),
		Settings:       make(map[string]string),
	}

	if list.QualityProfileId != nil {
		ir.QualityProfileID = int(*list.QualityProfileId)
	}

	if list.Monitor != nil {
		ir.Monitor = string(*list.Monitor)
	}

	if list.MinimumAvailability != nil {
		ir.MinimumAvailability = string(*list.MinimumAvailability)
	}

	// Convert fields to settings
	if list.Fields != nil {
		for _, f := range *list.Fields {
			if f.Name != nil && f.Value != nil {
				// Convert value to string
				switch v := f.Value.(type) {
				case string:
					ir.Settings[*f.Name] = v
				case float64:
					ir.Settings[*f.Name] = fmt.Sprintf("%v", v)
				default:
					ir.Settings[*f.Name] = fmt.Sprintf("%v", v)
				}
			}
		}
	}

	return ir
}
