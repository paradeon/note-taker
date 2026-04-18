package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// tmpFile creates a temp file and returns its path. Caller is responsible for removal.
func tmpFile(t *testing.T) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "notes-*.md")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	// Remove so hasContent sees it as absent (size 0 / missing).
	os.Remove(f.Name())
	return f.Name()
}

// writeFile writes content to path, creating parent dirs.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

// readFile reads and returns file content.
func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// ── defaultNoteFile ──────────────────────────────────────────────────────────

func TestDefaultNoteFile_EnvVar(t *testing.T) {
	t.Setenv("NOTE_FILE", "/tmp/custom-notes.md")
	got := defaultNoteFile()
	if got != "/tmp/custom-notes.md" {
		t.Errorf("got %q, want /tmp/custom-notes.md", got)
	}
}

func TestDefaultNoteFile_Default(t *testing.T) {
	t.Setenv("NOTE_FILE", "")
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, "notes", "quick-notes.md")
	got := defaultNoteFile()
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// ── hasContent ───────────────────────────────────────────────────────────────

func TestHasContent_Missing(t *testing.T) {
	if hasContent("/tmp/does-not-exist-xyz.md") {
		t.Error("expected false for missing file")
	}
}

func TestHasContent_Empty(t *testing.T) {
	f, _ := os.CreateTemp(t.TempDir(), "*.md")
	f.Close()
	if hasContent(f.Name()) {
		t.Error("expected false for empty file")
	}
}

func TestHasContent_NonEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.md")
	writeFile(t, path, "some content")
	if !hasContent(path) {
		t.Error("expected true for non-empty file")
	}
}

// ── processTags ──────────────────────────────────────────────────────────────

