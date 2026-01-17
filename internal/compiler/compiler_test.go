package compiler

import (
	"context"
	"testing"

	arrv1alpha1 "github.com/poiley/nebularr-operator/api/v1alpha1"
	"github.com/poiley/nebularr-operator/internal/adapters"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

func TestConvertCustomFormats(t *testing.T) {
	tests := []struct {
		name     string
		input    []arrv1alpha1.CustomFormatSpec
		expected []CustomFormatInput
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty input",
			input:    []arrv1alpha1.CustomFormatSpec{},
			expected: nil,
		},
		{
			name: "single custom format with one spec",
			input: []arrv1alpha1.CustomFormatSpec{
				{
					Name:  "DV",
					Score: 1500,
					Specifications: []arrv1alpha1.CustomFormatSpecificationSpec{
						{
							Name:  "Dolby Vision",
							Type:  "ReleaseTitleSpecification",
							Value: "\\b(dv|dovi|dolby[ .]?vision)\\b",
						},
					},
				},
			},
			expected: []CustomFormatInput{
				{
					Name:                "DV",
					IncludeWhenRenaming: false,
					Score:               1500,
					Specifications: []CustomFormatSpecInput{
						{
							Name:     "Dolby Vision",
							Type:     "ReleaseTitleSpecification",
							Negate:   false,
							Required: false,
							Value:    "\\b(dv|dovi|dolby[ .]?vision)\\b",
						},
					},
				},
			},
		},
		{
			name: "custom format with multiple specs and options",
			input: []arrv1alpha1.CustomFormatSpec{
				{
					Name:                "x265 HD",
					IncludeWhenRenaming: boolPtr(true),
					Score:               100,
					Specifications: []arrv1alpha1.CustomFormatSpecificationSpec{
						{
							Name:     "x265",
							Type:     "ReleaseTitleSpecification",
							Required: boolPtr(true),
							Value:    "[xh]\\.?265|hevc",
						},
						{
							Name:   "Not 4K",
							Type:   "ResolutionSpecification",
							Negate: boolPtr(true),
							Value:  "r2160p",
						},
					},
				},
			},
			expected: []CustomFormatInput{
				{
					Name:                "x265 HD",
					IncludeWhenRenaming: true,
					Score:               100,
					Specifications: []CustomFormatSpecInput{
						{
							Name:     "x265",
							Type:     "ReleaseTitleSpecification",
							Negate:   false,
							Required: true,
							Value:    "[xh]\\.?265|hevc",
						},
						{
							Name:     "Not 4K",
							Type:     "ResolutionSpecification",
							Negate:   true,
							Required: false,
							Value:    "r2160p",
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertCustomFormats(tt.input)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}

			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d custom formats, got %d", len(tt.expected), len(result))
			}

			for i, cf := range result {
				exp := tt.expected[i]
				if cf.Name != exp.Name {
					t.Errorf("custom format %d: expected name %q, got %q", i, exp.Name, cf.Name)
				}
				if cf.IncludeWhenRenaming != exp.IncludeWhenRenaming {
					t.Errorf("custom format %d: expected includeWhenRenaming %v, got %v", i, exp.IncludeWhenRenaming, cf.IncludeWhenRenaming)
				}
				if cf.Score != exp.Score {
					t.Errorf("custom format %d: expected score %d, got %d", i, exp.Score, cf.Score)
				}
				if len(cf.Specifications) != len(exp.Specifications) {
					t.Fatalf("custom format %d: expected %d specs, got %d", i, len(exp.Specifications), len(cf.Specifications))
				}
				for j, spec := range cf.Specifications {
					expSpec := exp.Specifications[j]
					if spec.Name != expSpec.Name {
						t.Errorf("spec %d.%d: expected name %q, got %q", i, j, expSpec.Name, spec.Name)
					}
					if spec.Type != expSpec.Type {
						t.Errorf("spec %d.%d: expected type %q, got %q", i, j, expSpec.Type, spec.Type)
					}
					if spec.Negate != expSpec.Negate {
						t.Errorf("spec %d.%d: expected negate %v, got %v", i, j, expSpec.Negate, spec.Negate)
					}
					if spec.Required != expSpec.Required {
						t.Errorf("spec %d.%d: expected required %v, got %v", i, j, expSpec.Required, spec.Required)
					}
					if spec.Value != expSpec.Value {
						t.Errorf("spec %d.%d: expected value %q, got %q", i, j, expSpec.Value, spec.Value)
					}
				}
			}
		})
	}
}

func TestConvertNotifications(t *testing.T) {
	tests := []struct {
		name            string
		input           []arrv1alpha1.NotificationSpec
		resolvedSecrets map[string]string
		expectedCount   int
	}{
		{
			name:          "nil input",
			input:         nil,
			expectedCount: 0,
		},
		{
			name:          "empty input",
			input:         []arrv1alpha1.NotificationSpec{},
			expectedCount: 0,
		},
		{
			name: "single notification",
			input: []arrv1alpha1.NotificationSpec{
				{
					Name:       "Discord",
					Type:       "Discord",
					OnGrab:     boolPtr(true),
					OnDownload: boolPtr(true),
					Settings: map[string]string{
						"webHookUrl": "https://discord.com/api/webhooks/...",
					},
				},
			},
			resolvedSecrets: map[string]string{},
			expectedCount:   1,
		},
		{
			name: "notification with secret settings",
			input: []arrv1alpha1.NotificationSpec{
				{
					Name:       "Telegram",
					Type:       "Telegram",
					OnDownload: boolPtr(true),
					SettingsSecretRef: &arrv1alpha1.SecretKeySelector{
						Name: "telegram-secret",
					},
				},
			},
			resolvedSecrets: map[string]string{
				"telegram-secret/botToken": "123456:ABC",
				"telegram-secret/chatId":   "-123456789",
			},
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertNotifications(tt.input, tt.resolvedSecrets)

			if tt.expectedCount == 0 {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}

			if len(result) != tt.expectedCount {
				t.Errorf("expected %d notifications, got %d", tt.expectedCount, len(result))
			}
		})
	}
}

