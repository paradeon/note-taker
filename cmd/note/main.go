package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func defaultNoteFile() string {
	if f := os.Getenv("NOTE_FILE"); f != "" {
		return f
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "notes", "quick-notes.md")
}

func printHelp(w io.Writer, file string) {
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  note — quick markdown note-taker")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  Usage:")
	fmt.Fprintln(w, "    note add <text...>           Append a timestamped note (URLs auto-linked)")
	fmt.Fprintln(w, "    note add \"quoted text\"       Quoted or unquoted — both work")
	fmt.Fprintln(w, "    note add --no-mdurl <text>   Skip auto-linking URLs")
	fmt.Fprintln(w, "    note list                    Display all notes without timestamps")
	fmt.Fprintln(w, "    note list -t <tag>[,<tag>]   Filter notes by one or more tags")
	fmt.Fprintln(w, "    note tags                    List all tags used in the notes file")
	fmt.Fprintln(w, "    note edit                    Open notes file in nvim")
	fmt.Fprintln(w, "    note edit <id>               Edit a single note inline (vi-mode)")
	fmt.Fprintln(w, "    note delete <id>...          Delete notes by id (space or comma separated)")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  Use -f / --file <path> with any action to target a specific file:")
	fmt.Fprintln(w, "    note add -f <path> <text...>")
	fmt.Fprintln(w, "    note list -f <path>")
	fmt.Fprintln(w, "    note edit -f <path>")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  Tags:")
	fmt.Fprintln(w, "    Use ,,tag1,tag2 shorthand — converted to #tag1 #tag2 on save:")
	fmt.Fprintln(w, "    note add fix login bug ,,auth,backend")
	fmt.Fprintln(w, "    Or embed #tags directly in the text:")
	fmt.Fprintln(w, "    note add fix login bug #auth #backend")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  Notes file:")
	fmt.Fprintln(w, "   ", file)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  Override the file by setting NOTE_FILE in your environment:")
	fmt.Fprintln(w, "    set -x NOTE_FILE \"$HOME/Dropbox/notes/quick-notes.md\"")
	fmt.Fprintln(w)
}

func hasContent(file string) bool {
	info, err := os.Stat(file)
	return err == nil && info.Size() > 0
}

func printNotes(w io.Writer, file string) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(w, string(data))
	return err
}

