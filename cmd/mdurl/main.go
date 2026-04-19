package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
)

var titleRe = regexp.MustCompile(`(?i)<title[^>]*>([^<]+)</title>`)

func isYouTube(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := strings.TrimPrefix(u.Hostname(), "www.")
	return host == "youtube.com" || host == "youtu.be"
}

func fetchYouTubeTitle(videoURL string) (string, error) {
	apiURL := "https://www.youtube.com/oembed?url=" + url.QueryEscape(videoURL) + "&format=json"
	resp, err := http.Get(apiURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("oembed status %d", resp.StatusCode)
	}
	var data struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}
	return data.Title, nil
}

func fetchTitle(rawURL string) (string, error) {
	if isYouTube(rawURL) {
		return fetchYouTubeTitle(rawURL)
	}

	req, err := http.NewRequest("GET", rawURL, nil)
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
	return rawURL, nil
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