func TestProcessTags_ConvertsSuffix(t *testing.T) {
	got := processTags("buy milk ,,groceries,food")
	want := "buy milk #groceries #food"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestProcessTags_ConvertsInline(t *testing.T) {
	got := processTags(",,auth,backend fix login bug")
	want := "#auth #backend fix login bug"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestProcessTags_SingleTag(t *testing.T) {
	got := processTags("call dentist ,,health")
	want := "call dentist #health"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestProcessTags_NoTags(t *testing.T) {
	got := processTags("just a plain note")
	want := "just a plain note"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestProcessTags_PreservesExistingHashtags(t *testing.T) {
	got := processTags("note #existing ,,new")
	want := "note #existing #new"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// ── appendNote ───────────────────────────────────────────────────────────────

func TestAppendNote_CreatesHeaderOnFirstNote(t *testing.T) {
	path := tmpFile(t)
	var buf bytes.Buffer

	if err := appendNote(&buf, path, "first note"); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, path)
	if !strings.HasPrefix(content, "# Quick Notes") {
		t.Errorf("expected header, got:\n%s", content)
	}
	if !strings.Contains(content, "first note") {
		t.Errorf("note text missing from file:\n%s", content)
	}
}

func TestAppendNote_NoHeaderOnSubsequentNotes(t *testing.T) {
	path := tmpFile(t)
	var buf bytes.Buffer

	appendNote(&buf, path, "first note")
	buf.Reset()
	appendNote(&buf, path, "second note")

	content := readFile(t, path)
	count := strings.Count(content, "# Quick Notes")
	if count != 1 {
		t.Errorf("expected header exactly once, found %d times:\n%s", count, content)
	}
}

func TestAppendNote_TimestampFormat(t *testing.T) {
	path := tmpFile(t)
	var buf bytes.Buffer

	before := time.Now()
	appendNote(&buf, path, "timed note")
	after := time.Now()

	content := readFile(t, path)
	// Check the date portion matches today (format: 2006-01-02)
	dateStr := before.Format("2006-01-02")
	if !strings.Contains(content, dateStr) {
		t.Errorf("expected date %s in content:\n%s", dateStr, content)
	}
	_ = after
}

func TestAppendNote_MultipleNotes(t *testing.T) {
	path := tmpFile(t)
	var buf bytes.Buffer

	appendNote(&buf, path, "note one")
	buf.Reset()
	appendNote(&buf, path, "note two")

	content := readFile(t, path)
	if !strings.Contains(content, "note one") || !strings.Contains(content, "note two") {
		t.Errorf("both notes should be present:\n%s", content)
	}
}

func TestAppendNote_OutputConfirmation(t *testing.T) {
	path := tmpFile(t)
	var buf bytes.Buffer

	appendNote(&buf, path, "hello")
	out := buf.String()
	if !strings.Contains(out, "✓ Note saved") {
		t.Errorf("expected confirmation message, got: %q", out)
	}
	if !strings.Contains(out, path) {
		t.Errorf("expected file path in confirmation, got: %q", out)
	}
}

func TestAppendNote_CreatesParentDirs(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "deep", "nested", "dir")
	path := filepath.Join(dir, "notes.md")
	var buf bytes.Buffer

	if err := appendNote(&buf, path, "nested"); err != nil {
		t.Fatal(err)
	}
	if !hasContent(path) {
		t.Error("expected file to be created in nested directory")
	}
}

// ── searchTag ────────────────────────────────────────────────────────────────

func TestSearchTag_NoFile(t *testing.T) {
	var buf bytes.Buffer
	searchTag(&buf, "/tmp/no-such-file-xyz.md", "foo")
	if !strings.Contains(buf.String(), "No notes yet") {
		t.Errorf("expected 'No notes yet', got: %q", buf.String())
	}
}

func TestSearchTag_NoMatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.md")
	writeFile(t, path, "- **2026-01-01** — unrelated note\n")
	var buf bytes.Buffer

	searchTag(&buf, path, "missing")
	if !strings.Contains(buf.String(), "No notes tagged #missing") {
		t.Errorf("expected no-match message, got: %q", buf.String())
	}
}

func TestSearchTag_Match(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.md")
	writeFile(t, path, "- **2026-01-01** — fix login #auth\n- **2026-01-01** — unrelated\n")
	var buf bytes.Buffer

	searchTag(&buf, path, "auth")
	out := buf.String()
	if !strings.Contains(out, "fix login #auth") {
		t.Errorf("expected matching line, got: %q", out)
	}
	if strings.Contains(out, "unrelated") {
		t.Errorf("non-matching line should be excluded, got: %q", out)
	}
}

func TestSearchTag_CaseInsensitive(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.md")
	writeFile(t, path, "- **2026-01-01** — deploy fix #Backend\n")
	var buf bytes.Buffer

	searchTag(&buf, path, "backend")
	if !strings.Contains(buf.String(), "#Backend") {
		t.Errorf("expected case-insensitive match, got: %q", buf.String())
	}
}

func TestSearchTag_MultipleMatches(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.md")
	writeFile(t, path, "- note one #go\n- note two #go\n- note three #python\n")
	var buf bytes.Buffer

	searchTag(&buf, path, "go")
	out := buf.String()
	if !strings.Contains(out, "note one") || !strings.Contains(out, "note two") {
		t.Errorf("expected both matching lines, got: %q", out)
	}
	if strings.Contains(out, "note three") {
		t.Errorf("non-matching line should be excluded, got: %q", out)
	}
}

// ── listTags ─────────────────────────────────────────────────────────────────

func TestListTags_NoFile(t *testing.T) {
	var buf bytes.Buffer
	listTags(&buf, "/tmp/no-such-file-xyz.md")
	if !strings.Contains(buf.String(), "No notes yet") {
		t.Errorf("expected 'No notes yet', got: %q", buf.String())
	}
}

func TestListTags_NoTags(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.md")
	writeFile(t, path, "- **2026-01-01** — plain note\n")
	var buf bytes.Buffer
	listTags(&buf, path)
	if !strings.Contains(buf.String(), "No tags found") {
		t.Errorf("expected 'No tags found', got: %q", buf.String())
	}
}

func TestListTags_ListsTags(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.md")
	writeFile(t, path, "- **2026-01-01** — fix login #auth\n- **2026-01-02** — deploy #backend\n")
	var buf bytes.Buffer
	listTags(&buf, path)
	out := buf.String()
	if !strings.Contains(out, "#auth") || !strings.Contains(out, "#backend") {
		t.Errorf("expected both tags, got: %q", out)
	}
}

func TestListTags_Deduplicates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.md")
	writeFile(t, path, "- **2026-01-01** — note #auth\n- **2026-01-02** — note #auth\n")
	var buf bytes.Buffer
	listTags(&buf, path)
	out := buf.String()
	if strings.Count(out, "#auth") != 1 {
		t.Errorf("expected #auth exactly once, got: %q", out)
	}
}

// ── listNotes ────────────────────────────────────────────────────────────────

func TestListNotes_FilterByTag(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.md")
	writeFile(t, path, "# Quick Notes\n\n- **2026-01-01 Thu 09:00** — fix login #auth\n- **2026-01-02 Fri 10:00** — deploy app #backend\n")
	var buf bytes.Buffer

	listNotes(&buf, path, []string{"auth"})
	out := buf.String()
	if !strings.Contains(out, "fix login #auth") {
		t.Errorf("expected matching note, got: %q", out)
	}
	if strings.Contains(out, "deploy app") {
		t.Errorf("non-matching note should be excluded, got: %q", out)
	}
}

func TestListNotes_FilterByMultipleTags(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.md")
	writeFile(t, path, "# Quick Notes\n\n- **2026-01-01 Thu 09:00** — fix login #auth\n- **2026-01-02 Fri 10:00** — deploy #backend\n- **2026-01-03 Sat 11:00** — unrelated\n")
	var buf bytes.Buffer

	listNotes(&buf, path, []string{"auth", "backend"})
	out := buf.String()
	if !strings.Contains(out, "fix login #auth") || !strings.Contains(out, "deploy #backend") {
		t.Errorf("expected both tagged notes, got: %q", out)
	}
	if strings.Contains(out, "unrelated") {
		t.Errorf("untagged note should be excluded, got: %q", out)
	}
}

func TestListNotes_FilterCaseInsensitive(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.md")
	writeFile(t, path, "# Quick Notes\n\n- **2026-01-01 Thu 09:00** — note #Auth\n")
	var buf bytes.Buffer

	listNotes(&buf, path, []string{"auth"})
	if !strings.Contains(buf.String(), "note #Auth") {
		t.Errorf("expected case-insensitive match, got: %q", buf.String())
	}
}

func TestListNotes_NoFile(t *testing.T) {
	var buf bytes.Buffer
	listNotes(&buf, "/tmp/no-such-file-xyz.md", nil)
	if !strings.Contains(buf.String(), "No notes yet") {
		t.Errorf("expected 'No notes yet', got: %q", buf.String())
	}
}

func TestListNotes_HidesTimestamps(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.md")
	writeFile(t, path, "# Quick Notes\n\n- **2026-01-02 Fri 09:00** — buy milk\n- **2026-01-03 Sat 10:00** — call dentist\n")
	var buf bytes.Buffer

	if err := listNotes(&buf, path, nil); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Contains(out, "2026-01-02") || strings.Contains(out, "2026-01-03") {
		t.Errorf("timestamps should be hidden, got: %q", out)
	}
	if !strings.Contains(out, "buy milk") || !strings.Contains(out, "call dentist") {
		t.Errorf("note text missing, got: %q", out)
	}
}

func TestListNotes_FormatWithID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.md")
	writeFile(t, path, "# Quick Notes\n\n- [3] **2026-01-02 Fri 09:00** — buy milk\n")
	var buf bytes.Buffer

	listNotes(&buf, path, nil)
	if !strings.Contains(buf.String(), "[3] buy milk") {
		t.Errorf("expected '[3] buy milk', got: %q", buf.String())
	}
}

