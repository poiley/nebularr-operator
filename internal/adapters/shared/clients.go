// Package shared provides common functionality used across multiple *arr adapters.
package shared

import (
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

// ExtractDownloadClientFields extracts common fields from a download client's Fields slice
// and populates the corresponding IR struct fields.
// categoryFieldNames are the possible names for the category field (e.g., "category", "tvCategory", "movieCategory").
func ExtractDownloadClientFields(fields []Field, ir *irv1.DownloadClientIR, categoryFieldNames ...string) {
	for _, f := range fields {
		switch f.Name {
		case "host":
			if v, ok := f.Value.(string); ok {
				ir.Host = v
			}
		case "port":
			if v, ok := f.Value.(float64); ok {
				ir.Port = int(v)
			}
		case "useSsl":
			if v, ok := f.Value.(bool); ok {
				ir.UseTLS = v
			}
		case "username":
			if v, ok := f.Value.(string); ok {
				ir.Username = v
			}
		case "password":
			if v, ok := f.Value.(string); ok {
				ir.Password = v
			}
		case "directory":
			if v, ok := f.Value.(string); ok {
				ir.Directory = v
			}
		default:
			// Check if it's a category field
			for _, catName := range categoryFieldNames {
				if f.Name == catName {
					if v, ok := f.Value.(string); ok {
						ir.Category = v
					}
					break
				}
			}
		}
	}
}

// ExtractIndexerFields extracts common fields from an indexer's Fields slice
// and populates the corresponding IR struct fields.
func ExtractIndexerFields(fields []Field, ir *irv1.IndexerIR) {
	for _, f := range fields {
		switch f.Name {
		case "baseUrl":
			if v, ok := f.Value.(string); ok {
				ir.URL = v
			}
		case "apiKey":
			if v, ok := f.Value.(string); ok {
				ir.APIKey = v
			}
		case "categories":
			if v, ok := f.Value.([]interface{}); ok {
				for _, cat := range v {
					if catNum, ok := cat.(float64); ok {
						ir.Categories = append(ir.Categories, int(catNum))
					}
				}
			}
		case "minimumSeeders":
			if v, ok := f.Value.(float64); ok {
				ir.MinimumSeeders = int(v)
			}
		case "seedCriteria.seedRatio":
			if v, ok := f.Value.(float64); ok {
				ir.SeedRatio = v
			}
		case "seedCriteria.seedTime":
			if v, ok := f.Value.(float64); ok {
				ir.SeedTimeMinutes = int(v)
			}
		}
	}
}
