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
	Short: "Show compilation state and check for drift across all providers",
	Long: `Shows the current state of all compiled providers.

Without --target, displays an overview of all providers: last apply time,
file count, and sync status. Drifted files are listed inline below
the summary table.

With --target, shows only the files that have drifted for that provider.
Use --all to see every tracked file.`,
	Example: `  $ xcaffold status
  $ xcaffold status --target claude
  $ xcaffold status --target claude --all`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runStatus,
}

func init() {
	statusCmd.Flags().StringVar(&statusTargetFlag, "target", "", "focus on a single provider")
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
		dir = globalXcafHome
	}

	statePath := state.StateFilePath(dir, statusBlueprintFlag)
	manifest, err := state.ReadState(statePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			projectName := filepath.Base(dir)
			fmt.Println(formatHeader(projectName, statusBlueprintFlag, globalFlag, "", ""))
			fmt.Println()
			fmt.Printf("  %s  No compilation state found.\n", glyphNever())
			fmt.Println()
			fmt.Printf("%s Run 'xcaffold apply' to compile all providers.\n", glyphArrow())
			return nil
		}
		return fmt.Errorf("could not read state: %w", err)
	}

	if statusTargetFlag != "" {
		return runStatusTarget(dir, manifest, statusTargetFlag, statusAllFlag)
	}
	return runStatusOverview(dir, manifest, statusAllFlag)
}

func runStatusOverview(dir string, manifest *state.StateManifest, showAll bool) error {
	projectName := filepath.Base(dir)
	lastApplied := findMostRecentApply(manifest.Targets)

	fmt.Println(formatHeader(projectName, statusBlueprintFlag, globalFlag, "", lastApplied))
	fmt.Println()

	rows, allDriftedFiles := buildProviderRows(dir, manifest)
	printProviderTable(rows)

	srcChanged := countChangedSources(dir, manifest.SourceFiles)
	printSourcesLine(len(manifest.SourceFiles), srcChanged)

	hasDrift := len(allDriftedFiles) > 0
	if hasDrift {
		printDriftBlock(manifest.Targets, allDriftedFiles)
	}

	if srcChanged > 0 {
		fmt.Println("\nSource changes:")
		printSourceChanges(dir, manifest.SourceFiles)
	}

	if showAll {
		printAllFilesPerProvider(dir, manifest)
	}

	return handleStatusOverviewCTA(hasDrift, srcChanged)
}

// findMostRecentApply finds the most recent apply timestamp across all providers.
func findMostRecentApply(targets map[string]state.TargetState) string {
	var lastApplied string
	var mostRecent time.Time
	for _, ts := range targets {
		t, err := time.Parse(time.RFC3339, ts.LastApplied)
		if err == nil && t.After(mostRecent) {
			mostRecent = t
			lastApplied = ts.LastApplied
		}
	}
	return lastApplied
}

// statusRow is used internally for building the provider table.
type statusRow struct {
	name    string
	count   int
	drifted int
	noState bool
}

// buildProviderRows constructs the table data and drifted files map.
func buildProviderRows(dir string, manifest *state.StateManifest) ([]statusRow, map[string][]state.DriftEntry) {
	var rows []statusRow
	allDriftedFiles := make(map[string][]state.DriftEntry)

	for _, name := range sortedTargetKeys(manifest.Targets) {
		ts := manifest.Targets[name]
		outputDir := filepath.Join(dir, compiler.OutputDir(name))
		entries := state.CollectDriftedFiles(dir, outputDir, ts)
		drifted := len(entries)
		rows = append(rows, statusRow{name: name, count: len(ts.Artifacts), drifted: drifted})
		if drifted > 0 {
			allDriftedFiles[name] = entries
		}
	}
	return rows, allDriftedFiles
}

