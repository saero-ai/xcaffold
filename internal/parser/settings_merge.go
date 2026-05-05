package parser

import (
	"fmt"
	"path/filepath"
	"reflect"

	"github.com/saero-ai/xcaffold/internal/ast"
)

// isEmptySettings reports whether s is a zero-value SettingsConfig.
func isEmptySettings(s ast.SettingsConfig) bool {
	return reflect.DeepEqual(s, ast.SettingsConfig{})
}

// mergeSettingsStrict merges two SettingsConfig values from the same directory.
// Scalar conflicts (both set to different values) produce an error naming both
// source files.  Maps are merged additively; duplicate map keys with different
// values are an error.  Hooks are always appended.
func mergeSettingsStrict(base, child ast.SettingsConfig, baseFile, childFile string) (ast.SettingsConfig, error) {
	if isEmptySettings(child) {
		return base, nil
	}
	if isEmptySettings(base) {
		return child, nil
	}

	bf := filepath.Base(baseFile)
	cf := filepath.Base(childFile)
	conflict := func(field string) error {
		return fmt.Errorf("settings conflict for %q between %s and %s", field, bf, cf)
	}

	out := base // shallow copy

	// --- string scalars ---
	strFields := []struct {
		name     string
		baseVal  string
		childVal string
		dst      *string
	}{
		{"effortLevel", base.EffortLevel, child.EffortLevel, &out.EffortLevel},
		{"defaultShell", base.DefaultShell, child.DefaultShell, &out.DefaultShell},
		{"language", base.Language, child.Language, &out.Language},
		{"outputStyle", base.OutputStyle, child.OutputStyle, &out.OutputStyle},
		{"plansDirectory", base.PlansDirectory, child.PlansDirectory, &out.PlansDirectory},
		{"model", base.Model, child.Model, &out.Model},
		{"otelHeadersHelper", base.OtelHeadersHelper, child.OtelHeadersHelper, &out.OtelHeadersHelper},
		{"autoMemoryDirectory", base.AutoMemoryDirectory, child.AutoMemoryDirectory, &out.AutoMemoryDirectory},
	}
	for _, f := range strFields {
		if f.childVal != "" {
			if f.baseVal != "" && f.baseVal != f.childVal {
				return ast.SettingsConfig{}, conflict(f.name)
			}
			*f.dst = f.childVal
		}
	}

	// --- *bool pointers ---
	boolFields := []struct {
		name     string
		baseVal  *bool
		childVal *bool
		dst      **bool
	}{
		{"includeGitInstructions", base.IncludeGitInstructions, child.IncludeGitInstructions, &out.IncludeGitInstructions},
		{"skipDangerousModePermissionPrompt", base.SkipDangerousModePermissionPrompt, child.SkipDangerousModePermissionPrompt, &out.SkipDangerousModePermissionPrompt},
		{"autoMemoryEnabled", base.AutoMemoryEnabled, child.AutoMemoryEnabled, &out.AutoMemoryEnabled},
		{"disableAllHooks", base.DisableAllHooks, child.DisableAllHooks, &out.DisableAllHooks},
		{"attribution", base.Attribution, child.Attribution, &out.Attribution},
		{"respectGitignore", base.RespectGitignore, child.RespectGitignore, &out.RespectGitignore},
		{"disableSkillShellExecution", base.DisableSkillShellExecution, child.DisableSkillShellExecution, &out.DisableSkillShellExecution},
		{"alwaysThinkingEnabled", base.AlwaysThinkingEnabled, child.AlwaysThinkingEnabled, &out.AlwaysThinkingEnabled},
	}
	for _, f := range boolFields {
		if f.childVal != nil {
			if f.baseVal != nil && *f.baseVal != *f.childVal {
				return ast.SettingsConfig{}, conflict(f.name)
			}
			*f.dst = f.childVal
		}
	}

	// --- *int pointer ---
	if child.CleanupPeriodDays != nil {
		if base.CleanupPeriodDays != nil && *base.CleanupPeriodDays != *child.CleanupPeriodDays {
			return ast.SettingsConfig{}, conflict("cleanupPeriodDays")
		}
		out.CleanupPeriodDays = child.CleanupPeriodDays
	}

	// --- any fields (Agent, Worktree, AutoMode) ---
	anyFields := []struct {
		name     string
		baseVal  any
		childVal any
		dst      *any
	}{
		{"agent", base.Agent, child.Agent, &out.Agent},
		{"worktree", base.Worktree, child.Worktree, &out.Worktree},
		{"autoMode", base.AutoMode, child.AutoMode, &out.AutoMode},
	}
	for _, f := range anyFields {
		if f.childVal != nil {
			if f.baseVal != nil {
				return ast.SettingsConfig{}, conflict(f.name)
			}
			*f.dst = f.childVal
		}
	}

	// --- struct pointers (Permissions, Sandbox, StatusLine) ---
	if child.Permissions != nil {
		if base.Permissions != nil {
			return ast.SettingsConfig{}, conflict("permissions")
		}
		out.Permissions = child.Permissions
	}
	if child.Sandbox != nil {
		if base.Sandbox != nil {
			return ast.SettingsConfig{}, conflict("sandbox")
		}
		out.Sandbox = child.Sandbox
	}
	if child.StatusLine != nil {
		if base.StatusLine != nil {
			return ast.SettingsConfig{}, conflict("statusLine")
		}
		out.StatusLine = child.StatusLine
	}

	// --- map[string]string (Env) ---
	merged, err := mergeStringMapStrict(base.Env, child.Env, "env", bf, cf)
	if err != nil {
		return ast.SettingsConfig{}, err
	}
	out.Env = merged

	// --- map[string]bool (EnabledPlugins) ---
	mergedPlugins, err := mergeBoolMapStrict(base.EnabledPlugins, child.EnabledPlugins, "enabledPlugins", bf, cf)
	if err != nil {
		return ast.SettingsConfig{}, err
	}
	out.EnabledPlugins = mergedPlugins

	// --- map[string]MCPConfig (MCPServers) ---
	mergedMCP, err := mergeMCPMapStrict(base.MCPServers, child.MCPServers, "mcpServers", bf, cf)
	if err != nil {
		return ast.SettingsConfig{}, err
	}
	out.MCPServers = mergedMCP

	// --- []string (AvailableModels, MdExcludes) ---
	out.AvailableModels = appendUnique(base.AvailableModels, child.AvailableModels)
	out.MdExcludes = appendUnique(base.MdExcludes, child.MdExcludes)

	// --- HookConfig (additive) ---
	out.Hooks = mergeHooksAdditive(base.Hooks, child.Hooks)

	return out, nil
}

