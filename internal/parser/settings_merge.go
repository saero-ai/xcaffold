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
// mergeScalarFields merges scalar string and int fields from child into base settings.
func mergeScalarFields(out *ast.SettingsConfig, base, child ast.SettingsConfig, conflict func(string) error) error {
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
				return conflict(f.name)
			}
			*f.dst = f.childVal
		}
	}

	// --- *int pointer ---
	if child.CleanupPeriodDays != nil {
		if base.CleanupPeriodDays != nil && *base.CleanupPeriodDays != *child.CleanupPeriodDays {
			return conflict("cleanupPeriodDays")
		}
		out.CleanupPeriodDays = child.CleanupPeriodDays
	}

	return nil
}

// mergeBooleanFields merges boolean pointer fields from child into base settings.
func mergeBooleanFields(out *ast.SettingsConfig, base, child ast.SettingsConfig, conflict func(string) error) error {
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
				return conflict(f.name)
			}
			*f.dst = f.childVal
		}
	}
	return nil
}

// mergeAnyFields merges untyped any fields from child into base settings.
func mergeAnyFields(out *ast.SettingsConfig, base, child ast.SettingsConfig, conflict func(string) error) error {
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
				return conflict(f.name)
			}
			*f.dst = f.childVal
		}
	}
	return nil
}

// mergeStructPointerFields merges struct pointer fields from child into base settings.
func mergeStructPointerFields(out *ast.SettingsConfig, base, child ast.SettingsConfig, conflict func(string) error) error {
	if child.Permissions != nil {
		if base.Permissions != nil {
			return conflict("permissions")
		}
		out.Permissions = child.Permissions
	}
	if child.Sandbox != nil {
		if base.Sandbox != nil {
			return conflict("sandbox")
		}
		out.Sandbox = child.Sandbox
	}
	if child.StatusLine != nil {
		if base.StatusLine != nil {
			return conflict("statusLine")
		}
		out.StatusLine = child.StatusLine
	}
	return nil
}

// mergeMapOpts groups parameters for merging map fields.
type mergeMapOpts struct {
	Field     string
	BaseFile  string
	ChildFile string
}

// mergeSettingsFieldsContext groups settings merge parameters.
type mergeSettingsFieldsContext struct {
	Out       *ast.SettingsConfig
	Base      ast.SettingsConfig
	Child     ast.SettingsConfig
	BaseFile  string
	ChildFile string
}

// mergeMapAndSliceFields merges map and slice fields from child into base settings.
func mergeMapAndSliceFields(ctx mergeSettingsFieldsContext) error {
	out, base, child, bf, cf := ctx.Out, ctx.Base, ctx.Child, ctx.BaseFile, ctx.ChildFile
	opts := mergeMapOpts{
		BaseFile:  bf,
		ChildFile: cf,
	}
	var err error

	// --- map[string]string (Env) ---
	opts.Field = "env"
	merged, err := mergeStringMapStrict(base.Env, child.Env, opts)
	if err != nil {
		return err
	}
	out.Env = merged

	// --- map[string]bool (EnabledPlugins) ---
	opts.Field = "enabledPlugins"
	mergedPlugins, err := mergeBoolMapStrict(base.EnabledPlugins, child.EnabledPlugins, opts)
	if err != nil {
		return err
	}
	out.EnabledPlugins = mergedPlugins

	// --- map[string]MCPConfig (MCPServers) ---
	opts.Field = "mcpServers"
	mergedMCP, err := mergeMCPMapStrict(base.MCPServers, child.MCPServers, opts)
	if err != nil {
		return err
	}
	out.MCPServers = mergedMCP

	// --- []string (AvailableModels, MdExcludes) ---
	out.AvailableModels = appendUnique(base.AvailableModels, child.AvailableModels)
	out.MdExcludes = appendUnique(base.MdExcludes, child.MdExcludes)

	// --- HookConfig (additive) ---
	out.Hooks = mergeHooksAdditive(base.Hooks, child.Hooks)

	return nil
}

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

	if err := mergeScalarFields(&out, base, child, conflict); err != nil {
		return ast.SettingsConfig{}, err
	}

	if err := mergeBooleanFields(&out, base, child, conflict); err != nil {
		return ast.SettingsConfig{}, err
	}

	if err := mergeAnyFields(&out, base, child, conflict); err != nil {
		return ast.SettingsConfig{}, err
	}

	if err := mergeStructPointerFields(&out, base, child, conflict); err != nil {
		return ast.SettingsConfig{}, err
	}

	if err := mergeMapAndSliceFields(mergeSettingsFieldsContext{
		Out:       &out,
		Base:      base,
		Child:     child,
		BaseFile:  bf,
		ChildFile: cf,
	}); err != nil {
		return ast.SettingsConfig{}, err
	}

	return out, nil
}

