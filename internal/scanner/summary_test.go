package scanner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bkohler93/myhelper/internal/config"
	"github.com/bkohler93/myhelper/internal/history"
)

// Test 1: GenerateSummaries calls ChatFn exactly once when all entries are in the same package.
func TestGenerateSummaries_OneCallPerPackage_SinglePackage(t *testing.T) {
	calls := 0
	fakeChatFn := func(cfg config.Config, msgs []history.Message) (string, error) {
		calls++
		return "# Package foo\nDesign summary.", nil
	}

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".myhelper", "summaries"), 0755); err != nil {
		t.Fatal(err)
	}

	entries := []FileEntry{
		{Path: "a.go", Package: "foo", Symbols: []string{"func A()"}},
		{Path: "b.go", Package: "foo", Symbols: []string{"func B()"}},
		{Path: "c.go", Package: "foo", Symbols: []string{"func C()"}},
	}

	err := GenerateSummaries(root, entries, config.Config{}, fakeChatFn, func(s string) {})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if calls != 1 {
		t.Errorf("expected 1 ChatFn call, got %d", calls)
	}
}

// Test 2: GenerateSummaries calls ChatFn twice when entries span two packages.
func TestGenerateSummaries_TwoCalls_TwoPackages(t *testing.T) {
	calls := 0
	fakeChatFn := func(cfg config.Config, msgs []history.Message) (string, error) {
		calls++
		return "# Summary", nil
	}

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".myhelper", "summaries"), 0755); err != nil {
		t.Fatal(err)
	}

	entries := []FileEntry{
		{Path: "a.go", Package: "foo", Symbols: []string{"func A()"}},
		{Path: "b.go", Package: "foo", Symbols: []string{"func B()"}},
		{Path: "c.go", Package: "bar", Symbols: []string{"func C()"}},
	}

	err := GenerateSummaries(root, entries, config.Config{}, fakeChatFn, func(s string) {})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if calls != 2 {
		t.Errorf("expected 2 ChatFn calls, got %d", calls)
	}
}

// Test 3: GenerateSummaries creates the output file .myhelper/summaries/foo.md.
func TestGenerateSummaries_OutputFileCreated(t *testing.T) {
	fakeChatFn := func(cfg config.Config, msgs []history.Message) (string, error) {
		return "# Package foo\nDesign summary.", nil
	}

	root := t.TempDir()

	entries := []FileEntry{
		{Path: "a.go", Package: "foo", Symbols: []string{"func A()"}},
	}

	err := GenerateSummaries(root, entries, config.Config{}, fakeChatFn, func(s string) {})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	outPath := filepath.Join(root, ".myhelper", "summaries", "foo.md")
	if _, err := os.Stat(outPath); os.IsNotExist(err) {
		t.Errorf("expected file %s to exist, but it does not", outPath)
	}
}

// Test 4: The content of .myhelper/summaries/foo.md equals the string returned by ChatFn.
func TestGenerateSummaries_OutputFileContent(t *testing.T) {
	expectedContent := "# Package foo\nDesign summary."
	fakeChatFn := func(cfg config.Config, msgs []history.Message) (string, error) {
		return expectedContent, nil
	}

	root := t.TempDir()

	entries := []FileEntry{
		{Path: "a.go", Package: "foo", Symbols: []string{"func A()"}},
	}

	err := GenerateSummaries(root, entries, config.Config{}, fakeChatFn, func(s string) {})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	outPath := filepath.Join(root, ".myhelper", "summaries", "foo.md")
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	if string(data) != expectedContent {
		t.Errorf("expected content %q, got %q", expectedContent, string(data))
	}
}

// Test 5: The messages passed to ChatFn contain all symbols from the package.
func TestGenerateSummaries_PromptContainsSymbols(t *testing.T) {
	var capturedMsgs []history.Message
	fakeChatFn := func(cfg config.Config, msgs []history.Message) (string, error) {
		capturedMsgs = msgs
		return "# Summary", nil
	}

	root := t.TempDir()

	entries := []FileEntry{
		{Path: "a.go", Package: "foo", Symbols: []string{"func Bar(x int) string", "type Baz struct"}},
	}

	err := GenerateSummaries(root, entries, config.Config{}, fakeChatFn, func(s string) {})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(capturedMsgs) == 0 {
		t.Fatal("no messages were passed to ChatFn")
	}

	fullContent := ""
	for _, m := range capturedMsgs {
		fullContent += m.Content
	}

	if !strings.Contains(fullContent, "func Bar(x int) string") {
		t.Errorf("expected prompt to contain 'func Bar(x int) string', got: %s", fullContent)
	}
	if !strings.Contains(fullContent, "type Baz struct") {
		t.Errorf("expected prompt to contain 'type Baz struct', got: %s", fullContent)
	}
}

