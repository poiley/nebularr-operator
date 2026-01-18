package sonarr

import (
	"context"

	"github.com/poiley/nebularr-operator/internal/adapters"
	"github.com/poiley/nebularr-operator/internal/adapters/httpclient"
	"github.com/poiley/nebularr-operator/internal/adapters/shared"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

// downloadClientIDMap tracks IDs for managed download clients
var downloadClientIDMap = make(map[string]int)

// getManagedDownloadClients retrieves download clients managed by Nebularr
func (a *Adapter) getManagedDownloadClients(ctx context.Context, c *httpclient.Client, tagID int) ([]irv1.DownloadClientIR, error) {
	var clients []DownloadClientResource
	if err := c.Get(ctx, "/api/v3/downloadclient", &clients); err != nil {
		return nil, err
	}

	var managed []irv1.DownloadClientIR
	for _, dc := range clients {
		if hasTag(dc.Tags, tagID) {
			downloadClientIDMap[dc.Name] = dc.ID
			managed = append(managed, a.clientToIR(&dc))
		}
	}

	return managed, nil
}

// clientToIR converts a Sonarr download client to IR
func (a *Adapter) clientToIR(dc *DownloadClientResource) irv1.DownloadClientIR {
	ir := irv1.DownloadClientIR{
		Name:                     dc.Name,
		Implementation:           dc.Implementation,
		Protocol:                 dc.Protocol,
		Enable:                   dc.Enable,
		Priority:                 dc.Priority,
		RemoveCompletedDownloads: dc.RemoveCompletedDownloads,
		RemoveFailedDownloads:    dc.RemoveFailedDownloads,
	}

	// Extract fields using shared helper
	shared.ExtractDownloadClientFields(dc.Fields, &ir, "category", "tvCategory")

	return ir
}

// diffDownloadClients computes changes needed for download clients using shared logic
func (a *Adapter) diffDownloadClients(current, desired *irv1.IR, changes *adapters.ChangeSet) error {
	adapters.DiffDownloadClients(current.DownloadClients, desired.DownloadClients, downloadClientIDMap, changes)
	return nil
}
