package sonarr

import (
	"context"

	"github.com/poiley/nebularr-operator/internal/adapters"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

// indexerIDMap tracks IDs for managed indexers
var indexerIDMap = make(map[string]int)

// getManagedIndexers retrieves indexers managed by Nebularr
func (a *Adapter) getManagedIndexers(ctx context.Context, c *httpClient, tagID int) ([]irv1.IndexerIR, error) {
	var indexers []IndexerResource
	if err := c.get(ctx, "/api/v3/indexer", &indexers); err != nil {
		return nil, err
	}

	var managed []irv1.IndexerIR
	for _, idx := range indexers {
		if hasTag(idx.Tags, tagID) {
			indexerIDMap[idx.Name] = idx.ID
			managed = append(managed, a.indexerToIR(&idx))
		}
	}

	return managed, nil
}

// indexerToIR converts a Sonarr indexer to IR
func (a *Adapter) indexerToIR(idx *IndexerResource) irv1.IndexerIR {
	ir := irv1.IndexerIR{
		Name:                    idx.Name,
		Implementation:          idx.Implementation,
		Protocol:                idx.Protocol,
		Enable:                  idx.Enable,
		Priority:                idx.Priority,
		EnableRss:               idx.EnableRss,
		EnableAutomaticSearch:   idx.EnableAutomaticSearch,
		EnableInteractiveSearch: idx.EnableInteractiveSearch,
	}

	// Extract fields
	for _, f := range idx.Fields {
		switch f.Name {
		case "baseUrl":
			if v, ok := f.Value.(string); ok {
				ir.URL = v
			}
		case "apiKey":
			if v, ok := f.Value.(string); ok {
				ir.APIKey = v
			}
		case "categories":
			if v, ok := f.Value.([]interface{}); ok {
				for _, cat := range v {
					if catNum, ok := cat.(float64); ok {
						ir.Categories = append(ir.Categories, int(catNum))
					}
				}
			}
		case "minimumSeeders":
			if v, ok := f.Value.(float64); ok {
				ir.MinimumSeeders = int(v)
			}
		case "seedCriteria.seedRatio":
			if v, ok := f.Value.(float64); ok {
				ir.SeedRatio = v
			}
		case "seedCriteria.seedTime":
			if v, ok := f.Value.(float64); ok {
				ir.SeedTimeMinutes = int(v)
			}
		}
	}

	return ir
}

// diffIndexers computes changes needed for indexers
func (a *Adapter) diffIndexers(current, desired *irv1.IR, changes *adapters.ChangeSet) error {
	var currentIndexers []irv1.IndexerIR
	var desiredIndexers []irv1.IndexerIR

	if current.Indexers != nil {
		currentIndexers = current.Indexers.Direct
	}
	if desired.Indexers != nil {
		desiredIndexers = desired.Indexers.Direct
	}

	currentMap := make(map[string]irv1.IndexerIR)
	for _, idx := range currentIndexers {
		currentMap[idx.Name] = idx
	}

	desiredMap := make(map[string]irv1.IndexerIR)
	for _, idx := range desiredIndexers {
		desiredMap[idx.Name] = idx
	}

	// Find creates and updates
	for name, desiredIdx := range desiredMap {
		currentIdx, exists := currentMap[name]
		if !exists {
			changes.Creates = append(changes.Creates, adapters.Change{
				ResourceType: adapters.ResourceIndexer,
				Name:         name,
				Payload:      desiredIdx,
			})
		} else if indexerNeedsUpdate(currentIdx, desiredIdx) {
			id := indexerIDMap[name]
			changes.Updates = append(changes.Updates, adapters.Change{
				ResourceType: adapters.ResourceIndexer,
				Name:         name,
				ID:           &id,
				Payload:      desiredIdx,
			})
		}
	}

	// Find deletes
	for name := range currentMap {
		if _, exists := desiredMap[name]; !exists {
			id := indexerIDMap[name]
			changes.Deletes = append(changes.Deletes, adapters.Change{
				ResourceType: adapters.ResourceIndexer,
				Name:         name,
				ID:           &id,
			})
		}
	}

	return nil
}

// indexerNeedsUpdate checks if indexer needs updating
func indexerNeedsUpdate(current, desired irv1.IndexerIR) bool {
	if current.URL != desired.URL {
		return true
	}
	if current.Enable != desired.Enable {
		return true
	}
	if current.Priority != desired.Priority {
		return true
	}
	return false
}
