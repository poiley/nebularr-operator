package lidarr

import (
	"context"

	"github.com/poiley/nebularr-operator/internal/adapters"
	"github.com/poiley/nebularr-operator/internal/adapters/httpclient"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

// getRootFolders retrieves all root folders
func (a *Adapter) getRootFolders(ctx context.Context, c *httpclient.Client) ([]irv1.RootFolderIR, error) {
	var folders []RootFolderResource
	if err := c.Get(ctx, "/api/v1/rootfolder", &folders); err != nil {
		return nil, err
	}

	result := make([]irv1.RootFolderIR, 0, len(folders))
	for _, f := range folders {
		result = append(result, irv1.RootFolderIR{
			Path:           f.Path,
			Name:           f.Name,
			DefaultMonitor: f.DefaultMonitorOption,
		})
	}

	return result, nil
}

// diffRootFolders computes changes needed for root folders using shared logic
func (a *Adapter) diffRootFolders(current, desired *irv1.IR, changes *adapters.ChangeSet) error {
	// Use shared diff logic (create-only, no deletes for safety)
	adapters.DiffRootFolders(current.RootFolders, desired.RootFolders, changes)
	return nil
}