// printProviderTable renders the provider status table.
func printProviderTable(rows []statusRow) {
	nameWidth := len("PROVIDER")
	for _, r := range rows {
		if len(r.name) > nameWidth {
			nameWidth = len(r.name)
		}
	}
	nameWidth += 2

	fmt.Printf("  %-*s  %5s   %s\n", nameWidth, "PROVIDER", "FILES", "STATUS")
	for _, r := range rows {
		var statusStr string
		if r.noState {
			statusStr = dim(glyphNever() + " never applied")
		} else if r.drifted > 0 {
			statusStr = colorRed(fmt.Sprintf("%s %d modified", glyphErr(), r.drifted))
		} else {
			statusStr = colorGreen(glyphOK() + " synced")
		}
		fmt.Printf("  %-*s  %5d   %s\n", nameWidth, r.name, r.count, statusStr)
	}
}

// printSourcesLine shows the sources status summary.
func printSourcesLine(sourceCount int, srcChanged int) {
	fmt.Printf("\n  Sources  %d .xcaf files  %s  ", sourceCount, glyphDot())
	if srcChanged == 0 {
		fmt.Println("no changes since last apply")
	} else {
		fmt.Printf("%d changed since last apply\n", srcChanged)
	}
}

// printDriftBlock shows all drifted artifacts grouped by provider.
func printDriftBlock(targets map[string]state.TargetState, allDriftedFiles map[string][]state.DriftEntry) {
	driftProviders := len(allDriftedFiles)
	fmt.Printf("\nDrift detected in %d %s:\n",
		driftProviders, plural(driftProviders, "provider", "providers"))
	for _, name := range sortedTargetKeys(targets) {
		entries, ok := allDriftedFiles[name]
		if !ok {
			continue
		}
		fmt.Printf("\n  %s\n", bold(name))
		for _, e := range entries {
			display, isRoot := formatArtifactPath(e.Path)
			annotation := ""
			if isRoot {
				annotation = "  " + dim("(root)")
			}
			var label string
			if e.Status == "missing" {
				label = colorRed("missing")
			} else {
				label = colorYellow("modified")
			}
			fmt.Printf("    %s  %-8s  %s%s\n", colorRed(glyphErr()), label, display, annotation)
		}
	}
}

// printAllFilesPerProvider shows per-provider grouped file listing when --all is set.
func printAllFilesPerProvider(dir string, manifest *state.StateManifest) {
	for _, name := range sortedTargetKeys(manifest.Targets) {
		ts := manifest.Targets[name]
		outputDir := filepath.Join(dir, compiler.OutputDir(name))
		fmt.Printf("\n  %s\n\n", bold(name))
		printAllFilesGrouped(dir, outputDir, ts)
	}
}

// handleStatusOverviewCTA returns the appropriate error or success status.
func handleStatusOverviewCTA(hasDrift bool, srcChanged int) error {
	if hasDrift || srcChanged > 0 {
		applyArgs := buildApplyCmd("", statusBlueprintFlag, globalFlag)
		fmt.Printf("\n%s Run '%s' to restore.\n", glyphArrow(), applyArgs)
		fmt.Printf("  Run 'xcaffold status --target <name>' for details.\n")
		return &driftDetectedError{msg: "drift detected"}
	}

	scopeLabel := "providers"
	if globalFlag {
		scopeLabel = "global providers"
	}
	fmt.Printf("\n%s All %s are in sync.\n", colorGreen(glyphOK()), scopeLabel)
	return nil
}

func runStatusTarget(dir string, manifest *state.StateManifest, target string, showAll bool) error {
	ts, ok := manifest.Targets[target]
	if !ok {
		return reportTargetNotFound(target, manifest.Targets)
	}

	projectName := filepath.Base(dir)
	outputDir := filepath.Join(dir, compiler.OutputDir(target))
	driftedEntries := state.CollectDriftedFiles(dir, outputDir, ts)
	drifted := len(driftedEntries)
	synced := len(ts.Artifacts) - drifted
	srcChanged := countChangedSources(dir, manifest.SourceFiles)

	fmt.Println(formatHeader(projectName, statusBlueprintFlag, globalFlag, target, ts.LastApplied))
	fmt.Println()

	printTargetSummary(synced, drifted, len(manifest.SourceFiles), srcChanged)
	if drifted > 0 {
		printDriftedArtifacts(driftedEntries)
	}

	printSourcesLine(len(manifest.SourceFiles), srcChanged)

	if showAll {
		fmt.Println()
		printAllFilesGrouped(dir, outputDir, ts)
	}

	return handleTargetCTA(target, drifted, srcChanged, showAll)
}

