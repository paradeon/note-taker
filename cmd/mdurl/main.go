package main

import (
	"flag"
	"fmt"
	"os"

	"note/internal/mdurl"
)

func main() {
	ytID := flag.String("y", "", "YouTube video ID (expands to full watch URL)")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: mdurl <url>")
		fmt.Fprintln(os.Stderr, "       mdurl -y <video-id>")
	}
	flag.Parse()

	var rawURL string
	if *ytID != "" {
		rawURL = "https://www.youtube.com/watch?v=" + *ytID
	} else if flag.NArg() == 1 {
		arg := flag.Arg(0)
		if desc, u, ok := mdurl.ParseMDURL(arg); ok {
			fmt.Printf("[%s](%s)\n", mdurl.NormalizeBrackets(desc), u)
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
	fmt.Printf("[%s](%s)\n", mdurl.NormalizeBrackets(title), rawURL)
}
