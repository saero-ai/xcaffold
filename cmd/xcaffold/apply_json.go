package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

var applyJSON bool
var jsonOutputWriter io.Writer = os.Stdout

// providerEvent represents a single provider compilation event in NDJSON output.
type providerEvent struct {
	Event        string `json:"event"`
	Provider     string `json:"provider"`
	DisplayLabel string `json:"displayLabel,omitempty"`
	FileCount    int    `json:"fileCount"`
	OutputDir    string `json:"outputDir"`
}

// summaryEvent represents the final summary event in NDJSON output.
type summaryEvent struct {
	Event          string `json:"event"`
	TotalProviders int    `json:"totalProviders"`
	TotalFiles     int    `json:"totalFiles"`
	Duration       string `json:"duration,omitempty"`
}

// emitJSONLine marshals an event to JSON and writes it as a single line.
func emitJSONLine(evt interface{}) error {
	b, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	_, _ = fmt.Fprintln(jsonOutputWriter, string(b))
	return nil
}

func init() {
	applyCmd.Flags().BoolVar(&applyJSON, "json", false, "Output compilation events as NDJSON")
}
