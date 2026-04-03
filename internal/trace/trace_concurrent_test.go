package trace

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// failWriter always returns an error on Write.
type failWriter struct{}

func (f *failWriter) Write(p []byte) (n int, err error) {
	return 0, errors.New("simulated write failure")
}

// TestRecord_ConcurrentSafety spawns 50 goroutines each recording 20 events
// (1000 total). It verifies the final summary count and that every JSONL line
// is valid JSON.
func TestRecord_ConcurrentSafety(t *testing.T) {
	var buf bytes.Buffer
	// Wrap buf in a mutex-safe writer because bytes.Buffer is not goroutine-safe
	// on its own — the Recorder's mutex protects writes, but we still need to
	// read the buffer safely after all goroutines finish.
	rec := NewRecorder(&buf)

	const goroutines = 50
	const eventsEach = 20

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < eventsEach; j++ {
				err := rec.Record(ToolCallEvent{
					Timestamp:    time.Now(),
					AgentID:      "agent",
					ToolName:     "Read",
					InputParams:  map[string]any{"file": "foo.go"},
					MockResponse: "ok",
					DurationMs:   int64(id*100 + j),
				})
				require.NoError(t, err)
			}
		}(i)
	}

	wg.Wait()

	// Verify total count via Summary.
	summary := rec.Summary()
	assert.Equal(t, goroutines*eventsEach, summary.TotalCalls, "expected 1000 total calls")

	// Verify every JSONL line is valid JSON.
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	assert.Len(t, lines, goroutines*eventsEach, "expected 1000 JSONL lines")

	for i, line := range lines {
		var event ToolCallEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Errorf("line %d is not valid JSON: %v\nline content: %s", i, err, line)
		}
	}
}

// TestRecord_ConcurrentSafety_RaceDetector runs 2 writer goroutines and 1
// reader goroutine simultaneously. This is specifically designed to surface
// data races when run with `go test -race`.
func TestRecord_ConcurrentSafety_RaceDetector(t *testing.T) {
	var buf bytes.Buffer
	rec := NewRecorder(&buf)

	var wg sync.WaitGroup

	// Writer 1
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_ = rec.Record(ToolCallEvent{
				Timestamp:  time.Now(),
				AgentID:    "writer-1",
				ToolName:   "Write",
				DurationMs: int64(i),
			})
		}
	}()

	// Writer 2
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_ = rec.Record(ToolCallEvent{
				Timestamp:  time.Now(),
				AgentID:    "writer-2",
				ToolName:   "Bash",
				DurationMs: int64(i),
			})
		}
	}()

	// Concurrent reader — calls Summary() while writers are active.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			_ = rec.Summary()
		}
	}()

	wg.Wait()

	// After all goroutines finish, the total should be exactly 200.
	summary := rec.Summary()
	assert.Equal(t, 200, summary.TotalCalls, "expected 200 total calls from two writers")
}

// TestRecord_WriterFailure verifies that when the underlying writer fails,
// Record() returns an error but the event is still appended to the in-memory
// log.
func TestRecord_WriterFailure(t *testing.T) {
	rec := NewRecorder(&failWriter{})

	event := ToolCallEvent{
		Timestamp:    time.Now(),
		AgentID:      "agent",
		ToolName:     "Read",
		InputParams:  map[string]any{"path": "/tmp/foo"},
		MockResponse: "content",
		DurationMs:   5,
	}

	err := rec.Record(event)
	require.Error(t, err, "expected error from failing writer")
	assert.Contains(t, err.Error(), "simulated write failure", "error should describe the write failure")

	// The event must still be in memory even though the write failed.
	summary := rec.Summary()
	assert.Equal(t, 1, summary.TotalCalls, "event should be stored in memory despite write error")
	require.Len(t, summary.Events, 1)
	assert.Equal(t, "Read", summary.Events[0].ToolName)
}

// TestRecord_WithMetadata verifies that a ToolCallEvent with a populated
// Metadata map roundtrips correctly through JSONL serialization.
func TestRecord_WithMetadata(t *testing.T) {
	var buf bytes.Buffer
	rec := NewRecorder(&buf)

	event := ToolCallEvent{
		Timestamp:    time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
		AgentID:      "meta-agent",
		ToolName:     "Glob",
		InputParams:  map[string]any{"pattern": "**/*.go"},
		MockResponse: "3 files",
		DurationMs:   42,
		Metadata: map[string]string{
			"run_id":  "abc-123",
			"version": "v1.2.3",
			"env":     "test",
		},
	}

	err := rec.Record(event)
	require.NoError(t, err)

	// Decode the JSONL line and verify metadata roundtrip.
	line := strings.TrimSpace(buf.String())
	var decoded ToolCallEvent
	err = json.Unmarshal([]byte(line), &decoded)
	require.NoError(t, err, "JSONL line must be valid JSON")

	assert.Equal(t, event.AgentID, decoded.AgentID)
	assert.Equal(t, event.ToolName, decoded.ToolName)
	assert.Equal(t, event.DurationMs, decoded.DurationMs)
	require.NotNil(t, decoded.Metadata, "metadata must not be nil after roundtrip")
	assert.Equal(t, "abc-123", decoded.Metadata["run_id"])
	assert.Equal(t, "v1.2.3", decoded.Metadata["version"])
	assert.Equal(t, "test", decoded.Metadata["env"])
}

// TestSummary_MultipleTool records 5 Read + 3 Write + 1 Bash events and
// verifies the CallsByTool breakdown in the returned Summary.
func TestSummary_MultipleTool(t *testing.T) {
	var buf bytes.Buffer
	rec := NewRecorder(&buf)

	tools := []struct {
		name  string
		count int
	}{
		{"Read", 5},
		{"Write", 3},
		{"Bash", 1},
	}

	for _, tc := range tools {
		for i := 0; i < tc.count; i++ {
			err := rec.Record(ToolCallEvent{
				Timestamp:  time.Now(),
				AgentID:    "agent",
				ToolName:   tc.name,
				DurationMs: int64(i),
			})
			require.NoError(t, err)
		}
	}

	summary := rec.Summary()

	assert.Equal(t, 9, summary.TotalCalls, "total should be 5+3+1=9")
	assert.Equal(t, 5, summary.CallsByTool["Read"], "Read count")
	assert.Equal(t, 3, summary.CallsByTool["Write"], "Write count")
	assert.Equal(t, 1, summary.CallsByTool["Bash"], "Bash count")
}
