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
	case adapters.AppReadarr:
		// Readarr uses book quality profiles
		if input.BookQuality != nil {
			ir.Quality = &irv1.QualityIR{
				Book: c.compileBookQuality(input.BookQuality, profileName),
			}
		}
		// Handle metadata profiles
		if input.MetadataProfile != nil {
			ir.MetadataProfiles = []*irv1.MetadataProfileIR{
				c.compileMetadataProfile(input.MetadataProfile, profileName),
			}
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

	// 5. Compile remote path mappings
	ir.RemotePathMappings = c.compileRemotePathMappings(input.RemotePathMappings)

	// 6. Compile indexers
	ir.Indexers = c.compileIndexers(input.Indexers, input.ConfigName)

	// 7. Compile root folders
	for _, path := range input.RootFolders {
		ir.RootFolders = append(ir.RootFolders, irv1.RootFolderIR{Path: path})
	}

	// 8. Compile import lists
	ir.ImportLists = c.compileImportListsToIR(input.ImportLists)

	// 9. Compile media management
	ir.MediaManagement = c.compileMediaManagementToIR(input.MediaManagement)

	// 10. Compile authentication
	ir.Authentication = c.compileAuthenticationToIR(input.Authentication)

	// 11. Compile notifications
	ir.Notifications = c.compileNotificationsToIR(input.Notifications, input.ConfigName)

	// 12. Compile custom formats (Radarr/Sonarr/Lidarr)
	if input.App == adapters.AppRadarr || input.App == adapters.AppSonarr || input.App == adapters.AppLidarr {
		ir.CustomFormats = c.compileCustomFormatsToIR(input.CustomFormats, input.ConfigName)

		// Populate format scores in quality profile from custom format scores
		if ir.Quality != nil && len(input.CustomFormats) > 0 {
			if ir.Quality.Video != nil {
				ir.Quality.Video.FormatScores = c.compileFormatScores(input.CustomFormats, input.ConfigName)
			}
			if ir.Quality.Audio != nil {
				ir.Quality.Audio.FormatScores = c.compileFormatScores(input.CustomFormats, input.ConfigName)
			}
		}
	}

	// 13. Compile delay profiles (Radarr/Sonarr/Lidarr)
	if input.App == adapters.AppRadarr || input.App == adapters.AppSonarr || input.App == adapters.AppLidarr {
		ir.DelayProfiles = c.compileDelayProfilesToIR(input.DelayProfiles)
	}

	// 14. Prune unsupported features based on capabilities
	if input.Capabilities != nil {
		ir.Unrealized = c.pruneUnsupported(ir, input.Capabilities)
	}

	// 15. Generate source hash for drift detection
	ir.SourceHash = c.hashInput(input)

	return ir, nil
}

// hashInput generates a deterministic hash of the compilation input
func (c *Compiler) hashInput(input CompileInput) string {
	// Create a simplified struct for hashing (exclude resolved secrets for security)
	hashable := struct {
		App                string
		ConfigName         string
		QualityPreset      string
		QualityOverrides   *presets.QualityOverrides
		NamingPreset       string
		DownloadClients    []DownloadClientInput
		RemotePathMappings []RemotePathMappingInput
		Indexers           *IndexersInput
		RootFolders        []string
		Notifications      []NotificationInput
		CustomFormats      []CustomFormatInput
		DelayProfiles      []DelayProfileInput
	}{
		App:                input.App,
		ConfigName:         input.ConfigName,
		QualityPreset:      input.QualityPreset,
		QualityOverrides:   input.QualityOverrides,
		NamingPreset:       input.NamingPreset,
		DownloadClients:    input.DownloadClients,
		RemotePathMappings: input.RemotePathMappings,
		Indexers:           input.Indexers,
		RootFolders:        input.RootFolders,
		Notifications:      input.Notifications,
		CustomFormats:      input.CustomFormats,
		DelayProfiles:      input.DelayProfiles,
	}

	data, err := json.Marshal(hashable)
	if err != nil {
		return ""
	}

	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash[:8]) // Use first 8 bytes (16 hex chars)
}
