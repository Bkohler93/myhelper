package planner

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const phasesDir = ".planning/phases"

// FindActivePlan scans .planning/phases/ for the highest-numbered directory
// that does not contain a *-SUMMARY.md file, and returns the path to its
// PLAN.md file. Returns an error if no active phase is found or if the
// phases directory cannot be read.
func FindActivePlan() (string, error) {
	entries, err := os.ReadDir(phasesDir)
	if err != nil {
		return "", fmt.Errorf("find active plan: read phases dir: %w", err)
	}

	// Collect directories with numeric prefix, sorted by prefix descending.
	type phaseDir struct {
		num  int
		name string
	}
	var dirs []phaseDir
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		parts := strings.SplitN(e.Name(), "-", 2)
		n, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		dirs = append(dirs, phaseDir{num: n, name: e.Name()})
	}
	sort.Slice(dirs, func(i, j int) bool { return dirs[i].num > dirs[j].num })

	// Find highest-numbered dir with no *-SUMMARY.md file.
	for _, d := range dirs {
		inner, err := os.ReadDir(filepath.Join(phasesDir, d.name))
		if err != nil {
			continue
		}
		hasSummary := false
		for _, f := range inner {
			if strings.HasSuffix(f.Name(), "-SUMMARY.md") {
				hasSummary = true
				break
			}
		}
		if hasSummary {
			continue
		}
		var planFile string
		for _, f := range inner {
			if strings.HasSuffix(f.Name(), "-PLAN.md") {
				planFile = f.Name()
				break
			}
		}
		if planFile == "" {
			return "", fmt.Errorf("find active plan: no PLAN.md in active phase dir %s", d.name)
		}
		return filepath.Join(phasesDir, d.name, planFile), nil
	}

	return "", fmt.Errorf("find active plan: no active phase found in .planning/phases/")
}