func TestListNotes_FormatWithoutID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.md")
	writeFile(t, path, "# Quick Notes\n\n- **2026-01-02 Fri 09:00** — buy milk\n")
	var buf bytes.Buffer

	listNotes(&buf, path, nil)
	if !strings.Contains(buf.String(), "[-] buy milk") {
		t.Errorf("expected '[-] buy milk', got: %q", buf.String())
	}
}

// ── nextNoteID ───────────────────────────────────────────────────────────────

func TestNextNoteID_EmptyFile(t *testing.T) {
	if got := nextNoteID("/tmp/no-such-file-xyz.md"); got != 1 {
		t.Errorf("got %d, want 1", got)
	}
}

func TestNextNoteID_AfterExisting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.md")
	writeFile(t, path, "- [1] **2026-01-01** — a\n- [3] **2026-01-02** — b\n")
	if got := nextNoteID(path); got != 4 {
		t.Errorf("got %d, want 4", got)
	}
}

func TestNextNoteID_IgnoresManualNotes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.md")
	writeFile(t, path, "- [2] **2026-01-01** — with id\n- **2026-01-02** — without id\n")
	if got := nextNoteID(path); got != 3 {
		t.Errorf("got %d, want 3", got)
	}
}

func TestAppendNote_EmbedID(t *testing.T) {
	path := tmpFile(t)
	var buf bytes.Buffer

	appendNote(&buf, path, "first")
	appendNote(&buf, path, "second")

	content := readFile(t, path)
	if !strings.Contains(content, "[1]") || !strings.Contains(content, "[2]") {
		t.Errorf("expected [1] and [2] in file:\n%s", content)
	}
}

