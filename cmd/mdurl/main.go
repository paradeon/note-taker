package main

import (
	"fmt"
	"os"

	"note/internal/mdurl"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: mdurl <url>")
		os.Exit(1)
	}
	url := os.Args[1]
	title := mdurl.FetchTitle(url)
	if title == "" {
		title = url
	}
	fmt.Printf("[%s](%s)\n", title, url)
}