func TestCompileCustomFormatsToIR(t *testing.T) {
	c := New()

	tests := []struct {
		name       string
		input      []CustomFormatInput
		configName string
		expected   []irv1.CustomFormatIR
	}{
		{
			name:       "nil input",
			input:      nil,
			configName: "test",
			expected:   nil,
		},
		{
			name:       "empty input",
			input:      []CustomFormatInput{},
			configName: "test",
			expected:   nil,
		},
		{
			name: "single custom format",
			input: []CustomFormatInput{
				{
					Name:                "DV",
					IncludeWhenRenaming: true,
					Score:               1500,
					Specifications: []CustomFormatSpecInput{
						{
							Name:     "Dolby Vision",
							Type:     "ReleaseTitleSpecification",
							Required: true,
							Value:    "\\b(dv|dovi)\\b",
						},
					},
				},
			},
			configName: "movies",
			expected: []irv1.CustomFormatIR{
				{
					Name:                "nebularr-movies-DV",
					IncludeWhenRenaming: true,
					Specifications: []irv1.FormatSpecIR{
						{
							Type:     "ReleaseTitleSpecification",
							Name:     "Dolby Vision",
							Required: true,
							Value:    "\\b(dv|dovi)\\b",
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.compileCustomFormatsToIR(tt.input, tt.configName)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}

			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d custom formats, got %d", len(tt.expected), len(result))
			}

			for i, cf := range result {
				exp := tt.expected[i]
				if cf.Name != exp.Name {
					t.Errorf("custom format %d: expected name %q, got %q", i, exp.Name, cf.Name)
				}
				if cf.IncludeWhenRenaming != exp.IncludeWhenRenaming {
					t.Errorf("custom format %d: expected includeWhenRenaming %v, got %v", i, exp.IncludeWhenRenaming, cf.IncludeWhenRenaming)
				}
			}
		})
	}
}

func TestCompileWithCustomFormats(t *testing.T) {
	c := New()

	input := CompileInput{
		App:        adapters.AppRadarr,
		ConfigName: "test-config",
		Namespace:  "default",
		URL:        "http://radarr:7878",
		APIKey:     "test-api-key",
		CustomFormats: []CustomFormatInput{
			{
				Name:  "HDR",
				Score: 500,
				Specifications: []CustomFormatSpecInput{
					{
						Name:  "HDR10",
						Type:  "ReleaseTitleSpecification",
						Value: "\\bhdr10\\b",
					},
				},
			},
		},
	}

	ir, err := c.Compile(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ir == nil {
		t.Fatal("expected IR, got nil")
	}

	if len(ir.CustomFormats) != 1 {
		t.Fatalf("expected 1 custom format, got %d", len(ir.CustomFormats))
	}

	cf := ir.CustomFormats[0]
	expectedName := "nebularr-test-config-HDR"
	if cf.Name != expectedName {
		t.Errorf("expected name %q, got %q", expectedName, cf.Name)
	}

	if len(cf.Specifications) != 1 {
		t.Fatalf("expected 1 specification, got %d", len(cf.Specifications))
	}

	spec := cf.Specifications[0]
	if spec.Type != "ReleaseTitleSpecification" {
		t.Errorf("expected type ReleaseTitleSpecification, got %s", spec.Type)
	}
	if spec.Value != "\\bhdr10\\b" {
		t.Errorf("expected value '\\bhdr10\\b', got %s", spec.Value)
	}

	// Verify format scores are populated in quality profile
	if ir.Quality == nil || ir.Quality.Video == nil {
		t.Fatal("expected quality profile to be present")
	}
	if ir.Quality.Video.FormatScores == nil {
		t.Fatal("expected format scores to be populated")
	}

	expectedFormatName := "nebularr-test-config-HDR"
	score, ok := ir.Quality.Video.FormatScores[expectedFormatName]
	if !ok {
		t.Errorf("expected format score for %q, but not found", expectedFormatName)
	}
	if score != 500 {
		t.Errorf("expected score 500 for %q, got %d", expectedFormatName, score)
	}
}

func TestCompileFormatScores(t *testing.T) {
	c := New()

	tests := []struct {
		name       string
		input      []CustomFormatInput
		configName string
		expected   map[string]int
	}{
		{
			name:       "nil input",
			input:      nil,
			configName: "test",
			expected:   nil,
		},
		{
			name:       "empty input",
			input:      []CustomFormatInput{},
			configName: "test",
			expected:   nil,
		},
		{
			name: "formats with non-zero scores",
			input: []CustomFormatInput{
				{Name: "DV", Score: 1500},
				{Name: "HDR10+", Score: 1000},
				{Name: "Atmos", Score: 500},
			},
			configName: "movies",
			expected: map[string]int{
				"nebularr-movies-DV":     1500,
				"nebularr-movies-HDR10+": 1000,
				"nebularr-movies-Atmos":  500,
			},
		},
		{
			name: "formats with zero scores are excluded",
			input: []CustomFormatInput{
				{Name: "DV", Score: 1500},
				{Name: "NoScore", Score: 0},
			},
			configName: "test",
			expected: map[string]int{
				"nebularr-test-DV": 1500,
			},
		},
		{
			name: "formats with negative scores are included",
			input: []CustomFormatInput{
				{Name: "BR-DISK", Score: -10000},
				{Name: "LQ", Score: -5000},
			},
			configName: "movies",
			expected: map[string]int{
				"nebularr-movies-BR-DISK": -10000,
				"nebularr-movies-LQ":      -5000,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.compileFormatScores(tt.input, tt.configName)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d scores, got %d", len(tt.expected), len(result))
			}

			for name, expectedScore := range tt.expected {
				score, ok := result[name]
				if !ok {
					t.Errorf("expected score for %q, but not found", name)
					continue
				}
				if score != expectedScore {
					t.Errorf("expected score %d for %q, got %d", expectedScore, name, score)
				}
			}
		})
	}
}

// Helper function to create bool pointer
func boolPtr(b bool) *bool {
	return &b
}
