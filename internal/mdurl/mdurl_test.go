package mdurl

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ── isYouTube ────────────────────────────────────────────────────────────────

func TestIsYouTube(t *testing.T) {
	cases := []struct {
		url  string
		want bool
	}{
		{"https://www.youtube.com/watch?v=abc", true},
		{"https://youtube.com/watch?v=abc", true},
		{"https://youtu.be/abc123", true},
		{"https://www.youtu.be/abc123", true},
		{"https://www.youtube.com/shorts/abc", true},
		{"https://example.com", false},
		{"https://notyoutube.com/watch?v=abc", false},
		{"not a url at all", false},
		{"", false},
	}
	for _, tc := range cases {
		if got := isYouTube(tc.url); got != tc.want {
			t.Errorf("isYouTube(%q) = %v, want %v", tc.url, got, tc.want)
		}
	}
}

// ── scrapeHTMLTitle ──────────────────────────────────────────────────────────

func TestScrapeHTMLTitle_Basic(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><head><title>Hello World</title></head><body></body></html>`)
	}))
	defer srv.Close()

	got := scrapeHTMLTitle(srv.URL)
	if got != "Hello World" {
		t.Errorf("got %q, want %q", got, "Hello World")
	}
}

func TestScrapeHTMLTitle_HTMLEntities(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><head><title>Hello &amp; World &#39;test&#39;</title></head></html>`)
	}))
	defer srv.Close()

	got := scrapeHTMLTitle(srv.URL)
	if got != "Hello & World 'test'" {
		t.Errorf("got %q, want %q", got, "Hello & World 'test'")
	}
}

func TestScrapeHTMLTitle_TitleWithWhitespace(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "<html><head><title>  Trimmed  </title></head></html>")
	}))
	defer srv.Close()

	got := scrapeHTMLTitle(srv.URL)
	if got != "Trimmed" {
		t.Errorf("got %q, want %q", got, "Trimmed")
	}
}

func TestScrapeHTMLTitle_CaseInsensitiveTag(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<HTML><HEAD><TITLE>Upper Case</TITLE></HEAD></HTML>`)
	}))
	defer srv.Close()

	got := scrapeHTMLTitle(srv.URL)
	if got != "Upper Case" {
		t.Errorf("got %q, want %q", got, "Upper Case")
	}
}

func TestScrapeHTMLTitle_Missing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body>no title element here</body></html>`)
	}))
	defer srv.Close()

	got := scrapeHTMLTitle(srv.URL)
	if got != "" {
		t.Errorf("expected empty string for missing title, got %q", got)
	}
}

func TestScrapeHTMLTitle_UnreachableURL(t *testing.T) {
	got := scrapeHTMLTitle("http://localhost:1") // nothing listening here
	if got != "" {
		t.Errorf("expected empty string for unreachable URL, got %q", got)
	}
}

func TestScrapeHTMLTitle_TitleWithAttributes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><head><title lang="en">With Attrs</title></head></html>`)
	}))
	defer srv.Close()

	got := scrapeHTMLTitle(srv.URL)
	if got != "With Attrs" {
		t.Errorf("got %q, want %q", got, "With Attrs")
	}
}

// ── ParseMDURL ───────────────────────────────────────────────────────────────

func TestParseMDURL(t *testing.T) {
	cases := []struct {
		input  string
		desc   string
		rawURL string
		ok     bool
	}{
		{
			input:  "[Hello World](https://example.com)",
			desc:   "Hello World",
			rawURL: "https://example.com",
			ok:     true,
		},
		{
			input:  "[蛇と蜘蛛 [中文字幕] - Hanime1.me](https://hanime1.me/watch?v=404983)",
			desc:   "蛇と蜘蛛 [中文字幕] - Hanime1.me",
			rawURL: "https://hanime1.me/watch?v=404983",
			ok:     true,
		},
		{
			input:  "[[outer [inner] text]](https://example.com)",
			desc:   "[outer [inner] text]",
			rawURL: "https://example.com",
			ok:     true,
		},
		{
			input:  "[no url]()",
			desc:   "no url",
			rawURL: "",
			ok:     true,
		},
		// not a markdown link
		{input: "https://example.com", ok: false},
		{input: "", ok: false},
		// missing closing paren
		{input: "[desc](https://example.com", ok: false},
		// missing opening bracket on URL
		{input: "[desc]https://example.com", ok: false},
		// unmatched bracket in desc
		{input: "[desc(https://example.com)", ok: false},
	}

	for _, tc := range cases {
		desc, rawURL, ok := ParseMDURL(tc.input)
		if ok != tc.ok {
			t.Errorf("ParseMDURL(%q): ok=%v, want %v", tc.input, ok, tc.ok)
			continue
		}
		if !ok {
			continue
		}
		if desc != tc.desc {
			t.Errorf("ParseMDURL(%q): desc=%q, want %q", tc.input, desc, tc.desc)
		}
		if rawURL != tc.rawURL {
			t.Errorf("ParseMDURL(%q): rawURL=%q, want %q", tc.input, rawURL, tc.rawURL)
		}
	}
}

// ── FindMDLinkEnd ────────────────────────────────────────────────────────────

func TestFindMDLinkEnd(t *testing.T) {
	cases := []struct {
		s     string
		start int
		want  int // -1 = no match
	}{
		{"[Go](https://golang.org)", 0, 24},
		// link embedded in text
		{"see [Go](https://golang.org) ok", 4, 28},
		// one level of nesting
		{"[Title [sub]](https://example.com)", 0, 34},
		// two levels of nesting
		{"[[outer [inner] text]](https://example.com)", 0, 43},
		// multiple bracket groups
		{"[[A] [B]](https://example.com)", 0, 30},
		// not a link
		{"just text", 0, -1},
		// missing closing paren
		{"[desc](https://example.com", 0, -1},
		// start past end
		{"[Go](https://golang.org)", 25, -1},
	}
	for _, tc := range cases {
		if got := FindMDLinkEnd(tc.s, tc.start); got != tc.want {
			t.Errorf("FindMDLinkEnd(%q, %d) = %d, want %d", tc.s, tc.start, got, tc.want)
		}
	}
}

// ── NormalizeBrackets ────────────────────────────────────────────────────────

func TestNormalizeBrackets(t *testing.T) {
	cases := []struct{ in, want string }{
		{"no brackets", "no brackets"},
		{"[foo]", "(foo)"},
		{"a [b] c [d]", "a (b) c (d)"},
		{"nested [a [b] c]", "nested (a (b) c)"},
	}
	for _, tc := range cases {
		if got := NormalizeBrackets(tc.in); got != tc.want {
			t.Errorf("NormalizeBrackets(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// ── FetchTitle ───────────────────────────────────────────────────────────────

func TestFetchTitle_NonYouTube(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><head><title>My Blog Post</title></head></html>`)
	}))
	defer srv.Close()

	got := FetchTitle(srv.URL)
	if got != "My Blog Post" {
		t.Errorf("got %q, want %q", got, "My Blog Post")
	}
}

func TestFetchTitle_NonYouTubeNoTitle(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body>no title</body></html>`)
	}))
	defer srv.Close()

	got := FetchTitle(srv.URL)
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestFetchTitle_Unreachable(t *testing.T) {
	got := FetchTitle("http://localhost:1")
	if got != "" {
		t.Errorf("expected empty string for unreachable URL, got %q", got)
	}
}
