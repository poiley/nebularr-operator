// Package compiler transforms CRD intent into Intermediate Representation (IR).
// It handles preset expansion, defaults merging, and capability pruning.
package compiler

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/poiley/nebularr-operator/internal/adapters"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
	"github.com/poiley/nebularr-operator/internal/presets"
)

// Compiler transforms CRD intent into IR
type Compiler struct {
	expander *presets.Expander
}

// New creates a new Compiler
func New() *Compiler {
	return &Compiler{
		expander: presets.NewExpander(),
	}
}

// CompileInput holds all inputs for compilation
type CompileInput struct {
	// App identifies which app this is for: radarr, sonarr, lidarr, prowlarr
	App string

	// ConfigName is the name of the config resource (used for profile naming)
	ConfigName string

	// Namespace is the namespace of the config resource
	Namespace string

	// Connection details
	URL    string
	APIKey string

	// Quality configuration
	QualityPreset    string
	QualityOverrides *presets.QualityOverrides

	// Naming configuration
	NamingPreset string

	// Download clients
	DownloadClients []DownloadClientInput

	// Indexers
	Indexers *IndexersInput

	// Root folders
	RootFolders []string

	// Capabilities for pruning unsupported features
	Capabilities *adapters.Capabilities

	// ResolvedSecrets maps secret references to resolved values
	ResolvedSecrets map[string]string
}

// DownloadClientInput holds download client configuration
type DownloadClientInput struct {
	Name                     string
	Implementation           string
	Host                     string
	Port                     int
	UseTLS                   bool
	Username                 string
	Password                 string
	Category                 string
	Priority                 int
	RemoveCompletedDownloads bool
	RemoveFailedDownloads    bool
}

// IndexersInput holds indexer configuration
type IndexersInput struct {
	// ProwlarrRef for delegating to Prowlarr
	ProwlarrRef *ProwlarrRefInput

	// Direct indexers
	Direct []IndexerInput
}

// ProwlarrRefInput references a Prowlarr instance
type ProwlarrRefInput struct {
	ConfigName   string
	AutoRegister bool
	Include      []string
	Exclude      []string
}

// IndexerInput holds direct indexer configuration
type IndexerInput struct {
	Name                    string
	Protocol                string
	Implementation          string
	URL                     string
	APIKey                  string
	Categories              []int
	Priority                int
	MinimumSeeders          int
	SeedRatio               float64
	SeedTimeMinutes         int
	EnableRss               bool
	EnableAutomaticSearch   bool
	EnableInteractiveSearch bool
}

// Compile transforms CRD intent into IR
func (c *Compiler) Compile(ctx context.Context, input CompileInput) (*irv1.IR, error) {
	ir := &irv1.IR{
		Version:     irv1.IRVersion,
		GeneratedAt: time.Now(),
		App:         input.App,
	}

	// 1. Set connection
	ir.Connection = &irv1.ConnectionIR{
		URL:    input.URL,
		APIKey: input.APIKey,
	}

	// 2. Expand quality preset based on app type
	profileName := fmt.Sprintf("nebularr-%s", input.ConfigName)
	switch input.App {
	case adapters.AppRadarr, adapters.AppSonarr:
		presetName := input.QualityPreset
		if presetName == "" {
			presetName = presets.DefaultVideoPreset
		}
		ir.Quality = &irv1.QualityIR{
			Video: c.expander.ExpandVideoPreset(presetName, input.QualityOverrides, profileName),
		}
	case adapters.AppLidarr:
		presetName := input.QualityPreset
		if presetName == "" {
			presetName = presets.DefaultAudioPreset
		}
		ir.Quality = &irv1.QualityIR{
			Audio: c.expander.ExpandAudioPreset(presetName, input.QualityOverrides, profileName),
		}
	}

	// 3. Expand naming preset based on app type
	namingPreset := input.NamingPreset
	if namingPreset == "" {
		namingPreset = presets.DefaultNamingPreset
	}
	switch input.App {
	case adapters.AppRadarr:
		ir.Naming = &irv1.NamingIR{
			Radarr: c.expander.ExpandRadarrNaming(namingPreset),
		}
	case adapters.AppSonarr:
		ir.Naming = &irv1.NamingIR{
			Sonarr: c.expander.ExpandSonarrNaming(namingPreset),
		}
	case adapters.AppLidarr:
		ir.Naming = &irv1.NamingIR{
			Lidarr: c.expander.ExpandLidarrNaming(namingPreset),
		}
	}

	// 4. Compile download clients
	ir.DownloadClients = c.compileDownloadClients(input.DownloadClients, input.ConfigName)

	// 5. Compile indexers
	ir.Indexers = c.compileIndexers(input.Indexers, input.ConfigName)

	// 6. Compile root folders
	for _, path := range input.RootFolders {
		ir.RootFolders = append(ir.RootFolders, irv1.RootFolderIR{Path: path})
	}

	// 7. Prune unsupported features based on capabilities
	if input.Capabilities != nil {
		ir.Unrealized = c.pruneUnsupported(ir, input.Capabilities)
	}

	// 8. Generate source hash for drift detection
	ir.SourceHash = c.hashInput(input)

	return ir, nil
}

