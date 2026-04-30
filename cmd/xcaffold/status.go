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
		dir = globalXcfHome
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
	return runStatusOverview(dir, manifest)
}

func runStatusOverview(dir string, manifest *state.StateManifest) error {
	projectName := filepath.Base(dir)

	// Find the most recent apply timestamp across all providers.
	var lastApplied string
	var mostRecent time.Time
	for _, ts := range manifest.Targets {
		t, err := time.Parse(time.RFC3339, ts.LastApplied)
		if err == nil && t.After(mostRecent) {
			mostRecent = t
			lastApplied = ts.LastApplied
		}
	}

	fmt.Println(formatHeader(projectName, statusBlueprintFlag, globalFlag, "", lastApplied))
	fmt.Println()

	// Collect per-provider rows and drifted file entries.
	type targetRow struct {
		name    string
		count   int
		drifted int
		noState bool
	}
	var rows []targetRow
	allDriftedFiles := make(map[string][]driftEntry)
	outputDirByProvider := make(map[string]string)

	nameWidth := len("PROVIDER")
	for _, name := range sortedTargetKeys(manifest.Targets) {
		if len(name) > nameWidth {
			nameWidth = len(name)
		}
	}
	nameWidth += 2

	for _, name := range sortedTargetKeys(manifest.Targets) {
		ts := manifest.Targets[name]
		outputDir := filepath.Join(dir, compiler.OutputDir(name))
		outputDirByProvider[name] = outputDir
		drifted, entries := collectDriftedFiles(dir, outputDir, ts)
		rows = append(rows, targetRow{name: name, count: len(ts.Artifacts), drifted: drifted})
		if drifted > 0 {
			allDriftedFiles[name] = entries
		}
	}

	// Provider table.
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

	// Sources line.
	srcChanged := countChangedSources(dir, manifest.SourceFiles)
	fmt.Printf("\n  Sources  %d .xcf files  %s  ", len(manifest.SourceFiles), glyphDot())
	if srcChanged == 0 {
		fmt.Println("no changes since last apply")
	} else {
		fmt.Printf("%d changed since last apply\n", srcChanged)
	}

	// Drift block.
	hasDrift := len(allDriftedFiles) > 0
	if hasDrift {
		driftProviders := len(allDriftedFiles)
		fmt.Printf("\nDrift detected in %d %s:\n",
			driftProviders, plural(driftProviders, "provider", "providers"))
		for _, name := range sortedTargetKeys(manifest.Targets) {
			entries, ok := allDriftedFiles[name]
			if !ok {
				continue
			}
			fmt.Printf("\n  %s\n", bold(name))
			for _, e := range entries {
				display, isRoot := formatArtifactPath(e.path)
				annotation := ""
				if isRoot {
					annotation = "  " + dim("(root)")
				}
				var label string
				if e.status == "missing" {
					label = colorRed("missing")
				} else {
					label = colorYellow("modified")
				}
				fmt.Printf("    %s  %-8s  %s%s\n", colorRed(glyphErr()), label, display, annotation)
			}
		}
	}

	// Source-change block — always shown when sources changed, even alongside artifact drift.
	if srcChanged > 0 {
		fmt.Println("\nSource changes:")
		printSourceChanges(dir, manifest.SourceFiles)
	}

	// CTA or success line.
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
		known := sortedTargetKeys(manifest.Targets)
		fmt.Fprintf(os.Stderr, "%s No state for provider %q.\n\n", colorRed(glyphErr()), target)
		fmt.Fprintf(os.Stderr, "  Known providers:  %s\n\n", strings.Join(known, ", "))
		fmt.Fprintf(os.Stderr, "%s Run 'xcaffold apply --target %s' to compile it first.\n",
			glyphArrow(), target)
		return &driftDetectedError{msg: "no state for provider " + target}
	}

	projectName := filepath.Base(dir)
	outputDir := filepath.Join(dir, compiler.OutputDir(target))
	drifted, driftedEntries := collectDriftedFiles(dir, outputDir, ts)
	synced := len(ts.Artifacts) - drifted
	srcChanged := countChangedSources(dir, manifest.SourceFiles)

	fmt.Println(formatHeader(projectName, statusBlueprintFlag, globalFlag, target, ts.LastApplied))
	fmt.Println()

	// Summary line.
	srcLabel := fmt.Sprintf("%d %s unchanged",
		len(manifest.SourceFiles),
		plural(len(manifest.SourceFiles), "source", "sources"))
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
		fmt.Println()
		for _, e := range driftedEntries {
			display, isRoot := formatArtifactPath(e.path)
			annotation := ""
			if isRoot {
				annotation = "  " + dim("(root)")
			}
			var label string
			if e.status == "missing" {
				label = colorRed("missing")
			} else {
				label = colorYellow("modified")
			}
			fmt.Printf("  %s  %-8s  %s%s\n", colorRed(glyphErr()), label, display, annotation)
		}
	}

	// Sources line.
	fmt.Printf("\n  Sources  %d .xcf files  %s  ", len(manifest.SourceFiles), glyphDot())
	if srcChanged == 0 {
		fmt.Println("no changes since last apply")
	} else {
		fmt.Printf("%d changed since last apply\n", srcChanged)
	}

	// --all grouped listing.
	if showAll {
		fmt.Println()
		printAllFilesGrouped(dir, outputDir, ts)
	}

	// CTA or success line.
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

