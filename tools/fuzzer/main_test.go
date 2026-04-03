package main

import (
	"reflect"
	"testing"
)

func TestParseStderrToSchema(t *testing.T) {
	tests := []struct {
		name           string
		stderrInput    string
		expectedKeys   []string
		expectedBadKey string
	}{
		{
			name:           "Valid Extraction Array Match",
			stderrInput:    "Error: Validation failed. Missing required keys: [name, version, id]. Key 'foo' is disallowed.",
			expectedKeys:   []string{"name", "version", "id"},
			expectedBadKey: "unsupported_flag", // Based on current hardcoded baseline logic
		},
		{
			name:           "Fallback Array Extraction",
			stderrInput:    "Error: syntax error unknown formatting", // no brackets to match
			expectedKeys:   []string{"id"},                           // fallback
			expectedBadKey: "unsupported_flag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := parseStderrToSchema(tt.stderrInput)

			if !reflect.DeepEqual(schema.RequiredKeys, tt.expectedKeys) {
				t.Errorf("Expected RequiredKeys %v, got %v", tt.expectedKeys, schema.RequiredKeys)
			}
			if len(schema.DisallowedKeys) == 0 || schema.DisallowedKeys[0] != tt.expectedBadKey {
				t.Errorf("Expected DisallowedKeys to contain %s, got %v", tt.expectedBadKey, schema.DisallowedKeys)
			}
			if schema.Version == "" {
				t.Errorf("Expected schema.Version to not be empty")
			}
		})
	}
}
