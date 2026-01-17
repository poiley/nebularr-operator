package sonarr

import (
	"context"
	"fmt"

	"github.com/poiley/nebularr-operator/internal/adapters"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

// getManagedNotifications retrieves notifications tagged with the ownership tag
func (a *Adapter) getManagedNotifications(ctx context.Context, c *httpClient, tagID int) ([]irv1.NotificationIR, error) {
	var notifications []NotificationResource
	if err := c.get(ctx, "/api/v3/notification", &notifications); err != nil {
		return nil, fmt.Errorf("failed to get notifications: %w", err)
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

// notificationToIR converts a Sonarr notification to IR
func (a *Adapter) notificationToIR(n *NotificationResource) irv1.NotificationIR {
	ir := irv1.NotificationIR{
		ID:             n.ID,
		Name:           n.Name,
		Implementation: n.Implementation,
		ConfigContract: n.ConfigContract,
		Enabled:        true,

		// Common event triggers
		OnGrab:                      n.OnGrab,
		OnDownload:                  n.OnDownload,
		OnUpgrade:                   n.OnUpgrade,
		OnRename:                    n.OnRename,
		OnHealthIssue:               n.OnHealthIssue,
		OnHealthRestored:            n.OnHealthRestored,
		OnApplicationUpdate:         n.OnApplicationUpdate,
		OnManualInteractionRequired: n.OnManualInteractionRequired,
		IncludeHealthWarnings:       n.IncludeHealthWarnings,

		// Sonarr-specific events
		OnSeriesAdd:                   n.OnSeriesAdd,
		OnSeriesDelete:                n.OnSeriesDelete,
		OnEpisodeFileDelete:           n.OnEpisodeFileDelete,
		OnEpisodeFileDeleteForUpgrade: n.OnEpisodeFileDeleteForUpgrade,

		// Tags
		Tags: n.Tags,
	}

	// Extract fields
	if len(n.Fields) > 0 {
		ir.Fields = make(map[string]interface{})
		for _, field := range n.Fields {
			if field.Name != "" && field.Value != nil {
				ir.Fields[field.Name] = field.Value
			}
		}
	}

	return ir
}

// irToNotification converts IR to a Sonarr notification resource
func (a *Adapter) irToNotification(ir *irv1.NotificationIR, tagID int) NotificationResource {
	n := NotificationResource{
		ID:             ir.ID,
		Name:           ir.Name,
		Implementation: ir.Implementation,
		ConfigContract: ir.ConfigContract,

		// Common event triggers
		OnGrab:                      ir.OnGrab,
		OnDownload:                  ir.OnDownload,
		OnUpgrade:                   ir.OnUpgrade,
		OnRename:                    ir.OnRename,
		OnHealthIssue:               ir.OnHealthIssue,
		OnHealthRestored:            ir.OnHealthRestored,
		OnApplicationUpdate:         ir.OnApplicationUpdate,
		OnManualInteractionRequired: ir.OnManualInteractionRequired,
		IncludeHealthWarnings:       ir.IncludeHealthWarnings,

		// Sonarr-specific events
		OnSeriesAdd:                   ir.OnSeriesAdd,
		OnSeriesDelete:                ir.OnSeriesDelete,
		OnEpisodeFileDelete:           ir.OnEpisodeFileDelete,
		OnEpisodeFileDeleteForUpgrade: ir.OnEpisodeFileDeleteForUpgrade,

		// Tags including ownership tag
		Tags: []int{tagID},
	}

	// Convert fields to API format
	if len(ir.Fields) > 0 {
		fields := make([]Field, 0, len(ir.Fields))
		for name, value := range ir.Fields {
			fields = append(fields, Field{
				Name:  name,
				Value: value,
			})
		}
		n.Fields = fields
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
	// Sonarr-specific events
	if a.OnSeriesAdd != b.OnSeriesAdd || a.OnSeriesDelete != b.OnSeriesDelete {
		return false
	}
	if a.OnEpisodeFileDelete != b.OnEpisodeFileDelete || a.OnEpisodeFileDeleteForUpgrade != b.OnEpisodeFileDeleteForUpgrade {
		return false
	}

	// Compare fields (simplified)
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

// createNotification creates a notification in Sonarr
func (a *Adapter) createNotification(ctx context.Context, c *httpClient, ir *irv1.NotificationIR, tagID int) error {
	notification := a.irToNotification(ir, tagID)

	var created NotificationResource
	if err := c.post(ctx, "/api/v3/notification", notification, &created); err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}

	return nil
}

// updateNotification updates a notification in Sonarr
func (a *Adapter) updateNotification(ctx context.Context, c *httpClient, ir *irv1.NotificationIR, tagID int) error {
	notification := a.irToNotification(ir, tagID)

	endpoint := fmt.Sprintf("/api/v3/notification/%d", ir.ID)
	var updated NotificationResource
	if err := c.put(ctx, endpoint, notification, &updated); err != nil {
		return fmt.Errorf("failed to update notification: %w", err)
	}

	return nil
}

// deleteNotification deletes a notification from Sonarr
func (a *Adapter) deleteNotification(ctx context.Context, c *httpClient, id int) error {
	endpoint := fmt.Sprintf("/api/v3/notification/%d", id)
	if err := c.delete(ctx, endpoint); err != nil {
		return fmt.Errorf("failed to delete notification: %w", err)
	}

	return nil
}
