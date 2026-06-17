package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/saero-ai/xcaffold/internal/state"
	"github.com/saero-ai/xcaffold/providers"
)

var statusJSONFlag bool

type statusProviderJSON struct {
	Name         string `json:"name"`
	DisplayLabel string `json:"displayLabel"`
	Status       string `json:"status"`
	DeprecatedBy string `json:"deprecatedBy"`
	SunsetDate   string `json:"sunsetDate"`
	FileCount    int    `json:"fileCount"`
	DriftCount   int    `json:"driftCount"`
	OutputDir    string `json:"outputDir"`
	LastApplied  string `json:"lastApplied"`
}

type statusSourcesJSON struct {
	Total   int `json:"total"`
	Changed int `json:"changed"`
}

type statusOutputJSON struct {
	Project   string               `json:"project"`
	Blueprint string               `json:"blueprint"`
	Providers []statusProviderJSON `json:"providers"`
	Sources   statusSourcesJSON    `json:"sources"`
}

func init() {
	statusCmd.Flags().BoolVar(&statusJSONFlag, "json", false, "Output status as JSON")
}

// printStatusJSON marshals the status data to JSON and prints it.
func printStatusJSON(manifest *state.StateManifest, baseDir, blueprint string) error {
	providerEntries := make([]statusProviderJSON, 0)
	for _, name := range sortedTargetKeys(manifest.Targets) {
		if statusTargetFlag != "" && name != statusTargetFlag {
			continue
		}
		entry := buildStatusProviderEntry(manifest, baseDir, name)
		providerEntries = append(providerEntries, entry)
	}

	sourceFiles := aggregateSourceFiles(manifest)
	if len(sourceFiles) == 0 && len(manifest.SourceFiles) > 0 {
		sourceFiles = manifest.SourceFiles
	}
	srcChanged := countChangedSources(baseDir, sourceFiles)

	output := statusOutputJSON{
		Project:   filepath.Base(baseDir),
		Blueprint: blueprint,
		Providers: providerEntries,
		Sources: statusSourcesJSON{
			Total:   len(sourceFiles),
			Changed: srcChanged,
		},
	}

	b, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal status JSON: %w", err)
	}
	fmt.Println(string(b))
	return nil
}

// buildStatusProviderEntry constructs a provider entry for JSON output.
func buildStatusProviderEntry(manifest *state.StateManifest, baseDir, name string) statusProviderJSON {
	ts := manifest.Targets[name]
	_, outputDir := resolveStatusOutputDir(baseDir, name, ts, globalFlag)

	driftEntries := state.CollectDriftedFiles(baseDir, outputDir, ts)
	driftCount := len(driftEntries)

	displayLabel := name
	var status, deprecatedBy, sunsetDate string

	if pmf, ok := providers.ManifestFor(name); ok {
		if pmf.DisplayLabel != "" {
			displayLabel = pmf.DisplayLabel
		}
		status = pmf.Status
		if status == "" {
			status = "active"
		}
		deprecatedBy = pmf.DeprecatedBy
		sunsetDate = pmf.SunsetDate
	}

	return statusProviderJSON{
		Name:         name,
		DisplayLabel: displayLabel,
		Status:       status,
		DeprecatedBy: deprecatedBy,
		SunsetDate:   sunsetDate,
		FileCount:    len(ts.Artifacts),
		DriftCount:   driftCount,
		OutputDir:    outputDir,
		LastApplied:  ts.LastApplied,
	}
}
