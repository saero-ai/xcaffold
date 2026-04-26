package main

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/saero-ai/xcaffold/internal/compiler"
	"github.com/saero-ai/xcaffold/internal/resolver"
	"github.com/saero-ai/xcaffold/internal/state"
	"github.com/spf13/cobra"
)

var statusTargetFlag string
var statusBlueprintFlag string
var statusAllFlag bool

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show compilation state and check for drift across all targets",
	Long: `Shows the current state of all compiled targets.

Without --target, displays an overview of all targets: last apply time,
artifact count, and sync status. Drifted files are listed inline below
the summary table.

With --target, shows only the files that have drifted for that target.
Use --all to see every tracked file.`,
	Example: `  $ xcaffold status
  $ xcaffold status --target claude
  $ xcaffold status --target claude --all`,
	SilenceUsage: true,
	RunE:         runStatus,
}

func init() {
	statusCmd.Flags().StringVar(&statusTargetFlag, "target", "", "focus on a single target")
	statusCmd.Flags().StringVar(&statusBlueprintFlag, "blueprint", "", "filter by blueprint")
	statusCmd.Flags().BoolVar(&statusAllFlag, "all", false, "show all files (default: drifted only)")
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	if statusBlueprintFlag != "" && globalFlag {
		return fmt.Errorf("--blueprint cannot be used with --global")
	}

	dir := projectRoot
	if globalFlag {
		dir = globalXcfHome
	}

	statePath := state.StateFilePath(dir, statusBlueprintFlag)
	manifest, err := state.ReadState(statePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Println("No state found. Run 'xcaffold apply' to compile.")
			return nil
		}
		return fmt.Errorf("could not read state: %w", err)
	}

	if statusTargetFlag != "" {
		return runStatusTarget(dir, manifest, statusTargetFlag, statusAllFlag)
	}
	return runStatusOverview(dir, manifest)
}

func runStatusOverview(dir string, manifest *state.StateManifest) error {
	projectName := filepath.Base(dir)
	var lastApplied string
	var mostRecent time.Time
	for _, ts := range manifest.Targets {
		t, err := time.Parse(time.RFC3339, ts.LastApplied)
		if err == nil && t.After(mostRecent) {
			mostRecent = t
			lastApplied = ts.LastApplied
		}
	}

	fmt.Printf("%s  ·  last applied %s\n\n", projectName, formatElapsed(lastApplied))

	type targetRow struct {
		name    string
		count   int
		drifted int
		noState bool
	}
	var rows []targetRow
	allDriftedFiles := make(map[string][]driftEntry)

	nameWidth := 0
	for _, name := range sortedTargetKeys(manifest.Targets) {
		ts := manifest.Targets[name]
		outputDir := filepath.Join(dir, compiler.OutputDir(name))
		drifted, entries := collectDriftedFiles(outputDir, ts)
		rows = append(rows, targetRow{name: name, count: len(ts.Artifacts), drifted: drifted})
		if drifted > 0 {
			allDriftedFiles[name] = entries
		}
		if len(name) > nameWidth {
			nameWidth = len(name)
		}
	}
	nameWidth += 2

	for _, r := range rows {
		status := "synced"
		if r.noState {
			status = "not applied yet"
		} else if r.drifted > 0 {
			status = fmt.Sprintf("%d modified", r.drifted)
		}
		fmt.Printf("  %-*s  %d artifacts    %s\n", nameWidth, r.name, r.count, status)
	}

	fmt.Printf("\nSources  %d .xcf files  ·  ", len(manifest.SourceFiles))
	srcChanged := countChangedSources(dir, manifest.SourceFiles)
	if srcChanged == 0 {
		fmt.Println("no changes since last apply")
	} else {
		fmt.Printf("%d changed\n", srcChanged)
	}

	if len(allDriftedFiles) > 0 {
		fmt.Println("\nModified files:")
		for _, name := range sortedTargetKeys(manifest.Targets) {
			if entries, ok := allDriftedFiles[name]; ok {
				fmt.Printf("\n  %s\n", name)
				for _, e := range entries {
					fmt.Printf("    %-16s  %s\n", e.status, e.path)
				}
			}
		}
		fmt.Println("\nRun 'xcaffold apply' to restore.")
		fmt.Println("Run 'xcaffold status --target <name>' to inspect a specific target.")
	} else {
		if srcChanged > 0 {
			fmt.Println("\nSource changes:")
			printSourceChanges(dir, manifest.SourceFiles)
			fmt.Println("\nRun 'xcaffold apply' to sync.")
		} else {
			fmt.Println("\nEverything is in sync.")
		}
	}

	return nil
}

func runStatusTarget(dir string, manifest *state.StateManifest, target string, showAll bool) error {
	ts, ok := manifest.Targets[target]
	if !ok {
		return fmt.Errorf("no state found for target %q — run 'xcaffold apply --target %s' first", target, target)
	}

	outputDir := filepath.Join(dir, compiler.OutputDir(target))
	projectName := filepath.Base(dir)
	drifted, driftedEntries := collectDriftedFiles(outputDir, ts)
	synced := len(ts.Artifacts) - drifted

	fmt.Printf("%s  ·  %s  ·  applied %s  ·  %d artifacts\n\n",
		projectName, target, formatElapsed(ts.LastApplied), len(ts.Artifacts))

	if drifted == 0 {
		fmt.Printf("  %d synced  ·  everything in sync\n", synced)
	} else {
		fmt.Printf("  %d synced  ·  %d modified\n\n", synced, drifted)
		for _, e := range driftedEntries {
			fmt.Printf("  %-16s  %s\n", e.status, e.path)
		}
	}

	fmt.Printf("\nSources  %d .xcf files  ·  ", len(manifest.SourceFiles))
	srcChanged := countChangedSources(dir, manifest.SourceFiles)
	if srcChanged == 0 {
		fmt.Println("no changes since last apply")
	} else {
		fmt.Printf("%d changed since last apply\n", srcChanged)
	}

	if showAll {
		fmt.Println()
		printAllFilesGrouped(outputDir, ts)
	} else if drifted > 0 {
		fmt.Printf("\nRun 'xcaffold apply --target %s' to restore.\n", target)
		fmt.Printf("Run 'xcaffold status --target %s --all' to see all files.\n", target)
	}

	return nil
}

