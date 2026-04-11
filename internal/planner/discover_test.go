package planner_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bkohler93/myhelper/internal/planner"
)

func TestFindActivePlan(t *testing.T) {
	// Helper: creates a temp dir structure and changes CWD to it.
	// Returns a cleanup func that restores original CWD.
	setup := func(t *testing.T) (string, func()) {
		t.Helper()
		orig, err := os.Getwd()
		if err != nil {
			t.Fatalf("getwd: %v", err)
		}
		tmp := t.TempDir()
		if err := os.Chdir(tmp); err != nil {
			t.Fatalf("chdir to tmp: %v", err)
		}
		return tmp, func() { os.Chdir(orig) }
	}

	t.Run("returns path for highest-numbered dir without SUMMARY.md", func(t *testing.T) {
		tmp, cleanup := setup(t)
		defer cleanup()

		// Create 14-done/ with a SUMMARY.md
		dir14 := filepath.Join(tmp, ".planning", "phases", "14-done")
		if err := os.MkdirAll(dir14, 0755); err != nil {
			t.Fatalf("mkdirall 14-done: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir14, "14-01-SUMMARY.md"), []byte("done"), 0644); err != nil {
			t.Fatalf("write summary: %v", err)
		}

		// Create 15-active/ with only a PLAN.md
		dir15 := filepath.Join(tmp, ".planning", "phases", "15-active")
		if err := os.MkdirAll(dir15, 0755); err != nil {
			t.Fatalf("mkdirall 15-active: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir15, "15-01-PLAN.md"), []byte("plan content"), 0644); err != nil {
			t.Fatalf("write plan: %v", err)
		}

		got, err := planner.FindActivePlan()
		if err != nil {
			t.Fatalf("expected nil error, got: %v", err)
		}
		if !strings.HasSuffix(filepath.ToSlash(got), "15-active/15-01-PLAN.md") {
			t.Errorf("expected path ending in 15-active/15-01-PLAN.md, got %q", got)
		}
	})

	t.Run("skips completed phases (all have SUMMARY.md)", func(t *testing.T) {
		tmp, cleanup := setup(t)
		defer cleanup()

		// Create 14-done/ with a SUMMARY.md
		dir14 := filepath.Join(tmp, ".planning", "phases", "14-done")
		if err := os.MkdirAll(dir14, 0755); err != nil {
			t.Fatalf("mkdirall 14-done: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir14, "14-01-SUMMARY.md"), []byte("done"), 0644); err != nil {
			t.Fatalf("write summary: %v", err)
		}

		// Create 15-done/ also with a SUMMARY.md
		dir15 := filepath.Join(tmp, ".planning", "phases", "15-done")
		if err := os.MkdirAll(dir15, 0755); err != nil {
			t.Fatalf("mkdirall 15-done: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir15, "15-01-SUMMARY.md"), []byte("also done"), 0644); err != nil {
			t.Fatalf("write summary: %v", err)
		}

		_, err := planner.FindActivePlan()
		if err == nil {
			t.Fatal("expected error when all phases have SUMMARY.md")
		}
		if !strings.Contains(err.Error(), "no active phase found") {
			t.Errorf("expected error to contain 'no active phase found', got %q", err.Error())
		}
	})

	t.Run("returns error when phases dir does not exist", func(t *testing.T) {
		_, cleanup := setup(t)
		defer cleanup()

		// tmp dir has no .planning/phases/ subdirectory at all
		_, err := planner.FindActivePlan()
		if err == nil {
			t.Fatal("expected error when phases dir does not exist")
		}
		if !strings.Contains(err.Error(), "read phases dir") {
			t.Errorf("expected error to contain 'read phases dir', got %q", err.Error())
		}
	})

	t.Run("returns error when active dir has no PLAN.md", func(t *testing.T) {
		tmp, cleanup := setup(t)
		defer cleanup()

		// Create 15-active/ with only a CONTEXT.md, no PLAN.md
		dir15 := filepath.Join(tmp, ".planning", "phases", "15-active")
		if err := os.MkdirAll(dir15, 0755); err != nil {
			t.Fatalf("mkdirall 15-active: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir15, "15-CONTEXT.md"), []byte("context"), 0644); err != nil {
			t.Fatalf("write context: %v", err)
		}

		_, err := planner.FindActivePlan()
		if err == nil {
			t.Fatal("expected error when active dir has no PLAN.md")
		}
		if !strings.Contains(err.Error(), "no PLAN.md") {
			t.Errorf("expected error to contain 'no PLAN.md', got %q", err.Error())
		}
	})

	t.Run("numeric sort: 9-foo comes before 10-bar (10 is higher)", func(t *testing.T) {
		tmp, cleanup := setup(t)
		defer cleanup()

		// Create 9-lower/ with a SUMMARY.md (completed)
		dir9 := filepath.Join(tmp, ".planning", "phases", "9-lower")
		if err := os.MkdirAll(dir9, 0755); err != nil {
			t.Fatalf("mkdirall 9-lower: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir9, "9-01-SUMMARY.md"), []byte("done"), 0644); err != nil {
			t.Fatalf("write summary: %v", err)
		}

		// Create 10-higher/ with only a PLAN.md (active)
		dir10 := filepath.Join(tmp, ".planning", "phases", "10-higher")
		if err := os.MkdirAll(dir10, 0755); err != nil {
			t.Fatalf("mkdirall 10-higher: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir10, "10-01-PLAN.md"), []byte("plan content"), 0644); err != nil {
			t.Fatalf("write plan: %v", err)
		}

		got, err := planner.FindActivePlan()
		if err != nil {
			t.Fatalf("expected nil error, got: %v", err)
		}
		if !strings.Contains(filepath.ToSlash(got), "10-higher") {
			t.Errorf("expected path to contain '10-higher' (integer sort), got %q", got)
		}
	})
}