// reportTargetNotFound outputs an error message and suggestion for a missing provider.
func reportTargetNotFound(target string, targets map[string]state.TargetState) error {
	known := sortedTargetKeys(targets)
	fmt.Fprintf(os.Stderr, "%s No state for provider %q.\n\n", colorRed(glyphErr()), target)
	fmt.Fprintf(os.Stderr, "  Known providers:  %s\n\n", strings.Join(known, ", "))
	fmt.Fprintf(os.Stderr, "%s Run 'xcaffold apply --target %s' to compile it first.\n",
		glyphArrow(), target)
	return &driftDetectedError{msg: "no state for provider " + target}
}

// printTargetSummary shows the artifact and source status for a target.
func printTargetSummary(synced int, drifted int, sourceCount int, srcChanged int) {
	srcLabel := fmt.Sprintf("%d %s unchanged",
		sourceCount,
		plural(sourceCount, "source", "sources"))
	if drifted == 0 {
		fmt.Printf("  %s  %s  %s\n",
			bold(fmt.Sprintf("%d synced", synced)),
			glyphDot(),
			srcLabel,
		)
	} else {
		fmt.Printf("  %s  %s  %s  %s  %s\n",
			bold(fmt.Sprintf("%d synced", synced)),
			glyphDot(),
			colorRed(fmt.Sprintf("%d modified", drifted)),
			glyphDot(),
			srcLabel,
		)
	}
}

// printDriftedArtifacts outputs the list of drifted artifacts.
func printDriftedArtifacts(driftedEntries []state.DriftEntry) {
	fmt.Println()
	for _, e := range driftedEntries {
		display, isRoot := formatArtifactPath(e.Path)
		annotation := ""
		if isRoot {
			annotation = "  " + dim("(root)")
		}
		var label string
		if e.Status == "missing" {
			label = colorRed("missing")
		} else {
			label = colorYellow("modified")
		}
		fmt.Printf("  %s  %-8s  %s%s\n", colorRed(glyphErr()), label, display, annotation)
	}
}

// handleTargetCTA returns the appropriate error or success status for a target.
func handleTargetCTA(target string, drifted int, srcChanged int, showAll bool) error {
	if drifted > 0 || srcChanged > 0 {
		applyArgs := buildApplyCmd(target, statusBlueprintFlag, globalFlag)
		fmt.Printf("\n%s Run '%s' to restore.\n", glyphArrow(), applyArgs)
		if !showAll {
			fmt.Printf("  Run 'xcaffold status --target %s --all' to see all files.\n", target)
		}
		return &driftDetectedError{msg: "drift detected"}
	}

	fmt.Printf("\n%s Everything in sync.\n", colorGreen(glyphOK()))
	return nil
}

// buildApplyCmd constructs the apply command string with all active flags included.
func buildApplyCmd(target, blueprint string, isGlobal bool) string {
	cmd := "xcaffold apply"
	if isGlobal {
		cmd += " --global"
	}
	if blueprint != "" {
		cmd += " --blueprint " + blueprint
	}
	if target != "" {
		cmd += " --target " + target
	}
	return cmd
}

type driftEntry struct {
	status string
	path   string
}

func countChangedSources(baseDir string, sourceFiles []state.SourceFile) int {
	_, _, driftCount := findChangedSources(baseDir, sourceFiles)
	return driftCount
}

func printSourceChanges(baseDir string, sourceFiles []state.SourceFile) {
	entries, _, _ := findChangedSources(baseDir, sourceFiles)
	for _, e := range entries {
		var prefix string
		switch e.status {
		case "source changed":
			prefix = colorYellow(glyphSrc()) + "  source changed"
		case "new source":
			prefix = colorYellow(glyphSrc()) + "  new source    "
		case "source removed":
			prefix = colorRed(glyphSrc()) + "  source removed"
		default:
			prefix = e.status
		}
		fmt.Printf("  %s    %s\n", prefix, e.path)
	}
}