func editNotes(file string) error {
	if !hasContent(file) {
		fmt.Println("No notes file yet. Creating", file+"...")
		if err := os.MkdirAll(filepath.Dir(file), 0755); err != nil {
			return err
		}
	}
	cmd := exec.Command("nvim", file)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func searchTag(w io.Writer, file, tag string) error {
	if !hasContent(file) {
		fmt.Fprintln(w, "No notes yet.")
		return nil
	}
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	needle := "#" + strings.ToLower(tag)
	var matches []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(strings.ToLower(line), needle) {
			matches = append(matches, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	if len(matches) == 0 {
		fmt.Fprintln(w, "No notes tagged #"+tag)
		return nil
	}
	fmt.Fprintln(w, "Notes tagged #"+tag+":")
	fmt.Fprintln(w)
	for _, line := range matches {
		fmt.Fprintln(w, line)
	}
	return nil
}

// noteLineRe matches note lines with an optional [N] id prefix.
// Group 1: id (may be empty), Group 2: note text.
var noteLineRe = regexp.MustCompile(`^- (?:\[(\d+)\] )?\*\*[^*]+\*\* — (.+)$`)

var noteIDLineRe = regexp.MustCompile(`^- \[(\d+)\]`)

func nextNoteID(file string) int {
	if !hasContent(file) {
		return 1
	}
	f, err := os.Open(file)
	if err != nil {
		return 1
	}
	defer f.Close()

	max := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if m := noteIDLineRe.FindStringSubmatch(scanner.Text()); m != nil {
			if id, err := strconv.Atoi(m[1]); err == nil && id > max {
				max = id
			}
		}
	}
	return max + 1
}

// processTags converts ,,tag1,tag2 tokens in text to #tag1 #tag2.
func processTags(text string) string {
	words := strings.Fields(text)
	for i, w := range words {
		if !strings.HasPrefix(w, ",,") {
			continue
		}
		tags := strings.Split(w[2:], ",")
		var hashtags []string
		for _, t := range tags {
			if t != "" {
				hashtags = append(hashtags, "#"+t)
			}
		}
		words[i] = strings.Join(hashtags, " ")
	}
	return strings.Join(words, " ")
}

func listNotes(w io.Writer, file string, tags []string, filterID int) error {
	if !hasContent(file) {
		fmt.Fprintln(w, "No notes yet. Try: note add your first thought")
		return nil
	}
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		m := noteLineRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		if len(tags) > 0 && !hasAnyTag(line, tags) {
			continue
		}
		id, text := m[1], m[2]
		if filterID > 0 {
			if lineID, _ := strconv.Atoi(id); lineID == filterID {
				fmt.Fprintln(w, text)
			}
			continue
		}
		if id != "" {
			fmt.Fprintf(w, "[%s] %s\n", id, text)
		} else {
			fmt.Fprintf(w, "[-] %s\n", text)
		}
	}
	return scanner.Err()
}

func hasAnyTag(line string, tags []string) bool {
	lower := strings.ToLower(line)
	for _, t := range tags {
		if strings.Contains(lower, "#"+strings.ToLower(t)) {
			return true
		}
	}
	return false
}

var tagRe = regexp.MustCompile(`#(\w+)`)

var rawURLRe = regexp.MustCompile(`https?://\S+`)

// processURLs replaces bare URLs in text with markdown links via mdurl.
// Silently leaves a URL unchanged if mdurl fails or is not installed.
func processURLs(text string) string {
	return rawURLRe.ReplaceAllStringFunc(text, func(url string) string {
		out, err := exec.Command("mdurl", url).Output()
		if err != nil {
			return url
		}
		return strings.TrimSpace(string(out))
	})
}

func listTags(w io.Writer, file string) error {
	if !hasContent(file) {
		fmt.Fprintln(w, "No notes yet.")
		return nil
	}
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	seen := map[string]bool{}
	var tags []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		for _, m := range tagRe.FindAllStringSubmatch(scanner.Text(), -1) {
			t := m[1]
			if !seen[t] {
				seen[t] = true
				tags = append(tags, t)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if len(tags) == 0 {
		fmt.Fprintln(w, "No tags found.")
		return nil
	}
	for _, t := range tags {
		fmt.Fprintln(w, "#"+t)
	}
	return nil
}

func parseIDs(args []string) ([]int, error) {
	seen := map[int]bool{}
	var ids []int
	for _, arg := range args {
		for _, part := range strings.Split(arg, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			n, err := strconv.Atoi(part)
			if err != nil || n < 1 {
				return nil, fmt.Errorf("invalid id %q", part)
			}
			if !seen[n] {
				seen[n] = true
				ids = append(ids, n)
			}
		}
	}
	return ids, nil
}

func deleteNotes(w io.Writer, file string, ids []int) error {
	if !hasContent(file) {
		fmt.Fprintln(w, "No notes yet.")
		return nil
	}

	target := map[int]bool{}
	for _, id := range ids {
		target[id] = true
	}

	f, err := os.Open(file)
	if err != nil {
		return err
	}
	var kept []string
	deleted := map[int]bool{}
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if m := noteIDLineRe.FindStringSubmatch(line); m != nil {
			if id, _ := strconv.Atoi(m[1]); target[id] {
				deleted[id] = true
				continue
			}
		}
		kept = append(kept, line)
	}
	scanErr := scanner.Err()
	f.Close()
	if scanErr != nil {
		return scanErr
	}

	// trim trailing blank lines but preserve final newline
	for len(kept) > 0 && kept[len(kept)-1] == "" {
		kept = kept[:len(kept)-1]
	}

	out, err := os.Create(file)
	if err != nil {
		return err
	}
	defer out.Close()
	bw := bufio.NewWriter(out)
	for _, line := range kept {
		fmt.Fprintln(bw, line)
	}
	if err := bw.Flush(); err != nil {
		return err
	}

	for _, id := range ids {
		if deleted[id] {
			fmt.Fprintf(w, "✓ Deleted [%d]\n", id)
		} else {
			fmt.Fprintf(w, "  Note [%d] not found\n", id)
		}
	}
	return nil
}

func editNoteByID(w io.Writer, file string, id int) error {
	if !hasContent(file) {
		return fmt.Errorf("note [%d] not found", id)
	}

	f, err := os.Open(file)
	if err != nil {
		return err
	}
	var lines []string
	var currentText string
	noteLineIdx := -1
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		lines = append(lines, line)
		if m := noteLineRe.FindStringSubmatch(line); m != nil {
			if lineID, _ := strconv.Atoi(m[1]); lineID == id {
				currentText = m[2]
				noteLineIdx = len(lines) - 1
			}
		}
	}
	f.Close()
	if err := scanner.Err(); err != nil {
		return err
	}
	if noteLineIdx == -1 {
		return fmt.Errorf("note [%d] not found", id)
	}

	// Use zsh's vared builtin: inline line editor with real vi key bindings.
	// bindkey -v enables vi mode; vared pre-fills $result with the current text.
	// The edited value is written to a temp file so we can capture it without
	// interfering with the interactive terminal session.
	tmp, err := os.CreateTemp("", "note-edit-*.txt")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	tmp.Close()
	defer os.Remove(tmpPath)

	prompt := fmt.Sprintf("[%d] › ", id)
	// $1 = initial text, $2 = prompt string, $3 = output file
	const script = `bindkey -v; result=$1; vared -p "$2" result; printf '%s' "$result" > "$3"`
	cmd := exec.Command("zsh", "-c", script, "--", currentText, prompt, tmpPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() != 0 {
			fmt.Fprintln(w, "Edit cancelled.")
			return nil
		}
		return err
	}

	data, err := os.ReadFile(tmpPath)
	if err != nil {
		return err
	}
	newText := strings.TrimSpace(processTags(string(data)))
	if newText == "" {
		fmt.Fprintln(w, "Note text cannot be empty, edit cancelled.")
		return nil
	}
	if newText == currentText {
		fmt.Fprintln(w, "No changes.")
		return nil
	}

	const sep = "** — "
	idx := strings.Index(lines[noteLineIdx], sep)
	if idx == -1 {
		return fmt.Errorf("could not parse note line [%d]", id)
	}
	lines[noteLineIdx] = lines[noteLineIdx][:idx+len(sep)] + newText

	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	out, err := os.Create(file)
	if err != nil {
		return err
	}
	defer out.Close()
	bw := bufio.NewWriter(out)
	for _, line := range lines {
		fmt.Fprintln(bw, line)
	}
	if err := bw.Flush(); err != nil {
		return err
	}

	fmt.Fprintf(w, "✓ Note [%d] updated\n", id)
	return nil
}

func appendNote(w io.Writer, file, text string) error {
	dir := filepath.Dir(file)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	needsHeader := !hasContent(file)

	f, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if needsHeader {
		if _, err := fmt.Fprintf(f, "# Quick Notes\n\n"); err != nil {
			return err
		}
	}

	id := nextNoteID(file)
	timestamp := time.Now().Format("2006-01-02 Mon 15:04")
	line := fmt.Sprintf("- [%d] **%s** — %s\n", id, timestamp, text)
	if _, err := fmt.Fprint(f, line); err != nil {
		return err
	}

	fmt.Fprintln(w, "✓ Note saved →", file)
	return nil
}

func main() {
	file := defaultNoteFile()
	args := os.Args[1:]

	var (
		flagFile    string
		flagTag     string
		flagNoMdurl bool
		contentArgs []string
	)

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "-t", "--tag":
			i++
			if i >= len(args) {
				fmt.Fprintf(os.Stderr, "note: flag '%s' requires a value\n", arg)
				os.Exit(1)
			}
			flagTag = args[i]
		case "-f", "--file":
			i++
			if i >= len(args) {
				fmt.Fprintf(os.Stderr, "note: flag '%s' requires a value\n", arg)
				os.Exit(1)
			}
			flagFile = args[i]
		case "--no-mdurl":
			flagNoMdurl = true
		default:
			if strings.HasPrefix(arg, "-") {
				fmt.Fprintf(os.Stderr, "note: unknown flag '%s'\n", arg)
				fmt.Fprintln(os.Stderr, "Run 'note -h' for help.")
				os.Exit(1)
			}
			contentArgs = append(contentArgs, arg)
		}
	}

	if flagFile != "" {
		file = flagFile
	}

	var err error
	switch {
	case len(contentArgs) == 0:
		printHelp(os.Stdout, file)
	default:
		action := contentArgs[0]
		switch action {
		case "add":
			if len(contentArgs) < 2 {
				fmt.Fprintln(os.Stderr, "note: 'add' requires note text")
				fmt.Fprintln(os.Stderr, "Run 'note -h' for help.")
				os.Exit(1)
			}
			text := processTags(strings.Join(contentArgs[1:], " "))
				if !flagNoMdurl {
					text = processURLs(text)
				}
				err = appendNote(os.Stdout, file, text)
		case "list":
			var filterTags []string
			if flagTag != "" {
				filterTags = strings.Split(flagTag, ",")
			}
			var filterID int
			if len(contentArgs) > 1 {
				filterID, err = strconv.Atoi(contentArgs[1])
				if err != nil || filterID < 1 {
					fmt.Fprintf(os.Stderr, "note: invalid id %q\n", contentArgs[1])
					os.Exit(1)
				}
				err = nil
			}
			err = listNotes(os.Stdout, file, filterTags, filterID)
		case "tags":
			err = listTags(os.Stdout, file)
		case "edit":
			if len(contentArgs) > 1 {
				id, parseErr := strconv.Atoi(contentArgs[1])
				if parseErr != nil || id < 1 {
					fmt.Fprintf(os.Stderr, "note: invalid id %q\n", contentArgs[1])
					os.Exit(1)
				}
				err = editNoteByID(os.Stdout, file, id)
			} else {
				err = editNotes(file)
			}
		case "delete":
			if len(contentArgs) < 2 {
				fmt.Fprintln(os.Stderr, "note: 'delete' requires at least one id")
				fmt.Fprintln(os.Stderr, "Run 'note' for help.")
				os.Exit(1)
			}
			ids, parseErr := parseIDs(contentArgs[1:])
			if parseErr != nil {
				fmt.Fprintln(os.Stderr, "note:", parseErr)
				os.Exit(1)
			}
			err = deleteNotes(os.Stdout, file, ids)
		default:
			fmt.Fprintf(os.Stderr, "note: unknown action '%s'\n", action)
			fmt.Fprintln(os.Stderr, "Run 'note -h' for help.")
			os.Exit(1)
		}
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "note:", err)
		os.Exit(1)
	}
}