type driftEntry struct {
	status string
	path   string
}

func collectDriftedFiles(outputDir string, ts state.TargetState) (int, []driftEntry) {
	var entries []driftEntry
	for _, artifact := range ts.Artifacts {
		absPath := filepath.Clean(filepath.Join(outputDir, artifact.Path))
		data, err := os.ReadFile(absPath)
		if err != nil {
			entries = append(entries, driftEntry{"not on disk", artifact.Path})
			continue
		}
		h := sha256.Sum256(data)
		if fmt.Sprintf("sha256:%x", h) != artifact.Hash {
			entries = append(entries, driftEntry{"modified", artifact.Path})
		}
	}
	return len(entries), entries
}

func countChangedSources(baseDir string, sourceFiles []state.SourceFile) int {
	_, _, driftCount := findChangedSources(baseDir, sourceFiles)
	return driftCount
}

func printSourceChanges(baseDir string, sourceFiles []state.SourceFile) {
	entries, _, _ := findChangedSources(baseDir, sourceFiles)
	for _, e := range entries {
		fmt.Printf("    %-16s  %s\n", e.status, e.path)
	}
}

func findChangedSources(baseDir string, sourceFiles []state.SourceFile) ([]driftEntry, map[string]string, int) {
	currentSources, err := resolver.FindXCFFiles(baseDir)
	if err != nil {
		return nil, nil, 0
	}

	prevByPath := make(map[string]string)
	for _, sf := range sourceFiles {
		prevByPath[sf.Path] = sf.Hash
	}

	currByPath := make(map[string]string)
	for _, absPath := range currentSources {
		rel, err := filepath.Rel(baseDir, absPath)
		if err != nil {
			continue
		}
		data, err := os.ReadFile(absPath)
		if err == nil {
			hash := sha256.Sum256(data)
			currByPath[rel] = fmt.Sprintf("sha256:%x", hash)
		}
	}

	var entries []driftEntry
	driftCount := 0

	for _, sf := range sourceFiles {
		if currHash, exists := currByPath[sf.Path]; !exists {
			entries = append(entries, driftEntry{"source removed", sf.Path})
			driftCount++
		} else if currHash != sf.Hash {
			entries = append(entries, driftEntry{"source changed", sf.Path})
			driftCount++
		}
	}
	var newSources []string
	for rel := range currByPath {
		if _, exists := prevByPath[rel]; !exists {
			newSources = append(newSources, rel)
			driftCount++
		}
	}
	sort.Strings(newSources)
	for _, rel := range newSources {
		entries = append(entries, driftEntry{"new source", rel})
	}

	return entries, currByPath, driftCount
}

func printAllFilesGrouped(outputDir string, ts state.TargetState) {
	type group struct {
		name    string
		total   int
		drifted int
		entries []driftEntry
	}

	groupsMap := make(map[string]*group)
	for _, artifact := range ts.Artifacts {
		absPath := filepath.Clean(filepath.Join(outputDir, artifact.Path))
		idx := strings.Index(artifact.Path, string(filepath.Separator))
		groupName := "(root)"
		if idx != -1 {
			groupName = artifact.Path[:idx+1]
		}

		if groupsMap[groupName] == nil {
			groupsMap[groupName] = &group{name: groupName}
		}
		g := groupsMap[groupName]
		g.total++

		data, err := os.ReadFile(absPath)
		if err != nil {
			g.drifted++
			g.entries = append(g.entries, driftEntry{"not on disk", artifact.Path})
			continue
		}
		h := sha256.Sum256(data)
		if fmt.Sprintf("sha256:%x", h) != artifact.Hash {
			g.drifted++
			g.entries = append(g.entries, driftEntry{"modified", artifact.Path})
		}
	}

	var ordered []*group
	for _, g := range groupsMap {
		ordered = append(ordered, g)
	}
	sort.Slice(ordered, func(i, j int) bool {
		return ordered[i].name < ordered[j].name
	})

	for _, g := range ordered {
		status := "synced"
		if g.drifted > 0 {
			status = fmt.Sprintf("%d modified", g.drifted)
		}
		fileWord := "file"
		if g.total != 1 {
			fileWord = "files"
		}
		fmt.Printf("  %-16s %2d %-7s  %s\n", g.name, g.total, fileWord, status)
		for _, e := range g.entries {
			fmt.Printf("    %-15s %s\n", e.status, e.path)
		}
	}
}

func formatElapsed(lastApplied string) string {
	t, err := time.Parse(time.RFC3339, lastApplied)
	if err != nil {
		return "unknown"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%d minutes ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%d hours ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%d days ago", int(d.Hours()/24))
	}
}

func sortedTargetKeys(m map[string]state.TargetState) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
