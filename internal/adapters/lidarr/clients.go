package lidarr

import (
	"context"

	"github.com/poiley/nebularr-operator/internal/adapters"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

// downloadClientIDMap tracks IDs for managed download clients
var downloadClientIDMap = make(map[string]int)

// getManagedDownloadClients retrieves download clients managed by Nebularr
func (a *Adapter) getManagedDownloadClients(ctx context.Context, c *httpClient, tagID int) ([]irv1.DownloadClientIR, error) {
	var clients []DownloadClientResource
	if err := c.get(ctx, "/api/v1/downloadclient", &clients); err != nil {
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

// clientToIR converts a Lidarr download client to IR
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

	// Extract fields
	for _, f := range dc.Fields {
		switch f.Name {
		case "host":
			if v, ok := f.Value.(string); ok {
				ir.Host = v
			}
		case "port":
			if v, ok := f.Value.(float64); ok {
				ir.Port = int(v)
			}
		case "useSsl":
			if v, ok := f.Value.(bool); ok {
				ir.UseTLS = v
			}
		case "username":
			if v, ok := f.Value.(string); ok {
				ir.Username = v
			}
		case "category", "musicCategory":
			if v, ok := f.Value.(string); ok {
				ir.Category = v
			}
		}
	}

	return ir
}

// diffDownloadClients computes changes needed for download clients using shared logic
func (a *Adapter) diffDownloadClients(current, desired *irv1.IR, changes *adapters.ChangeSet) error {
	// Use shared diff logic
	adapters.DiffDownloadClients(current.DownloadClients, desired.DownloadClients, downloadClientIDMap, changes)
	return nil
}
