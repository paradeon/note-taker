package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"regexp"
)

var mdLinkRe = regexp.MustCompile(`\[.*?\]\((https?://[^)]+)\)`)
var bareURLRe = regexp.MustCompile(`https?://\S+`)
var tagRe = regexp.MustCompile(`#(\w+)`)

func extractURLs(line string) []string {
	var urls []string
	for _, m := range mdLinkRe.FindAllStringSubmatch(line, -1) {
		urls = append(urls, m[1])
	}
	remaining := mdLinkRe.ReplaceAllString(line, "")
	for _, m := range bareURLRe.FindAllString(remaining, -1) {
		urls = append(urls, m)
	}
	return urls
}

func extractTags(line string) []string {
	var tags []string
	for _, m := range tagRe.FindAllStringSubmatch(line, -1) {
		tags = append(tags, "#"+m[1])
	}
	return tags
}

func main() {
	doURL := flag.Bool("u", false, "Extract URLs")
	doTag := flag.Bool("t", false, "Extract tags (#tag)")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: note-extractor [-u] [-t]")
		fmt.Fprintln(os.Stderr, "  Reads note lines from stdin and prints extracted values.")
		fmt.Fprintln(os.Stderr, "  -u  extract URLs (markdown or bare)")
		fmt.Fprintln(os.Stderr, "  -t  extract tags (#tag)")
		fmt.Fprintln(os.Stderr, "  If neither flag is given, both are extracted.")
	}
	flag.Parse()

	if !*doURL && !*doTag {
		*doURL = true
		*doTag = true
	}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		if *doURL {
			for _, u := range extractURLs(line) {
				fmt.Println(u)
			}
		}
		if *doTag {
			for _, t := range extractTags(line) {
				fmt.Println(t)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "note-extractor:", err)
		os.Exit(1)
	}
}
