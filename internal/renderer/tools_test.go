package renderer

import (
	"reflect"
	"testing"
)

func TestClaudeNativeTools_LengthInvariant(t *testing.T) {
	if len(claudeNativeTools) != 16 {
		t.Fatalf("claudeNativeTools has %d entries, expected 16 — update the map and this test together", len(claudeNativeTools))
	}
}

func TestIsClaudeNativeTool(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		expected bool
	}{
		{"Valid Claude Match", "Read", true},
		{"Case Sensitive Fail", "read", false},
		{"Invalid Claude Tool", "UnknownTool", false},
		{"MCP Tool ignores Claude check", "mcp_github_read", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isClaudeNativeTool(tt.toolName); got != tt.expected {
				t.Errorf("isClaudeNativeTool(%q) = %v, want %v", tt.toolName, got, tt.expected)
			}
		})
	}
}

func TestContainsClaudeNativeTools(t *testing.T) {
	if !containsClaudeNativeTools([]string{"mcp_foo", "Read", "custom"}) {
		t.Error("expected true when list contains Claude-native tool")
	}
	if containsClaudeNativeTools([]string{"mcp_foo", "read", "custom"}) {
		t.Error("expected false when list has no literal Claude-native tool matches")
	}
}

func TestSanitizeAgentTools(t *testing.T) {
	tests := []struct {
		name          string
		tools         []string
		caps          CapabilitySet
		targetName    string
		expectedTools []string
		expectedNotes int
		expectedCode  string
	}{
		{
			name:  "Provider Without Tools Support Silently Drops",
			tools: []string{"Bash", "Read"},
			caps: CapabilitySet{
				AgentNativeToolsOnly: false,
			},
			targetName:    "cursor",
			expectedTools: nil,
			expectedNotes: 0,
		},
		{
			name:  "Claude Provider Passes Through",
			tools: []string{"Bash", "Read", "mcp_test"},
			caps: CapabilitySet{
				AgentNativeToolsOnly: true,
			},
			targetName:    "claude",
			expectedTools: []string{"Bash", "Read", "mcp_test"},
			expectedNotes: 0,
		},
		{
			name:  "Provider With Tools Support Drops Claude Natives",
			tools: []string{"Bash", "Read", "mcp_github_read", "custom_tool"},
			caps: CapabilitySet{
				AgentNativeToolsOnly: false,
			},
			targetName:    "gemini",
			expectedTools: []string{"mcp_github_read", "custom_tool"},
			expectedNotes: 1,
			expectedCode:  CodeAgentToolsDropped,
		},
		{
			name:  "Provider With Tools Support Retains Fully Allowed List",
			tools: []string{"mcp_github_read", "custom_tool"},
			caps: CapabilitySet{
				AgentNativeToolsOnly: false,
			},
			targetName:    "gemini",
			expectedTools: []string{"mcp_github_read", "custom_tool"},
			expectedNotes: 0,
		},
		{
			name:  "Empty List Returns Nil",
			tools: []string{},
			caps: CapabilitySet{
				AgentNativeToolsOnly: false,
			},
			targetName:    "gemini",
			expectedTools: nil,
			expectedNotes: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTools, gotNotes := SanitizeAgentTools(tt.tools, tt.caps, tt.targetName, "test_agent")

			if !reflect.DeepEqual(gotTools, tt.expectedTools) {
				t.Errorf("SanitizeAgentTools() tools = %v, want %v", gotTools, tt.expectedTools)
			}

			if len(gotNotes) != tt.expectedNotes {
				t.Errorf("SanitizeAgentTools() note count = %v, want %v", len(gotNotes), tt.expectedNotes)
			}

			if tt.expectedNotes > 0 && len(gotNotes) > 0 {
				if gotNotes[0].Code != tt.expectedCode {
					t.Errorf("expected note code %q, got %q", tt.expectedCode, gotNotes[0].Code)
				}
			}
		})
	}
}

// TestSanitizeAgentTools_RegistryLookup verifies that SanitizeAgentTools uses
// the schema registry to determine field support. Cursor has tools marked
// "unsupported" in the registry, so tools must be dropped (nil returned).
func TestSanitizeAgentTools_RegistryLookup(t *testing.T) {
	tools := []string{"Bash", "Read", "mcp_custom"}
	caps := CapabilitySet{
		AgentNativeToolsOnly: false,
	}

	gotTools, gotNotes := SanitizeAgentTools(tools, caps, "cursor", "test_agent")

	if gotTools != nil {
		t.Errorf("expected nil tools for cursor (tools unsupported in registry), got %v", gotTools)
	}
	if len(gotNotes) != 0 {
		t.Errorf("expected 0 notes for silent drop, got %d", len(gotNotes))
	}
}