// mergeSettingsOverride merges two SettingsConfig values for extends resolution
// (global -> project).  The child always wins on conflict.
func mergeSettingsOverride(base, child ast.SettingsConfig) ast.SettingsConfig {
	out := base

	// --- string scalars: child wins if non-empty ---
	if child.EffortLevel != "" {
		out.EffortLevel = child.EffortLevel
	}
	if child.DefaultShell != "" {
		out.DefaultShell = child.DefaultShell
	}
	if child.Language != "" {
		out.Language = child.Language
	}
	if child.OutputStyle != "" {
		out.OutputStyle = child.OutputStyle
	}
	if child.PlansDirectory != "" {
		out.PlansDirectory = child.PlansDirectory
	}
	if child.Model != "" {
		out.Model = child.Model
	}
	if child.OtelHeadersHelper != "" {
		out.OtelHeadersHelper = child.OtelHeadersHelper
	}
	if child.AutoMemoryDirectory != "" {
		out.AutoMemoryDirectory = child.AutoMemoryDirectory
	}

	// --- *bool: child wins if non-nil ---
	if child.IncludeGitInstructions != nil {
		out.IncludeGitInstructions = child.IncludeGitInstructions
	}
	if child.SkipDangerousModePermissionPrompt != nil {
		out.SkipDangerousModePermissionPrompt = child.SkipDangerousModePermissionPrompt
	}
	if child.AutoMemoryEnabled != nil {
		out.AutoMemoryEnabled = child.AutoMemoryEnabled
	}
	if child.DisableAllHooks != nil {
		out.DisableAllHooks = child.DisableAllHooks
	}
	if child.Attribution != nil {
		out.Attribution = child.Attribution
	}
	if child.RespectGitignore != nil {
		out.RespectGitignore = child.RespectGitignore
	}
	if child.DisableSkillShellExecution != nil {
		out.DisableSkillShellExecution = child.DisableSkillShellExecution
	}
	if child.AlwaysThinkingEnabled != nil {
		out.AlwaysThinkingEnabled = child.AlwaysThinkingEnabled
	}

	// --- *int: child wins if non-nil ---
	if child.CleanupPeriodDays != nil {
		out.CleanupPeriodDays = child.CleanupPeriodDays
	}

	// --- any: child wins if non-nil ---
	if child.Agent != nil {
		out.Agent = child.Agent
	}
	if child.Worktree != nil {
		out.Worktree = child.Worktree
	}
	if child.AutoMode != nil {
		out.AutoMode = child.AutoMode
	}

	// --- struct pointers: child wins if non-nil ---
	if child.Permissions != nil {
		out.Permissions = child.Permissions
	}
	if child.Sandbox != nil {
		out.Sandbox = child.Sandbox
	}
	if child.StatusLine != nil {
		out.StatusLine = child.StatusLine
	}

	// --- maps: child keys override base ---
	out.Env = mergeStringMapOverride(base.Env, child.Env)
	out.EnabledPlugins = mergeBoolMapOverride(base.EnabledPlugins, child.EnabledPlugins)
	out.MCPServers = mergeMCPMapOverride(base.MCPServers, child.MCPServers)

	// --- slices: additive (append unique) ---
	out.AvailableModels = appendUnique(base.AvailableModels, child.AvailableModels)
	out.MdExcludes = appendUnique(base.MdExcludes, child.MdExcludes)

	// --- hooks: additive ---
	out.Hooks = mergeHooksAdditive(base.Hooks, child.Hooks)

	return out
}

