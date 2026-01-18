// Package shared provides common functionality used across multiple *arr adapters.
package shared

import (
	"github.com/poiley/nebularr-operator/internal/adapters"
)

// ApplyFunc is a function that applies a single change and returns an error if it fails.
type ApplyFunc func(change adapters.Change) error

// ApplyChanges executes creates, updates, and deletes from a ChangeSet using the provided
// callback functions. This consolidates the common apply loop pattern used by all adapters.
//
// The createFn, updateFn, and deleteFn callbacks are called for each change in the
// respective slice. Each callback should handle the adapter-specific logic for that
// operation type.
//
// Returns an ApplyResult with counts of applied/failed operations and any errors.
func ApplyChanges(
	changes *adapters.ChangeSet,
	createFn ApplyFunc,
	updateFn ApplyFunc,
	deleteFn ApplyFunc,
) *adapters.ApplyResult {
	result := &adapters.ApplyResult{}

	// Apply creates
	for _, change := range changes.Creates {
		if err := createFn(change); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, adapters.ApplyError{
				Change: change,
				Error:  err,
			})
		} else {
			result.Applied++
		}
	}

	// Apply updates
	for _, change := range changes.Updates {
		if err := updateFn(change); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, adapters.ApplyError{
				Change: change,
				Error:  err,
			})
		} else {
			result.Applied++
		}
	}

	// Apply deletes
	for _, change := range changes.Deletes {
		if err := deleteFn(change); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, adapters.ApplyError{
				Change: change,
				Error:  err,
			})
		} else {
			result.Applied++
		}
	}

	return result
}
