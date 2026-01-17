package prowlarr

import (
	"context"
	"fmt"
	"strings"

	"github.com/poiley/nebularr-operator/internal/adapters"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

// Package-level cache for download client IDs
var downloadClientIDCache = make(map[string]int) // "baseURL:name" -> ID

// getManagedDownloadClients retrieves download clients tagged with ownership tag
func (a *Adapter) getManagedDownloadClients(ctx context.Context, c *httpClient, tagID int) ([]irv1.DownloadClientIR, error) {
	var clients []DownloadClientResource
	if err := c.get(ctx, "/api/v1/downloadclient", &clients); err != nil {
		return nil, fmt.Errorf("failed to get download clients: %w", err)
	}

	managed := make([]irv1.DownloadClientIR, 0, len(clients))
	for _, client := range clients {
		if !hasTag(client.Tags, tagID) {
			continue
		}

		// Cache the ID
		cacheKey := fmt.Sprintf("%s:%s", c.baseURL, client.Name)
		downloadClientIDCache[cacheKey] = client.ID

		// Convert to IR
		ir := irv1.DownloadClientIR{
			Name:           client.Name,
			Protocol:       client.Protocol,
			Implementation: strings.ToLower(client.Implementation),
			Enable:         client.Enable,
			Priority:       client.Priority,
		}

		// Extract settings from fields
		for _, field := range client.Fields {
			switch field.Name {
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
			case "password":
				if v, ok := field.Value.(string); ok {
					ir.Password = v
				}
			case "category":
				if v, ok := field.Value.(string); ok {
					ir.Category = v
				}
			case "directory":
				if v, ok := field.Value.(string); ok {
					ir.Directory = v
				}
			}
		}

		managed = append(managed, ir)
	}

	return managed, nil
}

// diffDownloadClients computes changes needed for download clients
func (a *Adapter) diffDownloadClients(current, desired *irv1.ProwlarrIR, changes *adapters.ChangeSet) error {
	currentByName := make(map[string]irv1.DownloadClientIR)
	for _, client := range current.DownloadClients {
		currentByName[client.Name] = client
	}

	desiredByName := make(map[string]irv1.DownloadClientIR)
	for _, client := range desired.DownloadClients {
		desiredByName[client.Name] = client
	}

	// Find creates and updates
	for name, desiredClient := range desiredByName {
		currentClient, exists := currentByName[name]
		if !exists {
			// Create
			changes.Creates = append(changes.Creates, adapters.Change{
				ResourceType: adapters.ResourceDownloadClient,
				Name:         name,
				Payload:      desiredClient,
			})
		} else if !downloadClientsEqual(currentClient, desiredClient) {
			// Update
			changes.Updates = append(changes.Updates, adapters.Change{
				ResourceType: adapters.ResourceDownloadClient,
				Name:         name,
				Payload:      desiredClient,
			})
		}
	}

	// Find deletes
	for name := range currentByName {
		if _, exists := desiredByName[name]; !exists {
			changes.Deletes = append(changes.Deletes, adapters.Change{
				ResourceType: adapters.ResourceDownloadClient,
				Name:         name,
			})
		}
	}

	return nil
}

// downloadClientsEqual compares two download clients for equality
func downloadClientsEqual(a, b irv1.DownloadClientIR) bool {
	return a.Protocol == b.Protocol &&
		a.Implementation == b.Implementation &&
		a.Enable == b.Enable &&
		a.Priority == b.Priority &&
		a.Host == b.Host &&
		a.Port == b.Port &&
		a.UseTLS == b.UseTLS &&
		a.Username == b.Username &&
		a.Category == b.Category &&
		a.Directory == b.Directory
	// Note: Password is not compared (secret)
}

// createDownloadClient creates a download client in Prowlarr
func (a *Adapter) createDownloadClient(ctx context.Context, c *httpClient, client irv1.DownloadClientIR, tagID int) error {
	resource := DownloadClientResource{
		Name:           client.Name,
		Implementation: implFromClientType(client.Implementation),
		Protocol:       client.Protocol,
		Enable:         client.Enable,
		Priority:       client.Priority,
		Tags:           []int{tagID},
	}

	// Build fields
	resource.Fields = buildDownloadClientFields(client)

	var created DownloadClientResource
	if err := c.post(ctx, "/api/v1/downloadclient", resource, &created); err != nil {
		return fmt.Errorf("failed to create download client %s: %w", client.Name, err)
	}

	// Cache the ID
	cacheKey := fmt.Sprintf("%s:%s", c.baseURL, client.Name)
	downloadClientIDCache[cacheKey] = created.ID

	return nil
}

// updateDownloadClient updates an existing download client
func (a *Adapter) updateDownloadClient(ctx context.Context, c *httpClient, client irv1.DownloadClientIR, tagID int) error {
	cacheKey := fmt.Sprintf("%s:%s", c.baseURL, client.Name)
	id, ok := downloadClientIDCache[cacheKey]
	if !ok {
		var clients []DownloadClientResource
		if err := c.get(ctx, "/api/v1/downloadclient", &clients); err != nil {
			return fmt.Errorf("failed to get download clients: %w", err)
		}
		for _, existing := range clients {
			if existing.Name == client.Name {
				id = existing.ID
				downloadClientIDCache[cacheKey] = id
				break
			}
		}
		if id == 0 {
			return fmt.Errorf("download client %s not found", client.Name)
		}
	}

	resource := DownloadClientResource{
		ID:             id,
		Name:           client.Name,
		Implementation: implFromClientType(client.Implementation),
		Protocol:       client.Protocol,
		Enable:         client.Enable,
		Priority:       client.Priority,
		Tags:           []int{tagID},
	}

	resource.Fields = buildDownloadClientFields(client)

	path := fmt.Sprintf("/api/v1/downloadclient/%d", id)
	if err := c.put(ctx, path, resource, nil); err != nil {
		return fmt.Errorf("failed to update download client %s: %w", client.Name, err)
	}

	return nil
}

// deleteDownloadClient deletes a download client
func (a *Adapter) deleteDownloadClient(ctx context.Context, c *httpClient, name string) error {
	cacheKey := fmt.Sprintf("%s:%s", c.baseURL, name)
	id, ok := downloadClientIDCache[cacheKey]
	if !ok {
		var clients []DownloadClientResource
		if err := c.get(ctx, "/api/v1/downloadclient", &clients); err != nil {
			return fmt.Errorf("failed to get download clients: %w", err)
		}
		for _, existing := range clients {
			if existing.Name == name {
				id = existing.ID
				break
			}
		}
		if id == 0 {
			return nil // Already deleted
		}
	}

	path := fmt.Sprintf("/api/v1/downloadclient/%d", id)
	if err := c.delete(ctx, path); err != nil {
		return fmt.Errorf("failed to delete download client %s: %w", name, err)
	}

	delete(downloadClientIDCache, cacheKey)
	return nil
}

// implFromClientType converts IR client type to implementation name
func implFromClientType(clientType string) string {
	switch strings.ToLower(clientType) {
	case "qbittorrent":
		return "QBittorrent"
	case "transmission":
		return "Transmission"
	case "deluge":
		return "Deluge"
	case "rtorrent":
		return "RTorrent"
	case "sabnzbd":
		return "Sabnzbd"
	case "nzbget":
		return "NzbGet"
	default:
		return clientType
	}
}

// buildDownloadClientFields builds fields for a download client
func buildDownloadClientFields(client irv1.DownloadClientIR) []DownloadClientField {
	var fields []DownloadClientField

	if client.Host != "" {
		fields = append(fields, DownloadClientField{
			Name:  "host",
			Value: client.Host,
		})
	}

	if client.Port > 0 {
		fields = append(fields, DownloadClientField{
			Name:  "port",
			Value: client.Port,
		})
	}

	fields = append(fields, DownloadClientField{
		Name:  "useSsl",
		Value: client.UseTLS,
	})

	if client.Username != "" {
		fields = append(fields, DownloadClientField{
			Name:  "username",
			Value: client.Username,
		})
	}

	if client.Password != "" {
		fields = append(fields, DownloadClientField{
			Name:  "password",
			Value: client.Password,
		})
	}

	if client.Category != "" {
		fields = append(fields, DownloadClientField{
			Name:  "category",
			Value: client.Category,
		})
	}

	if client.Directory != "" {
		fields = append(fields, DownloadClientField{
			Name:  "directory",
			Value: client.Directory,
		})
	}

	return fields
}
