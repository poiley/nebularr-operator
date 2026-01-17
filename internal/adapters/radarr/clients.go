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

// getManagedDownloadClients retrieves download clients tagged with the ownership tag
func (a *Adapter) getManagedDownloadClients(ctx context.Context, c *client.Client, tagID int) ([]irv1.DownloadClientIR, error) {
	resp, err := c.GetApiV3Downloadclient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get download clients: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var clients []client.DownloadClientResource
	if err := json.NewDecoder(resp.Body).Decode(&clients); err != nil {
		return nil, fmt.Errorf("failed to decode download clients: %w", err)
	}

	result := make([]irv1.DownloadClientIR, 0, len(clients))
	for _, dc := range clients {
		// Check if this client has the ownership tag
		if !hasTag(dc.Tags, tagID) {
			continue
		}

		ir := a.downloadClientToIR(&dc)
		result = append(result, ir)
	}

	return result, nil
}

// downloadClientToIR converts a Radarr download client to IR
func (a *Adapter) downloadClientToIR(dc *client.DownloadClientResource) irv1.DownloadClientIR {
	ir := irv1.DownloadClientIR{
		Name:                     ptrToString(dc.Name),
		Enable:                   ptrToBool(dc.Enable),
		Priority:                 ptrToInt(dc.Priority),
		RemoveCompletedDownloads: ptrToBool(dc.RemoveCompletedDownloads),
		RemoveFailedDownloads:    ptrToBool(dc.RemoveFailedDownloads),
	}

	// Determine protocol from implementation
	impl := ptrToString(dc.Implementation)
	ir.Implementation = impl
	ir.Protocol = a.inferProtocol(impl)

	// Extract connection details from fields
	if dc.Fields != nil {
		for _, field := range *dc.Fields {
			name := ptrToString(field.Name)
			switch name {
			case "host":
				if v, ok := field.Value.(string); ok {
					ir.Host = v
				}
			case "port":
				if v, ok := field.Value.(float64); ok {
					ir.Port = int(v)
				}
			case "useSsl":
				if v, ok := field.Value.(bool); ok {
					ir.UseTLS = v
				}
			case "username":
				if v, ok := field.Value.(string); ok {
					ir.Username = v
				}
			case "movieCategory":
				if v, ok := field.Value.(string); ok {
					ir.Category = v
				}
			case "movieDirectory":
				if v, ok := field.Value.(string); ok {
					ir.Directory = v
				}
			}
		}
	}

	return ir
}

// inferProtocol determines the protocol based on implementation
func (a *Adapter) inferProtocol(impl string) string {
	switch impl {
	case "QBittorrent", "Transmission", "Deluge", "RTorrent", "Vuze", "UTorrent", "Aria2", "Flood":
		return irv1.ProtocolTorrent
	case "Sabnzbd", "NzbGet", "NzbVortex", "Pneumatic", "UsenetBlackhole":
		return irv1.ProtocolUsenet
	default:
		return ""
	}
}

// diffDownloadClients computes changes needed for download clients
func (a *Adapter) diffDownloadClients(current, desired *irv1.IR, changes *adapters.ChangeSet) error {
	currentMap := make(map[string]irv1.DownloadClientIR)
	for _, dc := range current.DownloadClients {
		currentMap[dc.Name] = dc
	}

	desiredMap := make(map[string]irv1.DownloadClientIR)
	for _, dc := range desired.DownloadClients {
		desiredMap[dc.Name] = dc
	}

	// Find creates and updates
	for name, desiredDC := range desiredMap {
		currentDC, exists := currentMap[name]
		if !exists {
			changes.Creates = append(changes.Creates, adapters.Change{
				ResourceType: adapters.ResourceDownloadClient,
				Name:         name,
				Payload:      &desiredDC,
			})
		} else if !adapters.DownloadClientsEqual(currentDC, desiredDC) {
			changes.Updates = append(changes.Updates, adapters.Change{
				ResourceType: adapters.ResourceDownloadClient,
				Name:         name,
				Payload:      &desiredDC,
			})
		}
	}

	// Find deletes
	for name := range currentMap {
		if _, exists := desiredMap[name]; !exists {
			changes.Deletes = append(changes.Deletes, adapters.Change{
				ResourceType: adapters.ResourceDownloadClient,
				Name:         name,
			})
		}
	}

	return nil
}
