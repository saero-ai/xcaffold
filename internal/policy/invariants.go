package policy

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/output"
)

// RunInvariants executes all security invariant checks against the compiled
// output and source config. It returns every violation found across all checks.
func RunInvariants(config *ast.XcaffoldConfig, compiled *output.Output) []error {
	var errs []error
	errs = append(errs, validateOutputPaths(compiled)...)
	errs = append(errs, validateAbsolutePaths(compiled)...)
	errs = append(errs, validateSettingsSchema(compiled)...)
	errs = append(errs, validateHookURLs(config)...)
	return errs
}

// validateOutputPaths rejects any output path containing a directory traversal
// sequence (".."). Traversal paths could escape the intended output directory.
func validateOutputPaths(compiled *output.Output) []error {
	var errs []error
	for path := range compiled.Files {
		if strings.Contains(path, "..") {
			errs = append(errs, fmt.Errorf(
				"invariant: output path %q contains directory traversal sequence", path,
			))
		}
	}
	return errs
}

// validateAbsolutePaths rejects any output path that is absolute. All rendered
// paths must be relative so they are resolved against the chosen base directory.
func validateAbsolutePaths(compiled *output.Output) []error {
	var errs []error
	for path := range compiled.Files {
		if strings.HasPrefix(path, "/") {
			errs = append(errs, fmt.Errorf(
				"invariant: output path %q is absolute; paths must be relative", path,
			))
		}
	}
	return errs
}

// validateSettingsSchema checks that no rendered settings file contains a null
// permissions field. A null permissions block is a common misconfiguration that
// silently disables provider-level sandboxing.
func validateSettingsSchema(compiled *output.Output) []error {
	var errs []error
	for path, content := range compiled.Files {
		if !strings.HasSuffix(path, "settings.json") && !strings.HasSuffix(path, "settings.local.json") {
			continue
		}
		if strings.Contains(content, `"permissions": null`) {
			errs = append(errs, fmt.Errorf(
				"invariant: %q contains null permissions field", path,
			))
		}
	}
	return errs
}

// hookLocation pairs a HookHandler with a human-readable description of where
// it was found, used to produce actionable error messages.
type hookLocation struct {
	handler     ast.HookHandler
	description string
}

// collectHookHandlers gathers every HookHandler from all three hook locations
// in the config: top-level Hooks, per-agent Hooks, and per-settings Hooks.
func collectHookHandlers(config *ast.XcaffoldConfig) []hookLocation {
	var locs []hookLocation

	for name, namedHook := range config.Hooks {
		for event, groups := range namedHook.Events {
			for _, group := range groups {
				for _, h := range group.Hooks {
					locs = append(locs, hookLocation{
						handler:     h,
						description: fmt.Sprintf("hooks[%s] event %s", name, event),
					})
				}
			}
		}
	}

	for agentName, agent := range config.Agents {
		for event, groups := range agent.Hooks {
			for _, group := range groups {
				for _, h := range group.Hooks {
					locs = append(locs, hookLocation{
						handler:     h,
						description: fmt.Sprintf("agent[%s] hook event %s", agentName, event),
					})
				}
			}
		}
	}

	for settingsName, settings := range config.Settings {
		for event, groups := range settings.Hooks {
			for _, group := range groups {
				for _, h := range group.Hooks {
					locs = append(locs, hookLocation{
						handler:     h,
						description: fmt.Sprintf("settings[%s] hook event %s", settingsName, event),
					})
				}
			}
		}
	}

	return locs
}

// validateHookURLs ensures that every hook handler with a URL field uses the
// HTTPS scheme. HTTP URLs risk credential interception and MITM attacks.
func validateHookURLs(config *ast.XcaffoldConfig) []error {
	var errs []error
	for _, loc := range collectHookHandlers(config) {
		u := loc.handler.URL
		if u == "" {
			continue
		}
		parsed, err := url.Parse(u)
		if err != nil {
			errs = append(errs, fmt.Errorf(
				"invariant: %s has unparseable URL %q: %w", loc.description, u, err,
			))
			continue
		}
		if parsed.Scheme != "https" {
			errs = append(errs, fmt.Errorf(
				"invariant: hook URL %q must use HTTPS scheme (got %q)", u, parsed.Scheme,
			))
		}
	}
	return errs
}