// compileDownloadClients converts download client inputs to IR
func (c *Compiler) compileDownloadClients(clients []DownloadClientInput, configName string) []irv1.DownloadClientIR {
	result := make([]irv1.DownloadClientIR, 0, len(clients))

	for _, dc := range clients {
		ir := irv1.DownloadClientIR{
			Name:                     fmt.Sprintf("nebularr-%s-%s", configName, dc.Name),
			Implementation:           dc.Implementation,
			Protocol:                 inferProtocol(dc.Implementation),
			Enable:                   true,
			Priority:                 dc.Priority,
			Host:                     dc.Host,
			Port:                     dc.Port,
			UseTLS:                   dc.UseTLS,
			Username:                 dc.Username,
			Password:                 dc.Password,
			Category:                 dc.Category,
			RemoveCompletedDownloads: dc.RemoveCompletedDownloads,
			RemoveFailedDownloads:    dc.RemoveFailedDownloads,
		}
		result = append(result, ir)
	}

	return result
}

// compileIndexers converts indexer inputs to IR
func (c *Compiler) compileIndexers(input *IndexersInput, configName string) *irv1.IndexersIR {
	if input == nil {
		return nil
	}

	result := &irv1.IndexersIR{}

	// Handle Prowlarr reference
	if input.ProwlarrRef != nil {
		result.ProwlarrRef = &irv1.ProwlarrRefIR{
			ConfigName:   input.ProwlarrRef.ConfigName,
			AutoRegister: input.ProwlarrRef.AutoRegister,
			Include:      input.ProwlarrRef.Include,
			Exclude:      input.ProwlarrRef.Exclude,
		}
		return result
	}

	// Handle direct indexers
	for _, idx := range input.Direct {
		ir := irv1.IndexerIR{
			Name:                    fmt.Sprintf("nebularr-%s-%s", configName, idx.Name),
			Protocol:                idx.Protocol,
			Implementation:          idx.Implementation,
			Enable:                  true,
			Priority:                idx.Priority,
			URL:                     idx.URL,
			APIKey:                  idx.APIKey,
			Categories:              idx.Categories,
			MinimumSeeders:          idx.MinimumSeeders,
			SeedRatio:               idx.SeedRatio,
			SeedTimeMinutes:         idx.SeedTimeMinutes,
			EnableRss:               idx.EnableRss,
			EnableAutomaticSearch:   idx.EnableAutomaticSearch,
			EnableInteractiveSearch: idx.EnableInteractiveSearch,
		}
		result.Direct = append(result.Direct, ir)
	}

	return result
}

