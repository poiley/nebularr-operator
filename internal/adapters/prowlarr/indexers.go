package prowlarr

import (
	"context"
	"fmt"

	"github.com/poiley/nebularr-operator/internal/adapters"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

// Package-level cache for indexer IDs
var indexerIDCache = make(map[string]int) // "baseURL:name" -> ID

// getManagedIndexers retrieves indexers tagged with ownership tag
func (a *Adapter) getManagedIndexers(ctx context.Context, c *httpClient, tagID int) ([]irv1.ProwlarrIndexerIR, error) {
	var indexers []IndexerResource
	if err := c.get(ctx, "/api/v1/indexer", &indexers); err != nil {
		return nil, fmt.Errorf("failed to get indexers: %w", err)
	}

	managed := make([]irv1.ProwlarrIndexerIR, 0, len(indexers))
	for _, idx := range indexers {
		if !hasTag(idx.Tags, tagID) {
			continue
		}

		// Cache the ID
		cacheKey := fmt.Sprintf("%s:%s", c.baseURL, idx.Name)
		indexerIDCache[cacheKey] = idx.ID

		// Convert to IR
		ir := irv1.ProwlarrIndexerIR{
			Name:       idx.Name,
			Definition: idx.DefinitionName,
			Enable:     idx.Enable,
			Priority:   idx.Priority,
		}

		// Extract settings from fields
		ir.Settings = make(map[string]string)
		for _, field := range idx.Fields {
			switch field.Name {
			case "baseUrl":
				if v, ok := field.Value.(string); ok {
					ir.BaseURL = v
				}
			case "apiKey":
				if v, ok := field.Value.(string); ok {
					ir.APIKey = v
				}
			default:
				if v, ok := field.Value.(string); ok {
					ir.Settings[field.Name] = v
				}
			}
		}

		managed = append(managed, ir)
	}

	return managed, nil
}

// diffIndexers computes changes needed for indexers
func (a *Adapter) diffIndexers(current, desired *irv1.ProwlarrIR, changes *adapters.ChangeSet) error {
	currentByName := make(map[string]irv1.ProwlarrIndexerIR)
	for _, idx := range current.Indexers {
		currentByName[idx.Name] = idx
	}

	desiredByName := make(map[string]irv1.ProwlarrIndexerIR)
	for _, idx := range desired.Indexers {
		desiredByName[idx.Name] = idx
	}

	// Find creates and updates
	for name, desiredIdx := range desiredByName {
		currentIdx, exists := currentByName[name]
		if !exists {
			// Create
			changes.Creates = append(changes.Creates, adapters.Change{
				ResourceType: adapters.ResourceIndexer,
				Name:         name,
				Payload:      desiredIdx,
			})
		} else if !indexersEqual(currentIdx, desiredIdx) {
			// Update
			changes.Updates = append(changes.Updates, adapters.Change{
				ResourceType: adapters.ResourceIndexer,
				Name:         name,
				Payload:      desiredIdx,
			})
		}
	}

	// Find deletes
	for name := range currentByName {
		if _, exists := desiredByName[name]; !exists {
			changes.Deletes = append(changes.Deletes, adapters.Change{
				ResourceType: adapters.ResourceIndexer,
				Name:         name,
			})
		}
	}

	return nil
}

// indexersEqual compares two indexers for equality
func indexersEqual(a, b irv1.ProwlarrIndexerIR) bool {
	if a.Definition != b.Definition ||
		a.Enable != b.Enable ||
		a.Priority != b.Priority ||
		a.BaseURL != b.BaseURL {
		return false
	}

	// Compare settings (ignoring APIKey since it's a secret)
	if len(a.Settings) != len(b.Settings) {
		return false
	}
	for k, v := range a.Settings {
		if b.Settings[k] != v {
			return false
		}
	}

	return true
}

// createIndexer creates an indexer in Prowlarr
func (a *Adapter) createIndexer(ctx context.Context, c *httpClient, idx irv1.ProwlarrIndexerIR, tagID int) error {
	// Build the resource
	resource := IndexerResource{
		Name:           idx.Name,
		DefinitionName: idx.Definition,
		Enable:         idx.Enable,
		Priority:       idx.Priority,
		Tags:           []int{tagID},
	}

	// Build fields from settings
	resource.Fields = []IndexerField{}

	if idx.BaseURL != "" {
		resource.Fields = append(resource.Fields, IndexerField{
			Name:  "baseUrl",
			Value: idx.BaseURL,
		})
	}

	if idx.APIKey != "" {
		resource.Fields = append(resource.Fields, IndexerField{
			Name:  "apiKey",
			Value: idx.APIKey,
		})
	}

	for k, v := range idx.Settings {
		resource.Fields = append(resource.Fields, IndexerField{
			Name:  k,
			Value: v,
		})
	}

	var created IndexerResource
	if err := c.post(ctx, "/api/v1/indexer", resource, &created); err != nil {
		return fmt.Errorf("failed to create indexer %s: %w", idx.Name, err)
	}

	// Cache the ID
	cacheKey := fmt.Sprintf("%s:%s", c.baseURL, idx.Name)
	indexerIDCache[cacheKey] = created.ID

	return nil
}

// updateIndexer updates an existing indexer
func (a *Adapter) updateIndexer(ctx context.Context, c *httpClient, idx irv1.ProwlarrIndexerIR, tagID int) error {
	// Get the cached ID
	cacheKey := fmt.Sprintf("%s:%s", c.baseURL, idx.Name)
	id, ok := indexerIDCache[cacheKey]
	if !ok {
		// Try to find it
		var indexers []IndexerResource
		if err := c.get(ctx, "/api/v1/indexer", &indexers); err != nil {
			return fmt.Errorf("failed to get indexers: %w", err)
		}
		for _, existing := range indexers {
			if existing.Name == idx.Name {
				id = existing.ID
				indexerIDCache[cacheKey] = id
				break
			}
		}
		if id == 0 {
			return fmt.Errorf("indexer %s not found", idx.Name)
		}
	}

	// Build the resource
	resource := IndexerResource{
		ID:             id,
		Name:           idx.Name,
		DefinitionName: idx.Definition,
		Enable:         idx.Enable,
		Priority:       idx.Priority,
		Tags:           []int{tagID},
	}

	// Build fields
	resource.Fields = []IndexerField{}

	if idx.BaseURL != "" {
		resource.Fields = append(resource.Fields, IndexerField{
			Name:  "baseUrl",
			Value: idx.BaseURL,
		})
	}

	if idx.APIKey != "" {
		resource.Fields = append(resource.Fields, IndexerField{
			Name:  "apiKey",
			Value: idx.APIKey,
		})
	}

	for k, v := range idx.Settings {
		resource.Fields = append(resource.Fields, IndexerField{
			Name:  k,
			Value: v,
		})
	}

	path := fmt.Sprintf("/api/v1/indexer/%d", id)
	if err := c.put(ctx, path, resource, nil); err != nil {
		return fmt.Errorf("failed to update indexer %s: %w", idx.Name, err)
	}

	return nil
}

// deleteIndexer deletes an indexer
func (a *Adapter) deleteIndexer(ctx context.Context, c *httpClient, name string) error {
	cacheKey := fmt.Sprintf("%s:%s", c.baseURL, name)
	id, ok := indexerIDCache[cacheKey]
	if !ok {
		// Try to find it
		var indexers []IndexerResource
		if err := c.get(ctx, "/api/v1/indexer", &indexers); err != nil {
			return fmt.Errorf("failed to get indexers: %w", err)
		}
		for _, existing := range indexers {
			if existing.Name == name {
				id = existing.ID
				break
			}
		}
		if id == 0 {
			// Already deleted
			return nil
		}
	}

	path := fmt.Sprintf("/api/v1/indexer/%d", id)
	if err := c.delete(ctx, path); err != nil {
		return fmt.Errorf("failed to delete indexer %s: %w", name, err)
	}

	// Remove from cache
	delete(indexerIDCache, cacheKey)

	return nil
}
