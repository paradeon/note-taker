package mdurl

import (
	"encoding/json"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

var htmlTitleRe = regexp.MustCompile(`(?i)<title[^>]*>([^<]+)</title>`)

var bracketReplacer = strings.NewReplacer("[", "(", "]", ")")

// NormalizeBrackets replaces [ and ] with ( and ) in s.
func NormalizeBrackets(s string) string { return bracketReplacer.Replace(s) }

// ParseMDURL parses a markdown link [desc](url), handling nested brackets in desc.
// Returns ok=false if s is not a valid markdown link.
func ParseMDURL(s string) (desc, rawURL string, ok bool) {
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

// FindMDLinkEnd returns the exclusive end index of a markdown link [desc](url)
// that begins at position start in s, using the same bracket-counting logic as
// ParseMDURL. Returns -1 if no valid markdown link starts there.
func FindMDLinkEnd(s string, start int) int {
	if start >= len(s) || s[start] != '[' {
		return -1
	}
	depth, i := 1, start+1
	for i < len(s) && depth > 0 {
		switch s[i] {
		case '[':
			depth++
		case ']':
			depth--
		}
		i++
	}
	// i now points one past the matching ']'
	if depth != 0 || i >= len(s) || s[i] != '(' {
		return -1
	}
	i++ // skip '('
	for i < len(s) && s[i] != ')' {
		i++
	}
	if i >= len(s) {
		return -1
	}
	return i + 1
}

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
		if t := fetchYouTubeTitle(rawURL); t != "" {
			return t
		}
		// oEmbed failed (e.g. age-restricted/private) — scrape HTML and strip suffix.
		t := scrapeHTMLTitle(rawURL)
		return strings.TrimSuffix(t, " - YouTube")
	}

	return scrapeHTMLTitle(rawURL)
}

func scrapeHTMLTitle(rawURL string) string {
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
			return html.UnescapeString(strings.TrimSpace(string(m[1])))
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