// ---------------------------------------------------------------------------
// Map merge helpers
// ---------------------------------------------------------------------------

func mergeStringMapStrict(base, child map[string]string, field, bf, cf string) (map[string]string, error) {
	if len(child) == 0 {
		return base, nil
	}
	if len(base) == 0 {
		return child, nil
	}
	merged := make(map[string]string, len(base)+len(child))
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range child {
		if existing, ok := merged[k]; ok && existing != v {
			return nil, fmt.Errorf("settings conflict for %q key %q between %s and %s", field, k, bf, cf)
		}
		merged[k] = v
	}
	return merged, nil
}

func mergeBoolMapStrict(base, child map[string]bool, field, bf, cf string) (map[string]bool, error) {
	if len(child) == 0 {
		return base, nil
	}
	if len(base) == 0 {
		return child, nil
	}
	merged := make(map[string]bool, len(base)+len(child))
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range child {
		if existing, ok := merged[k]; ok && existing != v {
			return nil, fmt.Errorf("settings conflict for %q key %q between %s and %s", field, k, bf, cf)
		}
		merged[k] = v
	}
	return merged, nil
}

func mergeMCPMapStrict(base, child map[string]ast.MCPConfig, field, bf, cf string) (map[string]ast.MCPConfig, error) {
	if len(child) == 0 {
		return base, nil
	}
	if len(base) == 0 {
		return child, nil
	}
	merged := make(map[string]ast.MCPConfig, len(base)+len(child))
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range child {
		if _, ok := merged[k]; ok {
			return nil, fmt.Errorf("settings conflict for %q key %q between %s and %s", field, k, bf, cf)
		}
		merged[k] = v
	}
	return merged, nil
}

func mergeStringMapOverride(base, child map[string]string) map[string]string {
	if len(child) == 0 {
		return base
	}
	if len(base) == 0 {
		return child
	}
	merged := make(map[string]string, len(base)+len(child))
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range child {
		merged[k] = v
	}
	return merged
}

func mergeBoolMapOverride(base, child map[string]bool) map[string]bool {
	if len(child) == 0 {
		return base
	}
	if len(base) == 0 {
		return child
	}
	merged := make(map[string]bool, len(base)+len(child))
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range child {
		merged[k] = v
	}
	return merged
}

func mergeMCPMapOverride(base, child map[string]ast.MCPConfig) map[string]ast.MCPConfig {
	if len(child) == 0 {
		return base
	}
	if len(base) == 0 {
		return child
	}
	merged := make(map[string]ast.MCPConfig, len(base)+len(child))
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range child {
		merged[k] = v
	}
	return merged
}

// appendUnique appends items from child that are not already in base.
func appendUnique(base, child []string) []string {
	if len(child) == 0 {
		return base
	}
	seen := make(map[string]struct{}, len(base))
	for _, v := range base {
		seen[v] = struct{}{}
	}
	out := make([]string, len(base))
	copy(out, base)
	for _, v := range child {
		if _, exists := seen[v]; !exists {
			out = append(out, v)
			seen[v] = struct{}{}
		}
	}
	return out
}