// pruneUnsupported removes features not supported by the service capabilities
func (c *Compiler) pruneUnsupported(ir *irv1.IR, caps *adapters.Capabilities) []irv1.UnrealizedFeature {
	var unrealized []irv1.UnrealizedFeature

	// Check video quality tiers against supported resolutions
	if ir.Quality != nil && ir.Quality.Video != nil {
		supportedRes := make(map[string]bool)
		for _, res := range caps.Resolutions {
			supportedRes[res] = true
		}

		prunedTiers := make([]irv1.VideoQualityTierIR, 0)
		for _, tier := range ir.Quality.Video.Tiers {
			if supportedRes[tier.Resolution] {
				prunedTiers = append(prunedTiers, tier)
			} else {
				unrealized = append(unrealized, irv1.UnrealizedFeature{
					Feature: fmt.Sprintf("resolution:%s", tier.Resolution),
					Reason:  "not supported by service",
				})
			}
		}
		ir.Quality.Video.Tiers = prunedTiers
	}

	// Check download client types
	if len(caps.DownloadClientTypes) > 0 {
		supportedTypes := make(map[string]bool)
		for _, t := range caps.DownloadClientTypes {
			supportedTypes[t] = true
		}

		prunedClients := make([]irv1.DownloadClientIR, 0)
		for _, dc := range ir.DownloadClients {
			// Normalize implementation name for comparison
			impl := normalizeImplementation(dc.Implementation)
			if supportedTypes[impl] {
				prunedClients = append(prunedClients, dc)
			} else {
				unrealized = append(unrealized, irv1.UnrealizedFeature{
					Feature: fmt.Sprintf("downloadclient:%s", dc.Implementation),
					Reason:  "not supported by service",
				})
			}
		}
		ir.DownloadClients = prunedClients
	}

	// Check indexer types
	if ir.Indexers != nil && len(caps.IndexerTypes) > 0 {
		supportedTypes := make(map[string]bool)
		for _, t := range caps.IndexerTypes {
			supportedTypes[t] = true
		}

		prunedIndexers := make([]irv1.IndexerIR, 0)
		for _, idx := range ir.Indexers.Direct {
			if supportedTypes[idx.Implementation] {
				prunedIndexers = append(prunedIndexers, idx)
			} else {
				unrealized = append(unrealized, irv1.UnrealizedFeature{
					Feature: fmt.Sprintf("indexer:%s", idx.Implementation),
					Reason:  "not supported by service",
				})
			}
		}
		ir.Indexers.Direct = prunedIndexers
	}

	return unrealized
}

// hashInput generates a deterministic hash of the compilation input
func (c *Compiler) hashInput(input CompileInput) string {
	// Create a simplified struct for hashing (exclude resolved secrets for security)
	hashable := struct {
		App              string
		ConfigName       string
		QualityPreset    string
		QualityOverrides *presets.QualityOverrides
		NamingPreset     string
		DownloadClients  []DownloadClientInput
		Indexers         *IndexersInput
		RootFolders      []string
	}{
		App:              input.App,
		ConfigName:       input.ConfigName,
		QualityPreset:    input.QualityPreset,
		QualityOverrides: input.QualityOverrides,
		NamingPreset:     input.NamingPreset,
		DownloadClients:  input.DownloadClients,
		Indexers:         input.Indexers,
		RootFolders:      input.RootFolders,
	}

	data, err := json.Marshal(hashable)
	if err != nil {
		return ""
	}

	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash[:8]) // Use first 8 bytes (16 hex chars)
}

// inferProtocol determines the protocol from implementation type
func inferProtocol(impl string) string {
	switch impl {
	case "qbittorrent", "QBittorrent", "transmission", "Transmission",
		"deluge", "Deluge", "rtorrent", "RTorrent", "Torznab":
		return irv1.ProtocolTorrent
	case "sabnzbd", "Sabnzbd", "nzbget", "NzbGet", "Newznab":
		return irv1.ProtocolUsenet
	default:
		return ""
	}
}

// normalizeImplementation converts implementation names to their canonical form
func normalizeImplementation(impl string) string {
	switch impl {
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
		return impl
	}
}
