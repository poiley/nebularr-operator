package sonarr

import (
	"context"
	"fmt"

	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

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

// getMediaManagementConfig fetches the current media management configuration
func (a *Adapter) getMediaManagementConfig(ctx context.Context, c *httpClient) (*MediaManagementConfigResource, error) {
	var config MediaManagementConfigResource
	if err := c.get(ctx, "/api/v3/config/mediamanagement", &config); err != nil {
		return nil, fmt.Errorf("failed to get media management config: %w", err)
	}
	return &config, nil
}

// updateMediaManagementConfig updates the media management configuration
func (a *Adapter) updateMediaManagementConfig(ctx context.Context, c *httpClient, config MediaManagementConfigResource) error {
	path := fmt.Sprintf("/api/v3/config/mediamanagement/%d", config.ID)
	var result MediaManagementConfigResource
	return c.put(ctx, path, config, &result)
}

// applyMediaManagement applies media management configuration from IR
func (a *Adapter) applyMediaManagement(ctx context.Context, c *httpClient, ir *irv1.MediaManagementIR) error {
	if ir == nil {
		return nil
	}

	// Get current config to preserve ID and unmanaged fields
	current, err := a.getMediaManagementConfig(ctx, c)
	if err != nil {
		return fmt.Errorf("failed to get current media management config: %w", err)
	}

	// Update only the fields we manage
	updated := *current
	updated.RecycleBin = ir.RecycleBin
	updated.RecycleBinCleanupDays = ir.RecycleBinCleanupDays
	updated.SetPermissionsLinux = ir.SetPermissions
	updated.ChmodFolder = ir.ChmodFolder
	updated.ChownGroup = ir.ChownGroup
	updated.DeleteEmptyFolders = ir.DeleteEmptyFolders
	updated.CreateEmptySeriesFolders = ir.CreateEmptyFolders
	updated.CopyUsingHardlinks = ir.UseHardlinks

	return a.updateMediaManagementConfig(ctx, c, updated)
}

// getMediaManagementIR converts the current config to IR format
func (a *Adapter) getMediaManagementIR(ctx context.Context, c *httpClient) (*irv1.MediaManagementIR, error) {
	config, err := a.getMediaManagementConfig(ctx, c)
	if err != nil {
		return nil, err
	}

	return a.mediaManagementConfigToIR(config), nil
}

// mediaManagementConfigToIR converts a MediaManagementConfigResource to IR
func (a *Adapter) mediaManagementConfigToIR(config *MediaManagementConfigResource) *irv1.MediaManagementIR {
	return &irv1.MediaManagementIR{
		RecycleBin:            config.RecycleBin,
		RecycleBinCleanupDays: config.RecycleBinCleanupDays,
		SetPermissions:        config.SetPermissionsLinux,
		ChmodFolder:           config.ChmodFolder,
		ChownGroup:            config.ChownGroup,
		DeleteEmptyFolders:    config.DeleteEmptyFolders,
		CreateEmptyFolders:    config.CreateEmptySeriesFolders,
		UseHardlinks:          config.CopyUsingHardlinks,
	}
}
