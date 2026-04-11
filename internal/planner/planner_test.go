package planner_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bkohler93/myhelper/internal/planner"
)

func TestParsePlan(t *testing.T) {
	fixturePath := filepath.Join("..", "..", ".planning", "phases", "14-ollama-client-extension", "14-01-PLAN.md")

	t.Run("parses 14-01-PLAN.md frontmatter correctly", func(t *testing.T) {
		plan, err := planner.ParsePlan(fixturePath)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if plan.Phase != "14-ollama-client-extension" {
			t.Errorf("expected Phase %q, got %q", "14-ollama-client-extension", plan.Phase)
		}
		if plan.PlanNum != 1 {
			t.Errorf("expected PlanNum 1, got %d", plan.PlanNum)
		}
		if plan.Wave != 1 {
			t.Errorf("expected Wave 1, got %d", plan.Wave)
		}
		if !plan.Autonomous {
			t.Errorf("expected Autonomous true, got false")
		}
		if len(plan.FilesModified) != 2 {
			t.Errorf("expected 2 files_modified, got %d: %v", len(plan.FilesModified), plan.FilesModified)
		}
		if len(plan.FilesModified) > 0 && plan.FilesModified[0] != "internal/ollama/client.go" {
			t.Errorf("expected FilesModified[0] %q, got %q", "internal/ollama/client.go", plan.FilesModified[0])
		}
	})

	t.Run("parses 14-01-PLAN.md tasks correctly", func(t *testing.T) {
		plan, err := planner.ParsePlan(fixturePath)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if len(plan.Tasks) != 2 {
			t.Errorf("expected 2 tasks, got %d", len(plan.Tasks))
		}
		if len(plan.Tasks) > 0 {
			if plan.Tasks[0].Name != "Task 1: Add Format field to chatRequest and implement ChatWithFormat" {
				t.Errorf("expected Tasks[0].Name %q, got %q",
					"Task 1: Add Format field to chatRequest and implement ChatWithFormat",
					plan.Tasks[0].Name)
			}
			if plan.Tasks[0].Files != "internal/ollama/client.go" {
				t.Errorf("expected Tasks[0].Files %q, got %q", "internal/ollama/client.go", plan.Tasks[0].Files)
			}
			if len(plan.Tasks[0].Action) == 0 {
				t.Error("expected Tasks[0].Action to be non-empty")
			}
		}
		if len(plan.Tasks) > 1 {
			if !strings.HasPrefix(plan.Tasks[1].Name, "Task 2:") {
				t.Errorf("expected Tasks[1].Name to start with 'Task 2:', got %q", plan.Tasks[1].Name)
			}
		}
	})

	t.Run("returns error on missing frontmatter delimiters", func(t *testing.T) {
		dir := t.TempDir()
		f := filepath.Join(dir, "plan.md")
		content := "# No frontmatter here\n\nJust some content.\n"
		if err := os.WriteFile(f, []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write temp file: %v", err)
		}
		_, err := planner.ParsePlan(f)
		if err == nil {
			t.Fatal("expected error for missing frontmatter delimiters")
		}
		if !strings.Contains(err.Error(), "missing frontmatter delimiters") {
			t.Errorf("expected error to contain 'missing frontmatter delimiters', got %q", err.Error())
		}
	})

	t.Run("returns error on task missing <name>", func(t *testing.T) {
		dir := t.TempDir()
		f := filepath.Join(dir, "plan.md")
		content := `---
phase: test-phase
plan: 1
wave: 1
files_modified:
  - src/foo.go
autonomous: true
---

<task type="auto">
  <files>src/foo.go</files>
  <action>Write something.</action>
</task>
`
		if err := os.WriteFile(f, []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write temp file: %v", err)
		}
		_, err := planner.ParsePlan(f)
		if err == nil {
			t.Fatal("expected error for task missing <name>")
		}
		if !strings.Contains(err.Error(), "missing <name>") {
			t.Errorf("expected error to contain 'missing <name>', got %q", err.Error())
		}
	})

	t.Run("returns error on task missing <files>", func(t *testing.T) {
		dir := t.TempDir()
		f := filepath.Join(dir, "plan.md")
		content := `---
phase: test-phase
plan: 1
wave: 1
files_modified:
  - src/foo.go
autonomous: true
---

<task type="auto">
  <name>Task 1: Do something</name>
  <action>Write something.</action>
</task>
`
		if err := os.WriteFile(f, []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write temp file: %v", err)
		}
		_, err := planner.ParsePlan(f)
		if err == nil {
			t.Fatal("expected error for task missing <files>")
		}
		if !strings.Contains(err.Error(), "missing <files>") {
			t.Errorf("expected error to contain 'missing <files>', got %q", err.Error())
		}
	})

	t.Run("returns error on task missing <action>", func(t *testing.T) {
		dir := t.TempDir()
		f := filepath.Join(dir, "plan.md")
		content := `---
phase: test-phase
plan: 1
wave: 1
files_modified:
  - src/foo.go
autonomous: true
---

<task type="auto">
  <name>Task 1: Do something</name>
  <files>src/foo.go</files>
</task>
`
		if err := os.WriteFile(f, []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write temp file: %v", err)
		}
		_, err := planner.ParsePlan(f)
		if err == nil {
			t.Fatal("expected error for task missing <action>")
		}
		if !strings.Contains(err.Error(), "missing <action>") {
			t.Errorf("expected error to contain 'missing <action>', got %q", err.Error())
		}
	})

	t.Run("empty <behavior> is not an error", func(t *testing.T) {
		dir := t.TempDir()
		f := filepath.Join(dir, "plan.md")
		content := `---
phase: test-phase
plan: 1
wave: 1
files_modified:
  - src/foo.go
autonomous: true
---

<task type="auto">
  <name>Task 1: Do something</name>
  <files>src/foo.go</files>
  <action>Write something.</action>
</task>
`
		if err := os.WriteFile(f, []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write temp file: %v", err)
		}
		plan, err := planner.ParsePlan(f)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if len(plan.Tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(plan.Tasks))
		}
		if plan.Tasks[0].Behavior != "" {
			t.Errorf("expected empty Behavior, got %q", plan.Tasks[0].Behavior)
		}
	})
}
