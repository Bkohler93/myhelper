package scanner

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bkohler93/myhelper/internal/config"
	"github.com/bkohler93/myhelper/internal/history"
)

// SchemaVersion is the canonical version string written to all artifact files.
const SchemaVersion = "1.0"

// ErrStaleFlatIndex is returned (or logged) when callers detect an old flat
// index.json in place of the new artifact files.
var ErrStaleFlatIndex = errors.New("index.json is a stale flat index; use artifact files instead")

// ProjectArtifact is the structure written to .myhelper/project.json.
type ProjectArtifact struct {
	SchemaVersion string `json:"schemaVersion"`
	ModulePath    string `json:"modulePath"`
	GoVersion     string `json:"goVersion"`
	FileCount     int    `json:"fileCount"`
	SymbolCount   int    `json:"symbolCount"`
	Summary       string `json:"summary"`
}

// PackageEntry represents a single package in packages.json.
type PackageEntry struct {
	ImportPath     string   `json:"importPath"`
	Files          []string `json:"files"`
	Responsibility string   `json:"responsibility"`
}

// PackagesArtifact is the structure written to .myhelper/packages.json.
type PackagesArtifact struct {
	SchemaVersion string         `json:"schemaVersion"`
	Packages      []PackageEntry `json:"packages"`
}

// FileArtifactEntry represents a single file in files.json.
type FileArtifactEntry struct {
	Path          string   `json:"path"`
	Package       string   `json:"package"`
	ExportedNames []string `json:"exportedNames"`
	Imports       []string `json:"imports"`
}

// FilesArtifact is the structure written to .myhelper/files.json.
type FilesArtifact struct {
	SchemaVersion string              `json:"schemaVersion"`
	Files         []FileArtifactEntry `json:"files"`
}

// SymbolsArtifact is the structure written to .myhelper/symbols.json.
type SymbolsArtifact struct {
	SchemaVersion string   `json:"schemaVersion"`
	Symbols       []Symbol `json:"symbols"`
}

