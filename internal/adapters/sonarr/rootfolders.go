package sonarr

import (
	"context"

	"github.com/poiley/nebularr-operator/internal/adapters"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

// rootFolderIDMap tracks IDs for root folders
var rootFolderIDMap = make(map[string]int)

// getRootFolders retrieves all root folders
func (a *Adapter) getRootFolders(ctx context.Context, c *httpClient) ([]irv1.RootFolderIR, error) {
	var folders []RootFolderResource
	if err := c.get(ctx, "/api/v3/rootfolder", &folders); err != nil {
		return nil, err
	}

	var result []irv1.RootFolderIR
	for _, f := range folders {
		rootFolderIDMap[f.Path] = f.ID
		result = append(result, irv1.RootFolderIR{
			Path: f.Path,
		})
	}

	return result, nil
}

// diffRootFolders computes changes needed for root folders
func (a *Adapter) diffRootFolders(current, desired *irv1.IR, changes *adapters.ChangeSet) error {
	currentMap := make(map[string]irv1.RootFolderIR)
	for _, rf := range current.RootFolders {
		currentMap[rf.Path] = rf
	}

	desiredMap := make(map[string]bool)
	for _, rf := range desired.RootFolders {
		desiredMap[rf.Path] = true
	}

	// Find creates (paths in desired but not in current)
	for _, rf := range desired.RootFolders {
		if _, exists := currentMap[rf.Path]; !exists {
			changes.Creates = append(changes.Creates, adapters.Change{
				ResourceType: adapters.ResourceRootFolder,
				Name:         rf.Path,
				Payload:      rf,
			})
		}
	}

	// We don't delete root folders automatically as they may contain media
	// This is a safety measure

	return nil
}
