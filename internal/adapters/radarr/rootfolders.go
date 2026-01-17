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

// getRootFolders retrieves all root folders from Radarr
func (a *Adapter) getRootFolders(ctx context.Context, c *client.Client) ([]irv1.RootFolderIR, error) {
	resp, err := c.GetApiV3Rootfolder(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get root folders: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var folders []client.RootFolderResource
	if err := json.NewDecoder(resp.Body).Decode(&folders); err != nil {
		return nil, fmt.Errorf("failed to decode root folders: %w", err)
	}

	result := make([]irv1.RootFolderIR, 0, len(folders))
	for _, folder := range folders {
		ir := irv1.RootFolderIR{
			Path: ptrToString(folder.Path),
		}
		result = append(result, ir)
	}

	return result, nil
}

// diffRootFolders computes changes needed for root folders
func (a *Adapter) diffRootFolders(current, desired *irv1.IR, changes *adapters.ChangeSet) error {
	// Build maps for comparison (by path)
	currentFolders := make(map[string]bool)
	desiredFolders := make(map[string]bool)

	for _, folder := range current.RootFolders {
		currentFolders[folder.Path] = true
	}

	for _, folder := range desired.RootFolders {
		desiredFolders[folder.Path] = true
	}

	// Find creates
	for path := range desiredFolders {
		if !currentFolders[path] {
			changes.Creates = append(changes.Creates, adapters.Change{
				ResourceType: adapters.ResourceRootFolder,
				Name:         path,
				Payload:      &irv1.RootFolderIR{Path: path},
			})
		}
	}

	// Note: We don't delete root folders as they may have content
	// Only create missing ones

	return nil
}
