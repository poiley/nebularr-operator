package lidarr

import (
	"context"

	"github.com/poiley/nebularr-operator/internal/adapters"
	"github.com/poiley/nebularr-operator/internal/adapters/httpclient"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

// indexerIDMap tracks IDs for managed indexers
var indexerIDMap = make(map[string]int)

// getManagedIndexers retrieves indexers managed by Nebularr
func (a *Adapter) getManagedIndexers(ctx context.Context, c *httpclient.Client, tagID int) ([]irv1.IndexerIR, error) {
	var indexers []IndexerResource
	if err := c.Get(ctx, "/api/v1/indexer", &indexers); err != nil {
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

// indexerToIR converts a Lidarr indexer to IR
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
		}
	}

	return ir
}

// diffIndexers computes changes needed for indexers using shared logic
func (a *Adapter) diffIndexers(current, desired *irv1.IR, changes *adapters.ChangeSet) error {
	var currentIndexers []irv1.IndexerIR
	var desiredIndexers []irv1.IndexerIR

	if current.Indexers != nil {
		currentIndexers = current.Indexers.Direct
	}
	if desired.Indexers != nil {
		desiredIndexers = desired.Indexers.Direct
	}

	// Use shared diff logic
	adapters.DiffIndexers(currentIndexers, desiredIndexers, indexerIDMap, changes)
	return nil
}
