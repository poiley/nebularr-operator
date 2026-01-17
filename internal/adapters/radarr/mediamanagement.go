package radarr

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/poiley/nebularr-operator/internal/adapters/radarr/client"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

// getMediaManagementConfig fetches the current media management configuration
func (a *Adapter) getMediaManagementConfig(ctx context.Context, c *client.Client) (*client.MediaManagementConfigResource, error) {
	resp, err := c.GetApiV3ConfigMediamanagement(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get media management config: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var config client.MediaManagementConfigResource
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode media management config: %w", err)
	}

	return &config, nil
}

// updateMediaManagementConfig updates the media management configuration
func (a *Adapter) updateMediaManagementConfig(ctx context.Context, c *client.Client, config client.MediaManagementConfigResource) error {
	if config.Id == nil {
		return fmt.Errorf("media management config ID is required")
	}

	idStr := fmt.Sprintf("%d", *config.Id)
	resp, err := c.PutApiV3ConfigMediamanagementId(ctx, idStr, config)
	if err != nil {
		return fmt.Errorf("failed to update media management config: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// applyMediaManagement applies media management configuration from IR
func (a *Adapter) applyMediaManagement(ctx context.Context, c *client.Client, ir *irv1.MediaManagementIR) error {
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
	updated.RecycleBin = stringPtr(ir.RecycleBin)
	updated.RecycleBinCleanupDays = int32Ptr(int32(ir.RecycleBinCleanupDays))
	updated.SetPermissionsLinux = boolPtr(ir.SetPermissions)
	updated.ChmodFolder = stringPtr(ir.ChmodFolder)
	updated.ChownGroup = stringPtr(ir.ChownGroup)
	updated.DeleteEmptyFolders = boolPtr(ir.DeleteEmptyFolders)
	updated.CreateEmptyMovieFolders = boolPtr(ir.CreateEmptyFolders)
	updated.CopyUsingHardlinks = boolPtr(ir.UseHardlinks)

	return a.updateMediaManagementConfig(ctx, c, updated)
}

// getMediaManagementIR converts the current config to IR format
func (a *Adapter) getMediaManagementIR(ctx context.Context, c *client.Client) (*irv1.MediaManagementIR, error) {
	config, err := a.getMediaManagementConfig(ctx, c)
	if err != nil {
		return nil, err
	}

	return a.mediaManagementConfigToIR(config), nil
}

// mediaManagementConfigToIR converts a client MediaManagementConfigResource to IR
func (a *Adapter) mediaManagementConfigToIR(config *client.MediaManagementConfigResource) *irv1.MediaManagementIR {
	ir := &irv1.MediaManagementIR{
		RecycleBin:            ptrToString(config.RecycleBin),
		RecycleBinCleanupDays: int(ptrToInt(config.RecycleBinCleanupDays)),
		SetPermissions:        ptrToBool(config.SetPermissionsLinux),
		ChmodFolder:           ptrToString(config.ChmodFolder),
		ChownGroup:            ptrToString(config.ChownGroup),
		DeleteEmptyFolders:    ptrToBool(config.DeleteEmptyFolders),
		CreateEmptyFolders:    ptrToBool(config.CreateEmptyMovieFolders),
		UseHardlinks:          ptrToBool(config.CopyUsingHardlinks),
	}

	return ir
}

// int32Ptr returns a pointer to an int32 value
func int32Ptr(i int32) *int32 {
	return &i
}