// overrideStringFields applies string field overrides from child to out.
func overrideStringFields(out *ast.SettingsConfig, base, child ast.SettingsConfig) {
	stringFields := []struct {
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
	for _, f := range stringFields {
		if f.childVal != "" {
			*f.dst = f.childVal
		}
	}
}

// overrideBoolPointerFields applies boolean pointer field overrides from child to out.
func overrideBoolPointerFields(out *ast.SettingsConfig, child ast.SettingsConfig) {
	boolFields := []struct {
		childVal *bool
		dst      **bool
	}{
		{child.IncludeGitInstructions, &out.IncludeGitInstructions},
		{child.SkipDangerousModePermissionPrompt, &out.SkipDangerousModePermissionPrompt},
		{child.AutoMemoryEnabled, &out.AutoMemoryEnabled},
		{child.DisableAllHooks, &out.DisableAllHooks},
		{child.Attribution, &out.Attribution},
		{child.RespectGitignore, &out.RespectGitignore},
		{child.DisableSkillShellExecution, &out.DisableSkillShellExecution},
		{child.AlwaysThinkingEnabled, &out.AlwaysThinkingEnabled},
	}
	for _, f := range boolFields {
		if f.childVal != nil {
			*f.dst = f.childVal
		}
	}
}

// overrideIntPointerFields applies int pointer field overrides from child to out.
func overrideIntPointerFields(out *ast.SettingsConfig, child ast.SettingsConfig) {
	if child.CleanupPeriodDays != nil {
		out.CleanupPeriodDays = child.CleanupPeriodDays
	}
}

// overrideAnyFields applies untyped any field overrides from child to out.
func overrideAnyFields(out *ast.SettingsConfig, child ast.SettingsConfig) {
	if child.Agent != nil {
		out.Agent = child.Agent
	}
	if child.Worktree != nil {
		out.Worktree = child.Worktree
	}
	if child.AutoMode != nil {
		out.AutoMode = child.AutoMode
	}
}

// overrideStructPointerFields applies struct pointer field overrides from child to out.
func overrideStructPointerFields(out *ast.SettingsConfig, child ast.SettingsConfig) {
	if child.Permissions != nil {
		out.Permissions = child.Permissions
	}
	if child.Sandbox != nil {
		out.Sandbox = child.Sandbox
	}
	if child.StatusLine != nil {
		out.StatusLine = child.StatusLine
	}
}

// overrideMapAndSliceFields applies map and slice field overrides from child to out.
func overrideMapAndSliceFields(out *ast.SettingsConfig, base, child ast.SettingsConfig) {
	out.Env = mergeStringMapOverride(base.Env, child.Env)
	out.EnabledPlugins = mergeBoolMapOverride(base.EnabledPlugins, child.EnabledPlugins)
	out.MCPServers = mergeMCPMapOverride(base.MCPServers, child.MCPServers)
	out.AvailableModels = appendUnique(base.AvailableModels, child.AvailableModels)
	out.MdExcludes = appendUnique(base.MdExcludes, child.MdExcludes)
	out.Hooks = mergeHooksAdditive(base.Hooks, child.Hooks)
}

// mergeSettingsOverride merges two SettingsConfig values for extends resolution
// (global -> project).  The child always wins on conflict.
func mergeSettingsOverride(base, child ast.SettingsConfig) ast.SettingsConfig {
	out := base

	overrideStringFields(&out, base, child)
	overrideBoolPointerFields(&out, child)
	overrideIntPointerFields(&out, child)
	overrideAnyFields(&out, child)
	overrideStructPointerFields(&out, child)
	overrideMapAndSliceFields(&out, base, child)

	return out
}

// ---------------------------------------------------------------------------
// Map merge helpers
// ---------------------------------------------------------------------------

func mergeStringMapStrict(base, child map[string]string, opts mergeMapOpts) (map[string]string, error) {
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
			return nil, fmt.Errorf("settings conflict for %q key %q between %s and %s", opts.Field, k, opts.BaseFile, opts.ChildFile)
		}
		merged[k] = v
	}
	return merged, nil
}

func mergeBoolMapStrict(base, child map[string]bool, opts mergeMapOpts) (map[string]bool, error) {
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
			return nil, fmt.Errorf("settings conflict for %q key %q between %s and %s", opts.Field, k, opts.BaseFile, opts.ChildFile)
		}
		merged[k] = v
	}
	return merged, nil
}

func mergeMCPMapStrict(base, child map[string]ast.MCPConfig, opts mergeMapOpts) (map[string]ast.MCPConfig, error) {
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
			return nil, fmt.Errorf("settings conflict for %q key %q between %s and %s", opts.Field, k, opts.BaseFile, opts.ChildFile)
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
