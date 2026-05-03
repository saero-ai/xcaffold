package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
)

func validateHooksDirectory(cfg *ast.XcaffoldConfig, projectDir string) []string {
	if len(cfg.Hooks) == 0 {
		return nil
	}

	var errors []string

	for hookID, hook := range cfg.Hooks {
		hookBase := filepath.Join(projectDir, "xcf", "hooks", hookID)

		for _, artifact := range hook.Artifacts {
			artifactPath := filepath.Join(hookBase, artifact)
			if _, err := os.Stat(artifactPath); os.IsNotExist(err) {
				errors = append(errors, fmt.Sprintf(
					"hook %q declares artifact %q but directory does not exist: %s",
					hookID, artifact, artifactPath))
			}
		}

		errors = append(errors, validateHookCommands(hookID, hook.Events, projectDir)...)
	}

	return errors
}

func validateHookCommands(hookID string, events ast.HookConfig, projectDir string) []string {
	var errors []string
	prefix := "xcf/hooks/"

	for _, groups := range events {
		for _, group := range groups {
			for _, handler := range group.Hooks {
				if handler.Command == "" {
					continue
				}
				for _, script := range extractScriptPaths(handler.Command, prefix) {
					fullPath := filepath.Join(projectDir, script)
					if _, err := os.Stat(fullPath); os.IsNotExist(err) {
						errors = append(errors, fmt.Sprintf(
							"hook %q references script that does not exist: %s",
							hookID, script))
					}
				}
			}
		}
	}

	return errors
}

func extractScriptPaths(command, prefix string) []string {
	var paths []string
	for _, part := range strings.Fields(command) {
		if strings.HasPrefix(part, prefix) {
			paths = append(paths, part)
		}
	}
	return paths
}
