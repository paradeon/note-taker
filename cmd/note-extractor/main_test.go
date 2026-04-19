package main

import (
	"reflect"
	"testing"
)

// ── extractURLs ──────────────────────────────────────────────────────────────

func TestExtractURLs_MarkdownLink(t *testing.T) {
	got := extractURLs("[Go](https://golang.org)")
	want := []string{"https://golang.org"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestExtractURLs_BareURL(t *testing.T) {
	got := extractURLs("check https://example.com out")
	want := []string{"https://example.com"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestExtractURLs_MultipleMarkdownLinks(t *testing.T) {
	got := extractURLs("[A](https://a.com) and [B](https://b.com)")
	want := []string{"https://a.com", "https://b.com"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestExtractURLs_MarkdownAndBare(t *testing.T) {
	got := extractURLs("[A](https://a.com) also https://b.com")
	want := []string{"https://a.com", "https://b.com"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestExtractURLs_BareURLNotDuplicatedFromMarkdown(t *testing.T) {
	// The URL inside [text](url) must not also appear as a bare URL match.
	got := extractURLs("[Go](https://golang.org)")
	if len(got) != 1 {
		t.Errorf("expected 1 URL, got %v", got)
	}
}

func TestExtractURLs_NoURL(t *testing.T) {
	got := extractURLs("plain text with no links")
	if len(got) != 0 {
		t.Errorf("expected empty, got %v", got)
	}
}

func TestExtractURLs_EmptyLine(t *testing.T) {
	got := extractURLs("")
	if len(got) != 0 {
		t.Errorf("expected empty, got %v", got)
	}
}

func TestExtractURLs_HTTPandHTTPS(t *testing.T) {
	got := extractURLs("http://insecure.com and https://secure.com")
	want := []string{"http://insecure.com", "https://secure.com"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// ── extractTags ──────────────────────────────────────────────────────────────

func TestExtractTags_Single(t *testing.T) {
	got := extractTags("fix login #auth")
	want := []string{"#auth"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestExtractTags_Multiple(t *testing.T) {
	got := extractTags("fix login #auth #backend")
	want := []string{"#auth", "#backend"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestExtractTags_NoTags(t *testing.T) {
	got := extractTags("plain note with no tags")
	if len(got) != 0 {
		t.Errorf("expected empty, got %v", got)
	}
}

func TestExtractTags_EmptyLine(t *testing.T) {
	got := extractTags("")
	if len(got) != 0 {
		t.Errorf("expected empty, got %v", got)
	}
}

func TestExtractTags_PreservesHashPrefix(t *testing.T) {
	got := extractTags("note #go")
	if len(got) != 1 || got[0] != "#go" {
		t.Errorf("got %v, want [#go]", got)
	}
}

func TestExtractTags_MixedWithURL(t *testing.T) {
	got := extractTags("see [Go](https://golang.org) for #go tips")
	want := []string{"#go"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
