package sonarr

import (
	"context"
	"fmt"

	"github.com/poiley/nebularr-operator/internal/adapters/httpclient"
	"github.com/poiley/nebularr-operator/internal/adapters/shared"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

const mediaManagementAPIPath = "/api/v3/config/mediamanagement"

// MediaManagementConfigResource represents Sonarr media management configuration
type MediaManagementConfigResource struct {
	ID                              int    `json:"id,omitempty"`
	RecycleBin                      string `json:"recycleBin"`
	RecycleBinCleanupDays           int    `json:"recycleBinCleanupDays"`
	SetPermissionsLinux             bool   `json:"setPermissionsLinux"`
	ChmodFolder                     string `json:"chmodFolder"`
	ChownGroup                      string `json:"chownGroup"`
	DeleteEmptyFolders              bool   `json:"deleteEmptyFolders"`
	CreateEmptySeriesFolders        bool   `json:"createEmptySeriesFolders"`
	CopyUsingHardlinks              bool   `json:"copyUsingHardlinks"`
	ImportExtraFiles                bool   `json:"importExtraFiles"`
	ExtraFileExtensions             string `json:"extraFileExtensions"`
	DownloadPropersAndRepacks       string `json:"downloadPropersAndRepacks"`
	EnableMediaInfo                 bool   `json:"enableMediaInfo"`
	MinimumFreeSpaceWhenImporting   int    `json:"minimumFreeSpaceWhenImporting"`
	RescanAfterRefresh              string `json:"rescanAfterRefresh"`
	FileDate                        string `json:"fileDate"`
	SkipFreeSpaceCheckWhenImporting bool   `json:"skipFreeSpaceCheckWhenImporting"`
}

// applyMediaManagement applies media management configuration from IR
func (a *Adapter) applyMediaManagement(ctx context.Context, c *httpclient.Client, ir *irv1.MediaManagementIR) error {
	if ir == nil {
		return nil
	}

	// Get current config to preserve ID and unmanaged fields
	current, err := shared.FetchConfig[MediaManagementConfigResource](ctx, c, mediaManagementAPIPath)
	if err != nil {
		return fmt.Errorf("failed to get current media management config: %w", err)
	}

	// Update only the fields we manage
	current.RecycleBin = ir.RecycleBin
	current.RecycleBinCleanupDays = ir.RecycleBinCleanupDays
	current.SetPermissionsLinux = ir.SetPermissions
	current.ChmodFolder = ir.ChmodFolder
	current.ChownGroup = ir.ChownGroup
	current.DeleteEmptyFolders = ir.DeleteEmptyFolders
	current.CreateEmptySeriesFolders = ir.CreateEmptyFolders
	current.CopyUsingHardlinks = ir.UseHardlinks

	return shared.UpdateConfig(ctx, c, mediaManagementAPIPath, current.ID, *current)
}

// getMediaManagementIR converts the current config to IR format
func (a *Adapter) getMediaManagementIR(ctx context.Context, c *httpclient.Client) (*irv1.MediaManagementIR, error) {
	config, err := shared.FetchConfig[MediaManagementConfigResource](ctx, c, mediaManagementAPIPath)
	if err != nil {
		return nil, err
	}

	return &irv1.MediaManagementIR{
		RecycleBin:            config.RecycleBin,
		RecycleBinCleanupDays: config.RecycleBinCleanupDays,
		SetPermissions:        config.SetPermissionsLinux,
		ChmodFolder:           config.ChmodFolder,
		ChownGroup:            config.ChownGroup,
		DeleteEmptyFolders:    config.DeleteEmptyFolders,
		CreateEmptyFolders:    config.CreateEmptySeriesFolders,
		UseHardlinks:          config.CopyUsingHardlinks,
	}, nil
}
