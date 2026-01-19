// Package shared provides common functionality used across multiple *arr adapters.
package shared

import (
	"fmt"

	"github.com/poiley/nebularr-operator/internal/adapters"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

// ImportListStats holds statistics from applying import lists
type ImportListStats struct {
	Created int
	Updated int
	Deleted int
	Skipped int
	Errors  []error
}

// DirectApplyCallbacks contains the callbacks for direct apply operations.
// Each callback is optional - if nil, that operation is skipped.
type DirectApplyCallbacks struct {
	// ApplyImportLists applies import lists and returns stats
	ApplyImportLists func() (*ImportListStats, error)
	// ApplyMediaManagement applies media management config
	ApplyMediaManagement func() error
	// ApplyAuthentication applies authentication config
	ApplyAuthentication func() error
}

// ApplyDirect applies configuration directly from IR using the provided callbacks.
// This handles the common pattern of applying import lists, media management,
// and authentication with proper result tracking.
func ApplyDirect(ir *irv1.IR, callbacks DirectApplyCallbacks) *adapters.ApplyResult {
	result := &adapters.ApplyResult{}

	// Apply import lists if callback provided and there are import lists
	if callbacks.ApplyImportLists != nil && len(ir.ImportLists) > 0 {
		stats, err := callbacks.ApplyImportLists()
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, adapters.ApplyError{
				Change: adapters.Change{ResourceType: adapters.ResourceImportList},
				Error:  fmt.Errorf("failed to apply import lists: %w", err),
			})
		} else if stats != nil {
			result.Applied += stats.Created + stats.Updated + stats.Deleted
			result.Skipped += stats.Skipped
			for _, e := range stats.Errors {
				result.Failed++
				result.Errors = append(result.Errors, adapters.ApplyError{
					Change: adapters.Change{ResourceType: adapters.ResourceImportList},
					Error:  e,
				})
			}
		}
	}

	// Apply media management if callback provided and config exists
	if callbacks.ApplyMediaManagement != nil && ir.MediaManagement != nil {
		if err := callbacks.ApplyMediaManagement(); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, adapters.ApplyError{
				Change: adapters.Change{ResourceType: adapters.ResourceMediaManagement},
				Error:  err,
			})
		} else {
			result.Applied++
		}
	}

	// Apply authentication if callback provided and config exists
	if callbacks.ApplyAuthentication != nil && ir.Authentication != nil {
		if err := callbacks.ApplyAuthentication(); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, adapters.ApplyError{
				Change: adapters.Change{ResourceType: adapters.ResourceAuthentication},
				Error:  err,
			})
		} else {
			result.Applied++
		}
	}

	return result
}
