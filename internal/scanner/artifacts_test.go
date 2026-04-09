package scanner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/bkohler93/myhelper/internal/config"
	"github.com/bkohler93/myhelper/internal/history"
)

// stubChatFn returns a fixed string for all calls — used to satisfy chatFn parameter.
func stubChatFn(_ config.Config, _ []history.Message) (string, error) {
	return "stub summary", nil
}

// setupTestProject creates a minimal two-file Go project in a temp directory.
// Returns the temp dir path; caller is responsible for os.RemoveAll.
func setupTestProject(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()

	// go.mod
	goMod := `module github.com/example/testpkg

go 1.24.2
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	// main.go
	mainGo := `package main

// ExampleFunc does something.
func ExampleFunc(x int) string { return "" }
`
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainGo), 0644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	// sub/sub.go
	if err := os.MkdirAll(filepath.Join(tmpDir, "sub"), 0755); err != nil {
		t.Fatalf("mkdir sub: %v", err)
	}
	subGo := `package sub

type ExampleStruct struct { Name string }
`
	if err := os.WriteFile(filepath.Join(tmpDir, "sub", "sub.go"), []byte(subGo), 0644); err != nil {
		t.Fatalf("write sub/sub.go: %v", err)
	}

	return tmpDir
}

func TestBuildArtifacts(t *testing.T) {
	tmpDir := setupTestProject(t)
	cfg := config.Config{}

	if err := BuildArtifacts(tmpDir, cfg, stubChatFn); err != nil {
		t.Fatalf("BuildArtifacts returned error: %v", err)
	}

	t.Run("creates_four_files", func(t *testing.T) {
		files := []string{
			filepath.Join(tmpDir, ".myhelper", "project.json"),
			filepath.Join(tmpDir, ".myhelper", "packages.json"),
			filepath.Join(tmpDir, ".myhelper", "files.json"),
			filepath.Join(tmpDir, ".myhelper", "symbols.json"),
		}
		for _, f := range files {
			if _, err := os.Stat(f); os.IsNotExist(err) {
				t.Errorf("expected file to exist: %s", f)
			}
		}
	})

	t.Run("project_json", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join(tmpDir, ".myhelper", "project.json"))
		if err != nil {
			t.Fatalf("read project.json: %v", err)
		}
		var artifact ProjectArtifact
		if err := json.Unmarshal(data, &artifact); err != nil {
			t.Fatalf("unmarshal project.json: %v", err)
		}
		if artifact.SchemaVersion != "1.0" {
			t.Errorf("SchemaVersion = %q, want %q", artifact.SchemaVersion, "1.0")
		}
		if artifact.ModulePath == "" {
			t.Error("ModulePath must not be empty")
		}
		if artifact.GoVersion == "" {
			t.Error("GoVersion must not be empty (must parse from go.mod)")
		}
		if artifact.FileCount <= 0 {
			t.Errorf("FileCount = %d, want > 0", artifact.FileCount)
		}
		if artifact.SymbolCount <= 0 {
			t.Errorf("SymbolCount = %d, want > 0", artifact.SymbolCount)
		}
		// No package summaries exist in the test project (no .myhelper/*.md files),
		// so BuildArtifacts skips the LLM call and leaves Summary empty.
		if artifact.Summary != "" {
			t.Errorf("Summary = %q, want %q (empty — no package summaries to aggregate)", artifact.Summary, "")
		}
	})

	t.Run("packages_json", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join(tmpDir, ".myhelper", "packages.json"))
		if err != nil {
			t.Fatalf("read packages.json: %v", err)
		}
		var artifact PackagesArtifact
		if err := json.Unmarshal(data, &artifact); err != nil {
			t.Fatalf("unmarshal packages.json: %v", err)
		}
		if artifact.SchemaVersion != "1.0" {
			t.Errorf("SchemaVersion = %q, want %q", artifact.SchemaVersion, "1.0")
		}
		if len(artifact.Packages) == 0 {
			t.Error("Packages must not be empty")
		}
		for _, pkg := range artifact.Packages {
			if pkg.ImportPath == "" {
				t.Error("PackageEntry.ImportPath must not be empty")
			}
			// Full module-qualified path must contain "/"
			hasSlash := false
			for _, c := range pkg.ImportPath {
				if c == '/' {
					hasSlash = true
					break
				}
			}
			if !hasSlash {
				t.Errorf("PackageEntry.ImportPath %q must be module-qualified (contain /)", pkg.ImportPath)
			}
			if len(pkg.Files) == 0 {
				t.Errorf("PackageEntry %q: Files must not be empty", pkg.ImportPath)
			}
		}
	})

	t.Run("files_json", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join(tmpDir, ".myhelper", "files.json"))
		if err != nil {
			t.Fatalf("read files.json: %v", err)
		}
		var artifact FilesArtifact
		if err := json.Unmarshal(data, &artifact); err != nil {
			t.Fatalf("unmarshal files.json: %v", err)
		}
		if artifact.SchemaVersion != "1.0" {
			t.Errorf("SchemaVersion = %q, want %q", artifact.SchemaVersion, "1.0")
		}
		if len(artifact.Files) == 0 {
			t.Error("Files must not be empty")
		}
		atLeastOneExported := false
		for _, fe := range artifact.Files {
			if fe.Path == "" {
				t.Error("FileArtifactEntry.Path must not be empty")
			}
			if fe.Package == "" {
				t.Error("FileArtifactEntry.Package must not be empty")
			}
			if len(fe.ExportedNames) > 0 {
				atLeastOneExported = true
			}
		}
		if !atLeastOneExported {
			t.Error("at least one file must have ExportedNames")
		}
	})

	t.Run("symbols_json", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join(tmpDir, ".myhelper", "symbols.json"))
		if err != nil {
			t.Fatalf("read symbols.json: %v", err)
		}
		var artifact SymbolsArtifact
		if err := json.Unmarshal(data, &artifact); err != nil {
			t.Fatalf("unmarshal symbols.json: %v", err)
		}
		if artifact.SchemaVersion != "1.0" {
			t.Errorf("SchemaVersion = %q, want %q", artifact.SchemaVersion, "1.0")
		}
		if len(artifact.Symbols) == 0 {
			t.Error("Symbols must not be empty")
		}
		validKinds := map[string]bool{"func": true, "method": true, "struct": true, "interface": true}
		for _, sym := range artifact.Symbols {
			if sym.FilePath == "" {
				t.Errorf("Symbol %q: FilePath must not be empty", sym.Name)
			}
			if !validKinds[sym.Kind] {
				t.Errorf("Symbol %q: Kind = %q, must be one of func/method/struct/interface", sym.Name, sym.Kind)
			}
		}
	})

	t.Run("schema_version", func(t *testing.T) {
		artifactFiles := []string{
			filepath.Join(tmpDir, ".myhelper", "project.json"),
			filepath.Join(tmpDir, ".myhelper", "packages.json"),
			filepath.Join(tmpDir, ".myhelper", "files.json"),
			filepath.Join(tmpDir, ".myhelper", "symbols.json"),
		}
		for _, f := range artifactFiles {
			data, err := os.ReadFile(f)
			if err != nil {
				t.Fatalf("read %s: %v", f, err)
			}
			var raw map[string]interface{}
			if err := json.Unmarshal(data, &raw); err != nil {
				t.Fatalf("unmarshal %s: %v", f, err)
			}
			sv, ok := raw["schemaVersion"]
			if !ok {
				t.Errorf("%s: missing schemaVersion key", f)
				continue
			}
			if sv != "1.0" {
				t.Errorf("%s: schemaVersion = %v, want \"1.0\"", f, sv)
			}
		}
	})
}
