package main

import (
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
)

var titleRe = regexp.MustCompile(`(?i)<title[^>]*>([^<]+)</title>`)

func fetchTitle(url string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Stream in 32KB chunks, keeping a small overlap to avoid splitting the tag across chunks.
	const chunkSize = 32 * 1024
	const overlap = 256
	buf := make([]byte, 0, chunkSize+overlap)
	tmp := make([]byte, chunkSize)
	for {
		n, readErr := resp.Body.Read(tmp)
		buf = append(buf, tmp[:n]...)
		if m := titleRe.FindSubmatch(buf); m != nil {
			return strings.TrimSpace(string(m[1])), nil
		}
		if readErr != nil {
			break
		}
		// Keep only the last `overlap` bytes to catch tags spanning chunk boundaries.
		if len(buf) > overlap {
			buf = buf[len(buf)-overlap:]
		}
	}
	return url, nil
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: mdurl <url>")
		os.Exit(1)
	}
	url := os.Args[1]
	title, err := fetchTitle(url)
	if err != nil {
		fmt.Fprintln(os.Stderr, "mdurl:", err)
		os.Exit(1)
	}
	fmt.Printf("[%s](%s)\n", title, url)
}
