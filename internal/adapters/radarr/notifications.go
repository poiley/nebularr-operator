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

// getManagedNotifications retrieves notifications tagged with the ownership tag
func (a *Adapter) getManagedNotifications(ctx context.Context, c *client.Client, tagID int) ([]irv1.NotificationIR, error) {
	resp, err := c.GetApiV3Notification(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get notifications: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var notifications []client.NotificationResource
	if err := json.NewDecoder(resp.Body).Decode(&notifications); err != nil {
		return nil, fmt.Errorf("failed to decode notifications: %w", err)
	}

	result := make([]irv1.NotificationIR, 0, len(notifications))
	for _, n := range notifications {
		// Check if this notification has the ownership tag
		if !hasTag(n.Tags, tagID) {
			continue
		}

		ir := a.notificationToIR(&n)
		result = append(result, ir)
	}

	return result, nil
}

// notificationToIR converts a Radarr notification to IR
func (a *Adapter) notificationToIR(n *client.NotificationResource) irv1.NotificationIR {
	ir := irv1.NotificationIR{
		ID:             int(ptrToInt32(n.Id)),
		Name:           ptrToString(n.Name),
		Implementation: ptrToString(n.Implementation),
		ConfigContract: ptrToString(n.ConfigContract),
		Enabled:        true, // If it's returned, it exists

		// Common event triggers
		OnGrab:                      ptrToBool(n.OnGrab),
		OnDownload:                  ptrToBool(n.OnDownload),
		OnUpgrade:                   ptrToBool(n.OnUpgrade),
		OnRename:                    ptrToBool(n.OnRename),
		OnHealthIssue:               ptrToBool(n.OnHealthIssue),
		OnHealthRestored:            ptrToBool(n.OnHealthRestored),
		OnApplicationUpdate:         ptrToBool(n.OnApplicationUpdate),
		OnManualInteractionRequired: ptrToBool(n.OnManualInteractionRequired),
		IncludeHealthWarnings:       ptrToBool(n.IncludeHealthWarnings),

		// Radarr-specific events
		OnMovieAdded:                ptrToBool(n.OnMovieAdded),
		OnMovieDelete:               ptrToBool(n.OnMovieDelete),
		OnMovieFileDelete:           ptrToBool(n.OnMovieFileDelete),
		OnMovieFileDeleteForUpgrade: ptrToBool(n.OnMovieFileDeleteForUpgrade),
	}

	// Extract fields
	if n.Fields != nil {
		ir.Fields = make(map[string]interface{})
		for _, field := range *n.Fields {
			name := ptrToString(field.Name)
			if name != "" && field.Value != nil {
				ir.Fields[name] = field.Value
			}
		}
	}

	// Extract tags
	if n.Tags != nil {
		ir.Tags = make([]int, len(*n.Tags))
		for i, t := range *n.Tags {
			ir.Tags[i] = int(t)
		}
	}

	return ir
}

// irToNotification converts IR to a Radarr notification resource
func (a *Adapter) irToNotification(ir *irv1.NotificationIR, tagID int) client.NotificationResource {
	n := client.NotificationResource{
		Name:           &ir.Name,
		Implementation: &ir.Implementation,
		ConfigContract: &ir.ConfigContract,

		// Common event triggers
		OnGrab:                      &ir.OnGrab,
		OnDownload:                  &ir.OnDownload,
		OnUpgrade:                   &ir.OnUpgrade,
		OnRename:                    &ir.OnRename,
		OnHealthIssue:               &ir.OnHealthIssue,
		OnHealthRestored:            &ir.OnHealthRestored,
		OnApplicationUpdate:         &ir.OnApplicationUpdate,
		OnManualInteractionRequired: &ir.OnManualInteractionRequired,
		IncludeHealthWarnings:       &ir.IncludeHealthWarnings,

		// Radarr-specific events
		OnMovieAdded:                &ir.OnMovieAdded,
		OnMovieDelete:               &ir.OnMovieDelete,
		OnMovieFileDelete:           &ir.OnMovieFileDelete,
		OnMovieFileDeleteForUpgrade: &ir.OnMovieFileDeleteForUpgrade,

		// Tags including ownership tag
		Tags: &[]int32{int32(tagID)},
	}

	// Set ID if updating
	if ir.ID > 0 {
		id := int32(ir.ID)
		n.Id = &id
	}

	// Convert fields to API format
	if len(ir.Fields) > 0 {
		fields := make([]client.Field, 0, len(ir.Fields))
		for name, value := range ir.Fields {
			fieldName := name
			fields = append(fields, client.Field{
				Name:  &fieldName,
				Value: value,
			})
		}
		n.Fields = &fields
	}

	return n
}

// diffNotifications computes changes needed for notifications
func (a *Adapter) diffNotifications(current, desired *irv1.IR, changes *adapters.ChangeSet) error {
	currentMap := make(map[string]irv1.NotificationIR)
	for _, n := range current.Notifications {
		currentMap[n.Name] = n
	}

	desiredMap := make(map[string]irv1.NotificationIR)
	for _, n := range desired.Notifications {
		desiredMap[n.Name] = n
	}

	// Find creates and updates
	for name, desiredN := range desiredMap {
		currentN, exists := currentMap[name]
		if !exists {
			changes.Creates = append(changes.Creates, adapters.Change{
				ResourceType: adapters.ResourceNotification,
				Name:         name,
				Payload:      &desiredN,
			})
		} else if !notificationsEqual(currentN, desiredN) {
			desiredN.ID = currentN.ID // Preserve the ID for update
			id := currentN.ID
			changes.Updates = append(changes.Updates, adapters.Change{
				ResourceType: adapters.ResourceNotification,
				Name:         name,
				ID:           &id,
				Payload:      &desiredN,
			})
		}
	}

	// Find deletes
	for name, currentN := range currentMap {
		if _, exists := desiredMap[name]; !exists {
			id := currentN.ID
			changes.Deletes = append(changes.Deletes, adapters.Change{
				ResourceType: adapters.ResourceNotification,
				Name:         name,
				ID:           &id,
			})
		}
	}

	return nil
}

// notificationsEqual checks if two notifications are equal (ignoring ID)
func notificationsEqual(a, b irv1.NotificationIR) bool {
	// Compare implementation and events
	if a.Implementation != b.Implementation {
		return false
	}
	if a.OnGrab != b.OnGrab || a.OnDownload != b.OnDownload || a.OnUpgrade != b.OnUpgrade {
		return false
	}
	if a.OnRename != b.OnRename || a.OnHealthIssue != b.OnHealthIssue || a.OnHealthRestored != b.OnHealthRestored {
		return false
	}
	if a.OnApplicationUpdate != b.OnApplicationUpdate || a.OnManualInteractionRequired != b.OnManualInteractionRequired {
		return false
	}
	if a.IncludeHealthWarnings != b.IncludeHealthWarnings {
		return false
	}
	// Radarr-specific events
	if a.OnMovieAdded != b.OnMovieAdded || a.OnMovieDelete != b.OnMovieDelete {
		return false
	}
	if a.OnMovieFileDelete != b.OnMovieFileDelete || a.OnMovieFileDeleteForUpgrade != b.OnMovieFileDeleteForUpgrade {
		return false
	}

	// Compare fields (simplified - could be more thorough)
	if len(a.Fields) != len(b.Fields) {
		return false
	}
	for k, v := range a.Fields {
		if bv, ok := b.Fields[k]; !ok || fmt.Sprintf("%v", v) != fmt.Sprintf("%v", bv) {
			return false
		}
	}

	return true
}

// createNotification creates a notification in Radarr
func (a *Adapter) createNotification(ctx context.Context, c *client.Client, ir *irv1.NotificationIR, tagID int) error {
	notification := a.irToNotification(ir, tagID)

	resp, err := c.PostApiV3Notification(ctx, nil, notification)
	if err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// updateNotification updates a notification in Radarr
func (a *Adapter) updateNotification(ctx context.Context, c *client.Client, ir *irv1.NotificationIR, tagID int) error {
	notification := a.irToNotification(ir, tagID)

	resp, err := c.PutApiV3NotificationId(ctx, int32(ir.ID), nil, notification)
	if err != nil {
		return fmt.Errorf("failed to update notification: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// deleteNotification deletes a notification from Radarr
func (a *Adapter) deleteNotification(ctx context.Context, c *client.Client, id int) error {
	resp, err := c.DeleteApiV3NotificationId(ctx, int32(id))
	if err != nil {
		return fmt.Errorf("failed to delete notification: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// ptrToInt32 safely dereferences an int32 pointer
func ptrToInt32(p *int32) int32 {
	if p == nil {
		return 0
	}
	return *p
}