// BuildArtifacts walks root, extracts symbols from every Go file, builds four
// structured artifact files under .myhelper/, and returns the first error
// encountered (if any).
//
// The four files produced are:
//   - project.json  — module-level summary (ProjectArtifact)
//   - packages.json — per-package import paths, files, and responsibility (PackagesArtifact)
//   - files.json    — per-file exported names and imports (FilesArtifact)
//   - symbols.json  — every exported symbol with full rich profile (SymbolsArtifact)
func BuildArtifacts(root string, cfg config.Config, chatFn ChatFn) error {
	// Ensure .myhelper/ directory exists.
	myhelperDir := filepath.Join(root, ".myhelper")
	if err := os.MkdirAll(myhelperDir, 0755); err != nil {
		return fmt.Errorf("BuildArtifacts: mkdir .myhelper: %w", err)
	}

	// Step 1: Walk root for relative .go file paths.
	relPaths, err := Walk(root)
	if err != nil {
		return fmt.Errorf("BuildArtifacts: walk: %w", err)
	}

	// Step 3: Read project meta (module name, go version).
	meta, err := ReadMeta(root)
	if err != nil {
		return fmt.Errorf("BuildArtifacts: ReadMeta: %w", err)
	}

	// Step 2: Extract symbols from each file, build per-file and per-package structures.
	var allSymbols []Symbol
	var fileEntries []FileArtifactEntry

	// pkgFiles maps import path -> []relPath
	pkgFiles := make(map[string][]string)
	// pkgShortName maps import path -> short package name (for summaries lookup)
	pkgShortName := make(map[string]string)

	for _, relPath := range relPaths {
		absPath := filepath.Join(root, relPath)
		syms, err := ExtractSymbolsFull(absPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "BuildArtifacts: skip %s: %v\n", relPath, err)
			continue
		}

		// Set FilePath on every symbol (relPath is relative to root).
		for i := range syms {
			syms[i].FilePath = relPath
		}

		// Derive import path from relPath.
		dir := filepath.ToSlash(filepath.Dir(relPath))
		var importPath string
		if dir == "." {
			importPath = meta.ModuleName
		} else {
			importPath = meta.ModuleName + "/" + dir
		}

		// Accumulate symbols.
		allSymbols = append(allSymbols, syms...)

		// Build FileArtifactEntry.
		exportedNames := make([]string, 0, len(syms))
		for _, s := range syms {
			exportedNames = append(exportedNames, s.Name)
		}

		var fileImports []string
		if len(syms) > 0 {
			fileImports = syms[0].Imports
		}
		if fileImports == nil {
			fileImports = []string{}
		}

		// Determine short package name: use first symbol's StableID prefix,
		// or fall back to last segment of import path.
		shortPkg := ""
		if len(syms) > 0 {
			stableID := syms[0].StableID
			if dot := strings.Index(stableID, "."); dot >= 0 {
				shortPkg = stableID[:dot]
			}
		}
		if shortPkg == "" {
			parts := strings.Split(importPath, "/")
			shortPkg = parts[len(parts)-1]
		}

		fileEntries = append(fileEntries, FileArtifactEntry{
			Path:          relPath,
			Package:       shortPkg,
			ExportedNames: exportedNames,
			Imports:       fileImports,
		})

		// Track package grouping.
		pkgFiles[importPath] = append(pkgFiles[importPath], relPath)
		pkgShortName[importPath] = shortPkg
	}

	// Ensure non-nil slices for JSON marshalling.
	if allSymbols == nil {
		allSymbols = []Symbol{}
	}
	if fileEntries == nil {
		fileEntries = []FileArtifactEntry{}
	}

	// Step 4: Build PackageEntry list, reading per-package responsibility summaries.
	summariesDir := filepath.Join(root, ".myhelper", "summaries")
	pkgEntries := make([]PackageEntry, 0, len(pkgFiles))
	var pkgSummaryParts []string

	for importPath, files := range pkgFiles {
		shortName := pkgShortName[importPath]
		responsibility := ""
		summaryPath := filepath.Join(summariesDir, shortName+".md")
		if data, err := os.ReadFile(summaryPath); err == nil {
			responsibility = string(data)
			pkgSummaryParts = append(pkgSummaryParts, fmt.Sprintf("Package %s:\n%s", importPath, responsibility))
		}
		pkgEntries = append(pkgEntries, PackageEntry{
			ImportPath:     importPath,
			Files:          files,
			Responsibility: responsibility,
		})
	}

	// Step 5: Generate project-level summary via one chatFn call.
	var projectSummary string
	if len(pkgSummaryParts) > 0 {
		var sb strings.Builder
		sb.WriteString("Below are per-package summaries for a Go project. Write a concise one-paragraph project overview describing what the project does, its key packages, and how they relate. Under 200 words. Plain prose, no markdown headers.\n\n")
		sb.WriteString(strings.Join(pkgSummaryParts, "\n\n"))
		resp, err := chatFn(cfg, []history.Message{{Role: "user", Content: sb.String()}})
		if err != nil {
			fmt.Fprintf(os.Stderr, "BuildArtifacts: chatFn summary: %v\n", err)
			projectSummary = ""
		} else {
			projectSummary = resp
		}
	} else {
		// No summaries available — try a direct chatFn call with file list as context.
		resp, err := chatFn(cfg, []history.Message{{Role: "user", Content: "Write a concise one-paragraph project overview. Under 200 words. Plain prose, no markdown headers."}})
		if err != nil {
			fmt.Fprintf(os.Stderr, "BuildArtifacts: chatFn summary: %v\n", err)
			projectSummary = ""
		} else {
			projectSummary = resp
		}
	}

	// Step 6: Write the four artifact files.
	projectArtifact := ProjectArtifact{
		SchemaVersion: SchemaVersion,
		ModulePath:    meta.ModuleName,
		GoVersion:     meta.GoVersion,
		FileCount:     len(fileEntries),
		SymbolCount:   len(allSymbols),
		Summary:       projectSummary,
	}
	if err := writeJSON(myhelperDir, "project.json", projectArtifact); err != nil {
		return err
	}

	packagesArtifact := PackagesArtifact{
		SchemaVersion: SchemaVersion,
		Packages:      pkgEntries,
	}
	if err := writeJSON(myhelperDir, "packages.json", packagesArtifact); err != nil {
		return err
	}

	filesArtifact := FilesArtifact{
		SchemaVersion: SchemaVersion,
		Files:         fileEntries,
	}
	if err := writeJSON(myhelperDir, "files.json", filesArtifact); err != nil {
		return err
	}

	symbolsArtifact := SymbolsArtifact{
		SchemaVersion: SchemaVersion,
		Symbols:       allSymbols,
	}
	if err := writeJSON(myhelperDir, "symbols.json", symbolsArtifact); err != nil {
		return err
	}

	return nil
}

// writeJSON marshals v as indented JSON and writes it to dir/filename.
func writeJSON(dir, filename string, v interface{}) error {
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("BuildArtifacts: marshal %s: %w", filename, err)
	}
	if err := os.WriteFile(filepath.Join(dir, filename), out, 0644); err != nil {
		return fmt.Errorf("BuildArtifacts: write %s: %w", filename, err)
	}
	return nil
}
