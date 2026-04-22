package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"note/internal/mdurl"
)

// parseMDURL parses a markdown link [desc](url), handling nested brackets in desc.
func parseMDURL(s string) (desc, rawURL string, ok bool) {
	if len(s) == 0 || s[0] != '[' {
		return
	}
	depth, i := 1, 1
	for i < len(s) && depth > 0 {
		switch s[i] {
		case '[':
			depth++
		case ']':
			depth--
		}
		i++
	}
	if depth != 0 || i >= len(s) || s[i] != '(' || s[len(s)-1] != ')' {
		return
	}
	return s[1 : i-1], s[i+1 : len(s)-1], true
}

func main() {
	ytID := flag.String("y", "", "YouTube video ID (expands to full watch URL)")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: mdurl <url>")
		fmt.Fprintln(os.Stderr, "       mdurl -y <video-id>")
	}
	flag.Parse()

	replaceBrackets := strings.NewReplacer("[", "(", "]", ")")

	var rawURL string
	if *ytID != "" {
		rawURL = "https://www.youtube.com/watch?v=" + *ytID
	} else if flag.NArg() == 1 {
		arg := flag.Arg(0)
		if desc, u, ok := parseMDURL(arg); ok {
			fmt.Printf("[%s](%s)\n", replaceBrackets.Replace(desc), u)
			return
		}
		rawURL = arg
	} else {
		flag.Usage()
		os.Exit(1)
	}

	title := mdurl.FetchTitle(rawURL)
	if title == "" {
		title = rawURL
	}
	fmt.Printf("[%s](%s)\n", replaceBrackets.Replace(title), rawURL)
}
