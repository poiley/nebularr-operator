package compiler

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	arrv1alpha1 "github.com/poiley/nebularr-operator/api/v1alpha1"
	"github.com/poiley/nebularr-operator/internal/adapters"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

// CompileProwlarrConfig compiles a ProwlarrConfig CRD to IR
// Note: Prowlarr is different from other *arr apps - it manages indexers
// natively and syncs them to downstream applications.
func (c *Compiler) CompileProwlarrConfig(ctx context.Context, config *arrv1alpha1.ProwlarrConfig, resolvedSecrets map[string]string, caps *adapters.Capabilities) (*irv1.IR, error) {
	// Create base IR with Prowlarr-specific data
	ir := &irv1.IR{
		Version:     irv1.IRVersion,
		GeneratedAt: time.Now(),
		App:         adapters.AppProwlarr,
		Connection: &irv1.ConnectionIR{
			URL:    config.Spec.Connection.URL,
			APIKey: resolvedSecrets["apiKey"],
		},
		Prowlarr: &irv1.ProwlarrIR{
			Connection: &irv1.ConnectionIR{
				URL:    config.Spec.Connection.URL,
				APIKey: resolvedSecrets["apiKey"],
			},
		},
	}

	// Compile indexers
	ir.Prowlarr.Indexers = compileProwlarrIndexers(config.Spec.Indexers, config.Name, resolvedSecrets)

	// Compile proxies
	ir.Prowlarr.Proxies = compileProwlarrProxies(config.Spec.Proxies, config.Name, resolvedSecrets)

	// Compile applications (pass Prowlarr URL so apps can connect back)
	ir.Prowlarr.Applications = compileProwlarrApplications(config.Spec.Applications, config.Name, config.Spec.Connection.URL, resolvedSecrets)

	// Compile download clients
	ir.Prowlarr.DownloadClients = compileProwlarrDownloadClients(config.Spec.DownloadClients, config.Name, resolvedSecrets)

	return ir, nil
}

// compileProwlarrIndexers converts CRD indexers to IR
func compileProwlarrIndexers(indexers []arrv1alpha1.ProwlarrIndexer, configName string, resolvedSecrets map[string]string) []irv1.ProwlarrIndexerIR {
	result := make([]irv1.ProwlarrIndexerIR, 0, len(indexers))

	for _, idx := range indexers {
		enabled := true
		if idx.Enabled != nil {
			enabled = *idx.Enabled
		}

		priority := idx.Priority
		if priority == 0 {
			priority = 25 // Default priority
		}

		ir := irv1.ProwlarrIndexerIR{
			Name:       fmt.Sprintf("nebularr-%s-%s", configName, idx.Name),
			Definition: idx.Definition,
			Enable:     enabled,
			Priority:   priority,
			BaseURL:    idx.BaseURL,
			Tags:       idx.Tags,
		}

		// Copy settings
		if len(idx.Settings) > 0 {
			ir.Settings = make(map[string]string)
			for k, v := range idx.Settings {
				ir.Settings[k] = v
			}
		}

		// Resolve API key from secret
		if idx.APIKeySecretRef != nil {
			keyName := idx.APIKeySecretRef.Key
			if keyName == "" {
				keyName = "apiKey"
			}
			secretKey := idx.APIKeySecretRef.Name + "/" + keyName
			if apiKey, ok := resolvedSecrets[secretKey]; ok {
				ir.APIKey = apiKey
			}
		}

		result = append(result, ir)
	}

	return result
}

// compileProwlarrProxies converts CRD proxies to IR
func compileProwlarrProxies(proxies []arrv1alpha1.IndexerProxy, configName string, resolvedSecrets map[string]string) []irv1.IndexerProxyIR {
	result := make([]irv1.IndexerProxyIR, 0, len(proxies))

	for _, proxy := range proxies {
		ir := irv1.IndexerProxyIR{
			Name:           fmt.Sprintf("nebularr-%s-%s", configName, proxy.Name),
			Type:           proxy.Type,
			Host:           proxy.Host,
			Port:           proxy.Port,
			RequestTimeout: proxy.RequestTimeout,
			Tags:           proxy.Tags,
		}

		// Set default timeout for FlareSolverr
		if ir.Type == irv1.ProxyTypeFlareSolverr && ir.RequestTimeout == 0 {
			ir.RequestTimeout = 60
		}

		// Resolve credentials from secret
		if proxy.CredentialsSecretRef != nil {
			usernameKey := proxy.CredentialsSecretRef.UsernameKey
			if usernameKey == "" {
				usernameKey = "username"
			}
			passwordKey := proxy.CredentialsSecretRef.PasswordKey
			if passwordKey == "" {
				passwordKey = "password"
			}

			secretPrefix := proxy.CredentialsSecretRef.Name + "/"
			if username, ok := resolvedSecrets[secretPrefix+usernameKey]; ok {
				ir.Username = username
			}
			if password, ok := resolvedSecrets[secretPrefix+passwordKey]; ok {
				ir.Password = password
			}
		}

		result = append(result, ir)
	}

	return result
}

