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
