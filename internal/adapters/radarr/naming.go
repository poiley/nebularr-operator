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

// getNamingConfig retrieves the naming configuration from Radarr
func (a *Adapter) getNamingConfig(ctx context.Context, c *client.Client) (*irv1.RadarrNamingIR, error) {
	resp, err := c.GetApiV3ConfigNaming(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get naming config: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var naming client.NamingConfigResource
	if err := json.NewDecoder(resp.Body).Decode(&naming); err != nil {
		return nil, fmt.Errorf("failed to decode naming config: %w", err)
	}

	ir := &irv1.RadarrNamingIR{
		RenameMovies:             ptrToBool(naming.RenameMovies),
		ReplaceIllegalCharacters: ptrToBool(naming.ReplaceIllegalCharacters),
		StandardMovieFormat:      ptrToString(naming.StandardMovieFormat),
		MovieFolderFormat:        ptrToString(naming.MovieFolderFormat),
	}

	if naming.ColonReplacementFormat != nil {
		ir.ColonReplacementFormat = colonReplacementToInt(*naming.ColonReplacementFormat)
	}

	return ir, nil
}

// colonReplacementToInt converts a Radarr colon replacement format to int
func colonReplacementToInt(format client.ColonReplacementFormat) int {
	switch format {
	case client.Delete:
		return irv1.ColonReplacementDelete
	case client.Dash:
		return irv1.ColonReplacementDash
	case client.SpaceDash:
		return irv1.ColonReplacementSpace
	case client.SpaceDashSpace:
		return irv1.ColonReplacementSpace
	case client.Smart:
		return irv1.ColonReplacementSmart
	default:
		return irv1.ColonReplacementDelete
	}
}

// intToColonReplacement converts an int to a Radarr colon replacement format
func intToColonReplacement(format int) client.ColonReplacementFormat {
	switch format {
	case irv1.ColonReplacementDelete:
		return client.Delete
	case irv1.ColonReplacementDash:
		return client.Dash
	case irv1.ColonReplacementSpace:
		return client.SpaceDash
	case irv1.ColonReplacementSmart:
		return client.Smart
	default:
		return client.Delete
	}
}

// diffNaming computes changes needed for naming configuration
func (a *Adapter) diffNaming(current, desired *irv1.IR, changes *adapters.ChangeSet) error {
	var currentNaming *irv1.RadarrNamingIR
	var desiredNaming *irv1.RadarrNamingIR

	if current.Naming != nil {
		currentNaming = current.Naming.Radarr
	}
	if desired.Naming != nil {
		desiredNaming = desired.Naming.Radarr
	}

	// No desired naming config - nothing to do
	if desiredNaming == nil {
		return nil
	}

	// Check if update needed
	if currentNaming == nil || !namingEqual(currentNaming, desiredNaming) {
		changes.Updates = append(changes.Updates, adapters.Change{
			ResourceType: adapters.ResourceNamingConfig,
			Name:         "naming",
			Payload:      desiredNaming,
		})
	}

	return nil
}

// namingEqual compares two naming configs for equality
func namingEqual(a, b *irv1.RadarrNamingIR) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.RenameMovies == b.RenameMovies &&
		a.ReplaceIllegalCharacters == b.ReplaceIllegalCharacters &&
		a.ColonReplacementFormat == b.ColonReplacementFormat &&
		a.StandardMovieFormat == b.StandardMovieFormat &&
		a.MovieFolderFormat == b.MovieFolderFormat
}
