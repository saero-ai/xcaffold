package trace

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"
)

// ToolCallEvent records a single intercepted tool call during a simulation run.
type ToolCallEvent struct {
	Timestamp    time.Time         `json:"timestamp"`
	AgentID      string            `json:"agent_id"`
	ToolName     string            `json:"tool_name"`
	InputParams  map[string]any    `json:"input_params"`
	MockResponse string            `json:"mock_response"`
	DurationMs   int64             `json:"duration_ms"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// Recorder writes ToolCallEvents as newline-delimited JSON (JSONL) to a writer.
// It is safe for concurrent use from multiple goroutines.
type Recorder struct {
	mu     sync.Mutex
	w      io.Writer
	events []ToolCallEvent
}

// NewRecorder returns a Recorder that writes events to w.
func NewRecorder(w io.Writer) *Recorder {
	return &Recorder{w: w}
}

// Record appends event to the in-memory log and writes it as a JSONL line.
// It is safe to call from multiple goroutines.
func (r *Recorder) Record(event ToolCallEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.events = append(r.events, event)

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("trace: failed to marshal event: %w", err)
	}

	if _, err := fmt.Fprintf(r.w, "%s\n", data); err != nil {
		return fmt.Errorf("trace: failed to write event: %w", err)
	}

	return nil
}

// Summary returns a human-readable report of all recorded tool calls.
// It is safe to call from multiple goroutines.
func (r *Recorder) Summary() Summary {
	r.mu.Lock()
	defer r.mu.Unlock()

	counts := make(map[string]int, len(r.events))
	for _, e := range r.events {
		counts[e.ToolName]++
	}
	return Summary{
		TotalCalls:  len(r.events),
		CallsByTool: counts,
		Events:      r.events,
	}
}

// Summary is a structured summary of a completed simulation run.
type Summary struct {
	TotalCalls  int             `json:"total_calls"`
	CallsByTool map[string]int  `json:"calls_by_tool"`
	Events      []ToolCallEvent `json:"events"`
}

// Print writes a human-readable summary to w.
func (s Summary) Print(w io.Writer) {
	fmt.Fprintf(w, "\n── Simulation Trace Summary ──────────────────────────\n")
	fmt.Fprintf(w, "  Total intercepted tool calls: %d\n", s.TotalCalls)
	if len(s.CallsByTool) > 0 {
		fmt.Fprintf(w, "  Breakdown by tool:\n")
		for tool, count := range s.CallsByTool {
			fmt.Fprintf(w, "    %-20s %d call(s)\n", tool, count)
		}
	}
	fmt.Fprintf(w, "──────────────────────────────────────────────────────\n")
}