func collectDriftedFiles(baseDir, outputDir string, ts state.TargetState) (int, []driftEntry) {
	var entries []driftEntry
	for _, artifact := range ts.Artifacts {
		// Handle root: prefix — strip it and use baseDir instead of outputDir
		var absPath string
		var storagePath string
		if strings.HasPrefix(artifact.Path, "root:") {
			// Root-level file: strip "root:" and resolve from baseDir
			storagePath = strings.TrimPrefix(artifact.Path, "root:")
			absPath = filepath.Clean(filepath.Join(baseDir, storagePath))
		} else {
			// Provider-scoped file: use outputDir
			storagePath = artifact.Path
			absPath = filepath.Clean(filepath.Join(outputDir, storagePath))
		}

		data, err := os.ReadFile(absPath)
		if err != nil {
			entries = append(entries, driftEntry{"missing", artifact.Path})
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

func printAllFilesGrouped(baseDir, outputDir string, ts state.TargetState) {
	type group struct {
		name    string
		total   int
		drifted int
		entries []driftEntry
	}

	groupsMap := make(map[string]*group)
	for _, artifact := range ts.Artifacts {
		// Handle root: prefix
		var absPath string
		var displayPath string
		if strings.HasPrefix(artifact.Path, "root:") {
			displayPath = strings.TrimPrefix(artifact.Path, "root:")
			absPath = filepath.Clean(filepath.Join(baseDir, displayPath))
		} else {
			displayPath = artifact.Path
			absPath = filepath.Clean(filepath.Join(outputDir, displayPath))
		}

		display, isRoot := formatArtifactPath(displayPath)

		var groupName string
		if isRoot {
			groupName = "(root)"
		} else {
			idx := strings.Index(display, string(filepath.Separator))
			if idx != -1 {
				groupName = display[:idx+1]
			} else {
				groupName = "(root)"
			}
		}

		if groupsMap[groupName] == nil {
			groupsMap[groupName] = &group{name: groupName}
		}
		g := groupsMap[groupName]
		g.total++

		data, err := os.ReadFile(absPath)
		if err != nil {
			g.drifted++
			g.entries = append(g.entries, driftEntry{"missing", displayPath})
			continue
		}
		h := sha256.Sum256(data)
		if fmt.Sprintf("sha256:%x", h) != artifact.Hash {
			g.drifted++
			g.entries = append(g.entries, driftEntry{"modified", displayPath})
		}
	}

	var ordered []*group
	for _, g := range groupsMap {
		ordered = append(ordered, g)
	}
	sort.Slice(ordered, func(i, j int) bool {
		return ordered[i].name < ordered[j].name
	})

	fmt.Printf("  %-16s  %5s   %s\n", "GROUP", "FILES", "STATUS")
	for _, g := range ordered {
		var statusStr string
		if g.drifted > 0 {
			statusStr = colorRed(fmt.Sprintf("%s %d modified", glyphErr(), g.drifted))
		} else {
			statusStr = colorGreen(glyphOK() + " synced")
		}
		fmt.Printf("  %-16s  %5d   %s\n", g.name, g.total, statusStr)
		for _, e := range g.entries {
			display, isRoot := formatArtifactPath(e.path)
			annotation := ""
			if isRoot {
				annotation = "  " + dim("(root)")
			}
			var label string
			if e.status == "missing" {
				label = colorRed("missing")
			} else {
				label = colorYellow("modified")
			}
			fmt.Printf("    %s  %-8s  %s%s\n", colorRed(glyphErr()), label, display, annotation)
		}
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