func findChangedSources(baseDir string, sourceFiles []state.SourceFile) ([]driftEntry, map[string]string, int) {
	currentSources, err := resolver.FindXCAFFiles(baseDir)
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

// fileGroup represents a directory group for file listing.
type fileGroup struct {
	name    string
	total   int
	drifted int
	entries []state.DriftEntry
}

func printAllFilesGrouped(baseDir, outputDir string, ts state.TargetState) {
	groupsMap := buildFileGroups(baseDir, outputDir, ts)

	var ordered []*fileGroup
	for _, g := range groupsMap {
		ordered = append(ordered, g)
	}
	sort.Slice(ordered, func(i, j int) bool {
		return ordered[i].name < ordered[j].name
	})

	printGroupTable(ordered)
}

// buildFileGroups scans artifacts and groups them by directory prefix.
func buildFileGroups(baseDir, outputDir string, ts state.TargetState) map[string]*fileGroup {
	groupsMap := make(map[string]*fileGroup)
	for _, artifact := range ts.Artifacts {
		absPath, displayPath := resolveArtifactPaths(baseDir, outputDir, artifact.Path)
		display, isRoot := formatArtifactPath(displayPath)
		groupName := getGroupName(display, isRoot)

		if groupsMap[groupName] == nil {
			groupsMap[groupName] = &fileGroup{name: groupName}
		}
		g := groupsMap[groupName]
		g.total++

		checkArtifactDrift(absPath, displayPath, artifact.Hash, g)
	}
	return groupsMap
}

// resolveArtifactPaths determines the absolute and display paths for an artifact.
func resolveArtifactPaths(baseDir, outputDir string, artifactPath string) (string, string) {
	var absPath string
	var displayPath string
	if strings.HasPrefix(artifactPath, "root:") {
		displayPath = strings.TrimPrefix(artifactPath, "root:")
		absPath = filepath.Clean(filepath.Join(baseDir, displayPath))
	} else {
		displayPath = artifactPath
		absPath = filepath.Clean(filepath.Join(outputDir, displayPath))
	}
	return absPath, displayPath
}

// getGroupName determines the directory group for a file.
func getGroupName(display string, isRoot bool) string {
	if isRoot {
		return "(root)"
	}
	idx := strings.Index(display, string(filepath.Separator))
	if idx != -1 {
		return display[:idx+1]
	}
	return "(root)"
}

// checkArtifactDrift checks if an artifact has drifted and records it.
func checkArtifactDrift(absPath string, displayPath string, expectedHash string, g *fileGroup) {
	data, err := os.ReadFile(absPath)
	if err != nil {
		g.drifted++
		g.entries = append(g.entries, state.DriftEntry{Status: "missing", Path: displayPath})
		return
	}
	h := sha256.Sum256(data)
	if fmt.Sprintf("sha256:%x", h) != expectedHash {
		g.drifted++
		g.entries = append(g.entries, state.DriftEntry{Status: "modified", Path: displayPath})
	}
}

// printGroupTable renders the grouped file listing table.
func printGroupTable(ordered []*fileGroup) {
	fmt.Printf("  %-16s  %5s   %s\n", "GROUP", "FILES", "STATUS")
	for _, g := range ordered {
		var statusStr string
		if g.drifted > 0 {
			statusStr = colorRed(fmt.Sprintf("%s %d modified", glyphErr(), g.drifted))
		} else {
			statusStr = colorGreen(glyphOK() + " synced")
		}
		fmt.Printf("  %-16s  %5d   %s\n", g.name, g.total, statusStr)
		printGroupEntries(g.entries)
	}
}

// printGroupEntries outputs the drifted entries within a group.
func printGroupEntries(entries []state.DriftEntry) {
	for _, e := range entries {
		display, isRoot := formatArtifactPath(e.Path)
		annotation := ""
		if isRoot {
			annotation = "  " + dim("(root)")
		}
		var label string
		if e.Status == "missing" {
			label = colorRed("missing")
		} else {
			label = colorYellow("modified")
		}
		fmt.Printf("    %s  %-8s  %s%s\n", colorRed(glyphErr()), label, display, annotation)
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
