package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractToolCallsFromResponse_StructuredContentArray(t *testing.T) {
	response := `{"content":[{"type":"tool_use","name":"Read","input":{"path":"test.go"}},{"type":"text","text":"hello"}]}`
	calls := extractToolCallsFromResponse(response)
	assert.Len(t, calls, 1)
	assert.Equal(t, "Read", calls[0].name)
	assert.Equal(t, map[string]any{"path": "test.go"}, calls[0].input)
}

func TestExtractToolCallsFromResponse_Empty(t *testing.T) {
	response := `{"content":[{"type":"text","text":"no tools needed"}]}`
	calls := extractToolCallsFromResponse(response)
	assert.Len(t, calls, 0)
}

func TestExtractToolCallsFromResponse_MultipleTools(t *testing.T) {
	response := `{"content":[
		{"type":"tool_use","name":"Read","input":{"path":"a.go"}},
		{"type":"tool_use","name":"Bash","input":{"command":"ls"}},
		{"type":"text","text":"done"}
	]}`
	calls := extractToolCallsFromResponse(response)
	assert.Len(t, calls, 2)
	assert.Equal(t, "Read", calls[0].name)
	assert.Equal(t, "Bash", calls[1].name)
}

func TestExtractToolCallsFromResponse_InlineObjects(t *testing.T) {
	// Model returns inline JSON objects instead of a structured content array.
	response := `{"type":"tool_use","name":"Write","input":{"path":"out.txt","content":"hello"}}`
	calls := extractToolCallsFromResponse(response)
	assert.Len(t, calls, 1)
	assert.Equal(t, "Write", calls[0].name)
}

func TestExtractToolCallsFromResponse_NoJSON(t *testing.T) {
	response := "I would start by reading the README file and then examining the codebase structure."
	calls := extractToolCallsFromResponse(response)
	assert.Len(t, calls, 0)
}

func TestExtractToolCallsFromResponse_EmptyString(t *testing.T) {
	calls := extractToolCallsFromResponse("")
	assert.Len(t, calls, 0)
}

func TestBuildTestPrompt_ContainsSystemAndTask(t *testing.T) {
	prompt := buildTestPrompt("system instructions here", "do something")
	assert.Contains(t, prompt, "system instructions here")
	assert.Contains(t, prompt, "do something")
}

func TestBuildTestPrompt_ContainsToolUseInstruction(t *testing.T) {
	prompt := buildTestPrompt("sys", "task")
	assert.Contains(t, prompt, "tool_use")
}
