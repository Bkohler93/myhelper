package scanner

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// ConfigFile holds the filename and raw content of a config file found at project root.
type ConfigFile struct {
	Name    string
	Content string
}

// ProjectMeta holds project-level metadata extracted from go.mod, README, and config files.
type ProjectMeta struct {
	ModuleName    string       `json:"module_name"`
	DirectDeps    []string     `json:"direct_deps"`
	ReadmeContent string       `json:"readme_content"`
	ConfigFiles   []ConfigFile `json:"config_files"`
}

// ReadMeta reads project metadata from the given root directory.
// It extracts the module name and direct dependencies from go.mod,
// reads README.md content, and reads config files (.json, .yaml, .toml).
// Missing files are handled gracefully — no error is returned for absent files.
func ReadMeta(root string) (ProjectMeta, error) {
	var meta ProjectMeta

	// Parse go.mod
	gomodData, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil && !os.IsNotExist(err) {
		return meta, err
	}
	if err == nil {
		inRequire := false
		sc := bufio.NewScanner(strings.NewReader(string(gomodData)))
		for sc.Scan() {
			line := sc.Text()
			trimmed := strings.TrimSpace(line)

			if strings.HasPrefix(trimmed, "module ") {
				meta.ModuleName = strings.TrimSpace(strings.TrimPrefix(trimmed, "module "))
				continue
			}

			if trimmed == "require (" {
				inRequire = true
				continue
			}

			if inRequire {
				if trimmed == ")" {
					inRequire = false
					continue
				}
				if trimmed == "" {
					continue
				}
				if strings.Contains(trimmed, "// indirect") {
					continue
				}
				meta.DirectDeps = append(meta.DirectDeps, trimmed)
				continue
			}

			// Single-line require: "require github.com/foo v1.0.0"
			if strings.HasPrefix(trimmed, "require ") && !strings.Contains(trimmed, "// indirect") {
				dep := strings.TrimSpace(strings.TrimPrefix(trimmed, "require "))
				if dep != "(" {
					meta.DirectDeps = append(meta.DirectDeps, dep)
				}
			}
		}
	}

	// Read README.md
	readmeData, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil && !os.IsNotExist(err) {
		return meta, err
	}
	if err == nil {
		meta.ReadmeContent = string(readmeData)
	}

	// Read config files (.json, .yaml, .toml)
	entries, err := os.ReadDir(root)
	if err != nil {
		return meta, err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext != ".json" && ext != ".yaml" && ext != ".toml" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, entry.Name()))
		if err != nil {
			return meta, err
		}
		meta.ConfigFiles = append(meta.ConfigFiles, ConfigFile{
			Name:    entry.Name(),
			Content: string(data),
		})
	}

	return meta, nil
}