// ── parseIDs ─────────────────────────────────────────────────────────────────

func TestParseIDs_SpaceSeparated(t *testing.T) {
	ids, err := parseIDs([]string{"1", "2", "3"})
	if err != nil || len(ids) != 3 || ids[0] != 1 || ids[2] != 3 {
		t.Errorf("got %v %v", ids, err)
	}
}

func TestParseIDs_CommaSeparated(t *testing.T) {
	ids, err := parseIDs([]string{"1,2,3"})
	if err != nil || len(ids) != 3 {
		t.Errorf("got %v %v", ids, err)
	}
}

func TestParseIDs_Mixed(t *testing.T) {
	ids, err := parseIDs([]string{"1,2", "3"})
	if err != nil || len(ids) != 3 {
		t.Errorf("got %v %v", ids, err)
	}
}

func TestParseIDs_Deduplicates(t *testing.T) {
	ids, err := parseIDs([]string{"1", "1", "2"})
	if err != nil || len(ids) != 2 {
		t.Errorf("got %v %v", ids, err)
	}
}

func TestParseIDs_InvalidReturnsError(t *testing.T) {
	if _, err := parseIDs([]string{"abc"}); err == nil {
		t.Error("expected error for non-integer id")
	}
}

// ── deleteNotes ───────────────────────────────────────────────────────────────

func TestDeleteNotes_RemovesMatchingLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.md")
	writeFile(t, path, "# Quick Notes\n\n- [1] **2026-01-01 Thu 09:00** — buy milk\n- [2] **2026-01-02 Fri 10:00** — call dentist\n- [3] **2026-01-03 Sat 11:00** — fix bug\n")
	var buf bytes.Buffer

	if err := deleteNotes(&buf, path, []int{1, 3}); err != nil {
		t.Fatal(err)
	}
	content := readFile(t, path)
	if strings.Contains(content, "[1]") || strings.Contains(content, "[3]") {
		t.Errorf("deleted notes still present:\n%s", content)
	}
	if !strings.Contains(content, "[2]") {
		t.Errorf("kept note missing:\n%s", content)
	}
}

func TestDeleteNotes_ReportsUnknownID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.md")
	writeFile(t, path, "# Quick Notes\n\n- [1] **2026-01-01 Thu 09:00** — buy milk\n")
	var buf bytes.Buffer

	deleteNotes(&buf, path, []int{99})
	if !strings.Contains(buf.String(), "not found") {
		t.Errorf("expected 'not found' message, got: %q", buf.String())
	}
}

func TestDeleteNotes_NoFile(t *testing.T) {
	var buf bytes.Buffer
	deleteNotes(&buf, "/tmp/no-such-file-xyz.md", []int{1})
	if !strings.Contains(buf.String(), "No notes yet") {
		t.Errorf("expected 'No notes yet', got: %q", buf.String())
	}
}

// ── printNotes ───────────────────────────────────────────────────────────────

func TestPrintNotes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.md")
	content := "# Quick Notes\n\n- **2026-01-01** — hello\n"
	writeFile(t, path, content)
	var buf bytes.Buffer

	if err := printNotes(&buf, path); err != nil {
		t.Fatal(err)
	}
	if buf.String() != content {
		t.Errorf("got %q, want %q", buf.String(), content)
	}
}

func TestPrintNotes_MissingFile(t *testing.T) {
	var buf bytes.Buffer
	err := printNotes(&buf, "/tmp/no-such-file-xyz.md")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

// ── printHelp ────────────────────────────────────────────────────────────────

func TestPrintHelp_ContainsKeyStrings(t *testing.T) {
	var buf bytes.Buffer
	printHelp(&buf, "/some/path/notes.md")
	out := buf.String()

	for _, want := range []string{
		"note — quick markdown note-taker",
		"note add",
		"note edit",
		"--file",
		"/some/path/notes.md",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("help output missing %q", want)
		}
	}
}
