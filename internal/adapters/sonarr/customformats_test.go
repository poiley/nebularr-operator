package sonarr

import (
	"testing"

	"github.com/poiley/nebularr-operator/internal/adapters"
	irv1 "github.com/poiley/nebularr-operator/internal/ir/v1"
)

func TestCustomFormatToIR(t *testing.T) {
	a := &Adapter{}

	tests := []struct {
		name     string
		input    CustomFormatResource
		expected irv1.CustomFormatIR
	}{
		{
			name: "basic custom format",
			input: CustomFormatResource{
				ID:                              1,
				Name:                            "x265",
				IncludeCustomFormatWhenRenaming: false,
				Specifications: []CustomFormatSpecification{
					{
						Name:           "x265",
						Implementation: "ReleaseTitleSpecification",
						Negate:         false,
						Required:       true,
						Fields: []Field{
							{Name: "value", Value: "[xh]\\.?265|hevc"},
						},
					},
				},
			},
			expected: irv1.CustomFormatIR{
				ID:                  1,
				Name:                "x265",
				IncludeWhenRenaming: false,
				Specifications: []irv1.FormatSpecIR{
					{
						Type:     "ReleaseTitleSpecification",
						Name:     "x265",
						Negate:   false,
						Required: true,
						Value:    "[xh]\\.?265|hevc",
					},
				},
			},
		},
		{
			name: "custom format with multiple specs",
			input: CustomFormatResource{
				ID:                              2,
				Name:                            "DV HDR10",
				IncludeCustomFormatWhenRenaming: true,
				Specifications: []CustomFormatSpecification{
					{
						Name:           "DV",
						Implementation: "ReleaseTitleSpecification",
						Negate:         false,
						Required:       true,
						Fields: []Field{
							{Name: "value", Value: "\\b(dv|dovi)\\b"},
						},
					},
					{
						Name:           "HDR10",
						Implementation: "ReleaseTitleSpecification",
						Negate:         false,
						Required:       false,
						Fields: []Field{
							{Name: "value", Value: "hdr10"},
						},
					},
				},
			},
			expected: irv1.CustomFormatIR{
				ID:                  2,
				Name:                "DV HDR10",
				IncludeWhenRenaming: true,
				Specifications: []irv1.FormatSpecIR{
					{
						Type:     "ReleaseTitleSpecification",
						Name:     "DV",
						Negate:   false,
						Required: true,
						Value:    "\\b(dv|dovi)\\b",
					},
					{
						Type:     "ReleaseTitleSpecification",
						Name:     "HDR10",
						Negate:   false,
						Required: false,
						Value:    "hdr10",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := a.customFormatToIR(&tt.input)

			if result.ID != tt.expected.ID {
				t.Errorf("expected ID %d, got %d", tt.expected.ID, result.ID)
			}
			if result.Name != tt.expected.Name {
				t.Errorf("expected name %q, got %q", tt.expected.Name, result.Name)
			}
			if result.IncludeWhenRenaming != tt.expected.IncludeWhenRenaming {
				t.Errorf("expected includeWhenRenaming %v, got %v", tt.expected.IncludeWhenRenaming, result.IncludeWhenRenaming)
			}
			if len(result.Specifications) != len(tt.expected.Specifications) {
				t.Fatalf("expected %d specs, got %d", len(tt.expected.Specifications), len(result.Specifications))
			}
			for i, spec := range result.Specifications {
				expSpec := tt.expected.Specifications[i]
				if spec.Type != expSpec.Type {
					t.Errorf("spec %d: expected type %q, got %q", i, expSpec.Type, spec.Type)
				}
				if spec.Name != expSpec.Name {
					t.Errorf("spec %d: expected name %q, got %q", i, expSpec.Name, spec.Name)
				}
				if spec.Negate != expSpec.Negate {
					t.Errorf("spec %d: expected negate %v, got %v", i, expSpec.Negate, spec.Negate)
				}
				if spec.Required != expSpec.Required {
					t.Errorf("spec %d: expected required %v, got %v", i, expSpec.Required, spec.Required)
				}
				if spec.Value != expSpec.Value {
					t.Errorf("spec %d: expected value %q, got %q", i, expSpec.Value, spec.Value)
				}
			}
		})
	}
}

func TestIRToCustomFormat(t *testing.T) {
	a := &Adapter{}

	tests := []struct {
		name     string
		input    irv1.CustomFormatIR
		expected CustomFormatResource
	}{
		{
			name: "basic IR to resource",
			input: irv1.CustomFormatIR{
				ID:                  5,
				Name:                "nebularr-test-HEVC",
				IncludeWhenRenaming: false,
				Specifications: []irv1.FormatSpecIR{
					{
						Type:     "ReleaseTitleSpecification",
						Name:     "HEVC",
						Negate:   false,
						Required: true,
						Value:    "hevc|x265",
					},
				},
			},
			expected: CustomFormatResource{
				ID:                              5,
				Name:                            "nebularr-test-HEVC",
				IncludeCustomFormatWhenRenaming: false,
				Specifications: []CustomFormatSpecification{
					{
						Name:           "HEVC",
						Implementation: "ReleaseTitleSpecification",
						Negate:         false,
						Required:       true,
						Fields: []Field{
							{Name: "value", Value: "hevc|x265"},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := a.irToCustomFormat(&tt.input)

			if result.ID != tt.expected.ID {
				t.Errorf("expected ID %d, got %d", tt.expected.ID, result.ID)
			}
			if result.Name != tt.expected.Name {
				t.Errorf("expected name %q, got %q", tt.expected.Name, result.Name)
			}
			if result.IncludeCustomFormatWhenRenaming != tt.expected.IncludeCustomFormatWhenRenaming {
				t.Errorf("expected includeCustomFormatWhenRenaming %v, got %v",
					tt.expected.IncludeCustomFormatWhenRenaming, result.IncludeCustomFormatWhenRenaming)
			}
			if len(result.Specifications) != len(tt.expected.Specifications) {
				t.Fatalf("expected %d specs, got %d", len(tt.expected.Specifications), len(result.Specifications))
			}
			for i, spec := range result.Specifications {
				expSpec := tt.expected.Specifications[i]
				if spec.Name != expSpec.Name {
					t.Errorf("spec %d: expected name %q, got %q", i, expSpec.Name, spec.Name)
				}
				if spec.Implementation != expSpec.Implementation {
					t.Errorf("spec %d: expected implementation %q, got %q", i, expSpec.Implementation, spec.Implementation)
				}
				if spec.Negate != expSpec.Negate {
					t.Errorf("spec %d: expected negate %v, got %v", i, expSpec.Negate, spec.Negate)
				}
				if spec.Required != expSpec.Required {
					t.Errorf("spec %d: expected required %v, got %v", i, expSpec.Required, spec.Required)
				}
			}
		})
	}
}

func TestCustomFormatsEqual(t *testing.T) {
	tests := []struct {
		name     string
		a        irv1.CustomFormatIR
		b        irv1.CustomFormatIR
		expected bool
	}{
		{
			name: "equal custom formats",
			a: irv1.CustomFormatIR{
				ID:                  1,
				Name:                "Test",
				IncludeWhenRenaming: true,
				Specifications: []irv1.FormatSpecIR{
					{Type: "ReleaseTitleSpecification", Name: "Test", Value: "test"},
				},
			},
			b: irv1.CustomFormatIR{
				ID:                  2, // ID should be ignored
				Name:                "Test",
				IncludeWhenRenaming: true,
				Specifications: []irv1.FormatSpecIR{
					{Type: "ReleaseTitleSpecification", Name: "Test", Value: "test"},
				},
			},
			expected: true,
		},
		{
			name: "different includeWhenRenaming",
			a: irv1.CustomFormatIR{
				Name:                "Test",
				IncludeWhenRenaming: true,
				Specifications:      []irv1.FormatSpecIR{},
			},
			b: irv1.CustomFormatIR{
				Name:                "Test",
				IncludeWhenRenaming: false,
				Specifications:      []irv1.FormatSpecIR{},
			},
			expected: false,
		},
		{
			name: "different spec count",
			a: irv1.CustomFormatIR{
				Name: "Test",
				Specifications: []irv1.FormatSpecIR{
					{Type: "ReleaseTitleSpecification", Name: "Test1", Value: "test1"},
				},
			},
			b: irv1.CustomFormatIR{
				Name: "Test",
				Specifications: []irv1.FormatSpecIR{
					{Type: "ReleaseTitleSpecification", Name: "Test1", Value: "test1"},
					{Type: "ReleaseTitleSpecification", Name: "Test2", Value: "test2"},
				},
			},
			expected: false,
		},
		{
			name: "different spec value",
			a: irv1.CustomFormatIR{
				Name: "Test",
				Specifications: []irv1.FormatSpecIR{
					{Type: "ReleaseTitleSpecification", Name: "Test", Value: "value1"},
				},
			},
			b: irv1.CustomFormatIR{
				Name: "Test",
				Specifications: []irv1.FormatSpecIR{
					{Type: "ReleaseTitleSpecification", Name: "Test", Value: "value2"},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := customFormatsEqual(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestDiffCustomFormats(t *testing.T) {
	a := &Adapter{}

	tests := []struct {
		name            string
		current         *irv1.IR
		desired         *irv1.IR
		expectedCreates int
		expectedUpdates int
		expectedDeletes int
	}{
		{
			name: "no changes",
			current: &irv1.IR{
				CustomFormats: []irv1.CustomFormatIR{
					{ID: 1, Name: "Test", Specifications: []irv1.FormatSpecIR{}},
				},
			},
			desired: &irv1.IR{
				CustomFormats: []irv1.CustomFormatIR{
					{Name: "Test", Specifications: []irv1.FormatSpecIR{}},
				},
			},
			expectedCreates: 0,
			expectedUpdates: 0,
			expectedDeletes: 0,
		},
		{
			name: "create new custom format",
			current: &irv1.IR{
				CustomFormats: []irv1.CustomFormatIR{},
			},
			desired: &irv1.IR{
				CustomFormats: []irv1.CustomFormatIR{
					{Name: "NewFormat", Specifications: []irv1.FormatSpecIR{}},
				},
			},
			expectedCreates: 1,
			expectedUpdates: 0,
			expectedDeletes: 0,
		},
		{
			name: "update existing custom format",
			current: &irv1.IR{
				CustomFormats: []irv1.CustomFormatIR{
					{ID: 1, Name: "Test", IncludeWhenRenaming: false, Specifications: []irv1.FormatSpecIR{}},
				},
			},
			desired: &irv1.IR{
				CustomFormats: []irv1.CustomFormatIR{
					{Name: "Test", IncludeWhenRenaming: true, Specifications: []irv1.FormatSpecIR{}},
				},
			},
			expectedCreates: 0,
			expectedUpdates: 1,
			expectedDeletes: 0,
		},
		{
			name: "delete custom format",
			current: &irv1.IR{
				CustomFormats: []irv1.CustomFormatIR{
					{ID: 1, Name: "ToDelete", Specifications: []irv1.FormatSpecIR{}},
				},
			},
			desired: &irv1.IR{
				CustomFormats: []irv1.CustomFormatIR{},
			},
			expectedCreates: 0,
			expectedUpdates: 0,
			expectedDeletes: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes := &adapters.ChangeSet{
				Creates: []adapters.Change{},
				Updates: []adapters.Change{},
				Deletes: []adapters.Change{},
			}

			err := a.diffCustomFormats(tt.current, tt.desired, changes)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(changes.Creates) != tt.expectedCreates {
				t.Errorf("expected %d creates, got %d", tt.expectedCreates, len(changes.Creates))
			}
			if len(changes.Updates) != tt.expectedUpdates {
				t.Errorf("expected %d updates, got %d", tt.expectedUpdates, len(changes.Updates))
			}
			if len(changes.Deletes) != tt.expectedDeletes {
				t.Errorf("expected %d deletes, got %d", tt.expectedDeletes, len(changes.Deletes))
			}
		})
	}
}

func TestSourceToInt(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"television", 1},
		{"televisionRaw", 2},
		{"webdl", 3},
		{"web", 3},
		{"webrip", 4},
		{"webRip", 4},
		{"dvd", 5},
		{"bluray", 6},
		{"blurayRaw", 7},
		{"unknown", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sourceToInt(tt.input)
			if result != tt.expected {
				t.Errorf("expected %d for %q, got %d", tt.expected, tt.input, result)
			}
		})
	}
}

func TestResolutionToInt(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"r360p", 360},
		{"r480p", 480},
		{"r540p", 540},
		{"r576p", 576},
		{"r720p", 720},
		{"r1080p", 1080},
		{"r2160p", 2160},
		{"unknown", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := resolutionToInt(tt.input)
			if result != tt.expected {
				t.Errorf("expected %d for %q, got %d", tt.expected, tt.input, result)
			}
		})
	}
}
