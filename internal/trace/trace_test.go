package trace

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeEvent(tool string) ToolCallEvent {
	return ToolCallEvent{
		Timestamp:    time.Now().UTC(),
		AgentID:      "developer",
		ToolName:     tool,
		InputParams:  map[string]any{"command": "npm test"},
		MockResponse: "[SIMULATED SUCCESS]",
		DurationMs:   12,
	}
}

func TestRecorder_WritesValidJSONL(t *testing.T) {
	var buf bytes.Buffer
	rec := NewRecorder(&buf)

	err := rec.Record(makeEvent("Bash"))
	require.NoError(t, err)

	err = rec.Record(makeEvent("Read"))
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	assert.Len(t, lines, 2)

	for _, line := range lines {
		var event ToolCallEvent
		require.NoError(t, json.Unmarshal([]byte(line), &event), "each line must be valid JSON")
	}
}

func TestRecorder_Summary_CountsCorrectly(t *testing.T) {
	var buf bytes.Buffer
	rec := NewRecorder(&buf)

	require.NoError(t, rec.Record(makeEvent("Bash")))
	require.NoError(t, rec.Record(makeEvent("Bash")))
	require.NoError(t, rec.Record(makeEvent("Write")))

	s := rec.Summary()
	assert.Equal(t, 3, s.TotalCalls)
	assert.Equal(t, 2, s.CallsByTool["Bash"])
	assert.Equal(t, 1, s.CallsByTool["Write"])
}

func TestRecorder_Empty_Summary(t *testing.T) {
	var buf bytes.Buffer
	rec := NewRecorder(&buf)
	s := rec.Summary()

	assert.Equal(t, 0, s.TotalCalls)
	assert.Empty(t, s.CallsByTool)
}

func TestSummary_Print_OutputsText(t *testing.T) {
	var buf bytes.Buffer
	rec := NewRecorder(&buf)
	require.NoError(t, rec.Record(makeEvent("Bash")))

	var out bytes.Buffer
	rec.Summary().Print(&out)

	output := out.String()
	assert.Contains(t, output, "Total intercepted tool calls: 1")
	assert.Contains(t, output, "Bash")
}
