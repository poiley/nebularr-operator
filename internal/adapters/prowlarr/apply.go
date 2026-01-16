package prowlarr

import (
	"context"
	"fmt"

	"github.com/poiley/nebularr-operator/internal/adapters"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

// applyCreate handles creation of a resource
func (a *Adapter) applyCreate(ctx context.Context, c *httpClient, change adapters.Change, tagID int) error {
	switch change.ResourceType {
	case adapters.ResourceIndexer:
		idx, ok := change.Payload.(irv1.ProwlarrIndexerIR)
		if !ok {
			return fmt.Errorf("invalid payload type for indexer")
		}
		return a.createIndexer(ctx, c, idx, tagID)

	case "IndexerProxy":
		proxy, ok := change.Payload.(irv1.IndexerProxyIR)
		if !ok {
			return fmt.Errorf("invalid payload type for proxy")
		}
		return a.createProxy(ctx, c, proxy, tagID)

	case adapters.ResourceApplication:
		app, ok := change.Payload.(irv1.ProwlarrApplicationIR)
		if !ok {
			return fmt.Errorf("invalid payload type for application")
		}
		return a.createApplication(ctx, c, app, tagID)

	case adapters.ResourceDownloadClient:
		client, ok := change.Payload.(irv1.DownloadClientIR)
		if !ok {
			return fmt.Errorf("invalid payload type for download client")
		}
		return a.createDownloadClient(ctx, c, client, tagID)

	default:
		return fmt.Errorf("unknown resource type: %s", change.ResourceType)
	}
}

// applyUpdate handles updating a resource
func (a *Adapter) applyUpdate(ctx context.Context, c *httpClient, change adapters.Change, tagID int) error {
	switch change.ResourceType {
	case adapters.ResourceIndexer:
		idx, ok := change.Payload.(irv1.ProwlarrIndexerIR)
		if !ok {
			return fmt.Errorf("invalid payload type for indexer")
		}
		return a.updateIndexer(ctx, c, idx, tagID)

	case "IndexerProxy":
		proxy, ok := change.Payload.(irv1.IndexerProxyIR)
		if !ok {
			return fmt.Errorf("invalid payload type for proxy")
		}
		return a.updateProxy(ctx, c, proxy, tagID)

	case adapters.ResourceApplication:
		app, ok := change.Payload.(irv1.ProwlarrApplicationIR)
		if !ok {
			return fmt.Errorf("invalid payload type for application")
		}
		return a.updateApplication(ctx, c, app, tagID)

	case adapters.ResourceDownloadClient:
		client, ok := change.Payload.(irv1.DownloadClientIR)
		if !ok {
			return fmt.Errorf("invalid payload type for download client")
		}
		return a.updateDownloadClient(ctx, c, client, tagID)

	default:
		return fmt.Errorf("unknown resource type: %s", change.ResourceType)
	}
}

// applyDelete handles deletion of a resource
func (a *Adapter) applyDelete(ctx context.Context, c *httpClient, change adapters.Change) error {
	switch change.ResourceType {
	case adapters.ResourceIndexer:
		return a.deleteIndexer(ctx, c, change.Name)

	case "IndexerProxy":
		return a.deleteProxy(ctx, c, change.Name)

	case adapters.ResourceApplication:
		return a.deleteApplication(ctx, c, change.Name)

	case adapters.ResourceDownloadClient:
		return a.deleteDownloadClient(ctx, c, change.Name)

	default:
		return fmt.Errorf("unknown resource type: %s", change.ResourceType)
	}
}