// Test 6: The message passed to ChatFn has Role == "user".
func TestGenerateSummaries_PromptIsUserRole(t *testing.T) {
	var capturedMsgs []history.Message
	fakeChatFn := func(cfg config.Config, msgs []history.Message) (string, error) {
		capturedMsgs = msgs
		return "# Summary", nil
	}

	root := t.TempDir()

	entries := []FileEntry{
		{Path: "a.go", Package: "foo", Symbols: []string{"func A()"}},
	}

	err := GenerateSummaries(root, entries, config.Config{}, fakeChatFn, func(s string) {})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(capturedMsgs) == 0 {
		t.Fatal("no messages were passed to ChatFn")
	}

	for i, m := range capturedMsgs {
		if m.Role != "user" {
			t.Errorf("message[%d] has role %q, expected %q", i, m.Role, "user")
		}
	}
}

// Test 7: GenerateSummaries with empty entries calls ChatFn zero times and returns no error.
func TestGenerateSummaries_EmptyEntries(t *testing.T) {
	calls := 0
	fakeChatFn := func(cfg config.Config, msgs []history.Message) (string, error) {
		calls++
		return "# Summary", nil
	}

	root := t.TempDir()

	err := GenerateSummaries(root, []FileEntry{}, config.Config{}, fakeChatFn, func(s string) {})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if calls != 0 {
		t.Errorf("expected 0 ChatFn calls, got %d", calls)
	}
}

// Test 8: When ChatFn returns an error, GenerateSummaries returns that error without writing a file.
func TestGenerateSummaries_ChatFnError(t *testing.T) {
	expectedErr := os.ErrPermission // use a sentinel error
	fakeChatFn := func(cfg config.Config, msgs []history.Message) (string, error) {
		return "", expectedErr
	}

	root := t.TempDir()

	entries := []FileEntry{
		{Path: "a.go", Package: "foo", Symbols: []string{"func A()"}},
	}

	err := GenerateSummaries(root, entries, config.Config{}, fakeChatFn, func(string) {})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}

	outPath := filepath.Join(root, ".myhelper", "summaries", "foo.md")
	if _, statErr := os.Stat(outPath); !os.IsNotExist(statErr) {
		t.Errorf("expected no output file after ChatFn error, but file exists at %s", outPath)
	}
}

// Test 9: GenerateSummaries skips packages whose name ends in _test — no file written, no ChatFn call for that package.
func TestGenerateSummaries_TestPackageSkipped(t *testing.T) {
	calls := 0
	fakeChatFn := func(cfg config.Config, msgs []history.Message) (string, error) {
		calls++
		return "# Summary", nil
	}

	root := t.TempDir()

	entries := []FileEntry{
		{Path: "foo.go", Package: "foo", Symbols: []string{"func A()"}},
		{Path: "foo_test.go", Package: "foo_test", Symbols: []string{"func TestA(t *testing.T)"}},
	}

	err := GenerateSummaries(root, entries, config.Config{}, fakeChatFn, func(s string) {})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// ChatFn must be called exactly once (for "foo", not "foo_test").
	if calls != 1 {
		t.Errorf("expected 1 ChatFn call (foo only), got %d", calls)
	}

	// foo.md must exist.
	fooPath := filepath.Join(root, ".myhelper", "summaries", "foo.md")
	if _, err := os.Stat(fooPath); os.IsNotExist(err) {
		t.Errorf("expected foo.md to exist, but it does not")
	}

	// foo_test.md must NOT exist.
	testPkgPath := filepath.Join(root, ".myhelper", "summaries", "foo_test.md")
	if _, err := os.Stat(testPkgPath); !os.IsNotExist(err) {
		t.Errorf("expected foo_test.md to not exist, but it does")
	}
}

// Test 10: The prompt passed to ChatFn contains a format directive (stable opening substring of summaryDirective).
func TestGenerateSummaries_PromptContainsFormatDirective(t *testing.T) {
	var capturedContent string
	fakeChatFn := func(cfg config.Config, msgs []history.Message) (string, error) {
		for _, m := range msgs {
			capturedContent += m.Content
		}
		return "# Summary", nil
	}

	root := t.TempDir()

	entries := []FileEntry{
		{Path: "foo.go", Package: "foo", Symbols: []string{"func A()"}},
	}

	err := GenerateSummaries(root, entries, config.Config{}, fakeChatFn, func(s string) {})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	const wantSubstr = "Identify the core purpose"
	if !strings.Contains(capturedContent, wantSubstr) {
		t.Errorf("expected prompt to contain %q as format directive, got:\n%s", wantSubstr, capturedContent)
	}
}
