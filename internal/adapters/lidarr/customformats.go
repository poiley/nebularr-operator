package lidarr

import (
	"context"
	"fmt"

	"github.com/poiley/nebularr-operator/internal/adapters"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

// getAllCustomFormats retrieves all custom formats from Lidarr
// This is used by CurrentState to get the current state for diffing
// Note: Requires Lidarr v2.0+
func (a *Adapter) getAllCustomFormats(ctx context.Context, c *httpClient) ([]irv1.CustomFormatIR, error) {
	var customFormats []CustomFormatResource
	if err := c.get(ctx, "/api/v1/customformat", &customFormats); err != nil {
		return nil, fmt.Errorf("failed to get custom formats: %w", err)
	}

	result := make([]irv1.CustomFormatIR, 0, len(customFormats))
	for _, cf := range customFormats {
		ir := a.customFormatToIR(&cf)
		result = append(result, ir)
	}

	return result, nil
}

// customFormatToIR converts a Lidarr custom format to IR
func (a *Adapter) customFormatToIR(cf *CustomFormatResource) irv1.CustomFormatIR {
	ir := irv1.CustomFormatIR{
		ID:                  cf.ID,
		Name:                cf.Name,
		IncludeWhenRenaming: cf.IncludeCustomFormatWhenRenaming,
		Specifications:      make([]irv1.FormatSpecIR, 0, len(cf.Specifications)),
	}

	for _, spec := range cf.Specifications {
		specIR := irv1.FormatSpecIR{
			Type:     spec.Implementation,
			Name:     spec.Name,
			Negate:   spec.Negate,
			Required: spec.Required,
		}

		// Extract the value field from the spec's fields
		for _, field := range spec.Fields {
			if field.Name == "value" {
				if v, ok := field.Value.(string); ok {
					specIR.Value = v
				} else {
					specIR.Value = fmt.Sprintf("%v", field.Value)
				}
				break
			}
		}

		ir.Specifications = append(ir.Specifications, specIR)
	}

	return ir
}

// irToCustomFormat converts IR to a Lidarr custom format resource
func (a *Adapter) irToCustomFormat(ir *irv1.CustomFormatIR) CustomFormatResource {
	cf := CustomFormatResource{
		ID:                              ir.ID,
		Name:                            ir.Name,
		IncludeCustomFormatWhenRenaming: ir.IncludeWhenRenaming,
		Specifications:                  make([]CustomFormatSpecification, 0, len(ir.Specifications)),
	}

	for _, spec := range ir.Specifications {
		cfSpec := CustomFormatSpecification{
			Name:           spec.Name,
			Implementation: spec.Type,
			Negate:         spec.Negate,
			Required:       spec.Required,
			Fields:         buildSpecFields(spec),
		}
		cf.Specifications = append(cf.Specifications, cfSpec)
	}

	return cf
}

// buildSpecFields creates the Fields array for a custom format specification
func buildSpecFields(spec irv1.FormatSpecIR) []Field {
	// Lidarr custom formats use simple string values for most specifications
	return []Field{
		{Name: "value", Value: spec.Value},
	}
}

// diffCustomFormats computes changes needed for custom formats
func (a *Adapter) diffCustomFormats(current, desired *irv1.IR, changes *adapters.ChangeSet) error {
	currentMap := make(map[string]irv1.CustomFormatIR)
	for _, cf := range current.CustomFormats {
		currentMap[cf.Name] = cf
	}

	desiredMap := make(map[string]irv1.CustomFormatIR)
	for _, cf := range desired.CustomFormats {
		desiredMap[cf.Name] = cf
	}

	// Find creates and updates
	for name, desiredCF := range desiredMap {
		currentCF, exists := currentMap[name]
		if !exists {
			payload := desiredCF // Copy to avoid pointer issues
			changes.Creates = append(changes.Creates, adapters.Change{
				ResourceType: adapters.ResourceCustomFormat,
				Name:         name,
				Payload:      &payload,
			})
		} else if !customFormatsEqual(currentCF, desiredCF) {
			desiredCF.ID = currentCF.ID // Preserve the ID for update
			id := currentCF.ID
			payload := desiredCF // Copy
			changes.Updates = append(changes.Updates, adapters.Change{
				ResourceType: adapters.ResourceCustomFormat,
				Name:         name,
				ID:           &id,
				Payload:      &payload,
			})
		}
	}

	// Find deletes
	for name, currentCF := range currentMap {
		if _, exists := desiredMap[name]; !exists {
			id := currentCF.ID
			changes.Deletes = append(changes.Deletes, adapters.Change{
				ResourceType: adapters.ResourceCustomFormat,
				Name:         name,
				ID:           &id,
			})
		}
	}

	return nil
}

// customFormatsEqual checks if two custom formats are equal (ignoring ID)
func customFormatsEqual(a, b irv1.CustomFormatIR) bool {
	if a.IncludeWhenRenaming != b.IncludeWhenRenaming {
		return false
	}

	if len(a.Specifications) != len(b.Specifications) {
		return false
	}

	// Compare specifications
	for i, specA := range a.Specifications {
		specB := b.Specifications[i]
		if specA.Type != specB.Type || specA.Name != specB.Name {
			return false
		}
		if specA.Negate != specB.Negate || specA.Required != specB.Required {
			return false
		}
		if specA.Value != specB.Value {
			return false
		}
	}

	return true
}

// createCustomFormat creates a custom format in Lidarr
func (a *Adapter) createCustomFormat(ctx context.Context, c *httpClient, ir *irv1.CustomFormatIR) error {
	customFormat := a.irToCustomFormat(ir)

	var created CustomFormatResource
	if err := c.post(ctx, "/api/v1/customformat", customFormat, &created); err != nil {
		return fmt.Errorf("failed to create custom format: %w", err)
	}

	return nil
}

// updateCustomFormat updates a custom format in Lidarr
func (a *Adapter) updateCustomFormat(ctx context.Context, c *httpClient, ir *irv1.CustomFormatIR) error {
	customFormat := a.irToCustomFormat(ir)

	endpoint := fmt.Sprintf("/api/v1/customformat/%d", ir.ID)
	var updated CustomFormatResource
	if err := c.put(ctx, endpoint, customFormat, &updated); err != nil {
		return fmt.Errorf("failed to update custom format: %w", err)
	}

	return nil
}

// deleteCustomFormat deletes a custom format from Lidarr
func (a *Adapter) deleteCustomFormat(ctx context.Context, c *httpClient, id int) error {
	endpoint := fmt.Sprintf("/api/v1/customformat/%d", id)
	if err := c.delete(ctx, endpoint); err != nil {
		return fmt.Errorf("failed to delete custom format: %w", err)
	}

	return nil
}
