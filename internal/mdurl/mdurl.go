package mdurl

import (
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

var htmlTitleRe = regexp.MustCompile(`(?i)<title[^>]*>([^<]+)</title>`)

func isYouTube(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := strings.TrimPrefix(u.Hostname(), "www.")
	return host == "youtube.com" || host == "youtu.be"
}

func fetchYouTubeTitle(rawURL string) string {
	apiURL := "https://www.youtube.com/oembed?url=" + url.QueryEscape(rawURL) + "&format=json"
	resp, err := http.Get(apiURL)
	if err != nil || resp.StatusCode != 200 {
		return ""
	}
	defer resp.Body.Close()
	var data struct {
		Title      string `json:"title"`
		AuthorName string `json:"author_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return ""
	}
	if data.AuthorName != "" {
		return data.AuthorName + " - " + data.Title
	}
	return data.Title
}

// FetchTitle retrieves the page title for the given URL.
// Returns an empty string if the title cannot be determined.
func FetchTitle(rawURL string) string {
	if isYouTube(rawURL) {
		return fetchYouTubeTitle(rawURL)
	}

	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	const chunkSize = 32 * 1024
	const overlap = 256
	buf := make([]byte, 0, chunkSize+overlap)
	tmp := make([]byte, chunkSize)
	for {
		n, readErr := resp.Body.Read(tmp)
		buf = append(buf, tmp[:n]...)
		if m := htmlTitleRe.FindSubmatch(buf); m != nil {
			return strings.TrimSpace(string(m[1]))
		}
		if readErr != nil {
			break
		}
		if len(buf) > overlap {
			buf = buf[len(buf)-overlap:]
		}
	}
	return ""
}
