package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
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
	fmt.Fprintln(w, "    note add <text...>           Append a timestamped note")
	fmt.Fprintln(w, "    note add \"quoted text\"       Quoted or unquoted — both work")
	fmt.Fprintln(w, "    note -e / --edit             Open notes in nvim")
	fmt.Fprintln(w, "    note -t / --tag <tag>        Show notes containing #<tag>")
	fmt.Fprintln(w, "    note -f / --file <path>      Use a specific notes file")
	fmt.Fprintln(w, "    note -h / --help             Show this help")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  Tags:")
	fmt.Fprintln(w, "    Embed #tags anywhere in your note text:")
	fmt.Fprintln(w, "    note add fix login bug #auth #backend")
	fmt.Fprintln(w, "    note -t auth")
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

	timestamp := time.Now().Format("2006-01-02 Mon 15:04")
	line := fmt.Sprintf("- **%s** — %s\n", timestamp, text)
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
		flagHelp    bool
		flagEdit    bool
		flagTag     string
		flagFile    string
		contentArgs []string
	)

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "-h", "--help":
			flagHelp = true
		case "-e", "--edit":
			flagEdit = true
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
	case flagHelp:
		printHelp(os.Stdout, file)
	case flagEdit:
		err = editNotes(file)
	case flagTag != "":
		err = searchTag(os.Stdout, file, flagTag)
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
			err = appendNote(os.Stdout, file, strings.Join(contentArgs[1:], " "))
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
