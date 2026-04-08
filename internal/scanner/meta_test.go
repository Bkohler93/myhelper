package scanner_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bkohler93/myhelper/internal/scanner"
)

func TestReadMeta(t *testing.T) {
	t.Run("module name extracted from go.mod", func(t *testing.T) {
		dir := t.TempDir()
		gomod := "module github.com/foo/bar\n\ngo 1.21\n"
		if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(gomod), 0644); err != nil {
			t.Fatal(err)
		}

		meta, err := scanner.ReadMeta(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if meta.ModuleName != "github.com/foo/bar" {
			t.Errorf("got ModuleName %q, want %q", meta.ModuleName, "github.com/foo/bar")
		}
	})

	t.Run("direct deps extracted, indirect excluded", func(t *testing.T) {
		dir := t.TempDir()
		gomod := `module github.com/foo/bar

go 1.21

require (
	github.com/spf13/cobra v1.10.2
	github.com/some/indirect v0.1.0 // indirect
)
`
		if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(gomod), 0644); err != nil {
			t.Fatal(err)
		}

		meta, err := scanner.ReadMeta(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(meta.DirectDeps) != 1 {
			t.Fatalf("got %d direct deps, want 1: %v", len(meta.DirectDeps), meta.DirectDeps)
		}
		if meta.DirectDeps[0] != "github.com/spf13/cobra v1.10.2" {
			t.Errorf("got DirectDeps[0] %q, want %q", meta.DirectDeps[0], "github.com/spf13/cobra v1.10.2")
		}
	})

	t.Run("no go.mod returns empty module name and nil deps, no error", func(t *testing.T) {
		dir := t.TempDir()

		meta, err := scanner.ReadMeta(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if meta.ModuleName != "" {
			t.Errorf("got ModuleName %q, want empty string", meta.ModuleName)
		}
		if meta.DirectDeps != nil {
			t.Errorf("got DirectDeps %v, want nil", meta.DirectDeps)
		}
	})

	t.Run("README.md content read", func(t *testing.T) {
		dir := t.TempDir()
		content := "# My Project\nDoes stuff."
		if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		meta, err := scanner.ReadMeta(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if meta.ReadmeContent != content {
			t.Errorf("got ReadmeContent %q, want %q", meta.ReadmeContent, content)
		}
	})

	t.Run("no README.md returns empty string, no error", func(t *testing.T) {
		dir := t.TempDir()

		meta, err := scanner.ReadMeta(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if meta.ReadmeContent != "" {
			t.Errorf("got ReadmeContent %q, want empty string", meta.ReadmeContent)
		}
	})

	t.Run("single config file read", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("port: 8080"), 0644); err != nil {
			t.Fatal(err)
		}

		meta, err := scanner.ReadMeta(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(meta.ConfigFiles) != 1 {
			t.Fatalf("got %d config files, want 1", len(meta.ConfigFiles))
		}
		if meta.ConfigFiles[0].Name != "config.yaml" {
			t.Errorf("got Name %q, want %q", meta.ConfigFiles[0].Name, "config.yaml")
		}
		if meta.ConfigFiles[0].Content != "port: 8080" {
			t.Errorf("got Content %q, want %q", meta.ConfigFiles[0].Content, "port: 8080")
		}
	})

	t.Run("multiple config files read", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "app.json"), []byte(`{"key":"val"}`), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "settings.toml"), []byte("[server]\nport=8080"), 0644); err != nil {
			t.Fatal(err)
		}

		meta, err := scanner.ReadMeta(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(meta.ConfigFiles) != 2 {
			t.Fatalf("got %d config files, want 2: %v", len(meta.ConfigFiles), meta.ConfigFiles)
		}
	})

	t.Run(".go file excluded from config files", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{"port":8080}`), 0644); err != nil {
			t.Fatal(err)
		}

		meta, err := scanner.ReadMeta(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(meta.ConfigFiles) != 1 {
			t.Fatalf("got %d config files, want 1: %v", len(meta.ConfigFiles), meta.ConfigFiles)
		}
		if meta.ConfigFiles[0].Name != "config.json" {
			t.Errorf("got Name %q, want %q", meta.ConfigFiles[0].Name, "config.json")
		}
	})

	t.Run("no config files returns nil or empty slice, no error", func(t *testing.T) {
		dir := t.TempDir()

		meta, err := scanner.ReadMeta(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(meta.ConfigFiles) != 0 {
			t.Errorf("got %d config files, want 0: %v", len(meta.ConfigFiles), meta.ConfigFiles)
		}
	})
}
