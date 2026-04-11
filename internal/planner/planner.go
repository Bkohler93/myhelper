package planner

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Plan holds the parsed contents of a GSD PLAN.md file.
type Plan struct {
	Phase         string
	PlanNum       int
	Wave          int
	FilesModified []string
	Autonomous    bool
	Tasks         []Task
}

// Task holds a single task block extracted from a PLAN.md file.
type Task struct {
	Name     string
	Files    string
	Behavior string
	Action   string
}

// ParsePlan reads the GSD PLAN.md file at path, extracts YAML frontmatter fields
// and XML task blocks, and returns a populated Plan or an error.
// Never silently drops tasks — a task with a missing required field returns an error.
func ParsePlan(path string) (Plan, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Plan{}, fmt.Errorf("read plan: %w", err)
	}

	content := string(data)

	// Phase 1: split frontmatter from body.
	// Frontmatter is between the first and second `---` delimiter lines.
	firstDelim := strings.Index(content, "---")
	if firstDelim == -1 {
		return Plan{}, fmt.Errorf("parse plan: missing frontmatter delimiters")
	}

	// Search for the second `---` after the first one (skip past first delimiter)
	afterFirst := content[firstDelim+3:]
	// The first delimiter might be at the very start; advance past the newline
	newlineAfterFirst := strings.Index(afterFirst, "\n")
	if newlineAfterFirst == -1 {
		return Plan{}, fmt.Errorf("parse plan: missing frontmatter delimiters")
	}
	afterFirstLine := afterFirst[newlineAfterFirst+1:]

	secondDelimIdx := strings.Index(afterFirstLine, "---")
	if secondDelimIdx == -1 {
		return Plan{}, fmt.Errorf("parse plan: missing frontmatter delimiters")
	}

	frontmatter := afterFirstLine[:secondDelimIdx]
	body := afterFirstLine[secondDelimIdx+3:]
	// Advance past the `---` line's newline
	if idx := strings.Index(body, "\n"); idx != -1 {
		body = body[idx+1:]
	}

	// Phase 2: parse frontmatter with bufio scanner.
	var p Plan
	inFilesModified := false

	sc := bufio.NewScanner(strings.NewReader(frontmatter))
	for sc.Scan() {
		line := sc.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "phase:") {
			p.Phase = strings.TrimSpace(strings.TrimPrefix(trimmed, "phase:"))
			inFilesModified = false
			continue
		}

		if strings.HasPrefix(trimmed, "plan:") {
			val := strings.TrimSpace(strings.TrimPrefix(trimmed, "plan:"))
			n, err := strconv.Atoi(val)
			if err != nil {
				return Plan{}, fmt.Errorf("parse plan: invalid plan number: %w", err)
			}
			p.PlanNum = n
			inFilesModified = false
			continue
		}

		if strings.HasPrefix(trimmed, "wave:") {
			val := strings.TrimSpace(strings.TrimPrefix(trimmed, "wave:"))
			n, err := strconv.Atoi(val)
			if err != nil {
				return Plan{}, fmt.Errorf("parse plan: invalid wave: %w", err)
			}
			p.Wave = n
			inFilesModified = false
			continue
		}

		if strings.HasPrefix(trimmed, "autonomous:") {
			val := strings.TrimSpace(strings.TrimPrefix(trimmed, "autonomous:"))
			p.Autonomous = val == "true"
			inFilesModified = false
			continue
		}

		if strings.HasPrefix(trimmed, "files_modified:") {
			inFilesModified = true
			continue
		}

		if inFilesModified && strings.HasPrefix(line, "  - ") {
			entry := strings.TrimSpace(strings.TrimPrefix(line, "  - "))
			p.FilesModified = append(p.FilesModified, entry)
			continue
		}

		// If we're in files_modified and hit a non-list line, check if it's a new key.
		if inFilesModified && !strings.HasPrefix(line, "  - ") && trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			inFilesModified = false
		}
	}

	// Phase 3: extract XML task blocks from body.
	remaining := body
	taskIdx := 0
	for {
		start := strings.Index(remaining, "<task")
		if start == -1 {
			break
		}
		end := strings.Index(remaining[start:], "</task>")
		if end == -1 {
			break
		}
		block := remaining[start : start+end+len("</task>")]
		remaining = remaining[start+end+len("</task>"):]

		t := Task{
			Name:     extractElement(block, "name"),
			Files:    extractElement(block, "files"),
			Behavior: extractElement(block, "behavior"),
			Action:   extractElement(block, "action"),
		}

		if t.Name == "" {
			return Plan{}, fmt.Errorf("parse plan: task %d missing <name>", taskIdx+1)
		}
		if t.Files == "" {
			return Plan{}, fmt.Errorf("parse plan: task %d missing <files>", taskIdx+1)
		}
		if t.Action == "" {
			return Plan{}, fmt.Errorf("parse plan: task %d missing <action>", taskIdx+1)
		}

		p.Tasks = append(p.Tasks, t)
		taskIdx++
	}

	return p, nil
}

// extractElement returns the trimmed inner text of <tag>...</tag> within block.
// Returns empty string if the tag is absent.
func extractElement(block, tag string) string {
	open := "<" + tag + ">"
	close := "</" + tag + ">"

	start := strings.Index(block, open)
	if start == -1 {
		return ""
	}
	start += len(open)

	end := strings.Index(block[start:], close)
	if end == -1 {
		return ""
	}

	return strings.TrimSpace(block[start : start+end])
}
