package compiler

import (
	"fmt"

	"github.com/poiley/nebularr-operator/internal/adapters"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

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
