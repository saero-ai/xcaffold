package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	tmpHome, err := os.MkdirTemp("", "xcaffold-test-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpHome)
	os.Setenv("HOME", tmpHome)
	os.Exit(m.Run())
}

// TestGlobalFlag_VisibleInHelp verifies that --global appears in the root
// command help output now that it is no longer hidden.
func TestGlobalFlag_VisibleInHelp(t *testing.T) {
	// rootHelpFunc writes to os.Stdout via fmt.Printf, so capture stdout.
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	rootHelpFunc(rootCmd, nil)

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	assert.Contains(t, output, "--global", "help output must show the --global flag")
}

// TestResolveGlobalConfig_AllCommands verifies that resolveGlobalConfig
// returns nil (no error) for every registered subcommand. Previously it
// blocked all commands except init/import with a "not yet available" error.
func TestResolveGlobalConfig_AllCommands(t *testing.T) {
	// Point HOME at a temp dir so path resolution works.
	home := t.TempDir()
	t.Setenv("HOME", home)

	commands := []string{"init", "import", "apply", "validate", "status", "list", "graph", "export", "test"}
	for _, name := range commands {
		t.Run(name, func(t *testing.T) {
			cmd := &cobra.Command{Use: name}
			err := resolveGlobalConfig(cmd)
			require.NoError(t, err, "resolveGlobalConfig must not block %q", name)

			// Verify the path was set to the expected default location.
			expected := filepath.Join(home, ".xcaffold", "xcaf", "global.xcaf")
			assert.Equal(t, expected, globalXcafPath,
				"globalXcafPath should resolve to ~/.xcaffold/xcaf/global.xcaf")
		})
	}
}

// TestResolveGlobalConfig_CustomConfigFlag verifies that when both --global
// and --config are set, the custom config path is used instead of the default.
func TestResolveGlobalConfig_CustomConfigFlag(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	customPath := filepath.Join(home, "custom", "my-global.xcaf")
	require.NoError(t, os.MkdirAll(filepath.Dir(customPath), 0o755))

	origConfig := configFlag
	origGlobal := globalFlag
	configFlag = customPath
	globalFlag = true
	defer func() {
		configFlag = origConfig
		globalFlag = origGlobal
	}()

	cmd := &cobra.Command{Use: "apply"}
	err := resolveGlobalConfig(cmd)
	require.NoError(t, err)
	assert.Equal(t, customPath, globalXcafPath,
		"globalXcafPath should use the --config value when both flags are set")
}

// TestValidateGlobal_NoLongerBlocked verifies that running validate with
// globalFlag=true no longer returns the old "not yet available" guard.
// The command may fail for other reasons (e.g., missing global.xcaf),
// but the specific gate error must be gone.
func TestValidateGlobal_NoLongerBlocked(t *testing.T) {
	globalFlag = true
	defer func() { globalFlag = false }()

	err := runValidate(validateCmd, []string{})
	if err != nil {
		assert.NotContains(t, err.Error(), "global scope is not yet available",
			"validate must not return the old global-scope guard error")
	}
}

// TestResolveGlobalConfig_RespectsXCAFFOLD_HOME verifies that the XCAFFOLD_HOME
// environment variable is respected when set, allowing test isolation and
// custom global homes.
func TestResolveGlobalConfig_RespectsXCAFFOLD_HOME(t *testing.T) {
	customHome := t.TempDir()
	t.Setenv("XCAFFOLD_HOME", customHome)

	cmd := &cobra.Command{Use: "apply"}
	err := resolveGlobalConfig(cmd)
	require.NoError(t, err)

	assert.Equal(t, customHome, globalXcafHome,
		"globalXcafHome should equal XCAFFOLD_HOME when set")

	expectedPath := filepath.Join(customHome, "xcaf", "global.xcaf")
	assert.Equal(t, expectedPath, globalXcafPath,
		"globalXcafPath should be computed relative to XCAFFOLD_HOME")
}

// TestResolveGlobalConfig_DefaultsToUserHome verifies that when XCAFFOLD_HOME
// is not set, the default ~/.xcaffold is used.
func TestResolveGlobalConfig_DefaultsToUserHome(t *testing.T) {
	t.Setenv("XCAFFOLD_HOME", "") // Ensure XCAFFOLD_HOME is not set

	home := t.TempDir()
	t.Setenv("HOME", home)

	cmd := &cobra.Command{Use: "apply"}
	err := resolveGlobalConfig(cmd)
	require.NoError(t, err)

	expectedHome := filepath.Join(home, ".xcaffold")
	assert.Equal(t, expectedHome, globalXcafHome,
		"globalXcafHome should default to ~/.xcaffold when XCAFFOLD_HOME is unset")
}
