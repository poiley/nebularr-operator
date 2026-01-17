package radarr

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/poiley/nebularr-operator/internal/adapters"
	"github.com/poiley/nebularr-operator/internal/adapters/radarr/client"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

// getManagedIndexers retrieves indexers tagged with the ownership tag
func (a *Adapter) getManagedIndexers(ctx context.Context, c *client.Client, tagID int) ([]irv1.IndexerIR, error) {
	resp, err := c.GetApiV3Indexer(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get indexers: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var indexers []client.IndexerResource
	if err := json.NewDecoder(resp.Body).Decode(&indexers); err != nil {
		return nil, fmt.Errorf("failed to decode indexers: %w", err)
	}

	result := make([]irv1.IndexerIR, 0, len(indexers))
	for _, idx := range indexers {
		// Check if this indexer has the ownership tag
		if !hasTag(idx.Tags, tagID) {
			continue
		}

		ir := a.indexerToIR(&idx)
		result = append(result, ir)
	}

	return result, nil
}

// indexerToIR converts a Radarr indexer to IR
func (a *Adapter) indexerToIR(idx *client.IndexerResource) irv1.IndexerIR {
	// Derive Enable from whether any of the search/rss options are enabled
	enableRss := ptrToBool(idx.EnableRss)
	enableAutoSearch := ptrToBool(idx.EnableAutomaticSearch)
	enableInteractive := ptrToBool(idx.EnableInteractiveSearch)
	enabled := enableRss || enableAutoSearch || enableInteractive

	ir := irv1.IndexerIR{
		Name:                    ptrToString(idx.Name),
		Enable:                  enabled,
		Priority:                ptrToInt(idx.Priority),
		Implementation:          ptrToString(idx.Implementation),
		EnableRss:               enableRss,
		EnableAutomaticSearch:   enableAutoSearch,
		EnableInteractiveSearch: enableInteractive,
	}

	// Determine protocol from implementation
	impl := ir.Implementation
	switch impl {
	case "Torznab", "TorrentRssIndexer":
		ir.Protocol = irv1.ProtocolTorrent
	case "Newznab":
		ir.Protocol = irv1.ProtocolUsenet
	}

	// Extract connection details from fields
	if idx.Fields != nil {
		for _, field := range *idx.Fields {
			name := ptrToString(field.Name)
			switch name {
			case "baseUrl":
				if v, ok := field.Value.(string); ok {
					ir.URL = v
				}
			case "apiKey":
				if v, ok := field.Value.(string); ok {
					ir.APIKey = v
				}
			case "categories":
				if v, ok := field.Value.([]interface{}); ok {
					for _, cat := range v {
						if catFloat, ok := cat.(float64); ok {
							ir.Categories = append(ir.Categories, int(catFloat))
						}
					}
				}
			case "minimumSeeders":
				if v, ok := field.Value.(float64); ok {
					ir.MinimumSeeders = int(v)
				}
			case "seedCriteria.seedRatio":
				if v, ok := field.Value.(float64); ok {
					ir.SeedRatio = v
				}
			case "seedCriteria.seedTime":
				if v, ok := field.Value.(float64); ok {
					ir.SeedTimeMinutes = int(v)
				}
			}
		}
	}

	return ir
}

// diffIndexers computes changes needed for indexers
func (a *Adapter) diffIndexers(current, desired *irv1.IR, changes *adapters.ChangeSet) error {
	// Build maps for comparison
	currentIndexers := make(map[string]*irv1.IndexerIR)
	desiredIndexers := make(map[string]*irv1.IndexerIR)

	if current.Indexers != nil {
		for i := range current.Indexers.Direct {
			idx := &current.Indexers.Direct[i]
			currentIndexers[idx.Name] = idx
		}
	}

	if desired.Indexers != nil {
		for i := range desired.Indexers.Direct {
			idx := &desired.Indexers.Direct[i]
			desiredIndexers[idx.Name] = idx
		}
	}

	// Find creates and updates
	for name, desiredIdx := range desiredIndexers {
		if _, exists := currentIndexers[name]; !exists {
			changes.Creates = append(changes.Creates, adapters.Change{
				ResourceType: adapters.ResourceIndexer,
				Name:         name,
				Payload:      desiredIdx,
			})
		} else {
			// Update
			changes.Updates = append(changes.Updates, adapters.Change{
				ResourceType: adapters.ResourceIndexer,
				Name:         name,
				Payload:      desiredIdx,
			})
		}
	}

	// Find deletes
	for name := range currentIndexers {
		if _, exists := desiredIndexers[name]; !exists {
			changes.Deletes = append(changes.Deletes, adapters.Change{
				ResourceType: adapters.ResourceIndexer,
				Name:         name,
			})
		}
	}

	return nil
}