// compileProwlarrApplications converts CRD applications to IR
func compileProwlarrApplications(apps []arrv1alpha1.ProwlarrApplication, configName string, prowlarrURL string, resolvedSecrets map[string]string) []irv1.ProwlarrApplicationIR {
	result := make([]irv1.ProwlarrApplicationIR, 0, len(apps))

	for _, app := range apps {
		syncLevel := app.SyncLevel
		if syncLevel == "" {
			syncLevel = irv1.SyncLevelFullSync
		}

		ir := irv1.ProwlarrApplicationIR{
			Name:        fmt.Sprintf("nebularr-%s-%s", configName, app.Name),
			Type:        app.Type,
			URL:         app.URL,
			ProwlarrURL: prowlarrURL, // Apps need this to connect back to Prowlarr
			SyncLevel:   syncLevel,
		}

		// Convert sync categories
		ir.SyncCategories = convertProwlarrCategories(app.SyncCategories, app.Type)

		// Resolve API key from secret
		if app.APIKeySecretRef != nil {
			keyName := app.APIKeySecretRef.Key
			if keyName == "" {
				keyName = "apiKey"
			}
			secretKey := app.APIKeySecretRef.Name + "/" + keyName
			if apiKey, ok := resolvedSecrets[secretKey]; ok {
				ir.APIKey = apiKey
			}
		}

		result = append(result, ir)
	}

	return result
}

// compileProwlarrDownloadClients converts CRD download clients to IR
func compileProwlarrDownloadClients(clients []arrv1alpha1.DownloadClientSpec, configName string, resolvedSecrets map[string]string) []irv1.DownloadClientIR {
	result := make([]irv1.DownloadClientIR, 0, len(clients))

	for _, dc := range clients {
		// Parse URL to extract host, port, and TLS setting
		host, port, useTLS := parseClientURL(dc.URL)

		// Determine implementation from Type field
		impl := dc.Type
		if impl == "" {
			impl = inferImplementationFromName(dc.Name)
		}

		ir := irv1.DownloadClientIR{
			Name:           fmt.Sprintf("nebularr-%s-%s", configName, dc.Name),
			Implementation: strings.ToLower(impl),
			Protocol:       inferProtocol(impl),
			Enable:         true,
			Priority:       dc.Priority,
			Host:           host,
			Port:           port,
			UseTLS:         useTLS,
			Category:       dc.Category,
		}

		// Resolve credentials from secrets
		if dc.CredentialsSecretRef != nil {
			usernameKey := dc.CredentialsSecretRef.UsernameKey
			if usernameKey == "" {
				usernameKey = "username"
			}
			passwordKey := dc.CredentialsSecretRef.PasswordKey
			if passwordKey == "" {
				passwordKey = "password"
			}

			secretPrefix := dc.CredentialsSecretRef.Name + "/"
			if username, ok := resolvedSecrets[secretPrefix+usernameKey]; ok {
				ir.Username = username
			}
			if password, ok := resolvedSecrets[secretPrefix+passwordKey]; ok {
				ir.Password = password
			}
		}

		result = append(result, ir)
	}

	return result
}

// convertProwlarrCategories converts string categories to numeric IDs
// If no categories specified, returns defaults based on app type
func convertProwlarrCategories(categories []string, appType string) []int {
	if len(categories) == 0 {
		// Return defaults based on app type
		if defaults, ok := irv1.DefaultSyncCategories[appType]; ok {
			return defaults
		}
		return nil
	}

	result := make([]int, 0, len(categories))

	for _, cat := range categories {
		// Try parsing as integer first
		if id, err := strconv.Atoi(cat); err == nil {
			result = append(result, id)
			continue
		}

		// Map human-readable names to IDs
		id := mapProwlarrCategoryName(cat)
		if id > 0 {
			result = append(result, id)
		}
	}

	return result
}

// mapProwlarrCategoryName maps human-readable category names to Newznab IDs
func mapProwlarrCategoryName(name string) int {
	categories := map[string]int{
		// Movies (2000 range)
		"movies":        2000,
		"movies-sd":     2030,
		"movies-hd":     2040,
		"movies-uhd":    2045,
		"movies-4k":     2045,
		"movies-bluray": 2050,
		"movies-webdl":  2010,
		"movies-dvd":    2020,
		"movies-3d":     2060,

		// TV (5000 range)
		"tv":         5000,
		"tv-foreign": 5010,
		"tv-sd":      5030,
		"tv-hd":      5040,
		"tv-uhd":     5045,
		"tv-4k":      5045,
		"tv-webdl":   5010,
		"tv-dvd":     5020,
		"tv-anime":   5070,
		"tv-sport":   5060,

		// Audio/Music (3000 range)
		"audio":          3000,
		"music":          3000,
		"audio-mp3":      3010,
		"music-mp3":      3010,
		"audio-video":    3020,
		"music-video":    3020,
		"audio-lossless": 3040,
		"music-lossless": 3040,
		"audio-flac":     3040,
		"music-flac":     3040,
		"audio-foreign":  3030,
		"music-foreign":  3030,

		// Books (7000 range)
		"books":        7000,
		"books-ebook":  7020,
		"books-comics": 7030,
		"books-mags":   7010,

		// XXX (6000 range)
		"xxx":       6000,
		"xxx-dvd":   6010,
		"xxx-wmv":   6020,
		"xxx-xvid":  6030,
		"xxx-x264":  6040,
		"xxx-other": 6050,

		// Other
		"other": 7010,
		"misc":  7999,
	}

	return categories[strings.ToLower(name)]
}
