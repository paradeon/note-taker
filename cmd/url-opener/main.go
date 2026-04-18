package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
)

var mdLinkRe = regexp.MustCompile(`\[.*?\]\((https?://[^)]+)\)`)

func extractURL(line string) string {
	m := mdLinkRe.FindStringSubmatch(line)
	if m == nil {
		return ""
	}
	return m[1]
}

func main() {
	opener := flag.String("a", "open", "command to open URLs with")
	useMpv := flag.Bool("m", false, "open URLs with mpv")
	useYtdlp := flag.Bool("y", false, "open URLs with yt-dlp")
	flag.Parse()

	if *useMpv {
		*opener = "mpv"
	} else if *useYtdlp {
		*opener = "yt-dlp"
	}

	openURL := func(url string) {
		cmd := exec.Command(*opener, url)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "url-opener: %v\n", err)
			os.Exit(1)
		}
	}

	args := flag.Args()
	if len(args) == 1 && args[0] == "-" {
		args = nil
	}

	if len(args) > 0 {
		for _, url := range args {
			openURL(url)
		}
		return
	}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		url := extractURL(scanner.Text())
		if url == "" {
			continue
		}
		openURL(url)
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "url-opener: %v\n", err)
		os.Exit(1)
	}
}
