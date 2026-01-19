package lidarr

import (
	"context"

	"github.com/poiley/nebularr-operator/internal/adapters"
	"github.com/poiley/nebularr-operator/internal/adapters/httpclient"
	"github.com/poiley/nebularr-operator/internal/adapters/shared"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

// getManagedIndexers retrieves indexers managed by Nebularr
func (a *Adapter) getManagedIndexers(ctx context.Context, c *httpclient.Client, tagID int) ([]irv1.IndexerIR, error) {
	var indexers []IndexerResource
	if err := c.Get(ctx, "/api/v1/indexer", &indexers); err != nil {
		return nil, err
	}

	var managed []irv1.IndexerIR
	for _, idx := range indexers {
		if hasTag(idx.Tags, tagID) {
			managed = append(managed, a.indexerToIR(&idx))
		}
	}

	return managed, nil
}

// indexerToIR converts a Lidarr indexer to IR
func (a *Adapter) indexerToIR(idx *IndexerResource) irv1.IndexerIR {
	ir := irv1.IndexerIR{
		ID:                      idx.ID, // Capture ID for updates/deletes
		Name:                    idx.Name,
		Implementation:          idx.Implementation,
		Protocol:                idx.Protocol,
		Enable:                  idx.Enable,
		Priority:                idx.Priority,
		EnableRss:               idx.EnableRss,
		EnableAutomaticSearch:   idx.EnableAutomaticSearch,
		EnableInteractiveSearch: idx.EnableInteractiveSearch,
	}

	// Extract fields using shared helper
	shared.ExtractIndexerFields(idx.Fields, &ir)

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

	adapters.DiffIndexersWithIR(currentIndexers, desiredIndexers, changes)
	return nil
}
