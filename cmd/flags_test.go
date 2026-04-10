package cmd

import (
	"bytes"
	"os"
	"testing"
)

// TestApplyFlagOverrides_QueryCommands verifies that all four query command files
// call ApplyFlagOverrides(&cfg) — the fix for CMD-03 tech debt.
// This is a source-scan regression guard so the call cannot be silently removed.
func TestApplyFlagOverrides_QueryCommands(t *testing.T) {
	commands := []string{
		"plan.go",
		"lookup.go",
		"starter.go",
		"pattern.go",
	}
	for _, name := range commands {
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("could not read %s: %v", name, err)
		}
		if !bytes.Contains(data, []byte("ApplyFlagOverrides(&cfg)")) {
			t.Errorf("%s: missing ApplyFlagOverrides(&cfg) call (CMD-03: must be present before retrieval.BuildContext)", name)
		}
	}
}

// TestNoContextFlag_Registered verifies that --no-context is registered as a
// persistent flag on rootCmd (CMD-01).
func TestNoContextFlag_Registered(t *testing.T) {
	f := rootCmd.PersistentFlags().Lookup("no-context")
	if f == nil {
		t.Fatal("--no-context persistent flag not registered on rootCmd (CMD-01)")
	}
	if f.DefValue != "false" {
		t.Errorf("--no-context default value = %q, want \"false\"", f.DefValue)
	}
}
