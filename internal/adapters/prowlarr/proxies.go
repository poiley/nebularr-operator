package prowlarr

import (
	"context"
	"fmt"

	"github.com/poiley/nebularr-operator/internal/adapters"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

// Package-level cache for proxy IDs
var proxyIDCache = make(map[string]int) // "baseURL:name" -> ID

// getManagedProxies retrieves proxies tagged with ownership tag
func (a *Adapter) getManagedProxies(ctx context.Context, c *httpClient, tagID int) ([]irv1.IndexerProxyIR, error) {
	var proxies []IndexerProxyResource
	if err := c.get(ctx, "/api/v1/indexerproxy", &proxies); err != nil {
		return nil, fmt.Errorf("failed to get proxies: %w", err)
	}

	var managed []irv1.IndexerProxyIR
	for _, proxy := range proxies {
		if !hasTag(proxy.Tags, tagID) {
			continue
		}

		// Cache the ID
		cacheKey := fmt.Sprintf("%s:%s", c.baseURL, proxy.Name)
		proxyIDCache[cacheKey] = proxy.ID

		// Convert to IR
		ir := irv1.IndexerProxyIR{
			Name: proxy.Name,
			Type: implToProxyType(proxy.Implementation),
		}

		// Extract settings from fields
		for _, field := range proxy.Fields {
			switch field.Name {
			case "host":
				if v, ok := field.Value.(string); ok {
					ir.Host = v
				}
			case "port":
				if v, ok := field.Value.(float64); ok {
					ir.Port = int(v)
				}
			case "username":
				if v, ok := field.Value.(string); ok {
					ir.Username = v
				}
			case "password":
				if v, ok := field.Value.(string); ok {
					ir.Password = v
				}
			case "requestTimeout":
				if v, ok := field.Value.(float64); ok {
					ir.RequestTimeout = int(v)
				}
			}
		}

		managed = append(managed, ir)
	}

	return managed, nil
}

// implToProxyType converts implementation name to IR proxy type
func implToProxyType(impl string) string {
	switch impl {
	case ProxyImplFlareSolverr:
		return irv1.ProxyTypeFlareSolverr
	case ProxyImplHTTP:
		return irv1.ProxyTypeHTTP
	case ProxyImplSocks4:
		return irv1.ProxyTypeSocks4
	case ProxyImplSocks5:
		return irv1.ProxyTypeSocks5
	default:
		return impl
	}
}

// proxyTypeToImpl converts IR proxy type to implementation name
func proxyTypeToImpl(proxyType string) string {
	switch proxyType {
	case irv1.ProxyTypeFlareSolverr:
		return ProxyImplFlareSolverr
	case irv1.ProxyTypeHTTP:
		return ProxyImplHTTP
	case irv1.ProxyTypeSocks4:
		return ProxyImplSocks4
	case irv1.ProxyTypeSocks5:
		return ProxyImplSocks5
	default:
		return proxyType
	}
}

// diffProxies computes changes needed for proxies
func (a *Adapter) diffProxies(current, desired *irv1.ProwlarrIR, changes *adapters.ChangeSet) error {
	currentByName := make(map[string]irv1.IndexerProxyIR)
	for _, proxy := range current.Proxies {
		currentByName[proxy.Name] = proxy
	}

	desiredByName := make(map[string]irv1.IndexerProxyIR)
	for _, proxy := range desired.Proxies {
		desiredByName[proxy.Name] = proxy
	}

	// Find creates and updates
	for name, desiredProxy := range desiredByName {
		currentProxy, exists := currentByName[name]
		if !exists {
			// Create
			changes.Creates = append(changes.Creates, adapters.Change{
				ResourceType: "IndexerProxy",
				Name:         name,
				Payload:      desiredProxy,
			})
		} else if !proxiesEqual(currentProxy, desiredProxy) {
			// Update
			changes.Updates = append(changes.Updates, adapters.Change{
				ResourceType: "IndexerProxy",
				Name:         name,
				Payload:      desiredProxy,
			})
		}
	}

	// Find deletes
	for name := range currentByName {
		if _, exists := desiredByName[name]; !exists {
			changes.Deletes = append(changes.Deletes, adapters.Change{
				ResourceType: "IndexerProxy",
				Name:         name,
			})
		}
	}

	return nil
}

// proxiesEqual compares two proxies for equality
func proxiesEqual(a, b irv1.IndexerProxyIR) bool {
	return a.Type == b.Type &&
		a.Host == b.Host &&
		a.Port == b.Port &&
		a.Username == b.Username &&
		a.RequestTimeout == b.RequestTimeout
	// Note: Password is not compared (secret)
}

// createProxy creates a proxy in Prowlarr
func (a *Adapter) createProxy(ctx context.Context, c *httpClient, proxy irv1.IndexerProxyIR, tagID int) error {
	resource := IndexerProxyResource{
		Name:           proxy.Name,
		Implementation: proxyTypeToImpl(proxy.Type),
		Tags:           []int{tagID},
	}

	// Build fields based on proxy type
	resource.Fields = buildProxyFields(proxy)

	var created IndexerProxyResource
	if err := c.post(ctx, "/api/v1/indexerproxy", resource, &created); err != nil {
		return fmt.Errorf("failed to create proxy %s: %w", proxy.Name, err)
	}

	// Cache the ID
	cacheKey := fmt.Sprintf("%s:%s", c.baseURL, proxy.Name)
	proxyIDCache[cacheKey] = created.ID

	return nil
}

// updateProxy updates an existing proxy
func (a *Adapter) updateProxy(ctx context.Context, c *httpClient, proxy irv1.IndexerProxyIR, tagID int) error {
	cacheKey := fmt.Sprintf("%s:%s", c.baseURL, proxy.Name)
	id, ok := proxyIDCache[cacheKey]
	if !ok {
		var proxies []IndexerProxyResource
		if err := c.get(ctx, "/api/v1/indexerproxy", &proxies); err != nil {
			return fmt.Errorf("failed to get proxies: %w", err)
		}
		for _, existing := range proxies {
			if existing.Name == proxy.Name {
				id = existing.ID
				proxyIDCache[cacheKey] = id
				break
			}
		}
		if id == 0 {
			return fmt.Errorf("proxy %s not found", proxy.Name)
		}
	}

	resource := IndexerProxyResource{
		ID:             id,
		Name:           proxy.Name,
		Implementation: proxyTypeToImpl(proxy.Type),
		Tags:           []int{tagID},
	}

	resource.Fields = buildProxyFields(proxy)

	path := fmt.Sprintf("/api/v1/indexerproxy/%d", id)
	if err := c.put(ctx, path, resource, nil); err != nil {
		return fmt.Errorf("failed to update proxy %s: %w", proxy.Name, err)
	}

	return nil
}

// deleteProxy deletes a proxy
func (a *Adapter) deleteProxy(ctx context.Context, c *httpClient, name string) error {
	cacheKey := fmt.Sprintf("%s:%s", c.baseURL, name)
	id, ok := proxyIDCache[cacheKey]
	if !ok {
		var proxies []IndexerProxyResource
		if err := c.get(ctx, "/api/v1/indexerproxy", &proxies); err != nil {
			return fmt.Errorf("failed to get proxies: %w", err)
		}
		for _, existing := range proxies {
			if existing.Name == name {
				id = existing.ID
				break
			}
		}
		if id == 0 {
			return nil // Already deleted
		}
	}

	path := fmt.Sprintf("/api/v1/indexerproxy/%d", id)
	if err := c.delete(ctx, path); err != nil {
		return fmt.Errorf("failed to delete proxy %s: %w", name, err)
	}

	delete(proxyIDCache, cacheKey)
	return nil
}

// buildProxyFields builds fields for a proxy based on its type
func buildProxyFields(proxy irv1.IndexerProxyIR) []IndexerProxyField {
	var fields []IndexerProxyField

	switch proxy.Type {
	case irv1.ProxyTypeFlareSolverr:
		fields = append(fields, IndexerProxyField{
			Name:  "host",
			Value: proxy.Host, // Full URL for FlareSolverr
		})
		if proxy.RequestTimeout > 0 {
			fields = append(fields, IndexerProxyField{
				Name:  "requestTimeout",
				Value: proxy.RequestTimeout,
			})
		}
	default:
		// HTTP/SOCKS proxies
		if proxy.Host != "" {
			fields = append(fields, IndexerProxyField{
				Name:  "host",
				Value: proxy.Host,
			})
		}
		if proxy.Port > 0 {
			fields = append(fields, IndexerProxyField{
				Name:  "port",
				Value: proxy.Port,
			})
		}
		if proxy.Username != "" {
			fields = append(fields, IndexerProxyField{
				Name:  "username",
				Value: proxy.Username,
			})
		}
		if proxy.Password != "" {
			fields = append(fields, IndexerProxyField{
				Name:  "password",
				Value: proxy.Password,
			})
		}
	}

	return fields
}
