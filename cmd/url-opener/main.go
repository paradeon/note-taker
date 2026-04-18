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

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		url := extractURL(scanner.Text())
		if url == "" {
			continue
		}
		cmd := exec.Command(*opener, url)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "url-opener: %v\n", err)
			os.Exit(1)
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "url-opener: %v\n", err)
		os.Exit(1)
	}
}
